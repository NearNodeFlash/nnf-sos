/*
 * Copyright 2023 Hewlett Packard Enterprise Development LP
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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/NearNodeFlash/nnf-sos/pkg/blockdevice"
	"github.com/NearNodeFlash/nnf-sos/pkg/blockdevice/lvm"
	"github.com/NearNodeFlash/nnf-sos/pkg/filesystem"
	"github.com/go-logr/logr"

	dwsv1alpha2 "github.com/DataWorkflowServices/dws/api/v1alpha2"
	nnfv1alpha1 "github.com/NearNodeFlash/nnf-sos/api/v1alpha1"
)

//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfnodestorages,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfnodestorages/finalizers,verbs=update
//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfstorageprofiles,verbs=get;create;list;watch;update;patch;delete;deletecollection

func getBlockDeviceAndFileSystemForKind(ctx context.Context, c client.Client, nnfNodeStorage *nnfv1alpha1.NnfNodeStorage, index int, log logr.Logger) (blockdevice.BlockDevice, filesystem.FileSystem, error) {

	blockDevice, err := newMockBlockDevice(ctx, c, nnfNodeStorage, index, log)
	if err != nil {
		return nil, nil, dwsv1alpha2.NewResourceError("could not create mock block device").WithError(err).WithMajor()
	}

	fileSystem, err := newMockFileSystem(ctx, c, nnfNodeStorage, blockDevice, index, log)
	if err != nil {
		return nil, nil, dwsv1alpha2.NewResourceError("could not create mock file system").WithError(err).WithMajor()
	}

	return blockDevice, fileSystem, nil
}

// getBlockDeviceAndFileSystem returns blockdevice and filesystem interfaces based on the allocation type and NnfStorageProfile.
func getBlockDeviceAndFileSystem(ctx context.Context, c client.Client, nnfNodeStorage *nnfv1alpha1.NnfNodeStorage, index int, log logr.Logger) (blockdevice.BlockDevice, filesystem.FileSystem, error) {
	_, found := os.LookupEnv("NNF_TEST_ENVIRONMENT")
	if found || os.Getenv("ENVIRONMENT") == "kind" {
		return getBlockDeviceAndFileSystemForKind(ctx, c, nnfNodeStorage, index, log)
	}

	nnfStorageProfile, err := getPinnedStorageProfileFromLabel(ctx, c, nnfNodeStorage)
	if err != nil {
		return nil, nil, dwsv1alpha2.NewResourceError("could not find pinned storage profile").WithError(err).WithFatal()
	}

	switch nnfNodeStorage.Spec.FileSystemType {
	case "raw":
		blockDevice, err := newLvmBlockDevice(ctx, c, nnfNodeStorage, nnfStorageProfile.Data.RawStorage.CmdLines, index, log)
		if err != nil {
			return nil, nil, dwsv1alpha2.NewResourceError("could not create LVM block device").WithError(err).WithMajor()
		}

		fileSystem, err := newBindFileSystem(ctx, c, nnfNodeStorage, blockDevice, index, log)
		if err != nil {
			return nil, nil, dwsv1alpha2.NewResourceError("could not create XFS file system").WithError(err).WithMajor()
		}

		return blockDevice, fileSystem, nil
	case "xfs":
		blockDevice, err := newLvmBlockDevice(ctx, c, nnfNodeStorage, nnfStorageProfile.Data.XFSStorage.CmdLines, index, log)
		if err != nil {
			return nil, nil, dwsv1alpha2.NewResourceError("could not create LVM block device").WithError(err).WithMajor()
		}

		fileSystem, err := newXfsFileSystem(ctx, c, nnfNodeStorage, nnfStorageProfile.Data.XFSStorage.CmdLines, blockDevice, index, log)
		if err != nil {
			return nil, nil, dwsv1alpha2.NewResourceError("could not create XFS file system").WithError(err).WithMajor()
		}

		return blockDevice, fileSystem, nil
	case "gfs2":
		blockDevice, err := newLvmBlockDevice(ctx, c, nnfNodeStorage, nnfStorageProfile.Data.GFS2Storage.CmdLines, index, log)
		if err != nil {
			return nil, nil, dwsv1alpha2.NewResourceError("could not create LVM block device").WithError(err).WithMajor()
		}

		fileSystem, err := newGfs2FileSystem(ctx, c, nnfNodeStorage, nnfStorageProfile.Data.GFS2Storage.CmdLines, blockDevice, index, log)
		if err != nil {
			return nil, nil, dwsv1alpha2.NewResourceError("could not create GFS2 file system").WithError(err).WithMajor()
		}

		return blockDevice, fileSystem, nil
	case "lustre":
		commandLines := nnfv1alpha1.NnfStorageProfileLustreCmdLines{}

		switch nnfNodeStorage.Spec.LustreStorage.TargetType {
		case "mgt":
			commandLines = nnfStorageProfile.Data.LustreStorage.MgtCmdLines
			break
		case "mgtmdt":
			commandLines = nnfStorageProfile.Data.LustreStorage.MgtMdtCmdLines
			break
		case "mdt":
			commandLines = nnfStorageProfile.Data.LustreStorage.MdtCmdLines
			break
		case "ost":
			commandLines = nnfStorageProfile.Data.LustreStorage.OstCmdLines
			break
		default:
			return nil, nil, dwsv1alpha2.NewResourceError("invalid Lustre target type %s", nnfNodeStorage.Spec.LustreStorage.TargetType).WithFatal()
		}

		blockDevice, err := newZpoolBlockDevice(ctx, c, nnfNodeStorage, commandLines, index, log)
		if err != nil {
			return nil, nil, dwsv1alpha2.NewResourceError("could not create zpool block device").WithError(err).WithMajor()
		}

		mountCommand := ""
		if _, found := os.LookupEnv("RABBIT_NODE"); found {
			mountCommand = nnfStorageProfile.Data.LustreStorage.MountRabbit
		} else {
			mountCommand = nnfStorageProfile.Data.LustreStorage.MountCompute
		}

		fileSystem, err := newLustreFileSystem(ctx, c, nnfNodeStorage, commandLines, mountCommand, blockDevice, index, log)
		if err != nil {
			return nil, nil, dwsv1alpha2.NewResourceError("could not create lustre file system").WithError(err).WithMajor()
		}

		return blockDevice, fileSystem, nil
	default:
		break
	}

	return nil, nil, dwsv1alpha2.NewResourceError("unsupported file system type %s", nnfNodeStorage.Spec.FileSystemType).WithMajor()
}

func newZpoolBlockDevice(ctx context.Context, c client.Client, nnfNodeStorage *nnfv1alpha1.NnfNodeStorage, cmdLines nnfv1alpha1.NnfStorageProfileLustreCmdLines, index int, log logr.Logger) (blockdevice.BlockDevice, error) {
	zpool := blockdevice.Zpool{}

	// This is for the fake NnfNodeStorage case. We don't need to create the zpool BlockDevice
	if nnfNodeStorage.Spec.BlockReference.Kind != reflect.TypeOf(nnfv1alpha1.NnfNodeBlockStorage{}).Name() {
		return newMockBlockDevice(ctx, c, nnfNodeStorage, index, log)
	}

	nnfNodeBlockStorage := &nnfv1alpha1.NnfNodeBlockStorage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nnfNodeStorage.GetName(),
			Namespace: nnfNodeStorage.GetNamespace(),
		},
	}

	if err := c.Get(ctx, client.ObjectKeyFromObject(nnfNodeBlockStorage), nnfNodeBlockStorage); err != nil {
		return nil, dwsv1alpha2.NewResourceError("could not get NnfNodeBlockStorage: %v", client.ObjectKeyFromObject(nnfNodeBlockStorage)).WithError(err).WithUserMessage("could not find storage allocation").WithMajor()
	}

	zpool.Log = log
	zpool.Devices = append([]string{}, nnfNodeBlockStorage.Status.Allocations[index].Accesses[os.Getenv("NNF_NODE_NAME")].DevicePaths...)
	zpool.Name = fmt.Sprintf("%s-%s-%d", nnfNodeStorage.Spec.LustreStorage.FileSystemName, nnfNodeStorage.Spec.LustreStorage.TargetType, index)
	zpool.DataSet = nnfNodeStorage.Spec.LustreStorage.TargetType

	zpool.CommandArgs.Create = cmdLines.ZpoolCreate

	return &zpool, nil
}

func newLvmBlockDevice(ctx context.Context, c client.Client, nnfNodeStorage *nnfv1alpha1.NnfNodeStorage, cmdLines nnfv1alpha1.NnfStorageProfileCmdLines, index int, log logr.Logger) (blockdevice.BlockDevice, error) {
	lvmDesc := blockdevice.Lvm{}
	devices := []string{}

	if nnfNodeStorage.Spec.BlockReference.Kind == reflect.TypeOf(nnfv1alpha1.NnfNodeBlockStorage{}).Name() {
		nnfNodeBlockStorage := &nnfv1alpha1.NnfNodeBlockStorage{
			ObjectMeta: metav1.ObjectMeta{
				Name:      nnfNodeStorage.GetName(),
				Namespace: nnfNodeStorage.GetNamespace(),
			},
		}

		err := c.Get(ctx, client.ObjectKeyFromObject(nnfNodeBlockStorage), nnfNodeBlockStorage)
		if err != nil {
			return nil, dwsv1alpha2.NewResourceError("could not get NnfNodeBlockStorage: %v", client.ObjectKeyFromObject(nnfNodeBlockStorage)).WithError(err).WithUserMessage("could not find storage allocation").WithMajor()
		}

		devices = nnfNodeBlockStorage.Status.Allocations[index].Accesses[os.Getenv("NNF_NODE_NAME")].DevicePaths
	}

	for _, device := range devices {
		pv := lvm.NewPhysicalVolume(ctx, device, log)
		lvmDesc.PhysicalVolumes = append(lvmDesc.PhysicalVolumes, pv)
	}

	vgName, err := volumeGroupName(ctx, c, nnfNodeStorage, index)
	if err != nil {
		return nil, dwsv1alpha2.NewResourceError("could not get volume group name").WithError(err).WithMajor()
	}

	lvName, err := logicalVolumeName(ctx, c, nnfNodeStorage, index)
	if err != nil {
		return nil, dwsv1alpha2.NewResourceError("could not get logical volume name").WithError(err).WithMajor()
	}

	lvmDesc.Log = log
	lvmDesc.VolumeGroup = lvm.NewVolumeGroup(ctx, vgName, lvmDesc.PhysicalVolumes, log)
	lvmDesc.LogicalVolume = lvm.NewLogicalVolume(ctx, lvName, lvmDesc.VolumeGroup, log)

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

func newMockBlockDevice(ctx context.Context, c client.Client, nnfNodeStorage *nnfv1alpha1.NnfNodeStorage, index int, log logr.Logger) (blockdevice.BlockDevice, error) {
	blockDevice := blockdevice.MockBlockDevice{
		Log: log,
	}

	return &blockDevice, nil
}

func newBindFileSystem(ctx context.Context, c client.Client, nnfNodeStorage *nnfv1alpha1.NnfNodeStorage, blockDevice blockdevice.BlockDevice, index int, log logr.Logger) (filesystem.FileSystem, error) {
	fs := filesystem.SimpleFileSystem{}

	fs.Log = log
	fs.BlockDevice = blockDevice
	fs.Type = "none"
	fs.MountTarget = "file"
	fs.TempDir = fmt.Sprintf("/mnt/temp/%s-%d", nnfNodeStorage.Name, index)

	fs.CommandArgs.Mount = "-o bind $DEVICE $MOUNT_PATH"

	return &fs, nil
}

func newGfs2FileSystem(ctx context.Context, c client.Client, nnfNodeStorage *nnfv1alpha1.NnfNodeStorage, cmdLines nnfv1alpha1.NnfStorageProfileCmdLines, blockDevice blockdevice.BlockDevice, index int, log logr.Logger) (filesystem.FileSystem, error) {
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
	fs.CommandArgs.Mkfs = fmt.Sprintf("-O %s", cmdLines.Mkfs)
	fs.CommandArgs.Vars = map[string]string{
		"$CLUSTER_NAME": nnfNodeStorage.Namespace,
		"$LOCK_SPACE":   fmt.Sprintf("fs-%02d-%x", index, nnfNodeStorage.GetUID()[0:5]),
		"$PROTOCOL":     "lock_dlm",
	}

	return &fs, nil
}

func newXfsFileSystem(ctx context.Context, c client.Client, nnfNodeStorage *nnfv1alpha1.NnfNodeStorage, cmdLines nnfv1alpha1.NnfStorageProfileCmdLines, blockDevice blockdevice.BlockDevice, index int, log logr.Logger) (filesystem.FileSystem, error) {
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
	fs.CommandArgs.Mkfs = cmdLines.Mkfs

	return &fs, nil
}

func newLustreFileSystem(ctx context.Context, c client.Client, nnfNodeStorage *nnfv1alpha1.NnfNodeStorage, cmdLines nnfv1alpha1.NnfStorageProfileLustreCmdLines, mountCommand string, blockDevice blockdevice.BlockDevice, index int, log logr.Logger) (filesystem.FileSystem, error) {
	fs := filesystem.LustreFileSystem{}

	fs.Log = log
	fs.BlockDevice = blockDevice
	fs.Name = nnfNodeStorage.Spec.LustreStorage.FileSystemName
	fs.TargetType = nnfNodeStorage.Spec.LustreStorage.TargetType
	fs.MgsAddress = nnfNodeStorage.Spec.LustreStorage.MgsAddress
	fs.Index = nnfNodeStorage.Spec.LustreStorage.StartIndex + index
	fs.BackFs = nnfNodeStorage.Spec.LustreStorage.BackFs

	fs.CommandArgs.Mkfs = cmdLines.Mkfs
	fs.CommandArgs.MountTarget = cmdLines.MountTarget
	fs.CommandArgs.Mount = mountCommand

	return &fs, nil
}

func newMockFileSystem(ctx context.Context, c client.Client, nnfNodeStorage *nnfv1alpha1.NnfNodeStorage, blockDevice blockdevice.BlockDevice, index int, log logr.Logger) (filesystem.FileSystem, error) {
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

func volumeGroupName(ctx context.Context, c client.Client, nnfNodeStorage *nnfv1alpha1.NnfNodeStorage, index int) (string, error) {
	labels := nnfNodeStorage.GetLabels()

	// Use the NnfStorage UID since the NnfStorage exists for as long as the storage allocation exists.
	// This is important for persistent instances
	nnfStorageUid, ok := labels[dwsv1alpha2.OwnerUidLabel]
	if !ok {
		return "", fmt.Errorf("missing Owner UID label on NnfNodeStorage")
	}
	directiveIndex, ok := labels[nnfv1alpha1.DirectiveIndexLabel]
	if !ok {
		return "", fmt.Errorf("missing directive index label on NnfNodeStorage")
	}

	return fmt.Sprintf("%s_%s_%d", nnfStorageUid, directiveIndex, index), nil
}

func logicalVolumeName(ctx context.Context, c client.Client, nnfNodeStorage *nnfv1alpha1.NnfNodeStorage, index int) (string, error) {
	// For now just return "lv" as the lv name. If we end up sharing a volume group for multiple lvs, then
	// this name needs to be something unique
	return "lv", nil
}
