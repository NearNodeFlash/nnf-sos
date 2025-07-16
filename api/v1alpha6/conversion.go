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
	unsafe "unsafe"

	mpicommonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	mpiv2beta1 "github.com/kubeflow/mpi-operator/pkg/apis/kubeflow/v2beta1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apiconversion "k8s.io/apimachinery/pkg/conversion"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	dwsv1alpha3 "github.com/DataWorkflowServices/dws/api/v1alpha3"
	dwsv1alpha5 "github.com/DataWorkflowServices/dws/api/v1alpha5"
	nnfv1alpha8 "github.com/NearNodeFlash/nnf-sos/api/v1alpha8"
	v1alpha8 "github.com/NearNodeFlash/nnf-sos/api/v1alpha8"
	utilconversion "github.com/NearNodeFlash/nnf-sos/github/cluster-api/util/conversion"
)

var convertlog = logf.Log.V(2).WithName("convert-v1alpha6")

func (src *NnfAccess) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfAccess To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha8.NnfAccess)

	if err := Convert_v1alpha6_NnfAccess_To_v1alpha8_NnfAccess(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha8.NnfAccess{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	return nil
}

func (dst *NnfAccess) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha8.NnfAccess)
	convertlog.Info("Convert NnfAccess From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha8_NnfAccess_To_v1alpha6_NnfAccess(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfContainerProfile) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfContainerProfile To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha8.NnfContainerProfile)

	if err := Convert_v1alpha6_NnfContainerProfile_To_v1alpha8_NnfContainerProfile(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha8.NnfContainerProfile{}
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
				dst.Data.NnfSpec = &nnfv1alpha8.NnfPodSpec{}
			}
			dst.Data.NnfSpec.Containers = append([]nnfv1alpha8.NnfContainer(nil), restored.Data.NnfSpec.Containers...)
			dst.Data.NnfSpec.InitContainers = append([]nnfv1alpha8.NnfContainer(nil), restored.Data.NnfSpec.InitContainers...)
			dst.Data.NnfSpec.Volumes = append([]corev1.Volume(nil), restored.Data.NnfSpec.Volumes...)

			if restored.Data.NnfSpec.TerminationGracePeriodSeconds != nil {
				dst.Data.NnfSpec.TerminationGracePeriodSeconds = pointer.Int64(*restored.Data.NnfSpec.TerminationGracePeriodSeconds)
			}

			if restored.Data.NnfSpec.ShareProcessNamespace != nil {
				dst.Data.NnfSpec.ShareProcessNamespace = pointer.Bool(*restored.Data.NnfSpec.ShareProcessNamespace)
			}

			dst.Data.NnfSpec.ImagePullSecrets = append([]corev1.LocalObjectReference(nil), restored.Data.NnfSpec.ImagePullSecrets...)

			if restored.Data.NnfSpec.AutomountServiceAccountToken != nil {
				dst.Data.NnfSpec.AutomountServiceAccountToken = pointer.Bool(*restored.Data.NnfSpec.AutomountServiceAccountToken)
			}

		}
		if restored.Data.NnfMPISpec != nil {
			if dst.Data.NnfMPISpec == nil {
				dst.Data.NnfMPISpec = &nnfv1alpha8.NnfMPISpec{}
			}

			dst.Data.NnfMPISpec.Launcher.Containers = append([]nnfv1alpha8.NnfContainer(nil), restored.Data.NnfMPISpec.Launcher.Containers...)
			dst.Data.NnfMPISpec.Launcher.InitContainers = append([]nnfv1alpha8.NnfContainer(nil), restored.Data.NnfMPISpec.Launcher.InitContainers...)
			dst.Data.NnfMPISpec.Launcher.Volumes = append([]corev1.Volume(nil), restored.Data.NnfMPISpec.Launcher.Volumes...)
			dst.Data.NnfMPISpec.Launcher.TerminationGracePeriodSeconds = restored.Data.NnfMPISpec.Launcher.TerminationGracePeriodSeconds
			dst.Data.NnfMPISpec.Launcher.ShareProcessNamespace = restored.Data.NnfMPISpec.Launcher.ShareProcessNamespace
			dst.Data.NnfMPISpec.Launcher.ImagePullSecrets = append([]corev1.LocalObjectReference(nil), restored.Data.NnfMPISpec.Launcher.ImagePullSecrets...)
			dst.Data.NnfMPISpec.Launcher.AutomountServiceAccountToken = restored.Data.NnfMPISpec.Launcher.AutomountServiceAccountToken

			dst.Data.NnfMPISpec.Worker.Containers = append([]nnfv1alpha8.NnfContainer(nil), restored.Data.NnfMPISpec.Worker.Containers...)
			dst.Data.NnfMPISpec.Worker.InitContainers = append([]nnfv1alpha8.NnfContainer(nil), restored.Data.NnfMPISpec.Worker.InitContainers...)
			dst.Data.NnfMPISpec.Worker.Volumes = append([]corev1.Volume(nil), restored.Data.NnfMPISpec.Worker.Volumes...)
			dst.Data.NnfMPISpec.Worker.TerminationGracePeriodSeconds = restored.Data.NnfMPISpec.Worker.TerminationGracePeriodSeconds
			dst.Data.NnfMPISpec.Worker.ShareProcessNamespace = restored.Data.NnfMPISpec.Worker.ShareProcessNamespace
			dst.Data.NnfMPISpec.Worker.ImagePullSecrets = append([]corev1.LocalObjectReference(nil), restored.Data.NnfMPISpec.Worker.ImagePullSecrets...)
			dst.Data.NnfMPISpec.Worker.AutomountServiceAccountToken = restored.Data.NnfMPISpec.Worker.AutomountServiceAccountToken

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
				dst.Data.NnfSpec = &nnfv1alpha8.NnfPodSpec{}
			}
			dst.Data.NnfSpec.FromCorePodSpec(src.Data.Spec)
		}
		if src.Data.MPISpec != nil {
			if dst.Data.NnfMPISpec == nil {
				dst.Data.NnfMPISpec = &nnfv1alpha8.NnfMPISpec{}
			}
			if src.Data.MPISpec.MPIReplicaSpecs[mpiv2beta1.MPIReplicaTypeLauncher] != nil {
				dst.Data.NnfMPISpec.Launcher.FromCorePodSpec(&src.Data.MPISpec.MPIReplicaSpecs[mpiv2beta1.MPIReplicaTypeLauncher].Template.Spec)
			}
			if src.Data.MPISpec.MPIReplicaSpecs[mpiv2beta1.MPIReplicaTypeWorker] != nil {
				dst.Data.NnfMPISpec.Worker.FromCorePodSpec(&src.Data.MPISpec.MPIReplicaSpecs[mpiv2beta1.MPIReplicaTypeWorker].Template.Spec)
			}

			dst.Data.NnfMPISpec.CopyOffload = src.Data.MPISpec.MPIReplicaSpecs[mpiv2beta1.MPIReplicaTypeLauncher].Template.Spec.ServiceAccountName == nnfv1alpha8.CopyOffloadServiceAccountName

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
	src := srcRaw.(*nnfv1alpha8.NnfContainerProfile)
	convertlog.Info("Convert NnfContainerProfile From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha8_NnfContainerProfile_To_v1alpha6_NnfContainerProfile(src, dst, nil); err != nil {
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
			dst.Data.MPISpec.MPIReplicaSpecs[mpiv2beta1.MPIReplicaTypeLauncher].Template.Spec.ServiceAccountName = nnfv1alpha8.CopyOffloadServiceAccountName
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
	dst := dstRaw.(*nnfv1alpha8.NnfDataMovement)

	if err := Convert_v1alpha6_NnfDataMovement_To_v1alpha8_NnfDataMovement(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha8.NnfDataMovement{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	return nil
}

func (dst *NnfDataMovement) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha8.NnfDataMovement)
	convertlog.Info("Convert NnfDataMovement From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha8_NnfDataMovement_To_v1alpha6_NnfDataMovement(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfDataMovementManager) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfDataMovementManager To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha8.NnfDataMovementManager)

	if err := Convert_v1alpha6_NnfDataMovementManager_To_v1alpha8_NnfDataMovementManager(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha8.NnfDataMovementManager{}
	hasAnno, err := utilconversion.UnmarshalData(src, restored)
	if err != nil {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	if hasAnno {
		dst.Spec.PodSpec.Containers = append([]nnfv1alpha8.NnfContainer(nil), restored.Spec.PodSpec.Containers...)
		dst.Spec.PodSpec.InitContainers = append([]nnfv1alpha8.NnfContainer(nil), restored.Spec.PodSpec.InitContainers...)
		dst.Spec.PodSpec.Volumes = append([]corev1.Volume(nil), restored.Spec.PodSpec.Volumes...)

		if restored.Spec.PodSpec.TerminationGracePeriodSeconds != nil {
			dst.Spec.PodSpec.TerminationGracePeriodSeconds = pointer.Int64(*restored.Spec.PodSpec.TerminationGracePeriodSeconds)
		}

		if restored.Spec.PodSpec.ShareProcessNamespace != nil {
			dst.Spec.PodSpec.ShareProcessNamespace = pointer.Bool(*restored.Spec.PodSpec.ShareProcessNamespace)
		}

		dst.Spec.PodSpec.ImagePullSecrets = append([]corev1.LocalObjectReference(nil), restored.Spec.PodSpec.ImagePullSecrets...)

		if restored.Spec.PodSpec.AutomountServiceAccountToken != nil {
			dst.Spec.PodSpec.AutomountServiceAccountToken = pointer.Bool(*restored.Spec.PodSpec.AutomountServiceAccountToken)
		}

	} else {
		dst.Spec.PodSpec.FromCorePodSpec(&src.Spec.Template.Spec)
	}

	return nil
}

func (dst *NnfDataMovementManager) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha8.NnfDataMovementManager)
	convertlog.Info("Convert NnfDataMovementManager From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha8_NnfDataMovementManager_To_v1alpha6_NnfDataMovementManager(src, dst, nil); err != nil {
		return err
	}

	dst.Spec.Template.Spec = *src.Spec.PodSpec.ToCorePodSpec()

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfDataMovementProfile) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfDataMovementProfile To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha8.NnfDataMovementProfile)

	if err := Convert_v1alpha6_NnfDataMovementProfile_To_v1alpha8_NnfDataMovementProfile(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha8.NnfDataMovementProfile{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	return nil
}

func (dst *NnfDataMovementProfile) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha8.NnfDataMovementProfile)
	convertlog.Info("Convert NnfDataMovementProfile From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha8_NnfDataMovementProfile_To_v1alpha6_NnfDataMovementProfile(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfLustreMGT) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfLustreMGT To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha8.NnfLustreMGT)

	if err := Convert_v1alpha6_NnfLustreMGT_To_v1alpha8_NnfLustreMGT(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha8.NnfLustreMGT{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	return nil
}

func (dst *NnfLustreMGT) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha8.NnfLustreMGT)
	convertlog.Info("Convert NnfLustreMGT From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha8_NnfLustreMGT_To_v1alpha6_NnfLustreMGT(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfNode) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfNode To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha8.NnfNode)

	if err := Convert_v1alpha6_NnfNode_To_v1alpha8_NnfNode(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha8.NnfNode{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	return nil
}

func (dst *NnfNode) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha8.NnfNode)
	convertlog.Info("Convert NnfNode From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha8_NnfNode_To_v1alpha6_NnfNode(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfNodeBlockStorage) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfNodeBlockStorage To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha8.NnfNodeBlockStorage)

	if err := Convert_v1alpha6_NnfNodeBlockStorage_To_v1alpha8_NnfNodeBlockStorage(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha8.NnfNodeBlockStorage{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	return nil
}

func (dst *NnfNodeBlockStorage) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha8.NnfNodeBlockStorage)
	convertlog.Info("Convert NnfNodeBlockStorage From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha8_NnfNodeBlockStorage_To_v1alpha6_NnfNodeBlockStorage(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfNodeECData) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfNodeECData To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha8.NnfNodeECData)

	if err := Convert_v1alpha6_NnfNodeECData_To_v1alpha8_NnfNodeECData(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha8.NnfNodeECData{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	return nil
}

func (dst *NnfNodeECData) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha8.NnfNodeECData)
	convertlog.Info("Convert NnfNodeECData From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha8_NnfNodeECData_To_v1alpha6_NnfNodeECData(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfNodeStorage) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfNodeStorage To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha8.NnfNodeStorage)

	if err := Convert_v1alpha6_NnfNodeStorage_To_v1alpha8_NnfNodeStorage(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha8.NnfNodeStorage{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	return nil
}

func (dst *NnfNodeStorage) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha8.NnfNodeStorage)
	convertlog.Info("Convert NnfNodeStorage From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha8_NnfNodeStorage_To_v1alpha6_NnfNodeStorage(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfPortManager) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfPortManager To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha8.NnfPortManager)

	if err := Convert_v1alpha6_NnfPortManager_To_v1alpha8_NnfPortManager(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha8.NnfPortManager{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	return nil
}

func (dst *NnfPortManager) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha8.NnfPortManager)
	convertlog.Info("Convert NnfPortManager From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha8_NnfPortManager_To_v1alpha6_NnfPortManager(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfStorage) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfStorage To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha8.NnfStorage)

	if err := Convert_v1alpha6_NnfStorage_To_v1alpha8_NnfStorage(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha8.NnfStorage{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	return nil
}

func (dst *NnfStorage) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha8.NnfStorage)
	convertlog.Info("Convert NnfStorage From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha8_NnfStorage_To_v1alpha6_NnfStorage(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfStorageProfile) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfStorageProfile To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha8.NnfStorageProfile)

	if err := Convert_v1alpha6_NnfStorageProfile_To_v1alpha8_NnfStorageProfile(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha8.NnfStorageProfile{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	return nil
}

func (dst *NnfStorageProfile) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha8.NnfStorageProfile)
	convertlog.Info("Convert NnfStorageProfile From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha8_NnfStorageProfile_To_v1alpha6_NnfStorageProfile(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion except for metadata.
	return utilconversion.MarshalData(src, dst)
}

func (src *NnfSystemStorage) ConvertTo(dstRaw conversion.Hub) error {
	convertlog.Info("Convert NnfSystemStorage To Hub", "name", src.GetName(), "namespace", src.GetNamespace())
	dst := dstRaw.(*nnfv1alpha8.NnfSystemStorage)

	if err := Convert_v1alpha6_NnfSystemStorage_To_v1alpha8_NnfSystemStorage(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &nnfv1alpha8.NnfSystemStorage{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	// EDIT THIS FUNCTION! If the annotation is holding anything that is
	// hub-specific then copy it into 'dst' from 'restored'.
	// Otherwise, you may comment out UnmarshalData() until it's needed.

	return nil
}

func (dst *NnfSystemStorage) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nnfv1alpha8.NnfSystemStorage)
	convertlog.Info("Convert NnfSystemStorage From Hub", "name", src.GetName(), "namespace", src.GetNamespace())

	if err := Convert_v1alpha8_NnfSystemStorage_To_v1alpha6_NnfSystemStorage(src, dst, nil); err != nil {
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

func autoConvert_v1alpha3_ResourceError_To_v1alpha5_ResourceError(in *dwsv1alpha3.ResourceError, out *dwsv1alpha5.ResourceError, s apiconversion.Scope) error {
	out.Error = (*dwsv1alpha5.ResourceErrorInfo)(unsafe.Pointer(in.Error))
	return nil
}

// Convert_v1alpha3_ResourceError_To_v1alpha5_ResourceError is an autogenerated conversion function.
func Convert_v1alpha3_ResourceError_To_v1alpha5_ResourceError(in *dwsv1alpha3.ResourceError, out *dwsv1alpha5.ResourceError, s apiconversion.Scope) error {
	return autoConvert_v1alpha3_ResourceError_To_v1alpha5_ResourceError(in, out, s)
}

func autoConvert_v1alpha5_ResourceError_To_v1alpha3_ResourceError(in *dwsv1alpha5.ResourceError, out *dwsv1alpha3.ResourceError, s apiconversion.Scope) error {
	out.Error = (*dwsv1alpha3.ResourceErrorInfo)(unsafe.Pointer(in.Error))
	return nil
}

// Convert_4_ResourceError_To_v1alpha3_ResourceError is an autogenerated conversion function.
func Convert_v1alpha5_ResourceError_To_v1alpha3_ResourceError(in *dwsv1alpha5.ResourceError, out *dwsv1alpha3.ResourceError, s apiconversion.Scope) error {
	return autoConvert_v1alpha5_ResourceError_To_v1alpha3_ResourceError(in, out, s)
}

func autoConvert_v1alpha3_ResourceErrorInfo_To_v1alpha5_ResourceErrorInfo(in *dwsv1alpha3.ResourceErrorInfo, out *dwsv1alpha5.ResourceErrorInfo, s apiconversion.Scope) error {
	out.UserMessage = in.UserMessage
	out.DebugMessage = in.DebugMessage
	out.Type = dwsv1alpha5.ResourceErrorType(in.Type)
	out.Severity = dwsv1alpha5.ResourceErrorSeverity(in.Severity)
	return nil
}

// Convert_v1alpha3_ResourceErrorInfo_To_v1alpha5_ResourceErrorInfo is an autogenerated conversion function.
func Convert_v1alpha3_ResourceErrorInfo_To_v1alpha5_ResourceErrorInfo(in *dwsv1alpha3.ResourceErrorInfo, out *dwsv1alpha5.ResourceErrorInfo, s apiconversion.Scope) error {
	return autoConvert_v1alpha3_ResourceErrorInfo_To_v1alpha5_ResourceErrorInfo(in, out, s)
}

func autoConvert_v1alpha5_ResourceErrorInfo_To_v1alpha3_ResourceErrorInfo(in *dwsv1alpha5.ResourceErrorInfo, out *dwsv1alpha3.ResourceErrorInfo, s apiconversion.Scope) error {
	out.UserMessage = in.UserMessage
	out.DebugMessage = in.DebugMessage
	out.Type = dwsv1alpha3.ResourceErrorType(in.Type)
	out.Severity = dwsv1alpha3.ResourceErrorSeverity(in.Severity)
	return nil
}

// Convert_v1alpha5_ResourceErrorInfo_To_v1alpha3_ResourceErrorInfo is an autogenerated conversion function.
func Convert_v1alpha5_ResourceErrorInfo_To_v1alpha3_ResourceErrorInfo(in *dwsv1alpha5.ResourceErrorInfo, out *dwsv1alpha3.ResourceErrorInfo, s apiconversion.Scope) error {
	return autoConvert_v1alpha5_ResourceErrorInfo_To_v1alpha3_ResourceErrorInfo(in, out, s)
}

// End of DWS ResourceError conversion routines.
// +crdbumper:carryforward:end

// The conversion-gen tool dropped these from zz_generated.conversion.go to
// force us to acknowledge that we are addressing the conversion requirements.

func Convert_v1alpha6_NnfContainerProfileData_To_v1alpha8_NnfContainerProfileData(in *NnfContainerProfileData, out *v1alpha8.NnfContainerProfileData, s apiconversion.Scope) error {
	return autoConvert_v1alpha6_NnfContainerProfileData_To_v1alpha8_NnfContainerProfileData(in, out, s)
}

func Convert_v1alpha8_NnfContainerProfileData_To_v1alpha6_NnfContainerProfileData(in *v1alpha8.NnfContainerProfileData, out *NnfContainerProfileData, s apiconversion.Scope) error {
	return autoConvert_v1alpha8_NnfContainerProfileData_To_v1alpha6_NnfContainerProfileData(in, out, s)
}

func Convert_v1alpha6_NnfDataMovementManagerSpec_To_v1alpha8_NnfDataMovementManagerSpec(in *NnfDataMovementManagerSpec, out *nnfv1alpha8.NnfDataMovementManagerSpec, s apiconversion.Scope) error {
	return autoConvert_v1alpha6_NnfDataMovementManagerSpec_To_v1alpha8_NnfDataMovementManagerSpec(in, out, s)
}

func Convert_v1alpha8_NnfDataMovementManagerSpec_To_v1alpha6_NnfDataMovementManagerSpec(in *nnfv1alpha8.NnfDataMovementManagerSpec, out *NnfDataMovementManagerSpec, s apiconversion.Scope) error {
	return autoConvert_v1alpha8_NnfDataMovementManagerSpec_To_v1alpha6_NnfDataMovementManagerSpec(in, out, s)
}
