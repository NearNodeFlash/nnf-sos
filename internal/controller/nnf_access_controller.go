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
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"time"

	"github.com/go-logr/logr"
	"golang.org/x/exp/slices"
	"golang.org/x/sync/errgroup"

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

	dwsv1alpha5 "github.com/DataWorkflowServices/dws/api/v1alpha5"
	"github.com/DataWorkflowServices/dws/utils/updater"
	nnfv1alpha8 "github.com/NearNodeFlash/nnf-sos/api/v1alpha8"
	"github.com/NearNodeFlash/nnf-sos/internal/controller/metrics"
	"github.com/NearNodeFlash/nnf-sos/pkg/helpers"
)

// NnfAccessReconciler reconciles a NnfAccess object
type NnfAccessReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *kruntime.Scheme
}

const (
	// finalizerNnfAccess defines the key used for the finalizer
	finalizerNnfAccess = "nnf.cray.hpe.com/nnf_access"

	// NnfAccessAnnotation is an annotation applied to the NnfStorage object used to
	// prevent multiple accesses to a non-clustered file system
	NnfAccessAnnotation = "nnf.cray.hpe.com/access"
)

//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfaccesses,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfaccesses/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfaccesses/finalizers,verbs=update
//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfstorages,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfstorages/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=dataworkflowservices.github.io,resources=clientmounts,verbs=get;list;watch;create;update;patch;delete;deletecollection
//+kubebuilder:rbac:groups=dataworkflowservices.github.io,resources=clientmounts/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=dataworkflowservices.github.io,resources=systemstatuses,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *NnfAccessReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, err error) {
	log := r.Log.WithValues("NnfAccess", req.NamespacedName)

	metrics.NnfAccessReconcilesTotal.Inc()

	access := &nnfv1alpha8.NnfAccess{}
	if err := r.Get(ctx, req.NamespacedName, access); err != nil {
		// ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	statusUpdater := updater.NewStatusUpdater[*nnfv1alpha8.NnfAccessStatus](access)
	defer func() { err = statusUpdater.CloseWithStatusUpdate(ctx, r.Client.Status(), err) }()
	defer func() { access.Status.SetResourceErrorAndLog(err, log) }()

	// Create a list of names of the client nodes. This is pulled from either
	// the Computes resource specified in the ClientReference or the NnfStorage
	// resource when no ClientReference is provided. These correspond to mounting
	// the compute nodes during PreRun and mounting the rabbit nodes for data
	// movement.
	clientList, err := r.getClientList(ctx, access)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Pick one or more devices for each client to mount. Each client gets one device
	// when access.spec.target=single (used for computes), and each client gets as many
	// devices as it has access to when access.spec.target=all (used for rabbits).
	storageMapping, err := r.mapClientStorage(ctx, access, clientList)
	if err != nil {
		return ctrl.Result{}, err
	}

	if !access.GetDeletionTimestamp().IsZero() {
		if !controllerutil.ContainsFinalizer(access, finalizerNnfAccess) {
			return ctrl.Result{}, nil
		}

		deleteStatus, err := dwsv1alpha5.DeleteChildren(ctx, r.Client, r.getChildObjects(), access)
		if err != nil {
			return ctrl.Result{}, err
		}

		if !deleteStatus.Complete() {
			err := r.removeOfflineClientMounts(ctx, access)
			if err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{RequeueAfter: time.Second}, nil
		}

		err = r.removeBlockStorageAccess(ctx, access, storageMapping)
		if err != nil {
			return ctrl.Result{}, err
		}

		// Unlock the NnfStorage so it can be used by another NnfAccess
		if err = r.unlockStorage(ctx, access); err != nil {
			return ctrl.Result{}, err
		}

		controllerutil.RemoveFinalizer(access, finalizerNnfAccess)
		if err := r.Update(ctx, access); err != nil {
			if !apierrors.IsConflict(err) {
				return ctrl.Result{}, err
			}

			return ctrl.Result{Requeue: true}, nil
		}

		return ctrl.Result{}, nil
	}

	// Add the finalizer if it doesn't exist
	if !controllerutil.ContainsFinalizer(access, finalizerNnfAccess) {
		controllerutil.AddFinalizer(access, finalizerNnfAccess)
		if err := r.Update(ctx, access); err != nil {
			if !apierrors.IsConflict(err) {
				return ctrl.Result{}, err
			}

			return ctrl.Result{Requeue: true}, nil
		}

		return ctrl.Result{}, nil
	}

	// Reset the status block if the desired state has changed
	if access.Spec.DesiredState != access.Status.State {
		access.Status.State = access.Spec.DesiredState
		access.Status.Ready = false
		access.Status.Error = nil

		return ctrl.Result{Requeue: true}, nil
	}

	var result *ctrl.Result = nil

	if access.Status.State == "mounted" {
		result, err = r.mount(ctx, access, clientList, storageMapping)
		if err != nil {
			return ctrl.Result{}, dwsv1alpha5.NewResourceError("").WithError(err).WithUserMessage("unable to mount file system on client nodes")
		}
	} else {
		result, err = r.unmount(ctx, access, clientList, storageMapping)
		if err != nil {
			return ctrl.Result{}, dwsv1alpha5.NewResourceError("").WithError(err).WithUserMessage("unable to unmount file system from client nodes")
		}
	}

	if result != nil {
		return *result, nil
	}

	if access.Status.Ready == false {
		log.Info("State achieved", "State", access.Status.State)
	}

	access.Status.Ready = true
	access.Status.Error = nil

	return ctrl.Result{}, nil
}

func (r *NnfAccessReconciler) mount(ctx context.Context, access *nnfv1alpha8.NnfAccess, clientList []string, storageMapping map[string][]dwsv1alpha5.ClientMountInfo) (*ctrl.Result, error) {
	// Lock the NnfStorage by adding an annotation with the name/namespace for this
	// NnfAccess. This is used for non-clustered file systems that can only be mounted
	// from a single host.
	wait, err := r.lockStorage(ctx, access)
	if err != nil {
		return nil, dwsv1alpha5.NewResourceError("unable to lock storage").WithError(err)
	}

	if wait {
		return &ctrl.Result{RequeueAfter: time.Second}, nil
	}

	// Request that the devices be made available on the correct nodes
	err = r.addBlockStorageAccess(ctx, access, storageMapping)
	if err != nil {
		if apierrors.IsConflict(err) {
			return &ctrl.Result{RequeueAfter: time.Second * 2}, nil
		}

		return nil, dwsv1alpha5.NewResourceError("unable to add endpoints to NnfNodeStorage").WithError(err)
	}

	// Wait for all the devices to be made available on the correct nodes
	ready, err := r.getBlockStorageAccessStatus(ctx, access, storageMapping)
	if err != nil {
		return nil, dwsv1alpha5.NewResourceError("unable to check endpoints for NnfNodeStorage").WithError(err)
	}

	if !ready {
		return &ctrl.Result{RequeueAfter: time.Second * 2}, nil
	}

	// Create the ClientMount resources. One ClientMount resource is created per client
	err = r.manageClientMounts(ctx, access, storageMapping)
	if err != nil {
		if apierrors.IsConflict(err) {
			return &ctrl.Result{}, nil
		}

		return nil, dwsv1alpha5.NewResourceError("unable to create ClientMount resources").WithError(err)
	}

	// Aggregate the status from all the ClientMount resources
	ready, err = r.getClientMountStatus(ctx, access, clientList)
	if err != nil {
		return nil, dwsv1alpha5.NewResourceError("unable to check ClientMount status").WithError(err)
	}

	// Wait for all of the ClientMounts to be ready
	if ready == false {
		return &ctrl.Result{RequeueAfter: time.Minute}, nil
	}

	return nil, nil
}

