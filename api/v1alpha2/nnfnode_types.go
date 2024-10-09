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
	"github.com/DataWorkflowServices/dws/utils/updater"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// NnfNodeSpec defines the desired state of NNF Node
type NnfNodeSpec struct {
	// Important: Run "make" to regenerate code after modifying this file

	// The unique name for this NNF Node
	Name string `json:"name,omitempty"`

	// Pod name for this NNF Node
	Pod string `json:"pod,omitempty"`

	// State reflects the desired state of this NNF Node resource
	// +kubebuilder:validation:Enum=Enable;Disable
	State NnfResourceStateType `json:"state"`
}

// NnfNodeStatus defines the observed status of NNF Node
type NnfNodeStatus struct {
	// Important: Run "make" to regenerate code after modifying this file

	// Status reflects the current status of the NNF Node
	Status NnfResourceStatusType `json:"status,omitempty"`

	Health NnfResourceHealthType `json:"health,omitempty"`

	// Fenced is true when the NNF Node is fenced by the STONITH agent, and false otherwise.
	Fenced bool `json:"fenced,omitempty"`

	// LNetNid is the LNet address for the NNF node
	LNetNid string `json:"lnetNid,omitempty"`

	Capacity          int64 `json:"capacity,omitempty"`
	CapacityAllocated int64 `json:"capacityAllocated,omitempty"`

	Servers []NnfServerStatus `json:"servers,omitempty"`

	Drives []NnfDriveStatus `json:"drives,omitempty"`
}

// NnfServerStatus defines the observed status of servers connected to this NNF Node
type NnfServerStatus struct {
	Hostname string `json:"hostname,omitempty"`

	NnfResourceStatus `json:",inline"`
}

// NnfDriveStatus defines the observe status of drives connected to this NNF Node
type NnfDriveStatus struct {
	// Model is the manufacturer information about the device
	Model string `json:"model,omitempty"`

	// The serial number for this storage controller.
	SerialNumber string `json:"serialNumber,omitempty"`

	// The firmware version of this storage controller.
	FirmwareVersion string `json:"firmwareVersion,omitempty"`

	// Physical slot location of the storage controller.
	Slot string `json:"slot,omitempty"`

	// Capacity in bytes of the device. The full capacity may not
	// be usable depending on what the storage driver can provide.
	Capacity int64 `json:"capacity,omitempty"`

	// WearLevel in percent for SSDs
	WearLevel int64 `json:"wearLevel,omitempty"`

	NnfResourceStatus `json:",inline"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="STATE",type="string",JSONPath=".spec.state",description="Current desired state"
//+kubebuilder:printcolumn:name="HEALTH",type="string",JSONPath=".status.health",description="Health of node"
//+kubebuilder:printcolumn:name="STATUS",type="string",JSONPath=".status.status",description="Current status of node"
//+kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
//+kubebuilder:printcolumn:name="POD",type="string",JSONPath=".spec.pod",description="Parent pod name",priority=1

// NnfNode is the Schema for the NnfNode API
type NnfNode struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NnfNodeSpec   `json:"spec,omitempty"`
	Status NnfNodeStatus `json:"status,omitempty"`
}

func (n *NnfNode) GetStatus() updater.Status[*NnfNodeStatus] {
	return &n.Status
}

//+kubebuilder:object:root=true

// NnfNodeList contains a list of NNF Nodes
type NnfNodeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NnfNode `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NnfNode{}, &NnfNodeList{})
}
