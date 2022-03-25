/*
Copyright 2022 Hewlett Packard Enterprise Development LP
*/

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ClientMountLustre defines the lustre device information for mounting
type ClientMountDeviceLustre struct {
	// Lustre fsname
	FileSystemName string `json:"fileSystemName"`

	// List of mgsAddresses of the form [address]@[lnet]
	// +kubebuilder:validation:MinItems=1
	MgsAddresses []string `json:"mgsAddresses"`
}

// ClientMountNVMeDesc uniquely describes an NVMe namespace
type ClientMountNVMeDesc struct {
	// Serial number of the base NVMe device
	DeviceSerial string `json:"deviceSerial"`

	// Id of the Namespace on the NVMe device (e.g., "2")
	NamespaceID string `json:"namespaceID"`

	// Globally unique namespace ID
	NamespaceGUID string `json:"namespaceGUID"`
}

type ClientMountLVMDeviceType string

const (
	ClientMountLVMDeviceTypeNVMe ClientMountLVMDeviceType = "nvme"
)

// ClientMountDeviceLVM defines an LVM device by the VG/LV pair and optionally
// the drives that are the PVs.
type ClientMountDeviceLVM struct {
	// Type of underlying block deices used for the PVs
	// +kubebuilder:validation:Enum=nvme
	DeviceType ClientMountLVMDeviceType `json:"deviceType"`

	// List of NVMe namespaces that are used by the VG
	NVMeInfo []ClientMountNVMeDesc `json:"nvmeInfo,omitempty"`

	// LVM volume group name
	VolumeGroup string `json:"volumeGroup,omitempty"`

	// LVM logical volume name
	LogicalVolume string `json:"logicalVolume,omitempty"`
}

// ClientMountDeviceReference is an reference to a different Kubernetes object
// where device information can be found
type ClientMountDeviceReference struct {
	// Object reference for the device information
	ObjectReference corev1.ObjectReference `json:"objectReference"`

	// Optional private data for the driver
	Data int `json:"data,omitempty"`
}

type ClientMountDeviceType string

const (
	// ClientMountDeviceTypeLustre is used to define the device as a Lustre file system
	ClientMountDeviceTypeLustre ClientMountDeviceType = "lustre"

	// ClientMountDeviceTypeLVM is used to define the device as a LVM logical volume
	ClientMountDeviceTypeLVM ClientMountDeviceType = "lvm"

	// ClientMountDeviceTypeReference is used when the device information is described in
	// a separate Kubernetes resource. The clientmountd (or another controller doing the mounts)
	// must know how to interpret the resource to extract the device information.
	ClientMountDeviceTypeReference ClientMountDeviceType = "reference"
)

// ClientMountDevice defines the device to mount
type ClientMountDevice struct {
	// +kubebuilder:validation:Enum=lustre;lvm;reference
	Type ClientMountDeviceType `json:"type"`

	// Lustre specific device information
	Lustre *ClientMountDeviceLustre `json:"lustre,omitempty"`

	// LVM logical volume specific device information
	LVM *ClientMountDeviceLVM `json:"lvm,omitempty"`

	DeviceReference *ClientMountDeviceReference `json:"deviceReference,omitempty"`
}

// ClientMountInfo defines a single mount
type ClientMountInfo struct {
	// Client path for mount target
	MountPath string `json:"mountPath"`

	// Description of the device to mount
	Device ClientMountDevice `json:"device"`

	// mount type
	// +kubebuilder:validation:Enum=lustre;xfs;gfs2;bind
	Type string `json:"type"`

	// Compute is the name of the compute node which shares this mount if present. Empty if not shared.
	Compute string `json:"compute,omitempty"`
}

type ClientMountState string

const (
	ClientMountStateMounted   ClientMountState = "mounted"
	ClientMountStateUnmounted ClientMountState = "unmounted"
)

// ClientMountSpec defines the desired state of ClientMount
type ClientMountSpec struct {
	// Name of the client node that is targeted by this mount
	Node string `json:"node"`

	// Desired state of the mount point
	// +kubebuilder:validation:Enum=mounted;unmounted
	DesiredState ClientMountState `json:"desiredState"`

	// List of mounts to create on this client
	// +kubebuilder:validation:MinItems=1
	Mounts []ClientMountInfo `json:"mounts"`
}

// ClientMountInfoStatus is the status for a single mount point
type ClientMountInfoStatus struct {
	// Current state
	// +kubebuilder:validation:Enum=mounted;unmounted
	State ClientMountState `json:"state"`

	// Ready indicates whether status.state has been achieved
	Ready bool `json:"ready"`

	// Error message of first error
	Message string `json:"message,omitempty"`
}

// ClientMountStatus defines the observed state of ClientMount
type ClientMountStatus struct {
	// List of mount statuses
	Mounts []ClientMountInfoStatus `json:"mounts"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ClientMount is the Schema for the clientmounts API
type ClientMount struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClientMountSpec   `json:"spec,omitempty"`
	Status ClientMountStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ClientMountList contains a list of ClientMount
type ClientMountList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClientMount `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ClientMount{}, &ClientMountList{})
}
