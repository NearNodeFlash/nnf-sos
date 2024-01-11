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

package v1alpha1

import (
	dwsv1alpha2 "github.com/DataWorkflowServices/dws/api/v1alpha2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// The required namespace for an NNF Data Movement operation. This is for system wide (lustre)
	// data movement.  Individual nodes may also perform data movement in which case they use the
	// NNF Node Name as the namespace.
	DataMovementNamespace = "nnf-dm-system"

	// The name of the default profile stored in the nnf-dm-config ConfigMap that is used to
	// configure Data Movement.
	DataMovementProfileDefault = "default"
)

// NnfDataMovementSpec defines the desired state of NnfDataMovement
type NnfDataMovementSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Source describes the source of the data movement operation
	Source *NnfDataMovementSpecSourceDestination `json:"source,omitempty"`

	// Destination describes the destination of the data movement operation
	Destination *NnfDataMovementSpecSourceDestination `json:"destination,omitempty"`

	// User Id specifies the user ID for the data movement operation. This value is used
	// in conjunction with the group ID to ensure the user has valid permissions to perform
	// the data movement operation.
	UserId uint32 `json:"userId,omitempty"`

	// Group Id specifies the group ID for the data movement operation. This value is used
	// in conjunction with the user ID to ensure the user has valid permissions to perform
	// the data movement operation.
	GroupId uint32 `json:"groupId,omitempty"`

	// Set to true if the data movement operation should be canceled.
	// +kubebuilder:default:=false
	Cancel bool `json:"cancel,omitempty"`

	// Profile specifies the name of profile in the nnf-dm-config ConfigMap to be used for
	// configuring data movement. Defaults to the default profile.
	// +kubebuilder:default:=default
	Profile string `json:"profile,omitempty"`

	// User defined configuration on how data movement should be performed. This overrides the
	// configuration defined in the nnf-dm-config ConfigMap. These values are typically set by the
	// Copy Offload API.
	UserConfig *NnfDataMovementConfig `json:"userConfig,omitempty"`
}

// NnfDataMovementSpecSourceDestination defines the desired source or destination of data movement
type NnfDataMovementSpecSourceDestination struct {

	// Path describes the location of the user data relative to the storage instance
	Path string `json:"path,omitempty"`

	// Storage describes the storage backing this data movement specification; Storage can reference
	// either NNF storage or global Lustre storage depending on the object references Kind field.
	StorageReference corev1.ObjectReference `json:"storageReference,omitempty"`
}

// NnfDataMovementConfig provides a way for a user to override the data movement behavior on a
// per DM basis.
type NnfDataMovementConfig struct {

	// Fake the Data Movement operation. The system "performs" Data Movement but the command to do so
	// is trivial. This means a Data Movement request is still submitted but the IO is skipped.
	// +kubebuilder:default:=false
	Dryrun bool `json:"dryrun,omitempty"`

	// Extra options to pass to the dcp command (used to perform data movement).
	DCPOptions string `json:"dcpOptions,omitempty"`

	// If true, enable the command's stdout to be saved in the log when the command completes
	// successfully. On failure, the output is always logged.
	// Note: Enabling this option may degrade performance.
	// +kubebuilder:default:=false
	LogStdout bool `json:"logStdout,omitempty"`

	// Similar to LogStdout, store the command's stdout in Status.Message when the command completes
	// successfully. On failure, the output is always stored.
	// Note: Enabling this option may degrade performance.
	// +kubebuilder:default:=false
	StoreStdout bool `json:"storeStdout,omitempty"`

	// The number of slots specified in the MPI hostfile. A value of 0 disables the use of slots in
	// the hostfile. Nil will defer to the value specified in the nnf-dm-config ConfigMap.
	Slots *int `json:"slots,omitempty"`

	// The number of max_slots specified in the MPI hostfile. A value of 0 disables the use of slots
	// in the hostfile. Nil will defer to the value specified in the nnf-dm-config ConfigMap.
	MaxSlots *int `json:"maxSlots,omitempty"`
}

