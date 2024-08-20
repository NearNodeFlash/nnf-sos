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
	"errors"
	"flag"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	. "github.com/NearNodeFlash/nnf-ec/pkg/api"
	event "github.com/NearNodeFlash/nnf-ec/pkg/manager-event"
	fabric "github.com/NearNodeFlash/nnf-ec/pkg/manager-fabric"
	msgreg "github.com/NearNodeFlash/nnf-ec/pkg/manager-message-registry/registries"

	ec "github.com/NearNodeFlash/nnf-ec/pkg/ec"

	"github.com/NearNodeFlash/nnf-ec/internal/switchtec/pkg/nvme"
	sf "github.com/NearNodeFlash/nnf-ec/pkg/rfsf/pkg/models"
)

const (
	ResourceBlockId = "Rabbit"

	// TODO: The ALL_CAPS name in nvme package should be renamed to a valid Go name
	CommonNamespaceIdentifier = nvme.COMMON_NAMESPACE_IDENTIFIER

	// Physical Function controller index
	PhysicalFunctionControllerIndex = 0
)

const (
	DefaultStoragePoolId = "0"
)

const (
	switchIdKey = "switchId"
	portIdKey   = "portId"
	slotKey     = "slot"

	storageIdKey = "storageId"
	volumeIdKey  = "volumeId"

	namespaceIdKey  = "namespaceId"
	controllerIdKey = "controllerId"
)

var mgr = Manager{id: ResourceBlockId}

func NewDefaultStorageService() StorageApi {
	return &mgr
}

// Manager -
type Manager struct {
	id string

	config *ConfigFile

	storage []Storage
	ctrl    NvmeDeviceController

	// Command-Line Options
	purge       bool // Purge existing namespaces on storage controllers
	purgeMockDb bool // Purge the persistent mock databse

	log ec.Logger
}

// Storage - Storage defines a generic storage device in the Redfish / Swordfish specification.
// In the NNF implementation
type Storage struct {
	id      string
	address string

	slot int64

	serialNumber     string
	modelNumber      string
	firmwareRevision string
	qualifiedName    string

	// Physical Function Controller ID
	physicalFunctionControllerId uint16

	// True if the host controller supports NVMe 1.3 Virtualization Management, false otherwise
	virtManagementEnabled bool

	// Capacity in bytes of the storage device. This value is read once and is fixed for
	// the life of the object.
	capacityBytes uint64

	// Unallocated capacity in bytes. This value is updated for any namespaces create or
	// delete operation that might shrink or grow the byte count as expected.
	unallocatedBytes uint64

	// Namespace Properties - Read using the Common Namespace Identifier (0xffffffff)
	// These are properties common to all namespaces for this controller (we use controller
	// zero as the basis for all other controllers - technically the spec supports uinque
	// LBA Formats per controller, but this is not done in practice by drive vendors.)
	lbaFormatIndex uint8
	blockSizeBytes uint64

	state sf.ResourceState

	// These values allow us to communicate a storage device with its corresponding
	// Fabric Controller. Read once during Port Up Events and remain fixed thereafter.
	fabricId string
	switchId string
	portId   string

	manager *Manager

	controllers []StorageController // List of Storage Controllers on the Storage device
	volumes     []Volume            // List of Volumes on the Storage device
	config      *ControllerConfig   // Link to the storage configuration

	device NvmeDeviceApi // Device interface for interaction with the underlying NVMe device

	log ec.Logger
}

// StorageController -
type StorageController struct {
	id string

	controllerId   uint16
	functionNumber uint16

	// These are attributes for a Secondary Controller that is manged
	// by the Primary Controller using NVMe 1.3 Virtualization Mgmt.
	vqResources uint16
	viResources uint16
	online      bool
}

// Volumes -
type Volume struct {
	id            string
	namespaceId   nvme.NamespaceIdentifier
	guid          nvme.NamespaceGloballyUniqueIdentifier
	capacityBytes uint64

	storage *Storage
	log     ec.Logger
}

type ProvidingVolume struct {
	Storage  *Storage
	VolumeId string
}

// TODO: We may want to put this manager under a resource block
//   /​redfish/​v1/​ResourceBlocks/​{ResourceBlockId} // <- Rabbit
//   /​redfish/​v1/​ResourceBlocks/​{ResourceBlockId}/​Systems/{​ComputerSystemId} // <- Also Rabbit & Computes
//   /​redfish/​v1/​ResourceBlocks/​{ResourceBlockId}/​Systems/{​ComputerSystemId}/​PCIeDevices/​{PCIeDeviceId}
//   /​redfish/​v1/​ResourceBlocks/​{ResourceBlockId}/​Systems/{​ComputerSystemId}/​PCIeDevices/​{PCIeDeviceId}/​PCIeFunctions/{​PCIeFunctionId}
//
//   /​redfish/​v1/​ResourceBlocks/{​ResourceBlockId}/​Systems/{​ComputerSystemId}/​Storage/​{StorageId}/​Controllers/​{ControllerId}

func init() {
	RegisterNvmeInterface(&mgr)
}

func findStorage(storageId string) *Storage {
	id, err := strconv.Atoi(storageId)
	if err != nil {
		return nil
	}

	if !(id < len(mgr.storage)) {
		return nil
	}

	return &mgr.storage[id]
}

func findStorageController(storageId, controllerId string) (*Storage, *StorageController) {
	s := findStorage(storageId)
	if s == nil {
		return nil, nil
	}

	return s, s.findController(controllerId)
}

