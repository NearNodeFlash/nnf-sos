package nnf

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	. "stash.us.cray.com/rabsw/nnf-ec/pkg/events"

	ec "stash.us.cray.com/rabsw/nnf-ec/pkg/ec"
	fabric "stash.us.cray.com/rabsw/nnf-ec/pkg/manager-fabric"
	nvme "stash.us.cray.com/rabsw/nnf-ec/pkg/manager-nvme"
	server "stash.us.cray.com/rabsw/nnf-ec/pkg/manager-server"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"

	openapi "stash.us.cray.com/rabsw/rfsf-openapi/pkg/common"
	sf "stash.us.cray.com/rabsw/rfsf-openapi/pkg/models"
)

var storageService = StorageService{}

func NewDefaultStorageService() StorageServiceApi {
	return NewAerService(&storageService)
}

type StorageService struct {
	id string

	config                   *ConfigFile
	serverControllerProvider server.ServerControllerProvider

	pools       []StoragePool
	groups      []StorageGroup
	endpoints   []Endpoint
	fileSystems []FileSystem

	// Index of the Id field of any Storage Service resource (Pools, Groups, Endpoints, FileSystems)
	// That is, given a Storage Service resource OdataId field, ResourceIndex will correspond to the
	// index within the OdataId splity by "/" i.e.     strings.split(OdataId, "/")[ResourceIndex]
	resourceIndex int
}

type StoragePool struct {
	id          string
	name        string
	description string

	uid    uuid.UUID
	policy AllocationPolicy

	allocatedVolume  AllocatedVolume
	providingVolumes []ProvidingVolume

	storageGroups []*StorageGroup
	fileSystem    *FileSystem

	storageService *StorageService
}

type AllocatedVolume struct {
	id            string
	capacityBytes int64
}

type ProvidingVolume struct {
	volume *nvme.Volume
}

type Endpoint struct {
	id           string
	name         string
	controllerId uint16
	state        sf.ResourceState

	fabricId string

	// This is the Server Controller used for managing the endpoint
	serverCtrl server.ServerControllerApi

	config         *ServerConfig
	storageService *StorageService
}

type StorageGroup struct {
	id string

	volume *AllocatedVolume

	//
	// Endpoint represents the initiator server which has the Storage Group
	// accessible as a local storage system. Only a single Endpoint can be
	// associated with a Storage Group.
	endpoint *Endpoint

	// Server Storage represents a connection to the physical server endpoint that manages the
	// storage devices. This can be locally managed on the NNF contorller itself, or
	// remotely managed through some magical being not yet determined.
	serverStorage *server.Storage

	// If a file system is present on the parent Storage Pool, this object
	// represents the Exported File Share for this Storage Group, or null
	// if no file-system is present.
	fileShare *ExportedFileShare

	storagePool    *StoragePool
	storageService *StorageService
}

type FileSystem struct {
	id          string
	accessModes []string

	fsApi  server.FileSystemApi
	shares []ExportedFileShare

	storagePool    *StoragePool
	storageService *StorageService
}

type ExportedFileShare struct {
	id        string
	mountRoot string

	storageGroup *StorageGroup
	fileSystem   *FileSystem
}

const (
	DefaultCapacitySourceId            = "0"
	DefaultStorageServiceId            = "unassigned" // This is loaded from config
	DefaultStoragePoolCapacitySourceId = "0"
	DefaultAllocatedVolumeId           = "0"
)

func isStorageService(storageServiceId string) bool { return storageServiceId == storageService.id }
func findStorageService(storageServiceId string) *StorageService {
	if !isStorageService(storageServiceId) {
		return nil
	}

	return &storageService
}

func findStoragePool(storageServiceId, storagePoolId string) (*StorageService, *StoragePool) {
	s := findStorageService(storageServiceId)
	if s == nil {
		return nil, nil
	}

	return s, s.findStoragePool(storagePoolId)
}

func findStorageGroup(storageServiceId, storageGroupId string) (*StorageService, *StorageGroup) {
	s := findStorageService(storageServiceId)
	if s == nil {
		return nil, nil
	}

	return s, s.findStorageGroup(storageGroupId)
}

func findEndpoint(storageServiecId, endpointId string) (*StorageService, *Endpoint) {
	s := findStorageService(storageServiecId)
	if s == nil {
		return nil, nil
	}

	return s, s.findEndpoint(endpointId)
}

func findFileSystem(storageServiceId, fileSystemId string) (*StorageService, *FileSystem) {
	s := findStorageService(storageServiceId)
	if s == nil {
		return nil, nil
	}

	return s, s.findFileSystem(fileSystemId)
}

func findExportedFileShare(storageServiceId, fileSystemId, fileShareId string) (*StorageService, *FileSystem, *ExportedFileShare) {
	s, fs := findFileSystem(storageServiceId, fileSystemId)
	if fs == nil {
		return nil, nil, nil
	}

	return s, fs, fs.findExportedFileShare(fileShareId)
}

func (s *StorageService) OdataId() string {
	return fmt.Sprintf("/redfish/v1/StorageServices/%s", s.id)
}

func (s *StorageService) OdataIdRef(ref string) sf.OdataV4IdRef {
	return sf.OdataV4IdRef{OdataId: fmt.Sprintf("%s%s", s.OdataId(), ref)}
}

func (s *StorageService) findStoragePool(storagePoolId string) *StoragePool {
	for poolIdx, pool := range s.pools {
		if pool.id == storagePoolId {
			return &s.pools[poolIdx]
		}
	}

	return nil
}

func (s *StorageService) findStorageGroup(storageGroupId string) *StorageGroup {
	for groupIdx, group := range s.groups {
		if group.id == storageGroupId {
			return &s.groups[groupIdx]
		}
	}

	return nil
}

func (s *StorageService) findEndpoint(endpointId string) *Endpoint {
	for endpointIdx, endpoint := range s.endpoints {
		if endpoint.id == endpointId {
			return &s.endpoints[endpointIdx]
		}
	}

	return nil
}

func (s *StorageService) findFileSystem(fileSystemId string) *FileSystem {
	for fileSystemIdx, fileSystem := range s.fileSystems {
		if fileSystem.id == fileSystemId {
			return &s.fileSystems[fileSystemIdx]
		}
	}

	return nil
}

