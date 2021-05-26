/*
 * Swordfish API
 *
 * This contains the definition of the Swordfish extensions to a Redfish service.
 *
 * API version: v1.2.c
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi

// ResourceV193PartLocation - The part location within the placement.
type ResourceV193PartLocation struct {

	// The number that represents the location of the part.  If LocationType is `slot` and this unit is in slot 2, the LocationOrdinalValue is 2.
	LocationOrdinalValue int64 `json:"LocationOrdinalValue,omitempty"`

	LocationType ResourceV193LocationType `json:"LocationType,omitempty"`

	Orientation ResourceV193Orientation `json:"Orientation,omitempty"`

	Reference ResourceV193Reference `json:"Reference,omitempty"`

	// The label of the part location, such as a silk-screened name or a printed label.
	ServiceLabel string `json:"ServiceLabel,omitempty"`
}
