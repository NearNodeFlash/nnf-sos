/*
 * Swordfish API
 *
 * This contains the definition of the Swordfish extensions to a Redfish service.
 *
 * API version: v1.2.c
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi

// ConnectionV100Connection - The Connection schema describes the access permissions endpoints, or groups of endpoints, have with other resources in the service.
type ConnectionV100Connection struct {

	// The OData description of a payload.
	OdataContext string `json:"@odata.context,omitempty"`

	// The current ETag of the resource.
	OdataEtag string `json:"@odata.etag,omitempty"`

	// The unique identifier for a resource.
	OdataId string `json:"@odata.id"`

	// The type of a resource.
	OdataType string `json:"@odata.type"`

	Actions ConnectionV100Actions `json:"Actions,omitempty"`

	ConnectionType ConnectionV100ConnectionType `json:"ConnectionType,omitempty"`

	// The description of this resource.  Used for commonality in the schema definitions.
	Description string `json:"Description,omitempty"`

	// The identifier that uniquely identifies the resource within the collection of similar resources.
	Id string `json:"Id"`

	Links ConnectionV100Links `json:"Links,omitempty"`

	// The name of the resource or array member.
	Name string `json:"Name"`

	// The OEM extension.
	Oem map[string]interface{} `json:"Oem,omitempty"`

	Status ResourceStatus `json:"Status,omitempty"`

	// The set of volumes and access capabilities specified for this connection.
	VolumeInfo []ConnectionV100VolumeInfo `json:"VolumeInfo,omitempty"`
}