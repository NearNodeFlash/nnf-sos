package telemetry

import (
	"fmt"
	"time"

	"github.com/senseyeio/duration"

	ec "stash.us.cray.com/rabsw/nnf-ec/pkg/ec"
	sf "stash.us.cray.com/rabsw/rfsf-openapi/pkg/models"
)

var TelemetryManager = manager{}

type manager struct {
	metrics []metric

	nextReportTime time.Time   // The next time at which the manager will wake and perform a measurement of the necessary metrics
	timer          *time.Timer // Timer used to wake the manager when necessary
}

// Metric Definition contains the definition, metadata, or characteristics for a metric.
// It contains links to the metric properties to which the definition applies.
type MetricDefinition = sf.MetricDefinitionV110MetricDefinition

// Metric Report Definition specifies the metric reports that are generated.
type MetricReportDefinition = sf.MetricReportDefinitionV133MetricReportDefinition

// Metric Report contains the readings and results of a Metric Report Definition.
type MetricReport = sf.MetricReportV140MetricReport

// Metric Report Value defines the metric data reported from the metric
type MetricReportValue = sf.MetricReportV140MetricValue

// Metric Report Generator defines the function interface for recording a metric
// based on the metric report definition.
type MetricReportGenerator func(*MetricReportDefinition) ([]MetricReportValue, error)

const (
	DefaultReportDuration time.Duration = time.Minute * 2
)

// Initialize the Telemetry Manager. The manager is responsible for periodically gathering
// metrics that have been registered through calls to RegisterMetric() and have a Periodic
// metric definition type.
func (m *manager) Initialize() error {
	m.timer = time.NewTimer(DefaultReportDuration)
	go m.run()
	return nil
}

// Register Metric allows users of the Telemetry Manager to create a new Metric that will be managed and
// reported by the Telemetry Manager.
func (m *manager) RegisterMetric(definition *MetricDefinition, reportDefinition *MetricReportDefinition, generator MetricReportGenerator) error {

	id := definition.Id
	if len(id) == 0 {
		return fmt.Errorf("metric id is missing. specify a unique id in the metric definition")
	} else if metric := findMetric(id); metric != nil {
		for idx := range m.metrics {
			if m.metrics[idx].id == metric.id {
				m.metrics = append(m.metrics[:idx], m.metrics[idx+1:]...)
				break
			}
		}
	}

	name := definition.Name
	if len(name) == 0 {
		return fmt.Errorf("metric %s has no name. Provide a descriptive name for this metric", id)
	}

	now := time.Now()

	createReportDefinitionWildcards := func(d *MetricDefinition) []sf.MetricReportDefinitionV133Wildcard {
		wildcards := make([]sf.MetricReportDefinitionV133Wildcard, len(d.Wildcards))
		for idx, wc := range d.Wildcards {
			wildcards[idx] = sf.MetricReportDefinitionV133Wildcard{
				Name:   wc.Name,
				Values: wc.Values,
			}
		}
		return wildcards
	}

	// Since we're using deep-copies of the definitions and reports, we need to initialize
	// some standard redfish values in all the data structures
	definition.OdataId = fmt.Sprintf("/redfish/v1/TelemetryService/MetricDefinitions/%s", id)
	definition.OdataType = "##MetricDefinition.v1_0_0.MetricDefinition"

	reportDefinition.Id = id
	reportDefinition.Name = fmt.Sprintf("%s Report Definition", name)
	reportDefinition.OdataId = fmt.Sprintf("/redfish/v1/TelemetryService/MetricReportDefinitions/%s", id)
	reportDefinition.OdataType = "#MetricReportDefinition.v1_3_3.MetricReportDefinition"
	reportDefinition.MetricReport = sf.OdataV4IdRef{OdataId: fmt.Sprintf("/redfish/v1/TelemetryService/MetricReports/%s", id)}
	reportDefinition.Wildcards = createReportDefinitionWildcards(definition)
	reportDefinition.MetricProperties = definition.MetricProperties

	metric := metric{
		id:               id,
		definition:       definition,
		reportDefinition: reportDefinition,
		generator:        generator,
		report: sf.MetricReportV140MetricReport{
			Id:                     id,
			Name:                   fmt.Sprintf("%s Report", name),
			OdataId:                fmt.Sprintf("/redfish/v1/TelemetryService/MetricReports/%s", id),
			OdataType:              "#MetricReport.v1_4_0.MetricReport",
			MetricReportDefinition: sf.OdataV4IdRef{OdataId: reportDefinition.OdataId},
			MetricValues:           make([]sf.MetricReportV140MetricValue, 0, reportDefinition.AppendLimit),
			Timestamp:              &now,
		},
	}

	// Compute the first report time for this metric
	metric.nextReportTime = metric.computeNextReportTime(now)

	TelemetryManager.metrics = append(TelemetryManager.metrics, metric)

	// If this metric arrives before our current expiration time, adjust the
	// metric timer to expire at this metrics desired time.
	if !metric.nextReportTime.IsZero() && metric.nextReportTime.Before(m.nextReportTime) {
		m.timer.Reset(metric.nextReportTime.Sub(now))
	}

	return nil
}

