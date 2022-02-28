/*
Copyright 2021 Hewlett Packard Enterprise Development LP
*/

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// NnfPersistentStorageInstanceSpec defines the desired state of NnfPersistentStorageInstance
type NnfPersistentStorageInstanceSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Name is the name given to this persistent storage instance.
	Name string `json:"name,omitempty"`

	// FsType describes the File System Type for this storage instance.
	FsType string `json:"fsType,omitempty"`

	// Servers refers to the NNF Servers that are to provide the backing storage for this storage instance
	Servers corev1.ObjectReference `json:"servers,omitempty"`
}

// NnfPersistentStorageInstanceStatus defines the observed state of NnfPersistentStorageInstance
type NnfPersistentStorageInstanceStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// NnfPersistentStorageInstance is the Schema for the nnfpersistentstorageinstances API
type NnfPersistentStorageInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NnfPersistentStorageInstanceSpec   `json:"spec,omitempty"`
	Status NnfPersistentStorageInstanceStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// NnfPersistentStorageInstanceList contains a list of NnfPersistentStorageInstance
type NnfPersistentStorageInstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NnfPersistentStorageInstance `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NnfPersistentStorageInstance{}, &NnfPersistentStorageInstanceList{})
}
