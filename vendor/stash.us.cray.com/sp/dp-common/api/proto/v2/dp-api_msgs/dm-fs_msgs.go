/*
 * gRPC protocol buffer message constants for communications to DP-API services
 *
 * Copyright 2020 HPE.  All Rights Reserved
 */

package dpapimsgs

// All global constants
const (
	FSECname = "File System Element Controller Service"
	FSECpath = "dp-fs-ec"
	FSECport = ":50055"

	FileSystem           = "FileSystem.FileSystem"
	FileSystemCollection = "FileSystemCollection.FileSystemCollection"
	FileSystemType       = "#FileSystem.v1_2_1.FileSystem"
	FileSystemName       = "FileSystem"

	CapacitySource     = "Capacity.CapacitySource"
	CapacitySourceType = "#Capacity.v1_1_2.CapacitySource"
	CapacitySourceName = "CapacitySource"

	CapacityInfo     = "Capacity.CapacityInfo"
	CapacityInfoType = "#Capacity.v1_1_2.CapacityInfo"
	CapacityInfoName = "CapacityInfo"

	Capacity     = "Capacity.Capacity"
	CapacityType = "#Capacity.v1_1_2.Capacity"
	CapacityName = "Capacity"
)
