package nnf

import (
	"net/http"

	sf "stash.us.cray.com/rabsw/rfsf-openapi/pkg/models"
)

// API defines the programming interface for the Redfish / Swordfish routes. Each method is
// responsible for decoding the http.Request into useable parameters and calling it's
// corresponding Handler method (see below). Each method should respond to the request by
// updating the http.ResponesWriter.
type Api interface {
	Initialize(NnfControllerInterface) error

	Id() string

	RedfishV1StorageServicesGet(w http.ResponseWriter, r *http.Request)
	RedfishV1StorageServicesStorageServiceIdGet(w http.ResponseWriter, r *http.Request)

	RedfishV1StorageServicesStorageServiceIdCapacitySourceGet(w http.ResponseWriter, r *http.Request)

	RedfishV1StorageServicesStorageServiceIdStoragePoolsGet(w http.ResponseWriter, r *http.Request)
	RedfishV1StorageServicesStorageServiceIdStoragePoolsPost(w http.ResponseWriter, r *http.Request)
	RedfishV1StorageServicesStorageServiceIdStoragePoolsStoragePoolIdGet(w http.ResponseWriter, r *http.Request)
	RedfishV1StorageServicesStorageServiceIdStoragePoolsStoragePoolIdDelete(w http.ResponseWriter, r *http.Request)
	RedfishV1StorageServicesStorageServiceIdStoragePoolsStoragePoolIdCapacitySourcesGet(w http.ResponseWriter, r *http.Request)
	RedfishV1StorageServicesStorageServiceIdStoragePoolsStoragePoolIdCapacitySourcesCapacitySourceIdGet(w http.ResponseWriter, r *http.Request)
	RedfishV1StorageServicesStorageServiceIdStoragePoolsStoragePoolIdCapacitySourcesCapacitySourceIdProvidingVolumesGet(w http.ResponseWriter, r *http.Request)
	RedfishV1StorageServicesStorageServiceIdStoragePoolsStoragePoolIdAllocatedVolumesGet(w http.ResponseWriter, r *http.Request)
	RedfishV1StorageServicesStorageServiceIdStoragePoolsStoragePoolIdAllocatedVolumesVolumeIdGet(w http.ResponseWriter, r *http.Request)

	RedfishV1StorageServicesStorageServiceIdStorageGroupsGet(w http.ResponseWriter, r *http.Request)
	RedfishV1StorageServicesStorageServiceIdStorageGroupsPost(w http.ResponseWriter, r *http.Request)
	RedfishV1StorageServicesStorageServiceIdStorageGroupsStorageGroupIdGet(w http.ResponseWriter, r *http.Request)
	RedfishV1StorageServicesStorageServiceIdStorageGroupsStorageGroupIdDelete(w http.ResponseWriter, r *http.Request)

	RedfishV1StorageServicesStorageServiceIdEndpointsGet(w http.ResponseWriter, r *http.Request)
	RedfishV1StorageServicesStorageServiceIdEndpointsEndpointIdGet(w http.ResponseWriter, r *http.Request)

	RedfishV1StorageServicesStorageServiceIdFileSystemsGet(w http.ResponseWriter, r *http.Request)
	RedfishV1StorageServicesStorageServiceIdFileSystemsPost(w http.ResponseWriter, r *http.Request)
	RedfishV1StorageServicesStorageServiceIdFileSystemsFileSystemIdGet(w http.ResponseWriter, r *http.Request)
	RedfishV1StorageServicesStorageServiceIdFileSystemsFileSystemIdDelete(w http.ResponseWriter, r *http.Request)

	RedfishV1StorageServicesStorageServiceIdFileSystemsFileSystemsIdExportedFileSharesGet(w http.ResponseWriter, r *http.Request)
	RedfishV1StorageServicesStorageServiceIdFileSystemsFileSystemsIdExportedFileSharesPost(w http.ResponseWriter, r *http.Request)
	RedfishV1StorageServicesStorageServiceIdFileSystemsFileSystemsIdExportedFileSharesExportedFileSharesIdGet(w http.ResponseWriter, r *http.Request)
	RedfishV1StorageServicesStorageServiceIdFileSystemsFileSystemsIdExportedFileSharesExportedFileSharesIdDelete(w http.ResponseWriter, r *http.Request)
}

// Storage Service API defines the interface for the above API methods to call. Each API method must have
// an equivalent method. Methods take request paramters and a Redfish / Swordfish model to populate.
type StorageServiceApi interface {
	Initialize(NnfControllerInterface) error

	Id() string

	StorageServicesGet(*sf.StorageServiceCollectionStorageServiceCollection) error
	StorageServiceIdGet(string, *sf.StorageServiceV150StorageService) error

	StorageServiceIdCapacitySourceGet(string, *sf.CapacityCapacitySource) error

	StorageServiceIdStoragePoolsGet(string, *sf.StoragePoolCollectionStoragePoolCollection) error
	StorageServiceIdStoragePoolsPost(string, *sf.StoragePoolV150StoragePool) error
	StorageServiceIdStoragePoolIdGet(string, string, *sf.StoragePoolV150StoragePool) error
	StorageServiceIdStoragePoolIdDelete(string, string) error
	StorageServiceIdStoragePoolIdCapacitySourcesGet(string, string, *sf.CapacitySourceCollectionCapacitySourceCollection) error
	StorageServiceIdStoragePoolIdCapacitySourceIdGet(string, string, string, *sf.CapacityCapacitySource) error
	StorageServiceIdStoragePoolIdCapacitySourceIdProvidingVolumesGet(string, string, string, *sf.VolumeCollectionVolumeCollection) error
	StorageServiceIdStoragePoolIdAlloctedVolumesGet(string, string, *sf.VolumeCollectionVolumeCollection) error
	StorageServiceIdStoragePoolIdAllocatedVolumeIdGet(string, string, string, *sf.VolumeV161Volume) error

	StorageServiceIdStorageGroupsGet(string, *sf.StorageGroupCollectionStorageGroupCollection) error
	StorageServiceIdStorageGroupPost(string, *sf.StorageGroupV150StorageGroup) error
	StorageServiceIdStorageGroupIdGet(string, string, *sf.StorageGroupV150StorageGroup) error
	StorageServiceIdStorageGroupIdDelete(string, string) error

	StorageServiceIdEndpointsGet(string, *sf.EndpointCollectionEndpointCollection) error
	StorageServiceIdEndpointIdGet(string, string, *sf.EndpointV150Endpoint) error

	StorageServiceIdFileSystemsGet(string, *sf.FileSystemCollectionFileSystemCollection) error
	StorageServiceIdFileSystemsPost(string, *sf.FileSystemV122FileSystem) error
	StorageServiceIdFileSystemIdGet(string, string, *sf.FileSystemV122FileSystem) error
	StorageServiceIdFileSystemIdDelete(string, string) error

	StorageServiceIdFileSystemIdExportedSharesGet(string, string, *sf.FileShareCollectionFileShareCollection) error
	StorageServiceIdFileSystemIdExportedSharesPost(string, string, *sf.FileShareV120FileShare) error
	StorageServiceIdFileSystemIdExportedShareIdGet(string, string, string, *sf.FileShareV120FileShare) error
	StorageServiceIdFileSystemIdExportedShareIdDelete(string, string, string) error
}
