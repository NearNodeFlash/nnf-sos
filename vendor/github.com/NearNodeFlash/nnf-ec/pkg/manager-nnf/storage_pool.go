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

	"github.com/google/uuid"

	"github.com/NearNodeFlash/nnf-ec/pkg/persistent"
	nvme2 "github.com/NearNodeFlash/nnf-ec/internal/switchtec/pkg/nvme"
	nvme "github.com/NearNodeFlash/nnf-ec/pkg/manager-nvme"
	sf "github.com/NearNodeFlash/nnf-ec/pkg/rfsf/pkg/models"
)

type StoragePool struct {
	id          string
	name        string
	description string

	uid    uuid.UUID
	policy AllocationPolicy

	allocatedVolume  AllocatedVolume
	providingVolumes []ProvidingVolume

	storageGroupIds []string
	fileSystemId    string

	storageService *StorageService
}

type AllocatedVolume struct {
	id            string
	capacityBytes uint64
}

type ProvidingVolume struct {
	storage  *nvme.Storage
	volumeId string
}

func (p *StoragePool) GetCapacityBytes() (capacityBytes uint64) {
	for _, pv := range p.providingVolumes {
		capacityBytes += pv.storage.FindVolume(pv.volumeId).GetCapaityBytes()
	}
	return capacityBytes
}

func (p *StoragePool) OdataId() string {
	return fmt.Sprintf("%s/StoragePools/%s", p.storageService.OdataId(), p.id)
}

func (p *StoragePool) OdataIdRef(ref string) sf.OdataV4IdRef {
	return sf.OdataV4IdRef{OdataId: fmt.Sprintf("%s%s", p.OdataId(), ref)}
}

func (p *StoragePool) isCapacitySource(capacitySourceId string) bool {
	return capacitySourceId == DefaultStoragePoolCapacitySourceId
}

func (p *StoragePool) isAllocatedVolume(volumeId string) bool {
	return volumeId == DefaultAllocatedVolumeId
}

func (p *StoragePool) capacitySourcesGet() []sf.CapacityCapacitySource {
	return []sf.CapacityCapacitySource{
		{
			OdataId:   p.OdataId() + "/CapacitySources",
			OdataType: "#CapacitySource.v1_0_0.CapacitySource",
			Name:      "Capacity Source",
			Id:        DefaultStoragePoolCapacitySourceId,

			ProvidedCapacity: sf.CapacityV120Capacity{
				// TODO
			},

			ProvidingVolumes: p.OdataIdRef(fmt.Sprintf("/CapacitySources/%s/ProvidingVolumes", DefaultStoragePoolCapacitySourceId)),
		},
	}
}

func (p *StoragePool) findStorageGroupByEndpoint(endpoint *Endpoint) *StorageGroup {
	for _, sgid := range p.storageGroupIds {
		sg := p.storageService.findStorageGroup(sgid)
		if sg != nil && sg.endpoint.id == endpoint.id {
			return sg
		}
	}

	return nil
}

func (p *StoragePool) recoverVolumes(volumes []storagePoolPersistentVolumeInfo, ignoreErrors bool) error {

	for _, volume := range volumes {
		storage := p.storageService.findStorage(volume.SerialNumber)

		if storage != nil {
			volume, err := storage.FindVolumeByNamespaceId(volume.NamespaceId)
			if err != nil {
				if ignoreErrors {
					continue
				}
				
				return err
			}

			p.providingVolumes = append(p.providingVolumes, ProvidingVolume{
				storage:  storage,
				volumeId: volume.Id(),
			})
		}
	}

	p.allocatedVolume = AllocatedVolume{
		id:            DefaultAllocatedVolumeId,
		capacityBytes: p.GetCapacityBytes(),
	}

	return nil
}

func (p *StoragePool) deallocateVolumes() error {
	for _, pv := range p.providingVolumes {
		if err := nvme.DeleteVolume(pv.storage.FindVolume(pv.volumeId)); err != nil {
			return err
		}
	}

	return nil
}

// Persistent Object API

const storagePoolRegistryPrefix = "SP"

const (
	// Allocation Log Entry is recorded after the storage pool successfully allocates the backing storage resources (i.e. NVMe Namespaces)
	storagePoolStorageCreateStartLogEntryType uint32 = iota
	storagePoolStorageCreateCompleteLogEntryType
	storagePoolStorageDeleteStartLogEntryType
	storagePoolStorageDeleteCompleteLogEntryType
)

type storagePoolPersistentMetadata struct {
	Name        string `json:"Name,omitempty"`
	Description string `json:"Description,omitempty"`
	Uid         string `json:"Uid"`
}

type storagePoolPersistentCreateCompleteLogEntry struct {
	Volumes       []storagePoolPersistentVolumeInfo `json:"Volumes,omitempty"`
	CapacityBytes uint64                            `json:"CapacityBytes"`
}

type storagePoolPersistentVolumeInfo struct {
	SerialNumber string                    `json:"SerialNumber"`
	NamespaceId  nvme2.NamespaceIdentifier `json:"NamespaceId"`
}

func (p *StoragePool) GetKey() string                       { return storagePoolRegistryPrefix + p.id }
func (p *StoragePool) GetProvider() PersistentStoreProvider { return p.storageService }

func (p *StoragePool) GenerateMetadata() ([]byte, error) {
	return json.Marshal(storagePoolPersistentMetadata{
		Name:        p.name,
		Description: p.description,
		Uid:         p.uid.String(),
	})
}

