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

package v1alpha8

import (
	dwsv1alpha6 "github.com/DataWorkflowServices/dws/api/v1alpha6"
	"github.com/DataWorkflowServices/dws/utils/updater"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// IMPORTANT: Run "make" to regenerate code after modifying this file
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// NnfNodeStorageSpec defines the desired storage attributes on a NNF Node.
// Storage spec are created on bequest of the user and fullfilled by the NNF Node Controller.
type NnfNodeStorageSpec struct {
	// Count is the number of allocations to make on this node. All of the allocations will
	// be created with the same parameters
	// +kubebuilder:validation:Minimum:=0
	Count int `json:"count"`

	// SharedAllocation is used when a single NnfNodeBlockStorage allocation is used by multiple NnfNodeStorage allocations
	SharedAllocation bool `json:"sharedAllocation"`

	// Capacity of an individual allocation
	Capacity int64 `json:"capacity,omitempty"`

	// User ID for file system
	UserID uint32 `json:"userID"`

	// Group ID for file system
	GroupID uint32 `json:"groupID"`

	// Extra variable substitutions to make for commands
	CommandVariables []CommandVariablesSpec `json:"commandVariables,omitempty"`

	// FileSystemType defines the type of the desired filesystem, or raw
	// block device.
	// +kubebuilder:validation:Enum=raw;lvm;zfs;xfs;gfs2;lustre
	// +kubebuilder:default:=raw
	FileSystemType string `json:"fileSystemType,omitempty"`

	// LustreStorageSpec describes the Lustre target created here, if
	// FileSystemType specifies a Lustre target.
	LustreStorage LustreStorageSpec `json:"lustreStorage,omitempty"`

	// BlockReference is an object reference to an NnfNodeBlockStorage
	BlockReference corev1.ObjectReference `json:"blockReference,omitempty"`
}

// LustreStorageSpec describes the Lustre target to be created here.
type LustreStorageSpec struct {
	// FileSystemName is the fsname parameter for the Lustre filesystem.
	// +kubebuilder:validation:MaxLength:=8
	FileSystemName string `json:"fileSystemName,omitempty"`

	// TargetType is the type of Lustre target to be created.
	// +kubebuilder:validation:Enum=mgt;mdt;mgtmdt;ost
	TargetType string `json:"targetType,omitempty"`

	// StartIndex is used to order a series of MDTs or OSTs.  This is used only
	// when creating MDT and OST targets. If count in the NnfNodeStorageSpec is more
	// than 1, then StartIndex is the index of the first allocation, and the indexes
	// increment from there.
	// +kubebuilder:validation:Minimum:=0
	StartIndex int `json:"startIndex,omitempty"`

	// MgsAddress is the NID of the MGS to use. This is used only when
	// creating MDT and OST targets.
	MgsAddress string `json:"mgsAddress,omitempty"`

	// BackFs is the type of backing filesystem to use.
	// +kubebuilder:validation:Enum=ldiskfs;zfs
	BackFs string `json:"backFs,omitempty"`

	// LustreComponents defines that list of NNF Nodes that are used for the components (e.g. OSTs)
	// in the lustre filesystem. This information is helpful when creating the lustre filesystem and
	// using PostMount commands (e.g. to set the striping).
	LustreComponents NnfStorageLustreComponents `json:"lustreComponents,omitempty"`
}

// NnfNodeStorageStatus defines the status for NnfNodeStorage
type NnfNodeStorageStatus struct {
	// Allocations is the list of storage allocations that were made
	Allocations []NnfNodeStorageAllocationStatus `json:"allocations,omitempty"`

	// Health is the health of all the allocations on the Rabbit
	Health NnfStorageHealth `json:"health,omitempty"`

	Ready bool `json:"ready,omitempty"`

	dwsv1alpha6.ResourceError `json:",inline"`
}

// NnfNodeStorageAllocationStatus defines the allocation status for each allocation in the NnfNodeStorage
type NnfNodeStorageAllocationStatus struct {
	// Name of the LVM VG
	VolumeGroup string `json:"volumeGroup,omitempty"`

	// Name of the LVM LV
	LogicalVolume string `json:"logicalVolume,omitempty"`

	// Health is the health of the individual allocation
	Health NnfStorageHealth `json:"health,omitempty"`

	Ready bool `json:"ready,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.ready"
// +kubebuilder:printcolumn:name="ERROR",type="string",JSONPath=".status.error.severity"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// NnfNodeStorage is the Schema for the NnfNodeStorage API
type NnfNodeStorage struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NnfNodeStorageSpec   `json:"spec,omitempty"`
	Status NnfNodeStorageStatus `json:"status,omitempty"`
}

func (ns *NnfNodeStorage) GetStatus() updater.Status[*NnfNodeStorageStatus] {
	return &ns.Status
}

//+kubebuilder:object:root=true

// NnfNodeStorageList contains a list of NNF Nodes
type NnfNodeStorageList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NnfNodeStorage `json:"items"`
}

func (n *NnfNodeStorageList) GetObjectList() []client.Object {
	objectList := []client.Object{}

	for i := range n.Items {
		objectList = append(objectList, &n.Items[i])
	}

	return objectList
}

func init() {
	SchemeBuilder.Register(&NnfNodeStorage{}, &NnfNodeStorageList{})
}
