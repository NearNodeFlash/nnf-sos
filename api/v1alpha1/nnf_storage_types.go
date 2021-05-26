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

	// Foo is an example field of Storage. Edit storage_types.go to remove/update
	Foo string `json:"foo,omitempty"`

	// Id defines the unique ID for this storage specification.
	Id string `json:"id"`

	// Capacity defines the desired size, in bytes, of storage allocation for
	// each server within the storage specification.
	Capacity int64 `json:"capacity"`

	// File System defines the file system that is constructed from underlying
	// storage subsystem.
	FileSystem string `json:"fileSystem,omitempty"`

	// Nodes define the list of NNF Nodes that will be the source
	// of storage provisioning and the servers that can utilize the storage
	Nodes []NnfStorageNodeSpec `json:"nodes,omitempty"`
}

type NnfStorageNodeSpec struct {
	// Name specifies the target NNF Node that will host the storage requested
	// by the specification.
	Name string `json:"name"`

	// Servers specifies the target NNF Servers that will be receiving the
	// file system requested by the specification.
	Servers []string `json:"servers"`
}

// StorageStatus defines the observed state of Storage
type NnfStorageStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Health string `json:"health"`

	State string `json:"state"`
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
