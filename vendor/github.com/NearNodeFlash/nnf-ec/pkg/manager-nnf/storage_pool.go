/*
 * Copyright 2020-2025 Hewlett Packard Enterprise Development LP
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

	nvme2 "github.com/NearNodeFlash/nnf-ec/internal/switchtec/pkg/nvme"
	event "github.com/NearNodeFlash/nnf-ec/pkg/manager-event"
	msgreg "github.com/NearNodeFlash/nnf-ec/pkg/manager-message-registry/registries"
	nvme "github.com/NearNodeFlash/nnf-ec/pkg/manager-nvme"
	"github.com/NearNodeFlash/nnf-ec/pkg/persistent"
	sf "github.com/NearNodeFlash/nnf-ec/pkg/rfsf/pkg/models"
)

// StoragePool represents a logical grouping of storage capacity that can be allocated and managed
// as a unit within the storage service.
type StoragePool struct {
	id          string
	name        string
	description string

	uid            uuid.UUID
	policy         AllocationPolicy
	volumeCapacity uint64

	allocatedVolume  AllocatedVolume
	providingVolumes []nvme.ProvidingVolume
	missingVolumes   []storagePoolPersistentVolumeInfo

	// Original persistent volume information from KV store
	persistedVolumes []storagePoolPersistentVolumeInfo

	storageGroupIds []string
	fileSystemId    string

	storageService *StorageService
}

// AllocatedVolume represents a volume that has been allocated in a storage pool
type AllocatedVolume struct {
	id            string
	capacityBytes uint64
}

// GetCapacityBytes - sum up the capacity of the volume recording the maximum volume size in the process
func (p *StoragePool) GetCapacityBytes() (capacityBytes uint64) {
	p.volumeCapacity = uint64(0)
	for _, pv := range p.providingVolumes {
		capacity := pv.Storage.FindVolume(pv.VolumeId).GetCapacityBytes()
		capacityBytes += capacity

		// Missing volumes will be allocated with the maximum volume capacity of the providing volumes
		p.volumeCapacity = max(p.volumeCapacity, capacity)
	}

	// Add on the capacity of the missing volumes
	capacityBytes += p.volumeCapacity * uint64(len(p.missingVolumes))
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

// recoverVolumes attempts to restore the state of volumes in the storage pool based on
// the provided slice of persisted volume information. It updates the pool's internal
// lists of persisted, missing, and providing volumes by:
//   - Skipping and marking as missing any volumes with an invalid namespace ID.
//   - Attempting to locate the corresponding storage device and namespace for each volume.
//   - Marking volumes as missing if the storage device or namespace cannot be found.
//   - Adding successfully located volumes to the providingVolumes list.
//
// Finally, it updates the allocatedVolume field to reflect the current pool capacity.
func (p *StoragePool) recoverVolumes(volumes []storagePoolPersistentVolumeInfo) error {
	log := p.storageService.log.WithValues(storagePoolIdKey, p.id)
	log.Info("recover volumes")

	// Store the persisted volumes information for later use
	p.persistedVolumes = make([]storagePoolPersistentVolumeInfo, len(volumes))
	copy(p.persistedVolumes, volumes)

	for _, volumeInfo := range volumes {
		log := log.WithValues("serialNumber", volumeInfo.SerialNumber, "namespaceId", volumeInfo.NamespaceID)

		// Consider volumes that have been invalidated (marked with invalid namespace ID) as missing.
		// They need a replacement.
		if volumeInfo.NamespaceID == invalidNamespaceID {
			log.V(2).Info("skipping invalidated volume during recovery")
			p.missingVolumes = append(p.missingVolumes, volumeInfo)
			continue
		}

		// Locate the NVMe Storage device by Serial Number
		storage := p.storageService.findStorage(volumeInfo.SerialNumber)
		if storage == nil {
			log.Info("storage device not found")
			p.missingVolumes = append(p.missingVolumes, volumeInfo)
			continue
		}

		// Locate the Volume by Namespace ID
		volumeID := uint32(volumeInfo.NamespaceID)
		_, err := storage.FindVolumeByNamespaceId(volumeInfo.NamespaceID)
		if err != nil {
			log.Error(err, "namespace not found", "slot", storage.Slot())
			p.missingVolumes = append(p.missingVolumes, volumeInfo)
			continue
		}

		p.providingVolumes = append(p.providingVolumes, nvme.ProvidingVolume{
			Storage:  storage,
			VolumeId: fmt.Sprintf("%d", volumeID),
		})
	}

	p.allocatedVolume = AllocatedVolume{
		id:            DefaultAllocatedVolumeId,
		capacityBytes: p.GetCapacityBytes(),
	}

	return nil
}

// checkVolumes validates the current state of volumes in the storage pool by rescanning
// associated storage devices and updating the pool's volume tracking lists.
//
// This method performs the following operations:
//  1. Rescans all storage devices that should contain volumes for this pool to refresh
//     their namespace information and detect any changes in device state
//  2. Clears and rebuilds both the missingVolumes and providingVolumes lists to ensure
//     they accurately reflect the current state
//  3. Attempts to recover volumes from persistent storage information, identifying
//     which volumes are still available and which are missing or invalid
//  4. Updates the pool's internal state to reflect any volumes that may have become
//     unavailable due to device failures, disconnections, or other issues
//
// The method is typically called during pool initialization, recovery operations,
// or when storage device states may have changed. It ensures the pool maintains
// an accurate view of its available storage resources.
func (p *StoragePool) checkVolumes() error {
	log := p.storageService.log.WithValues(storagePoolIdKey, p.id)
	log.Info("check volumes")

	// Rescan the storages to ensure our namespace information is up to date
	volumesInPool := make([]storagePoolPersistentVolumeInfo, len(p.persistedVolumes))
	copy(volumesInPool, p.persistedVolumes)

	for _, pv := range volumesInPool {
		log := log.WithValues("serialNumber", pv.SerialNumber, "namespaceId", pv.NamespaceID)
		storage := p.storageService.findStorage(pv.SerialNumber)
		if storage == nil {
			log.Info("storage device not found")
			continue
		}
		storage.Rescan()
	}

	p.missingVolumes = p.missingVolumes[:0]     // Clear the missing volumes list
	p.providingVolumes = p.providingVolumes[:0] // Clear the providing volumes list

	if err := p.recoverVolumes(volumesInPool); err != nil {
		log.Error(err, "Failed to recover volumes")
		return err
	}

	return nil
}

// Replace each missing volume with new volume on available Storage
func (p *StoragePool) replaceMissingVolumes() error {
	log := p.storageService.log.WithValues(storagePoolIdKey, p.id)
	log.Info("replace missing volumes")

	logMissingVolumesFunc := func() {
		log.V(2).Info("missing volumes", "missingVolumeCount", len(p.missingVolumes), "volumes", p.missingVolumes)
	}
	defer logMissingVolumesFunc()

	// Anything to do?
	if len(p.missingVolumes) == 0 {
		return nil
	}

	// Attempt to locate a storage device that is not providing a volume in this pool,
	// and create a new volume on it to replace the missing volume.
	// This is a best effort attempt to replace the missing volume
	// and may not be successful if there are no available storage devices,
	// or if the storage device is not able to provide a volume.
	// This is not a failure condition, but rather a best effort attempt
	// to replace the missing volume.
	// The caller should check the missing volumes list to determine
	// if there are any missing volumes that were not replaced
	unusedStorages := p.locateUnusedStorage()
	if len(unusedStorages) == 0 {
		return fmt.Errorf("Unable to find unused storage")
	}

	// It's all or nothing, we need to replace all the missing volumes or none.
	if len(unusedStorages) < len(p.missingVolumes) {
		log.V(2).Info("not enough unused storage", "unusedStorageCount", len(unusedStorages), "missingVolumeCount", len(p.missingVolumes))
		return fmt.Errorf("Not enough unused storage to replace missing volumes")
	}

	// Remove excess storages
	if len(unusedStorages) > len(p.missingVolumes) {
		unusedStorages = unusedStorages[:len(p.missingVolumes)]
	}

	// Replace each missing volume with a new volume on the unused storage
	for idx, missingVolume := range p.missingVolumes {
		log := log.WithValues("missingVolume", missingVolume)

		log.Info("replace missing volume")
		storage := unusedStorages[idx]

		// Add safety check before creating volume
		if storage == nil || !storage.IsEnabled() || storageDeviceIsNil(storage) {
			log.Error(fmt.Errorf("storage unavailable"), "Cannot create volume: storage is nil, not enabled, or device is nil")
			return fmt.Errorf("Cannot create volume: storage unavailable for missing volume %v", missingVolume)
		}

		volume, err := nvme.CreateVolume(storage, p.volumeCapacity)
		if err != nil {
			log.Error(err, "Failed to create replacement volume")
			return fmt.Errorf("Failed to create volume: %v", err)
		}
		pv := nvme.ProvidingVolume{
			Storage:  storage,
			VolumeId: volume.Id(),
		}
		log.Info("created replacement volume", "storage", storage.SerialNumber(), "volume", pv.VolumeId)

		p.providingVolumes = append(p.providingVolumes, pv)

		event.EventManager.PublishResourceEvent(msgreg.StoragePoolPatchedNnf(
			p.id,
			missingVolume.SerialNumber,
			volume.Id(),
			pv.Storage.SerialNumber(),
			pv.VolumeId), p)

		// Invalidate this volume in any other storage pools to prevent reuse
		if err := p.invalidateVolumeInOtherPools(pv.Storage.SerialNumber(), pv.Storage.FindVolume(pv.VolumeId).GetNamespaceId()); err != nil {
			log.Error(err, "Failed to invalidate volume in other storage pools", "serialNumber", pv.Storage.SerialNumber(), "namespaceId", pv.Storage.FindVolume(pv.VolumeId).GetNamespaceId())
			// This is not a fatal error, continue with the replacement
		}
	}

	// We've replaced all the missing volumes, so clear the list
	p.missingVolumes = nil

	return nil
}

// invalidateVolumeInOtherPools finds and invalidates the specified volume in any other storage pools
// to prevent reuse of the same volume in multiple pools. It replaces the namespace ID with a
// nonsense value (0xFFFFFFFF) in the persistent volume information of other pools.
func (p *StoragePool) invalidateVolumeInOtherPools(serialNumber string, namespaceID nvme2.NamespaceIdentifier) error {
	log := p.storageService.log.WithValues(storagePoolIdKey, p.id, "targetSerialNumber", serialNumber, "targetNamespaceId", namespaceID)
	log.V(2).Info("invalidating volume in other storage pools")

	invalidatedPools := 0

	// Check all other storage pools in the service
	for i := range p.storageService.pools {
		otherPool := &p.storageService.pools[i]
		if otherPool.id == p.id {
			continue // Skip self
		}

		poolModified := false

		// Check persistent volumes
		for j := range otherPool.persistedVolumes {
			persistentVol := &otherPool.persistedVolumes[j]
			if persistentVol.SerialNumber == serialNumber && persistentVol.NamespaceID == namespaceID {
				log.Info("invalidating persistent volume in storage pool", "otherPoolId", otherPool.id, "originalNamespaceId", persistentVol.NamespaceID)

				// Replace with a nonsense namespace ID to prevent reuse
				persistentVol.NamespaceID = invalidNamespaceID
				poolModified = true
			}
		}

		if poolModified {
			invalidatedPools++
		}
	}

	log.V(2).Info("volume invalidation complete", "invalidatedPools", invalidatedPools)
	return nil
}

// Locate Storages not providing a volume for this pool
func (p *StoragePool) locateUnusedStorage() []*nvme.Storage {
	log := p.storageService.log.WithValues(storagePoolIdKey, p.id)
	log.Info("locate unused storage")

	var unusedStorages []*nvme.Storage

	// Return the first storage that is not providing a volume, if any
	for _, s := range nvme.GetStorage() {
		if s.SerialNumber() == "" { // Skip unpopulated Storage slot
			continue
		}

		candidate := s
		for _, pv := range p.providingVolumes {
			if s.SerialNumber() == pv.Storage.SerialNumber() {
				candidate = nil
				break
			}
		}

		if candidate != nil {
			log.V(3).Info("found a storage", "slot", candidate.Slot())
			unusedStorages = append(unusedStorages, candidate)
		}
	}

	return unusedStorages
}

func (p *StoragePool) deallocateVolumes() error {
	log := p.storageService.log.WithValues(storagePoolIdKey, p.id)
	// In order to speed up deleting volumes, we format them first. Format runs asynchronously, so after
	// each format call, wait for completion before deleting the volume.

	runOnProvidingVolumes := func(volFn func(*nvme.Volume) error) error {
		for _, pv := range p.providingVolumes {
			volume := pv.Storage.FindVolume(pv.VolumeId)
			if volume == nil {
				err := fmt.Errorf("Volume not found")
				log.Error(err, "StoragePool deallocateVolumes volume not found", "volume", pv.VolumeId)
				continue
			}

			if err := volFn(volume); err != nil {
				log.Error(err, "Volume function failed", "function", volFn, "volume", pv.VolumeId)
				continue
			}
		}

		return nil
	}

	log.V(3).Info("Formatting volumes")
	if err := runOnProvidingVolumes(func(v *nvme.Volume) error { return v.Format() }); err != nil {
		return fmt.Errorf("Failed to format volumes: %v", err)
	}

	log.V(3).Info("Wait for format complete")
	if err := runOnProvidingVolumes(func(v *nvme.Volume) error { return v.WaitFormatComplete() }); err != nil {
		return fmt.Errorf("Failed to wait on format completions: %v", err)
	}

	log.V(3).Info("Deleting volumes")
	if err := runOnProvidingVolumes(func(v *nvme.Volume) error { return v.Delete() }); err != nil {
		return fmt.Errorf("Failed to delete volumes: %v", err)
	}

	return nil
}

// Persistent Object API

const storagePoolRegistryPrefix = "SP"

// Invalid namespace ID used to mark volumes as unusable
const invalidNamespaceID = nvme2.NamespaceIdentifier(0xFFFFFFFF)

const (
	// Allocation Log Entry is recorded after the storage pool successfully allocates the backing storage resources (i.e. NVMe Namespaces)
	storagePoolStorageCreateStartLogEntryType uint32 = iota
	storagePoolStorageCreateCompleteLogEntryType
	storagePoolStorageDeleteStartLogEntryType
	storagePoolStorageDeleteCompleteLogEntryType
	storagePoolStorageUpdateStartLogEntryType
	storagePoolStorageUpdateCompleteLogEntryType
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

type storagePoolPersistentUpdateCompleteLogEntry struct {
	Volumes       []storagePoolPersistentVolumeInfo `json:"Volumes,omitempty"`
	CapacityBytes uint64                            `json:"CapacityBytes"`
}

type storagePoolPersistentVolumeInfo struct {
	SerialNumber string                    `json:"SerialNumber"`
	NamespaceID  nvme2.NamespaceIdentifier `json:"NamespaceId"`
}

// GetKey returns the unique key for this storage pool in the persistent store
func (p *StoragePool) GetKey() string { return storagePoolRegistryPrefix + p.id }

// GetProvider returns the persistent store provider for this storage pool
func (p *StoragePool) GetProvider() PersistentStoreProvider { return p.storageService }

// GenerateMetadata serializes the storage pool's metadata to JSON for persistence
func (p *StoragePool) GenerateMetadata() ([]byte, error) {
	return json.Marshal(storagePoolPersistentMetadata{
		Name:        p.name,
		Description: p.description,
		Uid:         p.uid.String(),
	})
}

// GenerateStateData serializes storage pool state information to JSON based on the state type.
// For state type storagePoolStorageCreateCompleteLogEntryType, it creates a persistent log entry
// containing information about provided volumes (serial numbers and namespace IDs) and pool capacity.
//
// Parameters:
//   - state: uint32 identifier for the type of state data to generate
//
// Returns:
//   - []byte: serialized state data in JSON format
//   - error: any error encountered during serialization, or nil on success
//
// If the state type is not recognized, it returns nil, nil.
func (p *StoragePool) GenerateStateData(state uint32) ([]byte, error) {
	switch state {
	case storagePoolStorageCreateCompleteLogEntryType, storagePoolStorageUpdateCompleteLogEntryType:
		entry := storagePoolPersistentCreateCompleteLogEntry{
			Volumes:       make([]storagePoolPersistentVolumeInfo, len(p.providingVolumes)),
			CapacityBytes: p.GetCapacityBytes(),
		}

		for idx, pv := range p.providingVolumes {
			entry.Volumes[idx] = storagePoolPersistentVolumeInfo{
				SerialNumber: pv.Storage.SerialNumber(),
				NamespaceID:  pv.Storage.FindVolume(pv.VolumeId).GetNamespaceId(),
			}
		}

		// Store the persistent volumes information for later use
		p.persistedVolumes = make([]storagePoolPersistentVolumeInfo, len(entry.Volumes))
		copy(p.persistedVolumes, entry.Volumes)

		return json.Marshal(entry)
	}

	return nil, nil
}

// Rollback is called when a persistent object operation fails and needs to be rolled back.
// It handles the rollback of the storage pool based on the provided state type.
//
// For the state type storagePoolStorageCreateStartLogEntryType, it deallocates volumes and removes
// the storage pool from the storage service's list of pools.
// It returns an error if the rollback operation fails.
// If the state type is not recognized, it returns nil.
// Parameters:
//   - state: uint32 identifier for the type of state data to roll back
//
// Returns:
//   - error: any error encountered during the rollback, or nil on success
//
// If the state type is not recognized, it returns nil.
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

// NewStoragePoolRecoveryRegistry creates a new registry for storage pool recovery operations
func NewStoragePoolRecoveryRegistry(s *StorageService) persistent.Registry {
	return &storagePoolRecoveryRegistry{storageService: s}
}

func (*storagePoolRecoveryRegistry) Prefix() string { return storagePoolRegistryPrefix }

func (r *storagePoolRecoveryRegistry) NewReplay(id string) persistent.ReplayHandler {
	return &storagePoolRecoveryReplayHandler{storageService: r.storageService, id: id}
}

// The Storage Pool Recovery Replay Handler accepts TLVs from the kvstore to
// replay the actions that occurred on the storage pool.
type storagePoolRecoveryReplayHandler struct {
	// Reference to the storage service that manages this reply
	storageService *StorageService

	// The storage pool ID
	id string

	// Last log entry recorded in the log. This represents state of the storage lifetime
	lastLogEntryType uint32

	// The recovered storage pool. Only valid if last log entry > storagePoolStorageCreateStartLogEntryType
	storagePool *StoragePool

	// List of volume information associated with the storage pool. Only valid if last log entry > storagePoolStorageCreateCompleteLogEntryType
	volumes []storagePoolPersistentVolumeInfo
}

func (rh *storagePoolRecoveryReplayHandler) Metadata(data []byte) error {
	metadata := storagePoolPersistentMetadata{}
	if err := json.Unmarshal(data, &metadata); err != nil {
		return err
	}

	rh.storagePool = rh.storageService.createStoragePool(rh.id, metadata.Name, metadata.Description, uuid.MustParse(metadata.Uid), nil)

	rh.storagePool.allocatedVolume = AllocatedVolume{id: DefaultAllocatedVolumeId, capacityBytes: 0}

	return nil
}

func (rh *storagePoolRecoveryReplayHandler) Entry(typ uint32, data []byte) error {

	// Save the last log entry recorded in the ledger.
	rh.lastLogEntryType = typ

	switch typ {
	case storagePoolStorageCreateCompleteLogEntryType, storagePoolStorageUpdateCompleteLogEntryType:
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

		// TODO: delete storage pool

	case storagePoolStorageCreateCompleteLogEntryType, storagePoolStorageUpdateCompleteLogEntryType, storagePoolStorageDeleteStartLogEntryType:
		// Case 1. Create Complete: In this case, we've fully created the storage pool, and it is
		// fully recoverable and ready for use.

		// Case 2. Update Complete: In this case, we've fully updated the storage pool, and it is
		// fully recoverable and ready for use.

		// Case 3. Delete Start: We started a delete, but it did not finish. This means the storage pool
		// still exists, and its volumes are unknown. Here we try to recover the volumes, but ignore any
		// errors as the volume might be deleted. The client should retry the delete, at which point we
		// will delete any remaining volumes

		// Recover the namespaces that make up this storage pool
		if err := rh.storagePool.recoverVolumes(rh.volumes); err != nil {
			return false, err
		}

	case storagePoolStorageDeleteCompleteLogEntryType:
		// We've deleted all the volumes and the storage pool, but failed to delete the key. The client
		// should retry the delete, at which point we can delete the storage pool and the entry in
		// the store.
	}

	return false, nil
}

// Helper to check if storage device is nil
func storageDeviceIsNil(s interface{}) bool {
	if s == nil {
		return true
	}
	// Use reflection to check for the unexported field
	type deviceGetter interface{ DeviceApi() interface{} }
	if dg, ok := s.(deviceGetter); ok {
		return dg.DeviceApi() == nil
	}
	return false
}
