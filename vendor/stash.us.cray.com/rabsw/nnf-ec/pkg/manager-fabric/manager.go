package fabric

import (
	"fmt"
	"os"
	"strconv"

	log "github.com/sirupsen/logrus"

	"stash.us.cray.com/rabsw/nnf-ec/pkg/api"
	ec "stash.us.cray.com/rabsw/nnf-ec/pkg/ec"
	event "stash.us.cray.com/rabsw/nnf-ec/pkg/manager-event"
	msgreg "stash.us.cray.com/rabsw/nnf-ec/pkg/manager-message-registry/registries"

	"stash.us.cray.com/rabsw/nnf-ec/internal/switchtec/pkg/switchtec"

	openapi "stash.us.cray.com/rabsw/nnf-ec/pkg/rfsf/pkg/common"
	sf "stash.us.cray.com/rabsw/nnf-ec/pkg/rfsf/pkg/models"
)

const (
	FabricId = "Rabbit"
)

type Fabric struct {
	ctrl SwitchtecControllerInterface

	id     string
	config *ConfigFile

	switches       []Switch
	endpoints      []Endpoint
	endpointGroups []EndpointGroup
	connections    []Connection

	managementEndpointCount int
	upstreamEndpointCount   int
	downstreamEndpointCount int
}

type Switch struct {
	id  string
	idx int

	paxId int32
	path  string
	dev   SwitchtecDeviceInterface

	config *SwitchConfig
	ports  []Port

	fabric *Fabric

	// Information is cached on switch initialization
	model           string
	manufacturer    string
	serialNumber    string
	firmwareVersion string

	DEBUG_NEXT_SWITCH_LOG_PORT_ID int
}

type Port struct {
	id       string
	fabricId string

	portType sf.PortV130PortType
	portStatus

	swtch  *Switch
	config *PortConfig

	endpoints []*Endpoint
}

type portStatus struct {
	cfgLinkWidth uint8
	negLinkWidth uint8

	curLinkRateGBps float64
	maxLinkRateGBps float64

	linkStatus sf.PortV130LinkStatus
	linkState  sf.PortV130LinkState
}

type Endpoint struct {
	id    string
	index int
	name  string

	endpointType sf.EndpointV150EntityType

	// For Initiator Endpoints, this represents the USP index within the Fabric starting at zero
	// For Target Endpoints, this represents the Physical Controller (if zero) and the
	// Secondary Controller (VF) Id (if non-zero)
	controllerId uint16

	// For Rabbit Endpoint, Ports will be two and represents the Rabbit position viewed from both Switches.
	// For all other Endpoints, Ports will be a singular entry representing the Port of the parent Switch.
	ports []*Port

	fabric *Fabric

	// OEM fields -  marshalled?
	pdfid         uint16
	bound         bool
	boundPaxId    uint8
	boundHvdPhyId uint8
	boundHvdLogId uint8
}

// Endpoint Group is represents a collection of endpoints and the related Connection. Only one Endpoint Intitator
// may belong to a group, with the remaining endpoints expected to be Drive types. An Endpoint Group is equivalent
// to a Host-Virtualization Domain defiend on the Switchtec device, with the exception of the Endpoint Group
// containing the Processor (i.e Rabbit) - for this EPG contains two, smaller HVDs that point to the switch-local
// DSPs.
type EndpointGroup struct {
	id         string
	endpoints  []*Endpoint
	connection *Connection

	initiator **Endpoint

	fabric *Fabric
}

type Connection struct {
	endpointGroup *EndpointGroup
	// volumes       []VolumeInfo

	fabric *Fabric
}

// type VolumeInfo struct {
// 	odataid string
// }

var manager Fabric

func init() {
	api.RegisterFabricController(&manager)
}

func isFabric(id string) bool { return id == manager.id }

// TODO: Move these to the newer find functions
//func isEndpoint(id string) bool      { _, err := fabric.findEndpoint(id); return err == nil }
//func isEndpointGroup(id string) bool { _, err := fabric.findEndpointGroup(id); return err == nil }

func findFabric(fabricId string) *Fabric {
	if !isFabric(fabricId) {
		return nil
	}

	return &manager
}

func findSwitch(fabricId, switchId string) (*Fabric, *Switch) {
	f := findFabric(fabricId)
	if f == nil {
		return nil, nil
	}

	return f, f.findSwitch(switchId)
}

func (f *Fabric) findSwitch(switchId string) *Switch {
	for idx, s := range f.switches {
		if s.id == switchId {
			return &f.switches[idx]
		}
	}

	return nil
}

func findPort(fabricId, switchId, portId string) (*Fabric, *Switch, *Port) {
	f, s := findSwitch(fabricId, switchId)
	if s == nil {
		return nil, nil, nil
	}

	return f, s, s.findPort(portId)
}

func (s *Switch) findPort(portId string) *Port {
	for idx, p := range s.ports {
		if p.id == portId {
			return &s.ports[idx]
		}
	}

	return nil
}

func findEndpoint(fabricId, endpointId string) (*Fabric, *Endpoint) {
	f := findFabric(fabricId)
	if f == nil {
		return nil, nil
	}

	return f, f.findEndpoint(endpointId)
}

func (f *Fabric) findEndpoint(endpointId string) *Endpoint {
	for idx, ep := range f.endpoints {
		if ep.id == endpointId {
			return &f.endpoints[idx]
		}
	}

	return nil
}

func findEndpointGroup(fabricId, endpointGroupId string) (*Fabric, *EndpointGroup) {
	f := findFabric(fabricId)
	if f == nil {
		return nil, nil
	}

	return f, f.findEndpointGroup(endpointGroupId)
}

func (f *Fabric) findEndpointGroup(endpointGroupId string) *EndpointGroup {
	for idx, epg := range f.endpointGroups {
		if epg.id == endpointGroupId {
			return &f.endpointGroups[idx]
		}
	}

	return nil
}

func findConnection(fabricId, connectionId string) (*Fabric, *Connection) {
	f := findFabric(fabricId)
	if f == nil {
		return nil, nil
	}

	return f, f.findConnection(connectionId)
}

func (f *Fabric) findConnection(connectionId string) *Connection {
	for idx, c := range f.connections {
		if c.endpointGroup.id == connectionId {
			return &f.connections[idx]
		}
	}

	return nil
}

