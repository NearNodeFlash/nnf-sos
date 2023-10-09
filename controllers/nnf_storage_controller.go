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

package controllers

import (
	"context"
	"reflect"
	"runtime"
	"strconv"
	"time"

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

	dwsv1alpha2 "github.com/DataWorkflowServices/dws/api/v1alpha2"
	"github.com/DataWorkflowServices/dws/utils/updater"
	nnfv1alpha1 "github.com/NearNodeFlash/nnf-sos/api/v1alpha1"
	"github.com/NearNodeFlash/nnf-sos/controllers/metrics"
)

// NnfStorageReconciler reconciles a Storage object
type NnfStorageReconciler struct {
	client.Client
	Log          logr.Logger
	Scheme       *kruntime.Scheme
	ChildObjects []dwsv1alpha2.ObjectList
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
//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfnodestorages,verbs=get;list;watch;create;update;patch;delete;deletecollection
//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfstorageprofiles,verbs=get;create;list;watch;update;patch;delete;deletecollection

// The Storage Controller will list and make modifications to individual NNF Nodes, so include the
// RBAC policy for nnfnodes. This isn't strictly necessary since the same ClusterRole is shared for
// both controllers, but we include it here for completeness
// TODO: UPDATE THIS COMMENT
//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfnodestorages,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *NnfStorageReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, err error) {
	log := r.Log.WithValues("NnfStorage", req.NamespacedName)
	metrics.NnfStorageReconcilesTotal.Inc()

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
	statusUpdater := updater.NewStatusUpdater[*nnfv1alpha1.NnfStorageStatus](storage)
	defer func() { err = statusUpdater.CloseWithStatusUpdate(ctx, r.Client.Status(), err) }()
	defer func() {
		if err != nil || (!res.Requeue && res.RequeueAfter == 0) {
			storage.Status.SetResourceErrorAndLog(err, log)
		}
	}()

	// Check if the object is being deleted
	if !storage.GetDeletionTimestamp().IsZero() {
		if !controllerutil.ContainsFinalizer(storage, finalizerNnfStorage) {
			return ctrl.Result{}, nil
		}

		exists, err := r.teardownStorage(ctx, storage)
		if err != nil {
			return ctrl.Result{}, err
		}

		// Wait for all the nodeStorages to finish deleting before removing
		// the finalizer.
		if exists == nodeStoragesExist {
			return ctrl.Result{}, nil
		}

		controllerutil.RemoveFinalizer(storage, finalizerNnfStorage)
		if err := r.Update(ctx, storage); err != nil {
			if !apierrors.IsConflict(err) {
				return ctrl.Result{}, err
			}

			return ctrl.Result{Requeue: true}, nil
		}

		return ctrl.Result{}, nil
	}

	// Add a finalizer to ensure all NNF Node Storage resources created by this resource are properly deleted.
	if !controllerutil.ContainsFinalizer(storage, finalizerNnfStorage) {

		controllerutil.AddFinalizer(storage, finalizerNnfStorage)
		if err := r.Update(ctx, storage); err != nil {
			if !apierrors.IsConflict(err) {
				return ctrl.Result{}, err
			}

			return ctrl.Result{Requeue: true}, nil
		}

		return ctrl.Result{}, nil
	}

	// Initialize the status section of the NnfStorage if it hasn't been done already.
	if len(storage.Status.AllocationSets) != len(storage.Spec.AllocationSets) {
		storage.Status.AllocationSets = make([]nnfv1alpha1.NnfStorageAllocationSetStatus, len(storage.Spec.AllocationSets))
		for i := range storage.Status.AllocationSets {
			storage.Status.AllocationSets[i].Status = nnfv1alpha1.ResourceStarting
		}
		storage.Status.Status = nnfv1alpha1.ResourceStarting

		return ctrl.Result{}, nil
	}

	storage.Status.Error = nil

	// For each allocation, create the NnfNodeStorage resources to fan out to the Rabbit nodes
	for i, allocationSet := range storage.Spec.AllocationSets {
		// Add a reference to the external MGS PersistentStorageInstance if necessary
		if allocationSet.NnfStorageLustreSpec.PersistentMgsReference != (corev1.ObjectReference{}) {
			if err := r.addPersistentStorageReference(ctx, storage, allocationSet.NnfStorageLustreSpec.PersistentMgsReference); err != nil {
				return ctrl.Result{}, err
			}
		}

		res, err := r.createNodeStorage(ctx, storage, i)
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
		res, err := r.aggregateNodeStorageStatus(ctx, storage, i)
		if err != nil {
			return ctrl.Result{}, err
		}

		if res != nil {
			return *res, nil
		}
	}

	// Wait for all the allocation sets to be ready
	for _, allocationSet := range storage.Status.AllocationSets {
		if allocationSet.Status != nnfv1alpha1.ResourceReady {
			return ctrl.Result{RequeueAfter: time.Minute}, nil
		}
	}

	// For Lustre, the owner and group have to be set once all the Lustre targets
	// have completed. This is done on the Rabbit node that hosts OST 0.
	for i, allocationSet := range storage.Spec.AllocationSets {
		if allocationSet.TargetType != "OST" {
			continue
		}

		res, err := r.setLustreOwnerGroup(ctx, storage, i)
		if err != nil {
			return ctrl.Result{}, err
		}

		if res != nil {
			return *res, nil
		}
	}

	// All allocation sets are ready and the owner/group is set
	storage.Status.Status = nnfv1alpha1.ResourceReady

	return ctrl.Result{}, nil
}

