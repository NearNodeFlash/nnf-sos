/*
Copyright 2021 Hewlett Packard Enterprise Development LP
*/

package v1alpha1

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// NnfJobStorageInstanceSpec defines the desired state of NnfJobStorageInstance
type NnfJobStorageInstanceSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Name is the name of this storage instance within the job description. Names must be unique within
	// a job; while being referenced by copy_in and copy_out with the $JOB_DW_ keyword.
	Name string `json:"name,omitempty"`

	// FsType describes the File System Type for this storage instance.
	FsType string `json:"fsType,omitempty"`

	// Servers refers to the NNF Servers that are to provide the backing storage for this storage instance
	Servers corev1.ObjectReference `json:"servers,omitempty"`
}

// NnfJobStorageInstanceStatus defines the observed state of NnfJobStorageInstance
type NnfJobStorageInstanceStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// NnfJobStorageInstance is the Schema for the nnfjobstorageinstances API
type NnfJobStorageInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NnfJobStorageInstanceSpec   `json:"spec,omitempty"`
	Status NnfJobStorageInstanceStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// NnfJobStorageInstanceList contains a list of NnfJobStorageInstance
type NnfJobStorageInstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NnfJobStorageInstance `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NnfJobStorageInstance{}, &NnfJobStorageInstanceList{})
}

func NnfJobStorageInstanceMakeName(jobId int, workflowId string, dwIndex int) string {
	return fmt.Sprintf("job-%d-%s-%d", jobId, workflowId, dwIndex)
}