func (r *NnfAccessReconciler) unmount(ctx context.Context, access *nnfv1alpha8.NnfAccess, clientList []string, storageMapping map[string][]dwsv1alpha5.ClientMountInfo) (*ctrl.Result, error) {
	// Update client mounts to trigger unmount operation
	err := r.manageClientMounts(ctx, access, storageMapping)
	if err != nil {
		return nil, dwsv1alpha5.NewResourceError("unable to update ClientMount resources").WithError(err)
	}

	// Aggregate the status from all the ClientMount resources
	ready, err := r.getClientMountStatus(ctx, access, clientList)
	if err != nil {
		return nil, dwsv1alpha5.NewResourceError("unable to get ClientMount status").WithError(err)
	}

	// Wait for all of the ClientMounts to be ready
	if ready == false {
		return &ctrl.Result{RequeueAfter: time.Second}, nil
	}

	err = r.removeBlockStorageAccess(ctx, access, storageMapping)
	if err != nil {
		return nil, dwsv1alpha5.NewResourceError("unable to remove NnfNodeStorage endpoints").WithError(err)
	}

	// Wait for all the devices to be removed from the correct nodes
	ready, err = r.getBlockStorageAccessStatus(ctx, access, storageMapping)
	if err != nil {
		return nil, dwsv1alpha5.NewResourceError("unable to check endpoints for NnfNodeStorage").WithError(err)
	}

	if !ready {
		return &ctrl.Result{RequeueAfter: time.Second * 2}, nil
	}

	// Unlock the NnfStorage so it can be used by another NnfAccess
	if err = r.unlockStorage(ctx, access); err != nil {
		return nil, dwsv1alpha5.NewResourceError("unable to unlock storage").WithError(err)
	}

	return nil, nil
}

// lockStorage applies an annotation to the NnfStorage resource with the name and namespace of the NnfAccess resource.
// This acts as a lock to prevent multiple NnfAccess resources from mounting the same file system. This is only necessary
// for non-clustered file systems
func (r *NnfAccessReconciler) lockStorage(ctx context.Context, access *nnfv1alpha8.NnfAccess) (bool, error) {

	if access.Spec.StorageReference.Kind != reflect.TypeOf(nnfv1alpha8.NnfStorage{}).Name() {
		return false, fmt.Errorf("invalid StorageReference kind %s", access.Spec.StorageReference.Kind)
	}

	namespacedName := types.NamespacedName{
		Name:      access.Spec.StorageReference.Name,
		Namespace: access.Spec.StorageReference.Namespace,
	}

	nnfStorage := &nnfv1alpha8.NnfStorage{}
	if err := r.Get(ctx, namespacedName, nnfStorage); err != nil {
		return false, err
	}

	if !controllerutil.ContainsFinalizer(nnfStorage, access.Name) {
		controllerutil.AddFinalizer(nnfStorage, access.Name)
	}

	// Clustered file systems don't need to add the annotation
	if nnfStorage.Spec.FileSystemType == "xfs" {
		// Read the current annotations and make an empty map if necessary
		annotations := nnfStorage.GetAnnotations()
		if annotations == nil {
			annotations = make(map[string]string)
		}

		// Check if the NnfAccess annotation exists. If it does, check if it matches the
		// information from this NnfAccess. If they don't match then the storage is mounted
		// somewhere already and can't be used until it's unmounted.
		value := access.Name + "/" + access.Namespace
		annotation, exists := annotations[NnfAccessAnnotation]
		if exists {
			if annotation == value {
				return false, nil
			}

			return true, nil
		}

		// Update the NnfStorage resource to add the annotation
		annotations[NnfAccessAnnotation] = value
		nnfStorage.SetAnnotations(annotations)
	}

	if err := r.Update(ctx, nnfStorage); err != nil {
		return false, err
	}

	return false, nil
}

// unlockStorage removes the NnfAccess annotation from an NnfStorage resource if it was added from lockStorage()
func (r *NnfAccessReconciler) unlockStorage(ctx context.Context, access *nnfv1alpha8.NnfAccess) error {
	nnfStorage := &nnfv1alpha8.NnfStorage{}

	if access.Spec.StorageReference.Kind != reflect.TypeOf(nnfv1alpha8.NnfStorage{}).Name() {
		return nil
	}

	namespacedName := types.NamespacedName{
		Name:      access.Spec.StorageReference.Name,
		Namespace: access.Spec.StorageReference.Namespace,
	}

	if err := r.Get(ctx, namespacedName, nnfStorage); err != nil {
		return err
	}

	if nnfStorage.Spec.FileSystemType == "xfs" {
		annotations := nnfStorage.GetAnnotations()
		if annotations == nil {
			annotations = make(map[string]string)
		}
		value, exists := annotations[NnfAccessAnnotation]
		if !exists {
			return nil
		}

		// Only unlock the NnfStorage if this NnfAccess was the one that
		// added the lock. The value of the annotation is the name/namespace
		// of the NnfAccess that applied the lock.
		if value != access.Name+"/"+access.Namespace {
			return nil
		}

		delete(annotations, NnfAccessAnnotation)
		nnfStorage.SetAnnotations(annotations)
	}

	if controllerutil.ContainsFinalizer(nnfStorage, access.Name) {
		controllerutil.RemoveFinalizer(nnfStorage, access.Name)
	}

	err := r.Update(ctx, nnfStorage)
	if err != nil {
		return err
	}

	return nil
}

// getClientList returns the list of client node names from either the Computes resource of the NnfStorage resource
func (r *NnfAccessReconciler) getClientList(ctx context.Context, access *nnfv1alpha8.NnfAccess) ([]string, error) {
	if access.Spec.ClientReference != (corev1.ObjectReference{}) {
		return r.getClientListFromClientReference(ctx, access)
	}

	return r.getClientListFromStorageReference(ctx, access)
}

// getClientListFromClientReference returns a list of client nodes names from the Computes resource
func (r *NnfAccessReconciler) getClientListFromClientReference(ctx context.Context, access *nnfv1alpha8.NnfAccess) ([]string, error) {
	computes := &dwsv1alpha5.Computes{}

	if access.Spec.ClientReference.Kind != reflect.TypeOf(dwsv1alpha5.Computes{}).Name() {
		return nil, fmt.Errorf("Invalid ClientReference kind %s", access.Spec.ClientReference.Kind)
	}

	namespacedName := types.NamespacedName{
		Name:      access.Spec.ClientReference.Name,
		Namespace: access.Spec.ClientReference.Namespace,
	}

	if err := r.Get(ctx, namespacedName, computes); err != nil {
		return nil, err
	}

	clients := []string{}
	for _, c := range computes.Data {
		clients = append(clients, c.Name)
	}

	return clients, nil
}

