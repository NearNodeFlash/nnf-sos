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

// RsyncDataMovementSpec defines the desired state of RsyncDataMovement
type RsyncDataMovementSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Source is the source file or directory used during data movememnt
	Source string `json:"source,omitempty"`

	// Destination is the destination file or directory used during data movement
	Destination string `json:"destination,omitempty"`

	// Node Access refers to the
	NodeAccess corev1.ObjectReference `json:"nodeAccess,omitempty"`

	// Servers refers to the list of NNF Servers that will conduct the data movement from Source to Destination
	Servers corev1.ObjectReference `json:"servers,omitempty"`
}

// RsyncDataMovementStatus defines the observed state of RsyncDataMovement
type RsyncDataMovementStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// RsyncDataMovement is the Schema for the rsyncdatamovements API
type RsyncDataMovement struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RsyncDataMovementSpec   `json:"spec,omitempty"`
	Status RsyncDataMovementStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// RsyncDataMovementList contains a list of RsyncDataMovement
type RsyncDataMovementList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RsyncDataMovement `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RsyncDataMovement{}, &RsyncDataMovementList{})
}
