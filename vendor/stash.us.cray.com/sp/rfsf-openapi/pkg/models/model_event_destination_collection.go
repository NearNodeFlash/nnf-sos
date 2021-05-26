/* -----------------------------------------------------------------
 * model_event_destination_collection.go-
 *
 * Model for the event destination resource array.
 *
 * Author: Tim Morneau
 *
 * © Copyright 2020 Hewlett Packard Enterprise Development LP
 *
 * ----------------------------------------------------------------- */

package openapi

type EventDestinationCollection struct {
	OdataContext string `json:"@odata.context,omitempty"`

	OdataEtag string `json:"@odata.etag,omitempty"`

	OdataId string `json:"@odata.id"`

	OdataType string `json:"@odata.type"`

	Description string `json:"Description,omitempty"`

	// Members - The members of this collection.
	Members []EventDestination `json:"Members,omitempty"`

	// MembersOdataCount - This is the types of Events that can be subscribed to.
	MembersOdataCount int `json:"Members@odata.count,omitempty"`

	// MembersOdataNextLink - This is the types of Events that can be subscribed to.
	MembersOdataNextLink string `json:"Members@odata.nextLink,omitempty"`

	Name string `json:"Name,omitempty"`

	Oem map[string]interface{} `json:"Oem,omitempty"`
}
