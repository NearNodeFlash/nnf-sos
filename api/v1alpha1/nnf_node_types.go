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

// NnfNodeSpec defines the desired state of NNNF Node
type NnfNodeSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// The unique name for this NNF Node
	// https://connect.us.cray.com/confluence/display/HSOS/Shasta+HSS+Component+Naming+Convention#ShastaHSSComponentNamingConvention-2.1.4.3MountainCabinetComponents
	Name string `json:"name,omitempty"`

	// State reflects the desired state of this NNF Node
	// TODO: Publish State definitions
	State string `json:"state,omitempty"`
}

// NnfNodeStatus defines the observed status of NNF Node
type NnfNodeStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Health reflects the current health of this NNF NnfNode
	// TODO: Link to health spec
	Health string `json:"health"`

	// State reflects the current state of this NNF NnfNode
	State string `json:"state"`

	Capacity          int64 `json:"capacity"`
	CapacityAllocated int64 `json:"capacityAllocated"`

	Servers []NnfServerStatus `json:"servers"`
}

// NnfServerStatus defines the observed state of servers connected to this NNF Node
type NnfServerStatus struct {
	Id string `json:"id"`

	Name string `json:"name"`

	Health string `json:"health"`

	State string `json:"state"`
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