// Run is meant to capture metrics that have been registered with the Telemetry Manager through calls
// to RegisterMetric(). While run executes, it will periodically wake up to record metrics per their
// defined Metric Definition Type and Schedule.
func (m *manager) run() {

	for {
		select {

		case currentTime := <-m.timer.C:
			nextReportTime := currentTime.Add(DefaultReportDuration)
			for idx := range m.metrics {
				metric := &m.metrics[idx]

				if metric.nextReportTime.IsZero() {
					continue
				}

				if currentTime.After(metric.nextReportTime) {
					metric.lastReportTime = currentTime
					metric.nextReportTime = metric.computeNextReportTime(currentTime)

					if metric.reportDefinition.MetricReportDefinitionEnabled {
						// Generate the metric report and handle the results
						values, err := metric.generator(metric.reportDefinition)
						if err != nil {
							metric.reportDefinition.Status.Health = sf.CRITICAL_RH
						} else {
							metric.record(values)
						}
					}
				}

				if metric.nextReportTime.Before(nextReportTime) {
					nextReportTime = metric.nextReportTime
				}
			}

			// Reset our timer to expire at the next metric time
			m.nextReportTime = nextReportTime
			m.timer.Reset(nextReportTime.Sub(currentTime))
		}
	}
}

type metric struct {
	id               string
	definition       *MetricDefinition
	reportDefinition *MetricReportDefinition
	report           MetricReport
	generator        MetricReportGenerator
	lastReportTime   time.Time
	nextReportTime   time.Time
}

func findMetric(id string) *metric {
	for idx := range TelemetryManager.metrics {
		if TelemetryManager.metrics[idx].id == id {
			return &TelemetryManager.metrics[idx]
		}
	}

	return nil
}

func (metric *metric) computeNextReportTime(current time.Time) time.Time {
	switch metric.reportDefinition.MetricReportDefinitionType {
	case sf.PERIODIC_MRDV133MRDT:
		d, err := duration.ParseISO8601(metric.reportDefinition.Schedule.RecurrenceInterval)
		if err == nil {
			return d.Shift(current)
		}
	case sf.ON_REQUEST_MRDV133MRDT:
		return time.Unix(0, 0)
	}

	return current.Add(DefaultReportDuration)
}

func (metric *metric) record(vals []MetricReportValue) {
	definition := metric.reportDefinition
	report := &metric.report

	// When a metric report is updated - overwrite the entire report
	if definition.ReportUpdates == sf.OVERWRITE_MRDV133RUE {
		report.MetricValues = vals
		return
	}

	// The metric report is generated when the metric values change. Record the changed
	// values, and capture any new values that need to be appended
	if definition.MetricReportDefinitionType == sf.ON_CHANGE_MRDV133MRDT {
		vals = metric.recordUnchanged(vals)
	}

	remaining := int(definition.AppendLimit) - len(report.MetricValues)
	switch definition.ReportUpdates {
	case sf.OVERWRITE_MRDV133RUE:
		panic("this case should be handled prior on-change events are recorded")
	case sf.APPEND_STOPS_WHEN_FULL_MRDV133RUE:

		if remaining <= 0 {
			// Full - do nothing
		} else if len(vals) < remaining {
			metric.append(vals[:remaining])
		} else {
			metric.append(vals)
		}

	case sf.APPEND_WRAPS_WHEN_FULL_MRDV133RUE:

		if len(vals) < int(definition.AppendLimit) { // special case where we're capturing more than the allowed limit
			report.MetricValues = vals[:definition.AppendLimit]
		} else if remaining >= len(vals) {
			metric.append(vals)
		} else if remaining < 0 {
			report.MetricValues = report.MetricValues[:len(report.MetricValues)+remaining]
			metric.append(vals)
		}

	case sf.NEW_REPORT_MRDV133RUE:
		// TODO: I'm not sure we're going to want to do this
	default:
		return
	}
}

