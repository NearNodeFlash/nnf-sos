/*
 * Copyright 2025-2026 Hewlett Packard Enterprise Development LP
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
	"unsafe"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apiconversion "k8s.io/apimachinery/pkg/conversion"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	nnfv1alpha11 "github.com/NearNodeFlash/nnf-sos/api/v1alpha11"
	utilconversion "github.com/NearNodeFlash/nnf-sos/github/cluster-api/util/conversion"
)

var convertlog = logf.Log.V(2).WithName("convert-v1alpha9")

func (src *NnfAccess) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfAccess To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha11.NnfAccess)

	if err := Convert_v1alpha9_NnfAccess_To_v1alpha11_NnfAccess(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha11.NnfAccess{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	return nil
}

func (dst *NnfAccess) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha11.NnfAccess)
	convertlog.Info("Convert NnfAccess From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha11_NnfAccess_To_v1alpha9_NnfAccess(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfContainerProfile) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfContainerProfile To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha11.NnfContainerProfile)

	if err := Convert_v1alpha9_NnfContainerProfile_To_v1alpha11_NnfContainerProfile(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha11.NnfContainerProfile{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	return nil
}

func (dst *NnfContainerProfile) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha11.NnfContainerProfile)
	convertlog.Info("Convert NnfContainerProfile From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha11_NnfContainerProfile_To_v1alpha9_NnfContainerProfile(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfDataMovement) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfDataMovement To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha11.NnfDataMovement)

	if err := Convert_v1alpha9_NnfDataMovement_To_v1alpha11_NnfDataMovement(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha11.NnfDataMovement{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	return nil
}

func (dst *NnfDataMovement) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha11.NnfDataMovement)
	convertlog.Info("Convert NnfDataMovement From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha11_NnfDataMovement_To_v1alpha9_NnfDataMovement(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfDataMovementManager) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfDataMovementManager To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha11.NnfDataMovementManager)

	if err := Convert_v1alpha9_NnfDataMovementManager_To_v1alpha11_NnfDataMovementManager(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha11.NnfDataMovementManager{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	return nil
}

func (dst *NnfDataMovementManager) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha11.NnfDataMovementManager)
	convertlog.Info("Convert NnfDataMovementManager From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha11_NnfDataMovementManager_To_v1alpha9_NnfDataMovementManager(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfDataMovementProfile) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfDataMovementProfile To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha11.NnfDataMovementProfile)

	if err := Convert_v1alpha9_NnfDataMovementProfile_To_v1alpha11_NnfDataMovementProfile(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha11.NnfDataMovementProfile{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	return nil
}

func (dst *NnfDataMovementProfile) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha11.NnfDataMovementProfile)
	convertlog.Info("Convert NnfDataMovementProfile From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha11_NnfDataMovementProfile_To_v1alpha9_NnfDataMovementProfile(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfLustreMGT) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfLustreMGT To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha11.NnfLustreMGT)

	if err := Convert_v1alpha9_NnfLustreMGT_To_v1alpha11_NnfLustreMGT(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha11.NnfLustreMGT{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	return nil
}

func (dst *NnfLustreMGT) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha11.NnfLustreMGT)
	convertlog.Info("Convert NnfLustreMGT From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha11_NnfLustreMGT_To_v1alpha9_NnfLustreMGT(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfNode) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfNode To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha11.NnfNode)

	if err := Convert_v1alpha9_NnfNode_To_v1alpha11_NnfNode(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha11.NnfNode{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	return nil
}

func (dst *NnfNode) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha11.NnfNode)
	convertlog.Info("Convert NnfNode From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha11_NnfNode_To_v1alpha9_NnfNode(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfNodeBlockStorage) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfNodeBlockStorage To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha11.NnfNodeBlockStorage)

	if err := Convert_v1alpha9_NnfNodeBlockStorage_To_v1alpha11_NnfNodeBlockStorage(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha11.NnfNodeBlockStorage{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	return nil
}

func (dst *NnfNodeBlockStorage) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha11.NnfNodeBlockStorage)
	convertlog.Info("Convert NnfNodeBlockStorage From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha11_NnfNodeBlockStorage_To_v1alpha9_NnfNodeBlockStorage(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfNodeECData) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfNodeECData To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha11.NnfNodeECData)

	if err := Convert_v1alpha9_NnfNodeECData_To_v1alpha11_NnfNodeECData(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha11.NnfNodeECData{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	return nil
}

func (dst *NnfNodeECData) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha11.NnfNodeECData)
	convertlog.Info("Convert NnfNodeECData From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha11_NnfNodeECData_To_v1alpha9_NnfNodeECData(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfNodeStorage) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfNodeStorage To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha11.NnfNodeStorage)

	if err := Convert_v1alpha9_NnfNodeStorage_To_v1alpha11_NnfNodeStorage(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha11.NnfNodeStorage{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	return nil
}

func (dst *NnfNodeStorage) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha11.NnfNodeStorage)
	convertlog.Info("Convert NnfNodeStorage From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha11_NnfNodeStorage_To_v1alpha9_NnfNodeStorage(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfPortManager) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfPortManager To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha11.NnfPortManager)

	if err := Convert_v1alpha9_NnfPortManager_To_v1alpha11_NnfPortManager(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha11.NnfPortManager{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	return nil
}

func (dst *NnfPortManager) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha11.NnfPortManager)
	convertlog.Info("Convert NnfPortManager From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha11_NnfPortManager_To_v1alpha9_NnfPortManager(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfStorage) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfStorage To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha11.NnfStorage)

	if err := Convert_v1alpha9_NnfStorage_To_v1alpha11_NnfStorage(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha11.NnfStorage{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	return nil
}

func (dst *NnfStorage) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha11.NnfStorage)
	convertlog.Info("Convert NnfStorage From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha11_NnfStorage_To_v1alpha9_NnfStorage(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfStorageProfile) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfStorageProfile To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha11.NnfStorageProfile)

	if err := Convert_v1alpha9_NnfStorageProfile_To_v1alpha11_NnfStorageProfile(src, dst, nil); err != nil {
		return err
	}

	// Manual conversion for Lustre data due to structural differences
	dst.Data.LustreStorage.CombinedMGTMDT = src.Data.LustreStorage.CombinedMGTMDT

	// MGT options
	dst.Data.LustreStorage.MgtOptions.ExternalMGS = src.Data.LustreStorage.ExternalMGS
	dst.Data.LustreStorage.MgtOptions.StandaloneMGTPoolName = src.Data.LustreStorage.StandaloneMGTPoolName
	dst.Data.LustreStorage.MgtOptions.Capacity = src.Data.LustreStorage.CapacityMGT
	dst.Data.LustreStorage.MgtOptions.CmdLines.ZpoolCreate = src.Data.LustreStorage.MgtCmdLines.ZpoolCreate
	dst.Data.LustreStorage.MgtOptions.CmdLines.ZpoolReplace = src.Data.LustreStorage.MgtCmdLines.ZpoolReplace
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
	dst.Data.LustreStorage.MdtOptions.CmdLines.ZpoolReplace = src.Data.LustreStorage.MdtCmdLines.ZpoolReplace
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
	dst.Data.LustreStorage.MgtMdtOptions.CmdLines.ZpoolReplace = src.Data.LustreStorage.MgtMdtCmdLines.ZpoolReplace
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
	dst.Data.LustreStorage.OstOptions.CmdLines.ZpoolReplace = src.Data.LustreStorage.OstCmdLines.ZpoolReplace
	dst.Data.LustreStorage.OstOptions.CmdLines.Mkfs = src.Data.LustreStorage.OstCmdLines.Mkfs
	dst.Data.LustreStorage.OstOptions.CmdLines.MountTarget = src.Data.LustreStorage.OstCmdLines.MountTarget
	dst.Data.LustreStorage.OstOptions.CmdLines.PostActivate = src.Data.LustreStorage.OstCmdLines.PostActivate
	dst.Data.LustreStorage.OstOptions.CmdLines.PreDeactivate = src.Data.LustreStorage.OstCmdLines.PreDeactivate
	dst.Data.LustreStorage.OstOptions.ColocateComputes = src.Data.LustreStorage.OstOptions.ColocateComputes
	dst.Data.LustreStorage.OstOptions.Count = src.Data.LustreStorage.OstOptions.Count
	dst.Data.LustreStorage.OstOptions.Scale = src.Data.LustreStorage.OstOptions.Scale
	dst.Data.LustreStorage.OstOptions.StorageLabels = src.Data.LustreStorage.OstOptions.StorageLabels

	// Client options
	dst.Data.LustreStorage.ClientOptions.CmdLines.MountRabbit = src.Data.LustreStorage.ClientCmdLines.MountRabbit
	dst.Data.LustreStorage.ClientOptions.CmdLines.MountCompute = src.Data.LustreStorage.ClientCmdLines.MountCompute
	dst.Data.LustreStorage.ClientOptions.CmdLines.RabbitPostSetup = src.Data.LustreStorage.ClientCmdLines.RabbitPostSetup
	dst.Data.LustreStorage.ClientOptions.CmdLines.RabbitPreTeardown = src.Data.LustreStorage.ClientCmdLines.RabbitPreTeardown
	dst.Data.LustreStorage.ClientOptions.CmdLines.RabbitPreMount = src.Data.LustreStorage.ClientCmdLines.RabbitPreMount
	dst.Data.LustreStorage.ClientOptions.CmdLines.RabbitPostMount = src.Data.LustreStorage.ClientCmdLines.RabbitPostMount
	dst.Data.LustreStorage.ClientOptions.CmdLines.RabbitPreUnmount = src.Data.LustreStorage.ClientCmdLines.RabbitPreUnmount
	dst.Data.LustreStorage.ClientOptions.CmdLines.RabbitPostUnmount = src.Data.LustreStorage.ClientCmdLines.RabbitPostUnmount
	dst.Data.LustreStorage.ClientOptions.CmdLines.ComputePreMount = src.Data.LustreStorage.ClientCmdLines.ComputePreMount
	dst.Data.LustreStorage.ClientOptions.CmdLines.ComputePostMount = src.Data.LustreStorage.ClientCmdLines.ComputePostMount
	dst.Data.LustreStorage.ClientOptions.CmdLines.ComputePreUnmount = src.Data.LustreStorage.ClientCmdLines.ComputePreUnmount
	dst.Data.LustreStorage.ClientOptions.CmdLines.ComputePostUnmount = src.Data.LustreStorage.ClientCmdLines.ComputePostUnmount

	// Manually restore data.
	restored := &nnfv1alpha11.NnfStorageProfile{}
	hasAnno, err := utilconversion.UnmarshalData(src, restored)
	if err != nil {
		return err
	}

	if hasAnno {
		// Restore hub-specific fields from annotation
		dst.Data.LustreStorage.MgtMdtOptions.Capacity = restored.Data.LustreStorage.MgtMdtOptions.Capacity
		dst.Data.LustreStorage.MgtOptions.VariableOverride = restored.Data.LustreStorage.MgtOptions.VariableOverride
		dst.Data.LustreStorage.MgtMdtOptions.VariableOverride = restored.Data.LustreStorage.MgtMdtOptions.VariableOverride
		dst.Data.LustreStorage.MdtOptions.VariableOverride = restored.Data.LustreStorage.MdtOptions.VariableOverride
		dst.Data.LustreStorage.OstOptions.VariableOverride = restored.Data.LustreStorage.OstOptions.VariableOverride
		dst.Data.LustreStorage.ClientOptions.VariableOverride = restored.Data.LustreStorage.ClientOptions.VariableOverride
		dst.Data.GFS2Storage.VariableOverride = restored.Data.GFS2Storage.VariableOverride
		dst.Data.XFSStorage.VariableOverride = restored.Data.XFSStorage.VariableOverride
		dst.Data.RawStorage.VariableOverride = restored.Data.RawStorage.VariableOverride

		// Restore unmount fields
		dst.Data.LustreStorage.MgtOptions.CmdLines.UnmountTarget = restored.Data.LustreStorage.MgtOptions.CmdLines.UnmountTarget
		dst.Data.LustreStorage.MgtMdtOptions.CmdLines.UnmountTarget = restored.Data.LustreStorage.MgtMdtOptions.CmdLines.UnmountTarget
		dst.Data.LustreStorage.MdtOptions.CmdLines.UnmountTarget = restored.Data.LustreStorage.MdtOptions.CmdLines.UnmountTarget
		dst.Data.LustreStorage.OstOptions.CmdLines.UnmountTarget = restored.Data.LustreStorage.OstOptions.CmdLines.UnmountTarget
		dst.Data.LustreStorage.ClientOptions.CmdLines.UnmountRabbit = restored.Data.LustreStorage.ClientOptions.CmdLines.UnmountRabbit
		dst.Data.LustreStorage.ClientOptions.CmdLines.UnmountCompute = restored.Data.LustreStorage.ClientOptions.CmdLines.UnmountCompute

		// Restore ZpoolDestroy fields
		dst.Data.LustreStorage.MgtOptions.CmdLines.ZpoolDestroy = restored.Data.LustreStorage.MgtOptions.CmdLines.ZpoolDestroy
		dst.Data.LustreStorage.MgtMdtOptions.CmdLines.ZpoolDestroy = restored.Data.LustreStorage.MgtMdtOptions.CmdLines.ZpoolDestroy
		dst.Data.LustreStorage.MdtOptions.CmdLines.ZpoolDestroy = restored.Data.LustreStorage.MdtOptions.CmdLines.ZpoolDestroy
		dst.Data.LustreStorage.OstOptions.CmdLines.ZpoolDestroy = restored.Data.LustreStorage.OstOptions.CmdLines.ZpoolDestroy

		dst.Data.GFS2Storage.FileSystemCommands.RabbitCommands.Unmount = restored.Data.GFS2Storage.FileSystemCommands.RabbitCommands.Unmount
		dst.Data.GFS2Storage.FileSystemCommands.ComputeCommands.Unmount = restored.Data.GFS2Storage.FileSystemCommands.ComputeCommands.Unmount
		dst.Data.XFSStorage.FileSystemCommands.RabbitCommands.Unmount = restored.Data.XFSStorage.FileSystemCommands.RabbitCommands.Unmount
		dst.Data.XFSStorage.FileSystemCommands.ComputeCommands.Unmount = restored.Data.XFSStorage.FileSystemCommands.ComputeCommands.Unmount
		dst.Data.RawStorage.FileSystemCommands.RabbitCommands.Unmount = restored.Data.RawStorage.FileSystemCommands.RabbitCommands.Unmount
		dst.Data.RawStorage.FileSystemCommands.ComputeCommands.Unmount = restored.Data.RawStorage.FileSystemCommands.ComputeCommands.Unmount

		// Restore hub-specific PostTeardown fields
		dst.Data.GFS2Storage.BlockDeviceCommands.RabbitCommands.UserCommands.PostTeardown = restored.Data.GFS2Storage.BlockDeviceCommands.RabbitCommands.UserCommands.PostTeardown
		dst.Data.GFS2Storage.BlockDeviceCommands.ComputeCommands.UserCommands.PostTeardown = restored.Data.GFS2Storage.BlockDeviceCommands.ComputeCommands.UserCommands.PostTeardown
		dst.Data.XFSStorage.BlockDeviceCommands.RabbitCommands.UserCommands.PostTeardown = restored.Data.XFSStorage.BlockDeviceCommands.RabbitCommands.UserCommands.PostTeardown
		dst.Data.XFSStorage.BlockDeviceCommands.ComputeCommands.UserCommands.PostTeardown = restored.Data.XFSStorage.BlockDeviceCommands.ComputeCommands.UserCommands.PostTeardown
		dst.Data.RawStorage.BlockDeviceCommands.RabbitCommands.UserCommands.PostTeardown = restored.Data.RawStorage.BlockDeviceCommands.RabbitCommands.UserCommands.PostTeardown
		dst.Data.RawStorage.BlockDeviceCommands.ComputeCommands.UserCommands.PostTeardown = restored.Data.RawStorage.BlockDeviceCommands.ComputeCommands.UserCommands.PostTeardown
	}

	return nil
}

func (dst *NnfStorageProfile) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha11.NnfStorageProfile)
	convertlog.Info("Convert NnfStorageProfile From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha11_NnfStorageProfile_To_v1alpha9_NnfStorageProfile(src, dst, nil); err != nil {
		return err
	}

	// Manual conversion for Lustre data due to structural differences
	dst.Data.LustreStorage.CombinedMGTMDT = src.Data.LustreStorage.CombinedMGTMDT

	// MGT options
	dst.Data.LustreStorage.ExternalMGS = src.Data.LustreStorage.MgtOptions.ExternalMGS
	dst.Data.LustreStorage.StandaloneMGTPoolName = src.Data.LustreStorage.MgtOptions.StandaloneMGTPoolName
	dst.Data.LustreStorage.CapacityMGT = src.Data.LustreStorage.MgtOptions.Capacity
	dst.Data.LustreStorage.MgtCmdLines.ZpoolCreate = src.Data.LustreStorage.MgtOptions.CmdLines.ZpoolCreate
	dst.Data.LustreStorage.MgtCmdLines.ZpoolReplace = src.Data.LustreStorage.MgtOptions.CmdLines.ZpoolReplace
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
	dst.Data.LustreStorage.MdtCmdLines.ZpoolReplace = src.Data.LustreStorage.MdtOptions.CmdLines.ZpoolReplace
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
	dst.Data.LustreStorage.MgtMdtCmdLines.ZpoolReplace = src.Data.LustreStorage.MgtMdtOptions.CmdLines.ZpoolReplace
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
	dst.Data.LustreStorage.OstCmdLines.ZpoolReplace = src.Data.LustreStorage.OstOptions.CmdLines.ZpoolReplace
	dst.Data.LustreStorage.OstCmdLines.Mkfs = src.Data.LustreStorage.OstOptions.CmdLines.Mkfs
	dst.Data.LustreStorage.OstCmdLines.MountTarget = src.Data.LustreStorage.OstOptions.CmdLines.MountTarget
	dst.Data.LustreStorage.OstCmdLines.PostActivate = src.Data.LustreStorage.OstOptions.CmdLines.PostActivate
	dst.Data.LustreStorage.OstCmdLines.PreDeactivate = src.Data.LustreStorage.OstOptions.CmdLines.PreDeactivate
	dst.Data.LustreStorage.OstOptions.ColocateComputes = src.Data.LustreStorage.OstOptions.ColocateComputes
	dst.Data.LustreStorage.OstOptions.Count = src.Data.LustreStorage.OstOptions.Count
	dst.Data.LustreStorage.OstOptions.Scale = src.Data.LustreStorage.OstOptions.Scale
	dst.Data.LustreStorage.OstOptions.StorageLabels = src.Data.LustreStorage.OstOptions.StorageLabels

	// Client options
	dst.Data.LustreStorage.ClientCmdLines.MountRabbit = src.Data.LustreStorage.ClientOptions.CmdLines.MountRabbit
	dst.Data.LustreStorage.ClientCmdLines.MountCompute = src.Data.LustreStorage.ClientOptions.CmdLines.MountCompute
	dst.Data.LustreStorage.ClientCmdLines.RabbitPostSetup = src.Data.LustreStorage.ClientOptions.CmdLines.RabbitPostSetup
	dst.Data.LustreStorage.ClientCmdLines.RabbitPreTeardown = src.Data.LustreStorage.ClientOptions.CmdLines.RabbitPreTeardown
	dst.Data.LustreStorage.ClientCmdLines.RabbitPreMount = src.Data.LustreStorage.ClientOptions.CmdLines.RabbitPreMount
	dst.Data.LustreStorage.ClientCmdLines.RabbitPostMount = src.Data.LustreStorage.ClientOptions.CmdLines.RabbitPostMount
	dst.Data.LustreStorage.ClientCmdLines.RabbitPreUnmount = src.Data.LustreStorage.ClientOptions.CmdLines.RabbitPreUnmount
	dst.Data.LustreStorage.ClientCmdLines.RabbitPostUnmount = src.Data.LustreStorage.ClientOptions.CmdLines.RabbitPostUnmount
	dst.Data.LustreStorage.ClientCmdLines.ComputePreMount = src.Data.LustreStorage.ClientOptions.CmdLines.ComputePreMount
	dst.Data.LustreStorage.ClientCmdLines.ComputePostMount = src.Data.LustreStorage.ClientOptions.CmdLines.ComputePostMount
	dst.Data.LustreStorage.ClientCmdLines.ComputePreUnmount = src.Data.LustreStorage.ClientOptions.CmdLines.ComputePreUnmount
	dst.Data.LustreStorage.ClientCmdLines.ComputePostUnmount = src.Data.LustreStorage.ClientOptions.CmdLines.ComputePostUnmount

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfSystemStorage) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfSystemStorage To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha11.NnfSystemStorage)

	if err := Convert_v1alpha9_NnfSystemStorage_To_v1alpha11_NnfSystemStorage(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha11.NnfSystemStorage{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	return nil
}

func (dst *NnfSystemStorage) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha11.NnfSystemStorage)
	convertlog.Info("Convert NnfSystemStorage From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha11_NnfSystemStorage_To_v1alpha9_NnfSystemStorage(src, dst, nil); err != nil {
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

// Convert_v1alpha9_NnfStorageProfileLustreData_To_v1alpha11_NnfStorageProfileLustreData handles conversion from v1alpha9 to v1alpha10.
// This is a manual conversion because the types have different structures.
func Convert_v1alpha9_NnfStorageProfileLustreData_To_v1alpha11_NnfStorageProfileLustreData(in *NnfStorageProfileLustreData, out *nnfv1alpha11.NnfStorageProfileLustreData, s apiconversion.Scope) error {
	out.CombinedMGTMDT = in.CombinedMGTMDT

	// MGT options
	out.MgtOptions.ExternalMGS = in.ExternalMGS
	out.MgtOptions.StandaloneMGTPoolName = in.StandaloneMGTPoolName
	out.MgtOptions.Capacity = in.CapacityMGT
	out.MgtOptions.CmdLines.ZpoolCreate = in.MgtCmdLines.ZpoolCreate
	out.MgtOptions.CmdLines.ZpoolReplace = in.MgtCmdLines.ZpoolReplace
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
	out.MdtOptions.CmdLines.ZpoolReplace = in.MdtCmdLines.ZpoolReplace
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
	out.MgtMdtOptions.CmdLines.ZpoolReplace = in.MgtMdtCmdLines.ZpoolReplace
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
	out.OstOptions.CmdLines.ZpoolReplace = in.OstCmdLines.ZpoolReplace
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

// Convert_v1alpha11_NnfStorageProfileLustreData_To_v1alpha9_NnfStorageProfileLustreData handles conversion from v1alpha10 to v1alpha9.
// This is a manual conversion because the types have different structures.
func Convert_v1alpha11_NnfStorageProfileLustreData_To_v1alpha9_NnfStorageProfileLustreData(in *nnfv1alpha11.NnfStorageProfileLustreData, out *NnfStorageProfileLustreData, s apiconversion.Scope) error {
	out.CombinedMGTMDT = in.CombinedMGTMDT

	// MGT options
	out.ExternalMGS = in.MgtOptions.ExternalMGS
	out.StandaloneMGTPoolName = in.MgtOptions.StandaloneMGTPoolName
	out.CapacityMGT = in.MgtOptions.Capacity
	out.MgtCmdLines.ZpoolCreate = in.MgtOptions.CmdLines.ZpoolCreate
	out.MgtCmdLines.ZpoolReplace = in.MgtOptions.CmdLines.ZpoolReplace
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
	out.MdtCmdLines.ZpoolReplace = in.MdtOptions.CmdLines.ZpoolReplace
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
	out.MgtMdtCmdLines.ZpoolReplace = in.MgtMdtOptions.CmdLines.ZpoolReplace
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
	out.OstCmdLines.ZpoolReplace = in.OstOptions.CmdLines.ZpoolReplace
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

// Convert_v1alpha9_NnfStorageProfileLustreMiscOptions_To_v1alpha11_NnfStorageProfileLustreMgtOptions handles conversion.
func Convert_v1alpha9_NnfStorageProfileLustreMiscOptions_To_v1alpha11_NnfStorageProfileLustreMgtOptions(in *NnfStorageProfileLustreMiscOptions, out *nnfv1alpha11.NnfStorageProfileLustreMgtOptions, s apiconversion.Scope) error {
	out.ColocateComputes = in.ColocateComputes
	out.Count = in.Count
	out.Scale = in.Scale
	out.StorageLabels = in.StorageLabels
	return nil
}

// Convert_v1alpha9_NnfStorageProfileLustreMiscOptions_To_v1alpha11_NnfStorageProfileLustreMdtOptions handles conversion.
func Convert_v1alpha9_NnfStorageProfileLustreMiscOptions_To_v1alpha11_NnfStorageProfileLustreMdtOptions(in *NnfStorageProfileLustreMiscOptions, out *nnfv1alpha11.NnfStorageProfileLustreMdtOptions, s apiconversion.Scope) error {
	out.ColocateComputes = in.ColocateComputes
	out.Count = in.Count
	out.Scale = in.Scale
	out.StorageLabels = in.StorageLabels
	return nil
}

// Convert_v1alpha9_NnfStorageProfileLustreMiscOptions_To_v1alpha11_NnfStorageProfileLustreMgtMdtOptions handles conversion.
func Convert_v1alpha9_NnfStorageProfileLustreMiscOptions_To_v1alpha11_NnfStorageProfileLustreMgtMdtOptions(in *NnfStorageProfileLustreMiscOptions, out *nnfv1alpha11.NnfStorageProfileLustreMgtMdtOptions, s apiconversion.Scope) error {
	out.ColocateComputes = in.ColocateComputes
	out.Count = in.Count
	out.Scale = in.Scale
	out.StorageLabels = in.StorageLabels
	return nil
}

// Convert_v1alpha9_NnfStorageProfileLustreMiscOptions_To_v1alpha11_NnfStorageProfileLustreOstOptions handles conversion.
func Convert_v1alpha9_NnfStorageProfileLustreMiscOptions_To_v1alpha11_NnfStorageProfileLustreOstOptions(in *NnfStorageProfileLustreMiscOptions, out *nnfv1alpha11.NnfStorageProfileLustreOstOptions, s apiconversion.Scope) error {
	out.ColocateComputes = in.ColocateComputes
	out.Count = in.Count
	out.Scale = in.Scale
	out.StorageLabels = in.StorageLabels
	return nil
}

// Convert_v1alpha11_NnfStorageProfileLustreMgtOptions_To_v1alpha9_NnfStorageProfileLustreMiscOptions handles conversion.
func Convert_v1alpha11_NnfStorageProfileLustreMgtOptions_To_v1alpha9_NnfStorageProfileLustreMiscOptions(in *nnfv1alpha11.NnfStorageProfileLustreMgtOptions, out *NnfStorageProfileLustreMiscOptions, s apiconversion.Scope) error {
	out.ColocateComputes = in.ColocateComputes
	out.Count = in.Count
	out.Scale = in.Scale
	out.StorageLabels = in.StorageLabels
	return nil
}

// Convert_v1alpha11_NnfStorageProfileLustreMdtOptions_To_v1alpha9_NnfStorageProfileLustreMiscOptions handles conversion.
func Convert_v1alpha11_NnfStorageProfileLustreMdtOptions_To_v1alpha9_NnfStorageProfileLustreMiscOptions(in *nnfv1alpha11.NnfStorageProfileLustreMdtOptions, out *NnfStorageProfileLustreMiscOptions, s apiconversion.Scope) error {
	out.ColocateComputes = in.ColocateComputes
	out.Count = in.Count
	out.Scale = in.Scale
	out.StorageLabels = in.StorageLabels
	return nil
}

// Convert_v1alpha11_NnfStorageProfileLustreMgtMdtOptions_To_v1alpha9_NnfStorageProfileLustreMiscOptions handles conversion.
func Convert_v1alpha11_NnfStorageProfileLustreMgtMdtOptions_To_v1alpha9_NnfStorageProfileLustreMiscOptions(in *nnfv1alpha11.NnfStorageProfileLustreMgtMdtOptions, out *NnfStorageProfileLustreMiscOptions, s apiconversion.Scope) error {
	out.ColocateComputes = in.ColocateComputes
	out.Count = in.Count
	out.Scale = in.Scale
	out.StorageLabels = in.StorageLabels
	return nil
}

// Convert_v1alpha11_NnfStorageProfileLustreOstOptions_To_v1alpha9_NnfStorageProfileLustreMiscOptions handles conversion.
func Convert_v1alpha11_NnfStorageProfileLustreOstOptions_To_v1alpha9_NnfStorageProfileLustreMiscOptions(in *nnfv1alpha11.NnfStorageProfileLustreOstOptions, out *NnfStorageProfileLustreMiscOptions, s apiconversion.Scope) error {
	out.ColocateComputes = in.ColocateComputes
	out.Count = in.Count
	out.Scale = in.Scale
	out.StorageLabels = in.StorageLabels
	return nil
}

// Convert_v1alpha11_NnfStorageProfileSharedData_To_v1alpha9_NnfStorageProfileSharedData handles conversion.
// v1alpha10 has VariableOverride which doesn't exist in v1alpha9.
func Convert_v1alpha11_NnfStorageProfileSharedData_To_v1alpha9_NnfStorageProfileSharedData(in *nnfv1alpha11.NnfStorageProfileSharedData, out *NnfStorageProfileSharedData, s apiconversion.Scope) error {
	if err := Convert_v1alpha11_NnfStorageProfileFileSystem_To_v1alpha9_NnfStorageProfileFileSystem(&in.FileSystemCommands, &out.FileSystemCommands, s); err != nil {
		return err
	}
	if err := Convert_v1alpha11_NnfStorageProfileBlockDevice_To_v1alpha9_NnfStorageProfileBlockDevice(&in.BlockDeviceCommands, &out.BlockDeviceCommands, s); err != nil {
		return err
	}
	if err := Convert_v1alpha11_NnfStorageProfileUserCommands_To_v1alpha9_NnfStorageProfileUserCommands(&in.UserCommands, &out.UserCommands, s); err != nil {
		return err
	}
	out.SharedAllocation = in.SharedAllocation
	out.StorageLabels = in.StorageLabels
	out.CapacityScalingFactor = in.CapacityScalingFactor
	out.AllocationPadding = in.AllocationPadding
	// VariableOverride is lost during conversion from v1alpha10 to v1alpha9
	return nil
}

// Convert_v1alpha11_NnfStorageProfileLustreCmdLines_To_v1alpha9_NnfStorageProfileLustreCmdLines handles conversion.
// v1alpha10 has UnmountTarget and ZpoolDestroy which don't exist in v1alpha9.
func Convert_v1alpha11_NnfStorageProfileLustreCmdLines_To_v1alpha9_NnfStorageProfileLustreCmdLines(in *nnfv1alpha11.NnfStorageProfileLustreCmdLines, out *NnfStorageProfileLustreCmdLines, s apiconversion.Scope) error {
	out.ZpoolCreate = in.ZpoolCreate
	// ZpoolDestroy is lost during conversion from v1alpha10 to v1alpha9
	out.ZpoolReplace = in.ZpoolReplace
	out.Mkfs = in.Mkfs
	out.MountTarget = in.MountTarget
	// UnmountTarget is lost during conversion from v1alpha10 to v1alpha9
	out.PostActivate = *(*[]string)(unsafe.Pointer(&in.PostActivate))
	out.PreDeactivate = *(*[]string)(unsafe.Pointer(&in.PreDeactivate))
	return nil
}

// Convert_v1alpha11_NnfStorageProfileLustreClientCmdLines_To_v1alpha9_NnfStorageProfileLustreClientCmdLines handles conversion.
// v1alpha10 has UnmountRabbit and UnmountCompute which don't exist in v1alpha9.
func Convert_v1alpha11_NnfStorageProfileLustreClientCmdLines_To_v1alpha9_NnfStorageProfileLustreClientCmdLines(in *nnfv1alpha11.NnfStorageProfileLustreClientCmdLines, out *NnfStorageProfileLustreClientCmdLines, s apiconversion.Scope) error {
	out.MountRabbit = in.MountRabbit
	out.RabbitPostSetup = *(*[]string)(unsafe.Pointer(&in.RabbitPostSetup))
	out.RabbitPreTeardown = *(*[]string)(unsafe.Pointer(&in.RabbitPreTeardown))
	out.MountCompute = in.MountCompute
	// UnmountRabbit and UnmountCompute are lost during conversion from v1alpha10 to v1alpha9
	out.RabbitPreMount = *(*[]string)(unsafe.Pointer(&in.RabbitPreMount))
	out.RabbitPostMount = *(*[]string)(unsafe.Pointer(&in.RabbitPostMount))
	out.RabbitPreUnmount = *(*[]string)(unsafe.Pointer(&in.RabbitPreUnmount))
	out.RabbitPostUnmount = *(*[]string)(unsafe.Pointer(&in.RabbitPostUnmount))
	out.ComputePreMount = *(*[]string)(unsafe.Pointer(&in.ComputePreMount))
	out.ComputePostMount = *(*[]string)(unsafe.Pointer(&in.ComputePostMount))
	out.ComputePreUnmount = *(*[]string)(unsafe.Pointer(&in.ComputePreUnmount))
	out.ComputePostUnmount = *(*[]string)(unsafe.Pointer(&in.ComputePostUnmount))
	return nil
}

// Convert_v1alpha11_NnfStorageProfileRabbitFileSystemCommands_To_v1alpha9_NnfStorageProfileRabbitFileSystemCommands handles conversion.
// v1alpha10 has Unmount which doesn't exist in v1alpha9.
func Convert_v1alpha11_NnfStorageProfileRabbitFileSystemCommands_To_v1alpha9_NnfStorageProfileRabbitFileSystemCommands(in *nnfv1alpha11.NnfStorageProfileRabbitFileSystemCommands, out *NnfStorageProfileRabbitFileSystemCommands, s apiconversion.Scope) error {
	out.Mkfs = in.Mkfs
	out.Mount = in.Mount
	// Unmount is lost during conversion from v1alpha10 to v1alpha9
	if err := Convert_v1alpha11_NnfStorageProfileFileSystemUserCommands_To_v1alpha9_NnfStorageProfileFileSystemUserCommands(&in.UserCommands, &out.UserCommands, s); err != nil {
		return err
	}
	return nil
}

// Convert_v1alpha11_NnfStorageProfileComputeFileSystemCommands_To_v1alpha9_NnfStorageProfileComputeFileSystemCommands handles conversion.
// v1alpha10 has Unmount which doesn't exist in v1alpha9.
func Convert_v1alpha11_NnfStorageProfileComputeFileSystemCommands_To_v1alpha9_NnfStorageProfileComputeFileSystemCommands(in *nnfv1alpha11.NnfStorageProfileComputeFileSystemCommands, out *NnfStorageProfileComputeFileSystemCommands, s apiconversion.Scope) error {
	out.Mount = in.Mount
	// Unmount is lost during conversion from v1alpha10 to v1alpha9
	if err := Convert_v1alpha11_NnfStorageProfileFileSystemUserCommands_To_v1alpha9_NnfStorageProfileFileSystemUserCommands(&in.UserCommands, &out.UserCommands, s); err != nil {
		return err
	}
	return nil
}

// Convert_v1alpha11_NnfStorageProfileBlockDeviceUserCommands_To_v1alpha9_NnfStorageProfileBlockDeviceUserCommands handles conversion.
// v1alpha11 has PostTeardown which doesn't exist in v1alpha9.
func Convert_v1alpha11_NnfStorageProfileBlockDeviceUserCommands_To_v1alpha9_NnfStorageProfileBlockDeviceUserCommands(in *nnfv1alpha11.NnfStorageProfileBlockDeviceUserCommands, out *NnfStorageProfileBlockDeviceUserCommands, s apiconversion.Scope) error {
	out.PreActivate = in.PreActivate
	out.PostActivate = in.PostActivate
	out.PreDeactivate = in.PreDeactivate
	out.PostDeactivate = in.PostDeactivate
	// PostTeardown is lost during conversion from v1alpha11 to v1alpha9
	return nil
}
