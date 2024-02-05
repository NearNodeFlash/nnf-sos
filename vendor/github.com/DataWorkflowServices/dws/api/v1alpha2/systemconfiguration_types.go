/*
 * Copyright 2021-2024 Hewlett Packard Enterprise Development LP
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

	"github.com/DataWorkflowServices/dws/utils/updater"
)

// SystemConfigurationExternalComputeNode describes a compute node that is
// not directly matched with any of the nodes in the StorageNodes list.
type SystemConfigurationExternalComputeNode struct {
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
	// ExternalComputeNodes is the list of computes nodes that are not
	// directly matched with any of the StorageNodes.
	ExternalComputeNodes []SystemConfigurationExternalComputeNode `json:"externalComputeNodes,omitempty"`

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

func (in *SystemConfiguration) Computes() []*string {
	// We expect that there can be a large number of compute nodes and we don't
	// want to duplicate all of those names.
	// So we'll walk spec.storageNodes twice so we can set the
	// length/capacity for the array that will hold pointers to the names.
	num := 0
	for i1 := range in.Spec.StorageNodes {
		num += len(in.Spec.StorageNodes[i1].ComputesAccess)
	}
	// Add room for the external computes.
	num += len(in.Spec.ExternalComputeNodes)
	computes := make([]*string, num)
	idx := 0
	for i2 := range in.Spec.StorageNodes {
		for i3 := range in.Spec.StorageNodes[i2].ComputesAccess {
			computes[idx] = &in.Spec.StorageNodes[i2].ComputesAccess[i3].Name
			idx++
		}
	}
	// Add the external computes.
	for i4 := range in.Spec.ExternalComputeNodes {
		computes[idx] = &in.Spec.ExternalComputeNodes[i4].Name
		idx++
	}
	return computes
}
