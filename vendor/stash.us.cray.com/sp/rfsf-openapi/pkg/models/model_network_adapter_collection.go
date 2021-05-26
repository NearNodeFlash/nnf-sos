/*
 * Swordfish API
 *
 * This contains the definition of the Swordfish extensions to a Redfish service.
 *
 * API version: v1.0.7
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi

// NetworkAdapterCollection - A Collection of NetworkAdapter resource instances.
type NetworkAdapterCollection struct {
	OdataContext string `json:"@odata.context,omitempty"`

	OdataEtag string `json:"@odata.etag,omitempty"`

	OdataId string `json:"@odata.id"`

	OdataType string `json:"@odata.type"`

	Description string `json:"Description,omitempty"`

	// Contains the members of this collection.
	Members []map[string]interface{} `json:"Members"`

	MembersodataCount map[string]interface{} `json:"Members@odata.count,omitempty"`

	MembersodataNextLink map[string]interface{} `json:"Members@odata.nextLink,omitempty"`

	Name string `json:"Name"`

	Oem map[string]interface{} `json:"Oem,omitempty"`
}
