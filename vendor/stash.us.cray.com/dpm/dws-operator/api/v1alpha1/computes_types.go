/*
Copyright 2021 Hewlett Packard Enterprise Development LP
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ComputesData defines the compute nodes that are assigned to the workflow
type ComputesData struct {
	// Name is the identifer name for the compute node
	Name string `json:"name"`
}

//+kubebuilder:object:root=true

// Computes is the Schema for the computes API
type Computes struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Data []ComputesData `json:"data,omitempty"`
}

//+kubebuilder:object:root=true

// ComputesList contains a list of Computes
type ComputesList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Computes `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Computes{}, &ComputesList{})
}
