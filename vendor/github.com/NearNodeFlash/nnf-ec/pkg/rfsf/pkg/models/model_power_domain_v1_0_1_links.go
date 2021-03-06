/*
 * Swordfish API
 *
 * This contains the definition of the Swordfish extensions to a Redfish service.
 *
 * API version: v1.2.c
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi

// PowerDomainV101Links - The links to other resources that are related to this resource.
type PowerDomainV101Links struct {

	// An array of links to the floor power distribution units in this power domain.
	FloorPDUs []OdataV4IdRef `json:"FloorPDUs,omitempty"`

	// The number of items in a collection.
	FloorPDUsodataCount int64 `json:"FloorPDUs@odata.count,omitempty"`

	// An array of links to the managers responsible for managing this power domain.
	ManagedBy []OdataV4IdRef `json:"ManagedBy,omitempty"`

	// The number of items in a collection.
	ManagedByodataCount int64 `json:"ManagedBy@odata.count,omitempty"`

	// The OEM extension.
	Oem map[string]interface{} `json:"Oem,omitempty"`

	// An array of links to the rack-level power distribution units in this power domain.
	RackPDUs []OdataV4IdRef `json:"RackPDUs,omitempty"`

	// The number of items in a collection.
	RackPDUsodataCount int64 `json:"RackPDUs@odata.count,omitempty"`

	// An array of links to the switchgear in this power domain.
	Switchgear []OdataV4IdRef `json:"Switchgear,omitempty"`

	// The number of items in a collection.
	SwitchgearodataCount int64 `json:"Switchgear@odata.count,omitempty"`

	// An array of links to the transfer switches in this power domain.
	TransferSwitches []OdataV4IdRef `json:"TransferSwitches,omitempty"`

	// The number of items in a collection.
	TransferSwitchesodataCount int64 `json:"TransferSwitches@odata.count,omitempty"`
}
