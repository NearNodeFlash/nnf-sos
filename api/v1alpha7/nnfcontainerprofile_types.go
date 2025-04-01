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

package v1alpha7

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ContainerLabel   = "nnf.cray.hpe.com/container"
	ContainerUser    = "user"
	ContainerMPIUser = "mpiuser"
)

// NnfContainerProfileSpec defines the desired state of NnfContainerProfile
type NnfContainerProfileData struct {
	// Pinned is true if this instance is an immutable copy
	// +kubebuilder:default:=false
	Pinned bool `json:"pinned,omitempty"`

	// List of possible filesystems supported by this container profile
	Storages []NnfContainerProfileStorage `json:"storages,omitempty"`

	// Containers are launched in the PreRun state. Allow this many seconds for the containers to
	// start before declaring an error to the workflow.
	// Defaults to 300 if not set. A value of 0 disables this behavior.
	// +kubebuilder:default:=300
	// +kubebuilder:validation:Minimum:=0
	PreRunTimeoutSeconds *int64 `json:"preRunTimeoutSeconds,omitempty"`

	// Containers are expected to complete in the PostRun State. Allow this many seconds for the
	// containers to exit before declaring an error the workflow.
	// Defaults to 300 if not set. A value of 0 disables this behavior.
	// +kubebuilder:default:=300
	// +kubebuilder:validation:Minimum:=0
	PostRunTimeoutSeconds *int64 `json:"postRunTimeoutSeconds,omitempty"`

	// Specifies the number of times a container will be retried upon a failure. A new pod is
	// deployed on each retry. Defaults to 6 by kubernetes itself and must be set. A value of 0
	// disables retries.
	// +kubebuilder:validation:Minimum:=0
	// +kubebuilder:default:=6
	RetryLimit int32 `json:"retryLimit"`

	// UserID specifies the user ID that is allowed to use this profile. If this is specified, only
	// Workflows that have a matching user ID can select this profile.
	UserID *uint32 `json:"userID,omitempty"`

	// GroupID specifies the group ID that is allowed to use this profile. If this is specified,
	// only Workflows that have a matching group ID can select this profile.
	GroupID *uint32 `json:"groupID,omitempty"`

	// Number of ports to open for communication with the user container. These ports are opened on
	// the targeted NNF nodes and can be accessed outside of the k8s cluster (e.g. compute nodes).
	// The requested ports are made available as environment variables inside the container and in
	// the DWS workflow (NNF_CONTAINER_PORTS).
	NumPorts int32 `json:"numPorts,omitempty"`

	// Spec to define the containers created from this profile. This is used for non-MPI containers.
	// Either this or MPISpec must be provided, but not both.
	// +kubebuilder:validation:Rule="(self.spec != null) != (self.mpiSpec != null)",Message="Exactly one of 'spec' or 'mpiSpec' must be set."
	Spec *NnfContainerSpec `json:"spec,omitempty"`

	// MPIJobSpec to define the MPI containers created from this profile.
	// Either this or Spec must be provided, but not both.
	// +kubebuilder:validation:Rule="(self.spec != null) != (self.mpiSpec != null)",Message="Exactly one of 'spec' or 'mpiSpec' must be set."
	MPISpec *NnfMPIContainerSpec `json:"mpiSpec,omitempty"`
}

// TODO: Add validation for NnfContainerSpec
type NnfContainerSpec struct {
	Containers     []NnfContainer  `json:"containers"`
	InitContainers []NnfContainer  `json:"initContainers,omitempty"`
	Volumes        []corev1.Volume `json:"volumes,omitempty"`
}

type NnfMPIContainerSpec struct {
	Launcher    NnfContainerSpec `json:"launcher"`
	Worker      NnfContainerSpec `json:"worker"`
	CopyOffload bool             `json:"copyOffload,omitempty"`
}

type NnfContainer struct {
	Name         string                 `json:"name"`
	Image        string                 `json:"image"`
	Command      []string               `json:"command"`
	Args         []string               `json:"args,omitempty"`
	Env          []corev1.EnvVar        `json:"env,omitempty"`
	EnvFrom      []corev1.EnvFromSource `json:"envFrom,omitempty"`
	VolumeMounts []corev1.VolumeMount   `json:"volumeMounts,omitempty"`
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

	// For DW_GLOBAL_ (global lustre) storages, the access mode must match what is configured in
	// the LustreFilesystem resource for the namespace. Defaults to `ReadWriteMany` for global
	// lustre, otherwise empty.
	PVCMode corev1.PersistentVolumeAccessMode `json:"pvcMode,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion

// NnfContainerProfile is the Schema for the nnfcontainerprofiles API
type NnfContainerProfile struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Data NnfContainerProfileData `json:"data"`
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion

// NnfContainerProfileList contains a list of NnfContainerProfile
type NnfContainerProfileList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NnfContainerProfile `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NnfContainerProfile{}, &NnfContainerProfileList{})
}

// Copy an NnfContainerSpec into a corev1.PodSpec
func (s *NnfContainerSpec) DeepCopyIntoCore(out *corev1.PodSpec) {
	out.Containers = make([]corev1.Container, len(s.Containers))
	for i := range s.Containers {
		s.Containers[i].DeepCopyIntoCore(&out.Containers[i])
	}

	out.InitContainers = make([]corev1.Container, len(s.InitContainers))
	for i := range s.InitContainers {
		s.InitContainers[i].DeepCopyIntoCore(&out.InitContainers[i])
	}

	out.Volumes = make([]corev1.Volume, len(s.Volumes))
	for i := range s.Volumes {
		s.Volumes[i].DeepCopyInto(&out.Volumes[i])
	}
}

// Copy an NnfContainer into a corev1.Container
func (s *NnfContainer) DeepCopyIntoCore(out *corev1.Container) {
	out.Name = s.Name
	out.Image = s.Image

	out.Command = make([]string, len(s.Command))
	copy(out.Command, s.Command)

	out.Args = make([]string, len(s.Args))
	copy(out.Args, s.Args)

	out.Env = make([]corev1.EnvVar, len(s.Env))
	for i := range s.Env {
		s.Env[i].DeepCopyInto(&out.Env[i])
	}

	out.EnvFrom = make([]corev1.EnvFromSource, len(s.EnvFrom))
	for i := range s.EnvFrom {
		s.EnvFrom[i].DeepCopyInto(&out.EnvFrom[i])
	}

	out.VolumeMounts = make([]corev1.VolumeMount, len(s.VolumeMounts))
	for i := range s.VolumeMounts {
		s.VolumeMounts[i].DeepCopyInto(&out.VolumeMounts[i])
	}
}