// findPortByType - Finds the i'th port of portType in the fabric
func (f *Fabric) findPortByType(portType sf.PortV130PortType, idx int) *Port {
	switch portType {
	case sf.MANAGEMENT_PORT_PV130PT:
		return f.switches[idx].findPortByType(portType, 0)
	case sf.UPSTREAM_PORT_PV130PT:
		for _, s := range f.switches {
			if idx < s.config.UpstreamPortCount {
				return s.findPortByType(portType, idx)
			}
			idx = idx - s.config.UpstreamPortCount
		}
	case sf.DOWNSTREAM_PORT_PV130PT:
		for _, s := range f.switches {
			if idx < s.config.DownstreamPortCount {
				return s.findPortByType(portType, idx)
			}
			idx = idx - s.config.DownstreamPortCount
		}
	}

	return nil
}

func (f *Fabric) isManagementEndpoint(endpointIndex int) bool {
	return endpointIndex == 0
}

func (f *Fabric) isUpstreamEndpoint(idx int) bool {
	return !f.isManagementEndpoint(idx) && idx-f.managementEndpointCount < f.upstreamEndpointCount
}

func (f *Fabric) isDownstreamEndpoint(idx int) bool {
	return idx >= (f.managementEndpointCount + f.upstreamEndpointCount)
}

func (f *Fabric) getUpstreamEndpointRelativePortIndex(idx int) int {
	return idx - f.managementEndpointCount
}

func (f *Fabric) getDownstreamEndpointRelativePortIndex(idx int) int {
	return (idx - (f.managementEndpointCount + f.upstreamEndpointCount)) / (1 /*PF*/ + f.managementEndpointCount + f.upstreamEndpointCount)
}

// func (f *Fabric) getDownstreamEndpointIndex(deviceIdx int, functionIdx int) int {
// 	return (deviceIdx * (1 /*PF*/ + f.managementEndpointCount + f.upstreamEndpointCount)) + functionIdx
// }

func (s *Switch) isReady() bool { return len(s.path) != 0 }
func (s *Switch) isDown() bool  { return !s.isReady() }
func (s *Switch) setDown()      { s.path = "" }

func (s *Switch) identify() error {
	f := s.fabric
	for i := 0; i < len(f.switches); i++ {

		path := fmt.Sprintf("/dev/switchtec%d", i)

		log.Debugf("Identify Switch %s: Opening %s", s.id, path)
		dev, err := f.ctrl.Open(path)
		if os.IsNotExist(err) {
			log.WithError(err).Debugf("path %s", path)
			continue
		} else if err != nil {
			log.WithError(err).Warnf("Identify Switch %s: Open Error", s.id)
			return err
		}

		paxId, err := dev.Identify()
		if err != nil {
			log.WithError(err).Warnf("Identify Switch %s: Identify Error", s.id)
			return err
		}

		log.Infof("Identify Switch %s: Device ID: %d", s.id, paxId)
		if id := strconv.Itoa(int(paxId)); id == s.id {
			s.dev = dev
			s.path = path
			s.paxId = paxId

			log.Infof("Identify Switch %s: Loading Mfg Info", s.id)

			s.model = s.getModel()
			s.manufacturer = s.getManufacturer()
			s.serialNumber = s.getSerialNumber()
			s.firmwareVersion = s.getFirmwareVersion()

			return nil
		}
	}

	// At this point we've failed to locate our switch with the desired PAX ID. Check
	// if we're using management routing, which enables forwarding of commands to
	// the secondary PAX through the primary PAX
	if f.config.IsManagementRoutingEnabled() {
		log.Warningf("Switch %d is acting as primary device for switch %d", f.config.ManagementConfig.PrimaryDevice, f.config.ManagementConfig.SecondaryDevice)
		for _, sw := range f.switches {
			if sw.dev.Device().ID() == f.config.ManagementConfig.PrimaryDevice {

				// Open another path to the primary device
				s.dev, _ = f.ctrl.Open(sw.path)

				// Configure this device to access the secondary pax device
				s.dev.Device().SetID(f.config.ManagementConfig.SecondaryDevice)

				s.path = sw.path
				s.paxId = s.dev.Device().ID()

				s.model = sw.model
				s.manufacturer = sw.manufacturer
				s.serialNumber = sw.serialNumber
				s.firmwareVersion = sw.firmwareVersion

				return nil
			}
		}
	}

	return fmt.Errorf("Identify Switch %s: Could Not ID Switch", s.id) // TODO: Switch not found
}

func (s *Switch) refreshPortStatus() error {

	switchPortStatus, err := s.dev.GetPortStatus()
	if err != nil {
		return err
	}

StatusLoop:
	for _, st := range switchPortStatus {
		for portIdx := range s.ports {
			p := &s.ports[portIdx]

			if st.PhysPortId == uint8(p.config.Port) {

				getStatusFromState := func(state switchtec.PortLinkState) sf.PortV130LinkStatus {
					switch state {
					case switchtec.PortLinkState_Disable:
						return sf.NO_LINK_PV130LS
					case switchtec.PortLinkState_L0:
						return sf.LINK_UP_PV130LS
					case switchtec.PortLinkState_Detect:
						return sf.LINK_DOWN_PV130LS
					case switchtec.PortLinkState_Polling:
						return sf.STARTING_PV130LS
					case switchtec.PortLinkState_Config:
						return sf.TRAINING_PV130LS
					default:
						return sf.PortV130LinkStatus("Unknown")
					}
				}

				status := portStatus{
					cfgLinkWidth:    st.CfgLinkWidth,
					negLinkWidth:    st.NegLinkWidth,
					curLinkRateGBps: st.CurLinkRateGBps,
					maxLinkRateGBps: switchtec.GetDataRateGBps(uint8(s.config.pciGen)) * float64(p.config.Width),
					linkStatus:      getStatusFromState(st.LinkState),
					linkState:       sf.ENABLED_PV130LST,
				}

				if p.portStatus != status {

					eventMap := make(map[sf.PortV130PortType]func(string, string) event.Event)

					if status.linkStatus == sf.LINK_UP_PV130LS {
						if status.negLinkWidth == status.cfgLinkWidth && status.curLinkRateGBps == status.maxLinkRateGBps {
							eventMap[sf.UPSTREAM_PORT_PV130PT] = msgreg.UpstreamLinkEstablishedFabric
							eventMap[sf.MANAGEMENT_PORT_PV130PT] = msgreg.UpstreamLinkEstablishedFabric
							eventMap[sf.DOWNSTREAM_PORT_PV130PT] = msgreg.DownstreamLinkEstablishedFabric
							eventMap[sf.INTERSWITCH_PORT_PV130PT] = msgreg.InterswitchLinkEstablishedFabric
						} else {
							eventMap[sf.UPSTREAM_PORT_PV130PT] = msgreg.DegradedUpstreamLinkEstablishedFabric
							eventMap[sf.MANAGEMENT_PORT_PV130PT] = msgreg.DegradedUpstreamLinkEstablishedFabric
							eventMap[sf.DOWNSTREAM_PORT_PV130PT] = msgreg.DegradedDownstreamLinkEstablishedFabric
							eventMap[sf.INTERSWITCH_PORT_PV130PT] = msgreg.DegradedInterswitchLinkEstablishedFabric
						}
					} else {
						eventMap[sf.UPSTREAM_PORT_PV130PT] = msgreg.UpstreamLinkDroppedFabric
						eventMap[sf.MANAGEMENT_PORT_PV130PT] = msgreg.UpstreamLinkDroppedFabric
						eventMap[sf.DOWNSTREAM_PORT_PV130PT] = msgreg.DownstreamLinkDroppedFabric
						eventMap[sf.INTERSWITCH_PORT_PV130PT] = msgreg.InterswitchLinkDroppedFabric
					}

					p.portStatus = status
					if eventFunc, ok := eventMap[p.portType]; ok {
						event.EventManager.Publish(eventFunc(s.id, p.id))
					}
				}

				continue StatusLoop
			}
		}
	}

	return nil
}

