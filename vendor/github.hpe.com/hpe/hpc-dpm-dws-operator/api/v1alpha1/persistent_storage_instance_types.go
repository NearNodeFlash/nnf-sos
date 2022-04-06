/*
Copyright 2022 Hewlett Packard Enterprise Development LP
*/

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PersistentStorageInstanceSpec defines the desired state of PersistentStorageInstance
type PersistentStorageInstanceSpec struct {
	// Name is the name given to this persistent storage instance.
	Name string `json:"name"`

	// FsType describes the File System Type for this storage instance.
	// +kubebuilder:validation:Enum=raw;xfs;gfs2;lustre
	FsType string `json:"fsType"`

	// DWDirective is a copy of the #DW for this instance
	DWDirective string `json:"dwDirective"`

	// Servers refers to the Servers resource that provides the backing storage for this storage instance
	Servers corev1.ObjectReference `json:"servers,omitempty"`
}

// PersistentStorageInstanceStatus defines the observed state of PersistentStorageInstance
type PersistentStorageInstanceStatus struct {
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// PersistentStorageInstance is the Schema for the Persistentstorageinstances API
type PersistentStorageInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PersistentStorageInstanceSpec   `json:"spec,omitempty"`
	Status PersistentStorageInstanceStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// PersistentStorageInstanceList contains a list of PersistentStorageInstance
type PersistentStorageInstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PersistentStorageInstance `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PersistentStorageInstance{}, &PersistentStorageInstanceList{})
}
