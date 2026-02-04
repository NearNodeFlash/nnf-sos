/*
 * Copyright 2023-2025 Hewlett Packard Enterprise Development LP
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

package controller

import (
	"context"
	"fmt"
	"os"
	"reflect"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/NearNodeFlash/nnf-sos/pkg/blockdevice"
	"github.com/NearNodeFlash/nnf-sos/pkg/blockdevice/lvm"
	"github.com/NearNodeFlash/nnf-sos/pkg/filesystem"
	"github.com/go-logr/logr"

	dwsv1alpha7 "github.com/DataWorkflowServices/dws/api/v1alpha7"
	nnfv1alpha10 "github.com/NearNodeFlash/nnf-sos/api/v1alpha10"
)

//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfnodestorages,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfnodestorages/finalizers,verbs=update
//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfstorageprofiles,verbs=get;create;list;watch;update;patch;delete;deletecollection

func getBlockDeviceAndFileSystemForMock(ctx context.Context, c client.Client, nnfNodeStorage *nnfv1alpha10.NnfNodeStorage, index int, log logr.Logger) (blockdevice.BlockDevice, filesystem.FileSystem, error) {

	blockDevice, err := newMockBlockDevice(ctx, c, nnfNodeStorage, index, log)
	if err != nil {
		return nil, nil, dwsv1alpha7.NewResourceError("could not create mock block device").WithError(err).WithMajor()
	}

	fileSystem, err := newMockFileSystem(nnfNodeStorage, index, log)
	if err != nil {
		return nil, nil, dwsv1alpha7.NewResourceError("could not create mock file system").WithError(err).WithMajor()
	}

	return blockDevice, fileSystem, nil
}

func getBlockDeviceAndFileSystemForKind(ctx context.Context, c client.Client, nnfNodeStorage *nnfv1alpha10.NnfNodeStorage, index int, log logr.Logger) (blockdevice.BlockDevice, filesystem.FileSystem, error) {

	blockDevice, err := newMockBlockDevice(ctx, c, nnfNodeStorage, index, log)
	if err != nil {
		return nil, nil, dwsv1alpha7.NewResourceError("could not create mock block device").WithError(err).WithMajor()
	}

	fileSystem, err := newKindFileSystem(nnfNodeStorage, index, log)
	if err != nil {
		return nil, nil, dwsv1alpha7.NewResourceError("could not create mock file system").WithError(err).WithMajor()
	}

	return blockDevice, fileSystem, nil
}

// getBlockDeviceAndFileSystem returns blockdevice and filesystem interfaces based on the allocation type and NnfStorageProfile.
func getBlockDeviceAndFileSystem(ctx context.Context, c client.Client, nnfNodeStorage *nnfv1alpha10.NnfNodeStorage, index int, log logr.Logger) (blockdevice.BlockDevice, filesystem.FileSystem, error) {
	if _, found := os.LookupEnv("NNF_TEST_ENVIRONMENT"); found {
		return getBlockDeviceAndFileSystemForMock(ctx, c, nnfNodeStorage, index, log)

	}
	if os.Getenv("ENVIRONMENT") == "kind" {
		return getBlockDeviceAndFileSystemForKind(ctx, c, nnfNodeStorage, index, log)
	}

	nnfStorageProfile, err := getPinnedStorageProfileFromLabel(ctx, c, nnfNodeStorage)
	if err != nil {
		return nil, nil, dwsv1alpha7.NewResourceError("could not find pinned storage profile").WithError(err).WithFatal()
	}

	switch nnfNodeStorage.Spec.FileSystemType {
	case "raw":
		blockDevice, err := newLvmBlockDevice(ctx, c, nnfNodeStorage, nnfStorageProfile.Data.RawStorage.NnfStorageProfileSharedData, index, log)
		if err != nil {
			return nil, nil, dwsv1alpha7.NewResourceError("could not create LVM block device").WithError(err).WithMajor()
		}

		fileSystem, err := newBindFileSystem(ctx, c, nnfNodeStorage, nnfStorageProfile.Data.RawStorage.NnfStorageProfileSharedData, blockDevice, index, log)
		if err != nil {
			return nil, nil, dwsv1alpha7.NewResourceError("could not create XFS file system").WithError(err).WithMajor()
		}

		return blockDevice, fileSystem, nil
	case "xfs":
		blockDevice, err := newLvmBlockDevice(ctx, c, nnfNodeStorage, nnfStorageProfile.Data.XFSStorage.NnfStorageProfileSharedData, index, log)
		if err != nil {
			return nil, nil, dwsv1alpha7.NewResourceError("could not create LVM block device").WithError(err).WithMajor()
		}

		fileSystem, err := newXfsFileSystem(ctx, c, nnfNodeStorage, nnfStorageProfile.Data.XFSStorage.NnfStorageProfileSharedData, blockDevice, index, log)
		if err != nil {
			return nil, nil, dwsv1alpha7.NewResourceError("could not create XFS file system").WithError(err).WithMajor()
		}

		return blockDevice, fileSystem, nil
	case "gfs2":
		blockDevice, err := newLvmBlockDevice(ctx, c, nnfNodeStorage, nnfStorageProfile.Data.GFS2Storage.NnfStorageProfileSharedData, index, log)
		if err != nil {
			return nil, nil, dwsv1alpha7.NewResourceError("could not create LVM block device").WithError(err).WithMajor()
		}

		fileSystem, err := newGfs2FileSystem(ctx, c, nnfNodeStorage, nnfStorageProfile.Data.GFS2Storage.NnfStorageProfileSharedData, blockDevice, index, log)
		if err != nil {
			return nil, nil, dwsv1alpha7.NewResourceError("could not create GFS2 file system").WithError(err).WithMajor()
		}

		return blockDevice, fileSystem, nil
	case "lustre":
		var targetOptions nnfv1alpha10.NnfStorageProfileLustreTargetOptions

		switch nnfNodeStorage.Spec.LustreStorage.TargetType {
		case "mgt":
			targetOptions = nnfStorageProfile.Data.LustreStorage.MgtOptions.NnfStorageProfileLustreTargetOptions
		case "mgtmdt":
			targetOptions = nnfStorageProfile.Data.LustreStorage.MgtMdtOptions.NnfStorageProfileLustreTargetOptions
		case "mdt":
			targetOptions = nnfStorageProfile.Data.LustreStorage.MdtOptions.NnfStorageProfileLustreTargetOptions
		case "ost":
			targetOptions = nnfStorageProfile.Data.LustreStorage.OstOptions.NnfStorageProfileLustreTargetOptions
		default:
			return nil, nil, dwsv1alpha7.NewResourceError("invalid Lustre target type %s", nnfNodeStorage.Spec.LustreStorage.TargetType).WithFatal()
		}

		blockDevice, err := newZpoolBlockDevice(ctx, c, nnfNodeStorage, targetOptions, index, log)
		if err != nil {
			return nil, nil, dwsv1alpha7.NewResourceError("could not create zpool block device").WithError(err).WithMajor()
		}

		fileSystem, err := newLustreFileSystem(ctx, c, nnfNodeStorage, targetOptions, nnfStorageProfile.Data.LustreStorage.ClientOptions, blockDevice, index, log)
		if err != nil {
			return nil, nil, dwsv1alpha7.NewResourceError("could not create lustre file system").WithError(err).WithMajor()
		}

		return blockDevice, fileSystem, nil
	default:
		break
	}

	return nil, nil, dwsv1alpha7.NewResourceError("unsupported file system type %s", nnfNodeStorage.Spec.FileSystemType).WithMajor()
}

func isNodeBlockStorageCurrent(ctx context.Context, c client.Client, nnfNodeBlockStorage *nnfv1alpha10.NnfNodeBlockStorage) (bool, error) {
	if _, found := os.LookupEnv("NNF_TEST_ENVIRONMENT"); found {
		return true, nil
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      os.Getenv("NNF_POD_NAME"),
			Namespace: os.Getenv("NNF_POD_NAMESPACE"),
		},
	}

	if err := c.Get(ctx, client.ObjectKeyFromObject(pod), pod); err != nil {
		return false, dwsv1alpha7.NewResourceError("could not get pod: %v", client.ObjectKeyFromObject(pod)).WithError(err)
	}

	// The controllers for the NnfNodeStorage and NnfNodeBlockStorage both run in the same pod. Make sure that the NnfNodeBlockStorage
	// has been reconciled by the same instance of the pod that's currently running the NnfNodeStorage controller.
	for _, container := range pod.Status.ContainerStatuses {
		if container.Name != "manager" {
			continue
		}

		if container.State.Running == nil {
			return false, nil
		}

		if container.State.Running.StartedAt != nnfNodeBlockStorage.Status.PodStartTime {
			return false, nil
		}

		return true, nil
	}

	return false, nil
}

func newZpoolBlockDevice(ctx context.Context, c client.Client, nnfNodeStorage *nnfv1alpha10.NnfNodeStorage, targetOptions nnfv1alpha10.NnfStorageProfileLustreTargetOptions, index int, log logr.Logger) (blockdevice.BlockDevice, error) {
	zpool := blockdevice.Zpool{}

	// This is for the fake NnfNodeStorage case. We don't need to create the zpool BlockDevice
	if nnfNodeStorage.Spec.BlockReference.Kind != reflect.TypeOf(nnfv1alpha10.NnfNodeBlockStorage{}).Name() {
		return newMockBlockDevice(ctx, c, nnfNodeStorage, index, log)
	}

	nnfNodeBlockStorage := &nnfv1alpha10.NnfNodeBlockStorage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nnfNodeStorage.GetName(),
			Namespace: nnfNodeStorage.GetNamespace(),
		},
	}

	if err := c.Get(ctx, client.ObjectKeyFromObject(nnfNodeBlockStorage), nnfNodeBlockStorage); err != nil {
		return nil, dwsv1alpha7.NewResourceError("could not get NnfNodeBlockStorage: %v", client.ObjectKeyFromObject(nnfNodeBlockStorage)).WithError(err).WithUserMessage("could not find storage allocation").WithMajor()
	}

	if !nnfNodeBlockStorage.Status.Ready {
		return nil, dwsv1alpha7.NewResourceError("NnfNodeBlockStorage: %v not ready", client.ObjectKeyFromObject(nnfNodeBlockStorage))
	}

	// If the NnfNodeBlockStorage hasn't been updated by this pod yet, then wait for that to happen. The /dev paths may change if the node was
	// rebooted.
	current, err := isNodeBlockStorageCurrent(ctx, c, nnfNodeBlockStorage)
	if err != nil {
		return nil, err
	}

	if !current {
		return nil, dwsv1alpha7.NewResourceError("NnfNodeBlockStorage: %v has stale status", client.ObjectKeyFromObject(nnfNodeBlockStorage)).WithError(err)
	}

	zpoolName, err := zpoolName(ctx, c, nnfNodeStorage, nnfNodeStorage.Spec.LustreStorage.TargetType, nnfNodeStorage.Spec.LustreStorage.StartIndex+index)
	if err != nil {
		return nil, dwsv1alpha7.NewResourceError("could not create zpool name").WithError(err).WithMajor()
	}

	zpool.Log = log
	zpool.Devices = append([]string{}, nnfNodeBlockStorage.Status.Allocations[index].Accesses[os.Getenv("NNF_NODE_NAME")].DevicePaths...)
	zpool.Name = useOverrideIfPresent(targetOptions.VariableOverride, "$ZPOOL_NAME", zpoolName)
	zpool.DataSet = useOverrideIfPresent(targetOptions.VariableOverride, "$ZPOOL_DATA_SET", nnfNodeStorage.Spec.LustreStorage.TargetType)

	zpool.CommandArgs.Create = targetOptions.CmdLines.ZpoolCreate
	zpool.CommandArgs.Destroy = targetOptions.CmdLines.ZpoolDestroy
	zpool.CommandArgs.Replace = targetOptions.CmdLines.ZpoolReplace
	zpool.CommandArgs.Vars = unpackCommandVariables(nnfNodeStorage, index)
	zpool.CommandArgs.Vars = mergeVariables(zpool.CommandArgs.Vars, targetOptions.VariableOverride)

	return &zpool, nil
}

func newLvmBlockDevice(ctx context.Context, c client.Client, nnfNodeStorage *nnfv1alpha10.NnfNodeStorage, profileCommands nnfv1alpha10.NnfStorageProfileSharedData, index int, log logr.Logger) (blockdevice.BlockDevice, error) {
	lvmDesc := blockdevice.Lvm{}
	devices := []string{}

	blockDeviceCommands := profileCommands.BlockDeviceCommands

	blockIndex := index
	if nnfNodeStorage.Spec.SharedAllocation {
		blockIndex = 0
	}

	if nnfNodeStorage.Spec.BlockReference.Kind == reflect.TypeOf(nnfv1alpha10.NnfNodeBlockStorage{}).Name() {
		nnfNodeBlockStorage := &nnfv1alpha10.NnfNodeBlockStorage{
			ObjectMeta: metav1.ObjectMeta{
				Name:      nnfNodeStorage.GetName(),
				Namespace: nnfNodeStorage.GetNamespace(),
			},
		}

		err := c.Get(ctx, client.ObjectKeyFromObject(nnfNodeBlockStorage), nnfNodeBlockStorage)
		if err != nil {
			return nil, dwsv1alpha7.NewResourceError("could not get NnfNodeBlockStorage: %v", client.ObjectKeyFromObject(nnfNodeBlockStorage)).WithError(err).WithUserMessage("could not find storage allocation").WithMajor()
		}

		if !nnfNodeBlockStorage.Status.Ready {
			return nil, dwsv1alpha7.NewResourceError("NnfNodeBlockStorage: %v not ready", client.ObjectKeyFromObject(nnfNodeBlockStorage))
		}

		// If the NnfNodeBlockStorage hasn't been updated by this pod yet, then wait for that to happen. The /dev paths may change if the node was
		// rebooted.
		current, err := isNodeBlockStorageCurrent(ctx, c, nnfNodeBlockStorage)
		if err != nil {
			return nil, err
		}

		if !current {
			return nil, dwsv1alpha7.NewResourceError("NnfNodeBlockStorage: %v has stale status", client.ObjectKeyFromObject(nnfNodeBlockStorage)).WithError(err)
		}

		if len(nnfNodeBlockStorage.Status.Allocations) > 0 && len(nnfNodeBlockStorage.Status.Allocations[blockIndex].Accesses) > 0 {
			devices = nnfNodeBlockStorage.Status.Allocations[blockIndex].Accesses[os.Getenv("NNF_NODE_NAME")].DevicePaths
		} else {
			// FIXME: This scenario has been encountered after rapid transitions from Setup to
			// Teardown. The reason is not yet known, but we need a way to log and capture this
			// information when it happens. For now, log the error and continue. In the future, we
			// may need to handle this situation differently as we gain a better understanding.
			log.Error(fmt.Errorf("newLvmBlockDevice(): no status allocations or no status allocations accesses"),
				"status.allocations or status.allocations[].access have a length of 0",
				"allocations", nnfNodeBlockStorage.Status.Allocations,
				"spec", nnfNodeBlockStorage.Spec,
				"status", nnfNodeBlockStorage.Status,
			)
		}
	}

	for _, device := range devices {
		pv := lvm.NewPhysicalVolume(ctx, device, log)
		pv.Vars = unpackCommandVariables(nnfNodeStorage, index)
		pv.Vars = mergeVariables(pv.Vars, profileCommands.VariableOverride)

		lvmDesc.PhysicalVolumes = append(lvmDesc.PhysicalVolumes, pv)
	}

	vgName, err := volumeGroupName(ctx, c, nnfNodeStorage, blockIndex)
	if err != nil {
		return nil, dwsv1alpha7.NewResourceError("could not get volume group name").WithError(err).WithMajor()
	}

	lvName, err := logicalVolumeName(ctx, c, nnfNodeStorage, index)
	if err != nil {
		return nil, dwsv1alpha7.NewResourceError("could not get logical volume name").WithError(err).WithMajor()
	}

	percentVG := 100
	if nnfNodeStorage.Spec.SharedAllocation {
		percentVG = 100 / nnfNodeStorage.Spec.Count
	}

	vgName = useOverrideIfPresent(profileCommands.VariableOverride, "$VG_NAME", vgName)
	lvName = useOverrideIfPresent(profileCommands.VariableOverride, "$LV_NAME", lvName)
	percentVGString := useOverrideIfPresent(profileCommands.VariableOverride, "$PERCENT_VG", "")
	if len(percentVGString) > 0 {
		fmt.Sscanf(percentVGString, "%d", &percentVG)
	}

	lvmDesc.Log = log
	lvmDesc.VolumeGroup = lvm.NewVolumeGroup(ctx, vgName, lvmDesc.PhysicalVolumes, log)
	lvmDesc.VolumeGroup.Vars = unpackCommandVariables(nnfNodeStorage, index)
	lvmDesc.VolumeGroup.Vars = mergeVariables(lvmDesc.VolumeGroup.Vars, profileCommands.VariableOverride)

	lvmDesc.LogicalVolume = lvm.NewLogicalVolume(ctx, lvName, lvmDesc.VolumeGroup, nnfNodeStorage.Spec.Capacity, percentVG, log)
	lvmDesc.LogicalVolume.Vars = unpackCommandVariables(nnfNodeStorage, index)
	lvmDesc.LogicalVolume.Vars["$LV_INDEX"] = fmt.Sprintf("%d", index)
	lvmDesc.LogicalVolume.Vars = mergeVariables(lvmDesc.LogicalVolume.Vars, profileCommands.VariableOverride)

	if _, found := os.LookupEnv("RABBIT_NODE"); found {
		lvmDesc.CommandArgs.PvArgs.Create = blockDeviceCommands.RabbitCommands.PvCreate
		lvmDesc.CommandArgs.PvArgs.Remove = blockDeviceCommands.RabbitCommands.PvRemove

		lvmDesc.CommandArgs.VgArgs.Create = blockDeviceCommands.RabbitCommands.VgCreate
		lvmDesc.CommandArgs.VgArgs.LockStart = blockDeviceCommands.RabbitCommands.VgChange.LockStart
		lvmDesc.CommandArgs.VgArgs.LockStop = blockDeviceCommands.RabbitCommands.VgChange.LockStop
		lvmDesc.CommandArgs.VgArgs.Remove = blockDeviceCommands.RabbitCommands.VgRemove
		lvmDesc.CommandArgs.VgArgs.Extend = blockDeviceCommands.RabbitCommands.LVMRebuild.VgExtend
		lvmDesc.CommandArgs.VgArgs.Reduce = blockDeviceCommands.RabbitCommands.LVMRebuild.VgReduce

		lvmDesc.CommandArgs.LvArgs.Create = blockDeviceCommands.RabbitCommands.LvCreate
		lvmDesc.CommandArgs.LvArgs.Activate = blockDeviceCommands.RabbitCommands.LvChange.Activate
		lvmDesc.CommandArgs.LvArgs.Deactivate = blockDeviceCommands.RabbitCommands.LvChange.Deactivate
		lvmDesc.CommandArgs.LvArgs.Remove = blockDeviceCommands.RabbitCommands.LvRemove
		lvmDesc.CommandArgs.LvArgs.Repair = blockDeviceCommands.RabbitCommands.LVMRebuild.LvRepair

		lvmDesc.CommandArgs.UserArgs.PreActivate = blockDeviceCommands.RabbitCommands.UserCommands.PreActivate
		lvmDesc.CommandArgs.UserArgs.PostActivate = blockDeviceCommands.RabbitCommands.UserCommands.PostActivate
		lvmDesc.CommandArgs.UserArgs.PreDeactivate = blockDeviceCommands.RabbitCommands.UserCommands.PreDeactivate
		lvmDesc.CommandArgs.UserArgs.PostDeactivate = blockDeviceCommands.RabbitCommands.UserCommands.PostDeactivate
	} else {
		lvmDesc.CommandArgs.VgArgs.LockStart = blockDeviceCommands.ComputeCommands.VgChange.LockStart
		lvmDesc.CommandArgs.VgArgs.LockStop = blockDeviceCommands.ComputeCommands.VgChange.LockStop
		lvmDesc.CommandArgs.LvArgs.Activate = blockDeviceCommands.ComputeCommands.LvChange.Activate
		lvmDesc.CommandArgs.LvArgs.Deactivate = blockDeviceCommands.ComputeCommands.LvChange.Deactivate

		lvmDesc.CommandArgs.UserArgs.PreActivate = blockDeviceCommands.ComputeCommands.UserCommands.PreActivate
		lvmDesc.CommandArgs.UserArgs.PostActivate = blockDeviceCommands.ComputeCommands.UserCommands.PostActivate
		lvmDesc.CommandArgs.UserArgs.PreDeactivate = blockDeviceCommands.ComputeCommands.UserCommands.PreDeactivate
		lvmDesc.CommandArgs.UserArgs.PostDeactivate = blockDeviceCommands.ComputeCommands.UserCommands.PostDeactivate
	}

	return &lvmDesc, nil
}

func newMockBlockDevice(ctx context.Context, c client.Client, nnfNodeStorage *nnfv1alpha10.NnfNodeStorage, index int, log logr.Logger) (blockdevice.BlockDevice, error) {
	blockDevice := blockdevice.MockBlockDevice{
		Log: log,
	}

	return &blockDevice, nil
}

func newBindFileSystem(ctx context.Context, c client.Client, nnfNodeStorage *nnfv1alpha10.NnfNodeStorage, profileCommands nnfv1alpha10.NnfStorageProfileSharedData, blockDevice blockdevice.BlockDevice, index int, log logr.Logger) (filesystem.FileSystem, error) {
	fs := filesystem.SimpleFileSystem{}

	fileSystemCommands := profileCommands.FileSystemCommands

	fs.Log = log
	fs.BlockDevice = blockDevice
	fs.Type = useOverrideIfPresent(profileCommands.VariableOverride, "$FSTYPE", "none")
	fs.MountTarget = useOverrideIfPresent(profileCommands.VariableOverride, "$MOUNT_TARGET", "file")
	fs.TempDir = useOverrideIfPresent(profileCommands.VariableOverride, "$TEMP_DIR", fmt.Sprintf("/mnt/temp/%s-%d", nnfNodeStorage.Name, index))

	if _, found := os.LookupEnv("RABBIT_NODE"); found {
		fs.CommandArgs.Mount = fileSystemCommands.RabbitCommands.Mount
		fs.CommandArgs.Unmount = fileSystemCommands.RabbitCommands.Unmount
		fs.CommandArgs.PreMount = fileSystemCommands.RabbitCommands.UserCommands.PreMount
		fs.CommandArgs.PostMount = fileSystemCommands.RabbitCommands.UserCommands.PostMount
		fs.CommandArgs.PreUnmount = fileSystemCommands.RabbitCommands.UserCommands.PreUnmount
		fs.CommandArgs.PostUnmount = fileSystemCommands.RabbitCommands.UserCommands.PostUnmount

		fs.CommandArgs.PostActivate = profileCommands.UserCommands.PostActivate
		fs.CommandArgs.PreDeactivate = profileCommands.UserCommands.PreDeactivate
		fs.CommandArgs.PostSetup = profileCommands.UserCommands.PostSetup
		fs.CommandArgs.PreTeardown = profileCommands.UserCommands.PreTeardown
	} else {
		fs.CommandArgs.Mount = fileSystemCommands.ComputeCommands.Mount
		fs.CommandArgs.Unmount = fileSystemCommands.ComputeCommands.Unmount
		fs.CommandArgs.PreMount = fileSystemCommands.ComputeCommands.UserCommands.PreMount
		fs.CommandArgs.PostMount = fileSystemCommands.ComputeCommands.UserCommands.PostMount
		fs.CommandArgs.PreUnmount = fileSystemCommands.ComputeCommands.UserCommands.PreUnmount
		fs.CommandArgs.PostUnmount = fileSystemCommands.ComputeCommands.UserCommands.PostUnmount
	}

	fs.CommandArgs.Vars = unpackCommandVariables(nnfNodeStorage, index)
	fs.CommandArgs.Vars["$USERID"] = fmt.Sprintf("%d", nnfNodeStorage.Spec.UserID)
	fs.CommandArgs.Vars["$GROUPID"] = fmt.Sprintf("%d", nnfNodeStorage.Spec.GroupID)

	fs.CommandArgs.Vars = mergeVariables(fs.CommandArgs.Vars, profileCommands.VariableOverride)

	return &fs, nil
}

func newGfs2FileSystem(ctx context.Context, c client.Client, nnfNodeStorage *nnfv1alpha10.NnfNodeStorage, profileCommands nnfv1alpha10.NnfStorageProfileSharedData, blockDevice blockdevice.BlockDevice, index int, log logr.Logger) (filesystem.FileSystem, error) {
	fs := filesystem.SimpleFileSystem{}

	fileSystemCommands := profileCommands.FileSystemCommands

	fs.Log = log
	fs.BlockDevice = blockDevice
	fs.Type = useOverrideIfPresent(profileCommands.VariableOverride, "$FSTYPE", "gfs2")
	fs.MountTarget = useOverrideIfPresent(profileCommands.VariableOverride, "$MOUNT_TARGET", "directory")
	fs.TempDir = useOverrideIfPresent(profileCommands.VariableOverride, "$TEMP_DIR", fmt.Sprintf("/mnt/temp/%s-%d", nnfNodeStorage.Name, index))

	if _, found := os.LookupEnv("RABBIT_NODE"); found {
		fs.CommandArgs.Mount = fileSystemCommands.RabbitCommands.Mount
		fs.CommandArgs.Unmount = fileSystemCommands.RabbitCommands.Unmount
		fs.CommandArgs.Mkfs = fmt.Sprintf("-O %s", fileSystemCommands.RabbitCommands.Mkfs)
		fs.CommandArgs.PreMount = fileSystemCommands.RabbitCommands.UserCommands.PreMount
		fs.CommandArgs.PostMount = fileSystemCommands.RabbitCommands.UserCommands.PostMount
		fs.CommandArgs.PreUnmount = fileSystemCommands.RabbitCommands.UserCommands.PreUnmount
		fs.CommandArgs.PostUnmount = fileSystemCommands.RabbitCommands.UserCommands.PostUnmount

		fs.CommandArgs.PostActivate = profileCommands.UserCommands.PostActivate
		fs.CommandArgs.PreDeactivate = profileCommands.UserCommands.PreDeactivate
		fs.CommandArgs.PostSetup = profileCommands.UserCommands.PostSetup
		fs.CommandArgs.PreTeardown = profileCommands.UserCommands.PreTeardown
	} else {
		fs.CommandArgs.Mount = fileSystemCommands.ComputeCommands.Mount
		fs.CommandArgs.Unmount = fileSystemCommands.ComputeCommands.Unmount
		fs.CommandArgs.PreMount = fileSystemCommands.ComputeCommands.UserCommands.PreMount
		fs.CommandArgs.PostMount = fileSystemCommands.ComputeCommands.UserCommands.PostMount
		fs.CommandArgs.PreUnmount = fileSystemCommands.ComputeCommands.UserCommands.PreUnmount
		fs.CommandArgs.PostUnmount = fileSystemCommands.ComputeCommands.UserCommands.PostUnmount
	}

	fs.CommandArgs.Vars = unpackCommandVariables(nnfNodeStorage, index)
	fs.CommandArgs.Vars["$CLUSTER_NAME"] = nnfNodeStorage.Namespace
	fs.CommandArgs.Vars["$LOCK_SPACE"] = fmt.Sprintf("fs-%02d-%x", index, nnfNodeStorage.GetUID()[0:5])
	fs.CommandArgs.Vars["$PROTOCOL"] = "lock_dlm"
	fs.CommandArgs.Vars["$USERID"] = fmt.Sprintf("%d", nnfNodeStorage.Spec.UserID)
	fs.CommandArgs.Vars["$GROUPID"] = fmt.Sprintf("%d", nnfNodeStorage.Spec.GroupID)
	fs.CommandArgs.Vars = mergeVariables(fs.CommandArgs.Vars, profileCommands.VariableOverride)

	return &fs, nil
}

func newXfsFileSystem(ctx context.Context, c client.Client, nnfNodeStorage *nnfv1alpha10.NnfNodeStorage, profileCommands nnfv1alpha10.NnfStorageProfileSharedData, blockDevice blockdevice.BlockDevice, index int, log logr.Logger) (filesystem.FileSystem, error) {
	fs := filesystem.SimpleFileSystem{}

	fileSystemCommands := profileCommands.FileSystemCommands

	fs.Log = log
	fs.BlockDevice = blockDevice
	fs.Type = useOverrideIfPresent(profileCommands.VariableOverride, "$FSTYPE", "xfs")
	fs.MountTarget = useOverrideIfPresent(profileCommands.VariableOverride, "$MOUNT_TARGET", "directory")
	fs.TempDir = useOverrideIfPresent(profileCommands.VariableOverride, "$TEMP_DIR", fmt.Sprintf("/mnt/temp/%s-%d", nnfNodeStorage.Name, index))

	if _, found := os.LookupEnv("RABBIT_NODE"); found {
		fs.CommandArgs.Mount = fileSystemCommands.RabbitCommands.Mount
		fs.CommandArgs.Unmount = fileSystemCommands.RabbitCommands.Unmount
		fs.CommandArgs.Mkfs = fileSystemCommands.RabbitCommands.Mkfs
		fs.CommandArgs.PreMount = fileSystemCommands.RabbitCommands.UserCommands.PreMount
		fs.CommandArgs.PostMount = fileSystemCommands.RabbitCommands.UserCommands.PostMount
		fs.CommandArgs.PreUnmount = fileSystemCommands.RabbitCommands.UserCommands.PreUnmount
		fs.CommandArgs.PostUnmount = fileSystemCommands.RabbitCommands.UserCommands.PostUnmount

		fs.CommandArgs.PostActivate = profileCommands.UserCommands.PostActivate
		fs.CommandArgs.PreDeactivate = profileCommands.UserCommands.PreDeactivate
		fs.CommandArgs.PostSetup = profileCommands.UserCommands.PostSetup
		fs.CommandArgs.PreTeardown = profileCommands.UserCommands.PreTeardown
	} else {
		fs.CommandArgs.Mount = fileSystemCommands.ComputeCommands.Mount
		fs.CommandArgs.Unmount = fileSystemCommands.ComputeCommands.Unmount
		fs.CommandArgs.PreMount = fileSystemCommands.ComputeCommands.UserCommands.PreMount
		fs.CommandArgs.PostMount = fileSystemCommands.ComputeCommands.UserCommands.PostMount
		fs.CommandArgs.PreUnmount = fileSystemCommands.ComputeCommands.UserCommands.PreUnmount
		fs.CommandArgs.PostUnmount = fileSystemCommands.ComputeCommands.UserCommands.PostUnmount
	}

	fs.CommandArgs.Vars = unpackCommandVariables(nnfNodeStorage, index)
	fs.CommandArgs.Vars["$USERID"] = fmt.Sprintf("%d", nnfNodeStorage.Spec.UserID)
	fs.CommandArgs.Vars["$GROUPID"] = fmt.Sprintf("%d", nnfNodeStorage.Spec.GroupID)
	fs.CommandArgs.Vars = mergeVariables(fs.CommandArgs.Vars, profileCommands.VariableOverride)

	return &fs, nil
}

func newLustreFileSystem(ctx context.Context, c client.Client, nnfNodeStorage *nnfv1alpha10.NnfNodeStorage, targetOptions nnfv1alpha10.NnfStorageProfileLustreTargetOptions, clientOptions nnfv1alpha10.NnfStorageProfileLustreClientOptions, blockDevice blockdevice.BlockDevice, index int, log logr.Logger) (filesystem.FileSystem, error) {
	fs := filesystem.LustreFileSystem{}

	targetPath, err := lustreTargetPath(ctx, c, nnfNodeStorage, nnfNodeStorage.Spec.LustreStorage.TargetType, nnfNodeStorage.Spec.LustreStorage.StartIndex+index)
	if err != nil {
		return nil, dwsv1alpha7.NewResourceError("could not get lustre target mount path").WithError(err).WithMajor()
	}

	if _, found := os.LookupEnv("RABBIT_NODE"); found {
		fs.CommandArgs.Mount = clientOptions.CmdLines.MountRabbit
		fs.CommandArgs.Unmount = clientOptions.CmdLines.UnmountRabbit

		fs.CommandArgs.PreMount = clientOptions.CmdLines.RabbitPreMount
		fs.CommandArgs.PostMount = clientOptions.CmdLines.RabbitPostMount
		fs.CommandArgs.PreUnmount = clientOptions.CmdLines.RabbitPreUnmount
		fs.CommandArgs.PostUnmount = clientOptions.CmdLines.RabbitPostUnmount

		fs.CommandArgs.PostActivate = targetOptions.CmdLines.PostActivate
		fs.CommandArgs.PreDeactivate = targetOptions.CmdLines.PreDeactivate
		fs.CommandArgs.PostSetup = clientOptions.CmdLines.RabbitPostSetup
		fs.CommandArgs.PreTeardown = clientOptions.CmdLines.RabbitPreTeardown
	} else {
		fs.CommandArgs.Mount = clientOptions.CmdLines.MountCompute
		fs.CommandArgs.Unmount = clientOptions.CmdLines.UnmountCompute
		fs.CommandArgs.PreMount = clientOptions.CmdLines.ComputePreMount
		fs.CommandArgs.PostMount = clientOptions.CmdLines.ComputePostMount
		fs.CommandArgs.PreUnmount = clientOptions.CmdLines.ComputePreUnmount
		fs.CommandArgs.PostUnmount = clientOptions.CmdLines.ComputePostUnmount
	}

	fs.Log = log
	fs.BlockDevice = blockDevice

	fs.Name = useOverrideIfPresent(targetOptions.VariableOverride, "$FSNAME", nnfNodeStorage.Spec.LustreStorage.FileSystemName)
	fs.TargetType = useOverrideIfPresent(targetOptions.VariableOverride, "$TARGET_TYPE", nnfNodeStorage.Spec.LustreStorage.TargetType)
	fs.TargetPath = useOverrideIfPresent(targetOptions.VariableOverride, "$TARGET_PATH", targetPath)
	fs.MgsAddress = useOverrideIfPresent(targetOptions.VariableOverride, "$MGS_NID", nnfNodeStorage.Spec.LustreStorage.MgsAddress)
	fs.BackFs = useOverrideIfPresent(targetOptions.VariableOverride, "$BACKFS", nnfNodeStorage.Spec.LustreStorage.BackFs)
	fs.Index = nnfNodeStorage.Spec.LustreStorage.StartIndex + index
	fs.CommandArgs.Mkfs = targetOptions.CmdLines.Mkfs
	fs.CommandArgs.MountTarget = targetOptions.CmdLines.MountTarget
	fs.CommandArgs.UnmountTarget = targetOptions.CmdLines.UnmountTarget

	fs.TempDir = fmt.Sprintf("/mnt/temp/%s-%d", nnfNodeStorage.Name, index)

	components := nnfNodeStorage.Spec.LustreStorage.LustreComponents

	fs.CommandArgs.Vars = unpackCommandVariables(nnfNodeStorage, index)
	fs.CommandArgs.Vars["$USERID"] = fmt.Sprintf("%d", nnfNodeStorage.Spec.UserID)
	fs.CommandArgs.Vars["$GROUPID"] = fmt.Sprintf("%d", nnfNodeStorage.Spec.GroupID)
	fs.CommandArgs.Vars["$NUM_MDTS"] = fmt.Sprintf("%d", len(components.MDTs))
	fs.CommandArgs.Vars["$NUM_MGTS"] = fmt.Sprintf("%d", len(components.MGTs))
	fs.CommandArgs.Vars["$NUM_MGTMDTS"] = fmt.Sprintf("%d", len(components.MGTMDTs))
	fs.CommandArgs.Vars["$NUM_OSTS"] = fmt.Sprintf("%d", len(components.OSTs))
	fs.CommandArgs.Vars["$NUM_NNFNODES"] = fmt.Sprintf("%d", len(components.NNFNodes))

	if nnfNodeStorage.Spec.BlockReference.Kind != reflect.TypeOf(nnfv1alpha10.NnfNodeBlockStorage{}).Name() {
		fs.CommandArgs.Vars = mergeVariables(fs.CommandArgs.Vars, clientOptions.VariableOverride)
	} else {
		fs.CommandArgs.Vars = mergeVariables(fs.CommandArgs.Vars, targetOptions.VariableOverride)
	}

	return &fs, nil
}

func newMockFileSystem(nnfNodeStorage *nnfv1alpha10.NnfNodeStorage, index int, log logr.Logger) (filesystem.FileSystem, error) {
	path := os.Getenv("MOCK_FILE_SYSTEM_PATH")
	if len(path) == 0 {
		path = "/mnt/filesystems"
	}

	fs := filesystem.MockFileSystem{
		Log:  log,
		Path: fmt.Sprintf("/%s/%s-%d", path, nnfNodeStorage.GetName(), index),
	}
	return &fs, nil
}

func newKindFileSystem(nnfNodeStorage *nnfv1alpha10.NnfNodeStorage, index int, log logr.Logger) (filesystem.FileSystem, error) {
	path := os.Getenv("MOCK_FILE_SYSTEM_PATH")
	if len(path) == 0 {
		path = "/mnt/nnf"
	}

	fs := filesystem.KindFileSystem{
		Log:  log,
		Path: fmt.Sprintf("/%s/%s-%d", path, nnfNodeStorage.GetName(), index),
	}
	return &fs, nil

}

func lustreTargetPath(ctx context.Context, c client.Client, nnfNodeStorage *nnfv1alpha10.NnfNodeStorage, targetType string, index int) (string, error) {
	labels := nnfNodeStorage.GetLabels()

	// Use the NnfStorage UID since the NnfStorage exists for as long as the storage allocation exists.
	// This is important for persistent instances
	nnfStorageUid, ok := labels[dwsv1alpha7.OwnerUidLabel]
	if !ok {
		return "", fmt.Errorf("missing Owner UID label on NnfNodeStorage")
	}

	return fmt.Sprintf("/mnt/nnf/%s-%s-%d", nnfStorageUid, targetType, index), nil
}

func zpoolName(ctx context.Context, c client.Client, nnfNodeStorage *nnfv1alpha10.NnfNodeStorage, targetType string, index int) (string, error) {
	labels := nnfNodeStorage.GetLabels()

	// Use the NnfStorage UID since the NnfStorage exists for as long as the storage allocation exists.
	// This is important for persistent instances
	nnfStorageUid, ok := labels[dwsv1alpha7.OwnerUidLabel]
	if !ok {
		return "", fmt.Errorf("missing Owner UID label on NnfNodeStorage")
	}

	return fmt.Sprintf("pool-%s-%s-%d", nnfStorageUid, targetType, index), nil
}

func volumeGroupName(ctx context.Context, c client.Client, nnfNodeStorage *nnfv1alpha10.NnfNodeStorage, index int) (string, error) {
	labels := nnfNodeStorage.GetLabels()

	// Use the NnfStorage UID since the NnfStorage exists for as long as the storage allocation exists.
	// This is important for persistent instances
	nnfStorageUid, ok := labels[dwsv1alpha7.OwnerUidLabel]
	if !ok {
		return "", fmt.Errorf("missing Owner UID label on NnfNodeStorage")
	}
	directiveIndex, ok := labels[nnfv1alpha10.DirectiveIndexLabel]
	if !ok {
		return "", fmt.Errorf("missing directive index label on NnfNodeStorage")
	}

	if nnfNodeStorage.Spec.SharedAllocation {
		return fmt.Sprintf("%s_%s", nnfStorageUid, directiveIndex), nil
	}

	return fmt.Sprintf("%s_%s_%d", nnfStorageUid, directiveIndex, index), nil
}

func logicalVolumeName(ctx context.Context, c client.Client, nnfNodeStorage *nnfv1alpha10.NnfNodeStorage, index int) (string, error) {
	if nnfNodeStorage.Spec.SharedAllocation {
		// For a shared VG, the LV name must be unique in the VG
		return fmt.Sprintf("lv-%d", index), nil
	}

	return "lv", nil
}

func unpackCommandVariables(nnfNodeStorage *nnfv1alpha10.NnfNodeStorage, index int) map[string]string {
	variables := map[string]string{}

	for _, commandVariable := range nnfNodeStorage.Spec.CommandVariables {
		if !commandVariable.Indexed {
			variables[commandVariable.Name] = commandVariable.Value
		} else {
			variables[commandVariable.Name] = commandVariable.IndexedValues[index]
		}
	}

	return variables
}

// mergeVariables merges overrideVars into baseVars, with overrideVars taking precedence in case of key conflicts.
func mergeVariables(baseVars map[string]string, overrideVars map[string]string) map[string]string {
	if baseVars == nil {
		baseVars = make(map[string]string)
	}

	for key, value := range overrideVars {
		baseVars[key] = value
	}
	return baseVars
}

func useOverrideIfPresent(overrideVars map[string]string, key string, defaultValue string) string {
	if val, exists := overrideVars[key]; exists {
		return val
	}
	return defaultValue
}
