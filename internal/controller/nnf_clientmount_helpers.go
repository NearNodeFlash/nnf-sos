/*
 * Copyright 2021-2024 Hewlett Packard Enterprise Development LP
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

package controller

import (
	"context"
	"os"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	dwsv1alpha2 "github.com/DataWorkflowServices/dws/api/v1alpha2"
	nnfv1alpha1 "github.com/NearNodeFlash/nnf-sos/api/v1alpha1"
)

func InitializeClientMountStatus(clientMount *dwsv1alpha2.ClientMount) bool {
	updated := false

	// Create the status section if it doesn't exist yet
	if len(clientMount.Status.Mounts) != len(clientMount.Spec.Mounts) {
		clientMount.Status.Mounts = make([]dwsv1alpha2.ClientMountInfoStatus, len(clientMount.Spec.Mounts))
		updated = true
	}

	// Initialize the status section if the desired state doesn't match the status state
	if clientMount.Status.Mounts[0].State != clientMount.Spec.DesiredState {
		for i := 0; i < len(clientMount.Status.Mounts); i++ {
			clientMount.Status.Mounts[i].State = clientMount.Spec.DesiredState
			clientMount.Status.Mounts[i].Ready = false
		}
		clientMount.Status.AllReady = false
		updated = true
	}
	return updated
}

func ChangeMountAllPre(clientMount *dwsv1alpha2.ClientMount) {
	clientMount.Status.Error = nil
	clientMount.Status.AllReady = false
}

func ChangeMountAllPost(clientMount *dwsv1alpha2.ClientMount) {
	clientMount.Status.AllReady = true
}

// changeMmountAll mounts or unmounts all the file systems listed in the spec.Mounts list
func ChangeMountAll(ctx context.Context, clnt client.Client, clientMount *dwsv1alpha2.ClientMount, state dwsv1alpha2.ClientMountState, log logr.Logger) (bool, error) {
	var firstError error
	didWork := false
	for i := range clientMount.Spec.Mounts {
		mountChanged := false
		var err error

		switch state {
		case dwsv1alpha2.ClientMountStateMounted:
			mountChanged, err = changeMount(ctx, clnt, clientMount, i, true, log)
		case dwsv1alpha2.ClientMountStateUnmounted:
			mountChanged, err = changeMount(ctx, clnt, clientMount, i, false, log)
		default:
			return didWork, dwsv1alpha2.NewResourceError("invalid desired state %s", state).WithFatal()
		}

		if err != nil {
			if firstError == nil {
				firstError = err
			}
			clientMount.Status.Mounts[i].Ready = false
		} else {
			clientMount.Status.Mounts[i].Ready = true
		}
		if mountChanged {
			didWork = true
		}
	}

	return didWork, firstError
}

// changeMount mount or unmounts a single mount point described in the ClientMountInfo object
func changeMount(ctx context.Context, clnt client.Client, clientMount *dwsv1alpha2.ClientMount, index int, shouldMount bool, ilog logr.Logger) (bool, error) {
	log := ilog.WithValues("ClientMount", client.ObjectKeyFromObject(clientMount), "index", index)

	clientMountInfo := clientMount.Spec.Mounts[index]
	nnfNodeStorage := fakeNnfNodeStorage(clientMount, index)
	mountChanged := false

	_, fileSystem, err := getBlockDeviceAndFileSystem(ctx, clnt, nnfNodeStorage, clientMountInfo.Device.DeviceReference.Data, log)
	if err != nil {
		return false, dwsv1alpha2.NewResourceError("unable to get file system information").WithError(err).WithMajor()
	}

	if shouldMount {
		mounted, err := fileSystem.Mount(ctx, clientMountInfo.MountPath, clientMount.Status.Mounts[index].Ready)
		if err != nil {
			return false, dwsv1alpha2.NewResourceError("unable to mount file system").WithError(err).WithMajor()
		}
		if mounted {
			log.Info("Mounted file system", "Mount path", clientMountInfo.MountPath)
			mountChanged = true
		}

		if clientMount.Spec.Mounts[index].SetPermissions {
			if err := os.Chown(clientMountInfo.MountPath, int(clientMount.Spec.Mounts[index].UserID), int(clientMount.Spec.Mounts[index].GroupID)); err != nil {
				return mountChanged, dwsv1alpha2.NewResourceError("unable to set owner and group for file system").WithError(err).WithMajor()
			}
		}
	} else {
		unmounted, err := fileSystem.Unmount(ctx, clientMountInfo.MountPath)
		if err != nil {
			return false, dwsv1alpha2.NewResourceError("unable to unmount file system").WithError(err).WithMajor()
		}
		if unmounted {
			log.Info("Unmounted file system", "Mount path", clientMountInfo.MountPath)
			mountChanged = true
		}
	}

	return mountChanged, nil
}

// fakeNnfNodeStorage creates an NnfNodeStorage resource filled in with only the fields
// that are necessary to mount the file system. This is done to reduce the API server load
// because the compute nodes don't need to Get() the actual NnfNodeStorage.
func fakeNnfNodeStorage(clientMount *dwsv1alpha2.ClientMount, index int) *nnfv1alpha1.NnfNodeStorage {
	nnfNodeStorage := &nnfv1alpha1.NnfNodeStorage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clientMount.Spec.Mounts[index].Device.DeviceReference.ObjectReference.Name,
			Namespace: clientMount.Spec.Mounts[index].Device.DeviceReference.ObjectReference.Namespace,
			UID:       types.UID("fake_UID"),
		},
	}

	// These labels aren't exactly right (NnfStorage owns NnfNodeStorage), but the
	// labels that are important for doing the mount are there and correct
	dwsv1alpha2.InheritParentLabels(nnfNodeStorage, clientMount)
	labels := nnfNodeStorage.GetLabels()
	labels[nnfv1alpha1.DirectiveIndexLabel] = getTargetDirectiveIndexLabel(clientMount)
	labels[dwsv1alpha2.OwnerUidLabel] = getTargetOwnerUIDLabel(clientMount)
	nnfNodeStorage.SetLabels(labels)

	nnfNodeStorage.Spec.BlockReference = corev1.ObjectReference{
		Name:      "fake",
		Namespace: "fake",
		Kind:      "fake",
	}

	nnfNodeStorage.Spec.UserID = clientMount.Spec.Mounts[index].UserID
	nnfNodeStorage.Spec.GroupID = clientMount.Spec.Mounts[index].GroupID
	nnfNodeStorage.Spec.FileSystemType = clientMount.Spec.Mounts[index].Type
	if nnfNodeStorage.Spec.FileSystemType == "none" {
		nnfNodeStorage.Spec.FileSystemType = "raw"
	}

	if clientMount.Spec.Mounts[index].Type == "lustre" {
		nnfNodeStorage.Spec.LustreStorage.BackFs = "none"
		nnfNodeStorage.Spec.LustreStorage.TargetType = "ost"
		nnfNodeStorage.Spec.LustreStorage.FileSystemName = clientMount.Spec.Mounts[index].Device.Lustre.FileSystemName
		nnfNodeStorage.Spec.LustreStorage.MgsAddress = clientMount.Spec.Mounts[index].Device.Lustre.MgsAddresses
	}

	return nnfNodeStorage
}
