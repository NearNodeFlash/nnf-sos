/* -----------------------------------------------------------------
 * model_event_destination.go-
 *
 * Event destination model
 *
 * Author: Tim Morneau
 *
 * © Copyright 2020 Hewlett Packard Enterprise Development LP
 *
 * ----------------------------------------------------------------- */

package openapi

type EventDestination struct {
	OdataContext string `json:"@odata.context,omitempty"`

	OdataEtag string `json:"@odata.etag,omitempty"`

	OdataId string `json:"@odata.id"`

	OdataType string `json:"@odata.type"`

	// Actions shall contain the actions for this resource
	Actions map[string]interface{} `json:"Actions,omitempty"`

	// Context - This can be provided at registration time
	Context string `json:"Context,omitempty"`

	// DeliveryRetryPolicy - definition of the retry policy for this registration
	// RetryForever - The subscription is not suspended or terminated, and attempts at delivery of future events shall continue even after the maximum number of retries is reached.
	// SuspendRetries - The subscription is suspended after the maximum number of retries is reached.
	// TerminateAfterRetries - The subscription is terminated after the maximum number of retries is reached.
	DeliveryRetryPolicy string `json:"DeliveryRetryPolicy,omitempty"`

	// Description of the Assembly.
	Description string `json:"Description,omitempty"`

	// Destination - uri of the destination event receiver.
	Destination string `json:"Destination,omitempty"`

	// EventFormatType fThis property shall indicate the content types of the message that this service sends to the EventDestination.  If this property is not present, the EventFormatType shall be assumed to be Event.
	EventFormatType int `json:"EventFormatType,omitempty"`

	// Deprecated - kept for reference. The value of this property shall indicate the the content types of the message that this service will send to the EventDestination.  If this property is not present, the EventFormatType shall be assumed to be Event.
	EventTypes int `json:"EventTypes,omitempty"`

	// HttpHeaders - An array of settings for HTTP headers, such as authorization information.  This array is null or an empty array in responses.  An empty array is the preferred return value on read operations.
	HttpHeaders map[string]interface{} `json:"HttpHeaders,omitempty"`

	// Id for the event.
	Id string `json:"Id,omitempty"`

	// IncludeOriginOfCondition - An indication of whether the events subscribed to will also include the entire resource or object referenced the OriginOfCondition property in the event payload.
	IncludeOriginOfCondition bool `json:"IncludeOriginOfCondition,omitempty"`

	// MessageIds - The list of MessageIds that the service sends.  If this property is absent or the array is empty, events with any MessageId are sent to the subscriber.
	MessageIds []string `json:"MessageIds,omitempty"`

	// MetricReportDefinitions - A list of metric report definitions for which the service only sends related metric reports.  If this property is absent or the array is empty, metric reports that originate from any metric report definition are sent to the subscriber.
	MetricReportDefinitions []interface{} `json:"MetricReportDefinitions,omitempty"`

	// MetricReportDefinitionOdataCount - The list of MessageIds that the service sends.  If this property is absent or the array is empty, events with any MessageId are sent to the subscriber.
	MetricReportDefinitionOdataCount int `json:"MetricReportDefinitions@odata.count,omitempty"`

	// Name of the event destination resource.
	Name string `json:"Name,omitempty"`

	Oem map[string]interface{} `json:"Oem,omitempty"`

	// OriginResources - his property shall specify an array of Resources, Resource Collections, or Referenceable Members that are the only allowable values for the OriginOfCondition property within an EventRecord that the service sends to the subscriber.  The service shall not send events that originate from Resources, Resource Collections, or Referenceable Members, and that this array does not contain, to the subscriber.  If this property is absent or the array is empty, the service shall send events that originate from any Resource, Resource Collection, or Referenceable Member to the subscriber.
	OriginResources []interface{} `json:"OriginResources,omitempty"`

	// OriginResourcesOdataCount - The number of OriginResources in the array.
	OriginResourcesOdataCount int `json:"OriginResources@odata.count,omitempty"`

	// RegistryPrefixes - This property shall contain the array of the prefixes of the Message Registries that contain the MessageIds in the Events that shall be sent to the EventDestination.  If this property is absent or the array is empty, the service shall send events with MessageIds from any Message Registry.
	RegistryPrefixes []string `json:"RegistryPrefixes,omitempty"`

	// ResourceTypes - This property shall specify an array of Resource Type values.  When an event is generated, if the OriginOfCondition's Resource Type matches a value in this array, the event shall be sent to the event destination (unless it would be filtered by other property conditions such as RegistryPrefix).  If this property is absent or the array is empty, the service shall send Events from any Resource type to the subscriber.  This property shall contain only the general namespace for the type and not the versioned value.  For example, it shall not contain Task.v1_2_0.Task and instead shall contain Task.  To specify that a client is subscribing to metric reports, the EventTypes property should include `MetricReport`.
	ResourceTypes []string `json:"ResourceTypes,omitempty"`

	// SNMP - This property shall contain the settings for an SNMP event destination..
	SNMP string `json:"SNMP,omitempty"`

	// Status - Event Destination Resource status.
	Status map[string]interface{} `json:"Status,omitempty"`

	// SubordinateResources - This property shall indicate whether the subscription is for events in the OriginResources array and its subordinate Resources.  If `true` and the OriginResources array is specified, the subscription is for events in the OriginResources array and its subordinate Resources.  Note that Resources associated through the Links section are not considered subordinate.  If `false` and the OriginResources array is specified, the subscription shall be for events in the OriginResources array only.  If the OriginResources array is not present, this property shall have no relevance.
	SubordinateResources bool `json:"SubordinateResources,omitempty"`

	// SubscriptionType - This property shall indicate the type of subscription for events.  If this property is not present, the SubscriptionType shall be assumed to be RedfishEvent.
	SubscriptionType string `json:"SubscriptionType,omitempty"`
}