func (s *StorageService) createStoragePool(uid uuid.UUID, name string, description string, policy AllocationPolicy, providingVolumes []ProvidingVolume) *StoragePool {

	// Find a free Storage Pool Id
	var poolId = -1
	for _, p := range s.pools {
		id, _ := strconv.Atoi(p.id)

		if poolId <= id {
			poolId = id
		}
	}

	poolId = poolId + 1

	capacityBytes := int64(0)
	for _, v := range providingVolumes {
		capacityBytes += int64(v.volume.GetCapaityBytes())
	}

	p := &StoragePool{
		id:               strconv.Itoa(poolId),
		name:             name,
		description:      description,
		uid:              uid,
		policy:           policy,
		providingVolumes: providingVolumes,
		storageService:   s,
	}

	p.allocatedVolume = AllocatedVolume{
		id:            DefaultAllocatedVolumeId,
		capacityBytes: capacityBytes,
	}

	if p.name == "" {
		p.name = fmt.Sprintf("Storage Pool %s", p.id)
	}

	return p
}

func (s *StorageService) createStorageGroup(sp *StoragePool, endpoint *Endpoint) *StorageGroup {

	// Find a free Storage Group Id
	var groupId = -1
	for _, g := range s.groups {
		id, _ := strconv.Atoi(g.id)

		if groupId <= id {
			groupId = id
		}
	}

	groupId = groupId + 1

	return &StorageGroup{
		id:             strconv.Itoa(groupId),
		endpoint:       endpoint,
		serverStorage:  endpoint.serverCtrl.NewStorage(sp.uid),
		storagePool:    sp,
		storageService: s,
	}
}

func (s *StorageService) deleteStorageGroup(sg *StorageGroup) {
	sp := sg.storagePool
	for storageGroupIdx, storageGroup := range sp.storageGroups {
		if storageGroup.id == sg.id {
			sp.storageGroups = append(sp.storageGroups[:storageGroupIdx], sp.storageGroups[storageGroupIdx+1:]...)
			break
		}
	}

	for storageGroupIdx, storageGroup := range s.groups {
		if storageGroup.id == sg.id {
			s.groups = append(s.groups[:storageGroupIdx], s.groups[storageGroupIdx+1:]...)
			break
		}
	}
}

func (s *StorageService) allocateStoragePoolUid() uuid.UUID {
	for {
	Retry:
		uid := uuid.New()

		for _, p := range s.pools {
			if p.uid == uid {
				goto Retry
			}
		}

		return uid
	}
}

func (s *StorageService) createFileSystem(sp *StoragePool, fsApi server.FileSystemApi) *FileSystem {

	// Find a free File System Id
	var fileSystemId = -1
	for _, fs := range s.fileSystems {
		id, _ := strconv.Atoi(fs.id)

		if fileSystemId <= id {
			fileSystemId = id
		}
	}

	fileSystemId = fileSystemId + 1

	return &FileSystem{
		id:             strconv.Itoa(fileSystemId),
		fsApi:          fsApi,
		storagePool:    sp,
		storageService: s,
	}
}

func (p *StoragePool) OdataId() string {
	return fmt.Sprintf("%s/StoragePools/%s", p.storageService.OdataId(), p.id)
}

func (p *StoragePool) OdataIdRef(ref string) sf.OdataV4IdRef {
	return sf.OdataV4IdRef{OdataId: fmt.Sprintf("%s%s", p.OdataId(), ref)}
}

func (p *StoragePool) isCapacitySource(capacitySourceId string) bool {
	return capacitySourceId == DefaultStoragePoolCapacitySourceId
}

func (p *StoragePool) isAllocatedVolume(volumeId string) bool {
	return volumeId == DefaultAllocatedVolumeId
}

func (p *StoragePool) capacitySourcesGet() []sf.CapacityCapacitySource {
	return []sf.CapacityCapacitySource{
		{
			OdataId:   p.OdataId() + "/CapacitySources",
			OdataType: "#CapacitySource.v1_0_0.CapacitySource",
			Name:      "Capacity Source",
			Id:        DefaultStoragePoolCapacitySourceId,

			ProvidedCapacity: sf.CapacityV120Capacity{
				// TODO
			},

			ProvidingVolumes: p.OdataIdRef(fmt.Sprintf("/CapacitySources/%s/ProvidingVolumes", DefaultStoragePoolCapacitySourceId)),
		},
	}
}

func (p *StoragePool) findStorageGroupByEndpoint(endpoint *Endpoint) *StorageGroup {
	for _, sg := range p.storageGroups {
		if sg.endpoint.id == endpoint.id {
			return sg
		}
	}

	return nil
}

func (ep *Endpoint) OdataId() string {
	return fmt.Sprintf("%s/Endpoints/%s", ep.storageService.OdataId(), ep.id)
}

func (sg *StorageGroup) OdataId() string {
	return fmt.Sprintf("%s/StorageGroups/%s", sg.storageService.OdataId(), sg.id)
}

func (sg *StorageGroup) OdataIdRef(ref string) sf.OdataV4IdRef {
	return sf.OdataV4IdRef{OdataId: fmt.Sprintf("%s%s", sg.OdataId(), ref)}
}

func (sg *StorageGroup) status() *sf.ResourceStatus {
	state := sg.serverStorage.GetStatus().State()

	health := sf.OK_RH
	if state != sf.ENABLED_RST {
		health = sf.WARNING_RH
	}

	return &sf.ResourceStatus{Health: health, State: state}
}

func (fs *FileSystem) OdataId() string {
	return fmt.Sprintf("%s/FileSystems/%s", fs.storageService.OdataId(), fs.id)
}

func (fs *FileSystem) OdataIdRef(ref string) sf.OdataV4IdRef {
	return sf.OdataV4IdRef{OdataId: fmt.Sprintf("%s%s", fs.OdataId(), ref)}
}

func (fs *FileSystem) findExportedFileShare(id string) *ExportedFileShare {
	for fileShareIdx, fileShare := range fs.shares {
		if fileShare.id == id {
			return &fs.shares[fileShareIdx]
		}
	}

	return nil
}

