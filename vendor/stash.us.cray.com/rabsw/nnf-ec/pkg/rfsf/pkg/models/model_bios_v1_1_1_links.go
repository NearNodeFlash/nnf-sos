/*
 * Swordfish API
 *
 * This contains the definition of the Swordfish extensions to a Redfish service.
 *
 * API version: v1.2.c
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi

// BiosV111Links - The links to other resources that are related to this resource.
type BiosV111Links struct {

	ActiveSoftwareImage OdataV4IdRef `json:"ActiveSoftwareImage,omitempty"`

	// The OEM extension.
	Oem map[string]interface{} `json:"Oem,omitempty"`

	// The images that are associated with this BIOS.
	SoftwareImages []OdataV4IdRef `json:"SoftwareImages,omitempty"`

	// The number of items in a collection.
	SoftwareImagesodataCount int64 `json:"SoftwareImages@odata.count,omitempty"`
}