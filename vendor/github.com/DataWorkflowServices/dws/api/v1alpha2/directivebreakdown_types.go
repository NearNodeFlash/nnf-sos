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
	"github.com/DataWorkflowServices/dws/utils/updater"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type AllocationStrategy string

const (
	AllocatePerCompute    AllocationStrategy = "AllocatePerCompute"
	AllocateAcrossServers AllocationStrategy = "AllocateAcrossServers"
	AllocateSingleServer  AllocationStrategy = "AllocateSingleServer"
)

const (
	// DirectiveLifetimeJob specifies storage allocated for the lifetime of the job
	DirectiveLifetimeJob = "job"
	// DirectiveLifetimePersistent specifies storage allocated an indefinite lifetime usually longer than a job
	DirectiveLifetimePersistent = "persistent"
)

// AllocationSetColocationConstraint specifies how to colocate storage resources.
// A colocation constraint specifies how the location(s) of an allocation set should be
// selected with relation to other allocation sets. Locations for allocation sets with the
// same colocation key should be picked according to the colocation type.
type AllocationSetColocationConstraint struct {
	// Type of colocation constraint
	// +kubebuilder:validation:Enum=exclusive
	Type string `json:"type"`

	// Key shared by all the allocation sets that have their location constrained
	// in relation to each other.
	Key string `json:"key"`
}

// AllocationSetConstraints specifies the constraints required for colocation of Storage
// resources
type AllocationSetConstraints struct {
	// Labels is a list of labels is used to filter the Storage resources
	Labels []string `json:"labels,omitempty"`

	// Scale is a hint for the number of allocations to make based on a 1-10 value
	// +kubebuilder:validation:Minimum:=1
	// +kubebuilder:validation:Maximum:=10
	Scale int `json:"scale,omitempty"`

	// Count is the number of the allocations to make
	// +kubebuilder:validation:Minimum:=1
	Count int `json:"count,omitempty"`

	// Colocation is a list of constraints for which Storage resources
	// to pick in relation to Storage resources for other allocation sets.
	Colocation []AllocationSetColocationConstraint `json:"colocation,omitempty"`
}

// StorageAllocationSet defines the details of an allocation set
type StorageAllocationSet struct {
	// AllocationStrategy specifies the way to determine the number of allocations of the MinimumCapacity required for this AllocationSet.
	// +kubebuilder:validation:Enum=AllocatePerCompute;AllocateAcrossServers;AllocateSingleServer
	AllocationStrategy AllocationStrategy `json:"allocationStrategy"`

	// MinimumCapacity is the minumum number of bytes required to meet the needs of the filesystem that
	// will use the storage.
	// +kubebuilder:validation:Minimum:=1
	MinimumCapacity int64 `json:"minimumCapacity"`

	// Label is an identifier used to communicate from the DWS interface to internal interfaces
	// the filesystem use of this AllocationSet.
	// +kubebuilder:validation:Enum=raw;xfs;gfs2;mgt;mdt;mgtmdt;ost;
	Label string `json:"label"`

	// Constraint is an additional requirement pertaining to the suitability of Storage resources that may be used
	// for this AllocationSet
	Constraints AllocationSetConstraints `json:"constraints,omitempty"`
}

const (
	StorageLifetimePersistent = "persistent"
	StorageLifetimeJob        = "job"
)

// StorageBreakdown describes the storage requirements of a directive
type StorageBreakdown struct {
	// Lifetime is the duration of the allocation
	// +kubebuilder:validation:Enum=job;persistent
	Lifetime string `json:"lifetime"`

	// Reference is an ObjectReference to another resource
	Reference corev1.ObjectReference `json:"reference,omitempty"`

	// AllocationSets lists the allocations required to fulfill the #DW Directive
	AllocationSets []StorageAllocationSet `json:"allocationSets,omitempty"`
}

type ComputeLocationType string

const (
	ComputeLocationNetwork  ComputeLocationType = "network"
	ComputeLocationPhysical ComputeLocationType = "physical"
)

type ComputeLocationPriority string

const (
	ComputeLocationPriorityMandatory  ComputeLocationPriority = "mandatory"
	ComputeLocationPriorityBestEffort ComputeLocationPriority = "bestEffort"
)

type ComputeLocationAccess struct {
	// Type is the relationship between the compute nodes and the resource in the Reference
	// +kubebuilder:validation:Enum=physical;network
	Type ComputeLocationType `json:"type"`

	// Priority specifies whether the location constraint is mandatory or best effort
	// +kubebuilder:validation:Enum=mandatory;bestEffort
	Priority ComputeLocationPriority `json:"priority"`
}

// ComputeLocationConstraint describes a constraints on which compute nodes can be used with
// a directive based on their location
type ComputeLocationConstraint struct {
	Access []ComputeLocationAccess `json:"access"`

	// Reference is an object reference to a resource that contains the location information
	Reference corev1.ObjectReference `json:"reference"`
}

// ComputeConstraints describes the constraints to use when picking compute nodes
type ComputeConstraints struct {
	// Location is a list of location constraints
	Location []ComputeLocationConstraint `json:"location,omitempty"`
}

// ComputeBreakdown describes the compute requirements of a directive
type ComputeBreakdown struct {
	// Constraints to use when picking compute nodes
	Constraints ComputeConstraints `json:"constraints,omitempty"`
}

// DirectiveBreakdownSpec defines the directive string to breakdown
type DirectiveBreakdownSpec struct {
	// Directive is a copy of the #DW for this breakdown
	Directive string `json:"directive"`

	// User ID of the user associated with the job
	UserID uint32 `json:"userID"`
}

// DirectiveBreakdownStatus defines the storage information WLM needs to select NNF Nodes and request storage from the selected nodes
type DirectiveBreakdownStatus struct {
	// Storage is the storage breakdown for the directive
	Storage *StorageBreakdown `json:"storage,omitempty"`

	// Compute is the compute breakdown for the directive
	Compute *ComputeBreakdown `json:"compute,omitempty"`

	// Ready indicates whether AllocationSets have been generated (true) or not (false)
	Ready bool `json:"ready"`

	// Error information
	ResourceError `json:",inline"`
}

//+kubebuilder:object:root=true
//+kubebuilder:storageversion
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="READY",type="boolean",JSONPath=".status.ready",description="True if allocation sets have been generated"
//+kubebuilder:printcolumn:name="ERROR",type="string",JSONPath=".status.error.severity"
//+kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"

// DirectiveBreakdown is the Schema for the directivebreakdown API
type DirectiveBreakdown struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DirectiveBreakdownSpec   `json:"spec,omitempty"`
	Status DirectiveBreakdownStatus `json:"status,omitempty"`
}

func (db *DirectiveBreakdown) GetStatus() updater.Status[*DirectiveBreakdownStatus] {
	return &db.Status
}

//+kubebuilder:object:root=true

// DirectiveBreakdownList contains a list of DirectiveBreakdown
type DirectiveBreakdownList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DirectiveBreakdown `json:"items"`
}

func (d *DirectiveBreakdownList) GetObjectList() []client.Object {
	objectList := []client.Object{}

	for i := range d.Items {
		objectList = append(objectList, &d.Items[i])
	}

	return objectList
}

func init() {
	SchemeBuilder.Register(&DirectiveBreakdown{}, &DirectiveBreakdownList{})
}
