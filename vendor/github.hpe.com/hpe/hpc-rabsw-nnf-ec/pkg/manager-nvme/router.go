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
	ec "github.hpe.com/hpe/hpc-rabsw-nnf-ec/pkg/ec"
)

// Router contains all the Redfish / Swordfish API calls that are hosted by
// the NNF module. All handler calls are of the form RedfishV1{endpoint}, where
// and endpoint is unique to the RF/SF caller.
//
// Router calls must reflect the same function name as the RF/SF API, as the
// element controller will perform a 1:1 function call based on the RF/SF
// caller's name.

// DefaultApiRouter -
type DefaultApiRouter struct {
	servicer   Api
	controller NvmeController
}

// NewDefaultApiRouter -
func NewDefaultApiRouter(s Api, c NvmeController) ec.Router {
	return &DefaultApiRouter{servicer: s, controller: c}
}

// Name -
func (*DefaultApiRouter) Name() string {
	return "NVMe Namespace Manager"
}

// Init -
func (r *DefaultApiRouter) Init() error {
	return Initialize(r.controller)
}

// Start -
func (r *DefaultApiRouter) Start() error {
	return nil
}

// Close -
func (r *DefaultApiRouter) Close() error {
	return Close()
}

// Routes -
func (r *DefaultApiRouter) Routes() ec.Routes {
	s := r.servicer
	return ec.Routes{
		{
			Name:        "RedfishV1StorageGet",
			Method:      ec.GET_METHOD,
			Path:        "/redfish/v1/Storage",
			HandlerFunc: s.RedfishV1StorageGet,
		},
		{
			Name:        "RedfishV1StorageStorageIdGet",
			Method:      ec.GET_METHOD,
			Path:        "/redfish/v1/Storage/{StorageId}",
			HandlerFunc: s.RedfishV1StorageStorageIdGet,
		},
		{
			Name:        "RedfishV1StorageStorageIdStoragePoolsGet",
			Method:      ec.GET_METHOD,
			Path:        "/redfish/v1/Storage/{StorageId}/StoragePools",
			HandlerFunc: s.RedfishV1StorageStorageIdStoragePoolsGet,
		},
		{
			Name:        "RedfishV1StorageStorageIdStoragePoolsStoragePoolIdGet",
			Method:      ec.GET_METHOD,
			Path:        "/redfish/v1/Storage/{StorageId}/StoragePools/{StoragePoolId}",
			HandlerFunc: s.RedfishV1StorageStorageIdStoragePoolsStoragePoolIdGet,
		},
		{
			Name:        "RedfishV1StorageStorageIdControllersGet",
			Method:      ec.GET_METHOD,
			Path:        "/redfish/v1/Storage/{StorageId}/Controllers",
			HandlerFunc: s.RedfishV1StorageStorageIdControllersGet,
		},
		{
			Name:        "RedfishV1StorageStorageIdControllersControllerIdGet",
			Method:      ec.GET_METHOD,
			Path:        "/redfish/v1/Storage/{StorageId}/Controllers/{ControllerId}",
			HandlerFunc: s.RedfishV1StorageStorageIdControllersControllerIdGet,
		},
		{
			Name:        "RedfishV1StorageStorageIdVolumesGet",
			Method:      ec.GET_METHOD,
			Path:        "/redfish/v1/Storage/{StorageId}/Volumes",
			HandlerFunc: s.RedfishV1StorageStorageIdVolumesGet,
		},
		{
			Name:        "RedfishV1StorageStorageIdVolumesVolumeIdGet",
			Method:      ec.GET_METHOD,
			Path:        "/redfish/v1/Storage/{StorageId}/Volumes/{VolumeId}",
			HandlerFunc: s.RedfishV1StorageStorageIdVolumesVolumeIdGet,
		},
		{
			Name:        "RedfishV1StorageStorageIdVolumesPost",
			Method:      ec.POST_METHOD,
			Path:        "/redfish/v1/Storage/{StorageId}/Volumes",
			HandlerFunc: s.RedfishV1StorageStorageIdVolumesPost,
		},
		{
			Name:        "RedfishV1StorageStorageIdVolumesVolumeIdDelete",
			Method:      ec.DELETE_METHOD,
			Path:        "/redfish/v1/Storage/{StorageId}/Volumes/{VolumeId}",
			HandlerFunc: s.RedfishV1StorageStorageIdVolumesVolumeIdDelete,
		},
	}
}
