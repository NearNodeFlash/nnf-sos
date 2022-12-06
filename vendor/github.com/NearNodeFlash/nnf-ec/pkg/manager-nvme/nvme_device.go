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
	"encoding/binary"
	"fmt"
	"unsafe"

	fabric "github.com/NearNodeFlash/nnf-ec/pkg/manager-fabric"

	"github.com/NearNodeFlash/nnf-ec/internal/switchtec/pkg/nvme"
)

type SwitchtecNvmeController struct{}

func NewSwitchtecNvmeController() NvmeController {
	return &SwitchtecNvmeController{}
}

func (SwitchtecNvmeController) NewNvmeDeviceController() NvmeDeviceController {
	return &SwitchtecNvmeDeviceController{}
}

type SwitchtecNvmeDeviceController struct{}

func (SwitchtecNvmeDeviceController) Initialize() error { return nil }
func (SwitchtecNvmeDeviceController) Close() error      { return nil }

func (SwitchtecNvmeDeviceController) NewNvmeDevice(fabricId, switchId, portId string) (NvmeDeviceApi, error) {
	return newNvmeDevice(fabricId, switchId, portId)
}

type nvmeDevice struct {
	dev   *nvme.Device
	pdfid uint16
}

func newNvmeDevice(fabricId, switchId, portId string) (NvmeDeviceApi, error) {
	sdev := fabric.GetSwitchDevice(fabricId, switchId)
	if sdev == nil {
		return nil, fmt.Errorf("NVMe Device: Switchtec device not found: Fabric: %s Switch: %s", fabricId, switchId)
	}

	pdfid, err := fabric.GetPortPDFID(fabricId, switchId, portId, 0)
	if err != nil {
		return nil, fmt.Errorf("NVMe Device: Failed to retrieve PDFID: Fabric: %s Switch: %s Port %s Error: %w", fabricId, switchId, portId, err)
	}

	dev, err := nvme.Connect(sdev, pdfid)
	if err != nil {
		return nil, fmt.Errorf("NVMe Device: Failed to connect NVMe device: PDFID: %d Error: %w", pdfid, err)
	}

	return &nvmeDevice{dev: dev, pdfid: pdfid}, nil
}

// IsDirectDevice -
func (d *nvmeDevice) IsDirectDevice() bool { return false }

// IdentifyController -
func (d *nvmeDevice) IdentifyController(controllerId uint16) (*nvme.IdCtrl, error) {
	return d.dev.IdentifyController()
}

// IdentifyNamespace -
func (d *nvmeDevice) IdentifyNamespace(namespaceId nvme.NamespaceIdentifier) (*nvme.IdNs, error) {
	// NVMe specification defines two ways to identify namespaces.
	// 1) Through Active Namespace Management, returns Identify Namespace data structure for the specific
	//    NSID active on the controller, or the common namespace capabilities structure.
	// 2) Allocated Namespace Management, returns Identify Namespace data structure for the specified
	//    allocated NSID, regardless of its attach state to a controller.
	//
	// We prefer to manage namespaces regardless of their attach state to a particular controller, so
	// option 2 is preferred except in cases of the common namespace identifier (-1), in which case
	// we need to use option 1.

	present := true // Force identify namespace even if not attached to the particular controller
	if namespaceId == CommonNamespaceIdentifier {
		present = false
	}

	return d.dev.IdentifyNamespace(uint32(namespaceId), present)
}

// ListSecondary -
func (d *nvmeDevice) ListSecondary() (*nvme.SecondaryControllerList, error) {
	return d.dev.ListSecondary(0, 0)
}

// AssignControllerResources -
func (d *nvmeDevice) AssignControllerResources(controllerId uint16, resourceType SecondaryControllerResourceType, numResources uint32) error {
	resourceTypeMap := map[SecondaryControllerResourceType]nvme.VirtualManagementResourceType{
		VQResourceType: nvme.VQResourceType,
		VIResourceType: nvme.VIResourceType,
	}

	return d.dev.VirtualMgmt(controllerId, nvme.SecondaryAssignAction, resourceTypeMap[resourceType], numResources)
}

// OnlineController -
func (d *nvmeDevice) OnlineController(controllerId uint16) error {
	return d.dev.VirtualMgmt(controllerId, nvme.SecondaryOnlineAction, nvme.VQResourceType /*Ignored for OnlineAction*/, 0 /*Ignored for OnlineAction*/)
}

// ListNamespaces -
func (d *nvmeDevice) ListNamespaces(controllerId uint16) ([]nvme.NamespaceIdentifier, error) {
	list, err := d.dev.IdentifyNamespaceList(0, true)
	if err != nil {
		return nil, err
	}

	// Compress the returned list to only those IDs which are valid (non-zero)
	ret := make([]nvme.NamespaceIdentifier, len(list))
	var count = 0
	for _, id := range list {
		if id != 0 {
			ret[count] = id
			count++
		}
	}

	return ret[:count], nil
}

