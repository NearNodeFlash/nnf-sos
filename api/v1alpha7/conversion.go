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

package v1alpha7

import (
	unsafe "unsafe"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apiconversion "k8s.io/apimachinery/pkg/conversion"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	dwsv1alpha4 "github.com/DataWorkflowServices/dws/api/v1alpha4"
	dwsv1alpha7 "github.com/DataWorkflowServices/dws/api/v1alpha7"
	nnfv1alpha10 "github.com/NearNodeFlash/nnf-sos/api/v1alpha10"
	utilconversion "github.com/NearNodeFlash/nnf-sos/github/cluster-api/util/conversion"
)

var convertlog = logf.Log.V(2).WithName("convert-v1alpha7")

func (src *NnfAccess) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfAccess To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha10.NnfAccess)

	if err := Convert_v1alpha7_NnfAccess_To_v1alpha10_NnfAccess(src, dst, nil); err != nil {
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

	if err := Convert_v1alpha10_NnfAccess_To_v1alpha7_NnfAccess(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfContainerProfile) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfContainerProfile To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha10.NnfContainerProfile)

	if err := Convert_v1alpha7_NnfContainerProfile_To_v1alpha10_NnfContainerProfile(src, dst, nil); err != nil {
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

	if err := Convert_v1alpha10_NnfContainerProfile_To_v1alpha7_NnfContainerProfile(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfDataMovement) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfDataMovement To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha10.NnfDataMovement)

	if err := Convert_v1alpha7_NnfDataMovement_To_v1alpha10_NnfDataMovement(src, dst, nil); err != nil {
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

	if err := Convert_v1alpha10_NnfDataMovement_To_v1alpha7_NnfDataMovement(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfDataMovementManager) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfDataMovementManager To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha10.NnfDataMovementManager)

	if err := Convert_v1alpha7_NnfDataMovementManager_To_v1alpha10_NnfDataMovementManager(src, dst, nil); err != nil {
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

	if err := Convert_v1alpha10_NnfDataMovementManager_To_v1alpha7_NnfDataMovementManager(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfDataMovementProfile) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfDataMovementProfile To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha10.NnfDataMovementProfile)

	if err := Convert_v1alpha7_NnfDataMovementProfile_To_v1alpha10_NnfDataMovementProfile(src, dst, nil); err != nil {
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

	if err := Convert_v1alpha10_NnfDataMovementProfile_To_v1alpha7_NnfDataMovementProfile(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfLustreMGT) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfLustreMGT To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha10.NnfLustreMGT)

	if err := Convert_v1alpha7_NnfLustreMGT_To_v1alpha10_NnfLustreMGT(src, dst, nil); err != nil {
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

	if err := Convert_v1alpha10_NnfLustreMGT_To_v1alpha7_NnfLustreMGT(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfNode) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfNode To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha10.NnfNode)

	if err := Convert_v1alpha7_NnfNode_To_v1alpha10_NnfNode(src, dst, nil); err != nil {
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

	if err := Convert_v1alpha10_NnfNode_To_v1alpha7_NnfNode(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfNodeBlockStorage) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfNodeBlockStorage To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha10.NnfNodeBlockStorage)

	if err := Convert_v1alpha7_NnfNodeBlockStorage_To_v1alpha10_NnfNodeBlockStorage(src, dst, nil); err != nil {
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

	if err := Convert_v1alpha10_NnfNodeBlockStorage_To_v1alpha7_NnfNodeBlockStorage(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfNodeECData) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfNodeECData To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha10.NnfNodeECData)

	if err := Convert_v1alpha7_NnfNodeECData_To_v1alpha10_NnfNodeECData(src, dst, nil); err != nil {
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

	if err := Convert_v1alpha10_NnfNodeECData_To_v1alpha7_NnfNodeECData(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfNodeStorage) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfNodeStorage To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha10.NnfNodeStorage)

	if err := Convert_v1alpha7_NnfNodeStorage_To_v1alpha10_NnfNodeStorage(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha10.NnfNodeStorage{}
	hasAnno, err := utilconversion.UnmarshalData(src, restored)
	if err != nil {
		return err
	}

	if hasAnno {
		dst.Status.Health = restored.Status.Health
		for i := range restored.Status.Allocations {
			dst.Status.Allocations[i].Health = restored.Status.Allocations[i].Health
		}
	}

	return nil
}

