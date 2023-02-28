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
	"errors"

	events "github.com/NearNodeFlash/nnf-ec/pkg/manager-event"
	msgreg "github.com/NearNodeFlash/nnf-ec/pkg/manager-message-registry/registries"

	ec "github.com/NearNodeFlash/nnf-ec/pkg/ec"
	sf "github.com/NearNodeFlash/nnf-ec/pkg/rfsf/pkg/models"
)

// Automatic Error Reporting Service Wraps the provided Storage Service API with automatic
// error capture and reporting service.
type AerService struct {
	s StorageServiceApi
}

func NewAerService(s StorageServiceApi) StorageServiceApi {
	return &AerService{s: s}
}

func (aer *AerService) publish(err error) {

	// If the supplied error is of an Element Controller Error type, inspect
	// the error for an event pointer value and, if found, publish the event to
	// the event manager.
	var e *ec.ControllerError
	if errors.As(err, &e) {
		if event, ok := e.Event.(*events.Event); event != nil && ok {
			events.EventManager.Publish(*event)
		} else {
			events.EventManager.Publish(msgreg.ResourceOperationFailedNnf(e.ResourceType(), e.Error()))
		}
	}
}

// The main capture routine for tracking errors to the storage service
func (aer *AerService) c(err error) error {
	if err != nil {
		aer.publish(err)
	}

	return err
}

func (aer *AerService) Initialize(log ec.Logger, ctrl NnfControllerInterface) error {
	return aer.c(aer.s.Initialize(log, ctrl))
}

func (aer *AerService) Close() error {
	return aer.c(aer.s.Close())
}

func (aer *AerService) Id() string {
	return aer.s.Id()
}

func (aer *AerService) StorageServicesGet(m *sf.StorageServiceCollectionStorageServiceCollection) error {
	return aer.c(aer.s.StorageServicesGet(m))
}
func (aer *AerService) StorageServiceIdGet(id string, model *sf.StorageServiceV150StorageService) error {
	return aer.c(aer.s.StorageServiceIdGet(id, model))
}
func (aer *AerService) StorageServiceIdCapacitySourceGet(id string, model *sf.CapacityCapacitySource) error {
	return aer.c(aer.s.StorageServiceIdCapacitySourceGet(id, model))
}

func (aer *AerService) StorageServiceIdStoragePoolsGet(id string, model *sf.StoragePoolCollectionStoragePoolCollection) error {
	return aer.c(aer.s.StorageServiceIdStoragePoolsGet(id, model))
}
func (aer *AerService) StorageServiceIdStoragePoolsPost(id string, model *sf.StoragePoolV150StoragePool) error {
	return aer.c(aer.s.StorageServiceIdStoragePoolsPost(id, model))
}
func (aer *AerService) StorageServiceIdStoragePoolIdGet(id0 string, id1 string, model *sf.StoragePoolV150StoragePool) error {
	return aer.c(aer.s.StorageServiceIdStoragePoolIdGet(id0, id1, model))
}
func (aer *AerService) StorageServiceIdStoragePoolIdPut(id0 string, id1 string, model *sf.StoragePoolV150StoragePool) error {
	return aer.c(aer.s.StorageServiceIdStoragePoolIdPut(id0, id1, model))
}
func (aer *AerService) StorageServiceIdStoragePoolIdDelete(id0 string, id1 string) error {
	return aer.c(aer.s.StorageServiceIdStoragePoolIdDelete(id0, id1))
}
func (aer *AerService) StorageServiceIdStoragePoolIdCapacitySourcesGet(id0 string, id1 string, model *sf.CapacitySourceCollectionCapacitySourceCollection) error {
	return aer.c(aer.s.StorageServiceIdStoragePoolIdCapacitySourcesGet(id0, id1, model))
}
func (aer *AerService) StorageServiceIdStoragePoolIdCapacitySourceIdGet(id0 string, id1 string, id2 string, model *sf.CapacityCapacitySource) error {
	return aer.c(aer.s.StorageServiceIdStoragePoolIdCapacitySourceIdGet(id0, id1, id2, model))
}
func (aer *AerService) StorageServiceIdStoragePoolIdCapacitySourceIdProvidingVolumesGet(id0 string, id1 string, id2 string, model *sf.VolumeCollectionVolumeCollection) error {
	return aer.c(aer.s.StorageServiceIdStoragePoolIdCapacitySourceIdProvidingVolumesGet(id0, id1, id2, model))
}
func (aer *AerService) StorageServiceIdStoragePoolIdAlloctedVolumesGet(id0 string, id1 string, model *sf.VolumeCollectionVolumeCollection) error {
	return aer.c(aer.s.StorageServiceIdStoragePoolIdAlloctedVolumesGet(id0, id1, model))
}
func (aer *AerService) StorageServiceIdStoragePoolIdAllocatedVolumeIdGet(id0 string, id1 string, id2 string, model *sf.VolumeV161Volume) error {
	return aer.c(aer.s.StorageServiceIdStoragePoolIdAllocatedVolumeIdGet(id0, id1, id2, model))
}

