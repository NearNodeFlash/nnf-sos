/*
 * Copyright 2023 Hewlett Packard Enterprise Development LP
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

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ContainerLabel = "nnf.cray.hpe.com/container"
)

// NnfContainerProfileSpec defines the desired state of NnfContainerProfile
type NnfContainerProfileData struct {
	// Pinned is true if this instance is an immutable copy
	// +kubebuilder:default:=false
	Pinned bool `json:"pinned,omitempty"`

	// List of possible filesystems supported by this container profile
	Storages []NnfContainerProfileStorage `json:"storages,omitempty"`

	// TODO: This is a development option for now. This will most likely be renamed or removed in
	// order to hide the k8s job implementation from the user.
	// Specifies the duration in seconds relative to the startTime that the job may be continuously
	// active before the system tries to terminate it; value must be positive integer. If a Job is
	// suspended (at creation or through an update), this timer will effectively be stopped and
	// reset when the Job is resumed again. +optional
	ActiveDeadlineSeconds int64 `json:"activeDeadlineSeconds,omitempty"`

	// TODO: This is a development option for now. This will most likely be renamed or removed in
	// order to hide the k8s job implementation from the user.
	// Specifies the number of retries before marking this job failed. Defaults to 6 by Kubernetes itself.
	// +kubebuilder:default:=6
	BackoffLimit int32 `json:"backoffLimit"`

	// Template defines the containers that will be created from container profile
	Template corev1.PodTemplateSpec `json:"template"`
}

// NnfContainerProfileStorage defines the mount point information that will be available to the
// container
type NnfContainerProfileStorage struct {
	// Name specifies the name of the mounted filesystem; must match the user supplied #DW directive
	Name string `json:"name"`

	// Optional designates that this filesystem is available to be mounted, but can be ignored by
	// the user not supplying this filesystem in the #DW directives
	//+kubebuilder:default:=false
	Optional bool `json:"optional"`
}

// +kubebuilder:object:root=true

// NnfContainerProfile is the Schema for the nnfcontainerprofiles API
type NnfContainerProfile struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Data NnfContainerProfileData `json:"data,omitempty"`
}

// +kubebuilder:object:root=true

// NnfContainerProfileList contains a list of NnfContainerProfile
type NnfContainerProfileList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NnfContainerProfile `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NnfContainerProfile{}, &NnfContainerProfileList{})
}