func findStorageVolume(storageId, volumeId string) (*Storage, *Volume) {
	s := findStorage(storageId)
	if s == nil {
		return nil, nil
	}

	return s, s.findVolume(volumeId)
}

func findStoragePool(storageId, storagePoolId string) (*Storage, *interface{}) {
	return nil, nil
}

func CleanupVolumes(providingVolumes []ProvidingVolume) {
	for _, storage := range mgr.storage {
		if !storage.IsEnabled() {
			continue
		}

		var volIdsToKeep []string
		for _, vol := range providingVolumes {
			if vol.Storage.serialNumber == storage.serialNumber {
				volIdsToKeep = append(volIdsToKeep, vol.VolumeId)
			}
		}

		if err := storage.purgeVolumes(volIdsToKeep); err != nil {
			mgr.log.Error(err, "Failed to remove abandoned volumes", "storage", storage)
		}
	}
}

func (m *Manager) fmt(format string, a ...interface{}) string {
	return fmt.Sprintf("/redfish/v1") + fmt.Sprintf(format, a...)
}

// GetVolumes -
func (m *Manager) GetVolumes(controllerId string) ([]string, error) {
	volumes := []string{}
	for _, s := range m.storage {
		c := s.findController(controllerId)
		if c == nil {
			return volumes, ec.NewErrNotFound()
		}

		nsids, err := s.device.ListNamespaces(c.functionNumber)
		if err != nil {
			return volumes, err
		}

		for _, nsid := range nsids {
			for _, v := range s.volumes {
				if v.namespaceId == nsid {
					volumes = append(volumes, fmt.Sprintf("/redfish/v1/Storage/%s/Volumes/%s", s.id, v.id))
				}
			}
		}

	}

	return volumes, nil
}

func BindFlags(fs *flag.FlagSet) {
	fs.BoolVar(&mgr.purge, "purge", false, "Purge existing volumes on start")
	fs.BoolVar(&mgr.purgeMockDb, "purge-mock-db", false, "Purge the persistent mock-device database")
}

func ConvertRelativePortIndexToControllerIndex(index uint32) (uint16, error) {
	if !(index < mgr.config.Storage.Controller.Functions) {
		return 0, fmt.Errorf("Port Index %d is beyond supported controller count (%d)",
			index, mgr.config.Storage.Controller.Functions)
	}

	return uint16(index + 1), nil
}

func GetStorage() []*Storage {
	storage := make([]*Storage, len(mgr.storage))
	for idx := range storage {
		storage[idx] = &mgr.storage[idx]
	}

	return storage
}

func EnumerateStorage(storageHandlerFunc func(odataId string, capacityBytes uint64, unallocatedBytes uint64)) error {
	for _, s := range mgr.storage {
		storageHandlerFunc(s.OdataId()+"/StoragePools", s.capacityBytes, s.unallocatedBytes)
	}

	return nil
}

func CreateVolume(s *Storage, capacityBytes uint64) (*Volume, error) {
	return s.createVolume(capacityBytes)
}

func (s *Storage) UnallocatedBytes() uint64 { return s.unallocatedBytes }
func (s *Storage) IsEnabled() bool          { return s.state == sf.ENABLED_RST }
func (s *Storage) SerialNumber() string     { return s.serialNumber }

func (s *Storage) IsKioxiaDualPortConfiguration() bool {
	return false ||
		strings.Contains(s.qualifiedName, "com.kioxia:KCM6") ||
		strings.Contains(s.qualifiedName, "com.kioxia:KCD7") ||
		strings.Contains(s.qualifiedName, "com.kioxia:KCM7")
}

func (s *Storage) FindVolume(id string) *Volume {
	return s.findVolume(id)
}

func (s *Storage) FindVolumeByNamespaceId(namespaceId nvme.NamespaceIdentifier) (*Volume, error) {
	for idx, volume := range s.volumes {
		if volume.namespaceId == namespaceId {
			return &s.volumes[idx], nil
		}
	}

	return nil, ec.NewErrNotFound()
}

func (s *Storage) OdataId() string {
	return fmt.Sprintf("/redfish/v1/Storage/%s", s.id)
}

func (s *Storage) OdataIdRef(ref string) sf.OdataV4IdRef {
	return sf.OdataV4IdRef{OdataId: fmt.Sprintf("%s%s", s.OdataId(), ref)}
}