func (aer *AerService) StorageServiceIdStorageGroupsGet(id string, model *sf.StorageGroupCollectionStorageGroupCollection) error {
	return aer.c(aer.s.StorageServiceIdStorageGroupsGet(id, model))
}
func (aer *AerService) StorageServiceIdStorageGroupPost(id string, model *sf.StorageGroupV150StorageGroup) error {
	return aer.c(aer.s.StorageServiceIdStorageGroupPost(id, model))
}
func (aer *AerService) StorageServiceIdStorageGroupIdPut(id0 string, id1 string, model *sf.StorageGroupV150StorageGroup) error {
	return aer.c(aer.s.StorageServiceIdStorageGroupIdPut(id0, id1, model))
}
func (aer *AerService) StorageServiceIdStorageGroupIdGet(id0 string, id1 string, model *sf.StorageGroupV150StorageGroup) error {
	return aer.c(aer.s.StorageServiceIdStorageGroupIdGet(id0, id1, model))
}
func (aer *AerService) StorageServiceIdStorageGroupIdDelete(id0 string, id1 string) error {
	return aer.c(aer.s.StorageServiceIdStorageGroupIdDelete(id0, id1))
}

func (aer *AerService) StorageServiceIdEndpointsGet(id string, model *sf.EndpointCollectionEndpointCollection) error {
	return aer.c(aer.s.StorageServiceIdEndpointsGet(id, model))
}
func (aer *AerService) StorageServiceIdEndpointIdGet(id0 string, id1 string, model *sf.EndpointV150Endpoint) error {
	return aer.c(aer.s.StorageServiceIdEndpointIdGet(id0, id1, model))
}

func (aer *AerService) StorageServiceIdFileSystemsGet(id string, model *sf.FileSystemCollectionFileSystemCollection) error {
	return aer.c(aer.s.StorageServiceIdFileSystemsGet(id, model))
}
func (aer *AerService) StorageServiceIdFileSystemsPost(id string, model *sf.FileSystemV122FileSystem) error {
	return aer.c(aer.s.StorageServiceIdFileSystemsPost(id, model))
}
func (aer *AerService) StorageServiceIdFileSystemIdPut(id0 string, id1 string, model *sf.FileSystemV122FileSystem) error {
	return aer.c(aer.s.StorageServiceIdFileSystemIdPut(id0, id1, model))
}
func (aer *AerService) StorageServiceIdFileSystemIdGet(id0 string, id1 string, model *sf.FileSystemV122FileSystem) error {
	return aer.c(aer.s.StorageServiceIdFileSystemIdGet(id0, id1, model))
}
func (aer *AerService) StorageServiceIdFileSystemIdDelete(id0 string, id1 string) error {
	return aer.c(aer.s.StorageServiceIdFileSystemIdDelete(id0, id1))
}

func (aer *AerService) StorageServiceIdFileSystemIdExportedSharesGet(id0 string, id1 string, model *sf.FileShareCollectionFileShareCollection) error {
	return aer.c(aer.s.StorageServiceIdFileSystemIdExportedSharesGet(id0, id1, model))
}
func (aer *AerService) StorageServiceIdFileSystemIdExportedSharesPost(id0 string, id1 string, model *sf.FileShareV120FileShare) error {
	return aer.c(aer.s.StorageServiceIdFileSystemIdExportedSharesPost(id0, id1, model))
}
func (aer *AerService) StorageServiceIdFileSystemIdExportedShareIdPut(id0 string, id1 string, id2 string, model *sf.FileShareV120FileShare) error {
	return aer.c(aer.s.StorageServiceIdFileSystemIdExportedShareIdPut(id0, id1, id2, model))
}
func (aer *AerService) StorageServiceIdFileSystemIdExportedShareIdGet(id0 string, id1 string, id2 string, model *sf.FileShareV120FileShare) error {
	return aer.c(aer.s.StorageServiceIdFileSystemIdExportedShareIdGet(id0, id1, id2, model))
}
func (aer *AerService) StorageServiceIdFileSystemIdExportedShareIdDelete(id0 string, id1 string, id2 string) error {
	return aer.c(aer.s.StorageServiceIdFileSystemIdExportedShareIdDelete(id0, id1, id2))
}
