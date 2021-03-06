/*
 * Swordfish API
 *
 * This contains the definition of the Swordfish extensions to a Redfish service.
 *
 * API version: v1.2.c
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi

// VirtualMediaV132Actions - The available actions for this Resource.
type VirtualMediaV132Actions struct {

	VirtualMediaEjectMedia VirtualMediaV132EjectMedia `json:"#VirtualMedia.EjectMedia,omitempty"`

	VirtualMediaInsertMedia VirtualMediaV132InsertMedia `json:"#VirtualMedia.InsertMedia,omitempty"`

	// The available OEM-specific actions for this Resource.
	Oem map[string]interface{} `json:"Oem,omitempty"`
}
