/*
 * Swordfish API
 *
 * This contains the definition of the Swordfish extensions to a Redfish service.
 *
 * API version: v1.2.c
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi

// MemoryV1100ResetRequestBody - This action resets this memory device.
type MemoryV1100ResetRequestBody struct {

	ResetType ResourceResetType `json:"ResetType,omitempty"`
}
