/*
 * Swordfish API
 *
 * This contains the definition of the Swordfish extensions to a Redfish service.
 *
 * API version: v1.0.7
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi

// LifeTime - This object contains the Memory metrics for the lifetime of the Memory.
type LifeTime struct {

	// Number of blocks read for the lifetime of the Memory.
	BlocksRead int32 `json:"BlocksRead,omitempty"`

	// Number of blocks written for the lifetime of the Memory.
	BlocksWritten int32 `json:"BlocksWritten,omitempty"`
}
