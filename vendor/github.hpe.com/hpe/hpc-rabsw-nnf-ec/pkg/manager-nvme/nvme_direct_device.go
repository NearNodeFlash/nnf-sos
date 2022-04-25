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

package nvme

import (
	"fmt"

	"github.hpe.com/hpe/hpc-rabsw-nnf-ec/internal/switchtec/pkg/nvme"
	"github.hpe.com/hpe/hpc-rabsw-nnf-ec/pkg/api"
)

type DirectNvmeController struct {
	deviceRegexp string
}

func NewDirectDeviceNvmeController(deviceRegexp string) NvmeController {
	return &DirectNvmeController{deviceRegexp: deviceRegexp}
}

func (c *DirectNvmeController) NewNvmeDeviceController() NvmeDeviceController {
	return &DirectNvmeDeviceController{ctrl: c}
}

type DirectNvmeDeviceController struct {
	ctrl    *DirectNvmeController
	devices []string
}

func (c *DirectNvmeDeviceController) Initialize() error {

	devices, err := nvme.DeviceList(c.ctrl.deviceRegexp)
	if err != nil {
		return err
	}

	c.devices = devices

	return nil
}

func (c *DirectNvmeDeviceController) NewNvmeDevice(fabricId string, switchId string, portId string) (NvmeDeviceApi, error) {

	portIdx, err := api.FabricController.GetDownstreamPortRelativePortIndex(switchId, portId)
	if err != nil {
		return nil, err
	}

	if portIdx >= len(c.devices) {
		return nil, fmt.Errorf("Switch %s Port %s overflows device list (%d)", switchId, portId, portIdx)
	}

	device := &nvmeDirectDevice{
		cliDevice: cliDevice{
			path:    c.devices[portIdx],
			command: "nvme",
			pdfid:   0,
		},
	}

	return device, nil
}

func (c *DirectNvmeDeviceController) Close() error { return nil }

// Simple wrapper around the cli device
type nvmeDirectDevice struct {
	cliDevice
}

func (d *nvmeDirectDevice) IsDirectDevice() bool { return true }

func (d *nvmeDirectDevice) IdentifyController(controllerId uint16) (*nvme.IdCtrl, error) {
	return d.cliDevice.IdentifyController(controllerId)
}

func (d *nvmeDirectDevice) IdentifyNamespace(namespaceId nvme.NamespaceIdentifier) (*nvme.IdNs, error) {
	return d.cliDevice.IdentifyNamespace(namespaceId)
}

func (d *nvmeDirectDevice) ListSecondary() (*nvme.SecondaryControllerList, error) {
	return d.cliDevice.ListSecondary()
}

func (d *nvmeDirectDevice) AssignControllerResources(controllerId uint16, resourceType SecondaryControllerResourceType, numResources uint32) error {
	return d.cliDevice.AssignControllerResources(controllerId, resourceType, numResources)
}

func (d *nvmeDirectDevice) OnlineController(controllerId uint16) error {
	return d.cliDevice.OnlineController(controllerId)
}

func (d *nvmeDirectDevice) ListNamespaces(controllerId uint16) ([]nvme.NamespaceIdentifier, error) {
	return d.cliDevice.ListNamespaces(controllerId)
}

func (d *nvmeDirectDevice) ListAttachedControllers(namespaceId nvme.NamespaceIdentifier) ([]uint16, error) {
	return d.cliDevice.ListAttachedControllers(namespaceId)
}

func (d *nvmeDirectDevice) CreateNamespace(capacityBytes uint64, sectorSizeBytes uint64, sectorSizeIndex uint8) (nvme.NamespaceIdentifier, nvme.NamespaceGloballyUniqueIdentifier, error) {
	return d.cliDevice.CreateNamespace(capacityBytes, sectorSizeBytes, sectorSizeIndex)
}

func (d *nvmeDirectDevice) DeleteNamespace(namespaceId nvme.NamespaceIdentifier) error {
	return d.cliDevice.DeleteNamespace(namespaceId)
}

func (d *nvmeDirectDevice) AttachNamespace(namespaceId nvme.NamespaceIdentifier, controllers []uint16) error {
	return d.cliDevice.AttachNamespace(namespaceId, controllers)
}

func (d *nvmeDirectDevice) DetachNamespace(namespaceId nvme.NamespaceIdentifier, controllers []uint16) error {
	return d.cliDevice.DetachNamespace(namespaceId, controllers)
}

func (d *nvmeDirectDevice) GetNamespaceFeature(namespaceId nvme.NamespaceIdentifier) ([]byte, error) {
	panic("unimplemented")
}

func (*nvmeDirectDevice) SetNamespaceFeature(namespaceId nvme.NamespaceIdentifier, data []byte) error {
	panic("unimplemented")
}

func (d *nvmeDirectDevice) GetWearLevelAsPercentageUsed() (uint8, error) {
	return d.cliDevice.GetWearLevelAsPercentageUsed()
}
