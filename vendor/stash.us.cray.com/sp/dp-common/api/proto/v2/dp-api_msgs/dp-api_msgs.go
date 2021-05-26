/*
 * gRPC protocol buffer message constants for communications to DP-API services
 *
 * Copyright 2020 HPE.  All Rights Reserved
 */

package dpapimsgs

// All global constants
const (
	DPAPIname       = "DP-API REST gRPC Service"
	DPAPIREST       = ":8080"
	DPAPIserverPort = ":50051"

	OdataID           = "@odata.id"
	OdataType         = "@odata.type"
	OdataContext      = "@odata.context"
	OdataMembersCount = "Members@odata.count"
	OdataMembers      = "Members"

	ApplicationJSONtype = "application/json"

	RedfishVersion         = "1.0.7a"
	RedfishV1              = "/redfish/v1"
	RedfishSystems         = RedfishV1 + "/Systems"
	RedfishStorageSystems  = RedfishV1 + "/StorageSystems"
	RedfishStorageServices = RedfishV1 + "/StorageServices"
	RedfishFileSystems     = RedfishStorageServices + "/FileSystems"
	RedfishAccountServices = RedfishV1 + "/AccountServices"
	RedfishSessions        = RedfishV1 + "/Sessions"
	RedfishMeta            = RedfishV1 + "/$metadata"

	Name           = "Name"
	Systems        = "Systems"
	Sessions       = "Sessions"
	Storage        = "Storage"
	Drives         = "Drives"
	Status         = "Status"
	StorageGroups  = "StorageGroups"
	StoragePools   = "StoragePools"
	StorageVolumes = "Volumes"
	LocalHost      = "localhost.localdomain"
	Description    = "Description"

	StorageServicesCollection     = "StorageServicesCollection.StorageServicesCollection"
	StorageServicesCollectionType = "#StorageServicesCollection.v1_2_0.StorageServicesCollection"
	StorageServicesCollectionName = "Storage Services Collection"

	StorageServicesType = "#StorageServices.v1_2_0.StorageServices"
	StorageServicesName = "Storage Services"

	TaskType = "#Task.v1_5_0.Task"
	TaskName = "Task"

	NotAvailable = "NotAvailable"
)
