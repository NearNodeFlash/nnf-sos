/*
Copyright 2021 Hewlett Packard Enterprise Development LP
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ComputesSpec defines the desired state of Computes
type ComputesSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of Computes. Edit computes_types.go to remove/update
	Foo string `json:"foo,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Computes is the Schema for the computes API
type Computes struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ComputesSpec `json:"spec,omitempty"`
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
