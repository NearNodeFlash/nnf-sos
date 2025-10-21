/*
 * Copyright 2025 Hewlett Packard Enterprise Development LP
 * Other additional copyright holders may be indicated within.
 *
 * The entirety of this work is licensed under the Apache License,
 * Version 2.0 (the "License"); you may not use this file except
 * in compliance with the License.
 *
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package v1alpha9

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// NnfDataMovementManagerSpec defines the desired state of NnfDataMovementManager
type NnfDataMovementManagerSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of NnfDataMovementManager. Edit nnfdatamovementmanager_types.go to remove/update
	Foo string `json:"foo,omitempty"`
}

// NnfDataMovementManagerStatus defines the observed state of NnfDataMovementManager
type NnfDataMovementManagerStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// NnfDataMovementManager is the Schema for the nnfdatamovementmanagers API
type NnfDataMovementManager struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NnfDataMovementManagerSpec   `json:"spec,omitempty"`
	Status NnfDataMovementManagerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// NnfDataMovementManagerList contains a list of NnfDataMovementManager
type NnfDataMovementManagerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NnfDataMovementManager `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NnfDataMovementManager{}, &NnfDataMovementManagerList{})
}
