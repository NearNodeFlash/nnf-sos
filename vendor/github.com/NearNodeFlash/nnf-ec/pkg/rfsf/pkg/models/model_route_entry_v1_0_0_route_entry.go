/*
 * Swordfish API
 *
 * This contains the definition of the Swordfish extensions to a Redfish service.
 *
 * API version: v1.2.c
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi

// RouteEntryV100RouteEntry - The RouteEntry schema describes the content of route entry rows.  Each route entry contains route sets that list the possible routes for the route entry.
type RouteEntryV100RouteEntry struct {

	// The OData description of a payload.
	OdataContext string `json:"@odata.context,omitempty"`

	// The current ETag of the resource.
	OdataEtag string `json:"@odata.etag,omitempty"`

	// The unique identifier for a resource.
	OdataId string `json:"@odata.id"`

	// The type of a resource.
	OdataType string `json:"@odata.type"`

	Actions RouteEntryV100Actions `json:"Actions,omitempty"`

	// The description of this resource.  Used for commonality in the schema definitions.
	Description string `json:"Description,omitempty"`

	// The identifier that uniquely identifies the resource within the collection of similar resources.
	Id string `json:"Id"`

	// The minimum number of hops.
	MinimumHopCount int64 `json:"MinimumHopCount,omitempty"`

	// The name of the resource or array member.
	Name string `json:"Name"`

	// The OEM extension.
	Oem map[string]interface{} `json:"Oem,omitempty"`

	// The raw data of route entry rows.
	RawEntryHex string `json:"RawEntryHex,omitempty"`

	RouteSet OdataV4IdRef `json:"RouteSet,omitempty"`
}