// getClientListFromStorageReference returns a list of client node names from the NnfStorage resource. This is the list of Rabbit
// nodes that host the storage
func (r *NnfAccessReconciler) getClientListFromStorageReference(ctx context.Context, access *nnfv1alpha8.NnfAccess) ([]string, error) {

	if access.Spec.StorageReference.Kind != reflect.TypeOf(nnfv1alpha8.NnfStorage{}).Name() {
		return nil, fmt.Errorf("Invalid StorageReference kind %s", access.Spec.StorageReference.Kind)
	}

	namespacedName := types.NamespacedName{
		Name:      access.Spec.StorageReference.Name,
		Namespace: access.Spec.StorageReference.Namespace,
	}

	nnfStorage := &nnfv1alpha8.NnfStorage{}
	if err := r.Get(ctx, namespacedName, nnfStorage); err != nil {
		return nil, err
	}

	clients := []string{}
	for _, allocationSetSpec := range nnfStorage.Spec.AllocationSets {
		if nnfStorage.Spec.FileSystemType == "lustre" {
			if allocationSetSpec.NnfStorageLustreSpec.TargetType != "ost" {
				continue
			}
		}

		for _, node := range allocationSetSpec.Nodes {
			clients = append(clients, node.Name)
		}
	}

	return clients, nil
}

// mapClientStorage returns a map of the clients with a list of mounts to make. This picks a device for each client
func (r *NnfAccessReconciler) mapClientStorage(ctx context.Context, access *nnfv1alpha8.NnfAccess, clients []string) (map[string][]dwsv1alpha5.ClientMountInfo, error) {
	nnfStorage := &nnfv1alpha8.NnfStorage{}

	if access.Spec.StorageReference.Kind != reflect.TypeOf(nnfv1alpha8.NnfStorage{}).Name() {
		return nil, fmt.Errorf("Invalid StorageReference kind %s", access.Spec.StorageReference.Kind)
	}

	namespacedName := types.NamespacedName{
		Name:      access.Spec.StorageReference.Name,
		Namespace: access.Spec.StorageReference.Namespace,
	}

	if err := r.Get(ctx, namespacedName, nnfStorage); err != nil {
		return nil, err
	}

	// Call a helper function depending on the storage type
	for i := range nnfStorage.Spec.AllocationSets {
		var storageMapping map[string][]dwsv1alpha5.ClientMountInfo
		var err error

		if nnfStorage.Spec.FileSystemType == "lustre" {
			storageMapping, err = r.mapClientNetworkStorage(ctx, access, clients, nnfStorage, i)
		} else {
			storageMapping, err = r.mapClientLocalStorage(ctx, access, clients, nnfStorage, i)
		}

		if err != nil {
			return nil, err
		}

		if storageMapping != nil {
			return storageMapping, nil
		}
	}

	return nil, nil
}

// mapClientNetworkStorage provides the Lustre MGS address information for the clients. All clients get the same
// mount information
func (r *NnfAccessReconciler) mapClientNetworkStorage(ctx context.Context, access *nnfv1alpha8.NnfAccess, clients []string, nnfStorage *nnfv1alpha8.NnfStorage, setIndex int) (map[string][]dwsv1alpha5.ClientMountInfo, error) {
	storageMapping := make(map[string][]dwsv1alpha5.ClientMountInfo)

	for _, client := range clients {
		mountInfo := dwsv1alpha5.ClientMountInfo{}
		mountInfo.Type = nnfStorage.Spec.FileSystemType
		mountInfo.TargetType = "directory"
		mountInfo.MountPath = access.Spec.MountPath
		mountInfo.Device.Type = dwsv1alpha5.ClientMountDeviceTypeLustre
		mountInfo.Device.Lustre = &dwsv1alpha5.ClientMountDeviceLustre{}
		mountInfo.Device.Lustre.FileSystemName = nnfStorage.Status.FileSystemName
		mountInfo.Device.Lustre.MgsAddresses = nnfStorage.Status.MgsAddress

		// Make it easy for the nnf-dm daemon to find the NnfStorage.
		mountInfo.Device.DeviceReference = &dwsv1alpha5.ClientMountDeviceReference{
			ObjectReference: access.Spec.StorageReference,
		}

		if os.Getenv("ENVIRONMENT") == "kind" {
			mountInfo.UserID = access.Spec.UserID
			mountInfo.GroupID = access.Spec.GroupID
		}

		storageMapping[client] = append(storageMapping[client], mountInfo)
	}

	return storageMapping, nil
}

