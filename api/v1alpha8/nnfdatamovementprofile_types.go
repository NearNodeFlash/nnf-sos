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

package v1alpha8

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// NnfDataMovementProfileSpec defines the desired state of NnfDataMovementProfile
type NnfDataMovementProfileSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of NnfDataMovementProfile. Edit nnfdatamovementprofile_types.go to remove/update
	Foo string `json:"foo,omitempty"`
}

// NnfDataMovementProfileStatus defines the observed state of NnfDataMovementProfile
type NnfDataMovementProfileStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// NnfDataMovementProfile is the Schema for the nnfdatamovementprofiles API
type NnfDataMovementProfile struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NnfDataMovementProfileSpec   `json:"spec,omitempty"`
	Status NnfDataMovementProfileStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// NnfDataMovementProfileList contains a list of NnfDataMovementProfile
type NnfDataMovementProfileList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NnfDataMovementProfile `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NnfDataMovementProfile{}, &NnfDataMovementProfileList{})
}
