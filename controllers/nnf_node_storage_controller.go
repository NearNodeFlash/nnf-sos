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
	"crypto/md5"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/mount-utils"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	ec "github.com/NearNodeFlash/nnf-ec/pkg/ec"
	nnf "github.com/NearNodeFlash/nnf-ec/pkg/manager-nnf"
	nnfserver "github.com/NearNodeFlash/nnf-ec/pkg/manager-server"

	openapi "github.com/NearNodeFlash/nnf-ec/pkg/rfsf/pkg/common"
	sf "github.com/NearNodeFlash/nnf-ec/pkg/rfsf/pkg/models"

	dwsv1alpha1 "github.com/HewlettPackard/dws/api/v1alpha1"
	"github.com/HewlettPackard/dws/utils/updater"
	nnfv1alpha1 "github.com/NearNodeFlash/nnf-sos/api/v1alpha1"
	"github.com/NearNodeFlash/nnf-sos/controllers/metrics"
)

const (
	// finalizerNnfNodeStorage defines the key used in identifying the
	// storage object as being owned by this NNF Storage Reconciler. This
	// prevents the system from deleting the custom resource until the
	// reconciler has finished using the resource.
	finalizerNnfNodeStorage = "nnf.cray.hpe.com/nnf_node_storage"

	nnfNodeStorageResourceName = "nnf-node-storage"
)

// NnfNodeStorageReconciler contains the elements needed during reconciliation for NnfNodeStorage
type NnfNodeStorageReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *kruntime.Scheme

	types.NamespacedName
}