func (fs *FileSystem) createFileShare(sg *StorageGroup, mountRoot string) *ExportedFileShare {
	var fileShareId = -1
	for _, fileShare := range fs.shares {
		id, _ := strconv.Atoi(fileShare.id)

		if fileShareId <= id {
			fileShareId = id
		}
	}

	fileShareId = fileShareId + 1

	return &ExportedFileShare{
		id:           strconv.Itoa(fileShareId),
		storageGroup: sg,
		mountRoot:    mountRoot,
		fileSystem:   fs,
	}
}

func (sh *ExportedFileShare) initialize(mountpoint string) error {

	opts := server.FileSystemOptions{
		"mountpoint": mountpoint,
	}

	return sh.storageGroup.serverStorage.CreateFileSystem(sh.fileSystem.fsApi, opts)
}

func (sh *ExportedFileShare) getStatus() *sf.ResourceStatus {

	return &sf.ResourceStatus{
		Health: sf.OK_RH,
		State:  sh.storageGroup.serverStorage.GetStatus().State(),
	}
}

func (sh *ExportedFileShare) OdataId() string {
	return fmt.Sprintf("%s/ExportedFileShares/%s", sh.fileSystem.OdataId(), sh.id)
}

func (s *StorageService) Id() string {
	return s.id
}

//
// Initialize is responsible for initializing the NNF Storage Service; the
// Storage Service must complete initialization without error prior any
// access to the Storage Service. Failure to initialize will cause the
// storage service to misbehave.
//
func (*StorageService) Initialize(ctrl NnfControllerInterface) error {

	storageService = StorageService{
		id:                       DefaultStorageServiceId,
		serverControllerProvider: ctrl.ServerControllerProvider(),
	}

	s := &storageService

	conf, err := loadConfig()
	if err != nil {
		log.WithError(err).Errorf("Failed to load %s configuration", s.id)
		return err
	}

	s.id = conf.Id
	s.config = conf

	log.Debugf("NNF Storage Service '%s' Loaded...", conf.Metadata.Name)
	log.Debugf("  Remote Config    : %+v", conf.RemoteConfig)
	log.Debugf("  Allocation Config: %+v", conf.AllocationConfig)

	s.endpoints = make([]Endpoint, len(conf.RemoteConfig.Servers))
	for endpointIdx := range s.endpoints {
		s.endpoints[endpointIdx] = Endpoint{
			id:             strconv.Itoa(endpointIdx),
			state:          sf.UNAVAILABLE_OFFLINE_RST,
			config:         &conf.RemoteConfig.Servers[endpointIdx],
			storageService: s,
		}
	}

	// Index of a individual resource located off of the collections managed
	// by the NNF Storage Service.
	s.resourceIndex = strings.Count(s.OdataIdRef("/StoragePool/0").OdataId, "/")

	PortEventManager.Subscribe(PortEventSubscriber{
		HandlerFunc: PortEventHandler,
		Data:        s,
	})

	// Initialize the Server Manager - considered internal to
	// the NNF Manager
	if err := server.Initialize(); err != nil {
		log.WithError(err).Errorf("Failed to Initialize Server Manager")
		return err
	}

	return nil
}

// PortEventHandler is a callback to the Port Event Manager for receiving
// Port Events.
func PortEventHandler(event PortEvent, data interface{}) {
	s := data.(*StorageService)

	if event.PortType != USP_PortType {
		return
	}

	ep, err := fabric.GetEndpointFromPortEvent(event)
	if err != nil {
		log.WithError(err).Errorf("Unable to find endpoint for event %+v", event)
		return
	}

	endpointIdx := ep.Index()
	if !(endpointIdx < len(s.endpoints)) {
		log.Errorf("Endpoint index %d beyond supported endpoint count %d", endpointIdx, len(s.endpoints))
		return
	}

	isNnf := func(ep *fabric.Endpoint) bool {
		model := sf.EndpointV150Endpoint{}
		fabric.FabricIdEndpointsEndpointIdGet(event.FabricId, ep.Id(), &model)
		if model.ConnectedEntities[0].EntityType == sf.PROCESSOR_EV150ET {
			return true
		}

		return false
	}

	endpoint := &s.endpoints[endpointIdx]

	endpoint.id = ep.Id()
	endpoint.name = ep.Name()
	endpoint.controllerId = ep.ControllerId()
	endpoint.fabricId = event.FabricId // Should just link to the fabric.Endpoint

	opts := server.ServerControllerOptions{
		Local:   isNnf(ep),
		Address: endpoint.config.Address,
	}

	endpoint.serverCtrl = s.serverControllerProvider.NewServerController(opts)

	if endpoint.serverCtrl.Connected() {
		endpoint.state = sf.ENABLED_RST
	}
}

//
// Methods below this comment block are all service handlers that provide the various
// CRUD (Create, Read, Update, and Delete) functionality. The method pattern is always
// the same:
//    StorageService[Sub Resource ...][HTTP Method]([Ids string...], model *[Swordfish Model])
// where
//    Sub Resource: defines the hierarchy of resources, beginning at the NNF Storage
//       Service and into the Redfish/Swordfish object tree
//    HTTP Method: defines a HTTP access method. One of...
//       Post (Create): Used to create an object in the Storage Service
//       Get (Read): Used to read an object from the Storage Service
//       Patch (Update): Used to update an existing object managed by the Storage Service
//       Delete: Used to delete an object managed by the Storage Service
//    IDs: defines one or more IDs that identify a resource
//    Swordfish Model: a Swordfish structure for which HTTP Method should act on.
//       Create: the method will attempt to create the object from the provided model
//       Get: the method will populate the model.
//       Patch: the method will update parameters with parameters from the model
//       Delete: There is no model parameter.
//

func (*StorageService) StorageServicesGet(model *sf.StorageServiceCollectionStorageServiceCollection) error {

	model.MembersodataCount = 1
	model.Members = make([]sf.OdataV4IdRef, model.MembersodataCount)
	model.Members[0] = sf.OdataV4IdRef{
		OdataId: fmt.Sprintf("/redfish/v1/StorageServices/%s", storageService.id),
	}

	return nil
}

