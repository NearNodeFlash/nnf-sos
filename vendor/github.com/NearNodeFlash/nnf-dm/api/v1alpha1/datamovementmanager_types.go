/*
 * Copyright 2022 Hewlett Packard Enterprise Development LP
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
	DataMovementNamespace = "nnf-dm-system"

	DataMovementWorkerLabel = "dm.cray.hpe.com/worker"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// DataMovementManagerSpec defines the desired state of DataMovementManager
type DataMovementManagerSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Selector defines the pod selector used in scheduling the worker nodes. This value is duplicated
	// to the template.spec.metadata.labels to satisfy the requirements of the worker's Daemon Set.
	Selector metav1.LabelSelector `json:"selector"`

	// Template defines the pod template that is used for the basis of the worker Daemon Set that
	// manages the per node data movement operations.
	Template corev1.PodTemplateSpec `json:"template"`

	// Host Path defines the directory location of shared mounts on an individual worker node.
	HostPath string `json:"hostPath"`

	// Mount Path defines the location within the container at which the Host Path volume should be mounted.
	MountPath string `json:"mountPath"`
}

// DataMovementManagerStatus defines the observed state of DataMovementManager
type DataMovementManagerStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Ready indiciates the Data Movement Manager has achieved the desired readiness state
	// and all managed resources are initialized.
	Ready bool `json:"ready,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="READY",type="boolean",JSONPath=".status.ready",description="True if manager readied all resoures"
//+kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"

// DataMovementManager is the Schema for the datamovementmanagers API
type DataMovementManager struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DataMovementManagerSpec   `json:"spec,omitempty"`
	Status DataMovementManagerStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// DataMovementManagerList contains a list of DataMovementManager
type DataMovementManagerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DataMovementManager `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DataMovementManager{}, &DataMovementManagerList{})
}
