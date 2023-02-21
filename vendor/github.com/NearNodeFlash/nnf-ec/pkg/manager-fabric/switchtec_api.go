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

package fabric

import (
	"github.com/NearNodeFlash/nnf-ec/internal/switchtec/pkg/switchtec"
)

type SwitchtecControllerInterface interface {
	Open(path string) (SwitchtecDeviceInterface, error)
}

type SwitchtecDeviceInterface interface {
	// these are ugly getters for the nvme devices to get attributes
	// about the device interface. Not sure how to make this less crap
	// without adding an unnecessarily large abstraction. I'll keep it
	// for now until something better is found.
	Device() *switchtec.Device
	Path() *string

	Close()

	Identify() (int32, error)

	GetFirmwareVersion() (string, error)
	GetModel() (string, error)
	GetManufacturer() (string, error)
	GetSerialNumber() (string, error)

	GetPortStatus() ([]switchtec.PortLinkStat, error)
	GetPortMetrics() (PortMetrics, error)
	GetEvents() ([]switchtec.GfmsEvent, error)

	EnumerateEndpoint(uint8, func(epPort *switchtec.DumpEpPortDevice) error) error
	ResetEndpoint(uint16) error

	Bind(uint8, uint8, uint16) error
}

type PortMetric struct {
	PhysPortId uint8

	// NOTE: This is a terrible definition in the switchtec package - these are raw Tx/Rx counter values
	// that can be _used_ for calculating the bandwidth. Think about renaming this in the switchtec package
	// to something like "PortMetrics"
	switchtec.BandwidthCounter
}

type PortMetrics []PortMetric
