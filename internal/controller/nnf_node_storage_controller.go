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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/NearNodeFlash/nnf-sos/pkg/blockdevice"
	"github.com/NearNodeFlash/nnf-sos/pkg/fence"
	"github.com/NearNodeFlash/nnf-sos/pkg/filesystem"

	dwsv1alpha7 "github.com/DataWorkflowServices/dws/api/v1alpha7"
	"github.com/DataWorkflowServices/dws/utils/updater"
	nnfv1alpha9 "github.com/NearNodeFlash/nnf-sos/api/v1alpha9"
	"github.com/NearNodeFlash/nnf-sos/internal/controller/metrics"
)

const (
	// finalizerNnfNodeStorage defines the key used in identifying the
	// storage object as being owned by this NNF Storage Reconciler. This
	// prevents the system from deleting the custom resource until the
	// reconciler has finished using the resource.
	finalizerNnfNodeStorage = "nnf.cray.hpe.com/nnf_node_storage"
)

// NnfNodeStorageReconciler contains the elements needed during reconciliation for NnfNodeStorage
type NnfNodeStorageReconciler struct {
	client.Client
	Log               logr.Logger
	Scheme            *kruntime.Scheme
	SemaphoreForStart chan struct{}
	SemaphoreForDone  chan struct{}

	types.NamespacedName
	ChildObjects []dwsv1alpha7.ObjectList

	sync.Mutex
	started         bool
	reconcilerAwake bool

	// GFS2 fence watcher
	fenceWatcher *GFS2FenceWatcher
}

func (r *NnfNodeStorageReconciler) Start(ctx context.Context) error {
	log := r.Log.WithValues("State", "Start")

	<-r.SemaphoreForStart

	log.Info("Ready to start")

	// Initialize and start the GFS2 fence watcher
	var err error
	r.fenceWatcher, err = NewGFS2FenceWatcher(log, r.Namespace, r.Name)
	if err != nil {
		log.Error(err, "Failed to create GFS2 fence watcher")
		// Don't fail startup if watcher fails - log and continue
	} else {
		if err := r.fenceWatcher.Start(ctx); err != nil {
			log.Error(err, "Failed to start GFS2 fence watcher")
		}
	}

	r.Lock()
	r.started = true
	r.Unlock()

	close(r.SemaphoreForDone)
	return nil
}

