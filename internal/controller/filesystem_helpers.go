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

	dwsv1alpha5 "github.com/DataWorkflowServices/dws/api/v1alpha5"
	nnfv1alpha7 "github.com/NearNodeFlash/nnf-sos/api/v1alpha7"
)

//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfnodestorages,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfnodestorages/finalizers,verbs=update
//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfstorageprofiles,verbs=get;create;list;watch;update;patch;delete;deletecollection

func getBlockDeviceAndFileSystemForMock(ctx context.Context, c client.Client, nnfNodeStorage *nnfv1alpha7.NnfNodeStorage, index int, log logr.Logger) (blockdevice.BlockDevice, filesystem.FileSystem, error) {

	blockDevice, err := newMockBlockDevice(ctx, c, nnfNodeStorage, index, log)
	if err != nil {
		return nil, nil, dwsv1alpha5.NewResourceError("could not create mock block device").WithError(err).WithMajor()
	}

	fileSystem, err := newMockFileSystem(nnfNodeStorage, index, log)
	if err != nil {
		return nil, nil, dwsv1alpha5.NewResourceError("could not create mock file system").WithError(err).WithMajor()
	}

	return blockDevice, fileSystem, nil
}

func getBlockDeviceAndFileSystemForKind(ctx context.Context, c client.Client, nnfNodeStorage *nnfv1alpha7.NnfNodeStorage, index int, log logr.Logger) (blockdevice.BlockDevice, filesystem.FileSystem, error) {

	blockDevice, err := newMockBlockDevice(ctx, c, nnfNodeStorage, index, log)
	if err != nil {
		return nil, nil, dwsv1alpha5.NewResourceError("could not create mock block device").WithError(err).WithMajor()
	}

	fileSystem, err := newKindFileSystem(nnfNodeStorage, index, log)
	if err != nil {
		return nil, nil, dwsv1alpha5.NewResourceError("could not create mock file system").WithError(err).WithMajor()
	}

	return blockDevice, fileSystem, nil
}