func (s *Storage) initialize() error {

	log := s.manager.log.WithName(s.id).WithValues(storageIdKey, s.id, slotKey, s.slot)
	log.Info("Initialize storage device")

	s.log = log
	s.state = sf.STARTING_RST

	ctrl, err := s.device.IdentifyController(0)
	if err != nil {
		return fmt.Errorf("Initialize Storage %s: Failed to indentify common controller: Error: %w", s.id, err)
	}

	s.physicalFunctionControllerId = ctrl.ControllerId

	trimTrailingWhitespace := func(val []byte) string {
		return strings.TrimRight(string(val), " ")
	}

	s.serialNumber = trimTrailingWhitespace(ctrl.SerialNumber[:])
	s.modelNumber = trimTrailingWhitespace(ctrl.ModelNumber[:])
	s.firmwareRevision = trimTrailingWhitespace(ctrl.FirmwareRevision[:])
	s.qualifiedName = trimTrailingWhitespace(ctrl.NVMSubsystemNVMeQualifiedName[:])

	capacityToUint64s := func(c [16]byte) (lo uint64, hi uint64) {
		lo, hi = 0, 0
		for i := 0; i < 8; i++ {
			lo, hi = lo<<8, hi<<8
			lo += uint64(c[7-i])
			hi += uint64(c[15-i])
		}

		return lo, hi
	}

	totalCapBytesLo, totalCapBytesHi := capacityToUint64s(ctrl.TotalNVMCapacity)

	s.capacityBytes = totalCapBytesLo
	if totalCapBytesHi != 0 {
		return fmt.Errorf("Initialize Storage %s: Unsupported capacity 0x%x_%x: will overflow uint64 definition", s.id, totalCapBytesHi, totalCapBytesLo)
	}

	unallocatedCapBytesLo, unallocatedCapBytesHi := capacityToUint64s(ctrl.UnallocatedNVMCapacity)

	s.unallocatedBytes = unallocatedCapBytesLo
	if unallocatedCapBytesHi != 0 {
		return fmt.Errorf("Initialize Storage %s: Unsupported unallocated 0x%x_%x, will overflow uint64 definition", s.id, unallocatedCapBytesHi, unallocatedCapBytesLo)
	}

	s.virtManagementEnabled = ctrl.GetCapability(nvme.VirtualiztionManagementSupport)

	log.V(1).Info("Identified controller",
		"serialNumber", s.serialNumber,
		"modelNumber", s.modelNumber,
		"firmwareRevision", s.firmwareRevision,
		"capacityInBytes", s.capacityBytes,
		"unallocatedBytes", s.unallocatedBytes,
		"virtualizationManagement", s.virtManagementEnabled)

	var ns *nvme.IdNs
	identifySuccess := false
	for retryCount := 0; !identifySuccess; retryCount++ {
		var err error
		ns, err = s.device.IdentifyNamespace(CommonNamespaceIdentifier)
		if err != nil {
			if retryCount >= 2 {
				return fmt.Errorf("Initialize Storage %s: Failed to identify common namespace, retried %d times: Error: %w", s.id, retryCount, err)
			}

			s.log.Info("identify common namespace attempt failed, retrying", "retryCount", retryCount)
		} else {
			identifySuccess = true
		}
	}

	// Workaround for SSST drives that improperly report only one NumberOfLBAFormats, but actually
	// support two - with the second being the most performant 4096 sector size.
	if ns.NumberOfLBAFormats == 1 {

		if ((1 << ns.LBAFormats[1].LBADataSize) == 4096) && (ns.LBAFormats[1].RelativePerformance == 0) {
			log.Info("Incorrect number of LBA formats", "expected", 2, "actual", ns.NumberOfLBAFormats)
			ns.NumberOfLBAFormats = 2
		}
	}

	bestIndex := 0
	bestRelativePerformance := ^uint8(0)
	for i := 0; i < int(ns.NumberOfLBAFormats); i++ {
		if ns.LBAFormats[i].MetadataSize == 0 &&
			ns.LBAFormats[i].RelativePerformance < bestRelativePerformance {
			bestIndex = i
		}
	}
	s.lbaFormatIndex = uint8(bestIndex)
	s.blockSizeBytes = 1 << ns.LBAFormats[bestIndex].LBADataSize

	s.log = s.log.WithValues("serialNumber", s.serialNumber)
	s.log.Info("Initialized storage device")

	return nil
}

func (s *Storage) purge() error {
	if s.device == nil {
		return fmt.Errorf("Storage %s has no device", s.id)
	}

	namespaces, err := s.device.ListNamespaces(0)
	if err != nil {
		return err
	}

	for _, nsid := range namespaces {
		if err := s.device.DeleteNamespace(nsid); err != nil {
			return err
		}
	}

	return nil
}

// Delete all the volumes on this storage device except for the volumes that we want to keep
func (s *Storage) purgeVolumes(volIdsToKeep []string) error {
	if s.device == nil {
		return fmt.Errorf("Storage %s has no device", s.id)
	}

	var volIdsToDelete []string

vol_loop:
	for _, vol := range s.volumes {
		for _, volIdToKeep := range volIdsToKeep {
			if volIdToKeep == vol.id {
				continue vol_loop
			}
		}
		volIdsToDelete = append(volIdsToDelete, vol.id)
	}

	for _, volId := range volIdsToDelete {
		if err := s.deleteVolume(volId); err != nil {
			return err
		}
	}

	return nil
}

func (s *Storage) findController(controllerId string) *StorageController {
	for idx, ctrl := range s.controllers {
		if ctrl.id == controllerId {
			return &s.controllers[idx]
		}
	}

	return nil
}

func (s *Storage) getStatus() (stat sf.ResourceStatus) {
	if len(s.controllers) == 0 {
		stat.State = sf.UNAVAILABLE_OFFLINE_RST
		stat.Health = sf.CRITICAL_RH
	} else {
		stat.Health = sf.OK_RH
		stat.State = s.state
	}

	return stat
}

