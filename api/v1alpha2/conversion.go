/*
 * Copyright 2024-2025 Hewlett Packard Enterprise Development LP
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

package v1alpha2

import (
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apiconversion "k8s.io/apimachinery/pkg/conversion"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	nnfv1alpha5 "github.com/NearNodeFlash/nnf-sos/api/v1alpha5"
	utilconversion "github.com/NearNodeFlash/nnf-sos/github/cluster-api/util/conversion"
)

var convertlog = logf.Log.V(2).WithName("convert-v1alpha2")

func (src *NnfAccess) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfAccess To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha5.NnfAccess)

	if err := Convert_v1alpha2_NnfAccess_To_v1alpha5_NnfAccess(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha5.NnfAccess{}
	hasAnno, err := utilconversion.UnmarshalData(src, restored)
	if err != nil {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	if hasAnno {
		dst.Spec.IgnoreOfflineComputes = restored.Spec.IgnoreOfflineComputes
	} else {
		dst.Spec.IgnoreOfflineComputes = false
	}

	return nil
}

func (dst *NnfAccess) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha5.NnfAccess)
	convertlog.Info("Convert NnfAccess From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha5_NnfAccess_To_v1alpha2_NnfAccess(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfContainerProfile) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfContainerProfile To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha5.NnfContainerProfile)

	if err := Convert_v1alpha2_NnfContainerProfile_To_v1alpha5_NnfContainerProfile(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha5.NnfContainerProfile{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	return nil
}

func (dst *NnfContainerProfile) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha5.NnfContainerProfile)
	convertlog.Info("Convert NnfContainerProfile From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha5_NnfContainerProfile_To_v1alpha2_NnfContainerProfile(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfDataMovement) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfDataMovement To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha5.NnfDataMovement)

	if err := Convert_v1alpha2_NnfDataMovement_To_v1alpha5_NnfDataMovement(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha5.NnfDataMovement{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	return nil
}

func (dst *NnfDataMovement) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha5.NnfDataMovement)
	convertlog.Info("Convert NnfDataMovement From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha5_NnfDataMovement_To_v1alpha2_NnfDataMovement(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfDataMovementManager) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfDataMovementManager To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha5.NnfDataMovementManager)

	if err := Convert_v1alpha2_NnfDataMovementManager_To_v1alpha5_NnfDataMovementManager(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha5.NnfDataMovementManager{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	return nil
}

func (dst *NnfDataMovementManager) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha5.NnfDataMovementManager)
	convertlog.Info("Convert NnfDataMovementManager From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha5_NnfDataMovementManager_To_v1alpha2_NnfDataMovementManager(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfDataMovementProfile) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfDataMovementProfile To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha5.NnfDataMovementProfile)

	if err := Convert_v1alpha2_NnfDataMovementProfile_To_v1alpha5_NnfDataMovementProfile(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha5.NnfDataMovementProfile{}
	hasAnno, err := utilconversion.UnmarshalData(src, restored)
	if err != nil {
		return err
	}

	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	if hasAnno {
		dst.Data.MkdirCommand = restored.Data.MkdirCommand
	}

	return nil
}

func (dst *NnfDataMovementProfile) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha5.NnfDataMovementProfile)
	convertlog.Info("Convert NnfDataMovementProfile From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha5_NnfDataMovementProfile_To_v1alpha2_NnfDataMovementProfile(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfLustreMGT) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfLustreMGT To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha5.NnfLustreMGT)

	if err := Convert_v1alpha2_NnfLustreMGT_To_v1alpha5_NnfLustreMGT(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha5.NnfLustreMGT{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	return nil
}

func (dst *NnfLustreMGT) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha5.NnfLustreMGT)
	convertlog.Info("Convert NnfLustreMGT From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha5_NnfLustreMGT_To_v1alpha2_NnfLustreMGT(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfNode) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfNode To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha5.NnfNode)

	if err := Convert_v1alpha2_NnfNode_To_v1alpha5_NnfNode(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha5.NnfNode{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	return nil
}

func (dst *NnfNode) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha5.NnfNode)
	convertlog.Info("Convert NnfNode From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha5_NnfNode_To_v1alpha2_NnfNode(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfNodeBlockStorage) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfNodeBlockStorage To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha5.NnfNodeBlockStorage)

	if err := Convert_v1alpha2_NnfNodeBlockStorage_To_v1alpha5_NnfNodeBlockStorage(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha5.NnfNodeBlockStorage{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	return nil
}

func (dst *NnfNodeBlockStorage) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha5.NnfNodeBlockStorage)
	convertlog.Info("Convert NnfNodeBlockStorage From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha5_NnfNodeBlockStorage_To_v1alpha2_NnfNodeBlockStorage(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfNodeECData) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfNodeECData To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha5.NnfNodeECData)

	if err := Convert_v1alpha2_NnfNodeECData_To_v1alpha5_NnfNodeECData(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha5.NnfNodeECData{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	return nil
}

func (dst *NnfNodeECData) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha5.NnfNodeECData)
	convertlog.Info("Convert NnfNodeECData From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha5_NnfNodeECData_To_v1alpha2_NnfNodeECData(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfNodeStorage) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfNodeStorage To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha5.NnfNodeStorage)

	if err := Convert_v1alpha2_NnfNodeStorage_To_v1alpha5_NnfNodeStorage(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha5.NnfNodeStorage{}
	hasAnno, err := utilconversion.UnmarshalData(src, restored)
	if err != nil {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.
	if hasAnno {
		dst.Spec.LustreStorage.LustreComponents.MDTs = append([]string(nil), restored.Spec.LustreStorage.LustreComponents.MDTs...)
		dst.Spec.LustreStorage.LustreComponents.MGTs = append([]string(nil), restored.Spec.LustreStorage.LustreComponents.MGTs...)
		dst.Spec.LustreStorage.LustreComponents.MGTMDTs = append([]string(nil), restored.Spec.LustreStorage.LustreComponents.MGTMDTs...)
		dst.Spec.LustreStorage.LustreComponents.OSTs = append([]string(nil), restored.Spec.LustreStorage.LustreComponents.OSTs...)
		dst.Spec.LustreStorage.LustreComponents.NNFNodes = append([]string(nil), restored.Spec.LustreStorage.LustreComponents.NNFNodes...)
		dst.Spec.CommandVariables = make([]nnfv1alpha5.CommandVariablesSpec, len(restored.Spec.CommandVariables))
		copy(dst.Spec.CommandVariables, restored.Spec.CommandVariables)
	}

	return nil
}

func (dst *NnfNodeStorage) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha5.NnfNodeStorage)
	convertlog.Info("Convert NnfNodeStorage From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha5_NnfNodeStorage_To_v1alpha2_NnfNodeStorage(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfPortManager) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfPortManager To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha5.NnfPortManager)

	if err := Convert_v1alpha2_NnfPortManager_To_v1alpha5_NnfPortManager(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha5.NnfPortManager{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	return nil
}

func (dst *NnfPortManager) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha5.NnfPortManager)
	convertlog.Info("Convert NnfPortManager From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha5_NnfPortManager_To_v1alpha2_NnfPortManager(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfStorage) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfStorage To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha5.NnfStorage)

	if err := Convert_v1alpha2_NnfStorage_To_v1alpha5_NnfStorage(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha5.NnfStorage{}
	hasAnno, err := utilconversion.UnmarshalData(src, restored)
	if err != nil {
		return err
	}

	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.
	if hasAnno {
		dst.Status.LustreComponents.MDTs = append([]string(nil), restored.Status.LustreComponents.MDTs...)
		dst.Status.LustreComponents.MGTs = append([]string(nil), restored.Status.LustreComponents.MGTs...)
		dst.Status.LustreComponents.MGTMDTs = append([]string(nil), restored.Status.LustreComponents.MGTMDTs...)
		dst.Status.LustreComponents.OSTs = append([]string(nil), restored.Status.LustreComponents.OSTs...)
		dst.Status.LustreComponents.NNFNodes = append([]string(nil), restored.Status.LustreComponents.NNFNodes...)
		for i := range restored.Spec.AllocationSets {
			dst.Spec.AllocationSets[i].CommandVariables = make([]nnfv1alpha5.CommandVariablesSpec, len(restored.Spec.AllocationSets[i].CommandVariables))
			copy(dst.Spec.AllocationSets[i].CommandVariables, restored.Spec.AllocationSets[i].CommandVariables)
		}
	}

	return nil
}

func (dst *NnfStorage) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha5.NnfStorage)
	convertlog.Info("Convert NnfStorage From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha5_NnfStorage_To_v1alpha2_NnfStorage(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfStorageProfile) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfStorageProfile To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha5.NnfStorageProfile)

	if err := Convert_v1alpha2_NnfStorageProfile_To_v1alpha5_NnfStorageProfile(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha5.NnfStorageProfile{}
	hasAnno, err := utilconversion.UnmarshalData(src, restored)
	if err != nil {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	if hasAnno {
		dst.Data.LustreStorage.MgtCmdLines.PostActivate = append([]string(nil), restored.Data.LustreStorage.MgtCmdLines.PostActivate...)
		dst.Data.LustreStorage.MgtCmdLines.PreDeactivate = append([]string(nil), restored.Data.LustreStorage.MgtCmdLines.PreDeactivate...)
		dst.Data.LustreStorage.MgtCmdLines.PostMount = append([]string(nil), restored.Data.LustreStorage.MgtCmdLines.PostMount...)
		dst.Data.LustreStorage.MgtCmdLines.PreUnmount = append([]string(nil), restored.Data.LustreStorage.MgtCmdLines.PreUnmount...)
		dst.Data.LustreStorage.MgtMdtCmdLines.PostActivate = append([]string(nil), restored.Data.LustreStorage.MgtMdtCmdLines.PostActivate...)
		dst.Data.LustreStorage.MgtMdtCmdLines.PreDeactivate = append([]string(nil), restored.Data.LustreStorage.MgtMdtCmdLines.PreDeactivate...)
		dst.Data.LustreStorage.MgtMdtCmdLines.PostMount = append([]string(nil), restored.Data.LustreStorage.MgtMdtCmdLines.PostMount...)
		dst.Data.LustreStorage.MgtMdtCmdLines.PreUnmount = append([]string(nil), restored.Data.LustreStorage.MgtMdtCmdLines.PreUnmount...)
		dst.Data.LustreStorage.MdtCmdLines.PostActivate = append([]string(nil), restored.Data.LustreStorage.MdtCmdLines.PostActivate...)
		dst.Data.LustreStorage.MdtCmdLines.PreDeactivate = append([]string(nil), restored.Data.LustreStorage.MdtCmdLines.PreDeactivate...)
		dst.Data.LustreStorage.MdtCmdLines.PostMount = append([]string(nil), restored.Data.LustreStorage.MdtCmdLines.PostMount...)
		dst.Data.LustreStorage.MdtCmdLines.PreUnmount = append([]string(nil), restored.Data.LustreStorage.MdtCmdLines.PreUnmount...)
		dst.Data.LustreStorage.OstCmdLines.PostActivate = append([]string(nil), restored.Data.LustreStorage.OstCmdLines.PostActivate...)
		dst.Data.LustreStorage.OstCmdLines.PreDeactivate = append([]string(nil), restored.Data.LustreStorage.OstCmdLines.PreDeactivate...)
		dst.Data.LustreStorage.OstCmdLines.PostMount = append([]string(nil), restored.Data.LustreStorage.OstCmdLines.PostMount...)
		dst.Data.LustreStorage.OstCmdLines.PreUnmount = append([]string(nil), restored.Data.LustreStorage.OstCmdLines.PreUnmount...)
		dst.Data.RawStorage.CmdLines.PostMount = append([]string(nil), restored.Data.RawStorage.CmdLines.PostMount...)
		dst.Data.RawStorage.CmdLines.PreUnmount = append([]string(nil), restored.Data.RawStorage.CmdLines.PreUnmount...)
		dst.Data.XFSStorage.CmdLines.PostMount = append([]string(nil), restored.Data.XFSStorage.CmdLines.PostMount...)
		dst.Data.XFSStorage.CmdLines.PreUnmount = append([]string(nil), restored.Data.XFSStorage.CmdLines.PreUnmount...)
		dst.Data.GFS2Storage.CmdLines.PostMount = append([]string(nil), restored.Data.GFS2Storage.CmdLines.PostMount...)
		dst.Data.GFS2Storage.CmdLines.PreUnmount = append([]string(nil), restored.Data.GFS2Storage.CmdLines.PreUnmount...)
	}

	return nil
}

func (dst *NnfStorageProfile) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha5.NnfStorageProfile)
	convertlog.Info("Convert NnfStorageProfile From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha5_NnfStorageProfile_To_v1alpha2_NnfStorageProfile(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfSystemStorage) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfSystemStorage To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha5.NnfSystemStorage)

	if err := Convert_v1alpha2_NnfSystemStorage_To_v1alpha5_NnfSystemStorage(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha5.NnfSystemStorage{}
	hasAnno, err := utilconversion.UnmarshalData(src, restored)
	if err != nil {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	if hasAnno {
		dst.Spec.Shared = restored.Spec.Shared
		dst.Spec.IgnoreOfflineComputes = restored.Spec.IgnoreOfflineComputes
	} else {
		dst.Spec.Shared = true
		dst.Spec.IgnoreOfflineComputes = false
	}

	return nil
}

func (dst *NnfSystemStorage) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha5.NnfSystemStorage)
	convertlog.Info("Convert NnfSystemStorage From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha5_NnfSystemStorage_To_v1alpha2_NnfSystemStorage(src, dst, nil); err != nil {
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

// The conversion-gen tool dropped these from zz_generated.conversion.go to
// force us to acknowledge that we are addressing the conversion requirements.

func Convert_v1alpha5_NnfStorageProfileCmdLines_To_v1alpha2_NnfStorageProfileCmdLines(in *nnfv1alpha5.NnfStorageProfileCmdLines, out *NnfStorageProfileCmdLines, s apiconversion.Scope) error {
	return autoConvert_v1alpha5_NnfStorageProfileCmdLines_To_v1alpha2_NnfStorageProfileCmdLines(in, out, s)
}

func Convert_v1alpha5_NnfStorageProfileLustreCmdLines_To_v1alpha2_NnfStorageProfileLustreCmdLines(in *nnfv1alpha5.NnfStorageProfileLustreCmdLines, out *NnfStorageProfileLustreCmdLines, s apiconversion.Scope) error {
	return autoConvert_v1alpha5_NnfStorageProfileLustreCmdLines_To_v1alpha2_NnfStorageProfileLustreCmdLines(in, out, s)
}

func Convert_v1alpha5_NnfSystemStorageSpec_To_v1alpha2_NnfSystemStorageSpec(in *nnfv1alpha5.NnfSystemStorageSpec, out *NnfSystemStorageSpec, s apiconversion.Scope) error {
	return autoConvert_v1alpha5_NnfSystemStorageSpec_To_v1alpha2_NnfSystemStorageSpec(in, out, s)
}

func Convert_v1alpha5_NnfAccessSpec_To_v1alpha2_NnfAccessSpec(in *nnfv1alpha5.NnfAccessSpec, out *NnfAccessSpec, s apiconversion.Scope) error {
	return autoConvert_v1alpha5_NnfAccessSpec_To_v1alpha2_NnfAccessSpec(in, out, s)
}

func Convert_v1alpha5_NnfDataMovementProfileData_To_v1alpha2_NnfDataMovementProfileData(in *nnfv1alpha5.NnfDataMovementProfileData, out *NnfDataMovementProfileData, s apiconversion.Scope) error {
	return autoConvert_v1alpha5_NnfDataMovementProfileData_To_v1alpha2_NnfDataMovementProfileData(in, out, s)
}

func Convert_v1alpha5_LustreStorageSpec_To_v1alpha2_LustreStorageSpec(in *nnfv1alpha5.LustreStorageSpec, out *LustreStorageSpec, s apiconversion.Scope) error {
	return autoConvert_v1alpha5_LustreStorageSpec_To_v1alpha2_LustreStorageSpec(in, out, s)
}

func Convert_v1alpha5_NnfStorageLustreStatus_To_v1alpha2_NnfStorageLustreStatus(in *nnfv1alpha5.NnfStorageLustreStatus, out *NnfStorageLustreStatus, s apiconversion.Scope) error {
	return autoConvert_v1alpha5_NnfStorageLustreStatus_To_v1alpha2_NnfStorageLustreStatus(in, out, s)
}

func Convert_v1alpha5_NnfNodeStorageSpec_To_v1alpha2_NnfNodeStorageSpec(in *nnfv1alpha5.NnfNodeStorageSpec, out *NnfNodeStorageSpec, s apiconversion.Scope) error {
	return autoConvert_v1alpha5_NnfNodeStorageSpec_To_v1alpha2_NnfNodeStorageSpec(in, out, s)
}

func Convert_v1alpha5_NnfStorageAllocationSetSpec_To_v1alpha2_NnfStorageAllocationSetSpec(in *nnfv1alpha5.NnfStorageAllocationSetSpec, out *NnfStorageAllocationSetSpec, s apiconversion.Scope) error {
	return autoConvert_v1alpha5_NnfStorageAllocationSetSpec_To_v1alpha2_NnfStorageAllocationSetSpec(in, out, s)
}
