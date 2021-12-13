package nnf

import "github.hpe.com/hpe/hpc-rabsw-nnf-ec/internal/kvstore"

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
	// of Entry State and Exit State, where upon a call to NewPersistentObject or UpdatePersistentObject, the object
	// can store state data for either the Starting state or Ending state
	GenerateStateData(state uint32) ([]byte, error)

	// Rollback occurs when a call to NewPersistentObject or UpdatePersistentObject fails. We rollback to the
	// starting state.
	Rollback(startingState uint32) error
}

func NewPersistentObject(obj PersistentObjectApi, updateFunc func() error, startingState, endingState uint32) error {

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

func UpdatePersistentObject(obj PersistentObjectApi, updateFunc func() error, startingState, endingState uint32) error {

	ledger, err := obj.GetProvider().GetStore().OpenKey(obj.GetKey(), false)
	if err != nil {
		return err
	}
	defer ledger.Close()

	return executePersistentObjectTransaction(ledger, obj, updateFunc, startingState, endingState)
}

func DeletePersistentObject(obj PersistentObjectApi, deleteFunc func() error, startingState, endingState uint32) error {

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
