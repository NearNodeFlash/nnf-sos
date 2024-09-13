/*
 * Copyright 2023-2024 Hewlett Packard Enterprise Development LP
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

package v1alpha2

import (
	"github.com/DataWorkflowServices/dws/utils/updater"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// NnfPortManagerAllocationSpec defines the desired state for a single port allocation
type NnfPortManagerAllocationSpec struct {
	// Requester is an object reference to the requester of a ports.
	Requester corev1.ObjectReference `json:"requester"`

	// Count is the number of desired ports the requester needs. The port manager
	// will attempt to allocate this many ports.
	// +kubebuilder:default:=1
	Count int `json:"count"`
}

// NnfPortManagerSpec defines the desired state of NnfPortManager
type NnfPortManagerSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// SystemConfiguration is an object reference to the system configuration. The
	// Port Manager will use the available ports defined in the system configuration.
	SystemConfiguration corev1.ObjectReference `json:"systemConfiguration"`

	// Allocations is a list of allocation requests that the Port Manager will attempt
	// to satisfy. To request port resources from the port manager, clients should add
	// an entry to the allocations. Entries must be unique. The port manager controller
	// will attempt to allocate port resources for each allocation specification in the
	// list. To remove an allocation and free up port resources, remove the allocation
	// from the list.
	Allocations []NnfPortManagerAllocationSpec `json:"allocations"`
}

// AllocationStatus is the current status of a port requestor. A port that is in use by the respective owner
// will have a status of "InUse". A port that is freed by the owner but not yet reclaimed by the port manager
// will have a status of "Free". Any other status value indicates a failure of the port allocation.
// +kubebuilder:validation:Enum:=InUse;Free;Cooldown;InvalidConfiguration;InsufficientResources
type NnfPortManagerAllocationStatusStatus string

const (
	NnfPortManagerAllocationStatusInUse                 NnfPortManagerAllocationStatusStatus = "InUse"
	NnfPortManagerAllocationStatusFree                  NnfPortManagerAllocationStatusStatus = "Free"
	NnfPortManagerAllocationStatusCooldown              NnfPortManagerAllocationStatusStatus = "Cooldown"
	NnfPortManagerAllocationStatusInvalidConfiguration  NnfPortManagerAllocationStatusStatus = "InvalidConfiguration"
	NnfPortManagerAllocationStatusInsufficientResources NnfPortManagerAllocationStatusStatus = "InsufficientResources"
	// NOTE: You must ensure any new value is added to the above kubebuilder validation enum
)

// NnfPortManagerAllocationStatus defines the allocation status of a port for a given requester.
type NnfPortManagerAllocationStatus struct {
	// Requester is an object reference to the requester of the port resource, if one exists, or
	// empty otherwise.
	Requester *corev1.ObjectReference `json:"requester,omitempty"`

	// Ports is list of ports allocated to the owning resource.
	Ports []uint16 `json:"ports,omitempty"`

	// Status is the ownership status of the port.
	Status NnfPortManagerAllocationStatusStatus `json:"status"`

	// TimeUnallocated is when the port was unallocated. This is to ensure the proper cooldown
	// duration.
	TimeUnallocated *metav1.Time `json:"timeUnallocated,omitempty"`
}

// PortManagerStatus is the current status of the port manager.
// +kubebuilder:validation:Enum:=Ready;SystemConfigurationNotFound
type NnfPortManagerStatusStatus string

const (
	NnfPortManagerStatusReady                       NnfPortManagerStatusStatus = "Ready"
	NnfPortManagerStatusSystemConfigurationNotFound NnfPortManagerStatusStatus = "SystemConfigurationNotFound"
	// NOTE: You must ensure any new value is added in the above kubebuilder validation enum
)

// NnfPortManagerStatus defines the observed state of NnfPortManager
type NnfPortManagerStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Allocations is a list of port allocation status'.
	Allocations []NnfPortManagerAllocationStatus `json:"allocations,omitempty"`

	// Status is the current status of the port manager.
	Status NnfPortManagerStatusStatus `json:"status"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
// +kubebuilder:storageversion

// NnfPortManager is the Schema for the nnfportmanagers API
type NnfPortManager struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NnfPortManagerSpec   `json:"spec,omitempty"`
	Status NnfPortManagerStatus `json:"status,omitempty"`
}

func (mgr *NnfPortManager) GetStatus() updater.Status[*NnfPortManagerStatus] {
	return &mgr.Status
}

//+kubebuilder:object:root=true

// NnfPortManagerList contains a list of NnfPortManager
type NnfPortManagerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NnfPortManager `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NnfPortManager{}, &NnfPortManagerList{})
}