func (s *Switch) getStatus() (stat sf.ResourceStatus) {

	if s.dev == nil {
		stat.State = sf.UNAVAILABLE_OFFLINE_RST
	} else {
		stat.Health = sf.OK_RH
		stat.State = sf.ENABLED_RST
	}

	return stat
}

func (s *Switch) getDeviceStringByFunc(f func(dev SwitchtecDeviceInterface) (string, error)) string {
	if s.dev != nil {
		ret, err := f(s.dev)
		if err != nil {
			log.WithError(err).Warnf("Failed to retrieve device string")
		}

		return ret
	}

	return ""
}

func (s *Switch) getModel() string {
	return s.getDeviceStringByFunc(func(dev SwitchtecDeviceInterface) (string, error) {
		return dev.GetModel()
	})
}

func (s *Switch) getManufacturer() string {
	return s.getDeviceStringByFunc(func(dev SwitchtecDeviceInterface) (string, error) {
		return dev.GetManufacturer()
	})
}

func (s *Switch) getSerialNumber() string {
	return s.getDeviceStringByFunc(func(dev SwitchtecDeviceInterface) (string, error) {
		return dev.GetSerialNumber()
	})
}

func (s *Switch) getFirmwareVersion() string {
	return s.getDeviceStringByFunc(func(dev SwitchtecDeviceInterface) (string, error) {
		return dev.GetFirmwareVersion()
	})
}

// findPort - Finds the i'th port of portType in the switch
func (s *Switch) findPortByType(portType sf.PortV130PortType, idx int) *Port {
	for portIdx, port := range s.ports {
		if port.portType == portType {
			if idx == 0 {
				return &s.ports[portIdx]
			}
			idx--
		}
	}

	panic(fmt.Sprintf("Switch Port %d Not Found", idx))
}

func (s *Switch) findPortByPhysicalPortId(id uint8) *Port {
	for portIdx, port := range s.ports {
		if port.config.Port == int(id) {
			return &s.ports[portIdx]
		}
	}

	return nil
}

func (p *Port) GetBaseEndpointIndex() int { return p.endpoints[0].index }

func (p *Port) findEndpoint(functionId string) *Endpoint {
	id, err := strconv.Atoi(functionId)
	if err != nil {
		return nil
	}
	if !(id < len(p.endpoints)) {
		return nil
	}
	return p.endpoints[id]
}

func (p *Port) Initialize() error {
	log.Infof("Initialize Port %s: Name: %s Physical Port: %d", p.id, p.config.Name, p.config.Port)

	switch p.portType {
	case sf.DOWNSTREAM_PORT_PV130PT:

		processPort := func(port *Port) func(*switchtec.DumpEpPortDevice) error {
			return func(epPort *switchtec.DumpEpPortDevice) error {

				if switchtec.EpPortType(epPort.Hdr.Typ) == switchtec.NoneEpPortType {
					return fmt.Errorf("Port %s: Device not present", port.id)
				}

				if switchtec.EpPortType(epPort.Hdr.Typ) != switchtec.DeviceEpPortType {
					return fmt.Errorf("Port %s: Non-device type (%#02x)", port.id, epPort.Hdr.Typ)
				}

				if epPort.Ep.Functions == nil {
					panic(fmt.Sprintf("No EP Functions received for port %+v", port))
				}

				f := port.swtch.fabric
				if len(epPort.Ep.Functions) < 1 /*PF*/ +f.managementEndpointCount+f.upstreamEndpointCount {
					log.Warnf("Port %s: Insufficient function count %d", port.id, len(epPort.Ep.Functions))
				}

				for idx, f := range epPort.Ep.Functions {

					if len(p.endpoints) <= idx {
						break
					}

					ep := p.endpoints[idx]
					ep.controllerId = uint16(f.VFNum)

					ep.pdfid = f.PDFID
					ep.bound = f.Bound != 0
					ep.boundPaxId = f.BoundPAXID
					ep.boundHvdPhyId = f.BoundHVDPhyPID
					ep.boundHvdLogId = f.BoundHVDLogPID
				}

				return nil
			}
		}

		log.Infof("Initialize Port: Switch %s enumerting DSP %d", p.swtch.id, p.config.Port)
		if err := p.swtch.dev.EnumerateEndpoint(uint8(p.config.Port), processPort(p)); err != nil {
			log.WithError(err).Warnf("Initialize Port: Port Enumeration Failed: Physical Port %d", p.config.Port)
			return err
		}
	}

	return nil
}

