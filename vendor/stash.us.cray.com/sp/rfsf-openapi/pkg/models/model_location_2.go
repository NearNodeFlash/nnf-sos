/*
 * Swordfish API
 *
 * This contains the definition of the Swordfish extensions to a Redfish service.
 *
 * API version: v1.0.7
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi

type Location2 struct {

	// This indicates the location of the resource.
	Info string `json:"Info,omitempty"`

	// This represents the format of the Info property.
	InfoFormat string `json:"InfoFormat,omitempty"`

	Oem map[string]interface{} `json:"Oem,omitempty"`
}
