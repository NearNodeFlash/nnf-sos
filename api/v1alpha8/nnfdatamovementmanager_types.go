/*
 * Copyright 2022-2025 Hewlett Packard Enterprise Development LP
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
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/DataWorkflowServices/dws/utils/updater"
)

const (
	DataMovementWorkerLabel = "dm.cray.hpe.com/worker"

	// The name of the expected Data Movement manager. This is to ensure Data Movement is ready in
	// the DataIn/DataOut stages before attempting data movement operations.
	DataMovementManagerName = "nnf-dm-manager-controller-manager"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// NnfDataMovementManagerSpec defines the desired state of NnfDataMovementManager
type NnfDataMovementManagerSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Selector defines the pod selector used in scheduling the worker nodes. This value is duplicated
	// to the template.spec.metadata.labels to satisfy the requirements of the worker's Daemon Set.
	Selector metav1.LabelSelector `json:"selector"`

	// Spec defines the slim PodSpec that is used for the basis of the worker Daemon Set that
	// manages the per node data movement operations.
	PodSpec NnfPodSpec `json:"podSpec"`

	// UpdateStrategy defines the UpdateStrategy that is used for the basis of the worker Daemon Set
	// that manages the per node data movement operations.
	UpdateStrategy appsv1.DaemonSetUpdateStrategy `json:"updateStrategy"`

	// Host Path defines the directory location of shared mounts on an individual worker node.
	HostPath string `json:"hostPath"`

	// Mount Path defines the location within the container at which the Host Path volume should be mounted.
	MountPath string `json:"mountPath"`
}

// NnfDataMovementManagerStatus defines the observed state of NnfDataMovementManager
type NnfDataMovementManagerStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Ready indicates that the Data Movement Manager has achieved the desired readiness state
	// and all managed resources are initialized.
	// +kubebuilder:default:=false
	Ready bool `json:"ready"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
// +kubebuilder:storageversion
//+kubebuilder:printcolumn:name="READY",type="boolean",JSONPath=".status.ready",description="True if manager readied all resoures"
//+kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"

// NnfDataMovementManager is the Schema for the nnfdatamovementmanagers API
type NnfDataMovementManager struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NnfDataMovementManagerSpec   `json:"spec,omitempty"`
	Status NnfDataMovementManagerStatus `json:"status,omitempty"`
}

func (m *NnfDataMovementManager) GetStatus() updater.Status[*NnfDataMovementManagerStatus] {
	return &m.Status
}

//+kubebuilder:object:root=true

// NnfDataMovementManagerList contains a list of NnfDataMovementManager
type NnfDataMovementManagerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NnfDataMovementManager `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NnfDataMovementManager{}, &NnfDataMovementManagerList{})
}