func (r *NnfStorageReconciler) addPersistentStorageReference(ctx context.Context, nnfStorage *nnfv1alpha1.NnfStorage, persistentMgsReference corev1.ObjectReference) error {
	persistentStorage := &dwsv1alpha2.PersistentStorageInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      persistentMgsReference.Name,
			Namespace: persistentMgsReference.Namespace,
		},
	}

	if err := r.Get(ctx, client.ObjectKeyFromObject(persistentStorage), persistentStorage); err != nil {
		return dwsv1alpha2.NewResourceError("").WithUserMessage("PersistentStorage '%v' not found", client.ObjectKeyFromObject(persistentStorage)).WithMajor()
	}

	if persistentStorage.Status.State != dwsv1alpha2.PSIStateActive {
		return dwsv1alpha2.NewResourceError("").WithUserMessage("PersistentStorage is not active").WithFatal()
	}

	// Add a consumer reference to the persistent storage for this directive
	reference := corev1.ObjectReference{
		Name:      nnfStorage.Name,
		Namespace: nnfStorage.Namespace,
		Kind:      reflect.TypeOf(nnfv1alpha1.NnfStorage{}).Name(),
	}

	for _, existingReference := range persistentStorage.Spec.ConsumerReferences {
		if existingReference == reference {
			return nil
		}
	}

	persistentStorage.Spec.ConsumerReferences = append(persistentStorage.Spec.ConsumerReferences, reference)

	return r.Update(ctx, persistentStorage)
}

func (r *NnfStorageReconciler) removePersistentStorageReference(ctx context.Context, nnfStorage *nnfv1alpha1.NnfStorage, persistentMgsReference corev1.ObjectReference) error {
	persistentStorage := &dwsv1alpha2.PersistentStorageInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      persistentMgsReference.Name,
			Namespace: persistentMgsReference.Namespace,
		},
	}

	if err := r.Get(ctx, client.ObjectKeyFromObject(persistentStorage), persistentStorage); err != nil {
		return client.IgnoreNotFound(err)
	}

	// remove the consumer reference on the persistent storage for this directive
	reference := corev1.ObjectReference{
		Name:      nnfStorage.Name,
		Namespace: nnfStorage.Namespace,
		Kind:      reflect.TypeOf(nnfv1alpha1.NnfStorage{}).Name(),
	}

	for i, existingReference := range persistentStorage.Spec.ConsumerReferences {
		if existingReference == reference {
			persistentStorage.Spec.ConsumerReferences = append(persistentStorage.Spec.ConsumerReferences[:i], persistentStorage.Spec.ConsumerReferences[i+1:]...)
			return r.Update(ctx, persistentStorage)
		}
	}

	return nil
}

