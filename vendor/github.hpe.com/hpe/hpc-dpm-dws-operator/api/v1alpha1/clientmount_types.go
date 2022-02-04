/*
Copyright 2022 Hewlett Packard Enterprise Development LP
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ClientMountLustre defines the lustre device information for mounting
type ClientMountLustre struct {
	// Lustre fsname
	FileSystemName string `json:"fileSystemName"`

	// List of mgsAddresses of the form [address]@[lnet]
	// +kubebuilder:validation:MinItems=1
	MgsAddresses []string `json:"mgsAddresses"`
}

// ClientMountDevice defines the device to mount
type ClientMountDevice struct {
	// +kubebuilder:validation:Enum=lustre;nvme
	Type string `json:"type"`

	// NVMe specific device information
	NvmeNamespaceIds []string `json:"nvmeNamespaceIds,omitempty"`

	// Lustre specific device information
	Lustre ClientMountLustre `json:"lustre,omitempty"`
}

// ClientMountInfo defines a single mount
type ClientMountInfo struct {
	// Client path for mount target
	MountPath string `json:"mountPath"`

	// Description of the device to mount
	Device ClientMountDevice `json:"device"`

	// mount type
	// +kubebuilder:validation:Enum=lustre;xfs;gfs2
	Type string `json:"type"`
}

// ClientMountSpec defines the desired state of ClientMount
type ClientMountSpec struct {
	// Name of the client node that is targeted by this mount
	Node string `json:"node"`

	// Desired state of the mount point
	// +kubebuilder:validation:Enum=mounted;unmounted
	DesiredState string `json:"desiredState"`

	// List of mounts to create on this client
	// +kubebuilder:validation:MinItems=1
	Mounts []ClientMountInfo `json:"mounts"`
}

// ClientMountInfoStatus is the status for a single mount point
type ClientMountInfoStatus struct {
	// Current state
	// +kubebuilder:validation:Enum=mounted;unmounted
	State string `json:"state"`

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
