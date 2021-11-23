package event

import (
	"fmt"
	"regexp"

	sf "stash.us.cray.com/rabsw/nnf-ec/pkg/rfsf/pkg/models"
)

const (
	MaxNumEvents = 256
)

type Events = []Event

type Event struct {
	// The identifier that uniquely identifies the resource within the collection of similar resources.
	Id string `json:"Id"`

	// The time the event occurred.
	Timestamp string `json:"Timestamp,omitempty"`

	// The identifier that correlates events with the same root cause. If 0 , no other event is related to this event.
	GroupId string `json:"GroupId,omitempty"`

	// The identifier for the member within the collection.
	MemberId string `json:"MemberId,omitempty"`

	// The human-readable event message.
	Message string `json:"Message,omitempty"`

	// An array of message arguments that are substituted for the arguments in the message when looked up in the message registry.
	MessageArgs []string `json:"MessageArgs,omitempty"`

	// The identifier for the message.
	MessageId MessageId `json:"MessageId,omitempty"`

	// The severity of the message in this event.
	MessageSeverity sf.ResourceHealth `json:"MessageSeverity,omitempty"`

	// A link to the resource or object that originated the condition that caused the event to be generated.
	OriginOfCondition string `json:"OriginOfCondition,omitempty"`
}

func (e Event) OdataId() string {
	return fmt.Sprintf("redfish/v1/EventService/Events/%s", e.Id)
}

func (e Event) Is(event Event) bool {
	return e.MessageId == event.MessageId
}

func (e Event) CopyInto(model *sf.EventV161Event) {
	model.EventId = e.Id
	model.EventTimestamp = e.Timestamp
	model.EventGroupId = e.GroupId
	model.MemberId = e.MemberId
	model.Message = e.Message
	model.MessageArgs = e.MessageArgs
	model.MessageSeverity = e.MessageSeverity
	if len(e.OriginOfCondition) != 0 {
		model.OriginOfCondition = sf.OdataV4IdRef{OdataId: e.OriginOfCondition}
	}
}

func (e Event) Args(args ...*string) error {
	if len(args) > len(e.MessageArgs) {
		return fmt.Errorf("Requested arguments exceeds supplied arguments")
	}

	for idx, arg := range args {
		*arg = e.MessageArgs[idx]
	}

	return nil
}

type Resource interface {
	OdataId() string
}

// MessageId is an encoding of the Message Registry (Registry Prefix), Message Version, and Message Identifer
// The format is [Registry Prefix].[Major Version].[Minor Version].[Message Identifier]
//
// Example:
//		With a MessageId of "Alert.1.0.LanDisconnect"
//			The segment before the 1st period is the Message Registry Prefix: "Alert"
//			The segment between the 1st and 3rd period is the Registry Version: "1.0"
//				The segment between the 1st and 2nd period is the Major Version: "1"
//				The segment between the 2nd and 3rd period is the Minor Version: "0"
//			The segment after the 3rd period is the Message Identifier in the Registry: "LanDisconnect"
//
type MessageId string

func (m MessageId) RegistryPrefix() string  { return m.segment(0) }
func (m MessageId) RegistryVersion() string { return m.segment(1) }
func (m MessageId) Identifier() string      { return m.segment(2) }
func (m MessageId) String() string          { return string(m) }

func (m MessageId) segment(idx int) string {
	re := regexp.MustCompile(`^(\w+).(\d+.\d+).(\w+)`)
	return re.FindStringSubmatch(m.String())[idx]
}
