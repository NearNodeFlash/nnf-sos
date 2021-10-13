package nvme

import (
	"encoding/binary"
	"fmt"
	"math"
	"math/rand"
	"strconv"
	"strings"

	"stash.us.cray.com/rabsw/nnf-ec/internal/switchtec/pkg/nvme"
)

type MockNvmeController struct{}

func NewMockNvmeController() NvmeController {
	return &MockNvmeController{}
}

func (MockNvmeController) NewNvmeDeviceController() NvmeDeviceController {
	return &MockNvmeDeviceController{}
}

type MockNvmeDeviceController struct {
	mockPersistenceManager *MockNvmePersistenceManager
}

func (ctrl *MockNvmeDeviceController) Initialize() error {
	ctrl.mockPersistenceManager = &MockNvmePersistenceManager{}
	return ctrl.mockPersistenceManager.initialize()
}

func (ctrl *MockNvmeDeviceController) Close() error {
	return ctrl.mockPersistenceManager.close()
}

func (ctrl MockNvmeDeviceController) NewNvmeDevice(fabricId, switchId, portId string) (NvmeDeviceApi, error) {
	dev := &mockDevice{
		virtualizationManagement: true,
		capacity:                 mockCapacityInBytes,
		allocatedCapacity:        0,

		// identification data
		fabricId: fabricId,
		switchId: switchId,
		portId:   portId,

		persistenceMgr: ctrl.mockPersistenceManager,
	}

	for idx := range dev.controllers {
		dev.controllers[idx] = mockController{
			id:          uint16(idx),
			online:      false,
			vqresources: 0,
			viresources: 0,
		}
	}

	dev.namespaces[0].id = CommonNamespaceIdentifier
	for idx := range dev.namespaces {
		dev.namespaces[idx].idx = idx
	}

	if ctrl.mockPersistenceManager != nil {
		dev.persistenceMgr = ctrl.mockPersistenceManager
		if err := dev.persistenceMgr.load(dev); err != nil {
			dev.persistenceMgr.new(dev)
		}
	}

	return dev, nil
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

	fabricId string
	switchId string
	portId   string

	persistenceMgr *MockNvmePersistenceManager
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

func (d *mockDevice) id() string { return fmt.Sprintf("%s_%s_%s", d.fabricId, d.switchId, d.portId) }

func (d *mockDevice) unpack(id string) {
	s := strings.Split(id, "_")
	d.fabricId, d.switchId, d.portId = s[0], s[1], s[2]
}

func (d *mockDevice) generateControllerAttributes(ctrl *nvme.IdCtrl) error {
	switchId, err := strconv.Atoi(d.switchId)
	if err != nil {
		return err
	}
	portId, err := strconv.Atoi(d.portId)
	if err != nil {
		return err
	}

	deviceId := switchId*1000 + portId

	r := rand.New(rand.NewSource(int64(deviceId)))

	generateRandomBytes := func(dest []byte) {
		mockLetters := "MOCK"
		attributeLetters := "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

		for i := 0; i < len(dest); i++ {
			if i < len(mockLetters) && len(mockLetters) < len(dest) {
				dest[i] = mockLetters[i]
			} else {
				dest[i] = attributeLetters[r.Int63()%int64(len(attributeLetters))]
			}
		}
	}

	generateRandomBytes(ctrl.SerialNumber[:])
	generateRandomBytes(ctrl.ModelNumber[:])
	generateRandomBytes(ctrl.FirmwareRevision[:])
	generateRandomBytes(ctrl.NVMSubsystemNVMeQualifiedName[:])

	return nil
}

// IdentifyController -
func (d *mockDevice) IdentifyController(controllerId uint16) (*nvme.IdCtrl, error) {
	ctrl := new(nvme.IdCtrl)

	if err := d.generateControllerAttributes(ctrl); err != nil {
		return nil, err
	}

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

	list := make([]nvme.NamespaceIdentifier, 0)
	for _, ns := range d.namespaces {
		if ns.id != nvme.COMMON_NAMESPACE_IDENTIFIER && ns.id != nvme.NamespaceIdentifier(invalidNamespaceId) {
			list = append(list, ns.id)
		}
	}

	return list, nil
}

// ListAttachedControllers
func (d *mockDevice) ListAttachedControllers(namespaceId nvme.NamespaceIdentifier) ([]uint16, error) {

	ns := d.findNamespace(namespaceId)

	controllerIds := make([]uint16, 0)
	for _, ctrl := range ns.attachedControllers {
		if ctrl != nil {
			controllerIds = append(controllerIds, ctrl.id)
		}
	}

	return controllerIds, nil
}

// CreateNamespace -
func (d *mockDevice) CreateNamespace(capacityBytes uint64, sectorSizeBytes uint64, sectorSizeIndex uint8) (nvme.NamespaceIdentifier, nvme.NamespaceGloballyUniqueIdentifier, error) {

	if capacityBytes > (d.capacity - d.allocatedCapacity) {
		return 0, nvme.NamespaceGloballyUniqueIdentifier{}, fmt.Errorf("Insufficient Capacity")
	}

	ns := d.findNamespace(invalidNamespaceId) // find a free namespace
	if ns == nil {
		return 0, nvme.NamespaceGloballyUniqueIdentifier{}, fmt.Errorf("Could not find free namespace")
	}

	ns.id = nvme.NamespaceIdentifier(ns.idx)
	ns.capacity = capacityBytes / 4096 // 4096 is always assumed
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

	if d.persistenceMgr != nil {
		d.persistenceMgr.recordCreateNamespace(d, ns)
	}

	return ns.id, nvme.NamespaceGloballyUniqueIdentifier{}, nil
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

	if d.persistenceMgr != nil {
		d.persistenceMgr.recordDeleteNamespace(d, ns)
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

		if d.persistenceMgr != nil {
			d.persistenceMgr.recordAttachController(d, ns, c)
		}
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

		if d.persistenceMgr != nil {
			d.persistenceMgr.recordDetachController(d, ns, c)
		}
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
