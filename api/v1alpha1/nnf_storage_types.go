/*
Copyright 2021 Hewlett Packard Enterprise Development LP
*/

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type NnfStorageAllocationNodes struct {
	// Name of the node to make the allocation on
	Name string `json:"name"`

	// Number of allocations to make on this node
	Count int `json:"count"`
}

type NnfStorageLustreSpec struct {
	// FileSystemName is the fsname parameter for the Lustre filesystem.
	// +kubebuilder:validation:MaxLength:=8
	FileSystemName string `json:"fileSystemName,omitempty"`

	// TargetType is the type of Lustre target to be created.
	// +kubebuilder:validation:Enum=MGT;MDT;OST
	TargetType string `json:"targetType,omitempty"`

	// BackFs is the type of backing filesystem to use.
	// +kubebuilder:validation:Enum=ldiskfs;zfs
	BackFs string `json:"backFs,omitempty"`
}

type NnfStorageAllocationSetSpec struct {
	// Name is a human readable label for this set of allocations (e.g., xfs)
	Name string `json:"name"`

	// Capacity defines the capacity, in bytes, of this storage specification. The NNF Node itself
	// may split the storage among the available drives operating in the NNF Node.
	Capacity int64 `json:"capacity"`

	// FileSystemType defines the type of the desired filesystem, or raw
	// block device.
	// +kubebuilder:validation:Enum=raw;lvm;zfs;lustre;xfs
	// +kubebuilder:default:=raw
	FileSystemType string `json:"fileSystemType"`

	// Lustre specific configuration
	NnfStorageLustreSpec `json:",inline"`

	// Nodes is the list of Rabbit nodes to make allocations on
	Nodes []NnfStorageAllocationNodes `json:"nodes"`
}

// NnfStorageSpec defines the specification for requesting generic storage on a set
// of available NNF Nodes. This object is related to a #DW for NNF Storage, with the WLM
// making the determination for which NNF Nodes it wants to utilize.
type NnfStorageSpec struct {
	// AllocationSets is a list of different types of storage allocations to make. Each
	// AllocationSet describes an entire allocation spanning multiple Rabbits. For example,
	// an AllocationSet could be all of the OSTs in a Lustre filesystem, or all of the raw
	// block devices in a raw block configuration.
	AllocationSets []NnfStorageAllocationSetSpec `json:"allocationSets"`
}

type NnfStorageAllocationNodeStatus struct {
	Reference corev1.ObjectReference `json:"reference,omitempty"`
}

type NnfStorageAllocationSetStatus struct {
	// Status reflects the status of this NNF Storage
	Status NnfResourceStatusType `json:"status,omitempty"`

	// Health reflects the health of this NNF Storage
	Health NnfResourceHealthType `json:"health,omitempty"`

	// Reason is the human readable reason for the status
	Reason string `json:"reason,omitempty"`

	// NodeStorageReferences is the list of object references to the child NnfNodeStorage resources
	NodeStorageReferences []corev1.ObjectReference `json:"nodeStorageReferences"`
}

// NnfStorageStatus defines the observed status of NNF Storage.
type NnfStorageStatus struct {
	// Important: Run "make" to regenerate code after modifying this file

	// MgsNode is the NID of the MGS.
	MgsNode string `json:"mgsNode,omitempty"`

	// AllocationsSets holds the status information for each of the AllocationSets
	// from the spec.
	AllocationSets []NnfStorageAllocationSetStatus `json:"allocationSets,omitempty"`

	// TODO: Conditions
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