// ListAttachedControllers -
func (d *nvmeDevice) ListAttachedControllers(namespaceId nvme.NamespaceIdentifier) ([]uint16, error) {
	list, err := d.dev.IdentifyNamespaceControllerList(uint32(namespaceId))
	if err != nil {
		return nil, err
	}

	return list.Identifiers[:list.Num], nil
}

// GetNamespace -
func (d *nvmeDevice) GetNamespace(namespaceId nvme.NamespaceIdentifier) (*nvme.IdNs, error) {
	return d.dev.IdentifyNamespace(uint32(namespaceId), true)
}

// CreateNamespace -
func (d *nvmeDevice) CreateNamespace(sizeInSectors uint64, sectorSizeIndex uint8) (nvme.NamespaceIdentifier, nvme.NamespaceGloballyUniqueIdentifier, error) {

	id, err := d.dev.CreateNamespace(
		sizeInSectors,   // Size in Data Size Units (usually 4096)
		sizeInSectors,   // Capacity in Data Size Units (usually 4096),
		sectorSizeIndex, // LBA Format Index (see above)
		0,               // Data Protection Capaiblities (none)
		0x1,             // Capabilities (sharing = 1b)
		0,               // ANA Group Identifier (none)
		0,               // NVM Set Identifier (none)
		100,             // Timeout (???)
	)

	if err != nil {
		return nvme.NamespaceIdentifier(0), nvme.NamespaceGloballyUniqueIdentifier{}, err
	}

	// Turn around and identify the namespace so we can read the GUID that is assigned by the controller
	ns, err := d.dev.IdentifyNamespace(id, true)
	if err != nil {
		return nvme.NamespaceIdentifier(id), nvme.NamespaceGloballyUniqueIdentifier{}, err
	}

	return nvme.NamespaceIdentifier(id), ns.GloballyUniqueIdentifier, err
}

// DeleteNamespace -
func (d *nvmeDevice) DeleteNamespace(namespaceId nvme.NamespaceIdentifier) error {
	return d.dev.DeleteNamespace(uint32(namespaceId))
}

// FormatNamespace -
func (d *nvmeDevice) FormatNamespace(namespaceID nvme.NamespaceIdentifier) error {
	return d.dev.FormatNamespace(uint32(namespaceID))
}

// WaitFormatComplete -
func (d *nvmeDevice) WaitFormatComplete(namespaceID nvme.NamespaceIdentifier) error {
	return d.dev.WaitFormatComplete(uint32(namespaceID))
}

// AttachNamespace -
func (d *nvmeDevice) AttachNamespace(namespaceId nvme.NamespaceIdentifier, controllers []uint16) error {
	return d.dev.AttachNamespace(uint32(namespaceId), controllers)
}

// DetachNamespace -
func (d *nvmeDevice) DetachNamespace(namespaceId nvme.NamespaceIdentifier, controllers []uint16) error {
	return d.dev.DetachNamespace(uint32(namespaceId), controllers)
}

const (
	DefaultFeatureType = 1
)

func (d *nvmeDevice) SetNamespaceFeature(namespaceId nvme.NamespaceIdentifier, data []byte) error {
	builder := nvme.NewMiFeatureBuilder()
	builder.AddElement(DefaultFeatureType, 0, data)

	buf := make([]byte, nvme.FeatureBufferLength[nvme.MiNamespaceMetadata])
	copy(buf, builder.Bytes())

	return d.dev.SetFeature(uint32(namespaceId), nvme.MiNamespaceMetadata, 0, false, uint32(len(buf)), buf)
}

func (d *nvmeDevice) GetNamespaceFeature(namespaceId nvme.NamespaceIdentifier) ([]byte, error) {
	buf := make([]byte, nvme.FeatureBufferLength[nvme.MiNamespaceMetadata])

	if err := d.dev.GetFeature(uint32(namespaceId), nvme.MiNamespaceMetadata, 0, 0, uint32(len(buf)), buf); err != nil {
		return nil, err
	}

	hdr := nvme.MIHostMetadata{}
	ndesc := buf[unsafe.Offsetof(hdr.NumDescriptors)]
	data := buf[unsafe.Offsetof(hdr.DescriptorData):]

	for i := uint8(0); i < ndesc; i++ {
		desc := nvme.MIHostMetadataElementDescriptor{}

		// Ugly as can be! Maybe use encoding.binary/Read
		// which takes in a byte reader and dumps that data to
		// an interface?

		typ := data[unsafe.Offsetof(desc.Type)]
		len := binary.LittleEndian.Uint16(data[unsafe.Offsetof(desc.Len):])
		offset := uint16(unsafe.Offsetof(desc.Val))
		val := data[offset : offset+len]

		if typ == DefaultFeatureType {
			return val, nil
		}
	}

	return buf, nil
}

func (d *nvmeDevice) GetWearLevelAsPercentageUsed() (uint8, error) {

	log, err := d.dev.GetSmartLog()
	if err != nil {
		return 0, err
	}

	return log.PercentageUsed, nil
}