// mapClientLocalStorage picks storage device(s) for each client to access based on locality information
// from the (DWS) Storage resources.
func (r *NnfAccessReconciler) mapClientLocalStorage(ctx context.Context, access *nnfv1alpha8.NnfAccess, clients []string, nnfStorage *nnfv1alpha8.NnfStorage, setIndex int) (map[string][]dwsv1alpha5.ClientMountInfo, error) {
	allocationSetSpec := nnfStorage.Spec.AllocationSets[setIndex]

	// Use information from the NnfStorage resource to determine how many allocations
	// are on each Rabbit (allocationCount) and how many NnfNodeStorage resources were
	// created for each Rabbit (instanceCount). instanceCount will be greater than one
	// if the same Rabbit was listed multiple times in the Servers resource.
	storageCountMap := make(map[string]struct {
		allocationCount int
		instanceCount   int
	})

	for _, node := range allocationSetSpec.Nodes {
		storageCount, exists := storageCountMap[node.Name]
		if exists {
			storageCount.allocationCount += node.Count
			storageCount.instanceCount += 1
			storageCountMap[node.Name] = storageCount
		} else {
			storageCount.allocationCount = node.Count
			storageCount.instanceCount = 1
			storageCountMap[node.Name] = storageCount
		}
	}

	// existingStorage is a map of Rabbits nodes and which storage they have
	existingStorage := make(map[string][]dwsv1alpha5.ClientMountInfo)

	// Read each NnfNodeStorage resource and find the NVMe information for each
	// allocation.
	for nodeName, storageCount := range storageCountMap {
		matchLabels := dwsv1alpha5.MatchingOwner(nnfStorage)
		matchLabels[nnfv1alpha8.AllocationSetLabel] = allocationSetSpec.Name

		listOptions := []client.ListOption{
			matchLabels,
			client.InNamespace(nodeName),
		}

		nnfNodeStorageList := &nnfv1alpha8.NnfNodeStorageList{}
		if err := r.List(ctx, nnfNodeStorageList, listOptions...); err != nil {
			return nil, err
		}

		// Check that the correct number of NnfNodeStorage resources were found for this
		// Rabbit.
		if len(nnfNodeStorageList.Items) != storageCount.instanceCount {
			return nil, dwsv1alpha5.NewResourceError("incorrect number of NnfNodeStorages. found %d. Needed %d.", len(nnfNodeStorageList.Items), storageCount.instanceCount).WithMajor()
		}

		for _, nnfNodeStorage := range nnfNodeStorageList.Items {
			if !nnfNodeStorage.Status.Ready {
				continue
			}
			nnfNodeBlockStorage := &nnfv1alpha8.NnfNodeBlockStorage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      nnfNodeStorage.Name,
					Namespace: nnfNodeStorage.Namespace,
				},
			}

			if err := r.Get(ctx, client.ObjectKeyFromObject(nnfNodeBlockStorage), nnfNodeBlockStorage); err != nil {
				return nil, dwsv1alpha5.NewResourceError("unable to get nnfNodeBlockStorage: %v", client.ObjectKeyFromObject(nnfNodeBlockStorage)).WithError(err).WithMajor()
			}

			// Loop through each allocation to pull out the device information and build the
			// mount information
			for i := 0; i < nnfNodeStorage.Spec.Count; i++ {
				mountInfo := dwsv1alpha5.ClientMountInfo{}

				// Set the DeviceReference to the NnfNodeStorage allocation regardless of whether we're mounting on
				// the Rabbit or the compute node. The compute node ClientMount device type will not be set to "reference",
				// so clientmountd will not look at the DeviceReference struct. The DeviceReference information is used by
				// the data movement code to match up mounts between the Rabbit and compute node.
				mountInfo.Device.DeviceReference = &dwsv1alpha5.ClientMountDeviceReference{}
				mountInfo.Device.DeviceReference.ObjectReference.Kind = reflect.TypeOf(nnfv1alpha8.NnfNodeStorage{}).Name()
				mountInfo.Device.DeviceReference.ObjectReference.Name = nnfNodeStorage.Name
				mountInfo.Device.DeviceReference.ObjectReference.Namespace = nnfNodeStorage.Namespace
				mountInfo.Device.DeviceReference.Data = i

				if nnfStorage.Spec.FileSystemType == "raw" {
					mountInfo.Type = "none"
					mountInfo.TargetType = "file"
					mountInfo.Options = "bind"
					mountInfo.UserID = access.Spec.UserID
					mountInfo.GroupID = access.Spec.GroupID
					mountInfo.SetPermissions = true
				} else {
					mountInfo.TargetType = "directory"
					mountInfo.Type = nnfStorage.Spec.FileSystemType
				}

				if os.Getenv("ENVIRONMENT") == "kind" {
					mountInfo.UserID = access.Spec.UserID
					mountInfo.GroupID = access.Spec.GroupID
				}

				// If no ClientReference exists, then the mounts are for the Rabbit nodes. Use references
				// to the NnfNodeStorage resource so the client mounter can access the swordfish objects
				if access.Spec.ClientReference == (corev1.ObjectReference{}) {
					indexMountDir := getIndexMountDir(nnfNodeStorage.Namespace, i)
					mountInfo.MountPath = filepath.Join(access.Spec.MountPathPrefix, indexMountDir)
					mountInfo.Device.Type = dwsv1alpha5.ClientMountDeviceTypeReference
				} else {
					mountInfo.MountPath = access.Spec.MountPath
					mountInfo.Device.Type = dwsv1alpha5.ClientMountDeviceTypeLVM
					mountInfo.Device.LVM = &dwsv1alpha5.ClientMountDeviceLVM{}
					mountInfo.Device.LVM.VolumeGroup = nnfNodeStorage.Status.Allocations[i].VolumeGroup
					mountInfo.Device.LVM.LogicalVolume = nnfNodeStorage.Status.Allocations[i].LogicalVolume
					mountInfo.Device.LVM.DeviceType = dwsv1alpha5.ClientMountLVMDeviceTypeNVMe
					blockAllocationSetIndex := i
					if nnfNodeBlockStorage.Spec.SharedAllocation {
						blockAllocationSetIndex = 0
					}

					for _, nvmeDesc := range nnfNodeBlockStorage.Status.Allocations[blockAllocationSetIndex].Devices {
						mountInfo.Device.LVM.NVMeInfo = append(mountInfo.Device.LVM.NVMeInfo, dwsv1alpha5.ClientMountNVMeDesc{DeviceSerial: nvmeDesc.NQN, NamespaceID: nvmeDesc.NamespaceId})
					}
				}

				existingStorage[nnfNodeStorage.Namespace] = append(existingStorage[nnfNodeStorage.Namespace], mountInfo)
			}
		}
	}

	// storageMapping is a map of clients and a list of mounts to perform. It is initialized
	// with an empty list of mounts for each client
	storageMapping := make(map[string][]dwsv1alpha5.ClientMountInfo)
	for _, client := range clients {
		storageMapping[client] = []dwsv1alpha5.ClientMountInfo{}
	}

	// Loop through each Rabbit node in the existingStorage map, and find a client for
	// each of the allocations. This is done by finding the compute and servers list from
	// the Storage resource.
	for storageName := range existingStorage {
		namespacedName := types.NamespacedName{
			Name:      storageName,
			Namespace: "default",
		}

		storage := &dwsv1alpha5.Storage{}
		err := r.Get(ctx, namespacedName, storage)
		if err != nil {
			return nil, err
		}

		// Build a list of all nodes with access to the storage
		clients := []string{}
		for _, compute := range storage.Status.Access.Computes {
			clients = append(clients, compute.Name)
		}

		for _, server := range storage.Status.Access.Servers {
			clients = append(clients, server.Name)
		}

		// Check if each node in the clients list needs a mount by checking for an entry in
		// the storageMapping map. If it does, pull an allocation off of the existingStorage map
		// entry and add it to the StorageMapping entry.
		for _, client := range clients {
			if _, ok := storageMapping[client]; !ok {
				continue
			}

			if len(existingStorage[storageName]) == 0 {
				return nil, dwsv1alpha5.NewResourceError("").WithUserMessage("invalid matching between clients and storage. Too many clients for storage").WithWLM().WithFatal()
			}

			// If target==all, then the client wants to access all the storage it can see
			switch access.Spec.Target {
			case "all":
				storageMapping[client] = append(storageMapping[client], existingStorage[storageName]...)
				existingStorage[storageName] = []dwsv1alpha5.ClientMountInfo{}
			case "single":
				storageMapping[client] = append(storageMapping[client], existingStorage[storageName][0])
				existingStorage[storageName] = existingStorage[storageName][1:]
			case "shared":
				// Allow storages to be re-used for the shared case
				storageMapping[client] = append(storageMapping[client], existingStorage[storageName][0])
			default:
				return nil, dwsv1alpha5.NewResourceError("invalid target type '%s'", access.Spec.Target).WithFatal()
			}
		}
	}

	return storageMapping, nil
}

type mountReference struct {
	client          string
	allocationIndex int
}

