/*
 * Copyright 2025 Hewlett Packard Enterprise Development LP
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
	"os"
	"time"

	"github.com/NearNodeFlash/nnf-ec/internal/switchtec/pkg/nvme"
	"github.com/NearNodeFlash/nnf-ec/pkg/ec"
	sf "github.com/NearNodeFlash/nnf-ec/pkg/rfsf/pkg/models"
)

// StartNVMeMonitor starts a background goroutine that periodically queries all NVMe devices for their SMART log.
func StartNVMeMonitor(log ec.Logger) {
	// Read drive monitor period from environment variable
	defaultDriveMonitorPeriod := 1 * time.Hour
	driveMonitorPeriod := defaultDriveMonitorPeriod
	if periodStr := os.Getenv("NNF_DRIVE_MONITOR_PERIOD"); periodStr != "" {
		if d, err := time.ParseDuration(periodStr); err == nil {
			driveMonitorPeriod = d
		} else {
			log.Info("Invalid NNF_DRIVE_MONITOR_PERIOD, using default", "value", driveMonitorPeriod, "error", err)
		}
	}

	// A period of 0 means don't start the monitor.
	if driveMonitorPeriod == 0 {
		log.Info("Not starting NVMe monitor", "monitorPeriod", driveMonitorPeriod)
		return
	}

	monitor := NewMonitor(log, driveMonitorPeriod)
	go monitor.Run()
	log.Info("Started NVMe monitor", "monitorPeriod", driveMonitorPeriod)
}

// Monitor periodically queries all NVMe devices for their SMART log
// and can be started as a background goroutine.
type Monitor struct {
	Log        ec.Logger
	Interval   time.Duration
	GetDevices func() []*nvme.Device // Function to return all NVMe devices
}

// NewMonitor creates a new NVMe monitor with the given polling interval and device getter.
func NewMonitor(log ec.Logger, interval time.Duration) *Monitor {
	return &Monitor{
		Log:      log,
		Interval: interval,
	}
}

// Run starts the monitor loop. It periodically calls GetSmartLog on all devices.
func (m *Monitor) Run() {
	for {
		time.Sleep(m.Interval)

		storages := GetStorage()
		for _, storage := range storages {
			if !storage.IsEnabled() {
				continue
			}

			m.checkSmartLog(storage)
		}
	}
}

func (m *Monitor) checkSmartLog(storage *Storage) {
	log := m.Log

	smartLog, err := storage.device.GetSmartLog()
	if err != nil {
		log.Error(err, "smartlog request failed", "serial", storage.SerialNumber(), "slot", storage.Slot())
		storage.notify(sf.UNAVAILABLE_OFFLINE_RST)
	} else {

		// nvme.MangleSmartLog(smartLog)

		state := nvme.InterpretSmartLog(smartLog)
		if state != storage.state {
			log.Info("smartlog state change", "old state", storage.state, "new state", state, "serial", storage.SerialNumber(), "slot", storage.Slot())
			storage.notify(state)

			nvme.LogSmartLog(log, smartLog)
		}
	}
}
