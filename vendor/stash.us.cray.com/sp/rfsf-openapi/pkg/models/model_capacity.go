/* -----------------------------------------------------------------
 * model_capacity.go -
 *
 * SNIA Capacity model.
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

// Capacity - represents the properties for capacity for any data store.
// Swordfish Schema: http://redfish.dmtf.org/schemas/swordfish/Capacity.v1_1_2.json
type Capacity struct {
	// The capacity information relating to the user data.
	Data CapacityInfo `json:"Data,omitempty"`

	// Marks that the capacity is not necessarily fully allocated.
	IsThinProvisioned bool `json:"IsThinProvisioned,omitempty"`

	// The capacity information relating to  metadata.
	Metadata CapacityInfo `json:"Metadata,omitempty"`

	// The capacity information relating to snapshot or backup data.
	Snapshot CapacityInfo `json:"Snapshot,omitempty"`
}
