/*
 * Swordfish API
 *
 * This contains the definition of the Swordfish extensions to a Redfish service.
 *
 * API version: v1.2.c
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi
// MetricDefinitionV110Calculable : The types of calculations that can be applied to the metric reading.  Provides information to the client on the suitability of calculation using the metric reading.
type MetricDefinitionV110Calculable string

// List of MetricDefinition_v1_1_0_Calculable
const (
	NON_CALCULATABLE_MDV110C MetricDefinitionV110Calculable = "NonCalculatable"
	SUMMABLE_MDV110C MetricDefinitionV110Calculable = "Summable"
	NON_SUMMABLE_MDV110C MetricDefinitionV110Calculable = "NonSummable"
)