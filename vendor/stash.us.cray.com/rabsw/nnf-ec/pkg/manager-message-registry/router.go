package messageregistry

import (
	ec "stash.us.cray.com/rabsw/nnf-ec/pkg/ec"
)

type DefaultApiRouter struct {
	servicer Api
}

func NewDefaultApiRouter(s Api) ec.Router {
	return &DefaultApiRouter{servicer: s}
}

func (*DefaultApiRouter) Name() string {
	return "Message Registries"
}

func (*DefaultApiRouter) Init() error {
	return MessageRegistryManager.Initialize()
}

func (*DefaultApiRouter) Start() error {
	return nil
}

func (r *DefaultApiRouter) Routes() ec.Routes {
	s := r.servicer
	return ec.Routes{
		{
			Name:        "RedfishV1RegistriesGet",
			Method:      ec.GET_METHOD,
			Path:        "/redfish/v1/Registries",
			HandlerFunc: s.RedfishV1RegistriesGet,
		},
		{
			Name:        "RedfishV1RegistriesRegistryIdGet",
			Method:      ec.GET_METHOD,
			Path:        "/redfish/v1/Registries/{RegistryId}",
			HandlerFunc: r.servicer.RedfishV1RegistriesRegistryIdGet,
		},
		{
			Name:        "RedfishV1RegistriesRegistryIdGetRegistryGet",
			Method:      ec.GET_METHOD,
			Path:        "/redfish/v1/Registries/{RegistryId}/Registry",
			HandlerFunc: s.RedfishV1RegistriesRegistryIdRegistryGet,
		},
	}
}
