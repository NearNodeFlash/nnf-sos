/*
 * Swordfish API
 *
 * This contains the definition of the Swordfish extensions to a Redfish service.
 *
 * API version: v1.2.c
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi

// StorageV190Storage - The Storage schema defines a storage subsystem and its respective properties.  A storage subsystem represents a set of physical or virtual storage controllers and the resources, such as volumes, that can be accessed from that subsystem.
type StorageV190Storage struct {

	// The OData description of a payload.
	OdataContext string `json:"@odata.context,omitempty"`

	// The current ETag of the resource.
	OdataEtag string `json:"@odata.etag,omitempty"`

	// The unique identifier for a resource.
	OdataId string `json:"@odata.id"`

	// The type of a resource.
	OdataType string `json:"@odata.type"`

	Actions StorageV190Actions `json:"Actions,omitempty"`

	ConsistencyGroups OdataV4IdRef `json:"ConsistencyGroups,omitempty"`

	Controllers OdataV4IdRef `json:"Controllers,omitempty"`

	// The description of this resource.  Used for commonality in the schema definitions.
	Description string `json:"Description,omitempty"`

	// The set of drives attached to the storage controllers that this resource represents.
	Drives []OdataV4IdRef `json:"Drives,omitempty"`

	// The number of items in a collection.
	DrivesodataCount int64 `json:"Drives@odata.count,omitempty"`

	EndpointGroups OdataV4IdRef `json:"EndpointGroups,omitempty"`

	FileSystems OdataV4IdRef `json:"FileSystems,omitempty"`

	// The identifier that uniquely identifies the resource within the collection of similar resources.
	Id string `json:"Id"`

	// The durable names for the storage subsystem.
	Identifiers []ResourceIdentifier `json:"Identifiers,omitempty"`

	Links StorageV190Links `json:"Links,omitempty"`

	Location ResourceLocation `json:"Location,omitempty"`

	// The name of the resource or array member.
	Name string `json:"Name"`

	// The OEM extension.
	Oem map[string]interface{} `json:"Oem,omitempty"`

	// Redundancy information for the storage subsystem.
	Redundancy []RedundancyRedundancy `json:"Redundancy,omitempty"`

	// The number of items in a collection.
	RedundancyodataCount int64 `json:"Redundancy@odata.count,omitempty"`

	Status ResourceStatus `json:"Status,omitempty"`

	// The set of storage controllers that this resource represents.
	StorageControllers []StorageV190StorageController `json:"StorageControllers,omitempty"`

	// The number of items in a collection.
	StorageControllersodataCount int64 `json:"StorageControllers@odata.count,omitempty"`

	StorageGroups OdataV4IdRef `json:"StorageGroups,omitempty"`

	StoragePools OdataV4IdRef `json:"StoragePools,omitempty"`

	Volumes OdataV4IdRef `json:"Volumes,omitempty"`
}
