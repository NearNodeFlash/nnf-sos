/*
 * Swordfish API
 *
 * This contains the definition of the Swordfish extensions to a Redfish service.
 *
 * API version: v1.2.c
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi
// ComputerSystemV1130PowerRestorePolicyTypes : The enumerations of PowerRestorePolicyTypes specify the choice of power state for the system when power is applied.
type ComputerSystemV1130PowerRestorePolicyTypes string

// List of ComputerSystem_v1_13_0_PowerRestorePolicyTypes
const (
	ALWAYS_ON_CSV1130PRPT ComputerSystemV1130PowerRestorePolicyTypes = "AlwaysOn"
	ALWAYS_OFF_CSV1130PRPT ComputerSystemV1130PowerRestorePolicyTypes = "AlwaysOff"
	LAST_STATE_CSV1130PRPT ComputerSystemV1130PowerRestorePolicyTypes = "LastState"
)
