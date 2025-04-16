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

import (
	"encoding/json"
	"fmt"

	server "github.com/NearNodeFlash/nnf-ec/pkg/manager-server"
	"github.com/NearNodeFlash/nnf-ec/pkg/persistent"
	sf "github.com/NearNodeFlash/nnf-ec/pkg/rfsf/pkg/models"
)

// StorageGroup represents a mapping between a storage pool and an endpoint
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
	// represents the Exported File Share for this Storage Group, or empty
	// if no file-system is present.
	fileShareId string

	storagePoolId string

	storageService *StorageService
}

func (sg *StorageGroup) OdataId() string {
	return fmt.Sprintf("%s/StorageGroups/%s", sg.storageService.OdataId(), sg.id)
}

func (sg *StorageGroup) OdataIdRef(ref string) sf.OdataV4IdRef {
	return sf.OdataV4IdRef{OdataId: fmt.Sprintf("%s%s", sg.OdataId(), ref)}
}

func (sg *StorageGroup) status() sf.ResourceStatus {
	status, _ := sg.serverStorage.GetStatus()

	health := sf.OK_RH
	if status.State() != sf.ENABLED_RST {
		health = sf.WARNING_RH
	}

	return sf.ResourceStatus{Health: health, State: status.State()}
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

// storageGroupLogEntryTypeNames maps the numeric log entry type constants to their string representations
var storageGroupLogEntryTypeNames = []string{
	"storageGroupCreateStartLogEntryType",
	"storageGroupCreateCompleteLogEntryType",
	"storageGroupDeleteStartLogEntryType",
	"storageGroupDeleteCompleteLogEntryType",
}

// LogEntryTypeToString converts a numeric log entry type to its string representation
func LogEntryTypeToString(entryType uint32) string {
	if int(entryType) < len(storageGroupLogEntryTypeNames) {
		return storageGroupLogEntryTypeNames[entryType]
	}
	return fmt.Sprintf("Unknown(%d)", entryType)
}

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
		StoragePoolId: sg.storagePoolId,
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

		sp := sg.storageService.findStoragePool(sg.storagePoolId)
		if sp == nil {
			return fmt.Errorf("Rollback Storage Group %s Create Start: Storage Pool %s not found", sg.id, sg.storagePoolId)
		}

		for _, pv := range sp.providingVolumes {
			volume := pv.Storage.FindVolume(pv.VolumeId)
			if volume == nil {
				return fmt.Errorf("Rollback Storage Group %s Create Start: Volume %s not found", sg.id, pv.VolumeId)
			}

			if err := volume.DetachController(sg.endpoint.controllerId); err != nil {
				return err
			}
		}

	case storageGroupDeleteStartLogEntryType:
		// Rollback to a state where all controllers are detached from the storage pool

		sp := sg.storageService.findStoragePool(sg.storagePoolId)
		if sp == nil {
			return fmt.Errorf("Rollback Storage Group %s Delete Start: Storage Pool %s not found", sg.id, sg.storagePoolId)
		}

		for _, pv := range sp.providingVolumes {
			volume := pv.Storage.FindVolume(pv.VolumeId)
			if volume == nil {
				return fmt.Errorf("Rollback Storage Group %s Delete Start: Volume %s not found", sg.id, pv.VolumeId)
			}

			if err := volume.AttachController(sg.endpoint.controllerId); err != nil {
				return err
			}
		}
	}

	return nil
}

// recoverPool iterates through each volume in the associated storage pool and
// lists the controllers that volumes are attached to. It helps ensure that all
// necessary controller attachments are properly recovered during system recovery.
func (sg *StorageGroup) recoverPool() error {
	log := sg.storageService.log.WithValues("storageGroup", sg.id, "storagePool", sg.storagePoolId, "endpoint", sg.endpoint.id)

	sp := sg.storageService.findStoragePool(sg.storagePoolId)
	if sp == nil {
		return fmt.Errorf("recover storage group pool: storage pool %s not found", sg.storagePoolId)
	}

	log.V(1).Info("recover storage group")

	// Check each providing volume in the storage pool
	for _, pv := range sp.providingVolumes {
		volume := pv.Storage.FindVolume(pv.VolumeId)
		if volume == nil {
			log.Error(fmt.Errorf("volume not found"),
				"failed to recover volume",
				"storageGroup", sg.id,
				"volumeId", pv.VolumeId)
			continue
		}

		// Ensure volume is attached to the endpoint
		if err := volume.AttachControllerIfUnattached(sg.endpoint.controllerId); err != nil {
			return err
		}
	}

	return nil
}

