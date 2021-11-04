/*
Copyright 2021 Hewlett Packard Enterprise Development LP
*/

package controllers

import (
	"context"
	"fmt"
	"reflect"

	"github.com/go-logr/logr"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	nnfv1alpha1 "stash.us.cray.com/RABSW/nnf-sos/api/v1alpha1"
	dwsv1alpha1 "stash.us.cray.com/dpm/dws-operator/api/v1alpha1"
)

// DWSServersReconciler reconciles a DWS Servers object
type DWSServersReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
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

func (r *DWSServersReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, err error) {
	log := r.Log.WithValues("DwsServers", req.NamespacedName)

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

	if len(servers.Spec.AllocationSets) == 0 {
		servers.Status.Ready = false
		if err := r.Status().Update(ctx, servers); err != nil {
			if apierrors.IsConflict(err) {
				return ctrl.Result{Requeue: true}, nil
			}

			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}
	// Initialize the status section if it isn't filled in
	if len(servers.Status.AllocationSets) != len(servers.Spec.AllocationSets) {
		servers.Status.Ready = false
		for _, allocationSetSpec := range servers.Spec.AllocationSets {
			allocationSetStatus := dwsv1alpha1.ServersStatusAllocationSet{}
			allocationSetStatus.Label = allocationSetSpec.Label
			for _, storage := range allocationSetSpec.Storage {
				allocationSetStatus.Storage = append(allocationSetStatus.Storage, dwsv1alpha1.ServersStatusStorage{Name: storage.Name, AllocationSize: 0})
			}

			servers.Status.AllocationSets = append(servers.Status.AllocationSets, allocationSetStatus)
		}

		log.Info("Initializing servers status")
		if err := r.Status().Update(ctx, servers); err != nil {
			if apierrors.IsConflict(err) {
				return ctrl.Result{Requeue: true}, nil
			}

			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	return r.updateCapacityUsed(ctx, servers)
}

func (r *DWSServersReconciler) updateCapacityUsed(ctx context.Context, servers *dwsv1alpha1.Servers) (ctrl.Result, error) {
	log := r.Log.WithValues("Servers", types.NamespacedName{Name: servers.Name, Namespace: servers.Namespace})
	originalServers := servers.DeepCopy()

	if len(servers.Status.AllocationSets) == 0 {
		return ctrl.Result{}, nil
	}

	// Get the NnfStorage with the same name/namespace as the servers resource. It may not exist
	// yet if we're still in proposal phase.
	nnfStorage := &nnfv1alpha1.NnfStorage{}
	if err := r.Get(ctx, types.NamespacedName{Name: servers.Name, Namespace: servers.Namespace}, nnfStorage); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, err
	}

	ready := true
	for storageIndex := range nnfStorage.Spec.AllocationSets {
		// The nnfStorage may not have the status section filled in yet.
		if len(nnfStorage.Status.AllocationSets) < storageIndex+1 {
			ready = false
			break
		}

		allocationSet := nnfStorage.Status.AllocationSets[storageIndex]

		// Don't bother updating the capacities in the servers resource until
		// we know everything has been allocated in the nnfNodeStorage resources
		if allocationSet.Status == nnfv1alpha1.ResourceStarting {
			ready = false
			break
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

		// Make a map of the storage list in the servers status section so we can quickly match
		// them up to the nnfNodeStorage resources they correspond to based on name.
		storageStatusMap := make(map[string]*dwsv1alpha1.ServersStatusStorage)
		for i := range servers.Status.AllocationSets[serversIndex].Storage {
			storageStatus := &servers.Status.AllocationSets[serversIndex].Storage[i]
			storageStatus.AllocationSize = 0
			storageStatusMap[storageStatus.Name] = storageStatus
		}

		// Loop through the nnfNodeStorages corresponding to each of the Rabbit nodes and find
		// the allocated size
		for _, nnfNodeStorageRef := range allocationSet.NodeStorageReferences {
			nnfNodeStorage := &nnfv1alpha1.NnfNodeStorage{}
			namespacedName := types.NamespacedName{
				Name:      nnfNodeStorageRef.Name,
				Namespace: nnfNodeStorageRef.Namespace,
			}

			if err := r.Get(ctx, namespacedName, nnfNodeStorage); err != nil {
				return ctrl.Result{}, err
			}

			// There can be multiple allocations per Rabbit. Add them all up and present a
			// single size for the servers resource
			var allocationSize int64 = 0
			for _, nnfNodeAllocation := range nnfNodeStorage.Status.Allocations {
				if nnfNodeAllocation.CapacityAllocated == 0 {
					ready = false
				}
				allocationSize += nnfNodeAllocation.CapacityAllocated
			}

			storageStatus, ok := storageStatusMap[nnfNodeStorage.Namespace]
			if !ok {
				return ctrl.Result{}, fmt.Errorf("NnfNodeStorage %v has no corresponding storage entry in the servers resource", namespacedName)
			}

			storageStatus.AllocationSize = allocationSize
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

	if ready == false {
		return ctrl.Result{}, nil
	}

	if !reflect.DeepEqual(originalServers, servers) {
		log.Info("Updating servers status with allocation data")
		if err := r.Status().Update(ctx, servers); err != nil {
			if apierrors.IsConflict(err) {
				return ctrl.Result{Requeue: true}, nil
			}

			return ctrl.Result{}, err
		}
	}

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
	return ctrl.NewControllerManagedBy(mgr).
		For(&dwsv1alpha1.Servers{}).
		Watches(&source.Kind{Type: &nnfv1alpha1.NnfStorage{}}, handler.EnqueueRequestsFromMapFunc(nnfStorageServersMapFunc)).
		Complete(r)
}
