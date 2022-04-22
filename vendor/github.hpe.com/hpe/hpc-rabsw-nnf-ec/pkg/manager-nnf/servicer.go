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
	"fmt"
	"net/http"

	sf "github.hpe.com/hpe/hpc-rabsw-nnf-ec/pkg/rfsf/pkg/models"

	. "github.hpe.com/hpe/hpc-rabsw-nnf-ec/pkg/common"
)

const (
	StorageServiceOdataType = "#StorageService.v1_5_0.StorageService"
	StoragePoolOdataType    = "#StoragePool.v1_5_0.StoragePool"
	CapacitySourceOdataType = "#CapacitySource.v1_0_0.CapacitySource"
	VolumeOdataType         = "#Volume.v1_6_1.Volume"
	EndpointOdataType       = "#Endpoint.v1_5_0.Endpoint"
	StorageGroupOdataType   = "#StorageGroup.v1_5_0.StorageGroup"
	FileSystemOdataType     = "#FileSystem.v1_2_2.FileSystem"
	FileShareOdataType      = "#FileShare.v1_2_0.FileShare"
)

// DefaultApiService -
type DefaultApiService struct {
	ss StorageServiceApi
}

// NewDefaultApiService -
func NewDefaultApiService(ss StorageServiceApi) Api {
	return &DefaultApiService{ss: ss}
}

func (s *DefaultApiService) Id() string {
	return s.ss.Id()
}

func (s *DefaultApiService) Initialize(ctrl NnfControllerInterface) error {
	return s.ss.Initialize(ctrl)
}

func (s *DefaultApiService) Close() error {
	return s.ss.Close()
}

// RedfishV1StorageServicesGet -
func (s *DefaultApiService) RedfishV1StorageServicesGet(w http.ResponseWriter, r *http.Request) {

	model := sf.StorageServiceCollectionStorageServiceCollection{
		OdataId:   "/redfish/v1/StorageServices",
		OdataType: "#StorageServiceCollection.v1_0_0.StorageServiceCollection",
		Name:      "Storage Service Collection",
	}

	err := s.ss.StorageServicesGet(&model)

	EncodeResponse(model, err, w)
}

// RedfishV1StorageServicesStorageServiceIdGet -
func (s *DefaultApiService) RedfishV1StorageServicesStorageServiceIdGet(w http.ResponseWriter, r *http.Request) {
	params := Params(r)
	storageServiceId := params["StorageServiceId"]

	model := sf.StorageServiceV150StorageService{
		OdataId:   fmt.Sprintf("/redfish/v1/StorageServices/%s", storageServiceId),
		OdataType: StorageServiceOdataType,
		Name:      "Storage Service",
	}

	err := s.ss.StorageServiceIdGet(storageServiceId, &model)

	EncodeResponse(model, err, w)
}

// RedfishV1StorageServicesStorageServiceIdCapacitySourceGet -
func (s *DefaultApiService) RedfishV1StorageServicesStorageServiceIdCapacitySourceGet(w http.ResponseWriter, r *http.Request) {
	params := Params(r)
	storageServiceId := params["StorageServiceId"]

	model := sf.CapacityCapacitySource{
		OdataId:   fmt.Sprintf("/redfish/v1/StorageServices/%s/CapacitySource", storageServiceId),
		OdataType: CapacitySourceOdataType,
		Name:      "Capacity Source",
	}

	err := s.ss.StorageServiceIdCapacitySourceGet(storageServiceId, &model)

	EncodeResponse(model, err, w)
}

// RedfishV1StorageServicesStorageServiceIdStoragePoolsGet -
func (s *DefaultApiService) RedfishV1StorageServicesStorageServiceIdStoragePoolsGet(w http.ResponseWriter, r *http.Request) {
	params := Params(r)
	storageServiceId := params["StorageServiceId"]

	model := sf.StoragePoolCollectionStoragePoolCollection{
		OdataId:   fmt.Sprintf("/redfish/v1/StorageServices/%s/StoragePools", storageServiceId),
		OdataType: "#StoragePoolCollection.v1_0_0.StoragePoolCollection",
		Name:      "Storage Pool Collection",
	}

	err := s.ss.StorageServiceIdStoragePoolsGet(storageServiceId, &model)

	EncodeResponse(model, err, w)
}

