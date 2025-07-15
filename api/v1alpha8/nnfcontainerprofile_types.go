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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

const (
	ContainerLabel                = "nnf.cray.hpe.com/container"
	ContainerUser                 = "user"
	ContainerMPIUser              = "mpiuser"
	CopyOffloadServiceAccountName = "nnf-dm-copy-offload"
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

	// Number of ports to open for each container specified in the PodSpec. For MPI Jobs, this is
	// only for the Launcher container(s) listed in the MPIReplicaSet's PodSpec. These ports are
	// opened on the targeted NNF nodes and can be accessed outside the k8s cluster (e.g. compute
	// nodes). The requested ports are made available as environment variables inside the container
	// and in the DWS workflow (NNF_CONTAINER_PORTS).
	NumPorts int32 `json:"numPorts,omitempty"`

	// NnfSpec to define the containers created from this profile. This is used for non-MPI containers.
	// Either this or NnfMPISpec must be provided, but not both.
	// +kubebuilder:validation:Rule="(self.spec != null) != (self.mpiSpec != null)",Message="Exactly one of 'spec' or 'mpiSpec' must be set."
	NnfSpec *NnfPodSpec `json:"spec,omitempty"`

	// MPIJobSpec to define the MPI containers created from this profile.
	// Either this or NnfSpec must be provided, but not both.
	// +kubebuilder:validation:Rule="(self.spec != null) != (self.mpiSpec != null)",Message="Exactly one of 'spec' or 'mpiSpec' must be set."
	NnfMPISpec *NnfMPISpec `json:"mpiSpec,omitempty"`
}

