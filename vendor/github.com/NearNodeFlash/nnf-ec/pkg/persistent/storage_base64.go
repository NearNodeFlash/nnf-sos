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
	"encoding/base64"
	"fmt"
	"strings"
)

type base64PersistentStorageTransaction struct {
	data map[string]string
}

func NewBase64PersistentStorageTransaction(data map[string]string) PersistentStorageTransactionApi {
	return &base64PersistentStorageTransaction{data: data}
}

func (txn *base64PersistentStorageTransaction) Get(key string) ([]byte, error) {
	value, found := txn.data[key]
	if !found {
		return nil, fmt.Errorf("Key %s not found", key)
	}

	return base64.StdEncoding.DecodeString(value)
}

func (txn *base64PersistentStorageTransaction) Set(key string, value []byte) error {
	txn.data[key] = base64.StdEncoding.EncodeToString(value)
	return nil
}

func (txn *base64PersistentStorageTransaction) NewIterator(prefix string) PersistentStorageIteratorApi {
	itr := base64PersistentStorageIterator{
		keys:  make([]string, 0),
		data:  txn.data,
		index: 0,
	}

	for key := range txn.data {
		if strings.HasPrefix(key, prefix) {
			itr.keys = append(itr.keys, key)
		}
	}

	return &itr
}

type base64PersistentStorageIterator struct {
	keys  []string
	data  map[string]string
	index int
}

func (itr *base64PersistentStorageIterator) Close() {
	itr.keys = []string{}
}

func (itr *base64PersistentStorageIterator) Key() string {
	return itr.keys[itr.index]
}

func (itr *base64PersistentStorageIterator) Rewind() {
	itr.index = 0
}

func (itr *base64PersistentStorageIterator) Valid() bool {
	return itr.index < len(itr.keys)
}

func (itr *base64PersistentStorageIterator) Next() {
	itr.index += 1
}

func (itr *base64PersistentStorageIterator) Value() ([]byte, error) {
	key := itr.keys[itr.index]
	return base64.StdEncoding.DecodeString(itr.data[key])
}