func (*StorageService) StorageServiceIdGet(storageServiceId string, model *sf.StorageServiceV150StorageService) error {
	s := findStorageService(storageServiceId)
	if s == nil {
		return ec.NewErrNotFound().WithCause(fmt.Sprintf("Storage service '%s' not found", storageServiceId))
	}

	model.Id = s.id

	model.Status.State = sf.ENABLED_RST
	model.Status.Health = sf.OK_RH

	model.StoragePools = s.OdataIdRef("/StoragePools")
	model.StorageGroups = s.OdataIdRef("/StorageGroups")
	model.Endpoints = s.OdataIdRef("/Endpoints")
	model.FileSystems = s.OdataIdRef("/FileSystems")

	model.Links.CapacitySource = s.OdataIdRef("/CapacitySource")
	return nil
}

func (*StorageService) StorageServiceIdCapacitySourceGet(storageServiceId string, model *sf.CapacityCapacitySource) error {
	s := findStorageService(storageServiceId)
	if s == nil {
		return ec.NewErrNotFound().WithCause(fmt.Sprintf("Storage service '%s' not found", storageServiceId))
	}

	model.Id = DefaultCapacitySourceId

	totalCapacityBytes, totalUnallocatedBytes := uint64(0), uint64(0)
	if err := nvme.EnumerateStorage(func(odataId string, capacityBytes uint64, unallocatedBytes uint64) {

		// TODO: OdataId could be used to link to the underlying StoragePool that is the NVMe Device
		//       That would require a new Storage Pool Collection that lives off the CapacitySource and
		//       is properly linked.
		totalCapacityBytes += capacityBytes
		totalUnallocatedBytes += unallocatedBytes
	}); err != nil {
		return ec.NewErrInternalServerError().WithError(err).WithCause("Failed to enumerate storage")
	}

	model.ProvidedCapacity.Data.GuaranteedBytes = int64(totalUnallocatedBytes)
	model.ProvidedCapacity.Data.ProvisionedBytes = int64(totalCapacityBytes)
	model.ProvidedCapacity.Data.AllocatedBytes = int64(totalCapacityBytes - totalUnallocatedBytes)
	model.ProvidedCapacity.Data.ConsumedBytes = model.ProvidedCapacity.Data.AllocatedBytes

	return nil
}

func (*StorageService) StorageServiceIdStoragePoolsGet(storageServiceId string, model *sf.StoragePoolCollectionStoragePoolCollection) error {
	s := findStorageService(storageServiceId)
	if s == nil {
		return ec.NewErrNotFound().WithCause(fmt.Sprintf("Storage service '%s' not found", storageServiceId))
	}

	model.MembersodataCount = int64(len(s.pools))
	model.Members = make([]sf.OdataV4IdRef, model.MembersodataCount)
	for poolIdx, pool := range s.pools {
		model.Members[poolIdx] = sf.OdataV4IdRef{OdataId: pool.OdataId()}
	}

	return nil
}

// StorageServiceIdStoragePoolsPost -
func (*StorageService) StorageServiceIdStoragePoolsPost(storageServiceId string, model *sf.StoragePoolV150StoragePool) error {
	s := findStorageService(storageServiceId)
	if s == nil {
		return ec.NewErrNotFound().WithCause(fmt.Sprintf("Storage service '%s' not found", storageServiceId))
	}

	// TODO: Check the model for valid RAID configurations

	policy := NewAllocationPolicy(s.config.AllocationConfig, model.Oem)
	if policy == nil {
		log.Errorf("Failed to allocate storage policy. Config: %+v Oem: %+v", s.config.AllocationConfig, model.Oem)
		return ec.NewErrNotAcceptable().WithCause("Failed to allocate storage policy")
	}

	capacityBytes := model.CapacityBytes
	if capacityBytes == 0 {
		capacityBytes = model.Capacity.Data.AllocatedBytes
	}

	if capacityBytes == 0 {
		return ec.NewErrNotAcceptable().WithCause("No capacity defined")
	}

	if err := policy.Initialize(uint64(capacityBytes)); err != nil {
		log.WithError(err).Errorf("Failed to initialize storage policy")
		return ec.NewErrInternalServerError().WithError(err).WithCause("Failed to initialize storage policy")
	}

	if err := policy.CheckCapacity(); err != nil {
		log.WithError(err).Warnf("Storage Policy does not provide sufficient capacity to support requested %d bytes", capacityBytes)
		return ec.NewErrNotAcceptable().WithError(err).WithCause("Insufficient capacity available")
	}

	// All checks have completed; we're ready to create the storage pool
	uid := s.allocateStoragePoolUid()

	log.Infof("Allocating storage for PID %s", uid.String())
	volumes, err := policy.Allocate(uid)
	if err != nil {

		// TOOD: there may be partial volumes for this pool - we need to mark
		// them for delete in the ledger and attempt to release them.

		log.WithError(err).Errorf("Storage Policy allocation failed.")
		return ec.NewErrInternalServerError().WithError(err).WithCause("Failed to allocate storage volumes")
	}

	p := s.createStoragePool(uid, model.Name, model.Description, policy, volumes)
	s.pools = append(s.pools, *p)

	return s.StorageServiceIdStoragePoolIdGet(storageServiceId, p.id, model)
}

// StorageServiceIdStoragePoolIdGet -
func (*StorageService) StorageServiceIdStoragePoolIdGet(storageServiceId, storagePoolId string, model *sf.StoragePoolV150StoragePool) error {
	_, p := findStoragePool(storageServiceId, storagePoolId)
	if p == nil {
		return ec.NewErrNotFound().WithCause(fmt.Sprintf("Storage Pool '%s' not found", storagePoolId))
	}

	model.Id = p.id
	model.OdataId = p.OdataId()
	model.AllocatedVolumes = p.OdataIdRef("/AlloctedVolumes")

	model.BlockSizeBytes = 4096 // TODO
	model.Capacity = sf.CapacityV100Capacity{
		Data: sf.CapacityV100CapacityInfo{
			AllocatedBytes:   p.allocatedVolume.capacityBytes,
			ProvisionedBytes: p.allocatedVolume.capacityBytes,
			ConsumedBytes:    0, // TODO
		},
	}

	model.CapacityBytes = p.allocatedVolume.capacityBytes
	model.CapacitySources = p.capacitySourcesGet()
	model.CapacitySourcesodataCount = int64(len(model.CapacitySources))

	model.Identifier = sf.ResourceIdentifier{
		DurableName:       p.uid.String(),
		DurableNameFormat: sf.UUID_RV1100DNF,
	}

	model.Status.State = sf.ENABLED_RST
	model.Status.Health = sf.OK_RH

	model.Links.StorageGroupsodataCount = int64(len(p.storageGroups))
	model.Links.StorageGroups = make([]sf.OdataV4IdRef, model.Links.StorageGroupsodataCount)
	for idx, sg := range p.storageGroups {
		model.Links.StorageGroups[idx] = sf.OdataV4IdRef{OdataId: sg.OdataId()}
	}

	if p.fileSystem != nil {
		model.Links.FileSystem = sf.OdataV4IdRef{OdataId: p.fileSystem.OdataId()}
	}

	return nil
}

