/* -----------------------------------------------------------------
 * model_capacitysource.go -
 *
 * SNIA CapacitySource model.
 *
 * Author: Caleb Carlson
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

// CapacitySource - A description of the type and source of storage.
// Swordfish Schema: http://redfish.dmtf.org/schemas/swordfish/Capacity.v1_1_2.json
type CapacitySource struct {
	OdataContext string `json:"@odata.context,omitempty"`

	OdataEtag string `json:"@odata.etag,omitempty"`

	OdataId string `json:"@odata.id"`

	OdataType string `json:"@odata.type"`

	Description string `json:"Description,omitempty"`

	Id string `json:"Id"`

	Name string `json:"Name"`

	Oem map[string]interface{} `json:"Oem,omitempty"`

	// The operations currently running on the Drive.
	Operations []Operations `json:"Operations,omitempty"`

	// The amount of space that has been provided from the ProvidingDrives,
	// ProvidingVolumes, ProvidingMemory or ProvidingPools.
	ProvidedCapacity CapacityInfo `json:"ProvidedCapacity,omitempty"`

	// The ClassOfService provided from the ProvidingDrives, ProvidingVolumes,
	// ProvidingMemoryChunks, ProvidingMemory or ProvidingPools.
	ProvidedClassOfService map[string]interface{} `json:"ProvidedClassOfService,omitempty"`

	// The percentage of reads and writes that are predicted to still
	// be available for the mediaThe drive or drives that provide this space..
	ProvidingDrives []map[string]interface{} `json:"ProvidingDrives,omitempty"`

	// The memory that provides this space.
	ProvidingMemory map[string]interface{} `json:"ProvidingMemory,omitempty"`

	// The memory chunks that provide this space.
	ProvidingMemoryChunks map[string]interface{} `json:"ProvidingMemoryChunks,omitempty"`

	// The pool or pools that provide this space..
	ProvidingPools map[string]interface{} `json:"ProvidingPools,omitempty"`

	// The volume or volumes that provide this space.
	ProvidingVolumes VolumeCollection `json:"ProvidingVolumes,omitempty"`
}
