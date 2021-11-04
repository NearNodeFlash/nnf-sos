package nvme

import (
	"encoding/binary"
	"fmt"
	"math"

	"stash.us.cray.com/rabsw/nnf-ec/internal/switchtec/pkg/nvme"
)

type MockNvmeController struct{}

func NewMockNvmeController() NvmeController {
	return &MockNvmeController{}
}

func (MockNvmeController) NewNvmeDeviceController() NvmeDeviceController {
	return &MockNvmeDeviceController{}
}

type MockNvmeDeviceController struct{}

func (MockNvmeDeviceController) NewNvmeDevice(fabricId, switchId, portId string) (NvmeDeviceApi, error) {
	return newMockDevice(fabricId, switchId, portId)
}

const (
	invalidNamespaceId = nvme.NamespaceIdentifier(0)

	mockSecondaryControllerCount = 17
	mockMaximumNamespaceCount    = 32
	mockCapacityInBytes          = 2 << 40 // 2TiB
)

// Mock structurs defining the componenets of a NVMe Device
type mockDevice struct {
	virtualizationManagement bool
	controllers              [1 + mockSecondaryControllerCount]mockController
	namespaces               [mockMaximumNamespaceCount]mockNamespace
	capacity                 uint64
	allocatedCapacity        uint64
}

type mockController struct {
	id                uint16
	online            bool
	capacity          uint64
	allocatedCapacity uint64
	vqresources       uint32
	viresources       uint32
}

type mockNamespace struct {
	id                  nvme.NamespaceIdentifier
	idx                 int
	size                uint64
	capacity            uint64
	guid                [16]byte
	attachedControllers [mockSecondaryControllerCount]*mockController
	metadata            []byte
}

func newMockDevice(fabricId, switchId, portId string) (NvmeDeviceApi, error) {
	mock := mockDevice{
		virtualizationManagement: true,
		capacity:                 mockCapacityInBytes,
		allocatedCapacity:        0,
	}

	for idx := range mock.controllers {
		mock.controllers[idx] = mockController{
			id:          uint16(idx),
			online:      false,
			vqresources: 0,
			viresources: 0,
		}
	}

	mock.namespaces[0].id = CommonNamespaceIdentifier
	for idx := range mock.namespaces {
		mock.namespaces[idx].idx = idx
	}

	return &mock, nil
}

// IdentifyController -
func (d *mockDevice) IdentifyController(controllerId uint16) (*nvme.IdCtrl, error) {
	ctrl := new(nvme.IdCtrl)

	binary.LittleEndian.PutUint64(ctrl.TotalNVMCapacity[:], d.capacity)
	binary.LittleEndian.PutUint64(ctrl.UnallocatedNVMCapacity[:], d.capacity-d.allocatedCapacity)

	ctrl.OptionalAdminCommandSupport = nvme.VirtualiztionManagementSupport

	return ctrl, nil
}

// IdentifyNamespace -
func (d *mockDevice) IdentifyNamespace(namespaceId nvme.NamespaceIdentifier) (*nvme.IdNs, error) {
	ns := d.findNamespace(namespaceId)
	if ns == nil {
		return nil, fmt.Errorf("Namespace %d not found", namespaceId)
	}

	idns := new(nvme.IdNs)

	idns.Size = ns.capacity / 4096
	idns.Capacity = ns.capacity / 4096
	idns.MultiPathIOSharingCapabilities.Sharing = 1
	idns.FormattedLBASize = nvme.FormattedLBASize{Format: 0}

	idns.NumberOfLBAFormats = 1
	idns.LBAFormats[0].LBADataSize = uint8(math.Log2(4096))
	idns.LBAFormats[0].MetadataSize = 0
	idns.LBAFormats[0].RelativePerformance = 0

	return idns, nil
}

// ListSecondary -
func (d *mockDevice) ListSecondary() (*nvme.SecondaryControllerList, error) {
	ls := new(nvme.SecondaryControllerList)

	ls.Count = uint8(len(d.controllers)) - 1
	for idx, ctrl := range d.controllers[1:] {
		state := uint8(0)
		if ctrl.online {
			state = 1
		}

		ls.Entries[idx] = nvme.SecondaryControllerEntry{
			SecondaryControllerID:       ctrl.id,
			SecondaryControllerState:    state,
			VQFlexibleResourcesAssigned: uint16(ctrl.vqresources),
			VIFlexibleResourcesAssigned: uint16(ctrl.viresources),
			VirtualFunctionNumber:       uint16(idx + 1),
		}
	}
	return ls, nil
}

// AssignControllerResources -
func (d *mockDevice) AssignControllerResources(controllerId uint16, resourceType SecondaryControllerResourceType, numResources uint32) error {
	ctrl := &d.controllers[int(controllerId)]
	switch resourceType {
	case VQResourceType:
		ctrl.vqresources = numResources
	case VIResourceType:
		ctrl.viresources = numResources
	}
	return nil
}

