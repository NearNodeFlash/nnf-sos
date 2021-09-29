/*
Copyright 2021 Hewlett Packard Enterprise Development LP
*/

package controllers

import (
	"context"
	"os"
	"reflect"
	"strconv"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	nnf "stash.us.cray.com/rabsw/nnf-ec/pkg/manager-nnf"
	nnfserver "stash.us.cray.com/rabsw/nnf-ec/pkg/manager-server"

	openapi "stash.us.cray.com/rabsw/nnf-ec/pkg/rfsf/pkg/common"
	sf "stash.us.cray.com/rabsw/nnf-ec/pkg/rfsf/pkg/models"

	nnfv1alpha1 "stash.us.cray.com/RABSW/nnf-sos/api/v1alpha1"
)

const (
	nnfNodeStorageResourceName = "nnf-node-storage"
)

// NnfNodeStorageReconciler contains the elements needed during reconcilation for NnfNodeStorage
type NnfNodeStorageReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme

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
		if !controllerutil.ContainsFinalizer(nodeStorage, finalizer) {
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

		controllerutil.RemoveFinalizer(nodeStorage, finalizer)
		if err := r.Update(ctx, nodeStorage); err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	// First time setup requires programming of the storage status such that the resource
	// is labeled as "Starting" and all Conditions are initializaed. After this is done,
	// the resource obtains a finalizer to manage the resource lifetime.
	if !controllerutil.ContainsFinalizer(nodeStorage, finalizer) {
		controllerutil.AddFinalizer(nodeStorage, finalizer)
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
	log := r.Log.WithValues("NodeStorage", types.NamespacedName{Name: nodeStorage.Name, Namespace: nodeStorage.Namespace})
	ss := nnf.NewDefaultStorageService()

	allocationStatus := &nodeStorage.Status.Allocations[index]

	if len(allocationStatus.StoragePool.Id) > 0 {
		sp, err := r.getStoragePool(ss, allocationStatus.StoragePool.Id)
		if err != nil {
			statusUpdater.update(func(*nnfv1alpha1.NnfNodeStorageStatus) {
				nnfv1alpha1.SetGetResourceFailureCondition(allocationStatus.Conditions, err)
			})

			return &ctrl.Result{}, nil
		}

		equal := statusUpdater.updateWithEqual(func(*nnfv1alpha1.NnfNodeStorageStatus) {
			allocationStatus.StoragePool.Status = nnfv1alpha1.ResourceStatus(sp.Status)
			allocationStatus.StoragePool.Health = nnfv1alpha1.ResourceHealth(sp.Status)
		})

		if !equal {
			return &ctrl.Result{}, nil
		}

		return nil, nil
	}

	condition := &allocationStatus.Conditions[nnfv1alpha1.ConditionIndexCreateStoragePool]

	statusUpdater.update(func(*nnfv1alpha1.NnfNodeStorageStatus) {
		condition.LastTransitionTime = metav1.Now()
		condition.Status = metav1.ConditionTrue
	})

	sp, err := r.createStoragePool(ss, nodeStorage.Spec.Capacity)
	if err != nil {
		statusUpdater.updateError(condition, &allocationStatus.StoragePool, err)

		return &ctrl.Result{Requeue: true}, nil
	}

	statusUpdater.updateSuccess(condition, func(*nnfv1alpha1.NnfNodeStorageStatus) {
		allocationStatus.StoragePool.Id = sp.Id
		allocationStatus.StoragePool.Status = nnfv1alpha1.ResourceStatus(sp.Status)
		allocationStatus.StoragePool.Health = nnfv1alpha1.ResourceHealth(sp.Status)
		allocationStatus.CapacityAllocated = sp.CapacityBytes
	})

	log.Info("Created storage pool", "Id", sp.Id)

	return &ctrl.Result{}, nil
}

func (r *NnfNodeStorageReconciler) createBlockDevice(statusUpdater *nodeStorageStatusUpdater, nodeStorage *nnfv1alpha1.NnfNodeStorage, index int) (*ctrl.Result, error) {
	log := r.Log.WithValues("NodeStorage", types.NamespacedName{Name: nodeStorage.Name, Namespace: nodeStorage.Namespace})
	ss := nnf.NewDefaultStorageService()

	allocationStatus := &nodeStorage.Status.Allocations[index]

	// Create a Storage Group if none is currently present. Recall that a Storage Group
	// is a mapping from the Storage Pool to a Server Endpoint. Establishing a Storage
	// Group makes block storage available on the server, which itself is a precursor to
	// any file system built on top of the block storage.
	if len(allocationStatus.StorageGroup.Id) > 0 {
		sg, err := r.getStorageGroup(ss, allocationStatus.StorageGroup.Id)
		if err != nil {
			statusUpdater.update(func(*nnfv1alpha1.NnfNodeStorageStatus) {
				nnfv1alpha1.SetGetResourceFailureCondition(allocationStatus.Conditions, err)
			})

			return &ctrl.Result{}, nil
		}

		equal := statusUpdater.updateWithEqual(func(*nnfv1alpha1.NnfNodeStorageStatus) {
			allocationStatus.StorageGroup.Status = nnfv1alpha1.ResourceStatus(sg.Status)
			allocationStatus.StorageGroup.Health = nnfv1alpha1.ResourceHealth(sg.Status)
		})

		if !equal {
			return &ctrl.Result{}, nil
		}

		return nil, nil
	}

	condition := &allocationStatus.Conditions[nnfv1alpha1.ConditionIndexCreateStorageGroup]

	statusUpdater.update(func(*nnfv1alpha1.NnfNodeStorageStatus) {
		condition.LastTransitionTime = metav1.Now()
		condition.Status = metav1.ConditionTrue
	})

	sg, err := r.createStorageGroup(ss, allocationStatus.StoragePool.Id, os.Getenv("RABBIT_NODE"))
	if err != nil {
		statusUpdater.updateError(condition, &allocationStatus.StorageGroup, err)

		return &ctrl.Result{Requeue: true}, nil
	}

	statusUpdater.updateSuccess(condition, func(*nnfv1alpha1.NnfNodeStorageStatus) {
		allocationStatus.StorageGroup.Id = sg.Id
		allocationStatus.StorageGroup.Status = nnfv1alpha1.ResourceStatus(sg.Status)
		allocationStatus.StorageGroup.Health = nnfv1alpha1.ResourceHealth(sg.Status)
	})

	log.Info("Created storage group", "Id", sg.Id)

	return &ctrl.Result{}, nil
}

func (r *NnfNodeStorageReconciler) formatFileSystem(statusUpdater *nodeStorageStatusUpdater, nodeStorage *nnfv1alpha1.NnfNodeStorage, index int) (*ctrl.Result, error) {
	log := r.Log.WithValues("NodeStorage", types.NamespacedName{Name: nodeStorage.Name, Namespace: nodeStorage.Namespace})
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

	if len(allocationStatus.FileSystem.Id) == 0 {
		condition := &allocationStatus.Conditions[nnfv1alpha1.ConditionIndexCreateFileSystem]

		statusUpdater.update(func(*nnfv1alpha1.NnfNodeStorageStatus) {
			condition.Status = metav1.ConditionTrue
			condition.LastTransitionTime = metav1.Now()
		})

		oem := nnfserver.FileSystemOem{
			Type: nodeStorage.Spec.FileSystemType,

			// If not lustre, then these will be appropriate zero values.
			Name:       nodeStorage.Spec.LustreStorage.FileSystemName,
			Index:      nodeStorage.Spec.LustreStorage.StartIndex + index,
			MgsNode:    nodeStorage.Spec.LustreStorage.MgsNode,
			TargetType: nodeStorage.Spec.LustreStorage.TargetType,
			BackFs:     nodeStorage.Spec.LustreStorage.BackFs,
		}

		fs, err := r.createFileSystem(ss, allocationStatus.StoragePool.Id, oem)
		if err != nil {
			statusUpdater.updateError(condition, &allocationStatus.FileSystem, err)

			return &ctrl.Result{Requeue: true}, nil
		}

		statusUpdater.updateSuccess(condition, func(*nnfv1alpha1.NnfNodeStorageStatus) {
			allocationStatus.FileSystem.Id = fs.Id
			allocationStatus.FileSystem.Status = nnfv1alpha1.ResourceReady
			allocationStatus.FileSystem.Health = nnfv1alpha1.ResourceOkay
		})

		log.Info("Created filesystem", "Id", fs.Id)

		return &ctrl.Result{}, nil
	}

	if len(allocationStatus.FileShare.Id) == 0 {
		condition := &allocationStatus.Conditions[nnfv1alpha1.ConditionIndexCreateFileShare]

		statusUpdater.update(func(*nnfv1alpha1.NnfNodeStorageStatus) {
			condition.Status = metav1.ConditionTrue
			condition.LastTransitionTime = metav1.Now()
		})

		mountPath := ""
		if nodeStorage.Spec.FileSystemType == "lustre" {
			targetIndex := nodeStorage.Spec.LustreStorage.StartIndex + index
			mountPath = "/mnt/lustre/" + nodeStorage.Spec.LustreStorage.FileSystemName + "/" + nodeStorage.Spec.LustreStorage.TargetType + strconv.Itoa(targetIndex)
		}

		sh, err := r.createFileShare(ss, allocationStatus.FileSystem.Id, os.Getenv("RABBIT_NODE"), mountPath)
		if err != nil {
			statusUpdater.updateError(condition, &allocationStatus.FileShare, err)

			return &ctrl.Result{Requeue: true}, nil
		}

		statusUpdater.updateSuccess(condition, func(*nnfv1alpha1.NnfNodeStorageStatus) {
			allocationStatus.FileShare.Id = sh.Id
			allocationStatus.FileShare.Status = nnfv1alpha1.ResourceStatus(sh.Status)
			allocationStatus.FileShare.Health = nnfv1alpha1.ResourceHealth(sh.Status)
		})

		log.Info("Created file share", "Id", sh.Id)

		return &ctrl.Result{}, nil
	}

	sh, err := r.getFileShare(ss, allocationStatus.FileSystem.Id, allocationStatus.FileShare.Id)
	if err != nil {
		statusUpdater.update(func(*nnfv1alpha1.NnfNodeStorageStatus) {
			nnfv1alpha1.SetGetResourceFailureCondition(allocationStatus.Conditions, err)
		})

		return &ctrl.Result{}, nil
	}

	nid := ""
	if nidRaw, present := sh.Oem["NID"]; present {
		nid = nidRaw.(string)
	}

	// LNid information isn't passed up from SF right now. See bug RABSW-521.
	// Fake out an MGT LNid for now so we can pass it to the MDTs and OSTs.
	// TODO: Get the real LNid
	if nodeStorage.Spec.LustreStorage.TargetType == "MGT" {
		nid = "rabbit-01@tcp0"
	}

	equal := statusUpdater.updateWithEqual(func(*nnfv1alpha1.NnfNodeStorageStatus) {
		allocationStatus.FileShare.Status = nnfv1alpha1.ResourceStatus(sh.Status)
		allocationStatus.FileShare.Health = nnfv1alpha1.ResourceHealth(sh.Status)
		nodeStorage.Status.LustreStorage.Nid = nid
	})

	if !equal {
		return &ctrl.Result{}, nil
	}

	return nil, nil
}

func (r *NnfNodeStorageReconciler) deleteStorage(statusUpdater *nodeStorageStatusUpdater, nodeStorage *nnfv1alpha1.NnfNodeStorage, index int) (*ctrl.Result, error) {
	log := r.Log.WithValues("NodeStorage", types.NamespacedName{Name: nodeStorage.Name, Namespace: nodeStorage.Namespace})
	ss := nnf.NewDefaultStorageService()

	allocationStatus := &nodeStorage.Status.Allocations[index]
	if allocationStatus.StoragePool.Id == "" {
		return nil, nil
	}

	condition := &allocationStatus.Conditions[nnfv1alpha1.ConditionIndexDeleteStoragePool]

	statusUpdater.update(func(*nnfv1alpha1.NnfNodeStorageStatus) {
		condition.Status = metav1.ConditionTrue
		condition.LastTransitionTime = metav1.Now()
	})

	log.Info("Deleting storage pool", "Id", allocationStatus.StoragePool.Id)

	err := r.deleteStoragePool(ss, allocationStatus.StoragePool.Id)
	if err != nil {
		statusUpdater.updateError(condition, &allocationStatus.FileShare, err)

		return &ctrl.Result{Requeue: true}, nil
	}

	statusUpdater.update(func(*nnfv1alpha1.NnfNodeStorageStatus) {
		allocationStatus.StoragePool.Id = ""
		allocationStatus.StorageGroup.Id = ""
		allocationStatus.FileSystem.Id = ""
		allocationStatus.FileShare.Id = ""
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

	if nodeStorage.Spec.LustreStorage.TargetType == "MGT" {
		return true
	}

	if len(nodeStorage.Spec.LustreStorage.MgsNode) > 0 {
		return true
	}

	return false
}

func (r *NnfNodeStorageReconciler) createStoragePool(ss nnf.StorageServiceApi, capacity int64) (*sf.StoragePoolV150StoragePool, error) {
	sp := &sf.StoragePoolV150StoragePool{
		CapacityBytes: capacity,
		Oem: openapi.MarshalOem(nnf.AllocationPolicyOem{
			Policy:     nnf.SpareAllocationPolicyType,
			Compliance: nnf.RelaxedAllocationComplianceType,
		}),
	}

	if err := ss.StorageServiceIdStoragePoolsPost(ss.Id(), sp); err != nil {
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

func (r *NnfNodeStorageReconciler) createStorageGroup(ss nnf.StorageServiceApi, spID string, epID string) (*sf.StorageGroupV150StorageGroup, error) {
	sp, err := r.getStoragePool(ss, spID)
	if err != nil {
		return nil, err
	}

	ep, err := r.getEndpoint(ss, epID)
	if err != nil {
		return nil, err
	}

	sg := &sf.StorageGroupV150StorageGroup{
		Links: sf.StorageGroupV150Links{
			StoragePool:    sf.OdataV4IdRef{OdataId: sp.OdataId},
			ServerEndpoint: sf.OdataV4IdRef{OdataId: ep.OdataId},
		},
	}

	if err := ss.StorageServiceIdStorageGroupPost(ss.Id(), sg); err != nil {
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

func (r *NnfNodeStorageReconciler) createFileShare(ss nnf.StorageServiceApi, fsID string, epID string, mountPath string) (*sf.FileShareV120FileShare, error) {
	fs, err := r.getFileSystem(ss, fsID)
	if err != nil {
		return nil, err
	}

	ep, err := r.getEndpoint(ss, epID)
	if err != nil {
		return nil, err
	}

	sh := &sf.FileShareV120FileShare{
		FileSharePath: mountPath,
		Links: sf.FileShareV120Links{
			FileSystem: sf.OdataV4IdRef{OdataId: fs.OdataId},
			Endpoint:   sf.OdataV4IdRef{OdataId: ep.OdataId},
		},
	}

	if err := ss.StorageServiceIdFileSystemIdExportedSharesPost(ss.Id(), fs.Id, sh); err != nil {
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

func (r *NnfNodeStorageReconciler) createFileSystem(ss nnf.StorageServiceApi, spID string, oem nnfserver.FileSystemOem) (*sf.FileSystemV122FileSystem, error) {
	sp, err := r.getStoragePool(ss, spID)
	if err != nil {
		return nil, err
	}

	fs := &sf.FileSystemV122FileSystem{
		Links: sf.FileSystemV122Links{
			StoragePool: sf.OdataV4IdRef{OdataId: sp.OdataId},
		},
		Oem: openapi.MarshalOem(oem),
	}

	if err := ss.StorageServiceIdFileSystemsPost(ss.Id(), fs); err != nil {
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

func (s *nodeStorageStatusUpdater) updateWithEqual(update func(*nnfv1alpha1.NnfNodeStorageStatus)) bool {
	update(&s.storage.Status)
	if reflect.DeepEqual(s.storage.Status, s.existingStatus) {
		return true
	}

	return false
}

func (s *nodeStorageStatusUpdater) updateError(condition *metav1.Condition, status *nnfv1alpha1.NnfResourceStatus, err error) {
	status.Status = nnfv1alpha1.ResourceFailed
	condition.Reason = nnfv1alpha1.ConditionFailed
	condition.Message = err.Error()
}

func (s *nodeStorageStatusUpdater) updateSuccess(condition *metav1.Condition, update func(*nnfv1alpha1.NnfNodeStorageStatus)) {
	condition.Status = metav1.ConditionFalse
	condition.Reason = nnfv1alpha1.ConditionSuccess

	update(&s.storage.Status)
}

func (s *nodeStorageStatusUpdater) close(ctx context.Context, r *NnfNodeStorageReconciler) error {
	if !reflect.DeepEqual(s.storage.Status, s.existingStatus) {
		return r.Update(ctx, s.storage)
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NnfNodeStorageReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&nnfv1alpha1.NnfNodeStorage{}).
		Complete(r)
}
