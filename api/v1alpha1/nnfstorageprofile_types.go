/*
 * Copyright 2022 Hewlett Packard Enterprise Development LP
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

// NnfStorageProfileLustreCmdLines defines commandlines to use for mkfs, zpool, and other utilities.
type NnfStorageProfileLustreCmdLines struct {
	// ZpoolCreate specifies the zpool create commandline, minus the "zpool create".
	// This is where you may specify zpool create options, and the virtual device (vdev) such as
	// "mirror", or "draid".  See zpoolconcepts(7).
	ZpoolCreate string `json:"zpoolCreate,omitempty"`

	// Mkfs specifies the mkfs.lustre commandline, minus the "mkfs.lustre".
	// Use the --mkfsoptions argument to specify the zfs create options.  See zfsprops(7).
	// Use the --mountfsoptions argument to specify persistent mount options for the lustre targets.
	Mkfs string `json:"mkfs,omitempty"`
}

// NnfStorageProfileLustreMiscOptions defines options to use for the mount library, and other utilities.
type NnfStorageProfileLustreMiscOptions struct {
	// MountTarget specifies mount options for the mount-utils library to mount a lustre target on the Rabbit.
	// For persistent mount options for lustre targets, do not use this array; use the --mountfsoptions argument to mkfs.lustre instead.
	// Use one array element per option. Do not prepend the options with "-o".
	MountTarget []string `json:"mountTarget,omitempty"`
}

// NnfStorageProfileLustreData defines the Lustre-specific configuration
type NnfStorageProfileLustreData struct {
	// CombinedMGTMDT indicates whether the MGT and MDT should be created on the same target device
	// +kubebuilder:default:=false
	CombinedMGTMDT bool `json:"combinedMgtMdt,omitempty"`

	// ExternalMGS contains the NIDs of a pre-existing MGS that should be used
	ExternalMGS string `json:"externalMgs,omitempty"`

	// CapacityMGT specifies the size of the MGT device.
	// +kubebuilder:validation:Pattern:="^\\d+(KiB|KB|MiB|MB|GiB|GB|TiB|TB)$"
	// +kubebuilder:default:="1GiB"
	CapacityMGT string `json:"capacityMgt,omitempty"`

	// CapacityMDT specifies the size of the MDT device.  This is also
	// used for a combined MGT+MDT device.
	// +kubebuilder:validation:Pattern:="^\\d+(KiB|KB|MiB|MB|GiB|GB|TiB|TB)$"
	// +kubebuilder:default:="5GiB"
	CapacityMDT string `json:"capacityMdt,omitempty"`

	// ExclusiveMDT indicates that the MDT should not be colocated with any other target on the chosen server.
	// +kubebuilder:default:=false
	ExclusiveMDT bool `json:"exclusiveMdt,omitempty"`

	// MgtCmdLines contains commands to create an MGT target.
	MgtCmdLines NnfStorageProfileLustreCmdLines `json:"mgtCommandlines,omitempty"`

	// MdtCmdLines contains commands to create an MDT target.
	MdtCmdLines NnfStorageProfileLustreCmdLines `json:"mdtCommandlines,omitempty"`

	// MgtMdtCmdLines contains commands to create a combined MGT/MDT target.
	MgtMdtCmdLines NnfStorageProfileLustreCmdLines `json:"mgtMdtCommandlines,omitempty"`

	// OstCmdLines contains commands to create an OST target.
	OstCmdLines NnfStorageProfileLustreCmdLines `json:"ostCommandlines,omitempty"`

	// MgtOptions contains options to use for libraries used for an MGT target.
	MgtOptions NnfStorageProfileLustreMiscOptions `json:"mgtOptions,omitempty"`

	// MdtOptions contains options to use for libraries used for an MDT target.
	MdtOptions NnfStorageProfileLustreMiscOptions `json:"mdtOptions,omitempty"`

	// MgtMdtOptions contains options to use for libraries used for a combined MGT/MDT target.
	MgtMdtOptions NnfStorageProfileLustreMiscOptions `json:"mgtMdtOptions,omitempty"`

	// OstOptions contains options to use for libraries used for an OST target.
	OstOptions NnfStorageProfileLustreMiscOptions `json:"ostOptions,omitempty"`
}

// NnfStorageProfileCmdLines defines commandlines to use for mkfs, and other utilities.
type NnfStorageProfileCmdLines struct {
	// Mkfs specifies the mkfs commandline, minus the "mkfs".
	Mkfs string `json:"mkfs,omitempty"`

	// PvCreate specifies the pvcreate commandline, minus the "pvcreate".
	PvCreate string `json:"pvCreate,omitempty"`

	// VgCreate specifies the vgcreate commandline, minus the "vgcreate".
	VgCreate string `json:"vgCreate,omitempty"`

	// VgChange specifies the various vgchange commandlines, minus the "vgchange"
	VgChange NnfStorageProfileLVMVgChangeCmdLines `json:"vgChange,omitempty"`

	// VgCreate specifies the vgcreate commandline, minus the "vgremove".
	VgRemove string `json:"vgRemove,omitempty"`

	// LvCreate specifies the lvcreate commandline, minus the "lvcreate".
	LvCreate string `json:"lvCreate,omitempty"`

	// LvRemove specifies the lvcreate commandline, minus the "lvremove".
	LvRemove string `json:"lvRemove,omitempty"`
}

// NnfStorageProfileLVMVgChangeCmdLines
type NnfStorageProfileLVMVgChangeCmdLines struct {
	// The vgchange commandline for activation, minus the "vgchange" command
	Activate string `json:"activate,omitempty"`

	// The vgchange commandline for deactivation, minus the "vgchange" command
	Deactivate string `json:"deactivate,omitempty"`

	// The vgchange commandline for lockStart, minus the "vgchange" command
	LockStart string `json:"lockStart,omitempty"`
}

// NnfStorageProfileMiscOptions defines options to use for the mount library, and other utilities.
type NnfStorageProfileMiscOptions struct {
	// MountRabbit specifies mount options for mounting on the Rabbit.
	// Use one array element per option.  Do not prepend the options with "-o".
	MountRabbit []string `json:"mountRabbit,omitempty"`

	// MountCompute specifies mount options for mounting on the Compute.
	// Use one array element per option.  Do not prepend the options with "-o".
	MountCompute []string `json:"mountCompute,omitempty"`
}

// NnfStorageProfileGFS2Data defines the GFS2-specific configuration
type NnfStorageProfileGFS2Data struct {
	// CmdLines contains commands to create volumes and filesystems.
	CmdLines NnfStorageProfileCmdLines `json:"commandlines,omitempty"`

	// Options contains options for libraries.
	Options NnfStorageProfileMiscOptions `json:"options,omitempty"`
}

// NnfStorageProfileXFSData defines the XFS-specific configuration
type NnfStorageProfileXFSData struct {
	// CmdLines contains commands to create volumes and filesystems.
	CmdLines NnfStorageProfileCmdLines `json:"commandlines,omitempty"`

	// Options contains options for libraries.
	Options NnfStorageProfileMiscOptions `json:"options,omitempty"`
}

// NnfStorageProfileRawData defines the Raw-specific configuration
type NnfStorageProfileRawData struct {
	// CmdLines contains commands to create volumes and filesystems.
	CmdLines NnfStorageProfileCmdLines `json:"commandlines,omitempty"`
}

// NnfStorageProfileData defines the desired state of NnfStorageProfile
type NnfStorageProfileData struct {

	// Default is true if this instance is the default resource to use
	// +kubebuilder:default:=false
	Default bool `json:"default,omitempty"`

	// Pinned is true if this instance is an immutable copy
	// +kubebuilder:default:=false
	Pinned bool `json:"pinned,omitempty"`

	// LustreStorage defines the Lustre-specific configuration
	LustreStorage NnfStorageProfileLustreData `json:"lustreStorage"`

	// GFS2Storage defines the GFS2-specific configuration
	GFS2Storage NnfStorageProfileGFS2Data `json:"gfs2Storage"`

	// XFSStorage defines the XFS-specific configuration
	XFSStorage NnfStorageProfileXFSData `json:"xfsStorage"`

	// RawStorage defines the Raw-specific configuration
	RawStorage NnfStorageProfileRawData `json:"rawStorage"`
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

func (n *NnfStorageProfileList) GetObjectList() []client.Object {
	objectList := []client.Object{}

	for i := range n.Items {
		objectList = append(objectList, &n.Items[i])
	}

	return objectList
}

func init() {
	SchemeBuilder.Register(&NnfStorageProfile{}, &NnfStorageProfileList{})
}
