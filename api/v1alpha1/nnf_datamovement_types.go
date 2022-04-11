/*
Copyright 2021 Hewlett Packard Enterprise Development LP
*/

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// NnfDataMovementSpec defines the desired state of DataMovement
type NnfDataMovementSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Source describes the source of the data movement operation
	Source NnfDataMovementSpecSourceDestination `json:"source,omitempty"`

	// Destination describes the destination of the data movement operation
	Destination NnfDataMovementSpecSourceDestination `json:"destination,omitempty"`

	// User Id specifies the user ID for the data movement operation. This value is used
	// in conjunction with the group ID to ensure the user has valid permissions to perform
	// the data movement operation.
	UserId uint32 `json:"userId,omitempty"`

	// Group Id specifies the group ID for the data movement operation. This value is used
	// in conjunction with the user ID to ensure the user has valid permissions to perform
	// the data movement operation.
	GroupId uint32 `json:"groupId,omitempty"`
}

// DataMovementSpecSourceDestination defines the desired source or destination of data movement
type NnfDataMovementSpecSourceDestination struct {

	// Path describes the location of the user data relative to the storage instance
	Path string `json:"path,omitempty"`

	// Storage describes the storage backing this data movement specification; Storage can reference
	// either NNF storage or global Lustre storage depending on the object references Kind field.
	Storage *corev1.ObjectReference `json:"storageInstance,omitempty"`

	// Access references the NNF Access element that is needed to perform data movement. This provides
	// details as to the mount path and backing storage across NNF Nodes.
	Access *corev1.ObjectReference `json:"access,omitempty"`
}

// DataMovementStatus defines the observed state of DataMovement
type NnfDataMovementStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Job references the underlying job that performs data movement
	Job corev1.ObjectReference `json:"job,omitempty"`

	// Node Status reflects the status of individual NNF Node Data Movement operations
	NodeStatus []NnfDataMovementNodeStatus `json:"nodeStatus,omitempty"`

	// Conditions represents an array of conditions that refect the current
	// status of the data movement operation. Each condition type must be
	// one of Starting, Running, or Finished, reflect the three states that
	// data movement performs.
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// NnfDataMovementNodeStatus defines the observed state of a DataMovementNode
type NnfDataMovementNodeStatus struct {
	// Node is the node who's status this status element describes
	Node string `json:"node,omitempty"`

	// Count is the total number of resources managed by this node
	Count uint32 `json:"count,omitempty"`

	// Running is the number of resources running under this node
	Running uint32 `json:"running,omitemtpy"`

	// Complete is the number of resource completed by this node
	Complete uint32 `json:"complete,omitempty"`

	// Status is an array of status strings reported by resources in this node
	Status []string `json:"status,omitempty"`

	// Messages is an array of error messages reported by resources in this node
	Messages []string `json:"messages,omitempty"`
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
	DataMovementConditionReasonSuccess = "Success"
	DataMovementConditionReasonFailed  = "Failed"
	DataMovementConditionReasonInvalid = "Invalid"
)

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

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

func init() {
	SchemeBuilder.Register(&NnfDataMovement{}, &NnfDataMovementList{})
}
