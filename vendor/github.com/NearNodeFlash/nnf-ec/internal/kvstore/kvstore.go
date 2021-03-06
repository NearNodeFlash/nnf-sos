/*
 * Copyright 2020, 2021, 2022 Hewlett Packard Enterprise Development LP
 * Other additional copyright holders may be indicated within.
 *
 * The entirety of this work is licensed under the Apache License,
 * Version 2.0 (the "License"); you may not use this file except
 * in compliance with the License.
 *
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package kvstore

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/dgraph-io/badger/v3"
)

const (
	KeyPrefixLength = 2
)

type Store struct {
	path       string
	db         *badger.DB
	registries []Registry
}

func Open(path string, readOnly bool) (*Store, error) {

	opts := badger.DefaultOptions(path)
	opts.SyncWrites = true
	//opts.ReadOnly = readOnly // Causes ErrLogTruncate
	opts.BypassLockGuard = readOnly

	// Shrink the in-memory and on-disk size to a more manageable 8 MiB and 16 MiB, respectively;
	// We use very little data and the 64 MiB and 256 MiB defaults will cause OOM issues in kubernetes.
	// 8MiB seems to be the lower limit within badger, anything smaller and badger will complain with
	//   """
	//   Valuethreshold 1048576 greater than max batch size of 629145. Either reduce opt.ValueThreshold
	//   or increase opt.MaxTableSize.
	//   """
	opts.MemTableSize = 8 << 20
	opts.BlockCacheSize = 16 << 20

	db, err := badger.Open(opts)

	if err != nil {
		return nil, err
	}

	return &Store{path: path, db: db, registries: make([]Registry, 0)}, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) Register(registries []Registry) {
	for _, registry := range registries {
		for _, r := range s.registries {
			mlen := int(math.Min(float64(len(r.Prefix())), float64(len(registry.Prefix()))))
			if strings.Compare(r.Prefix()[0:mlen], registry.Prefix()[0:mlen]) == 0 {
				panic(fmt.Sprintf("Registry Prefix '%s' conflicts with existing registry '%s' ", registry.Prefix(), r.Prefix()))
			}
		}

		s.registries = append(s.registries, registry)
	}
}

func (s *Store) Replay() error {
	for _, r := range s.registries {

		deleteKeys := make([]string, 0)
		err := s.db.View(func(txn *badger.Txn) error {
			opts := badger.DefaultIteratorOptions
			if len(r.Prefix()) != 0 {
				opts.Prefix = []byte(r.Prefix())
			}

			itr := txn.NewIterator(opts)
			defer itr.Close()

			for itr.Rewind(); itr.Valid(); itr.Next() {
				item := itr.Item()
				key := item.Key()
				value := []byte{}

				err := item.Value(func(val []byte) error {
					value = append([]byte{}, val...)
					return nil
				})

				if err != nil {
					return err
				}

				delete, err := s.runReply(r, key, value)
				if err != nil {
					return err
				}

				// Record the key for deletion so it can be deleted
				// outside this transaction.
				if delete {
					deleteKeys = append(deleteKeys, string(key))
				}
			}

			return nil
		})

		if err != nil {
			return err
		}

		for _, key := range deleteKeys {
			if err := s.DeleteKey(key); err != nil {
				return err
			}
		}

	}

	return nil
}

func (s *Store) MakeKey(registry Registry, id string) string {
	return registry.Prefix() + id
}

// NewKey will create the provided key with metadata provided. It returns a logger which tracks updates
// to the key's value.
func (s *Store) NewKey(key string, metadata []byte) (*Ledger, error) {

	// Check that they key is in the registries
	for _, r := range s.registries {
		if strings.HasPrefix(key, r.Prefix()) {
			// Create the Metadata TLV
			tlv := newTlv(metadataTlvType, metadata)

			err := s.db.Update(func(txn *badger.Txn) error {
				return txn.Set([]byte(key), tlv.bytes())
			})

			if err != nil {
				return nil, err
			}

			if err := s.db.Sync(); err != nil {
				return nil, err
			}

			return s.newKeyLedger(key, tlv.bytes()), nil
		}
	}

	return nil, ErrRegistryNotFound
}

func (s *Store) OpenKey(key string) (*Ledger, error) {
	for _, r := range s.registries {
		if strings.HasPrefix(key, r.Prefix()) {

			ledger := s.existingKeyLedger(key)
			err := s.db.View(func(txn *badger.Txn) error {
				item, err := txn.Get([]byte(key))
				if err != nil {
					return err
				}

				return item.Value(func(val []byte) error {
					ledger.bytes = append([]byte{}, val...)
					return nil
				})
			})

			if err != nil {
				return nil, err
			}

			if err := s.db.Sync(); err != nil {
				return nil, err
			}

			return ledger, nil
		}
	}

	return nil, ErrRegistryNotFound
}

func (s *Store) DeleteKey(key string) error {
	ledger, err := s.OpenKey(key)
	if err != nil {
		return err
	}

	return ledger.Close(true)
}

var ErrRegistryNotFound = errors.New("registry not found")

type Registry interface {
	Prefix() string
	NewReplay(id string) ReplayHandler
}

func (s *Store) runReply(registry Registry, key []byte, data []byte) (delete bool, err error) {
	id := string(key[len(registry.Prefix()):])
	it := newIterator(data)
	replay := registry.NewReplay(string(id))
	for tlv, done := it.Next(); !done; tlv, done = it.Next() {
		if tlv.t == metadataTlvType {
			err = replay.Metadata(tlv.v)
		} else {
			err = replay.Entry(tlv.t, tlv.v)
		}

		if err != nil {
			return false, err
		}
	}

	return replay.Done()
}

type ReplayHandler interface {
	Metadata(data []byte) error
	Entry(t uint32, data []byte) error
	Done() (bool, error)
}

const (
	metadataTlvType uint32 = 0xFFFFFFFF
)

type tlv struct {
	t uint32
	l uint32
	v []byte
}

func newTlv(t uint32, v []byte) tlv {
	return tlv{t: t, l: uint32(len(v)), v: v}
}

func (tlv tlv) bytes() []byte {
	b := make([]byte, 8+tlv.l)
	binary.LittleEndian.PutUint32(b[0:4], tlv.t)
	binary.LittleEndian.PutUint32(b[4:8], tlv.l)
	copy(b[8:], tlv.v)
	return b
}

type tlvIterator struct {
	index int
	v     []byte
}

func newIterator(v []byte) *tlvIterator {
	return &tlvIterator{index: 0, v: v}
}

func (it *tlvIterator) Next() (tlv, bool) {
	index := it.index
	if index == len(it.v) {
		return tlv{}, true
	}

	tlv := tlv{
		t: binary.LittleEndian.Uint32(it.v[index+0 : index+4]),
		l: binary.LittleEndian.Uint32(it.v[index+4 : index+8]),
	}

	tlv.v = it.v[index+8 : index+8+int(tlv.l)]
	it.index += 8 + int(tlv.l)

	return tlv, false
}

type Ledger struct {
	s     *Store
	key   string
	bytes []byte
}

func (l *Ledger) Log(t uint32, v []byte) error {

	tlv := newTlv(t, v)

	err := l.s.db.Update(func(txn *badger.Txn) error {
		l.bytes = append(l.bytes, tlv.bytes()...)
		return txn.Set([]byte(l.key), l.bytes)
	})

	if err != nil {
		return err
	}

	return l.s.db.Sync()
}

func (l *Ledger) Close(delete bool) error {
	if delete {

		txn := l.s.db.NewTransaction(true)
		if err := txn.Delete([]byte(l.key)); err != nil {
			return err
		}

		return txn.Commit()
	}
	return nil
}

func (s *Store) newKeyLedger(key string, bytes []byte) *Ledger {
	return &Ledger{s: s, key: key, bytes: bytes}
}

func (s *Store) existingKeyLedger(key string) *Ledger {
	return &Ledger{s: s, key: key}
}
