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
