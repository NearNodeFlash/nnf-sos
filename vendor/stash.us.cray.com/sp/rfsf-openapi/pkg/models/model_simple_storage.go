/*
 * Swordfish API
 *
 * This contains the definition of the Swordfish extensions to a Redfish service.
 *
 * API version: v1.0.7
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi

// SimpleStorage - This is the schema definition for the Simple Storage resource.  It represents the properties of a storage controller and its directly-attached devices.
type SimpleStorage struct {
	OdataContext string `json:"@odata.context,omitempty"`

	OdataEtag string `json:"@odata.etag,omitempty"`

	OdataId string `json:"@odata.id"`

	OdataType string `json:"@odata.type"`

	Actions map[string]interface{} `json:"Actions,omitempty"`

	Description string `json:"Description,omitempty"`

	// The storage devices associated with this resource.
	Devices []Device `json:"Devices,omitempty"`

	Id string `json:"Id"`

	Links map[string]interface{} `json:"Links,omitempty"`

	Name string `json:"Name"`

	Oem map[string]interface{} `json:"Oem,omitempty"`

	Status map[string]interface{} `json:"Status,omitempty"`

	// The UEFI device path used to access this storage controller.
	UefiDevicePath string `json:"UefiDevicePath,omitempty"`
}
