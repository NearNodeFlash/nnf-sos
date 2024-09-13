/*
 * Copyright 2024 Hewlett Packard Enterprise Development LP
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
	"reflect"
	"time"

	"github.com/go-logr/logr"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	dwsv1alpha2 "github.com/DataWorkflowServices/dws/api/v1alpha2"
	"github.com/DataWorkflowServices/dws/utils/updater"
	nnfv1alpha2 "github.com/NearNodeFlash/nnf-sos/api/v1alpha2"
	"github.com/NearNodeFlash/nnf-sos/internal/controller/metrics"
)

// NnfSystemStorageReconciler reconciles a NnfSystemStorage object
type NnfSystemStorageReconciler struct {
	client.Client
	Log          logr.Logger
	Scheme       *kruntime.Scheme
	ChildObjects []dwsv1alpha2.ObjectList
}

const (
	// finalizerNnfSystemStorage defines the key used for the finalizer
	finalizerNnfSystemStorage = "nnf.cray.hpe.com/nnf_system_storage"
)

// +kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfsystemstorages,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfsystemstorages/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfsystemstorages/finalizers,verbs=update
// +kubebuilder:rbac:groups=dataworkflowservices.github.io,resources=computes,verbs=get;create;list;watch;update;patch;delete;deletecollection
// +kubebuilder:rbac:groups=dataworkflowservices.github.io,resources=servers,verbs=get;create;list;watch;update;patch;delete;deletecollection
// +kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfstorages,verbs=get;create;list;watch;update;patch;delete;deletecollection
// +kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfaccesses,verbs=get;create;list;watch;update;patch;delete;deletecollection

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *NnfSystemStorageReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, err error) {
	log := r.Log.WithValues("NnfSystemStorage", req.NamespacedName)

	metrics.NnfSystemStorageReconcilesTotal.Inc()

	nnfSystemStorage := &nnfv1alpha2.NnfSystemStorage{}
	if err := r.Get(ctx, req.NamespacedName, nnfSystemStorage); err != nil {
		// ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	statusUpdater := updater.NewStatusUpdater[*nnfv1alpha2.NnfSystemStorageStatus](nnfSystemStorage)
	defer func() { err = statusUpdater.CloseWithStatusUpdate(ctx, r.Client.Status(), err) }()
	defer func() { nnfSystemStorage.Status.SetResourceErrorAndLog(err, log) }()

	if !nnfSystemStorage.GetDeletionTimestamp().IsZero() {
		if !controllerutil.ContainsFinalizer(nnfSystemStorage, finalizerNnfSystemStorage) {
			return ctrl.Result{}, nil
		}

		deleteStatus, err := dwsv1alpha2.DeleteChildren(ctx, r.Client, r.ChildObjects, nnfSystemStorage)
		if err != nil {
			return ctrl.Result{}, err
		}

		if !deleteStatus.Complete() {
			return ctrl.Result{RequeueAfter: time.Second}, nil
		}

		controllerutil.RemoveFinalizer(nnfSystemStorage, finalizerNnfSystemStorage)
		if err := r.Update(ctx, nnfSystemStorage); err != nil {
			if !apierrors.IsConflict(err) {
				return ctrl.Result{}, err
			}

			return ctrl.Result{Requeue: true}, nil
		}

		return ctrl.Result{}, nil
	}

	// Add the finalizer if it doesn't exist
	if !controllerutil.ContainsFinalizer(nnfSystemStorage, finalizerNnfSystemStorage) {
		controllerutil.AddFinalizer(nnfSystemStorage, finalizerNnfSystemStorage)
		if err := r.Update(ctx, nnfSystemStorage); err != nil {
			if !apierrors.IsConflict(err) {
				return ctrl.Result{}, err
			}

			return ctrl.Result{Requeue: true}, nil
		}

		return ctrl.Result{}, nil
	}

	nnfSystemStorage.Status.Ready = false

	if err := r.createServers(ctx, nnfSystemStorage); err != nil {
		return ctrl.Result{}, err
	}

	if err := r.createComputes(ctx, nnfSystemStorage); err != nil {
		return ctrl.Result{}, err
	}

	if err := r.createNnfStorage(ctx, nnfSystemStorage); err != nil {
		return ctrl.Result{}, err
	}

	wait, err := r.waitForNnfStorage(ctx, nnfSystemStorage)
	if err != nil {
		return ctrl.Result{}, err
	}
	if wait {
		return ctrl.Result{}, nil
	}

	if err := r.createNnfAccess(ctx, nnfSystemStorage); err != nil {
		return ctrl.Result{}, err
	}

	wait, err = r.waitForNnfAccess(ctx, nnfSystemStorage)
	if err != nil {
		return ctrl.Result{}, err
	}
	if wait {
		return ctrl.Result{}, nil
	}

	nnfSystemStorage.Status.Ready = true

	return ctrl.Result{}, nil
}

// Get the SystemConfiguration. If a SystemConfiguration is specified in the NnfSystemStorage, use that.
// Otherwise, use the default/default SystemConfiguration.
func (r *NnfSystemStorageReconciler) getSystemConfiguration(ctx context.Context, nnfSystemStorage *nnfv1alpha2.NnfSystemStorage) (*dwsv1alpha2.SystemConfiguration, error) {
	systemConfiguration := &dwsv1alpha2.SystemConfiguration{}

	if nnfSystemStorage.Spec.SystemConfiguration != (corev1.ObjectReference{}) {
		if nnfSystemStorage.Spec.SystemConfiguration.Kind != reflect.TypeOf(dwsv1alpha2.SystemConfiguration{}).Name() {
			return nil, dwsv1alpha2.NewResourceError("system configuration is not of kind '%s'", reflect.TypeOf(dwsv1alpha2.SystemConfiguration{}).Name()).WithFatal()
		}

		systemConfiguration.ObjectMeta = metav1.ObjectMeta{
			Name:      nnfSystemStorage.Spec.SystemConfiguration.Name,
			Namespace: nnfSystemStorage.Spec.SystemConfiguration.Namespace,
		}
	} else {
		systemConfiguration.ObjectMeta = metav1.ObjectMeta{
			Name:      "default",
			Namespace: corev1.NamespaceDefault,
		}
	}

	if err := r.Get(ctx, client.ObjectKeyFromObject(systemConfiguration), systemConfiguration); err != nil {
		return nil, dwsv1alpha2.NewResourceError("could not get systemconfiguration '%v'", client.ObjectKeyFromObject(systemConfiguration)).WithError(err)
	}

	return systemConfiguration, nil
}

// Get the StorageProfile specified in the spec. We don't look for the default profile, a profile must be
// specified in the NnfSystemStorage spec, and it must be marked as pinned.
func (r *NnfSystemStorageReconciler) getStorageProfile(ctx context.Context, nnfSystemStorage *nnfv1alpha2.NnfSystemStorage) (*nnfv1alpha2.NnfStorageProfile, error) {
	if nnfSystemStorage.Spec.StorageProfile == (corev1.ObjectReference{}) {
		return nil, dwsv1alpha2.NewResourceError("StorageProfile must be specified").WithFatal()
	}

	if nnfSystemStorage.Spec.StorageProfile.Kind != reflect.TypeOf(nnfv1alpha2.NnfStorageProfile{}).Name() {
		return nil, dwsv1alpha2.NewResourceError("StorageProfile is not of kind '%s'", reflect.TypeOf(nnfv1alpha2.NnfStorageProfile{}).Name()).WithFatal()
	}

	storageProfile := &nnfv1alpha2.NnfStorageProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nnfSystemStorage.Spec.StorageProfile.Name,
			Namespace: nnfSystemStorage.Spec.StorageProfile.Namespace,
		},
	}

	if err := r.Get(ctx, client.ObjectKeyFromObject(storageProfile), storageProfile); err != nil {
		return nil, dwsv1alpha2.NewResourceError("could not get StorageProfile '%v'", client.ObjectKeyFromObject(storageProfile)).WithError(err)
	}

	return storageProfile, nil
}

// Create a Servers resource with one allocation on each Rabbit. If the IncludeRabbits array is not
// empty, only use those Rabbits. Otherwise, use all the Rabbits in the SystemConfiguration resource except
// those specified in the ExcludeRabbits array.
func (r *NnfSystemStorageReconciler) createServers(ctx context.Context, nnfSystemStorage *nnfv1alpha2.NnfSystemStorage) error {
	log := r.Log.WithValues("NnfSystemStorage", client.ObjectKeyFromObject(nnfSystemStorage))

	// Create a list of Rabbits to use
	rabbitList := []string{}

	if len(nnfSystemStorage.Spec.IncludeRabbits) != 0 {
		if len(nnfSystemStorage.Spec.ExcludeRabbits) != 0 {
			return dwsv1alpha2.NewResourceError("IncludeRabbits and ExcludeRabbits can not be used together").WithFatal()
		}

		rabbitList = append([]string(nil), nnfSystemStorage.Spec.IncludeRabbits...)
	} else {
		systemConfiguration, err := r.getSystemConfiguration(ctx, nnfSystemStorage)
		if err != nil {
			return err
		}

		for _, storageNode := range systemConfiguration.Spec.StorageNodes {
			if storageNode.Type != "Rabbit" {
				continue
			}

			excluded := false
			for _, excludedRabbit := range nnfSystemStorage.Spec.ExcludeRabbits {
				if storageNode.Name == excludedRabbit {
					excluded = true
					break
				}
			}

			if excluded {
				continue
			}

			rabbitList = append(rabbitList, storageNode.Name)
		}
	}

	// Look at the Storages resources an determine whether each of the Rabbits is enabled. If any are not,
	// remove them from the list.
	if nnfSystemStorage.Spec.ExcludeDisabledRabbits {
		tempRabbitList := rabbitList[:0]
		for _, rabbit := range rabbitList {
			storage := &dwsv1alpha2.Storage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      rabbit,
					Namespace: corev1.NamespaceDefault,
				},
			}

			if err := r.Get(ctx, client.ObjectKeyFromObject(storage), storage); err != nil {
				return dwsv1alpha2.NewResourceError("could not get Storage '%v'", client.ObjectKeyFromObject(storage)).WithError(err)
			}

			labels := storage.GetLabels()
			if labels == nil {
				continue
			}

			if storageType := labels[dwsv1alpha2.StorageTypeLabel]; storageType != "Rabbit" {
				continue
			}

			if storage.Spec.State == dwsv1alpha2.DisabledState || storage.Status.Status != dwsv1alpha2.ReadyStatus {
				continue
			}

			tempRabbitList = append(tempRabbitList, rabbit)
		}
		rabbitList = tempRabbitList
	}

	// Use the Rabbit list to fill in the servers resource with one allocation per Rabbit
	servers := &dwsv1alpha2.Servers{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nnfSystemStorage.GetName(),
			Namespace: nnfSystemStorage.GetNamespace(),
		},
	}

	result, err := ctrl.CreateOrUpdate(ctx, r.Client, servers,
		func() error {
			dwsv1alpha2.AddOwnerLabels(servers, nnfSystemStorage)
			addDirectiveIndexLabel(servers, 0)

			servers.Spec.AllocationSets = []dwsv1alpha2.ServersSpecAllocationSet{{
				Label:          "system-storage",
				AllocationSize: nnfSystemStorage.Spec.Capacity,
			}}

			servers.Spec.AllocationSets[0].Storage = []dwsv1alpha2.ServersSpecStorage{}

			for _, rabbitName := range rabbitList {
				servers.Spec.AllocationSets[0].Storage = append(servers.Spec.AllocationSets[0].Storage, dwsv1alpha2.ServersSpecStorage{Name: rabbitName, AllocationCount: 1})
			}

			return ctrl.SetControllerReference(nnfSystemStorage, servers, r.Scheme)
		})

	if err != nil {
		return dwsv1alpha2.NewResourceError("CreateOrUpdate failed for servers: %v", client.ObjectKeyFromObject(servers)).WithError(err)
	}

	if result == controllerutil.OperationResultCreated {
		log.Info("Created servers", "name", servers.Name)
	} else if result == controllerutil.OperationResultNone {
		// no change
	} else {
		log.Info("Updated servers", "name", servers.Name)
	}

	return nil
}

// Create a computes resource based on the Rabbits that are used. If the IncludeComputes array is specified, then only use
// those computes. Otherwise, use the SystemConfiguration to determine which computes are attached to the Rabbits specified
// in the servers resource and exclude any computes listed in ExcludeComputes. Additionally, the ComputesTarget field determines
// which of the Rabbits computes to include: all, even, odd, or a custom list. This is done using the index of the compute node
// in the SystemConfiguration.
func (r *NnfSystemStorageReconciler) createComputes(ctx context.Context, nnfSystemStorage *nnfv1alpha2.NnfSystemStorage) error {
	log := r.Log.WithValues("NnfSystemStorage", client.ObjectKeyFromObject(nnfSystemStorage))

	// Get a list of compute nodes to use
	computeList := []string{}

	if len(nnfSystemStorage.Spec.IncludeComputes) != 0 {
		if len(nnfSystemStorage.Spec.ExcludeComputes) != 0 {
			return dwsv1alpha2.NewResourceError("IncludeComputes and ExcludeComputes can not be used together").WithFatal()
		}

		computeList = append([]string(nil), nnfSystemStorage.Spec.IncludeComputes...)
	} else {
		systemConfiguration, err := r.getSystemConfiguration(ctx, nnfSystemStorage)
		if err != nil {
			return err
		}

		servers := &dwsv1alpha2.Servers{
			ObjectMeta: metav1.ObjectMeta{
				Name:      nnfSystemStorage.GetName(),
				Namespace: nnfSystemStorage.GetNamespace(),
			},
		}
		if err := r.Get(ctx, client.ObjectKeyFromObject(servers), servers); err != nil {
			return dwsv1alpha2.NewResourceError("could not get Servers: %v", client.ObjectKeyFromObject(servers)).WithError(err)
		}

		// Create a map of the Rabbit node names so it's easy to search
		rabbitMap := map[string]bool{}
		for _, rabbitStorage := range servers.Spec.AllocationSets[0].Storage {
			rabbitMap[rabbitStorage.Name] = true
		}

		// Make a list of compute node index values based on the ComputesTarget field
		var indexList []int
		switch nnfSystemStorage.Spec.ComputesTarget {
		case nnfv1alpha2.ComputesTargetAll:
			indexList = []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}
		case nnfv1alpha2.ComputesTargetEven:
			indexList = []int{0, 2, 4, 6, 8, 10, 12, 14}
		case nnfv1alpha2.ComputesTargetOdd:
			indexList = []int{1, 3, 5, 7, 9, 11, 13, 15}
		case nnfv1alpha2.ComputesTargetPattern:
			indexList = append([]int(nil), nnfSystemStorage.Spec.ComputesPattern...)
		default:
			return dwsv1alpha2.NewResourceError("undexpected ComputesTarget type '%s'", nnfSystemStorage.Spec.ComputesTarget).WithFatal()
		}

		indexMap := map[int]bool{}
		for _, index := range indexList {
			indexMap[index] = true
		}

		// Find the Rabbits in the SystemConfiguration and get the compute names for
		// the index values we're interested in
		for _, storageNode := range systemConfiguration.Spec.StorageNodes {
			if storageNode.Type != "Rabbit" {
				continue
			}

			if _, exists := rabbitMap[storageNode.Name]; !exists {
				continue
			}

			for _, compute := range storageNode.ComputesAccess {
				if _, exists := indexMap[compute.Index]; exists {
					excluded := false
					for _, excludedCompute := range nnfSystemStorage.Spec.ExcludeComputes {
						if compute.Name == excludedCompute {
							excluded = true
							break
						}
					}

					if excluded {
						continue
					}
					computeList = append(computeList, compute.Name)
				}
			}
		}
	}

	// Create a computes resource using the list of compute nodes.
	computes := &dwsv1alpha2.Computes{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nnfSystemStorage.GetName(),
			Namespace: nnfSystemStorage.GetNamespace(),
		},
	}

	result, err := ctrl.CreateOrUpdate(ctx, r.Client, computes,
		func() error {
			dwsv1alpha2.AddOwnerLabels(computes, nnfSystemStorage)
			addDirectiveIndexLabel(computes, 0)

			computes.Data = []dwsv1alpha2.ComputesData{}

			for _, computeName := range computeList {
				computes.Data = append(computes.Data, dwsv1alpha2.ComputesData{Name: computeName})
			}

			return ctrl.SetControllerReference(nnfSystemStorage, computes, r.Scheme)
		})

	if err != nil {
		return dwsv1alpha2.NewResourceError("CreateOrUpdate failed for computes: %v", client.ObjectKeyFromObject(computes)).WithError(err)
	}

	if result == controllerutil.OperationResultCreated {
		log.Info("Created computes", "name", computes.Name)
	} else if result == controllerutil.OperationResultNone {
		// no change
	} else {
		log.Info("Updated computes", "name", computes.Name)
	}

	return nil
}

// Create a NnfStorage resource using the list of Rabbits in the Servers resource
func (r *NnfSystemStorageReconciler) createNnfStorage(ctx context.Context, nnfSystemStorage *nnfv1alpha2.NnfSystemStorage) error {
	log := r.Log.WithValues("NnfSystemStorage", client.ObjectKeyFromObject(nnfSystemStorage))

	storageProfile, err := r.getStorageProfile(ctx, nnfSystemStorage)
	if err != nil {
		return err
	}

	servers := &dwsv1alpha2.Servers{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nnfSystemStorage.GetName(),
			Namespace: nnfSystemStorage.GetNamespace(),
		},
	}

	if err := r.Get(ctx, client.ObjectKeyFromObject(servers), servers); err != nil {
		return dwsv1alpha2.NewResourceError("could not get Servers: %v", client.ObjectKeyFromObject(servers)).WithError(err)
	}

	nnfStorage := &nnfv1alpha2.NnfStorage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nnfSystemStorage.GetName(),
			Namespace: nnfSystemStorage.GetNamespace(),
		},
	}

	result, err := ctrl.CreateOrUpdate(ctx, r.Client, nnfStorage,
		func() error {
			dwsv1alpha2.AddOwnerLabels(nnfStorage, nnfSystemStorage)
			addDirectiveIndexLabel(nnfStorage, 0)
			addPinnedStorageProfileLabel(nnfStorage, storageProfile)

			nnfStorage.Spec.FileSystemType = nnfSystemStorage.Spec.Type
			nnfStorage.Spec.UserID = 0
			nnfStorage.Spec.GroupID = 0

			// Need to remove all of the AllocationSets in the NnfStorage object before we begin
			nnfStorage.Spec.AllocationSets = []nnfv1alpha2.NnfStorageAllocationSetSpec{}

			// Iterate the Servers data elements to pull out the allocation sets for the server
			for i := range servers.Spec.AllocationSets {
				nnfAllocationSet := nnfv1alpha2.NnfStorageAllocationSetSpec{}

				nnfAllocationSet.Name = servers.Spec.AllocationSets[i].Label
				nnfAllocationSet.Capacity = servers.Spec.AllocationSets[i].AllocationSize
				nnfAllocationSet.SharedAllocation = true

				// Create Nodes for this allocation set.
				for _, storage := range servers.Spec.AllocationSets[i].Storage {
					node := nnfv1alpha2.NnfStorageAllocationNodes{Name: storage.Name, Count: storage.AllocationCount}
					nnfAllocationSet.Nodes = append(nnfAllocationSet.Nodes, node)
				}

				nnfStorage.Spec.AllocationSets = append(nnfStorage.Spec.AllocationSets, nnfAllocationSet)
			}

			return ctrl.SetControllerReference(nnfSystemStorage, nnfStorage, r.Scheme)
		})

	if err != nil {
		return dwsv1alpha2.NewResourceError("CreateOrUpdate failed for NnfStorage: %v", client.ObjectKeyFromObject(nnfStorage)).WithError(err)
	}

	if result == controllerutil.OperationResultCreated {
		log.Info("Created NnfStorage", "name", nnfStorage.Name)
	} else if result == controllerutil.OperationResultNone {
		// no change
	} else {
		log.Info("Updated NnfStorage", "name", nnfStorage.Name)
	}

	return nil
}

// Wait until the NnfStorage has completed. Any errors will bubble up to the NnfSystemStorage
func (r *NnfSystemStorageReconciler) waitForNnfStorage(ctx context.Context, nnfSystemStorage *nnfv1alpha2.NnfSystemStorage) (bool, error) {
	// Check whether the NnfStorage has finished
	nnfStorage := &nnfv1alpha2.NnfStorage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nnfSystemStorage.GetName(),
			Namespace: nnfSystemStorage.GetNamespace(),
		},
	}
	if err := r.Get(ctx, client.ObjectKeyFromObject(nnfStorage), nnfStorage); err != nil {
		return true, dwsv1alpha2.NewResourceError("could not get NnfStorage: %v", client.ObjectKeyFromObject(nnfStorage)).WithError(err)
	}

	// If the Status section has not been filled in yet, exit and wait.
	if len(nnfStorage.Status.AllocationSets) != len(nnfStorage.Spec.AllocationSets) {
		return true, nil
	}

	if nnfStorage.Status.Error != nil {
		return true, dwsv1alpha2.NewResourceError("storage resource error: %v", client.ObjectKeyFromObject(nnfStorage)).WithError(nnfStorage.Status.Error)
	}

	if !nnfStorage.Status.Ready {
		return true, nil
	}

	return false, nil
}

// Create an NnfAccess using the Computes resource we created earlier. This NnfAccess may or may not create any ClientMount
// resources depending on if MakeClientMounts was specified in the NnfSystemStorage spec. The NnfAccess target is "shared",
// meaning that multiple compute nodes will access the same storage.
func (r *NnfSystemStorageReconciler) createNnfAccess(ctx context.Context, nnfSystemStorage *nnfv1alpha2.NnfSystemStorage) error {
	log := r.Log.WithValues("NnfSystemStorage", client.ObjectKeyFromObject(nnfSystemStorage))

	storageProfile, err := r.getStorageProfile(ctx, nnfSystemStorage)
	if err != nil {
		return err
	}

	nnfAccess := &nnfv1alpha2.NnfAccess{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nnfSystemStorage.GetName(),
			Namespace: nnfSystemStorage.GetNamespace(),
		},
	}

	// Create an NNFAccess for the compute clients
	result, err := ctrl.CreateOrUpdate(ctx, r.Client, nnfAccess,
		func() error {
			dwsv1alpha2.AddOwnerLabels(nnfAccess, nnfSystemStorage)
			addPinnedStorageProfileLabel(nnfAccess, storageProfile)
			addDirectiveIndexLabel(nnfAccess, 0)

			nnfAccess.Spec.TeardownState = dwsv1alpha2.StatePostRun
			nnfAccess.Spec.DesiredState = "mounted"
			nnfAccess.Spec.UserID = 0
			nnfAccess.Spec.GroupID = 0
			nnfAccess.Spec.Target = "shared"
			nnfAccess.Spec.MakeClientMounts = nnfSystemStorage.Spec.MakeClientMounts
			nnfAccess.Spec.MountPath = nnfSystemStorage.Spec.ClientMountPath
			nnfAccess.Spec.ClientReference = corev1.ObjectReference{
				Name:      nnfSystemStorage.GetName(),
				Namespace: nnfSystemStorage.GetNamespace(),
				Kind:      reflect.TypeOf(dwsv1alpha2.Computes{}).Name(),
			}

			nnfAccess.Spec.StorageReference = corev1.ObjectReference{
				Name:      nnfSystemStorage.GetName(),
				Namespace: nnfSystemStorage.GetNamespace(),
				Kind:      reflect.TypeOf(nnfv1alpha2.NnfStorage{}).Name(),
			}

			return ctrl.SetControllerReference(nnfSystemStorage, nnfAccess, r.Scheme)
		})
	if err != nil {
		return dwsv1alpha2.NewResourceError("Could not CreateOrUpdate compute node NnfAccess: %v", client.ObjectKeyFromObject(nnfAccess)).WithError(err)
	}

	if result == controllerutil.OperationResultCreated {
		log.Info("Created NnfAccess", "name", nnfAccess.Name)
	} else if result == controllerutil.OperationResultNone {
		// no change
	} else {
		log.Info("Updated NnfAccess", "name", nnfAccess.Name)
	}

	return nil
}

// Wait for the NnfAccess to be ready. Any errors are bubbled up to the NnfSystemStorage
func (r *NnfSystemStorageReconciler) waitForNnfAccess(ctx context.Context, nnfSystemStorage *nnfv1alpha2.NnfSystemStorage) (bool, error) {
	nnfAccess := &nnfv1alpha2.NnfAccess{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nnfSystemStorage.GetName(),
			Namespace: nnfSystemStorage.GetNamespace(),
		},
	}

	if err := r.Get(ctx, client.ObjectKeyFromObject(nnfAccess), nnfAccess); err != nil {
		return true, dwsv1alpha2.NewResourceError("could not get NnfAccess: %v", client.ObjectKeyFromObject(nnfAccess)).WithError(err)
	}

	if nnfAccess.Status.Error != nil {
		return true, dwsv1alpha2.NewResourceError("NnfAccess resource error: %v", client.ObjectKeyFromObject(nnfAccess)).WithError(nnfAccess.Status.Error)
	}

	if nnfAccess.Status.State != nnfAccess.Spec.DesiredState {
		return true, nil
	}

	if !nnfAccess.Status.Ready {
		return true, nil
	}

	return false, nil
}

// NnfSystemStorageEnqueueAll enqueues all of the NnfSystemStorage resources after a watch is triggered
func (r *NnfSystemStorageReconciler) NnfSystemStorageEnqueueAll(ctx context.Context, o client.Object) []reconcile.Request {
	log := r.Log.WithValues("NnfSystemStorage", "Enqueue All")

	requests := []reconcile.Request{}

	// Find all the NnfSystemStorage resources and add them to the Request list
	nnfSystemStorageList := &nnfv1alpha2.NnfSystemStorageList{}
	if err := r.List(context.TODO(), nnfSystemStorageList, []client.ListOption{}...); err != nil {
		log.Info("Could not list NnfSystemStorage", "error", err)
		return requests
	}

	for _, nnfSystemStorage := range nnfSystemStorageList.Items {
		requests = append(requests, reconcile.Request{NamespacedName: types.NamespacedName{Name: nnfSystemStorage.GetName(), Namespace: nnfSystemStorage.GetNamespace()}})
	}

	return requests
}

// SetupWithManager sets up the controller with the Manager.
func (r *NnfSystemStorageReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.ChildObjects = []dwsv1alpha2.ObjectList{
		&nnfv1alpha2.NnfAccessList{},
		&nnfv1alpha2.NnfStorageList{},
		&dwsv1alpha2.ComputesList{},
		&dwsv1alpha2.ServersList{},
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&nnfv1alpha2.NnfSystemStorage{}).
		Owns(&dwsv1alpha2.Computes{}).
		Owns(&dwsv1alpha2.Servers{}).
		Owns(&nnfv1alpha2.NnfStorage{}).
		Owns(&nnfv1alpha2.NnfAccess{}).
		Watches(&dwsv1alpha2.Storage{}, handler.EnqueueRequestsFromMapFunc(r.NnfSystemStorageEnqueueAll)).
		Complete(r)
}