func (p *Port) bind() error {
	f := p.swtch.fabric

	if p.portStatus.linkStatus != sf.LINK_UP_PV130LS {
		log.Warnf("Port %+v: Port not up, skipping bind operation", p)
		return nil
	}

	log.Debugf("Port %s: Bind Operation Starting: Switch: %s, PAX: %d Physical Port: %d Type: %s", p.id, p.swtch.id, p.swtch.paxId, p.config.Port, p.portType)

	if p.portType != sf.DOWNSTREAM_PORT_PV130PT {
		panic(fmt.Sprintf("Port %s: Bind operation not allowed for port type %s", p.id, p.portType))
	}

	if len(p.endpoints) < 1 /*PF*/ +f.managementEndpointCount+f.upstreamEndpointCount {
		panic(fmt.Sprintf("Port %s: Insufficient endpoints defined for DSP", p.id))
	}

	if (p.endpoints[0].pdfid & 0x00FF) != 0 {
		panic(fmt.Sprintf("Port %s: Endpoint index zero expected to be physical function: %#04x", p.id, p.endpoints[0].pdfid))
	}

	for _, ep := range p.endpoints[1:] { // Skip PF

		for _, epg := range f.endpointGroups {
			initiator := *(epg.initiator)

			if initiator != epg.endpoints[0] {
				panic(fmt.Sprintf("Initiator endpoint %s must be at index 0 of an endpoint group to support connection algorithm", initiator.id))
			}

			if (initiator.endpointType != sf.PROCESSOR_EV150ET) && (initiator.endpointType != sf.STORAGE_INITIATOR_EV150ET) {
				panic(fmt.Sprintf("Initiator endpoint %s must be of type %s or %s and not %s", initiator.id, sf.PROCESSOR_EV150ET, sf.STORAGE_INITIATOR_EV150ET, initiator.endpointType))
			}

			if initiator.endpointType == sf.PROCESSOR_EV150ET {
				if len(initiator.ports) != 2 {
					panic("Processor endpoint expected to have two ports")
				}

				if initiator.ports[0].swtch.id == initiator.ports[1].swtch.id {
					panic("Processor endpoint ports should be on two different switches")
				}
			}

			for endpointIdx, endpoint := range epg.endpoints[1:] { // Skip initiator

				if ep == endpoint {
					// The Logical Port ID is the index into the HVD for the bind to occur.
					// That is simply the DSP index within the endpoint group (recall that
					// DSPs start at offset 1 in the EPG, with offset 0 being the initiator)
					logicalPortId := endpointIdx

					// For normal USPs of type Storage Initiator, the port is bound to the initiator
					// through the fabric. For the endpoint representing the Processor USP the port is bound
					// to the local switch and never through the fabric.
					for _, initiatorPort := range initiator.ports {
						s := initiatorPort.swtch

						if initiator.endpointType == sf.PROCESSOR_EV150ET {
							if !f.config.IsManagementRoutingEnabled() {
								if p.swtch.id != s.id {
									logicalPortId -= s.config.DownstreamPortCount
									continue
								}
							}
						}

						if initiatorPort.config.Port > int(^uint8(0)) {
							panic(fmt.Sprintf("Initiator port %d to large for bind operation", initiatorPort.config.Port))
						}

						if logicalPortId > int(^uint8(0)) {
							panic(fmt.Sprintf("Logical port ID %d to large for bind operation", logicalPortId))
						}

						if endpoint.pdfid == 0 {
							log.Errorf("Endpoint has no PDFID, skipping bind to initiator port %d (%s)", initiatorPort.config.Port, initiatorPort.config.Name)
							continue
						}

						if endpoint.bound {
							logFunc := log.Debugf
							if endpoint.boundPaxId != uint8(s.paxId) ||
								endpoint.boundHvdPhyId != uint8(initiatorPort.config.Port) ||
								endpoint.boundHvdLogId != uint8(logicalPortId) {
								logFunc = log.Errorf
							}

							logFunc("Already Bound: PAX: %d, Physical Port: %d, Logical Port: %d, PDFID: %#04x", endpoint.boundPaxId, endpoint.boundHvdPhyId, endpoint.boundHvdLogId, endpoint.pdfid)
							break
						}

						log.Infof("Bind: Switch: %s, PAX: %d, Physical Port: %d, Logical Port: %d, PDFID %#04x", s.id, s.paxId, initiatorPort.config.Port, logicalPortId, endpoint.pdfid)
						if err := s.dev.Bind(uint8(initiatorPort.config.Port), uint8(logicalPortId), endpoint.pdfid); err != nil {
							log.WithError(err).Errorf("Bind Failed: Switch %s: PAX: %d Port: %d, Logical Port: %d, PDFID: %#04x", s.id, s.paxId, initiatorPort.config.Port, logicalPortId, endpoint.pdfid)
						}

						break
					} // for range initiator.ports

					break
				} // if ep == endpoint
			}
		}
	}

	return nil
}

func (p *Port) notify(isDown bool) {

	if isDown {

		// Check if the port is already down before generating the port event. This can occur when
		// we refresh the port status because of another port event, and that causes this port
		// to be recorded as down.
		if p.portStatus.linkStatus != sf.LINK_DOWN_PV130LS {
			p.portStatus.linkStatus = sf.LINK_DOWN_PV130LS

			switch p.portType {
			case sf.DOWNSTREAM_PORT_PV130PT:
				event.EventManager.Publish(msgreg.DownstreamLinkDroppedFabric(p.swtch.id, p.id))
			case sf.UPSTREAM_PORT_PV130PT, sf.MANAGEMENT_PORT_PV130PT:
				event.EventManager.Publish(msgreg.UpstreamLinkDroppedFabric(p.swtch.id, p.id))
			case sf.INTERSWITCH_PORT_PV130PT:
				event.EventManager.Publish(msgreg.InterswitchLinkDroppedFabric(p.swtch.id, p.id))
			}
		}

	} else {

		// A port up event causes a full refresh of all ports on the switch to ensure
		// the most recent status is used. Since we don't have the ability to query
		// the status of a single port, we will read all ports.
		p.swtch.refreshPortStatus()
	}
}

// Getters for common endpoint calls
func (e *Endpoint) Id() string                      { return e.id }
func (e *Endpoint) Type() sf.EndpointV150EntityType { return e.endpointType }
func (e *Endpoint) Name() string                    { return e.name }
func (e *Endpoint) Index() int                      { return e.index }
func (e *Endpoint) ControllerId() uint16            { return e.controllerId }

func (e *Endpoint) OdataId() string {
	return fmt.Sprintf("/redfish/v1/Fabrics/%s/Endpoints/%s", e.fabric.id, e.id)
}

func (f *Fabric) fmt(format string, a ...interface{}) string {
	return fmt.Sprintf("/redfish/v1/Fabrics/%s", f.id) + fmt.Sprintf(format, a...)
}

func (s *Switch) fmt(format string, a ...interface{}) string {
	return s.fabric.fmt("/Switches/%s", s.id) + fmt.Sprintf(format, a...)
}

func (ep *Endpoint) fmt(format string, a ...interface{}) string {
	return ep.fabric.fmt("/Endpoints/%s", ep.id) + fmt.Sprintf(format, a...)
}

