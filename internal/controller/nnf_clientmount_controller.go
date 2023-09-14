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
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	dwsv1alpha2 "github.com/DataWorkflowServices/dws/api/v1alpha2"
	"github.com/DataWorkflowServices/dws/utils/updater"
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
//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfstorageprofiles,verbs=get;list;watch

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
	var firstError error
	for i := range clientMount.Spec.Mounts {
		var err error

		switch state {
		case dwsv1alpha2.ClientMountStateMounted:
			err = r.changeMount(ctx, clientMount, i, true)
		case dwsv1alpha2.ClientMountStateUnmounted:
			err = r.changeMount(ctx, clientMount, i, false)
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
func (r *NnfClientMountReconciler) changeMount(ctx context.Context, clientMount *dwsv1alpha2.ClientMount, index int, shouldMount bool) error {
	log := r.Log.WithValues("ClientMount", client.ObjectKeyFromObject(clientMount), "index", clientMount.Spec.Mounts[index].Device.DeviceReference.Data)

	clientMountInfo := clientMount.Spec.Mounts[index]
	nnfNodeStorage := r.fakeNnfNodeStorage(clientMount, index)

	_, fileSystem, err := getBlockDeviceAndFileSystem(ctx, r.Client, nnfNodeStorage, clientMountInfo.Device.DeviceReference.Data, log)
	if err != nil {
		return dwsv1alpha2.NewResourceError("unable to get file system information").WithError(err).WithMajor()
	}

	if shouldMount {
		mounted, err := fileSystem.Mount(ctx, clientMountInfo.MountPath, clientMountInfo.Options, clientMount.Status.Mounts[index].Ready)
		if err != nil {
			return dwsv1alpha2.NewResourceError("unable to mount file system").WithError(err).WithMajor()
		}
		if mounted {
			log.Info("Mounted file system", "Mount path", clientMountInfo.MountPath)
		}

		if clientMount.Spec.Mounts[index].SetPermissions {
			if err := os.Chown(clientMountInfo.MountPath, int(clientMount.Spec.Mounts[index].UserID), int(clientMount.Spec.Mounts[index].GroupID)); err != nil {
				return dwsv1alpha2.NewResourceError("unable to set owner and group for file system").WithError(err).WithMajor()
			}
		}
	} else {
		unmounted, err := fileSystem.Unmount(ctx, clientMountInfo.MountPath)
		if err != nil {
			return dwsv1alpha2.NewResourceError("unable to unmount file system").WithError(err).WithMajor()
		}
		if unmounted {
			log.Info("Unmounted file system", "Mount path", clientMountInfo.MountPath)
		}
	}

	return nil
}

// fakeNnfNodeStorage creates an NnfNodeStorage resource with filled in with only the fields
// that are necessary to mount the file system. This is done to reduce the API server load
// because the compute nodes don't need to Get() the actual NnfNodeStorage.
func (r *NnfClientMountReconciler) fakeNnfNodeStorage(clientMount *dwsv1alpha2.ClientMount, index int) *nnfv1alpha1.NnfNodeStorage {
	nnfNodeStorage := &nnfv1alpha1.NnfNodeStorage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clientMount.Spec.Mounts[index].Device.DeviceReference.ObjectReference.Name,
			Namespace: clientMount.Spec.Mounts[index].Device.DeviceReference.ObjectReference.Namespace,
		},
	}

	// These labels aren't exactly right (NnfStorage owns NnfNodeStorage), but the
	// labels that are important for doing the mount are there and correct
	dwsv1alpha2.InheritParentLabels(nnfNodeStorage, clientMount)

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
