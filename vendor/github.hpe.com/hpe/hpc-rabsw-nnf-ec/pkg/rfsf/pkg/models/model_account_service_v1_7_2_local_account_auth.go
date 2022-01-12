/*
 * Swordfish API
 *
 * This contains the definition of the Swordfish extensions to a Redfish service.
 *
 * API version: v1.2.c
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi

type AccountServiceV172LocalAccountAuth string

// List of AccountService_v1_7_2_LocalAccountAuth
const (
	ENABLED_ASV172LAA AccountServiceV172LocalAccountAuth = "Enabled"
	DISABLED_ASV172LAA AccountServiceV172LocalAccountAuth = "Disabled"
	FALLBACK_ASV172LAA AccountServiceV172LocalAccountAuth = "Fallback"
	LOCAL_FIRST_ASV172LAA AccountServiceV172LocalAccountAuth = "LocalFirst"
)