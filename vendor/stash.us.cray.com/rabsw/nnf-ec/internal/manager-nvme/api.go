/*
 * Near Node Flash NVMe Namespace API
 *
 * This file contains the API interface for the Near-Node Flash
 * NVMe Namespace API. Each NNF implementation must define these
 * methods. Please keep the names consisitent with the Redfish
 * API definitions.
 *
 * Author: Nate Roiger
 *
 * Copyright 2020 Hewlett Packard Enterprise Development LP
 */

package nvme

import (
	"net/http"
)

// Api - defines an interface for Near-Node Flash related methods
type Api interface {
	//RedfishV1ChassisChassisIdDrivesGet(w http.ResponseWriter, r *http.Request)
	//RedfishV1ChassisChassisIdDrivesDriveIdGet(w http.ResponseWriter, r *http.Request)

	RedfishV1StorageGet(w http.ResponseWriter, r *http.Request)
	RedfishV1StorageStorageIdGet(w http.ResponseWriter, r *http.Request)

	RedfishV1StorageStorageIdStoragePoolsGet(w http.ResponseWriter, r *http.Request)
	RedfishV1StorageStorageIdStoragePoolsStoragePoolIdGet(w http.ResponseWriter, r *http.Request)

	RedfishV1StorageStorageIdControllersGet(w http.ResponseWriter, r *http.Request)
	RedfishV1StorageStorageIdControllersControllerIdGet(w http.ResponseWriter, r *http.Request)

	RedfishV1StorageStorageIdVolumesGet(w http.ResponseWriter, r *http.Request)
	RedfishV1StorageStorageIdVolumesPost(w http.ResponseWriter, r *http.Request)

	RedfishV1StorageStorageIdVolumesVolumeIdGet(w http.ResponseWriter, r *http.Request)
	RedfishV1StorageStorageIdVolumesVolumeIdDelete(w http.ResponseWriter, r *http.Request)
}
