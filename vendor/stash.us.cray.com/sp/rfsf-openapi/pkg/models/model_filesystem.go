/* -----------------------------------------------------------------
 * model_filesystem.go -
 *
 * SNIA Filesystem model.
 *
 * Author: Tim Morneau
 *
 * Copyright 2020 Cray an HPE Company.  All Rights Reserved.
 *
 * Except as permitted by contract or express written permission of
 * Cray Inc., no part of this work or its content may be modified,
 * used, reproduced or disclosed in any form.  Modifications made
 * without express permission of Cray Inc. may damage  the system
 * the software is installed within, may disqualify  the user from
 * receiving support from Cray Inc. under support or maintenance
 * contracts, or require additional support services outside the
 * scope of those contracts to repair the software or system.
 *
 * ----------------------------------------------------------------- */

package openapi

// FileSystem - FileSystem contains properties used to describe a FileSystem
type FileSystem struct {
	OdataContext string `json:"@odata.context,omitempty"`

	OdataId string `json:"@odata.id,omitempty"`

	OdataType string `json:"@odata.type,omitempty"`

	//AccessCapabilities - An array of supported IO access capabilities.
	AccessCapabilities map[string]interface{} `json:"ActionCapabilities,omitempty"`

	//Actions - The available actions for this resource.
	Actions map[string]interface{} `json:"Actions,omitempty"`

	//BlockSizeBytes - The size of the smallest addressible unit (Block) of this volume in bytes.
	BlockSizeBytes float32 `json:"BlockSizeBytes,omitempty"`

	//Capacity - Capacity allocated to the file system.
	Capacity Capacity `json:"Capacity,omitempty"`

	//CapacityBytes - The size in bytes of this Volume.
	CapacityBytes float32 `json:"CapacityBytes,omitempty"`

	//CapacitySources - An array of space allocations for this volume.
	CapacitySources []CapacitySource `json:"CapacitySources,omitempty"`

	//CapacitySourcesOdataCount - Capacity sources array entry count.
	CapacitySourcesOdataCount int32 `json:"CapacitySources@odata.count,omitempty"`

	//CasePreserved - The case of file names is preserved by the file system.
	CasePreserved bool `json:"CasePreserved,omitempty"`

	//CaseSensitive - Case sensitive file names are supported by the file system.
	CaseSensitive bool `json:"CaseSensitive,omitempty"`

	//CharacterCodeSet - Supported character code standards for different alphabets and languages.
	CharacterCodeSet []string `json:"CharacterCodeSet,omitempty"`

	//ClusterSizeBytes - A value indicating the minimum file allocation size imposed by the file system.
	ClusterSizeBytes int64 `json:"ClusterSizeBytes,omitempty"`

	//Description - Description of the FileSystem.
	Description string `json:"Description,omitempty"`

	//ExportedShares -  An array of exported file shares of this file system.
	ExportedShares map[string]interface{} `json:"ExportedShares,omitempty"`

	//Id - Id of the FileSystem.
	Id string `json:"Id"`

	//Identifiers - Durable names for this file system.
	Identifiers map[string]string `json:"Identifiers,omitempty"`

	//IOStatistics - Statistics for this FileSystem.
	IOStatistics map[string]interface{} `json:"IOStatistics,omitempty"`

	//ImportedShares - An array of imported file shares.
	ImportedShares map[string]interface{} `json:"Importedshares,omitempty"`

	//Links - Contains links to other resources that are related to this resource.
	Links map[string]interface{} `json:"Links,omitempty"`

	//LowSpaceWarningThresholdPercents - An array of low space warning threshold percentages for the file system.
	LowSpaceWarningThresholdPercents []int `json:"LowSpaceWarningThresholdPercents,omitempty"`

	//MaxFileNameLengthBytes - A value indicating the maximum length of a file name within the file system.
	MaxFileNameLengthBytes uint64 `json:"MaxFileNameLengthBytes,omitempty"`

	//Name - Name of the FileSystem.
	Name string `json:"Name"`

	//Oem - The Cray-defined OEM extension property.
	Oem FileSystemOem `json:"Oem,omitempty"`

	//RecoverableCapacitySourceCount - Current number of capacity source resources that are available as replacements.
	RecoverableCapacitySourceCount int64 `json:"RecoverableCapacitySourceCount,omitempty"`

	//RemainingCapacity - The percentage of the capacity remaining in the FileSystem.
	// DEPRECATED - Use Capacity
	RemainingCapacityPercent int16 `json:"RemainingCapacityPercent,omitempty"`

	//ReplicaInfo - This value describes the replica attributes if this file system is a replica.
	ReplicaInfo map[string]interface{} `json:"ReplicaInfo,omitempty"`

	//ReplicaTargets - The resources that are target replicas of this source.
	ReplicaTargets map[string]interface{} `json:"ReplicaTargets,omitempty"`

	//ReplicaTargetsCount - Count of the ReplicaTargets members.
	ReplicaTargetsCount int `json:"ReplicaTargets@odata.count,omitempty"`
}

type FileSystemOem struct {

	//FsType - Specifies which type of filesystem the request is for: ["Lustre"]
	FsType string `json:"FsType,omitempty"`

	//State - Specifies the state of the filesystem: ["Mounted", "Unmounted"]
	State string `json:"State,omitempty"`

	//Lustre - Contains lustre-specific information required.
	Lustre Lustre `json:"Lustre,omitempty"`

	//MountDir - Specifies local mount path for the filesystem.
	MountDir string `json:"MountDir"`

	//Hostname - Specifies the hostname the filesystem is mounted on.
	Hostname string `json:"Hostname"`
}

type Lustre struct {

	//MgsNID - Contains networking information of the MGS.
	// Format: <IPv4 address>@<LND protocol><lnd#>
	// See http://wiki.lustre.org/Lustre_Networking_(LNET)_Overview
	MgsNID string `json:"MgsNID,omitempty"`

	//TargetType - Specifies which target the request is for: ["MGT", "MDT", "OST"]
	TargetType string `json:"TargetType,omitempty"`

	//TargetIndex - Specifies the index of the target.
	TargetIndex uint32 `json:"TargetIndex,omitempty"`

	//BackFsType - Specifies the backing FS type for Lustre: ["ldiskfs", "zfs"]
	BackFsType string `json:"BackFsType,omitempty"`
}
