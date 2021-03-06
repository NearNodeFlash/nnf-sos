/*
 * Swordfish API
 *
 * This contains the definition of the Swordfish extensions to a Redfish service.
 *
 * API version: v1.2.c
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi

// CompositionServiceV112CompositionService - The CompositionService schema describes a Composition Service and its properties and links to the Resources available for composition.
type CompositionServiceV112CompositionService struct {

	// The OData description of a payload.
	OdataContext string `json:"@odata.context,omitempty"`

	// The current ETag of the resource.
	OdataEtag string `json:"@odata.etag,omitempty"`

	// The unique identifier for a resource.
	OdataId string `json:"@odata.id"`

	// The type of a resource.
	OdataType string `json:"@odata.type"`

	Actions CompositionServiceV112Actions `json:"Actions,omitempty"`

	// An indication of whether this service is allowed to overprovision a composition relative to the composition request.
	AllowOverprovisioning bool `json:"AllowOverprovisioning,omitempty"`

	// An indication of whether a client can request that a specific Resource Zone fulfill a composition request.
	AllowZoneAffinity bool `json:"AllowZoneAffinity,omitempty"`

	// The description of this resource.  Used for commonality in the schema definitions.
	Description string `json:"Description,omitempty"`

	// The identifier that uniquely identifies the resource within the collection of similar resources.
	Id string `json:"Id"`

	// The name of the resource or array member.
	Name string `json:"Name"`

	// The OEM extension.
	Oem map[string]interface{} `json:"Oem,omitempty"`

	ResourceBlocks OdataV4IdRef `json:"ResourceBlocks,omitempty"`

	ResourceZones OdataV4IdRef `json:"ResourceZones,omitempty"`

	// An indication of whether this service is enabled.
	ServiceEnabled bool `json:"ServiceEnabled,omitempty"`

	Status ResourceStatus `json:"Status,omitempty"`
}
