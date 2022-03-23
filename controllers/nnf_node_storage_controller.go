/*
Copyright 2021 Hewlett Packard Enterprise Development LP
*/

package controllers

import (
	"context"
	"crypto/md5"
	"fmt"
	"net/http"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	ec "github.hpe.com/hpe/hpc-rabsw-nnf-ec/pkg/ec"
	nnf "github.hpe.com/hpe/hpc-rabsw-nnf-ec/pkg/manager-nnf"
	nnfserver "github.hpe.com/hpe/hpc-rabsw-nnf-ec/pkg/manager-server"

	openapi "github.hpe.com/hpe/hpc-rabsw-nnf-ec/pkg/rfsf/pkg/common"
	sf "github.hpe.com/hpe/hpc-rabsw-nnf-ec/pkg/rfsf/pkg/models"

	nnfv1alpha1 "github.hpe.com/hpe/hpc-rabsw-nnf-sos/api/v1alpha1"
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
	nodeStorage := &nnfv1alpha1.NnfNodeStorage{}
	if err := r.Get(ctx, req.NamespacedName, nodeStorage); err != nil {
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
	statusUpdater := newNodeStorageStatusUpdater(nodeStorage)
	defer func() {
		if err == nil {
			err = statusUpdater.close(ctx, r)
		}
	}()

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
			result, err := r.deleteStorage(statusUpdater, nodeStorage, i)
			if err != nil {
				return ctrl.Result{Requeue: true}, nil
			}
			if result != nil {
				return *result, nil
			}
		}

		controllerutil.RemoveFinalizer(nodeStorage, finalizerNnfNodeStorage)
		if err := r.Update(ctx, nodeStorage); err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	// First time setup requires programming of the storage status such that the resource
	// is labeled as "Starting" and all Conditions are initialized. After this is done,
	// the resource obtains a finalizer to manage the resource lifetime.
	if !controllerutil.ContainsFinalizer(nodeStorage, finalizerNnfNodeStorage) {
		controllerutil.AddFinalizer(nodeStorage, finalizerNnfNodeStorage)
		if err := r.Update(ctx, nodeStorage); err != nil {
			return ctrl.Result{Requeue: true}, nil
		}

		return ctrl.Result{}, nil
	}

	// Initialize the status section with empty allocation statuses.
	if len(nodeStorage.Status.Allocations) == 0 {
		statusUpdater.update(func(*nnfv1alpha1.NnfNodeStorageStatus) {
			nodeStorage.Status.Allocations = make([]nnfv1alpha1.NnfNodeStorageAllocationStatus, nodeStorage.Spec.Count)
			for i := range nodeStorage.Status.Allocations {
				nodeStorage.Status.Allocations[i].Conditions = nnfv1alpha1.NewConditions()
				nodeStorage.Status.Allocations[i].StoragePool.Status = nnfv1alpha1.ResourceStarting
				nodeStorage.Status.Allocations[i].StorageGroup.Status = nnfv1alpha1.ResourceStarting
				nodeStorage.Status.Allocations[i].FileSystem.Status = nnfv1alpha1.ResourceStarting
				nodeStorage.Status.Allocations[i].FileShare.Status = nnfv1alpha1.ResourceStarting
			}
		})

		return ctrl.Result{}, nil
	}

	// Loop through each allocation and create the storage
	for i := 0; i < nodeStorage.Spec.Count; i++ {
		// Allocate physical storage
		result, err := r.allocateStorage(statusUpdater, nodeStorage, i)
		if err != nil {
			return ctrl.Result{}, err
		}
		if result != nil {
			return *result, nil
		}

		// Create a block device in /dev that is accessible on the Rabbit node
		result, err = r.createBlockDevice(statusUpdater, nodeStorage, i)
		if err != nil {
			return ctrl.Result{}, err
		}
		if result != nil {
			return *result, nil
		}

		// Format the block device from the Rabbit with a file system (if needed)
		result, err = r.formatFileSystem(statusUpdater, nodeStorage, i)
		if err != nil {
			return ctrl.Result{}, err
		}
		if result != nil {
			return *result, nil
		}
	}

	return ctrl.Result{}, nil
}