func (s *Storage) createVolume(desiredCapacityInBytes uint64) (*Volume, error) {

	roundUpToMultiple := func(n, m uint64) uint64 { // Round 'n' up to a multiple of 'm'
		return ((n + m - 1) / m) * m
	}

	actualCapacityBytes := roundUpToMultiple(desiredCapacityInBytes, s.blockSizeBytes)

	s.log.V(2).Info("Creating namespace", "capacityInBytes", actualCapacityBytes, "formatIndex", s.lbaFormatIndex)
	namespaceId, guid, err := s.device.CreateNamespace(actualCapacityBytes/s.blockSizeBytes, s.lbaFormatIndex)
	if err != nil {
		return nil, err
	}

	id := strconv.Itoa(int(namespaceId))

	log := s.log.WithName(id).WithValues(namespaceIdKey, namespaceId)
	log.V(1).Info("Created namespace")

	volume := Volume{
		id:            id,
		namespaceId:   namespaceId,
		guid:          guid,
		capacityBytes: actualCapacityBytes,
		storage:       s,
		log:           log,
	}

	s.volumes = append(s.volumes, volume)

	s.unallocatedBytes -= actualCapacityBytes

	return &s.volumes[len(s.volumes)-1], nil
}

func (s *Storage) deleteVolume(volumeId string) error {

	for idx, volume := range s.volumes {
		if volume.id == volumeId {
			log := volume.log

			log.V(2).Info("Deleting namespace")
			if err := s.device.DeleteNamespace(volume.namespaceId); err != nil {
				log.Error(err, "Delete namespace failed")
				return err
			}
			log.V(1).Info("Deleted namespace")

			s.unallocatedBytes += volume.capacityBytes

			// remove the volume from the array
			copy(s.volumes[idx:], s.volumes[idx+1:]) // shift left 1 at idx
			s.volumes = s.volumes[:len(s.volumes)-1] // truncate tail

			return nil
		}
	}

	return ec.NewErrNotFound()
}

func (s *Storage) formatVolume(volumeID string) error {
	for _, volume := range s.volumes {
		if volume.id == volumeID {
			log := volume.log

			log.V(2).Info("Format namespace")
			if err := s.device.FormatNamespace(volume.namespaceId); err != nil {
				log.Error(err, "Format namespace failure")
				return err
			}
			log.V(1).Info("Formatted namespace")

			return nil
		}
	}

	return ec.NewErrNotFound()
}

func (s *Storage) recoverVolumes() error {
	s.log.V(1).Info("Recovering volumes")

	namespaces, err := s.device.ListNamespaces(0)
	if err != nil {
		s.log.Error(err, "Failed to list device namespaces")
	}

	s.volumes = make([]Volume, 0)
	for _, nsid := range namespaces {
		log := s.log.WithValues(namespaceIdKey, nsid)

		log.V(2).Info("Identify namespace")
		ns, err := s.device.IdentifyNamespace(nsid)
		if err != nil {
			log.Error(err, "Failed to identify namespace")
		}

		blockSizeBytes := uint64(1 << ns.LBAFormats[ns.FormattedLBASize.Format].LBADataSize)

		id := strconv.Itoa(int(nsid))
		volume := Volume{
			id:            id,
			namespaceId:   nsid,
			guid:          ns.GloballyUniqueIdentifier,
			capacityBytes: ns.Capacity * blockSizeBytes,
			storage:       s,
			log:           log.WithName(id),
		}

		s.volumes = append(s.volumes, volume)

		s.log.V(1).Info("Recovered Volume", "volume", volume)
	}

	return nil
}

func (s *Storage) findVolume(volumeId string) *Volume {
	for idx, v := range s.volumes {
		if v.id == volumeId {
			return &s.volumes[idx]
		}
	}

	return nil
}

func (v *Volume) Id() string                               { return v.id }
func (v *Volume) GetOdataId() string                       { return v.storage.OdataId() + "/Volumes/" + v.id }
func (v *Volume) GetCapaityBytes() uint64                  { return uint64(v.capacityBytes) }
func (v *Volume) GetNamespaceId() nvme.NamespaceIdentifier { return v.namespaceId }

func (v *Volume) GetGloballyUniqueIdentifier() nvme.NamespaceGloballyUniqueIdentifier {
	return v.guid
}

func (v *Volume) Delete() error { return v.storage.deleteVolume(v.id) }

func (v *Volume) AttachController(controllerId uint16) error { return v.attach(controllerId) }
func (v *Volume) DetachController(controllerId uint16) error { return v.detach(controllerId) }

func (v *Volume) Format() error {

	// According to the NVM Express Base Specification 2.0b section 5.14 Format NVM Command
	//   The scope of the format operation and the scope of the format with secure erase depend
	//   on the attributes that the controller supports for the Format NVM command and the
	//   Namespace Identifier specified in the command.
	//
	// So the namespace must be attached to a controller. We attach to the controller associated with
	// the physical function to avoid any noise generated by attaching the the volume to a working host.

	return v.runInAttachDetachBlock(func() error { return v.storage.formatVolume(v.id) })
}

func (v *Volume) SetFeature(data []byte) error {

	// Set feature requires the volume is attached to a controller to receive the feature data. We
	// attach to the controller associated with the the physical function to avoid any noise generated by
	// attaching the volume to a working host.

	return v.runInAttachDetachBlock(func() error { return v.storage.device.SetNamespaceFeature(v.namespaceId, data) })
}

func (v *Volume) runInAttachDetachBlock(fn func() error) error {
	const controllerIndex uint16 = PhysicalFunctionControllerIndex
	if err := v.attach(controllerIndex); err != nil {
		return err
	}

	if err := fn(); err != nil {
		return err
	}

	return v.detach(controllerIndex)
}