// OnlineController -
func (d *mockDevice) OnlineController(controllerId uint16) error {
	ctrl := &d.controllers[int(controllerId)]
	ctrl.online = true

	return nil
}

// ListNamespaces -
func (d *mockDevice) ListNamespaces(controllerId uint16) ([]nvme.NamespaceIdentifier, error) {
	nss := d.namespaces
	var count = 0
	for _, ns := range nss {
		if ns.id != 0 {
			count++
		}
	}

	list := make([]nvme.NamespaceIdentifier, count)
	for idx, ns := range nss {
		if ns.id != invalidNamespaceId {
			list[idx] = ns.id
		}
	}

	return list, nil
}

// CreateNamespace -
func (d *mockDevice) CreateNamespace(capacityBytes uint64, metadata []byte) (nvme.NamespaceIdentifier, error) {

	if capacityBytes > (d.capacity - d.allocatedCapacity) {
		return 0, fmt.Errorf("Insufficient Capacity")
	}

	ns := d.findNamespace(invalidNamespaceId)
	if ns == nil {
		return 0, fmt.Errorf("Could not find free namespace")
	}

	ns.id = nvme.NamespaceIdentifier(ns.idx)
	ns.capacity = capacityBytes / 4096
	ns.guid = [16]byte{
		0, 0, 0, 0,
		0, 0, 0, 0,
		0, 0, 0, 0,
		byte(ns.idx >> 12),
		byte(ns.idx >> 8),
		byte(ns.idx >> 4),
		byte(ns.idx >> 0),
	}

	for idx := range ns.attachedControllers {
		ns.attachedControllers[idx] = nil
	}

	d.allocatedCapacity += capacityBytes

	return ns.id, nil
}

// DeleteNamespace -
func (d *mockDevice) DeleteNamespace(namespaceId nvme.NamespaceIdentifier) error {
	ns := d.findNamespace(namespaceId)
	if ns == nil {
		return fmt.Errorf("Delete Namespace: Namespace %d not found", namespaceId)
	}

	ctrls := make([]uint16, 0)
	for ctrlIdx, ctrl := range ns.attachedControllers {
		if ctrl != nil {
			ctrls = append(ctrls, ctrl.id)
		}
		ns.attachedControllers[ctrlIdx] = nil
	}

	if len(ctrls) != 0 {
		if err := d.DetachNamespace(namespaceId, ctrls); err != nil {
			return err
		}
	}

	ns.id = invalidNamespaceId
	return nil
}

// AttachNamespace -
func (d *mockDevice) AttachNamespace(namespaceId nvme.NamespaceIdentifier, controllers []uint16) error {
	ns := d.findNamespace(namespaceId)
	if ns == nil {
		return fmt.Errorf("Attach Namespace: Namespace %d not found", namespaceId)
	}

	for _, c := range controllers {

		if !(c < uint16(len(d.controllers))) {
			return fmt.Errorf("Attach Namespace: Controller %d exceeds controller count", c)
		} else if ns.attachedControllers[c] != nil {
			return fmt.Errorf("Attach Namespace: Namespace %d already has controller %d attached", namespaceId, c)
		}

		ns.attachedControllers[c] = &d.controllers[c]
	}

	return nil
}

// DetachNamespace -
func (d *mockDevice) DetachNamespace(namespaceId nvme.NamespaceIdentifier, controllers []uint16) error {
	ns := d.findNamespace(namespaceId)
	if ns == nil {
		return fmt.Errorf("Detach Namespace: Namespace %d not found", namespaceId)
	}

	for _, c := range controllers {

		if !(c < uint16(len(d.controllers))) {
			return fmt.Errorf("Detach Namespace: Controller %d exceeds controller count", c)
		} else if ns.attachedControllers[c] == nil {
			return fmt.Errorf("Detach Namespace: Namespace %d already has controller %d detached", namespaceId, c)
		}

		ns.attachedControllers[c] = nil
	}

	return nil
}

func (d *mockDevice) SetNamespaceFeature(namespaceId nvme.NamespaceIdentifier, data []byte) error {
	ns := d.findNamespace(namespaceId)
	if ns == nil {
		return fmt.Errorf("Set Namespace Feature: Namespace %d not found", namespaceId)
	}
	ns.metadata = data
	return nil
}

func (d *mockDevice) GetNamespaceFeature(namespaceId nvme.NamespaceIdentifier) ([]byte, error) {
	ns := d.findNamespace(namespaceId)
	if ns == nil {
		return nil, fmt.Errorf("Get Namespace Feature: Namespace %d not found", namespaceId)
	}

	return ns.metadata, nil
}

func (d *mockDevice) findNamespace(namespaceId nvme.NamespaceIdentifier) *mockNamespace {

	for idx, ns := range d.namespaces {
		if ns.id == namespaceId {
			return &d.namespaces[idx]
		}
	}

	return nil
}