// StorageServiceIdStoragePoolIdDelete -
func (*StorageService) StorageServiceIdStoragePoolIdDelete(storageServiceId, storagePoolId string) error {
	s, p := findStoragePool(storageServiceId, storagePoolId)
	if p == nil {
		return ec.NewErrNotFound().WithCause(fmt.Sprintf("Storage Pool '%s' not found", storagePoolId))
	}

	if p.fileSystem != nil {
		if err := s.StorageServiceIdFileSystemIdDelete(s.id, p.fileSystem.id); err != nil {
			return ec.NewErrInternalServerError().WithError(err).WithCause(fmt.Sprintf("Failed to delete file system '%s'", p.fileSystem.id))
		}
	}

	for _, sg := range p.storageGroups {
		if err := s.StorageServiceIdStorageGroupIdDelete(s.id, sg.id); err != nil {
			return ec.NewErrInternalServerError().WithError(err).WithCause(fmt.Sprintf("Failed to delete storage group '%s'", sg.id))
		}
	}

	for _, pv := range p.providingVolumes {
		if err := nvme.DeleteVolume(pv.volume); err != nil {
			log.WithError(err).Errorf("Failed to delete volume from storage pool %s", p.id)
			return ec.NewErrInternalServerError().WithError(err).WithCause(fmt.Sprintf("Failed to delete volume"))
		}

		// TODO: If any delete fails, we're left with dangling volumes preventing
		// further deletion. Need to fix or recover from this. Maybe a transaction
		// log.
	}

	for idx, pool := range s.pools {
		if pool.id == storagePoolId {
			copy(s.pools[idx:], s.pools[idx+1:])
			s.pools = s.pools[:len(s.pools)-1]
		}
	}

	return nil
}

// StorageServiceIdStoragePoolIdCapacitySourcesGet -
func (*StorageService) StorageServiceIdStoragePoolIdCapacitySourcesGet(storageServiceId, storagePoolId string, model *sf.CapacitySourceCollectionCapacitySourceCollection) error {
	_, p := findStoragePool(storageServiceId, storagePoolId)
	if p == nil {
		return ec.NewErrNotFound().WithCause(fmt.Sprintf("Storage pool '%s' not found", storagePoolId))
	}

	model.Members = p.capacitySourcesGet()
	model.MembersodataCount = int64(len(model.Members))

	return nil
}

// StorageServiceIdStoragePoolIdCapacitySourceIdGet -
func (*StorageService) StorageServiceIdStoragePoolIdCapacitySourceIdGet(storageServiceId, storagePoolId, capacitySourceId string, model *sf.CapacityCapacitySource) error {
	_, p := findStoragePool(storageServiceId, storagePoolId)
	if p == nil {
		return ec.NewErrNotFound().WithCause(fmt.Sprintf("Storage pool '%s' not found", storagePoolId))
	}

	if !p.isCapacitySource(capacitySourceId) {
		return ec.NewErrNotFound().WithCause(fmt.Sprintf("Capacity source '%s' not found", capacitySourceId))
	}

	s := p.capacitySourcesGet()[0]

	model.Id = s.Id
	model.ProvidedCapacity = s.ProvidedCapacity
	model.ProvidingVolumes = s.ProvidingVolumes

	return nil
}

// StorageServiceIdStoragePoolIdCapacitySourceIdProvidingVolumesGet -
func (*StorageService) StorageServiceIdStoragePoolIdCapacitySourceIdProvidingVolumesGet(storageServiceId, storagePoolId, capacitySourceId string, model *sf.VolumeCollectionVolumeCollection) error {
	_, p := findStoragePool(storageServiceId, storagePoolId)
	if p == nil {
		return ec.NewErrNotFound().WithCause(fmt.Sprintf("Storage pool '%s' not found", storagePoolId))
	}

	if !p.isCapacitySource(capacitySourceId) {
		return ec.NewErrNotFound().WithCause(fmt.Sprintf("Capacity source '%s' not found", capacitySourceId))
	}

	model.MembersodataCount = int64(len(p.providingVolumes))
	model.Members = make([]sf.OdataV4IdRef, model.MembersodataCount)
	for idx, v := range p.providingVolumes {
		model.Members[idx] = sf.OdataV4IdRef{OdataId: v.volume.GetOdataId()}
	}

	return nil
}

// StorageServiceIdStoragePoolIdAlloctedVolumesGet -
func (*StorageService) StorageServiceIdStoragePoolIdAlloctedVolumesGet(storageServiceId, storagePoolId string, model *sf.VolumeCollectionVolumeCollection) error {
	_, p := findStoragePool(storageServiceId, storagePoolId)
	if p == nil {
		return ec.NewErrNotFound().WithCause(fmt.Sprintf("Storage pool '%s' not found", storagePoolId))
	}

	model.MembersodataCount = 1
	model.Members = []sf.OdataV4IdRef{
		p.OdataIdRef(fmt.Sprintf("/AllocatedVolumes/%s", DefaultAllocatedVolumeId)),
	}

	return nil
}

