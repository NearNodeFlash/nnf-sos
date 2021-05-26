/* -----------------------------------------------------------------
 * model_event_service.go-
 *
 * Model for the main event service resource.
 *
 * Author: Tim Morneau
 *
 * © Copyright 2020 Hewlett Packard Enterprise Development LP
 *
 * ----------------------------------------------------------------- */

package openapi

type EventService struct {
	OdataContext string `json:"@odata.context,omitempty"`

	OdataEtag string `json:"@odata.etag,omitempty"`

	OdataId string `json:"@odata.id"`

	OdataType string `json:"@odata.type"`

	// Actions shall contain the actions for this resource
	Actions map[string]interface{} `json:"Actions,omitempty"`

	// Context - This can be provided at registration time
	Context string `json:"Context,omitempty"`

	// DeliveryRetryAttempts - This is the number of attempts an event posting is retried before the subscription is terminated.
	// This retry is at the service level, meaning the HTTP POST to the Event Destination was returned by the HTTP operation as
	// unsuccessful (4xx or 5xx return code) or an HTTP timeout occurred this many times before the Event Destination subscription is terminated.

	DeliveryRetryAttempts int `json:"DeliveryRetryAttempts,omitempty"`

	// This represents the number of seconds between retry attempts for sending any given Event.
	DeliveryRetryIntervalSeconds int `json:"DeliveryRetryIntervalSeconds,omitempty"`

	// Description of the Assembly.
	Description string `json:"Description,omitempty"`

	// EventFormatTypes - Indicates the content types of the message that this service can send to the event destination.
	// todo: need to create an enumeration list of these types "Event", "MetricReport", others.
	EventFormatTypes []string `json:"EventFormatTypes,omitempty"`

	// EventTypesForSubscription - This is the types of Events that can be subscribed to.
	EventTypesForSubscription []string `json:"EventTypesForSubscription,omitempty"`

	// Id for the event.
	Id string `json:"Id,omitempty"`

	// IncludeOriginOfConditionSupported - An indication of whether the service supports including the resource payload of the origin of condition in the event payload.
	IncludeOriginOfConditionSupported bool `json:"IncludeOriginOfConditionSupported,omitempty"`

	// Name of the Assembly.
	Name string `json:"Name,omitempty"`

	Oem map[string]interface{} `json:"Oem,omitempty"`

	// RegistryPrefixes - A list of the Prefixes of the Message Registries that can be used for the RegistryPrefix property on a subscription.
	RegistryPrefixes []string `json:"RegistryPrefixes,omitempty"`

	// ResourceTypes - A list of @odata.type values (Schema names) that can be specified in a ResourceType on a subscription.
	ResourceTypes []string `json:"ResourceTypes,omitempty"`

	// SMTP - Settings for SMTP event delivery.
	SMTP map[string]interface{} `json:"SMTP,omitempty"`

	// SSEFilterPropertiesSupported - Contains a set of properties that indicate which properties are supported in the $filter query parameter for the ServerSentEventUri.
	SSEFilterPropertiesSupported map[string]interface{} `json:"SSEFilterPropertiesSupported,omitempty"`

	// ServerSentEventUri - Settings for SMTP event delivery.
	ServerSentEventUri string `json:"ServerSentEventUri,omitempty"`

	// ServiceEnabled - An indication of whether this service is enabled..
	ServiceEnabled bool `json:"ServiceEnabled,omitempty"`

	// Status - Status of the EventServiceResource - if there is a problem with the event service see this status.
	Status bool `json:"Status,omitempty"`

	Subscriptions []string `json:"Subscriptions,omitempty"`
}