//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfnodestorages,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfnodestorages/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.2/pkg/reconcile
func (r *NnfNodeStorageReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, err error) {

	metrics.NnfNodeStorageReconcilesTotal.Inc()

	nodeStorage := &nnfv1alpha1.NnfNodeStorage{}
	if err := r.Get(ctx, req.NamespacedName, nodeStorage); err != nil {
		// ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Ensure the NNF Storage Service is running prior to taking any action.
	ss := nnf.NewDefaultStorageService()
	storageService := &sf.StorageServiceV150StorageService{}
	if err := ss.StorageServiceIdGet(ss.Id(), storageService); err != nil {
		return ctrl.Result{}, err
	}

	if storageService.Status.State != sf.ENABLED_RST {
		return ctrl.Result{RequeueAfter: 1 * time.Second}, nil
	}

	// Use the Node Storage Status Updater to track updates to the storage status.
	// This ensures that only one call to r.Status().Update() is done even though we
	// update the status at several points in the process. We hijack the defer logic
	// to perform the status update if no other error is present in the system when
	// exiting this reconcile function. Note that "err" is the named return value,
	// so when we would normally call "return ctrl.Result{}, nil", at that time
	// "err" is nil - and if permitted we will update err with the result of
	// the r.Update()
	statusUpdater := updater.NewStatusUpdater[*nnfv1alpha1.NnfNodeStorageStatus](nodeStorage)
	defer func() { err = statusUpdater.CloseWithUpdate(ctx, r, err) }()

	// Check if the object is being deleted. Deletion is carefully coordinated around
	// the NNF resources being managed by this NNF Node Storage resource. For a
	// successful deletion, the NNF Storage Pool must be deleted. Deletion of the
	// Storage Pool handles the entire sub-tree of NNF resources (Storage Groups,
	// File System, and File Shares). The Finalizer on this NNF Node Storage resource
	// is present until the underlying NNF resources are deleted through the
	// storage service.
	if !nodeStorage.GetDeletionTimestamp().IsZero() {
		if !controllerutil.ContainsFinalizer(nodeStorage, finalizerNnfNodeStorage) {
			return ctrl.Result{}, nil
		}

		for i := range nodeStorage.Status.Allocations {
			// Release physical storage
			result, err := r.deleteStorage(nodeStorage, i)
			if err != nil {
				return ctrl.Result{Requeue: true}, nil
			}
			if result != nil {
				return *result, nil
			}
		}

		controllerutil.RemoveFinalizer(nodeStorage, finalizerNnfNodeStorage)
		if err := r.Update(ctx, nodeStorage); err != nil {
			if !apierrors.IsConflict(err) {
				return ctrl.Result{}, err
			}

			return ctrl.Result{Requeue: true}, nil
		}

		return ctrl.Result{}, nil
	}

	// First time setup requires programming of the storage status such that the resource
	// is labeled as "Starting" and all Conditions are initialized. After this is done,
	// the resource obtains a finalizer to manage the resource lifetime.
	if !controllerutil.ContainsFinalizer(nodeStorage, finalizerNnfNodeStorage) {
		controllerutil.AddFinalizer(nodeStorage, finalizerNnfNodeStorage)
		if err := r.Update(ctx, nodeStorage); err != nil {
			if !apierrors.IsConflict(err) {
				return ctrl.Result{}, err
			}

			return ctrl.Result{Requeue: true}, nil
		}

		return ctrl.Result{}, nil
	}

	// Initialize the status section with empty allocation statuses.
	if len(nodeStorage.Status.Allocations) == 0 {
		nodeStorage.Status.Allocations = make([]nnfv1alpha1.NnfNodeStorageAllocationStatus, nodeStorage.Spec.Count)

		for i := range nodeStorage.Status.Allocations {
			allocation := &nodeStorage.Status.Allocations[i]

			allocation.Conditions = nnfv1alpha1.NewConditions()
			allocation.StoragePool.Status = nnfv1alpha1.ResourceStarting
			allocation.StorageGroup.Status = nnfv1alpha1.ResourceStarting
			allocation.FileSystem.Status = nnfv1alpha1.ResourceStarting
			allocation.FileShare.Status = nnfv1alpha1.ResourceStarting
		}

		return ctrl.Result{}, nil
	}

	nodeStorage.Status.Error = nil

	// Loop through each allocation and create the storage
	for i := 0; i < nodeStorage.Spec.Count; i++ {
		// Allocate physical storage
		result, err := r.allocateStorage(nodeStorage, i)
		if err != nil {
			return ctrl.Result{}, err
		}
		if result != nil {
			return *result, nil
		}

		// Create a block device in /dev that is accessible on the Rabbit node
		result, err = r.createBlockDevice(ctx, nodeStorage, i)
		if err != nil {
			return ctrl.Result{}, err
		}
		if result != nil {
			return *result, nil
		}

		// Format the block device from the Rabbit with a file system (if needed)
		result, err = r.formatFileSystem(ctx, nodeStorage, i)
		if err != nil {
			return ctrl.Result{}, err
		}
		if result != nil {
			return *result, nil
		}
	}

	if nodeStorage.Spec.SetOwnerGroup && nodeStorage.Status.OwnerGroupStatus != nnfv1alpha1.ResourceReady {
		if nodeStorage.Status.OwnerGroupStatus == "" {
			nodeStorage.Status.OwnerGroupStatus = nnfv1alpha1.ResourceStarting

			return ctrl.Result{}, nil
		}

		if err := r.setLustreOwnerGroup(nodeStorage); err != nil {
			return ctrl.Result{}, err
		}

		nodeStorage.Status.OwnerGroupStatus = nnfv1alpha1.ResourceReady
	}

	return ctrl.Result{}, nil
}

func (r *NnfNodeStorageReconciler) allocateStorage(nodeStorage *nnfv1alpha1.NnfNodeStorage, index int) (*ctrl.Result, error) {
	log := r.Log.WithValues("NnfNodeStorage", types.NamespacedName{Name: nodeStorage.Name, Namespace: nodeStorage.Namespace})

	ss := nnf.NewDefaultStorageService()

	allocationStatus := &nodeStorage.Status.Allocations[index]

	condition := &allocationStatus.Conditions[nnfv1alpha1.ConditionIndexCreateStoragePool]
	if len(allocationStatus.StoragePool.ID) == 0 {
		condition.LastTransitionTime = metav1.Now()
		condition.Status = metav1.ConditionTrue
	}

	storagePoolID := fmt.Sprintf("%s-%d", nodeStorage.Name, index)
	sp, err := r.createStoragePool(ss, storagePoolID, nodeStorage.Spec.Capacity)
	if err != nil {
		updateError(condition, &allocationStatus.StoragePool, err)
		nodeStorage.Status.Error = dwsv1alpha1.NewResourceError("Could not create StoragePool", err)
		log.Info(nodeStorage.Status.Error.Error())

		return &ctrl.Result{Requeue: true}, nil
	}

	allocationStatus.StoragePool.Status = nnfv1alpha1.ResourceStatus(sp.Status)
	allocationStatus.StoragePool.Health = nnfv1alpha1.ResourceHealth(sp.Status)
	allocationStatus.CapacityAllocated = sp.CapacityBytes

	// If the SF ID is empty then we just created the resource. Save the ID in the NnfNodeStorage
	if len(allocationStatus.StoragePool.ID) == 0 {
		log.Info("Created storage pool", "Id", sp.Id)
		allocationStatus.StoragePool.ID = sp.Id
		condition.Status = metav1.ConditionFalse
		condition.Reason = nnfv1alpha1.ConditionSuccess
		condition.Message = ""

		return &ctrl.Result{}, nil
	}

	return nil, nil
}

func (r *NnfNodeStorageReconciler) createBlockDevice(ctx context.Context, nodeStorage *nnfv1alpha1.NnfNodeStorage, index int) (*ctrl.Result, error) {
	log := r.Log.WithValues("NnfNodeStorage", types.NamespacedName{Name: nodeStorage.Name, Namespace: nodeStorage.Namespace})
	ss := nnf.NewDefaultStorageService()

	allocationStatus := &nodeStorage.Status.Allocations[index]
	condition := &allocationStatus.Conditions[nnfv1alpha1.ConditionIndexCreateStorageGroup]

	// Create a Storage Group if none is currently present. Recall that a Storage Group
	// is a mapping from the Storage Pool to a Server Endpoint. Establishing a Storage
	// Group makes block storage available on the server, which itself is a prerequisite to
	// any file system built on top of the block storage.
	if len(allocationStatus.StorageGroup.ID) == 0 {
		condition.LastTransitionTime = metav1.Now()
		condition.Status = metav1.ConditionTrue
	}

	// Retrieve the collection of endpoints for us to map
	serverEndpointCollection := &sf.EndpointCollectionEndpointCollection{}
	if err := ss.StorageServiceIdEndpointsGet(ss.Id(), serverEndpointCollection); err != nil {
		nodeStorage.Status.Error = dwsv1alpha1.NewResourceError("Could not get service endpoint", err).WithFatal()
		log.Info(nodeStorage.Status.Error.Error())

		return &ctrl.Result{Requeue: true}, nil
	}

	// Get the Storage resource to map between compute node name and
	// endpoint index.
	namespacedName := types.NamespacedName{
		Name:      nodeStorage.Namespace,
		Namespace: "default",
	}

	storage := &dwsv1alpha1.Storage{}
	err := r.Get(ctx, namespacedName, storage)
	if err != nil {
		nodeStorage.Status.Error = dwsv1alpha1.NewResourceError("Could not read storage resource", err)
		log.Info(nodeStorage.Status.Error.Error())

		return &ctrl.Result{Requeue: true}, nil
	}

	// Build a list of all nodes with access to the storage
	clients := []string{}
	for _, server := range storage.Data.Access.Servers {
		clients = append(clients, server.Name)
	}

	for _, compute := range storage.Data.Access.Computes {
		clients = append(clients, compute.Name)
	}

	// Make a list of all the endpoints and set whether they need a storage group based
	// on the list of clients specified in the ClientEndpoints array
	accessList := make([]bool, len(serverEndpointCollection.Members))
	for _, nodeName := range nodeStorage.Spec.ClientEndpoints[index].NodeNames {
		for i, clientName := range clients {
			if nodeName == clientName {
				accessList[i] = true
			}
		}
	}

	// Loop through the list of endpoints and delete the StorageGroup for endpoints where
	// access==false, and create the StorageGroup for endpoints where access==true
	for clientIndex, access := range accessList {
		endpointRef := serverEndpointCollection.Members[clientIndex]
		endpointID := endpointRef.OdataId[strings.LastIndex(endpointRef.OdataId, "/")+1:]
		storageGroupID := fmt.Sprintf("%s-%d-%s", nodeStorage.Name, index, endpointID)

		// If the endpoint doesn't need a storage group, remove one if it exists
		if access == false {
			if _, err := r.getStorageGroup(ss, storageGroupID); err != nil {
				continue
			}

			if err := r.deleteStorageGroup(ss, storageGroupID); err != nil {
				nodeStorage.Status.Error = dwsv1alpha1.NewResourceError("Could not delete storage group", err).WithFatal()
				log.Info(nodeStorage.Status.Error.Error())

				return &ctrl.Result{Requeue: true}, nil
			}

			log.Info("Deleted storage group", "storageGroupID", storageGroupID)
		} else {
			// The kind environment doesn't support endpoints beyond the Rabbit
			if os.Getenv("ENVIRONMENT") == "kind" && endpointID != os.Getenv("RABBIT_NODE") {
				continue
			}

			endPoint, err := r.getEndpoint(ss, endpointID)
			if err != nil {
				nodeStorage.Status.Error = dwsv1alpha1.NewResourceError("Could not get endpoint", err).WithFatal()
				log.Info(nodeStorage.Status.Error.Error())

				return &ctrl.Result{Requeue: true}, nil
			}

			// Skip the endpoints that are not ready
			if nnfv1alpha1.StaticResourceStatus(endPoint.Status) != nnfv1alpha1.ResourceReady {
				continue
			}

			sg, err := r.createStorageGroup(ss, storageGroupID, allocationStatus.StoragePool.ID, endpointID)
			if err != nil {
				updateError(condition, &allocationStatus.StorageGroup, err)
				nodeStorage.Status.Error = dwsv1alpha1.NewResourceError("Could not create storage group", err).WithFatal()
				log.Info(nodeStorage.Status.Error.Error())

				return &ctrl.Result{Requeue: true}, nil
			}

			allocationStatus.StorageGroup.Status = nnfv1alpha1.ResourceStatus(sg.Status)
			allocationStatus.StorageGroup.Health = nnfv1alpha1.ResourceHealth(sg.Status)

			// If the SF ID is empty then we just created the resource. Save the ID in the NnfNodeStorage
			if len(allocationStatus.StorageGroup.ID) == 0 {
				log.Info("Created storage group", "storageGroupID", storageGroupID)
				allocationStatus.StorageGroup.ID = sg.Id
				condition.LastTransitionTime = metav1.Now()
				condition.Status = metav1.ConditionFalse // we are finished with this state
				condition.Reason = nnfv1alpha1.ConditionSuccess
				condition.Message = ""

				return &ctrl.Result{}, nil
			}
		}
	}

	return nil, nil
}

func (r *NnfNodeStorageReconciler) formatFileSystem(ctx context.Context, nodeStorage *nnfv1alpha1.NnfNodeStorage, index int) (*ctrl.Result, error) {
	log := r.Log.WithValues("NnfNodeStorage", types.NamespacedName{Name: nodeStorage.Name, Namespace: nodeStorage.Namespace})
	ss := nnf.NewDefaultStorageService()

	allocationStatus := &nodeStorage.Status.Allocations[index]

	// Check whether everything in the spec is filled in to make the FS. Lustre
	// MDTs and OSTs won't have their MgsNode field filled in until after the MGT
	// is created.
	if !r.isSpecComplete(nodeStorage) {
		return &ctrl.Result{}, nil
	}

	// Find the Rabbit node endpoint to collect LNet information
	endpoint, err := r.getEndpoint(ss, os.Getenv("RABBIT_NODE"))
	if err != nil {
		nnfv1alpha1.SetGetResourceFailureCondition(allocationStatus.Conditions, err)
		nodeStorage.Status.Error = dwsv1alpha1.NewResourceError("Could not get endpoint", err).WithFatal()
		log.Info(nodeStorage.Status.Error.Error())

		return &ctrl.Result{}, nil
	}

	nnfStorageProfile, err := getPinnedStorageProfileFromLabel(ctx, r.Client, nodeStorage)
	if err != nil {
		nnfv1alpha1.SetGetResourceFailureCondition(allocationStatus.Conditions, err)
		nodeStorage.Status.Error = dwsv1alpha1.NewResourceError("Could not find pinned storage profile", err).WithFatal()
		log.Info(nodeStorage.Status.Error.Error())

		return &ctrl.Result{}, nil
	}

	// Create the FileSystem
	condition := &allocationStatus.Conditions[nnfv1alpha1.ConditionIndexCreateFileSystem]
	if len(allocationStatus.FileSystem.ID) == 0 {
		condition.Status = metav1.ConditionTrue
		condition.LastTransitionTime = metav1.Now()
	}

	var fsType string
	if nodeStorage.Spec.FileSystemType == "raw" {
		fsType = "lvm"
	} else {
		fsType = nodeStorage.Spec.FileSystemType
	}
	oem := nnfserver.FileSystemOem{
		Type: fsType,
	}

	if oem.Type == "lustre" {
		setLusCmdLines := func(c *nnfv1alpha1.NnfStorageProfileLustreCmdLines) {
			oem.MkfsMount.Mkfs = c.Mkfs
			oem.ZfsCmd.ZpoolCreate = c.ZpoolCreate
		}

		setLusOpts := func(c *nnfv1alpha1.NnfStorageProfileLustreMiscOptions) {
			oem.MkfsMount.Mount = c.MountTarget
		}

		oem.Name = nodeStorage.Spec.LustreStorage.FileSystemName
		oem.Lustre.Index = nodeStorage.Spec.LustreStorage.StartIndex + index
		oem.Lustre.MgsNode = nodeStorage.Spec.LustreStorage.MgsNode
		oem.Lustre.TargetType = nodeStorage.Spec.LustreStorage.TargetType
		oem.Lustre.BackFs = nodeStorage.Spec.LustreStorage.BackFs

		switch nodeStorage.Spec.LustreStorage.TargetType {
		case "MGT":
			setLusCmdLines(&nnfStorageProfile.Data.LustreStorage.MgtCmdLines)
			setLusOpts(&nnfStorageProfile.Data.LustreStorage.MgtOptions)
		case "MDT":
			setLusCmdLines(&nnfStorageProfile.Data.LustreStorage.MdtCmdLines)
			setLusOpts(&nnfStorageProfile.Data.LustreStorage.MdtOptions)
		case "MGTMDT":
			setLusCmdLines(&nnfStorageProfile.Data.LustreStorage.MgtMdtCmdLines)
			setLusOpts(&nnfStorageProfile.Data.LustreStorage.MgtMdtOptions)
		case "OST":
			setLusCmdLines(&nnfStorageProfile.Data.LustreStorage.OstCmdLines)
			setLusOpts(&nnfStorageProfile.Data.LustreStorage.OstOptions)
		}
	}

	setCmdLines := func(c *nnfv1alpha1.NnfStorageProfileCmdLines) {
		oem.MkfsMount.Mkfs = c.Mkfs
		oem.LvmCmd.PvCreate = c.PvCreate
		oem.LvmCmd.VgCreate = c.VgCreate
		oem.LvmCmd.LvCreate = c.LvCreate
	}

	setOpts := func(c *nnfv1alpha1.NnfStorageProfileMiscOptions) {
		oem.MkfsMount.Mount = c.MountRabbit
	}

	if oem.Type == "gfs2" {
		// GFS2 requires a maximum of 16 alphanumeric, hyphen, or underscore characters. Allow up to 99 storage indicies and
		// generate a simple MD5SUM hash value from the node storage name for the tail end. Although not guaranteed, this
		// should reduce the likelihood of conflicts to a diminishingly small value.
		checksum := md5.Sum([]byte(nodeStorage.Name))
		oem.Name = fmt.Sprintf("fs-%02d-%x", index, string(checksum[0:5]))

		// The cluster name is the "name" of the Rabbit, which is mapped to the node storage namespace (since NNF Node Storage
		// is rabbit namespace scoped).
		oem.Gfs2.ClusterName = nodeStorage.Namespace
		setCmdLines(&nnfStorageProfile.Data.GFS2Storage.CmdLines)
		setOpts(&nnfStorageProfile.Data.GFS2Storage.Options)
	}

	if oem.Type == "xfs" {
		setCmdLines(&nnfStorageProfile.Data.XFSStorage.CmdLines)
		setOpts(&nnfStorageProfile.Data.XFSStorage.Options)
	}

	if oem.Type == "lvm" {
		setCmdLines(&nnfStorageProfile.Data.RawStorage.CmdLines)
	}

	fileSystemID := fmt.Sprintf("%s-%d", nodeStorage.Name, index)
	fs, err := r.createFileSystem(ss, fileSystemID, allocationStatus.StoragePool.ID, oem)
	if err != nil {
		updateError(condition, &allocationStatus.FileSystem, err)
		nodeStorage.Status.Error = dwsv1alpha1.NewResourceError("Could not create file system", err).WithFatal()
		log.Info(nodeStorage.Status.Error.Error())

		return &ctrl.Result{RequeueAfter: time.Minute * 2}, nil
	}

	allocationStatus.FileSystem.Status = nnfv1alpha1.ResourceReady
	allocationStatus.FileSystem.Health = nnfv1alpha1.ResourceOkay

	// If the SF ID is empty then we just created the resource. Save the ID in the NnfNodeStorage
	if len(allocationStatus.FileSystem.ID) == 0 {
		log.Info("Created filesystem", "Id", fs.Id)
		allocationStatus.FileSystem.ID = fs.Id
		condition.LastTransitionTime = metav1.Now()
		condition.Status = metav1.ConditionFalse
		condition.Reason = nnfv1alpha1.ConditionSuccess
		condition.Message = ""

		return &ctrl.Result{}, nil
	}

	// Create the FileShare
	condition = &allocationStatus.Conditions[nnfv1alpha1.ConditionIndexCreateFileShare]
	if len(allocationStatus.FileShare.ID) == 0 {
		condition.Status = metav1.ConditionTrue
		condition.LastTransitionTime = metav1.Now()
	}

	fileShareID := fmt.Sprintf("%s-%d", nodeStorage.Name, index)

	mountPath := ""
	sh, err := r.getFileShare(ss, fileShareID, allocationStatus.FileSystem.ID)
	if err == nil {
		mountPath = sh.FileSharePath
	}

	shareOptions := make(map[string]interface{})
	if nodeStorage.Spec.FileSystemType == "lustre" {
		targetIndex := nodeStorage.Spec.LustreStorage.StartIndex + index
		mountPath = "/mnt/lustre/" + nodeStorage.Spec.LustreStorage.FileSystemName + "/" + nodeStorage.Spec.LustreStorage.TargetType + strconv.Itoa(targetIndex)
	} else {
		shareOptions["volumeGroupName"] = volumeGroupName(fileShareID)
		shareOptions["logicalVolumeName"] = logicalVolumeName(fileShareID)
		shareOptions["userID"] = int(nodeStorage.Spec.UserID)
		shareOptions["groupID"] = int(nodeStorage.Spec.GroupID)
	}

	sh, err = r.createFileShare(ss, fileShareID, allocationStatus.FileSystem.ID, os.Getenv("RABBIT_NODE"), mountPath, shareOptions)
	if err != nil {
		updateError(condition, &allocationStatus.FileShare, err)
		nodeStorage.Status.Error = dwsv1alpha1.NewResourceError("Could not create file share", err).WithFatal()
		log.Info(nodeStorage.Status.Error.Error())

		return &ctrl.Result{RequeueAfter: time.Minute * 2}, nil
	}

	nid := ""
	if nidRaw, present := endpoint.Oem["LNetNids"]; present && nodeStorage.Spec.FileSystemType == "lustre" {
		nidList := nidRaw.([]string)
		if len(nidList) > 0 {
			// TODO: If there are multiple LNet Nids, have a way to pick
			// which network we want to use.
			nid = nidList[0]
		}
	}

	allocationStatus.FileShare.Status = nnfv1alpha1.ResourceStatus(sh.Status)
	allocationStatus.FileShare.Health = nnfv1alpha1.ResourceHealth(sh.Status)
	nodeStorage.Status.LustreStorage.Nid = nid

	// If the SF ID is empty then we just created the resource. Save the ID in the NnfNodeStorage
	if len(allocationStatus.FileShare.ID) == 0 {
		log.Info("Created file share", "Id", sh.Id)
		allocationStatus.FileShare.ID = sh.Id
		allocationStatus.VolumeGroup = volumeGroupName(fileShareID)
		allocationStatus.LogicalVolume = logicalVolumeName(fileShareID)
		condition.LastTransitionTime = metav1.Now()
		condition.Status = metav1.ConditionFalse
		condition.Reason = nnfv1alpha1.ConditionSuccess
		condition.Message = ""

		return &ctrl.Result{}, nil
	}

	return nil, nil
}

func (r *NnfNodeStorageReconciler) setLustreOwnerGroup(nodeStorage *nnfv1alpha1.NnfNodeStorage) (err error) {
	log := r.Log.WithValues("NnfNodeStorage", types.NamespacedName{Name: nodeStorage.Name, Namespace: nodeStorage.Namespace})

	_, found := os.LookupEnv("NNF_TEST_ENVIRONMENT")
	if found || os.Getenv("ENVIRONMENT") == "kind" {
		return nil
	}

	if nodeStorage.Spec.FileSystemType != "lustre" {
		return fmt.Errorf("Invalid file system type '%s' for setting owner/group", nodeStorage.Spec.FileSystemType)
	}

	target := "/mnt/nnf/client/" + nodeStorage.Name
	if err := os.MkdirAll(target, 0755); err != nil {
		log.Error(err, "Mkdir failed")
		return err
	}
	defer os.RemoveAll(target)

	mounter := mount.New("")
	mounted, err := mounter.IsMountPoint(target)
	if err != nil {
		return err
	}

	source := nodeStorage.Spec.LustreStorage.MgsNode + ":/" + nodeStorage.Spec.LustreStorage.FileSystemName

	if !mounted {
		if err := mounter.Mount(source, target, "lustre", nil); err != nil {
			log.Error(err, "Mount failed")
			return err
		}
	}
	defer func() {
		unmountErr := mounter.Unmount(target)
		if err == nil {
			err = unmountErr
		}
	}()

	if err := os.Chown(target, int(nodeStorage.Spec.UserID), int(nodeStorage.Spec.GroupID)); err != nil {
		log.Error(err, "Chown failed")
		return err
	}

	return nil
}

func (r *NnfNodeStorageReconciler) deleteStorage(nodeStorage *nnfv1alpha1.NnfNodeStorage, index int) (*ctrl.Result, error) {
	log := r.Log.WithValues("NnfNodeStorage", types.NamespacedName{Name: nodeStorage.Name, Namespace: nodeStorage.Namespace})

	ss := nnf.NewDefaultStorageService()

	allocationStatus := &nodeStorage.Status.Allocations[index]
	if allocationStatus.StoragePool.ID == "" {
		return nil, nil
	}

	condition := &allocationStatus.Conditions[nnfv1alpha1.ConditionIndexDeleteStoragePool]

	condition.Status = metav1.ConditionTrue
	condition.LastTransitionTime = metav1.Now()

	log.Info("Deleting storage pool", "Id", allocationStatus.StoragePool.ID)

	err := r.deleteStoragePool(ss, allocationStatus.StoragePool.ID)
	if err != nil {
		ecErr, ok := err.(*ec.ControllerError)

		// If the error is from a 404 error, then there's nothing to clean up and we
		// assume everything has been deleted
		if !ok || ecErr.StatusCode() != http.StatusNotFound {
			updateError(condition, &allocationStatus.FileShare, err)
			nodeStorage.Status.Error = dwsv1alpha1.NewResourceError("Could not delete storage pool", err).WithFatal()
			log.Info(nodeStorage.Status.Error.Error())

			return &ctrl.Result{Requeue: true}, nil
		}
	}

	allocationStatus.StoragePool.ID = ""
	allocationStatus.StorageGroup.ID = ""
	allocationStatus.FileSystem.ID = ""
	allocationStatus.FileShare.ID = ""
	allocationStatus.StoragePool.Status = nnfv1alpha1.ResourceDeleted
	allocationStatus.StorageGroup.Status = nnfv1alpha1.ResourceDeleted
	allocationStatus.FileSystem.Status = nnfv1alpha1.ResourceDeleted
	allocationStatus.FileShare.Status = nnfv1alpha1.ResourceDeleted
	allocationStatus.VolumeGroup = ""
	allocationStatus.LogicalVolume = ""
	nodeStorage.Status.LustreStorage.Nid = ""

	return &ctrl.Result{}, nil
}

func volumeGroupName(id string) string {
	return fmt.Sprintf("%s_vg", id)
}

func logicalVolumeName(id string) string {
	return fmt.Sprintf("%s_lv", id)
}

func (r *NnfNodeStorageReconciler) isSpecComplete(nodeStorage *nnfv1alpha1.NnfNodeStorage) bool {
	if nodeStorage.Spec.FileSystemType != "lustre" {
		return true
	}

	if nodeStorage.Spec.LustreStorage.TargetType == "MGT" || nodeStorage.Spec.LustreStorage.TargetType == "MGTMDT" {
		return true
	}

	if len(nodeStorage.Spec.LustreStorage.MgsNode) > 0 {
		return true
	}

	return false
}

func (r *NnfNodeStorageReconciler) createStoragePool(ss nnf.StorageServiceApi, id string, capacity int64) (*sf.StoragePoolV150StoragePool, error) {
	sp := &sf.StoragePoolV150StoragePool{
		Id:            id,
		CapacityBytes: capacity,
		Oem: openapi.MarshalOem(nnf.AllocationPolicyOem{
			Policy:     nnf.SpareAllocationPolicyType,
			Compliance: nnf.RelaxedAllocationComplianceType,
		}),
	}

	if err := ss.StorageServiceIdStoragePoolIdPut(ss.Id(), id, sp); err != nil {
		ecErr, ok := err.(*ec.ControllerError)
		if ok {
			resourceErr := dwsv1alpha1.NewResourceError("", err)
			switch ecErr.Cause() {
			case "Insufficient capacity available":
				return nil, resourceErr.WithUserMessage("Insufficient capacity available").WithFatal()
			default:
				return nil, err
			}
		}

		return nil, err
	}

	return sp, nil
}

func (r *NnfNodeStorageReconciler) getStoragePool(ss nnf.StorageServiceApi, id string) (*sf.StoragePoolV150StoragePool, error) {
	sp := &sf.StoragePoolV150StoragePool{}

	if err := ss.StorageServiceIdStoragePoolIdGet(ss.Id(), id, sp); err != nil {
		return nil, err
	}

	return sp, nil
}

func (r *NnfNodeStorageReconciler) deleteStoragePool(ss nnf.StorageServiceApi, id string) error {
	if err := ss.StorageServiceIdStoragePoolIdDelete(ss.Id(), id); err != nil {
		return err
	}

	return nil
}

func (r *NnfNodeStorageReconciler) getEndpoint(ss nnf.StorageServiceApi, id string) (*sf.EndpointV150Endpoint, error) {
	ep := &sf.EndpointV150Endpoint{}

	if err := ss.StorageServiceIdEndpointIdGet(ss.Id(), id, ep); err != nil {
		return nil, err
	}

	return ep, nil
}

func (r *NnfNodeStorageReconciler) createStorageGroup(ss nnf.StorageServiceApi, id string, spID string, epID string) (*sf.StorageGroupV150StorageGroup, error) {
	sp, err := r.getStoragePool(ss, spID)
	if err != nil {
		return nil, err
	}

	ep, err := r.getEndpoint(ss, epID)
	if err != nil {
		return nil, err
	}

	sg := &sf.StorageGroupV150StorageGroup{
		Id: id,
		Links: sf.StorageGroupV150Links{
			StoragePool:    sf.OdataV4IdRef{OdataId: sp.OdataId},
			ServerEndpoint: sf.OdataV4IdRef{OdataId: ep.OdataId},
		},
	}

	if err := ss.StorageServiceIdStorageGroupIdPut(ss.Id(), id, sg); err != nil {
		return nil, err
	}

	return sg, nil
}

func (r *NnfNodeStorageReconciler) getStorageGroup(ss nnf.StorageServiceApi, id string) (*sf.StorageGroupV150StorageGroup, error) {
	sg := &sf.StorageGroupV150StorageGroup{}

	if err := ss.StorageServiceIdStorageGroupIdGet(ss.Id(), id, sg); err != nil {
		return nil, err
	}

	return sg, nil
}

func (r *NnfNodeStorageReconciler) deleteStorageGroup(ss nnf.StorageServiceApi, id string) error {
	return ss.StorageServiceIdStorageGroupIdDelete(ss.Id(), id)
}

func (r *NnfNodeStorageReconciler) createFileShare(ss nnf.StorageServiceApi, id string, fsID string, epID string, mountPath string, options map[string]interface{}) (*sf.FileShareV120FileShare, error) {
	fs, err := r.getFileSystem(ss, fsID)
	if err != nil {
		return nil, err
	}

	ep, err := r.getEndpoint(ss, epID)
	if err != nil {
		return nil, err
	}

	sh := &sf.FileShareV120FileShare{
		Id:            id,
		FileSharePath: mountPath,
		Oem:           options,
		Links: sf.FileShareV120Links{
			FileSystem: sf.OdataV4IdRef{OdataId: fs.OdataId},
			Endpoint:   sf.OdataV4IdRef{OdataId: ep.OdataId},
		},
	}

	if err := ss.StorageServiceIdFileSystemIdExportedShareIdPut(ss.Id(), id, fs.Id, sh); err != nil {
		return nil, err
	}

	return sh, nil
}

func (r *NnfNodeStorageReconciler) getFileShare(ss nnf.StorageServiceApi, id string, fsID string) (*sf.FileShareV120FileShare, error) {
	fs, err := r.getFileSystem(ss, fsID)
	if err != nil {
		return nil, err
	}

	sh := &sf.FileShareV120FileShare{}

	if err := ss.StorageServiceIdFileSystemIdExportedShareIdGet(ss.Id(), fs.Id, id, sh); err != nil {
		return nil, err
	}

	return sh, nil
}

func (r *NnfNodeStorageReconciler) createFileSystem(ss nnf.StorageServiceApi, id string, spID string, oem nnfserver.FileSystemOem) (*sf.FileSystemV122FileSystem, error) {
	sp, err := r.getStoragePool(ss, spID)
	if err != nil {
		return nil, err
	}

	if oem.Name == "" {
		oem.Name = id
	}

	fs := &sf.FileSystemV122FileSystem{
		Id: id,
		Links: sf.FileSystemV122Links{
			StoragePool: sf.OdataV4IdRef{OdataId: sp.OdataId},
		},
		Oem: openapi.MarshalOem(oem),
	}

	if err := ss.StorageServiceIdFileSystemIdPut(ss.Id(), id, fs); err != nil {
		return nil, err
	}

	return fs, nil
}

func (r *NnfNodeStorageReconciler) getFileSystem(ss nnf.StorageServiceApi, id string) (*sf.FileSystemV122FileSystem, error) {
	fs := &sf.FileSystemV122FileSystem{}

	if err := ss.StorageServiceIdFileSystemIdGet(ss.Id(), id, fs); err != nil {
		return nil, err
	}

	return fs, nil
}

func updateError(condition *metav1.Condition, status *nnfv1alpha1.NnfResourceStatus, err error) {
	status.Status = nnfv1alpha1.ResourceFailed
	condition.Reason = nnfv1alpha1.ConditionFailed
	condition.Message = err.Error()
}

// SetupWithManager sets up the controller with the Manager.
func (r *NnfNodeStorageReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// nnf-ec is not thread safe, so we are limited to a single reconcile thread.
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(controller.Options{MaxConcurrentReconciles: 1}).
		For(&nnfv1alpha1.NnfNodeStorage{}).
		Complete(r)
}
