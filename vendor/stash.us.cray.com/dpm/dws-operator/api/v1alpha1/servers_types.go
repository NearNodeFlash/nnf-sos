/*
Copyright 2021 Hewlett Packard Enterprise Development LP
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ServersSpec defines the desired state of Servers
type ServersSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of Servers. Edit servers_types.go to remove/update
	Foo string `json:"foo,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Servers is the Schema for the servers API
type Servers struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ServersSpec `json:"spec,omitempty"`
}

//+kubebuilder:object:root=true

// ServersList contains a list of Servers
type ServersList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Servers `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Servers{}, &ServersList{})
}
