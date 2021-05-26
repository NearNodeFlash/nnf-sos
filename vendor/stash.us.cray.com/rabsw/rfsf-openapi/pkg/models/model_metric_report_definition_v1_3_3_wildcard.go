/*
 * Swordfish API
 *
 * This contains the definition of the Swordfish extensions to a Redfish service.
 *
 * API version: v1.2.c
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi

// MetricReportDefinitionV133Wildcard - The wildcard and its substitution values.
type MetricReportDefinitionV133Wildcard struct {

	// An array of values to substitute for the wildcard.
	Keys []string `json:"Keys,omitempty"`

	// The string used as a wildcard.
	Name string `json:"Name,omitempty"`

	// An array of values to substitute for the wildcard.
	Values []string `json:"Values,omitempty"`
}
