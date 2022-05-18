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

package telemetry

import (
	"net/http"
)

type Api interface {
	RedfishV1TelemetryServiceGet(w http.ResponseWriter, r *http.Request)

	RedfishV1TelemetryMetricDefinitionsGet(w http.ResponseWriter, r *http.Request)
	RedfishV1TelemetryMetricDefinitionIdGet(w http.ResponseWriter, r *http.Request)

	RedfishV1TelemetryMetricReportDefinitionsGet(w http.ResponseWriter, r *http.Request)
	RedfishV1TelemetryMetricReportDefinitionIdGet(w http.ResponseWriter, r *http.Request)

	RedfishV1TelemetryMetricReportsGet(w http.ResponseWriter, r *http.Request)
	RedfishV1TelemetryMetricReportIdGet(w http.ResponseWriter, r *http.Request)
}
