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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

const (
	// The required namespace for an NNF Data Movement operation. This is for system wide (lustre) data movement.
	// Individual nodes may also perform data movement in which case they use the NNF Node Name as the namespace.
	DataMovementNamespace = "nnf-dm-system"
)

// NnfDataMovementSpec defines the desired state of DataMovement
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
}

// DataMovementSpecSourceDestination defines the desired source or destination of data movement
type NnfDataMovementSpecSourceDestination struct {

	// Path describes the location of the user data relative to the storage instance
	Path string `json:"path,omitempty"`

	// Storage describes the storage backing this data movement specification; Storage can reference
	// either NNF storage or global Lustre storage depending on the object references Kind field.
	StorageReference corev1.ObjectReference `json:"storageReference,omitempty"`
}

// DataMovementStatus defines the observed state of DataMovement
type NnfDataMovementStatus struct {
	// Current state of data movement.
	// +kubebuilder:validation:Enum=Starting;Running;Finished
	State string `json:"state,omitempty"`

	// Status of the current state.
	// +kubebuilder:validation:Enum=Success;Failed;Invalid;Cancelled
	Status string `json:"status,omitempty"`

	// Message contains any text that explains the Status.
	Message string `json:"message,omitempty"`

	// StartTime reflects the time at which the Data Movement operation started.
	StartTime *metav1.MicroTime `json:"startTime,omitempty"`

	// EndTime reflects the time at which the Data Movement operation ended.
	EndTime *metav1.MicroTime `json:"endTime,omitempty"`
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
//+kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"

// NnfDataMovement is the Schema for the datamovements API
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
	// what workflow state the resource is no longer need and can be safely deleted. The finish logic filters objects
	// by the current workflow state to only delete the correct resources.
	DataMovementTeardownStateLabel = "nnf.cray.hpe.com/teardown_state"
)

func AddDataMovementTeardownStateLabel(object metav1.Object, state string) {
	labels := object.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}

	labels[DataMovementTeardownStateLabel] = state
	object.SetLabels(labels)
}

func init() {
	SchemeBuilder.Register(&NnfDataMovement{}, &NnfDataMovementList{})
}
