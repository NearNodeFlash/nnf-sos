/*
 * Copyright 2025 Hewlett Packard Enterprise Development LP
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

package v1alpha8

import (
	unsafe "unsafe"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apiconversion "k8s.io/apimachinery/pkg/conversion"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	dwsv1alpha6 "github.com/DataWorkflowServices/dws/api/v1alpha6"
	dwsv1alpha7 "github.com/DataWorkflowServices/dws/api/v1alpha7"
	nnfv1alpha10 "github.com/NearNodeFlash/nnf-sos/api/v1alpha10"
	utilconversion "github.com/NearNodeFlash/nnf-sos/github/cluster-api/util/conversion"
)

var convertlog = logf.Log.V(2).WithName("convert-v1alpha8")

func (src *NnfAccess) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfAccess To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha10.NnfAccess)

	if err := Convert_v1alpha8_NnfAccess_To_v1alpha10_NnfAccess(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha10.NnfAccess{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	return nil
}

func (dst *NnfAccess) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha10.NnfAccess)
	convertlog.Info("Convert NnfAccess From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha10_NnfAccess_To_v1alpha8_NnfAccess(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfContainerProfile) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfContainerProfile To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha10.NnfContainerProfile)

	if err := Convert_v1alpha8_NnfContainerProfile_To_v1alpha10_NnfContainerProfile(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha10.NnfContainerProfile{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	return nil
}

func (dst *NnfContainerProfile) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha10.NnfContainerProfile)
	convertlog.Info("Convert NnfContainerProfile From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha10_NnfContainerProfile_To_v1alpha8_NnfContainerProfile(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfDataMovement) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfDataMovement To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha10.NnfDataMovement)

	if err := Convert_v1alpha8_NnfDataMovement_To_v1alpha10_NnfDataMovement(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha10.NnfDataMovement{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	return nil
}

func (dst *NnfDataMovement) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha10.NnfDataMovement)
	convertlog.Info("Convert NnfDataMovement From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha10_NnfDataMovement_To_v1alpha8_NnfDataMovement(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfDataMovementManager) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfDataMovementManager To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha10.NnfDataMovementManager)

	if err := Convert_v1alpha8_NnfDataMovementManager_To_v1alpha10_NnfDataMovementManager(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha10.NnfDataMovementManager{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	return nil
}

func (dst *NnfDataMovementManager) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha10.NnfDataMovementManager)
	convertlog.Info("Convert NnfDataMovementManager From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha10_NnfDataMovementManager_To_v1alpha8_NnfDataMovementManager(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfDataMovementProfile) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfDataMovementProfile To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha10.NnfDataMovementProfile)

	if err := Convert_v1alpha8_NnfDataMovementProfile_To_v1alpha10_NnfDataMovementProfile(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha10.NnfDataMovementProfile{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	return nil
}

func (dst *NnfDataMovementProfile) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha10.NnfDataMovementProfile)
	convertlog.Info("Convert NnfDataMovementProfile From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha10_NnfDataMovementProfile_To_v1alpha8_NnfDataMovementProfile(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfLustreMGT) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfLustreMGT To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha10.NnfLustreMGT)

	if err := Convert_v1alpha8_NnfLustreMGT_To_v1alpha10_NnfLustreMGT(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha10.NnfLustreMGT{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	return nil
}

func (dst *NnfLustreMGT) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha10.NnfLustreMGT)
	convertlog.Info("Convert NnfLustreMGT From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha10_NnfLustreMGT_To_v1alpha8_NnfLustreMGT(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfNode) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfNode To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha10.NnfNode)

	if err := Convert_v1alpha8_NnfNode_To_v1alpha10_NnfNode(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha10.NnfNode{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	return nil
}

func (dst *NnfNode) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha10.NnfNode)
	convertlog.Info("Convert NnfNode From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha10_NnfNode_To_v1alpha8_NnfNode(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfNodeBlockStorage) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfNodeBlockStorage To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha10.NnfNodeBlockStorage)

	if err := Convert_v1alpha8_NnfNodeBlockStorage_To_v1alpha10_NnfNodeBlockStorage(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha10.NnfNodeBlockStorage{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	return nil
}

func (dst *NnfNodeBlockStorage) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha10.NnfNodeBlockStorage)
	convertlog.Info("Convert NnfNodeBlockStorage From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha10_NnfNodeBlockStorage_To_v1alpha8_NnfNodeBlockStorage(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfNodeECData) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfNodeECData To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha10.NnfNodeECData)

	if err := Convert_v1alpha8_NnfNodeECData_To_v1alpha10_NnfNodeECData(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha10.NnfNodeECData{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	return nil
}

func (dst *NnfNodeECData) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha10.NnfNodeECData)
	convertlog.Info("Convert NnfNodeECData From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha10_NnfNodeECData_To_v1alpha8_NnfNodeECData(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfNodeStorage) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfNodeStorage To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha10.NnfNodeStorage)

	if err := Convert_v1alpha8_NnfNodeStorage_To_v1alpha10_NnfNodeStorage(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha10.NnfNodeStorage{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	return nil
}

func (dst *NnfNodeStorage) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha10.NnfNodeStorage)
	convertlog.Info("Convert NnfNodeStorage From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha10_NnfNodeStorage_To_v1alpha8_NnfNodeStorage(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfPortManager) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfPortManager To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha10.NnfPortManager)

	if err := Convert_v1alpha8_NnfPortManager_To_v1alpha10_NnfPortManager(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha10.NnfPortManager{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	return nil
}

func (dst *NnfPortManager) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha10.NnfPortManager)
	convertlog.Info("Convert NnfPortManager From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha10_NnfPortManager_To_v1alpha8_NnfPortManager(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfStorage) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfStorage To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha10.NnfStorage)

	if err := Convert_v1alpha8_NnfStorage_To_v1alpha10_NnfStorage(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha10.NnfStorage{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	return nil
}

func (dst *NnfStorage) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha10.NnfStorage)
	convertlog.Info("Convert NnfStorage From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha10_NnfStorage_To_v1alpha8_NnfStorage(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfStorageProfile) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfStorageProfile To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha10.NnfStorageProfile)

	if err := Convert_v1alpha8_NnfStorageProfile_To_v1alpha10_NnfStorageProfile(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha10.NnfStorageProfile{}
	hasAnno, err := utilconversion.UnmarshalData(src, restored)
	if err != nil {
		return err
	}

	dst.Data.LustreStorage.ClientCmdLines.RabbitPostSetup = src.Data.LustreStorage.ClientCmdLines.RabbitPostMount
	dst.Data.LustreStorage.ClientCmdLines.RabbitPreTeardown = src.Data.LustreStorage.ClientCmdLines.RabbitPreUnmount

	dst.Data.GFS2Storage.StorageLabels = src.Data.GFS2Storage.StorageLabels
	dst.Data.GFS2Storage.CapacityScalingFactor = src.Data.GFS2Storage.CapacityScalingFactor
	dst.Data.GFS2Storage.FileSystemCommands.RabbitCommands.Mkfs = src.Data.GFS2Storage.CmdLines.Mkfs
	dst.Data.GFS2Storage.FileSystemCommands.RabbitCommands.Mount = src.Data.GFS2Storage.CmdLines.MountRabbit
	dst.Data.GFS2Storage.UserCommands.PostSetup = src.Data.GFS2Storage.CmdLines.PostMount
	dst.Data.GFS2Storage.UserCommands.PreTeardown = src.Data.GFS2Storage.CmdLines.PreUnmount
	dst.Data.GFS2Storage.FileSystemCommands.ComputeCommands.Mount = src.Data.GFS2Storage.CmdLines.MountCompute
	dst.Data.GFS2Storage.BlockDeviceCommands.SharedVg = src.Data.GFS2Storage.CmdLines.SharedVg
	dst.Data.GFS2Storage.BlockDeviceCommands.RabbitCommands.PvCreate = src.Data.GFS2Storage.CmdLines.PvCreate
	dst.Data.GFS2Storage.BlockDeviceCommands.RabbitCommands.PvRemove = src.Data.GFS2Storage.CmdLines.PvRemove
	dst.Data.GFS2Storage.BlockDeviceCommands.RabbitCommands.VgCreate = src.Data.GFS2Storage.CmdLines.VgCreate
	dst.Data.GFS2Storage.BlockDeviceCommands.RabbitCommands.VgRemove = src.Data.GFS2Storage.CmdLines.VgRemove
	dst.Data.GFS2Storage.BlockDeviceCommands.RabbitCommands.VgChange.LockStart = src.Data.GFS2Storage.CmdLines.VgChange.LockStart
	dst.Data.GFS2Storage.BlockDeviceCommands.RabbitCommands.VgChange.LockStop = src.Data.GFS2Storage.CmdLines.VgChange.LockStop
	dst.Data.GFS2Storage.BlockDeviceCommands.RabbitCommands.LvCreate = src.Data.GFS2Storage.CmdLines.LvCreate
	dst.Data.GFS2Storage.BlockDeviceCommands.RabbitCommands.LvRemove = src.Data.GFS2Storage.CmdLines.LvRemove
	dst.Data.GFS2Storage.BlockDeviceCommands.RabbitCommands.LvChange.Activate = src.Data.GFS2Storage.CmdLines.LvChange.Activate
	dst.Data.GFS2Storage.BlockDeviceCommands.RabbitCommands.LvChange.Deactivate = src.Data.GFS2Storage.CmdLines.LvChange.Deactivate
	dst.Data.GFS2Storage.BlockDeviceCommands.RabbitCommands.LVMRebuild.VgExtend = src.Data.GFS2Storage.CmdLines.LVMRebuild.VgExtend
	dst.Data.GFS2Storage.BlockDeviceCommands.RabbitCommands.LVMRebuild.VgReduce = src.Data.GFS2Storage.CmdLines.LVMRebuild.VgReduce
	dst.Data.GFS2Storage.BlockDeviceCommands.RabbitCommands.LVMRebuild.LvRepair = src.Data.GFS2Storage.CmdLines.LVMRebuild.LvRepair

	dst.Data.XFSStorage.StorageLabels = src.Data.XFSStorage.StorageLabels
	dst.Data.XFSStorage.CapacityScalingFactor = src.Data.XFSStorage.CapacityScalingFactor
	dst.Data.XFSStorage.FileSystemCommands.RabbitCommands.Mkfs = src.Data.XFSStorage.CmdLines.Mkfs
	dst.Data.XFSStorage.FileSystemCommands.RabbitCommands.Mount = src.Data.XFSStorage.CmdLines.MountRabbit
	dst.Data.XFSStorage.UserCommands.PostSetup = src.Data.XFSStorage.CmdLines.PostMount
	dst.Data.XFSStorage.UserCommands.PreTeardown = src.Data.XFSStorage.CmdLines.PreUnmount
	dst.Data.XFSStorage.FileSystemCommands.ComputeCommands.Mount = src.Data.XFSStorage.CmdLines.MountCompute
	dst.Data.XFSStorage.BlockDeviceCommands.SharedVg = src.Data.XFSStorage.CmdLines.SharedVg
	dst.Data.XFSStorage.BlockDeviceCommands.RabbitCommands.PvCreate = src.Data.XFSStorage.CmdLines.PvCreate
	dst.Data.XFSStorage.BlockDeviceCommands.RabbitCommands.PvRemove = src.Data.XFSStorage.CmdLines.PvRemove
	dst.Data.XFSStorage.BlockDeviceCommands.RabbitCommands.VgCreate = src.Data.XFSStorage.CmdLines.VgCreate
	dst.Data.XFSStorage.BlockDeviceCommands.RabbitCommands.VgRemove = src.Data.XFSStorage.CmdLines.VgRemove
	dst.Data.XFSStorage.BlockDeviceCommands.RabbitCommands.VgChange.LockStart = src.Data.XFSStorage.CmdLines.VgChange.LockStart
	dst.Data.XFSStorage.BlockDeviceCommands.RabbitCommands.VgChange.LockStop = src.Data.XFSStorage.CmdLines.VgChange.LockStop
	dst.Data.XFSStorage.BlockDeviceCommands.RabbitCommands.LvCreate = src.Data.XFSStorage.CmdLines.LvCreate
	dst.Data.XFSStorage.BlockDeviceCommands.RabbitCommands.LvRemove = src.Data.XFSStorage.CmdLines.LvRemove
	dst.Data.XFSStorage.BlockDeviceCommands.RabbitCommands.LvChange.Activate = src.Data.XFSStorage.CmdLines.LvChange.Activate
	dst.Data.XFSStorage.BlockDeviceCommands.RabbitCommands.LvChange.Deactivate = src.Data.XFSStorage.CmdLines.LvChange.Deactivate
	dst.Data.XFSStorage.BlockDeviceCommands.RabbitCommands.LVMRebuild.VgExtend = src.Data.XFSStorage.CmdLines.LVMRebuild.VgExtend
	dst.Data.XFSStorage.BlockDeviceCommands.RabbitCommands.LVMRebuild.VgReduce = src.Data.XFSStorage.CmdLines.LVMRebuild.VgReduce
	dst.Data.XFSStorage.BlockDeviceCommands.RabbitCommands.LVMRebuild.LvRepair = src.Data.XFSStorage.CmdLines.LVMRebuild.LvRepair

	dst.Data.RawStorage.StorageLabels = src.Data.RawStorage.StorageLabels
	dst.Data.RawStorage.CapacityScalingFactor = src.Data.RawStorage.CapacityScalingFactor
	dst.Data.RawStorage.FileSystemCommands.RabbitCommands.Mkfs = src.Data.RawStorage.CmdLines.Mkfs
	dst.Data.RawStorage.FileSystemCommands.RabbitCommands.Mount = src.Data.RawStorage.CmdLines.MountRabbit
	dst.Data.RawStorage.UserCommands.PostSetup = src.Data.RawStorage.CmdLines.PostMount
	dst.Data.RawStorage.UserCommands.PreTeardown = src.Data.RawStorage.CmdLines.PreUnmount
	dst.Data.RawStorage.FileSystemCommands.ComputeCommands.Mount = src.Data.RawStorage.CmdLines.MountCompute
	dst.Data.RawStorage.BlockDeviceCommands.SharedVg = src.Data.RawStorage.CmdLines.SharedVg
	dst.Data.RawStorage.BlockDeviceCommands.RabbitCommands.PvCreate = src.Data.RawStorage.CmdLines.PvCreate
	dst.Data.RawStorage.BlockDeviceCommands.RabbitCommands.PvRemove = src.Data.RawStorage.CmdLines.PvRemove
	dst.Data.RawStorage.BlockDeviceCommands.RabbitCommands.VgCreate = src.Data.RawStorage.CmdLines.VgCreate
	dst.Data.RawStorage.BlockDeviceCommands.RabbitCommands.VgRemove = src.Data.RawStorage.CmdLines.VgRemove
	dst.Data.RawStorage.BlockDeviceCommands.RabbitCommands.VgChange.LockStart = src.Data.RawStorage.CmdLines.VgChange.LockStart
	dst.Data.RawStorage.BlockDeviceCommands.RabbitCommands.VgChange.LockStop = src.Data.RawStorage.CmdLines.VgChange.LockStop
	dst.Data.RawStorage.BlockDeviceCommands.RabbitCommands.LvCreate = src.Data.RawStorage.CmdLines.LvCreate
	dst.Data.RawStorage.BlockDeviceCommands.RabbitCommands.LvRemove = src.Data.RawStorage.CmdLines.LvRemove
	dst.Data.RawStorage.BlockDeviceCommands.RabbitCommands.LvChange.Activate = src.Data.RawStorage.CmdLines.LvChange.Activate
	dst.Data.RawStorage.BlockDeviceCommands.RabbitCommands.LvChange.Deactivate = src.Data.RawStorage.CmdLines.LvChange.Deactivate
	dst.Data.RawStorage.BlockDeviceCommands.RabbitCommands.LVMRebuild.VgExtend = src.Data.RawStorage.CmdLines.LVMRebuild.VgExtend
	dst.Data.RawStorage.BlockDeviceCommands.RabbitCommands.LVMRebuild.VgReduce = src.Data.RawStorage.CmdLines.LVMRebuild.VgReduce
	dst.Data.RawStorage.BlockDeviceCommands.RabbitCommands.LVMRebuild.LvRepair = src.Data.RawStorage.CmdLines.LVMRebuild.LvRepair

	if hasAnno {
		dst.Data.LustreStorage.ClientCmdLines.RabbitPostSetup = restored.Data.LustreStorage.ClientCmdLines.RabbitPostSetup
		dst.Data.LustreStorage.ClientCmdLines.RabbitPreTeardown = restored.Data.LustreStorage.ClientCmdLines.RabbitPreTeardown
		dst.Data.LustreStorage.ClientCmdLines.RabbitPreMount = restored.Data.LustreStorage.ClientCmdLines.RabbitPreMount
		dst.Data.LustreStorage.ClientCmdLines.RabbitPostMount = restored.Data.LustreStorage.ClientCmdLines.RabbitPostMount
		dst.Data.LustreStorage.ClientCmdLines.RabbitPreUnmount = restored.Data.LustreStorage.ClientCmdLines.RabbitPreUnmount
		dst.Data.LustreStorage.ClientCmdLines.RabbitPostUnmount = restored.Data.LustreStorage.ClientCmdLines.RabbitPostUnmount
		dst.Data.LustreStorage.ClientCmdLines.ComputePreMount = restored.Data.LustreStorage.ClientCmdLines.ComputePreMount
		dst.Data.LustreStorage.ClientCmdLines.ComputePostMount = restored.Data.LustreStorage.ClientCmdLines.ComputePostMount
		dst.Data.LustreStorage.ClientCmdLines.ComputePreUnmount = restored.Data.LustreStorage.ClientCmdLines.ComputePreUnmount
		dst.Data.LustreStorage.ClientCmdLines.ComputePostUnmount = restored.Data.LustreStorage.ClientCmdLines.ComputePostUnmount

		dst.Data.GFS2Storage.SharedAllocation = restored.Data.GFS2Storage.SharedAllocation
		dst.Data.GFS2Storage.AllocationPadding = restored.Data.GFS2Storage.AllocationPadding
		dst.Data.GFS2Storage.UserCommands.PostActivate = restored.Data.GFS2Storage.UserCommands.PostActivate
		dst.Data.GFS2Storage.UserCommands.PreDeactivate = restored.Data.GFS2Storage.UserCommands.PreDeactivate
		dst.Data.GFS2Storage.FileSystemCommands.RabbitCommands.UserCommands.PreMount = restored.Data.GFS2Storage.FileSystemCommands.RabbitCommands.UserCommands.PreMount
		dst.Data.GFS2Storage.FileSystemCommands.RabbitCommands.UserCommands.PostMount = restored.Data.GFS2Storage.FileSystemCommands.RabbitCommands.UserCommands.PostMount
		dst.Data.GFS2Storage.FileSystemCommands.RabbitCommands.UserCommands.PreUnmount = restored.Data.GFS2Storage.FileSystemCommands.RabbitCommands.UserCommands.PreUnmount
		dst.Data.GFS2Storage.FileSystemCommands.RabbitCommands.UserCommands.PostUnmount = restored.Data.GFS2Storage.FileSystemCommands.RabbitCommands.UserCommands.PostUnmount
		dst.Data.GFS2Storage.FileSystemCommands.ComputeCommands.UserCommands.PreMount = restored.Data.GFS2Storage.FileSystemCommands.ComputeCommands.UserCommands.PreMount
		dst.Data.GFS2Storage.FileSystemCommands.ComputeCommands.UserCommands.PostMount = restored.Data.GFS2Storage.FileSystemCommands.ComputeCommands.UserCommands.PostMount
		dst.Data.GFS2Storage.FileSystemCommands.ComputeCommands.UserCommands.PreUnmount = restored.Data.GFS2Storage.FileSystemCommands.ComputeCommands.UserCommands.PreUnmount
		dst.Data.GFS2Storage.FileSystemCommands.ComputeCommands.UserCommands.PostUnmount = restored.Data.GFS2Storage.FileSystemCommands.ComputeCommands.UserCommands.PostUnmount
		dst.Data.GFS2Storage.BlockDeviceCommands.RabbitCommands.UserCommands.PreActivate = restored.Data.GFS2Storage.BlockDeviceCommands.RabbitCommands.UserCommands.PreActivate
		dst.Data.GFS2Storage.BlockDeviceCommands.RabbitCommands.UserCommands.PostActivate = restored.Data.GFS2Storage.BlockDeviceCommands.RabbitCommands.UserCommands.PostActivate
		dst.Data.GFS2Storage.BlockDeviceCommands.RabbitCommands.UserCommands.PreDeactivate = restored.Data.GFS2Storage.BlockDeviceCommands.RabbitCommands.UserCommands.PreDeactivate
		dst.Data.GFS2Storage.BlockDeviceCommands.RabbitCommands.UserCommands.PostDeactivate = restored.Data.GFS2Storage.BlockDeviceCommands.RabbitCommands.UserCommands.PostDeactivate
		dst.Data.GFS2Storage.BlockDeviceCommands.ComputeCommands.LvChange.Activate = restored.Data.GFS2Storage.BlockDeviceCommands.ComputeCommands.LvChange.Activate
		dst.Data.GFS2Storage.BlockDeviceCommands.ComputeCommands.LvChange.Deactivate = restored.Data.GFS2Storage.BlockDeviceCommands.ComputeCommands.LvChange.Deactivate
		dst.Data.GFS2Storage.BlockDeviceCommands.ComputeCommands.VgChange.LockStart = restored.Data.GFS2Storage.BlockDeviceCommands.ComputeCommands.VgChange.LockStart
		dst.Data.GFS2Storage.BlockDeviceCommands.ComputeCommands.VgChange.LockStop = restored.Data.GFS2Storage.BlockDeviceCommands.ComputeCommands.VgChange.LockStop
		dst.Data.GFS2Storage.BlockDeviceCommands.ComputeCommands.UserCommands.PreActivate = restored.Data.GFS2Storage.BlockDeviceCommands.ComputeCommands.UserCommands.PreActivate
		dst.Data.GFS2Storage.BlockDeviceCommands.ComputeCommands.UserCommands.PostActivate = restored.Data.GFS2Storage.BlockDeviceCommands.ComputeCommands.UserCommands.PostActivate
		dst.Data.GFS2Storage.BlockDeviceCommands.ComputeCommands.UserCommands.PreDeactivate = restored.Data.GFS2Storage.BlockDeviceCommands.ComputeCommands.UserCommands.PreDeactivate
		dst.Data.GFS2Storage.BlockDeviceCommands.ComputeCommands.UserCommands.PostDeactivate = restored.Data.GFS2Storage.BlockDeviceCommands.ComputeCommands.UserCommands.PostDeactivate

		dst.Data.XFSStorage.SharedAllocation = restored.Data.XFSStorage.SharedAllocation
		dst.Data.XFSStorage.AllocationPadding = restored.Data.XFSStorage.AllocationPadding
		dst.Data.XFSStorage.UserCommands.PostActivate = restored.Data.XFSStorage.UserCommands.PostActivate
		dst.Data.XFSStorage.UserCommands.PreDeactivate = restored.Data.XFSStorage.UserCommands.PreDeactivate
		dst.Data.XFSStorage.FileSystemCommands.RabbitCommands.UserCommands.PreMount = restored.Data.XFSStorage.FileSystemCommands.RabbitCommands.UserCommands.PreMount
		dst.Data.XFSStorage.FileSystemCommands.RabbitCommands.UserCommands.PostMount = restored.Data.XFSStorage.FileSystemCommands.RabbitCommands.UserCommands.PostMount
		dst.Data.XFSStorage.FileSystemCommands.RabbitCommands.UserCommands.PreUnmount = restored.Data.XFSStorage.FileSystemCommands.RabbitCommands.UserCommands.PreUnmount
		dst.Data.XFSStorage.FileSystemCommands.RabbitCommands.UserCommands.PostUnmount = restored.Data.XFSStorage.FileSystemCommands.RabbitCommands.UserCommands.PostUnmount
		dst.Data.XFSStorage.FileSystemCommands.ComputeCommands.UserCommands.PreMount = restored.Data.XFSStorage.FileSystemCommands.ComputeCommands.UserCommands.PreMount
		dst.Data.XFSStorage.FileSystemCommands.ComputeCommands.UserCommands.PostMount = restored.Data.XFSStorage.FileSystemCommands.ComputeCommands.UserCommands.PostMount
		dst.Data.XFSStorage.FileSystemCommands.ComputeCommands.UserCommands.PreUnmount = restored.Data.XFSStorage.FileSystemCommands.ComputeCommands.UserCommands.PreUnmount
		dst.Data.XFSStorage.FileSystemCommands.ComputeCommands.UserCommands.PostUnmount = restored.Data.XFSStorage.FileSystemCommands.ComputeCommands.UserCommands.PostUnmount
		dst.Data.XFSStorage.BlockDeviceCommands.RabbitCommands.UserCommands.PreActivate = restored.Data.XFSStorage.BlockDeviceCommands.RabbitCommands.UserCommands.PreActivate
		dst.Data.XFSStorage.BlockDeviceCommands.RabbitCommands.UserCommands.PostActivate = restored.Data.XFSStorage.BlockDeviceCommands.RabbitCommands.UserCommands.PostActivate
		dst.Data.XFSStorage.BlockDeviceCommands.RabbitCommands.UserCommands.PreDeactivate = restored.Data.XFSStorage.BlockDeviceCommands.RabbitCommands.UserCommands.PreDeactivate
		dst.Data.XFSStorage.BlockDeviceCommands.RabbitCommands.UserCommands.PostDeactivate = restored.Data.XFSStorage.BlockDeviceCommands.RabbitCommands.UserCommands.PostDeactivate
		dst.Data.XFSStorage.BlockDeviceCommands.ComputeCommands.LvChange.Activate = restored.Data.XFSStorage.BlockDeviceCommands.ComputeCommands.LvChange.Activate
		dst.Data.XFSStorage.BlockDeviceCommands.ComputeCommands.LvChange.Deactivate = restored.Data.XFSStorage.BlockDeviceCommands.ComputeCommands.LvChange.Deactivate
		dst.Data.XFSStorage.BlockDeviceCommands.ComputeCommands.VgChange.LockStart = restored.Data.XFSStorage.BlockDeviceCommands.ComputeCommands.VgChange.LockStart
		dst.Data.XFSStorage.BlockDeviceCommands.ComputeCommands.VgChange.LockStop = restored.Data.XFSStorage.BlockDeviceCommands.ComputeCommands.VgChange.LockStop
		dst.Data.XFSStorage.BlockDeviceCommands.ComputeCommands.UserCommands.PreActivate = restored.Data.XFSStorage.BlockDeviceCommands.ComputeCommands.UserCommands.PreActivate
		dst.Data.XFSStorage.BlockDeviceCommands.ComputeCommands.UserCommands.PostActivate = restored.Data.XFSStorage.BlockDeviceCommands.ComputeCommands.UserCommands.PostActivate
		dst.Data.XFSStorage.BlockDeviceCommands.ComputeCommands.UserCommands.PreDeactivate = restored.Data.XFSStorage.BlockDeviceCommands.ComputeCommands.UserCommands.PreDeactivate
		dst.Data.XFSStorage.BlockDeviceCommands.ComputeCommands.UserCommands.PostDeactivate = restored.Data.XFSStorage.BlockDeviceCommands.ComputeCommands.UserCommands.PostDeactivate

		dst.Data.RawStorage.SharedAllocation = restored.Data.RawStorage.SharedAllocation
		dst.Data.RawStorage.AllocationPadding = restored.Data.RawStorage.AllocationPadding
		dst.Data.RawStorage.UserCommands.PostActivate = restored.Data.RawStorage.UserCommands.PostActivate
		dst.Data.RawStorage.UserCommands.PreDeactivate = restored.Data.RawStorage.UserCommands.PreDeactivate
		dst.Data.RawStorage.FileSystemCommands.RabbitCommands.UserCommands.PreMount = restored.Data.RawStorage.FileSystemCommands.RabbitCommands.UserCommands.PreMount
		dst.Data.RawStorage.FileSystemCommands.RabbitCommands.UserCommands.PostMount = restored.Data.RawStorage.FileSystemCommands.RabbitCommands.UserCommands.PostMount
		dst.Data.RawStorage.FileSystemCommands.RabbitCommands.UserCommands.PreUnmount = restored.Data.RawStorage.FileSystemCommands.RabbitCommands.UserCommands.PreUnmount
		dst.Data.RawStorage.FileSystemCommands.RabbitCommands.UserCommands.PostUnmount = restored.Data.RawStorage.FileSystemCommands.RabbitCommands.UserCommands.PostUnmount
		dst.Data.RawStorage.FileSystemCommands.ComputeCommands.UserCommands.PreMount = restored.Data.RawStorage.FileSystemCommands.ComputeCommands.UserCommands.PreMount
		dst.Data.RawStorage.FileSystemCommands.ComputeCommands.UserCommands.PostMount = restored.Data.RawStorage.FileSystemCommands.ComputeCommands.UserCommands.PostMount
		dst.Data.RawStorage.FileSystemCommands.ComputeCommands.UserCommands.PreUnmount = restored.Data.RawStorage.FileSystemCommands.ComputeCommands.UserCommands.PreUnmount
		dst.Data.RawStorage.FileSystemCommands.ComputeCommands.UserCommands.PostUnmount = restored.Data.RawStorage.FileSystemCommands.ComputeCommands.UserCommands.PostUnmount
		dst.Data.RawStorage.BlockDeviceCommands.RabbitCommands.UserCommands.PreActivate = restored.Data.RawStorage.BlockDeviceCommands.RabbitCommands.UserCommands.PreActivate
		dst.Data.RawStorage.BlockDeviceCommands.RabbitCommands.UserCommands.PostActivate = restored.Data.RawStorage.BlockDeviceCommands.RabbitCommands.UserCommands.PostActivate
		dst.Data.RawStorage.BlockDeviceCommands.RabbitCommands.UserCommands.PreDeactivate = restored.Data.RawStorage.BlockDeviceCommands.RabbitCommands.UserCommands.PreDeactivate
		dst.Data.RawStorage.BlockDeviceCommands.RabbitCommands.UserCommands.PostDeactivate = restored.Data.RawStorage.BlockDeviceCommands.RabbitCommands.UserCommands.PostDeactivate
		dst.Data.RawStorage.BlockDeviceCommands.ComputeCommands.LvChange.Activate = restored.Data.RawStorage.BlockDeviceCommands.ComputeCommands.LvChange.Activate
		dst.Data.RawStorage.BlockDeviceCommands.ComputeCommands.LvChange.Deactivate = restored.Data.RawStorage.BlockDeviceCommands.ComputeCommands.LvChange.Deactivate
		dst.Data.RawStorage.BlockDeviceCommands.ComputeCommands.VgChange.LockStart = restored.Data.RawStorage.BlockDeviceCommands.ComputeCommands.VgChange.LockStart
		dst.Data.RawStorage.BlockDeviceCommands.ComputeCommands.VgChange.LockStop = restored.Data.RawStorage.BlockDeviceCommands.ComputeCommands.VgChange.LockStop
		dst.Data.RawStorage.BlockDeviceCommands.ComputeCommands.UserCommands.PreActivate = restored.Data.RawStorage.BlockDeviceCommands.ComputeCommands.UserCommands.PreActivate
		dst.Data.RawStorage.BlockDeviceCommands.ComputeCommands.UserCommands.PostActivate = restored.Data.RawStorage.BlockDeviceCommands.ComputeCommands.UserCommands.PostActivate
		dst.Data.RawStorage.BlockDeviceCommands.ComputeCommands.UserCommands.PreDeactivate = restored.Data.RawStorage.BlockDeviceCommands.ComputeCommands.UserCommands.PreDeactivate
		dst.Data.RawStorage.BlockDeviceCommands.ComputeCommands.UserCommands.PostDeactivate = restored.Data.RawStorage.BlockDeviceCommands.ComputeCommands.UserCommands.PostDeactivate
	} else {
		dst.Data.LustreStorage.ClientCmdLines.RabbitPostMount = []string{}

		dst.Data.GFS2Storage.BlockDeviceCommands.ComputeCommands.LvChange.Activate = src.Data.GFS2Storage.CmdLines.LvChange.Activate
		dst.Data.GFS2Storage.BlockDeviceCommands.ComputeCommands.LvChange.Deactivate = src.Data.GFS2Storage.CmdLines.LvChange.Deactivate
		dst.Data.GFS2Storage.BlockDeviceCommands.ComputeCommands.VgChange.LockStart = src.Data.GFS2Storage.CmdLines.VgChange.LockStart
		dst.Data.GFS2Storage.BlockDeviceCommands.ComputeCommands.VgChange.LockStop = src.Data.GFS2Storage.CmdLines.VgChange.LockStop

		dst.Data.XFSStorage.BlockDeviceCommands.ComputeCommands.LvChange.Activate = src.Data.XFSStorage.CmdLines.LvChange.Activate
		dst.Data.XFSStorage.BlockDeviceCommands.ComputeCommands.LvChange.Deactivate = src.Data.XFSStorage.CmdLines.LvChange.Deactivate
		dst.Data.XFSStorage.BlockDeviceCommands.ComputeCommands.VgChange.LockStart = src.Data.XFSStorage.CmdLines.VgChange.LockStart
		dst.Data.XFSStorage.BlockDeviceCommands.ComputeCommands.VgChange.LockStop = src.Data.XFSStorage.CmdLines.VgChange.LockStop

		dst.Data.RawStorage.BlockDeviceCommands.ComputeCommands.LvChange.Activate = src.Data.RawStorage.CmdLines.LvChange.Activate
		dst.Data.RawStorage.BlockDeviceCommands.ComputeCommands.LvChange.Deactivate = src.Data.RawStorage.CmdLines.LvChange.Deactivate
		dst.Data.RawStorage.BlockDeviceCommands.ComputeCommands.VgChange.LockStart = src.Data.RawStorage.CmdLines.VgChange.LockStart
		dst.Data.RawStorage.BlockDeviceCommands.ComputeCommands.VgChange.LockStop = src.Data.RawStorage.CmdLines.VgChange.LockStop
	}

	return nil
}

func (dst *NnfStorageProfile) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha10.NnfStorageProfile)
	convertlog.Info("Convert NnfStorageProfile From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha10_NnfStorageProfile_To_v1alpha8_NnfStorageProfile(src, dst, nil); err != nil {
		return err
	}

	dst.Data.LustreStorage.ClientCmdLines.RabbitPostMount = src.Data.LustreStorage.ClientCmdLines.RabbitPostSetup
	dst.Data.LustreStorage.ClientCmdLines.RabbitPreUnmount = src.Data.LustreStorage.ClientCmdLines.RabbitPreTeardown

	dst.Data.GFS2Storage.StorageLabels = src.Data.GFS2Storage.StorageLabels
	dst.Data.GFS2Storage.CapacityScalingFactor = src.Data.GFS2Storage.CapacityScalingFactor
	dst.Data.GFS2Storage.CmdLines.Mkfs = src.Data.GFS2Storage.FileSystemCommands.RabbitCommands.Mkfs
	dst.Data.GFS2Storage.CmdLines.MountRabbit = src.Data.GFS2Storage.FileSystemCommands.RabbitCommands.Mount
	dst.Data.GFS2Storage.CmdLines.PostMount = src.Data.GFS2Storage.UserCommands.PostSetup
	dst.Data.GFS2Storage.CmdLines.PreUnmount = src.Data.GFS2Storage.UserCommands.PreTeardown
	dst.Data.GFS2Storage.CmdLines.MountCompute = src.Data.GFS2Storage.FileSystemCommands.ComputeCommands.Mount
	dst.Data.GFS2Storage.CmdLines.SharedVg = src.Data.GFS2Storage.BlockDeviceCommands.SharedVg
	dst.Data.GFS2Storage.CmdLines.PvCreate = src.Data.GFS2Storage.BlockDeviceCommands.RabbitCommands.PvCreate
	dst.Data.GFS2Storage.CmdLines.PvRemove = src.Data.GFS2Storage.BlockDeviceCommands.RabbitCommands.PvRemove
	dst.Data.GFS2Storage.CmdLines.VgCreate = src.Data.GFS2Storage.BlockDeviceCommands.RabbitCommands.VgCreate
	dst.Data.GFS2Storage.CmdLines.VgRemove = src.Data.GFS2Storage.BlockDeviceCommands.RabbitCommands.VgRemove
	dst.Data.GFS2Storage.CmdLines.VgChange.LockStart = src.Data.GFS2Storage.BlockDeviceCommands.RabbitCommands.VgChange.LockStart
	dst.Data.GFS2Storage.CmdLines.VgChange.LockStop = src.Data.GFS2Storage.BlockDeviceCommands.RabbitCommands.VgChange.LockStop
	dst.Data.GFS2Storage.CmdLines.LvCreate = src.Data.GFS2Storage.BlockDeviceCommands.RabbitCommands.LvCreate
	dst.Data.GFS2Storage.CmdLines.LvRemove = src.Data.GFS2Storage.BlockDeviceCommands.RabbitCommands.LvRemove
	dst.Data.GFS2Storage.CmdLines.LvChange.Activate = src.Data.GFS2Storage.BlockDeviceCommands.RabbitCommands.LvChange.Activate
	dst.Data.GFS2Storage.CmdLines.LvChange.Deactivate = src.Data.GFS2Storage.BlockDeviceCommands.RabbitCommands.LvChange.Deactivate
	dst.Data.GFS2Storage.CmdLines.LVMRebuild.VgExtend = src.Data.GFS2Storage.BlockDeviceCommands.RabbitCommands.LVMRebuild.VgExtend
	dst.Data.GFS2Storage.CmdLines.LVMRebuild.VgReduce = src.Data.GFS2Storage.BlockDeviceCommands.RabbitCommands.LVMRebuild.VgReduce
	dst.Data.GFS2Storage.CmdLines.LVMRebuild.LvRepair = src.Data.GFS2Storage.BlockDeviceCommands.RabbitCommands.LVMRebuild.LvRepair

	dst.Data.XFSStorage.StorageLabels = src.Data.XFSStorage.StorageLabels
	dst.Data.XFSStorage.CapacityScalingFactor = src.Data.XFSStorage.CapacityScalingFactor
	dst.Data.XFSStorage.CmdLines.Mkfs = src.Data.XFSStorage.FileSystemCommands.RabbitCommands.Mkfs
	dst.Data.XFSStorage.CmdLines.MountRabbit = src.Data.XFSStorage.FileSystemCommands.RabbitCommands.Mount
	dst.Data.XFSStorage.CmdLines.PostMount = src.Data.XFSStorage.UserCommands.PostSetup
	dst.Data.XFSStorage.CmdLines.PreUnmount = src.Data.XFSStorage.UserCommands.PreTeardown
	dst.Data.XFSStorage.CmdLines.MountCompute = src.Data.XFSStorage.FileSystemCommands.ComputeCommands.Mount
	dst.Data.XFSStorage.CmdLines.SharedVg = src.Data.XFSStorage.BlockDeviceCommands.SharedVg
	dst.Data.XFSStorage.CmdLines.PvCreate = src.Data.XFSStorage.BlockDeviceCommands.RabbitCommands.PvCreate
	dst.Data.XFSStorage.CmdLines.PvRemove = src.Data.XFSStorage.BlockDeviceCommands.RabbitCommands.PvRemove
	dst.Data.XFSStorage.CmdLines.VgCreate = src.Data.XFSStorage.BlockDeviceCommands.RabbitCommands.VgCreate
	dst.Data.XFSStorage.CmdLines.VgRemove = src.Data.XFSStorage.BlockDeviceCommands.RabbitCommands.VgRemove
	dst.Data.XFSStorage.CmdLines.VgChange.LockStart = src.Data.XFSStorage.BlockDeviceCommands.RabbitCommands.VgChange.LockStart
	dst.Data.XFSStorage.CmdLines.VgChange.LockStop = src.Data.XFSStorage.BlockDeviceCommands.RabbitCommands.VgChange.LockStop
	dst.Data.XFSStorage.CmdLines.LvCreate = src.Data.XFSStorage.BlockDeviceCommands.RabbitCommands.LvCreate
	dst.Data.XFSStorage.CmdLines.LvRemove = src.Data.XFSStorage.BlockDeviceCommands.RabbitCommands.LvRemove
	dst.Data.XFSStorage.CmdLines.LvChange.Activate = src.Data.XFSStorage.BlockDeviceCommands.RabbitCommands.LvChange.Activate
	dst.Data.XFSStorage.CmdLines.LvChange.Deactivate = src.Data.XFSStorage.BlockDeviceCommands.RabbitCommands.LvChange.Deactivate
	dst.Data.XFSStorage.CmdLines.LVMRebuild.VgExtend = src.Data.XFSStorage.BlockDeviceCommands.RabbitCommands.LVMRebuild.VgExtend
	dst.Data.XFSStorage.CmdLines.LVMRebuild.VgReduce = src.Data.XFSStorage.BlockDeviceCommands.RabbitCommands.LVMRebuild.VgReduce
	dst.Data.XFSStorage.CmdLines.LVMRebuild.LvRepair = src.Data.XFSStorage.BlockDeviceCommands.RabbitCommands.LVMRebuild.LvRepair

	dst.Data.RawStorage.StorageLabels = src.Data.RawStorage.StorageLabels
	dst.Data.RawStorage.CapacityScalingFactor = src.Data.RawStorage.CapacityScalingFactor
	dst.Data.RawStorage.CmdLines.Mkfs = src.Data.RawStorage.FileSystemCommands.RabbitCommands.Mkfs
	dst.Data.RawStorage.CmdLines.MountRabbit = src.Data.RawStorage.FileSystemCommands.RabbitCommands.Mount
	dst.Data.RawStorage.CmdLines.PostMount = src.Data.RawStorage.UserCommands.PostSetup
	dst.Data.RawStorage.CmdLines.PreUnmount = src.Data.RawStorage.UserCommands.PreTeardown
	dst.Data.RawStorage.CmdLines.MountCompute = src.Data.RawStorage.FileSystemCommands.ComputeCommands.Mount
	dst.Data.RawStorage.CmdLines.SharedVg = src.Data.RawStorage.BlockDeviceCommands.SharedVg
	dst.Data.RawStorage.CmdLines.PvCreate = src.Data.RawStorage.BlockDeviceCommands.RabbitCommands.PvCreate
	dst.Data.RawStorage.CmdLines.PvRemove = src.Data.RawStorage.BlockDeviceCommands.RabbitCommands.PvRemove
	dst.Data.RawStorage.CmdLines.VgCreate = src.Data.RawStorage.BlockDeviceCommands.RabbitCommands.VgCreate
	dst.Data.RawStorage.CmdLines.VgRemove = src.Data.RawStorage.BlockDeviceCommands.RabbitCommands.VgRemove
	dst.Data.RawStorage.CmdLines.VgChange.LockStart = src.Data.RawStorage.BlockDeviceCommands.RabbitCommands.VgChange.LockStart
	dst.Data.RawStorage.CmdLines.VgChange.LockStop = src.Data.RawStorage.BlockDeviceCommands.RabbitCommands.VgChange.LockStop
	dst.Data.RawStorage.CmdLines.LvCreate = src.Data.RawStorage.BlockDeviceCommands.RabbitCommands.LvCreate
	dst.Data.RawStorage.CmdLines.LvRemove = src.Data.RawStorage.BlockDeviceCommands.RabbitCommands.LvRemove
	dst.Data.RawStorage.CmdLines.LvChange.Activate = src.Data.RawStorage.BlockDeviceCommands.RabbitCommands.LvChange.Activate
	dst.Data.RawStorage.CmdLines.LvChange.Deactivate = src.Data.RawStorage.BlockDeviceCommands.RabbitCommands.LvChange.Deactivate
	dst.Data.RawStorage.CmdLines.LVMRebuild.VgExtend = src.Data.RawStorage.BlockDeviceCommands.RabbitCommands.LVMRebuild.VgExtend
	dst.Data.RawStorage.CmdLines.LVMRebuild.VgReduce = src.Data.RawStorage.BlockDeviceCommands.RabbitCommands.LVMRebuild.VgReduce
	dst.Data.RawStorage.CmdLines.LVMRebuild.LvRepair = src.Data.RawStorage.BlockDeviceCommands.RabbitCommands.LVMRebuild.LvRepair

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfSystemStorage) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfSystemStorage To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha10.NnfSystemStorage)

	if err := Convert_v1alpha8_NnfSystemStorage_To_v1alpha10_NnfSystemStorage(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha10.NnfSystemStorage{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	return nil
}

func (dst *NnfSystemStorage) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha10.NnfSystemStorage)
	convertlog.Info("Convert NnfSystemStorage From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha10_NnfSystemStorage_To_v1alpha8_NnfSystemStorage(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

// The List-based ConvertTo/ConvertFrom routines are never used by the
// conversion webhook, but the conversion-verifier tool wants to see them.
// The conversion-gen tool generated the Convert_X_to_Y routines, should they
// ever be needed.

func resource(resource string) schema.GroupResource {
	return schema.GroupResource{Group: "nnf", Resource: resource}
}

func (src *NnfAccessList) ConvertTo(dstRaw conversion.Hub) error {
	return apierrors.NewMethodNotSupported(resource("NnfAccessList"), "ConvertTo")
}

func (dst *NnfAccessList) ConvertFrom(srcRaw conversion.Hub) error {
	return apierrors.NewMethodNotSupported(resource("NnfAccessList"), "ConvertFrom")
}

func (src *NnfContainerProfileList) ConvertTo(dstRaw conversion.Hub) error {
	return apierrors.NewMethodNotSupported(resource("NnfContainerProfileList"), "ConvertTo")
}

func (dst *NnfContainerProfileList) ConvertFrom(srcRaw conversion.Hub) error {
	return apierrors.NewMethodNotSupported(resource("NnfContainerProfileList"), "ConvertFrom")
}

func (src *NnfDataMovementList) ConvertTo(dstRaw conversion.Hub) error {
	return apierrors.NewMethodNotSupported(resource("NnfDataMovementList"), "ConvertTo")
}

func (dst *NnfDataMovementList) ConvertFrom(srcRaw conversion.Hub) error {
	return apierrors.NewMethodNotSupported(resource("NnfDataMovementList"), "ConvertFrom")
}

func (src *NnfDataMovementManagerList) ConvertTo(dstRaw conversion.Hub) error {
	return apierrors.NewMethodNotSupported(resource("NnfDataMovementManagerList"), "ConvertTo")
}

func (dst *NnfDataMovementManagerList) ConvertFrom(srcRaw conversion.Hub) error {
	return apierrors.NewMethodNotSupported(resource("NnfDataMovementManagerList"), "ConvertFrom")
}

func (src *NnfDataMovementProfileList) ConvertTo(dstRaw conversion.Hub) error {
	return apierrors.NewMethodNotSupported(resource("NnfDataMovementProfileList"), "ConvertTo")
}

func (dst *NnfDataMovementProfileList) ConvertFrom(srcRaw conversion.Hub) error {
	return apierrors.NewMethodNotSupported(resource("NnfDataMovementProfileList"), "ConvertFrom")
}

func (src *NnfLustreMGTList) ConvertTo(dstRaw conversion.Hub) error {
	return apierrors.NewMethodNotSupported(resource("NnfLustreMGTList"), "ConvertTo")
}

func (dst *NnfLustreMGTList) ConvertFrom(srcRaw conversion.Hub) error {
	return apierrors.NewMethodNotSupported(resource("NnfLustreMGTList"), "ConvertFrom")
}

func (src *NnfNodeList) ConvertTo(dstRaw conversion.Hub) error {
	return apierrors.NewMethodNotSupported(resource("NnfNodeList"), "ConvertTo")
}

func (dst *NnfNodeList) ConvertFrom(srcRaw conversion.Hub) error {
	return apierrors.NewMethodNotSupported(resource("NnfNodeList"), "ConvertFrom")
}

func (src *NnfNodeBlockStorageList) ConvertTo(dstRaw conversion.Hub) error {
	return apierrors.NewMethodNotSupported(resource("NnfNodeBlockStorageList"), "ConvertTo")
}

func (dst *NnfNodeBlockStorageList) ConvertFrom(srcRaw conversion.Hub) error {
	return apierrors.NewMethodNotSupported(resource("NnfNodeBlockStorageList"), "ConvertFrom")
}

func (src *NnfNodeECDataList) ConvertTo(dstRaw conversion.Hub) error {
	return apierrors.NewMethodNotSupported(resource("NnfNodeECDataList"), "ConvertTo")
}

func (dst *NnfNodeECDataList) ConvertFrom(srcRaw conversion.Hub) error {
	return apierrors.NewMethodNotSupported(resource("NnfNodeECDataList"), "ConvertFrom")
}

func (src *NnfNodeStorageList) ConvertTo(dstRaw conversion.Hub) error {
	return apierrors.NewMethodNotSupported(resource("NnfNodeStorageList"), "ConvertTo")
}

func (dst *NnfNodeStorageList) ConvertFrom(srcRaw conversion.Hub) error {
	return apierrors.NewMethodNotSupported(resource("NnfNodeStorageList"), "ConvertFrom")
}

func (src *NnfPortManagerList) ConvertTo(dstRaw conversion.Hub) error {
	return apierrors.NewMethodNotSupported(resource("NnfPortManagerList"), "ConvertTo")
}

func (dst *NnfPortManagerList) ConvertFrom(srcRaw conversion.Hub) error {
	return apierrors.NewMethodNotSupported(resource("NnfPortManagerList"), "ConvertFrom")
}

func (src *NnfStorageList) ConvertTo(dstRaw conversion.Hub) error {
	return apierrors.NewMethodNotSupported(resource("NnfStorageList"), "ConvertTo")
}

func (dst *NnfStorageList) ConvertFrom(srcRaw conversion.Hub) error {
	return apierrors.NewMethodNotSupported(resource("NnfStorageList"), "ConvertFrom")
}

func (src *NnfStorageProfileList) ConvertTo(dstRaw conversion.Hub) error {
	return apierrors.NewMethodNotSupported(resource("NnfStorageProfileList"), "ConvertTo")
}

func (dst *NnfStorageProfileList) ConvertFrom(srcRaw conversion.Hub) error {
	return apierrors.NewMethodNotSupported(resource("NnfStorageProfileList"), "ConvertFrom")
}

func (src *NnfSystemStorageList) ConvertTo(dstRaw conversion.Hub) error {
	return apierrors.NewMethodNotSupported(resource("NnfSystemStorageList"), "ConvertTo")
}

func (dst *NnfSystemStorageList) ConvertFrom(srcRaw conversion.Hub) error {
	return apierrors.NewMethodNotSupported(resource("NnfSystemStorageList"), "ConvertFrom")
}

// +crdbumper:carryforward:begin="Epilog"
// DWS ResourceError conversion routines. These must be present in the oldest
// spoke, and only in the oldest spoke.
// Then re-run "make generate-go-conversions" and it should find these and hook
// them up in the other spokes.

func autoConvert_v1alpha6_ResourceError_To_v1alpha7_ResourceError(in *dwsv1alpha6.ResourceError, out *dwsv1alpha7.ResourceError, s apiconversion.Scope) error {
	out.Error = (*dwsv1alpha7.ResourceErrorInfo)(unsafe.Pointer(in.Error))
	return nil
}

// Convert_v1alpha6_ResourceError_To_v1alpha7_ResourceError is an autogenerated conversion function.
func Convert_v1alpha6_ResourceError_To_v1alpha7_ResourceError(in *dwsv1alpha6.ResourceError, out *dwsv1alpha7.ResourceError, s apiconversion.Scope) error {
	return autoConvert_v1alpha6_ResourceError_To_v1alpha7_ResourceError(in, out, s)
}

func autoConvert_v1alpha7_ResourceError_To_v1alpha6_ResourceError(in *dwsv1alpha7.ResourceError, out *dwsv1alpha6.ResourceError, s apiconversion.Scope) error {
	out.Error = (*dwsv1alpha6.ResourceErrorInfo)(unsafe.Pointer(in.Error))
	return nil
}

// Convert_4_ResourceError_To_v1alpha6_ResourceError is an autogenerated conversion function.
func Convert_v1alpha7_ResourceError_To_v1alpha6_ResourceError(in *dwsv1alpha7.ResourceError, out *dwsv1alpha6.ResourceError, s apiconversion.Scope) error {
	return autoConvert_v1alpha7_ResourceError_To_v1alpha6_ResourceError(in, out, s)
}

func autoConvert_v1alpha6_ResourceErrorInfo_To_v1alpha7_ResourceErrorInfo(in *dwsv1alpha6.ResourceErrorInfo, out *dwsv1alpha7.ResourceErrorInfo, s apiconversion.Scope) error {
	out.UserMessage = in.UserMessage
	out.DebugMessage = in.DebugMessage
	out.Type = dwsv1alpha7.ResourceErrorType(in.Type)
	out.Severity = dwsv1alpha7.ResourceErrorSeverity(in.Severity)
	return nil
}

// Convert_v1alpha6_ResourceErrorInfo_To_v1alpha7_ResourceErrorInfo is an autogenerated conversion function.
func Convert_v1alpha6_ResourceErrorInfo_To_v1alpha7_ResourceErrorInfo(in *dwsv1alpha6.ResourceErrorInfo, out *dwsv1alpha7.ResourceErrorInfo, s apiconversion.Scope) error {
	return autoConvert_v1alpha6_ResourceErrorInfo_To_v1alpha7_ResourceErrorInfo(in, out, s)
}

func autoConvert_v1alpha7_ResourceErrorInfo_To_v1alpha6_ResourceErrorInfo(in *dwsv1alpha7.ResourceErrorInfo, out *dwsv1alpha6.ResourceErrorInfo, s apiconversion.Scope) error {
	out.UserMessage = in.UserMessage
	out.DebugMessage = in.DebugMessage
	out.Type = dwsv1alpha6.ResourceErrorType(in.Type)
	out.Severity = dwsv1alpha6.ResourceErrorSeverity(in.Severity)
	return nil
}

// Convert_v1alpha7_ResourceErrorInfo_To_v1alpha6_ResourceErrorInfo is an autogenerated conversion function.
func Convert_v1alpha7_ResourceErrorInfo_To_v1alpha6_ResourceErrorInfo(in *dwsv1alpha7.ResourceErrorInfo, out *dwsv1alpha6.ResourceErrorInfo, s apiconversion.Scope) error {
	return autoConvert_v1alpha7_ResourceErrorInfo_To_v1alpha6_ResourceErrorInfo(in, out, s)
}

// End of DWS ResourceError conversion routines.
// +crdbumper:carryforward:end

// Convert_v1alpha8_NnfStorageProfileGFS2Data_To_v1alpha10_NnfStorageProfileGFS2Data is an autogenerated conversion function.
func Convert_v1alpha8_NnfStorageProfileGFS2Data_To_v1alpha10_NnfStorageProfileGFS2Data(in *NnfStorageProfileGFS2Data, out *nnfv1alpha10.NnfStorageProfileGFS2Data, s apiconversion.Scope) error {
	return autoConvert_v1alpha8_NnfStorageProfileGFS2Data_To_v1alpha10_NnfStorageProfileGFS2Data(in, out, s)
}

// Convert_v1alpha10_NnfResourceStatus_To_v1alpha8_NnfResourceStatus converts NnfResourceStatus from hub to v1alpha8.
// This overrides the auto-generated function to map ResourceFenced to ResourceOffline.
func Convert_v1alpha10_NnfResourceStatus_To_v1alpha8_NnfResourceStatus(in *nnfv1alpha10.NnfResourceStatus, out *NnfResourceStatus, s apiconversion.Scope) error {
	out.ID = in.ID
	out.Name = in.Name
	out.Health = NnfResourceHealthType(in.Health)
	// Map ResourceFenced to ResourceOffline since v1alpha8 doesn't have Fenced
	if in.Status == nnfv1alpha10.ResourceFenced {
		out.Status = ResourceOffline
	} else {
		out.Status = NnfResourceStatusType(in.Status)
	}
	return nil
}

// Convert_v1alpha10_NnfStorageProfileGFS2Data_To_v1alpha8_NnfStorageProfileGFS2Data is an autogenerated conversion function.
func Convert_v1alpha10_NnfStorageProfileGFS2Data_To_v1alpha8_NnfStorageProfileGFS2Data(in *nnfv1alpha10.NnfStorageProfileGFS2Data, out *NnfStorageProfileGFS2Data, s apiconversion.Scope) error {
	return autoConvert_v1alpha10_NnfStorageProfileGFS2Data_To_v1alpha8_NnfStorageProfileGFS2Data(in, out, s)
}

// Convert_v1alpha10_NnfStorageProfileLustreClientCmdLines_To_v1alpha8_NnfStorageProfileLustreClientCmdLines is an autogenerated conversion function.
func Convert_v1alpha10_NnfStorageProfileLustreClientCmdLines_To_v1alpha8_NnfStorageProfileLustreClientCmdLines(in *nnfv1alpha10.NnfStorageProfileLustreClientCmdLines, out *NnfStorageProfileLustreClientCmdLines, s apiconversion.Scope) error {
	return autoConvert_v1alpha10_NnfStorageProfileLustreClientCmdLines_To_v1alpha8_NnfStorageProfileLustreClientCmdLines(in, out, s)
}

// Convert_v1alpha8_NnfStorageProfileRawData_To_v1alpha10_NnfStorageProfileRawData is an autogenerated conversion function.
func Convert_v1alpha8_NnfStorageProfileRawData_To_v1alpha10_NnfStorageProfileRawData(in *NnfStorageProfileRawData, out *nnfv1alpha10.NnfStorageProfileRawData, s apiconversion.Scope) error {
	return autoConvert_v1alpha8_NnfStorageProfileRawData_To_v1alpha10_NnfStorageProfileRawData(in, out, s)
}

// Convert_v1alpha10_NnfStorageProfileRawData_To_v1alpha8_NnfStorageProfileRawData is an autogenerated conversion function.
func Convert_v1alpha10_NnfStorageProfileRawData_To_v1alpha8_NnfStorageProfileRawData(in *nnfv1alpha10.NnfStorageProfileRawData, out *NnfStorageProfileRawData, s apiconversion.Scope) error {
	return autoConvert_v1alpha10_NnfStorageProfileRawData_To_v1alpha8_NnfStorageProfileRawData(in, out, s)
}

// Convert_v1alpha8_NnfStorageProfileXFSData_To_v1alpha10_NnfStorageProfileXFSData is an autogenerated conversion function.
func Convert_v1alpha8_NnfStorageProfileXFSData_To_v1alpha10_NnfStorageProfileXFSData(in *NnfStorageProfileXFSData, out *nnfv1alpha10.NnfStorageProfileXFSData, s apiconversion.Scope) error {
	return autoConvert_v1alpha8_NnfStorageProfileXFSData_To_v1alpha10_NnfStorageProfileXFSData(in, out, s)
}

// Convert_v1alpha10_NnfStorageProfileXFSData_To_v1alpha8_NnfStorageProfileXFSData is an autogenerated conversion function.
func Convert_v1alpha10_NnfStorageProfileXFSData_To_v1alpha8_NnfStorageProfileXFSData(in *nnfv1alpha10.NnfStorageProfileXFSData, out *NnfStorageProfileXFSData, s apiconversion.Scope) error {
	return autoConvert_v1alpha10_NnfStorageProfileXFSData_To_v1alpha8_NnfStorageProfileXFSData(in, out, s)
}
