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
	"reflect"
	"runtime"
	"strconv"
	"strings"

	"github.com/go-logr/logr"

	corev1 "k8s.io/api/core/v1"
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

	nnfv1alpha1 "github.hpe.com/hpe/hpc-rabsw-nnf-sos/api/v1alpha1"
)

// NnfStorageReconciler reconciles a Storage object
type NnfStorageReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *kruntime.Scheme
}

const (
	// finalizerNnfStorage defines the key used in identifying the storage
	// object as being owned by this NNF Storage Reconciler. This prevents
	// the system from deleting the custom resource until the reconciler
	// has finished in using the resource.
	finalizerNnfStorage = "nnf.cray.hpe.com/nnf_storage"

	// ownerAnnotation is a name/namespace pair used on the NnfNodeStorage resources
	// for owner information. See nnfNodeStorageMapFunc() below.
	ownerAnnotation = "nnf.cray.hpe.com/owner"
)

type nodeStoragesState bool

const (
	nodeStoragesExist   nodeStoragesState = true
	nodeStoragesDeleted nodeStoragesState = false
)

//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfstorages,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfstorages/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfstorages/finalizers,verbs=update

// The Storage Controller will list and make modifications to individual NNF Nodes, so include the
// RBAC policy for nnfnodes. This isn't strictly necessary since the same ClusterRole is shared for
// both controllers, but we include it here for completeness
// TODO: UPDATE THIS COMMENT
//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfnodestorages,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *NnfStorageReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, err error) {
	storage := &nnfv1alpha1.NnfStorage{}
	if err := r.Get(ctx, req.NamespacedName, storage); err != nil {
		// ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Create an updater for the entire node. This will handle calls to r.Status().Update() such
	// that we can repeatedly make calls to the internal update method, with the final update
	// occuring on the on function exit.
	statusUpdater := newStorageStatusUpdater(storage)
	defer func() {
		if err == nil {
			var updated bool
			updated, err = statusUpdater.close(ctx, r)

			if updated == true {
				res.Requeue = true
			}
		}
	}()

	// Check if the object is being deleted
	if !storage.GetDeletionTimestamp().IsZero() {
		if !controllerutil.ContainsFinalizer(storage, finalizerNnfStorage) {
			return ctrl.Result{}, nil
		}

		exists, err := r.teardownStorage(ctx, statusUpdater, storage)
		if err != nil {
			return ctrl.Result{}, err
		}

		// Wait for all the nodeStorages to finish deleting before removing
		// the finalizer.
		if exists == nodeStoragesExist {
			return ctrl.Result{}, err
		}

		controllerutil.RemoveFinalizer(storage, finalizerNnfStorage)
		if err := r.Update(ctx, storage); err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	// Add a finalizer to ensure all NNF Node Storage resources created by this resource are properly deleted.
	if !controllerutil.ContainsFinalizer(storage, finalizerNnfStorage) {

		controllerutil.AddFinalizer(storage, finalizerNnfStorage)
		if err := r.Update(ctx, storage); err != nil {
			return ctrl.Result{Requeue: true}, nil
		}

		return ctrl.Result{}, nil
	}

	// Initialize the status section of the NnfStorage if it hasn't been done already.
	if len(storage.Status.AllocationSets) == 0 {
		statusUpdater.update(func(*nnfv1alpha1.NnfStorageStatus) {
			storage.Status.AllocationSets = make([]nnfv1alpha1.NnfStorageAllocationSetStatus, len(storage.Spec.AllocationSets))
			for i := range storage.Status.AllocationSets {
				storage.Status.AllocationSets[i].NodeStorageReferences = make([]corev1.ObjectReference, len(storage.Spec.AllocationSets[i].Nodes))
				storage.Status.AllocationSets[i].Status = nnfv1alpha1.ResourceStarting
			}
		})

		return ctrl.Result{}, nil
	}

	// For each allocation, create the NnfNodeStorage resources to fan out to the Rabbit nodes
	for i := range storage.Spec.AllocationSets {
		res, err := r.createNodeStorage(ctx, statusUpdater, storage, i)
		if err != nil {
			return ctrl.Result{}, err
		}

		if res != nil {
			return *res, nil
		}
	}

	// Collect status information from the NnfNodeStorage resources and aggregate it into the
	// NnfStorage
	for i := range storage.Spec.AllocationSets {
		res, err := r.aggregateNodeStorageStatus(ctx, statusUpdater, storage, i)
		if err != nil {
			return ctrl.Result{}, err
		}

		if res != nil {
			return *res, nil
		}
	}

	return ctrl.Result{}, nil
}

// Create an NnfNodeStorage if it doesn't exist, or update it if it requires updating. Each
// Rabbit node gets an NnfNodeStorage, and there may be multiple allocations requested in it.
// This limits the number of resources that have to be broadcast to the Rabbits.
func (r *NnfStorageReconciler) createNodeStorage(ctx context.Context, statusUpdater *storageStatusUpdater, storage *nnfv1alpha1.NnfStorage, allocationSetIndex int) (*ctrl.Result, error) {
	log := r.Log.WithValues("NnfStorage", types.NamespacedName{Name: storage.Name, Namespace: storage.Namespace})
	allocationSet := storage.Spec.AllocationSets[allocationSetIndex]

	startIndex := 0
	for i, node := range allocationSet.Nodes {
		// Per Rabbit namespace.
		nnfNodeStorage := &nnfv1alpha1.NnfNodeStorage{
			ObjectMeta: metav1.ObjectMeta{
				Name:      nnfNodeStorageName(storage, allocationSetIndex, i),
				Namespace: node.Name,
			},
		}

		result, err := ctrl.CreateOrUpdate(ctx, r.Client, nnfNodeStorage,
			func() error {
				annotations := nnfNodeStorage.GetAnnotations()
				if annotations == nil {
					annotations = make(map[string]string)
				}
				annotations[ownerAnnotation] = storage.Name + "/" + storage.Namespace
				nnfNodeStorage.SetAnnotations(annotations)

				nnfNodeStorage.Spec.Capacity = allocationSet.Capacity
				nnfNodeStorage.Spec.Count = node.Count
				nnfNodeStorage.Spec.FileSystemType = storage.Spec.FileSystemType
				nnfNodeStorage.Spec.LustreStorage.StartIndex = startIndex
				nnfNodeStorage.Spec.LustreStorage.FileSystemName = allocationSet.FileSystemName
				nnfNodeStorage.Spec.LustreStorage.TargetType = allocationSet.TargetType
				nnfNodeStorage.Spec.LustreStorage.BackFs = allocationSet.BackFs

				// Create the list of client endpoints for each allocation and initialize it with
				// the rabbit node endpoint
				if len(nnfNodeStorage.Spec.ClientEndpoints) == 0 {
					nnfNodeStorage.Spec.ClientEndpoints = make([]nnfv1alpha1.ClientEndpointsSpec, node.Count)
					for k := range nnfNodeStorage.Spec.ClientEndpoints {
						nnfNodeStorage.Spec.ClientEndpoints[k].AllocationIndex = k
						nnfNodeStorage.Spec.ClientEndpoints[k].NodeNames = append(nnfNodeStorage.Spec.ClientEndpoints[k].NodeNames, node.Name)
					}
				}

				if allocationSet.TargetType == "MDT" || allocationSet.TargetType == "OST" {
					if len(allocationSet.ExternalMgsNid) > 0 {
						nnfNodeStorage.Spec.LustreStorage.MgsNode = allocationSet.ExternalMgsNid
					} else {
						nnfNodeStorage.Spec.LustreStorage.MgsNode = storage.Status.MgsNode
					}
				}

				return nil
			})
		startIndex += node.Count

		if err != nil {
			if !apierrors.IsConflict(err) {
				statusUpdater.updateError(&storage.Status.AllocationSets[allocationSetIndex], err)
			}

			return &ctrl.Result{Requeue: true}, nil
		}

		if result == controllerutil.OperationResultCreated {
			log.Info("Created NnfNodeStorage", "Name", nnfNodeStorage.Name, "Namespace", nnfNodeStorage.Namespace)
		} else if result == controllerutil.OperationResultNone {
			// no change
		} else {
			log.Info("Updated NnfNodeStorage", "Name", nnfNodeStorage.Name, "Namespace", nnfNodeStorage.Namespace)
		}

		// Add an object reference to the storage resource
		objectRef := corev1.ObjectReference{
			Name:      nnfNodeStorage.Name,
			Namespace: nnfNodeStorage.Namespace,
			Kind:      nnfNodeStorage.Kind,
		}

		statusUpdater.update(func(s *nnfv1alpha1.NnfStorageStatus) {
			storage.Status.AllocationSets[allocationSetIndex].NodeStorageReferences[i] = objectRef
		})
	}

	return nil, nil
}

// Get the status from all the child NnfNodeStorage resources and use them to build the status
// for the NnfStorage.
func (r *NnfStorageReconciler) aggregateNodeStorageStatus(ctx context.Context, statusUpdater *storageStatusUpdater, storage *nnfv1alpha1.NnfStorage, allocationSetIndex int) (*ctrl.Result, error) {
	allocationSet := &storage.Status.AllocationSets[allocationSetIndex]

	var health nnfv1alpha1.NnfResourceHealthType = nnfv1alpha1.ResourceOkay
	var status nnfv1alpha1.NnfResourceStatusType = nnfv1alpha1.ResourceReady

	allocationSet.AllocationCount = 0

	for _, nodeStorageReference := range allocationSet.NodeStorageReferences {
		namespacedName := types.NamespacedName{
			Name:      nodeStorageReference.Name,
			Namespace: nodeStorageReference.Namespace,
		}

		nnfNodeStorage := &nnfv1alpha1.NnfNodeStorage{}
		err := r.Get(ctx, namespacedName, nnfNodeStorage)
		if err != nil {
			if !apierrors.IsNotFound(err) {
				statusUpdater.updateError(allocationSet, err)
				return &ctrl.Result{Requeue: true}, nil
			}

			startingStatus := nnfv1alpha1.ResourceStarting
			startingStatus.UpdateIfWorseThan(&status)
			continue
		}

		if nnfNodeStorage.Spec.LustreStorage.TargetType == "MGT" || nnfNodeStorage.Spec.LustreStorage.TargetType == "MGTMDT" {
			statusUpdater.update(func(s *nnfv1alpha1.NnfStorageStatus) {
				s.MgsNode = nnfNodeStorage.Status.LustreStorage.Nid
			})
		}

		// Wait until the status section of the nnfNodeStorage has been initialized
		if len(nnfNodeStorage.Status.Allocations) != nnfNodeStorage.Spec.Count {
			statusUpdater.update(func(s *nnfv1alpha1.NnfStorageStatus) {
				// Set the Status to starting unless we've found a failure in one
				// of the earlier nnfNodeStorages
				startingStatus := nnfv1alpha1.ResourceStarting
				startingStatus.UpdateIfWorseThan(&status)
				allocationSet.Status = status
				allocationSet.Health = health
			})

			return &ctrl.Result{}, nil
		}

		for _, nodeAllocation := range nnfNodeStorage.Status.Allocations {
			if nodeAllocation.CapacityAllocated > 0 {
				allocationSet.AllocationCount++
			}

			nodeAllocation.StoragePool.Health.UpdateIfWorseThan(&health)
			nodeAllocation.StorageGroup.Health.UpdateIfWorseThan(&health)
			nodeAllocation.FileSystem.Health.UpdateIfWorseThan(&health)
			nodeAllocation.FileShare.Health.UpdateIfWorseThan(&health)

			nodeAllocation.StoragePool.Status.UpdateIfWorseThan(&status)
			nodeAllocation.StorageGroup.Status.UpdateIfWorseThan(&status)
			nodeAllocation.FileSystem.Status.UpdateIfWorseThan(&status)
			nodeAllocation.FileShare.Status.UpdateIfWorseThan(&status)
		}
	}

	statusUpdater.update(func(s *nnfv1alpha1.NnfStorageStatus) {
		allocationSet.Health = health
		allocationSet.Status = status
	})

	return nil, nil
}

// Delete all the child NnfNodeStorage resources. Don't trust the client cache
// or the object references in the storage resource. We may have created children
// that aren't in the cache and we may not have been able to add the object reference
// to the NnfStorage.
func (r *NnfStorageReconciler) teardownStorage(ctx context.Context, statusUpdater *storageStatusUpdater, storage *nnfv1alpha1.NnfStorage) (nodeStoragesState, error) {
	log := r.Log.WithValues("Workflow", types.NamespacedName{Name: storage.Name, Namespace: storage.Namespace})
	var firstErr error

	// Collect status information from the NnfNodeStorage resources and aggregate it into the
	// NnfStorage
	for i := range storage.Spec.AllocationSets {
		_, err := r.aggregateNodeStorageStatus(ctx, statusUpdater, storage, i)
		if err != nil {
			return nodeStoragesExist, err
		}
	}

	state := nodeStoragesDeleted

	for allocationSetIndex, allocationSet := range storage.Spec.AllocationSets {
		for i, node := range allocationSet.Nodes {
			namespacedName := types.NamespacedName{
				Name:      nnfNodeStorageName(storage, allocationSetIndex, i),
				Namespace: node.Name,
			}

			// Do a Get on the nnfNodeStorage first to check if it's already been marked
			// for deletion.
			nnfNodeStorage := &nnfv1alpha1.NnfNodeStorage{}
			err := r.Get(ctx, namespacedName, nnfNodeStorage)
			if err != nil {
				if !apierrors.IsNotFound(err) {
					if firstErr == nil {
						firstErr = err
					}
					log.Info("Unable to get NnfNodeStorage", "Error", err, "NnfNodeStorage", namespacedName)
				}
			} else {
				state = nodeStoragesExist
				if !nnfNodeStorage.GetDeletionTimestamp().IsZero() {
					continue
				}
			}

			nnfNodeStorage = &nnfv1alpha1.NnfNodeStorage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      nnfNodeStorageName(storage, allocationSetIndex, i),
					Namespace: node.Name,
				},
			}

			err = r.Delete(ctx, nnfNodeStorage)
			if err != nil {
				if !apierrors.IsNotFound(err) {
					if firstErr == nil {
						firstErr = err
					}
					log.Info("Unable to delete NnfNodeStorage", "Error", err, "NnfNodeStorage", nnfNodeStorage)
				}
			} else {
				state = nodeStoragesExist
			}
		}
	}

	return state, firstErr
}

