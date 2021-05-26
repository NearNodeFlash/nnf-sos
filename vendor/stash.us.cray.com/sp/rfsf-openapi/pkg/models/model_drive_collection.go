/*
 * Swordfish API
 *
 * This contains the definition of the Swordfish extensions to a Redfish service.
 *
 * Cray generated -
 *
 * Author: Tim Morneau
 */

package openapi

// Drives - The Drive schema represents a single physical disk drive for a system, including links to associated Volumes.
type Drives struct {
	OdataContext string `json:"@odata.context,omitempty"`

	OdataEtag string `json:"@odata.etag,omitempty"`

	OdataId string `json:"@odata.id"`

	OdataType string `json:"@odata.type"`

	Description string `json:"Description,omitempty"`
	// The below is a temporary implementation given the initial effort called the drive list
	// Drives and not Members
	Drives []map[string]interface{} `json:"Drives,omitempty"`

	Members []map[string]interface{} `json:"Members,omitempty"`

	OdataMembersCount int32 `json:"Members@odata.count,omitempty"`

	OdataMembersNextlink string `json:"Members@odata.nextLink,omitempty"`

	Name string `json:"Name"`

	Oem map[string]interface{} `json:"Oem,omitempty"`
}
