/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// NnfNodeSpec defines the desired state of NNF Node
type NnfNodeSpec struct {
	// Important: Run "make" to regenerate code after modifying this file

	// The unique name for this NNF Node
	// https://connect.us.cray.com/confluence/display/HSOS/Shasta+HSS+Component+Naming+Convention#ShastaHSSComponentNamingConvention-2.1.4.3MountainCabinetComponents
	Name string `json:"name,omitempty"`

	// State reflects the desired state of this NNF Node resource
	// +kubebuilder:validation:Enum=Enable;Disable
	State NnfResourceStateType `json:"state,omitempty"`

	// Storage reflects the NNF Node Storage allocations for this NNF Node
	Storage []NnfNodeStorageSpec `json:"storage,omitempty"`
}

// NnfNodeStorageSpec defines the desired storage on the NNF Node. Storage spec are created
// on bequest of the user and fullfilled by the NNF Node Controller.
type NnfNodeStorageSpec struct {

	// Uuid defines the unique identifer for this storage request. The client is responsible for
	// ensuing the Uuid is unique for all storage specifications on this NNF Node.
	Uuid string `json:"uuid"`

	// Capacity defines the capacity, in bytes, of this storage specification. The NNF Node itself
	// may split the storage among the available drives operating in the NNF Node.
	Capacity int64 `json:"capacity"`

	// FileSystem defines the desired file system that is to be established for this Storage
	// specification. An abscence of a file system means raw block storage will be allocated
	// and made available to the desired servers.
	FileSystem string `json:"fileSystem,omitempty"`

	// State reflects the desired state of the storage specification
	// +kubebuilder:validation:Enum=Create;Destroy
	State NnfResourceStateType `json:"state,omitempty"`

	// Servers is a list of NNF connected Servers that are to receive this NNF Storage resource. A
	// valid server will receive the physical storage that has been allocated via raw block or the
	// desired file system.
	Servers []NnfNodeStorageServerSpec `json:"servers,omitempty"`
}

// NnfNodeStorageServerSpec defines the desired server status for a given NNF Storage specification.
type NnfNodeStorageServerSpec struct {

	// Id defines the NNF Node unique identifier for this NNF Server resource. A valid Id field
	// will receive the storage allocated by the parent NNF Storage specification.
	Id string `json:"id"`

	// Path defines an option path for which the resource is made available on the NNF Server
	// resource. A valid path must adhear to the system's directory name rules and conventions and
	// cannot already exist on the system. The path is analogous to the mountpoint of the file system.
	Path string `json:"string,omitempty"`
}

// NnfNodeStatus defines the observed status of NNF Node
type NnfNodeStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Status reflects the current status of the NNF Node
	Status NnfResourceStatusType `json:"status,omitempty"`

	Health NnfResourceHealthType `json:"health,omitempty"`

	Capacity          int64 `json:"capacity,omitempty"`
	CapacityAllocated int64 `json:"capacityAllocated,omitempty"`

	Servers []NnfServerStatus `json:"servers,omitempty"`

	Storage []NnfNodeStorageStatus `json:"storage,omitempty"`
}

// NnfServerStatus defines the observed state of servers connected to this NNF Node
type NnfServerStatus struct {

	// Id reflects the NNF Node unique identifier for this NNF Server resource.
	Id string `json:"id,omitempty"`

	// Name reflects the common name of this NNF Server resource.
	Name string `json:"name,omitempty"`

	Status NnfResourceStatusType `json:"status,omitempty"`

	Health NnfResourceHealthType `json:"health,omitempty"`
}

type NnfNodeStorageStatus struct {

	// Reflects the unique identifier for the storage. When a storage spec is received
	// the controller will create a status object with the same UUID.
	Uuid string `json:"uuid,omitempty"`

	// Reflects the ID of the storage. Used internally to the controller and can be
	// safely ignored by the client.
	Id string `json:"id,omitempty"`

	// The current state of the storage
	Status NnfResourceStatusType `json:"status,omitempty"`

	// The current health of the storage
	Health NnfResourceHealthType `json:"health,omitempty"`

	// Total capacity allocated for the storage. This may differ than the requested storage
	// capacity as the system may round up to the requested capacity satisifiy underlying
	// storage requirements (i.e. block size / stripe size).
	CapacityAllocated int64 `json:"capacityAllocated,omitempty"`

	// Represents the used capacity. This is an aggregation of all the used capacities of
	// the underlying storage objects.
	CapacityUsed int64 `json:"capacityUsed,omitempty"`

	// Represents the time when the storage was created by the controller
	// It is represented in RFC3339 form and is in UTC.
	CreationTime *metav1.Time `json:"creationTime,omitempty"`

	// Represents the time when the storage was deleted by the controller. This field
	// is updated when the Storage specification State transitions to 'Delete' by the
	// client.
	// It is represented in RFC3339 form and is in UTC.
	DeletionTime *metav1.Time `json:"deletionTime,omitempty"`

	Servers []NnfNodeStorageServerStatus `json:"servers,omitempty"`

	Drives []NnfNodeStorageDriveStatus `json:"drives,omitempty"`

	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

type NnfNodeStorageServerStatus struct {
	NnfNodeStorageServerSpec `json:",inline"`

	Status NnfResourceStatusType `json:"status,omitempty"`

	Health NnfResourceHealthType `json:"health,omitempty"`

	// Represents the storage group that is supporting this server. A storage group is
	// the mapping from a group of drive namespaces to an individual server. This value
	// can be safely ignored by the client.
	StorageGroup NnfNodeStorageServerStorageGroupStatus `json:"storageGroup,omitempty"`

	// Represents the file share that is supporting this server. A file share is the
	// combination of a storage group and the associated file system parameters (type, mountpoint)
	// that makes up the available storage.
	FileShare NnfNodeStorageServerFileShareStatus `json:"fileShare,omitempty"`
}

type NnfNodeStorageServerStorageGroupStatus struct {
	Id string `json:"id,omitempty"`

	Status NnfResourceStatusType `json:"status,omitempty"`

	Health NnfResourceHealthType `json:"health,omitempty"`
}

type NnfNodeStorageServerFileShareStatus struct {
	Id string `json:"id,omitempty"`

	Status NnfResourceStatusType `json:"status,omitempty"`

	Health NnfResourceHealthType `json:"health,omitempty"`
}

type NnfNodeStorageDriveStatus struct {
	Id string `json:"id,omitempty"`

	Name string `json:"name,omitempty"`

	Status NnfResourceStatusType `json:"status,omitempty"`

	Health NnfResourceHealthType `json:"health,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// NnfNode is the Schema for the NnfNode API
type NnfNode struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NnfNodeSpec   `json:"spec,omitempty"`
	Status NnfNodeStatus `json:"status,omitempty"`
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
