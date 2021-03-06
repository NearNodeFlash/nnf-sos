/*
 * Swordfish API
 *
 * This contains the definition of the Swordfish extensions to a Redfish service.
 *
 * API version: v1.2.c
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi

// ResourceV130Placement - The placement within the addressed location.
type ResourceV130Placement struct {

	// Name of a rack location within a row.
	Rack string `json:"Rack,omitempty"`

	// Vertical location of the item in terms of RackOffsetUnits.
	RackOffset *float32 `json:"RackOffset,omitempty"`

	RackOffsetUnits ResourceV130RackUnits `json:"RackOffsetUnits,omitempty"`

	// Name of row.
	Row string `json:"Row,omitempty"`
}
