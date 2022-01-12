/*
 * Swordfish API
 *
 * This contains the definition of the Swordfish extensions to a Redfish service.
 *
 * API version: v1.2.c
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi

// ProcessorV1100Actions - The available actions for this resource.
type ProcessorV1100Actions struct {

	ProcessorReset ProcessorV1100Reset `json:"#Processor.Reset,omitempty"`

	// The available OEM-specific actions for this resource.
	Oem map[string]interface{} `json:"Oem,omitempty"`
}