// StorageServiceIdStoragePoolIdAllocatedVolumeIdGet -
func (*StorageService) StorageServiceIdStoragePoolIdAllocatedVolumeIdGet(storageServiceId, storagePoolId, volumeId string, model *sf.VolumeV161Volume) error {
	_, p := findStoragePool(storageServiceId, storagePoolId)
	if p == nil {
		return ec.NewErrNotFound().WithCause(fmt.Sprintf("Storage pool '%s' not found", storagePoolId))
	}

	if !p.isAllocatedVolume(volumeId) {
		return ec.NewErrNotFound().WithCause(fmt.Sprintf("Volume '%s' not found", volumeId))
	}

	model.Id = DefaultAllocatedVolumeId
	model.CapacityBytes = int64(p.allocatedVolume.capacityBytes)
	model.Capacity = sf.CapacityV100Capacity{
		// TODO???
	}

	model.Identifiers = []sf.ResourceIdentifier{
		{
			DurableName:       p.uid.String(),
			DurableNameFormat: sf.NGUID_RV1100DNF,
		},
	}

	model.VolumeType = sf.RAW_DEVICE_VVT

	return nil
}

// StorageServiceIdStorageGroupsGet -
func (*StorageService) StorageServiceIdStorageGroupsGet(storageServiceId string, model *sf.StorageGroupCollectionStorageGroupCollection) error {
	s := findStorageService(storageServiceId)
	if s == nil {
		return ec.NewErrNotFound().WithCause(fmt.Sprintf("Storage service '%s' not found", storageServiceId))
	}

	model.MembersodataCount = int64(len(s.groups))
	model.Members = make([]sf.OdataV4IdRef, model.MembersodataCount)
	for groupIdx, group := range s.groups {
		model.Members[groupIdx] = sf.OdataV4IdRef{OdataId: group.OdataId()}
	}

	return nil
}

// StorageServiceIdStorageGroupPost -
func (*StorageService) StorageServiceIdStorageGroupPost(storageServiceId string, model *sf.StorageGroupV150StorageGroup) error {
	s := findStorageService(storageServiceId)
	if s == nil {
		return ec.NewErrNotFound().WithCause(fmt.Sprintf("Storage service '%s' not found", storageServiceId))
	}

	fields := strings.Split(model.Links.StoragePool.OdataId, "/")
	if len(fields) != s.resourceIndex+1 {
		return ec.NewErrNotAcceptable().WithCause(fmt.Sprintf("Storage pool @odata.id '%s' malformed", model.Links.StoragePool.OdataId))
	}

	storagePoolId := fields[s.resourceIndex]

	_, sp := findStoragePool(storageServiceId, storagePoolId)
	if sp == nil {
		return ec.NewErrNotAcceptable().WithCause(fmt.Sprintf("Storage pool '%s' not found", storagePoolId))
	}

	fields = strings.Split(model.Links.ServerEndpoint.OdataId, "/")
	if len(fields) != s.resourceIndex+1 {
		return ec.NewErrNotAcceptable().WithCause(fmt.Sprintf("Server endpoint @odata.id '%s' malformed", model.Links.ServerEndpoint.OdataId))
	}

	endpointId := fields[s.resourceIndex]

	ep := s.findEndpoint(endpointId)
	if ep == nil {
		return ec.NewErrNotAcceptable().WithCause(fmt.Sprintf("Server endpoint '%s' not found", storagePoolId))
	}

	if !ep.serverCtrl.Connected() {
		return ec.NewErrNotAcceptable().WithCause(fmt.Sprintf("Server endpoint '%s' not connected", endpointId))
	}

	// Everything validated OK - create the Storage Group

	sg := s.createStorageGroup(sp, ep)
	sp.storageGroups = append(sp.storageGroups, sg)
	s.groups = append(s.groups, *sg)

	model.Id = sg.id
	model.OdataId = sg.OdataId()

	// And now we attempt to bring-up the storage group. Any error here
	// is logged, but does not cause the request to fail; instead the
	// resource status reflects the outcome.

	for _, volume := range sp.providingVolumes {
		if err := nvme.AttachControllers(volume.volume, []uint16{sg.endpoint.controllerId}); err != nil {
			log.WithError(err).Errorf("Failed to attach controllers")
		}
	}

	return s.StorageServiceIdStorageGroupIdGet(storageServiceId, sg.id, model)
}

// StorageServiceIdStorageGroupIdGet -
func (*StorageService) StorageServiceIdStorageGroupIdGet(storageServiceId, storageGroupId string, model *sf.StorageGroupV150StorageGroup) error {
	_, sg := findStorageGroup(storageServiceId, storageGroupId)
	if sg == nil {
		return ec.NewErrNotFound().WithCause(fmt.Sprintf("Storage group '%s' not found", storageGroupId))
	}

	model.Id = sg.id
	model.OdataId = sg.OdataId()

	// TODO: Mapped Volumes should point to the corresponding Storage Volume
	//       As they are present on the Server Storage Controller.

	// TODO:
	// model.Volumes - Should point to the volumes which are present on the Storage Endpoint,
	//                 once exposed. This means iterating on the Storage Endpoints and asking
	//                 every Storage Controller for status on the volumes in the Storage Pool.
	model.MappedVolumes = []sf.StorageGroupMappedVolume{
		{
			AccessCapability: sf.READ_WRITE_SGAC,
			Volume:           sg.storagePool.OdataIdRef(fmt.Sprintf("/AllocatedVolumes/%s", DefaultAllocatedVolumeId)),
		},
	}

	model.Links.ServerEndpoint = sf.OdataV4IdRef{OdataId: sg.endpoint.OdataId()}
	model.Links.StoragePool = sf.OdataV4IdRef{OdataId: sg.storagePool.OdataId()}

	model.Status = *sg.status()

	return nil
}

