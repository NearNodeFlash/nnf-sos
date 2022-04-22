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
