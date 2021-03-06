/*
 * Swordfish API
 *
 * This contains the definition of the Swordfish extensions to a Redfish service.
 *
 * API version: v1.2.c
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi

type JobV105JobState string

// List of Job_v1_0_5_JobState
const (
	NEW_JV105JST JobV105JobState = "New"
	STARTING_JV105JST JobV105JobState = "Starting"
	RUNNING_JV105JST JobV105JobState = "Running"
	SUSPENDED_JV105JST JobV105JobState = "Suspended"
	INTERRUPTED_JV105JST JobV105JobState = "Interrupted"
	PENDING_JV105JST JobV105JobState = "Pending"
	STOPPING_JV105JST JobV105JobState = "Stopping"
	COMPLETED_JV105JST JobV105JobState = "Completed"
	CANCELLED_JV105JST JobV105JobState = "Cancelled"
	EXCEPTION_JV105JST JobV105JobState = "Exception"
	SERVICE_JV105JST JobV105JobState = "Service"
	USER_INTERVENTION_JV105JST JobV105JobState = "UserIntervention"
	CONTINUE_JV105JST JobV105JobState = "Continue"
)
