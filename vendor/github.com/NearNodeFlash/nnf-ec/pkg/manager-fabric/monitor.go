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
	"math"
	"os"
	"time"

	"github.com/NearNodeFlash/nnf-ec/internal/switchtec/pkg/switchtec"
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

// Run Fabric Monitor forever
func (m *monitor) Run() {

	for {

		time.Sleep(time.Second * 60)

		for idx := range m.fabric.switches {
			s := &m.fabric.switches[idx]

			// The normal path is when the switch is operating without issue and we can
			// poll the switch for any events then process those events
			if s.isReady() {

				if events, err := s.dev.GetEvents(); err == nil {

					// In the steady state there will be no events.
					// Refresh the port status to ensure we're up to date.
					if len(events) == 0 {
						s.refreshPortStatus()
						continue
					}

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

	// Check if the switch path changed by trying to re-identify the switch.
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

func (m *monitor) getEventInfo(e switchtec.GfmsEvent) (uint8, bool /* is down event? */) {

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