func (epg *EndpointGroup) fmt(format string, a ...interface{}) string {
	return epg.fabric.fmt("/EndpointGroups/%s", epg.id) + fmt.Sprintf(format, a...)
}

func (c *Connection) fmt(format string, a ...interface{}) string {
	return c.fabric.fmt("/Connections/%s", c.endpointGroup.id) + fmt.Sprintf(format, a...)
}

// Initialize
func Initialize(ctrl SwitchtecControllerInterface) error {

	manager = Fabric{
		id:   FabricId,
		ctrl: ctrl,
	}
	m := &manager

	log.SetLevel(log.DebugLevel)
	log.Infof("Fabric Manager %s Initializing", m.id)

	conf, err := loadConfig()
	if err != nil {
		log.WithError(err).Errorf("Fabric Manager %s failed to load configuration", m.id)
		return err
	}
	m.config = conf

	log.Debugf("Fabric Configuration '%s' Loaded...", conf.Metadata.Name)
	log.Debugf("  Debug Level: %s", conf.DebugLevel)
	log.Debugf("  Management Ports: %d", conf.ManagementPortCount)
	log.Debugf("  Upstream Ports:   %d", conf.UpstreamPortCount)
	log.Debugf("  Downstream Ports: %d", conf.DownstreamPortCount)
	for _, switchConf := range conf.Switches {
		log.Debugf("  Switch %s Configuration: %s", switchConf.Id, switchConf.Metadata.Name)
		log.Debugf("    Management Ports: %d", switchConf.ManagementPortCount)
		log.Debugf("    Upstream Ports:   %d", switchConf.UpstreamPortCount)
		log.Debugf("    Downstream Ports: %d", switchConf.DownstreamPortCount)
	}

	level, err := log.ParseLevel(conf.DebugLevel)
	if err != nil {
		log.WithError(err).Errorf("Failed to parse debug level: %s", conf.DebugLevel)
		return err
	}

	log.SetLevel(level)

	m.switches = make([]Switch, len(conf.Switches))
	var fabricPortId = 0
	for switchIdx := range conf.Switches {
		switchConf := &conf.Switches[switchIdx]
		log.Infof("Initialize switch %s", switchConf.Id)
		m.switches[switchIdx] = Switch{
			id:     switchConf.Id,
			idx:    switchIdx,
			fabric: m,
			config: switchConf,
			ports:  make([]Port, len(switchConf.Ports)),
		}

		s := &m.switches[switchIdx]

		// TODO: This should probably move to Start() routine, although
		// if we can't find the switch Start() won't really do anything anyways.
		log.Infof("identify switch %s", switchConf.Id)
		if err := s.identify(); err != nil {
			log.WithError(err).Warnf("Failed to identify switch %s", s.id)
		}

		log.Infof("Switch %s identified: PAX %d", switchConf.Id, s.paxId)

		for portIdx, portConf := range switchConf.Ports {
			portType := portConf.getPortType()

			s.ports[portIdx] = Port{
				id:       strconv.Itoa(portIdx),
				fabricId: strconv.Itoa(fabricPortId),
				portType: portType,
				portStatus: portStatus{
					cfgLinkWidth:    0,
					negLinkWidth:    0,
					curLinkRateGBps: 0,
					maxLinkRateGBps: 0,
					linkStatus:      sf.NO_LINK_PV130LS,
					linkState:       sf.DISABLED_PV130LST,
				},
				swtch:     &m.switches[switchIdx],
				config:    &switchConf.Ports[portIdx],
				endpoints: []*Endpoint{},
			}

			fabricPortId++
		}
	}

	// create the endpoints

	// Endpoint and Port relation
	//
	//       Endpoint         Port           Switch
	// [0  ] Rabbit           Mgmt           0, 1              One endpoint per mgmt (one mgmt port per switch)
	// [1  ] Compute 0        USP0			 0                 One endpoint per compuete
	// [2  ] Compute 1        USP1           0
	//   ...
	// [N-1] Compute N        USPN           1
	// [N  ] Drive 0 PF       DSP0           0 ---------------|
	// [N+1] Drive 0 VF0      DSP0           0                | Each drive is enumerated out to M endpoints
	// [N+2] Drive 0 VF1      DSP0           0                |   1 for the physical function (unused)
	//   ...                                                  |   1 for the rabbit
	// [N+M] Drive 0 VFM-1    DSP0           0 ---------------|   1 per compute
	//

	m.managementEndpointCount = 1
	m.upstreamEndpointCount = m.config.UpstreamPortCount

	mangementAndUpstreamEndpointCount := m.managementEndpointCount + m.upstreamEndpointCount
	m.downstreamEndpointCount = (1 + // PF
		mangementAndUpstreamEndpointCount) * m.config.DownstreamPortCount

	log.Debugf("Creating Endpoints:")
	log.Debugf("   Management Endpoints: % 3d", m.managementEndpointCount)
	log.Debugf("   Upstream Endpoints:   % 3d", m.upstreamEndpointCount)
	log.Debugf("   Downstream Endpoints: % 3d", m.downstreamEndpointCount)

	m.endpoints = make([]Endpoint, mangementAndUpstreamEndpointCount+m.downstreamEndpointCount)

	for endpointIdx := range m.endpoints {
		e := &m.endpoints[endpointIdx]

		e.id = strconv.Itoa(endpointIdx)
		e.index = endpointIdx
		e.fabric = m

		switch {
		case m.isManagementEndpoint(endpointIdx):
			e.endpointType = sf.PROCESSOR_EV150ET

			e.ports = make([]*Port, len(m.switches))
			for switchIdx, s := range m.switches {
				port := s.findPortByType(sf.MANAGEMENT_PORT_PV130PT, 0)

				e.ports[switchIdx] = port
				e.name = port.config.Name

				port.endpoints = make([]*Endpoint, 1)
				port.endpoints[0] = e
			}
		case m.isUpstreamEndpoint(endpointIdx):
			e.endpointType = sf.STORAGE_INITIATOR_EV150ET

			port := m.findPortByType(sf.UPSTREAM_PORT_PV130PT, m.getUpstreamEndpointRelativePortIndex(endpointIdx))
			e.ports = make([]*Port, 1)
			e.ports[0] = port
			e.name = port.config.Name

			port.endpoints = make([]*Endpoint, 1)
			port.endpoints[0] = e

		case m.isDownstreamEndpoint(endpointIdx):
			port := m.findPortByType(sf.DOWNSTREAM_PORT_PV130PT, m.getDownstreamEndpointRelativePortIndex(endpointIdx))

			if port.portType != sf.DOWNSTREAM_PORT_PV130PT {
				panic(fmt.Sprintf("Port %s not a DSP", port.id))
			}

			e.endpointType = sf.DRIVE_EV150ET
			e.ports = make([]*Port, 1)
			e.ports[0] = port

			if len(port.endpoints) == 0 {
				port.endpoints = make([]*Endpoint, 1+ // PF
					mangementAndUpstreamEndpointCount)
				port.endpoints[0] = e
			} else {
				port.endpoints[endpointIdx-port.GetBaseEndpointIndex()] = e
			}

			e.name = fmt.Sprintf("%s - Function %d", port.config.Name, endpointIdx-port.GetBaseEndpointIndex()) // must be after endpoint port assignment

		default:
			panic(fmt.Errorf("unhandled endpoint index %d", endpointIdx))
		}

	}

	// create the endpoint groups & connections

	// An Endpoint Groups is created for each managment and upstream endpoints, with
	// the associated target endpoints linked to form the group. This is conceptually
	// equivalent to the Host Virtualization Domains that exist in the PAX Switch.

	// A Connection is made for every endpoint (also representing the HVD). Connections
	// contain the attached volumes. The two are linked.
	m.endpointGroups = make([]EndpointGroup, mangementAndUpstreamEndpointCount)
	m.connections = make([]Connection, mangementAndUpstreamEndpointCount)
	for endpointGroupIdx := range m.endpointGroups {
		endpointGroup := &m.endpointGroups[endpointGroupIdx]
		connection := &m.connections[endpointGroupIdx]
		connection.fabric = m

		endpointGroup.id = strconv.Itoa(endpointGroupIdx)
		endpointGroup.fabric = m

		endpointGroup.endpoints = make([]*Endpoint, 1 /*Initiator*/ +m.config.DownstreamPortCount)
		endpointGroup.initiator = &endpointGroup.endpoints[0]

		endpointGroup.endpoints[0] = &m.endpoints[endpointGroupIdx] // Mgmt or USP
		endpointGroup.endpoints[0].controllerId = uint16(endpointGroupIdx + 1)

		for idx := range endpointGroup.endpoints[1:] {
			endpointGroup.endpoints[1+idx] =
				&m.endpoints[mangementAndUpstreamEndpointCount+idx*(1 /*PF*/ +mangementAndUpstreamEndpointCount)+endpointGroupIdx+1]
		}

		endpointGroup.connection = connection
		connection.endpointGroup = endpointGroup
	}

	if err := initializeMetrics(); err != nil {
		return err
	}

	// Fabric Manager must wait for NVMe drives to be enabled prior to running the bind
	// operation that will route drives to upstream ports. By subscribing to the event
	// manager, we can receive the desired "PortAutomaticallyEnabledFabric" event and
	// act accordingly.
	event.EventManager.Subscribe(m)

	log.Infof("Fabric Manager %s Initialization Finished", m.id)
	return nil
}

