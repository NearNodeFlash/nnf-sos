/*
Copyright 2021 Hewlett Packard Enterprise Development LP
*/

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// IMPORTANT: Run "make" to regenerate code after modifying this file
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// NNF Node Storage Specifications defines the desired storage attributes on a NNF Node.
// Storage spec are created on bequest of the user and fullfilled by the NNF Node Controller.
type NnfNodeStorageSpec struct {

	// Owner points to the NNF Storage that owns this NNF Node Storage.
	Owner corev1.ObjectReference `json:"owner,omitempty"`

	// Capacity defines the capacity, in bytes, of this storage specification. The NNF Node itself
	// may split the storage among the available drives operating in the NNF Node.
	Capacity int64 `json:"capacity,omitempty"`

	// FileSystem defines the desired file system that is to be established for this Storage
	// specification.
	// +kubebuilder:validation:MaxLength:=8
	FileSystemName string `json:"fileSystemName,omitempty"`

	// FileSystemType defines the type of the desired filesystem, or raw
	// block device.
	// +kubebuilder:validation:Enum=raw;lvm;zfs;lustre
	// +kubebuilder:default:=raw
	FileSystemType string `json:"fileSystemType"`

	// LustreStorageSpec describes the Lustre target created here, if
	// FileSystemType specifies a Lustre target.
	LustreStorage LustreStorageSpec `json:"lustreStorage,omitempty"`

	// Servers is a list of NNF connected Servers that are to receive this NNF Storage resource. A
	// valid server will receive the physical storage that has been allocated via raw block or the
	// desired file system.
	Servers []NnfNodeStorageServerSpec `json:"servers,omitempty"`
}

// NNF Node Storage Server Spec defines the desired server state for a given NNF Storage specification.
type NnfNodeStorageServerSpec struct {

	// Id defines the NNF Node unique identifier for this NNF Server resource. A valid Id field
	// will receive the storage allocated by the parent NNF Storage specification.
	Id string `json:"id,omitempty"`

	// Name defines the name of the NNF Server resource. If provided, the name is validated against
	// the supplied Id. Can be used to identify a NNF Server resource by a human-readable value.
	// TODO: Validate name/id pair in nnf_node_controller.go
	Name string `json:"name,omitempty"`

	// Path defines an option path for which the resource is made available on the NNF Server
	// resource. A valid path must adhear to the system's directory name rules and conventions and
	// cannot already exist on the system. The path is analogous to the mountpoint of the file system.
	Path string `json:"path,omitempty"`
}

// LustreStorageSpec describes the Lustre target to be created here.
type LustreStorageSpec struct {

	// TargetType is the type of Lustre target to be created.
	// +kubebuilder:validation:Enum=MGT;MDT;OST
	TargetType string `json:"targetType"`

	// Index is used to order a series of MDTs or OSTs.  This is used only
	// when creating MDT and OST targets.
	// +kubebuilder:validation:Minimum:=0
	Index int `json:"index,omitempty"`

	// MgsNode is the NID of the MGS to use. This is used only when
	// creating MDT and OST targets.
	MgsNode string `json:"mgsNode,omitempty"`

	// Type of backing filesystem to use.
	// +kubebuilder:validation:Enum=ldiskfs;zfs
	// +kubebuilder:default:=ldiskfs
	BackFs string `json:"backFs,omitempty"`
}

// NNF Node Storage Status
type NnfNodeStorageStatus struct {
	// Reflects the unique identifier for the storage. When a storage spec is received
	// the controller will create a status object with the same UUID.
	Uuid string `json:"uuid,omitempty"`

	NnfResourceStatus `json:",inline"`

	// Total capacity allocated for the storage. This may differ than the requested storage
	// capacity as the system may round up to the requested capacity satisifiy underlying
	// storage requirements (i.e. block size / stripe size).
	CapacityAllocated int64 `json:"capacityAllocated,omitempty"`

	// Represents the used capacity. This is an aggregation of all the used capacities of
	// the underlying storage objects.
	CapacityUsed int64 `json:"capacityUsed,omitempty"`

	// LustreStorageStatus describes the Lustre target created here.
	LustreStorage LustreStorageStatus `json:"lustreStorage,omitempty"`

	// Represents the time when the storage was created by the controller
	// It is represented in RFC3339 form and is in UTC.
	CreationTime *metav1.Time `json:"creationTime,omitempty"`

	// Represents the time when the storage was deleted by the controller. This field
	// is updated when the Storage specification State transitions to 'Delete' by the
	// client.
	// It is represented in RFC3339 form and is in UTC.
	DeletionTime *metav1.Time `json:"deletionTime,omitempty"`

	Servers []NnfNodeStorageServerStatus `json:"servers,omitempty"`

	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

type NnfNodeStorageServerStatus struct {
	NnfResourceStatus `json:",inline"`

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
	NnfResourceStatus `json:",inline"`
}

type NnfNodeStorageServerFileShareStatus struct {
	NnfResourceStatus `json:",inline"`
}

// LustreStorageStatus describes the Lustre target created here.
type LustreStorageStatus struct {

	// Nid (LNet Network Identifier) of this node. This is populated on MGS nodes only.
	Nid string `json:"nid,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// NnfNodeStorage is the Schema for the NnfNodeStorage API
type NnfNodeStorage struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NnfNodeStorageSpec   `json:"spec,omitempty"`
	Status NnfNodeStorageStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// NnfNodeStorageList contains a list of NNF Nodes
type NnfNodeStorageList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NnfNodeStorage `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NnfNodeStorage{}, &NnfNodeStorageList{})
}
