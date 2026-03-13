/*
 * Copyright 2026 Hewlett Packard Enterprise Development LP
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

package v1alpha11

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NnfContainerDataVolumeCommand is the DW directive command type that created the storage.
// +kubebuilder:validation:Enum=jobdw;persistentdw
type NnfContainerDataVolumeCommand string

const (
	ContainerDataVolumeCommandJobDW        NnfContainerDataVolumeCommand = "jobdw"
	ContainerDataVolumeCommandPersistentDW NnfContainerDataVolumeCommand = "persistentdw"
)

// NnfContainerDataVolume describes an NNF storage volume to be mounted into a user container.
type NnfContainerDataVolume struct {
	// Name is the volume name defined in the container profile
	Name string `json:"name"`

	// Command is the DW directive command type that created the storage.
	Command NnfContainerDataVolumeCommand `json:"command"`

	// DirectiveIndex is the index of the #DW directive in the workflow that created this storage.
	DirectiveIndex int `json:"directiveIndex"`

	// MountPath is the path at which the storage will be mounted on the NNF node
	MountPath string `json:"mountPath"`
}

// NnfContainerDataData defines the data stored in NnfContainerData
type NnfContainerDataData struct {
	// Volumes is the list of NNF storage volumes to be mounted into the user container.
	// +optional
	Volumes []NnfContainerDataVolume `json:"volumes,omitempty"`
}

//+kubebuilder:object:root=true
// +kubebuilder:storageversion

// NnfContainerData is the Schema for the nnfcontainerdata API
type NnfContainerData struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Data NnfContainerDataData `json:"data,omitempty"`
}

//+kubebuilder:object:root=true

// NnfContainerDataList contains a list of NnfContainerData
type NnfContainerDataList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NnfContainerData `json:"items"`
}

func (n *NnfContainerDataList) GetObjectList() []client.Object {
	objectList := []client.Object{}

	for i := range n.Items {
		objectList = append(objectList, &n.Items[i])
	}

	return objectList
}

func init() {
	SchemeBuilder.Register(&NnfContainerData{}, &NnfContainerDataList{})
}