// WaitFormatComplete waits for Format Completion by polling until the namespace Utilization reaches zero.
func (v *Volume) WaitFormatComplete() error {
	log := v.log

	log.V(2).Info("Wait for format completion")
	ns, err := v.storage.device.IdentifyNamespace(v.namespaceId)
	if err != nil {
		return err
	}

	for ns.Utilization != 0 {
		log.V(3).Info("Namespace in use", "utilization", ns.Utilization)

		const delay = 100 * time.Millisecond

		// Pause briefly for format to make progress
		time.Sleep(delay)

		lastUtilization := ns.Utilization

		ns, err = v.storage.device.IdentifyNamespace(v.namespaceId)
		if err != nil {
			return err
		}

		if lastUtilization == ns.Utilization {
			return fmt.Errorf("Device %s Format Stalled: Namespace: %d Delay: %s Utilization: %d", v.storage.id, v.namespaceId, delay.String(), ns.Utilization)
		}
	}

	log.V(1).Info("Format completed", "utilization", ns.Utilization)

	return nil
}

func (v *Volume) controllerIDFromIndex(controllerIndex uint16) uint16 {
	// Controller indicies to be passed into the nvme-manager;
	// For Kioxia Dual Port drives (not production), the secondary controller IDs
	// start at 3, with controller IDs one and two representing the dual port physical functions.
	//
	// For Direct Devices, the Rabbit is controlling the drive through the physical functions; we
	// still use the secondary controller values for all other ports, but we need to remap the
	// first index to the physical function.

	if controllerIndex == PhysicalFunctionControllerIndex {
		return v.storage.physicalFunctionControllerId
	}

	var controllerID uint16
	if v.storage.device.IsDirectDevice() {
		if controllerIndex == 1 {
			controllerID = v.storage.physicalFunctionControllerId
		}
	} else if v.storage.virtManagementEnabled {
		controllerID = v.storage.controllers[controllerIndex].controllerId
	} else if v.storage.IsKioxiaDualPortConfiguration() {
		controllerID = controllerIndex + 2
	}

	return controllerID
}

func (v *Volume) attach(controllerIndex uint16) error {
	controllerID := v.controllerIDFromIndex(controllerIndex)

	log := v.log.WithValues(controllerIdKey, controllerID)
	log.V(2).Info("Attach namespace", "controllerIndex", controllerIndex)

	err := v.storage.device.AttachNamespace(v.namespaceId, []uint16{controllerID})
	if err != nil {
		log.Error(err, "Attach namespace failed")

		var cmdErr *nvme.CommandError
		if errors.As(err, &cmdErr) {
			if cmdErr.StatusCode != nvme.NamespaceAlreadyAttached {
				return err
			}
		} else {
			return err
		}
	}

	log.V(1).Info("Attached namespace")

	return nil
}

func (v *Volume) detach(controllerIndex uint16) error {
	controllerID := v.controllerIDFromIndex(controllerIndex)

	log := v.log.WithValues(controllerIdKey, controllerID)
	log.V(2).Info("Detach namespace", "controllerIndex", controllerIndex)

	err := v.storage.device.DetachNamespace(v.namespaceId, []uint16{controllerID})

	if err != nil {
		log.Error(err, "Detach namespace failed")

		var cmdErr *nvme.CommandError
		if errors.As(err, &cmdErr) {
			if cmdErr.StatusCode != nvme.NamespaceNotAttached {
				return err
			}
		} else {
			return err
		}
	}

	log.V(1).Info("Detached namespace")

	return nil
}

// Initialize the controller
func Initialize(log ec.Logger, ctrl NvmeController) error {
	log.Info("Initialize NVMe Manager")

	const name = "nvme"
	log.V(2).Info("Creating logger", "name", name)
	log = log.WithName(name)
	mgr.log = log

	conf, err := loadConfig()
	if err != nil {
		log.Error(err, "failed to load configuration", "id", mgr.id)
		return err
	}

	mgr.config = conf

	log.V(1).Info("Loaded configuration",
		"virtualFunctions", conf.Storage.Controller.Functions,
		"resources", conf.Storage.Controller.Resources)

	mgr.log = log

	mgr.storage = make([]Storage, len(conf.Storage.Devices))
	for storageIdx, storageDevice := range conf.Storage.Devices {
		mgr.storage[storageIdx] = Storage{
			id:      strconv.Itoa(storageIdx),
			config:  &conf.Storage.Controller,
			address: storageDevice,
			state:   sf.ABSENT_RST,
			manager: &mgr,
		}
	}

	mgr.ctrl = ctrl.NewNvmeDeviceController()
	if err := mgr.ctrl.Initialize(); err != nil {
		log.Error(err, "Failed to initialize NVMe Device Controller")
		return err
	}

	event.EventManager.Subscribe(&mgr)

	return nil
}

func Close() error {
	return mgr.ctrl.Close()
}

func (m *Manager) EventHandler(e event.Event) error {
	log := m.log.WithValues("eventId", e.Id, "eventMessage", e.Message, "eventArgs", e.MessageArgs)

	linkEstablished := e.Is(msgreg.DownstreamLinkEstablishedFabric("", "")) || e.Is(msgreg.DegradedDownstreamLinkEstablishedFabric("", ""))
	linkDropped := e.Is(msgreg.DownstreamLinkDroppedFabric("", ""))

	if linkEstablished || linkDropped {
		log.V(2).Info("Link event received")

		var switchId, portId string
		if err := e.Args(&switchId, &portId); err != nil {
			return ec.NewErrInternalServerError().WithError(err).WithCause("internal event format illformed")
		}

		idx, err := fabric.FabricController.GetDownstreamPortRelativePortIndex(switchId, portId)
		if err != nil {
			return ec.NewErrInternalServerError().WithError(err).WithCause("downstream relative port index not found")
		}

		if !(idx < len(m.storage)) {
			return fmt.Errorf("No storage device exists for index %d", idx)
		}

		storage := &m.storage[idx]
		storage.fabricId = fabric.FabricId
		storage.switchId = switchId
		storage.portId = portId

		port, err := fabric.FabricController.GetSwitchPort(switchId, portId)
		if err != nil {
			return ec.NewErrInternalServerError().WithError(err).WithCause("downstream switch port not found")
		}

		storage.slot = port.GetSlot()
		storage.log = m.log.WithValues(slotKey, storage.slot)

		if linkEstablished {
			return storage.LinkEstablishedEventHandler(switchId, portId)
		}

		if linkDropped {
			return storage.LinkDroppedEventHandler()
		}
	}

	return nil
}