// Record any unchanged metrics in the supplied values and return the list of report values
// that have changed since the last
func (metric *metric) recordUnchanged(newVals []MetricReportValue) []MetricReportValue {

	changedVals := make([]MetricReportValue, 0)
	for _, newVal := range newVals {
		valChanged := true
		for curIdx := range metric.report.MetricValues {
			curVal := metric.report.MetricValues[len(metric.report.MetricValues)-curIdx-1]
			if curVal.MetricProperty == newVal.MetricProperty {
				if curVal.MetricValue == newVal.MetricProperty {
					valChanged = false
				}
			}
		}

		if valChanged {
			changedVals = append(changedVals, newVal)
		}
	}

	return changedVals
}

func (metric *metric) append(vals []MetricReportValue) {
	metric.report.MetricValues = append(metric.report.MetricValues, vals...)
}

func Get(model *sf.TelemetryServiceV121TelemetryService) error {

	model.MetricDefinitions = sf.OdataV4IdRef{OdataId: "/redfish/v1/TelemetryService/MetricDefinitions"}
	model.MetricReportDefinitions = sf.OdataV4IdRef{OdataId: "/redfish/v1/TelemetryService/MetricReportDefinitions"}
	model.MetricReports = sf.OdataV4IdRef{OdataId: "/redfish/v1/TelemetryService/MetricReports"}

	return nil
}

func MetricDefinitionsGet(model *sf.MetricDefinitionCollectionMetricDefinitionCollection) error {

	model.MembersodataCount = int64(len(TelemetryManager.metrics))
	model.Members = make([]sf.OdataV4IdRef, model.MembersodataCount)

	for idx := range TelemetryManager.metrics {
		def := TelemetryManager.metrics[idx].definition
		model.Members[idx] = sf.OdataV4IdRef{OdataId: def.OdataId}
	}

	return nil
}

func MetricDefinitionIdGet(model *sf.MetricDefinitionV110MetricDefinition, id string) error {

	m := findMetric(id)
	if m == nil {
		return ec.NewErrNotFound()
	}

	*model = *m.definition

	return nil
}

func MetricReportDefinitionsGet(model *sf.MetricReportDefinitionCollectionMetricReportDefinitionCollection) error {

	model.MembersodataCount = int64(len(TelemetryManager.metrics))
	model.Members = make([]sf.OdataV4IdRef, model.MembersodataCount)

	for idx := range TelemetryManager.metrics {
		def := TelemetryManager.metrics[idx].reportDefinition
		model.Members[idx] = sf.OdataV4IdRef{OdataId: def.OdataId}
	}

	return nil
}

func MetricReportDefinitionIdGet(model *sf.MetricReportDefinitionV133MetricReportDefinition, id string) error {
	m := findMetric(id)
	if m == nil {
		return ec.NewErrNotFound()
	}

	*model = *m.reportDefinition

	return nil
}

func MetricReportsGet(model *sf.MetricReportCollectionMetricReportCollection) error {

	model.MembersodataCount = int64(len(TelemetryManager.metrics))
	model.Members = make([]sf.OdataV4IdRef, model.MembersodataCount)

	for idx := range TelemetryManager.metrics {
		rep := &TelemetryManager.metrics[idx].report
		model.Members[idx] = sf.OdataV4IdRef{OdataId: rep.OdataId}
	}

	return nil
}

func MetricReportIdGet(model *sf.MetricReportV140MetricReport, id string) error {
	m := findMetric(id)
	if m == nil {
		return ec.NewErrNotFound()
	}

	if m.reportDefinition.MetricReportDefinitionType == sf.ON_REQUEST_MRDV133MRDT {
		values, err := m.generator(m.reportDefinition)
		if err != nil {
			return ec.NewErrInternalServerError().WithError(err).WithCause("Failed to generate metric report")
		}

		m.record(values)
	}

	*model = m.report

	return nil
}
