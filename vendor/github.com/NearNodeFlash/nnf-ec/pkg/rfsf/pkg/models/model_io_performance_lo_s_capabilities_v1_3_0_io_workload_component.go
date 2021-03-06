/*
 * Swordfish API
 *
 * This contains the definition of the Swordfish extensions to a Redfish service.
 *
 * API version: v1.2.c
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi

// IoPerformanceLoSCapabilitiesV130IoWorkloadComponent - Describe a component of a IO workload.
type IoPerformanceLoSCapabilitiesV130IoWorkloadComponent struct {

	// Average I/O Size for this component.
	AverageIOBytes int64 `json:"AverageIOBytes,omitempty"`

	// Duration that this component is active.
	Duration string `json:"Duration,omitempty"`

	IOAccessPattern IOPerformanceLoSCapabilitiesV130IOAccessPattern `json:"IOAccessPattern,omitempty"`

	// Percent of data for this workload component.
	PercentOfData int64 `json:"PercentOfData,omitempty"`

	// Percent of total IOPS for this workload component.
	PercentOfIOPS int64 `json:"PercentOfIOPS,omitempty"`

	Schedule ScheduleSchedule `json:"Schedule,omitempty"`
}
