/*
 * Copyright 2022 Hewlett Packard Enterprise Development LP
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
	"github.com/dgraph-io/badger/v3"
)

func NewLocalPersistentStorageProvider() PersistentStorageProvider {
	return &localPersistentStorageProvider{}
}

type localPersistentStorageProvider struct{}

// NewPersistentStorageInterface implements PersistentStorageProvider
func (*localPersistentStorageProvider) NewPersistentStorageInterface(path string, readOnly bool) (PersistentStorageApi, error) {
	s := &localPersistentStorage{}
	return s, s.open(path, readOnly)
}

type localPersistentStorage struct {
	*badger.DB
}

func (s *localPersistentStorage) open(path string, readOnly bool) (err error) {
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

	s.DB, err = badger.Open(opts)
	return err
}

func (s *localPersistentStorage) Close() error {
	return s.DB.Close()
}

func (s *localPersistentStorage) View(fn func(PersistentStorageTransactionApi) error) error {
	return s.DB.View(func(txn *badger.Txn) error {
		return fn(&localPersistentStorageTransaction{txn})
	})
}

func (s *localPersistentStorage) Update(fn func(PersistentStorageTransactionApi) error) error {
	return s.DB.Update(func(txn *badger.Txn) error {
		return fn(&localPersistentStorageTransaction{txn})
	})
}

func (s *localPersistentStorage) Delete(key string) error {
	txn := s.DB.NewTransaction(true)
	if err := txn.Delete([]byte(key)); err != nil {
		return err
	}

	return txn.Commit()
}

type localPersistentStorageTransaction struct {
	*badger.Txn
}

func (txn *localPersistentStorageTransaction) NewIterator(prefix string) PersistentStorageIteratorApi {
	opts := badger.DefaultIteratorOptions
	if len(prefix) != 0 {
		opts.Prefix = []byte(prefix)
	}

	return &localPersistentStorageIterator{txn.Txn.NewIterator(opts)}
}

func (txn *localPersistentStorageTransaction) Set(key string, value []byte) error {
	return txn.Txn.Set([]byte(key), []byte(value))
}

func (txn *localPersistentStorageTransaction) Get(key string) ([]byte, error) {
	value := []byte{}
	item, err := txn.Txn.Get([]byte(key))
	if err != nil {
		return value, err
	}

	err = item.Value(func(val []byte) error {
		value = append(value, val...)
		return nil
	})

	return value, err

}

type localPersistentStorageIterator struct {
	*badger.Iterator
}

func (itr *localPersistentStorageIterator) Rewind() {
	itr.Iterator.Rewind()
}

func (itr *localPersistentStorageIterator) Valid() bool {
	return itr.Iterator.Valid()
}

func (itr *localPersistentStorageIterator) Next() {
	itr.Iterator.Next()
}

func (itr *localPersistentStorageIterator) Key() string {
	return string(itr.Iterator.Item().Key())
}

func (itr *localPersistentStorageIterator) Value() ([]byte, error) {
	item := itr.Iterator.Item()
	value := []byte{}

	err := item.Value(func(val []byte) error {
		value = append(value, val...)
		return nil
	})

	return value, err
}

func (itr *localPersistentStorageIterator) Close() {
	itr.Iterator.Close()
}