// Create an NnfNodeStorage if it doesn't exist, or update it if it requires updating. Each
// Rabbit node gets an NnfNodeStorage, and there may be multiple allocations requested in it.
// This limits the number of resources that have to be broadcast to the Rabbits.
func (r *NnfStorageReconciler) createNodeStorage(ctx context.Context, storage *nnfv1alpha1.NnfStorage, allocationSetIndex int) (*ctrl.Result, error) {
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
				dwsv1alpha2.InheritParentLabels(nnfNodeStorage, storage)
				dwsv1alpha2.AddOwnerLabels(nnfNodeStorage, storage)

				labels := nnfNodeStorage.GetLabels()
				labels[nnfv1alpha1.AllocationSetLabel] = allocationSet.Name
				nnfNodeStorage.SetLabels(labels)

				nnfNodeStorage.Spec.UserID = storage.Spec.UserID
				nnfNodeStorage.Spec.GroupID = storage.Spec.GroupID
				nnfNodeStorage.Spec.Capacity = allocationSet.Capacity
				nnfNodeStorage.Spec.Count = node.Count
				nnfNodeStorage.Spec.FileSystemType = storage.Spec.FileSystemType
				nnfNodeStorage.Spec.LustreStorage.StartIndex = startIndex
				nnfNodeStorage.Spec.LustreStorage.FileSystemName = allocationSet.FileSystemName
				nnfNodeStorage.Spec.LustreStorage.BackFs = allocationSet.BackFs
				nnfNodeStorage.Spec.LustreStorage.TargetType = allocationSet.TargetType

				// If this isn't the first allocation, then change MGTMDT to MDT so that we only get a single MGT
				if allocationSet.TargetType == "MGTMDT" && startIndex != 0 {
					nnfNodeStorage.Spec.LustreStorage.TargetType = "MDT"
				}

				// Create the list of client endpoints for each allocation and initialize it with
				// the rabbit node endpoint
				if len(nnfNodeStorage.Spec.ClientEndpoints) == 0 {
					nnfNodeStorage.Spec.ClientEndpoints = make([]nnfv1alpha1.ClientEndpointsSpec, node.Count)
					for k := range nnfNodeStorage.Spec.ClientEndpoints {
						nnfNodeStorage.Spec.ClientEndpoints[k].AllocationIndex = k
						nnfNodeStorage.Spec.ClientEndpoints[k].NodeNames = append(nnfNodeStorage.Spec.ClientEndpoints[k].NodeNames, node.Name)
					}
				}

				if nnfNodeStorage.Spec.LustreStorage.TargetType == "MDT" || nnfNodeStorage.Spec.LustreStorage.TargetType == "OST" {
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
				return nil, err
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
	}

	return nil, nil
}

// Get the status from all the child NnfNodeStorage resources and use them to build the status
// for the NnfStorage.
func (r *NnfStorageReconciler) aggregateNodeStorageStatus(ctx context.Context, storage *nnfv1alpha1.NnfStorage, allocationSetIndex int) (*ctrl.Result, error) {
	allocationSet := &storage.Status.AllocationSets[allocationSetIndex]

	var health nnfv1alpha1.NnfResourceHealthType = nnfv1alpha1.ResourceOkay
	var status nnfv1alpha1.NnfResourceStatusType = nnfv1alpha1.ResourceReady

	allocationSet.AllocationCount = 0

	nnfNodeStorageList := &nnfv1alpha1.NnfNodeStorageList{}
	matchLabels := dwsv1alpha2.MatchingOwner(storage)
	matchLabels[nnfv1alpha1.AllocationSetLabel] = storage.Spec.AllocationSets[allocationSetIndex].Name

	listOptions := []client.ListOption{
		matchLabels,
	}

	if err := r.List(ctx, nnfNodeStorageList, listOptions...); err != nil {
		return &ctrl.Result{Requeue: true}, nil
	}

	// Ensure that we found all the NnfNodeStorage resources we were expecting
	if len(nnfNodeStorageList.Items) != len(storage.Spec.AllocationSets[allocationSetIndex].Nodes) {
		status = nnfv1alpha1.ResourceStarting
	}

	for _, nnfNodeStorage := range nnfNodeStorageList.Items {
		if nnfNodeStorage.Spec.LustreStorage.TargetType == "MGT" || nnfNodeStorage.Spec.LustreStorage.TargetType == "MGTMDT" {
			storage.Status.MgsNode = nnfNodeStorage.Status.LustreStorage.Nid
		}

		// Wait until the status section of the nnfNodeStorage has been initialized
		if len(nnfNodeStorage.Status.Allocations) != nnfNodeStorage.Spec.Count {
			// Set the Status to starting unless we've found a failure in one
			// of the earlier nnfNodeStorages
			startingStatus := nnfv1alpha1.ResourceStarting
			startingStatus.UpdateIfWorseThan(&status)
			allocationSet.Status = status
			allocationSet.Health = health

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

		if nnfNodeStorage.Status.Error != nil {
			storage.Status.SetResourceError(nnfNodeStorage.Status.Error)
		}
	}

	allocationSet.Health = health
	allocationSet.Status = status

	return nil, nil
}

// setLustreOwnerGroup sets the "SetOwnerGroup" field in the NnfNodeStorage for OST 0 in a Lustre
// file system. This tells the node controller on the Rabbit to mount the Lustre file system and set
// the owner and group.
func (r *NnfStorageReconciler) setLustreOwnerGroup(ctx context.Context, storage *nnfv1alpha1.NnfStorage, allocationSetIndex int) (*ctrl.Result, error) {
	allocationSet := storage.Spec.AllocationSets[allocationSetIndex]

	if len(allocationSet.Nodes) == 0 {
		return nil, nil
	}

	nnfNodeStorage := &nnfv1alpha1.NnfNodeStorage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nnfNodeStorageName(storage, allocationSetIndex, 0),
			Namespace: allocationSet.Nodes[0].Name,
		},
	}

	if err := r.Get(ctx, client.ObjectKeyFromObject(nnfNodeStorage), nnfNodeStorage); err != nil {
		return nil, err
	}

	if !nnfNodeStorage.Spec.SetOwnerGroup {
		nnfNodeStorage.Spec.SetOwnerGroup = true

		if err := r.Update(ctx, nnfNodeStorage); err != nil {
			return nil, err
		}
	}

	if nnfNodeStorage.Status.OwnerGroupStatus != nnfv1alpha1.ResourceReady {
		return &ctrl.Result{}, nil
	}

	return nil, nil
}