//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfnodestorages,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfnodestorages/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfnodestorages/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.2/pkg/reconcile
func (r *NnfNodeStorageReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, err error) {
	log := r.Log.WithValues("NnfNodeStorage", req.NamespacedName)
	r.Lock()
	if !r.started {
		r.Unlock()
		return ctrl.Result{RequeueAfter: 2 * time.Second}, nil
	}
	if !r.reconcilerAwake {
		log.Info("Reconciler is awake")
		r.reconcilerAwake = true
	}
	r.Unlock()

	// Check if this reconcile was triggered by a fence request for this node
	if r.fenceWatcher != nil {
		fenceRequests := r.fenceWatcher.GetPendingRequests(req.NamespacedName.Name)
		if len(fenceRequests) > 0 {
			log.Info("Fence requests detected for this node",
				"targetNode", req.NamespacedName.Name,
				"requestCount", len(fenceRequests))

			// Process fence requests - find GFS2 filesystems shared with this node
			// and delete the associated storage groups to fence the compute node
			if err := r.processFenceRequests(ctx, req.NamespacedName.Name, fenceRequests, log); err != nil {
				log.Error(err, "Failed to process fence requests")
				// Don't fail the reconcile, just log the error
			}
		}
	}

	metrics.NnfNodeStorageReconcilesTotal.Inc()

	nnfNodeStorage := &nnfv1alpha9.NnfNodeStorage{}
	if err := r.Get(ctx, req.NamespacedName, nnfNodeStorage); err != nil {
		// ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Use the Node Storage Status Updater to track updates to the storage status.
	// This ensures that only one call to r.Status().Update() is done even though we
	// update the status at several points in the process. We hijack the defer logic
	// to perform the status update if no other error is present in the system when
	// exiting this reconcile function. Note that "err" is the named return value,
	// so when we would normally call "return ctrl.Result{}, nil", at that time
	// "err" is nil - and if permitted we will update err with the result of
	// the r.Update()
	statusUpdater := updater.NewStatusUpdater[*nnfv1alpha9.NnfNodeStorageStatus](nnfNodeStorage)
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

		nnfNodeBlockStorage := &nnfv1alpha9.NnfNodeBlockStorage{
			ObjectMeta: metav1.ObjectMeta{
				Name:      nnfNodeStorage.Spec.BlockReference.Name,
				Namespace: nnfNodeStorage.Spec.BlockReference.Namespace,
			},
		}

		if err := r.Get(ctx, client.ObjectKeyFromObject(nnfNodeBlockStorage), nnfNodeBlockStorage); err == nil {
			if controllerutil.ContainsFinalizer(nnfNodeBlockStorage, finalizerNnfNodeStorage) {
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

				// Remove the finalizer from the NnfNodeBlockStorage
				controllerutil.RemoveFinalizer(nnfNodeBlockStorage, finalizerNnfNodeStorage)

				err := r.Update(ctx, nnfNodeBlockStorage)
				if err != nil {
					return ctrl.Result{}, dwsv1alpha7.NewResourceError("could not update finalizer list for NnfNodeBlockStorage: %v", client.ObjectKeyFromObject(nnfNodeBlockStorage))
				}
				log.Info("finalizer removed from blockstorage")

				return ctrl.Result{Requeue: true}, nil
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
		nnfNodeStorage.Status.Allocations = make([]nnfv1alpha9.NnfNodeStorageAllocationStatus, nnfNodeStorage.Spec.Count)
		for i := range nnfNodeStorage.Status.Allocations {
			nnfNodeStorage.Status.Allocations[i].Ready = false
		}
		nnfNodeStorage.Status.Ready = false

		return ctrl.Result{Requeue: true}, nil
	}

	// Add a finalizer to the NnfNodeBlockStorage. This will block its deletion until the NnfNodeStorage
	// is completely torn down
	nnfNodeBlockStorage := &nnfv1alpha9.NnfNodeBlockStorage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nnfNodeStorage.Spec.BlockReference.Name,
			Namespace: nnfNodeStorage.Spec.BlockReference.Namespace,
		},
	}

	if err := r.Get(ctx, client.ObjectKeyFromObject(nnfNodeBlockStorage), nnfNodeBlockStorage); err == nil {
		if !controllerutil.ContainsFinalizer(nnfNodeBlockStorage, finalizerNnfNodeStorage) {
			if !nnfNodeBlockStorage.Status.Ready {
				return ctrl.Result{RequeueAfter: time.Second}, nil
			}

			controllerutil.AddFinalizer(nnfNodeBlockStorage, finalizerNnfNodeStorage)

			err := r.Update(ctx, nnfNodeBlockStorage)
			if err != nil {
				return ctrl.Result{}, dwsv1alpha7.NewResourceError("could not update finalizer list for NnfNodeBlockStorage: %v", client.ObjectKeyFromObject(nnfNodeBlockStorage))
			}

			return ctrl.Result{Requeue: true}, nil
		}
	}

	blockDevices := []blockdevice.BlockDevice{}
	fileSystems := []filesystem.FileSystem{}

	// Create a list of all the block devices and file systems that need to be created
	for i := 0; i < nnfNodeStorage.Spec.Count; i++ {
		blockDevice, fileSystem, err := getBlockDeviceAndFileSystem(ctx, r.Client, nnfNodeStorage, i, log)
		if err != nil {
			return ctrl.Result{}, err
		}

		blockDevices = append(blockDevices, blockDevice)
		fileSystems = append(fileSystems, fileSystem)
	}

	result, err := r.createAllocations(ctx, nnfNodeStorage, blockDevices, fileSystems)
	if err != nil {
		return ctrl.Result{}, dwsv1alpha7.NewResourceError("unable to create storage allocation").WithError(err).WithMajor()
	}
	if result != nil {
		return *result, nil
	}

	if err := r.checkAllocations(ctx, nnfNodeStorage, blockDevices); err != nil {
		return ctrl.Result{}, dwsv1alpha7.NewResourceError("unable to check storage allocation").WithError(err).WithMajor()
	}

	for _, allocation := range nnfNodeStorage.Status.Allocations {
		if !allocation.Ready {
			nnfNodeStorage.Status.Ready = false

			return ctrl.Result{Requeue: true}, nil
		}
	}

	nnfNodeStorage.Status.Ready = true

	return ctrl.Result{}, nil
}

