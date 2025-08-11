/*
 * Copyright 2021-2025 Hewlett Packard Enterprise Development LP
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

package v1alpha5

import (
	"github.com/DataWorkflowServices/dws/utils/updater"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Important: Run "make" to regenerate code after modifying this file

// ServersSpecStorage specifies info required to identify the storage to
// use, and the number of allocations to make on that storage.
// ServersSpecAllocationSet.AllocationSize specifies the size of each allocation.
type ServersSpecStorage struct {
	// The name of the storage
	Name string `json:"name"`

	// The number of allocations to create of the size in bytes specified in ServersSpecAllocationSet
	// +kubebuilder:validation:Minimum=1
	AllocationCount int `json:"allocationCount"`
}

// ServersSpecAllocationSet is a set of allocations that all share the same allocation
// size and allocation type (e.g., XFS)
type ServersSpecAllocationSet struct {
	// Label as specified in the DirectiveBreakdown
	Label string `json:"label"`

	// Allocation size in bytes
	// +kubebuilder:validation:Minimum=1
	AllocationSize int64 `json:"allocationSize"`

	// List of storage resources where allocations are created
	Storage []ServersSpecStorage `json:"storage"`
}

// ServersSpec defines the desired state of Servers
type ServersSpec struct {
	AllocationSets []ServersSpecAllocationSet `json:"allocationSets,omitempty"`
}

// ServersStatusStorage is the status of the allocations on a storage
type ServersStatusStorage struct {
	// Allocation size in bytes
	AllocationSize int64 `json:"allocationSize"`

	// Ready indicates whether all the allocations on the server have been successfully created
	Ready bool `json:"ready"`
}

// ServersStatusAllocationSet is the status of a set of allocations
type ServersStatusAllocationSet struct {
	// Label as specified in the DirectiveBreakdown
	Label string `json:"label"`

	// List of storage resources that have allocations
	Storage map[string]ServersStatusStorage `json:"storage"`
}

// ServersStatus specifies whether the Servers has achieved the
// ready condition along with the allocationSets that are managed
// by the Servers resource.
type ServersStatus struct {
	Ready          bool                         `json:"ready"`
	LastUpdate     *metav1.MicroTime            `json:"lastUpdate,omitempty"`
	AllocationSets []ServersStatusAllocationSet `json:"allocationSets,omitempty"`

	// Error information
	ResourceError `json:",inline"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="READY",type="boolean",JSONPath=".status.ready",description="True if allocation sets have been generated"
//+kubebuilder:printcolumn:name="ERROR",type="string",JSONPath=".status.error.severity"
//+kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"

// Servers is the Schema for the servers API
type Servers struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ServersSpec   `json:"spec,omitempty"`
	Status ServersStatus `json:"status,omitempty"`
}

func (s *Servers) GetStatus() updater.Status[*ServersStatus] {
	return &s.Status
}

//+kubebuilder:object:root=true

// ServersList contains a list of Servers
type ServersList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Servers `json:"items"`
}

// GetObjectList returns a list of Servers references.
func (s *ServersList) GetObjectList() []client.Object {
	objectList := []client.Object{}

	for i := range s.Items {
		objectList = append(objectList, &s.Items[i])
	}

	return objectList
}

func init() {
	SchemeBuilder.Register(&Servers{}, &ServersList{})
}
