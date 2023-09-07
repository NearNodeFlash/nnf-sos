/*
 * Copyright 2021-2023 Hewlett Packard Enterprise Development LP
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

package v1alpha2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/HewlettPackard/dws/utils/updater"
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

	// Ports is the list of ports available for communication between nodes in the system.
	// Valid values are single integers, or a range of values of the form "START-END" where
	// START is an integer value that represents the start of a port range and END is an
	// integer value that represents the end of the port range (inclusive).
	Ports []intstr.IntOrString `json:"ports,omitempty"`

	// PortsCooldownInSeconds is the number of seconds to wait before a port can be reused. Defaults
	// to 60 seconds (to match the typical value for the kernel's TIME_WAIT). A value of 0 means the
	// ports can be reused immediately.
	// +kubebuilder:default:=60
	PortsCooldownInSeconds int `json:"portsCooldownInSeconds"`
}

// SystemConfigurationStatus defines the status of SystemConfiguration
type SystemConfigurationStatus struct {
	// Ready indicates when the SystemConfiguration has been reconciled
	Ready bool `json:"ready"`
}

//+kubebuilder:object:root=true
//+kubebuilder:storageversion
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

func (s *SystemConfiguration) GetStatus() updater.Status[*SystemConfigurationStatus] {
	return &s.Status
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
