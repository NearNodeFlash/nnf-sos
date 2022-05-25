/*
 * Copyright 2021, 2022 Hewlett Packard Enterprise Development LP
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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ComputesData defines the compute nodes that are assigned to the workflow
type ComputesData struct {
	// Name is the identifer name for the compute node
	Name string `json:"name"`
}

//+kubebuilder:object:root=true

// Computes is the Schema for the computes API
type Computes struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Data []ComputesData `json:"data,omitempty"`
}

//+kubebuilder:object:root=true

// ComputesList contains a list of Computes
type ComputesList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Computes `json:"items"`
}

func (c *ComputesList) GetObjectList() []client.Object {
	objectList := []client.Object{}

	for i := range c.Items {
		objectList = append(objectList, &c.Items[i])
	}

	return objectList
}

func init() {
	SchemeBuilder.Register(&Computes{}, &ComputesList{})
}
