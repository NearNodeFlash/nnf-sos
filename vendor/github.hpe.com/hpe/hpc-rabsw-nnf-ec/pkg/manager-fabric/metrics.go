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
	"fmt"
	"time"

	telemetry "github.hpe.com/hpe/hpc-rabsw-nnf-ec/pkg/manager-telemetry"
	sf "github.hpe.com/hpe/hpc-rabsw-nnf-ec/pkg/rfsf/pkg/models"
)

func initializeMetrics() error {

	// Initialize a metric for recording the Tx and Rx bytes for each switch/port
	// It just so happens each switch has the same configuration, so we can use
	// wildcards to represent the the available devices.

	switchIds := func() []string {
		ids := make([]string, len(manager.switches))
		for idx, s := range manager.switches {
			ids[idx] = s.id
		}
		return ids
	}()

	portIds := func() []string {
		s := manager.switches[0]
		ids := make([]string, len(s.ports))
		for idx, p := range s.ports {
			ids[idx] = p.id
		}
		return ids
	}()

	wildcards := []sf.MetricDefinitionV110Wildcard{
		{
			Name:   "SwitchId",
			Values: switchIds,
		},
		{
			Name:   "PortId",
			Values: portIds,
		},
	}

	path := fmt.Sprintf("/redfish/v1/Fabrics/%s/Switches/{SwitchId}/Ports/{PortId}/Metrics/", manager.id)
	properties := []string{
		path + "RxBytes",
		path + "TxBytes",
	}

	if err := telemetry.TelemetryManager.RegisterMetric(
		&telemetry.MetricDefinition{
			Id:               "SwitchPortTxRx",
			Name:             "Switch Port Tx / Rx Metric Definition",
			MetricType:       sf.COUNTER_MDV110MT,
			Implementation:   sf.DIGITAL_METER_MDV110IT,
			MetricDataType:   sf.INTEGER_MDV110MDT,
			Units:            "Bytes",
			Wildcards:        wildcards,
			MetricProperties: properties,
		},
		&telemetry.MetricReportDefinition{
			MetricReportDefinitionType:    sf.ON_REQUEST_MRDV133MRDT,
			MetricReportDefinitionEnabled: true,
			ReportActions: []sf.MetricReportDefinitionV133ReportActionsEnum{
				sf.LOG_TO_METRIC_REPORTS_COLLECTION_MRDV133RAE,
			},
			ReportUpdates: sf.OVERWRITE_MRDV133RUE,
		},
		func(mrd *telemetry.MetricReportDefinition) ([]telemetry.MetricReportValue, error) {

			vals := make([]telemetry.MetricReportValue, len(switchIds)*len(portIds)*len(properties))
			for switchIdx, s := range manager.switches {
				metrics, err := s.dev.GetPortMetrics()
				if err != nil {
					return nil, err
				}

				timestamp := time.Now()
				offset := switchIdx * len(portIds) * len(properties)
				for idx, metric := range metrics {
					p := s.findPortByPhysicalPortId(metric.PhysPortId)

					vals[idx*len(properties)+offset+0] = telemetry.MetricReportValue{
						Timestamp:      &timestamp,
						MetricValue:    fmt.Sprintf("%d", metric.Ingress.Total()),
						MetricProperty: fmt.Sprintf("/redfish/v1/Fabrics/%s/Switches/%s/Ports/%s/Metrics/RxBytes", manager.id, s.id, p.id),
					}
					vals[idx*len(properties)+offset+1] = telemetry.MetricReportValue{
						Timestamp:      &timestamp,
						MetricValue:    fmt.Sprintf("%d", metric.Egress.Total()),
						MetricProperty: fmt.Sprintf("/redfish/v1/Fabrics/%s/Switches/%s/Ports/%s/Metrics/TxBytes", manager.id, s.id, p.id),
					}
				}
			}

			return vals, nil
		},
	); err != nil {
		return err
	}

	return nil
}