func (r *NnfNodeStorageReconciler) allocateStorage(statusUpdater *nodeStorageStatusUpdater, nodeStorage *nnfv1alpha1.NnfNodeStorage, index int) (*ctrl.Result, error) {
	log := r.Log.WithValues("NnfNodeStorage", types.NamespacedName{Name: nodeStorage.Name, Namespace: nodeStorage.Namespace})
	ss := nnf.NewDefaultStorageService()

	allocationStatus := &nodeStorage.Status.Allocations[index]

	condition := &allocationStatus.Conditions[nnfv1alpha1.ConditionIndexCreateStoragePool]
	if len(allocationStatus.StoragePool.ID) == 0 {
		statusUpdater.update(func(*nnfv1alpha1.NnfNodeStorageStatus) {
			condition.LastTransitionTime = metav1.Now()
			condition.Status = metav1.ConditionTrue
		})
	}

	storagePoolID := fmt.Sprintf("%s-%d", nodeStorage.Name, index)
	sp, err := r.createStoragePool(ss, storagePoolID, nodeStorage.Spec.Capacity)
	if err != nil {
		statusUpdater.updateError(condition, &allocationStatus.StoragePool, err)

		return &ctrl.Result{Requeue: true}, nil
	}

	// If the SF ID is empty then we just created the resource. Save the ID in the NnfNodeStorage
	if len(allocationStatus.StoragePool.ID) == 0 {
		log.Info("Created storage pool", "Id", sp.Id)
		statusUpdater.update(func(*nnfv1alpha1.NnfNodeStorageStatus) {
			allocationStatus.StoragePool.ID = sp.Id
			condition.Status = metav1.ConditionFalse
			condition.Reason = nnfv1alpha1.ConditionSuccess
			condition.Message = ""
		})
	}

	statusUpdater.update(func(*nnfv1alpha1.NnfNodeStorageStatus) {
		allocationStatus.StoragePool.Status = nnfv1alpha1.ResourceStatus(sp.Status)
		allocationStatus.StoragePool.Health = nnfv1alpha1.ResourceHealth(sp.Status)
		allocationStatus.CapacityAllocated = sp.CapacityBytes
	})

	return nil, nil
}

func (r *NnfNodeStorageReconciler) createBlockDevice(statusUpdater *nodeStorageStatusUpdater, nodeStorage *nnfv1alpha1.NnfNodeStorage, index int) (*ctrl.Result, error) {
	log := r.Log.WithValues("NnfNodeStorage", types.NamespacedName{Name: nodeStorage.Name, Namespace: nodeStorage.Namespace})
	ss := nnf.NewDefaultStorageService()

	allocationStatus := &nodeStorage.Status.Allocations[index]
	condition := &allocationStatus.Conditions[nnfv1alpha1.ConditionIndexCreateStorageGroup]

	// Create a Storage Group if none is currently present. Recall that a Storage Group
	// is a mapping from the Storage Pool to a Server Endpoint. Establishing a Storage
	// Group makes block storage available on the server, which itself is a prerequisite to
	// any file system built on top of the block storage.
	if len(allocationStatus.StorageGroup.ID) == 0 {
		statusUpdater.update(func(*nnfv1alpha1.NnfNodeStorageStatus) {
			condition.LastTransitionTime = metav1.Now()
			condition.Status = metav1.ConditionTrue
		})
	}

	// Retrieve the collection of endpoints for us to map
	serverEndpointCollection := &sf.EndpointCollectionEndpointCollection{}
	if err := ss.StorageServiceIdEndpointsGet(ss.Id(), serverEndpointCollection); err != nil {
		log.Error(err, "Failed to retrieve Storage Service Endpoints")
		return &ctrl.Result{}, err
	}

	// Iterate over the server endpoints to ensure we've reflected
	// the status of each server (Compute & Rabbit)
	for idx := range serverEndpointCollection.Members {

		serverEndpoint := serverEndpointCollection.Members[idx]
		id := serverEndpoint.OdataId[strings.LastIndex(serverEndpoint.OdataId, "/")+1:]
		endPoint, err := r.getEndpoint(ss, id)
		if err != nil {
			log.Error(err, "Failed to get endpoint", "id", id)
			continue // Check all endpoints
		}

		// Skip the endpoints that are not ready
		if nnfv1alpha1.StaticResourceStatus(endPoint.Status) != nnfv1alpha1.ResourceReady {
			continue
		}

		storageGroupID := fmt.Sprintf("%s-%s", nodeStorage.Name, id)

		sg, err := r.createStorageGroup(ss, storageGroupID, allocationStatus.StoragePool.ID, id)
		if err != nil {
			statusUpdater.updateError(condition, &allocationStatus.StorageGroup, err)

			return &ctrl.Result{Requeue: true}, nil
		}

		log.Info("Created storage group", "Id", sg.Id, "storageGroupID", storageGroupID)

		// If the SF ID is empty then we just created the resource. Save the ID in the NnfNodeStorage
		if len(allocationStatus.StorageGroup.ID) == 0 {
			statusUpdater.update(func(*nnfv1alpha1.NnfNodeStorageStatus) {
				allocationStatus.StorageGroup.ID = sg.Id
				condition.LastTransitionTime = metav1.Now()
				condition.Status = metav1.ConditionFalse // we are finished with this state
				condition.Reason = nnfv1alpha1.ConditionSuccess
				condition.Message = ""
			})
		}

		statusUpdater.update(func(*nnfv1alpha1.NnfNodeStorageStatus) {
			allocationStatus.StorageGroup.Status = nnfv1alpha1.ResourceStatus(sg.Status)
			allocationStatus.StorageGroup.Health = nnfv1alpha1.ResourceHealth(sg.Status)
		})

	}

	return nil, nil
}