// addNodeStorageEndpoints adds the compute node information to the NnfNodeStorage resource
// so it can make the NVMe namespaces accessible on the compute node. This is done on the rabbit
// by creating StorageGroup resources through swordfish for the correct endpoint.
func (r *NnfAccessReconciler) addBlockStorageAccess(ctx context.Context, access *nnfv1alpha8.NnfAccess, storageMapping map[string][]dwsv1alpha5.ClientMountInfo) error {
	// NnfNodeStorage clientReferences only need to be added for compute nodes. If
	// this nnfAccess is not for compute nodes, then there's no work to do.
	if access.Spec.ClientReference == (corev1.ObjectReference{}) {
		return nil
	}

	nodeStorageMap := make(map[corev1.ObjectReference][]mountReference)

	// Make a map of NnfNodeStorage references that holds a list of which compute nodes
	// access which allocation index.
	for client, storageList := range storageMapping {
		for _, mount := range storageList {
			if mount.Device.DeviceReference == nil {
				continue
			}

			if mount.Device.DeviceReference.ObjectReference.Kind != reflect.TypeOf(nnfv1alpha8.NnfNodeStorage{}).Name() {
				continue
			}

			mountRef := mountReference{
				client:          client,
				allocationIndex: mount.Device.DeviceReference.Data,
			}

			nodeStorageMap[mount.Device.DeviceReference.ObjectReference] = append(nodeStorageMap[mount.Device.DeviceReference.ObjectReference], mountRef)
		}
	}

	// Loop through the NnfNodeBlockStorages and add client access information for each of the
	// computes that need access to an allocation.
	for nodeStorageReference, mountRefList := range nodeStorageMap {
		nnfNodeStorage := &nnfv1alpha8.NnfNodeStorage{
			ObjectMeta: metav1.ObjectMeta{
				Name:      nodeStorageReference.Name,
				Namespace: nodeStorageReference.Namespace,
			},
		}

		if err := r.Get(ctx, client.ObjectKeyFromObject(nnfNodeStorage), nnfNodeStorage); err != nil {
			return err
		}

		nnfNodeBlockStorage := &nnfv1alpha8.NnfNodeBlockStorage{
			ObjectMeta: metav1.ObjectMeta{
				Name:      nnfNodeStorage.Spec.BlockReference.Name,
				Namespace: nnfNodeStorage.Spec.BlockReference.Namespace,
			},
		}

		if err := r.Get(ctx, client.ObjectKeyFromObject(nnfNodeBlockStorage), nnfNodeBlockStorage); err != nil {
			return err
		}

		oldNnfNodeBlockStorage := *nnfNodeBlockStorage.DeepCopy()
		// The clientEndpoints field is an array of each of the allocations on the Rabbit
		// node that holds a list of the endpoints to expose the allocation to. The endpoints
		// are the swordfish endpoints, so 0 is the rabbit, and 1-16 are the computes. Start out
		// by clearing all compute node endpoints from the allocations.
		for i := range nnfNodeBlockStorage.Spec.Allocations {
			nnfNodeBlockStorage.Spec.Allocations[i].Access = []string{nnfNodeBlockStorage.Namespace}
		}

		// Add compute node endpoints for each of the allocations. Increment the compute node
		// index found from the "storage" resource to account for the 0 index being the rabbit
		// in swordfish.
		for _, mountRef := range mountRefList {
			allocationIndex := mountRef.allocationIndex
			if nnfNodeBlockStorage.Spec.SharedAllocation {
				allocationIndex = 0
			}
			// Add the client name to the access list if it's not already there
			if slices.IndexFunc(nnfNodeBlockStorage.Spec.Allocations[allocationIndex].Access, func(n string) bool { return n == mountRef.client }) < 0 {
				nnfNodeBlockStorage.Spec.Allocations[allocationIndex].Access = append(nnfNodeBlockStorage.Spec.Allocations[allocationIndex].Access, mountRef.client)
			}
		}

		if reflect.DeepEqual(oldNnfNodeBlockStorage, *nnfNodeBlockStorage) {
			continue
		}

		if err := r.Update(ctx, nnfNodeBlockStorage); err != nil {
			return err
		}
	}

	return nil
}

func (r *NnfAccessReconciler) getBlockStorageAccessStatus(ctx context.Context, access *nnfv1alpha8.NnfAccess, storageMapping map[string][]dwsv1alpha5.ClientMountInfo) (bool, error) {
	// NnfNodeStorage clientReferences only need to be checked for compute nodes. If
	// this nnfAccess is not for compute nodes, then there's no work to do.
	if access.Spec.ClientReference == (corev1.ObjectReference{}) {
		return true, nil
	}

	nodeStorageMap := make(map[corev1.ObjectReference][]mountReference)

	// Make a map of NnfNodeStorage references that were mounted by this
	// nnfAccess
	for client, storageList := range storageMapping {
		for _, mount := range storageList {
			if mount.Device.DeviceReference == nil {
				continue
			}

			if mount.Device.DeviceReference.ObjectReference.Kind != reflect.TypeOf(nnfv1alpha8.NnfNodeStorage{}).Name() {
				continue
			}

			mountRef := mountReference{
				client:          client,
				allocationIndex: mount.Device.DeviceReference.Data,
			}

			nodeStorageMap[mount.Device.DeviceReference.ObjectReference] = append(nodeStorageMap[mount.Device.DeviceReference.ObjectReference], mountRef)

		}
	}

	nnfNodeBlockStorages := []nnfv1alpha8.NnfNodeBlockStorage{}

	for nodeStorageReference := range nodeStorageMap {
		nnfNodeStorage := &nnfv1alpha8.NnfNodeStorage{
			ObjectMeta: metav1.ObjectMeta{
				Name:      nodeStorageReference.Name,
				Namespace: nodeStorageReference.Namespace,
			},
		}

		if err := r.Get(ctx, client.ObjectKeyFromObject(nnfNodeStorage), nnfNodeStorage); err != nil {
			return false, err
		}

		nnfNodeBlockStorage := &nnfv1alpha8.NnfNodeBlockStorage{
			ObjectMeta: metav1.ObjectMeta{
				Name:      nnfNodeStorage.Spec.BlockReference.Name,
				Namespace: nnfNodeStorage.Spec.BlockReference.Namespace,
			},
		}

		if err := r.Get(ctx, client.ObjectKeyFromObject(nnfNodeBlockStorage), nnfNodeBlockStorage); err != nil {
			return false, err
		}

		nnfNodeBlockStorages = append(nnfNodeBlockStorages, *nnfNodeBlockStorage)
	}

	for _, nnfNodeBlockStorage := range nnfNodeBlockStorages {
		if nnfNodeBlockStorage.Status.Error != nil {
			return false, dwsv1alpha5.NewResourceError("Node: %s", nnfNodeBlockStorage.GetNamespace()).WithError(nnfNodeBlockStorage.Status.Error)
		}
	}

	for _, nnfNodeBlockStorage := range nnfNodeBlockStorages {
		storage := &dwsv1alpha5.Storage{
			ObjectMeta: metav1.ObjectMeta{
				Name:      nnfNodeBlockStorage.GetNamespace(),
				Namespace: corev1.NamespaceDefault,
			},
		}

		// If we're ignoring offline Computes, then get the Storage resource once here so it can be
		// used later
		if access.Spec.IgnoreOfflineComputes {
			if err := r.Get(ctx, client.ObjectKeyFromObject(storage), storage); err != nil {
				return false, err
			}
		}

		for allocationIndex, allocation := range nnfNodeBlockStorage.Spec.Allocations {
			// Check that all of the StorageGroups we've requested have been created
			for _, nodeName := range allocation.Access {
				blockAccess, exists := nnfNodeBlockStorage.Status.Allocations[allocationIndex].Accesses[nodeName]
				if access.Spec.IgnoreOfflineComputes {
					computeOffline := false
					for _, compute := range storage.Status.Access.Computes {
						if compute.Name != nodeName {
							continue
						}

						if compute.Status == dwsv1alpha5.OfflineStatus {
							computeOffline = true
							break
						}
					}

					// If the compute is offline, don't check its status
					if computeOffline {
						continue
					}
				}

				// if the map entry doesn't exist in the status section for this node yet, then keep waiting
				if !exists {
					return false, nil
				}

				// Check that the storage group has been created
				if blockAccess.StorageGroupId == "" {
					return false, nil
				}
			}

			// Check that there aren't any extra StorageGroups present
			for statusNodeName := range nnfNodeBlockStorage.Status.Allocations[allocationIndex].Accesses {
				found := false
				for _, specNodeName := range allocation.Access {
					if specNodeName == statusNodeName {
						found = true
						break
					}
				}

				if found {
					continue
				}

				if access.Spec.IgnoreOfflineComputes {
					computeOffline := false
					for _, compute := range storage.Status.Access.Computes {
						if compute.Name != statusNodeName {
							continue
						}

						if compute.Status == dwsv1alpha5.OfflineStatus {
							computeOffline = true
							break
						}
					}

					// If the compute is offline, don't check its status
					if computeOffline {
						continue
					}
				}

				return false, nil
			}
		}
	}

	return true, nil
}

