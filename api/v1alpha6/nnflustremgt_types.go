/*
 * Copyright 2024-2025 Hewlett Packard Enterprise Development LP
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

package v1alpha6

import (
	dwsv1alpha3 "github.com/DataWorkflowServices/dws/api/v1alpha3"
	"github.com/DataWorkflowServices/dws/utils/updater"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NnfLustreMGTSpec defines the desired state of NnfLustreMGT
type NnfLustreMGTSpec struct {
	// Addresses is the list of LNet addresses for the MGT
	Addresses []string `json:"addresses"`

	// FsNameBlackList is a list of fsnames that can't be used. This may be
	// necessary if the MGT hosts file systems external to Rabbit
	FsNameBlackList []string `json:"fsNameBlackList,omitempty"`

	// FsNameStart is the starting fsname to be used
	// +kubebuilder:validation:MaxLength:=8
	// +kubebuilder:validation:MinLength:=8
	FsNameStart string `json:"fsNameStart,omitempty"`

	// FsNameStartReference can be used to add a configmap where the starting fsname is
	// stored. If this reference is set, it takes precendence over FsNameStart. The configmap
	// will be updated with the next available fsname anytime an fsname is used.
	FsNameStartReference corev1.ObjectReference `json:"fsNameStartReference,omitempty"`

	// ClaimList is the list of currently in use fsnames
	ClaimList []corev1.ObjectReference `json:"claimList,omitempty"`
}

// NnfLustreMGTStatus defines the current state of NnfLustreMGT
type NnfLustreMGTStatus struct {
	// FsNameNext is the next available fsname that hasn't been used
	// +kubebuilder:validation:MaxLength:=8
	// +kubebuilder:validation:MinLength:=8
	FsNameNext string `json:"fsNameNext,omitempty"`

	// ClaimList is the list of currently in use fsnames
	ClaimList []NnfLustreMGTStatusClaim `json:"claimList,omitempty"`

	dwsv1alpha3.ResourceError `json:",inline"`
}

type NnfLustreMGTStatusClaim struct {
	Reference corev1.ObjectReference `json:"reference,omitempty"`
	FsName    string                 `json:"fsname,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// NnfLustreMGT is the Schema for the nnfstorageprofiles API
type NnfLustreMGT struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NnfLustreMGTSpec   `json:"spec,omitempty"`
	Status NnfLustreMGTStatus `json:"status,omitempty"`
}

func (a *NnfLustreMGT) GetStatus() updater.Status[*NnfLustreMGTStatus] {
	return &a.Status
}

//+kubebuilder:object:root=true

// NnfLustreMGTList contains a list of NnfLustreMGT
type NnfLustreMGTList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NnfLustreMGT `json:"items"`
}

func (n *NnfLustreMGTList) GetObjectList() []client.Object {
	objectList := []client.Object{}

	for i := range n.Items {
		objectList = append(objectList, &n.Items[i])
	}

	return objectList
}

func init() {
	SchemeBuilder.Register(&NnfLustreMGT{}, &NnfLustreMGTList{})
}