// StorageServiceIdStorageGroupIdDelete -
func (*StorageService) StorageServiceIdStorageGroupIdDelete(storageServiceId, storageGroupId string) error {
	s, sg := findStorageGroup(storageServiceId, storageGroupId)
	if sg == nil {
		return ec.NewErrNotFound().WithCause(fmt.Sprintf("Storage group '%s' not found", storageGroupId))
	}

	if sg.fileShare != nil {
		return ec.NewErrNotAcceptable().WithCause(fmt.Sprintf("Storage group '%s' file share present", storageGroupId))
	}

	sp := sg.storagePool

	// Detach the endpoint from the NVMe namespaces

	for _, volume := range sp.providingVolumes {
		if err := nvme.DetachControllers(volume.volume, []uint16{sg.endpoint.controllerId}); err != nil {
			log.WithError(err).Errorf("Failed to detach controllers")
			return ec.NewErrInternalServerError().WithError(err).WithCause(fmt.Sprintf("Storage group '%s' Failed to detach controller '%d'", storageGroupId, sg.endpoint.controllerId))
		}
	}

	// Notify the Server the namespaces were removed

	if err := sg.serverStorage.Delete(); err != nil {
		return ec.NewErrInternalServerError().WithError(err).WithCause(fmt.Sprintf("Storage group '%s' server delete failed", storageGroupId))
	}

	s.deleteStorageGroup(sg)

	return nil
}

// StorageServiceIdEndpointsGet -
func (*StorageService) StorageServiceIdEndpointsGet(storageServiceId string, model *sf.EndpointCollectionEndpointCollection) error {
	s := findStorageService(storageServiceId)
	if s == nil {
		return ec.NewErrNotFound().WithCause(fmt.Sprintf("Storage service '%s' not found", storageServiceId))
	}

	model.MembersodataCount = int64(len(s.endpoints))
	model.Members = make([]sf.OdataV4IdRef, model.MembersodataCount)
	for idx, ep := range s.endpoints {
		model.Members[idx] = sf.OdataV4IdRef{OdataId: ep.OdataId()}
	}

	return nil
}

// StorageServiceIdEndpointIdGet -
func (*StorageService) StorageServiceIdEndpointIdGet(storageServiceId, endpointId string, model *sf.EndpointV150Endpoint) error {
	_, ep := findEndpoint(storageServiceId, endpointId)
	if ep == nil {
		return ec.NewErrNotFound().WithCause(fmt.Sprintf("Endpoint '%s' not found", endpointId))
	}

	model.Id = ep.id
	model.OdataId = ep.OdataId()

	// Ask the fabric manager to fill it the endpoint details
	if err := fabric.FabricIdEndpointsEndpointIdGet(ep.fabricId, ep.id, model); err != nil {
		return ec.NewErrInternalServerError().WithError(err).WithCause(fmt.Sprintf("Fabric endpoint '%s' not found", ep.id))
	}

	model.OdataId = ep.OdataId() // Done twice so the fabric manager doesn't hijak the @odata.id

	// Since the Fabric Manager is only aware of PCI-e Connectivity, if we have a different
	// view of the endpoint from our ability to controller the server, record the state here.
	if model.Status.State == sf.ENABLED_RST {
		if ep.serverCtrl.Connected() {
			ep.state = sf.ENABLED_RST
		} else {
			ep.state = sf.UNAVAILABLE_OFFLINE_RST
		}

		model.Status.State = ep.state
	}

	return nil
}

// StorageServiceIdFileSystemsGet -
func (*StorageService) StorageServiceIdFileSystemsGet(storageServiceId string, model *sf.FileSystemCollectionFileSystemCollection) error {
	s := findStorageService(storageServiceId)
	if s == nil {
		return ec.NewErrNotFound().WithCause(fmt.Sprintf("Storage service '%s' not found", storageServiceId))
	}

	model.MembersodataCount = int64(len(s.fileSystems))
	model.Members = make([]sf.OdataV4IdRef, model.MembersodataCount)
	for idx, fileSystem := range s.fileSystems {
		model.Members[idx] = sf.OdataV4IdRef{OdataId: fileSystem.OdataId()}
	}

	return nil
}

// StorageServiceIdFileSystemsPost -
func (*StorageService) StorageServiceIdFileSystemsPost(storageServiceId string, model *sf.FileSystemV122FileSystem) error {
	s := findStorageService(storageServiceId)
	if s == nil {
		return ec.NewErrNotFound().WithCause(fmt.Sprintf("Storage service '%s' not found", storageServiceId))
	}

	// Extract the StoragePoolId from the POST model
	fields := strings.Split(model.Links.StoragePool.OdataId, "/")
	if len(fields) != s.resourceIndex+1 {
		return ec.NewErrNotAcceptable().WithCause(fmt.Sprintf("Storage pool @odata.id '%s' malformed", model.Links.StoragePool.OdataId))
	}
	storagePoolId := fields[s.resourceIndex]

	// Find the existing storage pool - the file system will link to the providing pool
	sp := s.findStoragePool(storagePoolId)
	if sp == nil {
		return ec.NewErrNotAcceptable().WithCause(fmt.Sprintf("Storage pool '%s' not found", storagePoolId))
	}

	if sp.fileSystem != nil {
		return ec.NewErrNotAcceptable().WithCause(fmt.Sprintf("Storage pool '%s' no file system defined", storagePoolId))
	}

	oem := server.FileSystemOem{}
	if err := openapi.UnmarshalOem(model.Oem, &oem); err != nil {
		return ec.NewErrBadRequest().WithError(err).WithCause("Failed to unmarshal file system OEM fields")
	}

	fsApi := server.FileSystemController.NewFileSystem(oem)
	if fsApi == nil {
		return ec.NewErrNotAcceptable().WithCause(fmt.Sprintf("File system '%s' not found", oem.Type))
	}

	fs := s.createFileSystem(sp, fsApi)
	sp.fileSystem = fs
	s.fileSystems = append(s.fileSystems, *fs)

	return s.StorageServiceIdFileSystemIdGet(storageServiceId, fs.id, model)
}

// StorageServiceIdFileSystemIdGet -
func (*StorageService) StorageServiceIdFileSystemIdGet(storageServiceId, fileSystemId string, model *sf.FileSystemV122FileSystem) error {
	_, fs := findFileSystem(storageServiceId, fileSystemId)
	if fs == nil {
		return ec.NewErrNotFound().WithCause(fmt.Sprintf("File system '%s' not found", fileSystemId))
	}

	model.Id = fs.id
	model.OdataId = fs.OdataId()

	model.CapacityBytes = fs.storagePool.allocatedVolume.capacityBytes
	model.StoragePool = sf.OdataV4IdRef{OdataId: fs.storagePool.OdataId()}
	model.ExportedShares = fs.OdataIdRef("/ExportedFileShares")

	return nil
}

