/*
 * Copyright 2023-2025 Hewlett Packard Enterprise Development LP
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
	dwsv1alpha5 "github.com/DataWorkflowServices/dws/api/v1alpha5"
	"github.com/DataWorkflowServices/dws/utils/updater"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type NnfNodeBlockStorageAllocationSpec struct {
	// Aggregate capacity of the block devices for each allocation
	Capacity int64 `json:"capacity,omitempty"`

	// List of nodes where /dev devices should be created
	Access []string `json:"access,omitempty"`
}

// NnfNodeBlockStorageSpec defines the desired storage attributes on a NNF Node.
// Storage spec are created on request of the user and fullfilled by the NNF Node Controller.
type NnfNodeBlockStorageSpec struct {
	// SharedAllocation is used when a single NnfNodeBlockStorage allocation is used by multiple NnfNodeStorage allocations
	SharedAllocation bool `json:"sharedAllocation"`

	// Allocations is the list of storage allocations to make
	Allocations []NnfNodeBlockStorageAllocationSpec `json:"allocations,omitempty"`
}

type NnfNodeBlockStorageStatus struct {
	// Allocations is the list of storage allocations that were made
	Allocations []NnfNodeBlockStorageAllocationStatus `json:"allocations,omitempty"`

	dwsv1alpha5.ResourceError `json:",inline"`

	// PodStartTime is the value of pod.status.containerStatuses[].state.running.startedAt from the pod that did
	// last successful full reconcile of the NnfNodeBlockStorage. This is used to tell whether the /dev paths
	// listed in the status section are from the current boot of the node.
	PodStartTime metav1.Time `json:"podStartTime,omitempty"`

	Ready bool `json:"ready"`
}

type NnfNodeBlockStorageDeviceStatus struct {
	// NQN of the base NVMe device
	NQN string `json:"NQN"`

	// Id of the Namespace on the NVMe device (e.g., "2")
	NamespaceId string `json:"namespaceId"`

	// Total capacity allocated for the storage. This may differ from the requested storage
	// capacity as the system may round up to the requested capacity to satisify underlying
	// storage requirements (i.e. block size / stripe size).
	CapacityAllocated int64 `json:"capacityAllocated,omitempty"`
}

type NnfNodeBlockStorageAccessStatus struct {
	// /dev paths for each of the block devices
	DevicePaths []string `json:"devicePaths,omitempty"`

	// Redfish ID for the storage group
	StorageGroupId string `json:"storageGroupId,omitempty"`
}

type NnfNodeBlockStorageAllocationStatus struct {
	// Accesses is a map of node name to the access status
	Accesses map[string]NnfNodeBlockStorageAccessStatus `json:"accesses,omitempty"`

	// List of NVMe namespaces used by this allocation
	Devices []NnfNodeBlockStorageDeviceStatus `json:"devices,omitempty"`

	// Total capacity allocated for the storage. This may differ from the requested storage
	// capacity as the system may round up to the requested capacity to satisify underlying
	// storage requirements (i.e. block size / stripe size).
	CapacityAllocated int64 `json:"capacityAllocated,omitempty"`

	// Redfish ID for the storage pool
	StoragePoolId string `json:"storagePoolId,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.ready"
// +kubebuilder:printcolumn:name="ERROR",type="string",JSONPath=".status.error.severity"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
type NnfNodeBlockStorage struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NnfNodeBlockStorageSpec   `json:"spec,omitempty"`
	Status NnfNodeBlockStorageStatus `json:"status,omitempty"`
}

func (ns *NnfNodeBlockStorage) GetStatus() updater.Status[*NnfNodeBlockStorageStatus] {
	return &ns.Status
}

// +kubebuilder:object:root=true

// NnfNodeBlockStorageList contains a list of NNF Nodes
type NnfNodeBlockStorageList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NnfNodeBlockStorage `json:"items"`
}

func (n *NnfNodeBlockStorageList) GetObjectList() []client.Object {
	objectList := []client.Object{}

	for i := range n.Items {
		objectList = append(objectList, &n.Items[i])
	}

	return objectList
}

func init() {
	SchemeBuilder.Register(&NnfNodeBlockStorage{}, &NnfNodeBlockStorageList{})
}
