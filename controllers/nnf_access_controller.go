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
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"

	"github.com/go-logr/logr"
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
	"sigs.k8s.io/controller-runtime/pkg/source"

	dwsv1alpha1 "github.com/HewlettPackard/dws/api/v1alpha1"
	"github.com/HewlettPackard/dws/utils/updater"
	nnfv1alpha1 "github.com/NearNodeFlash/nnf-sos/api/v1alpha1"
	"github.com/NearNodeFlash/nnf-sos/controllers/metrics"
)

// NnfAccessReconciler reconciles a NnfAccess object
type NnfAccessReconciler struct {
	client.Client
	Log          logr.Logger
	Scheme       *kruntime.Scheme
	ChildObjects []dwsv1alpha1.ObjectList
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
//+kubebuilder:rbac:groups=dws.cray.hpe.com,resources=clientmounts,verbs=get;list;watch;create;update;patch;delete;deletecollection
//+kubebuilder:rbac:groups=dws.cray.hpe.com,resources=clientmounts/status,verbs=get;update;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *NnfAccessReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, err error) {
	log := r.Log.WithValues("NnfAccess", req.NamespacedName)

	metrics.NnfAccessReconcilesTotal.Inc()

	access := &nnfv1alpha1.NnfAccess{}
	if err := r.Get(ctx, req.NamespacedName, access); err != nil {
		// ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	statusUpdater := updater.NewStatusUpdater[*nnfv1alpha1.NnfAccessStatus](access)
	defer func() { err = statusUpdater.CloseWithStatusUpdate(ctx, r, err) }()

	// Create a list of names of the client nodes. This is pulled from either
	// the Computes resource specified in the ClientReference or the NnfStorage
	// resource when no ClientReference is provided. These correspond to mounting
	// the compute nodes during pre_run and mounting the rabbit nodes for data
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

		deleteStatus, err := dwsv1alpha1.DeleteChildren(ctx, r.Client, r.ChildObjects, access)
		if err != nil {
			return ctrl.Result{}, err
		}

		if deleteStatus == dwsv1alpha1.DeleteRetry {
			return ctrl.Result{}, nil
		}

		err = r.removeNodeStorageEndpoints(ctx, access, storageMapping)
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

	// Lock the NnfStorage by adding an annotation with the name/namespace for this
	// NnfAccess. This is used for non-clustered file systems that can only be mounted
	// from a single host.
	wait, err := r.lockStorage(ctx, access)
	if err != nil {
		return ctrl.Result{}, err
	}

	if wait {
		return ctrl.Result{}, nil
	}

	// Add compute node information to the storage map, if necessary.
	err = r.addNodeStorageEndpoints(ctx, access, storageMapping)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Create the ClientMount resources. One ClientMount resource is created per client
	err = r.createClientMounts(ctx, access, storageMapping, log)
	if err != nil {
		return ctrl.Result{}, err
	}

	ready, err := r.getNodeStorageEndpointStatus(ctx, access, storageMapping)
	if err != nil {
		return ctrl.Result{}, err
	}

	if ready == false {
		return ctrl.Result{}, nil
	}

	// Aggregate the status from all the ClientMount resources
	ready, err = r.getClientMountStatus(ctx, access, clientList)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Wait for all of the ClientMounts to be ready before setting the Ready field
	if ready == false {
		return ctrl.Result{}, nil
	}

	if access.Status.Ready == false {
		log.Info("State achieved", "State", access.Status.State)
	}

	access.Status.Ready = true
	access.Status.Error = nil

	return ctrl.Result{}, nil
}

// lockStorage applies an annotation to the NnfStorage resource with the name and namespace of the NnfAccess resource.
// This acts as a lock to prevent multiple NnfAccess resources from mounting the same file system. This is only necessary
// for non-clustered file systems
func (r *NnfAccessReconciler) lockStorage(ctx context.Context, access *nnfv1alpha1.NnfAccess) (bool, error) {

	if access.Spec.StorageReference.Kind != reflect.TypeOf(nnfv1alpha1.NnfStorage{}).Name() {
		return false, fmt.Errorf("Invalid StorageReference kind %s", access.Spec.StorageReference.Kind)
	}

	namespacedName := types.NamespacedName{
		Name:      access.Spec.StorageReference.Name,
		Namespace: access.Spec.StorageReference.Namespace,
	}

	nnfStorage := &nnfv1alpha1.NnfStorage{}
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

	err := r.Update(ctx, nnfStorage)
	if err != nil {
		return false, err
	}

	return false, nil
}

// unlockStorage removes the NnfAccess annotation from an NnfStorage resource if it was added from lockStorage()
func (r *NnfAccessReconciler) unlockStorage(ctx context.Context, access *nnfv1alpha1.NnfAccess) error {
	nnfStorage := &nnfv1alpha1.NnfStorage{}

	if access.Spec.StorageReference.Kind != reflect.TypeOf(nnfv1alpha1.NnfStorage{}).Name() {
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
		_, exists := annotations[NnfAccessAnnotation]
		if !exists {
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
func (r *NnfAccessReconciler) getClientList(ctx context.Context, access *nnfv1alpha1.NnfAccess) ([]string, error) {
	if access.Spec.ClientReference != (corev1.ObjectReference{}) {
		return r.getClientListFromClientReference(ctx, access)
	}

	return r.getClientListFromStorageReference(ctx, access)
}

// getClientListFromClientReference returns a list of client nodes names from the Computes resource
func (r *NnfAccessReconciler) getClientListFromClientReference(ctx context.Context, access *nnfv1alpha1.NnfAccess) ([]string, error) {
	computes := &dwsv1alpha1.Computes{}

	if access.Spec.ClientReference.Kind != reflect.TypeOf(dwsv1alpha1.Computes{}).Name() {
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
func (r *NnfAccessReconciler) getClientListFromStorageReference(ctx context.Context, access *nnfv1alpha1.NnfAccess) ([]string, error) {

	if access.Spec.StorageReference.Kind != reflect.TypeOf(nnfv1alpha1.NnfStorage{}).Name() {
		return nil, fmt.Errorf("Invalid StorageReference kind %s", access.Spec.StorageReference.Kind)
	}

	namespacedName := types.NamespacedName{
		Name:      access.Spec.StorageReference.Name,
		Namespace: access.Spec.StorageReference.Namespace,
	}

	nnfStorage := &nnfv1alpha1.NnfStorage{}
	if err := r.Get(ctx, namespacedName, nnfStorage); err != nil {
		return nil, err
	}

	clients := []string{}
	for _, allocationSetSpec := range nnfStorage.Spec.AllocationSets {
		if nnfStorage.Spec.FileSystemType == "lustre" {
			if allocationSetSpec.NnfStorageLustreSpec.TargetType != "OST" {
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
func (r *NnfAccessReconciler) mapClientStorage(ctx context.Context, access *nnfv1alpha1.NnfAccess, clients []string) (map[string][]dwsv1alpha1.ClientMountInfo, error) {
	nnfStorage := &nnfv1alpha1.NnfStorage{}

	if access.Spec.StorageReference.Kind != reflect.TypeOf(nnfv1alpha1.NnfStorage{}).Name() {
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
		var storageMapping map[string][]dwsv1alpha1.ClientMountInfo
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
func (r *NnfAccessReconciler) mapClientNetworkStorage(ctx context.Context, access *nnfv1alpha1.NnfAccess, clients []string, nnfStorage *nnfv1alpha1.NnfStorage, setIndex int) (map[string][]dwsv1alpha1.ClientMountInfo, error) {
	allocationSet := nnfStorage.Spec.AllocationSets[setIndex]

	if allocationSet.ExternalMgsNid == "" && allocationSet.TargetType != "MGT" && allocationSet.TargetType != "MGTMDT" {
		// Look elsewhere for the MGS NID.
		return nil, nil
	}

	storageMapping := make(map[string][]dwsv1alpha1.ClientMountInfo)

	for _, client := range clients {
		mountInfo := dwsv1alpha1.ClientMountInfo{}
		mountInfo.Type = nnfStorage.Spec.FileSystemType
		mountInfo.TargetType = "directory"
		mountInfo.MountPath = access.Spec.MountPath
		mountInfo.Device.Type = dwsv1alpha1.ClientMountDeviceTypeLustre
		mountInfo.Device.Lustre = &dwsv1alpha1.ClientMountDeviceLustre{}
		mountInfo.Device.Lustre.FileSystemName = allocationSet.FileSystemName
		if allocationSet.ExternalMgsNid != "" {
			mountInfo.Device.Lustre.MgsAddresses = allocationSet.ExternalMgsNid
		} else {
			mountInfo.Device.Lustre.MgsAddresses = nnfStorage.Status.MgsNode
		}

		// Make it easy for the nnf-dm daemon to find the NnfStorage.
		mountInfo.Device.DeviceReference = &dwsv1alpha1.ClientMountDeviceReference{
			ObjectReference: access.Spec.StorageReference,
		}

		storageMapping[client] = append(storageMapping[client], mountInfo)
	}

	return storageMapping, nil
}

// mapClientLocalStorage picks storage device(s) for each client to access based on locality information
// from the (DWS) Storage resources.
func (r *NnfAccessReconciler) mapClientLocalStorage(ctx context.Context, access *nnfv1alpha1.NnfAccess, clients []string, nnfStorage *nnfv1alpha1.NnfStorage, setIndex int) (map[string][]dwsv1alpha1.ClientMountInfo, error) {
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
	existingStorage := make(map[string][]dwsv1alpha1.ClientMountInfo)

	// Read each NnfNodeStorage resource and find the NVMe information for each
	// allocation.
	for nodeName, storageCount := range storageCountMap {
		matchLabels := dwsv1alpha1.MatchingOwner(nnfStorage)
		matchLabels[nnfv1alpha1.AllocationSetLabel] = allocationSetSpec.Name

		listOptions := []client.ListOption{
			matchLabels,
			client.InNamespace(nodeName),
		}

		nnfNodeStorageList := &nnfv1alpha1.NnfNodeStorageList{}
		if err := r.List(ctx, nnfNodeStorageList, listOptions...); err != nil {
			return nil, err
		}

		// Check that the correct number of NnfNodeStorage resources were found for this
		// Rabbit.
		if len(nnfNodeStorageList.Items) != storageCount.instanceCount {
			return nil, fmt.Errorf("Incorrect number of NnfNodeStorages. found %d. Needed %d.", len(nnfNodeStorageList.Items), storageCount.instanceCount)
		}

		for _, nnfNodeStorage := range nnfNodeStorageList.Items {
			// Loop through each allocation to pull out the device information and build the
			// mount information
			for i := 0; i < nnfNodeStorage.Spec.Count; i++ {
				mountInfo := dwsv1alpha1.ClientMountInfo{}

				// Set the DeviceReference to the NnfNodeStorage allocation regardless of whether we're mounting on
				// the Rabbit or the compute node. The compute node ClientMount device type will not be set to "reference",
				// so clientmountd will not look at the DeviceReference struct. The DeviceReference information is used by
				// the data movement code to match up mounts between the Rabbit and compute node.
				mountInfo.Device.DeviceReference = &dwsv1alpha1.ClientMountDeviceReference{}
				mountInfo.Device.DeviceReference.ObjectReference.Kind = reflect.TypeOf(nnfv1alpha1.NnfNodeStorage{}).Name()
				mountInfo.Device.DeviceReference.ObjectReference.Name = nnfNodeStorage.Name
				mountInfo.Device.DeviceReference.ObjectReference.Namespace = nnfNodeStorage.Namespace
				mountInfo.Device.DeviceReference.Data = i

				if nnfStorage.Spec.FileSystemType == "raw" {
					mountInfo.Type = "none"
					mountInfo.TargetType = "file"
					mountInfo.Options = "bind"
				} else {
					mountInfo.TargetType = "directory"
					mountInfo.Type = nnfStorage.Spec.FileSystemType
				}

				// If no ClientReference exists, then the mounts are for the Rabbit nodes. Use references
				// to the NnfNodeStorage resource so the client mounter can access the swordfish objects
				if access.Spec.ClientReference == (corev1.ObjectReference{}) {
					mountInfo.Device.Type = dwsv1alpha1.ClientMountDeviceTypeReference
					mountInfo.MountPath = filepath.Join(access.Spec.MountPathPrefix, strconv.Itoa(i))
				} else {
					mountInfo.MountPath = access.Spec.MountPath
					mountInfo.Device.Type = dwsv1alpha1.ClientMountDeviceTypeLVM
					mountInfo.Device.LVM = &dwsv1alpha1.ClientMountDeviceLVM{}
					mountInfo.Device.LVM.VolumeGroup = nnfNodeStorage.Status.Allocations[i].VolumeGroup
					mountInfo.Device.LVM.LogicalVolume = nnfNodeStorage.Status.Allocations[i].LogicalVolume
					mountInfo.Device.LVM.DeviceType = dwsv1alpha1.ClientMountLVMDeviceTypeNVMe
					for _, nvme := range nnfNodeStorage.Status.Allocations[i].NVMeList {
						nvmeDesc := dwsv1alpha1.ClientMountNVMeDesc{}
						nvmeDesc.DeviceSerial = nvme.DeviceSerial
						nvmeDesc.NamespaceID = nvme.NamespaceID
						nvmeDesc.NamespaceGUID = nvme.NamespaceGUID
						mountInfo.Device.LVM.NVMeInfo = append(mountInfo.Device.LVM.NVMeInfo, nvmeDesc)
					}
				}

				existingStorage[nnfNodeStorage.Namespace] = append(existingStorage[nnfNodeStorage.Namespace], mountInfo)
			}
		}
	}

	// storageMapping is a map of clients and a list of mounts to perform. It is initialized
	// with an empty list of mounts for each client
	storageMapping := make(map[string][]dwsv1alpha1.ClientMountInfo)
	for _, client := range clients {
		storageMapping[client] = []dwsv1alpha1.ClientMountInfo{}
	}

	// Loop through each Rabbit node in the existingStorage map, and find a client for
	// each of the allocations. This is done by finding the compute and servers list from
	// the Storage resource.
	for storageName := range existingStorage {
		namespacedName := types.NamespacedName{
			Name:      storageName,
			Namespace: "default",
		}

		storage := &dwsv1alpha1.Storage{}
		err := r.Get(ctx, namespacedName, storage)
		if err != nil {
			return nil, err
		}

		// Build a list of all nodes with access to the storage
		clients := []string{}
		for _, compute := range storage.Data.Access.Computes {
			clients = append(clients, compute.Name)
		}

		for _, server := range storage.Data.Access.Servers {
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
				return nil, fmt.Errorf("Invalid matching between clients and storage. Too many clients for storage %s", storageName)
			}

			// If target==all, then the client wants to access all the storage it can see
			if access.Spec.Target == "all" {
				storageMapping[client] = append(storageMapping[client], existingStorage[storageName]...)
				existingStorage[storageName] = []dwsv1alpha1.ClientMountInfo{}
			} else {
				storageMapping[client] = append(storageMapping[client], existingStorage[storageName][0])
				existingStorage[storageName] = existingStorage[storageName][1:]
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
func (r *NnfAccessReconciler) addNodeStorageEndpoints(ctx context.Context, access *nnfv1alpha1.NnfAccess, storageMapping map[string][]dwsv1alpha1.ClientMountInfo) error {
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

			if mount.Device.DeviceReference.ObjectReference.Kind != reflect.TypeOf(nnfv1alpha1.NnfNodeStorage{}).Name() {
				continue
			}

			mountRef := mountReference{
				client:          client,
				allocationIndex: mount.Device.DeviceReference.Data,
			}

			nodeStorageMap[mount.Device.DeviceReference.ObjectReference] = append(nodeStorageMap[mount.Device.DeviceReference.ObjectReference], mountRef)
		}
	}

	// Loop through the NnfNodeStorages and add clientEndpoint information for each of the
	// computes that need access to an allocation.
	for nodeStorageReference, mountRefList := range nodeStorageMap {
		namespacedName := types.NamespacedName{
			Name:      nodeStorageReference.Name,
			Namespace: nodeStorageReference.Namespace,
		}

		nnfNodeStorage := &nnfv1alpha1.NnfNodeStorage{}
		err := r.Get(ctx, namespacedName, nnfNodeStorage)
		if err != nil {
			return err
		}

		oldNnfNodeStorage := *nnfNodeStorage.DeepCopy()

		// The clientEndpoints field is an array of each of the allocations on the Rabbit
		// node that holds a list of the endpoints to expose the allocation to. The endpoints
		// are the swordfish endpoints, so 0 is the rabbit, and 1-16 are the computes. Start out
		// by clearing all compute node endpoints from the allocations.
		for i := range nnfNodeStorage.Spec.ClientEndpoints {
			nnfNodeStorage.Spec.ClientEndpoints[i].NodeNames = nnfNodeStorage.Spec.ClientEndpoints[i].NodeNames[:1]
		}

		// Add compute node endpoints for each of the allocations. Increment the compute node
		// index found from the "storage" resource to account for the 0 index being the rabbit
		// in swordfish.
		for _, mountRef := range mountRefList {
			clientEndpoints := &nnfNodeStorage.Spec.ClientEndpoints[mountRef.allocationIndex].NodeNames
			*clientEndpoints = append(*clientEndpoints, mountRef.client)
		}

		if reflect.DeepEqual(oldNnfNodeStorage, *nnfNodeStorage) {
			continue
		}

		err = r.Update(ctx, nnfNodeStorage)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *NnfAccessReconciler) getNodeStorageEndpointStatus(ctx context.Context, access *nnfv1alpha1.NnfAccess, storageMapping map[string][]dwsv1alpha1.ClientMountInfo) (bool, error) {
	// NnfNodeStorage clientReferences only need to be checked for compute nodes. If
	// this nnfAccess is not for compute nodes, then there's no work to do.
	if access.Spec.ClientReference == (corev1.ObjectReference{}) {
		return true, nil
	}

	nodeStorageMap := make(map[corev1.ObjectReference]bool)

	// Make a map of NnfNodeStorage references that were mounted by this
	// nnfAccess
	for _, storageList := range storageMapping {
		for _, mount := range storageList {
			if mount.Device.DeviceReference == nil {
				continue
			}

			if mount.Device.DeviceReference.ObjectReference.Kind != reflect.TypeOf(nnfv1alpha1.NnfNodeStorage{}).Name() {
				continue
			}

			nodeStorageMap[mount.Device.DeviceReference.ObjectReference] = true
		}
	}

	// Update each of the NnfNodeStorage resources to remove the clientEndpoints that
	// were added earlier. Leave the first endpoint since that corresponds to the
	// rabbit node.
	for nodeStorageReference := range nodeStorageMap {
		namespacedName := types.NamespacedName{
			Name:      nodeStorageReference.Name,
			Namespace: nodeStorageReference.Namespace,
		}

		nnfNodeStorage := &nnfv1alpha1.NnfNodeStorage{}
		err := r.Get(ctx, namespacedName, nnfNodeStorage)
		if err != nil {
			return false, err
		}

		if nnfNodeStorage.Status.Error != nil {
			access.Status.Error = nnfNodeStorage.Status.Error
			return false, nil
		}
	}

	return true, nil
}

// removeNodeStorageEndpoints modifies the NnfNodeStorage resources to remove the client endpoints for the
// compute nodes that had mounted the storage. This causes NnfNodeStorage to remove the StorageGroups for
// those compute nodes and remove access to the NVMe namespaces from the computes.
func (r *NnfAccessReconciler) removeNodeStorageEndpoints(ctx context.Context, access *nnfv1alpha1.NnfAccess, storageMapping map[string][]dwsv1alpha1.ClientMountInfo) error {
	// NnfNodeStorage clientReferences only need to be removed for compute nodes. If
	// this nnfAccess is not for compute nodes, then there's no work to do.
	if access.Spec.ClientReference == (corev1.ObjectReference{}) {
		return nil
	}

	nodeStorageMap := make(map[corev1.ObjectReference]bool)

	// Make a map of NnfNodeStorage references that were mounted by this
	// nnfAccess
	for _, storageList := range storageMapping {
		for _, mount := range storageList {
			if mount.Device.DeviceReference == nil {
				continue
			}

			if mount.Device.DeviceReference.ObjectReference.Kind != reflect.TypeOf(nnfv1alpha1.NnfNodeStorage{}).Name() {
				continue
			}

			nodeStorageMap[mount.Device.DeviceReference.ObjectReference] = true
		}
	}

	// Update each of the NnfNodeStorage resources to remove the clientEndpoints that
	// were added earlier. Leave the first endpoint since that corresponds to the
	// rabbit node.
	for nodeStorageReference := range nodeStorageMap {
		namespacedName := types.NamespacedName{
			Name:      nodeStorageReference.Name,
			Namespace: nodeStorageReference.Namespace,
		}

		nnfNodeStorage := &nnfv1alpha1.NnfNodeStorage{}
		err := r.Get(ctx, namespacedName, nnfNodeStorage)
		if err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			return err
		}

		oldNnfNodeStorage := *nnfNodeStorage.DeepCopy()

		for i := range nnfNodeStorage.Spec.ClientEndpoints {
			nnfNodeStorage.Spec.ClientEndpoints[i].NodeNames = nnfNodeStorage.Spec.ClientEndpoints[i].NodeNames[:1]
		}

		if reflect.DeepEqual(oldNnfNodeStorage, *nnfNodeStorage) {
			continue
		}

		err = r.Update(ctx, nnfNodeStorage)
		if err != nil {
			return err
		}
	}

	return nil
}

// createClientMounts creates the ClientMount resources based on the information in the storageMapping map.
func (r *NnfAccessReconciler) createClientMounts(ctx context.Context, access *nnfv1alpha1.NnfAccess, storageMapping map[string][]dwsv1alpha1.ClientMountInfo, log logr.Logger) error {
	g := new(errgroup.Group)

	for clientName, storageList := range storageMapping {
		clientName := clientName
		storageList := storageList

		// Start a goroutine for each ClientMount to create
		g.Go(func() error {
			clientMount := &dwsv1alpha1.ClientMount{
				ObjectMeta: metav1.ObjectMeta{
					Name:      clientMountName(access),
					Namespace: clientName,
				},
			}
			result, err := ctrl.CreateOrUpdate(ctx, r.Client, clientMount,
				func() error {
					dwsv1alpha1.InheritParentLabels(clientMount, access)
					dwsv1alpha1.AddOwnerLabels(clientMount, access)

					clientMount.Spec.Node = clientName
					clientMount.Spec.DesiredState = dwsv1alpha1.ClientMountState(access.Spec.DesiredState)
					clientMount.Spec.Mounts = storageList

					return nil
				})

			namespacedName := client.ObjectKeyFromObject(clientMount).String()
			if err != nil {
				log.Error(err, "failed to create or update ClientMount", "name", namespacedName)
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
func (r *NnfAccessReconciler) getClientMountStatus(ctx context.Context, access *nnfv1alpha1.NnfAccess, clientList []string) (bool, error) {
	clientMount := &dwsv1alpha1.ClientMount{}

	for _, clientName := range clientList {
		namespacedName := types.NamespacedName{
			Name:      clientMountName(access),
			Namespace: clientName,
		}

		err := r.Get(ctx, namespacedName, clientMount)
		if err != nil {
			if apierrors.IsNotFound(err) {
				return false, nil
			}

			return false, err
		}

		if len(clientMount.Status.Mounts) != len(clientMount.Spec.Mounts) {
			return false, nil
		}

		if clientMount.Status.Error != nil {
			access.Status.Error = clientMount.Status.Error
			return false, nil
		}

		for _, mount := range clientMount.Status.Mounts {
			if string(mount.State) != access.Status.State {
				return false, nil
			}

			if mount.Ready == false {
				return false, nil
			}
		}
	}

	return true, nil
}

func clientMountName(access *nnfv1alpha1.NnfAccess) string {
	return access.Namespace + "-" + access.Name
}

// SetupWithManager sets up the controller with the Manager.
func (r *NnfAccessReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.ChildObjects = []dwsv1alpha1.ObjectList{
		&dwsv1alpha1.ClientMountList{},
	}

	maxReconciles := runtime.GOMAXPROCS(0)
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(controller.Options{MaxConcurrentReconciles: maxReconciles}).
		For(&nnfv1alpha1.NnfAccess{}).
		Watches(&source.Kind{Type: &dwsv1alpha1.ClientMount{}}, handler.EnqueueRequestsFromMapFunc(dwsv1alpha1.OwnerLabelMapFunc)).
		Complete(r)
}
