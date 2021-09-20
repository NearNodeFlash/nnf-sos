/*
Copyright 2021 Hewlett Packard Enterprise Development LP
*/

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.
type DWRecord struct {
	// Array index of the #DW directive in original WFR
	DWDirectiveIndex int `json:"dwDirectiveIndex"`
	// Copy of the #DW for this breakdown
	DWDirective string `json:"dwDirective"`
}

// AllocationSetComponents define the details of the allocation
type AllocationSetComponents struct {
	// +kubebuilder:validation:Enum=AllocatePerCompute;AllocateAcrossServers;AllocateSingleServer;AssignPerCompute;AssignAcrossServers;
	AllocationStrategy string `json:"allocationStrategy"`
	MinimumCapacity    int64  `json:"minimumCapacity"`
	Label              string `json:"label"`
	Constraint         string `json:"constraint"`
}

// DirectiveBreakdownSpec defines the storage information WLM needs to select NNF Nodes and request storage from the selected nodes
type DirectiveBreakdownSpec struct {
	DW   DWRecord `json:"dwRecord"`
	Name string   `json:"name"`
	Type string   `json:"type"`

	// +kubebuilder:validation:Enum=job;persistent
	Lifetime string `json:"lifetime"`

	// Reference to the Server CR to be filled in by WLM
	Servers corev1.ObjectReference `json:"servers,omitempty"`

	// Allocation sets required to fulfill the #DW Directive
	AllocationSet []AllocationSetComponents `json:"allocationSet"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// DirectiveBreakdown is the Schema for the directivebreakdown API
type DirectiveBreakdown struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec DirectiveBreakdownSpec `json:"spec,omitempty"`
}

//+kubebuilder:object:root=true

// DirectiveBreakdownList contains a list of DirectiveBreakdown
type DirectiveBreakdownList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DirectiveBreakdown `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DirectiveBreakdown{}, &DirectiveBreakdownList{})
}
