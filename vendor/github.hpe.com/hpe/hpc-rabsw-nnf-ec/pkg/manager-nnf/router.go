package nnf

import (
	ec "github.hpe.com/hpe/hpc-rabsw-nnf-ec/pkg/ec"
)

type DefaultApiRouter struct {
	servicer   Api
	controller NnfControllerInterface
}

func NewDefaultApiRouter(servicer Api, ctrl NnfControllerInterface) ec.Router {
	return &DefaultApiRouter{servicer: servicer, controller: ctrl}
}

func (*DefaultApiRouter) Name() string {
	return "NNF Storage Service Manager"
}

func (r *DefaultApiRouter) Init() error {
	return r.servicer.Initialize(r.controller)
}

func (r *DefaultApiRouter) Start() error {
	return nil
}

func (r *DefaultApiRouter) Close() error {
	return r.servicer.Close()
}

func (r *DefaultApiRouter) Routes() ec.Routes {
	s := r.servicer
	return ec.Routes{
		{
			Name:        "RedfishV1StorageServicesGet",
			Method:      ec.GET_METHOD,
			Path:        "/redfish/v1/StorageServices",
			HandlerFunc: s.RedfishV1StorageServicesGet,
		},
		{
			Name:        "RedfishV1StorageServicesStorageServiceIdGet",
			Method:      ec.GET_METHOD,
			Path:        "/redfish/v1/StorageServices/{StorageServiceId}",
			HandlerFunc: s.RedfishV1StorageServicesStorageServiceIdGet,
		},

		/* -------------------- STORAGE SERVICE CAPACITY ------------------- */

		{
			Name:        "RedfishV1StorageServicesStorageServiceIdCapacitySourcesGet",
			Method:      ec.GET_METHOD,
			Path:        "/redfish/v1/StorageServices/{StorageServiceId}/CapacitySource",
			HandlerFunc: s.RedfishV1StorageServicesStorageServiceIdCapacitySourceGet,
		},

		/* ------------------------- STORAGE POOLS ------------------------- */

		{
			Name:        "RedfishV1StorageServicesStorageServiceIdStoragePoolsGet",
			Method:      ec.GET_METHOD,
			Path:        "/redfish/v1/StorageServices/{StorageServiceId}/StoragePools",
			HandlerFunc: s.RedfishV1StorageServicesStorageServiceIdStoragePoolsGet,
		},
		{
			Name:        "RedfishV1StorageServicesStorageServiceIdStoragePoolsPost",
			Method:      ec.POST_METHOD,
			Path:        "/redfish/v1/StorageServices/{StorageServiceId}/StoragePools",
			HandlerFunc: s.RedfishV1StorageServicesStorageServiceIdStoragePoolsPost,
		},
		{
			Name:        "RedfishV1StorageServicesStorageServiceIdStoragePoolsStoragePoolIdPut",
			Method:      ec.PUT_METHOD,
			Path:        "/redfish/v1/StorageServices/{StorageServiceId}/StoragePools/{StoragePoolId}",
			HandlerFunc: s.RedfishV1StorageServicesStorageServiceIdStoragePoolsStoragePoolIdPut,
		},
		{
			Name:        "RedfishV1StorageServicesStorageServiceIdStoragePoolsStoragePoolIdGet",
			Method:      ec.GET_METHOD,
			Path:        "/redfish/v1/StorageServices/{StorageServiceId}/StoragePools/{StoragePoolId}",
			HandlerFunc: s.RedfishV1StorageServicesStorageServiceIdStoragePoolsStoragePoolIdGet,
		},
		{
			Name:        "RedfishV1StorageServicesStorageServiceIdStoragePoolsStoragePoolIdDelete",
			Method:      ec.DELETE_METHOD,
			Path:        "/redfish/v1/StorageServices/{StorageServiceId}/StoragePools/{StoragePoolId}",
			HandlerFunc: s.RedfishV1StorageServicesStorageServiceIdStoragePoolsStoragePoolIdDelete,
		},
		{
			Name:        "RedfishV1StorageServicesStorageServiceIdStoragePoolsStoragePoolIdCapacitySourcesGet",
			Method:      ec.GET_METHOD,
			Path:        "/redfish/v1/StorageServices/{StorageServiceId}/StoragePools/{StoragePoolId}/CapacitySources",
			HandlerFunc: s.RedfishV1StorageServicesStorageServiceIdStoragePoolsStoragePoolIdCapacitySourcesGet,
		},
		{
			Name:        "RedfishV1StorageServicesStorageServiceIdStoragePoolsStoragePoolIdCapacitySourcesCapacitySourceIdGet",
			Method:      ec.GET_METHOD,
			Path:        "/redfish/v1/StorageServices/{StorageServiceId}/StoragePools/{StoragePoolId}/CapacitySources/{CapacitySourceId}",
			HandlerFunc: s.RedfishV1StorageServicesStorageServiceIdStoragePoolsStoragePoolIdCapacitySourcesCapacitySourceIdGet,
		},
		{
			Name:        "RedfishV1StorageServicesStorageServiceIdStoragePoolsStoragePoolIdCapacitySourcesCapacitySourceIdProvidingVolumesGet",
			Method:      ec.GET_METHOD,
			Path:        "/redfish/v1/StorageServices/{StorageServiceId}/StoragePools/{StoragePoolId}/CapacitySources/{CapacitySourceId}/ProvidingVolumes",
			HandlerFunc: s.RedfishV1StorageServicesStorageServiceIdStoragePoolsStoragePoolIdCapacitySourcesCapacitySourceIdProvidingVolumesGet,
		},
		{
			Name:        "RedfishV1StorageServicesStorageServiceIdStoragePoolsStoragePoolIdAllocatedVolumesGet",
			Method:      ec.GET_METHOD,
			Path:        "/redfish/v1/StorageServices/{StorageServiceId}/StoragePools/{StoragePoolId}/AllocatedVolumes",
			HandlerFunc: s.RedfishV1StorageServicesStorageServiceIdStoragePoolsStoragePoolIdAllocatedVolumesGet,
		},
		{
			Name:        "RedfishV1StorageServicesStorageServiceIdStoragePoolsStoragePoolIdAllocatedVolumesVolumeIdGet",
			Method:      ec.GET_METHOD,
			Path:        "/redfish/v1/StorageServices/{StorageServiceId}/StoragePools/{StoragePoolId}/AllocatedVolumes/{VolumeId}",
			HandlerFunc: s.RedfishV1StorageServicesStorageServiceIdStoragePoolsStoragePoolIdAllocatedVolumesVolumeIdGet,
		},

		/* ------------------------- STORAGE GROUPS ------------------------ */

		{
			Name:        "RedfishV1StorageServicesStorageServiceIdStorageGroupsGet",
			Method:      ec.GET_METHOD,
			Path:        "/redfish/v1/StorageServices/{StorageServiceId}/StorageGroups",
			HandlerFunc: s.RedfishV1StorageServicesStorageServiceIdStorageGroupsGet,
		},
		{
			Name:        "RedfishV1StorageServicesStorageServiceIdStorageGroupsPost",
			Method:      ec.POST_METHOD,
			Path:        "/redfish/v1/StorageServices/{StorageServiceId}/StorageGroups",
			HandlerFunc: s.RedfishV1StorageServicesStorageServiceIdStorageGroupsPost,
		},
		{
			Name:        "RedfishV1StorageServicesStorageServiceIdStorageGroupsStorageGroupIdPut",
			Method:      ec.PUT_METHOD,
			Path:        "/redfish/v1/StorageServices/{StorageServiceId}/StorageGroups/{StorageGroupId}",
			HandlerFunc: s.RedfishV1StorageServicesStorageServiceIdStorageGroupsStorageGroupIdPut,
		},
		{
			Name:        "RedfishV1StorageServicesStorageServiceIdStorageGroupsStorageGroupIdGet",
			Method:      ec.GET_METHOD,
			Path:        "/redfish/v1/StorageServices/{StorageServiceId}/StorageGroups/{StorageGroupId}",
			HandlerFunc: s.RedfishV1StorageServicesStorageServiceIdStorageGroupsStorageGroupIdGet,
		},
		{
			Name:        "RedfishV1StorageServicesStorageServiceIdStorageGroupsStorageGroupIdDelete",
			Method:      ec.DELETE_METHOD,
			Path:        "/redfish/v1/StorageServices/{StorageServiceId}/StorageGroups/{StorageGroupId}",
			HandlerFunc: s.RedfishV1StorageServicesStorageServiceIdStorageGroupsStorageGroupIdDelete,
		},

		/* --------------------------- ENDPOINTS --------------------------- */

		{
			Name:        "RedfishV1StorageServicesStorageServiceIdEndpointsGet",
			Method:      ec.GET_METHOD,
			Path:        "/redfish/v1/StorageServices/{StorageServiceId}/Endpoints",
			HandlerFunc: s.RedfishV1StorageServicesStorageServiceIdEndpointsGet,
		},

		{
			Name:        "RedfishV1StorageServicesStorageServiceIdEndpointsEndpointIdGet",
			Method:      ec.GET_METHOD,
			Path:        "/redfish/v1/StorageServices/{StorageServiceId}/Endpoints/{EndpointId}",
			HandlerFunc: s.RedfishV1StorageServicesStorageServiceIdEndpointsEndpointIdGet,
		},

		/* -------------------------- FILE SYSTEMS ------------------------- */

		{
			Name:        "RedfishV1StorageServicesStorageServiceIdFileSystemsGet",
			Method:      ec.GET_METHOD,
			Path:        "/redfish/v1/StorageServices/{StorageServiceId}/FileSystems",
			HandlerFunc: s.RedfishV1StorageServicesStorageServiceIdFileSystemsGet,
		},
		{
			Name:        "RedfishV1StorageServicesStorageServiceIdFileSystemsPost",
			Method:      ec.POST_METHOD,
			Path:        "/redfish/v1/StorageServices/{StorageServiceId}/FileSystems",
			HandlerFunc: s.RedfishV1StorageServicesStorageServiceIdFileSystemsPost,
		},
		{
			Name:        "RedfishV1StorageServicesStorageServiceIdFileSystemsFileSystemIdPut",
			Method:      ec.PUT_METHOD,
			Path:        "/redfish/v1/StorageServices/{StorageServiceId}/FileSystems/{FileSystemId}",
			HandlerFunc: s.RedfishV1StorageServicesStorageServiceIdFileSystemsFileSystemIdPut,
		},
		{
			Name:        "RedfishV1StorageServicesStorageServiceIdFileSystemsFileSystemIdGet",
			Method:      ec.GET_METHOD,
			Path:        "/redfish/v1/StorageServices/{StorageServiceId}/FileSystems/{FileSystemId}",
			HandlerFunc: s.RedfishV1StorageServicesStorageServiceIdFileSystemsFileSystemIdGet,
		},
		{
			Name:        "RedfishV1StorageServicesStorageServiceIdFileSystemsFileSystemIdDelete",
			Method:      ec.DELETE_METHOD,
			Path:        "/redfish/v1/StorageServices/{StorageServiceId}/FileSystems/{FileSystemId}",
			HandlerFunc: s.RedfishV1StorageServicesStorageServiceIdFileSystemsFileSystemIdDelete,
		},

		/* --------------------------- FILE SHARES ------------------------- */

		{
			Name:        "RedfishV1StorageServicesStorageServiceIdFileSystemsFileSystemsIdExportedFileSharesGet",
			Method:      ec.GET_METHOD,
			Path:        "/redfish/v1/StorageServices/{StorageServiceId}/FileSystems/{FileSystemsId}/ExportedFileShares",
			HandlerFunc: s.RedfishV1StorageServicesStorageServiceIdFileSystemsFileSystemsIdExportedFileSharesGet,
		},
		{
			Name:        "RedfishV1StorageServicesStorageServiceIdFileSystemsFileSystemsIdExportedFileSharesPost",
			Method:      ec.POST_METHOD,
			Path:        "/redfish/v1/StorageServices/{StorageServiceId}/FileSystems/{FileSystemsId}/ExportedFileShares",
			HandlerFunc: s.RedfishV1StorageServicesStorageServiceIdFileSystemsFileSystemsIdExportedFileSharesPost,
		},
		{
			Name:        "RedfishV1StorageServicesStorageServiceIdFileSystemsFileSystemsIdExportedFileSharesExportedFileSharesIdPut",
			Method:      ec.PUT_METHOD,
			Path:        "/redfish/v1/StorageServices/{StorageServiceId}/FileSystems/{FileSystemsId}/ExportedFileShares/{ExportedFileSharesId}",
			HandlerFunc: s.RedfishV1StorageServicesStorageServiceIdFileSystemsFileSystemsIdExportedFileSharesExportedFileSharesIdPut,
		},
		{
			Name:        "RedfishV1StorageServicesStorageServiceIdFileSystemsFileSystemsIdExportedFileSharesExportedFileSharesIdGet",
			Method:      ec.GET_METHOD,
			Path:        "/redfish/v1/StorageServices/{StorageServiceId}/FileSystems/{FileSystemsId}/ExportedFileShares/{ExportedFileSharesId}",
			HandlerFunc: s.RedfishV1StorageServicesStorageServiceIdFileSystemsFileSystemsIdExportedFileSharesExportedFileSharesIdGet,
		},
		{
			Name:        "RedfishV1StorageServicesStorageServiceIdFileSystemsFileSystemsIdExportedFileSharesExportedFileSharesIdDelete",
			Method:      ec.GET_METHOD,
			Path:        "/redfish/v1/StorageServices/{StorageServiceId}/FileSystems/{FileSystemsId}/ExportedFileShares/{ExportedFileSharesId}",
			HandlerFunc: s.RedfishV1StorageServicesStorageServiceIdFileSystemsFileSystemsIdExportedFileSharesExportedFileSharesIdDelete,
		},
	}
}
