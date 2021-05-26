/* -----------------------------------------------------------------
 * model_manager_collection.go -
 *
 * DMTF ManagerCollection model.
 *
 * Author: Daniel Matthews
 *
 * Copyright 2020 Cray an HPE Company.  All Rights Reserved.
 *
 * ----------------------------------------------------------------- */

package openapi

// ManagerCollection - This schema defines a Collection of Manager resources.
type ManagerCollection struct {
	OdataContext string `json:"@odata.context,omitempty"`

	OdataEtag string `json:"@odata.etag,omitempty"`

	OdataId string `json:"@odata.id"`

	OdataType string `json:"@odata.type"`

	// Contains the members of this collection.
	Members []map[string]interface{} `json:"Members"`

	MembersodataCount int `json:"Members@odata.count"`

	MembersodataNextLink map[string]interface{} `json:"Members@odata.nextLink,omitempty"`

	Name string `json:"Name"`

	Oem map[string]interface{} `json:"Oem,omitempty"`
}
