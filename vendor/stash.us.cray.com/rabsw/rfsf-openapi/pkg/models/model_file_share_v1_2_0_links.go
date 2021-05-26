/*
 * Swordfish API
 *
 * This contains the definition of the Swordfish extensions to a Redfish service.
 *
 * API version: v1.2.c
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi

// FileShareV120Links - The links object contains the links to other resources that are related to this resource.
type FileShareV120Links struct {
	ClassOfService OdataV4IdRef `json:"ClassOfService,omitempty"`

	Endpoint OdataV4IdRef `json:"Endpoint,omitempty"`

	FileSystem OdataV4IdRef `json:"FileSystem,omitempty"`

	// The OEM extension.
	Oem map[string]interface{} `json:"Oem,omitempty"`
}
