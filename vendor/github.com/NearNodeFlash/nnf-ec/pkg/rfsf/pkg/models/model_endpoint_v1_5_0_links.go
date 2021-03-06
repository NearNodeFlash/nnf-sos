/*
 * Swordfish API
 *
 * This contains the definition of the Swordfish extensions to a Redfish service.
 *
 * API version: v1.2.c
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi

// EndpointV150Links - The links to other resources that are related to this resource.
type EndpointV150Links struct {

	// An array of links to the address pools associated with this endpoint.
	AddressPools []OdataV4IdRef `json:"AddressPools,omitempty"`

	// The number of items in a collection.
	AddressPoolsodataCount int64 `json:"AddressPools@odata.count,omitempty"`

	// An array of links to the ports that connect to this endpoint.
	ConnectedPorts []OdataV4IdRef `json:"ConnectedPorts,omitempty"`

	// The number of items in a collection.
	ConnectedPortsodataCount int64 `json:"ConnectedPorts@odata.count,omitempty"`

	// The connections to which this endpoint belongs.
	Connections []OdataV4IdRef `json:"Connections,omitempty"`

	// The number of items in a collection.
	ConnectionsodataCount int64 `json:"Connections@odata.count,omitempty"`

	// An array of links to the endpoints that cannot be used in zones if this endpoint is in a zone.
	MutuallyExclusiveEndpoints []OdataV4IdRef `json:"MutuallyExclusiveEndpoints,omitempty"`

	// The number of items in a collection.
	MutuallyExclusiveEndpointsodataCount int64 `json:"MutuallyExclusiveEndpoints@odata.count,omitempty"`

	// When NetworkDeviceFunction resources are present, this array contains links to the network device functions that connect to this endpoint.
	NetworkDeviceFunction []OdataV4IdRef `json:"NetworkDeviceFunction,omitempty"`

	// The number of items in a collection.
	NetworkDeviceFunctionodataCount int64 `json:"NetworkDeviceFunction@odata.count,omitempty"`

	// The OEM extension.
	Oem map[string]interface{} `json:"Oem,omitempty"`

	// An array of links to the physical ports associated with this endpoint.
	Ports []OdataV4IdRef `json:"Ports,omitempty"`

	// The number of items in a collection.
	PortsodataCount int64 `json:"Ports@odata.count,omitempty"`
}
