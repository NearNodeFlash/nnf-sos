package fabric

import (
	"stash.us.cray.com/rabsw/ec"
)

// Router contains all the Redfish / Swordfish API calls that are hosted by
// the NNF module. All handler calls are of the form RedfishV1{endpoint}, where
// and endpoint is unique to the RF/SF caller.
//
// Router calls must reflect the same function name as the RF/SF API, as the
// element controller will perform a 1:1 function call based on the RF/SF
// caller's name.

// DefaultApiRouter -
type DefaultApiRouter struct {
	servicer   Api
	controller SwitchtecControllerInterface
}

// NewDefaultApiRouter -
func NewDefaultApiRouter(s Api, c SwitchtecControllerInterface) ec.Router {
	return &DefaultApiRouter{servicer: s, controller: c}
}

// Name -
func (*DefaultApiRouter) Name() string {
	return "Fabric Manager"
}

// Init -
func (r *DefaultApiRouter) Init() error {
	return Initialize(r.controller)
}

// Start -
func (r *DefaultApiRouter) Start() error {
	return Start()
}

// Routes -
func (r *DefaultApiRouter) Routes() ec.Routes {
	s := r.servicer
	return ec.Routes{
		{
			Name:        "RedfishV1FabricsGet",
			Method:      ec.GET_METHOD,
			Path:        "/redfish/v1/Fabrics",
			HandlerFunc: s.RedfishV1FabricsGet,
		},
		{
			Name:        "RedfishV1FabricsFabricIdGet",
			Method:      ec.GET_METHOD,
			Path:        "/redfish/v1/Fabrics/{FabricId}",
			HandlerFunc: s.RedfishV1FabricsFabricIdGet,
		},
		{
			Name:        "RedfishV1FabricsFabricIdSwitchesGet",
			Method:      ec.GET_METHOD,
			Path:        "/redfish/v1/Fabrics/{FabricId}/Switches",
			HandlerFunc: s.RedfishV1FabricsFabricIdSwitchesGet,
		},
		{
			Name:        "RedfishV1FabricsFabricIdSwitchesSwitchIdGet",
			Method:      ec.GET_METHOD,
			Path:        "/redfish/v1/Fabrics/{FabricId}/Switches/{SwitchId}",
			HandlerFunc: s.RedfishV1FabricsFabricIdSwitchesSwitchIdGet,
		},
		{
			Name:        "RedfishV1FabricsFabricIdSwitchesSwitchIdPortsGet",
			Method:      ec.GET_METHOD,
			Path:        "/redfish/v1/Fabrics/{FabricId}/Switches/{SwitchId}/Ports",
			HandlerFunc: s.RedfishV1FabricsFabricIdSwitchesSwitchIdPortsGet,
		},
		{
			Name:        "RedfishV1FabricsFabricIdSwitchesSwitchIdPortsPortIdGet",
			Method:      ec.GET_METHOD,
			Path:        "/redfish/v1/Fabrics/{FabricId}/Switches/{SwitchId}/Ports/{PortId}",
			HandlerFunc: s.RedfishV1FabricsFabricIdSwitchesSwitchIdPortsPortIdGet,
		},
		{
			Name:        "RedfishV1FabricsFabricIdEndpointsGet",
			Method:      ec.GET_METHOD,
			Path:        "/redfish/v1/Fabrics/{FabricId}/Endpoints",
			HandlerFunc: s.RedfishV1FabricsFabricIdEndpointsGet,
		},
		{
			Name:        "RedfishV1FabricsFabricIdEndpointsEndpointIdGet",
			Method:      ec.GET_METHOD,
			Path:        "/redfish/v1/Fabrics/{FabricId}/Endpoints/{EndpointId}",
			HandlerFunc: s.RedfishV1FabricsFabricIdEndpointsEndpointIdGet,
		},
		{
			Name:        "RedfishV1FabricsFabricIdEndpointGroupsGet",
			Method:      ec.GET_METHOD,
			Path:        "/redfish/v1/Fabrics/{FabricId}/EndpointGroups",
			HandlerFunc: s.RedfishV1FabricsFabricIdEndpointGroupsGet,
		},
		{
			Name:        "RedfishV1FabricsFabricIdEndpointGroupsEndpointGroupIdGet",
			Method:      ec.GET_METHOD,
			Path:        "/redfish/v1/Fabrics/{FabricId}/EndpointGroups/{EndpointGroupId}",
			HandlerFunc: s.RedfishV1FabricsFabricIdEndpointGroupsEndpointGroupIdGet,
		},
		{
			Name:        "RedfishV1FabricsFabricIdConnectionsGet",
			Method:      ec.GET_METHOD,
			Path:        "/redfish/v1/Fabrics/{FabricId}/Connections",
			HandlerFunc: s.RedfishV1FabricsFabricIdConnectionsGet,
		},
		{
			Name:        "RedfishV1FabricsFabricIdConnectionsConnectionIdGet",
			Method:      ec.GET_METHOD,
			Path:        "/redfish/v1/Fabrics/{FabricId}/Connections/{ConnectionId}",
			HandlerFunc: s.RedfishV1FabricsFabricIdConnectionsConnectionIdGet,
		},
		{
			Name:        "RedfishV1FabricsFabricIdConnectionsConnectionIdPatch",
			Method:      ec.PATCH_METHOD,
			Path:        "/redfish/v1/Fabrics/{FabricId}/Connections/{ConnectionId}",
			HandlerFunc: s.RedfishV1FabricsFabricIdConnectionsConnectionIdPatch,
		},
	}
}
