/*
 * gRPC protocol buffer message constants for communications to DP-API services
 *
 * Copyright 2020 HPE.  All Rights Reserved
 */

package dpapimsgs

// All global constants
const (
	DMECname                   = "Drive Management Element Controller Service"
	DMECpath                   = "dp-drive-mgmt-elmnt-cntrl"
	DMECport                   = ":50052"
	StoragePoolsCollection     = "StoragePoolCollection.StoragePoolCollection"
	StoragePoolsCollectionType = "#StoragePoolCollection.v1_2_0.StoragePoolCollection"
	StoragePoolsCollectionName = "Storage Pool Collection"

	StoragePool     = "StoragePool.StoragePool"
	StoragePoolType = "#StoragePool.v1_2_0.StoragePool"
	StoragePoolName = "Storage Pool"

	VolumesCollection     = "VolumeCollection.VolumeCollection"
	VolumesCollectionType = "#VolumeCollection.v1_2_0.VolumeCollection"
	VolumesCollectionName = "Volume Collection"

	Volume     = "Volume.Volume"
	VolumeType = "#Volume.v1_2_0.Volume"
	VolumeName = "Volume"

	DriveCollection     = "DriveCollection.DriveCollection"
	DriveCollectionType = "#DriveCollection.v1_5_0.DriveCollection"
	DriveCollectionName = "Drive Collection"

	Drive     = "Drive.Drive"
	DriveType = "#Drive.v1_5_0.Drive"
	DriveName = "Drive"
)
