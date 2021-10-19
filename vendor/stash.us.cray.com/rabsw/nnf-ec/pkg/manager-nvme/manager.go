package nvme

import (
	"flag"
	"fmt"
	"math"
	"strconv"

	. "stash.us.cray.com/rabsw/nnf-ec/pkg/api"
	event "stash.us.cray.com/rabsw/nnf-ec/pkg/manager-event"
	fabric "stash.us.cray.com/rabsw/nnf-ec/pkg/manager-fabric"
	msgreg "stash.us.cray.com/rabsw/nnf-ec/pkg/manager-message-registry/registries"

	log "github.com/sirupsen/logrus"

	ec "stash.us.cray.com/rabsw/nnf-ec/pkg/ec"

	"stash.us.cray.com/rabsw/nnf-ec/internal/switchtec/pkg/nvme"
	sf "stash.us.cray.com/rabsw/nnf-ec/pkg/rfsf/pkg/models"
)

const (
	ResourceBlockId = "Rabbit"

	// TODO: The ALL_CAPS name in nvme package should be renamed to a valid Go name
	CommonNamespaceIdentifier = nvme.COMMON_NAMESPACE_IDENTIFIER
)

const (
	defaultStoragePoolId = "0"
)

// Manager -
type Manager struct {
	id string

	config *ConfigFile

	storage []Storage
	ctrl    NvmeDeviceController

	// Command-Line Options
	purge       bool // Purge existing namespaces on storage controllers
	purgeMockDb bool // Purge the persistent mock databse
}

// Storage - Storage defines a generic storage device in the Redfish / Swordfish specification.
// In the NNF implementation
type Storage struct {
	id      string
	address string

	serialNumber     string
	modelNumber      string
	firmwareRevision string

	// Physical Function Controller ID
	pfid uint16

	// True if the host controller supports NVMe 1.3 Virtualization Management, false otherwise
	virtManagementEnabled bool

	// Capacity in bytes of the storage device. This value is read once and is fixed for
	// the life of the object.
	capacityBytes uint64

	// Unallocted capacity in bytes. This value is updated for any namespaces create or
	// delete operation that might shrink or grow the byte count as expected.
	unallocatedBytes uint64

	// Namespace Properties - Read using the Common Namespace Identifier (0xffffffff)
	// These are properties common to all namespaces for this controller (we use controller
	// zero as the basis for all other controllers - technically the spec supports uinque
	// LBA Formats per controller, but this is not done in practice by drive vendors.)
	lbaFormatIndex uint8
	blockSizeBytes uint32

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

	storage             *Storage
	attachedControllers []*StorageController
}

// TODO: We may want to put this manager under a resource block
//   /​redfish/​v1/​ResourceBlocks/​{ResourceBlockId} // <- Rabbit
//   /​redfish/​v1/​ResourceBlocks/​{ResourceBlockId}/​Systems/{​ComputerSystemId} // <- Also Rabbit & Computes
//   /​redfish/​v1/​ResourceBlocks/​{ResourceBlockId}/​Systems/{​ComputerSystemId}/​PCIeDevices/​{PCIeDeviceId}
//   /​redfish/​v1/​ResourceBlocks/​{ResourceBlockId}/​Systems/{​ComputerSystemId}/​PCIeDevices/​{PCIeDeviceId}/​PCIeFunctions/{​PCIeFunctionId}
//
//   /​redfish/​v1/​ResourceBlocks/{​ResourceBlockId}/​Systems/{​ComputerSystemId}/​Storage/​{StorageId}/​Controllers/​{ControllerId}

var mgr = Manager{id: ResourceBlockId}

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
		if err := s.refreshCapacity(); err != nil {
			return err
		}

		storageHandlerFunc(s.fmt("/StoragePools"), s.capacityBytes, s.unallocatedBytes)
	}

	return nil
}

func CreateVolume(s *Storage, capacityBytes uint64) (*Volume, error) {
	return s.createVolume(capacityBytes)
}

func DeleteVolume(v *Volume) error {
	return v.storage.deleteVolume(v.id)
}

func AttachControllers(v *Volume, controllers []uint16) error {
	return v.attach(controllers)
}

func DetachControllers(v *Volume, controllers []uint16) error {
	return v.detach(controllers)
}

func ReattachControllers(v *Volume) error {
	return v.recoverAttachedControllers()
}

