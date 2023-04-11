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
	"strings"

	"github.com/HewlettPackard/dws/utils/updater"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// LustreFileSystemSpec defines the desired state of LustreFileSystem
type LustreFileSystemSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Name is the name of the Lustre file system.
	// +kubebuilder:validation:MaxLength:=8
	// +kubebuilder:validation:MinLength:=1
	Name string `json:"name"`

	// MgsNids is the list of comma- and colon- separated NIDs of the MGS
	// nodes to use for accessing the Lustre file system.
	MgsNids string `json:"mgsNids"`

	// MountRoot is the mount path used to access the Lustre file system from a host. Data Movement
	// directives and Container Profiles can reference this field.
	MountRoot string `json:"mountRoot"`

	// StorageClassName refers to the StorageClass to use for this file system.
	// +kubebuilder:default:="nnf-lustre-fs"
	StorageClassName string `json:"storageClassName,omitempty"`

	// Namespaces defines a map of namespaces with access to the Lustre file systems
	Namespaces map[string]LustreFileSystemNamespaceSpec `json:"namespaces,omitempty"`
}

// LustreFileSystemAccessSpec defines the desired state of Lustre File System Accesses
type LustreFileSystemNamespaceSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Modes list the persistent volume access modes for accessing the Lustre file system.
	Modes []corev1.PersistentVolumeAccessMode `json:"modes,omitempty"`
}

// LustreFileSystemStatus defines the observed status of LustreFileSystem
type LustreFileSystemStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Namespaces contains the namespaces supported for this Lustre file system and their corresponding status.
	Namespaces map[string]LustreFileSystemNamespaceStatus `json:"namespaces,omitempty"`
}

// LustreFileSystemAccessStatus defines the observe status of access to the LustreFileSystem
type LustreFileSystemNamespaceStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Modes contains the modes supported for this namespace and their corresponding access sttatus.
	Modes map[corev1.PersistentVolumeAccessMode]LustreFileSystemNamespaceAccessStatus `json:"modes,omitempty"`
}

// LustreFileSystemNamespaceAccessStatus defines the observe status of namespace access to the LustreFileSystem
type LustreFileSystemNamespaceAccessStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// State represents the current state of the namespace access
	State NamespaceAccessState `json:"state"`

	// PersistentVolumeRef holds a reference to the persistent volume, if present
	PersistentVolumeRef *corev1.LocalObjectReference `json:"persistentVolumeRef,omitempty"`

	// PersistentVolumeClaimRef holds a reference to the persistent volume claim, if present
	PersistentVolumeClaimRef *corev1.LocalObjectReference `json:"persistentVolumeClaimRef,omitempty"`
}

type NamespaceAccessState string

const (
	// NamespaceAccessPending - used to indicate the namespace access not yet ready
	NamespaceAccessPending NamespaceAccessState = "Pending"

	// NamespaceAccessReady - used to indicate the namespace access is ready
	NamespaceAccessReady NamespaceAccessState = "Ready"
)

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="FSNAME",type="string",JSONPath=".spec.name",description="Lustre file system name"
//+kubebuilder:printcolumn:name="MGSNIDS",type="string",JSONPath=".spec.mgsNids",description="List of MGS NIDs"
//+kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
//+kubebuilder:printcolumn:name="MountRoot",type="string",JSONPath=".spec.mountRoot",priority=1,description="Mount path used to mount filesystem"
//+kubebuilder:printcolumn:name="StorageClass",type="string",JSONPath=".spec.storageClassName",priority=1,description="StorageClass to use"

// LustreFileSystem is the Schema for the lustrefilesystems API
type LustreFileSystem struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LustreFileSystemSpec   `json:"spec,omitempty"`
	Status LustreFileSystemStatus `json:"status,omitempty"`
}

func (fs *LustreFileSystem) PersistentVolumeName(namespace string, mode corev1.PersistentVolumeAccessMode) string {
	return fs.Name + "-" + namespace + "-" + strings.ToLower(string(mode)) + "-pv"
}

func (fs *LustreFileSystem) PersistentVolumeClaimName(namespace string, mode corev1.PersistentVolumeAccessMode) string {
	return fs.Name + "-" + namespace + "-" + strings.ToLower(string(mode)) + "-pvc"
}

func (fs *LustreFileSystem) GetStatus() updater.Status[*LustreFileSystemStatus] {
	return &fs.Status
}

//+kubebuilder:object:root=true

// LustreFileSystemList contains a list of LustreFileSystem
type LustreFileSystemList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LustreFileSystem `json:"items"`
}

func (list *LustreFileSystemList) GetObjectList() []client.Object {
	objectList := make([]client.Object, len(list.Items))

	for i := range list.Items {
		objectList[i] = &list.Items[i]
	}

	return objectList
}

func init() {
	SchemeBuilder.Register(&LustreFileSystem{}, &LustreFileSystemList{})
}
