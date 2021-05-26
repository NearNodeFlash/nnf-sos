/* -----------------------------------------------------------------
 * model_filesystem_collection.go -
 *
 * SNIA FileSystemCollection model.
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

// FileSystemCollection - Contains a collection of references to FileSystem resource instances.
// Modeled from Swordfish schema: http://redfish.dmtf.org/schemas/swordfish/FileSystemCollection.json
type FileSystemCollection struct {
	OdataContext string `json:"@odata.context,omitempty"`

	OdataEtag string `json:"@odata.etag,omitempty"`

	OdataId string `json:"@odata.id"`

	OdataType string `json:"@odata.type"`

	Description string `json:"Description,omitempty"`

	Members []map[string]interface{} `json:"Members"`

	MembersodataCount int `json:"Members@odata.count"`

	MembersodataNextLink map[string]interface{} `json:"Members@odata.nextLink,omitempty"`

	Name string `json:"Name"`

	Oem map[string]interface{} `json:"Oem,omitempty"`
}
