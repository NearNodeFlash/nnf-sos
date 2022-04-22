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

package nvme

import (
	"fmt"
	"net/http"

	. "github.hpe.com/hpe/hpc-rabsw-nnf-ec/pkg/common"

	sf "github.hpe.com/hpe/hpc-rabsw-nnf-ec/pkg/rfsf/pkg/models"
)

// DefaultApiService -
type DefaultApiService struct {
	api StorageApi
}

// NewDefaultApiService -
func NewDefaultApiService() Api {
	return &DefaultApiService{api: NewDefaultStorageService()}
}

// RedfishV1StorageGet
func (s *DefaultApiService) RedfishV1StorageGet(w http.ResponseWriter, r *http.Request) {

	model := sf.StorageCollectionStorageCollection{
		OdataId:   fmt.Sprintf("/redfish/v1/Storage"),
		OdataType: "#StorageCollection.v1_0_0.StorageCollection",
		Name:      "Storage Collection",
	}

	err := s.api.Get(&model)

	EncodeResponse(model, err, w)
}

// RedfishV1StorageStorageIdGet
func (s *DefaultApiService) RedfishV1StorageStorageIdGet(w http.ResponseWriter, r *http.Request) {
	params := Params(r)
	storageId := params["StorageId"]

	model := sf.StorageV190Storage{
		OdataId:   fmt.Sprintf("/redfish/v1/Storage/%s", storageId),
		OdataType: "#Storage.v1_9_0.Storage",
		Name:      "Storage",
	}

	err := s.api.StorageIdGet(storageId, &model)

	EncodeResponse(model, err, w)
}

// RedfishV1StorageStorageIdStoragePoolsGet
func (s *DefaultApiService) RedfishV1StorageStorageIdStoragePoolsGet(w http.ResponseWriter, r *http.Request) {
	params := Params(r)
	storageId := params["StorageId"]

	model := sf.StoragePoolCollectionStoragePoolCollection{
		OdataId:   fmt.Sprintf("/redfish/v1/Storage/%s/StoragePools", storageId),
		OdataType: "#StoragePoolCollection.v1_0_0.StoragePoolCollection",
		Name:      "Storage Pool Collection",
	}

	err := s.api.StorageIdStoragePoolsGet(storageId, &model)

	EncodeResponse(model, err, w)
}

// RedfishV1StorageStorageIdStoragePoolsStoragePoolIdGet
func (s *DefaultApiService) RedfishV1StorageStorageIdStoragePoolsStoragePoolIdGet(w http.ResponseWriter, r *http.Request) {
	params := Params(r)
	storageId := params["StorageId"]
	storagePoolId := params["StoragePoolId"]

	model := sf.StoragePoolV150StoragePool{
		OdataId:   fmt.Sprintf("/redfish/v1/Storage/%s/StoragePools/%s", storageId, storagePoolId),
		OdataType: "#StoragePool.v1_5_0.StoragePool",
		Name:      "Storage Pool",
	}

	err := s.api.StorageIdStoragePoolsStoragePoolIdGet(storageId, storagePoolId, &model)

	EncodeResponse(model, err, w)
}

// RedfishV1StorageStorageIdControllersGet
func (s *DefaultApiService) RedfishV1StorageStorageIdControllersGet(w http.ResponseWriter, r *http.Request) {
	params := Params(r)
	storageId := params["StorageId"]

	model := sf.StorageControllerCollectionStorageControllerCollection{
		OdataId:   fmt.Sprintf("/redfish/v1/Storage/%s/StorageControllers", storageId),
		OdataType: "#StorageControllerCollection.v1_0_0.StorageControllerCollection",
		Name:      "Storage Controller Collection",
	}

	err := s.api.StorageIdControllersGet(storageId, &model)

	EncodeResponse(model, err, w)
}

// RedfishV1StorageStorageIdControllersControllerIdGet
func (s *DefaultApiService) RedfishV1StorageStorageIdControllersControllerIdGet(w http.ResponseWriter, r *http.Request) {
	params := Params(r)
	storageId := params["StorageId"]
	controllerId := params["ControllerId"]

	model := sf.StorageControllerV100StorageController{
		OdataId:   fmt.Sprintf("/redfish/v1/Storage/%s/StorageControllers/%s", storageId, controllerId),
		OdataType: "#StorageController.v1_0_0.StorageController",
		Name:      "Storage Controller",
	}

	err := s.api.StorageIdControllersControllerIdGet(storageId, controllerId, &model)

	EncodeResponse(model, err, w)
}

// RedfishV1StorageStorageIdVolumesGet
func (s *DefaultApiService) RedfishV1StorageStorageIdVolumesGet(w http.ResponseWriter, r *http.Request) {
	params := Params(r)
	storageId := params["StorageId"]

	model := sf.VolumeCollectionVolumeCollection{
		OdataId:   fmt.Sprintf("/redfish/v1/Storage/%s/Volumes", storageId),
		OdataType: "#VolumeCollection.v1_0_0.VolumeCollection",
		Name:      "Volume Collection",
	}

	err := s.api.StorageIdVolumesGet(storageId, &model)

	EncodeResponse(model, err, w)
}

// RedfishV1StorageStorageIdVolumesPost
func (s *DefaultApiService) RedfishV1StorageStorageIdVolumesPost(w http.ResponseWriter, r *http.Request) {
	params := Params(r)
	storageId := params["StorageId"]

	var model sf.VolumeV161Volume

	if err := UnmarshalRequest(r, &model); err != nil {
		EncodeResponse(model, err, w)
		return
	}

	if err := s.api.StorageIdVolumesPost(storageId, &model); err != nil {
		EncodeResponse(model, err, w)
		return
	}

	model.OdataId = fmt.Sprintf("/redfish/v1/Storage/%s/Volumes/%s", storageId, model.Id)
	model.OdataType = "#Volume.v1_6_1.Volume"
	model.Name = "Volume"

	EncodeResponse(model, nil, w)
}

// RedfishV1StorageStorageIdVolumesVolumeIdGet -
func (s *DefaultApiService) RedfishV1StorageStorageIdVolumesVolumeIdGet(w http.ResponseWriter, r *http.Request) {
	params := Params(r)
	storageId := params["StorageId"]
	volumeId := params["VolumeId"]

	model := sf.VolumeV161Volume{
		OdataId:   fmt.Sprintf("/redfish/v1/Storage/%s/Volumes/%s", storageId, volumeId),
		OdataType: "Volume.v1_6_1.Volume",
		Name:      "Volume",
	}

	err := s.api.StorageIdVolumeIdGet(storageId, volumeId, &model)

	EncodeResponse(model, err, w)
}

// RedfishV1StorageStorageIdVolumesVolumeIdDelete -
func (s *DefaultApiService) RedfishV1StorageStorageIdVolumesVolumeIdDelete(w http.ResponseWriter, r *http.Request) {
	params := Params(r)
	storageId := params["StorageId"]
	volumeId := params["VolumeId"]

	err := s.api.StorageIdVolumeIdDelete(storageId, volumeId)

	EncodeResponse(nil, err, w)
}
