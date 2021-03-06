/*
 * Swordfish API
 *
 * This contains the definition of the Swordfish extensions to a Redfish service.
 *
 * API version: v1.2.c
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi

type NetworkDeviceFunctionV150AuthenticationMethod string

// List of NetworkDeviceFunction_v1_5_0_AuthenticationMethod
const (
	NONE_NDFV150AM NetworkDeviceFunctionV150AuthenticationMethod = "None"
	CHAP_NDFV150AM NetworkDeviceFunctionV150AuthenticationMethod = "CHAP"
	MUTUAL_CHAP_NDFV150AM NetworkDeviceFunctionV150AuthenticationMethod = "MutualCHAP"
)
