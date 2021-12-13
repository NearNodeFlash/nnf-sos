package event

import (
	"net/http"
)

type Api interface {
	Initialize() error

	RedfishV1EventServiceGet(w http.ResponseWriter, r *http.Request)
	RedfishV1EventServiceEventSubscriptionsGet(w http.ResponseWriter, r *http.Request)
	RedfishV1EventServiceEventSubscriptionsPost(w http.ResponseWriter, r *http.Request)
	RedfishV1EventServiceEventSubscriptionIdGet(w http.ResponseWriter, r *http.Request)
	RedfishV1EventServiceEventSubscriptionIdDelete(w http.ResponseWriter, r *http.Request)
	RedfishV1EventServiceEventsGet(w http.ResponseWriter, r *http.Request)
	RedfishV1EventServiceEventEventIdGet(w http.ResponseWriter, r *http.Request)
}
