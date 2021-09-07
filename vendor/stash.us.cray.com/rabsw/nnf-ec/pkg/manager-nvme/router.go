package nvme

import (
	ec "stash.us.cray.com/rabsw/nnf-ec/pkg/ec"
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
			Path:        "/redfish/v1/Storage/{StorageId}/StoragePool/{StoragePoolId}",
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
