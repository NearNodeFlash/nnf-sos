/*
 * Copyright 2021, 2022 Hewlett Packard Enterprise Development LP
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

package controllers

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"runtime"
	"strconv"
	"time"

	"github.com/go-logr/logr"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	dwsv1alpha1 "github.com/HewlettPackard/dws/api/v1alpha1"
	nnfv1alpha1 "github.com/NearNodeFlash/nnf-sos/api/v1alpha1"
)

// DWSServersReconciler reconciles a DWS Servers object
type DWSServersReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *kruntime.Scheme
}

const (
	// finalizerNnfServers defines the key used to prevent the system from deleting the
	// resource until this reconciler has finished doing clean up
	finalizerNnfServers = "nnf.cray.hpe.com/nnf-servers"
)

type deletedStorage bool

const (
	storageDeleted        deletedStorage = true
	storageStillAllocated deletedStorage = false
)

//+kubebuilder:rbac:groups=dws.cray.hpe.com,resources=servers,verbs=get;create;list;watch;update;patch;delete
//+kubebuilder:rbac:groups=dws.cray.hpe.com,resources=servers/status,verbs=get;create;list;watch;update;patch;delete
//+kubebuilder:rbac:groups=dws.cray.hpe.com,resources=servers/finalizers,verbs=get;create;list;watch;update;patch;delete
//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfstorages,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfstorages/status,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfnodestorages,verbs=get;list;watch;update;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Workflow object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.9.2/pkg/reconcile
func (r *DWSServersReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, err error) {

	servers := &dwsv1alpha1.Servers{}
	if err := r.Get(ctx, req.NamespacedName, servers); err != nil {
		// ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Check if the object is being deleted
	if !servers.GetDeletionTimestamp().IsZero() {
		if !controllerutil.ContainsFinalizer(servers, finalizerNnfServers) {
			return ctrl.Result{}, nil
		}

		complete, err := r.checkDeletedStorage(ctx, servers)
		if err != nil {
			return ctrl.Result{}, err
		}

		if complete == false {
			return r.updateCapacityUsed(ctx, servers)
		}

		controllerutil.RemoveFinalizer(servers, finalizerNnfServers)
		if err := r.Update(ctx, servers); err != nil {
			if apierrors.IsConflict(err) {
				return ctrl.Result{Requeue: true}, nil
			}

			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	// Add a finalizer to ensure the DWS Storage resource created by this resource is properly deleted.
	if !controllerutil.ContainsFinalizer(servers, finalizerNnfServers) {

		controllerutil.AddFinalizer(servers, finalizerNnfServers)
		if err := r.Update(ctx, servers); err != nil {
			if apierrors.IsConflict(err) {
				return ctrl.Result{Requeue: true}, nil
			}

			return ctrl.Result{Requeue: true}, nil
		}

		return ctrl.Result{}, nil
	}

	// In the case where WLM has not filled in Servers Spec information generate a status update
	// to initialize the LastUpdate timestamp but don't proceed to create the Status section
	// since there is nothing to report status on.
	if len(servers.Spec.AllocationSets) == 0 {
		if servers.Status.LastUpdate == nil {
			servers.Status.Ready = false
			return r.statusUpdate(ctx, servers, false)
		}

		return ctrl.Result{}, nil
	}

	// Initialize the status section if it isn't filled in
	if len(servers.Status.AllocationSets) == 0 {
		servers.Status.Ready = false
		return r.statusSetEmpty(ctx, servers)
	}

	return r.updateCapacityUsed(ctx, servers)
}

func (r *DWSServersReconciler) updateCapacityUsed(ctx context.Context, servers *dwsv1alpha1.Servers) (ctrl.Result, error) {
	originalServers := servers.DeepCopy()

	if len(servers.Status.AllocationSets) == 0 {
		return ctrl.Result{}, nil
	}

	// Get the NnfStorage with the same name/namespace as the servers resource. It may not exist
	// yet if we're still in proposal phase, or if it was deleted in teardown.
	nnfStorage := &nnfv1alpha1.NnfStorage{}
	if err := r.Get(ctx, types.NamespacedName{Name: servers.Name, Namespace: servers.Namespace}, nnfStorage); err != nil {
		if apierrors.IsNotFound(err) {
			return r.statusSetEmpty(ctx, servers)
		}

		return ctrl.Result{}, err
	}

	ready := true
	expectedAllocations := 0
	actualAllocations := 0

	for storageIndex := range nnfStorage.Spec.AllocationSets {
		// The nnfStorage may not have the status section filled in yet.
		if len(nnfStorage.Status.AllocationSets) < storageIndex+1 {
			ready = false
			break
		}

		allocationSet := nnfStorage.Status.AllocationSets[storageIndex]

		if allocationSet.Status != nnfv1alpha1.ResourceReady {
			ready = false
		}

		// Increment the actual and expected allocation counts from this allocationSet
		actualAllocations += allocationSet.AllocationCount
		for _, node := range nnfStorage.Spec.AllocationSets[storageIndex].Nodes {
			expectedAllocations += node.Count
		}

		// Use the label to find the allocation set in the servers resource that matches
		// the nnfStorage allocation set. Don't assume that the allocation set indices match
		// between the two resources
		label := nnfStorage.Spec.AllocationSets[storageIndex].Name
		serversIndex := -1
		for i, serversAllocation := range servers.Spec.AllocationSets {
			if serversAllocation.Label == label {
				serversIndex = i
				break
			}
		}

		// If the nnfStorage was created using information from the Servers resource, then
		// we should always find a match.
		if serversIndex == -1 {
			return ctrl.Result{}, fmt.Errorf("Unable to find allocation label %s", label)
		}

		// Loop through the nnfNodeStorages corresponding to each of the Rabbit nodes and find
		matchLabels := dwsv1alpha1.MatchingOwner(nnfStorage)
		matchLabels[nnfv1alpha1.AllocationSetLabel] = label

		listOptions := []client.ListOption{
			matchLabels,
		}

		nnfNodeStorageList := &nnfv1alpha1.NnfNodeStorageList{}
		if err := r.List(ctx, nnfNodeStorageList, listOptions...); err != nil {
			return ctrl.Result{}, err
		}

		if len(nnfNodeStorageList.Items) != len(nnfStorage.Spec.AllocationSets[storageIndex].Nodes) {
			ready = false
		}

		for _, nnfNodeStorage := range nnfNodeStorageList.Items {
			// There can be multiple allocations per Rabbit. Add them all up and present a
			// single size for the servers resource
			var allocationSize int64
			for _, nnfNodeAllocation := range nnfNodeStorage.Status.Allocations {
				if nnfNodeAllocation.CapacityAllocated == 0 {
					ready = false
				}
				allocationSize += nnfNodeAllocation.CapacityAllocated
			}

			servers.Status.AllocationSets[serversIndex].Storage[nnfNodeStorage.Namespace] = dwsv1alpha1.ServersStatusStorage{AllocationSize: allocationSize}
		}

		for _, storageStatus := range servers.Status.AllocationSets[serversIndex].Storage {
			if storageStatus.AllocationSize == 0 {
				ready = false
				break
			}
		}
	}

	// Switch from "ready = true" to "ready = false" if necessary. Once we've set the servers resource
	// to "ready = true", don't switch it back again.
	if servers.Status.Ready == false && ready == true {
		servers.Status.Ready = true
	}

	// Avoid updates when nothing has changed.
	if reflect.DeepEqual(originalServers, servers) {
		return ctrl.Result{}, nil
	}

	// Force the update (no batch) if all the allocations were made or all the
	// allocations were deleted
	batch := true
	if expectedAllocations == actualAllocations || actualAllocations == 0 {
		batch = false
	}

	return r.statusUpdate(ctx, servers, batch)
}

// Either the NnfStorage has not been created yet, or it existed and has been deleted
func (r *DWSServersReconciler) statusSetEmpty(ctx context.Context, servers *dwsv1alpha1.Servers) (ctrl.Result, error) {
	// Keep the original to check later for updates
	originalServers := servers.DeepCopy()

	servers.Status.AllocationSets = []dwsv1alpha1.ServersStatusAllocationSet{}
	for _, allocationSetSpec := range servers.Spec.AllocationSets {
		allocationSetStatus := dwsv1alpha1.ServersStatusAllocationSet{}
		allocationSetStatus.Label = allocationSetSpec.Label
		allocationSetStatus.Storage = make(map[string]dwsv1alpha1.ServersStatusStorage)
		for _, storage := range allocationSetSpec.Storage {
			allocationSetStatus.Storage[storage.Name] = dwsv1alpha1.ServersStatusStorage{AllocationSize: 0}
		}

		servers.Status.AllocationSets = append(servers.Status.AllocationSets, allocationSetStatus)
	}

	// If nothing has changed avoid the update here because every update modifies LastUpdate
	// which in turn generates another update.
	if reflect.DeepEqual(originalServers, servers) {
		return ctrl.Result{}, nil
	}

	// Update the status with batch=false to prevent batching. Using statusUpdate will keep the LastUpdate
	// field valid
	return r.statusUpdate(ctx, servers, false)
}

// Update Status if we've eclipsed the batch time
func (r *DWSServersReconciler) statusUpdate(ctx context.Context, servers *dwsv1alpha1.Servers, batch bool) (ctrl.Result, error) {
	log := r.Log.WithValues("Servers", types.NamespacedName{Name: servers.Name, Namespace: servers.Namespace})
	if batch == true && servers.Status.LastUpdate != nil {
		batchTime, err := strconv.Atoi(os.Getenv("SERVERS_BATCH_TIME_MSEC"))
		if err != nil {
			batchTime = 0
		}

		// Check if the last update time was more than SERVERS_BATCH_TIME_MSEC ago. If it's not,
		// then return without doing the Update() and requeue.
		if metav1.NowMicro().Time.Before(servers.Status.LastUpdate.Time.Add(time.Millisecond * time.Duration(batchTime))) {
			log.Info("Batching status update")
			// Requeue in case nothing triggers a reconcile after the batch time is over
			return ctrl.Result{RequeueAfter: time.Millisecond * time.Duration(batchTime)}, nil
		}
	}

	t := metav1.NowMicro()
	servers.Status.LastUpdate = &t

	if err := r.Status().Update(ctx, servers); err != nil {
		if apierrors.IsConflict(err) {
			return ctrl.Result{Requeue: true}, nil
		}

		return ctrl.Result{}, err
	}

	log.Info("Status updated")

	return ctrl.Result{}, nil
}

// Wait for the NnfStorage resource to be deleted. We'll update the servers status to reflect
// capacity being freed.
func (r *DWSServersReconciler) checkDeletedStorage(ctx context.Context, servers *dwsv1alpha1.Servers) (deletedStorage, error) {
	log := r.Log.WithValues("Servers", types.NamespacedName{Name: servers.Name, Namespace: servers.Namespace})

	// Get the NnfStorage with the same name/namespace as the servers resource
	nnfStorage := &nnfv1alpha1.NnfStorage{}
	if err := r.Get(ctx, types.NamespacedName{Name: servers.Name, Namespace: servers.Namespace}, nnfStorage); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("NnfStorage is deleted")
			return storageDeleted, nil
		}

		return storageStillAllocated, err
	}

	return storageStillAllocated, nil
}

// Map a NnfStorage resource to a Servers resource. There isn't an owner reference between
// these objects
func nnfStorageServersMapFunc(o client.Object) []reconcile.Request {
	return []reconcile.Request{
		// The servers resource has the same name/namespace as the NnfStorage resource
		{NamespacedName: types.NamespacedName{
			Name:      o.GetName(),
			Namespace: o.GetNamespace(),
		}},
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *DWSServersReconciler) SetupWithManager(mgr ctrl.Manager) error {
	maxReconciles := runtime.GOMAXPROCS(0)
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(controller.Options{MaxConcurrentReconciles: maxReconciles}).
		For(&dwsv1alpha1.Servers{}).
		Watches(&source.Kind{Type: &nnfv1alpha1.NnfStorage{}}, handler.EnqueueRequestsFromMapFunc(nnfStorageServersMapFunc)).
		Complete(r)
}
