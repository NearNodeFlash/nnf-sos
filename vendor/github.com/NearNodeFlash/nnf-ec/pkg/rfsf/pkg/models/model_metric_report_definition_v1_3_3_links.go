/*
 * Swordfish API
 *
 * This contains the definition of the Swordfish extensions to a Redfish service.
 *
 * API version: v1.2.c
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi

// MetricReportDefinitionV133Links - The links to other resources that are related to this resource.
type MetricReportDefinitionV133Links struct {

	// The OEM extension.
	Oem map[string]interface{} `json:"Oem,omitempty"`

	// The triggers that cause this metric report definition to generate a new metric report upon a trigger occurrence when the TriggerActions property contains `RedfishMetricReport`.
	Triggers []OdataV4IdRef `json:"Triggers,omitempty"`

	// The number of items in a collection.
	TriggersodataCount int64 `json:"Triggers@odata.count,omitempty"`
}
