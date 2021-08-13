package fabric

import (
	"math"
	"os"
	"time"

	"stash.us.cray.com/rabsw/switchtec-fabric/pkg/switchtec"
)

// The Fabric Monitor is responsible for ensuring that the fabriic and related sub-resource
// are updated with the latest information from the switch. This runs as a background
// thread, and periodically queries the fabric.
func NewFabricMonitor(f *Fabric) *monitor {
	return &monitor{fabric: f}
}

type monitor struct {
	fabric *Fabric
}

// Run will run the Fabirc Monitor forever
func (m *monitor) Run() {

	for {

		time.Sleep(time.Second * 60)

		for idx := range m.fabric.switches {
			s := &m.fabric.switches[idx]

			// The normal path is when the switch is operating without issue and we can
			// poll the switch for any events, and process those events
			if s.isReady() {

				if events, err := s.dev.GetEvents(); err == nil {
					for _, event := range events {
						physPortId, isDown := m.getEventInfo(event)

						if physPortId == invalidPhysicalPortId {
							continue
						}

						if p := s.findPortByPhysicalPortId(physPortId); p != nil {
							p.notify(isDown)
						}
					}

					continue
				}
			}

			m.checkSwitchStatus(s)
		}
	}

}

func (*monitor) checkSwitchStatus(s *Switch) {

	// Check if the switch path changed by trying to re-identifying the switch.
	// If the switch is found, it's likely the switch path has changed and we
	// need to re-open the switch.
	if err := s.identify(); err != nil {

		s.setDown()

		// Check if the kernel sees the switchtec device
		if _, err := os.Stat(s.path); os.IsNotExist(err) {
			// TODO: Go Device Missing
		} else {
			// TODO: Go Device Error
		}
	}

}

const invalidPhysicalPortId = math.MaxUint8

func (m *monitor) getEventInfo(e switchtec.GfmsEvent) (uint8, bool) {

	switch e.Id {
	case switchtec.FabricLinkUp_GfmsEvent, switchtec.FabricLinkDown_GfmsEvent:
		return 0, e.Id == switchtec.FabricLinkDown_GfmsEvent // TODO: This should consult with the Fabric Manager and return the interswitch-port
	case switchtec.HostLinkUp_GfmsEvent, switchtec.HostLinkDown_GfmsEvent:
		return uint8(switchtec.NewHostGfmsEvent(e).(*switchtec.HostGfmsEvent).PhysPortId), e.Id == switchtec.HostLinkDown_GfmsEvent
	case switchtec.DeviceAdd_GfmsEvent, switchtec.DeviceDelete_GfmsEvent:
		return uint8(switchtec.NewDeviceGfmsEvent(e).(*switchtec.DeviceGfmsEvent).PhysPortId), e.Id == switchtec.DeviceDelete_GfmsEvent
	}

	return invalidPhysicalPortId, false
}
