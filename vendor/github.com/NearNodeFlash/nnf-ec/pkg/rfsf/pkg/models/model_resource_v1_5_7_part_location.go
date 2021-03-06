/*
 * Swordfish API
 *
 * This contains the definition of the Swordfish extensions to a Redfish service.
 *
 * API version: v1.2.c
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi

// ResourceV157PartLocation - The part location within the placement.
type ResourceV157PartLocation struct {

	// The number that represents the location of the part.  If LocationType is `slot` and this unit is in slot 2, the LocationOrdinalValue is 2.
	LocationOrdinalValue int64 `json:"LocationOrdinalValue,omitempty"`

	LocationType ResourceV157LocationType `json:"LocationType,omitempty"`

	Orientation ResourceV157Orientation `json:"Orientation,omitempty"`

	Reference ResourceV157Reference `json:"Reference,omitempty"`

	// The label of the part location, such as a silk-screened name or a printed label.
	ServiceLabel string `json:"ServiceLabel,omitempty"`
}
