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

// NnfStorageSpec defines the specification for requesting generic storage on a set
// of available NNF Nodes. This object is related to a #DW for NNF Storage, with the WLM
// making the determination for which NNF Nodes it wants to untilize.
type NnfStorageSpec struct {
	// Important: Run "make" to regenerate code after modifying this file

	// Uuid defines the unique identifier for this storage specification. Can
	// be any value specified by the user, but must be unique against all
	// existing storage specifications.
	Uuid string `json:"uuid,omitempty"`

	// Capacity defines the desired size, in bytes, of storage allocation for
	// each NNF Node listed by the storage specification. The allocated capacity
	// may be larger to meet underlying storage requirements (i.e. allocated
	// capacity may round up to a multiple of the stripe size)
	Capacity int64 `json:"capacity"`

	// Workflow reference records the workflow that created this NnfStorage
	WorkflowReference corev1.ObjectReference `json:"workflowReference,omitempty"`

	// Nodes define the list of NNF Nodes that will be the source of storage provisioning
	// and the servers that can utilize the storage
	Nodes []NnfStorageNodeSpec `json:"nodes,omitempty"`
}

// NnfStorageNodeSpec defines the desired state of an individual NNF Node within
// the NNF Storage Specification.
type NnfStorageNodeSpec struct {

	// Node specifies the target NNF Node that will host the storage requested
	// by the specification. Duplicate names within the parent's list of
	// nodes is not allowed.
	Node string `json:"node"`

	// NNF Node Storage specification is an embedding of the per-node specification
	// for storage.
	NnfNodeStorageSpec `json:",inline"`
}

// NnfStorageStatus defines the observed status of NNF Storage.
type NnfStorageStatus struct {
	// Important: Run "make" to regenerate code after modifying this file

	// Status reflects the status of this NNF Storage
	Status NnfResourceStatusType `json:"status,omitempty"`

	// Health reflects the health of this NNF Storage
	Health NnfResourceHealthType `json:"health,omitempty"`

	// Nodes defines status of inidividual NNF Nodes that are managed by
	// the parent NNF Storage specification. Each node in the specification
	// has a companion status element of the same name.
	Nodes []NnfStorageNodeStatus `json:"nodes,omitempty"`

	// TODO: Conditions
}

// NnfStorageNodeStatus defines the observed status of a storage present
// on one particular NNF Node defined within the specifiction. Each NNF Storage
// Node Status shall have a corresponding NNF Storage Node Spec with equivalent
// Name field. This creates a Spec-Status pair for inspecting elements.
type NnfStorageNodeStatus struct {

	// Node specifies the target NNF Node that will host the storage requested
	// by the specification. Duplicate names within the parent's list of
	// nodes is not allowed.
	Node string `json:"node"`

	// NnfNodeStorageStatus specifies the node storage status of a particular NNF Node.
	NnfNodeStorageStatus `json:",inline"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Storage is the Schema for the storages API
type NnfStorage struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NnfStorageSpec   `json:"spec,omitempty"`
	Status NnfStorageStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// StorageList contains a list of Storage
type NnfStorageList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NnfStorage `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NnfStorage{}, &NnfStorageList{})
}