// Start -
func Start() error {
	m := manager
	log.Infof("Starting Fabric Manager %s", m.id)

	// Enumerate over the switch ports and report events to the event
	// manager
	for switchIdx := range m.switches {
		s := &m.switches[switchIdx]

		if !s.isReady() {
			log.Errorf("Failed to start switch %s: Switch is Down", s.id)
			continue
		}

		for portIdx := range s.ports {
			p := &s.ports[portIdx]
			if err := p.Initialize(); err != nil {
				log.WithError(err).Errorf("Switch %s Port %s failed to initialize", s.id, p.id)
			}
		}

		s.refreshPortStatus()
	}

	// Notify the event manager the fabric manager is ready
	event.EventManager.Publish(msgreg.FabricReadyNnf(m.id))

	// Run the Fabric Monitor in a background thread.
	go NewFabricMonitor(&m).Run()

	return nil
}

func (m *Fabric) EventHandler(e event.Event) error {
	if e.Is(msgreg.PortAutomaticallyEnabledFabric("", "")) {
		var switchId, portId string
		e.Args(&switchId, &portId)

		_, _, p := findPort(m.id, switchId, portId)
		if p == nil {
			return ec.NewErrInternalServerError().WithCause("Internal event illformed")
		}

		if p.portType == sf.DOWNSTREAM_PORT_PV130PT {
			if err := p.bind(); err != nil {
				log.WithError(err).Errorf("Port %s (switch: %s port: %d) failed to bind", p.id, p.swtch.id, p.config.Port)
			}
		}
	}

	return nil
}

// GetEndpoint - Returns the first endpoint for the given switch port. For USP, there will only ever be one ID. For DSP there will
// be an endpoint for each Physical and Virtual Functions on the DSP, but the
// first ID (corresponding to the PF) is what is returned.
func GetEndpoint(switchId, portId string) (*Endpoint, error) {
	for _, s := range manager.switches {
		for _, p := range s.ports {
			if s.id == switchId && p.id == portId {
				return p.endpoints[0], nil
			}
		}
	}

	return nil, ec.NewErrNotFound()
}

// Get -
func Get(model *sf.FabricCollectionFabricCollection) error {
	model.MembersodataCount = 1
	model.Members = make([]sf.OdataV4IdRef, model.MembersodataCount)
	model.Members[0].OdataId = fmt.Sprintf("/redfish/v1/Fabrics/%s", manager.id)

	return nil
}

// FabricIdGet -
func FabricIdGet(fabricId string, model *sf.FabricV120Fabric) error {
	f := findFabric(fabricId)
	if f == nil {
		return ec.NewErrNotFound()
	}

	model.FabricType = sf.PC_IE_PP
	model.Switches.OdataId = f.fmt("/Switches")
	model.Connections.OdataId = f.fmt("/Connections")
	model.Endpoints.OdataId = f.fmt("/Endpoints")
	model.EndpointGroups.OdataId = f.fmt("/EndpointGroups")

	return nil
}

// FabricIdSwitchesGet -
func FabricIdSwitchesGet(fabricId string, model *sf.SwitchCollectionSwitchCollection) error {
	f := findFabric(fabricId)
	if f == nil {
		return ec.NewErrNotFound()
	}

	model.MembersodataCount = int64(len(f.switches))
	model.Members = make([]sf.OdataV4IdRef, model.MembersodataCount)
	for idx, s := range f.switches {
		model.Members[idx].OdataId = s.fmt("") // fmt.Sprintf("/redfish/v1/Fabrics/%s/Switches/%s", fabricId, s.id)
	}

	return nil
}