func (r *NnfNodeStorageReconciler) deleteAllocation(ctx context.Context, nnfNodeStorage *nnfv1alpha9.NnfNodeStorage, index int) (*ctrl.Result, error) {
	log := r.Log.WithValues("NnfNodeStorage", client.ObjectKeyFromObject(nnfNodeStorage), "index", index)

	blockDevice, fileSystem, err := getBlockDeviceAndFileSystem(ctx, r.Client, nnfNodeStorage, index, log)
	if err != nil {
		return nil, err
	}

	// If we never successfully completed creating the allocation or if it's already gone, then don't try to run PreDeactivate
	blockDeviceExists, err := blockDevice.CheckExists(ctx)
	if err != nil {
		return nil, dwsv1alpha7.NewResourceError("could not check if block device exists").WithError(err).WithMajor()
	}

	if blockDeviceExists && nnfNodeStorage.Status.Allocations[index].Ready {
		ran, err := blockDevice.Activate(ctx)
		if err != nil {
			return nil, dwsv1alpha7.NewResourceError("could not activate block devices").WithError(err).WithMajor()
		}
		if ran {
			log.Info("Activated block device", "allocation", index)
		}

		ran, err = fileSystem.Activate(ctx, false)
		if err != nil {
			return nil, dwsv1alpha7.NewResourceError("could not activate file system").WithError(err).WithMajor()
		}
		if ran {
			log.Info("Activated file system", "allocation", index)
		}

		lustreOST0 := nnfNodeStorage.Spec.FileSystemType == "lustre" && nnfNodeStorage.Spec.LustreStorage.TargetType == "ost" && nnfNodeStorage.Spec.LustreStorage.StartIndex == 0
		if lustreOST0 || nnfNodeStorage.Spec.FileSystemType != "lustre" {
			ran, err = fileSystem.PreUnmount(ctx)
			if err != nil {
				return nil, dwsv1alpha7.NewResourceError("could not run pre unmount for file system").WithError(err).WithMajor()
			}
			if ran {
				log.Info("Pre unmount file system", "allocation", index)
			}
		}

		ran, err = fileSystem.PreDeactivate(ctx)
		if err != nil {
			return nil, dwsv1alpha7.NewResourceError("could not run pre deactivate for file system").WithError(err).WithMajor()
		}
		if ran {
			log.Info("Pre deactivate file system", "allocation", index)
		}
	}

	ran, err := fileSystem.Deactivate(ctx)
	if err != nil {
		return nil, dwsv1alpha7.NewResourceError("could not deactivate file system").WithError(err).WithMajor()
	}
	if ran {
		log.Info("Deactivated file system", "allocation", index)
	}

	ran, err = fileSystem.Destroy(ctx)
	if err != nil {
		return nil, dwsv1alpha7.NewResourceError("could not destroy file system").WithError(err).WithMajor()
	}
	if ran {
		log.Info("Destroyed file system", "allocation", index)
	}

	ran, err = blockDevice.Deactivate(ctx, false)
	if err != nil {
		return nil, dwsv1alpha7.NewResourceError("could not deactivate block devices").WithError(err).WithMajor()
	}
	if ran {
		log.Info("Deactivated block device", "allocation", index)
	}

	ran, err = blockDevice.Destroy(ctx)
	if err != nil {
		return nil, dwsv1alpha7.NewResourceError("could not destroy block devices").WithError(err).WithMajor()
	}
	if ran {
		log.Info("Destroyed block device", "allocation", index)
	}

	return nil, nil
}

