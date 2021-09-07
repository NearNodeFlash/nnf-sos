package event

import (
	"fmt"

	sf "stash.us.cray.com/rabsw/rfsf-openapi/pkg/models"
)

// Event Manager:
//
// The Event Manager is an implementation of the Redfish / Swordfish Event Service
// It contains attributes describing the current state of the service such as status
// and retry information.
//
// Subscriptions and Subscribers:
//
// An Event Subscription represents a event destination in a client. Clients may
// subscribe to certain events that they are interested in by specifying the event type.
// When an event occurs, the Event Manager will notify any subscribers that were registerd
// to receive this event. A user supplied context is returned with the event for the
// subscriber (i.e user_arg style)
//
// Messages and Message Registries:
//
// A Message Registry is an array of Messages and their attributes. Each entry is
// defined by a MessageId, and contains a description, severity, and number and type of
// arguments, and a proposed resolution (if any). Message arguments substitute wildcard
// varabiles in the message so the client can return useful information for the user.
// Redfish Events, Errors, and Log Records have a unified format.
//
// Example:
//
// Say a Port Down event has occurred, identified by a fabric switch port.
// 		{
// 			"@odata.id": "/redfish/v1/EventService/Events/1",
// 			"Events": [
// 				{
//					"EventType": "Alert",
//					"EventId": "ABCD123",
//					"Severity": "Warning",
//					"Message": "Fabirc switch port is down",
//					"MessageId": "Alert.1.0.FabircSwitchPortDown",
//					"MessageArgs": [
//						"Rabbit", "0", "Port 0"
//					],
//					"OriginOfCondition": {
//						"@odata.id": "/redfish/v1/Fabrics/Rabbit/Switches/0/Ports/0"
//					}
//				}
//			]
//		}
//
// There should always be a Message Registry that is paried with a Message ID. In this case
// there would be a Message Registry with the following
//
//		{
//			"Id": "Alert.1.0.0",
//			"RegistryPrefix": "Alert",
//			"RegistryVersion": "1.0.0",
//			"Messages": {
//				"FabircSwitchPortDown": {
//					"Description": "A Port is down on Fabirc %1 Switch %2 Port %3",
//					"Message": "Port connectivity was lost on Fabric %1 Switch %2 Port %3",
//					"Severity": "Critical",
//					"NumberOfArgs": 3,
// 					"Resolution": "Reseat the device located in Port %3"
//				}
//			}
//		}
//
// [end example]
//
// Client Behavior:
//
// When a client receives an event (by creating a subscription with the event service), they
// first use the MessageId to find the corresponding registry, version, and message.
// In the above example, the MessageId is "Alert.1.0.FabircSwitchPortDown".  This is broken
// down by the client into three parts
// 		- Registry Prefix: 			"Alert"
//		- Major / Minor Version: 	"1.0"
//		- Message Identifier:		"FabircSwitchPortDown"
//
// The client must then use that information to scan the available registries and locate the
// correct Message Registry. The Event Service contains a link of the registires in use, so
// the client can scan those registries and do a lookup.
//
// There are a bunch of registries already created by DMTF: https://redfish.dmtf.org/registries/
//
// The client then populates the message for the end user by taking the message property and
// substituting in the arguments contained within the event.

var EventManager *manager

type manager struct {
	subscriptions []eventSubscription
}

type eventSubscription struct {
	id          string
	destination string
	types       []sf.EventEventType
	context     string
}

func (e *eventSubscription) OdataId() string { return fmt.Sprintf("/redfish/v1/EventService/%s", e.id) }
func (e *eventSubscription) Name() string    { return fmt.Sprintf("EventSubscription %s", e.id) }

func (m *manager) Initialize() error {
	EventManager = m
	return nil
}

func (*manager) Subscribe() error {
	return nil
}

func (*manager) Get(model *sf.EventServiceV170EventService) error {
	model.Id = "EventService"
	model.Name = "Event Service"
	model.ServiceEnabled = true
	model.EventTypesForSubscription = []sf.EventEventType{
		sf.RESOURCE_ADDED_EET,
		sf.RESOURCE_REMOVED_EET,
		sf.RESOURCE_UPDATED_EET,
		sf.STATUS_CHANGE_EET,
		sf.ALERT_EET,
	}

	model.Subscriptions = sf.OdataV4IdRef{OdataId: "/redfish/v1/EventService/Subscriptions"}

	return nil
}

func (m *manager) EventSubscriptionsGet(model *sf.EventDestinationCollectionEventDestinationCollection) error {

	model.MembersodataCount = int64(len(m.subscriptions))
	model.Members = make([]sf.OdataV4IdRef, model.MembersodataCount)
	for idx, sub := range m.subscriptions {
		model.Members[idx] = sf.OdataV4IdRef{OdataId: sub.OdataId()}
	}

	return nil
}

func (m *manager) EventSubscriptionsPost(model *sf.EventDestinationV190EventDestination) error {
	return nil
}

func (m *manager) EventSubscriptionsSubscriptionIdGet(id string, model *sf.EventDestinationV190EventDestination) error {
	return nil
}

func (m *manager) EventSubscriptionsSubscriptionIdDelete(id string) error {
	return nil
}

func (m *manager) EventsGet(model *sf.EventCollectionEventCollection) error {
	return nil
}

func (m *manager) EventsEventIdGet(id string, model *sf.EventV161Event) error {
	return nil
}
