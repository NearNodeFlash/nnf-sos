/*
 * Copyright 2021-2023 Hewlett Packard Enterprise Development LP
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

package v1alpha2

import (
	"fmt"
	"strings"

	"github.com/DataWorkflowServices/dws/utils/updater"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	// WorkflowNameLabel is defined for resources that relate to the name of a DWS Workflow
	WorkflowNameLabel = "dataworkflowservices.github.io/workflow.name"

	// WorkflowNamespaceLabel is defined for resources that relate to the namespace of a DWS Workflow
	WorkflowNamespaceLabel = "dataworkflowservices.github.io/workflow.namespace"

	// WorkflowUIDLabel holds the UID of the parent workflow resource
	WorkflowUidLabel = "dataworkflowservices.github.io/workflow.uid"
)

// WorkflowState is the enumeration of the state of the workflow
// +kubebuilder:validation:Enum:=Proposal;Setup;DataIn;PreRun;PostRun;DataOut;Teardown
type WorkflowState string

// WorkflowState values
const (
	StateProposal WorkflowState = "Proposal"
	StateSetup    WorkflowState = "Setup"
	StateDataIn   WorkflowState = "DataIn"
	StatePreRun   WorkflowState = "PreRun"
	StatePostRun  WorkflowState = "PostRun"
	StateDataOut  WorkflowState = "DataOut"
	StateTeardown WorkflowState = "Teardown"
)

// Next reports the next state after state s
func (s WorkflowState) next() WorkflowState {
	switch s {
	case "":
		return StateProposal
	case StateProposal:
		return StateSetup
	case StateSetup:
		return StateDataIn
	case StateDataIn:
		return StatePreRun
	case StatePreRun:
		return StatePostRun
	case StatePostRun:
		return StateDataOut
	case StateDataOut:
		return StateTeardown
	}

	panic(s)
}

// Last reports whether the state s is the last state
func (s WorkflowState) last() bool {
	return s == StateTeardown
}

// After reports whether the state s is after t
func (s WorkflowState) after(t WorkflowState) bool {

	for !t.last() {
		next := t.next()
		if s == next {
			return true
		}
		t = next
	}

	return false
}

// Strings associated with workflow statuses
const (
	StatusPending            = "Pending"
	StatusQueued             = "Queued"
	StatusRunning            = "Running"
	StatusCompleted          = "Completed"
	StatusTransientCondition = "TransientCondition"
	StatusError              = "Error"
	StatusDriverWait         = "DriverWait"
)

// ToStatus will return a Status* string that goes with
// the given severity.
func (severity ResourceErrorSeverity) ToStatus() (string, error) {
	switch severity {
	case SeverityMinor:
		return StatusRunning, nil
	case SeverityMajor:
		return StatusTransientCondition, nil
	case SeverityFatal:
		return StatusError, nil
	default:
		return "", fmt.Errorf("unknown severity: %s", string(severity))
	}
}

// SeverityStringToStatus will return a Status* string that goes with
// the given severity.
// An empty severity string will be considered a minor severity.
func SeverityStringToStatus(severity string) (string, error) {
	switch strings.ToLower(severity) {
	case "", "minor":
		return SeverityMinor.ToStatus()
	case "major":
		return SeverityMajor.ToStatus()
	case "fatal":
		return SeverityFatal.ToStatus()
	default:
		return "", fmt.Errorf("unknown severity: %s", severity)
	}
}

// WorkflowSpec defines the desired state of Workflow
type WorkflowSpec struct {
	// Desired state for the workflow to be in. Unless progressing to the teardown state,
	// this can only be set to the next state when the current desired state has been achieved.
	DesiredState WorkflowState `json:"desiredState"`

	// WLMID identifies the Workflow Manager (WLM), and is set by the WLM
	// when it creates the workflow resource.
	WLMID string `json:"wlmID"`

	// JobID is the WLM job ID that corresponds to this workflow, and is
	// set by the WLM when it creates the workflow resource.
	JobID intstr.IntOrString `json:"jobID"`

	// UserID specifies the user ID for the workflow. The User ID is used by the various states
	// in the workflow to ensure the user has permissions to perform certain actions. Used in
	// conjunction with Group ID to run subtasks with UserID:GroupID credentials
	UserID uint32 `json:"userID"`

	// GroupID specifies the group ID for the workflow. The Group ID is used by the various states
	// in the workflow to ensure the group has permissions to perform certain actions. Used in
	// conjunction with User ID to run subtasks with UserID:GroupID credentials.
	GroupID uint32 `json:"groupID"`

	// Hurry indicates that the workflow's driver should kill the job in a hurry when this workflow enters its teardown state.
	// The driver must release all resources and unmount any filesystems that were mounted as part of the workflow, though some drivers would have done this anyway as part of their teardown state.
	// The driver must also kill any in-progress data transfers, or skip any data transfers that have not yet begun.
	// +kubebuilder:default:=false
	Hurry bool `json:"hurry,omitempty"`

	// List of #DW strings from a WLM job script
	DWDirectives []string `json:"dwDirectives"`
}

// WorkflowDriverStatus defines the status information provided by integration drivers.
type WorkflowDriverStatus struct {
	DriverID string `json:"driverID"`
	TaskID   string `json:"taskID"`
	DWDIndex int    `json:"dwdIndex"`

	WatchState WorkflowState `json:"watchState"`

	LastHB    int64 `json:"lastHB"`
	Completed bool  `json:"completed"`

	// User readable reason.
	// For the CDS driver, this could be the state of the underlying
	// data movement request
	// +kubebuilder:validation:Enum=Pending;Queued;Running;Completed;TransientCondition;Error;DriverWait
	Status string `json:"status,omitempty"`

	// Message provides additional details on the current status of the resource
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
	State WorkflowState `json:"state,omitempty"`

	// Ready can be 'True', 'False'
	// Indicates whether State has been reached.
	Ready bool `json:"ready"`

	// User readable reason and status message.
	// - Completed: The workflow has reached the state in workflow.Status.State.
	// - DriverWait: The underlying drivers are currently running.
	// - TransientCondition: A driver has encountered an error that might be recoverable.
	// - Error: A driver has encountered an error that will not recover.
	// +kubebuilder:validation:Enum=Completed;DriverWait;TransientCondition;Error
	Status string `json:"status,omitempty"`

	// Message provides additional details on the current status of the resource
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
//+kubebuilder:storageversion
//+kubebuilder:printcolumn:name="STATE",type="string",JSONPath=".status.state",description="Current state"
//+kubebuilder:printcolumn:name="READY",type="boolean",JSONPath=".status.ready",description="True if current state is achieved"
//+kubebuilder:printcolumn:name="STATUS",type="string",JSONPath=".status.status",description="Indicates achievement of current state"
//+kubebuilder:printcolumn:name="JOBID",type="string",JSONPath=".spec.jobID",description="Job ID"
//+kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
//+kubebuilder:printcolumn:name="DESIREDSTATE",type="string",JSONPath=".spec.desiredState",description="Desired state",priority=1
//+kubebuilder:printcolumn:name="DESIREDSTATECHANGE",type="date",JSONPath=".status.desiredStateChange",description="Time of most recent desiredState change",priority=1
//+kubebuilder:printcolumn:name="ELAPSEDTIMELASTSTATE",type="string",JSONPath=".status.elapsedTimeLastState",description="Duration of last state change",priority=1
//+kubebuilder:printcolumn:name="UID",type="string",JSONPath=".spec.userID",description="UID",priority=1
//+kubebuilder:printcolumn:name="GID",type="string",JSONPath=".spec.groupID",description="GID",priority=1

// Workflow is the Schema for the workflows API
type Workflow struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WorkflowSpec   `json:"spec,omitempty"`
	Status WorkflowStatus `json:"status,omitempty"`
}

func (c *Workflow) GetStatus() updater.Status[*WorkflowStatus] {
	return &c.Status
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