// Delete all the child NnfNodeStorage resources. Don't trust the client cache
// or the object references in the storage resource. We may have created children
// that aren't in the cache and we may not have been able to add the object reference
// to the NnfStorage.
func (r *NnfStorageReconciler) teardownStorage(ctx context.Context, storage *nnfv1alpha1.NnfStorage) (nodeStoragesState, error) {
	// Collect status information from the NnfNodeStorage resources and aggregate it into the
	// NnfStorage
	for i := range storage.Status.AllocationSets {
		_, err := r.aggregateNodeStorageStatus(ctx, storage, i)
		if err != nil {
			return nodeStoragesExist, err
		}
	}

	deleteStatus, err := dwsv1alpha2.DeleteChildren(ctx, r.Client, r.ChildObjects, storage)
	if err != nil {
		return nodeStoragesExist, err
	}

	if !deleteStatus.Complete() {
		return nodeStoragesExist, nil
	}

	for _, allocationSet := range storage.Spec.AllocationSets {
		if allocationSet.NnfStorageLustreSpec.PersistentMgsReference != (corev1.ObjectReference{}) {
			if err := r.removePersistentStorageReference(ctx, storage, allocationSet.NnfStorageLustreSpec.PersistentMgsReference); err != nil {
				return nodeStoragesExist, err
			}
		}
	}

	return nodeStoragesDeleted, nil
}

// Build up the name of an NnfNodeStorage. This is a long name because:
// - NnfStorages from multiple namespaces create NnfNodeStorages in the same namespace
// - Different allocations in an NnfStorage could be targeting the same Rabbit node (e.g., MGS and MDS on the same Rabbit)
// - The same Rabbit node could be listed more than once within the same allocation.
func nnfNodeStorageName(storage *nnfv1alpha1.NnfStorage, allocationSetIndex int, i int) string {
	return storage.Namespace + "-" + storage.Name + "-" + storage.Spec.AllocationSets[allocationSetIndex].Name + "-" + strconv.Itoa(i)
}

// SetupWithManager sets up the controller with the Manager.
func (r *NnfStorageReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.ChildObjects = []dwsv1alpha2.ObjectList{
		&nnfv1alpha1.NnfNodeStorageList{},
		&nnfv1alpha1.NnfStorageProfileList{},
	}

	maxReconciles := runtime.GOMAXPROCS(0)
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(controller.Options{MaxConcurrentReconciles: maxReconciles}).
		For(&nnfv1alpha1.NnfStorage{}).
		Watches(&nnfv1alpha1.NnfNodeStorage{}, handler.EnqueueRequestsFromMapFunc(dwsv1alpha2.OwnerLabelMapFunc)).
		Complete(r)
}
