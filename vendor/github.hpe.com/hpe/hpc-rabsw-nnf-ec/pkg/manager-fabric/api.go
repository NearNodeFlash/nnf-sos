/*
 * Near Node Flash Fabric API
 *
 * This file contains the API interface for the Near-Node Flash
 * Element Controller. Each NNF implementation must define these
 * methods. Please keep the names consisitent with the Redfish
 * API definitions.
 *
 * Author: Nate Roiger
 *
 * Copyright 2020 Hewlett Packard Enterprise Development LP
 */

package fabric

import (
	"net/http"
)

// Api - defines an interface for Near-Node Flash related methods
type Api interface {
	RedfishV1FabricsGet(w http.ResponseWriter, r *http.Request)
	RedfishV1FabricsFabricIdGet(w http.ResponseWriter, r *http.Request)

	RedfishV1FabricsFabricIdSwitchesGet(w http.ResponseWriter, r *http.Request)
	RedfishV1FabricsFabricIdSwitchesSwitchIdGet(w http.ResponseWriter, r *http.Request)

	RedfishV1FabricsFabricIdSwitchesSwitchIdPortsGet(w http.ResponseWriter, r *http.Request)
	RedfishV1FabricsFabricIdSwitchesSwitchIdPortsPortIdGet(w http.ResponseWriter, r *http.Request)

	RedfishV1FabricsFabricIdEndpointsGet(w http.ResponseWriter, r *http.Request)
	RedfishV1FabricsFabricIdEndpointsEndpointIdGet(w http.ResponseWriter, r *http.Request)

	RedfishV1FabricsFabricIdEndpointGroupsGet(w http.ResponseWriter, r *http.Request)
	RedfishV1FabricsFabricIdEndpointGroupsEndpointGroupIdGet(w http.ResponseWriter, r *http.Request)

	RedfishV1FabricsFabricIdConnectionsGet(w http.ResponseWriter, r *http.Request)
	RedfishV1FabricsFabricIdConnectionsConnectionIdGet(w http.ResponseWriter, r *http.Request)
	RedfishV1FabricsFabricIdConnectionsConnectionIdPatch(w http.ResponseWriter, r *http.Request)
}