func (s *Storage) LinkEstablishedEventHandler(switchId, portId string) error {
	log := s.log.WithValues(switchIdKey, switchId, portIdKey, portId)

	// Connect
	device, err := s.manager.ctrl.NewNvmeDevice(fabric.FabricId, switchId, portId)
	if err != nil {
		log.Error(err, "Could not allocate storage controller")
		return err
	}

	s.device = device

	if err := s.initialize(); err != nil {
		log.Error(err, "Failed to initialize storage device")
		return err
	}

	log = s.log // switch to using the storage logger

	if s.manager.purge {
		log.Info("Purging existing volumes")
		if err := s.purge(); err != nil {
			log.Error(err, "Failed to purge storage volumes")
		}
	}

	if s.device.IsDirectDevice() {
		s.controllers = []StorageController{
			// Physical Function
			{
				id:             "0",
				controllerId:   s.physicalFunctionControllerId,
				functionNumber: s.physicalFunctionControllerId,
			},
			// The fabric manager expects the Rabbit to be at index 1 in the virtualized topology
			{
				id:             "1",
				controllerId:   s.physicalFunctionControllerId,
				functionNumber: s.physicalFunctionControllerId,
			},
		}

	} else if !s.virtManagementEnabled {
		s.controllers = make([]StorageController, 1 /* PF */ +s.config.Functions)
		for idx := range s.controllers {
			s.controllers[idx] = StorageController{
				id:             strconv.Itoa(idx),
				controllerId:   uint16(idx),
				functionNumber: uint16(idx),
			}
		}

	} else {
		log.V(2).Info("List Secondary Controllers")

		ls, err := device.ListSecondary()
		if err != nil {
			log.Error(err, "List Secondary failed")
			return err
		}

		count := ls.Count
		if count > uint8(s.config.Functions) {
			count = uint8(s.config.Functions)
		}

		s.controllers = make([]StorageController, 1 /*PF*/ +count)

		// Initialize the PF
		s.controllers[0] = StorageController{
			id:             "0",
			controllerId:   s.physicalFunctionControllerId,
			functionNumber: s.physicalFunctionControllerId,
		}

		for idx, sc := range ls.Entries[:count] {

			if sc.SecondaryControllerID == 0 {
				log.Info("Secondary Controller ID overlaps with PF Controller ID")
				break
			}

			s.controllers[idx+ /*PF*/ 1] = StorageController{
				id:             strconv.Itoa(int(sc.SecondaryControllerID)),
				controllerId:   sc.SecondaryControllerID,
				functionNumber: sc.VirtualFunctionNumber,

				vqResources: sc.VQFlexibleResourcesAssigned,
				viResources: sc.VIFlexibleResourcesAssigned,
				online:      sc.SecondaryControllerState&0x01 == 1,
			}

			ctrl := &s.controllers[idx+1]
			log := log.WithValues(controllerIdKey, ctrl.id)

			if !s.IsKioxiaDualPortConfiguration() {
				log.V(2).Info("Initialize VQ resources", "resources", s.config.Resources)
				if sc.VQFlexibleResourcesAssigned != uint16(s.config.Resources) {
					if err := s.device.AssignControllerResources(sc.SecondaryControllerID, VQResourceType, s.config.Resources); err != nil {
						log.Error(err, "Failed to assign VQ Resources")
						break
					}

					ctrl.vqResources = uint16(s.config.Resources)
				}

				log.V(2).Info("Initialize VI resources", "resources", s.config.Resources)
				if sc.VIFlexibleResourcesAssigned != uint16(s.config.Resources) {
					if err := s.device.AssignControllerResources(sc.SecondaryControllerID, VIResourceType, s.config.Resources); err != nil {
						log.Error(err, "Failed to assign VI resources")
						break
					}

					ctrl.viResources = uint16(s.config.Resources)
				}

				if sc.SecondaryControllerState&0x01 == 0 {
					log.V(2).Info("Reset controller")
					if err := fabric.FabricController.ResetEndpoint(switchId, portId, idx); err != nil {
						log.Error(err, "Failed to reset controller")
					}
				}
			}

			if sc.SecondaryControllerState&0x01 == 0 {
				log.V(2).Info("Online controller")
				if err := s.device.OnlineController(sc.SecondaryControllerID); err != nil {
					log.Error(err, "Failed to online controller")
					break
				}

				ctrl.online = true
			}

			log.V(2).Info("Secondary controller initialized")

		} // for := secondary controllers

		for _, ctrl := range s.controllers[1:] {
			if !ctrl.online {
				s.state = sf.DISABLED_RST
				log.Info("Secondary Controller Offline")
				return nil
			}
		}
	}

	log.V(1).Info("Storage initialized")

	// Recover existing volumes
	log.V(2).Info("Recovering volumes")
	if err := s.recoverVolumes(); err != nil {
		log.Error(err, "Failed to recover existing volumes")
		return err
	}

	log.Info("Storage Ready")
	s.state = sf.ENABLED_RST

	event.EventManager.Publish(msgreg.PortAutomaticallyEnabledFabric(switchId, portId))

	return nil
}

