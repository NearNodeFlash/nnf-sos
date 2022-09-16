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

package api

import openapi "github.com/NearNodeFlash/nnf-ec/pkg/rfsf/pkg/common"

// FabricApi - Presents an API into the fabric outside of the fabric manager
// TODO: This should be obsolete - the NVMe Namespace Manager can
//
//	include the Fabric Manager (but NOT the other way around!!! Go doesn't
//	support circular bindings.
type FabricControllerApi interface {
	// Locates the index of the Downstream Port DSP within the Fabric Controller's list of all Downstream Endpoints, regardless of current endpoint status.
	GetDownstreamPortRelativePortIndex(switchId, portId string) (int, error)

	FindDownstreamEndpoint(portId, functionId string) (string, error)

	GetPortPartLocation(switchId, portId string) (*openapi.PartLocation, error)
}

// FabricDeviceControllerApi defines the interface for controlling a device on the fabric.
type FabricDeviceControllerApi interface {
}

var FabricController FabricControllerApi

func RegisterFabricController(f FabricControllerApi) {
	FabricController = f
}
