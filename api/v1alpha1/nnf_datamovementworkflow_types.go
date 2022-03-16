/*
Copyright 2021 Hewlett Packard Enterprise Development LP
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// NnfDataMovementWorkflowSpec defines the desired state of NnfDataMovementWorkflow
type NnfDataMovementWorkflowSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// NnfDataMovementWorkflowStatus defines the observed state of NnfDataMovementWorkflow
type NnfDataMovementWorkflowStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// NnfDataMovementWorkflow is the Schema for the nnfdatamovementworkflows API
type NnfDataMovementWorkflow struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NnfDataMovementWorkflowSpec   `json:"spec,omitempty"`
	Status NnfDataMovementWorkflowStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// NnfDataMovementWorkflowList contains a list of NnfDataMovementWorkflow
type NnfDataMovementWorkflowList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NnfDataMovementWorkflow `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NnfDataMovementWorkflow{}, &NnfDataMovementWorkflowList{})
}
