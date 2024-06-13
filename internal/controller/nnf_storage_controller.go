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
	"fmt"
	"os"
	"reflect"
	"runtime"
	"strconv"

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
	"github.com/NearNodeFlash/nnf-sos/internal/controller/metrics"
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

	// Minimum size of lustre allocation sizes. If a user requests less than this, then the capacity
	// is set to this value.
	minimumLustreAllocationSizeInBytes = 4000000000
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
//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfnodeblockstorages,verbs=get;list;watch;create;update;patch;delete;deletecollection
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
	defer func() { storage.Status.SetResourceErrorAndLog(err, log) }()

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
			storage.Status.AllocationSets[i].Ready = false
		}
		storage.Status.Ready = false

		return ctrl.Result{}, nil
	}

	storage.Status.Error = nil

	// For each allocation, create the NnfNodeBlockStorage resources to fan out to the Rabbit nodes
	for i := range storage.Spec.AllocationSets {
		res, err := r.createNodeBlockStorage(ctx, storage, i)
		if err != nil {
			return ctrl.Result{}, err
		}

		if res != nil {
			return *res, nil
		}
	}

	// Collect status information from the NnfNodeBlockStorage resources and aggregate it into the
	// NnfStorage
	for i := range storage.Spec.AllocationSets {
		res, err := r.aggregateNodeBlockStorageStatus(ctx, storage, i)
		if err != nil {
			return ctrl.Result{}, err
		}

		if res != nil {
			return *res, nil
		}
	}

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
			if *res == (ctrl.Result{}) {
				continue
			} else {
				return *res, nil
			}
		}
	}

	// Wait for all the allocation sets to be ready
	for _, allocationSet := range storage.Status.AllocationSets {
		if allocationSet.Ready == false {
			return ctrl.Result{}, nil
		}
	}

	if storage.Spec.FileSystemType == "lustre" && storage.Status.Ready == false {
		res, err := r.setLustreOwnerGroup(ctx, storage)
		if err != nil {
			return ctrl.Result{}, err
		}

		if res != nil {
			return *res, nil
		}
	}

	// All allocation sets are ready and the owner/group is set
	storage.Status.Ready = true

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

func (r *NnfStorageReconciler) createNodeBlockStorage(ctx context.Context, nnfStorage *nnfv1alpha1.NnfStorage, allocationSetIndex int) (*ctrl.Result, error) {
	log := r.Log.WithValues("NnfStorage", client.ObjectKeyFromObject(nnfStorage))

	allocationSet := nnfStorage.Spec.AllocationSets[allocationSetIndex]
	fsType := nnfStorage.Spec.FileSystemType

	for i, node := range allocationSet.Nodes {
		// Per Rabbit namespace.
		nnfNodeBlockStorage := &nnfv1alpha1.NnfNodeBlockStorage{
			ObjectMeta: metav1.ObjectMeta{
				Name:      nnfNodeStorageName(nnfStorage, allocationSetIndex, i),
				Namespace: node.Name,
			},
		}

		result, err := ctrl.CreateOrUpdate(ctx, r.Client, nnfNodeBlockStorage,
			func() error {
				dwsv1alpha2.InheritParentLabels(nnfNodeBlockStorage, nnfStorage)
				dwsv1alpha2.AddOwnerLabels(nnfNodeBlockStorage, nnfStorage)

				labels := nnfNodeBlockStorage.GetLabels()
				labels[nnfv1alpha1.AllocationSetLabel] = allocationSet.Name
				nnfNodeBlockStorage.SetLabels(labels)

				if len(nnfNodeBlockStorage.Spec.Allocations) == 0 {
					nnfNodeBlockStorage.Spec.Allocations = make([]nnfv1alpha1.NnfNodeBlockStorageAllocationSpec, node.Count)
				}

				if len(nnfNodeBlockStorage.Spec.Allocations) != node.Count {
					return dwsv1alpha2.NewResourceError("block storage allocation count incorrect. found %v, expected %v", len(nnfNodeBlockStorage.Spec.Allocations), node.Count).WithFatal()
				}

				for i := range nnfNodeBlockStorage.Spec.Allocations {

					// For lustre (zfs), bump up the capacity if less than the floor. This is to
					// ensure that zpool create does not fail if the size is too small.
					capacity := allocationSet.Capacity
					if fsType == "lustre" && capacity < minimumLustreAllocationSizeInBytes {
						capacity = minimumLustreAllocationSizeInBytes
					}
					nnfNodeBlockStorage.Spec.Allocations[i].Capacity = capacity
					if len(nnfNodeBlockStorage.Spec.Allocations[i].Access) == 0 {
						nnfNodeBlockStorage.Spec.Allocations[i].Access = append(nnfNodeBlockStorage.Spec.Allocations[i].Access, node.Name)
					}
				}

				return nil
			})
		if err != nil {
			if !apierrors.IsConflict(err) {
				return nil, err
			}

			return &ctrl.Result{Requeue: true}, nil
		}

		if result == controllerutil.OperationResultCreated {
			log.Info("Created NnfNodeBlockStorage", "Name", nnfNodeBlockStorage.Name, "Namespace", nnfNodeBlockStorage.Namespace)
		} else if result == controllerutil.OperationResultNone {
			// no change
		} else {
			log.Info("Updated NnfNodeBlockStorage", "Name", nnfNodeBlockStorage.Name, "Namespace", nnfNodeBlockStorage.Namespace)
		}
	}

	return nil, nil
}