// removeNodeStorageEndpoints modifies the NnfNodeStorage resources to remove the client endpoints for the
// compute nodes that had mounted the storage. This causes NnfNodeStorage to remove the StorageGroups for
// those compute nodes and remove access to the NVMe namespaces from the computes.
func (r *NnfAccessReconciler) removeBlockStorageAccess(ctx context.Context, access *nnfv1alpha8.NnfAccess, storageMapping map[string][]dwsv1alpha5.ClientMountInfo) error {
	// NnfNodeStorage clientReferences only need to be removed for compute nodes. If
	// this nnfAccess is not for compute nodes, then there's no work to do.
	if access.Spec.ClientReference == (corev1.ObjectReference{}) {
		return nil
	}

	nodeBlockStorageMap := make(map[corev1.ObjectReference]bool)

	// Make a map of NnfNodeStorage references that were mounted by this
	// nnfAccess
	for _, storageList := range storageMapping {
		for _, mount := range storageList {
			if mount.Device.DeviceReference == nil {
				continue
			}

			if mount.Device.DeviceReference.ObjectReference.Kind != reflect.TypeOf(nnfv1alpha8.NnfNodeStorage{}).Name() {
				continue
			}

			nodeBlockStorageMap[mount.Device.DeviceReference.ObjectReference] = true
		}
	}

	// Update each of the NnfNodeBlockStorage resources to remove the access that
	// was added earlier. Leave the first entry since that corresponds to the
	// rabbit node.
	for nodeBlockStorageReference := range nodeBlockStorageMap {
		namespacedName := types.NamespacedName{
			Name:      nodeBlockStorageReference.Name,
			Namespace: nodeBlockStorageReference.Namespace,
		}

		nnfNodeBlockStorage := &nnfv1alpha8.NnfNodeBlockStorage{}
		err := r.Get(ctx, namespacedName, nnfNodeBlockStorage)
		if err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			return err
		}

		oldNnfNodeBlockStorage := *nnfNodeBlockStorage.DeepCopy()

		for i := range nnfNodeBlockStorage.Spec.Allocations {
			nnfNodeBlockStorage.Spec.Allocations[i].Access = nnfNodeBlockStorage.Spec.Allocations[i].Access[:1]
		}
		if reflect.DeepEqual(oldNnfNodeBlockStorage, *nnfNodeBlockStorage) {
			continue
		}

		err = r.Update(ctx, nnfNodeBlockStorage)
		if err != nil {
			return err
		}
	}

	return nil
}

// manageClientMounts creates or updates the ClientMount resources based on the information in the storageMapping map.
func (r *NnfAccessReconciler) manageClientMounts(ctx context.Context, access *nnfv1alpha8.NnfAccess, storageMapping map[string][]dwsv1alpha5.ClientMountInfo) error {
	log := r.Log.WithValues("NnfAccess", client.ObjectKeyFromObject(access))

	if !access.Spec.MakeClientMounts {
		return nil
	}

	if access.Spec.StorageReference.Kind != reflect.TypeOf(nnfv1alpha8.NnfStorage{}).Name() {
		return dwsv1alpha5.NewResourceError("invalid StorageReference kind %s", access.Spec.StorageReference.Kind).WithFatal()
	}

	nnfStorage := &nnfv1alpha8.NnfStorage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      access.Spec.StorageReference.Name,
			Namespace: access.Spec.StorageReference.Namespace,
		},
	}

	if err := r.Get(ctx, client.ObjectKeyFromObject(nnfStorage), nnfStorage); err != nil {
		return dwsv1alpha5.NewResourceError("could not get NnfStorage %v", client.ObjectKeyFromObject(nnfStorage)).WithError(err).WithMajor()
	}

	// The targetIndex is the directive index of the NnfStorage. This is needed in clientmountd because
	// some blockdevice/filesystem names (e.g., lvm volume group) are generated with the directive index.
	// The directive index of the NnfAccess/ClientMount might be different from the NnfStorage during
	// data movement mounts, persistent storage mounts, and user container mounts.
	targetIndex := getDirectiveIndexLabel(nnfStorage)

	g := new(errgroup.Group)

	for clientName, storageList := range storageMapping {
		clientName := clientName
		storageList := storageList

		// Start a goroutine for each ClientMount to create
		g.Go(func() error {
			clientMount := &dwsv1alpha5.ClientMount{
				ObjectMeta: metav1.ObjectMeta{
					Name:      clientMountName(access),
					Namespace: clientName,
				},
			}
			result, err := ctrl.CreateOrUpdate(ctx, r.Client, clientMount,
				func() error {
					dwsv1alpha5.InheritParentLabels(clientMount, access)
					dwsv1alpha5.AddOwnerLabels(clientMount, access)
					setTargetDirectiveIndexLabel(clientMount, targetIndex)
					setTargetOwnerUIDLabel(clientMount, string(nnfStorage.GetUID()))

					clientMount.Spec.Node = clientName
					clientMount.Spec.DesiredState = dwsv1alpha5.ClientMountState(access.Spec.DesiredState)
					clientMount.Spec.Mounts = storageList

					return nil
				})

			namespacedName := client.ObjectKeyFromObject(clientMount).String()
			if err != nil {
				if !apierrors.IsConflict(err) {
					log.Error(err, "failed to create or update ClientMount", "name", namespacedName)
				}

				return err
			}
			if result == controllerutil.OperationResultCreated {
				log.Info("Created ClientMount", "name", namespacedName)
			} else if result == controllerutil.OperationResultNone {
				// no change
			} else {
				log.Info("Updated ClientMount", "name", namespacedName)
			}
			return err
		})
	}

	// Wait for the goroutines to finish and return the first error
	return g.Wait()
}

