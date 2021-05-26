package events

type PortType string

const (
	PORT_TYPE_UNKNOWN     PortType = "Unknown"
	PORT_TYPE_USP                  = "USP"
	PORT_TYPE_DSP                  = "DSP"
	PORT_TYPE_INTERSWITCH          = "Interswitch"
)

type PortEventType string

const (
	PORT_EVENT_UNKNOWN PortEventType = "Unknown"
	PORT_EVENT_UP                    = "Up"
	PORT_EVENT_DOWN                  = "Down"
	PORT_EVENT_READY                 = "Ready"
)

// PortEvent -
type PortEvent struct {
	FabricId  string
	SwitchId  string
	PortId    string
	PortType  PortType
	EventType PortEventType
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
