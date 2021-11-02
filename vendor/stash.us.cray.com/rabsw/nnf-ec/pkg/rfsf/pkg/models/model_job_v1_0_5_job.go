/*
 * Swordfish API
 *
 * This contains the definition of the Swordfish extensions to a Redfish service.
 *
 * API version: v1.2.c
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi

import (
	"time"
)

// JobV105Job - The Job schema contains information about a job that a Redfish job service schedules or executes.  Clients create jobs to describe a series of operations that occur at periodic intervals.
type JobV105Job struct {

	// The OData description of a payload.
	OdataContext string `json:"@odata.context,omitempty"`

	// The current ETag of the resource.
	OdataEtag string `json:"@odata.etag,omitempty"`

	// The unique identifier for a resource.
	OdataId string `json:"@odata.id"`

	// The type of a resource.
	OdataType string `json:"@odata.type"`

	Actions JobV105Actions `json:"Actions,omitempty"`

	// The person or program that created this job entry.
	CreatedBy string `json:"CreatedBy,omitempty"`

	// The description of this resource.  Used for commonality in the schema definitions.
	Description string `json:"Description,omitempty"`

	// The date and time when the job was completed.
	EndTime time.Time `json:"EndTime,omitempty"`

	// An indication of whether the contents of the payload should be hidden from view after the job has been created.  If `true`, responses do not return the payload.  If `false`, responses return the payload.  If this property is not present when the job is created, the default is `false`.
	HidePayload bool `json:"HidePayload,omitempty"`

	// The identifier that uniquely identifies the resource within the collection of similar resources.
	Id string `json:"Id"`

	JobState JobV105JobState `json:"JobState,omitempty"`

	JobStatus ResourceHealth `json:"JobStatus,omitempty"`

	// The maximum amount of time the job is allowed to execute.
	MaxExecutionTime string `json:"MaxExecutionTime,omitempty"`

	// An array of messages associated with the job.
	Messages []MessageMessage `json:"Messages,omitempty"`

	// The name of the resource or array member.
	Name string `json:"Name"`

	// The OEM extension.
	Oem map[string]interface{} `json:"Oem,omitempty"`

	Payload JobV105Payload `json:"Payload,omitempty"`

	// The completion percentage of this job.
	PercentComplete int64 `json:"PercentComplete,omitempty"`

	Schedule ScheduleSchedule `json:"Schedule,omitempty"`

	// The date and time when the job was started or is scheduled to start.
	StartTime time.Time `json:"StartTime,omitempty"`

	// The serialized execution order of the job steps.
	StepOrder []string `json:"StepOrder,omitempty"`

	Steps OdataV4IdRef `json:"Steps,omitempty"`
}