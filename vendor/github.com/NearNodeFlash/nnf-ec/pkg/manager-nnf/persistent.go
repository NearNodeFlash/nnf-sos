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

package nnf

import "github.com/NearNodeFlash/nnf-ec/internal/kvstore"

// Persistent Controller API provides an interface for creating, updating, and deleting persistent objects.
type PersistentControllerApi interface {
	CreatePersistentObject(obj PersistentObjectApi, updateFunc func() error, startingState, endingState uint32) error
	UpdatePersistentObject(obj PersistentObjectApi, updateFunc func() error, startingState, endingState uint32) error
	DeletePersistentObject(obj PersistentObjectApi, deleteFunc func() error, startingState, endingState uint32) error
}

type DefaultPersistentController struct{}

func NewDefaultPersistentController() PersistentControllerApi {
	return &DefaultPersistentController{}
}

type PersistentStoreProvider interface {
	GetStore() *kvstore.Store
}

// Persistent Object API provides interface for creating or updating a persistent object.
type PersistentObjectApi interface {
	GetKey() string
	GetProvider() PersistentStoreProvider

	// GenerateMetadata will generate the metadata when a new persistent object is created. Metadata does not change
	// over the object lifetime.
	GenerateMetadata() ([]byte, error)

	// GenerateStateData is called when any transaction occurs on the persistent object. Usually this is for occurances
	// of Entry State and Exit State, where upon a call to CreatePersistentObject or UpdatePersistentObject, the object
	// can store state data for either the Starting state or Ending state
	GenerateStateData(state uint32) ([]byte, error)

	// Rollback occurs when a call to CreatePersistentObject or UpdatePersistentObject fails. We rollback to the
	// starting state.
	Rollback(startingState uint32) error
}

func (*DefaultPersistentController) CreatePersistentObject(obj PersistentObjectApi, updateFunc func() error, startingState, endingState uint32) error {

	metadata, err := obj.GenerateMetadata()
	if err != nil {
		return err
	}

	ledger, err := obj.GetProvider().GetStore().NewKey(obj.GetKey(), metadata)
	if err != nil {
		return err
	}
	defer ledger.Close()

	return executePersistentObjectTransaction(ledger, obj, updateFunc, startingState, endingState)
}

func (*DefaultPersistentController) UpdatePersistentObject(obj PersistentObjectApi, updateFunc func() error, startingState, endingState uint32) error {

	ledger, err := obj.GetProvider().GetStore().OpenKey(obj.GetKey(), false)
	if err != nil {
		return err
	}
	defer ledger.Close()

	return executePersistentObjectTransaction(ledger, obj, updateFunc, startingState, endingState)
}

func (*DefaultPersistentController) DeletePersistentObject(obj PersistentObjectApi, deleteFunc func() error, startingState, endingState uint32) error {

	ledger, err := obj.GetProvider().GetStore().OpenKey(obj.GetKey(), true)
	if err != nil {
		return err
	}
	defer ledger.Close()

	return executePersistentObjectTransaction(ledger, obj, deleteFunc, startingState, endingState)
}

func executePersistentObjectTransaction(ledger *kvstore.Ledger, obj PersistentObjectApi, updateFunc func() error, startingState, endingState uint32) error {

	data, err := obj.GenerateStateData(startingState)
	if err != nil {
		return err
	}

	if err := ledger.Log(startingState, data); err != nil {
		return err
	}

	if err := updateFunc(); err != nil {
		obj.Rollback(startingState)
		return err
	}

	data, err = obj.GenerateStateData(endingState)
	if err != nil {
		return err
	}

	if err := ledger.Log(endingState, data); err != nil {
		return err
	}

	return nil
}

type MockPersistentController struct{}

func NewMockPersistentController() PersistentControllerApi {
	return &MockPersistentController{}
}

func (*MockPersistentController) CreatePersistentObject(obj PersistentObjectApi, createFunc func() error, startingState, endingState uint32) error {
	return createFunc()
}

func (*MockPersistentController) UpdatePersistentObject(obj PersistentObjectApi, updateFunc func() error, startingState, endingState uint32) error {
	return updateFunc()
}

func (*MockPersistentController) DeletePersistentObject(obj PersistentObjectApi, deleteFunc func() error, startingState, endingState uint32) error {
	return deleteFunc()
}