// FabricIdSwitchesSwitchIdGet -
func FabricIdSwitchesSwitchIdGet(fabricId string, switchId string, model *sf.SwitchV140Switch) error {
	_, s := findSwitch(fabricId, switchId)
	if s == nil {
		return ec.NewErrNotFound()
	}

	model.Id = switchId
	model.SwitchType = sf.PC_IE_PP

	model.Status = s.getStatus()
	model.Model = s.getModel()
	model.Manufacturer = s.getManufacturer()
	model.SerialNumber = s.getSerialNumber()
	model.FirmwareVersion = s.getFirmwareVersion()

	model.Ports.OdataId = fmt.Sprintf("/redfish/v1/Fabrics/%s/Switches/%s/Ports", fabricId, switchId)

	return nil
}

// FabricIdSwitchesSwitchIdPortsGet -
func FabricIdSwitchesSwitchIdPortsGet(fabricId string, switchId string, model *sf.PortCollectionPortCollection) error {
	_, s := findSwitch(fabricId, switchId)
	if s == nil {
		return ec.NewErrNotFound()
	}

	model.MembersodataCount = int64(len(s.ports))
	model.Members = make([]sf.OdataV4IdRef, len(s.ports))
	for idx, port := range s.ports {
		model.Members[idx].OdataId = fmt.Sprintf("/redfish/v1/Fabrics/%s/Switches/%s/Ports/%s", fabricId, switchId, port.id)
	}

	return nil
}

// FabricIdSwitchesSwitchIdPortsPortIdGet -
func FabricIdSwitchesSwitchIdPortsPortIdGet(fabricId string, switchId string, portId string, model *sf.PortV130Port) error {
	_, _, p := findPort(fabricId, switchId, portId)

	model.Name = p.config.Name
	model.Id = p.id

	model.PortProtocol = sf.PC_IE_PP
	model.PortMedium = sf.ELECTRICAL_PV130PM
	model.PortType = p.portType
	model.PortId = strconv.Itoa(p.config.Port)

	model.Width = int64(p.config.Width)
	model.ActiveWidth = 0 // TODO

	//model.MaxSpeedGbps = 0 // TODO
	//model.CurrentSpeedGbps = 0 // TODO

	model.LinkState = sf.ENABLED_PV130LST
	model.LinkStatus = p.linkStatus

	model.Links.AssociatedEndpointsodataCount = int64(len(p.endpoints))
	model.Links.AssociatedEndpoints = make([]sf.OdataV4IdRef, model.Links.AssociatedEndpointsodataCount)
	for idx, ep := range p.endpoints {
		model.Links.AssociatedEndpoints[idx].OdataId = fmt.Sprintf("/redfish/v1/Fabrics/%s/Endpoints/%s", fabricId, ep.id)
	}

	return nil
}

// FabricIdEndpointsGet -
func FabricIdEndpointsGet(fabricId string, model *sf.EndpointCollectionEndpointCollection) error {
	f := findFabric(fabricId)
	if f == nil {
		return ec.NewErrNotFound()
	}

	model.MembersodataCount = int64(len(f.endpoints))
	model.Members = make([]sf.OdataV4IdRef, model.MembersodataCount)
	for idx, ep := range f.endpoints {
		model.Members[idx].OdataId = ep.fmt("") //fmt.Sprintf("/redfish/v1/Fabrics/%s/Endpoints/%d", fabricId, idx)
	}

	return nil
}

// FabricIdEndpointsEndpointIdGet -
func FabricIdEndpointsEndpointIdGet(fabricId string, endpointId string, model *sf.EndpointV150Endpoint) error {
	_, ep := findEndpoint(fabricId, endpointId)
	if ep == nil {
		return ec.NewErrNotFound()
	}

	role := func(ep *Endpoint) sf.EndpointV150EntityRole {
		if ep.endpointType == sf.DRIVE_EV150ET {
			return sf.TARGET_EV150ER
		}

		return sf.INITIATOR_EV150ER
	}

	model.Id = ep.id
	model.Name = ep.name
	model.EndpointProtocol = sf.PC_IE_PP
	model.ConnectedEntities = make([]sf.EndpointV150ConnectedEntity, 1)
	model.ConnectedEntities = []sf.EndpointV150ConnectedEntity{{
		EntityType: ep.endpointType,
		EntityRole: role(ep),
	}}

	model.PciId = sf.EndpointV150PciId{
		ClassCode:      "", // TODO
		DeviceId:       "", // TODO
		FunctionNumber: 0,  // TODO
	}

	model.Links.Ports = make([]sf.OdataV4IdRef, len(ep.ports))
	for idx, port := range ep.ports {
		model.Links.Ports[idx].OdataId = fmt.Sprintf("/redfish/v1/Fabrics/%s/Switches/%s/Ports/%s", fabricId, port.swtch.id, port.id)
	}

	// TODO: Correctly report endpoint state
	model.Status = sf.ResourceStatus{
		State:  sf.ENABLED_RST,
		Health: sf.OK_RH,
	}

	type Oem struct {
		Pdfid         int
		Bound         bool
		BoundPaxId    int
		BoundHvdPhyId int
		BoundHvdLogId int
	}

	oem := Oem{
		Pdfid:         int(ep.pdfid),
		Bound:         ep.bound,
		BoundPaxId:    int(ep.boundPaxId),
		BoundHvdPhyId: int(ep.boundHvdPhyId),
		BoundHvdLogId: int(ep.boundHvdLogId),
	}

	model.Oem = openapi.MarshalOem(oem)

	return nil
}

// FabricIdEndpointGroupsGet -
func FabricIdEndpointGroupsGet(fabricId string, model *sf.EndpointGroupCollectionEndpointGroupCollection) error {
	f := findFabric(fabricId)
	if f == nil {
		return ec.NewErrNotFound()
	}

	model.MembersodataCount = int64(len(f.endpointGroups))
	model.Members = make([]sf.OdataV4IdRef, model.MembersodataCount)
	for idx, epg := range f.endpointGroups {
		model.Members[idx].OdataId = epg.fmt("") // fmt.Sprintf("/redfish/v1/Fabrics/%s/EndpointGroups/%d", fabricId, idx)
	}

	return nil
}

