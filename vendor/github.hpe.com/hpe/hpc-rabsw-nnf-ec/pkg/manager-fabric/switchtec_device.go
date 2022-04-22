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
	"os"
	"strconv"

	"github.hpe.com/hpe/hpc-rabsw-nnf-ec/internal/switchtec/pkg/switchtec"
)

type SwitchtecController struct{}

func NewSwitchtecController() SwitchtecControllerInterface {
	return &SwitchtecController{}
}

func (SwitchtecController) Exists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func (SwitchtecController) Open(path string) (SwitchtecDeviceInterface, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, err
	}

	dev, err := switchtec.Open(path)
	return &SwitchtecDevice{dev: dev}, err
}

type SwitchtecDevice struct {
	dev *switchtec.Device
}

func (d *SwitchtecDevice) Device() *switchtec.Device {
	return d.dev
}

func (d *SwitchtecDevice) Path() *string {
	panic("Switchtec Device does not support Path getter")
}
func (d *SwitchtecDevice) Close() {
	d.dev.Close()
}

func (d *SwitchtecDevice) Identify() (int32, error) {
	return d.dev.Identify()
}

func (d *SwitchtecDevice) GetFirmwareVersion() (string, error) {
	return d.dev.GetFirmwareVersion()
}

func (d *SwitchtecDevice) GetModel() (string, error) {
	id, err := d.dev.GetDeviceId()

	return strconv.Itoa(int(id)), err
}

func (d *SwitchtecDevice) GetManufacturer() (string, error) {
	return "Microsemi", nil
}

func (d *SwitchtecDevice) GetSerialNumber() (string, error) {
	sn, err := d.dev.GetSerialNumber()
	return strconv.Itoa(int(sn)), err
}

func (d *SwitchtecDevice) GetPortStatus() ([]switchtec.PortLinkStat, error) {
	return d.dev.LinkStat()
}

func (d *SwitchtecDevice) GetPortMetrics() (PortMetrics, error) {
	portIds, counters, err := d.dev.BandwidthCounterAll(false)
	if err != nil {
		return nil, err
	}

	metrics := make(PortMetrics, len(portIds))
	for idx := range portIds {
		metrics[idx] = PortMetric{
			PhysPortId:       portIds[idx].PhysPortId,
			BandwidthCounter: counters[idx],
		}
	}

	return metrics, nil
}

func (d *SwitchtecDevice) GetEvents() ([]switchtec.GfmsEvent, error) {
	return d.dev.GetGfmsEvents()
}

func (d *SwitchtecDevice) EnumerateEndpoint(id uint8, f func(epPort *switchtec.DumpEpPortDevice) error) error {
	return d.dev.GfmsEpPortDeviceEnumerate(id, f)
}

func (d *SwitchtecDevice) Bind(hostPhysPortId, hostLogPortId uint8, pdfid uint16) error {
	return d.dev.Bind(uint8(d.dev.ID()), hostPhysPortId, hostLogPortId, pdfid)
}
