/*
 * Swordfish API
 *
 * This contains the definition of the Swordfish extensions to a Redfish service.
 *
 * API version: v1.2.c
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi

// AddressPoolV110GenZ - Gen-Z related properties for an address pool.
type AddressPoolV110GenZ struct {

	// The Access Key required for this address pool.
	AccessKey string `json:"AccessKey,omitempty"`

	// The maximum value for the Component Identifier (CID).
	MaxCID int64 `json:"MaxCID,omitempty"`

	// The maximum value for the Subnet Identifier (SID).
	MaxSID int64 `json:"MaxSID,omitempty"`

	// The minimum value for the Component Identifier (CID).
	MinCID int64 `json:"MinCID,omitempty"`

	// The minimum value for the Subnet Identifier (SID).
	MinSID int64 `json:"MinSID,omitempty"`
}