// Get the status from all the child NnfNodeBlockStorage resources and use them to build the status
// for the NnfStorage.
func (r *NnfStorageReconciler) aggregateNodeBlockStorageStatus(ctx context.Context, nnfStorage *nnfv1alpha1.NnfStorage, allocationSetIndex int) (*ctrl.Result, error) {
	allocationSet := &nnfStorage.Status.AllocationSets[allocationSetIndex]
	allocationSet.AllocationCount = 0

	nnfNodeBlockStorageList := &nnfv1alpha1.NnfNodeBlockStorageList{}
	matchLabels := dwsv1alpha2.MatchingOwner(nnfStorage)
	matchLabels[nnfv1alpha1.AllocationSetLabel] = nnfStorage.Spec.AllocationSets[allocationSetIndex].Name

	listOptions := []client.ListOption{
		matchLabels,
	}

	if err := r.List(ctx, nnfNodeBlockStorageList, listOptions...); err != nil {
		return &ctrl.Result{Requeue: true}, nil
	}

	// Ensure that we found all the NnfNodeStorage resources we were expecting
	if len(nnfNodeBlockStorageList.Items) != len(nnfStorage.Spec.AllocationSets[allocationSetIndex].Nodes) {
		return &ctrl.Result{}, nil
	}

	for _, nnfNodeBlockStorage := range nnfNodeBlockStorageList.Items {
		for _, nodeAllocation := range nnfNodeBlockStorage.Status.Allocations {
			if nodeAllocation.CapacityAllocated > 0 {
				allocationSet.AllocationCount++
			}
		}

		if nnfNodeBlockStorage.Status.Error != nil {
			return &ctrl.Result{}, nnfNodeBlockStorage.Status.Error
		}

		if nnfNodeBlockStorage.Status.Ready == false {
			return &ctrl.Result{}, nil
		}
	}

	return nil, nil
}

