/* -----------------------------------------------------------------
 * model_redundancy.go -
 *
 * DMTF Redundancy model.
 *
 * Author: Daniel Matthews
 *
 * Copyright 2020 Cray an HPE Company.  All Rights Reserved.
 *
 * ----------------------------------------------------------------- */

package openapi

// This schema defines a Collection of Redundancy resource.
type Redundancy struct {
	OdataContext string `json:"@odata.context,omitempty"`

	OdataEtag string `json:"@odata.etag,omitempty"`

	OdataId string `json:"@odata.id"`

	OdataType string `json:"@odata.type"`

	Actions map[string]interface{} `json:"Actions,omitempty"`

	MaxNumSupported int `json:"MaxNumSupported"`

	MemberId string `json:"MemberId"`

	MinNumNeeded int `json:"MinNumNeeded"`

	Mode string `json:"Mode,omitempty"`

	Name string `json:"Name"`

	Id string `json:"Id"`
	// HPE specific HA Redundancy Attributes
	Oem HaRedundancyOem `json:"Oem,omitempty"`

	RedundancyEnabled bool `json:"RedundancyEnabled,omitempty"`
	// RedundancySet should track the OdataId links of all resources
	// this Redundancy resource will represent:
	// FileSystem, Data Volume, WIB Volume, JNRL Volume
	RedundancySet []map[string]interface{} `json:"RedundancySet"`

	RedundancySetOdataCount int `json:"RedundancySet@odata.count,omitempty"`

	Status map[string]interface{} `json:"Status,omitempty"`
}

type HaRedundancyOem struct {
	// OST, MDT, MGT
	TargetType string `json:"TargetType,omitempty"`
	// Name of main data Volume which this CRM group will manage
	ArrayName string `json:"ArrayName,omitempty"`
	// Resource group name
	ResourceGroupName string `json:"ResourceGroupName,omitempty"`
	// Resource group status - started, stopped, disabled
	ResourceGroupStatus string `json:"ResourceGroupStatus,omitempty"`
	// Primary hostname to start Resource group
	PrimaryLocation string `json:"PrimaryLocation,omitempty"`
	// Secondary hostname to start Resource group
	SecondaryLocation string `json:"SecondaryLocation,omitempty"`
	// Actual location Resource group is started at
	// Empty if not started
	Location string `json:"Location,omitempty"`
	// Shows whether resource group is currently failed over
	FailedOver bool `json:"FailedOver,omitempty"`
	/*
		Key should be resource name
		value should be resource status info
		ex oss groups:
		 Resource Group: kjlmo206_md0-group
		     kjlmo206_md0-wibr	(ocf::heartbeat:XYRAID):	Started kjlmo206
		     kjlmo206_md0-jnlr	(ocf::heartbeat:XYRAID):	Started kjlmo206
		     kjlmo206_md0-wibs	(ocf::heartbeat:XYMNTR):	Started kjlmo206
		     kjlmo206_md0-raid	(ocf::heartbeat:XYRAID):	Started kjlmo206
		     kjlmo206_md0-fsys	(ocf::heartbeat:XYMNTR):	Started kjlmo206
		     kjlmo206_md0-stop	(ocf::heartbeat:XYSTOP):	Started kjlmo206
	*/
	ResourceGroupAttrs []PrimitiveInfo `json:"ResourceGroupAttrs,omitempty"`
	// HA Networking Information
	HaNetworking HaNetworkingInfo `json:"HaNetworking,omitempty"`
	// StorageServiceId that owns the FileSystem and Volume resources to be managed by HA CRM resources this Redundancy resource holds
	StorageServiceId string `json:"StorageServiceId,omitempty"`
}

type PrimitiveInfo struct {
	// Primitive Name
	Name string `json:"Name,omitempty"`
	// Started, Stopped, Disabled, Error
	Status string `json:"Status,omitempty"`
	// Shows location of a started resource
	// Empty if not started
	Location string `json:"Location,omitempty"`
	// Shows whether resource is currently failed over
	FailedOver bool `json:"FailedOver,omitempty"`
	// Holds any potential Errors associated with resource
	ErrorMessage string `json:"ErrorMessage,omitempty"`
}

type HaNetworkingInfo struct {
	HaLNet LNetInfo `json:"LNetInfo,omitempty"`
}

type LNetInfo struct {
	MgsNid     string `json:"MgsNid,omitempty"`
	LocalNid   string `json:"LocalNid,omitempty"`
	PartnerNid string `json:"PartnerNid,omitempty"`
}
