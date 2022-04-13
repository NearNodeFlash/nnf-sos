/*
Copyright 2022 Hewlett Packard Enterprise Development LP
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NnfStorageProfileLustreData defines the Lustre-specific configuration
type NnfStorageProfileLustreData struct {
	// CombinedMGTMDT indicates whether the MGT and MDT should be created on the same target device
	// +kubebuilder:default:=false
	CombinedMGTMDT bool `json:"combinedMgtMdt,omitempty"`

	// ExternalMGS contains the NIDs of a pre-existing MGS that should be used
	ExternalMGS []string `json:"externalMgs,omitempty"`
}

// NnfStorageProfileGFS2Data defines the GFS2-specific configuration
type NnfStorageProfileGFS2Data struct {
	// Placeholder
	// +kubebuilder:default:=false
	Placeholder bool `json:"placeholder,omitempty"`
}

// NnfStorageProfileXFSData defines the XFS-specific configuration
type NnfStorageProfileXFSData struct {
	// Placeholder
	// +kubebuilder:default:=false
	Placeholder bool `json:"placeholder,omitempty"`
}

// NnfStorageProfileRawData defines the Raw-specific configuration
type NnfStorageProfileRawData struct {
	// Placeholder
	// +kubebuilder:default:=false
	Placeholder bool `json:"placeholder,omitempty"`
}

// NnfStorageProfileData defines the desired state of NnfStorageProfile
type NnfStorageProfileData struct {

	// Default is true if this instance is the default resource to use
	// +kubebuilder:default:=false
	Default bool `json:"default,omitempty"`

	// LustreStorage defines the Lustre-specific configuration
	LustreStorage *NnfStorageProfileLustreData `json:"lustreStorage,omitempty"`

	// GFS2Storage defines the GFS2-specific configuration
	GFS2Storage *NnfStorageProfileGFS2Data `json:"gfs2Storage,omitempty"`

	// XFSStorage defines the XFS-specific configuration
	XFSStorage *NnfStorageProfileXFSData `json:"xfsStorage,omitempty"`

	// RawStorage defines the Raw-specific configuration
	RawStorage *NnfStorageProfileRawData `json:"rawStorage,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:printcolumn:name="DEFAULT",type="boolean",JSONPath=".data.default",description="True if this is the default instance"
//+kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"

// NnfStorageProfile is the Schema for the nnfstorageprofiles API
type NnfStorageProfile struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Data NnfStorageProfileData `json:"data,omitempty"`
}

//+kubebuilder:object:root=true

// NnfStorageProfileList contains a list of NnfStorageProfile
type NnfStorageProfileList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NnfStorageProfile `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NnfStorageProfile{}, &NnfStorageProfileList{})
}