func (s *Storage) LinkDroppedEventHandler() error {
	s.state = sf.UNAVAILABLE_OFFLINE_RST
	s.controllers = nil
	s.capacityBytes = 0

	return nil
}

// Get -
func (mgr *Manager) Get(model *sf.StorageCollectionStorageCollection) error {
	model.MembersodataCount = int64(len(mgr.storage))
	model.Members = make([]sf.OdataV4IdRef, int(model.MembersodataCount))
	for idx, s := range mgr.storage {
		model.Members[idx].OdataId = s.OdataId()
	}
	return nil
}

// StorageIdGet -
func (mgr *Manager) StorageIdGet(storageId string, model *sf.StorageV190Storage) error {
	s := findStorage(storageId)
	if s == nil {
		return ec.NewErrNotFound()
	}

	model.Id = s.id
	model.Status = s.getStatus()
	model.Identifiers = []sf.ResourceIdentifier{
		{
			DurableName:       s.qualifiedName,
			DurableNameFormat: sf.NQN_RV1100DNF,
		},
	}

	model.Location.PartLocation.LocationOrdinalValue = s.slot
	model.Location.PartLocation.LocationType = sf.SLOT_RV1100LT

	model.Controllers = s.OdataIdRef("/Controllers")
	model.StoragePools = s.OdataIdRef("/StoragePools")
	model.Volumes = s.OdataIdRef("/Volumes")

	return nil
}

// StorageIdStoragePoolsGet -
func (mgr *Manager) StorageIdStoragePoolsGet(storageId string, model *sf.StoragePoolCollectionStoragePoolCollection) error {
	s := findStorage(storageId)
	if s == nil {
		return ec.NewErrNotFound()
	}

	model.MembersodataCount = 1
	model.Members = make([]sf.OdataV4IdRef, model.MembersodataCount)
	model.Members[0] = s.OdataIdRef("/StoragePools/" + DefaultStoragePoolId)

	return nil
}

// StorageIdStoragePoolsStoragePoolIdGet -
func (mgr *Manager) StorageIdStoragePoolsStoragePoolIdGet(storageId, storagePoolId string, model *sf.StoragePoolV150StoragePool) error {
	if storagePoolId != DefaultStoragePoolId {
		return ec.NewErrNotFound().WithCause(fmt.Sprintf("storage pool %s not found", storagePoolId))
	}

	s := findStorage(storageId)
	if s == nil {
		return ec.NewErrNotFound().WithCause(fmt.Sprintf("storage %s not found", storageId))
	}

	model.CapacityBytes = int64(s.capacityBytes)

	// TODO: This should reflect the total namespaces allocated over the drive
	model.Capacity = sf.CapacityV100Capacity{
		Data: sf.CapacityV100CapacityInfo{
			AllocatedBytes:   int64(s.capacityBytes - s.unallocatedBytes),
			ConsumedBytes:    int64(s.capacityBytes - s.unallocatedBytes),
			GuaranteedBytes:  int64(s.unallocatedBytes),
			ProvisionedBytes: int64(s.capacityBytes),
		},
	}

	if s.capacityBytes == 0 {
		// If a drive could not be found, don't divide by zero.
		model.RemainingCapacityPercent = 0
	} else {
		model.RemainingCapacityPercent = int64(float64(s.unallocatedBytes/s.capacityBytes) * 100.0)
	}

	return nil
}

// StorageIdControllersGet -
func (mgr *Manager) StorageIdControllersGet(storageId string, model *sf.StorageControllerCollectionStorageControllerCollection) error {
	s := findStorage(storageId)
	if s == nil {
		return ec.NewErrNotFound()
	}

	model.MembersodataCount = int64(len(s.controllers))
	model.Members = make([]sf.OdataV4IdRef, model.MembersodataCount)
	for idx, c := range s.controllers {
		model.Members[idx] = s.OdataIdRef("/Controllers/" + c.id)
	}

	return nil
}

