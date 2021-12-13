package telemetry

import (
	ec "github.hpe.com/hpe/hpc-rabsw-nnf-ec/pkg/ec"
)

// Router contains all the Redfish / Swordfish API calls for the Telemetry Service
type DefaultApiRouter struct {
	servicer Api
}

func NewDefaultApiRouter(s Api) ec.Router {
	return &DefaultApiRouter{servicer: s}
}

func (*DefaultApiRouter) Name() string {
	return "Telemetry Manager"
}

func (*DefaultApiRouter) Init() error {
	return TelemetryManager.Initialize()
}

func (*DefaultApiRouter) Start() error {
	return nil
}

func (*DefaultApiRouter) Close() error {
	return nil
}

func (r *DefaultApiRouter) Routes() ec.Routes {
	s := r.servicer
	return ec.Routes{
		{
			Name:        "RedfishV1TelemetryServiceGet",
			Method:      ec.GET_METHOD,
			Path:        "/redfish/v1/TelemetryService",
			HandlerFunc: s.RedfishV1TelemetryServiceGet,
		},
		{
			Name:        "RedfishV1TelemetryMetricDefinitionsGet",
			Method:      ec.GET_METHOD,
			Path:        "/redfish/v1/TelemetryService/MetricDefinitions",
			HandlerFunc: s.RedfishV1TelemetryMetricDefinitionsGet,
		},
		{
			Name:        "RedfishV1TelemetryMetricDefinitionIdGet",
			Method:      ec.GET_METHOD,
			Path:        "/redfish/v1/TelemetryService/MetricDefinitions/{MetricDefinitionId}",
			HandlerFunc: s.RedfishV1TelemetryMetricDefinitionIdGet,
		},
		{
			Name:        "RedfishV1TelemetryMetricReportDefinitionsGet",
			Method:      ec.GET_METHOD,
			Path:        "/redfish/v1/TelemetryService/MetricReportDefinitions",
			HandlerFunc: s.RedfishV1TelemetryMetricReportDefinitionsGet,
		},
		{
			Name:        "RedfishV1TelemetryMetricReportDefinitionIdGet",
			Method:      ec.GET_METHOD,
			Path:        "/redfish/v1/TelemetryService/MetricReportDefinitions/{MetricReportDefinitionId}",
			HandlerFunc: s.RedfishV1TelemetryMetricReportDefinitionIdGet,
		},
		{
			Name:        "RedfishV1TelemetryMetricReportsGet",
			Method:      ec.GET_METHOD,
			Path:        "/redfish/v1/TelemetryService/MetricReports",
			HandlerFunc: s.RedfishV1TelemetryMetricReportsGet,
		},
		{
			Name:        "RedfishV1TelemetryMetricReportIdGet",
			Method:      ec.GET_METHOD,
			Path:        "/redfish/v1/TelemetryService/MetricReports/{MetricReportId}",
			HandlerFunc: s.RedfishV1TelemetryMetricReportIdGet,
		},
	}
}
