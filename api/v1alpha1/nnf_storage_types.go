/*
Copyright 2021 Hewlett Packard Enterprise Development LP
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// StorageSpec defines the desired state of Storage
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

	// File System defines the file system that is constructed from underlying
	// storage subsystem.
	FileSystem string `json:"fileSystem,omitempty"`

	// Nodes define the list of NNF Nodes that will be the source of storage provisioning
	// and the servers that can utilize the storage
	Nodes []NnfStorageNodeSpec `json:"nodes,omitempty"`
}

// NnfStorageNodeSpec defines the desired state of an individual NNF Node.
type NnfStorageNodeSpec struct {

	// Index specifies the unique index of this NNF Storage Node specification within
	// the context of the parent Storage specification.
	Index int `json:"index,omitempty"`

	// Node specifies the target NNF Node that will host the storage requested
	// by the specification. Duplicate names within the parent's list of
	// nodes is not allowed.
	Node string `json:"node"`

	// Servers specifies the target NNF Servers that will be receiving the
	// storage requested by the specification's file system type, or block
	// storage if no file system is specified.
	Servers []NnfStorageServerSpec `json:"servers,omitempty"`
}

// NnfStorageServerSpec defines the desired state of storage presented to
// a server connecting to the NNF Storage Node.
type NnfStorageServerSpec struct {
	NnfNodeStorageServerSpec `json:",inline"`

	/*
		// The unique Id for the NNF Storage Server, as provided by NNF Node
		// Storage Server value for a particular NNF Node. Valid ID's can
		// be reported by querying the NNF Node's status and looking at the
		// {.status.servers[*]} listing
		Id string `json:"id"`

		// The name of the NNF Storage Server, as provided by the NNF Node
		// Storage Server value for a given NNF Node. This is an informative
		// field and can be omitted. If provided, the Name must match the name
		// field of the {.status.servers[*]} with the ID.
		Name string `json:"name,omitempty"`

		// Path defines the path on the NNF Storage Server where the file system
		// will appear. Only applicable for non raw-block storage.
		Path string `json:"path,omitempty"`
	*/
}

// NnfStorageStatus defines the observed status of NNF Storage specification
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
// on one particular NNF Node defined within the specifiction
type NnfStorageNodeStatus struct {

	// Status reflects the current status of the NNF Node
	Status NnfResourceStatusType `json:"status,omitempty"`

	// Health reflects the current health of the NNF Node
	Health NnfResourceHealthType `json:"health,omitempty"`

	// Storage reflects the current health of NNF Storage on the NNF Node
	Storage NnfNodeStorageStatus `json:"storage,omitempty"`
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