func (dst *NnfNodeStorage) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha10.NnfNodeStorage)
	convertlog.Info("Convert NnfNodeStorage From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha10_NnfNodeStorage_To_v1alpha7_NnfNodeStorage(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfPortManager) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfPortManager To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha10.NnfPortManager)

	if err := Convert_v1alpha7_NnfPortManager_To_v1alpha10_NnfPortManager(src, dst, nil); err != nil {
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

	if err := Convert_v1alpha10_NnfPortManager_To_v1alpha7_NnfPortManager(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfStorage) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfStorage To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha10.NnfStorage)

	if err := Convert_v1alpha7_NnfStorage_To_v1alpha10_NnfStorage(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha10.NnfStorage{}
	hasAnno, err := utilconversion.UnmarshalData(src, restored)
	if err != nil {
		return err
	}

	if hasAnno {
		dst.Status.Health = restored.Status.Health
		for i := range restored.Status.AllocationSets {
			dst.Status.AllocationSets[i].Health = restored.Status.AllocationSets[i].Health
		}
	}

	return nil
}

func (dst *NnfStorage) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha10.NnfStorage)
	convertlog.Info("Convert NnfStorage From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha10_NnfStorage_To_v1alpha7_NnfStorage(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfStorageProfile) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfStorageProfile To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha10.NnfStorageProfile)

	if err := Convert_v1alpha7_NnfStorageProfile_To_v1alpha10_NnfStorageProfile(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha10.NnfStorageProfile{}
	hasAnno, err := utilconversion.UnmarshalData(src, restored)
	if err != nil {
		return err
	}

	dst.Data.LustreStorage.ClientOptions.CmdLines.RabbitPostSetup = src.Data.LustreStorage.ClientCmdLines.RabbitPostMount
	dst.Data.LustreStorage.ClientOptions.CmdLines.RabbitPreTeardown = src.Data.LustreStorage.ClientCmdLines.RabbitPreUnmount

	// MGT options
	dst.Data.LustreStorage.MgtOptions.ExternalMGS = src.Data.LustreStorage.ExternalMGS
	dst.Data.LustreStorage.MgtOptions.StandaloneMGTPoolName = src.Data.LustreStorage.StandaloneMGTPoolName
	dst.Data.LustreStorage.MgtOptions.Capacity = src.Data.LustreStorage.CapacityMGT
	dst.Data.LustreStorage.MgtOptions.CmdLines.ZpoolCreate = src.Data.LustreStorage.MgtCmdLines.ZpoolCreate
	dst.Data.LustreStorage.MgtOptions.CmdLines.Mkfs = src.Data.LustreStorage.MgtCmdLines.Mkfs
	dst.Data.LustreStorage.MgtOptions.CmdLines.MountTarget = src.Data.LustreStorage.MgtCmdLines.MountTarget
	dst.Data.LustreStorage.MgtOptions.CmdLines.PostActivate = src.Data.LustreStorage.MgtCmdLines.PostActivate
	dst.Data.LustreStorage.MgtOptions.CmdLines.PreDeactivate = src.Data.LustreStorage.MgtCmdLines.PreDeactivate
	dst.Data.LustreStorage.MgtOptions.ColocateComputes = src.Data.LustreStorage.MgtOptions.ColocateComputes
	dst.Data.LustreStorage.MgtOptions.Count = src.Data.LustreStorage.MgtOptions.Count
	dst.Data.LustreStorage.MgtOptions.Scale = src.Data.LustreStorage.MgtOptions.Scale
	dst.Data.LustreStorage.MgtOptions.StorageLabels = src.Data.LustreStorage.MgtOptions.StorageLabels
	dst.Data.LustreStorage.MgtOptions.PreMountCommands = src.Data.LustreStorage.PreMountMGTCmds

	// MDT options
	dst.Data.LustreStorage.MdtOptions.Capacity = src.Data.LustreStorage.CapacityMDT
	dst.Data.LustreStorage.MdtOptions.Exclusive = src.Data.LustreStorage.ExclusiveMDT
	dst.Data.LustreStorage.MdtOptions.CmdLines.ZpoolCreate = src.Data.LustreStorage.MdtCmdLines.ZpoolCreate
	dst.Data.LustreStorage.MdtOptions.CmdLines.Mkfs = src.Data.LustreStorage.MdtCmdLines.Mkfs
	dst.Data.LustreStorage.MdtOptions.CmdLines.MountTarget = src.Data.LustreStorage.MdtCmdLines.MountTarget
	dst.Data.LustreStorage.MdtOptions.CmdLines.PostActivate = src.Data.LustreStorage.MdtCmdLines.PostActivate
	dst.Data.LustreStorage.MdtOptions.CmdLines.PreDeactivate = src.Data.LustreStorage.MdtCmdLines.PreDeactivate
	dst.Data.LustreStorage.MdtOptions.ColocateComputes = src.Data.LustreStorage.MdtOptions.ColocateComputes
	dst.Data.LustreStorage.MdtOptions.Count = src.Data.LustreStorage.MdtOptions.Count
	dst.Data.LustreStorage.MdtOptions.Scale = src.Data.LustreStorage.MdtOptions.Scale
	dst.Data.LustreStorage.MdtOptions.StorageLabels = src.Data.LustreStorage.MdtOptions.StorageLabels

	// MGT/MDT combined options
	dst.Data.LustreStorage.MgtMdtOptions.Capacity = src.Data.LustreStorage.CapacityMDT
	dst.Data.LustreStorage.MgtMdtOptions.CmdLines.ZpoolCreate = src.Data.LustreStorage.MgtMdtCmdLines.ZpoolCreate
	dst.Data.LustreStorage.MgtMdtOptions.CmdLines.Mkfs = src.Data.LustreStorage.MgtMdtCmdLines.Mkfs
	dst.Data.LustreStorage.MgtMdtOptions.CmdLines.MountTarget = src.Data.LustreStorage.MgtMdtCmdLines.MountTarget
	dst.Data.LustreStorage.MgtMdtOptions.CmdLines.PostActivate = src.Data.LustreStorage.MgtMdtCmdLines.PostActivate
	dst.Data.LustreStorage.MgtMdtOptions.CmdLines.PreDeactivate = src.Data.LustreStorage.MgtMdtCmdLines.PreDeactivate
	dst.Data.LustreStorage.MgtMdtOptions.ColocateComputes = src.Data.LustreStorage.MgtMdtOptions.ColocateComputes
	dst.Data.LustreStorage.MgtMdtOptions.Count = src.Data.LustreStorage.MgtMdtOptions.Count
	dst.Data.LustreStorage.MgtMdtOptions.Scale = src.Data.LustreStorage.MgtMdtOptions.Scale
	dst.Data.LustreStorage.MgtMdtOptions.StorageLabels = src.Data.LustreStorage.MgtMdtOptions.StorageLabels

	// OST options
	dst.Data.LustreStorage.OstOptions.CapacityScalingFactor = src.Data.LustreStorage.CapacityScalingFactor
	dst.Data.LustreStorage.OstOptions.CmdLines.ZpoolCreate = src.Data.LustreStorage.OstCmdLines.ZpoolCreate
	dst.Data.LustreStorage.OstOptions.CmdLines.Mkfs = src.Data.LustreStorage.OstCmdLines.Mkfs
	dst.Data.LustreStorage.OstOptions.CmdLines.MountTarget = src.Data.LustreStorage.OstCmdLines.MountTarget
	dst.Data.LustreStorage.OstOptions.CmdLines.PostActivate = src.Data.LustreStorage.OstCmdLines.PostActivate
	dst.Data.LustreStorage.OstOptions.CmdLines.PreDeactivate = src.Data.LustreStorage.OstCmdLines.PreDeactivate
	dst.Data.LustreStorage.OstOptions.ColocateComputes = src.Data.LustreStorage.OstOptions.ColocateComputes
	dst.Data.LustreStorage.OstOptions.Count = src.Data.LustreStorage.OstOptions.Count
	dst.Data.LustreStorage.OstOptions.Scale = src.Data.LustreStorage.OstOptions.Scale
	dst.Data.LustreStorage.OstOptions.StorageLabels = src.Data.LustreStorage.OstOptions.StorageLabels

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

	if hasAnno {
		dst.Data.LustreStorage.ClientOptions.CmdLines.RabbitPostSetup = restored.Data.LustreStorage.ClientOptions.CmdLines.RabbitPostSetup
		dst.Data.LustreStorage.ClientOptions.CmdLines.RabbitPreTeardown = restored.Data.LustreStorage.ClientOptions.CmdLines.RabbitPreTeardown
		dst.Data.LustreStorage.ClientOptions.CmdLines.RabbitPreMount = restored.Data.LustreStorage.ClientOptions.CmdLines.RabbitPreMount
		dst.Data.LustreStorage.ClientOptions.CmdLines.RabbitPostMount = restored.Data.LustreStorage.ClientOptions.CmdLines.RabbitPostMount
		dst.Data.LustreStorage.ClientOptions.CmdLines.RabbitPreUnmount = restored.Data.LustreStorage.ClientOptions.CmdLines.RabbitPreUnmount
		dst.Data.LustreStorage.ClientOptions.CmdLines.RabbitPostUnmount = restored.Data.LustreStorage.ClientOptions.CmdLines.RabbitPostUnmount
		dst.Data.LustreStorage.ClientOptions.CmdLines.ComputePreMount = restored.Data.LustreStorage.ClientOptions.CmdLines.ComputePreMount
		dst.Data.LustreStorage.ClientOptions.CmdLines.ComputePostMount = restored.Data.LustreStorage.ClientOptions.CmdLines.ComputePostMount
		dst.Data.LustreStorage.ClientOptions.CmdLines.ComputePreUnmount = restored.Data.LustreStorage.ClientOptions.CmdLines.ComputePreUnmount
		dst.Data.LustreStorage.ClientOptions.CmdLines.ComputePostUnmount = restored.Data.LustreStorage.ClientOptions.CmdLines.ComputePostUnmount
		dst.Data.LustreStorage.MgtOptions.CmdLines.ZpoolReplace = restored.Data.LustreStorage.MgtOptions.CmdLines.ZpoolReplace
		dst.Data.LustreStorage.MgtMdtOptions.CmdLines.ZpoolReplace = restored.Data.LustreStorage.MgtMdtOptions.CmdLines.ZpoolReplace
		dst.Data.LustreStorage.MdtOptions.CmdLines.ZpoolReplace = restored.Data.LustreStorage.MdtOptions.CmdLines.ZpoolReplace
		dst.Data.LustreStorage.OstOptions.CmdLines.ZpoolReplace = restored.Data.LustreStorage.OstOptions.CmdLines.ZpoolReplace
		dst.Data.LustreStorage.MgtMdtOptions.Capacity = restored.Data.LustreStorage.MgtMdtOptions.Capacity
		dst.Data.LustreStorage.MgtOptions.VariableOverride = restored.Data.LustreStorage.MgtOptions.VariableOverride
		dst.Data.LustreStorage.MgtMdtOptions.VariableOverride = restored.Data.LustreStorage.MgtMdtOptions.VariableOverride
		dst.Data.LustreStorage.MdtOptions.VariableOverride = restored.Data.LustreStorage.MdtOptions.VariableOverride
		dst.Data.LustreStorage.OstOptions.VariableOverride = restored.Data.LustreStorage.OstOptions.VariableOverride
		dst.Data.LustreStorage.ClientOptions.VariableOverride = restored.Data.LustreStorage.ClientOptions.VariableOverride

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
		dst.Data.GFS2Storage.BlockDeviceCommands.RabbitCommands.LVMRebuild.VgExtend = restored.Data.GFS2Storage.BlockDeviceCommands.RabbitCommands.LVMRebuild.VgExtend
		dst.Data.GFS2Storage.BlockDeviceCommands.RabbitCommands.LVMRebuild.VgReduce = restored.Data.GFS2Storage.BlockDeviceCommands.RabbitCommands.LVMRebuild.VgReduce
		dst.Data.GFS2Storage.BlockDeviceCommands.RabbitCommands.LVMRebuild.LvRepair = restored.Data.GFS2Storage.BlockDeviceCommands.RabbitCommands.LVMRebuild.LvRepair
		dst.Data.GFS2Storage.BlockDeviceCommands.ComputeCommands.LvChange.Activate = restored.Data.GFS2Storage.BlockDeviceCommands.ComputeCommands.LvChange.Activate
		dst.Data.GFS2Storage.BlockDeviceCommands.ComputeCommands.LvChange.Deactivate = restored.Data.GFS2Storage.BlockDeviceCommands.ComputeCommands.LvChange.Deactivate
		dst.Data.GFS2Storage.BlockDeviceCommands.ComputeCommands.VgChange.LockStart = restored.Data.GFS2Storage.BlockDeviceCommands.ComputeCommands.VgChange.LockStart
		dst.Data.GFS2Storage.BlockDeviceCommands.ComputeCommands.VgChange.LockStop = restored.Data.GFS2Storage.BlockDeviceCommands.ComputeCommands.VgChange.LockStop
		dst.Data.GFS2Storage.BlockDeviceCommands.ComputeCommands.UserCommands.PreActivate = restored.Data.GFS2Storage.BlockDeviceCommands.ComputeCommands.UserCommands.PreActivate
		dst.Data.GFS2Storage.BlockDeviceCommands.ComputeCommands.UserCommands.PostActivate = restored.Data.GFS2Storage.BlockDeviceCommands.ComputeCommands.UserCommands.PostActivate
		dst.Data.GFS2Storage.BlockDeviceCommands.ComputeCommands.UserCommands.PreDeactivate = restored.Data.GFS2Storage.BlockDeviceCommands.ComputeCommands.UserCommands.PreDeactivate
		dst.Data.GFS2Storage.BlockDeviceCommands.ComputeCommands.UserCommands.PostDeactivate = restored.Data.GFS2Storage.BlockDeviceCommands.ComputeCommands.UserCommands.PostDeactivate
		dst.Data.GFS2Storage.VariableOverride = restored.Data.GFS2Storage.VariableOverride

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
		dst.Data.XFSStorage.BlockDeviceCommands.RabbitCommands.LVMRebuild.VgExtend = restored.Data.XFSStorage.BlockDeviceCommands.RabbitCommands.LVMRebuild.VgExtend
		dst.Data.XFSStorage.BlockDeviceCommands.RabbitCommands.LVMRebuild.VgReduce = restored.Data.XFSStorage.BlockDeviceCommands.RabbitCommands.LVMRebuild.VgReduce
		dst.Data.XFSStorage.BlockDeviceCommands.RabbitCommands.LVMRebuild.LvRepair = restored.Data.XFSStorage.BlockDeviceCommands.RabbitCommands.LVMRebuild.LvRepair
		dst.Data.XFSStorage.BlockDeviceCommands.ComputeCommands.LvChange.Activate = restored.Data.XFSStorage.BlockDeviceCommands.ComputeCommands.LvChange.Activate
		dst.Data.XFSStorage.BlockDeviceCommands.ComputeCommands.LvChange.Deactivate = restored.Data.XFSStorage.BlockDeviceCommands.ComputeCommands.LvChange.Deactivate
		dst.Data.XFSStorage.BlockDeviceCommands.ComputeCommands.VgChange.LockStart = restored.Data.XFSStorage.BlockDeviceCommands.ComputeCommands.VgChange.LockStart
		dst.Data.XFSStorage.BlockDeviceCommands.ComputeCommands.VgChange.LockStop = restored.Data.XFSStorage.BlockDeviceCommands.ComputeCommands.VgChange.LockStop
		dst.Data.XFSStorage.BlockDeviceCommands.ComputeCommands.UserCommands.PreActivate = restored.Data.XFSStorage.BlockDeviceCommands.ComputeCommands.UserCommands.PreActivate
		dst.Data.XFSStorage.BlockDeviceCommands.ComputeCommands.UserCommands.PostActivate = restored.Data.XFSStorage.BlockDeviceCommands.ComputeCommands.UserCommands.PostActivate
		dst.Data.XFSStorage.BlockDeviceCommands.ComputeCommands.UserCommands.PreDeactivate = restored.Data.XFSStorage.BlockDeviceCommands.ComputeCommands.UserCommands.PreDeactivate
		dst.Data.XFSStorage.BlockDeviceCommands.ComputeCommands.UserCommands.PostDeactivate = restored.Data.XFSStorage.BlockDeviceCommands.ComputeCommands.UserCommands.PostDeactivate
		dst.Data.XFSStorage.VariableOverride = restored.Data.XFSStorage.VariableOverride

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
		dst.Data.RawStorage.BlockDeviceCommands.RabbitCommands.LVMRebuild.VgExtend = restored.Data.RawStorage.BlockDeviceCommands.RabbitCommands.LVMRebuild.VgExtend
		dst.Data.RawStorage.BlockDeviceCommands.RabbitCommands.LVMRebuild.VgReduce = restored.Data.RawStorage.BlockDeviceCommands.RabbitCommands.LVMRebuild.VgReduce
		dst.Data.RawStorage.BlockDeviceCommands.RabbitCommands.LVMRebuild.LvRepair = restored.Data.RawStorage.BlockDeviceCommands.RabbitCommands.LVMRebuild.LvRepair
		dst.Data.RawStorage.BlockDeviceCommands.ComputeCommands.LvChange.Activate = restored.Data.RawStorage.BlockDeviceCommands.ComputeCommands.LvChange.Activate
		dst.Data.RawStorage.BlockDeviceCommands.ComputeCommands.LvChange.Deactivate = restored.Data.RawStorage.BlockDeviceCommands.ComputeCommands.LvChange.Deactivate
		dst.Data.RawStorage.BlockDeviceCommands.ComputeCommands.VgChange.LockStart = restored.Data.RawStorage.BlockDeviceCommands.ComputeCommands.VgChange.LockStart
		dst.Data.RawStorage.BlockDeviceCommands.ComputeCommands.VgChange.LockStop = restored.Data.RawStorage.BlockDeviceCommands.ComputeCommands.VgChange.LockStop
		dst.Data.RawStorage.BlockDeviceCommands.ComputeCommands.UserCommands.PreActivate = restored.Data.RawStorage.BlockDeviceCommands.ComputeCommands.UserCommands.PreActivate
		dst.Data.RawStorage.BlockDeviceCommands.ComputeCommands.UserCommands.PostActivate = restored.Data.RawStorage.BlockDeviceCommands.ComputeCommands.UserCommands.PostActivate
		dst.Data.RawStorage.BlockDeviceCommands.ComputeCommands.UserCommands.PreDeactivate = restored.Data.RawStorage.BlockDeviceCommands.ComputeCommands.UserCommands.PreDeactivate
		dst.Data.RawStorage.BlockDeviceCommands.ComputeCommands.UserCommands.PostDeactivate = restored.Data.RawStorage.BlockDeviceCommands.ComputeCommands.UserCommands.PostDeactivate
		dst.Data.RawStorage.VariableOverride = restored.Data.RawStorage.VariableOverride
	}

	return nil
}

func (dst *NnfStorageProfile) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha10.NnfStorageProfile)
	convertlog.Info("Convert NnfStorageProfile From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha10_NnfStorageProfile_To_v1alpha7_NnfStorageProfile(src, dst, nil); err != nil {
		return err
	}

	dst.Data.LustreStorage.ClientCmdLines.RabbitPostMount = src.Data.LustreStorage.ClientOptions.CmdLines.RabbitPostSetup
	dst.Data.LustreStorage.ClientCmdLines.RabbitPreUnmount = src.Data.LustreStorage.ClientOptions.CmdLines.RabbitPreTeardown

	// MGT options
	dst.Data.LustreStorage.ExternalMGS = src.Data.LustreStorage.MgtOptions.ExternalMGS
	dst.Data.LustreStorage.StandaloneMGTPoolName = src.Data.LustreStorage.MgtOptions.StandaloneMGTPoolName
	dst.Data.LustreStorage.CapacityMGT = src.Data.LustreStorage.MgtOptions.Capacity
	dst.Data.LustreStorage.MgtCmdLines.ZpoolCreate = src.Data.LustreStorage.MgtOptions.CmdLines.ZpoolCreate
	dst.Data.LustreStorage.MgtCmdLines.Mkfs = src.Data.LustreStorage.MgtOptions.CmdLines.Mkfs
	dst.Data.LustreStorage.MgtCmdLines.MountTarget = src.Data.LustreStorage.MgtOptions.CmdLines.MountTarget
	dst.Data.LustreStorage.MgtCmdLines.PostActivate = src.Data.LustreStorage.MgtOptions.CmdLines.PostActivate
	dst.Data.LustreStorage.MgtCmdLines.PreDeactivate = src.Data.LustreStorage.MgtOptions.CmdLines.PreDeactivate
	dst.Data.LustreStorage.MgtOptions.ColocateComputes = src.Data.LustreStorage.MgtOptions.ColocateComputes
	dst.Data.LustreStorage.MgtOptions.Count = src.Data.LustreStorage.MgtOptions.Count
	dst.Data.LustreStorage.MgtOptions.Scale = src.Data.LustreStorage.MgtOptions.Scale
	dst.Data.LustreStorage.MgtOptions.StorageLabels = src.Data.LustreStorage.MgtOptions.StorageLabels
	dst.Data.LustreStorage.PreMountMGTCmds = src.Data.LustreStorage.MgtOptions.PreMountCommands

	// MDT options
	dst.Data.LustreStorage.CapacityMDT = src.Data.LustreStorage.MdtOptions.Capacity
	dst.Data.LustreStorage.ExclusiveMDT = src.Data.LustreStorage.MdtOptions.Exclusive
	dst.Data.LustreStorage.MdtCmdLines.ZpoolCreate = src.Data.LustreStorage.MdtOptions.CmdLines.ZpoolCreate
	dst.Data.LustreStorage.MdtCmdLines.Mkfs = src.Data.LustreStorage.MdtOptions.CmdLines.Mkfs
	dst.Data.LustreStorage.MdtCmdLines.MountTarget = src.Data.LustreStorage.MdtOptions.CmdLines.MountTarget
	dst.Data.LustreStorage.MdtCmdLines.PostActivate = src.Data.LustreStorage.MdtOptions.CmdLines.PostActivate
	dst.Data.LustreStorage.MdtCmdLines.PreDeactivate = src.Data.LustreStorage.MdtOptions.CmdLines.PreDeactivate
	dst.Data.LustreStorage.MdtOptions.ColocateComputes = src.Data.LustreStorage.MdtOptions.ColocateComputes
	dst.Data.LustreStorage.MdtOptions.Count = src.Data.LustreStorage.MdtOptions.Count
	dst.Data.LustreStorage.MdtOptions.Scale = src.Data.LustreStorage.MdtOptions.Scale
	dst.Data.LustreStorage.MdtOptions.StorageLabels = src.Data.LustreStorage.MdtOptions.StorageLabels

	// MGT/MDT combined options
	dst.Data.LustreStorage.MgtMdtCmdLines.ZpoolCreate = src.Data.LustreStorage.MgtMdtOptions.CmdLines.ZpoolCreate
	dst.Data.LustreStorage.MgtMdtCmdLines.Mkfs = src.Data.LustreStorage.MgtMdtOptions.CmdLines.Mkfs
	dst.Data.LustreStorage.MgtMdtCmdLines.MountTarget = src.Data.LustreStorage.MgtMdtOptions.CmdLines.MountTarget
	dst.Data.LustreStorage.MgtMdtCmdLines.PostActivate = src.Data.LustreStorage.MgtMdtOptions.CmdLines.PostActivate
	dst.Data.LustreStorage.MgtMdtCmdLines.PreDeactivate = src.Data.LustreStorage.MgtMdtOptions.CmdLines.PreDeactivate
	dst.Data.LustreStorage.MgtMdtOptions.ColocateComputes = src.Data.LustreStorage.MgtMdtOptions.ColocateComputes
	dst.Data.LustreStorage.MgtMdtOptions.Count = src.Data.LustreStorage.MgtMdtOptions.Count
	dst.Data.LustreStorage.MgtMdtOptions.Scale = src.Data.LustreStorage.MgtMdtOptions.Scale
	dst.Data.LustreStorage.MgtMdtOptions.StorageLabels = src.Data.LustreStorage.MgtMdtOptions.StorageLabels

	// OST options
	dst.Data.LustreStorage.CapacityScalingFactor = src.Data.LustreStorage.OstOptions.CapacityScalingFactor
	dst.Data.LustreStorage.OstCmdLines.ZpoolCreate = src.Data.LustreStorage.OstOptions.CmdLines.ZpoolCreate
	dst.Data.LustreStorage.OstCmdLines.Mkfs = src.Data.LustreStorage.OstOptions.CmdLines.Mkfs
	dst.Data.LustreStorage.OstCmdLines.MountTarget = src.Data.LustreStorage.OstOptions.CmdLines.MountTarget
	dst.Data.LustreStorage.OstCmdLines.PostActivate = src.Data.LustreStorage.OstOptions.CmdLines.PostActivate
	dst.Data.LustreStorage.OstCmdLines.PreDeactivate = src.Data.LustreStorage.OstOptions.CmdLines.PreDeactivate
	dst.Data.LustreStorage.OstOptions.ColocateComputes = src.Data.LustreStorage.OstOptions.ColocateComputes
	dst.Data.LustreStorage.OstOptions.Count = src.Data.LustreStorage.OstOptions.Count
	dst.Data.LustreStorage.OstOptions.Scale = src.Data.LustreStorage.OstOptions.Scale
	dst.Data.LustreStorage.OstOptions.StorageLabels = src.Data.LustreStorage.OstOptions.StorageLabels

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

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfSystemStorage) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfSystemStorage To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha10.NnfSystemStorage)

	if err := Convert_v1alpha7_NnfSystemStorage_To_v1alpha10_NnfSystemStorage(src, dst, nil); err != nil {
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

	if err := Convert_v1alpha10_NnfSystemStorage_To_v1alpha7_NnfSystemStorage(src, dst, nil); err != nil {
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

func autoConvert_v1alpha4_ResourceError_To_v1alpha7_ResourceError(in *dwsv1alpha4.ResourceError, out *dwsv1alpha7.ResourceError, s apiconversion.Scope) error {
	out.Error = (*dwsv1alpha7.ResourceErrorInfo)(unsafe.Pointer(in.Error))
	return nil
}

// Convert_v1alpha4_ResourceError_To_v1alpha7_ResourceError is an autogenerated conversion function.
func Convert_v1alpha4_ResourceError_To_v1alpha7_ResourceError(in *dwsv1alpha4.ResourceError, out *dwsv1alpha7.ResourceError, s apiconversion.Scope) error {
	return autoConvert_v1alpha4_ResourceError_To_v1alpha7_ResourceError(in, out, s)
}

func autoConvert_v1alpha7_ResourceError_To_v1alpha4_ResourceError(in *dwsv1alpha7.ResourceError, out *dwsv1alpha4.ResourceError, s apiconversion.Scope) error {
	out.Error = (*dwsv1alpha4.ResourceErrorInfo)(unsafe.Pointer(in.Error))
	return nil
}

// Convert_4_ResourceError_To_v1alpha4_ResourceError is an autogenerated conversion function.
func Convert_v1alpha7_ResourceError_To_v1alpha4_ResourceError(in *dwsv1alpha7.ResourceError, out *dwsv1alpha4.ResourceError, s apiconversion.Scope) error {
	return autoConvert_v1alpha7_ResourceError_To_v1alpha4_ResourceError(in, out, s)
}

func autoConvert_v1alpha4_ResourceErrorInfo_To_v1alpha7_ResourceErrorInfo(in *dwsv1alpha4.ResourceErrorInfo, out *dwsv1alpha7.ResourceErrorInfo, s apiconversion.Scope) error {
	out.UserMessage = in.UserMessage
	out.DebugMessage = in.DebugMessage
	out.Type = dwsv1alpha7.ResourceErrorType(in.Type)
	out.Severity = dwsv1alpha7.ResourceErrorSeverity(in.Severity)
	return nil
}

// Convert_v1alpha4_ResourceErrorInfo_To_v1alpha7_ResourceErrorInfo is an autogenerated conversion function.
func Convert_v1alpha4_ResourceErrorInfo_To_v1alpha7_ResourceErrorInfo(in *dwsv1alpha4.ResourceErrorInfo, out *dwsv1alpha7.ResourceErrorInfo, s apiconversion.Scope) error {
	return autoConvert_v1alpha4_ResourceErrorInfo_To_v1alpha7_ResourceErrorInfo(in, out, s)
}

func autoConvert_v1alpha7_ResourceErrorInfo_To_v1alpha4_ResourceErrorInfo(in *dwsv1alpha7.ResourceErrorInfo, out *dwsv1alpha4.ResourceErrorInfo, s apiconversion.Scope) error {
	out.UserMessage = in.UserMessage
	out.DebugMessage = in.DebugMessage
	out.Type = dwsv1alpha4.ResourceErrorType(in.Type)
	out.Severity = dwsv1alpha4.ResourceErrorSeverity(in.Severity)
	return nil
}

// Convert_v1alpha7_ResourceErrorInfo_To_v1alpha4_ResourceErrorInfo is an autogenerated conversion function.
func Convert_v1alpha7_ResourceErrorInfo_To_v1alpha4_ResourceErrorInfo(in *dwsv1alpha7.ResourceErrorInfo, out *dwsv1alpha4.ResourceErrorInfo, s apiconversion.Scope) error {
	return autoConvert_v1alpha7_ResourceErrorInfo_To_v1alpha4_ResourceErrorInfo(in, out, s)
}

// End of DWS ResourceError conversion routines.
// +crdbumper:carryforward:end

// Convert_v1alpha10_NnfStorageProfileLustreCmdLines_To_v1alpha7_NnfStorageProfileLustreCmdLines is an autogenerated conversion function.
func Convert_v1alpha10_NnfStorageProfileLustreCmdLines_To_v1alpha7_NnfStorageProfileLustreCmdLines(in *nnfv1alpha10.NnfStorageProfileLustreCmdLines, out *NnfStorageProfileLustreCmdLines, s apiconversion.Scope) error {
	return autoConvert_v1alpha10_NnfStorageProfileLustreCmdLines_To_v1alpha7_NnfStorageProfileLustreCmdLines(in, out, s)
}

func Convert_v1alpha10_NnfNodeStorageAllocationStatus_To_v1alpha7_NnfNodeStorageAllocationStatus(in *nnfv1alpha10.NnfNodeStorageAllocationStatus, out *NnfNodeStorageAllocationStatus, s apiconversion.Scope) error {
	return autoConvert_v1alpha10_NnfNodeStorageAllocationStatus_To_v1alpha7_NnfNodeStorageAllocationStatus(in, out, s)
}

func Convert_v1alpha10_NnfNodeStorageStatus_To_v1alpha7_NnfNodeStorageStatus(in *nnfv1alpha10.NnfNodeStorageStatus, out *NnfNodeStorageStatus, s apiconversion.Scope) error {
	return autoConvert_v1alpha10_NnfNodeStorageStatus_To_v1alpha7_NnfNodeStorageStatus(in, out, s)
}

func Convert_v1alpha10_NnfStorageAllocationSetStatus_To_v1alpha7_NnfStorageAllocationSetStatus(in *nnfv1alpha10.NnfStorageAllocationSetStatus, out *NnfStorageAllocationSetStatus, s apiconversion.Scope) error {
	return autoConvert_v1alpha10_NnfStorageAllocationSetStatus_To_v1alpha7_NnfStorageAllocationSetStatus(in, out, s)
}

func Convert_v1alpha10_NnfStorageStatus_To_v1alpha7_NnfStorageStatus(in *nnfv1alpha10.NnfStorageStatus, out *NnfStorageStatus, s apiconversion.Scope) error {
	return autoConvert_v1alpha10_NnfStorageStatus_To_v1alpha7_NnfStorageStatus(in, out, s)
}

// Convert_v1alpha7_NnfStorageProfileGFS2Data_To_v1alpha10_NnfStorageProfileGFS2Data is an autogenerated conversion function.
func Convert_v1alpha7_NnfStorageProfileGFS2Data_To_v1alpha10_NnfStorageProfileGFS2Data(in *NnfStorageProfileGFS2Data, out *nnfv1alpha10.NnfStorageProfileGFS2Data, s apiconversion.Scope) error {
	return autoConvert_v1alpha7_NnfStorageProfileGFS2Data_To_v1alpha10_NnfStorageProfileGFS2Data(in, out, s)
}

// Convert_v1alpha10_NnfResourceStatus_To_v1alpha7_NnfResourceStatus converts NnfResourceStatus from hub to v1alpha7.
// This overrides the auto-generated function to map ResourceFenced to ResourceOffline.
func Convert_v1alpha10_NnfResourceStatus_To_v1alpha7_NnfResourceStatus(in *nnfv1alpha10.NnfResourceStatus, out *NnfResourceStatus, s apiconversion.Scope) error {
	out.ID = in.ID
	out.Name = in.Name
	out.Health = NnfResourceHealthType(in.Health)
	// Map ResourceFenced to ResourceOffline since v1alpha7 doesn't have Fenced
	if in.Status == nnfv1alpha10.ResourceFenced {
		out.Status = ResourceOffline
	} else {
		out.Status = NnfResourceStatusType(in.Status)
	}
	return nil
}

// Convert_v1alpha10_NnfStorageProfileGFS2Data_To_v1alpha7_NnfStorageProfileGFS2Data is an autogenerated conversion function.
func Convert_v1alpha10_NnfStorageProfileGFS2Data_To_v1alpha7_NnfStorageProfileGFS2Data(in *nnfv1alpha10.NnfStorageProfileGFS2Data, out *NnfStorageProfileGFS2Data, s apiconversion.Scope) error {
	return autoConvert_v1alpha10_NnfStorageProfileGFS2Data_To_v1alpha7_NnfStorageProfileGFS2Data(in, out, s)
}

// Convert_v1alpha10_NnfStorageProfileLustreClientCmdLines_To_v1alpha7_NnfStorageProfileLustreClientCmdLines is an autogenerated conversion function.
func Convert_v1alpha10_NnfStorageProfileLustreClientCmdLines_To_v1alpha7_NnfStorageProfileLustreClientCmdLines(in *nnfv1alpha10.NnfStorageProfileLustreClientCmdLines, out *NnfStorageProfileLustreClientCmdLines, s apiconversion.Scope) error {
	return autoConvert_v1alpha10_NnfStorageProfileLustreClientCmdLines_To_v1alpha7_NnfStorageProfileLustreClientCmdLines(in, out, s)
}

// Convert_v1alpha7_NnfStorageProfileRawData_To_v1alpha10_NnfStorageProfileRawData is an autogenerated conversion function.
func Convert_v1alpha7_NnfStorageProfileRawData_To_v1alpha10_NnfStorageProfileRawData(in *NnfStorageProfileRawData, out *nnfv1alpha10.NnfStorageProfileRawData, s apiconversion.Scope) error {
	return autoConvert_v1alpha7_NnfStorageProfileRawData_To_v1alpha10_NnfStorageProfileRawData(in, out, s)
}

// Convert_v1alpha10_NnfStorageProfileRawData_To_v1alpha7_NnfStorageProfileRawData is an autogenerated conversion function.
func Convert_v1alpha10_NnfStorageProfileRawData_To_v1alpha7_NnfStorageProfileRawData(in *nnfv1alpha10.NnfStorageProfileRawData, out *NnfStorageProfileRawData, s apiconversion.Scope) error {
	return autoConvert_v1alpha10_NnfStorageProfileRawData_To_v1alpha7_NnfStorageProfileRawData(in, out, s)
}

// Convert_v1alpha7_NnfStorageProfileXFSData_To_v1alpha10_NnfStorageProfileXFSData is an autogenerated conversion function.
func Convert_v1alpha7_NnfStorageProfileXFSData_To_v1alpha10_NnfStorageProfileXFSData(in *NnfStorageProfileXFSData, out *nnfv1alpha10.NnfStorageProfileXFSData, s apiconversion.Scope) error {
	return autoConvert_v1alpha7_NnfStorageProfileXFSData_To_v1alpha10_NnfStorageProfileXFSData(in, out, s)
}

// Convert_v1alpha10_NnfStorageProfileXFSData_To_v1alpha7_NnfStorageProfileXFSData is an autogenerated conversion function.
func Convert_v1alpha10_NnfStorageProfileXFSData_To_v1alpha7_NnfStorageProfileXFSData(in *nnfv1alpha10.NnfStorageProfileXFSData, out *NnfStorageProfileXFSData, s apiconversion.Scope) error {
	return autoConvert_v1alpha10_NnfStorageProfileXFSData_To_v1alpha7_NnfStorageProfileXFSData(in, out, s)
}

// Convert_v1alpha7_NnfStorageProfileLustreData_To_v1alpha10_NnfStorageProfileLustreData handles conversion from v1alpha7 to v1alpha10.
// This is a manual conversion because the types have different structures.
func Convert_v1alpha7_NnfStorageProfileLustreData_To_v1alpha10_NnfStorageProfileLustreData(in *NnfStorageProfileLustreData, out *nnfv1alpha10.NnfStorageProfileLustreData, s apiconversion.Scope) error {
	out.CombinedMGTMDT = in.CombinedMGTMDT

	// MGT options
	out.MgtOptions.ExternalMGS = in.ExternalMGS
	out.MgtOptions.StandaloneMGTPoolName = in.StandaloneMGTPoolName
	out.MgtOptions.Capacity = in.CapacityMGT
	out.MgtOptions.CmdLines.ZpoolCreate = in.MgtCmdLines.ZpoolCreate
	out.MgtOptions.CmdLines.Mkfs = in.MgtCmdLines.Mkfs
	out.MgtOptions.CmdLines.MountTarget = in.MgtCmdLines.MountTarget
	out.MgtOptions.CmdLines.PostActivate = in.MgtCmdLines.PostActivate
	out.MgtOptions.CmdLines.PreDeactivate = in.MgtCmdLines.PreDeactivate
	out.MgtOptions.ColocateComputes = in.MgtOptions.ColocateComputes
	out.MgtOptions.Count = in.MgtOptions.Count
	out.MgtOptions.Scale = in.MgtOptions.Scale
	out.MgtOptions.StorageLabels = in.MgtOptions.StorageLabels
	out.MgtOptions.PreMountCommands = in.PreMountMGTCmds

	// MDT options
	out.MdtOptions.Capacity = in.CapacityMDT
	out.MdtOptions.Exclusive = in.ExclusiveMDT
	out.MdtOptions.CmdLines.ZpoolCreate = in.MdtCmdLines.ZpoolCreate
	out.MdtOptions.CmdLines.Mkfs = in.MdtCmdLines.Mkfs
	out.MdtOptions.CmdLines.MountTarget = in.MdtCmdLines.MountTarget
	out.MdtOptions.CmdLines.PostActivate = in.MdtCmdLines.PostActivate
	out.MdtOptions.CmdLines.PreDeactivate = in.MdtCmdLines.PreDeactivate
	out.MdtOptions.ColocateComputes = in.MdtOptions.ColocateComputes
	out.MdtOptions.Count = in.MdtOptions.Count
	out.MdtOptions.Scale = in.MdtOptions.Scale
	out.MdtOptions.StorageLabels = in.MdtOptions.StorageLabels

	// MGT/MDT combined options
	out.MgtMdtOptions.Capacity = in.CapacityMDT
	out.MgtMdtOptions.CmdLines.ZpoolCreate = in.MgtMdtCmdLines.ZpoolCreate
	out.MgtMdtOptions.CmdLines.Mkfs = in.MgtMdtCmdLines.Mkfs
	out.MgtMdtOptions.CmdLines.MountTarget = in.MgtMdtCmdLines.MountTarget
	out.MgtMdtOptions.CmdLines.PostActivate = in.MgtMdtCmdLines.PostActivate
	out.MgtMdtOptions.CmdLines.PreDeactivate = in.MgtMdtCmdLines.PreDeactivate
	out.MgtMdtOptions.ColocateComputes = in.MgtMdtOptions.ColocateComputes
	out.MgtMdtOptions.Count = in.MgtMdtOptions.Count
	out.MgtMdtOptions.Scale = in.MgtMdtOptions.Scale
	out.MgtMdtOptions.StorageLabels = in.MgtMdtOptions.StorageLabels

	// OST options
	out.OstOptions.CapacityScalingFactor = in.CapacityScalingFactor
	out.OstOptions.CmdLines.ZpoolCreate = in.OstCmdLines.ZpoolCreate
	out.OstOptions.CmdLines.Mkfs = in.OstCmdLines.Mkfs
	out.OstOptions.CmdLines.MountTarget = in.OstCmdLines.MountTarget
	out.OstOptions.CmdLines.PostActivate = in.OstCmdLines.PostActivate
	out.OstOptions.CmdLines.PreDeactivate = in.OstCmdLines.PreDeactivate
	out.OstOptions.ColocateComputes = in.OstOptions.ColocateComputes
	out.OstOptions.Count = in.OstOptions.Count
	out.OstOptions.Scale = in.OstOptions.Scale
	out.OstOptions.StorageLabels = in.OstOptions.StorageLabels

	// Client options
	out.ClientOptions.CmdLines.MountRabbit = in.ClientCmdLines.MountRabbit
	out.ClientOptions.CmdLines.MountCompute = in.ClientCmdLines.MountCompute
	out.ClientOptions.CmdLines.RabbitPostSetup = in.ClientCmdLines.RabbitPostMount
	out.ClientOptions.CmdLines.RabbitPreTeardown = in.ClientCmdLines.RabbitPreUnmount

	return nil
}

// Convert_v1alpha10_NnfStorageProfileLustreData_To_v1alpha7_NnfStorageProfileLustreData handles conversion from v1alpha10 to v1alpha7.
// This is a manual conversion because the types have different structures.
func Convert_v1alpha10_NnfStorageProfileLustreData_To_v1alpha7_NnfStorageProfileLustreData(in *nnfv1alpha10.NnfStorageProfileLustreData, out *NnfStorageProfileLustreData, s apiconversion.Scope) error {
	out.CombinedMGTMDT = in.CombinedMGTMDT

	// MGT options
	out.ExternalMGS = in.MgtOptions.ExternalMGS
	out.StandaloneMGTPoolName = in.MgtOptions.StandaloneMGTPoolName
	out.CapacityMGT = in.MgtOptions.Capacity
	out.MgtCmdLines.ZpoolCreate = in.MgtOptions.CmdLines.ZpoolCreate
	// ZpoolReplace is lost during conversion from v1alpha10 to v1alpha7
	out.MgtCmdLines.Mkfs = in.MgtOptions.CmdLines.Mkfs
	out.MgtCmdLines.MountTarget = in.MgtOptions.CmdLines.MountTarget
	out.MgtCmdLines.PostActivate = in.MgtOptions.CmdLines.PostActivate
	out.MgtCmdLines.PreDeactivate = in.MgtOptions.CmdLines.PreDeactivate
	out.MgtOptions.ColocateComputes = in.MgtOptions.ColocateComputes
	out.MgtOptions.Count = in.MgtOptions.Count
	out.MgtOptions.Scale = in.MgtOptions.Scale
	out.MgtOptions.StorageLabels = in.MgtOptions.StorageLabels
	out.PreMountMGTCmds = in.MgtOptions.PreMountCommands

	// MDT options
	out.CapacityMDT = in.MdtOptions.Capacity
	out.ExclusiveMDT = in.MdtOptions.Exclusive
	out.MdtCmdLines.ZpoolCreate = in.MdtOptions.CmdLines.ZpoolCreate
	// ZpoolReplace is lost during conversion from v1alpha10 to v1alpha7
	out.MdtCmdLines.Mkfs = in.MdtOptions.CmdLines.Mkfs
	out.MdtCmdLines.MountTarget = in.MdtOptions.CmdLines.MountTarget
	out.MdtCmdLines.PostActivate = in.MdtOptions.CmdLines.PostActivate
	out.MdtCmdLines.PreDeactivate = in.MdtOptions.CmdLines.PreDeactivate
	out.MdtOptions.ColocateComputes = in.MdtOptions.ColocateComputes
	out.MdtOptions.Count = in.MdtOptions.Count
	out.MdtOptions.Scale = in.MdtOptions.Scale
	out.MdtOptions.StorageLabels = in.MdtOptions.StorageLabels

	// MGT/MDT combined options
	out.MgtMdtCmdLines.ZpoolCreate = in.MgtMdtOptions.CmdLines.ZpoolCreate
	// ZpoolReplace is lost during conversion from v1alpha10 to v1alpha7
	out.MgtMdtCmdLines.Mkfs = in.MgtMdtOptions.CmdLines.Mkfs
	out.MgtMdtCmdLines.MountTarget = in.MgtMdtOptions.CmdLines.MountTarget
	out.MgtMdtCmdLines.PostActivate = in.MgtMdtOptions.CmdLines.PostActivate
	out.MgtMdtCmdLines.PreDeactivate = in.MgtMdtOptions.CmdLines.PreDeactivate
	out.MgtMdtOptions.ColocateComputes = in.MgtMdtOptions.ColocateComputes
	out.MgtMdtOptions.Count = in.MgtMdtOptions.Count
	out.MgtMdtOptions.Scale = in.MgtMdtOptions.Scale
	out.MgtMdtOptions.StorageLabels = in.MgtMdtOptions.StorageLabels

	// OST options
	out.CapacityScalingFactor = in.OstOptions.CapacityScalingFactor
	out.OstCmdLines.ZpoolCreate = in.OstOptions.CmdLines.ZpoolCreate
	// ZpoolReplace is lost during conversion from v1alpha10 to v1alpha7
	out.OstCmdLines.Mkfs = in.OstOptions.CmdLines.Mkfs
	out.OstCmdLines.MountTarget = in.OstOptions.CmdLines.MountTarget
	out.OstCmdLines.PostActivate = in.OstOptions.CmdLines.PostActivate
	out.OstCmdLines.PreDeactivate = in.OstOptions.CmdLines.PreDeactivate
	out.OstOptions.ColocateComputes = in.OstOptions.ColocateComputes
	out.OstOptions.Count = in.OstOptions.Count
	out.OstOptions.Scale = in.OstOptions.Scale
	out.OstOptions.StorageLabels = in.OstOptions.StorageLabels

	// Client options
	out.ClientCmdLines.MountRabbit = in.ClientOptions.CmdLines.MountRabbit
	out.ClientCmdLines.MountCompute = in.ClientOptions.CmdLines.MountCompute
	out.ClientCmdLines.RabbitPostMount = in.ClientOptions.CmdLines.RabbitPostSetup
	out.ClientCmdLines.RabbitPreUnmount = in.ClientOptions.CmdLines.RabbitPreTeardown

	return nil
}

// Convert_v1alpha7_NnfStorageProfileLustreMiscOptions_To_v1alpha10_NnfStorageProfileLustreMgtOptions handles conversion.
func Convert_v1alpha7_NnfStorageProfileLustreMiscOptions_To_v1alpha10_NnfStorageProfileLustreMgtOptions(in *NnfStorageProfileLustreMiscOptions, out *nnfv1alpha10.NnfStorageProfileLustreMgtOptions, s apiconversion.Scope) error {
	out.ColocateComputes = in.ColocateComputes
	out.Count = in.Count
	out.Scale = in.Scale
	out.StorageLabels = in.StorageLabels
	return nil
}

// Convert_v1alpha7_NnfStorageProfileLustreMiscOptions_To_v1alpha10_NnfStorageProfileLustreMdtOptions handles conversion.
func Convert_v1alpha7_NnfStorageProfileLustreMiscOptions_To_v1alpha10_NnfStorageProfileLustreMdtOptions(in *NnfStorageProfileLustreMiscOptions, out *nnfv1alpha10.NnfStorageProfileLustreMdtOptions, s apiconversion.Scope) error {
	out.ColocateComputes = in.ColocateComputes
	out.Count = in.Count
	out.Scale = in.Scale
	out.StorageLabels = in.StorageLabels
	return nil
}

// Convert_v1alpha7_NnfStorageProfileLustreMiscOptions_To_v1alpha10_NnfStorageProfileLustreMgtMdtOptions handles conversion.
func Convert_v1alpha7_NnfStorageProfileLustreMiscOptions_To_v1alpha10_NnfStorageProfileLustreMgtMdtOptions(in *NnfStorageProfileLustreMiscOptions, out *nnfv1alpha10.NnfStorageProfileLustreMgtMdtOptions, s apiconversion.Scope) error {
	out.ColocateComputes = in.ColocateComputes
	out.Count = in.Count
	out.Scale = in.Scale
	out.StorageLabels = in.StorageLabels
	return nil
}

// Convert_v1alpha7_NnfStorageProfileLustreMiscOptions_To_v1alpha10_NnfStorageProfileLustreOstOptions handles conversion.
func Convert_v1alpha7_NnfStorageProfileLustreMiscOptions_To_v1alpha10_NnfStorageProfileLustreOstOptions(in *NnfStorageProfileLustreMiscOptions, out *nnfv1alpha10.NnfStorageProfileLustreOstOptions, s apiconversion.Scope) error {
	out.ColocateComputes = in.ColocateComputes
	out.Count = in.Count
	out.Scale = in.Scale
	out.StorageLabels = in.StorageLabels
	return nil
}

// Convert_v1alpha10_NnfStorageProfileLustreMgtOptions_To_v1alpha7_NnfStorageProfileLustreMiscOptions handles conversion.
func Convert_v1alpha10_NnfStorageProfileLustreMgtOptions_To_v1alpha7_NnfStorageProfileLustreMiscOptions(in *nnfv1alpha10.NnfStorageProfileLustreMgtOptions, out *NnfStorageProfileLustreMiscOptions, s apiconversion.Scope) error {
	out.ColocateComputes = in.ColocateComputes
	out.Count = in.Count
	out.Scale = in.Scale
	out.StorageLabels = in.StorageLabels
	return nil
}

// Convert_v1alpha10_NnfStorageProfileLustreMdtOptions_To_v1alpha7_NnfStorageProfileLustreMiscOptions handles conversion.
func Convert_v1alpha10_NnfStorageProfileLustreMdtOptions_To_v1alpha7_NnfStorageProfileLustreMiscOptions(in *nnfv1alpha10.NnfStorageProfileLustreMdtOptions, out *NnfStorageProfileLustreMiscOptions, s apiconversion.Scope) error {
	out.ColocateComputes = in.ColocateComputes
	out.Count = in.Count
	out.Scale = in.Scale
	out.StorageLabels = in.StorageLabels
	return nil
}

// Convert_v1alpha10_NnfStorageProfileLustreMgtMdtOptions_To_v1alpha7_NnfStorageProfileLustreMiscOptions handles conversion.
func Convert_v1alpha10_NnfStorageProfileLustreMgtMdtOptions_To_v1alpha7_NnfStorageProfileLustreMiscOptions(in *nnfv1alpha10.NnfStorageProfileLustreMgtMdtOptions, out *NnfStorageProfileLustreMiscOptions, s apiconversion.Scope) error {
	out.ColocateComputes = in.ColocateComputes
	out.Count = in.Count
	out.Scale = in.Scale
	out.StorageLabels = in.StorageLabels
	return nil
}

// Convert_v1alpha10_NnfStorageProfileLustreOstOptions_To_v1alpha7_NnfStorageProfileLustreMiscOptions handles conversion.
func Convert_v1alpha10_NnfStorageProfileLustreOstOptions_To_v1alpha7_NnfStorageProfileLustreMiscOptions(in *nnfv1alpha10.NnfStorageProfileLustreOstOptions, out *NnfStorageProfileLustreMiscOptions, s apiconversion.Scope) error {
	out.ColocateComputes = in.ColocateComputes
	out.Count = in.Count
	out.Scale = in.Scale
	out.StorageLabels = in.StorageLabels
	return nil
}