// RedfishV1StorageServicesStorageServiceIdStoragePoolsPost -
func (s *DefaultApiService) RedfishV1StorageServicesStorageServiceIdStoragePoolsPost(w http.ResponseWriter, r *http.Request) {
	params := Params(r)
	storageServiceId := params["StorageServiceId"]

	var model sf.StoragePoolV150StoragePool

	if err := UnmarshalRequest(r, &model); err != nil {
		EncodeResponse(model, err, w)
		return
	}

	if err := s.ss.StorageServiceIdStoragePoolsPost(storageServiceId, &model); err != nil {
		EncodeResponse(model, err, w)
		return
	}

	model.OdataId = fmt.Sprintf("/redfish/v1/StorageServices/%s/StoragePools/%s", storageServiceId, model.Id)
	model.OdataType = StoragePoolOdataType

	EncodeResponse(model, nil, w)
}

// RedfishV1StorageServicesStorageServiceIdStoragePoolsStoragePoolIdGet -
func (s *DefaultApiService) RedfishV1StorageServicesStorageServiceIdStoragePoolsStoragePoolIdGet(w http.ResponseWriter, r *http.Request) {
	params := Params(r)
	storageServiceId := params["StorageServiceId"]
	storagePoolId := params["StoragePoolId"]

	model := sf.StoragePoolV150StoragePool{
		OdataId:   fmt.Sprintf("/redfish/v1/StorageServices/%s/StoragePools/%s", storageServiceId, storagePoolId),
		OdataType: StoragePoolOdataType,
		Name:      "Storage Pool",
	}

	err := s.ss.StorageServiceIdStoragePoolIdGet(storageServiceId, storagePoolId, &model)

	EncodeResponse(model, err, w)
}

// RedfishV1StorageServicesStorageServiceIdStoragePoolsStoragePoolIdPut -
func (s *DefaultApiService) RedfishV1StorageServicesStorageServiceIdStoragePoolsStoragePoolIdPut(w http.ResponseWriter, r *http.Request) {
	params := Params(r)
	storageServiceId := params["StorageServiceId"]
	storagePoolId := params["StoragePoolId"]

	var model sf.StoragePoolV150StoragePool
	if err := UnmarshalRequest(r, &model); err != nil {
		EncodeResponse(model, err, w)
		return
	}

	err := s.ss.StorageServiceIdStoragePoolIdPut(storageServiceId, storagePoolId, &model)

	EncodeResponse(model, err, w)
}

// RedfishV1StorageServicesStorageServiceIdStoragePoolsStoragePoolIdDelete -
func (s *DefaultApiService) RedfishV1StorageServicesStorageServiceIdStoragePoolsStoragePoolIdDelete(w http.ResponseWriter, r *http.Request) {
	params := Params(r)
	storageServiceId := params["StorageServiceId"]
	storagePoolId := params["StoragePoolId"]

	model := sf.StoragePoolV150StoragePool{
		OdataId:   fmt.Sprintf("/redfish/v1/StorageServices/%s/StoragePools/%s", storageServiceId, storagePoolId),
		OdataType: StoragePoolOdataType,
		Name:      "Storage Pool",
	}

	err := s.ss.StorageServiceIdStoragePoolIdDelete(storageServiceId, storagePoolId)

	EncodeResponse(model, err, w)
}

