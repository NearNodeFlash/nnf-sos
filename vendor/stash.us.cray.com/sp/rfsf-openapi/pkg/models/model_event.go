/* -----------------------------------------------------------------
 * model_event.go-
 *
 * Event Manage the events for incoming requests and receive dbus events from
 * element controllers via dbus.
 *
 * Author: Tim Morneau
 *
 * © Copyright 2020 Hewlett Packard Enterprise Development LP
 *
 * ----------------------------------------------------------------- */

package openapi

type Event struct {
	OdataContext string `json:"@odata.context,omitempty"`

	OdataEtag string `json:"@odata.etag,omitempty"`

	OdataId string `json:"@odata.id"`

	OdataType string `json:"@odata.type"`

	// Actions shall contain the actions for this resource
	Actions map[string]interface{} `json:"Actions,omitempty"`

	// Context - This can be provided at registration time
	Context string `json:"Context,omitempty"`

	// Description of the Assembly.
	Description string `json:"Description,omitempty"`

	// Events - an array of event records
	Events []EventRecord `json:"Events,omitempty"`

	// Count for the Events array above.
	EventsOdataCount int `json:"Events@odata.count,omitempty"`

	// Id for the event.
	Id string `json:"Id,omitempty"`

	// Name of the Assembly.
	Name string `json:"Name,omitempty"`

	Oem map[string]interface{} `json:"Oem,omitempty"`
}
