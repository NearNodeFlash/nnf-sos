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
)

const (
	// StorageTypeLabel is the label key used for tagging Storage resources
	// with a driver specific label. For example: dws.cray.hpe.com/storage=Rabbit
	StorageTypeLabel = "dws.cray.hpe.com/storage"
)

// StorageSpec defines the desired specifications of Storage resource
type StorageSpec struct {
	// State describes the desired state of the Storage resource.
	// +kubebuilder:default:=Enabled
	State ResourceState `json:"state,omitempty"`
}

// StorageDevice contains the details of the storage hardware
type StorageDevice struct {
	// Model is the manufacturer information about the device
	Model string `json:"model,omitempty"`

	// The serial number for this storage controller.
	SerialNumber string `json:"serialNumber,omitempty"`

	// The firmware version of this storage controller.
	FirmwareVersion string `json:"firmwareVersion,omitempty"`

	// Physical slot location of the storage controller.
	Slot string `json:"slot,omitempty"`

	// Capacity in bytes of the device. The full capacity may not
	// be usable depending on what the storage driver can provide.
	Capacity int64 `json:"capacity,omitempty"`

	// WearLevel in percent for SSDs. A value of 100 indicates the estimated endurance of the non-volatile memory
	// has been consumed, but may not indicate a storage failure.
	WearLevel *int64 `json:"wearLevel,omitempty"`

	// Status of the individual device
	Status ResourceStatus `json:"status,omitempty"`
}

// Node provides the status of either a compute or a server
type Node struct {
	// Name is the Kubernetes name of the node
	Name string `json:"name,omitempty"`

	// Status of the node
	Status ResourceStatus `json:"status,omitempty"`
}

// StorageAccessProtocol is the enumeration of supported protocols.
// +kubebuilder:validation:Enum:=PCIe
type StorageAccessProtocol string

const (
	PCIe StorageAccessProtocol = "PCIe"
)

// StorageAccess contains nodes and the protocol that may access the storage
type StorageAccess struct {
	// Protocol is the method that this storage can be accessed
	Protocol StorageAccessProtocol `json:"protocol,omitempty"`

	// Servers is the list of non-compute nodes that have access to the storage
	Servers []Node `json:"servers,omitempty"`

	// Computes is the list of compute nodes that have access to the storage
	Computes []Node `json:"computes,omitempty"`
}

// StorageType is the enumeration of storage types.
// +kubebuilder:validation:Enum:=NVMe
type StorageType string

const (
	NVMe StorageType = "NVMe"
)

// StorageData contains the data about the storage
type StorageStatus struct {
	// Type describes what type of storage this is
	Type StorageType `json:"type,omitempty"`

	// Devices is the list of physical devices that make up this storage
	Devices []StorageDevice `json:"devices,omitempty"`

	// Access contains the information about where the storage is accessible
	Access StorageAccess `json:"access,omitempty"`

	// Capacity is the number of bytes this storage provides. This is the
	// total accessible bytes as determined by the driver and may be different
	// than the sum of the devices' capacities.
	// +kubebuilder:default:=0
	Capacity int64 `json:"capacity"`

	// Status is the overall status of the storage
	Status ResourceStatus `json:"status,omitempty"`

	// Reboot Required is true if the node requires a reboot and false otherwise. A reboot my be
	// necessary to recover from certain hardware failures or high-availability clustering events.
	RebootRequired bool `json:"rebootRequired,omitempty"`

	// Message provides additional details on the current status of the resource
	Message string `json:"message,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="State",type="string",JSONPath=".spec.state",description="State of the storage resource"
//+kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.status",description="Status of the storage resource"
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// Storage is the Schema for the storages API
type Storage struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   StorageSpec   `json:"spec"`
	Status StorageStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// StorageList contains a list of Storage
type StorageList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Storage `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Storage{}, &StorageList{})
}
