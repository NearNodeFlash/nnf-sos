/*
 * Swordfish API
 *
 * This contains the definition of the Swordfish extensions to a Redfish service.
 *
 * API version: v1.2.c
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi

// EventDestinationV190SyslogFilter - A syslog filter.
type EventDestinationV190SyslogFilter struct {

	// The types of programs that can log messages.
	LogFacilities []EventDestinationV190SyslogFacility `json:"LogFacilities,omitempty"`

	LowestSeverity EventDestinationV190SyslogSeverity `json:"LowestSeverity,omitempty"`
}
