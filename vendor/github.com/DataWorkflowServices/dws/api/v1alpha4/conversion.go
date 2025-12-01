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

package v1alpha4

import (
	"os"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apiconversion "k8s.io/apimachinery/pkg/conversion"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	dwsv1alpha7 "github.com/DataWorkflowServices/dws/api/v1alpha7"
	utilconversion "github.com/DataWorkflowServices/dws/github/cluster-api/util/conversion"
)

var convertlog = logf.Log.V(2).WithName("convert-v1alpha4")

func (src *ClientMount) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert ClientMount To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*dwsv1alpha7.ClientMount)

	if err := Convert_v1alpha4_ClientMount_To_v1alpha7_ClientMount(src, dst, nil); err != nil {
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

	if err := Convert_v1alpha7_ClientMount_To_v1alpha4_ClientMount(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *Computes) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert Computes To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*dwsv1alpha7.Computes)

	if err := Convert_v1alpha4_Computes_To_v1alpha7_Computes(src, dst, nil); err != nil {
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

	if err := Convert_v1alpha7_Computes_To_v1alpha4_Computes(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *DWDirectiveRule) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert DWDirectiveRule To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*dwsv1alpha7.DWDirectiveRule)

	if err := Convert_v1alpha4_DWDirectiveRule_To_v1alpha7_DWDirectiveRule(src, dst, nil); err != nil {
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

	if err := Convert_v1alpha7_DWDirectiveRule_To_v1alpha4_DWDirectiveRule(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *DirectiveBreakdown) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert DirectiveBreakdown To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*dwsv1alpha7.DirectiveBreakdown)

	if err := Convert_v1alpha4_DirectiveBreakdown_To_v1alpha7_DirectiveBreakdown(src, dst, nil); err != nil {
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

	if err := Convert_v1alpha7_DirectiveBreakdown_To_v1alpha4_DirectiveBreakdown(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *PersistentStorageInstance) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert PersistentStorageInstance To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*dwsv1alpha7.PersistentStorageInstance)

	if err := Convert_v1alpha4_PersistentStorageInstance_To_v1alpha7_PersistentStorageInstance(src, dst, nil); err != nil {
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

	if err := Convert_v1alpha7_PersistentStorageInstance_To_v1alpha4_PersistentStorageInstance(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *Servers) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert Servers To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*dwsv1alpha7.Servers)

	if err := Convert_v1alpha4_Servers_To_v1alpha7_Servers(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &dwsv1alpha7.Servers{}
	hasAnno, err := utilconversion.UnmarshalData(src, restored)
	if err != nil {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	if hasAnno {
		for index := range dst.Status.AllocationSets {
			if len(restored.Status.AllocationSets) < index {
				break
			}

			for name := range dst.Status.AllocationSets[index].Storage {
				if serverStatus, exists := restored.Status.AllocationSets[index].Storage[name]; exists {
					serverStatus.AllocationSize = dst.Status.AllocationSets[index].Storage[name].AllocationSize
					dst.Status.AllocationSets[index].Storage[name] = serverStatus
				}
			}
		}
	}

	return nil
}

func (dst *Servers) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*dwsv1alpha7.Servers)
	convertlog.Info("Convert Servers From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha7_Servers_To_v1alpha4_Servers(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *Storage) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert Storage To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*dwsv1alpha7.Storage)

	if err := Convert_v1alpha4_Storage_To_v1alpha7_Storage(src, dst, nil); err != nil {
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

	if err := Convert_v1alpha7_Storage_To_v1alpha4_Storage(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *SystemConfiguration) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert SystemConfiguration To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*dwsv1alpha7.SystemConfiguration)

	if err := Convert_v1alpha4_SystemConfiguration_To_v1alpha7_SystemConfiguration(src, dst, nil); err != nil {
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

	if err := Convert_v1alpha7_SystemConfiguration_To_v1alpha4_SystemConfiguration(src, dst, nil); err != nil {
		return err
	}

	// +crdbumper:carryforward:begin="SystemConfiguration.ConvertFrom"
	// In a non-test environment, ENVIRONMENT will be set. Don't save Hub data in the
	// annotations in this case. The SystemConfiguration resource can be very large, and
	// the annotation will be too large to store. In a test environment, we want the Hub
	// data saved in the annotations to test hub-spoke-hub and spoke-hub-spoke conversions.
	if _, found := os.LookupEnv("ENVIRONMENT"); found {
		return nil
	}
	// +crdbumper:carryforward:end

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *SystemStatus) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert SystemStatus To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*dwsv1alpha7.SystemStatus)

	if err := Convert_v1alpha4_SystemStatus_To_v1alpha7_SystemStatus(src, dst, nil); err != nil {
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

	if err := Convert_v1alpha7_SystemStatus_To_v1alpha4_SystemStatus(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *Workflow) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert Workflow To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*dwsv1alpha7.Workflow)

	if err := Convert_v1alpha4_Workflow_To_v1alpha7_Workflow(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &dwsv1alpha7.Workflow{}
	hasAnno, err := utilconversion.UnmarshalData(src, restored)
	if err != nil {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	if hasAnno {
		dst.Spec.ForceReady = restored.Spec.ForceReady
	}

	return nil
}

func (dst *Workflow) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*dwsv1alpha7.Workflow)
	convertlog.Info("Convert Workflow From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha7_Workflow_To_v1alpha4_Workflow(src, dst, nil); err != nil {
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

// The conversion-gen tool dropped these from zz_generated.conversion.go to
// force us to acknowledge that we are addressing the conversion requirements.

func Convert_v1alpha7_ServersStatusStorage_To_v1alpha4_ServersStatusStorage(in *dwsv1alpha7.ServersStatusStorage, out *ServersStatusStorage, s apiconversion.Scope) error {
	return autoConvert_v1alpha7_ServersStatusStorage_To_v1alpha4_ServersStatusStorage(in, out, s)
}

func Convert_v1alpha7_WorkflowSpec_To_v1alpha4_WorkflowSpec(in *dwsv1alpha7.WorkflowSpec, out *WorkflowSpec, s apiconversion.Scope) error {
	return autoConvert_v1alpha7_WorkflowSpec_To_v1alpha4_WorkflowSpec(in, out, s)
}

// convertResourceStatusFromHub converts a v1alpha7 ResourceStatus to v1alpha4.
// FencedStatus is mapped to OfflineStatus since Fenced doesn't exist in older API versions.
func convertResourceStatusFromHub(in dwsv1alpha7.ResourceStatus) ResourceStatus {
	if in == dwsv1alpha7.FencedStatus {
		return OfflineStatus
	}
	return ResourceStatus(in)
}

// Convert_v1alpha7_Node_To_v1alpha4_Node handles conversion with FencedStatus mapping.
func Convert_v1alpha7_Node_To_v1alpha4_Node(in *dwsv1alpha7.Node, out *Node, s apiconversion.Scope) error {
	if err := autoConvert_v1alpha7_Node_To_v1alpha4_Node(in, out, s); err != nil {
		return err
	}
	out.Status = convertResourceStatusFromHub(in.Status)
	return nil
}

// Convert_v1alpha7_StorageDevice_To_v1alpha4_StorageDevice handles conversion with FencedStatus mapping.
func Convert_v1alpha7_StorageDevice_To_v1alpha4_StorageDevice(in *dwsv1alpha7.StorageDevice, out *StorageDevice, s apiconversion.Scope) error {
	if err := autoConvert_v1alpha7_StorageDevice_To_v1alpha4_StorageDevice(in, out, s); err != nil {
		return err
	}
	out.Status = convertResourceStatusFromHub(in.Status)
	return nil
}

// Convert_v1alpha7_StorageStatus_To_v1alpha4_StorageStatus handles conversion with FencedStatus mapping.
func Convert_v1alpha7_StorageStatus_To_v1alpha4_StorageStatus(in *dwsv1alpha7.StorageStatus, out *StorageStatus, s apiconversion.Scope) error {
	if err := autoConvert_v1alpha7_StorageStatus_To_v1alpha4_StorageStatus(in, out, s); err != nil {
		return err
	}
	out.Status = convertResourceStatusFromHub(in.Status)
	return nil
}
