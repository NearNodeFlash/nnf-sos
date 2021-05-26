/* -----------------------------------------------------------------
 * model_event_record.go-
 *
 * Event record - the base piece provided for each event occurence and
 * registered via model_event.go
 *
 * Author: Tim Morneau
 *
 * © Copyright 2020 Hewlett Packard Enterprise Development LP
 *
 * ----------------------------------------------------------------- */
package openapi

// Note: EventRecords only exist in the context of an Event.
type EventRecord struct {
	// Actions shall contain the actions for this resource
	Actions map[string]interface{} `json:"Actions,omitempty"`

	// Context - This can be provided at registration time
	Context string `json:"Context,omitempty"`

	// EventId - identifier for the given event
	EventId string `json:"EventId,omitempty"`

	// EventTimeStamp - marker in time for this event
	EventTimestamp string `json:"EventTimeStamp,omitempty"`

	// EventType - the type of event
	EventType string `json:"EventType,omitempty"`

	// The value of this string shall uniquely identify the member within the collection.
	MemberId string `json:"MemberId,omitempty"`

	// Message -This property shall contain an optional human readable message.
	Message string `json:"Message,omitempty"`

	// MessageArgs - This array of message arguments are substituted for the arguments in the message when looked up in the message registry.
	MessageArgs []string `json:"MessageArgs,omitempty"`

	// MessageId - This is the key for this message which can be used to look up the message in a message registry.
	MessageId string `json:"MessageId,omitempty"`

	// Severity - This is the severity of the event.
	Severity string `json:"Severity,omitempty"`
}
