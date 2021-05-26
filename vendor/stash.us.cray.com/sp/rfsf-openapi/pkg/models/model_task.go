/* -----------------------------------------------------------------
 * model_task.go
 *
 * Defines the Redfish Task resource as a Golang model.
 *
 * Author: Caleb Carlson
 *
 * © Copyright 2020 Hewlett Packard Enterprise Development LP
 *
 * ----------------------------------------------------------------- */

package openapi

import (
	"time"
)

// Task - This resource contains information about a specific Task scheduled by or being executed by a Redfish service's Task Service.
type Task struct {
	OdataContext string `json:"@odata.context,omitempty"`

	OdataEtag string `json:"@odata.etag,omitempty"`

	OdataId string `json:"@odata.id"`

	OdataType string `json:"@odata.type"`

	//Actions - The available actions for this Resource.
	Actions map[string]interface{} `json:"Actions,omitempty"`

	//Description - Human-readable description of the Task resource.
	Description string `json:"Description,omitempty"`

	//EndTime - The date-time stamp that the task was last completed.
	EndTime time.Time `json:"EndTime,omitempty"`

	//HidePayload - Indicates that the contents of the Payload should be hidden from view after the Task has been created.  When set to True, the Payload object will not be returned on GET.
	HidePayload bool `json:"HidePayload,omitempty"`

	//Id - Unique ID of the Task resource.
	Id string `json:"Id"`

	//Messages - This is an array of messages associated with the task.
	Messages []Message `json:"Messages,omitempty"`

	//Name - The name of the Task resource.
	Name string `json:"Name"`

	//Oem - The OEM extension property, as defined by Cray.
	Oem map[string]interface{} `json:"Oem,omitempty"`

	//Payload - The HTTP and JSON payload details for this task, unless they are hidden from view by the service.
	Payload map[string]interface{} `json:"Payload,omitempty"`

	//PercentComplete - The completion percentage of this Task.
	PercentComplete int32 `json:"PercentComplete,omitempty"`

	//StartTime - The date-time stamp that the task was last started.
	StartTime time.Time `json:"StartTime,omitempty"`

	//TaskMonitor - The URI of the Task Monitor for this task.
	TaskMonitor string `json:"TaskMonitor,omitempty"`

	//TaskState - The state of the task.
	TaskState TaskState `json:"TaskState,omitempty"`

	//TaskStatus - The completion status of the task.
	TaskStatus string `json:"TaskStatus,omitempty"`
}
