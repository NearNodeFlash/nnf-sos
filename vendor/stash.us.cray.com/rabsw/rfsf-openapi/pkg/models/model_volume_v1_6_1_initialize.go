/*
 * Swordfish API
 *
 * This contains the definition of the Swordfish extensions to a Redfish service.
 *
 * API version: v1.2.c
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi

// VolumeV161Initialize - This action is used to prepare the contents of the volume for use by the system. If InitializeMethod is not specified in the request body, but the property InitializeMethod is specified, the property InitializeMethod value should be used. If neither is specified, the InitializeMethod should be Foreground.
type VolumeV161Initialize struct {

	// Link to invoke action
	Target string `json:"target,omitempty"`

	// Friendly action name
	Title string `json:"title,omitempty"`
}
