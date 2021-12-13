/*
Copyright 2021 Hewlett Packard Enterprise Development LP
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Important: Run "make" to regenerate code after modifying this file

// ServersSpecStorage specifies info required to identify the storage to
// use, and the number of allocations to make on that storage.
// ServersSpecAllocationSet.AllocationSize specifies the size of each allocation.
type ServersSpecStorage struct {
	// The name of the storage
	Name string `json:"name"`

	// The number of allocations to create of the size in bytes specified in ServersSpecAllocationSet
	// +kubebuilder:validation:Minimum=1
	AllocationCount int `json:"allocationCount"`
}

// ServersSpecAllocationSet is a set of allocations that all share the same allocation
// size and allocation type (e.g., XFS)
type ServersSpecAllocationSet struct {
	// Label as specified in the DirectiveBreakdown
	Label string `json:"label"`

	// Allocation size in bytes
	// +kubebuilder:validation:Minimum=1
	AllocationSize int64 `json:"allocationSize"`

	// List of storage resources where allocations are created
	Storage []ServersSpecStorage `json:"storage"`
}

// ServersSpec defines the desired state of Servers
type ServersSpec struct {
	AllocationSets []ServersSpecAllocationSet `json:"allocationSets,omitempty"`
}

// ServersStatusStorage is the status of the allocations on a storage
type ServersStatusStorage struct {
	// The name of the storage
	Name string `json:"name"`

	// Allocation size in bytes
	AllocationSize int64 `json:"allocationSize"`
}

// ServersStatusAllocationSet is the status of a set of allocations
type ServersStatusAllocationSet struct {
	// Label as specified in the DirectiveBreakdown
	Label string `json:"label"`

	// List of storage resources that have allocations
	Storage []ServersStatusStorage `json:"storage"`
}

type ServersStatus struct {
	Ready          bool                         `json:"ready"`
	LastUpdate     *metav1.MicroTime            `json:"lastUpdate,omitempty"`
	AllocationSets []ServersStatusAllocationSet `json:"allocationSets,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Servers is the Schema for the servers API
type Servers struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ServersSpec   `json:"spec,omitempty"`
	Status ServersStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ServersList contains a list of Servers
type ServersList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Servers `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Servers{}, &ServersList{})
}
