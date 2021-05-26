/*
 * Swordfish API
 *
 * This contains the definition of the Swordfish extensions to a Redfish service.
 *
 * API version: v1.0.7
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi

// StatelessAddressAutoConfiguration - Stateless Address Automatic Configuration (SLAAC) parameters for this interface.
type StatelessAddressAutoConfiguration struct {

	// Indicates whether IPv4 SLAAC is enabled for this interface.
	IPv4AutoConfigEnabled bool `json:"IPv4AutoConfigEnabled,omitempty"`

	// Indicates whether IPv6 SLAAC is enabled for this interface.
	IPv6AutoConfigEnabled bool `json:"IPv6AutoConfigEnabled,omitempty"`
}
