/*
Copyright 2021 Hewlett Packard Enterprise Development LP
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// NnfNodeSpec defines the desired state of NNF Node
type NnfNodeSpec struct {
	// Important: Run "make" to regenerate code after modifying this file

	// The unique name for this NNF Node
	// https://connect.us.cray.com/confluence/display/HSOS/Shasta+HSS+Component+Naming+Convention#ShastaHSSComponentNamingConvention-2.1.4.3MountainCabinetComponents
	Name string `json:"name,omitempty"`

	// Pod name for this NNF Node
	Pod string `json:"pod,omitempty"`

	// State reflects the desired state of this NNF Node resource
	// +kubebuilder:validation:Enum=Enable;Disable
	State NnfResourceStateType `json:"state"`
}

// NnfNodeStatus defines the observed status of NNF Node
type NnfNodeStatus struct {
	// Important: Run "make" to regenerate code after modifying this file

	// Status reflects the current status of the NNF Node
	Status NnfResourceStatusType `json:"status,omitempty"`

	Health NnfResourceHealthType `json:"health,omitempty"`

	Capacity          int64 `json:"capacity,omitempty"`
	CapacityAllocated int64 `json:"capacityAllocated,omitempty"`

	Servers []NnfServerStatus `json:"servers,omitempty"`

	Drives []NnfDriveStatus `json:"drives,omitempty"`
}

// NnfServerStatus defines the observed status of servers connected to this NNF Node
type NnfServerStatus struct {
	NnfResourceStatus `json:",inline"`
}

// NnfDriveStatus defines the observe status of drives connected to this NNF Node
type NnfDriveStatus struct {
	NnfResourceStatus `json:",inline"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// NnfNode is the Schema for the NnfNode API
type NnfNode struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NnfNodeSpec   `json:"spec,omitempty"`
	Status NnfNodeStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// NnfNodeList contains a list of NNF Nodes
type NnfNodeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NnfNode `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NnfNode{}, &NnfNodeList{})
}
