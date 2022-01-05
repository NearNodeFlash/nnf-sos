/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// DataMovementSpec defines the desired state of DataMovement
type DataMovementSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Source DataMovementSpecSourceDestination `json:"source,omitempty"`

	Destination DataMovementSpecSourceDestination `json:"destination,omitempty"`

	Servers corev1.ObjectReference `json:"storageInstance,omitempty"`

	NodeAccess corev1.ObjectReference `json:"nodeAccess,omitempty"`
}

// DataMovementSpecSourceDestination defines the desired source or destination of data movement
type DataMovementSpecSourceDestination struct {
	Path string `json:"path,omitempty"`

	StorageInstance *corev1.ObjectReference `json:"storageInstance,omitempty"`
}

// DataMovementStatus defines the observed state of DataMovement
type DataMovementStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Job corev1.ObjectReference `json:"job,omitempty"`

	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

const (
	DataMovementConditionCreating = "Creating"
	DataMovementConditionRunning  = "Running"
	DataMovementConditionFinished = "Finished"
)

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// DataMovement is the Schema for the datamovements API
type DataMovement struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DataMovementSpec   `json:"spec,omitempty"`
	Status DataMovementStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// DataMovementList contains a list of DataMovement
type DataMovementList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DataMovement `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DataMovement{}, &DataMovementList{})
}