// StorageServiceIdFileSystemIdDelete -
func (*StorageService) StorageServiceIdFileSystemIdDelete(storageServiceId, fileSystemId string) error {
	s, fs := findFileSystem(storageServiceId, fileSystemId)
	if fs == nil {
		return ec.NewErrNotFound().WithCause(fmt.Sprintf("File system '%s' not found", fileSystemId))
	}

	for _, sh := range fs.shares {
		if err := s.StorageServiceIdFileSystemIdExportedShareIdDelete(s.id, fs.id, sh.id); err != nil {
			return ec.NewErrInternalServerError().WithError(err).WithCause(fmt.Sprintf("Exported share '%s' failed delete", sh.id))
		}
	}

	if err := fs.fsApi.Delete(); err != nil {
		return ec.NewErrInternalServerError().WithError(err).WithCause(fmt.Sprintf("File system '%s' failed delete", fileSystemId))
	}

	for fileSystemIdx, fileSystem := range s.fileSystems {
		if fileSystem.id == fileSystemId {
			s.fileSystems = append(s.fileSystems[:fileSystemIdx], s.fileSystems[fileSystemIdx+1:]...)
			break
		}
	}

	return nil
}

// StorageServiceIdFileSystemIdExportedSharesGet -
func (*StorageService) StorageServiceIdFileSystemIdExportedSharesGet(storageServiceId, fileSystemId string, model *sf.FileShareCollectionFileShareCollection) error {
	_, fs := findFileSystem(storageServiceId, fileSystemId)
	if fs == nil {
		return ec.NewErrNotFound().WithCause(fmt.Sprintf("File system '%s' not found", fileSystemId))
	}

	model.MembersodataCount = int64(len(fs.shares))
	model.Members = make([]sf.OdataV4IdRef, model.MembersodataCount)
	for idx, sh := range fs.shares {
		model.Members[idx] = sf.OdataV4IdRef{OdataId: sh.OdataId()}
	}
	return nil
}

// StorageServiceIdFileSystemIdExportedSharesPost -
func (*StorageService) StorageServiceIdFileSystemIdExportedSharesPost(storageServiceId, fileSystemId string, model *sf.FileShareV120FileShare) error {
	s, fs := findFileSystem(storageServiceId, fileSystemId)
	if fs == nil {
		return ec.NewErrNotFound().WithCause(fmt.Sprintf("File system '%s' not found", fileSystemId))
	}

	fields := strings.Split(model.Links.Endpoint.OdataId, "/")
	if len(fields) != s.resourceIndex+1 {
		return ec.NewErrNotAcceptable().WithCause(fmt.Sprintf("Storage pool @odata.id '%s' malformed", model.Links.Endpoint.OdataId))
	}

	endpointId := fields[s.resourceIndex]
	ep := s.findEndpoint(endpointId)
	if ep == nil {
		return ec.NewErrNotAcceptable().WithCause(fmt.Sprintf("Endpoint '%s' not found", endpointId))
	}

	if len(model.FileSharePath) == 0 {
		return ec.NewErrNotAcceptable().WithCause("File share path not defined")
	}

	// Find the Storage Group Endpoint - There should be a Storage Group
	// Endpoint that has an association to the fs.storagePool and endpoint.
	// This represents the physical devices on the server that backs the
	// File System and supports the Exported Share.
	sg := fs.storagePool.findStorageGroupByEndpoint(ep)
	if sg == nil {
		return ec.NewErrNotAcceptable().WithCause(fmt.Sprintf("Storage group not found"))
	}

refreshState:
	switch sg.status().State {
	case sf.ENABLED_RST:
		break
	case sf.STARTING_RST:
		log.Infof("Storage group starting, delay 1s")
		time.Sleep(time.Second)
		goto refreshState
	default:
		return ec.NewErrNotAcceptable()
	}

	sh := fs.createFileShare(sg, model.FileSharePath)
	sg.fileShare = sh
	fs.shares = append(fs.shares, *sh)

	if err := sh.initialize(model.FileSharePath); err != nil {
		log.WithError(err).Errorf("Failed to initialize file share for path %s", model.FileSharePath)
		// No error is returned, status is reflected in model
	}

	model.Id = sh.id
	model.OdataId = sh.OdataId()

	return s.StorageServiceIdFileSystemIdExportedShareIdGet(storageServiceId, fileSystemId, sh.id, model)
}

// StorageServiceIdFileSystemIdExportedShareIdGet -
func (*StorageService) StorageServiceIdFileSystemIdExportedShareIdGet(storageServiceId, fileSystemId, exportedShareId string, model *sf.FileShareV120FileShare) error {
	_, fs, sh := findExportedFileShare(storageServiceId, fileSystemId, exportedShareId)
	if sh == nil {
		return ec.NewErrNotFound().WithCause(fmt.Sprintf("File share '%s' not found", exportedShareId))
	}

	model.Id = sh.id
	model.OdataId = sh.OdataId()
	model.FileSharePath = sh.mountRoot
	model.Links.FileSystem = sf.OdataV4IdRef{OdataId: fs.OdataId()}
	model.Links.Endpoint = sf.OdataV4IdRef{OdataId: sh.storageGroup.endpoint.OdataId()}

	model.Status = *sh.getStatus() // TODO

	return nil
}

// StorageServiceIdFileSystemIdExportedShareIdDelete -
func (*StorageService) StorageServiceIdFileSystemIdExportedShareIdDelete(storageServiceId, fileSystemId, exportedShareId string) error {
	_, fs, sh := findExportedFileShare(storageServiceId, fileSystemId, exportedShareId)
	if sh == nil {
		return ec.NewErrNotFound().WithCause(fmt.Sprintf("File share '%s' not found", exportedShareId))
	}

	if err := sh.storageGroup.serverStorage.Delete(); err != nil {
		return ec.NewErrInternalServerError().WithError(err).WithCause(fmt.Sprintf("File share '%s' failed delete", exportedShareId))
	}

	for shareIdx, share := range fs.shares {
		if share.id == exportedShareId {
			fs.shares = append(fs.shares[:shareIdx], fs.shares[shareIdx+1:]...)
			break
		}
	}

	return nil
}
