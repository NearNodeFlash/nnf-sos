/*
 * Swordfish API
 *
 * This contains the definition of the Swordfish extensions to a Redfish service.
 *
 * API version: v1.0.7
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi

// DhcPv4Configuration - DHCPv4 configuration for this interface.
type DhcPv4Configuration struct {

	// Determines whether DHCPv4 is enabled on this interface.
	DHCPEnabled bool `json:"DHCPEnabled,omitempty"`

	// Determines whether to use DHCPv4-supplied DNS servers.
	UseDNSServers bool `json:"UseDNSServers,omitempty"`

	// Determines whether to use a DHCPv4-supplied domain name.
	UseDomainName bool `json:"UseDomainName,omitempty"`

	// Determines whether to use a DHCPv4-supplied gateway.
	UseGateway bool `json:"UseGateway,omitempty"`

	// Determines whether to use DHCPv4-supplied NTP servers.
	UseNTPServers bool `json:"UseNTPServers,omitempty"`

	// Determines whether to use DHCPv4-supplied static routes.
	UseStaticRoutes bool `json:"UseStaticRoutes,omitempty"`
}