// getBlockDeviceAndFileSystem returns blockdevice and filesystem interfaces based on the allocation type and NnfStorageProfile.
func getBlockDeviceAndFileSystem(ctx context.Context, c client.Client, nnfNodeStorage *nnfv1alpha7.NnfNodeStorage, index int, log logr.Logger) (blockdevice.BlockDevice, filesystem.FileSystem, error) {
	if _, found := os.LookupEnv("NNF_TEST_ENVIRONMENT"); found {
		return getBlockDeviceAndFileSystemForMock(ctx, c, nnfNodeStorage, index, log)

	}
	if os.Getenv("ENVIRONMENT") == "kind" {
		return getBlockDeviceAndFileSystemForKind(ctx, c, nnfNodeStorage, index, log)
	}

	nnfStorageProfile, err := getPinnedStorageProfileFromLabel(ctx, c, nnfNodeStorage)
	if err != nil {
		return nil, nil, dwsv1alpha5.NewResourceError("could not find pinned storage profile").WithError(err).WithFatal()
	}

	switch nnfNodeStorage.Spec.FileSystemType {
	case "raw":
		blockDevice, err := newLvmBlockDevice(ctx, c, nnfNodeStorage, nnfStorageProfile.Data.RawStorage.CmdLines, index, log)
		if err != nil {
			return nil, nil, dwsv1alpha5.NewResourceError("could not create LVM block device").WithError(err).WithMajor()
		}

		fileSystem, err := newBindFileSystem(ctx, c, nnfNodeStorage, nnfStorageProfile.Data.RawStorage.CmdLines, blockDevice, index, log)
		if err != nil {
			return nil, nil, dwsv1alpha5.NewResourceError("could not create XFS file system").WithError(err).WithMajor()
		}

		return blockDevice, fileSystem, nil
	case "xfs":
		blockDevice, err := newLvmBlockDevice(ctx, c, nnfNodeStorage, nnfStorageProfile.Data.XFSStorage.CmdLines, index, log)
		if err != nil {
			return nil, nil, dwsv1alpha5.NewResourceError("could not create LVM block device").WithError(err).WithMajor()
		}

		fileSystem, err := newXfsFileSystem(ctx, c, nnfNodeStorage, nnfStorageProfile.Data.XFSStorage.CmdLines, blockDevice, index, log)
		if err != nil {
			return nil, nil, dwsv1alpha5.NewResourceError("could not create XFS file system").WithError(err).WithMajor()
		}

		return blockDevice, fileSystem, nil
	case "gfs2":
		blockDevice, err := newLvmBlockDevice(ctx, c, nnfNodeStorage, nnfStorageProfile.Data.GFS2Storage.CmdLines, index, log)
		if err != nil {
			return nil, nil, dwsv1alpha5.NewResourceError("could not create LVM block device").WithError(err).WithMajor()
		}

		fileSystem, err := newGfs2FileSystem(ctx, c, nnfNodeStorage, nnfStorageProfile.Data.GFS2Storage.CmdLines, blockDevice, index, log)
		if err != nil {
			return nil, nil, dwsv1alpha5.NewResourceError("could not create GFS2 file system").WithError(err).WithMajor()
		}

		return blockDevice, fileSystem, nil
	case "lustre":
		var commandLines nnfv1alpha7.NnfStorageProfileLustreCmdLines

		switch nnfNodeStorage.Spec.LustreStorage.TargetType {
		case "mgt":
			commandLines = nnfStorageProfile.Data.LustreStorage.MgtCmdLines
		case "mgtmdt":
			commandLines = nnfStorageProfile.Data.LustreStorage.MgtMdtCmdLines
		case "mdt":
			commandLines = nnfStorageProfile.Data.LustreStorage.MdtCmdLines
		case "ost":
			commandLines = nnfStorageProfile.Data.LustreStorage.OstCmdLines
		default:
			return nil, nil, dwsv1alpha5.NewResourceError("invalid Lustre target type %s", nnfNodeStorage.Spec.LustreStorage.TargetType).WithFatal()
		}

		blockDevice, err := newZpoolBlockDevice(ctx, c, nnfNodeStorage, commandLines, index, log)
		if err != nil {
			return nil, nil, dwsv1alpha5.NewResourceError("could not create zpool block device").WithError(err).WithMajor()
		}

		fileSystem, err := newLustreFileSystem(ctx, c, nnfNodeStorage, commandLines, nnfStorageProfile.Data.LustreStorage.ClientCmdLines, blockDevice, index, log)
		if err != nil {
			return nil, nil, dwsv1alpha5.NewResourceError("could not create lustre file system").WithError(err).WithMajor()
		}

		return blockDevice, fileSystem, nil
	default:
		break
	}

	return nil, nil, dwsv1alpha5.NewResourceError("unsupported file system type %s", nnfNodeStorage.Spec.FileSystemType).WithMajor()
}

