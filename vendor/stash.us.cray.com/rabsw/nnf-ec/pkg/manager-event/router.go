package event

import (
	"stash.us.cray.com/rabsw/nnf-ec/pkg/ec"
)

// Router contains all the Redfish / Swordfish API calls for the Event Service
type DefaultApiRouter struct {
	servicer Api
}

func NewDefaultApiRouter(s Api) ec.Router {
	return &DefaultApiRouter{servicer: s}
}

func (*DefaultApiRouter) Name() string {
	return "Event Manager"
}

func (r *DefaultApiRouter) Init() error {
	return r.servicer.Initialize()
}

func (*DefaultApiRouter) Start() error {
	return nil
}

func (r *DefaultApiRouter) Routes() ec.Routes {
	s := r.servicer
	return ec.Routes{
		{
			Name:        "RedfishV1EventServiceGet",
			Method:      ec.GET_METHOD,
			Path:        "/redfish/v1/EventService",
			HandlerFunc: s.RedfishV1EventServiceGet,
		},

		/* ---------------------- Event Subscriptions ---------------------- */

		{
			Name:        "RedfishV1EventServiceEventSubscriptionsGet",
			Method:      ec.GET_METHOD,
			Path:        "/redfish/v1/EventService/Subscriptions",
			HandlerFunc: s.RedfishV1EventServiceEventSubscriptionsGet,
		},
		{
			Name:        "RedfishV1EventServiceEventSubscriptionsPost",
			Method:      ec.POST_METHOD,
			Path:        "/redfish/v1/EventService/Subscriptions",
			HandlerFunc: s.RedfishV1EventServiceEventSubscriptionsPost,
		},
		{
			Name:        "RedfishV1EventServiceEventSubscriptionIdGet",
			Method:      ec.GET_METHOD,
			Path:        "/redfish/v1/EventService/Subscriptions/{SubscriptionId}",
			HandlerFunc: s.RedfishV1EventServiceEventSubscriptionIdGet,
		},
		{
			Name:        "RedfishV1EventServiceEventSubscriptionIdDelete",
			Method:      ec.DELETE_METHOD,
			Path:        "/redfish/v1/EventService/Subscriptions/{SubscriptionId}",
			HandlerFunc: s.RedfishV1EventServiceEventSubscriptionIdDelete,
		},

		/* ---------------------------- Events ----------------------------- */

		{
			Name:        "RedfishV1EventServiceEventsGet",
			Method:      ec.GET_METHOD,
			Path:        "/redfish/v1/EventService/Events",
			HandlerFunc: s.RedfishV1EventServiceEventsGet,
		},
		{
			Name:        "RedfishV1EventServiceEventIdGet",
			Method:      ec.GET_METHOD,
			Path:        "/redfish/v1/EventService/Events/{EventId}",
			HandlerFunc: s.RedfishV1EventServiceEventEventIdGet,
		},
	}
}
