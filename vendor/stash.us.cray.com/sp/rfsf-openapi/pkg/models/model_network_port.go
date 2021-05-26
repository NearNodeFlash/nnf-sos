/*
 * Swordfish API
 *
 * This contains the definition of the Swordfish extensions to a Redfish service.
 *
 * API version: v1.0.7
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi

// NetworkPort - A Network Port represents a discrete physical port capable of connecting to a network.
type NetworkPort struct {
	OdataContext string `json:"@odata.context,omitempty"`

	OdataEtag string `json:"@odata.etag,omitempty"`

	OdataId string `json:"@odata.id"`

	OdataType string `json:"@odata.type"`

	Actions map[string]interface{} `json:"Actions,omitempty"`

	ActiveLinkTechnology map[string]interface{} `json:"ActiveLinkTechnology,omitempty"`

	// The array of configured network addresses (MAC or WWN) that are associated with this Network Port, including the programmed address of the lowest numbered Network Device Function, the configured but not active address if applicable, the address for hardware port teaming, or other network addresses.
	AssociatedNetworkAddresses []string `json:"AssociatedNetworkAddresses,omitempty"`

	// Network Port Current Link Speed.
	CurrentLinkSpeedMbps int32 `json:"CurrentLinkSpeedMbps,omitempty"`

	Description string `json:"Description,omitempty"`

	// Whether IEEE 802.3az Energy Efficient Ethernet (EEE) is enabled for this network port.
	EEEEnabled bool `json:"EEEEnabled,omitempty"`

	// The FC Fabric Name provided by the switch.
	FCFabricName string `json:"FCFabricName,omitempty"`

	FCPortConnectionType map[string]interface{} `json:"FCPortConnectionType,omitempty"`

	FlowControlConfiguration map[string]interface{} `json:"FlowControlConfiguration,omitempty"`

	FlowControlStatus map[string]interface{} `json:"FlowControlStatus,omitempty"`

	Id string `json:"Id"`

	LinkStatus map[string]interface{} `json:"LinkStatus,omitempty"`

	// The maximum frame size supported by the port.
	MaxFrameSize int32 `json:"MaxFrameSize,omitempty"`

	Name string `json:"Name"`

	// The array of maximum bandwidth allocation percentages for the Network Device Functions associated with this port.
	NetDevFuncMaxBWAlloc []NetDevFuncMaxBwAlloc `json:"NetDevFuncMaxBWAlloc,omitempty"`

	// The array of minimum bandwidth allocation percentages for the Network Device Functions associated with this port.
	NetDevFuncMinBWAlloc []NetDevFuncMinBwAlloc `json:"NetDevFuncMinBWAlloc,omitempty"`

	// The number of ports not on this adapter that this port has discovered.
	NumberDiscoveredRemotePorts int32 `json:"NumberDiscoveredRemotePorts,omitempty"`

	Oem map[string]interface{} `json:"Oem,omitempty"`

	// The physical port number label for this port.
	PhysicalPortNumber string `json:"PhysicalPortNumber,omitempty"`

	// The largest maximum transmission unit (MTU) that can be configured for this network port.
	PortMaximumMTU int32 `json:"PortMaximumMTU,omitempty"`

	// Whether or not the port has detected enough signal on enough lanes to establish link.
	SignalDetected bool `json:"SignalDetected,omitempty"`

	Status map[string]interface{} `json:"Status,omitempty"`

	// The set of Ethernet capabilities that this port supports.
	SupportedEthernetCapabilities []SupportedEthernetCapabilities `json:"SupportedEthernetCapabilities,omitempty"`

	// The self-described link capabilities of this port.
	SupportedLinkCapabilities []SupportedLinkCapabilities `json:"SupportedLinkCapabilities,omitempty"`

	// The Vendor Identification for this port.
	VendorId string `json:"VendorId,omitempty"`

	// Whether Wake on LAN (WoL) is enabled for this network port.
	WakeOnLANEnabled bool `json:"WakeOnLANEnabled,omitempty"`
}
