/*
 * Swordfish API
 *
 * This contains the definition of the Swordfish extensions to a Redfish service.
 *
 * API version: v1.0.7
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi

// SecureEraseUnitRequestBody - This defines the action for securely erasing given regions using the NIST SP800-88 Purge: Cryptograhic Erase.
type SecureEraseUnitRequestBody struct {

	// Passphrase for doing the operation.
	Passphrase string `json:"Passphrase"`

	// Memory region ID for which this action to be applied.
	RegionId string `json:"RegionId"`
}