func FabricIdEndpointGroupsEndpointIdGet(fabricId string, groupId string, model *sf.EndpointGroupV130EndpointGroup) error {
	f, epg := findEndpointGroup(fabricId, groupId)
	if epg == nil {
		return ec.NewErrNotFound()
	}

	model.Links.EndpointsodataCount = int64(len(epg.endpoints))
	model.Links.Endpoints = make([]sf.OdataV4IdRef, model.Links.EndpointsodataCount)
	for idx, ep := range epg.endpoints {
		model.Links.Endpoints[idx].OdataId = ep.fmt("") //fmt.Sprintf("/redfish/v1/Fabrics/%s/Endpoints/%s", fabricId, ep.id)
	}

	model.Links.ConnectionsodataCount = 1
	model.Links.Connections = make([]sf.OdataV4IdRef, model.Links.ConnectionsodataCount)
	model.Links.Connections[0].OdataId = f.fmt("/Connections/%s", epg.id) //fmt.Sprintf("/redfish/v1/Fabrics/%s/Connections/%s", fabricId, epg.id)

	model.Id = epg.id

	return nil
}

// FabricIdConnectionsGet -
func FabricIdConnectionsGet(fabricId string, model *sf.ConnectionCollectionConnectionCollection) error {
	f := findFabric(fabricId)
	if f == nil {
		return ec.NewErrNotFound()
	}

	model.MembersodataCount = int64(len(f.connections))
	model.Members = make([]sf.OdataV4IdRef, model.MembersodataCount)
	for idx, c := range f.connections {
		model.Members[idx].OdataId = c.fmt("") //fmt.Sprintf("/redfish/v1/Fabrics/%s/Connections/%s", fabricId, c.endpointGroup.id)
	}

	return nil
}

// FabricIdConnectionsConnectionIdGet
func FabricIdConnectionsConnectionIdGet(fabricId string, connectionId string, model *sf.ConnectionV100Connection) error {
	f, c := findConnection(fabricId, connectionId)
	if c == nil {
		return ec.NewErrNotFound()
	}

	endpointGroup := c.endpointGroup
	initiator := *endpointGroup.initiator

	model.Id = connectionId
	model.ConnectionType = sf.STORAGE_CV100CT

	model.Links.InitiatorEndpointsodataCount = 1
	model.Links.InitiatorEndpoints = make([]sf.OdataV4IdRef, model.Links.InitiatorEndpointsodataCount)
	model.Links.InitiatorEndpoints[0].OdataId = f.fmt("/Endpoints/%s", initiator.id) // fmt.Sprintf("/redfish/v1/Fabrics/%s/Endpoints/%s", fabricId, initiator.id)

	model.Links.TargetEndpointsodataCount = int64(len(endpointGroup.endpoints) - 1)
	model.Links.TargetEndpoints = make([]sf.OdataV4IdRef, model.Links.TargetEndpointsodataCount)
	for idx, ep := range endpointGroup.endpoints[1:] {
		model.Links.TargetEndpoints[idx].OdataId = ep.fmt("") // fmt.Sprintf("/redfish/v1/Fabrics/%s/Endpoints/%s", fabricId, endpoint.id)
	}

	// TODO: This should be by controllerId uint16 (not a string)
	controllerId := strconv.Itoa(int(initiator.controllerId))
	volumes, err := api.NvmeInterface.GetVolumes(controllerId)
	if err != nil {
		return err
	}

	model.VolumeInfo = make([]sf.ConnectionV100VolumeInfo, len(volumes))
	for idx, volume := range volumes {
		v := &model.VolumeInfo[idx]

		v.Volume.OdataId = volume
		v.AccessState = sf.OPTIMIZED_CV100AST
		v.AccessCapabilities = []sf.ConnectionV100AccessCapability{
			sf.READ_CV100AC,
			sf.WRITE_CV100AC,
		}
	}

	return nil
}

// FabricIdConnectionsConnectionIdPatch -
func FabricIdConnectionsConnectionIdPatch(fabricId string, connectionId string, model *sf.ConnectionV100Connection) error {
	return ec.NewErrNotAcceptable()
	/*
		if !isFabric(fabricId) {
			return ec.NewErrNotFound()
		}

		c, err := fabric.findConnection(connectionId)
		if err != nil {
			return err
		}

		initiator := *c.endpointGroup.initiator

		for _, volumeInfo := range model.VolumeInfo {
			odataid := volumeInfo.Volume.OdataId
			if err := NvmeInterface.AttachVolume(odataid, initiator.controllerId); err != nil {
				return err
			}
		}

		return nil
	*/
}

func GetSwitchDevice(fabricId, switchId string) *switchtec.Device {
	_, s := findSwitch(fabricId, switchId)
	if s == nil {
		return nil
	}

	return s.dev.Device()
}

func GetSwitchPath(fabricId, switchId string) *string {
	_, s := findSwitch(fabricId, switchId)
	if s == nil {
		return nil
	}

	return s.dev.Path()
}

// GetPortPDFID
func GetPortPDFID(fabricId, switchId, portId string, controllerId uint16) (uint16, error) {
	_, _, p := findPort(fabricId, switchId, portId)
	if p == nil {
		return 0, fmt.Errorf("Port %s not found in fabric %s switch %s", portId, fabricId, portId)
	}

	if p.portType != sf.DOWNSTREAM_PORT_PV130PT {
		return 0, fmt.Errorf("Port %s of Type %s has no PDFID", portId, p.portType)
	}

	if !(int(controllerId) < len(p.endpoints)) {
		return 0, fmt.Errorf("controller ID beyond available port endpoints")
	}

	return p.endpoints[int(controllerId)].pdfid, nil
}

func (f *Fabric) GetDownstreamPortRelativePortIndex(switchId, portId string) (int, error) {
	s := f.findSwitch(switchId)
	if s == nil {
		return -1, fmt.Errorf("Switch not found: Switch: %s", switchId)
	}

	relativePortIndex := 0
	for _, s := range f.switches {
		for _, p := range s.ports {
			if p.portType == sf.DOWNSTREAM_PORT_PV130PT {

				if s.id == switchId && p.id == portId {
					return relativePortIndex, nil
				}

				relativePortIndex++
			}
		}
	}

	return -1, fmt.Errorf("Switch Port not found: Switch: %s Port: %s", switchId, portId)

}

// FindDownstreamEndpoint -
func (f *Fabric) FindDownstreamEndpoint(portId, functionId string) (string, error) {
	idx, err := strconv.Atoi(portId)
	if err != nil {
		return "", ec.NewErrNotFound()
	}
	port := f.findPortByType(sf.DOWNSTREAM_PORT_PV130PT, idx)
	if port == nil {
		return "", ec.NewErrNotFound()
	}
	ep := port.findEndpoint(functionId)
	if ep == nil {
		return "", ec.NewErrNotFound()
	}

	return fmt.Sprintf("/redfish/v1/Fabrics/%s/Endpoints/%s", f.id, ep.id), nil
}