// StorageIdControllersControllerIdGet -
func (mgr *Manager) StorageIdControllersControllerIdGet(storageId, controllerId string, model *sf.StorageControllerV100StorageController) error {
	s, c := findStorageController(storageId, controllerId)
	if c == nil {
		return ec.NewErrNotFound().WithCause(fmt.Sprintf("Storage Controller not found: Storage: %s Controller: %s", storageId, controllerId))
	}

	// Fill in the relative endpoint for this storage controller
	endpointId, err := fabric.FabricController.FindDownstreamEndpoint(storageId, controllerId)
	if err != nil {
		return ec.NewErrNotFound().WithError(err).WithCause(fmt.Sprintf("Storage Controller fabric endpoint not found: Storage: %s Controller: %s", storageId, controllerId))
	}

	percentageUsage, err := s.device.GetWearLevelAsPercentageUsed()
	if err != nil {
		return ec.NewErrInternalServerError().WithError(err).WithCause(fmt.Sprintf("Storage Controller failed to retrieve SMART data: Storage: %s", storageId))
	}
	model.Id = c.id

	model.SerialNumber = s.serialNumber
	model.FirmwareVersion = s.firmwareRevision
	model.Model = s.modelNumber

	model.Links.EndpointsodataCount = 1
	model.Links.Endpoints = make([]sf.OdataV4IdRef, model.Links.EndpointsodataCount)
	model.Links.Endpoints[0].OdataId = endpointId

	// model.Links.PCIeFunctions

	/*
		f := sf.PcIeFunctionV123PcIeFunction{
			ClassCode: "",
			DeviceClass: "",
			DeviceId: "",
			VendorId: "",
			SubsystemId: "",
			SubsystemVendorId: "",
			FunctionId: 0,
			FunctionType: sf.PHYSICAL_PCIFV123FT, // or sf.VIRTUAL_PCIFV123FT
			Links: sf.PcIeFunctionV123Links {
				StorageControllersodataCount: 1,
				StorageControllers: make([]sf.StorageStorageController, 1),
			},
		}
	*/

	model.NVMeControllerProperties = sf.StorageControllerV100NvMeControllerProperties{
		ControllerType:           sf.IO_SCV100NVMCT, // OR ADMIN IF PF
		NVMeSMARTPercentageUsage: percentageUsage,
	}

	return nil
}

// StorageIdVolumesGet -
func (mgr *Manager) StorageIdVolumesGet(storageId string, model *sf.VolumeCollectionVolumeCollection) error {
	s := findStorage(storageId)
	if s == nil {
		return ec.NewErrNotFound()
	}

	// TODO: If s.ctrl is down - fail

	model.MembersodataCount = int64(len(s.volumes))
	model.Members = make([]sf.OdataV4IdRef, model.MembersodataCount)
	for idx, volume := range s.volumes {
		model.Members[idx] = s.OdataIdRef("/Volumes/" + volume.id)
	}

	return nil
}

// StorageIdVolumeIdGet -
func (mgr *Manager) StorageIdVolumeIdGet(storageId, volumeId string, model *sf.VolumeV161Volume) error {
	s, v := findStorageVolume(storageId, volumeId)
	if v == nil {
		return ec.NewErrNotFound()
	}

	// TODO: If s.ctrl is down - fail

	ns, err := s.device.IdentifyNamespace(nvme.NamespaceIdentifier(v.namespaceId))
	if err != nil {
		v.log.Error(err, "Identify Namespace Failed")
		return ec.NewErrInternalServerError()
	}

	lbaFormat := ns.LBAFormats[ns.FormattedLBASize.Format]
	blockSizeInBytes := int64(math.Pow(2, float64(lbaFormat.LBADataSize)))

	model.BlockSizeBytes = blockSizeInBytes
	model.CapacityBytes = int64(ns.Capacity) * blockSizeInBytes
	model.Id = v.id
	model.Identifiers = make([]sf.ResourceIdentifier, 2)
	model.Identifiers = []sf.ResourceIdentifier{
		{
			DurableNameFormat: sf.NSID_RV1100DNF,
			DurableName:       fmt.Sprintf("%d", v.namespaceId),
		},
		{
			DurableNameFormat: sf.NGUID_RV1100DNF,
			DurableName:       ns.GloballyUniqueIdentifier.String(),
		},
	}

	model.Capacity = sf.CapacityV100Capacity{
		IsThinProvisioned: ns.Features.Thinp == 1,
		Data: sf.CapacityV100CapacityInfo{
			AllocatedBytes: int64(ns.Capacity) * blockSizeInBytes,
			ConsumedBytes:  int64(ns.Utilization) * blockSizeInBytes,
		},
	}

	model.NVMeNamespaceProperties = sf.VolumeV161NvMeNamespaceProperties{
		FormattedLBASize:                  fmt.Sprintf("%d", model.BlockSizeBytes),
		IsShareable:                       ns.MultiPathIOSharingCapabilities.Sharing == 1,
		MetadataTransferredAtEndOfDataLBA: lbaFormat.MetadataSize != 0,
		NamespaceId:                       fmt.Sprintf("%d", v.namespaceId),
		NumberLBAFormats:                  int64(ns.NumberOfLBAFormats),
	}

	model.VolumeType = sf.RAW_DEVICE_VVT

	model.Links.OwningStorageResource.OdataId = s.OdataId()

	// TODO: Find the attached status of the volume - if it is attached via a connection
	// to an endpoint that should go in model.Links.ClientEndpoints or model.Links.ServerEndpoints

	// TODO: Maybe StorageGroups??? An array of references to Storage Groups that includes this volume.
	// Storage Groups could be the Rabbit Slice

	// TODO: Should reference the Storage Pool

	return nil
}

// StorageIdVolumesPost -
func (mgr *Manager) StorageIdVolumesPost(storageId string, model *sf.VolumeV161Volume) error {
	s := findStorage(storageId)
	if s == nil {
		return ec.NewErrNotFound()
	}

	volume, err := s.createVolume(uint64(model.CapacityBytes))

	// TODO: We should parse the error and make it more obvious (404, 405, etc)
	if err != nil {
		return err
	}

	return mgr.StorageIdVolumeIdGet(storageId, volume.id, model)
}

// StorageIdVolumeIdDelete -
func (mgr *Manager) StorageIdVolumeIdDelete(storageId, volumeId string) error {
	s, v := findStorageVolume(storageId, volumeId)
	if v == nil {
		return ec.NewErrBadRequest().WithCause(fmt.Sprintf("storage volume id %s not found", volumeId))
	}

	return s.deleteVolume(volumeId)
}
