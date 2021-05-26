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

// StorageSpec defines the desired state of Storage
type NnfStorageSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Id defines the unique ID for this storage specification.
	Id string `json:"id,omitempty"`

	// Capacity defines the desired size, in bytes, of storage allocation for
	// each server within the storage specification.
	Capacity int64 `json:"capacity"`

	// File System defines the file system that is constructed from underlying
	// storage subsystem.
	FileSystem string `json:"fileSystem,omitempty"`

	// Nodes define the list of NNF Nodes that will be the source
	// of storage provisioning and the servers that can utilize the storage
	Nodes []NnfStorageNodeSpec `json:"nodes"`
}

// NnfStorageNodeSpec defines the desired state of an individual NNF Storage
// Node.
type NnfStorageNodeSpec struct {
	// Name specifies the target NNF Node that will host the storage requested
	// by the specification.
	Name string `json:"name"`

	// Servers specifies the target NNF Servers that will be receiving the
	// file system requested by the specification.
	Servers []NnfStorageServerSpec `json:"servers"`
}

// NnfStorageServerSpec defines the desired state of storage presented to
// a server connecting to the NNF Storage Node.
type NnfStorageServerSpec struct {
	// The unique name of the NNF Storage Server, as provided by the NNF Node
	// Server value for a given NNF Node. The Server shall receive the
	// storage provisioned on this NNF Storage Node.
	Name string `json:"name"`

	// The unique Id for the NNF Storage Server, as provided by NNF Node
	// Server value for a particular NNF Node
	Id string `json:"id"`

	// Defines the desired path for the provided storage onto
	// the server listed in Name
	Path string `json:"path,omitempty"`
}

// NnfStorageStatus defines the observed state of NNF Storage
type NnfStorageStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Health string `json:"health,omitempty"`

	State string `json:"state,omitempty"`

	Nodes []NnfStorageNodeStatus `json:"nodes"`
}

// NnfStorageNodeStatus defines the observed state of a NNF Storage
// present on a NNF Node.
type NnfStorageNodeStatus struct {
	Id string `json:"id,omitempty"`

	Health string `json:"health,omitempty"`

	State string `json:"state,omitempty"`

	Servers []NnfStorageServerStatus `json:"servers"`
}

// NnfStorageServerStatus defines the observed state of NNF Storage
// present on a NNF Server.
type NnfStorageServerStatus struct {
	NnfStorageServerSpec `json:",inline"`

	StorageId string `json:"storageId,omitempty"`

	ShareId string `json:"shareId,omitempty"`

	Health string `json:"health,omitempty"`

	State string `json:"state,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Storage is the Schema for the storages API
type NnfStorage struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NnfStorageSpec   `json:"spec,omitempty"`
	Status NnfStorageStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// StorageList contains a list of Storage
type NnfStorageList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NnfStorage `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NnfStorage{}, &NnfStorageList{})
}
