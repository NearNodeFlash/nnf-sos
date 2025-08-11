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

package v1alpha6

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:validation:Enum=Enabled;Disabled
type SystemNodeStatus string

const (
	SystemNodeStatusEnabled  SystemNodeStatus = "Enabled"
	SystemNodeStatusDisabled SystemNodeStatus = "Disabled"
)

// SystemStatusData defines the data in the SystemStatus
type SystemStatusData struct {
	// Nodes is a map of node name to node status
	Nodes map[string]SystemNodeStatus `json:"nodes,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// SystemStatus is the Schema for the systemstatuses API
type SystemStatus struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Data SystemStatusData `json:"data,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// SystemStatusList contains a list of SystemStatus
type SystemStatusList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SystemStatus `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SystemStatus{}, &SystemStatusList{})
}
