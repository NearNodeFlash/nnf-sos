/*
 * Copyright 2021, 2022 Hewlett Packard Enterprise Development LP
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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SystemConfigurationComputeNode describes a compute node in the system
type SystemConfigurationComputeNode struct {
	// Name of the compute node
	Name string `json:"name"`
}

// SystemConfigurationComputeNodeReference describes a compute node that
// has access to a server.
type SystemConfigurationComputeNodeReference struct {
	// Name of the compute node
	Name string `json:"name"`

	// Index of the compute node from the server
	Index int `json:"index"`
}

// SystemConfigurationStorageNode describes a storage node in the system
type SystemConfigurationStorageNode struct {
	// Type is the type of server
	// +kubebuilder:validation:Enum=Rabbit
	Type string `json:"type"`

	// Name of the server node
	Name string `json:"name"`

	// ComputesAccess is the list of compute nodes that can use the server
	ComputesAccess []SystemConfigurationComputeNodeReference `json:"computesAccess,omitempty"`
}

// SystemConfigurationSpec describes the node layout of the system. This is filled in by
// an administrator at software installation time.
type SystemConfigurationSpec struct {
	// ComputeNodes is the list of compute nodes on the system
	ComputeNodes []SystemConfigurationComputeNode `json:"computeNodes,omitempty"`

	// StorageNodes is the list of storage nodes on the system
	StorageNodes []SystemConfigurationStorageNode `json:"storageNodes,omitempty"`
}

// SystemConfigurationStatus defines the status of SystemConfiguration
type SystemConfigurationStatus struct {
	// Ready indicates when the SystemConfiguration has been reconciled
	Ready bool `json:"ready"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="READY",type="boolean",JSONPath=".status.ready",description="True if SystemConfiguration is reconciled"
//+kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"

// SystemConfiguration is the Schema for the systemconfigurations API
type SystemConfiguration struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SystemConfigurationSpec   `json:"spec,omitempty"`
	Status SystemConfigurationStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// SystemConfigurationList contains a list of SystemConfiguration
type SystemConfigurationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SystemConfiguration `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SystemConfiguration{}, &SystemConfigurationList{})
}
