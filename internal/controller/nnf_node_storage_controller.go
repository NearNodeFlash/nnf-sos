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
	"runtime"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	dwsv1alpha2 "github.com/DataWorkflowServices/dws/api/v1alpha2"
	"github.com/DataWorkflowServices/dws/utils/updater"
	nnfv1alpha1 "github.com/NearNodeFlash/nnf-sos/api/v1alpha1"
	"github.com/NearNodeFlash/nnf-sos/internal/controller/metrics"
)

const (
	// finalizerNnfNodeStorage defines the key used in identifying the
	// storage object as being owned by this NNF Storage Reconciler. This
	// prevents the system from deleting the custom resource until the
	// reconciler has finished using the resource.
	finalizerNnfNodeStorage = "nnf.cray.hpe.com/nnf_node_storage"

	nnfNodeStorageResourceName = "nnf-node-storage"
)

// NnfNodeStorageReconciler contains the elements needed during reconciliation for NnfNodeStorage
type NnfNodeStorageReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *kruntime.Scheme

	types.NamespacedName
	ChildObjects []dwsv1alpha2.ObjectList
}

//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfnodestorages,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfnodestorages/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfnodestorages/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.2/pkg/reconcile
func (r *NnfNodeStorageReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, err error) {
	log := r.Log.WithValues("NnfNodeStorage", req.NamespacedName)
	metrics.NnfNodeStorageReconcilesTotal.Inc()

	nnfNodeStorage := &nnfv1alpha1.NnfNodeStorage{}
	if err := r.Get(ctx, req.NamespacedName, nnfNodeStorage); err != nil {
		// ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	log.Info("dump", "raw", nnfNodeStorage)
	// Use the Node Storage Status Updater to track updates to the storage status.
	// This ensures that only one call to r.Status().Update() is done even though we
	// update the status at several points in the process. We hijack the defer logic
	// to perform the status update if no other error is present in the system when
	// exiting this reconcile function. Note that "err" is the named return value,
	// so when we would normally call "return ctrl.Result{}, nil", at that time
	// "err" is nil - and if permitted we will update err with the result of
	// the r.Update()
	statusUpdater := updater.NewStatusUpdater[*nnfv1alpha1.NnfNodeStorageStatus](nnfNodeStorage)
	defer func() { err = statusUpdater.CloseWithStatusUpdate(ctx, r.Client.Status(), err) }()
	defer func() { nnfNodeStorage.Status.SetResourceErrorAndLog(err, log) }()

	// Check if the object is being deleted. Deletion is carefully coordinated around
	// the NNF resources being managed by this NNF Node Storage resource. For a
	// successful deletion, the NNF Storage Pool must be deleted. Deletion of the
	// Storage Pool handles the entire sub-tree of NNF resources (Storage Groups,
	// File System, and File Shares). The Finalizer on this NNF Node Storage resource
	// is present until the underlying NNF resources are deleted through the
	// storage service.
	if !nnfNodeStorage.GetDeletionTimestamp().IsZero() {
		if !controllerutil.ContainsFinalizer(nnfNodeStorage, finalizerNnfNodeStorage) {
			return ctrl.Result{}, nil
		}

		for i := range nnfNodeStorage.Status.Allocations {
			// Release physical storage
			result, err := r.deleteAllocation(ctx, nnfNodeStorage, i)
			if err != nil {

				return ctrl.Result{}, err
			}
			if result != nil {
				return *result, nil
			}
		}

		controllerutil.RemoveFinalizer(nnfNodeStorage, finalizerNnfNodeStorage)
		if err := r.Update(ctx, nnfNodeStorage); err != nil {
			if !apierrors.IsConflict(err) {
				return ctrl.Result{}, err
			}

			return ctrl.Result{Requeue: true}, nil
		}

		return ctrl.Result{}, nil
	}

	// First time setup requires programming of the storage status such that the resource
	// is labeled as "Starting". After this is done,
	// the resource obtains a finalizer to manage the resource lifetime.
	if !controllerutil.ContainsFinalizer(nnfNodeStorage, finalizerNnfNodeStorage) {
		controllerutil.AddFinalizer(nnfNodeStorage, finalizerNnfNodeStorage)
		if err := r.Update(ctx, nnfNodeStorage); err != nil {
			if !apierrors.IsConflict(err) {

				return ctrl.Result{}, err
			}

			return ctrl.Result{Requeue: true}, nil
		}

		return ctrl.Result{}, nil
	}

	// Initialize the status section with empty allocation statuses.
	if len(nnfNodeStorage.Status.Allocations) == 0 {
		nnfNodeStorage.Status.Allocations = make([]nnfv1alpha1.NnfNodeStorageAllocationStatus, nnfNodeStorage.Spec.Count)
		for i := range nnfNodeStorage.Status.Allocations {
			nnfNodeStorage.Status.Allocations[i].Ready = false
		}
		nnfNodeStorage.Status.Ready = false

		return ctrl.Result{Requeue: true}, nil
	}

	// Loop through each allocation and create the storage
	for i := 0; i < nnfNodeStorage.Spec.Count; i++ {
		result, err := r.createAllocation(ctx, nnfNodeStorage, i)
		if err != nil {
			return ctrl.Result{}, dwsv1alpha2.NewResourceError("unable to format file system for allocation %v", i).WithError(err).WithMajor()
		}
		if result != nil {
			return *result, nil
		}
	}

	for _, allocation := range nnfNodeStorage.Status.Allocations {
		if allocation.Ready == false {
			nnfNodeStorage.Status.Ready = false

			log.Info("ready = false", "dump", nnfNodeStorage.Status)
			return ctrl.Result{Requeue: true}, nil
		}
	}

	nnfNodeStorage.Status.Ready = true

	log.Info("ready = true")
	return ctrl.Result{}, nil
}

func (r *NnfNodeStorageReconciler) deleteAllocation(ctx context.Context, nnfNodeStorage *nnfv1alpha1.NnfNodeStorage, index int) (*ctrl.Result, error) {
	log := r.Log.WithValues("NnfNodeStorage", client.ObjectKeyFromObject(nnfNodeStorage), "index", index)

	blockDevice, fileSystem, err := getBlockDeviceAndFileSystem(ctx, r.Client, nnfNodeStorage, index, log)
	if err != nil {
		return nil, err
	}

	ran, err := fileSystem.Deactivate(ctx)
	if err != nil {
		return nil, dwsv1alpha2.NewResourceError("could not deactivate file system").WithError(err).WithMajor()
	}
	if ran {
		log.Info("Deactivated file system", "allocation", index)
	}

	ran, err = fileSystem.Destroy(ctx)
	if err != nil {
		return nil, dwsv1alpha2.NewResourceError("could not destroy file system").WithError(err).WithMajor()
	}
	if ran {
		log.Info("Destroyed file system", "allocation", index)
	}

	ran, err = blockDevice.Deactivate(ctx)
	if err != nil {
		return nil, dwsv1alpha2.NewResourceError("could not deactivate block devices").WithError(err).WithMajor()
	}
	if ran {
		log.Info("Deactivated block device", "allocation", index)
	}

	ran, err = blockDevice.Destroy(ctx)
	if err != nil {
		return nil, dwsv1alpha2.NewResourceError("could not destroy block devices").WithError(err).WithMajor()
	}
	if ran {
		log.Info("Destroyed block device", "allocation", index)
	}

	return nil, nil
}

func (r *NnfNodeStorageReconciler) createAllocation(ctx context.Context, nnfNodeStorage *nnfv1alpha1.NnfNodeStorage, index int) (*ctrl.Result, error) {
	log := r.Log.WithValues("NnfNodeStorage", client.ObjectKeyFromObject(nnfNodeStorage), "index", index)

	blockDevice, fileSystem, err := getBlockDeviceAndFileSystem(ctx, r.Client, nnfNodeStorage, index, log)
	if err != nil {
		return nil, err
	}

	allocationStatus := &nnfNodeStorage.Status.Allocations[index]
	ran, err := blockDevice.Create(ctx, allocationStatus.Ready)
	if err != nil {
		return nil, dwsv1alpha2.NewResourceError("could not create block devices").WithError(err).WithMajor()
	}
	if ran {
		log.Info("Created block device", "allocation", index)
	}

	// We don't need to activate the block device here. It will be activated either when there is a mkfs, or when it's used
	// by a ClientMount
	ran, err = fileSystem.Create(ctx, allocationStatus.Ready)
	if err != nil {
		return nil, dwsv1alpha2.NewResourceError("could not create file system").WithError(err).WithMajor()
	}
	if ran {
		log.Info("Created file system", "allocation", index)
	}

	ran, err = fileSystem.Activate(ctx, allocationStatus.Ready)
	if err != nil {
		return nil, dwsv1alpha2.NewResourceError("could not activate file system").WithError(err).WithMajor()
	}
	if ran {
		log.Info("Activated file system", "allocation", index)
	}

	ran, err = fileSystem.SetPermissions(ctx, nnfNodeStorage.Spec.UserID, nnfNodeStorage.Spec.GroupID, allocationStatus.Ready)
	if err != nil {
		return nil, dwsv1alpha2.NewResourceError("could not set file system permissions").WithError(err).WithMajor()
	}
	if ran {
		log.Info("Set file system permission", "allocation", index)
	}

	allocationStatus.Ready = true

	return nil, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NnfNodeStorageReconciler) SetupWithManager(mgr ctrl.Manager) error {
	maxReconciles := runtime.GOMAXPROCS(0)
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(controller.Options{MaxConcurrentReconciles: maxReconciles}).
		For(&nnfv1alpha1.NnfNodeStorage{}).
		Complete(r)
}