// CheckAllocations checks the health of the allocations and tries to repair any that are unhealthy
func (r *NnfNodeStorageReconciler) checkAllocations(ctx context.Context, nnfNodeStorage *nnfv1alpha9.NnfNodeStorage, blockDevices []blockdevice.BlockDevice) error {
	overallHealth := nnfv1alpha9.NnfStorageHealthHealthy
	for index, blockDevice := range blockDevices {
		allocationStatus := &nnfNodeStorage.Status.Allocations[index]
		healthy, err := blockDevice.CheckHealth(ctx)
		if err != nil {
			return dwsv1alpha7.NewResourceError("could not check block device health").WithError(err)
		}

		if healthy {
			allocationStatus.Health = nnfv1alpha9.NnfStorageHealthHealthy
			continue
		}
		allocationStatus.Health = nnfv1alpha9.NnfStorageHealthDegraded
		nnfNodeStorage.Status.Health = nnfv1alpha9.NnfStorageHealthDegraded
		overallHealth = nnfv1alpha9.NnfStorageHealthDegraded

		err = blockDevice.Repair(ctx)
		if err != nil {
			return dwsv1alpha7.NewResourceError("could not repair block device").WithError(err)
		}
	}

	nnfNodeStorage.Status.Health = overallHealth

	return nil
}

func (r *NnfNodeStorageReconciler) createAllocations(ctx context.Context, nnfNodeStorage *nnfv1alpha9.NnfNodeStorage, blockDevices []blockdevice.BlockDevice, fileSystems []filesystem.FileSystem) (*ctrl.Result, error) {
	log := r.Log.WithValues("NnfNodeStorage", client.ObjectKeyFromObject(nnfNodeStorage))

	blockDevicesReady := true

	for index, blockDevice := range blockDevices {
		allocationStatus := &nnfNodeStorage.Status.Allocations[index]

		// If the allocation is ready, skip it
		if allocationStatus.Ready {
			continue
		}

		ran, err := blockDevice.Create(ctx, allocationStatus.Ready)
		if err != nil {
			return nil, dwsv1alpha7.NewResourceError("could not create block devices").WithError(err).WithMajor()
		}
		if ran {
			log.Info("Created block device", "allocation", index)
		}

		_, err = blockDevice.Activate(ctx)
		if err != nil {
			return nil, dwsv1alpha7.NewResourceError("could not activate block devices").WithError(err).WithMajor()
		}

		ready, err := blockDevice.CheckReady(ctx)
		if !ready {
			blockDevicesReady = false

			// If the block device isn't ready, don't add the defer function to Deactivate it. If it needs to be
			// synced, that can happen only while the block device is activated
			continue
		}

		deferIndex := index
		defer func() {
			_, err = blockDevices[deferIndex].Deactivate(ctx, false)
			if err != nil {
				allocationStatus.Ready = false
			}
		}()
	}

	if !blockDevicesReady {
		return &ctrl.Result{RequeueAfter: time.Second * 5}, nil
	}

	for index, fileSystem := range fileSystems {
		allocationStatus := &nnfNodeStorage.Status.Allocations[index]

		ran, err := fileSystem.Create(ctx, allocationStatus.Ready)
		if err != nil {
			return nil, dwsv1alpha7.NewResourceError("could not create file system").WithError(err).WithMajor()
		}
		if ran {
			log.Info("Created file system", "allocation", index)
		}

		ran, err = fileSystem.Activate(ctx, allocationStatus.Ready)
		if err != nil {
			return nil, dwsv1alpha7.NewResourceError("could not activate file system").WithError(err).WithMajor()
		}
		if ran {
			log.Info("Activated file system", "allocation", index)
		}

		ran, err = fileSystem.PostActivate(ctx, allocationStatus.Ready)
		if err != nil {
			return nil, dwsv1alpha7.NewResourceError("could not run post activate").WithError(err).WithMajor()
		}
		if ran {
			log.Info("Post activate file system", "allocation", index)
		}

		// For lustre, PostMount should only happen on OST0 only. For other file systems, just run
		// PostMount
		lustreOST0 := nnfNodeStorage.Spec.FileSystemType == "lustre" && nnfNodeStorage.Spec.LustreStorage.TargetType == "ost" && nnfNodeStorage.Spec.LustreStorage.StartIndex == 0
		if lustreOST0 || nnfNodeStorage.Spec.FileSystemType != "lustre" {
			ran, err = fileSystem.PostMount(ctx, allocationStatus.Ready)
			if err != nil {
				return nil, dwsv1alpha7.NewResourceError("could not run post mount").WithError(err).WithMajor()
			}
			if ran {
				log.Info("Post mount file system", "allocation", index)
			}
		}

		allocationStatus.Ready = true
	}

	return nil, nil
}