// getClientMountStatus aggregates the status from all the ClientMount resources
func (r *NnfAccessReconciler) getClientMountStatus(ctx context.Context, access *nnfv1alpha8.NnfAccess, clientList []string) (bool, error) {
	log := r.Log.WithValues("NnfAccess", client.ObjectKeyFromObject(access))

	if !access.Spec.MakeClientMounts {
		return true, nil
	}

	clientMountList := &dwsv1alpha5.ClientMountList{}
	matchLabels := dwsv1alpha5.MatchingOwner(access)

	listOptions := []client.ListOption{
		matchLabels,
	}

	if err := r.List(ctx, clientMountList, listOptions...); err != nil {
		return false, dwsv1alpha5.NewResourceError("could not list ClientMounts").WithError(err)
	}

	// make a map with empty data of the client names to allow easy searching
	clientNameMap := map[string]struct{}{}
	for _, clientName := range clientList {
		clientNameMap[clientName] = struct{}{}
	}

	clientMounts := []dwsv1alpha5.ClientMount{}
	for _, clientMount := range clientMountList.Items {
		if _, exists := clientNameMap[clientMount.GetNamespace()]; exists {
			clientMounts = append(clientMounts, clientMount)
		}
	}

	// Check the clientmounts for any errors first
	for _, clientMount := range clientMounts {
		if clientMount.Status.Error != nil {
			return false, dwsv1alpha5.NewResourceError("Node: %s", clientMount.GetNamespace()).WithError(clientMount.Status.Error)
		}
	}

	childTimeoutString := os.Getenv("NNF_CHILD_RESOURCE_TIMEOUT_SECONDS")
	if len(childTimeoutString) > 0 {
		childTimeout, err := strconv.Atoi(childTimeoutString)
		if err != nil {
			log.Info("Error: Invalid NNF_CHILD_RESOURCE_TIMEOUT_SECONDS. Defaulting to 300 seconds", "value", childTimeoutString)
			childTimeout = 300
		}

		for _, clientMount := range clientMounts {
			// check if the finalizer has been added by the controller on the Rabbit
			if len(clientMount.GetFinalizers()) > 0 {
				continue
			}

			if clientMount.GetCreationTimestamp().Add(time.Duration(time.Duration(childTimeout) * time.Second)).Before(time.Now()) {
				return false, dwsv1alpha5.NewResourceError("Node: %s: ClientMount has not been reconciled after %d seconds", clientMount.GetNamespace(), childTimeout).WithMajor()
			}

			return false, nil
		}
	}

	// Check whether the clientmounts have finished mounting/unmounting
	for _, clientMount := range clientMounts {
		if len(clientMount.Status.Mounts) != len(clientMount.Spec.Mounts) {
			return false, nil
		}

		for _, mount := range clientMount.Status.Mounts {
			if string(mount.State) != access.Status.State || !mount.Ready {
				if string(mount.State) == "unmounted" {
					offline, err := r.checkOfflineCompute(ctx, access, &clientMount)
					if err != nil {
						return false, err
					}
					// If the compute node is down, then ignore an unmount failure. If the compute node
					// comes back up, the file system won't be mounted again since spec.desiredState is "unmounted"
					if offline {
						log.Info("ignoring status from offline compute node", "node name", clientMount.GetNamespace())
						continue
					}
				}

				return false, nil
			}
		}
	}

	if len(clientMounts) != len(clientList) {
		if access.GetDeletionTimestamp().IsZero() {
			log.Info("unexpected number of ClientMounts", "found", len(clientMounts), "expected", len(clientList))
		}
		return false, nil
	}
	return true, nil
}

func clientMountName(access *nnfv1alpha8.NnfAccess) string {
	return access.Namespace + "-" + access.Name
}

// removeOfflineClientMounts deletes the NnfClientMount finalizer from any ClientMounts that
// have a Compute node marked as "Offline" in the Storage resource
func (r *NnfAccessReconciler) removeOfflineClientMounts(ctx context.Context, nnfAccess *nnfv1alpha8.NnfAccess) error {
	log := r.Log.WithValues("NnfAccess", client.ObjectKeyFromObject(nnfAccess))

	if !nnfAccess.Spec.MakeClientMounts {
		return nil
	}

	// Optional timeout before starting to remove the finalizers
	deletionTimeoutString := os.Getenv("NNF_CHILD_DELETION_TIMEOUT_SECONDS")
	if len(deletionTimeoutString) > 0 {
		childTimeout, err := strconv.Atoi(deletionTimeoutString)
		if err != nil {
			log.Info("Error: Invalid NNF_CHILD_DELETION_TIMEOUT_SECONDS. Defaulting to 300 seconds", "value", deletionTimeoutString)
			childTimeout = 300
		}

		// Don't check for offline computes until after the timeout has elapsed
		if nnfAccess.GetDeletionTimestamp().Add(time.Duration(time.Duration(childTimeout) * time.Second)).After(time.Now()) {
			return nil
		}
	}

	// Find all ClientMounts owned by the NnfAccess
	clientMountList := &dwsv1alpha5.ClientMountList{}
	matchLabels := dwsv1alpha5.MatchingOwner(nnfAccess)

	listOptions := []client.ListOption{
		matchLabels,
	}

	if err := r.List(ctx, clientMountList, listOptions...); err != nil {
		return dwsv1alpha5.NewResourceError("could not list ClientMounts").WithError(err)
	}

	// For each ClientMount, check whether the Compute node is marked as "Offline". If it is,
	// remove the NnfClientMount finalizer and update the resource
	for _, clientMount := range clientMountList.Items {
		offline, err := r.checkOfflineCompute(ctx, nnfAccess, &clientMount)
		if err != nil {
			return err
		}

		if !offline {
			continue
		}

		log.Info("removing finalizer from offline compute node", "node name", clientMount.GetNamespace())

		// Remove the finalizer from the ClientMount since the compute node is offline
		controllerutil.RemoveFinalizer(&clientMount, finalizerNnfClientMount)
		if err := r.Update(ctx, &clientMount); err != nil {
			if !apierrors.IsConflict(err) {
				return dwsv1alpha5.NewResourceError("could not update ClientMount to remove finalizer for offline compute: %v", client.ObjectKeyFromObject(&clientMount)).WithError(err)
			}
		}
	}

	return nil
}

