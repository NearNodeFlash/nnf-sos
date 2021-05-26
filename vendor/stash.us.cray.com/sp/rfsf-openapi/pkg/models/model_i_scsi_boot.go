/*
 * Swordfish API
 *
 * This contains the definition of the Swordfish extensions to a Redfish service.
 *
 * API version: v1.0.7
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi

// IScsiBoot - This type describes iSCSI boot capabilities, status, and configuration of a network device function.
type IScsiBoot struct {

	AuthenticationMethod map[string]interface{} `json:"AuthenticationMethod,omitempty"`

	// The shared secret for CHAP authentication.
	CHAPSecret string `json:"CHAPSecret,omitempty"`

	// The username for CHAP authentication.
	CHAPUsername string `json:"CHAPUsername,omitempty"`

	IPAddressType map[string]interface{} `json:"IPAddressType,omitempty"`

	// Whether the iSCSI boot initiator uses DHCP to obtain the iniator name, IP address, and netmask.
	IPMaskDNSViaDHCP bool `json:"IPMaskDNSViaDHCP,omitempty"`

	// The IPv6 or IPv4 iSCSI boot default gateway.
	InitiatorDefaultGateway string `json:"InitiatorDefaultGateway,omitempty"`

	// The IPv6 or IPv4 address of the iSCSI initiator.
	InitiatorIPAddress string `json:"InitiatorIPAddress,omitempty"`

	// The iSCSI initiator name.
	InitiatorName string `json:"InitiatorName,omitempty"`

	// The IPv6 or IPv4 netmask of the iSCSI boot initiator.
	InitiatorNetmask string `json:"InitiatorNetmask,omitempty"`

	// The CHAP Secret for 2-way CHAP authentication.
	MutualCHAPSecret string `json:"MutualCHAPSecret,omitempty"`

	// The CHAP Username for 2-way CHAP authentication.
	MutualCHAPUsername string `json:"MutualCHAPUsername,omitempty"`

	// The IPv6 or IPv4 address of the primary DNS server for the iSCSI boot initiator.
	PrimaryDNS string `json:"PrimaryDNS,omitempty"`

	// The logical unit number (LUN) for the primary iSCSI boot target.
	PrimaryLUN int32 `json:"PrimaryLUN,omitempty"`

	// The IP address (IPv6 or IPv4) for the primary iSCSI boot target.
	PrimaryTargetIPAddress string `json:"PrimaryTargetIPAddress,omitempty"`

	// The name of the iSCSI primary boot target.
	PrimaryTargetName string `json:"PrimaryTargetName,omitempty"`

	// The TCP port for the primary iSCSI boot target.
	PrimaryTargetTCPPort int32 `json:"PrimaryTargetTCPPort,omitempty"`

	// This indicates if the primary VLAN is enabled.
	PrimaryVLANEnable bool `json:"PrimaryVLANEnable,omitempty"`

	// The 802.1q VLAN ID to use for iSCSI boot from the primary target.
	PrimaryVLANId int32 `json:"PrimaryVLANId,omitempty"`

	// Whether IPv6 router advertisement is enabled for the iSCSI boot target.
	RouterAdvertisementEnabled bool `json:"RouterAdvertisementEnabled,omitempty"`

	// The IPv6 or IPv4 address of the secondary DNS server for the iSCSI boot initiator.
	SecondaryDNS string `json:"SecondaryDNS,omitempty"`

	// The logical unit number (LUN) for the secondary iSCSI boot target.
	SecondaryLUN int32 `json:"SecondaryLUN,omitempty"`

	// The IP address (IPv6 or IPv4) for the secondary iSCSI boot target.
	SecondaryTargetIPAddress string `json:"SecondaryTargetIPAddress,omitempty"`

	// The name of the iSCSI secondary boot target.
	SecondaryTargetName string `json:"SecondaryTargetName,omitempty"`

	// The TCP port for the secondary iSCSI boot target.
	SecondaryTargetTCPPort int32 `json:"SecondaryTargetTCPPort,omitempty"`

	// This indicates if the secondary VLAN is enabled.
	SecondaryVLANEnable bool `json:"SecondaryVLANEnable,omitempty"`

	// The 802.1q VLAN ID to use for iSCSI boot from the secondary target.
	SecondaryVLANId int32 `json:"SecondaryVLANId,omitempty"`

	// Whether the iSCSI boot target name, LUN, IP address, and netmask should be obtained from DHCP.
	TargetInfoViaDHCP bool `json:"TargetInfoViaDHCP,omitempty"`
}
