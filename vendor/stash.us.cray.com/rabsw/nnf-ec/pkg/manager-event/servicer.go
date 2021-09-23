package event

import (
	"fmt"
	"net/http"

	sf "stash.us.cray.com/rabsw/nnf-ec/pkg/rfsf/pkg/models"

	. "stash.us.cray.com/rabsw/nnf-ec/pkg/common"
	ec "stash.us.cray.com/rabsw/nnf-ec/pkg/ec"
)

type DefaultApiService struct {
	*manager
}

func NewDefaultApiService() Api {
	return &DefaultApiService{manager: &EventManager}
}

func (s *DefaultApiService) Initialize() error {
	return s.manager.Initialize()
}

func (s *DefaultApiService) RedfishV1EventServiceGet(w http.ResponseWriter, r *http.Request) {

	model := sf.EventServiceV170EventService{
		OdataId:   "/redfish/v1/EventService",
		OdataType: "#EventService.v1_7_0.EventService",
		Name:      "Event Service",
	}

	err := s.Get(&model)

	EncodeResponse(model, err, w)
}

func (s *DefaultApiService) RedfishV1EventServiceEventSubscriptionsGet(w http.ResponseWriter, r *http.Request) {

	model := sf.EventDestinationCollectionEventDestinationCollection{
		OdataId:   "/redfish/v1/EventService/Subscriptions",
		OdataType: "#EventDestinationCollection.v1_0_0.EventDestinationCollection",
		Name:      "Event Destination Collection",
	}

	err := s.EventSubscriptionsGet(&model)

	EncodeResponse(model, err, w)
}

func (s *DefaultApiService) RedfishV1EventServiceEventSubscriptionsPost(w http.ResponseWriter, r *http.Request) {

	model := sf.EventDestinationV190EventDestination{}

	if err := UnmarshalRequest(r, &model); err != nil {
		err = ec.NewErrBadRequest().WithError(err).WithCause("Failed to unmarshal request")
		EncodeResponse(model, err, w)
		return
	}

	err := s.EventSubscriptionsPost(&model)

	EncodeResponse(model, err, w)
}

func (s *DefaultApiService) RedfishV1EventServiceEventSubscriptionIdGet(w http.ResponseWriter, r *http.Request) {
	params := Params(r)
	subscriptionId := params["SubscriptionId"]

	model := sf.EventDestinationV190EventDestination{
		OdataId:   fmt.Sprintf("/redfish/v1/EventService/Subscriptions/%s", subscriptionId),
		OdataType: "#EventDestination.v1_9_0.EventDestination",
		Name:      "Event Destination",
	}

	err := s.EventSubscriptionsSubscriptionIdGet(subscriptionId, &model)

	EncodeResponse(model, err, w)
}

func (s *DefaultApiService) RedfishV1EventServiceEventSubscriptionIdDelete(w http.ResponseWriter, r *http.Request) {
	params := Params(r)
	subscriptionId := params["SubscriptionId"]

	err := s.EventSubscriptionsSubscriptionIdDelete(subscriptionId)

	EncodeResponse(nil, err, w)
}

func (s *DefaultApiService) RedfishV1EventServiceEventsGet(w http.ResponseWriter, r *http.Request) {

	model := sf.EventCollectionEventCollection{
		OdataId:   fmt.Sprintf("/redfish/v1/EventService/Events"),
		OdataType: "#EventCollection.v1_0_0.EventCollection",
		Name:      "Event Collection",
	}

	err := s.EventsGet(&model)

	EncodeResponse(model, err, w)
}

func (s *DefaultApiService) RedfishV1EventServiceEventEventIdGet(w http.ResponseWriter, r *http.Request) {
	params := Params(r)
	eventId := params["EventId"]

	model := sf.EventV161Event{
		OdataId:   fmt.Sprintf("/redfish/v1/EventService/Events/%s", eventId),
		OdataType: "#Event.v1_6_1.Event",
	}

	err := s.EventsEventIdGet(eventId, &model)

	EncodeResponse(model, err, w)
}
