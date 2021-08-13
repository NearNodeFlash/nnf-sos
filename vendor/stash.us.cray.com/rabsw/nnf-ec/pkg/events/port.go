package events

type PortType string

const (
	Unknown_PortType     PortType = "Unknown"
	USP_PortType                  = "USP"
	DSP_PortType                  = "DSP"
	Interswitch_PortType          = "Interswitch"
)

type PortEventType string

const (
	Unknown_PortEventType          PortEventType = "Unknown"
	Up_PortEventType                             = "Up"
	Down_PortEventType                           = "Down"
	AttributeChanged_PortEventType               = "Changed"
)

type PortLinkState string

const (
	Active_PortLinkState   PortLinkState = "Active"
	Degraded_PortLinkState               = "Degraded"
	Inactive_PortLinkState               = "Inactive"
)

type PortConfiguration string

const (
	Ready_PortConfiguration    PortConfiguration = "Ready"
	NotReady_PortConfiguration                   = "NotReady"
)

// PortEvent -
type PortEvent struct {
	EventType PortEventType
	PortType  PortType

	FabricId string
	SwitchId string
	PortId   string

	LinkState     PortLinkState
	Configuration PortConfiguration
}

// PortEventHandlerFunc
type PortEventHandlerFunc func(PortEvent, interface{})

// PortEventSubscriber
type PortEventSubscriber struct {
	HandlerFunc PortEventHandlerFunc
	Data        interface{}
}

// PortEventManager -
var PortEventManager portEventManager

type portEventManager struct {
	subscribers []PortEventSubscriber
}

// Subscribe -
func (mgr *portEventManager) Subscribe(s PortEventSubscriber) {
	mgr.subscribers = append(mgr.subscribers, s)
}

// Publish
func (mgr *portEventManager) Publish(event PortEvent) {
	for _, s := range mgr.subscribers {
		s.HandlerFunc(event, s.Data)
	}
}