// Create an NnfNodeStorage if it doesn't exist, or update it if it requires updating. Each
// Rabbit node gets an NnfNodeStorage, and there may be multiple allocations requested in it.
// This limits the number of resources that have to be broadcast to the Rabbits.
func (r *NnfStorageReconciler) createNodeStorage(ctx context.Context, storage *nnfv1alpha1.NnfStorage, allocationSetIndex int) (*ctrl.Result, error) {
	log := r.Log.WithValues("NnfStorage", types.NamespacedName{Name: storage.Name, Namespace: storage.Namespace})

	mgsAddress := storage.Spec.AllocationSets[allocationSetIndex].MgsAddress
	if storage.Spec.FileSystemType == "lustre" {
		mgsNode := ""
		for i, allocationSet := range storage.Spec.AllocationSets {
			if allocationSet.TargetType == "mgt" || allocationSet.TargetType == "mgtmdt" {
				// Wait for the MGT to be set up before creating nnfnodestorages for the other allocation sets
				if allocationSetIndex != i {
					if storage.Status.AllocationSets[i].Ready == false {
						return nil, nil
					}
				}

				mgsNode = allocationSet.Nodes[0].Name
			}
		}

		if mgsNode != "" {
			nnfNode := &nnfv1alpha1.NnfNode{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "nnf-nlc",
					Namespace: mgsNode,
				},
			}

			if err := r.Get(ctx, client.ObjectKeyFromObject(nnfNode), nnfNode); err != nil {
				return &ctrl.Result{}, dwsv1alpha2.NewResourceError("could not get NnfNode: %v", client.ObjectKeyFromObject(nnfNode)).WithError(err)
			}

			mgsAddress = nnfNode.Status.LNetNid
		}
	}

	// Save the MGS address in the status section so we don't have to look in the NnfNodeStorage
	storage.Status.MgsAddress = mgsAddress

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

				nnfNodeStorage.Spec.BlockReference = corev1.ObjectReference{
					Name:      nnfNodeStorageName(storage, allocationSetIndex, i),
					Namespace: node.Name,
					Kind:      reflect.TypeOf(nnfv1alpha1.NnfNodeBlockStorage{}).Name(),
				}
				nnfNodeStorage.Spec.UserID = storage.Spec.UserID
				nnfNodeStorage.Spec.GroupID = storage.Spec.GroupID
				nnfNodeStorage.Spec.Count = node.Count
				nnfNodeStorage.Spec.FileSystemType = storage.Spec.FileSystemType
				if storage.Spec.FileSystemType == "lustre" {
					nnfNodeStorage.Spec.LustreStorage.StartIndex = startIndex
					nnfNodeStorage.Spec.LustreStorage.FileSystemName = allocationSet.FileSystemName
					nnfNodeStorage.Spec.LustreStorage.BackFs = allocationSet.BackFs
					nnfNodeStorage.Spec.LustreStorage.TargetType = allocationSet.TargetType
					nnfNodeStorage.Spec.LustreStorage.MgsAddress = mgsAddress

					// If this isn't the first allocation, then change MGTMDT to MDT so that we only get a single MGT
					if allocationSet.TargetType == "mgtmdt" && startIndex != 0 {
						nnfNodeStorage.Spec.LustreStorage.TargetType = "mdt"
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
	log := r.Log.WithValues("NnfStorage", types.NamespacedName{Name: storage.Name, Namespace: storage.Namespace})

	nnfNodeStorageList := &nnfv1alpha1.NnfNodeStorageList{}
	matchLabels := dwsv1alpha2.MatchingOwner(storage)
	matchLabels[nnfv1alpha1.AllocationSetLabel] = storage.Spec.AllocationSets[allocationSetIndex].Name

	listOptions := []client.ListOption{
		matchLabels,
	}

	if err := r.List(ctx, nnfNodeStorageList, listOptions...); err != nil {
		return &ctrl.Result{Requeue: true}, nil
	}

	for _, nnfNodeStorage := range nnfNodeStorageList.Items {
		if nnfNodeStorage.Status.Error != nil {
			return &ctrl.Result{}, nnfNodeStorage.Status.Error
		}

		if nnfNodeStorage.Status.Ready == false {
			return &ctrl.Result{}, nil
		}
	}

	// Ensure that we found all the NnfNodeStorage resources we were expecting
	if len(nnfNodeStorageList.Items) != len(storage.Spec.AllocationSets[allocationSetIndex].Nodes) {
		log.Info("Bad count", "found", len(nnfNodeStorageList.Items), "expected", len(storage.Spec.AllocationSets[allocationSetIndex].Nodes))
		return &ctrl.Result{}, nil
	}

	storage.Status.AllocationSets[allocationSetIndex].Ready = true

	return nil, nil
}

func (r *NnfStorageReconciler) setLustreOwnerGroup(ctx context.Context, nnfStorage *nnfv1alpha1.NnfStorage) (*ctrl.Result, error) {
	log := r.Log.WithValues("NnfStorage", client.ObjectKeyFromObject(nnfStorage))

	// Don't create the clientmount in the test environment. Some tests don't fake out the
	// NnfStorage enough to have it be successful.
	if _, found := os.LookupEnv("NNF_TEST_ENVIRONMENT"); found {
		return nil, nil
	}

	// Don't create the clientmount in kind until the kind environment creates fake file systems.
	if os.Getenv("ENVIRONMENT") == "kind" {
		return nil, nil
	}

	if nnfStorage.Spec.FileSystemType != "lustre" {
		return &ctrl.Result{}, dwsv1alpha2.NewResourceError("invalid file system type '%s' for setLustreOwnerGroup", nnfStorage.Spec.FileSystemType).WithFatal()
	}

	// If this NnfStorage is for a standalone MGT, then we don't need to set the owner and group
	if len(nnfStorage.Spec.AllocationSets) == 1 && nnfStorage.Spec.AllocationSets[0].Name == "mgt" {
		return nil, nil
	}

	index := func() int {
		for i, allocationSet := range nnfStorage.Spec.AllocationSets {
			if allocationSet.Name == "ost" {
				return i
			}
		}
		return -1
	}()

	if index == -1 {
		return &ctrl.Result{}, dwsv1alpha2.NewResourceError("no ost allocation set").WithFatal()
	}

	allocationSet := nnfStorage.Spec.AllocationSets[index]
	if len(allocationSet.Nodes) == 0 {
		return &ctrl.Result{}, dwsv1alpha2.NewResourceError("zero length node array for OST").WithFatal()
	}

	tempMountDir := os.Getenv("NNF_TEMP_MOUNT_PATH")
	if len(tempMountDir) == 0 {
		tempMountDir = "/mnt/tmp/"
	}

	clientMount := &dwsv1alpha2.ClientMount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-ownergroup", nnfStorage.Name),
			Namespace: allocationSet.Nodes[0].Name,
		},
	}

	if err := r.Get(ctx, client.ObjectKeyFromObject(clientMount), clientMount); err != nil {
		if !apierrors.IsNotFound(err) {
			return &ctrl.Result{}, dwsv1alpha2.NewResourceError("could not get clientmount for setting lustre owner/group").WithError(err).WithMajor()
		}
		index := func() int {
			for i, allocationSet := range nnfStorage.Spec.AllocationSets {
				if allocationSet.Name == "ost" {
					return i
				}
			}
			return -1
		}()

		if index == -1 {
			return &ctrl.Result{}, dwsv1alpha2.NewResourceError("no ost allocation set").WithFatal()
		}

		allocationSet := nnfStorage.Spec.AllocationSets[index]
		if len(allocationSet.Nodes) == 0 {
			return &ctrl.Result{}, dwsv1alpha2.NewResourceError("zero length node array for OST").WithFatal()
		}

		tempMountDir := os.Getenv("NNF_TEMP_MOUNT_PATH")
		if len(tempMountDir) == 0 {
			tempMountDir = "/mnt/tmp/"
		}

		dwsv1alpha2.InheritParentLabels(clientMount, nnfStorage)
		dwsv1alpha2.AddOwnerLabels(clientMount, nnfStorage)

		clientMount.Spec.Node = allocationSet.Nodes[0].Name
		clientMount.Spec.DesiredState = dwsv1alpha2.ClientMountStateMounted
		clientMount.Spec.Mounts = []dwsv1alpha2.ClientMountInfo{
			dwsv1alpha2.ClientMountInfo{
				Type:       nnfStorage.Spec.FileSystemType,
				TargetType: "directory",
				MountPath:  fmt.Sprintf("/%s/%s", tempMountDir, nnfNodeStorageName(nnfStorage, index, 0)),
				Device: dwsv1alpha2.ClientMountDevice{
					Type: dwsv1alpha2.ClientMountDeviceTypeLustre,
					Lustre: &dwsv1alpha2.ClientMountDeviceLustre{
						FileSystemName: allocationSet.FileSystemName,
						MgsAddresses:   nnfStorage.Status.MgsAddress,
					},
					DeviceReference: &dwsv1alpha2.ClientMountDeviceReference{
						ObjectReference: corev1.ObjectReference{
							Name:      nnfNodeStorageName(nnfStorage, index, 0),
							Namespace: allocationSet.Nodes[0].Name,
						},
					},
				},

				UserID:         nnfStorage.Spec.UserID,
				GroupID:        nnfStorage.Spec.GroupID,
				SetPermissions: true,
			},
		}

		if err := r.Create(ctx, clientMount); err != nil {
			return &ctrl.Result{}, dwsv1alpha2.NewResourceError("could not create lustre owner/group ClientMount resource").WithError(err).WithMajor()
		}

		log.Info("Created clientMount for setting Lustre owner/group")

		return &ctrl.Result{}, nil
	}

	if clientMount.Status.Error != nil {
		nnfStorage.Status.SetResourceError(clientMount.Status.Error)
	}

	if len(clientMount.Status.Mounts) == 0 {
		return &ctrl.Result{}, nil
	}

	switch clientMount.Status.Mounts[0].State {
	case dwsv1alpha2.ClientMountStateMounted:
		if clientMount.Status.Mounts[0].Ready == false {
			return &ctrl.Result{}, nil
		}

		clientMount.Spec.DesiredState = dwsv1alpha2.ClientMountStateUnmounted
		if err := r.Update(ctx, clientMount); err != nil {
			if !apierrors.IsConflict(err) {
				return &ctrl.Result{}, err
			}

			return &ctrl.Result{Requeue: true}, nil
		}

		log.Info("Updated clientMount to unmount Lustre owner/group mount")

		return &ctrl.Result{}, nil
	case dwsv1alpha2.ClientMountStateUnmounted:
		if clientMount.Status.Mounts[0].Ready == false {
			return &ctrl.Result{}, nil
		}

		// The ClientMount successfully unmounted. It will be deleted when the NnfStorage is deleted
		return nil, nil
	}

	return &ctrl.Result{}, nil
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
		&dwsv1alpha2.ClientMountList{},
		&nnfv1alpha1.NnfNodeStorageList{},
		&nnfv1alpha1.NnfNodeBlockStorageList{},
		&nnfv1alpha1.NnfStorageProfileList{},
	}

	maxReconciles := runtime.GOMAXPROCS(0)
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(controller.Options{MaxConcurrentReconciles: maxReconciles}).
		For(&nnfv1alpha1.NnfStorage{}).
		Watches(&nnfv1alpha1.NnfNodeStorage{}, handler.EnqueueRequestsFromMapFunc(dwsv1alpha2.OwnerLabelMapFunc)).
		Watches(&nnfv1alpha1.NnfNodeBlockStorage{}, handler.EnqueueRequestsFromMapFunc(dwsv1alpha2.OwnerLabelMapFunc)).
		Watches(&dwsv1alpha2.ClientMount{}, handler.EnqueueRequestsFromMapFunc(dwsv1alpha2.OwnerLabelMapFunc)).
		Complete(r)
}
