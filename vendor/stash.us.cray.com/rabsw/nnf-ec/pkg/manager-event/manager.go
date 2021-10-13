package event

import (
	"fmt"
	"strconv"

	ec "stash.us.cray.com/rabsw/nnf-ec/pkg/ec"
	msgreg "stash.us.cray.com/rabsw/nnf-ec/pkg/manager-message-registry"

	sf "stash.us.cray.com/rabsw/nnf-ec/pkg/rfsf/pkg/models"
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

var EventManager = manager{}

const (
	DefaultDeliveryRetryAttempts        = 5
	DefaultDeliveryRetryIntervalSeconds = 60
)

type manager struct {
	subscriptions []subscription
	events        Events

	// The maximum number of events the Event Manager maintains in the event log. The event log
	// will wrap and overwrite prior events if the number of events exceeds this value.
	maxEvents int

	// The total number of events recorded by the Event Manager since initialization.
	numEvents int

	deliveryRetryAttempts int

	deliveryRetryInvervalSeconds int
}

type subscription struct {
	id       string
	prefixes []string
	s        Subscription
	t        sf.EventDestinationV190SubscriptionType
}

func (e *subscription) OdataId() string { return fmt.Sprintf("/redfish/v1/EventService/%s", e.id) }
func (e *subscription) Name() string    { return fmt.Sprintf("EventSubscription %s", e.id) }

func (m *manager) Initialize() error {

	m.events = make([]Event, MaxNumEvents, MaxNumEvents)
	m.maxEvents = MaxNumEvents
	m.numEvents = 0

	m.deliveryRetryAttempts = DefaultDeliveryRetryAttempts
	m.deliveryRetryInvervalSeconds = DefaultDeliveryRetryIntervalSeconds

	return nil
}

// Subscribe will add the subscription to the Event Manager. When an event is published to the Event Manager
// (through the Publish() method), the Event Manager will broadcast the event to all registered subscriptions.
func (m *manager) Subscribe(s Subscription) {
	m.addSubscription(s, sf.OEM_EDV190ST)
}

// Publish will publish the provided event to all interested subscriptions. It will also add the event
// to the Event Managers list of historic events. The Event must contain a valid MessageId - that is
// to say the event's MessageId must be backed by an entry in the Message Registry.
func (m *manager) Publish(e Event) {
	e.Id = strconv.Itoa(m.numEvents)

	m.events[m.numEvents%m.maxEvents] = e
	m.numEvents++

	for _, s := range m.subscriptions {
		s.s.EventHandler(e)
	}
}

func (m *manager) addSubscription(s Subscription, t sf.EventDestinationV190SubscriptionType) {
	var sid = -1
	for _, s := range m.subscriptions {
		id, _ := strconv.Atoi(s.id)
		if sid <= id {
			sid = id
		}
	}

	sid = sid + 1

	m.subscriptions = append(m.subscriptions, subscription{
		id: fmt.Sprintf("%d", sid),
		s:  s,
		t:  t,
	})
}

func (m *manager) deleteSubscription(s *subscription) {
	for subIdx, sub := range m.subscriptions {
		if sub.id == s.id {
			m.subscriptions = append(m.subscriptions[:subIdx], m.subscriptions[subIdx+1:]...)
			break
		}
	}
}

func (m *manager) findSubscription(id string) *subscription {
	for idx := range m.subscriptions {
		if m.subscriptions[idx].id == id {
			return &m.subscriptions[idx]
		}
	}

	return nil
}

// Get
func (m *manager) Get(model *sf.EventServiceV170EventService) error {
	model.Id = "EventService"
	model.Name = "Event Service"
	model.ServiceEnabled = true
	model.DeliveryRetryAttempts = int64(m.deliveryRetryAttempts)
	model.DeliveryRetryIntervalSeconds = int64(m.deliveryRetryInvervalSeconds)

	model.RegistryPrefixes = make([]string, len(msgreg.RegistryFiles))
	for idx, reg := range msgreg.RegistryFiles {
		model.RegistryPrefixes[idx] = reg.Model.RegistryPrefix
	}

	model.Subscriptions = sf.OdataV4IdRef{OdataId: "/redfish/v1/EventService/Subscriptions"}

	return nil
}

// EventSubscriptionsGet
func (m *manager) EventSubscriptionsGet(model *sf.EventDestinationCollectionEventDestinationCollection) error {

	model.MembersodataCount = int64(len(m.subscriptions))
	model.Members = make([]sf.OdataV4IdRef, model.MembersodataCount)
	for idx, sub := range m.subscriptions {
		model.Members[idx] = sf.OdataV4IdRef{OdataId: sub.OdataId()}
	}

	return nil
}

// EventSubscriptionsPost
func (m *manager) EventSubscriptionsPost(model *sf.EventDestinationV190EventDestination) error {

	if (model.DeliveryRetryPolicy != sf.RETRY_FOREVER_EDV190DRP) && (model.DeliveryRetryPolicy != sf.TERMINATE_AFTER_RETRIES_EDV190DRP) {
		return ec.NewErrNotAcceptable().WithCause(fmt.Sprintf("retry policy %s is not supported by the event service", string(model.DeliveryRetryPolicy)))
	}

	m.addSubscription(RedfishSubscription{
		Context:             model.Context,
		Destination:         model.Destination,
		DeliveryRetryPolicy: model.DeliveryRetryPolicy,
	}, sf.REDFISH_EVENT_EDV190ST)

	return m.EventSubscriptionsSubscriptionIdGet(m.subscriptions[len(m.subscriptions)-1].id, model)
}

// EventSubscriptionsSubscriptionIdGet
func (m *manager) EventSubscriptionsSubscriptionIdGet(id string, model *sf.EventDestinationV190EventDestination) error {
	s := m.findSubscription(id)
	if s == nil {
		return ec.NewErrNotFound().WithCause(fmt.Sprintf("subscription %s not found", id))
	}

	// TODO: The subscription should populate more of the model

	model.Id = s.id
	model.SubscriptionType = s.t

	return nil
}

// EventSubscriptionsSubscriptionIdDelete
func (m *manager) EventSubscriptionsSubscriptionIdDelete(id string) error {
	s := m.findSubscription(id)
	if s == nil {
		return ec.NewErrNotFound().WithCause(fmt.Sprintf("subscription %s not found", id))
	}

	m.deleteSubscription(s)

	return nil
}

// EventsGet
func (m *manager) EventsGet(model *sf.EventCollectionEventCollection) error {

	count := m.numEvents
	start := 0
	if m.numEvents > m.maxEvents {
		count = m.maxEvents
		start = (m.numEvents % m.maxEvents)
	}

	model.MembersodataCount = int64(count)
	model.Members = make([]sf.OdataV4IdRef, model.MembersodataCount)
	for idx := range m.events[:count] {
		model.Members[idx] = sf.OdataV4IdRef{OdataId: m.events[(start+idx)%m.maxEvents].OdataId()}
	}

	return nil
}

// EventsEventIdGet
func (m *manager) EventsEventIdGet(id string, model *sf.EventV161Event) error {
	idx, err := strconv.Atoi(id)
	if err != nil {
		return ec.NewErrBadRequest().WithError(err).WithCause(fmt.Sprintf("event id %s is non-integer type", id))
	}

	if m.numEvents < idx {
		return ec.NewErrNotFound().WithCause(fmt.Sprintf("event id %s not found", id))
	}

	m.events[idx%m.maxEvents].CopyInto(model)

	return nil
}
