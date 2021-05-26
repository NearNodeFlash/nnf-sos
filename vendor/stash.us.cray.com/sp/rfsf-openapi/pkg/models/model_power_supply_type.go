/*
 * Swordfish API
 *
 * This contains the definition of the Swordfish extensions to a Redfish service.
 *
 * API version: v1.0.7
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi

type PowerSupplyType string

// List of PowerSupplyType
const (
	UNKNOWN_PST PowerSupplyType = "Unknown"
	AC_PST PowerSupplyType = "AC"
	DC_PST PowerSupplyType = "DC"
	A_COR_DC PowerSupplyType = "ACorDC"
)
