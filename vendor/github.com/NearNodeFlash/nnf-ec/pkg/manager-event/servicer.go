/*
 * Copyright 2020, 2021, 2022 Hewlett Packard Enterprise Development LP
 * Other additional copyright holders may be indicated within.
 *
 * The entirety of this work is licensed under the Apache License,
 * Version 2.0 (the "License"); you may not use this file except
 * in compliance with the License.
 *
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package event

import (
	"fmt"
	"net/http"

	sf "github.com/NearNodeFlash/nnf-ec/pkg/rfsf/pkg/models"

	. "github.com/NearNodeFlash/nnf-ec/pkg/common"
	ec "github.com/NearNodeFlash/nnf-ec/pkg/ec"
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