// NnfDataMovementCommandStatus defines the observed status of the underlying data movement
// command (MPI File Utils' `dcp` command).
type NnfDataMovementCommandStatus struct {
	// The command that was executed during data movement.
	Command string `json:"command,omitempty"`

	// ElapsedTime reflects the elapsed time since the underlying data movement command started.
	ElapsedTime metav1.Duration `json:"elapsedTime,omitempty"`

	// Progress refects the progress of the underlying data movement command as captured from standard output.
	// A best effort is made to parse the command output as a percentage. If no progress has
	// yet to be measured than this field is omitted. If the latest command output does not
	// contain a valid percentage, then the value is unchanged from the previously parsed value.
	ProgressPercentage *int32 `json:"progress,omitempty"`

	// LastMessage reflects the last message received over standard output or standard error as
	// captured by the underlying data movement command.
	LastMessage string `json:"lastMessage,omitempty"`

	// LastMessageTime reflects the time at which the last message was received over standard output or
	// standard error by the underlying data movement command.
	LastMessageTime metav1.MicroTime `json:"lastMessageTime,omitempty"`
}

// NnfDataMovementStatus defines the observed state of NnfDataMovement
type NnfDataMovementStatus struct {
	// Current state of data movement.
	// +kubebuilder:validation:Enum=Starting;Running;Finished
	State string `json:"state,omitempty"`

	// Status of the current state.
	// +kubebuilder:validation:Enum=Success;Failed;Invalid;Cancelled
	Status string `json:"status,omitempty"`

	// Message contains any text that explains the Status. If Data Movement failed or storeStdout is
	// enabled, this will contain the command's output.
	Message string `json:"message,omitempty"`

	// StartTime reflects the time at which the Data Movement operation started.
	StartTime *metav1.MicroTime `json:"startTime,omitempty"`

	// EndTime reflects the time at which the Data Movement operation ended.
	EndTime *metav1.MicroTime `json:"endTime,omitempty"`

	// Restarts contains the number of restarts of the Data Movement operation.
	Restarts int `json:"restarts,omitempty"`

	// CommandStatus reflects the current status of the underlying Data Movement command
	// as it executes. The command status is polled at a certain frequency to avoid excessive
	// updates to the Data Movement resource.
	CommandStatus *NnfDataMovementCommandStatus `json:"commandStatus,omitempty"`

	dwsv1alpha2.ResourceError `json:",inline"`
}

// Types describing the various data movement status conditions.
const (
	DataMovementConditionTypeStarting = "Starting"
	DataMovementConditionTypeRunning  = "Running"
	DataMovementConditionTypeFinished = "Finished"
)

// Reasons describing the various data movement status conditions. Must be
// in CamelCase format (see metav1.Condition)
const (
	DataMovementConditionReasonSuccess   = "Success"
	DataMovementConditionReasonFailed    = "Failed"
	DataMovementConditionReasonInvalid   = "Invalid"
	DataMovementConditionReasonCancelled = "Cancelled"
)

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="STATE",type="string",JSONPath=".status.state",description="Current state"
//+kubebuilder:printcolumn:name="STATUS",type="string",JSONPath=".status.status",description="Status of current state"
//+kubebuilder:printcolumn:name="ERROR",type="string",JSONPath=".status.error.severity"
//+kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"

// NnfDataMovement is the Schema for the nnfdatamovements API
type NnfDataMovement struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NnfDataMovementSpec   `json:"spec,omitempty"`
	Status NnfDataMovementStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// NnfDataMovementList contains a list of NnfDataMovement
type NnfDataMovementList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NnfDataMovement `json:"items"`
}

func (n *NnfDataMovementList) GetObjectList() []client.Object {
	objectList := []client.Object{}

	for i := range n.Items {
		objectList = append(objectList, &n.Items[i])
	}

	return objectList
}

const (
	// DataMovementTeardownStateLabel is the label applied to Data Movement and related resources that describes
	// the workflow state when the resource is no longer need and can be safely deleted.
	DataMovementTeardownStateLabel = "nnf.cray.hpe.com/teardown_state"
)

func AddDataMovementTeardownStateLabel(object metav1.Object, state dwsv1alpha2.WorkflowState) {
	labels := object.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}

	labels[DataMovementTeardownStateLabel] = string(state)
	object.SetLabels(labels)
}

func init() {
	SchemeBuilder.Register(&NnfDataMovement{}, &NnfDataMovementList{})
}
