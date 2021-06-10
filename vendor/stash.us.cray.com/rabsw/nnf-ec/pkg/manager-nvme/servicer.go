package nvme

import (
	"fmt"
	"net/http"

	. "stash.us.cray.com/rabsw/nnf-ec/pkg/common"
	
	sf "stash.us.cray.com/rabsw/rfsf-openapi/pkg/models"
)

// DefaultApiService -
type DefaultApiService struct {
}

// NewDefaultApiService -
func NewDefaultApiService() Api {
	return &DefaultApiService{}
}

// RedfishV1StorageGet
func (*DefaultApiService) RedfishV1StorageGet(w http.ResponseWriter, r *http.Request) {

	model := sf.StorageCollectionStorageCollection{
		OdataId:   fmt.Sprintf("/redfish/v1/Storage"),
		OdataType: "#StorageCollection.v1_0_0.StorageCollection",
		Name:      "Storage Collection",
	}

	err := Get(&model)

	EncodeResponse(model, err, w)
}

// RedfishV1StorageStorageIdGet
func (*DefaultApiService) RedfishV1StorageStorageIdGet(w http.ResponseWriter, r *http.Request) {
	params := Params(r)
	storageId := params["StorageId"]

	model := sf.StorageV190Storage{
		OdataId:   fmt.Sprintf("/redfish/v1/Storage/%s", storageId),
		OdataType: "#Storage.v1_9_0.Storage",
		Name:      "Storage",
	}

	err := StorageIdGet(storageId, &model)

	EncodeResponse(model, err, w)
}

// RedfishV1StorageStorageIdStoragePoolsGet
func (*DefaultApiService) RedfishV1StorageStorageIdStoragePoolsGet(w http.ResponseWriter, r *http.Request) {
	params := Params(r)
	storageId := params["StorageId"]

	model := sf.StoragePoolCollectionStoragePoolCollection{
		OdataId:   fmt.Sprintf("/redfish/v1/Storage/%s/StoragePools", storageId),
		OdataType: "#StoragePoolCollection.v1_0_0.StoragePoolCollection",
		Name:      "Storage Pool Collection",
	}

	err := StorageIdStoragePoolsGet(storageId, &model)

	EncodeResponse(model, err, w)
}

// RedfishV1StorageStorageIdStoragePoolsStoragePoolIdGet
func (*DefaultApiService) RedfishV1StorageStorageIdStoragePoolsStoragePoolIdGet(w http.ResponseWriter, r *http.Request) {
	params := Params(r)
	storageId := params["StorageId"]
	storagePoolId := params["StoragePoolId"]

	model := sf.StoragePoolV150StoragePool{
		OdataId:   fmt.Sprintf("/redfish/v1/Storage/%s/StoragePools/%s", storageId, storagePoolId),
		OdataType: "#StoragePool.v1_5_0.StoragePool",
		Name:      "Storage Pool",
	}

	err := StorageIdStoragePoolIdGet(storageId, storagePoolId, &model)

	EncodeResponse(model, err, w)
}

// RedfishV1StorageStorageIdControllersGet
func (*DefaultApiService) RedfishV1StorageStorageIdControllersGet(w http.ResponseWriter, r *http.Request) {
	params := Params(r)
	storageId := params["StorageId"]

	model := sf.StorageControllerCollectionStorageControllerCollection{
		OdataId:   fmt.Sprintf("/redfish/v1/Storage/%s/StorageControllers", storageId),
		OdataType: "#StorageControllerCollection.v1_0_0.StorageControllerCollection",
		Name:      "Storage Controller Collection",
	}

	err := StorageIdControllersGet(storageId, &model)

	EncodeResponse(model, err, w)
}

// RedfishV1StorageStorageIdControllersControllerIdGet
func (*DefaultApiService) RedfishV1StorageStorageIdControllersControllerIdGet(w http.ResponseWriter, r *http.Request) {
	params := Params(r)
	storageId := params["StorageId"]
	controllerId := params["ControllerId"]

	model := sf.StorageControllerV100StorageController{
		OdataId:   fmt.Sprintf("/redfish/v1/Storage/%s/StorageControllers/%s", storageId, controllerId),
		OdataType: "#StorageController.v1_0_0.StorageController",
		Name:      "Storage Controller",
	}

	err := StorageIdControllerIdGet(storageId, controllerId, &model)

	EncodeResponse(model, err, w)
}

// RedfishV1StorageStorageIdVolumesGet
func (*DefaultApiService) RedfishV1StorageStorageIdVolumesGet(w http.ResponseWriter, r *http.Request) {
	params := Params(r)
	storageId := params["StorageId"]

	model := sf.VolumeCollectionVolumeCollection{
		OdataId:   fmt.Sprintf("/redfish/v1/Storage/%s/Volumes", storageId),
		OdataType: "#VolumeCollection.v1_0_0.VolumeCollection",
		Name:      "Volume Collection",
	}

	err := StorageIdVolumesGet(storageId, &model)

	EncodeResponse(model, err, w)
}

// RedfishV1StorageStorageIdVolumesPost
func (*DefaultApiService) RedfishV1StorageStorageIdVolumesPost(w http.ResponseWriter, r *http.Request) {
	params := Params(r)
	storageId := params["StorageId"]

	var model sf.VolumeV161Volume

	if err := UnmarshalRequest(r, &model); err != nil {
		EncodeResponse(model, err, w)
		return
	}

	if err := StorageIdVolumePost(storageId, &model); err != nil {
		EncodeResponse(model, err, w)
		return
	}

	model.OdataId = fmt.Sprintf("/redfish/v1/Storage/%s/Volumes/%s", storageId, model.Id)
	model.OdataType = "#Volume.v1_6_1.Volume"
	model.Name = "Volume"

	EncodeResponse(model, nil, w)
}

// RedfishV1StorageStorageIdVolumesVolumeIdGet -
func (*DefaultApiService) RedfishV1StorageStorageIdVolumesVolumeIdGet(w http.ResponseWriter, r *http.Request) {
	params := Params(r)
	storageId := params["StorageId"]
	volumeId := params["VolumeId"]

	model := sf.VolumeV161Volume{
		OdataId:   fmt.Sprintf("/redfish/v1/Storage/%s/Volumes/%s", storageId, volumeId),
		OdataType: "Volume.v1_6_1.Volume",
		Name:      "Volume",
	}

	err := StorageIdVolumeIdGet(storageId, volumeId, &model)

	EncodeResponse(model, err, w)
}

// RedfishV1StorageStorageIdVolumesVolumeIdDelete -
func (*DefaultApiService) RedfishV1StorageStorageIdVolumesVolumeIdDelete(w http.ResponseWriter, r *http.Request) {
	params := Params(r)
	storageId := params["StorageId"]
	volumeId := params["VolumeId"]

	err := StorageIdVolumeIdDelete(storageId, volumeId)

	EncodeResponse(nil, err, w)
}