// NnfPodSpec represents the specification of a pod that can be used in a container profile. This is
// a slimmed down version of a corev1.PodSpec to reduce the size of the CRD.
type NnfPodSpec struct {
	// Containers are the list of containers that will be created in the pod.
	// +kubebuilder:validation:MinItems=1
	Containers []NnfContainer `json:"containers"`

	// InitContainers are the list of init containers that will be created in the pod before the
	// main containers.
	InitContainers []NnfContainer `json:"initContainers,omitempty"`

	// Volumes are the list of volumes that will be available to the pod
	Volumes []corev1.Volume `json:"volumes,omitempty"`

	// TerminationGracePeriodSeconds is the duration in seconds the pod needs to terminate gracefully.
	TerminationGracePeriodSeconds *int64 `json:"terminationGracePeriodSeconds,omitempty"`

	// ShareProcessNamespace indicates whether the containers in the pod share a process namespace.
	ShareProcessNamespace *bool `json:"shareProcessNamespace,omitempty"`

	// ImagePullSecrets is a list of references to secrets for pulling images.
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`

	// AutomountServiceAccountToken indicates whether the service account token should be mounted.
	AutomountServiceAccountToken *bool `json:"automountServiceAccountToken,omitempty"`
}

// NnfMPISpec represents the specification of an MPI job that can be used in a container profile.
type NnfMPISpec struct {

	// Launcher is the specification for the launcher container in the MPI job. In a typical MPI
	// job, the launcher runs an MPI application with mpirun and contacts the workers to distribute
	// the job.
	Launcher NnfPodSpec `json:"launcher"`

	// Worker is the specification for the worker containers in the MPI job. In a typical MPI job,
	// the workers are running sshd and listening for the launcher to connect.
	Worker NnfPodSpec `json:"worker"`

	// CopyOffload indicates that this profile is configured to drive the NNF Copy Offload API. This
	// instructions the NNF software to configure specifies for the Copy Offload API (e.g.
	// serviceAccount).
	// +kubebuilder:default:=false
	CopyOffload bool `json:"copyOffload,omitempty"`

	// Specifies the number of slots per worker used in hostfile.
	// Note: This is only for container directives that do not use Copy Offload. For Copy Offload
	// API, the slots field in the specific Data Movement profile will be used instead of this
	// value.
	SlotsPerWorker *int32 `json:"slotsPerWorker,omitempty"`
}

// NnfContainer defines the specification of a container that can be used in a container profile.
// This is a slimmed down version of a corev1.Container to reduce the size of the CRD.
type NnfContainer struct {
	// Name of the container specified as a DNS_LABEL. Each container in a pod must have a unique
	// name (DNS_LABEL).
	// +kubebuilder:validation:Pattern="^[a-z0-9]([-a-z0-9]*[a-z0-9])?$"
	// +kubebuilder:validation:MaxLength=253
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Container image name. More info: https://kubernetes.io/docs/concepts/containers/images
	// +kubebuilder:validation:MaxLength=253
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Required
	Image string `json:"image"`

	// Entrypoint array. Not executed within a shell. The container image's ENTRYPOINT is used if
	// this is not provided. Variable references $(VAR_NAME) are expanded using the container's
	// environment. If a variable cannot be resolved, the reference in the input string will be
	// unchanged. Double $$ are reduced to a single $, which allows for escaping the $(VAR_NAME)
	// syntax: i.e. "$$(VAR_NAME)" will produce the string literal "$(VAR_NAME)". Escaped references
	// will never be expanded, regardless of whether the variable exists or not.
	// More info:
	// https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell
	// +kubebuilder:validation:Optional
	Command []string `json:"command"`

	// Arguments to the entrypoint. The container image's CMD is used if this is not provided.
	// Variable references $(VAR_NAME) are expanded using the container's environment. If a variable
	// cannot be resolved, the reference in the input string will be unchanged. Double $$ are
	// reduced to a single $, which allows for escaping the $(VAR_NAME) syntax: i.e. "$$(VAR_NAME)"
	// will produce the string literal "$(VAR_NAME)". Escaped references will never be expanded,
	// regardless of whether the variable exists or not. More info:
	// https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell
	Args []string `json:"args,omitempty"`

	// List of environment variables to set in the container.
	Env []corev1.EnvVar `json:"env,omitempty"`

	// List of sources to populate environment variables in the container. The keys defined within a
	// source must be a C_IDENTIFIER. All invalid keys will be reported as an event when the
	// container is starting. When a key exists in multiple sources, the value associated with the
	// last source will take precedence. Values defined by an Env with a duplicate key will take
	// precedence.
	EnvFrom []corev1.EnvFromSource `json:"envFrom,omitempty"`

	// Pod volumes to mount into the container's filesystem. NNF Volumes will be patched in from the
	// Storages field.
	VolumeMounts []corev1.VolumeMount `json:"volumeMounts,omitempty"`

	// Container's working directory.
	// +kubebuilder:validation:MaxLength=253
	WorkingDir string `json:"workingDir,omitempty"`

	// RestartPolicy defines the restart behavior of individual containers in a pod. This field may
	// only be set for init containers, and the only allowed value is "Always". For non-init
	// containers or when this field is not specified, the restart behavior is defined by the Pod's
	// restart policy and the container type.
	RestartPolicy *corev1.ContainerRestartPolicy `json:"restartPolicy,omitempty"`

	// Periodic probe of container liveness.
	// Container will be restarted if the probe fails.
	LivenessProbe *corev1.Probe `json:"livenessProbe,omitempty"`

	// Periodic probe of container service readiness.
	// Container will be removed from service endpoints if the probe fails.
	ReadinessProbe *corev1.Probe `json:"readinessProbe,omitempty"`

	// StartupProbe indicates that the Pod has successfully initialized.
	// If specified, no other probes are executed until this completes successfully.
	// If this probe fails, the Pod will be restarted, just as if the livenessProbe failed.
	// This can be used to provide different probe parameters at the beginning of a Pod's lifecycle,
	// when it might take a long time to load data or warm a cache, than during steady-state operation.
	StartupProbe *corev1.Probe `json:"startupProbe,omitempty"`

	// Path at which the file to which the container's termination message
	// will be written is mounted into the container's filesystem.
	// Message written is intended to be brief final status, such as an assertion failure message.
	// Will be truncated by the node if greater than 4096 bytes. The total message length across
	// all containers will be limited to 12kb.
	// Defaults to /dev/termination-log.
	TerminationMessagePath string `json:"terminationMessagePath,omitempty"`

	// Indicate how the termination message should be populated. File will use the contents of
	// terminationMessagePath to populate the container status message on both success and failure.
	// FallbackToLogsOnError will use the last chunk of container log output if the termination
	// message file is empty and the container exited with an error.
	// The log output is limited to 2048 bytes or 80 lines, whichever is smaller.
	// Defaults to File.
	TerminationMessagePolicy corev1.TerminationMessagePolicy `json:"terminationMessagePolicy,omitempty"`

	// Image pull policy for the container.  One of Always, Never, IfNotPresent. Set this to Always
	// if :latest tag is specified in the image field.
	// Defaults to IfNotPresent
	// +kubebuilder:validation:Enum=Always;Never;IfNotPresent
	// +kubebuilder:default:=IfNotPresent
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`
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

// Convert an NnfPodSpec into a corev1.PodSpec and return it.
func (s *NnfPodSpec) ToCorePodSpec() *corev1.PodSpec {
	if s == nil {
		return nil
	}

	out := &corev1.PodSpec{}

	if len(s.Containers) > 0 {
		out.Containers = make([]corev1.Container, len(s.Containers))
		for i := range s.Containers {
			out.Containers[i] = *s.Containers[i].ToCoreContainer()
		}
	}

	if len(s.InitContainers) > 0 {
		out.InitContainers = make([]corev1.Container, len(s.InitContainers))
		for i := range s.InitContainers {
			out.InitContainers[i] = *s.InitContainers[i].ToCoreContainer()
		}
	}

	if len(s.Volumes) > 0 {
		out.Volumes = make([]corev1.Volume, len(s.Volumes))
		for i := range s.Volumes {
			s.Volumes[i].DeepCopyInto(&out.Volumes[i])
		}
	}

	if s.TerminationGracePeriodSeconds != nil {
		out.TerminationGracePeriodSeconds = pointer.Int64(*s.TerminationGracePeriodSeconds)
	}

	if s.ShareProcessNamespace != nil {
		out.ShareProcessNamespace = pointer.Bool(*s.ShareProcessNamespace)
	}

	out.ImagePullSecrets = append([]corev1.LocalObjectReference{}, s.ImagePullSecrets...)

	if s.AutomountServiceAccountToken != nil {
		out.AutomountServiceAccountToken = pointer.Bool(*s.AutomountServiceAccountToken)
	}

	return out
}

// Convert an NnfContainer into a corev1.Container and return it
func (s *NnfContainer) ToCoreContainer() *corev1.Container {
	if s == nil {
		return nil
	}

	out := &corev1.Container{
		Name:                     s.Name,
		Image:                    s.Image,
		Command:                  append([]string{}, s.Command...),
		Args:                     append([]string{}, s.Args...),
		Env:                      append([]corev1.EnvVar{}, s.Env...),
		EnvFrom:                  append([]corev1.EnvFromSource{}, s.EnvFrom...),
		VolumeMounts:             append([]corev1.VolumeMount{}, s.VolumeMounts...),
		WorkingDir:               s.WorkingDir,
		TerminationMessagePath:   s.TerminationMessagePath,
		TerminationMessagePolicy: s.TerminationMessagePolicy,
		ImagePullPolicy:          s.ImagePullPolicy,
	}

	if s.RestartPolicy != nil {
		out.RestartPolicy = new(corev1.ContainerRestartPolicy)
		*out.RestartPolicy = *s.RestartPolicy
	}

	if s.LivenessProbe != nil {
		out.LivenessProbe = s.LivenessProbe.DeepCopy()
	}

	if s.ReadinessProbe != nil {
		out.ReadinessProbe = s.ReadinessProbe.DeepCopy()
	}

	if s.StartupProbe != nil {
		out.StartupProbe = s.StartupProbe.DeepCopy()
	}

	return out
}

// Copy a corev1.PodSpec into an NnfPodSpec.
func (s *NnfPodSpec) FromCorePodSpec(in *corev1.PodSpec) {
	if in == nil {
		return
	}

	s.Containers = make([]NnfContainer, len(in.Containers))
	for i := range in.Containers {
		s.Containers[i].FromCoreContainer(&in.Containers[i])
	}

	s.InitContainers = make([]NnfContainer, len(in.InitContainers))
	for i := range in.InitContainers {
		s.InitContainers[i].FromCoreContainer(&in.InitContainers[i])
	}

	s.Volumes = make([]corev1.Volume, len(in.Volumes))
	for i := range in.Volumes {
		in.Volumes[i].DeepCopyInto(&s.Volumes[i])
	}

	if in.TerminationGracePeriodSeconds != nil {
		s.TerminationGracePeriodSeconds = pointer.Int64(*in.TerminationGracePeriodSeconds)
	}

	if in.ShareProcessNamespace != nil {
		s.ShareProcessNamespace = pointer.Bool(*in.ShareProcessNamespace)
	}

	s.ImagePullSecrets = append([]corev1.LocalObjectReference{}, in.ImagePullSecrets...)

	if in.AutomountServiceAccountToken != nil {
		s.AutomountServiceAccountToken = pointer.Bool(*in.AutomountServiceAccountToken)
	}
}

// Copy a corev1.Container into an NnfContainer
func (s *NnfContainer) FromCoreContainer(in *corev1.Container) {
	if in == nil {
		return
	}

	s.Name = in.Name
	s.Image = in.Image
	s.Command = append([]string{}, in.Command...)
	s.Args = append([]string{}, in.Args...)
	s.Env = append([]corev1.EnvVar{}, in.Env...)
	s.EnvFrom = append([]corev1.EnvFromSource{}, in.EnvFrom...)
	s.VolumeMounts = append([]corev1.VolumeMount{}, in.VolumeMounts...)
	s.WorkingDir = in.WorkingDir
	s.TerminationMessagePath = in.TerminationMessagePath
	s.TerminationMessagePolicy = in.TerminationMessagePolicy
	s.ImagePullPolicy = in.ImagePullPolicy

	if in.RestartPolicy != nil {
		s.RestartPolicy = new(corev1.ContainerRestartPolicy)
		*s.RestartPolicy = *in.RestartPolicy
	}

	if in.LivenessProbe != nil {
		s.LivenessProbe = in.LivenessProbe.DeepCopy()
	}

	if in.ReadinessProbe != nil {
		s.ReadinessProbe = in.ReadinessProbe.DeepCopy()
	}

	if in.StartupProbe != nil {
		s.StartupProbe = in.StartupProbe.DeepCopy()
	}
}
