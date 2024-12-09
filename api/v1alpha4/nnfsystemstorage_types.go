/*
 * Copyright 2024 Hewlett Packard Enterprise Development LP
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

package v1alpha4

import (
	dwsv1alpha2 "github.com/DataWorkflowServices/dws/api/v1alpha2"
	"github.com/DataWorkflowServices/dws/utils/updater"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type NnfSystemStorageComputesTarget string

const (
	ComputesTargetAll     NnfSystemStorageComputesTarget = "all"
	ComputesTargetEven    NnfSystemStorageComputesTarget = "even"
	ComputesTargetOdd     NnfSystemStorageComputesTarget = "odd"
	ComputesTargetPattern NnfSystemStorageComputesTarget = "pattern"
)

// NnfSystemStorageSpec defines the desired state of NnfSystemStorage
type NnfSystemStorageSpec struct {
	// SystemConfiguration is an object reference to the SystemConfiguration resource to use. If this
	// field is empty, name: default namespace: default is used.
	SystemConfiguration corev1.ObjectReference `json:"systemConfiguration,omitempty"`

	// ExludeRabbits is a list of Rabbits to exclude from the Rabbits in the SystemConfiguration
	ExcludeRabbits []string `json:"excludeRabbits,omitempty"`

	// IncludeRabbits is a list of Rabbits to use rather than getting the list of Rabbits from the
	// SystemConfiguration
	IncludeRabbits []string `json:"includeRabbits,omitempty"`

	// ExcludeDisabledRabbits looks at the Storage resource for a Rabbit and does not use it if it's
	// marked as "disabled"
	// +kubebuilder:default:=false
	ExcludeDisabledRabbits bool `json:"excludeDisabledRabbits,omitempty"`

	// ExcludeComputes is a list of compute nodes to exclude from the the compute nodes listed in the
	// SystemConfiguration
	ExcludeComputes []string `json:"excludeComputes,omitempty"`

	// IncludeComputes is a list of computes nodes to use rather than getting the list of compute nodes
	// from the SystemConfiguration
	IncludeComputes []string `json:"includeComputes,omitempty"`

	// ComputesTarget specifies which computes to make the storage accessible to
	// +kubebuilder:validation:Enum=all;even;odd;pattern
	// +kubebuilder:default:=all
	ComputesTarget NnfSystemStorageComputesTarget `json:"computesTarget,omitempty"`

	// ComputesPattern is a list of compute node indexes (0-15) to make the storage accessible to. This
	// is only used if ComputesTarget is "pattern"
	// +kubebuilder:validation:MaxItems=16
	// +kubebuilder:validation:items:Maximum=15
	// +kubebuilder:validation:items:Minimum=0
	ComputesPattern []int `json:"computesPattern,omitempty"`

	// Capacity is the allocation size on each Rabbit
	// +kubebuilder:default:=1073741824
	Capacity int64 `json:"capacity"`

	// Type is the file system type to use for the storage allocation
	// +kubebuilder:validation:Enum=raw;xfs;gfs2
	// +kubebuilder:default:=raw
	Type string `json:"type,omitempty"`

	// Shared will create one allocation per Rabbit rather than one allocation
	// per compute node.
	// +kubebuilder:default:=true
	Shared bool `json:"shared"`

	// StorageProfile is an object reference to the storage profile to use
	StorageProfile corev1.ObjectReference `json:"storageProfile"`

	// +kubebuilder:default:=false
	IgnoreOfflineComputes bool `json:"ignoreOfflineComputes"`

	// MakeClientMounts specifies whether to make ClientMount resources or just
	// make the devices available to the client
	// +kubebuilder:default:=false
	MakeClientMounts bool `json:"makeClientMounts"`

	// ClientMountPath is an optional path for where to mount the file system on the computes
	ClientMountPath string `json:"clientMountPath,omitempty"`
}

// NnfSystemStorageStatus defines the observed state of NnfSystemStorage
type NnfSystemStorageStatus struct {
	// Ready signifies whether all work has been completed
	Ready bool `json:"ready"`

	dwsv1alpha2.ResourceError `json:",inline"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// NnfSystemStorage is the Schema for the nnfsystemstorages API
type NnfSystemStorage struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NnfSystemStorageSpec   `json:"spec,omitempty"`
	Status NnfSystemStorageStatus `json:"status,omitempty"`
}

func (a *NnfSystemStorage) GetStatus() updater.Status[*NnfSystemStorageStatus] {
	return &a.Status
}

// +kubebuilder:object:root=true
// NnfSystemStorageList contains a list of NnfSystemStorage
type NnfSystemStorageList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NnfSystemStorage `json:"items"`
}

func (n *NnfSystemStorageList) GetObjectList() []client.Object {
	objectList := []client.Object{}

	for i := range n.Items {
		objectList = append(objectList, &n.Items[i])
	}

	return objectList
}

func init() {
	SchemeBuilder.Register(&NnfSystemStorage{}, &NnfSystemStorageList{})
}