func (p *StoragePool) GenerateStateData(state uint32) ([]byte, error) {
	switch state {
	case storagePoolStorageCreateCompleteLogEntryType:
		entry := storagePoolPersistentCreateCompleteLogEntry{
			Volumes:       make([]storagePoolPersistentVolumeInfo, len(p.providingVolumes)),
			CapacityBytes: p.GetCapacityBytes(),
		}

		for idx, pv := range p.providingVolumes {
			entry.Volumes[idx] = storagePoolPersistentVolumeInfo{
				SerialNumber: pv.storage.SerialNumber(),
				NamespaceId:  pv.storage.FindVolume(pv.volumeId).GetNamespaceId(),
			}
		}

		return json.Marshal(entry)
	}

	return nil, nil
}

func (p *StoragePool) Rollback(state uint32) error {
	switch state {
	case storagePoolStorageCreateStartLogEntryType:
		if err := p.deallocateVolumes(); err != nil {
			return err
		}

		s := p.storageService
		for idx, pool := range s.pools {
			if pool.id == p.id {
				copy(s.pools[idx:], s.pools[idx+1:])
				s.pools = s.pools[:len(s.pools)-1]
			}
		}
	}

	return nil
}

// Persistent Object Recovery API

type storagePoolRecoveryRegistry struct {
	storageService *StorageService
}

func NewStoragePoolRecoveryRegistry(s *StorageService) persistent.Registry {
	return &storagePoolRecoveryRegistry{storageService: s}
}

func (*storagePoolRecoveryRegistry) Prefix() string { return storagePoolRegistryPrefix }

func (r *storagePoolRecoveryRegistry) NewReplay(id string) persistent.ReplayHandler {
	return &storagePoolRecoveryReplayHandler{storageService: r.storageService, storagePool: StoragePool{id: id}}
}

// The Storage Pool Recovery Replay Handler accepts TLVs from the kvstore to
// replay the actions that occurred on the storage pool.
type storagePoolRecoveryReplayHandler struct {
	// Reference to the storage service that manages this reply
	storageService *StorageService

	// Last log entry recorded in the log. This represents state of the storage lifetime
	lastLogEntryType uint32

	// The recovered storage pool. Only valid if last log entry > storagePoolStorageCreateStartLogEntryType
	storagePool StoragePool

	// List of volume information associated with the storage pool. Only valid if last log entry > storagePoolStorageCreateCompleteLogEntryType
	volumes []storagePoolPersistentVolumeInfo
}

func (rh *storagePoolRecoveryReplayHandler) Metadata(data []byte) error {
	metadata := storagePoolPersistentMetadata{}
	if err := json.Unmarshal(data, &metadata); err != nil {
		return err
	}

	rh.storageService.createStoragePool(rh.storagePool.id, metadata.Name, metadata.Description, uuid.MustParse(metadata.Uid), nil)
	rh.storagePool = StoragePool{
		id:          rh.storagePool.id,
		name:        metadata.Name,
		description: metadata.Description,
		uid:         uuid.MustParse(metadata.Uid),
		allocatedVolume: AllocatedVolume{
			id:            DefaultAllocatedVolumeId,
			capacityBytes: 0,
		},
		storageService: rh.storageService,
	}

	return nil
}

func (rh *storagePoolRecoveryReplayHandler) Entry(typ uint32, data []byte) error {

	// Save the last log entry recorded in the ledger.
	rh.lastLogEntryType = typ

	switch typ {
	case storagePoolStorageCreateCompleteLogEntryType:
		// We have fully initialized the storage pool and all providing volumes have been allocated. Unpack
		// the data and fill in the storage pool's list of volumes.
		entry := storagePoolPersistentCreateCompleteLogEntry{}
		if err := json.Unmarshal(data, &entry); err != nil {
			return err
		}

		rh.volumes = entry.Volumes
	}

	return nil
}

func (rh *storagePoolRecoveryReplayHandler) Done() (bool, error) {

	switch rh.lastLogEntryType {
	case storagePoolStorageCreateStartLogEntryType:
		// In this case the storage pool started, but didn't finish. We may have outstanding namespaces
		// that should be cleaned up. Since we don't know _which_ namespaces are unassigned at this
		// point in time (as there may be other storage pools that will claim the namespaces), we will
		// defer to the storage service to automatically clean up abandoned namespaces after all
		// storage pools have been initialized.

	case storagePoolStorageCreateCompleteLogEntryType, storagePoolStorageDeleteStartLogEntryType:
		// Case 1. Create Complete: In this case, we've fully created the storage pool and it should be
		// fully recoverable and place back in use.

		// Case 2. Delete Start: We started a delete but it did not finish. This means the storage pool
		// still exists, and its volumes are unknown. Here we try to recover the volumes, but ignore any
		// errors as the volume might be deleted. The client should retry the delete, at which point we
		// will delete any remaining volumes

		// Recover the namespaces that make up this storage pool
		if err := rh.storagePool.recoverVolumes(rh.volumes, rh.lastLogEntryType == storagePoolStorageDeleteStartLogEntryType); err != nil {
			return false, err
		}

		rh.storageService.pools = append(rh.storageService.pools, rh.storagePool)

	case storagePoolStorageDeleteCompleteLogEntryType:
		// We've deleted all the volumes and the storage pool, but failed to delete the key. The client
		// should retry the delete, at which point we can delete the storage pool and the entry in
		// the store.

		rh.storageService.pools = append(rh.storageService.pools, rh.storagePool)
	}

	return false, nil
}
