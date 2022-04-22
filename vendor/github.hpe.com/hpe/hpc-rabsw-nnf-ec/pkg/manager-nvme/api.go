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
 */

package nvme

import (
	"net/http"

	sf "github.hpe.com/hpe/hpc-rabsw-nnf-ec/pkg/rfsf/pkg/models"
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

type StorageApi interface {
	Get(*sf.StorageCollectionStorageCollection) error
	StorageIdGet(string, *sf.StorageV190Storage) error

	StorageIdStoragePoolsGet(string, *sf.StoragePoolCollectionStoragePoolCollection) error
	StorageIdStoragePoolsStoragePoolIdGet(string, string, *sf.StoragePoolV150StoragePool) error

	StorageIdControllersGet(string, *sf.StorageControllerCollectionStorageControllerCollection) error
	StorageIdControllersControllerIdGet(string, string, *sf.StorageControllerV100StorageController) error

	StorageIdVolumesGet(string, *sf.VolumeCollectionVolumeCollection) error
	StorageIdVolumesPost(string, *sf.VolumeV161Volume) error

	StorageIdVolumeIdGet(string, string, *sf.VolumeV161Volume) error
	StorageIdVolumeIdDelete(string, string) error
}
