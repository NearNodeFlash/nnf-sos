/*
 * Swordfish API
 *
 * This contains the definition of the Swordfish extensions to a Redfish service.
 *
 * API version: v1.0.7
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi

type AddressState string

// List of AddressState
const (
	PREFERRED AddressState = "Preferred"
	DEPRECATED AddressState = "Deprecated"
	TENTATIVE AddressState = "Tentative"
	FAILED AddressState = "Failed"
)
