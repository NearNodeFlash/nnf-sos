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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
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
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	dwsv1alpha7 "github.com/DataWorkflowServices/dws/api/v1alpha7"
	"github.com/DataWorkflowServices/dws/utils/updater"
	nnfv1alpha9 "github.com/NearNodeFlash/nnf-sos/api/v1alpha9"
	"github.com/NearNodeFlash/nnf-sos/internal/controller/metrics"
	"github.com/NearNodeFlash/nnf-sos/pkg/blockdevice/nvme"
	"github.com/NearNodeFlash/nnf-sos/pkg/command"
)

type ClientType string

const (
	ClientCompute ClientType = "Compute"
	ClientRabbit  ClientType = "Rabbit"
)

const (
	// finalizerNnfClientMount defines the finalizer name that this controller
	// uses on the ClientMount resource. This prevents the ClientMount resource
	// from being fully deleted until this controller removes the finalizer.
	finalizerNnfClientMount = "nnf.cray.hpe.com/nnf_clientmount"

	// filepath to append to clientmount directory to store lustre information in (servers resource)
	lustreServersFilepath = ".nnf-servers.json"

	// quickRetryDelay is the delay used when a quick retry is requested after ns-rescan discovers devices
	quickRetryDelay = 1 * time.Second
)

// quickRetryError is a sentinel error that indicates a quick retry should be performed
// rather than using exponential backoff. This is used when ns-rescan discovers new devices
// and we want to retry activation quickly.
type quickRetryError struct {
	err error
}

func (e *quickRetryError) Error() string {
	return e.err.Error()
}

func (e *quickRetryError) Unwrap() error {
	return e.err
}

func newQuickRetryError(err error) *quickRetryError {
	return &quickRetryError{err: err}
}

// NnfClientMountReconciler contains the pieces used by the reconciler
type NnfClientMountReconciler struct {
	client.Client
	Log               logr.Logger
	Scheme            *kruntime.Scheme
	SemaphoreForStart chan struct{}
	ClientType        ClientType

	sync.Mutex
	started         bool
	reconcilerAwake bool
}

func (r *NnfClientMountReconciler) Start(ctx context.Context) error {
	log := r.Log.WithValues("State", "Start")

	<-r.SemaphoreForStart

	log.Info("Ready to start")

	r.Lock()
	r.started = true
	r.Unlock()

	return nil
}