// Build up the name of an NnfNodeStorage. This is a long name because:
// - NnfStorages from multiple namespaces create NnfNodeStorages in the same namespace
// - Different allocations in an NnfStorage could be targeting the same Rabbit node (e.g., MGS and MDS on the same Rabbit)
// - The same Rabbit node could be listed more than once within the same allocation.
func nnfNodeStorageName(storage *nnfv1alpha1.NnfStorage, allocationSetIndex int, i int) string {
	return storage.Namespace + "-" + storage.Name + "-" + storage.Spec.AllocationSets[allocationSetIndex].Name + "-" + strconv.Itoa(i)
}

// Storage Status Updater handles finalizing of updates to the storage object for updates to a Storage object.
// Kubernete's requires only one such update per generation, with successive generations requiring a requeue.
// This object allows repeated calls to Update(), with the final call to reconciler.Status().Update() occurring
// only when the updater is Closed()
type storageStatusUpdater struct {
	storage        *nnfv1alpha1.NnfStorage
	existingStatus nnfv1alpha1.NnfStorageStatus
}

func newStorageStatusUpdater(s *nnfv1alpha1.NnfStorage) *storageStatusUpdater {
	return &storageStatusUpdater{
		storage:        s,
		existingStatus: (*s.DeepCopy()).Status,
	}
}

