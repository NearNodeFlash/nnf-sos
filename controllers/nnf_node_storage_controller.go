/*
Copyright 2021 Hewlett Packard Enterprise Development LP
*/

package controllers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	nnf "stash.us.cray.com/rabsw/nnf-ec/pkg/manager-nnf"
	nnfserver "stash.us.cray.com/rabsw/nnf-ec/pkg/manager-server"

	openapi "stash.us.cray.com/rabsw/rfsf-openapi/pkg/common"
	sf "stash.us.cray.com/rabsw/rfsf-openapi/pkg/models"

	nnfv1alpha1 "stash.us.cray.com/RABSW/nnf-sos/api/v1alpha1"
)

const (
	NnfNodeStorageResourceName = "nnf-node-storage"
)

type NnfNodeStorageReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme

	types.NamespacedName
}

//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfnodestorages,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfnodestorages/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfnodestorages/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.2/pkg/reconcile
func (r *NnfNodeStorageReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, err error) {
	log := r.Log.WithValues("Storage", req.NamespacedName)

	storage := &nnfv1alpha1.NnfNodeStorage{}
	if err := r.Get(ctx, req.NamespacedName, storage); err != nil {
		log.Error(err, "Failed to get node storage")
		return ctrl.Result{}, err
	}

	// Connect to the NNF Storage Service. This permits control over NNF resources
	// on this particular NNF Node. All NNF CRUD operations are over this service.
	ss := nnf.NewDefaultStorageService()

	spec := &storage.Spec
	status := &storage.Status

	// Use the Node Storage Status Updater to track updates to the storage status.
	// This ensures that only one call to r.Status().Update() is even though we
	// update the status at several points in the process. We hijack the defer logic
	// to perform the status update if no other error is present in the system when
	// exiting this reconcile function. Note that "err" is the name returned value,
	// so when we would normally call "return ctrl.Result{}, nil", at that time
	// "err" is nil - and if permitted we will update err with the result of
	// the r.Status().Update()
	statusUpdater := NewNodeStorageStatusUpdater(storage)
	defer func() {
		if err == nil {
			if err = statusUpdater.Close(r, ctx); err != nil {
				r.Log.Info(fmt.Sprintf("Failed to update status with error %s", err))
			}
		} else {
			r.Log.Info(fmt.Sprintf("err before defer begins, %s", err))
		}
	}()

	// Check if the object is being deleted. Deletion is carefully coordinated around
	// the NNF resources being managed by this NNF Node Storage resource. For a
	// successful deletion, the NNF Storage Pool must be deleted. Deletion of the
	// Storage Pool handles the entire sub-tree of NNF resources (Storage Groups,
	// File System, and File Shares). The Finalizer on this NNF Node Storage resource
	// is present until the underling NNF resources are deleted through the
	// storage service.
	if !storage.GetDeletionTimestamp().IsZero() {

		if !controllerutil.ContainsFinalizer(storage, finalizer) {
			return ctrl.Result{}, nil
		}

		log.Info("Deleting storage...")

		if status.Status != nnfv1alpha1.ResourceDeleted {
			if len(status.Id) != 0 {
				condition := &status.Conditions[nnfv1alpha1.ConditionIndexDeleteStoragePool]

				statusUpdater.Update(func(*nnfv1alpha1.NnfNodeStorageStatus) {
					status.Status = nnfv1alpha1.ResourceDeleting
					condition.Status = metav1.ConditionTrue
					condition.LastTransitionTime = metav1.Now()
				})

				if err := ss.StorageServiceIdStoragePoolIdDelete(ss.Id(), status.Id); err != nil {
					statusUpdater.Update(func(*nnfv1alpha1.NnfNodeStorageStatus) {

						condition.Reason = nnfv1alpha1.ConditionFailed
						condition.Message = err.Error()
					})

					return ctrl.Result{RequeueAfter: time.Second}, nil
				}

				statusUpdater.Update(func(*nnfv1alpha1.NnfNodeStorageStatus) {
					status.Id = ""
					status.Status = nnfv1alpha1.ResourceDeleted
				})

				return ctrl.Result{Requeue: true}, nil
			}
		}

		controllerutil.RemoveFinalizer(storage, finalizer)
		if err := r.Update(ctx, storage); err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	// First time setup requires programming of the storage status such that the resource
	// is labeled as "Starting" and all Conditions are initializaed. After this is done,
	// the resource obtains a finalizer to manage the resource lifetime.
	if !controllerutil.ContainsFinalizer(storage, finalizer) {

		if status.Status != nnfv1alpha1.ResourceStarting {

			statusUpdater.Update(func(*nnfv1alpha1.NnfNodeStorageStatus) {
				creationTime := metav1.Now()
				status.Status = nnfv1alpha1.ResourceStarting
				status.CreationTime = &creationTime
				status.Conditions = nnfv1alpha1.NewConditions()

				status.Servers = make([]nnfv1alpha1.NnfNodeStorageServerStatus, len(spec.Servers))
				for serverIdx := range spec.Servers {
					serverSpec := &spec.Servers[serverIdx]
					serverStatus := &status.Servers[serverIdx]

					serverStatus.Status = nnfv1alpha1.ResourceStarting
					serverStatus.Id = serverSpec.Id
					serverStatus.Name = serverSpec.Name
				}
			})

			return ctrl.Result{Requeue: true}, nil
		}

		controllerutil.AddFinalizer(storage, finalizer)
		if err := r.Update(ctx, storage); err != nil {
			return ctrl.Result{}, nil
		}

		// Requeue so we can track further changes to the object
		return ctrl.Result{Requeue: true}, nil
	}

	// At this point we have a valid request for storage and the desired
	// servers with access to the storage. Work to reconcile the request.

	// A failed storage status is not recoverable. Stay in the failed state
	// until the resource is deleted.
	if status.Status == nnfv1alpha1.ResourceFailed {
		return ctrl.Result{}, nil
	}

	if len(status.Id) == 0 {

		condition := &storage.Status.Conditions[nnfv1alpha1.ConditionIndexCreateStoragePool]

		statusUpdater.Update(func(*nnfv1alpha1.NnfNodeStorageStatus) {
			condition.Status = metav1.ConditionTrue
			condition.LastTransitionTime = metav1.Now()
		})

		sp := &sf.StoragePoolV150StoragePool{
			CapacityBytes: storage.Spec.Capacity,
			Oem: openapi.MarshalOem(nnf.AllocationPolicyOem{
				Policy:     nnf.SpareAllocationPolicyType,
				Compliance: nnf.RelaxedAllocationComplianceType,
			}),
		}

		if err := ss.StorageServiceIdStoragePoolsPost(ss.Id(), sp); err != nil {

			statusUpdater.Update(func(*nnfv1alpha1.NnfNodeStorageStatus) {
				status.Status = nnfv1alpha1.ResourceFailed

				condition.Reason = nnfv1alpha1.ConditionFailed
				condition.Message = err.Error()
			})

			return ctrl.Result{}, nil
		}

		statusUpdater.Update(func(*nnfv1alpha1.NnfNodeStorageStatus) {
			status.Id = sp.Id
			status.Status = nnfv1alpha1.ResourceStatus(sp.Status)
			status.Health = nnfv1alpha1.ResourceHealth(sp.Status)
			status.CapacityAllocated = sp.CapacityBytes

			condition.Status = metav1.ConditionFalse
			condition.Reason = nnfv1alpha1.ConditionSuccess
		})

		return ctrl.Result{Requeue: true}, nil
	}

	sp := sf.StoragePoolV150StoragePool{}

	if err := ss.StorageServiceIdStoragePoolIdGet(ss.Id(), storage.Status.Id, &sp); err != nil {
		statusUpdater.Update(func(*nnfv1alpha1.NnfNodeStorageStatus) {
			nnfv1alpha1.SetGetResourceFailureCondition(status.Conditions, err)
		})

		return ctrl.Result{}, nil
	}

	if status.Status != nnfv1alpha1.ResourceStatus(sp.Status) || status.Health != nnfv1alpha1.ResourceHealth(sp.Status) {
		statusUpdater.Update(func(*nnfv1alpha1.NnfNodeStorageStatus) {
			status.Status = nnfv1alpha1.ResourceStatus(sp.Status)
			status.Health = nnfv1alpha1.ResourceHealth(sp.Status)
		})

		return ctrl.Result{Requeue: true}, nil
	}

	// Check for the requested file system defined in the specification. If found,
	// ensure a file system exists for the storage pool, either by creating one
	// or retrieving the existing file system.
	fs := &sf.FileSystemV122FileSystem{}

	if len(spec.FileSystemName) != 0 {
		if len(sp.Links.FileSystem.OdataId) == 0 {

			condition := &status.Conditions[nnfv1alpha1.ConditionIndexCreateFileSystem]

			statusUpdater.Update(func(*nnfv1alpha1.NnfNodeStorageStatus) {
				condition.Status = metav1.ConditionTrue
				condition.LastTransitionTime = metav1.Now()
			})

			oem := nnfserver.FileSystemOem{
				Name: spec.FileSystemName,
				Type: spec.FileSystemType,
				// If not lustre, then these will be appropriate zero values.
				//Index:      spec.LustreStorage.Index,
				MgsNode:    spec.LustreStorage.MgsNode,
				TargetType: spec.LustreStorage.TargetType,
			}
			fs = &sf.FileSystemV122FileSystem{
				Links: sf.FileSystemV122Links{
					StoragePool: sf.OdataV4IdRef{OdataId: sp.OdataId},
				},
				Oem: openapi.MarshalOem(oem),
			}

			log.Info("FileSystemsPost", "fs", fs)

			if err := ss.StorageServiceIdFileSystemsPost(ss.Id(), fs); err != nil {
				log.Info("FileSystemsPost", "err", err.Error())
				statusUpdater.Update(func(*nnfv1alpha1.NnfNodeStorageStatus) {
					status.Status = nnfv1alpha1.ResourceFailed

					condition.Reason = nnfv1alpha1.ConditionFailed
					condition.Message = err.Error()
				})

				return ctrl.Result{}, nil
			}

			statusUpdater.Update(func(*nnfv1alpha1.NnfNodeStorageStatus) {
				condition.Status = metav1.ConditionFalse
				condition.Reason = nnfv1alpha1.ConditionSuccess
			})

			return ctrl.Result{Requeue: true}, nil
		} else {

			fsid := sp.Links.FileSystem.OdataId[strings.LastIndex(sp.Links.FileSystem.OdataId, "/")+1:]

			if err := ss.StorageServiceIdFileSystemIdGet(ss.Id(), fsid, fs); err != nil {
				statusUpdater.Update(func(*nnfv1alpha1.NnfNodeStorageStatus) {
					status.Status = nnfv1alpha1.ResourceFailed
					nnfv1alpha1.SetGetResourceFailureCondition(status.Conditions, err)
				})

				return ctrl.Result{}, nil
			}
		}
	} // if len(spec.FileSystem) != 0

	// Next, we wish to establish storage groups, which is a mapping from the storage
	// pool to a particular server - This is taking a number of NVMe namespaces spread
	// across the drives and attaching them to the NVMe Controller that represents the
	// server.
	for serverIdx := range spec.Servers {
		serverSpec := &spec.Servers[serverIdx]
		serverStatus := &status.Servers[serverIdx]

		// Make sure the client hasn't rearranged the server spec and status fields.
		if serverSpec.Id != serverStatus.Id {
			statusUpdater.Update(func(*nnfv1alpha1.NnfNodeStorageStatus) {
				status.Status = nnfv1alpha1.ResourceInvalid
				nnfv1alpha1.SetResourceInvalidCondition(
					status.Conditions, fmt.Errorf("Server Id %s does not match Server Status Id %s", serverSpec.Id, serverStatus.Id),
				)
			})

			return ctrl.Result{}, nil
		}

		// Retrieve the server endpoint to ensure it is up and available for storage
		ep := &sf.EndpointV150Endpoint{}

		if err := ss.StorageServiceIdEndpointIdGet(ss.Id(), serverSpec.Id, ep); err != nil {
			// TODO: We should differentiate between a bad request (bad server id/name) and a failed
			// request (could not get status)

			statusUpdater.Update(func(*nnfv1alpha1.NnfNodeStorageStatus) {
				nnfv1alpha1.SetGetResourceFailureCondition(status.Conditions, err)
			})

			return ctrl.Result{}, nil
		}

		sgid := serverStatus.StorageGroup.Id

		// Create a Storage Group if none is currently present. Recall that a Storage Group
		// is a mapping from the Storage Pool to a Server Endpoint. Establishing a Storage
		// Group makes block storage available on the server, which itself is a precursor to
		// any file system built on top of the block storage.
		if len(sgid) == 0 {

			condition := &status.Conditions[nnfv1alpha1.ConditionIndexCreateStorageGroup]

			statusUpdater.Update(func(*nnfv1alpha1.NnfNodeStorageStatus) {
				condition.Status = metav1.ConditionTrue
				condition.ObservedGeneration = int64(serverIdx)
				condition.LastTransitionTime = metav1.Now()
			})

			sg := &sf.StorageGroupV150StorageGroup{
				Links: sf.StorageGroupV150Links{
					StoragePool:    sf.OdataV4IdRef{OdataId: sp.OdataId},
					ServerEndpoint: sf.OdataV4IdRef{OdataId: ep.OdataId},
				},
			}

			if err := ss.StorageServiceIdStorageGroupPost(ss.Id(), sg); err != nil {
				statusUpdater.Update(func(*nnfv1alpha1.NnfNodeStorageStatus) {
					status.Status = nnfv1alpha1.ResourceFailed
					serverStatus.Status = nnfv1alpha1.ResourceFailed

					condition.Reason = nnfv1alpha1.ResourceFailed
					condition.Message = err.Error()
				})

				return ctrl.Result{}, nil
			}

			statusUpdater.Update(func(*nnfv1alpha1.NnfNodeStorageStatus) {
				serverStatus.StorageGroup.Id = sg.Id
				serverStatus.StorageGroup.Status = nnfv1alpha1.ResourceStatus(sg.Status)
				serverStatus.StorageGroup.Health = nnfv1alpha1.ResourceHealth(sg.Status)

				condition.Status = metav1.ConditionFalse
				condition.Reason = nnfv1alpha1.ConditionSuccess
			})

			return ctrl.Result{Requeue: true}, nil
		} else { // if len(sgid) == 0

			// For existing storage groups, we refresh the status to ensure everything is
			// operational.
			sg := &sf.StorageGroupV150StorageGroup{}

			if err := ss.StorageServiceIdStorageGroupIdGet(ss.Id(), sgid, sg); err != nil {
				statusUpdater.Update(func(*nnfv1alpha1.NnfNodeStorageStatus) {
					nnfv1alpha1.SetGetResourceFailureCondition(status.Conditions, err)
				})

				return ctrl.Result{}, nil
			}

			if serverStatus.StorageGroup.Status != nnfv1alpha1.ResourceStatus(sg.Status) || serverStatus.StorageGroup.Health != nnfv1alpha1.ResourceHealth(sg.Status) {
				statusUpdater.Update(func(*nnfv1alpha1.NnfNodeStorageStatus) {
					serverStatus.StorageGroup.Status = nnfv1alpha1.ResourceStatus(sg.Status)
					serverStatus.StorageGroup.Health = nnfv1alpha1.ResourceHealth(sg.Status)
				})

				return ctrl.Result{Requeue: true}, nil
			}
		}

		// Check if the specification defines a path for this server resource. A valid path
		// is equivalent to establishing a File Share - which consists of the Server Endpoint
		// and the File System. Creating a File Share can be seen as taking the File System that
		// is paired to the Storage Pool, and performing the necessary calls to make the
		// File System accessible from the Server Endpoint.
		if len(serverSpec.Path) != 0 {

			shid := serverStatus.FileShare.Id

			if len(shid) == 0 {

				condition := &status.Conditions[nnfv1alpha1.ConditionIndexCreateFileShare]

				statusUpdater.Update(func(*nnfv1alpha1.NnfNodeStorageStatus) {
					condition.Status = metav1.ConditionTrue
					condition.ObservedGeneration = int64(serverIdx)
					condition.LastTransitionTime = metav1.Now()
				})

				sh := &sf.FileShareV120FileShare{
					FileSharePath: serverSpec.Path,
					Links: sf.FileShareV120Links{
						FileSystem: sf.OdataV4IdRef{OdataId: fs.OdataId},
						Endpoint:   sf.OdataV4IdRef{OdataId: ep.OdataId},
					},
				}

				log.Info("ExportedSharesPost", "idx", serverIdx, "sh", sh)

				if err := ss.StorageServiceIdFileSystemIdExportedSharesPost(ss.Id(), fs.Id, sh); err != nil {
					log.Info("ExportedSharesPost", "idx", serverIdx, "err", err.Error())
					status.Status = nnfv1alpha1.ResourceFailed
					serverStatus.Status = nnfv1alpha1.ResourceFailed

					condition.Reason = nnfv1alpha1.ConditionFailed
					condition.Message = err.Error()

					return ctrl.Result{}, nil
				}

				statusUpdater.Update(func(*nnfv1alpha1.NnfNodeStorageStatus) {
					serverStatus.FileShare.Id = sh.Id
					serverStatus.FileShare.Status = nnfv1alpha1.ResourceStatus(sh.Status)
					serverStatus.FileShare.Health = nnfv1alpha1.ResourceHealth(sh.Status)

					condition.Status = metav1.ConditionFalse
					condition.Reason = nnfv1alpha1.ConditionSuccess
				})

				return ctrl.Result{Requeue: true}, nil
			} else { // if len(shid) == 0

				sh := &sf.FileShareV120FileShare{}

				if err := ss.StorageServiceIdFileSystemIdExportedShareIdGet(ss.Id(), fs.Id, shid, sh); err != nil {
					statusUpdater.Update(func(*nnfv1alpha1.NnfNodeStorageStatus) {
						serverStatus.Status = nnfv1alpha1.ResourceFailed
						nnfv1alpha1.SetGetResourceFailureCondition(status.Conditions, err)
					})

					return ctrl.Result{}, nil
				}

				if serverStatus.FileShare.Status != nnfv1alpha1.ResourceStatus(sh.Status) || serverStatus.FileShare.Health != nnfv1alpha1.ResourceHealth(sh.Status) {
					statusUpdater.Update(func(*nnfv1alpha1.NnfNodeStorageStatus) {
						serverStatus.FileShare.Status = nnfv1alpha1.ResourceStatus(sh.Status)
						serverStatus.FileShare.Health = nnfv1alpha1.ResourceHealth(sh.Status)
					})

					return ctrl.Result{Requeue: true}, nil
				}

				if nidRaw, present := sh.Oem["NID"]; present {
					nid := nidRaw.(string)
					if status.LustreStorage.Nid != nid {
						statusUpdater.Update(func(*nnfv1alpha1.NnfNodeStorageStatus) {
							status.LustreStorage.Nid = nid
						})

						return ctrl.Result{Requeue: true}, nil
					}
				}
			}
		} // if len(serverSpec.Path) != 0

		// At this stage, the Server sub-resource is fully initialized. Reflect this in the
		// server status
		if serverStatus.Status == nnfv1alpha1.ResourceStarting {
			statusUpdater.Update(func(status *nnfv1alpha1.NnfNodeStorageStatus) {
				serverStatus.Status = nnfv1alpha1.ResourceReady
			})

			return ctrl.Result{Requeue: true}, nil
		}

	} // for serverIdx := range spec.Servers

	// At this stage, the Storage resource is fully initialized. Reflect this in the
	// Storage status
	if status.Status == nnfv1alpha1.ResourceStarting {
		statusUpdater.Update(func(status *nnfv1alpha1.NnfNodeStorageStatus) {
			status.Status = nnfv1alpha1.ResourceReady
		})
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NnfNodeStorageReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&nnfv1alpha1.NnfNodeStorage{}).
		Complete(r)
}

type nodeStorageStatusUpdater struct {
	storage     *nnfv1alpha1.NnfNodeStorage
	needsUpdate bool
}

func NewNodeStorageStatusUpdater(s *nnfv1alpha1.NnfNodeStorage) *nodeStorageStatusUpdater {
	return &nodeStorageStatusUpdater{
		storage:     s,
		needsUpdate: false,
	}
}

func (s *nodeStorageStatusUpdater) Update(update func(*nnfv1alpha1.NnfNodeStorageStatus)) {
	update(&s.storage.Status)
	s.needsUpdate = true
}

func (s *nodeStorageStatusUpdater) Close(r *NnfNodeStorageReconciler, ctx context.Context) error {
	defer func() { s.needsUpdate = false }()
	if s.needsUpdate {
		return r.Status().Update(ctx, s.storage)
	}
	return nil
}
