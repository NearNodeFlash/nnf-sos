/*
 * Swordfish API
 *
 * This contains the definition of the Swordfish extensions to a Redfish service.
 *
 * API version: v1.0.7
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi

// Volume2Actions - The available actions for this resource.
type Volume2Actions struct {

	VolumeInitialize map[string]interface{} `json:"#Volume.Initialize,omitempty"`

	Oem map[string]map[string]interface{} `json:"Oem,omitempty"`
}
