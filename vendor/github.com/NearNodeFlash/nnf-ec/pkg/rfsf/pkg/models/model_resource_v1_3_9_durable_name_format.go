/*
 * Swordfish API
 *
 * This contains the definition of the Swordfish extensions to a Redfish service.
 *
 * API version: v1.2.c
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi

type ResourceV139DurableNameFormat string

// List of Resource_v1_3_9_DurableNameFormat
const (
	NAA_RV139DNF ResourceV139DurableNameFormat = "NAA"
	I_QN_RV139DNF ResourceV139DurableNameFormat = "iQN"
	FC_WWN_RV139DNF ResourceV139DurableNameFormat = "FC_WWN"
	UUID_RV139DNF ResourceV139DurableNameFormat = "UUID"
	EUI_RV139DNF ResourceV139DurableNameFormat = "EUI"
)