func isNodeBlockStorageCurrent(ctx context.Context, c client.Client, nnfNodeBlockStorage *nnfv1alpha7.NnfNodeBlockStorage) (bool, error) {
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
		return false, dwsv1alpha5.NewResourceError("could not get pod: %v", client.ObjectKeyFromObject(pod)).WithError(err)
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

func newZpoolBlockDevice(ctx context.Context, c client.Client, nnfNodeStorage *nnfv1alpha7.NnfNodeStorage, cmdLines nnfv1alpha7.NnfStorageProfileLustreCmdLines, index int, log logr.Logger) (blockdevice.BlockDevice, error) {
	zpool := blockdevice.Zpool{}

	// This is for the fake NnfNodeStorage case. We don't need to create the zpool BlockDevice
	if nnfNodeStorage.Spec.BlockReference.Kind != reflect.TypeOf(nnfv1alpha7.NnfNodeBlockStorage{}).Name() {
		return newMockBlockDevice(ctx, c, nnfNodeStorage, index, log)
	}

	nnfNodeBlockStorage := &nnfv1alpha7.NnfNodeBlockStorage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nnfNodeStorage.GetName(),
			Namespace: nnfNodeStorage.GetNamespace(),
		},
	}

	if err := c.Get(ctx, client.ObjectKeyFromObject(nnfNodeBlockStorage), nnfNodeBlockStorage); err != nil {
		return nil, dwsv1alpha5.NewResourceError("could not get NnfNodeBlockStorage: %v", client.ObjectKeyFromObject(nnfNodeBlockStorage)).WithError(err).WithUserMessage("could not find storage allocation").WithMajor()
	}

	if !nnfNodeBlockStorage.Status.Ready {
		return nil, dwsv1alpha5.NewResourceError("NnfNodeBlockStorage: %v not ready", client.ObjectKeyFromObject(nnfNodeBlockStorage))
	}

	// If the NnfNodeBlockStorage hasn't been updated by this pod yet, then wait for that to happen. The /dev paths may change if the node was
	// rebooted.
	current, err := isNodeBlockStorageCurrent(ctx, c, nnfNodeBlockStorage)
	if err != nil {
		return nil, err
	}

	if !current {
		return nil, dwsv1alpha5.NewResourceError("NnfNodeBlockStorage: %v has stale status", client.ObjectKeyFromObject(nnfNodeBlockStorage)).WithError(err)
	}

	zpoolName, err := zpoolName(ctx, c, nnfNodeStorage, nnfNodeStorage.Spec.LustreStorage.TargetType, nnfNodeStorage.Spec.LustreStorage.StartIndex+index)
	if err != nil {
		return nil, dwsv1alpha5.NewResourceError("could not create zpool name").WithError(err).WithMajor()
	}

	zpool.Log = log
	zpool.Devices = append([]string{}, nnfNodeBlockStorage.Status.Allocations[index].Accesses[os.Getenv("NNF_NODE_NAME")].DevicePaths...)
	zpool.Name = zpoolName
	zpool.DataSet = nnfNodeStorage.Spec.LustreStorage.TargetType

	zpool.CommandArgs.Create = cmdLines.ZpoolCreate
	zpool.CommandArgs.Vars = unpackCommandVariables(nnfNodeStorage, index)

	return &zpool, nil
}