// RedfishV1StorageServicesStorageServiceIdStoragePoolsStoragePoolIdCapacitySourcesGet -
func (s *DefaultApiService) RedfishV1StorageServicesStorageServiceIdStoragePoolsStoragePoolIdCapacitySourcesGet(w http.ResponseWriter, r *http.Request) {
	params := Params(r)
	storageServiceId := params["StorageServiceId"]
	storagePoolId := params["StoragePoolId"]

	model := sf.CapacitySourceCollectionCapacitySourceCollection{
		OdataId:   fmt.Sprintf("/redfish/v1/StorageServices/%s/StoragePools/%s/CapacitySources", storageServiceId, storagePoolId),
		OdataType: "#CapacitySourceCollection.v1_0_0.CapacitySourceCollection",
		Name:      "Capacity Source Collection",
	}

	err := s.ss.StorageServiceIdStoragePoolIdCapacitySourcesGet(storageServiceId, storagePoolId, &model)

	EncodeResponse(model, err, w)
}

// RedfishV1StorageServicesStorageServiceIdStoragePoolsStoragePoolIdCapacitySourcesCapacitySourceIdGet -
func (s *DefaultApiService) RedfishV1StorageServicesStorageServiceIdStoragePoolsStoragePoolIdCapacitySourcesCapacitySourceIdGet(w http.ResponseWriter, r *http.Request) {
	params := Params(r)
	storageServiceId := params["StorageServiceId"]
	storagePoolId := params["StoragePoolId"]
	capacitySourceId := params["CapacitySourceId"]

	model := sf.CapacityCapacitySource{
		OdataId:   fmt.Sprintf("/redfish/v1/StorageServices/%s/StoragePools/%s/CapacitySources/%s", storageServiceId, storagePoolId, capacitySourceId),
		OdataType: CapacitySourceOdataType,
		Name:      "Capacity Source",
	}

	err := s.ss.StorageServiceIdStoragePoolIdCapacitySourceIdGet(storageServiceId, storagePoolId, capacitySourceId, &model)

	EncodeResponse(model, err, w)
}

// RedfishV1StorageServicesStorageServiceIdStoragePoolsStoragePoolIdCapacitySourcesCapacitySourceIdProvidingVolumesGet(w http.ResponseWriter, r *http.Request)
func (s *DefaultApiService) RedfishV1StorageServicesStorageServiceIdStoragePoolsStoragePoolIdCapacitySourcesCapacitySourceIdProvidingVolumesGet(w http.ResponseWriter, r *http.Request) {
	params := Params(r)
	storageServiceId := params["StorageServiceId"]
	storagePoolId := params["StoragePoolId"]
	capacitySourceId := params["CapacitySourceId"]

	model := sf.VolumeCollectionVolumeCollection{
		OdataId:   fmt.Sprintf("/redfish/v1/StorageServices/%s/StoragePools/%s/CapacitySources/%s/ProvidingVolumes", storageServiceId, storagePoolId, capacitySourceId),
		OdataType: "#VolumeCollection.v1_0_0.VolumeCollection",
		Name:      "Providing Volume Collection",
	}

	err := s.ss.StorageServiceIdStoragePoolIdCapacitySourceIdProvidingVolumesGet(storageServiceId, storagePoolId, capacitySourceId, &model)

	EncodeResponse(model, err, w)
}

// RedfishV1StorageServicesStorageServiceIdStoragePoolsStoragePoolIdAllocatedVolumesGet -
func (s *DefaultApiService) RedfishV1StorageServicesStorageServiceIdStoragePoolsStoragePoolIdAllocatedVolumesGet(w http.ResponseWriter, r *http.Request) {
	params := Params(r)
	storageServiceId := params["StorageServiceId"]
	storagePoolId := params["StoragePoolId"]

	model := sf.VolumeCollectionVolumeCollection{
		OdataId:   fmt.Sprintf("/redfish/v1/StorageServices/%s/StoragePools/%s/AllocatedVolumes", storageServiceId, storagePoolId),
		OdataType: "#VolumeCollection.v1_0_0.VolumeCollection",
		Name:      "Allocated Volume Collection",
	}

	err := s.ss.StorageServiceIdStoragePoolIdAlloctedVolumesGet(storageServiceId, storagePoolId, &model)

	EncodeResponse(model, err, w)
}

