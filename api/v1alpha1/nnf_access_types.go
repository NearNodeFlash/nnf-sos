/*
Copyright 2021 Hewlett Packard Enterprise Development LP
*/

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NnfAccessSpec defines the desired state of NnfAccess
type NnfAccessSpec struct {
	// DesiredState is the desired state for the mounts on the client
	// +kubebuilder:validation:Enum=mounted;unmounted
	DesiredState string `json:"desiredState"`

	// Target specifies which storage targets the client should mount
	// - single: Only one of the storage the client can access
	// - all: All of the storage the client can access
	// +kubebuilder:validation:Enum=single;all
	Target string `json:"target"`

	// ClientReference is for a client resource. (DWS) Computes is the only client
	// resource type currently supported
	ClientReference corev1.ObjectReference `json:"clientReference,omitempty"`

	// MountPath for the storage target on the client
	MountPath string `json:"mountPath,omitempty"`

	// MountPathPrefix to  mount the storage target on the client when there is
	// more than one mount on a client
	MountPathPrefix string `json:"mountPathPrefix,omitempty"`

	// StorageReference is the NnfStorage reference
	StorageReference corev1.ObjectReference `json:"storageReference"`
}

// NnfAccessStatus defines the observed state of NnfAccess
type NnfAccessStatus struct {
	// State is the current state
	// +kubebuilder:validation:Enum=mounted;unmounted
	State string `json:"state"`

	// Ready signifies whether status.state has been achieved
	Ready bool `json:"ready"`

	// Error message
	Message string `json:"message,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// NnfAccess is the Schema for the nnfaccesses API
type NnfAccess struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NnfAccessSpec   `json:"spec,omitempty"`
	Status NnfAccessStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// NnfAccessList contains a list of NnfAccess
type NnfAccessList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NnfAccess `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NnfAccess{}, &NnfAccessList{})
}
