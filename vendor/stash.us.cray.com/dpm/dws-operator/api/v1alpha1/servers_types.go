/*
Copyright 2021 Hewlett Packard Enterprise Development LP
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Important: Run "make" to regenerate code after modifying this file

// Host specifies info required to identify the host (name) on which to
// create storage, and the number of storage allocations on each Host.
// ServersData.AllocationSize specifies the size of each allocation.
type Host struct {
	// The name of the host
	Name string `json:"name"`

	// The number of allocations to create of the size in bytes specified in ServerData
	// +kubebuilder:validation:Minimum=1
	AllocationCount int `json:"allocationCount"`
}

// ServersData defines the desired state of Servers
type ServersData struct {
	// Label as specified in the DirectiveBreakdown
	Label string `json:"label"`

	// Allocation size in bytes
	// +kubebuilder:validation:Minimum=1
	AllocationSize int64 `json:"allocationSize"`

	// List of hosts where allocations are created
	Hosts []Host `json:"hosts"`
}

//+kubebuilder:object:root=true

// Servers is the Schema for the servers API
type Servers struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Data []ServersData `json:"data,omitempty"`
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