//+kubebuilder:rbac:groups=dataworkflowservices.github.io,resources=clientmounts,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=dataworkflowservices.github.io,resources=clientmounts/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=dataworkflowservices.github.io,resources=clientmounts/finalizers,verbs=update
//+kubebuilder:rbac:groups=dataworkflowservices.github.io,resources=servers,verbs=get;list;watch
//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfstorageprofiles,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *NnfClientMountReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, err error) {
	log := r.Log.WithValues("ClientMount", req.NamespacedName)
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

	metrics.NnfClientMountReconcilesTotal.Inc()

	clientMount := &dwsv1alpha7.ClientMount{}
	if err := r.Get(ctx, req.NamespacedName, clientMount); err != nil {
		// ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	// Create a status updater that handles the call to status().Update() if any of the fields
	// in clientMount.Status change
	statusUpdater := updater.NewStatusUpdater[*dwsv1alpha7.ClientMountStatus](clientMount)
	defer func() { err = statusUpdater.CloseWithStatusUpdate(ctx, r.Client.Status(), err) }()
	defer func() { clientMount.Status.SetResourceErrorAndLog(err, log) }()

	// Handle cleanup if the resource is being deleted
	if !clientMount.GetDeletionTimestamp().IsZero() {
		if !controllerutil.ContainsFinalizer(clientMount, finalizerNnfClientMount) {
			return ctrl.Result{}, nil
		}

		// Unmount everything before removing the finalizer
		log.Info("Unmounting all file systems due to resource deletion")
		if err := r.changeMountAll(ctx, clientMount, dwsv1alpha7.ClientMountStateUnmounted); err != nil {
			return ctrl.Result{}, err
		}

		// Rescan the NVMe devices now that the StorageGroup has been deleted on the Rabbit.
		// This is a workaround for a problem on some compute nodes where the NVMe namespaces don't
		// disappear by themselves. Only do this for file systems that are using local storage.
		localMount := false
		for _, mount := range clientMount.Spec.Mounts {
			if mount.Type != "lustre" {
				localMount = true
			}
		}

		if localMount {
			if err := nvme.NvmeRescanDevices(log); err != nil {
				return ctrl.Result{}, dwsv1alpha7.NewResourceError("could not rescan NVMe devices").WithError(err).WithMajor()
			}
		}

		for _, mount := range clientMount.Spec.Mounts {
			if mount.Type == "lustre" || mount.Device.Type != dwsv1alpha7.ClientMountDeviceTypeLVM {
				continue
			}

			existingDevices, err := nvme.NvmeListDevices(log)
			if err != nil {
				return ctrl.Result{}, dwsv1alpha7.NewResourceError("could not get NVMe device list").WithError(err).WithMajor()
			}

			for _, mountDevice := range mount.Device.LVM.NVMeInfo {
				for _, existingDevice := range existingDevices {
					if existingDevice.NQN == mountDevice.DeviceSerial && strconv.Itoa(int(existingDevice.NSID)) == mountDevice.NamespaceID {
						log.Info("NVMe device is still present but should have been removed", "NQN", mountDevice.DeviceSerial, "NSID", mountDevice.NamespaceID)
					}
				}
			}
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
		clientMount.Status.Mounts = make([]dwsv1alpha7.ClientMountInfoStatus, len(clientMount.Spec.Mounts))
	}

	// Initialize the status section if the desired state doesn't match the status state
	if clientMount.Status.Mounts[0].State != clientMount.Spec.DesiredState {
		r.resetMountStatus(clientMount)

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
	clientMount.Status.AllReady = false

	if err := r.changeMountAll(ctx, clientMount, clientMount.Spec.DesiredState); err != nil {
		resourceError := dwsv1alpha7.NewResourceError("mount/unmount failed").WithError(err)
		log.Info(resourceError.Error())

		// Check if this is a quick retry error (e.g., after ns-rescan discovered new devices)
		var qrErr *quickRetryError
		if errors.As(err, &qrErr) {
			log.Info("Quick retry requested after NVMe rescan", "delay", quickRetryDelay)
			return ctrl.Result{RequeueAfter: quickRetryDelay}, nil
		}

		return ctrl.Result{}, resourceError
	}

	clientMount.Status.AllReady = true

	return ctrl.Result{}, nil
}

// resetMountStatus resets all mount statuses to the desired state and marks them as not ready
func (r *NnfClientMountReconciler) resetMountStatus(clientMount *dwsv1alpha7.ClientMount) {
	for i := 0; i < len(clientMount.Status.Mounts); i++ {
		clientMount.Status.Mounts[i].State = clientMount.Spec.DesiredState
		clientMount.Status.Mounts[i].Ready = false
	}
	clientMount.Status.AllReady = false
}

// changeMmountAll mounts or unmounts all the file systems listed in the spec.Mounts list
func (r *NnfClientMountReconciler) changeMountAll(ctx context.Context, clientMount *dwsv1alpha7.ClientMount, state dwsv1alpha7.ClientMountState) error {
	// Reset status for all mounts if any are in a different state
	for i := range clientMount.Spec.Mounts {
		if clientMount.Status.Mounts[i].State != state {
			r.resetMountStatus(clientMount)
			break
		}
	}

	var firstError error
	for i := range clientMount.Spec.Mounts {

		var err error

		switch state {
		case dwsv1alpha7.ClientMountStateMounted:
			err = r.changeMount(ctx, clientMount, i, true /* shouldMount */)
		case dwsv1alpha7.ClientMountStateUnmounted:
			err = r.changeMount(ctx, clientMount, i, false /* should(NOT)Mount */)
		default:
			return dwsv1alpha7.NewResourceError("invalid desired state %s", state).WithFatal()
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
func (r *NnfClientMountReconciler) changeMount(ctx context.Context, clientMount *dwsv1alpha7.ClientMount, index int, shouldMount bool) error {
	log := r.Log.WithValues("ClientMount", client.ObjectKeyFromObject(clientMount), "index", index)

	clientMountInfo := clientMount.Spec.Mounts[index]
	nnfNodeStorage, err := r.fakeNnfNodeStorage(ctx, clientMount, index)
	if err != nil {
		return dwsv1alpha7.NewResourceError("unable to build NnfNodeStorage").WithError(err).WithMajor()
	}

	blockDevice, fileSystem, err := getBlockDeviceAndFileSystem(ctx, r.Client, nnfNodeStorage, clientMountInfo.Device.DeviceReference.Data, log)
	if err != nil {
		return dwsv1alpha7.NewResourceError("unable to get file system information").WithError(err).WithMajor()
	}

	if shouldMount {
		ran, err := blockDevice.PreActivate(ctx, clientMount.Status.Mounts[index].Ready)
		if err != nil {
			return dwsv1alpha7.NewResourceError("unable to run block device PreActivate commands").WithError(err).WithMajor()
		}
		if ran {
			log.Info("Ran PreActivate commands")
		}

		activated, err := blockDevice.Activate(ctx)
		if err != nil && clientMountInfo.Device.Type == dwsv1alpha7.ClientMountDeviceTypeLVM {
			// If we weren't able to activate the block device, then rescan for the NVMe namespaces. If the rescan is
			// successful the block device will be activated on the next reconcile
			if err := nvme.NvmeRescanDevices(log); err != nil {
				return dwsv1alpha7.NewResourceError("could not rescan NVMe devices").WithError(err).WithMajor()
			}

			existingDevices, err := nvme.NvmeListDevices(log)
			if err != nil {
				return dwsv1alpha7.NewResourceError("could not get NVMe device list").WithError(err).WithMajor()
			}

			// Check if all expected devices are now present after the rescan
			allDevicesFound := true
			for _, expectedDevice := range clientMountInfo.Device.LVM.NVMeInfo {
				found := false
				for _, existingDevice := range existingDevices {
					if existingDevice.NQN == expectedDevice.DeviceSerial && strconv.Itoa(int(existingDevice.NSID)) == expectedDevice.NamespaceID {
						found = true
						break
					}
				}
				if !found {
					log.Info("Could not find expected NVMe device", "NQN", expectedDevice.DeviceSerial, "NSID", expectedDevice.NamespaceID)
					allDevicesFound = false
				}
			}

			activateErr := dwsv1alpha7.NewResourceError("unable to activate block device").WithError(err).WithMajor()
			// If all devices are now present after rescan, request a quick retry
			if allDevicesFound {
				return newQuickRetryError(activateErr)
			}
			return activateErr
		}
		if activated {
			log.Info("Activated block device", "block device path", blockDevice.GetDevice())
		}

		ran, err = blockDevice.PostActivate(ctx, clientMount.Status.Mounts[index].Ready)
		if err != nil {
			return dwsv1alpha7.NewResourceError("unable to run block device PostActivate commands").WithError(err).WithMajor()
		}
		if ran {
			log.Info("Ran PostActivate commands")
		}

		ran, err = fileSystem.PreMount(ctx, clientMountInfo.MountPath, clientMount.Status.Mounts[index].Ready)
		if err != nil {
			return dwsv1alpha7.NewResourceError("unable to run file system PreMount commands").WithError(err).WithMajor()
		}
		if ran {
			log.Info("Ran PreMount commands", "Mount path", clientMountInfo.MountPath)
		}

		mounted, err := fileSystem.Mount(ctx, clientMountInfo.MountPath, clientMount.Status.Mounts[index].Ready)
		if err != nil {
			return dwsv1alpha7.NewResourceError("unable to mount file system").WithError(err).WithMajor()
		}
		if mounted {
			log.Info("Mounted file system", "Mount path", clientMountInfo.MountPath)
		}

		if clientMount.Spec.Mounts[index].SetPermissions {
			if err := os.Chown(clientMountInfo.MountPath, int(clientMount.Spec.Mounts[index].UserID), int(clientMount.Spec.Mounts[index].GroupID)); err != nil {
				return dwsv1alpha7.NewResourceError("unable to set owner and group for file system").WithError(err).WithMajor()
			}

			// If we're setting permissions then we know this is only happening once.  Dump the
			// servers resource to a file that can be accessed on the computes. Users can then
			// obtain ost/mdt information.
			// FIXME: decouple from SetPermissions?
			if clientMount.Spec.Mounts[index].Type == "lustre" {
				serversFilepath := filepath.Join(clientMountInfo.MountPath, lustreServersFilepath)
				if err := r.dumpServersToFile(ctx, clientMount, serversFilepath, clientMount.Spec.Mounts[index].UserID, clientMount.Spec.Mounts[index].GroupID); err != nil {
					return dwsv1alpha7.NewResourceError("unable to dump servers resource to file on clientmount path").WithError(err).WithMajor()
				}
			}
		}

		ran, err = fileSystem.PostMount(ctx, clientMountInfo.MountPath, clientMount.Status.Mounts[index].Ready)
		if err != nil {
			return dwsv1alpha7.NewResourceError("unable to run file system PostMount commands").WithError(err).WithMajor()
		}
		if ran {
			log.Info("Ran PostMount commands", "Mount path", clientMountInfo.MountPath)
		}

	} else {

		ran, err := fileSystem.PreUnmount(ctx, clientMountInfo.MountPath, clientMount.Status.Mounts[index].Ready)
		if err != nil {
			return dwsv1alpha7.NewResourceError("unable to run file system PreUnmount commands").WithError(err).WithMajor()
		}
		if ran {
			log.Info("Ran PreUnmount commands", "Mount path", clientMountInfo.MountPath)
		}

		// Skip the unmount/deactivate if already complete to avoid hitting expected errors
		if clientMount.Status.Mounts[index].Ready {
			log.Info("Skipping unmount/deactivate, already complete", "Mount path", clientMountInfo.MountPath)
			return nil
		}

		unmounted, err := fileSystem.Unmount(ctx, clientMountInfo.MountPath)
		if err != nil {
			return dwsv1alpha7.NewResourceError("unable to unmount file system").WithError(err).WithMajor()
		}
		if unmounted {
			log.Info("Unmounted file system", "Mount path", clientMountInfo.MountPath)
		}

		ran, err = fileSystem.PostUnmount(ctx, clientMountInfo.MountPath, clientMount.Status.Mounts[index].Ready)
		if err != nil {
			return dwsv1alpha7.NewResourceError("unable to run file system PostUnmount commands").WithError(err).WithMajor()
		}
		if ran {
			log.Info("Ran PostUnmount commands", "Mount path", clientMountInfo.MountPath)
		}

		ran, err = blockDevice.PreDeactivate(ctx, clientMount.Status.Mounts[index].Ready)
		if err != nil {
			return dwsv1alpha7.NewResourceError("unable to run block device PreDeactivate commands").WithError(err).WithMajor()
		}
		if ran {
			log.Info("Ran PreDeactivate commands")
		}

		// If this is an unmount on a compute node, we can fully deactivate the block device since we won't use it
		// again. If this is a rabbit node, we do a minimal deactivation. For LVM this means leaving the lockspace up
		fullDeactivate := false
		if r.ClientType == ClientCompute {
			fullDeactivate = true
		}
		deactivated, err := blockDevice.Deactivate(ctx, fullDeactivate)
		if err != nil {
			log.Error(err, "unable to deactivate block device")

			// Log some debug information to figure out why the deactivate failed
			devices, err := nvme.NvmeGetNamespaceDevices()
			if err != nil {
				log.Info("failed to get namespace devices", "error", err)
			} else {
				log.Info("deactivate failed", "current namespace devices", devices)
			}

			output, err := command.Run("dlm_tool ls -n", log)
			if err != nil {
				log.Info("failed to run dlm_tool", "error", err)
			} else {
				log.Info("deactivate failed", "dlm_tool output", output)
			}

			return dwsv1alpha7.NewResourceError("unable to deactivate block device").WithError(err).WithMajor()
		}
		if deactivated {
			log.Info("Deactivated block device", "block device path", blockDevice.GetDevice())
		}

		ran, err = blockDevice.PostDeactivate(ctx, clientMount.Status.Mounts[index].Ready)
		if err != nil {
			return dwsv1alpha7.NewResourceError("unable to run block device PostDeactivate commands").WithError(err).WithMajor()
		}
		if ran {
			log.Info("Ran PostDeactivate commands")
		}

	}

	return nil
}

// Retrieve the Servers resource for the workflow and write it to a dotfile on the mount path for compute users to retrieve
func (r *NnfClientMountReconciler) dumpServersToFile(ctx context.Context, clientMount *dwsv1alpha7.ClientMount, path string, uid, gid uint32) error {

	// Get the NnfServers Resource
	server, err := r.getServerForClientMount(ctx, clientMount)
	if err != nil {
		return dwsv1alpha7.NewResourceError("could not retrieve corresponding NnfServer resource for this ClientMount").WithError(err).WithMajor()
	}

	// Dump server resource to file on mountpoint (e.g. .nnf-lustre)
	file, err := os.Create(path)
	if err != nil {
		return dwsv1alpha7.NewResourceError("could not create servers file").WithError(err).WithMajor()
	}
	defer file.Close()

	encoder := json.NewEncoder(file)

	components := getLustreMappingFromServer(server)
	err = encoder.Encode(components)
	if err != nil {
		return dwsv1alpha7.NewResourceError("could not write JSON to file").WithError(err).WithMajor()
	}

	// Change permissions to user
	if err := os.Chown(path, int(uid), int(gid)); err != nil {
		return dwsv1alpha7.NewResourceError("unable to set owner and group").WithError(err).WithMajor()
	}

	return nil
}

// Retrieve the ClientMount's corresponding NnfServer resource. To do this, we first need to get the corresponding NnfStorage resource. That is done by
// looking at the owner of the ClientMount resource. It should be NnfStorage. Then, we inspect the NnfStorage resource's owner. In this case, there can
// be two different owners:
//
// 1. Workflow (non-persistent storage case)
// 2. PersistentStorageInstance (persistent storage case)
//
// Once we understand who owns the NnfStorage resource, we can then obtain the NnfServer resource through slightly different methods.
func (r *NnfClientMountReconciler) getServerForClientMount(ctx context.Context, clientMount *dwsv1alpha7.ClientMount) (*dwsv1alpha7.Servers, error) {
	storageKind := "NnfStorage"
	persistentKind := "PersistentStorageInstance"
	workflowKind := "Workflow"

	// Get the owner and directive index from ClientMount's labels
	ownerKind, ownerExists := clientMount.Labels[dwsv1alpha7.OwnerKindLabel]
	ownerName, ownerNameExists := clientMount.Labels[dwsv1alpha7.OwnerNameLabel]
	ownerNS, ownerNSExists := clientMount.Labels[dwsv1alpha7.OwnerNamespaceLabel]
	_, idxExists := clientMount.Labels[nnfv1alpha9.DirectiveIndexLabel]

	// We should expect the owner to be NnfStorage and have the expected labels
	if !ownerExists || !ownerNameExists || !ownerNSExists || !idxExists || ownerKind != storageKind {
		return nil, dwsv1alpha7.NewResourceError("expected owner to be of kind NnfStorage and have the expected labels").WithMajor()
	}

	// Retrieve the NnfStorage resource
	storage := &nnfv1alpha9.NnfStorage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ownerName,
			Namespace: ownerNS,
		},
	}
	if err := r.Get(ctx, client.ObjectKeyFromObject(storage), storage); err != nil {
		return nil, dwsv1alpha7.NewResourceError("unable retrieve NnfStorage resource").WithError(err).WithMajor()
	}

	// Get the owner and directive index from NnfStorage's labels
	ownerKind, ownerExists = storage.Labels[dwsv1alpha7.OwnerKindLabel]
	ownerName, ownerNameExists = storage.Labels[dwsv1alpha7.OwnerNameLabel]
	ownerNS, ownerNSExists = storage.Labels[dwsv1alpha7.OwnerNamespaceLabel]
	idx, idxExists := storage.Labels[nnfv1alpha9.DirectiveIndexLabel]

	// We should expect the owner of the NnfStorage to be Workflow or PersistentStorageInstance and
	// have the expected labels
	if !ownerExists || !ownerNameExists || !ownerNSExists || !idxExists || (ownerKind != workflowKind && ownerKind != persistentKind) {
		return nil, dwsv1alpha7.NewResourceError("expected owner to be of kind Workflow or PersistentStorageInstance and have the expected labels").WithMajor()
	}

	// If the owner is a workflow, then we can use the workflow labels and directive index to get
	// the Servers Resource.
	var listOptions []client.ListOption
	if ownerKind == workflowKind {
		listOptions = []client.ListOption{
			client.MatchingLabels(map[string]string{
				dwsv1alpha7.WorkflowNameLabel:      ownerName,
				dwsv1alpha7.WorkflowNamespaceLabel: ownerNS,
				nnfv1alpha9.DirectiveIndexLabel:    idx,
			}),
		}
	} else {
		// Otherwise the owner is a PersistentStorageInstance and we'll need to use the owner
		// labels. It also will not have a directive index.
		listOptions = []client.ListOption{
			client.MatchingLabels(map[string]string{
				dwsv1alpha7.OwnerKindLabel:      ownerKind,
				dwsv1alpha7.OwnerNameLabel:      ownerName,
				dwsv1alpha7.OwnerNamespaceLabel: ownerNS,
			}),
		}
	}

	serversList := &dwsv1alpha7.ServersList{}
	if err := r.List(ctx, serversList, listOptions...); err != nil {
		return nil, dwsv1alpha7.NewResourceError("unable retrieve NnfServers resource").WithError(err).WithMajor()
	}

	// We should only have 1
	if len(serversList.Items) != 1 {
		return nil, dwsv1alpha7.NewResourceError(fmt.Sprintf("wrong number of NnfServers resources: expected 1, got %d", len(serversList.Items))).WithMajor()
	}

	return &serversList.Items[0], nil
}

// Go through the Server's allocation sets to determine the number of Lustre components and rabbit
// nodes. Returns a map with keys for each lustre component type and also the nnf nodes involved. The
// list of nnf nodes is kept unique, but mdts, osts, etc can include a node multiple times.
func getLustreMappingFromServer(server *dwsv1alpha7.Servers) map[string][]string {
	nnfNodeKey := "nnfNode"
	components := map[string][]string{
		"mdt":      []string{},
		"mgt":      []string{},
		"mgtmdt":   []string{},
		"ost":      []string{},
		nnfNodeKey: []string{},
	}
	rabbitMap := make(map[string]bool) // use a map to keep the list unique

	// Gather the info from the allocation set
	for _, allocationSet := range server.Spec.AllocationSets {
		label := allocationSet.Label
		for _, storage := range allocationSet.Storage {
			node := storage.Name

			// add to the list for that lustre component for each allocationCount
			for i := 0; i < storage.AllocationCount; i++ {
				components[label] = append(components[label], node)
			}

			// add to the unique list of rabbits
			if _, found := rabbitMap[node]; !found {
				rabbitMap[node] = true
				components[nnfNodeKey] = append(components[nnfNodeKey], node)
			}
		}
	}

	return components
}

// fakeNnfNodeStorage creates an NnfNodeStorage resource filled in with only the fields
// that are necessary to mount the file system. This is done to reduce the API server load
// because the compute nodes don't need to Get() the actual NnfNodeStorage.
func (r *NnfClientMountReconciler) fakeNnfNodeStorage(ctx context.Context, clientMount *dwsv1alpha7.ClientMount, index int) (*nnfv1alpha9.NnfNodeStorage, error) {
	nnfNodeStorage := &nnfv1alpha9.NnfNodeStorage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clientMount.Spec.Mounts[index].Device.DeviceReference.ObjectReference.Name,
			Namespace: clientMount.Spec.Mounts[index].Device.DeviceReference.ObjectReference.Namespace,
			UID:       types.UID("fake_UID"),
		},
	}

	// These labels aren't exactly right (NnfStorage owns NnfNodeStorage), but the
	// labels that are important for doing the mount are there and correct
	dwsv1alpha7.InheritParentLabels(nnfNodeStorage, clientMount)
	labels := nnfNodeStorage.GetLabels()
	labels[nnfv1alpha9.DirectiveIndexLabel] = getTargetDirectiveIndexLabel(clientMount)
	labels[dwsv1alpha7.OwnerUidLabel] = getTargetOwnerUIDLabel(clientMount)
	nnfNodeStorage.SetLabels(labels)

	nnfNodeStorage.Spec.BlockReference = corev1.ObjectReference{
		Name:      "fake",
		Namespace: "fake",
		Kind:      "fake",
	}

	nnfNodeStorage.Spec.UserID = clientMount.Spec.Mounts[index].UserID
	nnfNodeStorage.Spec.GroupID = clientMount.Spec.Mounts[index].GroupID
	nnfNodeStorage.Spec.FileSystemType = clientMount.Spec.Mounts[index].Type
	nnfNodeStorage.Spec.Count = 1
	if nnfNodeStorage.Spec.FileSystemType == "none" {
		nnfNodeStorage.Spec.FileSystemType = "raw"
	}

	if clientMount.Spec.Mounts[index].Type == "lustre" {
		nnfNodeStorage.Spec.LustreStorage.BackFs = "none"
		nnfNodeStorage.Spec.LustreStorage.TargetType = "ost"
		nnfNodeStorage.Spec.LustreStorage.FileSystemName = clientMount.Spec.Mounts[index].Device.Lustre.FileSystemName
		nnfNodeStorage.Spec.LustreStorage.MgsAddress = clientMount.Spec.Mounts[index].Device.Lustre.MgsAddresses
	}

	nnfStorageProfile, err := getPinnedStorageProfileFromLabel(ctx, r.Client, nnfNodeStorage)
	if err != nil {
		return nil, dwsv1alpha7.NewResourceError("unable to find pinned storage profile").WithError(err).WithMajor()
	}

	switch nnfNodeStorage.Spec.FileSystemType {
	case "raw":
		nnfNodeStorage.Spec.SharedAllocation = nnfStorageProfile.Data.RawStorage.BlockDeviceCommands.SharedVg
	case "xfs":
		nnfNodeStorage.Spec.SharedAllocation = nnfStorageProfile.Data.XFSStorage.BlockDeviceCommands.SharedVg
	case "gfs2":
		nnfNodeStorage.Spec.SharedAllocation = nnfStorageProfile.Data.GFS2Storage.BlockDeviceCommands.SharedVg
	}

	return nnfNodeStorage, nil
}

func filterByRabbitNamespacePrefixForTest() predicate.Predicate {
	return predicate.NewPredicateFuncs(func(object client.Object) bool {
		return strings.HasPrefix(object.GetNamespace(), "rabbit")
	})
}

// SetupWithManager sets up the controller with the Manager.
func (r *NnfClientMountReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.Add(r); err != nil {
		return err
	}
	maxReconciles := runtime.GOMAXPROCS(0)
	builder := ctrl.NewControllerManagedBy(mgr).
		WithOptions(controller.Options{MaxConcurrentReconciles: maxReconciles}).
		For(&dwsv1alpha7.ClientMount{})

	if _, found := os.LookupEnv("NNF_TEST_ENVIRONMENT"); found {
		builder = builder.WithEventFilter(filterByRabbitNamespacePrefixForTest())
	}

	return builder.Complete(r)
}
