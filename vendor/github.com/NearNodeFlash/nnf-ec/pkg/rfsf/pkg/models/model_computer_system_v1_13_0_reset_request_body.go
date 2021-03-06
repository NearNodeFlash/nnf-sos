/*
 * Swordfish API
 *
 * This contains the definition of the Swordfish extensions to a Redfish service.
 *
 * API version: v1.2.c
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi

// ComputerSystemV1130ResetRequestBody - This action resets the system.
type ComputerSystemV1130ResetRequestBody struct {

	ResetType ResourceResetType `json:"ResetType,omitempty"`
}