// Persistent Object Recovery API

type storageGroupRecoveryRegistry struct {
	storageService *StorageService
}

// NewStorageGroupRecoveryRegistry creates a new registry for recovering storage groups
func NewStorageGroupRecoveryRegistry(s *StorageService) persistent.Registry {
	return &storageGroupRecoveryRegistry{storageService: s}
}

func (r *storageGroupRecoveryRegistry) Prefix() string { return storageGroupRegistryPrefix }

func (r *storageGroupRecoveryRegistry) NewReplay(id string) persistent.ReplayHandler {
	return &storageGroupRecoveryReplyHandler{id: id, storageService: r.storageService}
}

type storageGroupRecoveryReplyHandler struct {
	id               string
	lastLogEntryType uint32
	storageService   *StorageService
}

func (rh *storageGroupRecoveryReplyHandler) Metadata(data []byte) error {
	metadata := storageGroupPersistentMetadata{}
	if err := json.Unmarshal(data, &metadata); err != nil {
		return err
	}

	storagePool := rh.storageService.findStoragePool(metadata.StoragePoolId)
	if storagePool == nil {
		return fmt.Errorf("storagePool %s not found", metadata.StoragePoolId)
	}
	endpoint := rh.storageService.findEndpoint(metadata.EndpointId)
	if endpoint == nil {
		return fmt.Errorf("endpoint %s not found", metadata.EndpointId)
	}

	rh.storageService.createStorageGroup(rh.id, storagePool, endpoint)

	return nil
}

func (rh *storageGroupRecoveryReplyHandler) Entry(typ uint32, data []byte) error {
	rh.lastLogEntryType = typ

	return nil
}

func (rh *storageGroupRecoveryReplyHandler) Done() (bool, error) {

	sg := rh.storageService.findStorageGroup(rh.id)
	if sg == nil {
		return true, fmt.Errorf("Storage Group Recovery: Storage Group %s not found", rh.id)
	}

	switch rh.lastLogEntryType {
	case storageGroupCreateStartLogEntryType:
		// In this case, the storage group started but didn't finish. We may have outstanding controllers
		// attached to the endpoint that we don't want to preserve since the client did not get confirmation
		// of the action. We want to detach any controllers for this <Pool, Endpoint> pair.

		sp := rh.storageService.findStoragePool(sg.storagePoolId)
		if sp == nil {
			return false, fmt.Errorf("Storage Group %s Recover: Storage Pool %s not found", sg.id, sg.storagePoolId)
		}
		for _, pv := range sp.providingVolumes {
			volume := pv.Storage.FindVolume(pv.VolumeId)
			if volume == nil {
				return false, fmt.Errorf("Storage Group %s Recover: Volume %s not found", sg.id, pv.VolumeId)
			}

			if err := volume.DetachController(sg.endpoint.controllerId); err != nil {
				return false, err
			}
		}

		sp.storageService.deleteStorageGroup(sg)

		return true, nil

	case storageGroupCreateCompleteLogEntryType:
		// In this case, we've created the storage group, and it exists without error. There is nothing to do
		// here (the storage group is already part of the storage service from the call to Metadata()).
		// The attachment to a particular endpoint is persistently maintained in the NVMe controller, so
		// for namespaces that formerly were attached, there is nothing to do.

		// Verify all volumes are attached to the endpoint
		if err := sg.recoverPool(); err != nil {
			return false, err
		}

	case storageGroupDeleteStartLogEntryType:
		// Delete Start: We started the delete operation but did not complete it. We may have remaining connections
		// to the controllers, and we need to find those that remain, but we can expect some to be missing. This is done
		// by recovering the list of controllers attached to the volume.
	case storageGroupDeleteCompleteLogEntryType:
		// We've deleted all the connections, but the key remains. We should garbage collect the key
		// from the store. We don't have a guarantee that the client received the completion for
		// the delete; they may try to delete it again, and we should just ignore it.
	}

	rh.storageService.log.V(2).Info("recover storage group complete",
		"storageGroup", sg.id,
		"lastLogEntryType", LogEntryTypeToString(rh.lastLogEntryType))
	return false, nil
}