// checkOfflineCompute finds the Storage resource for the Rabbit physically attached to the clientMount's
// compute node, and checks whether the compute node is marked as "Offline" (indicating no PCIe connection).
// It also checks the SystemStatus resource to see if the compute node is marked as Disabled.
func (r *NnfAccessReconciler) checkOfflineCompute(ctx context.Context, nnfAccess *nnfv1alpha8.NnfAccess, clientMount *dwsv1alpha5.ClientMount) (bool, error) {
	if nnfAccess.Spec.ClientReference == (corev1.ObjectReference{}) {
		return false, nil
	}

	// Find the name of the Rabbit attached to the compute node
	rabbitName, err := r.getRabbitFromClientMount(ctx, clientMount)
	if err != nil {
		return false, err
	}

	// Get the Storage resource for the Rabbit
	storage := &dwsv1alpha5.Storage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rabbitName,
			Namespace: "default",
		},
	}

	if err := r.Get(ctx, client.ObjectKeyFromObject(storage), storage); err != nil {
		return false, dwsv1alpha5.NewResourceError("could not get Storage %v", client.ObjectKeyFromObject(storage)).WithError(err).WithMajor()
	}

	// Check the status of the compute node in the Storage resource to determine if it's offline
	computeName := clientMount.GetNamespace()
	for _, compute := range storage.Status.Access.Computes {
		if compute.Name == computeName {
			if compute.Status == dwsv1alpha5.OfflineStatus {
				return true, nil
			}
			break
		}
	}

	systemStatus := &dwsv1alpha5.SystemStatus{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "default",
			Namespace: corev1.NamespaceDefault,
		},
	}

	if err := r.Get(ctx, client.ObjectKeyFromObject(systemStatus), systemStatus); err != nil {
		// Don't rely on the SystemStatus existing. If it's not there, just return that the compute node
		// isn't offline
		return false, nil
	}

	if status, found := systemStatus.Data.Nodes[computeName]; found {
		if status == dwsv1alpha5.SystemNodeStatusDisabled {
			return true, nil
		}
	}

	return false, nil
}

// getRabbitFromClientMount finds the name of the Rabbit that is physically connected to the ClientMount's
// compute node
func (r *NnfAccessReconciler) getRabbitFromClientMount(ctx context.Context, clientMount *dwsv1alpha5.ClientMount) (string, error) {
	// If the ClientMount has a block device reference that points to an NnNodeStorage, use the namespace of the NnfNodeStorage
	// as the Rabbit name
	if clientMount.Spec.Mounts[0].Device.DeviceReference.ObjectReference.Kind == reflect.TypeOf(nnfv1alpha8.NnfNodeStorage{}).Name() {
		return clientMount.Spec.Mounts[0].Device.DeviceReference.ObjectReference.Namespace, nil
	}

	return helpers.GetRabbitFromCompute(ctx, r.Client, clientMount.GetNamespace())
}

// For rabbit mounts, use unique index mount directories that consist of <namespace>-<index>.  These
// unique directories guard against potential data loss when doing copy out data movement
// operations. Having the namespace (rabbit name) included in the mount path ensures that these
// individual compute mount points are unique.
//
// Ex:
//
//	/mnt/nnf/7b5dda61-9d91-4b50-a0d3-f863d0aac25b-0/rabbit-node-1-0
//	/mnt/nnf/7b5dda61-9d91-4b50-a0d3-f863d0aac25b-0/rabbit-node-2-0
//
// If you did not include the namespace, then the paths would be:
//
//	/mnt/nnf/7b5dda61-9d91-4b50-a0d3-f863d0aac25b-0/0
//	/mnt/nnf/7b5dda61-9d91-4b50-a0d3-f863d0aac25b-0/0
//
// When data movement copies the MountPathPrefix to global lustre, then the contents of these
// directories are merged together and data loss can occur if the contents do not have
// unique filenames.
func getIndexMountDir(namespace string, index int) string {
	return fmt.Sprintf("%s-%s", namespace, strconv.Itoa(index))
}

// ComputesEnqueueRequests triggers on a Computes resource. It finds any NnfAccess resources with the
// same owner as the Computes resource and adds them to the Request list.
func (r *NnfAccessReconciler) ComputesEnqueueRequests(ctx context.Context, o client.Object) []reconcile.Request {
	log := r.Log.WithValues("Computes", "Enqueue")

	requests := []reconcile.Request{}

	// Ensure the storage resource is updated with the latest NNF Node resource status
	computes := &dwsv1alpha5.Computes{
		ObjectMeta: metav1.ObjectMeta{
			Name:      o.GetName(),
			Namespace: o.GetNamespace(),
		},
	}

	if err := r.Get(ctx, client.ObjectKeyFromObject(computes), computes); err != nil {
		return requests
	}

	labels := computes.GetLabels()
	if labels == nil {
		return []reconcile.Request{}
	}

	ownerName, exists := labels[dwsv1alpha5.OwnerNameLabel]
	if !exists {
		return []reconcile.Request{}
	}

	ownerNamespace, exists := labels[dwsv1alpha5.OwnerNamespaceLabel]
	if !exists {
		return []reconcile.Request{}
	}

	ownerKind, exists := labels[dwsv1alpha5.OwnerKindLabel]
	if !exists {
		return []reconcile.Request{}
	}

	// Find all the NnfAccess resource with the same owner as the Computes resource
	listOptions := []client.ListOption{
		client.MatchingLabels(map[string]string{
			dwsv1alpha5.OwnerKindLabel:      ownerKind,
			dwsv1alpha5.OwnerNameLabel:      ownerName,
			dwsv1alpha5.OwnerNamespaceLabel: ownerNamespace,
		}),
	}

	nnfAccessList := &nnfv1alpha8.NnfAccessList{}
	if err := r.List(context.TODO(), nnfAccessList, listOptions...); err != nil {
		log.Info("Could not list NnfAccesses", "error", err)
		return requests
	}

	for _, nnfAccess := range nnfAccessList.Items {
		requests = append(requests, reconcile.Request{NamespacedName: types.NamespacedName{Name: nnfAccess.GetName(), Namespace: nnfAccess.GetNamespace()}})
	}

	return requests
}

func (r *NnfAccessReconciler) getChildObjects() []dwsv1alpha5.ObjectList {
	return []dwsv1alpha5.ObjectList{
		&dwsv1alpha5.ClientMountList{},
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *NnfAccessReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// NOTE: NNF Access controller also depends on NNF Storage and NNF Node Storage status'
	// as part of its reconcile sequence. But since there is not a very good way to translate
	// from these resources to the associated NNF Access resource as one would typically do
	// in an EqueueRequestsFromMapFunc(), the Reconciler instead requeues until the necessary
	// resource state is observed.
	//
	// For NNF Storage updates, a job DW maps well to the two NNF Access
	//     i.e. o.GetName() + "-computes"     and o.GetName() + "-servers"
	// But for a persistent DW there is no good translation.
	//
	// For NNF Node Storage updates, a job DW is pretty straight forward using ownership
	// labels to get the parent NNF Storage. But for a persistent DW it has the same problem
	// as NNF Storage.
	//
	// Matt or Tony might be able to clean this up.

	maxReconciles := runtime.GOMAXPROCS(0)
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(controller.Options{MaxConcurrentReconciles: maxReconciles}).
		For(&nnfv1alpha8.NnfAccess{}).
		Watches(&dwsv1alpha5.Computes{}, handler.EnqueueRequestsFromMapFunc(r.ComputesEnqueueRequests)).
		Watches(&dwsv1alpha5.ClientMount{}, handler.EnqueueRequestsFromMapFunc(dwsv1alpha5.OwnerLabelMapFunc)).
		Complete(r)
}
