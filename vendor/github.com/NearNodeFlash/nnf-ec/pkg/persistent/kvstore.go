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

package persistent

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"strings"
)

const (
	KeyPrefixLength = 2
)

type Store struct {
	path       string
	storage    PersistentStorageApi
	registries []Registry
}

func Open(path string, readOnly bool) (*Store, error) {
	s, err := StorageProvider.NewPersistentStorageInterface(path, readOnly)
	return &Store{path: path, storage: s, registries: make([]Registry, 0)}, err
}

func (s *Store) Close() error { return s.storage.Close() }

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

		err := s.storage.View(func(txn PersistentStorageTransactionApi) error {
			itr := txn.NewIterator(r.Prefix())
			defer itr.Close()

			for itr.Rewind(); itr.Valid(); itr.Next() {

				key := itr.Key()
				value, err := itr.Value()
				if err != nil {
					return err
				}

				delete, err := s.runReplay(r, key, value)
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

			err := s.storage.Update(func(txn PersistentStorageTransactionApi) error {
				return txn.Set(key, tlv.bytes())
			})

			if err != nil {
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
			err := s.storage.View(func(txn PersistentStorageTransactionApi) error {
				value, err := txn.Get(key)
				if err != nil {
					return err
				}
				ledger.bytes = value
				return nil
			})

			if err != nil {
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

func (s *Store) runReplay(registry Registry, key string, data []byte) (delete bool, err error) {
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
	l.bytes = append(l.bytes, tlv.bytes()...)

	err := l.s.storage.Update(func(txn PersistentStorageTransactionApi) error {
		return txn.Set(l.key, l.bytes)
	})

	return err
}

func (l *Ledger) Close(delete bool) error {
	if delete {
		return l.s.storage.Delete(l.key)
	}
	return nil
}

func (s *Store) newKeyLedger(key string, bytes []byte) *Ledger {
	return &Ledger{s: s, key: key, bytes: bytes}
}

func (s *Store) existingKeyLedger(key string) *Ledger {
	return &Ledger{s: s, key: key}
}
