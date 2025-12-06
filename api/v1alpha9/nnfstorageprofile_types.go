/*
 * Copyright 2022-2025 Hewlett Packard Enterprise Development LP
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

package v1alpha9

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type NnfStorageProfileLustreClientCmdLines struct {

	// MountRabbit specifies mount options for making the Lustre client mount on the Rabbit.
	MountRabbit string `json:"mountRabbit,omitempty"`

	// RabbitPostSetup specifies a list of commands to run on the Rabbit with a Lustre client mounted after the Lustre file system
	// has been set up. This includes mounting the Lustre filesystem beforehand and unmounting it afterward.
	RabbitPostSetup []string `json:"rabbitPostSetup,omitempty"`

	// RabbitPreTeardown specifies a list of commands to run on the Rabbit with a Lustre client mounted before the
	// Lustre file system is torn down. This includes mounting the Lustre filesystem beforehand and
	// unmounting it afterward.
	RabbitPreTeardown []string `json:"rabbitPreTeardown,omitempty"`

	// MountCompute specifies mount options for making the Lustre client mount on the Compute.
	MountCompute string `json:"mountCompute,omitempty"`

	// RabbitPreMount specifies a list of commands to run on the Rabbit before a Lustre client is mounted
	RabbitPreMount []string `json:"rabbitPreMount,omitempty"`

	// RabbitPostMount specifies a list of commands to run on the Rabbit after a Lustre client is mounted
	RabbitPostMount []string `json:"rabbitPostMount,omitempty"`

	// RabbitPreUnmount specifies a list of commands to run on the Rabbit before a Lustre client is unmounted
	RabbitPreUnmount []string `json:"rabbitPreUnmount,omitempty"`

	// RabbitPostUnmount specifies a list of commands to run on the Rabbit after a Lustre client is unmounted
	RabbitPostUnmount []string `json:"rabbitPostUnmount,omitempty"`

	// ComputePreMount specifies a list of commands to run on the Compute node before a Lustre client is mounted
	ComputePreMount []string `json:"computePreMount,omitempty"`

	// ComputePostMount specifies a list of commands to run on the Compute node after a Lustre client is mounted
	ComputePostMount []string `json:"computePostMount,omitempty"`

	// ComputePreUnmount specifies a list of commands to run on the Compute node before a Lustre client is unmounted
	ComputePreUnmount []string `json:"computePreUnmount,omitempty"`

	// ComputePostUnmount specifies a list of commands to run on the Compute node after a Lustre client is unmounted
	ComputePostUnmount []string `json:"computePostUnmount,omitempty"`
}

// NnfStorageProfileLustreCmdLines defines commandlines to use for mkfs, zpool, and other utilities
// for Lustre allocations.
type NnfStorageProfileLustreCmdLines struct {
	// ZpoolCreate specifies the zpool create commandline, minus the "zpool create".
	// This is where you may specify zpool create options, and the virtual device (vdev) such as
	// "mirror", or "draid".  See zpoolconcepts(7).
	ZpoolCreate string `json:"zpoolCreate,omitempty"`

	ZpoolReplace string `json:"zpoolReplace,omitempty"`

	// Mkfs specifies the mkfs.lustre commandline, minus the "mkfs.lustre".
	// Use the --mkfsoptions argument to specify the zfs create options.  See zfsprops(7).
	// Use the --mountfsoptions argument to specify persistent mount options for the lustre targets.
	Mkfs string `json:"mkfs,omitempty"`

	// MountTarget specifies the mount command line for the lustre target.
	// For persistent mount options for lustre targets, do not use this array; use the --mountfsoptions
	// argument to mkfs.lustre instead.
	MountTarget string `json:"mountTarget,omitempty"`

	// PostActivate specifies a list of commands to run on the Rabbit after the
	// Lustre target has been activated
	PostActivate []string `json:"postActivate,omitempty"`

	// PreDeactivate specifies a list of commands to run on the Rabbit before the
	// Lustre target is deactivated
	PreDeactivate []string `json:"preDeactivate,omitempty"`
}

// NnfStorageProfileLustreMiscOptions defines options to use for the mount library, and other utilities.
type NnfStorageProfileLustreMiscOptions struct {
	// ColocateComputes indicates that the Lustre target should be placed on a Rabbit node that has a physical connection
	// to the compute nodes in a workflow
	// +kubebuilder:default:=false
	ColocateComputes bool `json:"colocateComputes"`

	// Count specifies how many Lustre targets to create
	// +kubebuilder:validation:Minimum:=1
	Count int `json:"count,omitempty"`

	// Scale provides a unitless value to determine how many Lustre targets to create
	// +kubebuilder:validation:Minimum:=1
	// +kubebuilder:validation:Maximum:=10
	Scale int `json:"scale,omitempty"`

	// Storagelabels defines a list of labels that are added to the DirectiveBreakdown
	// labels constraint. This restricts allocations to Storage resources with these labels
	StorageLabels []string `json:"storageLabels,omitempty"`
}

// NnfStorageProfileLustreData defines the Lustre-specific configuration
type NnfStorageProfileLustreData struct {
	// CombinedMGTMDT indicates whether the MGT and MDT should be created on the same target device
	// +kubebuilder:default:=false
	CombinedMGTMDT bool `json:"combinedMgtMdt,omitempty"`

	// ExternalMGS specifies the use of an existing MGS rather than creating one. This can
	// be either the NID(s) of a pre-existing MGS that should be used, or it can be an NNF Persistent
	// Instance that was created with the "StandaloneMGTPoolName" option. In the latter case, the format
	// is "pool:poolName" where "poolName" is the argument from "StandaloneMGTPoolName". A single MGS will
	// be picked from the pool.
	ExternalMGS string `json:"externalMgs,omitempty"`

	// CapacityMGT specifies the size of the MGT device.
	// +kubebuilder:validation:Pattern:="^\\d+(KiB|KB|MiB|MB|GiB|GB|TiB|TB)$"
	// +kubebuilder:default:="5GiB"
	CapacityMGT string `json:"capacityMgt,omitempty"`

	// CapacityMDT specifies the size of the MDT device.  This is also
	// used for a combined MGT+MDT device.
	// +kubebuilder:validation:Pattern:="^\\d+(KiB|KB|MiB|MB|GiB|GB|TiB|TB)$"
	// +kubebuilder:default:="5GiB"
	CapacityMDT string `json:"capacityMdt,omitempty"`

	// ExclusiveMDT indicates that the MDT should not be colocated with any other target on the chosen server.
	// +kubebuilder:default:=false
	ExclusiveMDT bool `json:"exclusiveMdt,omitempty"`

	// CapacityScalingFactor is a scaling factor for the OST capacity requested in the DirectiveBreakdown
	// +kubebuilder:default:="1.0"
	CapacityScalingFactor string `json:"capacityScalingFactor,omitempty"`

	// StandaloneMGTPoolName creates a Lustre MGT without a MDT or OST. This option can only be used when creating
	// a persistent Lustre instance. The MGS is placed into a named pool that can be used by the "ExternalMGS" option.
	// Multiple pools can be created.
	StandaloneMGTPoolName string `json:"standaloneMgtPoolName,omitempty"`

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

	// ClientCmdLines contains commands to mount a Lustre client
	ClientCmdLines NnfStorageProfileLustreClientCmdLines `json:"clientCommandLines,omitempty"`

	// PreMountMGTCmds contains commands to be run on the MGT after all Lustre targets are created and activated
	PreMountMGTCmds []string `json:"preMountMGTCommands,omitempty"`
}

type NnfStorageProfileRabbitBlockDeviceCommands struct {
	// PvCreate specifies the pvcreate commandline, minus the "pvcreate".
	PvCreate string `json:"pvCreate,omitempty"`

	// PvRemove specifies the pvremove commandline, minus the "pvremove".
	PvRemove string `json:"pvRemove,omitempty"`

	// VgCreate specifies the vgcreate commandline, minus the "vgcreate".
	VgCreate string `json:"vgCreate,omitempty"`

	// VgChange specifies the various vgchange commandlines, minus the "vgchange"
	VgChange NnfStorageProfileLVMVgChangeCmdLines `json:"vgChange,omitempty"`

	// LVMRebuild specifies command lines needed for rebuilding LVM RAID devices
	LVMRebuild NnfStorageProfileLVMRebuildCmdLines `json:"lvmRebuild,omitempty"`

	// VgCreate specifies the vgcreate commandline, minus the "vgremove".
	VgRemove string `json:"vgRemove,omitempty"`

	// LvCreate specifies the lvcreate commandline, minus the "lvcreate".
	LvCreate string `json:"lvCreate,omitempty"`

	// LvChange specifies the various lvchange commandlines, minus the "lvchange"
	LvChange NnfStorageProfileLVMLvChangeCmdLines `json:"lvChange,omitempty"`

	// LvRemove specifies the lvcreate commandline, minus the "lvremove".
	LvRemove string `json:"lvRemove,omitempty"`

	// UserCommands allows admins to specify commands that can be run at various
	// points the Rabbit workflow
	UserCommands NnfStorageProfileBlockDeviceUserCommands `json:"userCommands,omitempty"`
}

type NnfStorageProfileComputeBlockDeviceCommands struct {
	// VgChange specifies the various vgchange commandlines, minus the "vgchange"
	VgChange NnfStorageProfileLVMVgChangeCmdLines `json:"vgChange,omitempty"`

	// LvChange specifies the various lvchange commandlines, minus the "lvchange"
	LvChange NnfStorageProfileLVMLvChangeCmdLines `json:"lvChange,omitempty"`

	// UserCommands allows admins to specify commands that can be run at various
	// points the Rabbit workflow
	UserCommands NnfStorageProfileBlockDeviceUserCommands `json:"userCommands,omitempty"`
}

type NnfStorageProfileBlockDevice struct {
	// SharedVg specifies that allocations from a workflow on the same Rabbit should share an
	// LVM VolumeGroup
	// +kubebuilder:default:=false
	SharedVg bool `json:"sharedVg,omitempty"`

	// RabbitCommands specifies the commands that are run on the Rabbit nodes
	RabbitCommands NnfStorageProfileRabbitBlockDeviceCommands `json:"rabbitCommands,omitempty"`

	// ComputeCommands specifies the commands that are run on the Compute nodes
	ComputeCommands NnfStorageProfileComputeBlockDeviceCommands `json:"computeCommands,omitempty"`
}

type NnfStorageProfileBlockDeviceUserCommands struct {
	// PreActivate specifies a list of commands to run before the block device is activated
	PreActivate []string `json:"preActivate,omitempty"`

	// PostActivate specifies a list of commands to run after the block device is activated
	PostActivate []string `json:"postActivate,omitempty"`

	// PreDeactivate specifies a list of commands to run before the block device is deactivated
	PreDeactivate []string `json:"preDeactivate,omitempty"`

	// PostDeactivate specifies a list of commands to run after the block device is deactivated
	PostDeactivate []string `json:"postDeactivate,omitempty"`
}

type NnfStorageProfileRabbitFileSystemCommands struct {
	// Mkfs specifies the mkfs commandline, minus the "mkfs".
	Mkfs string `json:"mkfs,omitempty"`

	// Mount specifies mount options for mounting on the Rabbit.
	Mount string `json:"mount,omitempty"`

	// UserCommands specifies commands that can be run on the Rabbit when the file system is mounted/unmounted
	// as a client during DataIn, PreRun, PostRun, and DataOut
	UserCommands NnfStorageProfileFileSystemUserCommands `json:"userCommands,omitempty"`
}

type NnfStorageProfileComputeFileSystemCommands struct {
	// Mount specifies mount options for mounting on the Rabbit.
	Mount string `json:"mount,omitempty"`

	// UserCommands specifies commands that can be run on the Compute node when the file system is
	// mounted/unmounted in PreRun and PostRun
	UserCommands NnfStorageProfileFileSystemUserCommands `json:"userCommands,omitempty"`
}

type NnfStorageProfileFileSystem struct {
	// RabbitCommands specifies the commands that are run on the Rabbit nodes
	RabbitCommands NnfStorageProfileRabbitFileSystemCommands `json:"rabbitCommands,omitempty"`

	// ComputeCommands specifies the commands that are run on the Compute nodes
	ComputeCommands NnfStorageProfileComputeFileSystemCommands `json:"computeCommands,omitempty"`
}

type NnfStorageProfileFileSystemUserCommands struct {
	// PreMount specifies a list of commands to run before the file system is mounted
	PreMount []string `json:"preMount,omitempty"`

	// PostMount specifies a list of commands to run after the file system is mounted
	PostMount []string `json:"postMount,omitempty"`

	// PreUnmount specifies a list of commands to run before the file system is unmounted
	PreUnmount []string `json:"preUnmount,omitempty"`

	// PostUnmount specifies a list of commands to run after the file system is unmounted
	PostUnmount []string `json:"postUnmount,omitempty"`
}

// NnfStorageProfileLVMVgChangeCmdLines
type NnfStorageProfileLVMRebuildCmdLines struct {
	// VgExtend specifies the command for adding PVs to a VG
	VgExtend string `json:"vgExtend,omitempty"`

	// VgReduce specifies the command for removing PVs from a VG
	VgReduce string `json:"vgReduce,omitempty"`

	// LvRepair specifies the command for repairing a LV after a drive failure
	LvRepair string `json:"lvRepair,omitempty"`
}

// NnfStorageProfileLVMVgChangeCmdLines
type NnfStorageProfileLVMVgChangeCmdLines struct {
	// The vgchange commandline for lockStart, minus the "vgchange" command
	LockStart string `json:"lockStart,omitempty"`

	// The vgchange commandline for lockStop, minus the "vgchange" command
	LockStop string `json:"lockStop,omitempty"`
}

// NnfStorageProfileLVMVgChangeCmdLines
type NnfStorageProfileLVMLvChangeCmdLines struct {
	// The lvchange commandline for activate, minus the "lvchange" command
	Activate string `json:"activate,omitempty"`

	// The lvchange commandline for deactivate, minus the "lvchange" command
	Deactivate string `json:"deactivate,omitempty"`
}

type NnfStorageProfileUserCommands struct {
	// PostActivate specifies a list of commands to run after the file system is activated
	PostActivate []string `json:"postActivate,omitempty"`

	// PostActivate specifies a list of commands to run before the file system is deactivated
	PreDeactivate []string `json:"preDeactivate,omitempty"`

	// PostSetup specifies a list of commands to run after the file system is setup. The file system
	// is mounted on the Rabbit when these commands are run
	PostSetup []string `json:"postSetup,omitempty"`

	// PreTeardown specifies a list of commands to run before the file system is destroyed. The file system
	// is mounted on the Rabbit when these commands are run
	PreTeardown []string `json:"preTeardown,omitempty"`
}

// NnfStorageProfileSharedData defines the shared parameters for simple file systems (xfs, gfs2, and raw)
type NnfStorageProfileSharedData struct {
	// FileSystemCommands specifies commands that are run when configuring the file system
	FileSystemCommands NnfStorageProfileFileSystem `json:"fileSystemCommands,omitempty"`

	// BlockDeviceCommands specifies commands that are run when configuring the block device
	BlockDeviceCommands NnfStorageProfileBlockDevice `json:"blockDeviceCommands,omitempty"`

	// UserCommands specifies commands that can be run on the Rabbit during allocation setup
	// and teardown
	UserCommands NnfStorageProfileUserCommands `json:"userCommands,omitempty"`

	// SharedAllocation is used to share an allocation across multiple clients
	SharedAllocation bool `json:"sharedAllocation,omitempty"`

	// Storagelabels defines a list of labels that are added to the DirectiveBreakdown
	// labels constraint. This restricts allocations to Storage resources with these labels
	StorageLabels []string `json:"storageLabels,omitempty"`

	// CapacityScalingFactor is a scaling factor for the capacity requested in the DirectiveBreakdown
	// +kubebuilder:default:="1.0"
	CapacityScalingFactor string `json:"capacityScalingFactor,omitempty"`

	// AllocationPadding is a way to increase the size of the physical allocation without increasing
	// the capacity of the allocation visible to a user. This allows extra capacity for block device overhead.
	// Format can be as a percentage (e.g., 3.5%) or a fixed value (e.g., 100MiB).
	AllocationPadding string `json:"alllocationPadding,omitempty"`
}

// NnfStorageProfileRawData defines the XFS-specific configuration
type NnfStorageProfileXFSData struct {
	NnfStorageProfileSharedData `json:",inline"`
}

// NnfStorageProfileRawData defines the GFS2-specific configuration
type NnfStorageProfileGFS2Data struct {
	NnfStorageProfileSharedData `json:",inline"`
}

// NnfStorageProfileRawData defines the Raw-specific configuration
type NnfStorageProfileRawData struct {
	NnfStorageProfileSharedData `json:",inline"`
}

// NnfStorageProfileData defines the desired state of NnfStorageProfile
type NnfStorageProfileData struct {
	// Default is true if this instance is the default resource to use
	// +kubebuilder:default:=false
	Default bool `json:"default,omitempty"`

	// Pinned is true if this instance is describing an active storage resource
	// +kubebuilder:default:=false
	Pinned bool `json:"pinned,omitempty"`

	// LustreStorage defines the Lustre-specific configuration
	LustreStorage NnfStorageProfileLustreData `json:"lustreStorage,omitempty"`

	// GFS2Storage defines the GFS2-specific configuration
	GFS2Storage NnfStorageProfileGFS2Data `json:"gfs2Storage,omitempty"`

	// XFSStorage defines the XFS-specific configuration
	XFSStorage NnfStorageProfileXFSData `json:"xfsStorage,omitempty"`

	// RawStorage defines the Raw-specific configuration
	RawStorage NnfStorageProfileRawData `json:"rawStorage,omitempty"`
}

//+kubebuilder:object:root=true
// +kubebuilder:storageversion
//+kubebuilder:printcolumn:name="DEFAULT",type="boolean",JSONPath=".data.default",description="True if this is the default instance"
//+kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"

// NnfStorageProfile is the Schema for the nnfstorageprofiles API
type NnfStorageProfile struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Data NnfStorageProfileData `json:"data,omitempty"`
}

//+kubebuilder:object:root=true
// +kubebuilder:storageversion

// NnfStorageProfileList contains a list of NnfStorageProfile
type NnfStorageProfileList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NnfStorageProfile `json:"items"`
}

func (n *NnfStorageProfile) GetLustreMiscOptions(target string) NnfStorageProfileLustreMiscOptions {
	switch target {
	case "mgt":
		return n.Data.LustreStorage.MgtOptions
	case "mdt":
		return n.Data.LustreStorage.MdtOptions
	case "mgtmdt":
		return n.Data.LustreStorage.MgtMdtOptions
	case "ost":
		return n.Data.LustreStorage.OstOptions
	default:
		panic("Invalid target type")
	}
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