func (s *Storage) UnallocatedBytes() uint64 { return s.unallocatedBytes }
func (s *Storage) IsEnabled() bool          { return s.state == sf.ENABLED_RST }
func (s *Storage) SerialNumber() string     { return s.serialNumber }

func (s *Storage) FindVolume(namespaceId nvme.NamespaceIdentifier) (*Volume, error) {
	for idx, volume := range s.volumes {
		if volume.namespaceId == namespaceId {
			return &s.volumes[idx], nil
		}
	}

	return nil, ec.NewErrNotFound()
}

func (s *Storage) fmt(format string, a ...interface{}) string {
	return fmt.Sprintf("/redfish/v1/Storage/%s", s.id) + fmt.Sprintf(format, a...)
}

func (s *Storage) initialize() error {
	s.state = sf.STARTING_RST

	ctrl, err := s.device.IdentifyController(0)
	if err != nil {
		return fmt.Errorf("Initialize Storage %s: Failed to indentify common controller: Error: %w", s.id, err)
	}

	s.pfid = ctrl.ControllerId

	s.serialNumber = string(ctrl.SerialNumber[:])
	s.modelNumber = string(ctrl.ModelNumber[:])
	s.firmwareRevision = string(ctrl.FirmwareRevision[:])

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

	ns, err := s.device.IdentifyNamespace(CommonNamespaceIdentifier)
	if err != nil {
		return fmt.Errorf("Initialize Storage %s: Failed to identify common namespace: Error: %w", s.id, err)
	}

	// Workaround for SSST drives that improperly report only one NumberOfLBAFormats, but actually
	// support two - with the second being the most performant 4096 sector size.
	if ns.NumberOfLBAFormats == 1 {

		if ((1 << ns.LBAFormats[1].LBADataSize) == 4096) && (ns.LBAFormats[1].RelativePerformance == 0) {
			log.Warnf("Initialize Storage %s: Detected Device %s; Incorrect number of LBA Formats. Expected: 2 Actual: %d", s.id, s.serialNumber, ns.NumberOfLBAFormats)
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

	return nil
}

func (s *Storage) purge() error {
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
	} else {
		stat.Health = sf.OK_RH
		stat.State = s.state
	}

	return stat
}

func (s *Storage) refreshCapacity() error {

	ctrl, err := s.device.IdentifyController(0)
	if err != nil {
		return err
	}

	capacityToUnit64 := func(c [16]byte) (lo uint64, hi uint64) {
		lo, hi = 0, 0
		for i := 0; i < 8; i++ {
			lo, hi = lo<<8, hi<<8
			lo += uint64(c[7-i])
			hi += uint64(c[15-i])
		}

		return lo, hi
	}

	totalCapacityLo, totalCapacityHi := capacityToUnit64(ctrl.TotalNVMCapacity)

	s.capacityBytes = totalCapacityLo
	if totalCapacityHi != 0 {
		return fmt.Errorf("Unsupported Capacity 0x%x_%x: will overflow uint64", totalCapacityHi, totalCapacityLo)
	}

	unallocatedCapacityLo, unallocatedCapacityHi := capacityToUnit64(ctrl.UnallocatedNVMCapacity)
	if unallocatedCapacityHi != 0 {
		return fmt.Errorf("Unsupported Capacity 0x%x_%x: will overflow uint64", unallocatedCapacityHi, unallocatedCapacityLo)
	}

	s.unallocatedBytes = unallocatedCapacityLo

	return nil
}

func (s *Storage) createVolume(capacityBytes uint64) (*Volume, error) {
	namespaceId, guid, err := s.device.CreateNamespace(capacityBytes, uint64(s.blockSizeBytes), s.lbaFormatIndex)
	// TODO: CreateNamespace can round up the requested capacity
	// Need to pass in a pointer here and then get the updated capacity
	// bytes programmed into the volume.
	if err != nil {
		return nil, err
	}

	id := strconv.Itoa(int(namespaceId))
	s.volumes = append(s.volumes, Volume{
		id:            id,
		namespaceId:   namespaceId,
		guid:          guid,
		capacityBytes: capacityBytes,
		storage:       s,
	})

	return &s.volumes[len(s.volumes)-1], nil
}

func (s *Storage) deleteVolume(volumeId string) error {
	for idx, volume := range s.volumes {
		if volume.id == volumeId {
			if err := s.device.DeleteNamespace(volume.namespaceId); err != nil {
				return err
			}

			// remove the volume from the array
			copy(s.volumes[idx:], s.volumes[idx+1:]) // shift left 1 at idx
			s.volumes = s.volumes[:len(s.volumes)-1] // truncate tail

			return nil
		}
	}

	return ec.NewErrNotFound()
}

func (s *Storage) recoverVolumes() error {
	namespaces, err := s.device.ListNamespaces(0)
	if err != nil {
		log.WithError(err).Errorf("Storage %s Failed to list device namespaces", s.id)
	}
	s.volumes = make([]Volume, 0)
	for _, nsid := range namespaces {
		ns, err := s.device.IdentifyNamespace(nsid)
		if err != nil {
			log.WithError(err).Errorf("Storage %s Failed to identify namespaces %d", s.id, nsid)
		}

		s.volumes = append(s.volumes, Volume{
			id:            strconv.Itoa(int(nsid)),
			namespaceId:   nsid,
			guid:          ns.GloballyUniqueIdentifier,
			capacityBytes: ns.Capacity,
			storage:       s,
		})

		if err := s.volumes[len(s.volumes)-1].recoverAttachedControllers(); err != nil {
			log.WithError(err).Errorf("Storage %s Failed to recover controllers", s.id)
		}
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

func (v *Volume) GetOdataId() string {
	return v.storage.fmt("/Volumes/%s", v.id)
}

func (v *Volume) GetCapaityBytes() uint64 {
	return uint64(v.capacityBytes)
}

func (v *Volume) GetNamespaceId() nvme.NamespaceIdentifier {
	return v.namespaceId
}

func (v *Volume) GetGloballyUniqueIdentifier() nvme.NamespaceGloballyUniqueIdentifier {
	return v.guid
}

func (v *Volume) SetFeature(data []byte) error {

	ctrls := []uint16{v.storage.pfid}
	if err := v.attach(ctrls); err != nil {
		return err
	}

	if err := v.storage.device.SetNamespaceFeature(v.namespaceId, data); err != nil {
		return err
	}

	return v.detach(ctrls)
}

func (v *Volume) DetachController(controllerId uint16) error {
	controllerIds, err := v.storage.device.ListAttachedControllers(v.namespaceId)
	if err != nil {
		return err
	}

	for _, ctrlId := range controllerIds {
		if ctrlId == controllerId {
			if err := v.detach([]uint16{controllerId}); err != nil {
				return err
			}
		}
	}

	return nil
}

func (v *Volume) attach(controllerIds []uint16) error {
	if err := v.storage.device.AttachNamespace(v.namespaceId, controllerIds); err != nil {
		return err
	}

	for _, controllerId := range controllerIds {
		for ctrlIdx, ctrl := range v.storage.controllers {
			if ctrl.controllerId == controllerId {
				v.attachedControllers = append(v.attachedControllers, &v.storage.controllers[ctrlIdx])
				break
			}
		}
	}

	return nil
}

func (v *Volume) detach(controllerIds []uint16) error {
	if err := v.storage.device.DetachNamespace(v.namespaceId, controllerIds); err != nil {
		return err
	}

	for _, controllerId := range controllerIds {
		for ctrlIdx, ctrl := range v.attachedControllers {
			if ctrl.controllerId == controllerId {
				v.attachedControllers =
					append(v.attachedControllers[:ctrlIdx], v.attachedControllers[ctrlIdx+1:]...)
				break
			}
		}
	}

	return nil
}

func (v *Volume) recoverAttachedControllers() error {
	if len(v.attachedControllers) != 0 {
		panic(fmt.Sprintf("Recover Attached Controllers - Volume %s has controllers already attached", v.id))
	}

	controllerIds, err := v.storage.device.ListAttachedControllers(v.namespaceId)
	if err != nil {
		return fmt.Errorf("Recover Attached Controllers - Failed to list attached controllers: Namespace: %d Error: %w", v.namespaceId, err)
	}

	for _, controllerId := range controllerIds {
		found := false
		for ctrlIdx, ctrl := range v.storage.controllers {
			if ctrl.controllerId == controllerId {
				v.attachedControllers = append(v.attachedControllers, &v.storage.controllers[ctrlIdx])
				found = true
				break
			}
		}

		if !found {
			return fmt.Errorf("Recover Attached Controllers - Storage controller not found: Controller: %d", controllerId)
		}
	}

	return nil
}

// Initialize
func Initialize(ctrl NvmeController) error {

	log.SetLevel(log.DebugLevel) // TODO: Config file or command-line option
	log.Infof("Initialize %s NVMe Namespace Manager", mgr.id)

	conf, err := loadConfig()
	if err != nil {
		log.WithError(err).Errorf("Failed to load %s configuration", mgr.id)
		return err
	}

	mgr.config = conf

	log.Debugf("NVMe Configuration '%s' Loaded...", conf.Metadata.Name)
	log.Debugf("  Debug Level: %s", conf.DebugLevel)
	log.Debugf("  Controller Config:")
	log.Debugf("    Virtual Functions: %d", conf.Storage.Controller.Functions)
	log.Debugf("    Num Resources: %d", conf.Storage.Controller.Resources)
	log.Debugf("  Device List: %+v", conf.Storage.Devices)

	level, err := log.ParseLevel(conf.DebugLevel)
	if err != nil {
		log.WithError(err).Errorf("Failed to parse debug level: %s", conf.DebugLevel)
		return err
	}

	log.SetLevel(level)

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
		log.WithError(err).Errorf("Failed to initialize NVMe Device Controller")
		return err
	}

	event.EventManager.Subscribe(&mgr)

	return nil
}

func Close() error {
	return mgr.ctrl.Close()
}

func (m *Manager) EventHandler(e event.Event) error {

	linkEstablished := e.Is(msgreg.DownstreamLinkEstablishedFabric("", "")) || e.Is(msgreg.DegradedDownstreamLinkEstablishedFabric("", ""))
	linkDropped := e.Is(msgreg.DownstreamLinkDroppedFabric("", ""))

	if linkEstablished || linkDropped {
		log.Infof("NVMe Manager: Event Received %+v", e)

		var switchId, portId string
		if err := e.Args(&switchId, &portId); err != nil {
			return ec.NewErrInternalServerError().WithError(err).WithCause("internal event format illformed")
		}

		idx, err := FabricController.GetDownstreamPortRelativePortIndex(switchId, portId)
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
	// Connect
	device, err := s.manager.ctrl.NewNvmeDevice(fabric.FabricId, switchId, portId)
	if err != nil {
		log.WithError(err).Errorf("Storage %s - Could not allocate storage controller", s.id)
		return err
	}

	s.device = device

	if err := s.initialize(); err != nil {
		log.WithError(err).Errorf("Storage %s - Failed to initialize device controller", s.id)
		return err
	}

	if s.manager.purge {
		log.Warnf("Storage %s - Starting purge of existing volumes", s.id)
		if err := s.purge(); err != nil {
			log.WithError(err).Errorf("Storage %s - Failed to purge storage device", s.id)
		}
	}

	if !s.virtManagementEnabled {
		s.controllers = make([]StorageController, 1 /* PF */ +s.config.Functions)
		for idx := range s.controllers {
			s.controllers[idx] = StorageController{
				id:             strconv.Itoa(idx),
				controllerId:   uint16(idx),
				functionNumber: uint16(idx),
			}
		}

	} else {

		ls, err := device.ListSecondary()
		if err != nil {
			log.WithError(err).Errorf("List Secondary command failed")
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
			controllerId:   0,
			functionNumber: 0,
		}

		for idx, sc := range ls.Entries[:count] {
			if sc.SecondaryControllerID == 0 {
				log.Errorf("Secondary Controller ID overlaps with PF Controller ID")
				break
			}

			s.controllers[idx+1] = StorageController{
				id:             strconv.Itoa(int(sc.SecondaryControllerID)),
				controllerId:   sc.SecondaryControllerID,
				functionNumber: sc.VirtualFunctionNumber,

				vqResources: sc.VQFlexibleResourcesAssigned,
				viResources: sc.VIFlexibleResourcesAssigned,
				online:      sc.SecondaryControllerState&0x01 == 1,
			}

			ctrl := &s.controllers[idx+1]

			log.Debugf("Storage %s Initialize Secondary Controller %s", s.id, ctrl.id)
			if sc.VQFlexibleResourcesAssigned != uint16(s.config.Resources) {
				if err := s.device.AssignControllerResources(sc.SecondaryControllerID, VQResourceType, s.config.Resources-uint32(sc.VQFlexibleResourcesAssigned)); err != nil {
					log.WithError(err).Errorf("Secondary Controller %d: Failed to assign VQ Resources", sc.SecondaryControllerID)
					break
				}

				ctrl.vqResources = uint16(s.config.Resources)
			}

			if sc.VIFlexibleResourcesAssigned != uint16(s.config.Resources) {
				if err := s.device.AssignControllerResources(sc.SecondaryControllerID, VIResourceType, s.config.Resources-uint32(sc.VIFlexibleResourcesAssigned)); err != nil {
					log.WithError(err).Errorf("Secondary Controller %d: Failed to assign VI Resources", sc.SecondaryControllerID)
					break
				}

				ctrl.viResources = uint16(s.config.Resources)
			}

			if sc.SecondaryControllerState&0x01 == 0 {
				if err := s.device.OnlineController(sc.SecondaryControllerID); err != nil {
					log.WithError(err).Errorf("Secondary Controller %d: Failed to online controller", sc.SecondaryControllerID)
					break
				}

				ctrl.online = true
			}

			log.Infof("Storage %s Secondary Controller %s Initialized", s.id, ctrl.id)

		} // for := secondary controllers

		for _, ctrl := range s.controllers[1:] {
			if !ctrl.online {
				s.state = sf.DISABLED_RST
				log.Errorf("Secondary Controller %s Offline - Storage %s Not Ready.", ctrl.id, s.id)
				return nil
			}
		}
	}

	// Recover existing volumes
	s.recoverVolumes()

	log.Infof("Storage %s - Ready", s.id)
	s.state = sf.ENABLED_RST

	event.EventManager.Publish(msgreg.PortAutomaticallyEnabledFabric(switchId, portId))

	return nil
}

func (s *Storage) LinkDroppedEventHandler() error {
	s.state = sf.UNAVAILABLE_OFFLINE_RST
	s.controllers = nil

	return nil
}

// Get -
func Get(model *sf.StorageCollectionStorageCollection) error {
	model.MembersodataCount = int64(len(mgr.storage))
	model.Members = make([]sf.OdataV4IdRef, int(model.MembersodataCount))
	for idx, s := range mgr.storage {
		model.Members[idx].OdataId = s.fmt("") // fmt.Sprintf("/redfish/v1/Storage/%s", s.id)
	}
	return nil
}

// StorageIdGet -
func StorageIdGet(storageId string, model *sf.StorageV190Storage) error {
	s := findStorage(storageId)
	if s == nil {
		return ec.NewErrNotFound()
	}

	model.Id = s.id
	model.Status = s.getStatus()

	// TODO: The model is missing a bunch of stuff
	// Manufacturer, Model, PartNumber, SerialNumber, etc.

	model.Controllers.OdataId = fmt.Sprintf("/redfish/v1/Storage/%s/Controllers", storageId)
	model.StoragePools.OdataId = fmt.Sprintf("/redfish/v1/Storage/%s/StoragePools", storageId)
	model.Volumes.OdataId = fmt.Sprintf("/redfish/v1/Storage/%s/Volumes", storageId)

	return nil
}

// StorageIdStoragePoolsGet -
func StorageIdStoragePoolsGet(storageId string, model *sf.StoragePoolCollectionStoragePoolCollection) error {
	s := findStorage(storageId)
	if s == nil {
		return ec.NewErrNotFound()
	}

	model.MembersodataCount = 1
	model.Members = make([]sf.OdataV4IdRef, model.MembersodataCount)
	model.Members[0].OdataId = fmt.Sprintf("/redfish/v1/Storage/%s/StoragePools/%s", storageId, defaultStoragePoolId)

	return nil
}

// StorageIdStoragePoolIdGet -
func StorageIdStoragePoolIdGet(storageId, storagePoolId string, model *sf.StoragePoolV150StoragePool) error {
	if storagePoolId != defaultStoragePoolId {
		return ec.NewErrNotFound().WithCause(fmt.Sprintf("storage pool %s not found", storagePoolId))
	}

	s := findStorage(storageId)
	if s == nil {
		return ec.NewErrNotFound().WithCause(fmt.Sprintf("storage %s not found", storageId))
	}

	if err := s.refreshCapacity(); err != nil {
		return ec.NewErrInternalServerError().WithError(err).WithCause(fmt.Sprintf("storage %s read capacity failed", storageId))
	}

	// TODO: This should reflect the total namespaces allocated over the drive
	model.Capacity = sf.CapacityV100Capacity{
		Data: sf.CapacityV100CapacityInfo{
			AllocatedBytes:   int64(s.capacityBytes - s.unallocatedBytes),
			ConsumedBytes:    int64(s.capacityBytes - s.unallocatedBytes),
			GuaranteedBytes:  int64(s.unallocatedBytes),
			ProvisionedBytes: int64(s.capacityBytes),
		},
	}

	model.RemainingCapacityPercent = int64(float64(s.unallocatedBytes/s.capacityBytes) * 100.0)

	return nil
}

// StorageIdControllersGet -
func StorageIdControllersGet(storageId string, model *sf.StorageControllerCollectionStorageControllerCollection) error {
	s := findStorage(storageId)
	if s == nil {
		return ec.NewErrNotFound()
	}

	model.MembersodataCount = int64(len(s.controllers))
	model.Members = make([]sf.OdataV4IdRef, model.MembersodataCount)
	for idx, c := range s.controllers {
		model.Members[idx].OdataId = fmt.Sprintf("/redfish/v1/Storage/%s/Controllers/%s", storageId, c.id)
	}

	return nil
}

// StorageIdControllerIdGet -
func StorageIdControllerIdGet(storageId, controllerId string, model *sf.StorageControllerV100StorageController) error {
	_, c := findStorageController(storageId, controllerId)
	if c == nil {
		return ec.NewErrNotFound().WithCause(fmt.Sprintf("Storage Controller not found: Storage: %s Controller: %s", storageId, controllerId))
	}

	// Fill in the relative endpoint for this storage controller
	endpointId, err := FabricController.FindDownstreamEndpoint(storageId, controllerId)
	if err != nil {
		return ec.NewErrNotFound().WithError(err).WithCause(fmt.Sprintf("Storage Controller fabric endpoint not found: Storage: %s Controller: %s", storageId, controllerId))
	}

	model.Id = c.id

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
		ControllerType: sf.IO_SCV100NVMCT, // OR ADMIN IF PF
	}

	return nil
}

// StorageIdVolumesGet -
func StorageIdVolumesGet(storageId string, model *sf.VolumeCollectionVolumeCollection) error {
	s := findStorage(storageId)
	if s == nil {
		return ec.NewErrNotFound()
	}

	// TODO: If s.ctrl is down - fail

	model.MembersodataCount = int64(len(s.volumes))
	model.Members = make([]sf.OdataV4IdRef, model.MembersodataCount)
	for idx, volume := range s.volumes {
		model.Members[idx].OdataId = s.fmt("/Volumes/%s", volume.id)
	}

	return nil
}

// StorageIdVolumeIdGet -
func StorageIdVolumeIdGet(storageId, volumeId string, model *sf.VolumeV161Volume) error {
	s, v := findStorageVolume(storageId, volumeId)
	if v == nil {
		return ec.NewErrNotFound()
	}

	// TODO: If s.ctrl is down - fail

	ns, err := s.device.IdentifyNamespace(nvme.NamespaceIdentifier(v.namespaceId))
	if err != nil {
		log.WithError(err).Errorf("Identify Namespace Failed: NSID %d", v.namespaceId)
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

	// TODO: Find the attached status of the volume - if it is attached via a connection
	// to an endpoint that should go in model.Links.ClientEndpoints or model.Links.ServerEndpoints

	// TODO: Maybe StorageGroups??? An array of references to Storage Groups that includes this volume.
	// Storage Groups could be the Rabbit Slice

	// TODO: Should reference the Storage Pool

	return nil
}

// StorageIdVolumePost -
func StorageIdVolumePost(storageId string, model *sf.VolumeV161Volume) error {
	s := findStorage(storageId)
	if s == nil {
		return ec.NewErrNotFound()
	}

	volume, err := s.createVolume(uint64(model.CapacityBytes))

	// TODO: We should parse the error and make it more obvious (404, 405, etc)
	if err != nil {
		return err
	}

	return StorageIdVolumeIdGet(storageId, volume.id, model)
}

// StorageIdVolumeIdDelete -
func StorageIdVolumeIdDelete(storageId, volumeId string) error {
	s, v := findStorageVolume(storageId, volumeId)
	if v == nil {
		return ec.NewErrBadRequest().WithCause(fmt.Sprintf("storage volume id %s not found", volumeId))
	}

	return s.deleteVolume(volumeId)
}