// RedfishV1StorageServicesStorageServiceIdStoragePoolsStoragePoolIdAllocatedVolumesVolumeIdGet
func (s *DefaultApiService) RedfishV1StorageServicesStorageServiceIdStoragePoolsStoragePoolIdAllocatedVolumesVolumeIdGet(w http.ResponseWriter, r *http.Request) {
	params := Params(r)
	storageServiceId := params["StorageServiceId"]
	storagePoolId := params["StoragePoolId"]
	volumeId := params["VolumeId"]

	model := sf.VolumeV161Volume{
		OdataId:   fmt.Sprintf("/redfish/v1/StorageServices/%s/StoragePools/%s/AllocatedVolumes/%s", storageServiceId, storagePoolId, volumeId),
		OdataType: VolumeOdataType,
		Name:      "Volume",
	}

	err := s.ss.StorageServiceIdStoragePoolIdAllocatedVolumeIdGet(storageServiceId, storagePoolId, volumeId, &model)

	EncodeResponse(model, err, w)
}

// RedfishV1StorageServicesStorageServiceIdStorageGroupsGet
func (s *DefaultApiService) RedfishV1StorageServicesStorageServiceIdStorageGroupsGet(w http.ResponseWriter, r *http.Request) {
	params := Params(r)
	storageServiceId := params["StorageServiceId"]

	model := sf.StorageGroupCollectionStorageGroupCollection{
		OdataId:   fmt.Sprintf("/redfish/v1/StorageServices/%s/StorageGroups", storageServiceId),
		OdataType: "#StorageGroupCollection.v1_0_0.StorageGroupCollection",
		Name:      "Storage Group Collection",
	}

	err := s.ss.StorageServiceIdStorageGroupsGet(storageServiceId, &model)

	EncodeResponse(model, err, w)
}

// RedfishV1StorageServicesStorageServiceIdStorageGroupsPost -
func (s *DefaultApiService) RedfishV1StorageServicesStorageServiceIdStorageGroupsPost(w http.ResponseWriter, r *http.Request) {
	params := Params(r)
	storageServiceId := params["StorageServiceId"]

	var model sf.StorageGroupV150StorageGroup

	if err := UnmarshalRequest(r, &model); err != nil {
		EncodeResponse(model, err, w)
		return
	}

	if err := s.ss.StorageServiceIdStorageGroupPost(storageServiceId, &model); err != nil {
		EncodeResponse(model, err, w)
		return
	}

	model.OdataId = fmt.Sprintf("/redfish/v1/StorageServices/%s/StorageGroups/%s", storageServiceId, model.Id)
	model.OdataType = StorageGroupOdataType
	model.Name = "Storage Group"

	EncodeResponse(model, nil, w)
}

// RedfishV1StorageServicesStorageServiceIdStorageGroupsStorageGroupIdPut
func (s *DefaultApiService) RedfishV1StorageServicesStorageServiceIdStorageGroupsStorageGroupIdPut(w http.ResponseWriter, r *http.Request) {
	params := Params(r)
	storageServiceId := params["StorageServiceId"]
	storageGroupId := params["StorageGroupId"]

	var model sf.StorageGroupV150StorageGroup
	if err := UnmarshalRequest(r, &model); err != nil {
		EncodeResponse(model, err, w)
		return
	}

	err := s.ss.StorageServiceIdStorageGroupIdPut(storageServiceId, storageGroupId, &model)

	EncodeResponse(model, err, w)
}

// RedfishV1StorageServicesStorageServiceIdStorageGroupsStorageGroupIdGet
func (s *DefaultApiService) RedfishV1StorageServicesStorageServiceIdStorageGroupsStorageGroupIdGet(w http.ResponseWriter, r *http.Request) {
	params := Params(r)
	storageServiceId := params["StorageServiceId"]
	storageGroupId := params["StorageGroupId"]

	model := sf.StorageGroupV150StorageGroup{
		OdataId:   fmt.Sprintf("/redfish/v1/StorageServices/%s/StorageGroups/%s", storageServiceId, storageGroupId),
		OdataType: StorageGroupOdataType,
		Name:      "Storage Group",
	}

	err := s.ss.StorageServiceIdStorageGroupIdGet(storageServiceId, storageGroupId, &model)

	EncodeResponse(model, err, w)
}

