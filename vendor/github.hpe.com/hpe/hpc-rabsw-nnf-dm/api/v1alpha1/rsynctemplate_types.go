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
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// RsyncTemplateSpec defines the desired state of RsyncTemplate
type RsyncTemplateSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Selector defines the pod selector used in scheduling the rsync nodes. This value is duplicated
	// to the template.spec.metadata.labels to satisfy the requirements of the Rsync Daemon Set.
	Selector metav1.LabelSelector `json:"selector"`

	// Template defines the pod template that is used for the basis of the Rsync Daemon set that
	// manages the Rsync Node Data Movement requests.
	Template v1.PodTemplateSpec `json:"template"`

	// Host Path defines the directory location of shared mounts on an individual rsync node.
	HostPath string `json:"hostPath"`

	// Mount Path defines the location within the container at which the Host Path volume should be mounted.
	MountPath string `json:"mountPath"`

	// Disable Lustre File Systems will disable the lustre file systems from within the generated
	// Rsync Daemon Set. The result is an Rsync Daemon Set without any volume mounts. This is
	// strictly used for testing and should be omitted during normal operation.
	DisableLustreFileSystems *bool `json:"disableLustreFileSystems,omitempty"`
}

// RsyncTemplateStatus defines the observed state of RsyncTemplate
type RsyncTemplateStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// RsyncTemplate is the Schema for the rsynctemplates API
type RsyncTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RsyncTemplateSpec   `json:"spec,omitempty"`
	Status RsyncTemplateStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// RsyncTemplateList contains a list of RsyncTemplate
type RsyncTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RsyncTemplate `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RsyncTemplate{}, &RsyncTemplateList{})
}
