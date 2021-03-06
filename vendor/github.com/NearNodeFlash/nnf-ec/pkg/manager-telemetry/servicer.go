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
	"fmt"
	"net/http"

	. "github.com/NearNodeFlash/nnf-ec/pkg/common"

	sf "github.com/NearNodeFlash/nnf-ec/pkg/rfsf/pkg/models"
)

type DefaultApiService struct{}

func NewDefaultApiService() Api {
	return &DefaultApiService{}
}

func (*DefaultApiService) RedfishV1TelemetryServiceGet(w http.ResponseWriter, r *http.Request) {
	model := sf.TelemetryServiceV121TelemetryService{
		OdataId:   "/redfish/v1/TelemetryService",
		OdataType: "#TelemetryService.v1_2_0.TelemetryService",
		Name:      "TelemetryService",
	}

	err := Get(&model)

	EncodeResponse(model, err, w)
}

func (*DefaultApiService) RedfishV1TelemetryMetricDefinitionsGet(w http.ResponseWriter, r *http.Request) {

	model := sf.MetricDefinitionCollectionMetricDefinitionCollection{
		OdataId:   "/redfish/v1/TelemetryService/MetricDefinitions",
		OdataType: "#MetricDefintionCollection.v1_0_0.MetricDefinitionCollection",
		Name:      "Metric Definition Collection",
	}

	err := MetricDefinitionsGet(&model)

	EncodeResponse(model, err, w)
}

func (*DefaultApiService) RedfishV1TelemetryMetricDefinitionIdGet(w http.ResponseWriter, r *http.Request) {
	params := Params(r)
	metricDefinitionId := params["MetricDefinitionId"]

	model := sf.MetricDefinitionV110MetricDefinition{
		OdataId:   fmt.Sprintf("/redfish/v1/TelemetryService/MetricDefinitions/%s", metricDefinitionId),
		OdataType: "#MetricDefinition.v1_0_0.MetricDefinition",
		Name:      "Metric Definition",
	}

	err := MetricDefinitionIdGet(&model, metricDefinitionId)

	EncodeResponse(model, err, w)
}

func (*DefaultApiService) RedfishV1TelemetryMetricReportDefinitionsGet(w http.ResponseWriter, r *http.Request) {

	model := sf.MetricReportDefinitionCollectionMetricReportDefinitionCollection{
		OdataId:   "/redfish/v1/TelemetryService/MetricReportDefinitions",
		OdataType: "#MetricReportDefinitionCollection.v1_0_0.MetricReportDefinitionCollection",
		Name:      "Metric Report Definition Collection",
	}

	err := MetricReportDefinitionsGet(&model)

	EncodeResponse(model, err, w)
}

func (*DefaultApiService) RedfishV1TelemetryMetricReportDefinitionIdGet(w http.ResponseWriter, r *http.Request) {
	params := Params(r)
	metricReportDefinitionId := params["MetricReportDefinitionId"]

	model := sf.MetricReportDefinitionV133MetricReportDefinition{
		OdataId:   fmt.Sprintf("/redfish/v1/TelemetryService/MetricReportDefinitions/%s", metricReportDefinitionId),
		OdataType: "#MetricReportDefinition.v1_3_3.MetricReportDefinition",
		Name:      "Metric Report Definition",
	}

	err := MetricReportDefinitionIdGet(&model, metricReportDefinitionId)

	EncodeResponse(model, err, w)
}

func (*DefaultApiService) RedfishV1TelemetryMetricReportsGet(w http.ResponseWriter, r *http.Request) {

	model := sf.MetricReportCollectionMetricReportCollection{
		OdataId:   "/redfish/v1/TelemetryService/MetricReports",
		OdataType: "#MetricReportCollection.v1_0_0.MetricReportCollection",
		Name:      "Metric Report Collection",
	}

	err := MetricReportsGet(&model)

	EncodeResponse(model, err, w)
}

func (*DefaultApiService) RedfishV1TelemetryMetricReportIdGet(w http.ResponseWriter, r *http.Request) {
	params := Params(r)
	metricReportId := params["MetricReportId"]

	model := sf.MetricReportV140MetricReport{
		OdataId:   fmt.Sprintf("/redfish/v1/TelemetryService/MetricReport/%s", metricReportId),
		OdataType: "#MetricReport.v1_4_0.MetricReport",
		Name:      "Metric Report",
	}

	err := MetricReportIdGet(&model, metricReportId)

	EncodeResponse(model, err, w)
}