//RedfishV1StorageServicesStorageServiceIdStorageGroupsStorageGroupIdDelete(w http.ResponseWriter, r *http.Request)
func (s *DefaultApiService) RedfishV1StorageServicesStorageServiceIdStorageGroupsStorageGroupIdDelete(w http.ResponseWriter, r *http.Request) {
	params := Params(r)
	storageServiceId := params["StorageServiceId"]
	storageGroupId := params["StorageGroupId"]

	model := sf.StorageGroupV150StorageGroup{
		OdataId:   fmt.Sprintf("/redfish/v1/StorageServices/%s/StorageGroups/%s", storageServiceId, storageGroupId),
		OdataType: StorageGroupOdataType,
		Name:      "Storage Group",
	}

	err := s.ss.StorageServiceIdStorageGroupIdDelete(storageServiceId, storageGroupId)

	EncodeResponse(model, err, w)
}

// 	RedfishV1StorageServicesStorageServiceIdEndpointsGet -
func (s *DefaultApiService) RedfishV1StorageServicesStorageServiceIdEndpointsGet(w http.ResponseWriter, r *http.Request) {
	params := Params(r)
	storageServiceId := params["StorageServiceId"]

	model := sf.EndpointCollectionEndpointCollection{
		OdataId:   fmt.Sprintf("/redfish/v1/StorageServices/%s/Endpoints", storageServiceId),
		OdataType: "#EndpointCollection.v1_0_0.EndpointCollection",
		Name:      "Endpoint Collection",
	}

	err := s.ss.StorageServiceIdEndpointsGet(storageServiceId, &model)

	EncodeResponse(model, err, w)
}

// RedfishV1StorageServicesStorageServiceIdEndpointsEndpointIdGet
func (s *DefaultApiService) RedfishV1StorageServicesStorageServiceIdEndpointsEndpointIdGet(w http.ResponseWriter, r *http.Request) {
	params := Params(r)
	storageServiceId := params["StorageServiceId"]
	endpointId := params["EndpointId"]

	model := sf.EndpointV150Endpoint{
		OdataId:   fmt.Sprintf("/redfish/v1/StorageServices/%s/Endpoints/%s", storageServiceId, endpointId),
		OdataType: EndpointOdataType,
		Name:      "Endpoint",
	}

	err := s.ss.StorageServiceIdEndpointIdGet(storageServiceId, endpointId, &model)

	EncodeResponse(model, err, w)
}

// RedfishV1StorageServicesStorageServiceIdFileSystemsGet -
func (s *DefaultApiService) RedfishV1StorageServicesStorageServiceIdFileSystemsGet(w http.ResponseWriter, r *http.Request) {
	params := Params(r)
	storageServiceId := params["StorageServiceId"]

	model := sf.FileSystemCollectionFileSystemCollection{
		OdataId:   fmt.Sprintf("/redfish/v1/StorageServices/%s/FileSystems", storageServiceId),
		OdataType: "#FileSystemCollection.v1_0_0.FileSystemCollection",
		Name:      "File System Collection",
	}

	err := s.ss.StorageServiceIdFileSystemsGet(storageServiceId, &model)

	EncodeResponse(model, err, w)
}

// RedfishV1StorageServicesStorageServiceIdFileSystemsPost -
func (s *DefaultApiService) RedfishV1StorageServicesStorageServiceIdFileSystemsPost(w http.ResponseWriter, r *http.Request) {
	params := Params(r)
	storageServiceId := params["StorageServiceId"]

	var model sf.FileSystemV122FileSystem

	if err := UnmarshalRequest(r, &model); err != nil {
		EncodeResponse(nil, err, w)
		return
	}

	if err := s.ss.StorageServiceIdFileSystemsPost(storageServiceId, &model); err != nil {
		EncodeResponse(nil, err, w)
		return
	}

	model.OdataId = fmt.Sprintf("/redfish/v1/StorageServices/%s/FileSystems/%s", storageServiceId, model.Id)
	model.OdataType = FileSystemOdataType
	model.Name = "File System"

	EncodeResponse(model, nil, w)
}

