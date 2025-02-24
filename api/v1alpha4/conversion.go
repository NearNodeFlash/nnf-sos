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

package v1alpha4

import (
	unsafe "unsafe"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apiconversion "k8s.io/apimachinery/pkg/conversion"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	dwsv1alpha2 "github.com/DataWorkflowServices/dws/api/v1alpha2"
	dwsv1alpha3 "github.com/DataWorkflowServices/dws/api/v1alpha3"
	nnfv1alpha6 "github.com/NearNodeFlash/nnf-sos/api/v1alpha6"
	utilconversion "github.com/NearNodeFlash/nnf-sos/github/cluster-api/util/conversion"
)

var convertlog = logf.Log.V(2).WithName("convert-v1alpha4")

func (src *NnfAccess) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfAccess To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha6.NnfAccess)

	if err := Convert_v1alpha4_NnfAccess_To_v1alpha6_NnfAccess(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha6.NnfAccess{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	return nil
}

func (dst *NnfAccess) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha6.NnfAccess)
	convertlog.Info("Convert NnfAccess From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha6_NnfAccess_To_v1alpha4_NnfAccess(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfContainerProfile) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfContainerProfile To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha6.NnfContainerProfile)

	if err := Convert_v1alpha4_NnfContainerProfile_To_v1alpha6_NnfContainerProfile(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha6.NnfContainerProfile{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	return nil
}

func (dst *NnfContainerProfile) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha6.NnfContainerProfile)
	convertlog.Info("Convert NnfContainerProfile From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha6_NnfContainerProfile_To_v1alpha4_NnfContainerProfile(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfDataMovement) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfDataMovement To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha6.NnfDataMovement)

	if err := Convert_v1alpha4_NnfDataMovement_To_v1alpha6_NnfDataMovement(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha6.NnfDataMovement{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	return nil
}

func (dst *NnfDataMovement) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha6.NnfDataMovement)
	convertlog.Info("Convert NnfDataMovement From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha6_NnfDataMovement_To_v1alpha4_NnfDataMovement(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfDataMovementManager) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfDataMovementManager To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha6.NnfDataMovementManager)

	if err := Convert_v1alpha4_NnfDataMovementManager_To_v1alpha6_NnfDataMovementManager(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha6.NnfDataMovementManager{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	return nil
}

func (dst *NnfDataMovementManager) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha6.NnfDataMovementManager)
	convertlog.Info("Convert NnfDataMovementManager From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha6_NnfDataMovementManager_To_v1alpha4_NnfDataMovementManager(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfDataMovementProfile) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfDataMovementProfile To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha6.NnfDataMovementProfile)

	if err := Convert_v1alpha4_NnfDataMovementProfile_To_v1alpha6_NnfDataMovementProfile(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha6.NnfDataMovementProfile{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	return nil
}

func (dst *NnfDataMovementProfile) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha6.NnfDataMovementProfile)
	convertlog.Info("Convert NnfDataMovementProfile From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha6_NnfDataMovementProfile_To_v1alpha4_NnfDataMovementProfile(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfLustreMGT) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfLustreMGT To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha6.NnfLustreMGT)

	if err := Convert_v1alpha4_NnfLustreMGT_To_v1alpha6_NnfLustreMGT(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha6.NnfLustreMGT{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	return nil
}

func (dst *NnfLustreMGT) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha6.NnfLustreMGT)
	convertlog.Info("Convert NnfLustreMGT From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha6_NnfLustreMGT_To_v1alpha4_NnfLustreMGT(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfNode) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfNode To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha6.NnfNode)

	if err := Convert_v1alpha4_NnfNode_To_v1alpha6_NnfNode(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha6.NnfNode{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	return nil
}

func (dst *NnfNode) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha6.NnfNode)
	convertlog.Info("Convert NnfNode From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha6_NnfNode_To_v1alpha4_NnfNode(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfNodeBlockStorage) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfNodeBlockStorage To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha6.NnfNodeBlockStorage)

	if err := Convert_v1alpha4_NnfNodeBlockStorage_To_v1alpha6_NnfNodeBlockStorage(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha6.NnfNodeBlockStorage{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	return nil
}

func (dst *NnfNodeBlockStorage) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha6.NnfNodeBlockStorage)
	convertlog.Info("Convert NnfNodeBlockStorage From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha6_NnfNodeBlockStorage_To_v1alpha4_NnfNodeBlockStorage(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfNodeECData) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfNodeECData To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha6.NnfNodeECData)

	if err := Convert_v1alpha4_NnfNodeECData_To_v1alpha6_NnfNodeECData(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha6.NnfNodeECData{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	return nil
}

func (dst *NnfNodeECData) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha6.NnfNodeECData)
	convertlog.Info("Convert NnfNodeECData From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha6_NnfNodeECData_To_v1alpha4_NnfNodeECData(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfNodeStorage) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfNodeStorage To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha6.NnfNodeStorage)

	if err := Convert_v1alpha4_NnfNodeStorage_To_v1alpha6_NnfNodeStorage(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha6.NnfNodeStorage{}
	hasAnno, err := utilconversion.UnmarshalData(src, restored)
	if err != nil {
		return err
	}

	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.
	if hasAnno {
		dst.Spec.CommandVariables = make([]nnfv1alpha6.CommandVariablesSpec, len(restored.Spec.CommandVariables))
		copy(dst.Spec.CommandVariables, restored.Spec.CommandVariables)
	}

	return nil
}

func (dst *NnfNodeStorage) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha6.NnfNodeStorage)
	convertlog.Info("Convert NnfNodeStorage From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha6_NnfNodeStorage_To_v1alpha4_NnfNodeStorage(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfPortManager) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfPortManager To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha6.NnfPortManager)

	if err := Convert_v1alpha4_NnfPortManager_To_v1alpha6_NnfPortManager(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha6.NnfPortManager{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	return nil
}

func (dst *NnfPortManager) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha6.NnfPortManager)
	convertlog.Info("Convert NnfPortManager From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha6_NnfPortManager_To_v1alpha4_NnfPortManager(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfStorage) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfStorage To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha6.NnfStorage)

	if err := Convert_v1alpha4_NnfStorage_To_v1alpha6_NnfStorage(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha6.NnfStorage{}
	hasAnno, err := utilconversion.UnmarshalData(src, restored)
	if err != nil {
		return err
	}

	if hasAnno {
		for i := range restored.Spec.AllocationSets {
			dst.Spec.AllocationSets[i].CommandVariables = make([]nnfv1alpha6.CommandVariablesSpec, len(restored.Spec.AllocationSets[i].CommandVariables))
			copy(dst.Spec.AllocationSets[i].CommandVariables, restored.Spec.AllocationSets[i].CommandVariables)
		}
	}

	return nil
}

func (dst *NnfStorage) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha6.NnfStorage)
	convertlog.Info("Convert NnfStorage From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha6_NnfStorage_To_v1alpha4_NnfStorage(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfStorageProfile) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfStorageProfile To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha6.NnfStorageProfile)

	if err := Convert_v1alpha4_NnfStorageProfile_To_v1alpha6_NnfStorageProfile(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha6.NnfStorageProfile{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	return nil
}

func (dst *NnfStorageProfile) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha6.NnfStorageProfile)
	convertlog.Info("Convert NnfStorageProfile From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha6_NnfStorageProfile_To_v1alpha4_NnfStorageProfile(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfSystemStorage) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfSystemStorage To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha6.NnfSystemStorage)

	if err := Convert_v1alpha4_NnfSystemStorage_To_v1alpha6_NnfSystemStorage(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha6.NnfSystemStorage{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	return nil
}

func (dst *NnfSystemStorage) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha6.NnfSystemStorage)
	convertlog.Info("Convert NnfSystemStorage From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha6_NnfSystemStorage_To_v1alpha4_NnfSystemStorage(src, dst, nil); err != nil {
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

func autoConvert_v1alpha2_ResourceError_To_v1alpha3_ResourceError(in *dwsv1alpha2.ResourceError, out *dwsv1alpha3.ResourceError, s apiconversion.Scope) error {
	out.Error = (*dwsv1alpha3.ResourceErrorInfo)(unsafe.Pointer(in.Error))
	return nil
}

// Convert_v1alpha2_ResourceError_To_v1alpha3_ResourceError is an autogenerated conversion function.
func Convert_v1alpha2_ResourceError_To_v1alpha3_ResourceError(in *dwsv1alpha2.ResourceError, out *dwsv1alpha3.ResourceError, s apiconversion.Scope) error {
	return autoConvert_v1alpha2_ResourceError_To_v1alpha3_ResourceError(in, out, s)
}

func autoConvert_v1alpha3_ResourceError_To_v1alpha2_ResourceError(in *dwsv1alpha3.ResourceError, out *dwsv1alpha2.ResourceError, s apiconversion.Scope) error {
	out.Error = (*dwsv1alpha2.ResourceErrorInfo)(unsafe.Pointer(in.Error))
	return nil
}

// Convert_v1alpha3_ResourceError_To_v1alpha2_ResourceError is an autogenerated conversion function.
func Convert_v1alpha3_ResourceError_To_v1alpha2_ResourceError(in *dwsv1alpha3.ResourceError, out *dwsv1alpha2.ResourceError, s apiconversion.Scope) error {
	return autoConvert_v1alpha3_ResourceError_To_v1alpha2_ResourceError(in, out, s)
}

func autoConvert_v1alpha2_ResourceErrorInfo_To_v1alpha3_ResourceErrorInfo(in *dwsv1alpha2.ResourceErrorInfo, out *dwsv1alpha3.ResourceErrorInfo, s apiconversion.Scope) error {
	out.UserMessage = in.UserMessage
	out.DebugMessage = in.DebugMessage
	out.Type = dwsv1alpha3.ResourceErrorType(in.Type)
	out.Severity = dwsv1alpha3.ResourceErrorSeverity(in.Severity)
	return nil
}

// Convert_v1alpha2_ResourceErrorInfo_To_v1alpha3_ResourceErrorInfo is an autogenerated conversion function.
func Convert_v1alpha2_ResourceErrorInfo_To_v1alpha3_ResourceErrorInfo(in *dwsv1alpha2.ResourceErrorInfo, out *dwsv1alpha3.ResourceErrorInfo, s apiconversion.Scope) error {
	return autoConvert_v1alpha2_ResourceErrorInfo_To_v1alpha3_ResourceErrorInfo(in, out, s)
}

func autoConvert_v1alpha3_ResourceErrorInfo_To_v1alpha2_ResourceErrorInfo(in *dwsv1alpha3.ResourceErrorInfo, out *dwsv1alpha2.ResourceErrorInfo, s apiconversion.Scope) error {
	out.UserMessage = in.UserMessage
	out.DebugMessage = in.DebugMessage
	out.Type = dwsv1alpha2.ResourceErrorType(in.Type)
	out.Severity = dwsv1alpha2.ResourceErrorSeverity(in.Severity)
	return nil
}

// Convert_v1alpha3_ResourceErrorInfo_To_v1alpha2_ResourceErrorInfo is an autogenerated conversion function.
func Convert_v1alpha3_ResourceErrorInfo_To_v1alpha2_ResourceErrorInfo(in *dwsv1alpha3.ResourceErrorInfo, out *dwsv1alpha2.ResourceErrorInfo, s apiconversion.Scope) error {
	return autoConvert_v1alpha3_ResourceErrorInfo_To_v1alpha2_ResourceErrorInfo(in, out, s)
}

// End of DWS ResourceError conversion routines.
// +crdbumper:carryforward:end

// The conversion-gen tool dropped these from zz_generated.conversion.go to
// force us to acknowledge that we are addressing the conversion requirements.

func Convert_v1alpha6_NnfNodeStorageSpec_To_v1alpha4_NnfNodeStorageSpec(in *nnfv1alpha6.NnfNodeStorageSpec, out *NnfNodeStorageSpec, s apiconversion.Scope) error {
	return autoConvert_v1alpha6_NnfNodeStorageSpec_To_v1alpha4_NnfNodeStorageSpec(in, out, s)
}

func Convert_v1alpha6_NnfStorageAllocationSetSpec_To_v1alpha4_NnfStorageAllocationSetSpec(in *nnfv1alpha6.NnfStorageAllocationSetSpec, out *NnfStorageAllocationSetSpec, s apiconversion.Scope) error {
	return autoConvert_v1alpha6_NnfStorageAllocationSetSpec_To_v1alpha4_NnfStorageAllocationSetSpec(in, out, s)
}
