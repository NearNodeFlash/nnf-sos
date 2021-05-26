/*
 * Swordfish API
 *
 * This contains the definition of the Swordfish extensions to a Redfish service.
 *
 * API version: v1.0.7
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi

// NetDevFuncMinBwAlloc - A minimum bandwidth allocation percentage for a Network Device Functions associated a port.
type NetDevFuncMinBwAlloc struct {

	// The minimum bandwidth allocation percentage allocated to the corresponding network device function instance.
	MinBWAllocPercent int32 `json:"MinBWAllocPercent,omitempty"`

	NetworkDeviceFunction map[string]interface{} `json:"NetworkDeviceFunction,omitempty"`
}