// RedfishV1StorageServicesStorageServiceIdFileSystemsFileSystemIdPut
func (s *DefaultApiService) RedfishV1StorageServicesStorageServiceIdFileSystemsFileSystemIdPut(w http.ResponseWriter, r *http.Request) {
	params := Params(r)
	storageServiceId := params["StorageServiceId"]
	fileSystemId := params["FileSystemId"]

	var model sf.FileSystemV122FileSystem
	if err := UnmarshalRequest(r, &model); err != nil {
		EncodeResponse(nil, err, w)
		return
	}

	err := s.ss.StorageServiceIdFileSystemIdPut(storageServiceId, fileSystemId, &model)

	EncodeResponse(model, err, w)
}

// RedfishV1StorageServicesStorageServiceIdFileSystemsFileSystemIdGet -
func (s *DefaultApiService) RedfishV1StorageServicesStorageServiceIdFileSystemsFileSystemIdGet(w http.ResponseWriter, r *http.Request) {
	params := Params(r)
	storageServiceId := params["StorageServiceId"]
	fileSystemId := params["FileSystemId"]

	model := sf.FileSystemV122FileSystem{
		OdataId:   fmt.Sprintf("/redfish/v1/StorageServices/%s/FileSystems/%s", storageServiceId, fileSystemId),
		OdataType: FileSystemOdataType,
		Name:      "File System",
	}

	err := s.ss.StorageServiceIdFileSystemIdGet(storageServiceId, fileSystemId, &model)

	EncodeResponse(model, err, w)
}

// 	RedfishV1StorageServicesStorageServiceIdFileSystemsFileSystemIdDelete -
func (s *DefaultApiService) RedfishV1StorageServicesStorageServiceIdFileSystemsFileSystemIdDelete(w http.ResponseWriter, r *http.Request) {
	params := Params(r)
	storageServiceId := params["StorageServiceId"]
	fileSystemId := params["FileSystemId"]

	model := sf.FileSystemV122FileSystem{
		OdataId:   fmt.Sprintf("/redfish/v1/StorageServices/%s/FileSystems/%s", storageServiceId, fileSystemId),
		OdataType: FileSystemOdataType,
		Name:      "File System",
	}

	err := s.ss.StorageServiceIdFileSystemIdDelete(storageServiceId, fileSystemId)

	EncodeResponse(model, err, w)
}

// 	RedfishV1StorageServicesStorageServiceIdFileSystemsFileSystemsIdExportedFileSharesGet -
func (s *DefaultApiService) RedfishV1StorageServicesStorageServiceIdFileSystemsFileSystemsIdExportedFileSharesGet(w http.ResponseWriter, r *http.Request) {
	params := Params(r)
	storageServiceId := params["StorageServiceId"]
	fileSystemId := params["FileSystemsId"]

	model := sf.FileShareCollectionFileShareCollection{
		OdataId:   fmt.Sprintf("/redfish/v1/StorageServices/%s/FileSystems/%s/ExportedShares", storageServiceId, fileSystemId),
		OdataType: "#FileShareCollection.v1_0_0.FileShareCollection",
		Name:      "File Share Collection",
	}

	err := s.ss.StorageServiceIdFileSystemIdExportedSharesGet(storageServiceId, fileSystemId, &model)

	EncodeResponse(model, err, w)
}

