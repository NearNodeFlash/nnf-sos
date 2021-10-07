package nnf

import (
	"encoding/json"
	"fmt"

	nvme "stash.us.cray.com/rabsw/nnf-ec/pkg/manager-nvme"
	server "stash.us.cray.com/rabsw/nnf-ec/pkg/manager-server"

	"stash.us.cray.com/rabsw/nnf-ec/internal/kvstore"
	sf "stash.us.cray.com/rabsw/nnf-ec/pkg/rfsf/pkg/models"
)

type StorageGroup struct {
	id          string
	name        string
	description string

	volume *AllocatedVolume

	//
	// Endpoint represents the initiator server which has the Storage Group
	// accessible as a local storage system. Only a single Endpoint can be
	// associated with a Storage Group.
	endpoint *Endpoint

	// Server Storage represents a connection to the physical server endpoint that manages the
	// storage devices. This can be locally managed on the NNF contorller itself, or
	// remotely managed through some magical being not yet determined.
	serverStorage *server.Storage

	// If a file system is present on the parent Storage Pool, this object
	// represents the Exported File Share for this Storage Group, or null
	// if no file-system is present.
	fileShare *FileShare

	storagePool    *StoragePool
	storageService *StorageService
}

func (sg *StorageGroup) OdataId() string {
	return fmt.Sprintf("%s/StorageGroups/%s", sg.storageService.OdataId(), sg.id)
}

func (sg *StorageGroup) OdataIdRef(ref string) sf.OdataV4IdRef {
	return sf.OdataV4IdRef{OdataId: fmt.Sprintf("%s%s", sg.OdataId(), ref)}
}

func (sg *StorageGroup) status() *sf.ResourceStatus {
	state := sg.serverStorage.GetStatus().State()

	health := sf.OK_RH
	if state != sf.ENABLED_RST {
		health = sf.WARNING_RH
	}

	return &sf.ResourceStatus{Health: health, State: state}
}

/*
Persistent Object API
*/

const storageGroupRegistryPrefix = "SG"

const (
	storageGroupCreateStartLogEntryType = iota
	storageGroupCreateCompleteLogEntryType
	storageGroupDeleteStartLogEntryType
	storageGroupDeleteCompleteLogEntryType
)

type storageGroupPersistentMetadata struct {
	Name          string `json:"Name"`
	Description   string `json:"Description"`
	StoragePoolId string `json:"StoragePoolId"`
	EndpointId    string `json:"EndpointId"`
}

func (sg *StorageGroup) GetKey() string                       { return storageGroupRegistryPrefix + sg.id }
func (sg *StorageGroup) GetProvider() PersistentStoreProvider { return sg.storageService }

func (sg *StorageGroup) GenerateMetadata() ([]byte, error) {
	return json.Marshal(storageGroupPersistentMetadata{
		Name:          sg.name,
		Description:   sg.description,
		StoragePoolId: sg.storagePool.id,
		EndpointId:    sg.endpoint.id,
	})
}

func (sg *StorageGroup) GenerateStateData(state uint32) ([]byte, error) {
	// TODO: Should we generate state data after create complete? This could be a list
	// providing volume IDs and their corresponding controller id... But we can always
	// look that information up.
	return nil, nil
}

func (sg *StorageGroup) Rollback(state uint32) error {
	switch state {
	case storageGroupCreateStartLogEntryType:
		// Rollback to a state where no controllers are attached to the storage pool

		for _, v := range sg.storagePool.providingVolumes {
			if err := v.volume.DetachController(sg.endpoint.controllerId); err != nil {
				return err
			}
		}
	}

	return nil
}

// Persistent Object Recovery API

type storageGroupRecoveryRegistry struct {
	storageService *StorageService
}

func NewStorageGroupRecoveryRegistry(s *StorageService) kvstore.Registry {
	return &storageGroupRecoveryRegistry{storageService: s}
}

func (r *storageGroupRecoveryRegistry) Prefix() string { return storageGroupRegistryPrefix }

func (r *storageGroupRecoveryRegistry) NewReplay(id string) kvstore.ReplayHandler {
	return &storageGroupRecoveryReplyHandler{id: id, storageService: r.storageService}
}

type storageGroupRecoveryReplyHandler struct {
	id               string
	lastLogEntryType uint32
	storageGroup     *StorageGroup
	storageService   *StorageService
}

func (rh *storageGroupRecoveryReplyHandler) Metadata(data []byte) error {
	metadata := storageGroupPersistentMetadata{}
	if err := json.Unmarshal(data, &metadata); err != nil {
		return err
	}

	storagePool := rh.storageService.findStoragePool(metadata.StoragePoolId)
	endpoint := rh.storageService.findEndpoint(metadata.EndpointId)

	rh.storageService.groups = append(rh.storageService.groups,
		StorageGroup{
			id:             rh.storageGroup.id,
			endpoint:       endpoint,
			serverStorage:  endpoint.serverCtrl.NewStorage(storagePool.uid),
			storagePool:    storagePool,
			storageService: rh.storageService,
		})

	rh.storageGroup = &rh.storageService.groups[len(rh.storageService.groups)-1]

	return nil
}

func (rh *storageGroupRecoveryReplyHandler) Entry(typ uint32, data []byte) error {
	rh.lastLogEntryType = typ

	return nil
}

func (rh *storageGroupRecoveryReplyHandler) Done() error {

	switch rh.lastLogEntryType {
	case storageGroupCreateStartLogEntryType:
		// In this case the storage group started, but didn't finish. We may have outstanding controllers
		// attached to the endpoint that we don't want to preserve since the client did not get confirmation
		// of the action. We want to detach any controllers for this <Pool, Endpoint> pair.

		for _, volume := range rh.storageGroup.storagePool.providingVolumes {
			if err := nvme.DetachControllers(volume.volume, []uint16{rh.storageGroup.endpoint.controllerId}); err != nil {
				return err
			}
		}

		// TODO: Delete the storage group from the storage service

	case storageGroupCreateCompleteLogEntryType:
		// In this case we've created the storage group, and it exists without error. There is nothing to do
		// here (the storage group is already part of the storage service from the call to Metadata())
	case storageGroupDeleteStartLogEntryType:
		// Delete Start: We started the delete operation but did not complete it. We may have remaining connections
		// to the controllers; and we need to find those that remain, but we can expect some to be missing. This is done
		// by recovering the list of controllers attached to the volume.

	case storageGroupDeleteCompleteLogEntryType:
		// We've deleted all the connections but the key remains. We should garbage collect the key
		// from the store. We don't have a guarentee that the client received the completion for
		// the delete; they may try to delete it again and we should just ignore it.
	}

	return nil
}