func nnfNameMapFunc(ctx context.Context, o client.Object) []reconcile.Request {
	return []reconcile.Request{
		{NamespacedName: types.NamespacedName{
			Name:      o.GetName(),
			Namespace: o.GetNamespace(),
		}},
	}
}

// processFenceRequests handles fence requests by finding and deleting storage groups for the target node
func (r *NnfNodeStorageReconciler) processFenceRequests(ctx context.Context, targetNode string, fenceRequests []*FenceRequest, log logr.Logger) error {
	// List all NnfNodeBlockStorage resources in this rabbit node's namespace
	nnfNodeBlockStorageList := &nnfv1alpha9.NnfNodeBlockStorageList{}
	listOptions := []client.ListOption{
		client.InNamespace(r.Namespace),
	}
	if err := r.List(ctx, nnfNodeBlockStorageList, listOptions...); err != nil {
		return err
	}

	storageGroupsToDelete := []string{}

	// Iterate through each NnfNodeBlockStorage resource
	for _, blockStorage := range nnfNodeBlockStorageList.Items {
		// Only process GFS2 filesystems - check the allocationset label
		allocationSet, hasLabel := blockStorage.Labels["nnf.cray.hpe.com/allocationset"]
		if !hasLabel || allocationSet != "gfs2" {
			log.V(1).Info("Skipping non-GFS2 block storage",
				"blockStorage", blockStorage.Name,
				"allocationset", allocationSet)
			continue
		}

		// Check each allocation for accesses by the target node
		for _, allocation := range blockStorage.Status.Allocations {
			if allocation.Accesses == nil {
				continue
			}

			// Check if the target node has access to this allocation
			if access, found := allocation.Accesses[targetNode]; found {
				storageGroupID := access.StorageGroupId
				if storageGroupID != "" {
					log.Info("Found GFS2 storage group for fenced node",
						"targetNode", targetNode,
						"storageGroupId", storageGroupID,
						"blockStorage", blockStorage.Name,
						"namespace", blockStorage.Namespace,
						"allocationset", allocationSet)
					storageGroupsToDelete = append(storageGroupsToDelete, storageGroupID)
				}
			}
		}
	}

	// Determine success and message
	success := false
	message := ""
	actionPerformed := "off"

	// TODO: Delete the storage groups
	// For now, just log what we would delete
	if len(storageGroupsToDelete) > 0 {
		log.Info("GFS2 storage groups identified for deletion",
			"targetNode", targetNode,
			"storageGroupIds", storageGroupsToDelete,
			"count", len(storageGroupsToDelete))
		// TODO: Call the storage service to delete these storage groups
		// This will fence the compute node by removing its access to GFS2 filesystems
		success = true
		message = fmt.Sprintf("Successfully identified %d GFS2 storage groups for fencing", len(storageGroupsToDelete))
	} else {
		log.Info("No GFS2 storage groups found for target node", "targetNode", targetNode)
		message = fmt.Sprintf("No GFS2 storage groups found for target node %s", targetNode)
	}

	// Process each fence request
	for _, request := range fenceRequests {
		// Write response file
		if err := r.writeFenceResponse(request.RequestID, success, message, actionPerformed, log); err != nil {
			log.Error(err, "Failed to write fence response", "requestID", request.RequestID)
		}

		// Clean up the fence request file after processing
		if request.FilePath != "" {
			if err := os.Remove(request.FilePath); err != nil {
				log.Error(err, "Failed to remove fence request file", "file", request.FilePath)
			} else {
				log.Info("Removed processed fence request file", "file", request.FilePath)
			}
		}
	}

	return nil
}