// RedfishV1StorageServicesStorageServiceIdFileSystemsFileSystemsIdExportedFileSharesPost -
func (s *DefaultApiService) RedfishV1StorageServicesStorageServiceIdFileSystemsFileSystemsIdExportedFileSharesPost(w http.ResponseWriter, r *http.Request) {
	params := Params(r)
	storageServiceId := params["StorageServiceId"]
	fileSystemId := params["FileSystemsId"]

	var model sf.FileShareV120FileShare

	if err := UnmarshalRequest(r, &model); err != nil {
		EncodeResponse(model, err, w)
		return
	}

	err := s.ss.StorageServiceIdFileSystemIdExportedSharesPost(storageServiceId, fileSystemId, &model)

	model.OdataId = fmt.Sprintf("/redfish/v1/StorageServices/%s/FileSystems/%s/ExportedShares/%s", storageServiceId, fileSystemId, model.Id)
	model.OdataType = FileShareOdataType
	model.Name = "Exported File Share"

	EncodeResponse(model, err, w)
}

// RedfishV1StorageServicesStorageServiceIdFileSystemsFileSystemsIdExportedFileSharesExportedFileSharesIdPut -
func (s *DefaultApiService) RedfishV1StorageServicesStorageServiceIdFileSystemsFileSystemsIdExportedFileSharesExportedFileSharesIdPut(w http.ResponseWriter, r *http.Request) {
	params := Params(r)
	storageServiceId := params["StorageServiceId"]
	fileSystemId := params["FileSystemsId"]
	exportedShareId := params["ExportedFileSharesId"]

	var model sf.FileShareV120FileShare
	if err := UnmarshalRequest(r, &model); err != nil {
		EncodeResponse(model, err, w)
		return
	}

	err := s.ss.StorageServiceIdFileSystemIdExportedShareIdPut(storageServiceId, fileSystemId, exportedShareId, &model)

	model.OdataId = fmt.Sprintf("/redfish/v1/StorageServices/%s/FileSystems/%s/ExportedShares/%s", storageServiceId, fileSystemId, model.Id)
	model.OdataType = FileShareOdataType
	model.Name = "Exported File Share"

	EncodeResponse(model, err, w)
}

// RedfishV1StorageServicesStorageServiceIdFileSystemsFileSystemsIdExportedFileSharesExportedFileSharesIdGet -
func (s *DefaultApiService) RedfishV1StorageServicesStorageServiceIdFileSystemsFileSystemsIdExportedFileSharesExportedFileSharesIdGet(w http.ResponseWriter, r *http.Request) {
	params := Params(r)
	storageServiceId := params["StorageServiceId"]
	fileSystemId := params["FileSystemsId"]
	exportedShareId := params["ExportedFileSharesId"]

	model := sf.FileShareV120FileShare{
		OdataId:   fmt.Sprintf("/redfish/v1/StorageServices/%s/FileSystems/%s/ExportedShares/%s", storageServiceId, fileSystemId, exportedShareId),
		OdataType: FileShareOdataType,
		Name:      "Exported File Share",
	}

	err := s.ss.StorageServiceIdFileSystemIdExportedShareIdGet(storageServiceId, fileSystemId, exportedShareId, &model)

	EncodeResponse(model, err, w)
}

// RedfishV1StorageServicesStorageServiceIdFileSystemsFileSystemsIdExportedFileSharesExportedFileSharesIdDelete -
func (s *DefaultApiService) RedfishV1StorageServicesStorageServiceIdFileSystemsFileSystemsIdExportedFileSharesExportedFileSharesIdDelete(w http.ResponseWriter, r *http.Request) {
	params := Params(r)
	storageServiceId := params["StorageServiceId"]
	fileSystemId := params["FileSystemsId"]
	exportedShareId := params["ExportedFileSharesId"]

	model := sf.FileShareV120FileShare{
		OdataId:   fmt.Sprintf("/redfish/v1/StorageServices/%s/FileSystems/%s/ExportedShares/%s", storageServiceId, fileSystemId, exportedShareId),
		OdataType: FileShareOdataType,
		Name:      "Exported File Share",
	}

	err := s.ss.StorageServiceIdFileSystemIdExportedShareIdDelete(storageServiceId, fileSystemId, exportedShareId)

	EncodeResponse(model, err, w)
}
