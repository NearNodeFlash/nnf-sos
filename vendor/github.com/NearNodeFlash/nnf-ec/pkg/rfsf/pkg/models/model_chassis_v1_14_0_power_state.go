/*
 * Swordfish API
 *
 * This contains the definition of the Swordfish extensions to a Redfish service.
 *
 * API version: v1.2.c
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi

type ChassisV1140PowerState string

// List of Chassis_v1_14_0_PowerState
const (
	ON_CV1140PST ChassisV1140PowerState = "On"
	OFF_CV1140PST ChassisV1140PowerState = "Off"
	POWERING_ON_CV1140PST ChassisV1140PowerState = "PoweringOn"
	POWERING_OFF_CV1140PST ChassisV1140PowerState = "PoweringOff"
)
