/*
Copyright 2021 Hewlett Packard Enterprise Development LP
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
	UserID       int    `json:"userID"`

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
	Reason  string `json:"reason,omitempty"`
	Message string `json:"message,omitempty"`

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
	Reason  string `json:"reason,omitempty"`
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
