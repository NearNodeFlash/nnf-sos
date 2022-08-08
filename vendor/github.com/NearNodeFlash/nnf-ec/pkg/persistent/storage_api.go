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

var StorageProvider = NewLocalPersistentStorageProvider()

type PersistentStorageProvider interface {
	NewPersistentStorageInterface(path string, readOnly bool) (PersistentStorageApi, error)
}

// Persistent Storage API provides an interface for interacting with persistent storage
type PersistentStorageApi interface {
	View(func(txn PersistentStorageTransactionApi) error) error
	Update(func(txn PersistentStorageTransactionApi) error) error
	Delete(key string) error

	Close() error
}

// Persistent Storage Transaction API provides an interface for interacting with persistent storage transactions
type PersistentStorageTransactionApi interface {
	NewIterator(prefix string) PersistentStorageIteratorApi
	Set(key string, value []byte) error
	Get(key string) ([]byte, error)
}

// Persistent Storage Iterator API provides an iterface for interacting with persistent storage iterators
type PersistentStorageIteratorApi interface {
	Rewind()
	Valid() bool
	Next()

	Key() string
	Value() ([]byte, error)

	Close()
}
