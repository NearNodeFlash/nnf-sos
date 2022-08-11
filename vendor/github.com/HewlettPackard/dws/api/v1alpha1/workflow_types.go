/*
 * Copyright 2021, 2022 Hewlett Packard Enterprise Development LP
 * Other additional copyright holders may be indicated within.
 *
 * The entirety of this work is licensed under the Apache License,
 * Version 2.0 (the "License"); you may not use this file except
 * in compliance with the License.
 *
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package v1alpha1

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// WorkflowNameLabel is defined for resources that relate to the name of a DWS Workflow
	WorkflowNameLabel = "dws.cray.hpe.com/workflow.name"

	// WorkflowNamespaceLabel is defined for resources that relate to the namespace of a DWS Workflow
	WorkflowNamespaceLabel = "dws.cray.hpe.com/workflow.namespace"
)

// WorkflowState is the enumeration of The state of the workflow
type WorkflowState int

// State enumerations
const (
	StateProposal WorkflowState = iota
	StateSetup
	StateDataIn
	StatePreRun
	StatePostRun
	StateDataOut
	StateTeardown
)

var workflowStrings = [...]string{
	"proposal",
	"setup",
	"data_in",
	"pre_run",
	"post_run",
	"data_out",
	"teardown",
}

const (
	StatusPending    = "Pending"
	StatusQueued     = "Queued"
	StatusRunning    = "Running"
	StatusCompleted  = "Completed"
	StatusError      = "Error"
	StatusDriverWait = "DriverWait"
)

func (s WorkflowState) String() string {
	return workflowStrings[s]
}

// GetWorkflowState returns the WorkflowState constant that matches the
// string value passed in
func GetWorkflowState(state string) (WorkflowState, error) {
	for i := StateProposal; i <= StateTeardown; i++ {
		if i.String() == state {
			return i, nil
		}
	}

	return StateProposal, fmt.Errorf("invalid workflow state '%s'", state)
}

// WorkflowSpec defines the desired state of Workflow
type WorkflowSpec struct {
	// Desired state for the workflow to be in. Unless progressing to the teardown state,
	// this can only be set to the next state when the current desired state has been achieved.
	// +kubebuilder:validation:Enum=proposal;setup;data_in;pre_run;post_run;data_out;teardown
	DesiredState string `json:"desiredState"`
	WLMID        string `json:"wlmID"`
	JobID        int    `json:"jobID"`

	// UserID specifies the user ID for the workflow. The User ID is used by the various states
	// in the workflow to ensure the user has permissions to perform certain actions. Used in
	// conjunction with Group ID to run subtasks with UserID:GroupID credentials
	UserID uint32 `json:"userID"`

	// GroupID specifies the group ID for the workflow. The Group ID is used by the various states
	// in the workflow to ensure the group has permissions to perform certain actions. Used in
	// conjunction with User ID to run subtasks with UserID:GroupID credentials.
	GroupID uint32 `json:"groupID"`

	// List of #DW strings from a WLM job script
	DWDirectives []string `json:"dwDirectives"`
}

// WorkflowDriverStatus defines the status information provided by integration drivers.
type WorkflowDriverStatus struct {
	DriverID   string `json:"driverID"`
	TaskID     string `json:"taskID"`
	DWDIndex   int    `json:"dwdIndex"`
	WatchState string `json:"watchState"`
	LastHB     int64  `json:"lastHB"`
	Completed  bool   `json:"completed"`

	// User readable reason.
	// For the CDS driver, this could be the state of the underlying
	// data movement request:  Pending, Queued, Running, Completed or Error
	// +kubebuilder:validation:Enum=Pending;Queued;Running;Completed;Error
	Status string `json:"status,omitempty"`

	Message string `json:"message,omitempty"`

	// Driver error string. This is not rolled up into the workflow's
	// overall status section
	Error string `json:"error,omitempty"`

	// CompleteTime reflects the time that the workflow reconciler marks the driver complete
	CompleteTime *metav1.MicroTime `json:"completeTime,omitempty"`
}

// WorkflowStatus defines the observed state of the Workflow
type WorkflowStatus struct {
	// The state the resource is currently transitioning to.
	// Updated by the controller once started.
	// +kubebuilder:default=proposal
	State string `json:"state"`

	// Ready can be 'True', 'False'
	// Indicates whether State has been reached.
	Ready bool `json:"ready"`

	// User readable reason and status message
	// +kubebuilder:validation:Enum=Completed;DriverWait;Error
	Status  string `json:"status,omitempty"`
	Message string `json:"message,omitempty"`

	// Set of DW environment variable settings for WLM to apply to the job.
	//		- DW_JOB_STRIPED
	//		- DW_JOB_PRIVATE
	//		- DW_JOB_STRIPED_CACHE
	//		- DW_JOB_LDBAL_CACHE
	//		- DW_PERSISTENT_STRIPED_{resname}
	Env map[string]string `json:"env,omitempty"`

	// List of registered drivers and related status.  Updated by drivers.
	Drivers []WorkflowDriverStatus `json:"drivers,omitempty"`

	// List of #DW directive breakdowns indicating to WLM what to allocate on what Server
	// 1 DirectiveBreakdown per #DW Directive that requires storage
	DirectiveBreakdowns []corev1.ObjectReference `json:"directiveBreakdowns,omitempty"`

	// Reference to Computes
	Computes corev1.ObjectReference `json:"computes,omitempty"`

	// Time of the most recent desiredState change
	DesiredStateChange *metav1.MicroTime `json:"desiredStateChange,omitempty"`

	// Time of the most recent desiredState's achieving Ready status
	ReadyChange *metav1.MicroTime `json:"readyChange,omitempty"`

	// Duration of the last state change
	ElapsedTimeLastState string `json:"elapsedTimeLastState,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:printcolumn:name="STATE",type="string",JSONPath=".status.state",description="Current state"
//+kubebuilder:printcolumn:name="READY",type="boolean",JSONPath=".status.ready",description="True if current state is achieved"
//+kubebuilder:printcolumn:name="STATUS",type="string",JSONPath=".status.status",description="Indicates achievement of current state"
//+kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
//+kubebuilder:printcolumn:name="JOBID",type="integer",JSONPath=".spec.jobID",description="Job ID",priority=1
//+kubebuilder:printcolumn:name="DESIREDSTATE",type="string",JSONPath=".spec.desiredState",description="Desired state",priority=1
//+kubebuilder:printcolumn:name="DESIREDSTATECHANGE",type="date",JSONPath=".status.desiredStateChange",description="Time of most recent desiredState change",priority=1
//+kubebuilder:printcolumn:name="ELAPSEDTIMELASTSTATE",type="string",JSONPath=".status.elapsedTimeLastState",description="Duration of last state change",priority=1

// Workflow is the Schema for the workflows API
type Workflow struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WorkflowSpec   `json:"spec,omitempty"`
	Status WorkflowStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// WorkflowList contains a list of Workflows
type WorkflowList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Workflow `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Workflow{}, &WorkflowList{})
}
