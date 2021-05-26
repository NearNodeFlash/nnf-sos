/* -----------------------------------------------------------------
 * model_manager.go -
 *
 * DMTF Manager model.
 *
 * Author: Daniel Matthews
 *
 * Copyright 2020 Cray an HPE Company.  All Rights Reserved.
 *
 * ----------------------------------------------------------------- */

package openapi

// This schema defines a Manager resource.
type Manager struct {
	OdataContext string `json:"@odata.context,omitempty"`

	OdataEtag string `json:"@odata.etag,omitempty"`

	OdataId string `json:"@odata.id"`

	OdataType string `json:"@odata.type"`
	// Actions defined for Manager
	// Will need OemActions for specific HA actions
	Actions map[string]interface{} `json:"Actions,omitempty"`

	Description string `json:"Description,omitempty"`

	// proxied
	EthernetInterfaces []map[string]interface{} `json:"EthernetInterfaces,omitempty"`
	// BMC firmware version - Proxied
	FirmwareVersion string `json:"FirmwareVersion,omitempty"`
	// Canister Serial Number
	Id string `json:"Id"`
	// lasttime manager was reset or rebooted - proxied
	LastResetTime string `json:"LastResetTime,omitempty"`

	Links map[string]interface{} `json:"Links,omitempty"`
	// BMC LogServices - proxied
	LogServices map[string]interface{} `json:"LogServices,omitempty"`
	// Can be anything "HA"
	ManagerType string `json:"ManagerType"`
	// Canister Manufacturer - proxied
	Manufacturer string `json:"Manufacturer,omitempty"`
	// Canister Model - proxied
	Model string `json:"Model,omitempty"`

	Name string `json:"Name"`
	// proxied 
	NetworkProtocol map[string]interface{} `json:"NetworkProtocol,omitempty"`
	// HPE specific HA Manager Oem attributes
	Oem HaOem `json:"Oem,omitempty"`
	// PartNumber of the canister - proxied
	PartNumber string `json:"PartNumber,omitempty"`
	// PowerState of the host - proxied
	PowerState string `json:"PowerState,omitempty"`
	// HA Redundancy Resources
	Redundancy []map[string]interface{} `json:"Redundancy"`

	RedundancyOdataCount int `json:"Redundancy@odata.count,omitempty"`

	SerialNumber string `json:"SerialNumber,omitempty"`
	// Status / Health of the BMC - proxied 
	Status map[string]interface{} `json:"Status,omitempty"`

	UUID string `json:"UUID,omitempty"`
}

type HaOem struct {
	// primary or secondary node in node pair
	IsPrimary bool `json:"IsPrimary,omitempty"`
	// hostname of pod
	Hostname string `json:"Hostname,omitempty"`
	// IP of host
	HostIP string `json:"HostIP,omitempty"`
	//Pod name
	PodName string `json:"PodName,omitempty"`
	// Pod IP
	PodIP string `json:"PodIP,omitempty"`
	// primary ipmi address - proxied
	IpmiAddr string `json:"IpmiAddr,omitempty"` 
	// secondary ipmi address - proxied
	IpmiSecAddr string `json:"IpmiSecAddr,omitempty"`
	// ipmi username - proxied
	IpmiUser string `json:"IpmiUser,omitempty"`
	// ipmi password - proxied
	IpmiPass string `json:"IpmiPass,omitempty"`
	// enclosure serial number, verify Manager pairs belong to same enclosure
	ChassisSerialNumber string `json:"ChassisSerialNumber,omitempty"`
	// information about node pair partner
	PartnerInfo HaPartnerInfo `json:"PartnerInfo,omitempty"`
	// HA daemon info
	DaemonInfo HADaemonInfo `json:"DaemonInfo,omitempty"`
}

type HaPartnerInfo struct {
	// primary or secondary node in node pair
	IsPrimary bool `json:"IsPrimary,omitempty"`
	// partner hostname
	Hostname string `json:"Hostname,omitempty"`
	// IP of partner host
	HostIP string `json:"HostIP,omitempty"` 
	// Pod name of partner
	PodName string `json:"PodName,omitempty"` 
	// Pod IP of partner
	PodIP string `json:"PodIP,omitempty"`
	// partner primary ipmi address - proxied
	IpmiAddr string `json:"IpmiAddr,omitempty"`
	// partner secondary ipmi address - proxied
	IpmiSecAddr string `json:"IpmiSecAddr,omitempty"`
	// ipmi username - proxied
	IpmiUser string `json:"IpmiUser,omitempty"`
	// ipmi password - proxied
	IpmiPass string `json:"IpmiPass,omitempty"`
	// partner serial number (ManagerId)
	SerialNumber string `json:"SerialNumber,omitempty"`
	// enclosure serial number, verify Manager pairs belong to same enclosure
	ChassisSerialNumber string `json:"ChassisSerialNumber,omitempty"`
	// partner HA daemon info
	DaemonInfo HADaemonInfo `json:"DaemonInfo,omitempty"`
}

type HADaemonInfo struct {
	// Corosync enabled
	CorosyncEnabled bool `json:"CorosyncEnabled,omitempty"`
	// Corosync status
	CorosyncStatus string `json:"CorosyncStatus,omitempty"`
	// Pacemaker enabled
	PacemakerEnabled bool `json:"PacemakerEnabled,omitempty"`
	// Pacemaker status
	PacemakerStatus string `json:"PacemakerStatus,omitempty"`
	// remote-tls-port configured
	RemoteTlsConfigured bool `json:"RemoteTlsConfigured,omitempty"`
}
