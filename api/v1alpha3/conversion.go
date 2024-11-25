/*
 * Copyright 2024 Hewlett Packard Enterprise Development LP
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

package v1alpha3

import (
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apiconversion "k8s.io/apimachinery/pkg/conversion"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	nnfv1alpha4 "github.com/NearNodeFlash/nnf-sos/api/v1alpha4"
	utilconversion "github.com/NearNodeFlash/nnf-sos/github/cluster-api/util/conversion"
)

var convertlog = logf.Log.V(2).WithName("convert-v1alpha3")

func (src *NnfAccess) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfAccess To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha4.NnfAccess)

	if err := Convert_v1alpha3_NnfAccess_To_v1alpha4_NnfAccess(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha4.NnfAccess{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	return nil
}

func (dst *NnfAccess) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha4.NnfAccess)
	convertlog.Info("Convert NnfAccess From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha4_NnfAccess_To_v1alpha3_NnfAccess(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfContainerProfile) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfContainerProfile To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha4.NnfContainerProfile)

	if err := Convert_v1alpha3_NnfContainerProfile_To_v1alpha4_NnfContainerProfile(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha4.NnfContainerProfile{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	return nil
}

func (dst *NnfContainerProfile) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha4.NnfContainerProfile)
	convertlog.Info("Convert NnfContainerProfile From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha4_NnfContainerProfile_To_v1alpha3_NnfContainerProfile(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfDataMovement) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfDataMovement To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha4.NnfDataMovement)

	if err := Convert_v1alpha3_NnfDataMovement_To_v1alpha4_NnfDataMovement(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha4.NnfDataMovement{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	return nil
}

func (dst *NnfDataMovement) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha4.NnfDataMovement)
	convertlog.Info("Convert NnfDataMovement From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha4_NnfDataMovement_To_v1alpha3_NnfDataMovement(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfDataMovementManager) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfDataMovementManager To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha4.NnfDataMovementManager)

	if err := Convert_v1alpha3_NnfDataMovementManager_To_v1alpha4_NnfDataMovementManager(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha4.NnfDataMovementManager{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	return nil
}

func (dst *NnfDataMovementManager) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha4.NnfDataMovementManager)
	convertlog.Info("Convert NnfDataMovementManager From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha4_NnfDataMovementManager_To_v1alpha3_NnfDataMovementManager(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfDataMovementProfile) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfDataMovementProfile To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha4.NnfDataMovementProfile)

	if err := Convert_v1alpha3_NnfDataMovementProfile_To_v1alpha4_NnfDataMovementProfile(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha4.NnfDataMovementProfile{}
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
	src := srcRaw.(*nnfv1alpha4.NnfDataMovementProfile)
	convertlog.Info("Convert NnfDataMovementProfile From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha4_NnfDataMovementProfile_To_v1alpha3_NnfDataMovementProfile(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfLustreMGT) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfLustreMGT To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha4.NnfLustreMGT)

	if err := Convert_v1alpha3_NnfLustreMGT_To_v1alpha4_NnfLustreMGT(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha4.NnfLustreMGT{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	return nil
}

func (dst *NnfLustreMGT) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha4.NnfLustreMGT)
	convertlog.Info("Convert NnfLustreMGT From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha4_NnfLustreMGT_To_v1alpha3_NnfLustreMGT(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfNode) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfNode To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha4.NnfNode)

	if err := Convert_v1alpha3_NnfNode_To_v1alpha4_NnfNode(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha4.NnfNode{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	return nil
}

func (dst *NnfNode) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha4.NnfNode)
	convertlog.Info("Convert NnfNode From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha4_NnfNode_To_v1alpha3_NnfNode(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfNodeBlockStorage) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfNodeBlockStorage To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha4.NnfNodeBlockStorage)

	if err := Convert_v1alpha3_NnfNodeBlockStorage_To_v1alpha4_NnfNodeBlockStorage(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha4.NnfNodeBlockStorage{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	return nil
}

func (dst *NnfNodeBlockStorage) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha4.NnfNodeBlockStorage)
	convertlog.Info("Convert NnfNodeBlockStorage From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha4_NnfNodeBlockStorage_To_v1alpha3_NnfNodeBlockStorage(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfNodeECData) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfNodeECData To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha4.NnfNodeECData)

	if err := Convert_v1alpha3_NnfNodeECData_To_v1alpha4_NnfNodeECData(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha4.NnfNodeECData{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	return nil
}

func (dst *NnfNodeECData) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha4.NnfNodeECData)
	convertlog.Info("Convert NnfNodeECData From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha4_NnfNodeECData_To_v1alpha3_NnfNodeECData(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfNodeStorage) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfNodeStorage To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha4.NnfNodeStorage)

	if err := Convert_v1alpha3_NnfNodeStorage_To_v1alpha4_NnfNodeStorage(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha4.NnfNodeStorage{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	return nil
}

func (dst *NnfNodeStorage) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha4.NnfNodeStorage)
	convertlog.Info("Convert NnfNodeStorage From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha4_NnfNodeStorage_To_v1alpha3_NnfNodeStorage(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfPortManager) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfPortManager To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha4.NnfPortManager)

	if err := Convert_v1alpha3_NnfPortManager_To_v1alpha4_NnfPortManager(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha4.NnfPortManager{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	return nil
}

func (dst *NnfPortManager) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha4.NnfPortManager)
	convertlog.Info("Convert NnfPortManager From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha4_NnfPortManager_To_v1alpha3_NnfPortManager(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfStorage) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfStorage To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha4.NnfStorage)

	if err := Convert_v1alpha3_NnfStorage_To_v1alpha4_NnfStorage(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha4.NnfStorage{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	return nil
}

func (dst *NnfStorage) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha4.NnfStorage)
	convertlog.Info("Convert NnfStorage From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha4_NnfStorage_To_v1alpha3_NnfStorage(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfStorageProfile) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfStorageProfile To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha4.NnfStorageProfile)

	if err := Convert_v1alpha3_NnfStorageProfile_To_v1alpha4_NnfStorageProfile(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha4.NnfStorageProfile{}
	hasAnno, err := utilconversion.UnmarshalData(src, restored)
	if err != nil {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.
	if hasAnno {
		dst.Data.LustreStorage.MgtCmdLines.PostMount = append([]string(nil), restored.Data.LustreStorage.MgtCmdLines.PostMount...)
		dst.Data.LustreStorage.MgtCmdLines.PreUnmount = append([]string(nil), restored.Data.LustreStorage.MgtCmdLines.PreUnmount...)
		dst.Data.LustreStorage.MgtMdtCmdLines.PostMount = append([]string(nil), restored.Data.LustreStorage.MgtMdtCmdLines.PostMount...)
		dst.Data.LustreStorage.MgtMdtCmdLines.PreUnmount = append([]string(nil), restored.Data.LustreStorage.MgtMdtCmdLines.PreUnmount...)
		dst.Data.LustreStorage.MdtCmdLines.PostMount = append([]string(nil), restored.Data.LustreStorage.MdtCmdLines.PostMount...)
		dst.Data.LustreStorage.MdtCmdLines.PreUnmount = append([]string(nil), restored.Data.LustreStorage.MdtCmdLines.PreUnmount...)
		dst.Data.LustreStorage.OstCmdLines.PostMount = append([]string(nil), restored.Data.LustreStorage.OstCmdLines.PostMount...)
		dst.Data.LustreStorage.OstCmdLines.PreUnmount = append([]string(nil), restored.Data.LustreStorage.OstCmdLines.PreUnmount...)
		dst.Data.RawStorage.CmdLines.PostMount = append([]string(nil), restored.Data.RawStorage.CmdLines.PostMount...)
		dst.Data.RawStorage.CmdLines.PreUnmount = append([]string(nil), restored.Data.RawStorage.CmdLines.PreUnmount...)
		dst.Data.XFSStorage.CmdLines.PostMount = append([]string(nil), restored.Data.XFSStorage.CmdLines.PostMount...)
		dst.Data.XFSStorage.CmdLines.PreUnmount = append([]string(nil), restored.Data.XFSStorage.CmdLines.PreUnmount...)
		dst.Data.GFS2Storage.CmdLines.PostMount = append([]string(nil), restored.Data.GFS2Storage.CmdLines.PostMount...)
		dst.Data.GFS2Storage.CmdLines.PreUnmount = append([]string(nil), restored.Data.GFS2Storage.CmdLines.PreUnmount...)
	} else {
		// For non-lustre, PostActivate is now PostMount and PreDeactivate is now PreUnmount
		dst.Data.RawStorage.CmdLines.PostMount = src.Data.RawStorage.CmdLines.PostActivate
		dst.Data.XFSStorage.CmdLines.PostMount = src.Data.XFSStorage.CmdLines.PostActivate
		dst.Data.GFS2Storage.CmdLines.PostMount = src.Data.GFS2Storage.CmdLines.PostActivate
		dst.Data.RawStorage.CmdLines.PreUnmount = src.Data.RawStorage.CmdLines.PreDeactivate
		dst.Data.XFSStorage.CmdLines.PreUnmount = src.Data.XFSStorage.CmdLines.PreDeactivate
		dst.Data.GFS2Storage.CmdLines.PreUnmount = src.Data.GFS2Storage.CmdLines.PreDeactivate
	}

	return nil
}

func (dst *NnfStorageProfile) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha4.NnfStorageProfile)
	convertlog.Info("Convert NnfStorageProfile From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha4_NnfStorageProfile_To_v1alpha3_NnfStorageProfile(src, dst, nil); err != nil {
		return err
	}

	// For non-lustre, PostActivate is now PostMount and PreDeactivate is now PreUnmount
	dst.Data.RawStorage.CmdLines.PostActivate = src.Data.RawStorage.CmdLines.PostMount
	dst.Data.XFSStorage.CmdLines.PostActivate = src.Data.XFSStorage.CmdLines.PostMount
	dst.Data.GFS2Storage.CmdLines.PostActivate = src.Data.GFS2Storage.CmdLines.PostMount
	dst.Data.RawStorage.CmdLines.PreDeactivate = src.Data.RawStorage.CmdLines.PreUnmount
	dst.Data.XFSStorage.CmdLines.PreDeactivate = src.Data.XFSStorage.CmdLines.PreUnmount
	dst.Data.GFS2Storage.CmdLines.PreDeactivate = src.Data.GFS2Storage.CmdLines.PreUnmount

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfSystemStorage) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfSystemStorage To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha4.NnfSystemStorage)

	if err := Convert_v1alpha3_NnfSystemStorage_To_v1alpha4_NnfSystemStorage(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha4.NnfSystemStorage{}
	hasAnno, err := utilconversion.UnmarshalData(src, restored)
	if err != nil {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	if hasAnno {
		dst.Spec.Shared = restored.Spec.Shared
	} else {
		dst.Spec.Shared = true
	}

	return nil
}

func (dst *NnfSystemStorage) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha4.NnfSystemStorage)
	convertlog.Info("Convert NnfSystemStorage From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha4_NnfSystemStorage_To_v1alpha3_NnfSystemStorage(src, dst, nil); err != nil {
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

// Convert_v1alpha3_NnfStorageProfileCmdLines_To_v1alpha4_NnfStorageProfileCmdLines is an autogenerated conversion function.
func Convert_v1alpha3_NnfStorageProfileCmdLines_To_v1alpha4_NnfStorageProfileCmdLines(in *NnfStorageProfileCmdLines, out *nnfv1alpha4.NnfStorageProfileCmdLines, s apiconversion.Scope) error {
	return autoConvert_v1alpha3_NnfStorageProfileCmdLines_To_v1alpha4_NnfStorageProfileCmdLines(in, out, s)
}

// Convert_v1alpha4_NnfStorageProfileCmdLines_To_v1alpha3_NnfStorageProfileCmdLines is an autogenerated conversion function.
func Convert_v1alpha4_NnfStorageProfileCmdLines_To_v1alpha3_NnfStorageProfileCmdLines(in *nnfv1alpha4.NnfStorageProfileCmdLines, out *NnfStorageProfileCmdLines, s apiconversion.Scope) error {
	return autoConvert_v1alpha4_NnfStorageProfileCmdLines_To_v1alpha3_NnfStorageProfileCmdLines(in, out, s)
}

// Convert_v1alpha4_NnfStorageProfileLustreCmdLines_To_v1alpha3_NnfStorageProfileLustreCmdLines is an autogenerated conversion function.
func Convert_v1alpha4_NnfStorageProfileLustreCmdLines_To_v1alpha3_NnfStorageProfileLustreCmdLines(in *nnfv1alpha4.NnfStorageProfileLustreCmdLines, out *NnfStorageProfileLustreCmdLines, s apiconversion.Scope) error {
	return autoConvert_v1alpha4_NnfStorageProfileLustreCmdLines_To_v1alpha3_NnfStorageProfileLustreCmdLines(in, out, s)
}

func Convert_v1alpha4_NnfSystemStorageSpec_To_v1alpha3_NnfSystemStorageSpec(in *nnfv1alpha4.NnfSystemStorageSpec, out *NnfSystemStorageSpec, s apiconversion.Scope) error {
	return autoConvert_v1alpha4_NnfSystemStorageSpec_To_v1alpha3_NnfSystemStorageSpec(in, out, s)
}

func Convert_v1alpha4_NnfDataMovementProfileData_To_v1alpha3_NnfDataMovementProfileData(in *nnfv1alpha4.NnfDataMovementProfileData, out *NnfDataMovementProfileData, s apiconversion.Scope) error {
	return autoConvert_v1alpha4_NnfDataMovementProfileData_To_v1alpha3_NnfDataMovementProfileData(in, out, s)
}
