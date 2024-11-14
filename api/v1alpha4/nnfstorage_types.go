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

package v1alpha4

import (
	dwsv1alpha2 "github.com/DataWorkflowServices/dws/api/v1alpha2"
	"github.com/DataWorkflowServices/dws/utils/updater"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	AllocationSetLabel = "nnf.cray.hpe.com/allocationset"
)

// NnfStorageAllocationNodes identifies the node and properties of the allocation to make on that node
type NnfStorageAllocationNodes struct {
	// Name of the node to make the allocation on
	Name string `json:"name"`

	// Number of allocations to make on this node
	Count int `json:"count"`
}

// NnfStorageLustreSpec defines the specifications for a Lustre filesystem
type NnfStorageLustreSpec struct {
	// TargetType is the type of Lustre target to be created.
	// +kubebuilder:validation:Enum=mgt;mdt;mgtmdt;ost
	TargetType string `json:"targetType,omitempty"`

	// BackFs is the type of backing filesystem to use.
	// +kubebuilder:validation:Enum=ldiskfs;zfs
	BackFs string `json:"backFs,omitempty"`

	// MgsAddress is the NID of the MGS when a pre-existing MGS is
	// provided in the NnfStorageProfile
	MgsAddress string `json:"mgsAddress,omitempty"`

	// PersistentMgsReference is a reference to a persistent storage that is providing
	// the external MGS.
	PersistentMgsReference corev1.ObjectReference `json:"persistentMgsReference,omitempty"`
}

// NnfStorageAllocationSetSpec defines the details for an allocation set
type NnfStorageAllocationSetSpec struct {
	// Name is a human readable label for this set of allocations (e.g., xfs)
	Name string `json:"name"`

	// Capacity defines the capacity, in bytes, of this storage specification. The NNF Node itself
	// may split the storage among the available drives operating in the NNF Node.
	Capacity int64 `json:"capacity"`

	// Lustre specific configuration
	NnfStorageLustreSpec `json:",inline"`

	// SharedAllocation shares a single block storage allocation between multiple file system allocations
	// (within the same workflow) on a Rabbit
	SharedAllocation bool `json:"sharedAllocation"`

	// Nodes is the list of Rabbit nodes to make allocations on
	Nodes []NnfStorageAllocationNodes `json:"nodes"`
}

// NnfStorageSpec defines the specification for requesting generic storage on a set
// of available NNF Nodes. This object is related to a #DW for NNF Storage, with the WLM
// making the determination for which NNF Nodes it wants to utilize.
type NnfStorageSpec struct {

	// FileSystemType defines the type of the desired filesystem, or raw
	// block device.
	// +kubebuilder:validation:Enum=raw;lvm;zfs;xfs;gfs2;lustre
	// +kubebuilder:default:=raw
	FileSystemType string `json:"fileSystemType,omitempty"`

	// User ID for file system
	UserID uint32 `json:"userID"`

	// Group ID for file system
	GroupID uint32 `json:"groupID"`

	// AllocationSets is a list of different types of storage allocations to make. Each
	// AllocationSet describes an entire allocation spanning multiple Rabbits. For example,
	// an AllocationSet could be all of the OSTs in a Lustre filesystem, or all of the raw
	// block devices in a raw block configuration.
	AllocationSets []NnfStorageAllocationSetSpec `json:"allocationSets"`
}

// NnfStorageAllocationSetStatus contains the status information for an allocation set
type NnfStorageAllocationSetStatus struct {
	Ready bool `json:"ready,omitempty"`

	// AllocationCount is the total number of allocations that currently
	// exist
	AllocationCount int `json:"allocationCount"`
}

type NnfStorageLustreStatus struct {
	// MgsAddress is the NID of the MGS.
	MgsAddress string `json:"mgsAddress,omitempty"`

	// FileSystemName is the fsname parameter for the Lustre filesystem.
	// +kubebuilder:validation:MaxLength:=8
	FileSystemName string `json:"fileSystemName,omitempty"`

	// LustgreMgtReference is an object reference to the NnfLustreMGT resource used
	// by the NnfStorage
	LustreMgtReference corev1.ObjectReference `json:"lustreMgtReference,omitempty"`
}

// NnfStorageStatus defines the observed status of NNF Storage.
type NnfStorageStatus struct {
	NnfStorageLustreStatus `json:",inline"`

	// AllocationsSets holds the status information for each of the AllocationSets
	// from the spec.
	AllocationSets []NnfStorageAllocationSetStatus `json:"allocationSets,omitempty"`

	dwsv1alpha2.ResourceError `json:",inline"`

	// Ready reflects the status of this NNF Storage
	Ready bool `json:"ready,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
// +kubebuilder:storageversion
//+kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.ready"
//+kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
//+kubebuilder:printcolumn:name="ERROR",type="string",JSONPath=".status.error.severity"

// NnfStorage is the Schema for the storages API
type NnfStorage struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NnfStorageSpec   `json:"spec,omitempty"`
	Status NnfStorageStatus `json:"status,omitempty"`
}

func (s *NnfStorage) GetStatus() updater.Status[*NnfStorageStatus] {
	return &s.Status
}

//+kubebuilder:object:root=true

// NnfStorageList contains a list of Storage
type NnfStorageList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NnfStorage `json:"items"`
}

func (n *NnfStorageList) GetObjectList() []client.Object {
	objectList := []client.Object{}

	for i := range n.Items {
		objectList = append(objectList, &n.Items[i])
	}

	return objectList
}

func init() {
	SchemeBuilder.Register(&NnfStorage{}, &NnfStorageList{})
}