func newLvmBlockDevice(ctx context.Context, c client.Client, nnfNodeStorage *nnfv1alpha7.NnfNodeStorage, cmdLines nnfv1alpha7.NnfStorageProfileCmdLines, index int, log logr.Logger) (blockdevice.BlockDevice, error) {
	lvmDesc := blockdevice.Lvm{}
	devices := []string{}

	blockIndex := index
	if nnfNodeStorage.Spec.SharedAllocation {
		blockIndex = 0
	}

	if nnfNodeStorage.Spec.BlockReference.Kind == reflect.TypeOf(nnfv1alpha7.NnfNodeBlockStorage{}).Name() {
		nnfNodeBlockStorage := &nnfv1alpha7.NnfNodeBlockStorage{
			ObjectMeta: metav1.ObjectMeta{
				Name:      nnfNodeStorage.GetName(),
				Namespace: nnfNodeStorage.GetNamespace(),
			},
		}

		err := c.Get(ctx, client.ObjectKeyFromObject(nnfNodeBlockStorage), nnfNodeBlockStorage)
		if err != nil {
			return nil, dwsv1alpha5.NewResourceError("could not get NnfNodeBlockStorage: %v", client.ObjectKeyFromObject(nnfNodeBlockStorage)).WithError(err).WithUserMessage("could not find storage allocation").WithMajor()
		}

		if !nnfNodeBlockStorage.Status.Ready {
			return nil, dwsv1alpha5.NewResourceError("NnfNodeBlockStorage: %v not ready", client.ObjectKeyFromObject(nnfNodeBlockStorage))
		}

		// If the NnfNodeBlockStorage hasn't been updated by this pod yet, then wait for that to happen. The /dev paths may change if the node was
		// rebooted.
		current, err := isNodeBlockStorageCurrent(ctx, c, nnfNodeBlockStorage)
		if err != nil {
			return nil, err
		}

		if !current {
			return nil, dwsv1alpha5.NewResourceError("NnfNodeBlockStorage: %v has stale status", client.ObjectKeyFromObject(nnfNodeBlockStorage)).WithError(err)
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

		lvmDesc.PhysicalVolumes = append(lvmDesc.PhysicalVolumes, pv)
	}

	vgName, err := volumeGroupName(ctx, c, nnfNodeStorage, blockIndex)
	if err != nil {
		return nil, dwsv1alpha5.NewResourceError("could not get volume group name").WithError(err).WithMajor()
	}

	lvName, err := logicalVolumeName(ctx, c, nnfNodeStorage, index)
	if err != nil {
		return nil, dwsv1alpha5.NewResourceError("could not get logical volume name").WithError(err).WithMajor()
	}

	percentVG := 100
	if nnfNodeStorage.Spec.SharedAllocation {
		percentVG = 100 / nnfNodeStorage.Spec.Count
	}

	lvmDesc.Log = log
	lvmDesc.VolumeGroup = lvm.NewVolumeGroup(ctx, vgName, lvmDesc.PhysicalVolumes, log)
	lvmDesc.VolumeGroup.Vars = unpackCommandVariables(nnfNodeStorage, index)

	lvmDesc.LogicalVolume = lvm.NewLogicalVolume(ctx, lvName, lvmDesc.VolumeGroup, nnfNodeStorage.Spec.Capacity, percentVG, log)
	lvmDesc.LogicalVolume.Vars = unpackCommandVariables(nnfNodeStorage, index)

	lvmDesc.CommandArgs.PvArgs.Create = cmdLines.PvCreate
	lvmDesc.CommandArgs.PvArgs.Remove = cmdLines.PvRemove

	lvmDesc.CommandArgs.VgArgs.Create = cmdLines.VgCreate
	lvmDesc.CommandArgs.VgArgs.LockStart = cmdLines.VgChange.LockStart
	lvmDesc.CommandArgs.VgArgs.LockStop = cmdLines.VgChange.LockStop
	lvmDesc.CommandArgs.VgArgs.Remove = cmdLines.VgRemove

	lvmDesc.CommandArgs.LvArgs.Create = cmdLines.LvCreate
	lvmDesc.CommandArgs.LvArgs.Activate = cmdLines.LvChange.Activate
	lvmDesc.CommandArgs.LvArgs.Deactivate = cmdLines.LvChange.Deactivate
	lvmDesc.CommandArgs.LvArgs.Remove = cmdLines.LvRemove

	return &lvmDesc, nil
}

func newMockBlockDevice(ctx context.Context, c client.Client, nnfNodeStorage *nnfv1alpha7.NnfNodeStorage, index int, log logr.Logger) (blockdevice.BlockDevice, error) {
	blockDevice := blockdevice.MockBlockDevice{
		Log: log,
	}

	return &blockDevice, nil
}

func newBindFileSystem(ctx context.Context, c client.Client, nnfNodeStorage *nnfv1alpha7.NnfNodeStorage, cmdLines nnfv1alpha7.NnfStorageProfileCmdLines, blockDevice blockdevice.BlockDevice, index int, log logr.Logger) (filesystem.FileSystem, error) {
	fs := filesystem.SimpleFileSystem{}

	fs.Log = log
	fs.BlockDevice = blockDevice
	fs.Type = "none"
	fs.MountTarget = "file"
	fs.TempDir = fmt.Sprintf("/mnt/temp/%s-%d", nnfNodeStorage.Name, index)

	fs.CommandArgs.Mount = "-o bind $DEVICE $MOUNT_PATH"
	fs.CommandArgs.PostMount = cmdLines.PostMount
	fs.CommandArgs.PreUnmount = cmdLines.PreUnmount

	fs.CommandArgs.Vars = unpackCommandVariables(nnfNodeStorage, index)
	fs.CommandArgs.Vars["$USERID"] = fmt.Sprintf("%d", nnfNodeStorage.Spec.UserID)
	fs.CommandArgs.Vars["$GROUPID"] = fmt.Sprintf("%d", nnfNodeStorage.Spec.GroupID)

	return &fs, nil
}

func newGfs2FileSystem(ctx context.Context, c client.Client, nnfNodeStorage *nnfv1alpha7.NnfNodeStorage, cmdLines nnfv1alpha7.NnfStorageProfileCmdLines, blockDevice blockdevice.BlockDevice, index int, log logr.Logger) (filesystem.FileSystem, error) {
	fs := filesystem.SimpleFileSystem{}

	fs.Log = log
	fs.BlockDevice = blockDevice
	fs.Type = "gfs2"
	fs.MountTarget = "directory"
	fs.TempDir = fmt.Sprintf("/mnt/temp/%s-%d", nnfNodeStorage.Name, index)

	if _, found := os.LookupEnv("RABBIT_NODE"); found {
		fs.CommandArgs.Mount = cmdLines.MountRabbit
	} else {
		fs.CommandArgs.Mount = cmdLines.MountCompute
	}
	fs.CommandArgs.PostMount = cmdLines.PostMount
	fs.CommandArgs.PreUnmount = cmdLines.PreUnmount
	fs.CommandArgs.Mkfs = fmt.Sprintf("-O %s", cmdLines.Mkfs)

	fs.CommandArgs.Vars = unpackCommandVariables(nnfNodeStorage, index)
	fs.CommandArgs.Vars["$CLUSTER_NAME"] = nnfNodeStorage.Namespace
	fs.CommandArgs.Vars["$LOCK_SPACE"] = fmt.Sprintf("fs-%02d-%x", index, nnfNodeStorage.GetUID()[0:5])
	fs.CommandArgs.Vars["$PROTOCOL"] = "lock_dlm"
	fs.CommandArgs.Vars["$USERID"] = fmt.Sprintf("%d", nnfNodeStorage.Spec.UserID)
	fs.CommandArgs.Vars["$GROUPID"] = fmt.Sprintf("%d", nnfNodeStorage.Spec.GroupID)

	return &fs, nil
}

func newXfsFileSystem(ctx context.Context, c client.Client, nnfNodeStorage *nnfv1alpha7.NnfNodeStorage, cmdLines nnfv1alpha7.NnfStorageProfileCmdLines, blockDevice blockdevice.BlockDevice, index int, log logr.Logger) (filesystem.FileSystem, error) {
	fs := filesystem.SimpleFileSystem{}

	fs.Log = log
	fs.BlockDevice = blockDevice
	fs.Type = "xfs"
	fs.MountTarget = "directory"
	fs.TempDir = fmt.Sprintf("/mnt/temp/%s-%d", nnfNodeStorage.Name, index)

	if _, found := os.LookupEnv("RABBIT_NODE"); found {
		fs.CommandArgs.Mount = cmdLines.MountRabbit
	} else {
		fs.CommandArgs.Mount = cmdLines.MountCompute
	}
	fs.CommandArgs.PostMount = cmdLines.PostMount
	fs.CommandArgs.PreUnmount = cmdLines.PreUnmount
	fs.CommandArgs.Mkfs = cmdLines.Mkfs

	fs.CommandArgs.Vars = unpackCommandVariables(nnfNodeStorage, index)
	fs.CommandArgs.Vars["$USERID"] = fmt.Sprintf("%d", nnfNodeStorage.Spec.UserID)
	fs.CommandArgs.Vars["$GROUPID"] = fmt.Sprintf("%d", nnfNodeStorage.Spec.GroupID)

	return &fs, nil
}

func newLustreFileSystem(ctx context.Context, c client.Client, nnfNodeStorage *nnfv1alpha7.NnfNodeStorage, targetCmdLines nnfv1alpha7.NnfStorageProfileLustreCmdLines, clientCmdLines nnfv1alpha7.NnfStorageProfileLustreClientCmdLines, blockDevice blockdevice.BlockDevice, index int, log logr.Logger) (filesystem.FileSystem, error) {
	fs := filesystem.LustreFileSystem{}

	targetPath, err := lustreTargetPath(ctx, c, nnfNodeStorage, nnfNodeStorage.Spec.LustreStorage.TargetType, nnfNodeStorage.Spec.LustreStorage.StartIndex+index)
	if err != nil {
		return nil, dwsv1alpha5.NewResourceError("could not get lustre target mount path").WithError(err).WithMajor()
	}

	mountCommand := ""
	if _, found := os.LookupEnv("RABBIT_NODE"); found {
		mountCommand = clientCmdLines.MountRabbit
	} else {
		mountCommand = clientCmdLines.MountCompute
	}

	fs.Log = log
	fs.BlockDevice = blockDevice
	fs.Name = nnfNodeStorage.Spec.LustreStorage.FileSystemName
	fs.TargetType = nnfNodeStorage.Spec.LustreStorage.TargetType
	fs.TargetPath = targetPath
	fs.MgsAddress = nnfNodeStorage.Spec.LustreStorage.MgsAddress
	fs.Index = nnfNodeStorage.Spec.LustreStorage.StartIndex + index
	fs.BackFs = nnfNodeStorage.Spec.LustreStorage.BackFs
	fs.CommandArgs.Mkfs = targetCmdLines.Mkfs
	fs.CommandArgs.MountTarget = targetCmdLines.MountTarget
	fs.CommandArgs.Mount = mountCommand
	fs.CommandArgs.PostActivate = targetCmdLines.PostActivate
	fs.CommandArgs.PostMount = clientCmdLines.RabbitPostMount
	fs.CommandArgs.PreUnmount = clientCmdLines.RabbitPreUnmount
	fs.CommandArgs.PreDeactivate = targetCmdLines.PreDeactivate
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

	return &fs, nil
}

func newMockFileSystem(nnfNodeStorage *nnfv1alpha7.NnfNodeStorage, index int, log logr.Logger) (filesystem.FileSystem, error) {
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

func newKindFileSystem(nnfNodeStorage *nnfv1alpha7.NnfNodeStorage, index int, log logr.Logger) (filesystem.FileSystem, error) {
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

func lustreTargetPath(ctx context.Context, c client.Client, nnfNodeStorage *nnfv1alpha7.NnfNodeStorage, targetType string, index int) (string, error) {
	labels := nnfNodeStorage.GetLabels()

	// Use the NnfStorage UID since the NnfStorage exists for as long as the storage allocation exists.
	// This is important for persistent instances
	nnfStorageUid, ok := labels[dwsv1alpha5.OwnerUidLabel]
	if !ok {
		return "", fmt.Errorf("missing Owner UID label on NnfNodeStorage")
	}

	return fmt.Sprintf("/mnt/nnf/%s-%s-%d", nnfStorageUid, targetType, index), nil
}

func zpoolName(ctx context.Context, c client.Client, nnfNodeStorage *nnfv1alpha7.NnfNodeStorage, targetType string, index int) (string, error) {
	labels := nnfNodeStorage.GetLabels()

	// Use the NnfStorage UID since the NnfStorage exists for as long as the storage allocation exists.
	// This is important for persistent instances
	nnfStorageUid, ok := labels[dwsv1alpha5.OwnerUidLabel]
	if !ok {
		return "", fmt.Errorf("missing Owner UID label on NnfNodeStorage")
	}

	return fmt.Sprintf("pool-%s-%s-%d", nnfStorageUid, targetType, index), nil
}

func volumeGroupName(ctx context.Context, c client.Client, nnfNodeStorage *nnfv1alpha7.NnfNodeStorage, index int) (string, error) {
	labels := nnfNodeStorage.GetLabels()

	// Use the NnfStorage UID since the NnfStorage exists for as long as the storage allocation exists.
	// This is important for persistent instances
	nnfStorageUid, ok := labels[dwsv1alpha5.OwnerUidLabel]
	if !ok {
		return "", fmt.Errorf("missing Owner UID label on NnfNodeStorage")
	}
	directiveIndex, ok := labels[nnfv1alpha7.DirectiveIndexLabel]
	if !ok {
		return "", fmt.Errorf("missing directive index label on NnfNodeStorage")
	}

	if nnfNodeStorage.Spec.SharedAllocation {
		return fmt.Sprintf("%s_%s", nnfStorageUid, directiveIndex), nil
	}

	return fmt.Sprintf("%s_%s_%d", nnfStorageUid, directiveIndex, index), nil
}

func logicalVolumeName(ctx context.Context, c client.Client, nnfNodeStorage *nnfv1alpha7.NnfNodeStorage, index int) (string, error) {
	if nnfNodeStorage.Spec.SharedAllocation {
		// For a shared VG, the LV name must be unique in the VG
		return fmt.Sprintf("lv-%d", index), nil
	}

	return "lv", nil
}

func unpackCommandVariables(nnfNodeStorage *nnfv1alpha7.NnfNodeStorage, index int) map[string]string {
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
