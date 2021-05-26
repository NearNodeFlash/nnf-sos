/* -----------------------------------------------------------------
 * model_capacity_info.go -
 *
 * SNIA CapacityInfo model.
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

// CapacityInfo - The capacity of specific data type in a data store.
// Swordfish Schema: http://redfish.dmtf.org/schemas/swordfish/Capacity.v1_1_2.json
type CapacityInfo struct {
	// The number of bytes currently allocated by the storage system in this data store for this data type.
	AllocatedBytes uint64 `json:"AllocatedBytes,omitempty"`

	// The number of bytes consumed in this data store for this data type.
	ConsumedBytes uint64 `json:"ConsumedBytes,omitempty"`

	// The number of bytes the storage system guarantees can be allocated in this data store for this data type.
	GuaranteedBytes uint64 `json:"GuaranteedBytes,omitempty"`

	// The maximum number of bytes that can be allocated in this data store for this data type.
	ProvisionedBytes uint64 `json:"ProvisionedBytes,omitempty"`
}
