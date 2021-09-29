/*
Copyright 2021 Hewlett Packard Enterprise Development LP
*/

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// The DWRecord contains the index of the Datawarp directive (#DW) within the workflow
// along with a copy of the actual #DW
type DWRecord struct {
	// DWDirectiveIndex is the index of the #DW directive in the workflow
	DWDirectiveIndex int `json:"dwDirectiveIndex"`

	// DWDirective is a copy of the #DW for this breakdown
	DWDirective string `json:"dwDirective"`
}

// AllocationSetComponents define the details of the allocation
type AllocationSetComponents struct {
	// AllocationStrategy specifies the way to determine the number of allocations of the MinimumCapacity required for this AllocationSet.
	// +kubebuilder:validation:Enum=AllocatePerCompute;AllocateAcrossServers;AllocateSingleServer;AssignPerCompute;AssignAcrossServers;
	AllocationStrategy string `json:"allocationStrategy"`

	// MinimumCapacity is the minumum number of bytes required to meet the needs of the filesystem that
	// will use the storage.
	// +kubebuilder:validation:Minimum:=1
	MinimumCapacity int64 `json:"minimumCapacity"`

	// Label is an identifier used to communicate from the DWS interface to internal interfaces
	// the filesystem use of this AllocationSet.
	// +kubebuilder:validation:Enum=raw;xfs;gfs2;mgt;mdt;ost;
	Label string `json:"label"`

	// Constraint is an additional requirement pertaining to the suitability of Servers that may be used for this AllocationSet
	// +kubebuilder:validation:Enum=MayNotBeShared;
	Constraint string `json:"constraint,omitempty"`
}

// DirectiveBreakdownSpec defines the storage information WLM needs to select NNF Nodes and request storage from the selected nodes
type DirectiveBreakdownSpec struct {
	DW DWRecord `json:"dwRecord"`

	// Name is the identifier for this directive breakdown
	Name string `json:"name"`

	// Type is the type specified in the #DW directive
	// +kubebuilder:validation:Enum=raw;xfs;lustre
	Type string `json:"type"`

	// Lifetime is the duration of the allocation
	// +kubebuilder:validation:Enum=job;persistent
	Lifetime string `json:"lifetime"`

	// Servers is a reference to the Server CR
	Servers corev1.ObjectReference `json:"servers,omitempty"`

	// AllocationSets lists the allocations required to fulfill the #DW Directive
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
