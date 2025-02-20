/*
 * Copyright 2021-2025 Hewlett Packard Enterprise Development LP
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

	dwsv1alpha3 "github.com/DataWorkflowServices/dws/api/v1alpha3"
	"github.com/DataWorkflowServices/dws/utils/updater"
	nnfv1alpha6 "github.com/NearNodeFlash/nnf-sos/api/v1alpha6"
	"github.com/NearNodeFlash/nnf-sos/internal/controller/metrics"
	"github.com/NearNodeFlash/nnf-sos/pkg/var_handler"
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
//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnflustremgts,verbs=get;list;watch;create;update;patch;delete;deletecollection

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

	storage := &nnfv1alpha6.NnfStorage{}
	if err := r.Get(ctx, req.NamespacedName, storage); err != nil {
		// ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Create an updater for the entire node. This will handle calls to r.Status().Update() such
	// that we can repeatedly make calls to the internal update method, with the final update
	// occuring on the on function exit.
	statusUpdater := updater.NewStatusUpdater[*nnfv1alpha6.NnfStorageStatus](storage)
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
			return ctrl.Result{RequeueAfter: (2 * time.Second)}, nil
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
		storage.Status.AllocationSets = make([]nnfv1alpha6.NnfStorageAllocationSetStatus, len(storage.Spec.AllocationSets))
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

	// Collect the lists of nodes for each lustre component used for the filesystem
	if storage.Spec.FileSystemType == "lustre" {
		components := getLustreMappingFromStorage(storage)
		storage.Status.LustreComponents = nnfv1alpha6.NnfStorageLustreComponents{
			MDTs:     components["mdt"],
			MGTs:     components["mgt"],
			MGTMDTs:  components["mgtmdt"],
			OSTs:     components["ost"],
			NNFNodes: components["nnfNode"],
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
		res, err := r.aggregateNodeStorageStatus(ctx, storage, i, false, false)
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
		if !allocationSet.Ready {
			return ctrl.Result{}, nil
		}
	}

	if storage.Spec.FileSystemType == "lustre" && !storage.Status.Ready {
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

func (r *NnfStorageReconciler) addPersistentStorageReference(ctx context.Context, nnfStorage *nnfv1alpha6.NnfStorage, persistentMgsReference corev1.ObjectReference) error {
	persistentStorage := &dwsv1alpha3.PersistentStorageInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      persistentMgsReference.Name,
			Namespace: persistentMgsReference.Namespace,
		},
	}

	if err := r.Get(ctx, client.ObjectKeyFromObject(persistentStorage), persistentStorage); err != nil {
		return dwsv1alpha3.NewResourceError("").WithUserMessage("PersistentStorage '%v' not found", client.ObjectKeyFromObject(persistentStorage)).WithMajor()
	}

	if persistentStorage.Status.State != dwsv1alpha3.PSIStateActive {
		return dwsv1alpha3.NewResourceError("").WithUserMessage("PersistentStorage is not active").WithFatal()
	}

	// Add a consumer reference to the persistent storage for this directive
	reference := corev1.ObjectReference{
		Name:      nnfStorage.Name,
		Namespace: nnfStorage.Namespace,
		Kind:      reflect.TypeOf(nnfv1alpha6.NnfStorage{}).Name(),
	}

	for _, existingReference := range persistentStorage.Spec.ConsumerReferences {
		if existingReference == reference {
			return nil
		}
	}

	persistentStorage.Spec.ConsumerReferences = append(persistentStorage.Spec.ConsumerReferences, reference)

	return r.Update(ctx, persistentStorage)
}

func (r *NnfStorageReconciler) removePersistentStorageReference(ctx context.Context, nnfStorage *nnfv1alpha6.NnfStorage, persistentMgsReference corev1.ObjectReference) error {
	persistentStorage := &dwsv1alpha3.PersistentStorageInstance{
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
		Kind:      reflect.TypeOf(nnfv1alpha6.NnfStorage{}).Name(),
	}

	for i, existingReference := range persistentStorage.Spec.ConsumerReferences {
		if existingReference == reference {
			persistentStorage.Spec.ConsumerReferences = append(persistentStorage.Spec.ConsumerReferences[:i], persistentStorage.Spec.ConsumerReferences[i+1:]...)
			return r.Update(ctx, persistentStorage)
		}
	}

	return nil
}

func (r *NnfStorageReconciler) createNodeBlockStorage(ctx context.Context, nnfStorage *nnfv1alpha6.NnfStorage, allocationSetIndex int) (*ctrl.Result, error) {
	log := r.Log.WithValues("NnfStorage", client.ObjectKeyFromObject(nnfStorage))

	allocationSet := nnfStorage.Spec.AllocationSets[allocationSetIndex]
	fsType := nnfStorage.Spec.FileSystemType

	for i, node := range allocationSet.Nodes {
		// Per Rabbit namespace.
		nnfNodeBlockStorage := &nnfv1alpha6.NnfNodeBlockStorage{
			ObjectMeta: metav1.ObjectMeta{
				Name:      nnfNodeStorageName(nnfStorage, allocationSetIndex, i),
				Namespace: node.Name,
			},
		}

		result, err := ctrl.CreateOrUpdate(ctx, r.Client, nnfNodeBlockStorage,
			func() error {
				dwsv1alpha3.InheritParentLabels(nnfNodeBlockStorage, nnfStorage)
				dwsv1alpha3.AddOwnerLabels(nnfNodeBlockStorage, nnfStorage)

				labels := nnfNodeBlockStorage.GetLabels()
				labels[nnfv1alpha6.AllocationSetLabel] = allocationSet.Name
				nnfNodeBlockStorage.SetLabels(labels)

				expectedAllocations := node.Count
				if allocationSet.SharedAllocation {
					expectedAllocations = 1
				}
				nnfNodeBlockStorage.Spec.SharedAllocation = allocationSet.SharedAllocation

				if len(nnfNodeBlockStorage.Spec.Allocations) == 0 {
					nnfNodeBlockStorage.Spec.Allocations = make([]nnfv1alpha6.NnfNodeBlockStorageAllocationSpec, expectedAllocations)
				}

				if len(nnfNodeBlockStorage.Spec.Allocations) != expectedAllocations {
					return dwsv1alpha3.NewResourceError("block storage allocation count incorrect. found %v, expected %v", len(nnfNodeBlockStorage.Spec.Allocations), expectedAllocations).WithFatal()
				}

				for i := range nnfNodeBlockStorage.Spec.Allocations {

					// For lustre (zfs), bump up the capacity if less than the floor. This is to
					// ensure that zpool create does not fail if the size is too small.
					capacity := allocationSet.Capacity
					if fsType == "lustre" && capacity < minimumLustreAllocationSizeInBytes {
						capacity = minimumLustreAllocationSizeInBytes
					}
					if allocationSet.SharedAllocation {
						nnfNodeBlockStorage.Spec.Allocations[i].Capacity = capacity * int64(node.Count)
					} else {
						nnfNodeBlockStorage.Spec.Allocations[i].Capacity = capacity
					}

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
func (r *NnfStorageReconciler) aggregateNodeBlockStorageStatus(ctx context.Context, nnfStorage *nnfv1alpha6.NnfStorage, allocationSetIndex int) (*ctrl.Result, error) {
	log := r.Log.WithValues("NnfStorage", types.NamespacedName{Name: nnfStorage.Name, Namespace: nnfStorage.Namespace})

	allocationSet := &nnfStorage.Status.AllocationSets[allocationSetIndex]
	allocationSet.AllocationCount = 0

	nnfNodeBlockStorageList := &nnfv1alpha6.NnfNodeBlockStorageList{}
	matchLabels := dwsv1alpha3.MatchingOwner(nnfStorage)
	matchLabels[nnfv1alpha6.AllocationSetLabel] = nnfStorage.Spec.AllocationSets[allocationSetIndex].Name

	listOptions := []client.ListOption{
		matchLabels,
	}

	if err := r.List(ctx, nnfNodeBlockStorageList, listOptions...); err != nil {
		return &ctrl.Result{}, dwsv1alpha3.NewResourceError("could not list NnfNodeBlockStorages").WithError(err)
	}

	// make a map with empty data of the Rabbit names to allow easy searching
	nodeNameMap := map[string]struct{}{}
	for _, node := range nnfStorage.Spec.AllocationSets[allocationSetIndex].Nodes {
		nodeNameMap[node.Name] = struct{}{}
	}

	// prune out any entries that aren't in the NnfStorage. This can happen if the NnfStorage was modified
	// after it was created, as is the case with NnfStorages from an NnfSystemStorage
	nnfNodeBlockStorages := []nnfv1alpha6.NnfNodeBlockStorage{}
	for _, nnfNodeBlockStorage := range nnfNodeBlockStorageList.Items {
		if _, exists := nodeNameMap[nnfNodeBlockStorage.GetNamespace()]; exists {
			nnfNodeBlockStorages = append(nnfNodeBlockStorages, nnfNodeBlockStorage)
		}
	}

	for _, nnfNodeBlockStorage := range nnfNodeBlockStorages {
		for _, nodeAllocation := range nnfNodeBlockStorage.Status.Allocations {
			if nodeAllocation.CapacityAllocated > 0 {
				allocationSet.AllocationCount++
			}
		}
	}

	for _, nnfNodeBlockStorage := range nnfNodeBlockStorages {
		if nnfNodeBlockStorage.Status.Error != nil {
			return &ctrl.Result{}, dwsv1alpha3.NewResourceError("Node: %s", nnfNodeBlockStorage.GetNamespace()).WithError(nnfNodeBlockStorage.Status.Error)
		}
	}

	childTimeoutString := os.Getenv("NNF_CHILD_RESOURCE_TIMEOUT_SECONDS")
	if len(childTimeoutString) > 0 {
		childTimeout, err := strconv.Atoi(childTimeoutString)
		if err != nil {
			log.Info("Error: Invalid NNF_CHILD_RESOURCE_TIMEOUT_SECONDS. Defaulting to 300 seconds", "value", childTimeoutString)
			childTimeout = 300
		}

		for _, nnfNodeBlockStorage := range nnfNodeBlockStorages {
			// check if the finalizer has been added by the controller on the Rabbit
			if len(nnfNodeBlockStorage.GetFinalizers()) > 0 {
				continue
			}

			if nnfNodeBlockStorage.GetCreationTimestamp().Add(time.Duration(time.Duration(childTimeout) * time.Second)).Before(time.Now()) {
				return &ctrl.Result{}, dwsv1alpha3.NewResourceError("Node: %s: NnfNodeBlockStorage has not been reconciled after %d seconds", nnfNodeBlockStorage.GetNamespace(), childTimeout).WithMajor()
			}

			return &ctrl.Result{RequeueAfter: time.Minute}, nil
		}
	}

	for _, nnfNodeBlockStorage := range nnfNodeBlockStorages {
		if !nnfNodeBlockStorage.Status.Ready {
			return &ctrl.Result{}, nil
		}
	}

	// Ensure that we found all the NnfNodeBlockStorage resources we were expecting. This can be expected
	// transiently as it takes time for the client cache to be updated. Log a message in case the count
	// never reaches the expected value.
	if len(nnfNodeBlockStorages) != len(nnfStorage.Spec.AllocationSets[allocationSetIndex].Nodes) {
		if nnfStorage.GetDeletionTimestamp().IsZero() {
			log.Info("unexpected number of NnfNodeBlockStorages", "found", len(nnfNodeBlockStorages), "expected", len(nnfStorage.Spec.AllocationSets[allocationSetIndex].Nodes))
		}
		return &ctrl.Result{}, nil
	}

	return nil, nil
}

// Create an NnfNodeStorage if it doesn't exist, or update it if it requires updating. Each
// Rabbit node gets an NnfNodeStorage, and there may be multiple allocations requested in it.
// This limits the number of resources that have to be broadcast to the Rabbits.
func (r *NnfStorageReconciler) createNodeStorage(ctx context.Context, nnfStorage *nnfv1alpha6.NnfStorage, allocationSetIndex int) (*ctrl.Result, error) {
	log := r.Log.WithValues("NnfStorage", types.NamespacedName{Name: nnfStorage.Name, Namespace: nnfStorage.Namespace})

	if nnfStorage.Spec.FileSystemType == "lustre" {
		mgsAddress := nnfStorage.Spec.AllocationSets[allocationSetIndex].MgsAddress

		mgsNode := ""
		if mgsAddress == "" {
			for i, allocationSet := range nnfStorage.Spec.AllocationSets {
				if allocationSet.TargetType == "mgt" || allocationSet.TargetType == "mgtmdt" {
					// Wait for the MGT to be set up before creating nnfnodestorages for the other allocation sets
					if allocationSetIndex != i {
						if !nnfStorage.Status.AllocationSets[i].Ready {
							return nil, nil
						}
					}

					mgsNode = allocationSet.Nodes[0].Name

				}
			}

			if mgsNode != "" {
				nnfNode := &nnfv1alpha6.NnfNode{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "nnf-nlc",
						Namespace: mgsNode,
					},
				}

				if err := r.Get(ctx, client.ObjectKeyFromObject(nnfNode), nnfNode); err != nil {
					return &ctrl.Result{}, dwsv1alpha3.NewResourceError("could not get NnfNode: %v", client.ObjectKeyFromObject(nnfNode)).WithError(err)
				}

				mgsAddress = nnfNode.Status.LNetNid
			}
		}

		// Save the MGS address in the status section so we don't have to look in the NnfNodeStorage
		nnfStorage.Status.MgsAddress = mgsAddress

		// Create the NnfLustreMGT resource if this allocation set is for an MGT
		allocationSet := nnfStorage.Spec.AllocationSets[allocationSetIndex]
		if allocationSet.TargetType == "mgt" || allocationSet.TargetType == "mgtmdt" {
			nnfLustreMgt := &nnfv1alpha6.NnfLustreMGT{
				ObjectMeta: metav1.ObjectMeta{
					Name:      nnfStorage.GetName(),
					Namespace: mgsNode,
				},
				Spec: nnfv1alpha6.NnfLustreMGTSpec{
					Addresses:   []string{mgsAddress},
					FsNameStart: "aaaaaaaa",
				},
			}

			dwsv1alpha3.InheritParentLabels(nnfLustreMgt, nnfStorage)
			dwsv1alpha3.AddOwnerLabels(nnfLustreMgt, nnfStorage)
			if err := r.Create(ctx, nnfLustreMgt); err != nil {
				if !apierrors.IsAlreadyExists(err) {
					return nil, dwsv1alpha3.NewResourceError("could not create NnfLustreMGT").WithError(err).WithMajor()
				}
			} else {
				log.Info("Created NnfLustreMGT", "Name", nnfLustreMgt.Name, "Namespace", nnfLustreMgt.Namespace)
			}
		}

		// Pick an fsname if we haven't done so already. Standalone MGT doesn't need an fsname
		fsname := nnfStorage.Status.FileSystemName
		if fsname == "" && !(len(nnfStorage.Spec.AllocationSets) == 1 && nnfStorage.Spec.AllocationSets[0].Name == "mgt") {
			fsname, err := r.getFsName(ctx, nnfStorage)
			if err != nil {
				return nil, dwsv1alpha3.NewResourceError("could not get available fsname").WithError(err).WithMajor()
			}
			if fsname == "" {
				return &ctrl.Result{Requeue: true}, nil
			}

			nnfStorage.Status.FileSystemName = fsname
		}
	}

	allocationSet := nnfStorage.Spec.AllocationSets[allocationSetIndex]
	lustreOST := nnfStorage.Spec.FileSystemType == "lustre" && allocationSet.TargetType == "ost"

	// When creating lustre filesystems, we want to create Lustre OST0 last so we can signal to the
	// NnfNodeStorage controller when it is OK to run PostMount commands. OST0 should be created
	// last and only when all of the other NnfNodeStorage for each allocation sets is ready. Until
	// those are ready, skip the creation of OST0.
	skipOST0 := false
	if lustreOST {
		for i := range nnfStorage.Spec.AllocationSets {
			res, err := r.aggregateNodeStorageStatus(ctx, nnfStorage, i, false, true)
			if err != nil {
				return &ctrl.Result{}, err
			}

			if res != nil {
				if *res == (ctrl.Result{}) {
					skipOST0 = true // not ready, skip OST0
					continue
				} else {
					return res, nil
				}
			}
		}
	}

	startIndex := 0
	for i, node := range allocationSet.Nodes {
		// Per Rabbit namespace.
		nnfNodeStorage := &nnfv1alpha6.NnfNodeStorage{
			ObjectMeta: metav1.ObjectMeta{
				Name:      nnfNodeStorageName(nnfStorage, allocationSetIndex, i),
				Namespace: node.Name,
			},
		}

		// Do not create lustre OST0 until all other NnfNodeStorages are ready
		if lustreOST && startIndex == 0 && skipOST0 {
			startIndex += node.Count
			continue
		}

		storage := &dwsv1alpha3.Storage{
			ObjectMeta: metav1.ObjectMeta{
				Name:      node.Name,
				Namespace: corev1.NamespaceDefault,
			},
		}

		if err := r.Get(ctx, client.ObjectKeyFromObject(storage), storage); err != nil {
			return &ctrl.Result{}, dwsv1alpha3.NewResourceError("could not get Storage resource: %v", client.ObjectKeyFromObject(storage)).WithError(err)
		}

		result, err := ctrl.CreateOrUpdate(ctx, r.Client, nnfNodeStorage,
			func() error {
				dwsv1alpha3.InheritParentLabels(nnfNodeStorage, nnfStorage)
				dwsv1alpha3.AddOwnerLabels(nnfNodeStorage, nnfStorage)

				labels := nnfNodeStorage.GetLabels()
				labels[nnfv1alpha6.AllocationSetLabel] = allocationSet.Name
				if lustreOST && startIndex == 0 {
					labels[nnfv1alpha6.AllocationSetOST0Label] = "true"
				}
				nnfNodeStorage.SetLabels(labels)

				nnfNodeStorage.Spec.BlockReference = corev1.ObjectReference{
					Name:      nnfNodeStorageName(nnfStorage, allocationSetIndex, i),
					Namespace: node.Name,
					Kind:      reflect.TypeOf(nnfv1alpha6.NnfNodeBlockStorage{}).Name(),
				}
				nnfNodeStorage.Spec.Capacity = allocationSet.Capacity
				nnfNodeStorage.Spec.UserID = nnfStorage.Spec.UserID
				nnfNodeStorage.Spec.GroupID = nnfStorage.Spec.GroupID
				nnfNodeStorage.Spec.Count = node.Count
				nnfNodeStorage.Spec.SharedAllocation = allocationSet.SharedAllocation
				nnfNodeStorage.Spec.FileSystemType = nnfStorage.Spec.FileSystemType
				nnfNodeStorage.Spec.CommandVariables = []nnfv1alpha6.CommandVariablesSpec{}

				varMap := map[string]string{}
				for computeIndex := range storage.Status.Access.Computes {
					varMap[fmt.Sprintf("$COMPUTE%d_HOSTNAME", computeIndex)] = storage.Status.Access.Computes[computeIndex].Name
				}
				v := var_handler.NewVarHandler(varMap)

				for _, commandVariable := range allocationSet.CommandVariables {
					newCommandVariable := nnfv1alpha6.CommandVariablesSpec{
						Indexed: commandVariable.Indexed,
						Name:    commandVariable.Name,
						Value:   v.ReplaceAll(commandVariable.Value),
					}

					for _, indexedValue := range commandVariable.IndexedValues {
						newCommandVariable.IndexedValues = append(newCommandVariable.IndexedValues, v.ReplaceAll(indexedValue))
					}

					nnfNodeStorage.Spec.CommandVariables = append(nnfNodeStorage.Spec.CommandVariables, newCommandVariable)
				}

				if nnfStorage.Spec.FileSystemType == "lustre" {
					nnfNodeStorage.Spec.LustreStorage.StartIndex = startIndex
					nnfNodeStorage.Spec.LustreStorage.BackFs = allocationSet.BackFs
					nnfNodeStorage.Spec.LustreStorage.TargetType = allocationSet.TargetType
					nnfNodeStorage.Spec.LustreStorage.FileSystemName = nnfStorage.Status.FileSystemName
					nnfNodeStorage.Spec.LustreStorage.MgsAddress = nnfStorage.Status.MgsAddress
					nnfNodeStorage.Spec.LustreStorage.LustreComponents = nnfStorage.Status.LustreComponents

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
// for the NnfStorage. When skipOST0 is set, expect 1 less NnfNodeStorage resource when processing
// allocationSets for Lustre OST.
func (r *NnfStorageReconciler) aggregateNodeStorageStatus(ctx context.Context, storage *nnfv1alpha6.NnfStorage, allocationSetIndex int, deleting, skipOST0 bool) (*ctrl.Result, error) {
	log := r.Log.WithValues("NnfStorage", types.NamespacedName{Name: storage.Name, Namespace: storage.Namespace})
	lustreOST := storage.Spec.FileSystemType == "lustre" && storage.Spec.AllocationSets[allocationSetIndex].TargetType == "ost"

	nnfNodeStorageList := &nnfv1alpha6.NnfNodeStorageList{}
	matchLabels := dwsv1alpha3.MatchingOwner(storage)
	matchLabels[nnfv1alpha6.AllocationSetLabel] = storage.Spec.AllocationSets[allocationSetIndex].Name

	listOptions := []client.ListOption{
		matchLabels,
	}

	if err := r.List(ctx, nnfNodeStorageList, listOptions...); err != nil {
		return &ctrl.Result{}, dwsv1alpha3.NewResourceError("could not list NnfNodeStorages").WithError(err)
	}

	// make a map with empty data of the Rabbit names to allow easy searching
	nodeNameMap := map[string]struct{}{}
	for _, node := range storage.Spec.AllocationSets[allocationSetIndex].Nodes {
		nodeNameMap[node.Name] = struct{}{}
	}

	// prune out any entries that aren't in the NnfStorage. This can happen if the NnfStorage was modified
	// after it was created, as is the case with NnfStorages from an NnfSystemStorage
	nnfNodeStorages := []nnfv1alpha6.NnfNodeStorage{}
	for _, nnfNodeStorage := range nnfNodeStorageList.Items {
		if _, exists := nodeNameMap[nnfNodeStorage.GetNamespace()]; exists {
			nnfNodeStorages = append(nnfNodeStorages, nnfNodeStorage)
		}
	}

	for _, nnfNodeStorage := range nnfNodeStorages {
		// If we're in the delete path, only propagate errors for storages that are deleting. Errors
		// from creation aren't interesting anymore
		if deleting && nnfNodeStorage.GetDeletionTimestamp().IsZero() {
			continue
		}
		if nnfNodeStorage.Status.Error != nil {
			return &ctrl.Result{}, dwsv1alpha3.NewResourceError("Node: %s", nnfNodeStorage.GetNamespace()).WithError(nnfNodeStorage.Status.Error)
		}
	}

	childTimeoutString := os.Getenv("NNF_CHILD_RESOURCE_TIMEOUT_SECONDS")
	if len(childTimeoutString) > 0 {
		childTimeout, err := strconv.Atoi(childTimeoutString)
		if err != nil {
			log.Info("Error: Invalid NNF_CHILD_RESOURCE_TIMEOUT_SECONDS. Defaulting to 300 seconds", "value", childTimeoutString)
			childTimeout = 300
		}

		for _, nnfNodeStorage := range nnfNodeStorages {
			// check if the finalizer has been added by the controller on the Rabbit
			if len(nnfNodeStorage.GetFinalizers()) > 0 {
				continue
			}

			if nnfNodeStorage.GetCreationTimestamp().Add(time.Duration(time.Duration(childTimeout) * time.Second)).Before(time.Now()) {
				return &ctrl.Result{}, dwsv1alpha3.NewResourceError("Node: %s: NnfNodeStorage has not been reconciled after %d seconds", nnfNodeStorage.GetNamespace(), childTimeout).WithMajor()
			}

			return &ctrl.Result{RequeueAfter: time.Minute}, nil
		}
	}

	for _, nnfNodeStorage := range nnfNodeStorages {
		if !nnfNodeStorage.Status.Ready {
			return &ctrl.Result{}, nil
		}
	}

	// Ensure that we found all the NnfNodeStorage resources we were expecting. This can be expected
	// transiently as it takes time for the client cache to be updated. Log a message in case the count
	// never reaches the expected value.
	found := len(nnfNodeStorages)
	expected := len(storage.Spec.AllocationSets[allocationSetIndex].Nodes)

	// In the Lustre OST0 case, the NnfNodeStorage has not been created yet, so we can safely expect
	// 1 less than the total number of OSTs.
	if lustreOST && skipOST0 {
		expected = expected - 1
	}

	if found != expected {
		if storage.GetDeletionTimestamp().IsZero() {
			log.Info("unexpected number of NnfNodeStorages", "found", found, "expected", expected)
		}
		return &ctrl.Result{}, nil
	}

	storage.Status.AllocationSets[allocationSetIndex].Ready = true

	return nil, nil
}

func (r *NnfStorageReconciler) getLustreMgt(ctx context.Context, nnfStorage *nnfv1alpha6.NnfStorage) (*nnfv1alpha6.NnfLustreMGT, error) {
	if nnfStorage.Status.LustreMgtReference != (corev1.ObjectReference{}) {
		nnfLustreMgt := &nnfv1alpha6.NnfLustreMGT{
			ObjectMeta: metav1.ObjectMeta{
				Name:      nnfStorage.Status.LustreMgtReference.Name,
				Namespace: nnfStorage.Status.LustreMgtReference.Namespace,
			},
		}

		if err := r.Get(ctx, client.ObjectKeyFromObject(nnfLustreMgt), nnfLustreMgt); err != nil {
			return nil, dwsv1alpha3.NewResourceError("could not get nnfLustreMgt: %v", client.ObjectKeyFromObject(nnfLustreMgt)).WithError(err)
		}

		return nnfLustreMgt, nil
	}

	nnfLustreMgtList := &nnfv1alpha6.NnfLustreMGTList{}
	if err := r.List(ctx, nnfLustreMgtList, []client.ListOption{}...); err != nil {
		return nil, dwsv1alpha3.NewResourceError("could not list NnfLustreMGTs").WithError(err).WithMajor()
	}

	var nnfLustreMgt *nnfv1alpha6.NnfLustreMGT = nil
	for i := range nnfLustreMgtList.Items {
		if func(list []string, search string) bool {
			for _, element := range list {
				if element == search {
					return true
				}
			}
			return false
		}(nnfLustreMgtList.Items[i].Spec.Addresses, nnfStorage.Status.MgsAddress) == false {
			continue
		}

		if nnfLustreMgt != nil {
			return nil, dwsv1alpha3.NewResourceError("multiple MGTs found for address %s", nnfStorage.Status.MgsAddress).WithFatal().WithWLM()
		}

		nnfLustreMgt = &nnfLustreMgtList.Items[i]
	}

	if nnfLustreMgt == nil {
		return nil, dwsv1alpha3.NewResourceError("").WithUserMessage("no NnfLustreMGT resource found for MGS address: %s", nnfStorage.Status.MgsAddress).WithMajor()
	}

	return nnfLustreMgt, nil
}

func (r *NnfStorageReconciler) getFsName(ctx context.Context, nnfStorage *nnfv1alpha6.NnfStorage) (string, error) {
	nnfLustreMgt, err := r.getLustreMgt(ctx, nnfStorage)
	if err != nil {
		return "", dwsv1alpha3.NewResourceError("could not get NnfLustreMGT for address: %s", nnfStorage.Status.MgsAddress).WithError(err)
	}

	// Save the reference to the NnfLustreMGT resource in the NnfStorage before adding an fsname claim
	if nnfStorage.Status.LustreMgtReference == (corev1.ObjectReference{}) {
		nnfStorage.Status.LustreMgtReference = corev1.ObjectReference{
			Name:      nnfLustreMgt.Name,
			Namespace: nnfLustreMgt.Namespace,
			Kind:      reflect.TypeOf(nnfv1alpha6.NnfLustreMGT{}).Name(),
		}

		// This will update the status section of the NnfStorage with the reference and requeue
		return "", nil
	}

	reference := corev1.ObjectReference{
		Name:      nnfStorage.Name,
		Namespace: nnfStorage.Namespace,
		Kind:      reflect.TypeOf(nnfv1alpha6.NnfStorage{}).Name(),
	}

	// Check the status section of the NnfLustreMGT to see if an fsname has been assigned yet
	for _, existingClaim := range nnfLustreMgt.Status.ClaimList {
		if existingClaim.Reference == reference {
			return existingClaim.FsName, nil
		}
	}

	// Check whether the claim already exists in the Spec claim list
	for _, existingClaim := range nnfLustreMgt.Spec.ClaimList {
		if existingClaim == reference {
			return "", nil
		}
	}

	// Add our reference to the claim list
	nnfLustreMgt.Spec.ClaimList = append(nnfLustreMgt.Spec.ClaimList, reference)
	if err := r.Update(ctx, nnfLustreMgt); err != nil {
		if apierrors.IsConflict(err) {
			return "", nil
		}

		return "", dwsv1alpha3.NewResourceError("could not update NnfLustreMGT").WithError(err).WithMajor()
	}

	return "", nil

}

func (r *NnfStorageReconciler) setLustreOwnerGroup(ctx context.Context, nnfStorage *nnfv1alpha6.NnfStorage) (*ctrl.Result, error) {
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
		return &ctrl.Result{}, dwsv1alpha3.NewResourceError("invalid file system type '%s' for setLustreOwnerGroup", nnfStorage.Spec.FileSystemType).WithFatal()
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
		return &ctrl.Result{}, dwsv1alpha3.NewResourceError("no ost allocation set").WithFatal()
	}

	allocationSet := nnfStorage.Spec.AllocationSets[index]
	if len(allocationSet.Nodes) == 0 {
		return &ctrl.Result{}, dwsv1alpha3.NewResourceError("zero length node array for OST").WithFatal()
	}

	tempMountDir := os.Getenv("NNF_TEMP_MOUNT_PATH")
	if len(tempMountDir) == 0 {
		tempMountDir = "/mnt/tmp/"
	}

	clientMount := &dwsv1alpha3.ClientMount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-ownergroup", nnfStorage.Name),
			Namespace: allocationSet.Nodes[0].Name,
		},
	}

	if err := r.Get(ctx, client.ObjectKeyFromObject(clientMount), clientMount); err != nil {
		if !apierrors.IsNotFound(err) {
			return &ctrl.Result{}, dwsv1alpha3.NewResourceError("could not get clientmount for setting lustre owner/group").WithError(err).WithMajor()
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
			return &ctrl.Result{}, dwsv1alpha3.NewResourceError("no ost allocation set").WithFatal()
		}

		allocationSet := nnfStorage.Spec.AllocationSets[index]
		if len(allocationSet.Nodes) == 0 {
			return &ctrl.Result{}, dwsv1alpha3.NewResourceError("zero length node array for OST").WithFatal()
		}

		dwsv1alpha3.InheritParentLabels(clientMount, nnfStorage)
		dwsv1alpha3.AddOwnerLabels(clientMount, nnfStorage)

		clientMount.Spec.Node = allocationSet.Nodes[0].Name
		clientMount.Spec.DesiredState = dwsv1alpha3.ClientMountStateMounted
		clientMount.Spec.Mounts = []dwsv1alpha3.ClientMountInfo{
			{
				Type:       nnfStorage.Spec.FileSystemType,
				TargetType: "directory",
				MountPath:  getTempClientMountDir(nnfStorage, index),
				Device: dwsv1alpha3.ClientMountDevice{
					Type: dwsv1alpha3.ClientMountDeviceTypeLustre,
					Lustre: &dwsv1alpha3.ClientMountDeviceLustre{
						FileSystemName: nnfStorage.Status.FileSystemName,
						MgsAddresses:   nnfStorage.Status.MgsAddress,
					},
					DeviceReference: &dwsv1alpha3.ClientMountDeviceReference{
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
			return &ctrl.Result{}, dwsv1alpha3.NewResourceError("could not create lustre owner/group ClientMount resource").WithError(err).WithMajor()
		}

		log.Info("Created clientMount for setting Lustre owner/group")

		return &ctrl.Result{}, nil
	}

	if clientMount.Status.Error != nil {
		return &ctrl.Result{}, dwsv1alpha3.NewResourceError("Node: %s", clientMount.GetNamespace()).WithError(clientMount.Status.Error)
	}

	if len(clientMount.Status.Mounts) == 0 {
		return &ctrl.Result{}, nil
	}

	switch clientMount.Status.Mounts[0].State {
	case dwsv1alpha3.ClientMountStateMounted:
		if !clientMount.Status.Mounts[0].Ready {
			return &ctrl.Result{}, nil
		}

		clientMount.Spec.DesiredState = dwsv1alpha3.ClientMountStateUnmounted
		if err := r.Update(ctx, clientMount); err != nil {
			if !apierrors.IsConflict(err) {
				return &ctrl.Result{}, err
			}

			return &ctrl.Result{Requeue: true}, nil
		}

		log.Info("Updated clientMount to unmount Lustre owner/group mount")

		return &ctrl.Result{}, nil
	case dwsv1alpha3.ClientMountStateUnmounted:
		if !clientMount.Status.Mounts[0].Ready {
			return &ctrl.Result{}, nil
		}

		// The ClientMount successfully unmounted. It will be deleted when the NnfStorage is deleted
		return nil, nil
	}

	return &ctrl.Result{}, nil
}

func getTempMountDir() string {
	tempMountDir := os.Getenv("NNF_TEMP_MOUNT_PATH")
	if len(tempMountDir) == 0 {
		tempMountDir = "/mnt/tmp/"
	}
	return tempMountDir
}

func getTempClientMountDir(nnfStorage *nnfv1alpha6.NnfStorage, index int) string {
	return fmt.Sprintf("/%s/%s", getTempMountDir(), nnfNodeStorageName(nnfStorage, index, 0))
}

// Get the status from all the child NnfNodeStorage resources and use them to build the status
// for the NnfStorage.
func (r *NnfStorageReconciler) aggregateClientMountStatus(ctx context.Context, storage *nnfv1alpha6.NnfStorage, deleting bool) error {
	clientMountList := &dwsv1alpha3.ClientMountList{}
	matchLabels := dwsv1alpha3.MatchingOwner(storage)

	listOptions := []client.ListOption{
		matchLabels,
	}

	if err := r.List(ctx, clientMountList, listOptions...); err != nil {
		return dwsv1alpha3.NewResourceError("could not list ClientMounts").WithError(err)
	}

	for _, clientMount := range clientMountList.Items {
		// If we're in the delete path, only propagate errors for storages that are deleting. Errors
		// from creation aren't interesting anymore
		if deleting && clientMount.GetDeletionTimestamp().IsZero() {
			continue
		}
		if clientMount.Status.Error != nil {
			return dwsv1alpha3.NewResourceError("Node: %s", clientMount.GetNamespace()).WithError(clientMount.Status.Error)
		}
	}

	return nil
}

// Delete all the child NnfNodeStorage resources. Don't trust the client cache
// or the object references in the storage resource. We may have created children
// that aren't in the cache and we may not have been able to add the object reference
// to the NnfStorage.
func (r *NnfStorageReconciler) teardownStorage(ctx context.Context, storage *nnfv1alpha6.NnfStorage) (nodeStoragesState, error) {
	// Delete any clientmounts that were created by the NnfStorage.
	deleteStatus, err := dwsv1alpha3.DeleteChildren(ctx, r.Client, []dwsv1alpha3.ObjectList{&dwsv1alpha3.ClientMountList{}}, storage)
	if err != nil {
		return nodeStoragesExist, err
	}

	if err := r.aggregateClientMountStatus(ctx, storage, true); err != nil {
		return nodeStoragesExist, err
	}

	if !deleteStatus.Complete() {
		return nodeStoragesExist, nil
	}

	if storage.Spec.FileSystemType == "lustre" {
		childObjects := []dwsv1alpha3.ObjectList{
			&nnfv1alpha6.NnfNodeStorageList{},
		}

		// Delete OST0 first so that PreUnmount commands can happen
		ost0DeleteStatus, err := dwsv1alpha3.DeleteChildrenWithLabels(ctx, r.Client, childObjects, storage, client.MatchingLabels{nnfv1alpha6.AllocationSetOST0Label: "true"})
		if err != nil {
			return nodeStoragesExist, err
		}

		// Collect status information from the NnfNodeStorage resources and aggregate it into the
		// NnfStorage
		for i := range storage.Status.AllocationSets {
			_, err := r.aggregateNodeStorageStatus(ctx, storage, i, true, false)
			if err != nil {
				return nodeStoragesExist, err
			}
		}

		// Ensure OST0 is deleted before continuing
		if !ost0DeleteStatus.Complete() {
			return nodeStoragesExist, nil
		}

		// Then, delete the rest of the OSTs and MDTs so we can drop the claim on the NnfLustreMgt
		// resource. This will trigger an lctl command to run to remove the fsname from the MGT.
		ostDeleteStatus, err := dwsv1alpha3.DeleteChildrenWithLabels(ctx, r.Client, childObjects, storage, client.MatchingLabels{nnfv1alpha6.AllocationSetLabel: "ost"})
		if err != nil {
			return nodeStoragesExist, err
		}

		mdtDeleteStatus, err := dwsv1alpha3.DeleteChildrenWithLabels(ctx, r.Client, childObjects, storage, client.MatchingLabels{nnfv1alpha6.AllocationSetLabel: "mdt"})
		if err != nil {
			return nodeStoragesExist, err
		}

		// Collect status information from the NnfNodeStorage resources and aggregate it into the
		// NnfStorage
		for i := range storage.Status.AllocationSets {
			_, err := r.aggregateNodeStorageStatus(ctx, storage, i, true, false)
			if err != nil {
				return nodeStoragesExist, err
			}
		}

		if !ostDeleteStatus.Complete() || !mdtDeleteStatus.Complete() {
			return nodeStoragesExist, nil
		}

		// Remove the claim on the fsname from the MGT. Wait until the lctl command has run on the MGT
		// since this may be an MGT made as part of a jobdw
		released, err := r.releaseLustreMgt(ctx, storage)
		if err != nil {
			return nodeStoragesExist, dwsv1alpha3.NewResourceError("could not release LustreMGT resource").WithError(err)
		}

		if !released {
			return nodeStoragesExist, nil
		}

		// If this Lustre file system was using an MGS from a pool, then remove our reference to the PersistentStorageInstance
		// for the MGS
		for _, allocationSet := range storage.Spec.AllocationSets {
			if allocationSet.NnfStorageLustreSpec.PersistentMgsReference != (corev1.ObjectReference{}) {
				if err := r.removePersistentStorageReference(ctx, storage, allocationSet.NnfStorageLustreSpec.PersistentMgsReference); err != nil {
					return nodeStoragesExist, err
				}
			}
		}
	}

	// Delete any remaining child objects including the MGT allocation set for Lustre
	deleteStatus, err = dwsv1alpha3.DeleteChildren(ctx, r.Client, r.getChildObjects(), storage)
	if err != nil {
		return nodeStoragesExist, err
	}

	// Collect status information from the NnfNodeStorage resources and aggregate it into the
	// NnfStorage
	for i := range storage.Status.AllocationSets {
		_, err := r.aggregateNodeStorageStatus(ctx, storage, i, true, false)
		if err != nil {
			return nodeStoragesExist, err
		}
	}

	if !deleteStatus.Complete() {
		return nodeStoragesExist, nil
	}

	return nodeStoragesDeleted, nil
}

// releaseLustreMGT removes the claim from NnfLustreMGT and returns "true" once the NnfLustreMGT has removed
// the entry from the status section, indicating that the fsname has been removed from the MGT
func (r *NnfStorageReconciler) releaseLustreMgt(ctx context.Context, storage *nnfv1alpha6.NnfStorage) (bool, error) {
	if storage.Spec.FileSystemType != "lustre" {
		return true, nil
	}

	if storage.Status.LustreMgtReference == (corev1.ObjectReference{}) {
		return true, nil
	}

	nnfLustreMgt := &nnfv1alpha6.NnfLustreMGT{
		ObjectMeta: metav1.ObjectMeta{
			Name:      storage.Status.LustreMgtReference.Name,
			Namespace: storage.Status.LustreMgtReference.Namespace,
		},
	}

	if err := r.Get(ctx, client.ObjectKeyFromObject(nnfLustreMgt), nnfLustreMgt); err != nil {
		if apierrors.IsNotFound(err) {

			return true, nil
		}
		return false, dwsv1alpha3.NewResourceError("could not get nnfLustreMgt: %v", client.ObjectKeyFromObject(nnfLustreMgt)).WithError(err)
	}

	// Remove our claim from the spec section.
	for i, reference := range nnfLustreMgt.Spec.ClaimList {
		if reference.Name == storage.GetName() && reference.Namespace == storage.GetNamespace() {
			nnfLustreMgt.Spec.ClaimList = append(nnfLustreMgt.Spec.ClaimList[:i], nnfLustreMgt.Spec.ClaimList[i+1:]...)

			if err := r.Update(ctx, nnfLustreMgt); err != nil {
				return false, dwsv1alpha3.NewResourceError("could not remove reference from nnfLustreMgt: %v", client.ObjectKeyFromObject(nnfLustreMgt)).WithError(err)
			}

			return false, nil
		}
	}

	// Wait for the claim to disappear from the status section. This means the fsname has been erased from the MGT
	for _, claim := range nnfLustreMgt.Status.ClaimList {
		if claim.Reference.Name == storage.GetName() && claim.Reference.Namespace == storage.GetNamespace() {
			return false, nil
		}
	}

	return true, nil
}

// Build up the name of an NnfNodeStorage. This is a long name because:
// - NnfStorages from multiple namespaces create NnfNodeStorages in the same namespace
// - Different allocations in an NnfStorage could be targeting the same Rabbit node (e.g., MGS and MDS on the same Rabbit)
// - The same Rabbit node could be listed more than once within the same allocation.
func nnfNodeStorageName(storage *nnfv1alpha6.NnfStorage, allocationSetIndex int, i int) string {
	nodeName := storage.Spec.AllocationSets[allocationSetIndex].Nodes[i].Name

	// If the same Rabbit is listed more than once, the index on the end of the name needs to show
	// which instance this is.
	duplicateRabbitIndex := 0
	for j, node := range storage.Spec.AllocationSets[allocationSetIndex].Nodes {
		if j == i {
			break
		}

		if node.Name == nodeName {
			duplicateRabbitIndex++
		}
	}

	return storage.Namespace + "-" + storage.Name + "-" + storage.Spec.AllocationSets[allocationSetIndex].Name + "-" + strconv.Itoa(duplicateRabbitIndex)
}

// Get the NnfNodeStorage for Lustre OST0 for a given NnfStorage
func (r *NnfStorageReconciler) getLustreOST0(ctx context.Context, storage *nnfv1alpha6.NnfStorage) (*nnfv1alpha6.NnfNodeStorage, error) {
	if storage.Spec.FileSystemType != "lustre" {
		return nil, nil
	}

	// Get al the NnfNodeStorages for the OSTs
	nnfNodeStorageList := &nnfv1alpha6.NnfNodeStorageList{}
	matchLabels := dwsv1alpha3.MatchingOwner(storage)
	matchLabels[nnfv1alpha6.AllocationSetLabel] = "ost"

	listOptions := []client.ListOption{
		matchLabels,
	}

	if err := r.List(ctx, nnfNodeStorageList, listOptions...); err != nil {
		return nil, dwsv1alpha3.NewResourceError("could not list NnfNodeStorages").WithError(err)
	}

	for _, nnfNodeStorage := range nnfNodeStorageList.Items {
		if nnfNodeStorage.Spec.LustreStorage.StartIndex == 0 {
			return &nnfNodeStorage, nil
		}
	}

	return nil, nil
}

// Go through the Storage's allocation sets to determine the number of Lustre components and rabbit
// nodes. Returns a map with keys for each lustre component type and also the nnf nodes involved.
// The list of nnf nodes is kept unique, but mdts, osts, etc can include a node multiple times.
func getLustreMappingFromStorage(storage *nnfv1alpha6.NnfStorage) map[string][]string {
	nnfNodeKey := "nnfNode"
	componentMap := map[string][]string{
		"mdt":      {},
		"mgt":      {},
		"mgtmdt":   {},
		"ost":      {},
		nnfNodeKey: {},
	}
	rabbitMap := make(map[string]bool) // use a map to keep the list unique

	// Gather the info from the allocation set
	for _, allocationSet := range storage.Spec.AllocationSets {
		name := allocationSet.Name
		for _, storage := range allocationSet.Nodes {
			node := storage.Name

			// add to the list for that lustre component for each Count
			for i := 0; i < storage.Count; i++ {
				componentMap[name] = append(componentMap[name], node)
			}

			// add to the unique list of rabbits
			if _, found := rabbitMap[node]; !found {
				rabbitMap[node] = true
				componentMap[nnfNodeKey] = append(componentMap[nnfNodeKey], node)
			}
		}
	}

	return componentMap
}

func (r *NnfStorageReconciler) getChildObjects() []dwsv1alpha3.ObjectList {
	return []dwsv1alpha3.ObjectList{
		&dwsv1alpha3.ClientMountList{},
		&nnfv1alpha6.NnfNodeStorageList{},
		&nnfv1alpha6.NnfNodeBlockStorageList{},
		&nnfv1alpha6.NnfLustreMGTList{},
		&nnfv1alpha6.NnfStorageProfileList{},
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *NnfStorageReconciler) SetupWithManager(mgr ctrl.Manager) error {
	maxReconciles := runtime.GOMAXPROCS(0)
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(controller.Options{MaxConcurrentReconciles: maxReconciles}).
		For(&nnfv1alpha6.NnfStorage{}).
		Watches(&nnfv1alpha6.NnfNodeStorage{}, handler.EnqueueRequestsFromMapFunc(dwsv1alpha3.OwnerLabelMapFunc)).
		Watches(&nnfv1alpha6.NnfNodeBlockStorage{}, handler.EnqueueRequestsFromMapFunc(dwsv1alpha3.OwnerLabelMapFunc)).
		Watches(&dwsv1alpha3.ClientMount{}, handler.EnqueueRequestsFromMapFunc(dwsv1alpha3.OwnerLabelMapFunc)).
		Complete(r)
}