func (s *storageStatusUpdater) updateError(allocationSet *nnfv1alpha1.NnfStorageAllocationSetStatus, err error) {
	allocationSet.Reason = err.Error()
}

func (s *storageStatusUpdater) update(update func(s *nnfv1alpha1.NnfStorageStatus)) {
	update(&s.storage.Status)
}

func (s *storageStatusUpdater) close(ctx context.Context, r *NnfStorageReconciler) (bool, error) {
	if !reflect.DeepEqual(s.storage.Status, s.existingStatus) {
		return true, r.Status().Update(ctx, s.storage)
	}

	return false, nil
}

// Map function to translate an NnfNodeStorage to an NnfStorage. We can't use
// EnqueueRequestForOwner() because the NnfNodeStorage resources are in a different
// namespace than the NnfStorage resource, and owner references can't bridge namespaces.
// The owner information is stored in an annotation.
func nnfNodeStorageMapFunc(o client.Object) []reconcile.Request {
	annotations := o.GetAnnotations()

	owner, exists := annotations[ownerAnnotation]
	if exists == false {
		return []reconcile.Request{}
	}

	components := strings.Split(owner, "/")
	return []reconcile.Request{
		{NamespacedName: types.NamespacedName{
			Name:      components[0],
			Namespace: components[1],
		}},
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *NnfStorageReconciler) SetupWithManager(mgr ctrl.Manager) error {
	maxReconciles := runtime.GOMAXPROCS(0)
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(controller.Options{MaxConcurrentReconciles: maxReconciles}).
		For(&nnfv1alpha1.NnfStorage{}).
		Watches(&source.Kind{Type: &nnfv1alpha1.NnfNodeStorage{}}, handler.EnqueueRequestsFromMapFunc(nnfNodeStorageMapFunc)).
		Complete(r)
}
