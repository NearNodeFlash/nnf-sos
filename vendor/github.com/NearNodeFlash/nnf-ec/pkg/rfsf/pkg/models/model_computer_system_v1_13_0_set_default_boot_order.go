/*
 * Swordfish API
 *
 * This contains the definition of the Swordfish extensions to a Redfish service.
 *
 * API version: v1.2.c
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi

// ComputerSystemV1130SetDefaultBootOrder - This action sets the BootOrder to the default settings.
type ComputerSystemV1130SetDefaultBootOrder struct {

	// Link to invoke action
	Target string `json:"target,omitempty"`

	// Friendly action name
	Title string `json:"title,omitempty"`
}
