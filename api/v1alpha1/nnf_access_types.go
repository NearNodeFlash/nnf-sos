/*
 * Copyright 2021-2023 Hewlett Packard Enterprise Development LP
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
	dwsv1alpha2 "github.com/HewlettPackard/dws/api/v1alpha2"
	"github.com/HewlettPackard/dws/utils/updater"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NnfAccessSpec defines the desired state of NnfAccess
type NnfAccessSpec struct {
	// DesiredState is the desired state for the mounts on the client
	// +kubebuilder:validation:Enum=mounted;unmounted
	DesiredState string `json:"desiredState"`

	// TeardownState is the desired state of the workflow for this NNF Access resource to
	// be torn down and deleted.
	// +kubebuilder:validation:Enum:=PreRun;PostRun;Teardown
	// +kubebuilder:validation:Type:=string
	TeardownState dwsv1alpha2.WorkflowState `json:"teardownState"`

	// Target specifies which storage targets the client should mount
	// - single: Only one of the storage the client can access
	// - all: All of the storage the client can access
	// +kubebuilder:validation:Enum=single;all
	Target string `json:"target"`

	// UserID for the new mount. Currently only used for raw
	UserID uint32 `json:"userID"`

	// GroupID for the new mount. Currently only used for raw
	GroupID uint32 `json:"groupID"`

	// ClientReference is for a client resource. (DWS) Computes is the only client
	// resource type currently supported
	ClientReference corev1.ObjectReference `json:"clientReference,omitempty"`

	// MountPath for the storage target on the client
	MountPath string `json:"mountPath,omitempty"`

	// MountPathPrefix to  mount the storage target on the client when there is
	// more than one mount on a client
	MountPathPrefix string `json:"mountPathPrefix,omitempty"`

	// StorageReference is the NnfStorage reference
	StorageReference corev1.ObjectReference `json:"storageReference"`
}

// NnfAccessStatus defines the observed state of NnfAccess
type NnfAccessStatus struct {
	// State is the current state
	// +kubebuilder:validation:Enum=mounted;unmounted
	State string `json:"state"`

	// Ready signifies whether status.state has been achieved
	Ready bool `json:"ready"`

	dwsv1alpha2.ResourceError `json:",inline"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="DESIREDSTATE",type="string",JSONPath=".spec.desiredState",description="The desired state"
//+kubebuilder:printcolumn:name="STATE",type="string",JSONPath=".status.state",description="The current state"
//+kubebuilder:printcolumn:name="READY",type="boolean",JSONPath=".status.ready",description="Whether the state has been achieved"
//+kubebuilder:printcolumn:name="ERROR",type="string",JSONPath=".status.error.severity"
//+kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"

// NnfAccess is the Schema for the nnfaccesses API
type NnfAccess struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NnfAccessSpec   `json:"spec,omitempty"`
	Status NnfAccessStatus `json:"status,omitempty"`
}

func (a *NnfAccess) GetStatus() updater.Status[*NnfAccessStatus] {
	return &a.Status
}

//+kubebuilder:object:root=true

// NnfAccessList contains a list of NnfAccess
type NnfAccessList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NnfAccess `json:"items"`
}

func (n *NnfAccessList) GetObjectList() []client.Object {
	objectList := []client.Object{}

	for i := range n.Items {
		objectList = append(objectList, &n.Items[i])
	}

	return objectList
}

func init() {
	SchemeBuilder.Register(&NnfAccess{}, &NnfAccessList{})
}
