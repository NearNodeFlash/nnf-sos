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

package v1alpha6

import (
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	dwsv1alpha7 "github.com/DataWorkflowServices/dws/api/v1alpha7"
	utilconversion "github.com/DataWorkflowServices/dws/github/cluster-api/util/conversion"
)

var convertlog = logf.Log.V(2).WithName("convert-v1alpha6")

func (src *ClientMount) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert ClientMount To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*dwsv1alpha7.ClientMount)

	if err := Convert_v1alpha6_ClientMount_To_v1alpha7_ClientMount(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &dwsv1alpha7.ClientMount{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	return nil
}

func (dst *ClientMount) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*dwsv1alpha7.ClientMount)
	convertlog.Info("Convert ClientMount From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha7_ClientMount_To_v1alpha6_ClientMount(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *Computes) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert Computes To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*dwsv1alpha7.Computes)

	if err := Convert_v1alpha6_Computes_To_v1alpha7_Computes(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &dwsv1alpha7.Computes{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	return nil
}

func (dst *Computes) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*dwsv1alpha7.Computes)
	convertlog.Info("Convert Computes From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha7_Computes_To_v1alpha6_Computes(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *DWDirectiveRule) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert DWDirectiveRule To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*dwsv1alpha7.DWDirectiveRule)

	if err := Convert_v1alpha6_DWDirectiveRule_To_v1alpha7_DWDirectiveRule(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &dwsv1alpha7.DWDirectiveRule{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	return nil
}

func (dst *DWDirectiveRule) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*dwsv1alpha7.DWDirectiveRule)
	convertlog.Info("Convert DWDirectiveRule From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha7_DWDirectiveRule_To_v1alpha6_DWDirectiveRule(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *DirectiveBreakdown) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert DirectiveBreakdown To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*dwsv1alpha7.DirectiveBreakdown)

	if err := Convert_v1alpha6_DirectiveBreakdown_To_v1alpha7_DirectiveBreakdown(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &dwsv1alpha7.DirectiveBreakdown{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	return nil
}

func (dst *DirectiveBreakdown) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*dwsv1alpha7.DirectiveBreakdown)
	convertlog.Info("Convert DirectiveBreakdown From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha7_DirectiveBreakdown_To_v1alpha6_DirectiveBreakdown(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *PersistentStorageInstance) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert PersistentStorageInstance To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*dwsv1alpha7.PersistentStorageInstance)

	if err := Convert_v1alpha6_PersistentStorageInstance_To_v1alpha7_PersistentStorageInstance(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &dwsv1alpha7.PersistentStorageInstance{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	return nil
}

func (dst *PersistentStorageInstance) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*dwsv1alpha7.PersistentStorageInstance)
	convertlog.Info("Convert PersistentStorageInstance From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha7_PersistentStorageInstance_To_v1alpha6_PersistentStorageInstance(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *Servers) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert Servers To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*dwsv1alpha7.Servers)

	if err := Convert_v1alpha6_Servers_To_v1alpha7_Servers(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &dwsv1alpha7.Servers{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	return nil
}

func (dst *Servers) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*dwsv1alpha7.Servers)
	convertlog.Info("Convert Servers From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha7_Servers_To_v1alpha6_Servers(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *Storage) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert Storage To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*dwsv1alpha7.Storage)

	if err := Convert_v1alpha6_Storage_To_v1alpha7_Storage(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &dwsv1alpha7.Storage{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	return nil
}

func (dst *Storage) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*dwsv1alpha7.Storage)
	convertlog.Info("Convert Storage From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha7_Storage_To_v1alpha6_Storage(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *SystemConfiguration) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert SystemConfiguration To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*dwsv1alpha7.SystemConfiguration)

	if err := Convert_v1alpha6_SystemConfiguration_To_v1alpha7_SystemConfiguration(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &dwsv1alpha7.SystemConfiguration{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	return nil
}

func (dst *SystemConfiguration) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*dwsv1alpha7.SystemConfiguration)
	convertlog.Info("Convert SystemConfiguration From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha7_SystemConfiguration_To_v1alpha6_SystemConfiguration(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *SystemStatus) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert SystemStatus To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*dwsv1alpha7.SystemStatus)

	if err := Convert_v1alpha6_SystemStatus_To_v1alpha7_SystemStatus(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &dwsv1alpha7.SystemStatus{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	return nil
}

func (dst *SystemStatus) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*dwsv1alpha7.SystemStatus)
	convertlog.Info("Convert SystemStatus From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha7_SystemStatus_To_v1alpha6_SystemStatus(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *Workflow) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert Workflow To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*dwsv1alpha7.Workflow)

	if err := Convert_v1alpha6_Workflow_To_v1alpha7_Workflow(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &dwsv1alpha7.Workflow{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	return nil
}

func (dst *Workflow) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*dwsv1alpha7.Workflow)
	convertlog.Info("Convert Workflow From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha7_Workflow_To_v1alpha6_Workflow(src, dst, nil); err != nil {
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
	return schema.GroupResource{Group: "dws", Resource: resource}
}

func (src *ClientMountList) ConvertTo(dstRaw conversion.Hub) error {
	return apierrors.NewMethodNotSupported(resource("ClientMountList"), "ConvertTo")
}

func (dst *ClientMountList) ConvertFrom(srcRaw conversion.Hub) error {
	return apierrors.NewMethodNotSupported(resource("ClientMountList"), "ConvertFrom")
}

func (src *ComputesList) ConvertTo(dstRaw conversion.Hub) error {
	return apierrors.NewMethodNotSupported(resource("ComputesList"), "ConvertTo")
}

func (dst *ComputesList) ConvertFrom(srcRaw conversion.Hub) error {
	return apierrors.NewMethodNotSupported(resource("ComputesList"), "ConvertFrom")
}

func (src *DWDirectiveRuleList) ConvertTo(dstRaw conversion.Hub) error {
	return apierrors.NewMethodNotSupported(resource("DWDirectiveRuleList"), "ConvertTo")
}

func (dst *DWDirectiveRuleList) ConvertFrom(srcRaw conversion.Hub) error {
	return apierrors.NewMethodNotSupported(resource("DWDirectiveRuleList"), "ConvertFrom")
}

func (src *DirectiveBreakdownList) ConvertTo(dstRaw conversion.Hub) error {
	return apierrors.NewMethodNotSupported(resource("DirectiveBreakdownList"), "ConvertTo")
}

func (dst *DirectiveBreakdownList) ConvertFrom(srcRaw conversion.Hub) error {
	return apierrors.NewMethodNotSupported(resource("DirectiveBreakdownList"), "ConvertFrom")
}

func (src *PersistentStorageInstanceList) ConvertTo(dstRaw conversion.Hub) error {
	return apierrors.NewMethodNotSupported(resource("PersistentStorageInstanceList"), "ConvertTo")
}

func (dst *PersistentStorageInstanceList) ConvertFrom(srcRaw conversion.Hub) error {
	return apierrors.NewMethodNotSupported(resource("PersistentStorageInstanceList"), "ConvertFrom")
}

func (src *ServersList) ConvertTo(dstRaw conversion.Hub) error {
	return apierrors.NewMethodNotSupported(resource("ServersList"), "ConvertTo")
}

func (dst *ServersList) ConvertFrom(srcRaw conversion.Hub) error {
	return apierrors.NewMethodNotSupported(resource("ServersList"), "ConvertFrom")
}

func (src *StorageList) ConvertTo(dstRaw conversion.Hub) error {
	return apierrors.NewMethodNotSupported(resource("StorageList"), "ConvertTo")
}

func (dst *StorageList) ConvertFrom(srcRaw conversion.Hub) error {
	return apierrors.NewMethodNotSupported(resource("StorageList"), "ConvertFrom")
}

func (src *SystemConfigurationList) ConvertTo(dstRaw conversion.Hub) error {
	return apierrors.NewMethodNotSupported(resource("SystemConfigurationList"), "ConvertTo")
}

func (dst *SystemConfigurationList) ConvertFrom(srcRaw conversion.Hub) error {
	return apierrors.NewMethodNotSupported(resource("SystemConfigurationList"), "ConvertFrom")
}

func (src *SystemStatusList) ConvertTo(dstRaw conversion.Hub) error {
	return apierrors.NewMethodNotSupported(resource("SystemStatusList"), "ConvertTo")
}

func (dst *SystemStatusList) ConvertFrom(srcRaw conversion.Hub) error {
	return apierrors.NewMethodNotSupported(resource("SystemStatusList"), "ConvertFrom")
}

func (src *WorkflowList) ConvertTo(dstRaw conversion.Hub) error {
	return apierrors.NewMethodNotSupported(resource("WorkflowList"), "ConvertTo")
}

func (dst *WorkflowList) ConvertFrom(srcRaw conversion.Hub) error {
	return apierrors.NewMethodNotSupported(resource("WorkflowList"), "ConvertFrom")
}
