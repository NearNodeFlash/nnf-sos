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
	"github.hpe.com/hpe/hpc-dpm-dws-operator/utils/dwdparse"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

//+kubebuilder:object:root=true

// DWDirectiveRule is the Schema for the DWDirective API
type DWDirectiveRule struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec []dwdparse.DWDirectiveRuleSpec `json:"spec,omitempty"`
}

//+kubebuilder:object:root=true

// DWDirectiveRuleList contains a list of DWDirective
type DWDirectiveRuleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DWDirectiveRule `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DWDirectiveRule{}, &DWDirectiveRuleList{})
}