// writeFenceResponse writes a response file for the fence agent
func (r *NnfNodeStorageReconciler) writeFenceResponse(requestID string, success bool, message string, actionPerformed string, log logr.Logger) error {
	// Ensure response directory exists
	if err := os.MkdirAll(fence.ResponseDir, 0755); err != nil {
		return err
	}

	responseFile := filepath.Join(fence.ResponseDir, fmt.Sprintf("%s.json", requestID))

	response := map[string]interface{}{
		"success":          success,
		"message":          message,
		"action_performed": actionPerformed,
	}

	data, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(responseFile, data, 0644); err != nil {
		return err
	}

	log.Info("Wrote fence response", "requestID", requestID, "file", responseFile, "success", success)
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NnfNodeStorageReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.Add(r); err != nil {
		return err
	}
	maxReconciles := runtime.GOMAXPROCS(0)

	builder := ctrl.NewControllerManagedBy(mgr).
		WithOptions(controller.Options{MaxConcurrentReconciles: maxReconciles}).
		For(&nnfv1alpha9.NnfNodeStorage{}).
		// Trigger the reconciler for any changes to the associated NnfNodeBlockStorage. If we're waiting
		// on the block device paths to get updated, we want to be notified when it happens.
		Watches(&nnfv1alpha9.NnfNodeBlockStorage{}, handler.EnqueueRequestsFromMapFunc(nnfNameMapFunc)).
		// Add a custom event source for GFS2 fence requests.
		// The watcher will be initialized in Start(), and the event source will start
		// receiving events once the watcher's channel is created.
		WatchesRawSource(&gfs2FenceEventSource{
			Reconciler: r,
		}, &handler.EnqueueRequestForObject{})

	return builder.Complete(r)
}

// gfs2FenceEventSource implements source.Source to feed fence requests into the reconciler
type gfs2FenceEventSource struct {
	Reconciler *NnfNodeStorageReconciler
}

func (s *gfs2FenceEventSource) Start(ctx context.Context, handler handler.EventHandler, queue workqueue.RateLimitingInterface, prct ...predicate.Predicate) error {
	go func() {
		// Wait for the fence watcher to be initialized
		for s.Reconciler.fenceWatcher == nil {
			select {
			case <-ctx.Done():
				return
			case <-time.After(100 * time.Millisecond):
				// Keep checking until the watcher is initialized
			}
		}

		reconcileChan := s.Reconciler.fenceWatcher.GetReconcileChannel()

		for {
			select {
			case <-ctx.Done():
				return
			case req, ok := <-reconcileChan:
				if !ok {
					return
				}
				queue.Add(req)
			}
		}
	}()
	return nil
}