func (r *NnfNodeStorageReconciler) formatFileSystem(statusUpdater *nodeStorageStatusUpdater, nodeStorage *nnfv1alpha1.NnfNodeStorage, index int) (*ctrl.Result, error) {
	log := r.Log.WithValues("NnfNodeStorage", types.NamespacedName{Name: nodeStorage.Name, Namespace: nodeStorage.Namespace})
	ss := nnf.NewDefaultStorageService()

	allocationStatus := &nodeStorage.Status.Allocations[index]

	// Raw storage doesn't need a file system
	if nodeStorage.Spec.FileSystemType == "raw" {
		return nil, nil
	}

	// Check whether everything in the spec is filled in to make the FS. Lustre
	// MDTs and OSTs won't have their MgsNode field filled in until after the MGT
	// is created.
	if !r.isSpecComplete(nodeStorage) {
		return &ctrl.Result{}, nil
	}

	// Find the Rabbit node endpoint to collect LNet information
	endpoint, err := r.getEndpoint(ss, os.Getenv("RABBIT_NODE"))
	if err != nil {
		statusUpdater.update(func(*nnfv1alpha1.NnfNodeStorageStatus) {
			nnfv1alpha1.SetGetResourceFailureCondition(allocationStatus.Conditions, err)
		})

		return &ctrl.Result{}, nil
	}

	// Create the FileSystem
	condition := &allocationStatus.Conditions[nnfv1alpha1.ConditionIndexCreateFileSystem]
	if len(allocationStatus.FileSystem.ID) == 0 {
		statusUpdater.update(func(*nnfv1alpha1.NnfNodeStorageStatus) {
			condition.Status = metav1.ConditionTrue
			condition.LastTransitionTime = metav1.Now()
		})
	}

	oem := nnfserver.FileSystemOem{
		Type: nodeStorage.Spec.FileSystemType,

		// If not lustre, then these will be appropriate zero values.
		Name:       nodeStorage.Spec.LustreStorage.FileSystemName,
		Index:      nodeStorage.Spec.LustreStorage.StartIndex + index,
		MgsNode:    nodeStorage.Spec.LustreStorage.MgsNode,
		TargetType: nodeStorage.Spec.LustreStorage.TargetType,
		BackFs:     nodeStorage.Spec.LustreStorage.BackFs,
	}

	if oem.Type == "gfs2" {
		// GFS2 requires a maximum of 16 alphanumeric, hyphen, or underscore characters. Allow up to 99 storage indecies and
		// generate a simple MD5SUM hash value from the node storage name for the tail end. Although not guaranteed, this
		// should reduce the likelihood of conflicts to a diminishingly small value.
		checksum := md5.Sum([]byte(nodeStorage.Name))
		oem.Name = fmt.Sprintf("fs-%02d-%x", index, string(checksum[0:5]))
		oem.ClusterName = nodeStorage.Name
	}

	fileSystemID := fmt.Sprintf("%s-%d", nodeStorage.Name, index)
	fs, err := r.createFileSystem(ss, fileSystemID, allocationStatus.StoragePool.ID, oem)
	if err != nil {
		statusUpdater.updateError(condition, &allocationStatus.FileSystem, err)

		return &ctrl.Result{Requeue: true}, nil
	}

	// If the SF ID is empty then we just created the resource. Save the ID in the NnfNodeStorage
	if len(allocationStatus.FileSystem.ID) == 0 {
		log.Info("Created filesystem", "Id", fs.Id)
		statusUpdater.update(func(*nnfv1alpha1.NnfNodeStorageStatus) {
			allocationStatus.FileSystem.ID = fs.Id
			condition.LastTransitionTime = metav1.Now()
			condition.Status = metav1.ConditionFalse
			condition.Reason = nnfv1alpha1.ConditionSuccess
			condition.Message = ""
		})
	}

	statusUpdater.update(func(*nnfv1alpha1.NnfNodeStorageStatus) {
		allocationStatus.FileSystem.Status = nnfv1alpha1.ResourceReady
		allocationStatus.FileSystem.Health = nnfv1alpha1.ResourceOkay
	})

	// Create the FileShare
	condition = &allocationStatus.Conditions[nnfv1alpha1.ConditionIndexCreateFileShare]
	if len(allocationStatus.FileShare.ID) == 0 {
		statusUpdater.update(func(*nnfv1alpha1.NnfNodeStorageStatus) {
			condition.Status = metav1.ConditionTrue
			condition.LastTransitionTime = metav1.Now()
		})
	}

	mountPath := ""
	if nodeStorage.Spec.FileSystemType == "lustre" {
		targetIndex := nodeStorage.Spec.LustreStorage.StartIndex + index
		mountPath = "/mnt/lustre/" + nodeStorage.Spec.LustreStorage.FileSystemName + "/" + nodeStorage.Spec.LustreStorage.TargetType + strconv.Itoa(targetIndex)
	}

	fileShareID := fmt.Sprintf("%s-%d", nodeStorage.Name, index)
	sh, err := r.createFileShare(ss, fileShareID, allocationStatus.FileSystem.ID, os.Getenv("RABBIT_NODE"), mountPath)
	if err != nil {
		statusUpdater.updateError(condition, &allocationStatus.FileShare, err)

		return &ctrl.Result{Requeue: true}, nil
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

	// If the SF ID is empty then we just created the resource. Save the ID in the NnfNodeStorage
	if len(allocationStatus.FileShare.ID) == 0 {
		log.Info("Created file share", "Id", sh.Id)
		statusUpdater.update(func(*nnfv1alpha1.NnfNodeStorageStatus) {
			allocationStatus.FileShare.ID = sh.Id
			condition.LastTransitionTime = metav1.Now()
			condition.Status = metav1.ConditionFalse
			condition.Reason = nnfv1alpha1.ConditionSuccess
			condition.Message = ""
		})
	}

	statusUpdater.update(func(*nnfv1alpha1.NnfNodeStorageStatus) {
		allocationStatus.FileShare.Status = nnfv1alpha1.ResourceStatus(sh.Status)
		allocationStatus.FileShare.Health = nnfv1alpha1.ResourceHealth(sh.Status)
		nodeStorage.Status.LustreStorage.Nid = nid
	})

	return nil, nil
}

func (r *NnfNodeStorageReconciler) deleteStorage(statusUpdater *nodeStorageStatusUpdater, nodeStorage *nnfv1alpha1.NnfNodeStorage, index int) (*ctrl.Result, error) {
	log := r.Log.WithValues("NnfNodeStorage", types.NamespacedName{Name: nodeStorage.Name, Namespace: nodeStorage.Namespace})
	ss := nnf.NewDefaultStorageService()

	allocationStatus := &nodeStorage.Status.Allocations[index]
	if allocationStatus.StoragePool.ID == "" {
		return nil, nil
	}

	condition := &allocationStatus.Conditions[nnfv1alpha1.ConditionIndexDeleteStoragePool]

	statusUpdater.update(func(*nnfv1alpha1.NnfNodeStorageStatus) {
		condition.Status = metav1.ConditionTrue
		condition.LastTransitionTime = metav1.Now()
	})

	log.Info("Deleting storage pool", "Id", allocationStatus.StoragePool.ID)

	err := r.deleteStoragePool(ss, allocationStatus.StoragePool.ID)
	if err != nil {
		ecErr, ok := err.(*ec.ControllerError)

		// If the error is from a 404 error, then there's nothing to clean up and we
		// assume everything has been deleted
		if !ok || ecErr.StatusCode() != http.StatusNotFound {
			statusUpdater.updateError(condition, &allocationStatus.FileShare, err)

			return &ctrl.Result{Requeue: true}, nil
		}
	}

	statusUpdater.update(func(*nnfv1alpha1.NnfNodeStorageStatus) {
		allocationStatus.StoragePool.ID = ""
		allocationStatus.StorageGroup.ID = ""
		allocationStatus.FileSystem.ID = ""
		allocationStatus.FileShare.ID = ""
		allocationStatus.StoragePool.Status = nnfv1alpha1.ResourceDeleted
		allocationStatus.StorageGroup.Status = nnfv1alpha1.ResourceDeleted
		allocationStatus.FileSystem.Status = nnfv1alpha1.ResourceDeleted
		allocationStatus.FileShare.Status = nnfv1alpha1.ResourceDeleted
		nodeStorage.Status.LustreStorage.Nid = ""
	})

	return &ctrl.Result{}, nil
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

func (r *NnfNodeStorageReconciler) createFileShare(ss nnf.StorageServiceApi, id string, fsID string, epID string, mountPath string) (*sf.FileShareV120FileShare, error) {
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

func (r *NnfNodeStorageReconciler) getFileShare(ss nnf.StorageServiceApi, fsID string, id string) (*sf.FileShareV120FileShare, error) {
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

type nodeStorageStatusUpdater struct {
	storage        *nnfv1alpha1.NnfNodeStorage
	existingStatus nnfv1alpha1.NnfNodeStorageStatus
}

func newNodeStorageStatusUpdater(n *nnfv1alpha1.NnfNodeStorage) *nodeStorageStatusUpdater {
	return &nodeStorageStatusUpdater{
		storage:        n,
		existingStatus: (*n.DeepCopy()).Status,
	}
}

func (s *nodeStorageStatusUpdater) update(update func(*nnfv1alpha1.NnfNodeStorageStatus)) {
	update(&s.storage.Status)
}

func (s *nodeStorageStatusUpdater) updateError(condition *metav1.Condition, status *nnfv1alpha1.NnfResourceStatus, err error) {
	status.Status = nnfv1alpha1.ResourceFailed
	condition.Reason = nnfv1alpha1.ConditionFailed
	condition.Message = err.Error()
}

func (s *nodeStorageStatusUpdater) close(ctx context.Context, r *NnfNodeStorageReconciler) error {
	if !reflect.DeepEqual(s.storage.Status, s.existingStatus) {
		err := r.Update(ctx, s.storage)
		if !apierrors.IsConflict(err) {
			return err
		}
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NnfNodeStorageReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// nnf-ec is not thread safe, so we are limited to a single reconcile thread.
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(controller.Options{MaxConcurrentReconciles: 1}).
		For(&nnfv1alpha1.NnfNodeStorage{}).
		Complete(r)
}
