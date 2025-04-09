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

package v1alpha5

import (
	unsafe "unsafe"

	mpicommonv1 "github.com/kubeflow/common/pkg/apis/common/v1"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apiconversion "k8s.io/apimachinery/pkg/conversion"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	dwsv1alpha2 "github.com/DataWorkflowServices/dws/api/v1alpha2"
	dwsv1alpha3 "github.com/DataWorkflowServices/dws/api/v1alpha3"
	nnfv1alpha7 "github.com/NearNodeFlash/nnf-sos/api/v1alpha7"
	utilconversion "github.com/NearNodeFlash/nnf-sos/github/cluster-api/util/conversion"
	mpiv2beta1 "github.com/kubeflow/mpi-operator/pkg/apis/kubeflow/v2beta1"
	corev1 "k8s.io/api/core/v1"
)

var convertlog = logf.Log.V(2).WithName("convert-v1alpha5")

func (src *NnfAccess) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfAccess To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha7.NnfAccess)

	if err := Convert_v1alpha5_NnfAccess_To_v1alpha7_NnfAccess(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha7.NnfAccess{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	return nil
}

func (dst *NnfAccess) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha7.NnfAccess)
	convertlog.Info("Convert NnfAccess From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha7_NnfAccess_To_v1alpha5_NnfAccess(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfContainerProfile) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfContainerProfile To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha7.NnfContainerProfile)

	if err := Convert_v1alpha5_NnfContainerProfile_To_v1alpha7_NnfContainerProfile(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha7.NnfContainerProfile{}
	hasAnno, err := utilconversion.UnmarshalData(src, restored)
	if err != nil {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	if hasAnno {
		if restored.Data.NnfSpec != nil {
			if dst.Data.NnfSpec == nil {
				dst.Data.NnfSpec = &nnfv1alpha7.NnfPodSpec{}
			}
			dst.Data.NnfSpec.Containers = append([]nnfv1alpha7.NnfContainer(nil), restored.Data.NnfSpec.Containers...)
			dst.Data.NnfSpec.InitContainers = append([]nnfv1alpha7.NnfContainer(nil), restored.Data.NnfSpec.InitContainers...)
			dst.Data.NnfSpec.Volumes = append([]corev1.Volume(nil), restored.Data.NnfSpec.Volumes...)
		}
		if restored.Data.NnfMPISpec != nil {
			if dst.Data.NnfMPISpec == nil {
				dst.Data.NnfMPISpec = &nnfv1alpha7.NnfMPISpec{}
			}

			dst.Data.NnfMPISpec.Launcher.Containers = append([]nnfv1alpha7.NnfContainer(nil), restored.Data.NnfMPISpec.Launcher.Containers...)
			dst.Data.NnfMPISpec.Launcher.InitContainers = append([]nnfv1alpha7.NnfContainer(nil), restored.Data.NnfMPISpec.Launcher.InitContainers...)
			dst.Data.NnfMPISpec.Launcher.Volumes = append([]corev1.Volume(nil), restored.Data.NnfMPISpec.Launcher.Volumes...)

			dst.Data.NnfMPISpec.Worker.Containers = append([]nnfv1alpha7.NnfContainer(nil), restored.Data.NnfMPISpec.Worker.Containers...)
			dst.Data.NnfMPISpec.Worker.InitContainers = append([]nnfv1alpha7.NnfContainer(nil), restored.Data.NnfMPISpec.Worker.InitContainers...)
			dst.Data.NnfMPISpec.Worker.Volumes = append([]corev1.Volume(nil), restored.Data.NnfMPISpec.Worker.Volumes...)

			dst.Data.NnfMPISpec.CopyOffload = restored.Data.NnfMPISpec.CopyOffload

			if restored.Data.NnfMPISpec.SlotsPerWorker != nil {
				if dst.Data.NnfMPISpec.SlotsPerWorker == nil {
					dst.Data.NnfMPISpec.SlotsPerWorker = new(int32)
				}
				*dst.Data.NnfMPISpec.SlotsPerWorker = *restored.Data.NnfMPISpec.SlotsPerWorker
			}
		}
	} else {
		if src.Data.Spec != nil {
			if dst.Data.NnfSpec == nil {
				dst.Data.NnfSpec = &nnfv1alpha7.NnfPodSpec{}
			}
			dst.Data.NnfSpec.FromCorePodSpec(src.Data.Spec)
		}
		if src.Data.MPISpec != nil {
			if dst.Data.NnfMPISpec == nil {
				dst.Data.NnfMPISpec = &nnfv1alpha7.NnfMPISpec{}
			}
			if src.Data.MPISpec.MPIReplicaSpecs[mpiv2beta1.MPIReplicaTypeLauncher] != nil {
				dst.Data.NnfMPISpec.Launcher.FromCorePodSpec(&src.Data.MPISpec.MPIReplicaSpecs[mpiv2beta1.MPIReplicaTypeLauncher].Template.Spec)
			}
			if src.Data.MPISpec.MPIReplicaSpecs[mpiv2beta1.MPIReplicaTypeWorker] != nil {
				dst.Data.NnfMPISpec.Worker.FromCorePodSpec(&src.Data.MPISpec.MPIReplicaSpecs[mpiv2beta1.MPIReplicaTypeWorker].Template.Spec)
			}

			dst.Data.NnfMPISpec.CopyOffload = src.Data.MPISpec.MPIReplicaSpecs[mpiv2beta1.MPIReplicaTypeLauncher].Template.Spec.ServiceAccountName == nnfv1alpha7.CopyOffloadServiceAccountName

			if src.Data.MPISpec.SlotsPerWorker != nil {
				if dst.Data.NnfMPISpec.SlotsPerWorker == nil {
					dst.Data.NnfMPISpec.SlotsPerWorker = new(int32)
				}
				*dst.Data.NnfMPISpec.SlotsPerWorker = *src.Data.MPISpec.SlotsPerWorker
			}
		}
	}

	return nil
}

func (dst *NnfContainerProfile) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha7.NnfContainerProfile)
	convertlog.Info("Convert NnfContainerProfile From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha7_NnfContainerProfile_To_v1alpha5_NnfContainerProfile(src, dst, nil); err != nil {
		return err
	}

	if src.Data.NnfSpec != nil {
		if dst.Data.Spec == nil {
			dst.Data.Spec = &corev1.PodSpec{}
		}
		dst.Data.Spec = src.Data.NnfSpec.ToCorePodSpec()
	}
	if src.Data.NnfMPISpec != nil {
		if dst.Data.MPISpec == nil {
			dst.Data.MPISpec = &mpiv2beta1.MPIJobSpec{}
		}

		if dst.Data.MPISpec.MPIReplicaSpecs == nil {
			dst.Data.MPISpec.MPIReplicaSpecs = make(map[mpiv2beta1.MPIReplicaType]*mpicommonv1.ReplicaSpec)
		}

		if dst.Data.MPISpec.MPIReplicaSpecs[mpiv2beta1.MPIReplicaTypeLauncher] == nil {
			dst.Data.MPISpec.MPIReplicaSpecs[mpiv2beta1.MPIReplicaTypeLauncher] = &mpicommonv1.ReplicaSpec{}
		}
		dst.Data.MPISpec.MPIReplicaSpecs[mpiv2beta1.MPIReplicaTypeLauncher].Template.Spec = *src.Data.NnfMPISpec.Launcher.ToCorePodSpec()

		if dst.Data.MPISpec != nil && dst.Data.MPISpec.MPIReplicaSpecs[mpiv2beta1.MPIReplicaTypeWorker] == nil {
			dst.Data.MPISpec.MPIReplicaSpecs[mpiv2beta1.MPIReplicaTypeWorker] = &mpicommonv1.ReplicaSpec{}
		}
		dst.Data.MPISpec.MPIReplicaSpecs[mpiv2beta1.MPIReplicaTypeWorker].Template.Spec = *src.Data.NnfMPISpec.Worker.ToCorePodSpec()

		if src.Data.NnfMPISpec.CopyOffload {
			dst.Data.MPISpec.MPIReplicaSpecs[mpiv2beta1.MPIReplicaTypeLauncher].Template.Spec.ServiceAccountName = nnfv1alpha7.CopyOffloadServiceAccountName
		}

		if src.Data.NnfMPISpec.SlotsPerWorker != nil {
			if dst.Data.MPISpec.SlotsPerWorker == nil {
				dst.Data.MPISpec.SlotsPerWorker = new(int32)
			}
			*dst.Data.MPISpec.SlotsPerWorker = *src.Data.NnfMPISpec.SlotsPerWorker
		}
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfDataMovement) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfDataMovement To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha7.NnfDataMovement)

	if err := Convert_v1alpha5_NnfDataMovement_To_v1alpha7_NnfDataMovement(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha7.NnfDataMovement{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	return nil
}

func (dst *NnfDataMovement) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha7.NnfDataMovement)
	convertlog.Info("Convert NnfDataMovement From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha7_NnfDataMovement_To_v1alpha5_NnfDataMovement(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfDataMovementManager) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfDataMovementManager To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha7.NnfDataMovementManager)

	if err := Convert_v1alpha5_NnfDataMovementManager_To_v1alpha7_NnfDataMovementManager(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha7.NnfDataMovementManager{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	return nil
}

func (dst *NnfDataMovementManager) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha7.NnfDataMovementManager)
	convertlog.Info("Convert NnfDataMovementManager From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha7_NnfDataMovementManager_To_v1alpha5_NnfDataMovementManager(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfDataMovementProfile) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfDataMovementProfile To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha7.NnfDataMovementProfile)

	if err := Convert_v1alpha5_NnfDataMovementProfile_To_v1alpha7_NnfDataMovementProfile(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha7.NnfDataMovementProfile{}
	hasAnno, err := utilconversion.UnmarshalData(src, restored)
	if err != nil {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.
	if hasAnno {
		dst.Data.SetprivCommand = restored.Data.SetprivCommand
	}

	return nil
}

func (dst *NnfDataMovementProfile) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha7.NnfDataMovementProfile)
	convertlog.Info("Convert NnfDataMovementProfile From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha7_NnfDataMovementProfile_To_v1alpha5_NnfDataMovementProfile(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfLustreMGT) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfLustreMGT To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha7.NnfLustreMGT)

	if err := Convert_v1alpha5_NnfLustreMGT_To_v1alpha7_NnfLustreMGT(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha7.NnfLustreMGT{}
	hasAnno, err := utilconversion.UnmarshalData(src, restored)
	if err != nil {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.
	if hasAnno {
		dst.Status.CommandList = append([]nnfv1alpha7.NnfLustreMGTStatusCommand(nil), restored.Status.CommandList...)
		dst.Spec.CommandList = append([]nnfv1alpha7.NnfLustreMGTSpecCommand(nil), restored.Spec.CommandList...)
	}

	return nil
}

func (dst *NnfLustreMGT) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha7.NnfLustreMGT)
	convertlog.Info("Convert NnfLustreMGT From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha7_NnfLustreMGT_To_v1alpha5_NnfLustreMGT(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfNode) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfNode To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha7.NnfNode)

	if err := Convert_v1alpha5_NnfNode_To_v1alpha7_NnfNode(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha7.NnfNode{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	return nil
}

func (dst *NnfNode) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha7.NnfNode)
	convertlog.Info("Convert NnfNode From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha7_NnfNode_To_v1alpha5_NnfNode(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfNodeBlockStorage) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfNodeBlockStorage To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha7.NnfNodeBlockStorage)

	if err := Convert_v1alpha5_NnfNodeBlockStorage_To_v1alpha7_NnfNodeBlockStorage(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha7.NnfNodeBlockStorage{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	return nil
}

func (dst *NnfNodeBlockStorage) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha7.NnfNodeBlockStorage)
	convertlog.Info("Convert NnfNodeBlockStorage From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha7_NnfNodeBlockStorage_To_v1alpha5_NnfNodeBlockStorage(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfNodeECData) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfNodeECData To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha7.NnfNodeECData)

	if err := Convert_v1alpha5_NnfNodeECData_To_v1alpha7_NnfNodeECData(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha7.NnfNodeECData{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	return nil
}

func (dst *NnfNodeECData) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha7.NnfNodeECData)
	convertlog.Info("Convert NnfNodeECData From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha7_NnfNodeECData_To_v1alpha5_NnfNodeECData(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfNodeStorage) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfNodeStorage To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha7.NnfNodeStorage)

	if err := Convert_v1alpha5_NnfNodeStorage_To_v1alpha7_NnfNodeStorage(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha7.NnfNodeStorage{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	return nil
}

func (dst *NnfNodeStorage) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha7.NnfNodeStorage)
	convertlog.Info("Convert NnfNodeStorage From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha7_NnfNodeStorage_To_v1alpha5_NnfNodeStorage(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfPortManager) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfPortManager To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha7.NnfPortManager)

	if err := Convert_v1alpha5_NnfPortManager_To_v1alpha7_NnfPortManager(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha7.NnfPortManager{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	return nil
}

func (dst *NnfPortManager) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha7.NnfPortManager)
	convertlog.Info("Convert NnfPortManager From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha7_NnfPortManager_To_v1alpha5_NnfPortManager(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfStorage) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfStorage To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha7.NnfStorage)

	if err := Convert_v1alpha5_NnfStorage_To_v1alpha7_NnfStorage(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha7.NnfStorage{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	return nil
}

func (dst *NnfStorage) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha7.NnfStorage)
	convertlog.Info("Convert NnfStorage From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha7_NnfStorage_To_v1alpha5_NnfStorage(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfStorageProfile) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfStorageProfile To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha7.NnfStorageProfile)

	if err := Convert_v1alpha5_NnfStorageProfile_To_v1alpha7_NnfStorageProfile(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha7.NnfStorageProfile{}
	hasAnno, err := utilconversion.UnmarshalData(src, restored)
	if err != nil {
		return err
	}

	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.
	if hasAnno {
		dst.Data.LustreStorage.PreMountMGTCmds = append([]string(nil), restored.Data.LustreStorage.PreMountMGTCmds...)
		dst.Data.LustreStorage.ClientCmdLines.RabbitPostMount = restored.Data.LustreStorage.ClientCmdLines.RabbitPostMount
		dst.Data.LustreStorage.ClientCmdLines.RabbitPreUnmount = restored.Data.LustreStorage.ClientCmdLines.RabbitPreUnmount
		dst.Data.LustreStorage.ClientCmdLines.MountRabbit = restored.Data.LustreStorage.ClientCmdLines.MountRabbit
		dst.Data.LustreStorage.ClientCmdLines.MountCompute = restored.Data.LustreStorage.ClientCmdLines.MountCompute
	} else {
		dst.Data.LustreStorage.ClientCmdLines.RabbitPostMount = src.Data.LustreStorage.OstCmdLines.PostMount
		dst.Data.LustreStorage.ClientCmdLines.RabbitPreUnmount = src.Data.LustreStorage.OstCmdLines.PreUnmount
		dst.Data.LustreStorage.ClientCmdLines.MountRabbit = src.Data.LustreStorage.MountRabbit
		dst.Data.LustreStorage.ClientCmdLines.MountCompute = src.Data.LustreStorage.MountCompute
	}

	return nil
}

func (dst *NnfStorageProfile) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha7.NnfStorageProfile)
	convertlog.Info("Convert NnfStorageProfile From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha7_NnfStorageProfile_To_v1alpha5_NnfStorageProfile(src, dst, nil); err != nil {
		return err
	}

	dst.Data.LustreStorage.MountRabbit = src.Data.LustreStorage.ClientCmdLines.MountRabbit
	dst.Data.LustreStorage.MountCompute = src.Data.LustreStorage.ClientCmdLines.MountCompute
	dst.Data.LustreStorage.OstCmdLines.PostMount = src.Data.LustreStorage.ClientCmdLines.RabbitPostMount
	dst.Data.LustreStorage.OstCmdLines.PreUnmount = src.Data.LustreStorage.ClientCmdLines.RabbitPreUnmount

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfSystemStorage) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfSystemStorage To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha7.NnfSystemStorage)

	if err := Convert_v1alpha5_NnfSystemStorage_To_v1alpha7_NnfSystemStorage(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha7.NnfSystemStorage{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	return nil
}

func (dst *NnfSystemStorage) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha7.NnfSystemStorage)
	convertlog.Info("Convert NnfSystemStorage From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha7_NnfSystemStorage_To_v1alpha5_NnfSystemStorage(src, dst, nil); err != nil {
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

func Convert_v1alpha7_NnfLustreMGTSpec_To_v1alpha5_NnfLustreMGTSpec(in *nnfv1alpha7.NnfLustreMGTSpec, out *NnfLustreMGTSpec, s apiconversion.Scope) error {
	return autoConvert_v1alpha7_NnfLustreMGTSpec_To_v1alpha5_NnfLustreMGTSpec(in, out, s)
}

func Convert_v1alpha7_NnfLustreMGTStatus_To_v1alpha5_NnfLustreMGTStatus(in *nnfv1alpha7.NnfLustreMGTStatus, out *NnfLustreMGTStatus, s apiconversion.Scope) error {
	return autoConvert_v1alpha7_NnfLustreMGTStatus_To_v1alpha5_NnfLustreMGTStatus(in, out, s)
}

func Convert_v1alpha7_NnfStorageProfileLustreData_To_v1alpha5_NnfStorageProfileLustreData(in *nnfv1alpha7.NnfStorageProfileLustreData, out *NnfStorageProfileLustreData, s apiconversion.Scope) error {
	return autoConvert_v1alpha7_NnfStorageProfileLustreData_To_v1alpha5_NnfStorageProfileLustreData(in, out, s)
}

func Convert_v1alpha5_NnfStorageProfileLustreData_To_v1alpha7_NnfStorageProfileLustreData(in *NnfStorageProfileLustreData, out *nnfv1alpha7.NnfStorageProfileLustreData, s apiconversion.Scope) error {
	return autoConvert_v1alpha5_NnfStorageProfileLustreData_To_v1alpha7_NnfStorageProfileLustreData(in, out, s)
}

func Convert_v1alpha5_NnfStorageProfileLustreCmdLines_To_v1alpha7_NnfStorageProfileLustreCmdLines(in *NnfStorageProfileLustreCmdLines, out *nnfv1alpha7.NnfStorageProfileLustreCmdLines, s apiconversion.Scope) error {
	return autoConvert_v1alpha5_NnfStorageProfileLustreCmdLines_To_v1alpha7_NnfStorageProfileLustreCmdLines(in, out, s)
}

func Convert_v1alpha7_NnfDataMovementProfileData_To_v1alpha5_NnfDataMovementProfileData(in *nnfv1alpha7.NnfDataMovementProfileData, out *NnfDataMovementProfileData, s apiconversion.Scope) error {
	return autoConvert_v1alpha7_NnfDataMovementProfileData_To_v1alpha5_NnfDataMovementProfileData(in, out, s)
}

func Convert_v1alpha5_NnfContainerProfileData_To_v1alpha7_NnfContainerProfileData(in *NnfContainerProfileData, out *nnfv1alpha7.NnfContainerProfileData, s apiconversion.Scope) error {
	return autoConvert_v1alpha5_NnfContainerProfileData_To_v1alpha7_NnfContainerProfileData(in, out, s)
}

func Convert_v1alpha7_NnfContainerProfileData_To_v1alpha5_NnfContainerProfileData(in *nnfv1alpha7.NnfContainerProfileData, out *NnfContainerProfileData, s apiconversion.Scope) error {
	return autoConvert_v1alpha7_NnfContainerProfileData_To_v1alpha5_NnfContainerProfileData(in, out, s)
}
