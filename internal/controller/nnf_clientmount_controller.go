/*
 * Copyright 2021-2023 Hewlett Packard Enterprise Development LP
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
	"runtime"
	"strings"
	"time"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/mount-utils"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	nnf "github.com/NearNodeFlash/nnf-ec/pkg/manager-nnf"

	dwsv1alpha2 "github.com/DataWorkflowServices/dws/api/v1alpha2"
	"github.com/DataWorkflowServices/dws/utils/updater"
	sf "github.com/NearNodeFlash/nnf-ec/pkg/rfsf/pkg/models"
	nnfv1alpha1 "github.com/NearNodeFlash/nnf-sos/api/v1alpha1"
	"github.com/NearNodeFlash/nnf-sos/internal/controller/metrics"
)

const (
	// finalizerNnfClientMount defines the finalizer name that this controller
	// uses on the ClientMount resource. This prevents the ClientMount resource
	// from being fully deleted until this controller removes the finalizer.
	finalizerNnfClientMount = "nnf.cray.hpe.com/nnf_clientmount"
)

// NnfClientMountReconciler contains the pieces used by the reconciler
type NnfClientMountReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *kruntime.Scheme
}

//+kubebuilder:rbac:groups=dataworkflowservices.github.io,resources=clientmounts,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=dataworkflowservices.github.io,resources=clientmounts/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=dataworkflowservices.github.io,resources=clientmounts/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *NnfClientMountReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, err error) {
	log := r.Log.WithValues("ClientMount", req.NamespacedName)

	metrics.NnfClientMountReconcilesTotal.Inc()

	clientMount := &dwsv1alpha2.ClientMount{}
	if err := r.Get(ctx, req.NamespacedName, clientMount); err != nil {
		// ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	// Create a status updater that handles the call to status().Update() if any of the fields
	// in clientMount.Status change
	statusUpdater := updater.NewStatusUpdater[*dwsv1alpha2.ClientMountStatus](clientMount)
	defer func() { err = statusUpdater.CloseWithStatusUpdate(ctx, r.Client.Status(), err) }()
	defer func() { clientMount.Status.SetResourceErrorAndLog(err, log) }()

	// Ensure the NNF Storage Service is running prior to taking any action.
	ss := nnf.NewDefaultStorageService()
	storageService := &sf.StorageServiceV150StorageService{}
	if err := ss.StorageServiceIdGet(ss.Id(), storageService); err != nil {
		return ctrl.Result{}, dwsv1alpha2.NewResourceError("unable to get redfish storage service status").WithError(err).WithMajor()
	}

	if storageService.Status.State != sf.ENABLED_RST {
		return ctrl.Result{RequeueAfter: 1 * time.Second}, nil
	}

	// Handle cleanup if the resource is being deleted
	if !clientMount.GetDeletionTimestamp().IsZero() {
		if !controllerutil.ContainsFinalizer(clientMount, finalizerNnfClientMount) {
			return ctrl.Result{}, nil
		}

		// Unmount everything before removing the finalizer
		log.Info("Unmounting all file systems due to resource deletion")
		if err := r.changeMountAll(ctx, clientMount, dwsv1alpha2.ClientMountStateUnmounted); err != nil {
			return ctrl.Result{}, err
		}

		controllerutil.RemoveFinalizer(clientMount, finalizerNnfClientMount)
		if err := r.Update(ctx, clientMount); err != nil {
			if !apierrors.IsConflict(err) {
				return ctrl.Result{}, err
			}

			return ctrl.Result{Requeue: true}, nil
		}

		return ctrl.Result{}, nil
	}

	// Create the status section if it doesn't exist yet
	if len(clientMount.Status.Mounts) != len(clientMount.Spec.Mounts) {
		clientMount.Status.Mounts = make([]dwsv1alpha2.ClientMountInfoStatus, len(clientMount.Spec.Mounts))
	}

	// Initialize the status section if the desired state doesn't match the status state
	if clientMount.Status.Mounts[0].State != clientMount.Spec.DesiredState {
		for i := 0; i < len(clientMount.Status.Mounts); i++ {
			clientMount.Status.Mounts[i].State = clientMount.Spec.DesiredState
			clientMount.Status.Mounts[i].Ready = false
		}

		return ctrl.Result{}, nil
	}

	// Add finalizer if it doesn't exist
	if !controllerutil.ContainsFinalizer(clientMount, finalizerNnfClientMount) {
		controllerutil.AddFinalizer(clientMount, finalizerNnfClientMount)
		if err := r.Update(ctx, clientMount); err != nil {
			if !apierrors.IsConflict(err) {
				return ctrl.Result{}, err
			}

			return ctrl.Result{Requeue: true}, nil
		}

		return ctrl.Result{}, nil
	}

	clientMount.Status.Error = nil

	if err := r.changeMountAll(ctx, clientMount, clientMount.Spec.DesiredState); err != nil {
		resourceError := dwsv1alpha2.NewResourceError("mount/unmount failed").WithError(err)
		log.Info(resourceError.Error())

		clientMount.Status.Error = resourceError
		return ctrl.Result{RequeueAfter: time.Second * time.Duration(10)}, nil
	}

	return ctrl.Result{}, nil
}

// changeMmountAll mounts or unmounts all the file systems listed in the spec.Mounts list
func (r *NnfClientMountReconciler) changeMountAll(ctx context.Context, clientMount *dwsv1alpha2.ClientMount, state dwsv1alpha2.ClientMountState) error {
	log := r.Log.WithValues("ClientMount", types.NamespacedName{Name: clientMount.Name, Namespace: clientMount.Namespace})

	var firstError error
	for i, mount := range clientMount.Spec.Mounts {
		var err error

		switch state {
		case dwsv1alpha2.ClientMountStateMounted:
			err = r.changeMount(ctx, mount, true, log)
		case dwsv1alpha2.ClientMountStateUnmounted:
			err = r.changeMount(ctx, mount, false, log)
		default:
			return dwsv1alpha2.NewResourceError("invalid desired state %s", state).WithFatal()
		}

		if err != nil {
			if firstError == nil {
				firstError = err
			}
			clientMount.Status.Mounts[i].Ready = false
		} else {
			clientMount.Status.Mounts[i].Ready = true
		}
	}

	return firstError
}

// changeMount mount or unmounts a single mount point described in the ClientMountInfo object
func (r *NnfClientMountReconciler) changeMount(ctx context.Context, clientMountInfo dwsv1alpha2.ClientMountInfo, shouldMount bool, log logr.Logger) error {

	if os.Getenv("ENVIRONMENT") == "kind" {
		if shouldMount {
			if err := os.MkdirAll(clientMountInfo.MountPath, 0755); err != nil {
				return dwsv1alpha2.NewResourceError("make directory failed: %s", clientMountInfo.MountPath).WithError(err).WithMajor()
			}

			log.Info("Fake mounted file system", "Mount path", clientMountInfo.MountPath)
		} else {
			// Return if the directory was already removed
			if _, err := os.Stat(clientMountInfo.MountPath); os.IsNotExist(err) {
				return nil
			}

			if err := os.RemoveAll(clientMountInfo.MountPath); err != nil {
				return dwsv1alpha2.NewResourceError("remove directory failed: %s", clientMountInfo.MountPath).WithError(err).WithMajor()
			}

			log.Info("Fake unmounted file system", "Mount path", clientMountInfo.MountPath)
		}

		if clientMountInfo.SetPermissions {
			if err := os.Chown(clientMountInfo.MountPath, int(clientMountInfo.UserID), int(clientMountInfo.GroupID)); err != nil {
				return dwsv1alpha2.NewResourceError("chown failed: %s", clientMountInfo.MountPath).WithError(err).WithMajor()
			}
		}

		return nil
	}

	switch clientMountInfo.Device.Type {
	case dwsv1alpha2.ClientMountDeviceTypeLustre:
		mountPath := clientMountInfo.MountPath

		_, testEnv := os.LookupEnv("NNF_TEST_ENVIRONMENT")

		var mounter mount.Interface
		if testEnv {
			mounter = mount.NewFakeMounter([]mount.MountPoint{})
		} else {
			mounter = mount.New("")
		}

		isNotMountPoint, _ := mount.IsNotMountPoint(mounter, mountPath)

		if shouldMount {
			if isNotMountPoint {

				mountSource := clientMountInfo.Device.Lustre.MgsAddresses +
					":/" +
					clientMountInfo.Device.Lustre.FileSystemName

				if !testEnv {
					if err := os.MkdirAll(mountPath, 0755); err != nil {
						return dwsv1alpha2.NewResourceError("make directory failed: %s", mountPath).WithError(err).WithMajor()
					}
				}

				if err := mounter.Mount(mountSource, mountPath, "lustre", nil); err != nil {
					return dwsv1alpha2.NewResourceError("unable to mount file system").WithError(err).WithMajor()
				}
			}
		} else {
			if !isNotMountPoint {
				if err := mounter.Unmount(mountPath); err != nil {
					return dwsv1alpha2.NewResourceError("unable to unmount file system").WithError(err).WithMajor()
				}
			}
		}

	case dwsv1alpha2.ClientMountDeviceTypeReference:

		namespacedName := types.NamespacedName{
			Name:      clientMountInfo.Device.DeviceReference.ObjectReference.Name,
			Namespace: clientMountInfo.Device.DeviceReference.ObjectReference.Namespace,
		}

		nodeStorage := &nnfv1alpha1.NnfNodeStorage{}
		if err := r.Get(ctx, namespacedName, nodeStorage); err != nil {
			return err
		}

		allocationStatus := nodeStorage.Status.Allocations[clientMountInfo.Device.DeviceReference.Data]
		fileShare, err := r.getFileShare(allocationStatus.FileSystem.ID, allocationStatus.FileShare.ID)
		if err != nil {
			return dwsv1alpha2.NewResourceError("could not get file share").WithError(err).WithMajor()
		}

		if shouldMount {
			fileShare.FileSharePath = clientMountInfo.MountPath
		} else {
			fileShare.FileSharePath = ""
		}

		fileShare, err = r.updateFileShare(allocationStatus.FileSystem.ID, fileShare)
		if err != nil {
			return dwsv1alpha2.NewResourceError("could not update file share").WithError(err).WithMajor()
		}

	default:
		return dwsv1alpha2.NewResourceError("invalid device type %s", clientMountInfo.Device.Type).WithFatal()
	}

	if shouldMount {
		log.Info("Mounted file system", "Mount path", clientMountInfo.MountPath)
	} else {
		log.Info("Unmounted file system", "Mount path", clientMountInfo.MountPath)
	}

	return nil
}

func (r *NnfClientMountReconciler) updateFileShare(fileSystemId string, fileShare *sf.FileShareV120FileShare) (*sf.FileShareV120FileShare, error) {
	ss := nnf.NewDefaultStorageService()

	if err := ss.StorageServiceIdFileSystemIdExportedShareIdPut(ss.Id(), fileSystemId, fileShare.Id, fileShare); err != nil {
		return nil, err
	}

	return fileShare, nil
}

func (r *NnfClientMountReconciler) getFileShare(fileSystemId string, fileShareId string) (*sf.FileShareV120FileShare, error) {
	ss := nnf.NewDefaultStorageService()
	sh := &sf.FileShareV120FileShare{}

	if err := ss.StorageServiceIdFileSystemIdExportedShareIdGet(ss.Id(), fileSystemId, fileShareId, sh); err != nil {
		return nil, err
	}

	return sh, nil
}

func filterByRabbitNamespacePrefixForTest() predicate.Predicate {
	return predicate.NewPredicateFuncs(func(object client.Object) bool {
		return strings.HasPrefix(object.GetNamespace(), "rabbit")
	})
}

// SetupWithManager sets up the controller with the Manager.
func (r *NnfClientMountReconciler) SetupWithManager(mgr ctrl.Manager) error {

	maxReconciles := runtime.GOMAXPROCS(0)
	builder := ctrl.NewControllerManagedBy(mgr).
		WithOptions(controller.Options{MaxConcurrentReconciles: maxReconciles}).
		For(&dwsv1alpha2.ClientMount{})

	if _, found := os.LookupEnv("NNF_TEST_ENVIRONMENT"); found {
		builder = builder.WithEventFilter(filterByRabbitNamespacePrefixForTest())
	}

	return builder.Complete(r)
}
