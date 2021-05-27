/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	nnf "stash.us.cray.com/rabsw/nnf-ec/pkg"
	sf "stash.us.cray.com/rabsw/rfsf-openapi/pkg/models"

	nnfv1alpha1 "stash.us.cray.com/RABSW/nnf-sos/api/v1alpha1"
)

// NnfStorageReconciler reconciles a Storage object
type NnfStorageReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// Finalizer defines the key used in identifying the storage object as
// being owned by this NNF Storage Reconciler. This presents the system
// from deleting the custom resource until the reconciler has finished
// in using the resource.
const (
	finalizer = "nnf.cray.com/finalizer"
)

//+kubebuilder:rbac:groups=nnf.cray.com,namespace=nnf-system,resources=nnfstorages,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=nnf.cray.com,namespace=nnf-system,resources=nnfstorages/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=nnf.cray.com,namespace=nnf-system,resources=nnfstorages/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Storage object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.2/pkg/reconcile
func (r *NnfStorageReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("storage", req.NamespacedName)

	storage := &nnfv1alpha1.NnfStorage{}
	if err := r.Get(ctx, req.NamespacedName, storage); err != nil {
		log.Error(err, "Storage not found", "Request.NamespacedName", req.NamespacedName)
		return ctrl.Result{}, err
	}

	// Check if the object is being deleted
	if !storage.GetDeletionTimestamp().IsZero() {
		log.Info("Deleting storage...", "Storage.NamespacedName", types.NamespacedName{Name: storage.Name, Namespace: storage.Namespace})

		if !controllerutil.ContainsFinalizer(storage, finalizer) {
			return ctrl.Result{}, nil
		}

		// TODO: Delete the storage instance

		log.Info("Finalizing storage", "Storage.NamespacedName", types.NamespacedName{Name: storage.Name, Namespace: storage.Namespace})
		controllerutil.RemoveFinalizer(storage, finalizer)
		if err := r.Update(ctx, storage); err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	if !controllerutil.ContainsFinalizer(storage, finalizer) {

		controllerutil.AddFinalizer(storage, finalizer)
		if err := r.Update(ctx, storage); err != nil {
			return ctrl.Result{}, err
		}

		// Initialze the Status' of the storage
		storage.Status.Nodes = make([]nnfv1alpha1.NnfStorageNodeStatus, len(storage.Spec.Nodes))
		for nodeIdx := range storage.Status.Nodes {
			spec := &storage.Spec.Nodes[nodeIdx]
			status := &storage.Status.Nodes[nodeIdx]

			status.State = "Starting"
			status.Servers = make([]nnfv1alpha1.NnfStorageServerStatus, len(spec.Servers))
			for serverIdx := range spec.Servers {
				serverStatus := &status.Servers[serverIdx]
				serverStatus.State = "Starting"
				spec.Servers[serverIdx].DeepCopyInto(&serverStatus.NnfStorageServerSpec)
			}
		}

		log.Info("Initialized storage status", "NnfStorage", fmt.Sprintf("%+v", storage))

		if err := r.Status().Update(ctx, storage); err != nil {
			return ctrl.Result{}, err
		}
	}

	// At this point we've received a request to provision storage based on the list
	// of NNF Node Controllers supplied in the specification. For each NNF Node Controller,
	// we connect to the controller and allocate the required storage.
	for nodeIdx := range storage.Status.Nodes {

		node := &storage.Status.Nodes[nodeIdx]
		nodeName := storage.Spec.Nodes[nodeIdx].Name

		log.Info("Initializing storage on node", "Node.Name", nodeName)

		addr := fmt.Sprintf("%s.%s", ServiceName(nodeName), storage.Namespace)
		port := fmt.Sprintf("%d", nnf.Port)

		log.Info("Connecting to storage service", "StorageService.Addr", addr, "StorageService.Port", port)
		conn, err := nnf.NewStorageServiceConnection(addr, port)
		if conn == nil {
			log.Error(err, "Failed to connect to storage service", "StorageService.Addr", addr, "StorageService.Port", port)
			return ctrl.Result{}, err
		}

		if len(node.Id) == 0 {
			// Request a storage pool is created with the desired capacity
			log.Info("Creating storage pool...")
			pool, err := conn.CreateStoragePool(storage.Spec.Capacity)
			if err != nil {
				log.Error(err, "Failed to create storage pool")

				// TODO: If the storage pool cannot be created because of inadaquate resources, we
				// should reflect that the status, not continuously return an error.
				return ctrl.Result{}, errors.NewBadRequest(err.Error())
			}

			log.Info("Created storage pool", "StoragePool.Id", pool.Id)

			node.Id = pool.Id
			if err := r.Status().Update(ctx, storage); err != nil {
				log.Error(err, "Storage status update failed")
				return ctrl.Result{}, err
			}
		}

		pool, err := conn.GetStoragePool(node.Id)
		if err != nil {
			log.Error(err, "Failed to retrieve storage pool", "StoragePool.Id", node.Id)
			return ctrl.Result{}, err
		}

		log.Info("Retrieved storage pool", "StoragePool.Id", pool.Id, "StoragePool.Status.Health", string(pool.Status.Health), "StoragePool.Status.State", string(pool.Status.State))
		if node.Health != string(pool.Status.Health) || node.State != string(pool.Status.State) {
			node.Health = string(pool.Status.Health)
			node.State = string(pool.Status.State)
			r.Status().Update(ctx, storage)
		}

		// Check if a file system is defined
		var fs *sf.FileSystemV122FileSystem = nil
		if len(storage.Spec.FileSystem) != 0 {

			if len(pool.Links.FileSystem.OdataId) == 0 {
				log.Info("Creating file system...", "StoragePool.Id", pool.Id, "FileSystem", storage.Spec.FileSystem)
				fs, err = conn.CreateFileSystem(pool, storage.Spec.FileSystem)
				if err != nil {
					log.Error(err, "Failed to create file system")
					return ctrl.Result{}, err
				}
			} else {
				fs, err = conn.GetFileSystem(pool.Links.FileSystem.OdataId)
				if err != nil {
					log.Error(err, "Failed to retrieve file system")
					return ctrl.Result{}, err
				}
			}
		}

		for serverIdx := range node.Servers {
			server := &node.Servers[serverIdx]

			s, err := conn.GetServer(server.Id)
			if err != nil {
				log.Error(err, "Failed to retrieve server", "Server.Id", server.Id)
				return ctrl.Result{}, err
			}

			if len(server.StorageId) == 0 {
				log.Info("Creating storage group...", "StoragePool.Id", pool.Id)

				sg, err := conn.CreateStorageGroup(pool, s)
				if err != nil {
					log.Error(err, "Failed to create storage group")
					return ctrl.Result{}, err
				}

				log.Info("Created storage group", "StorageGroup.Id", sg.Id)
				server.StorageId = sg.Id
				r.Status().Update(ctx, storage)
			}

			if len(server.ShareId) != 0 {
				if fs == nil {
					err = errors.NewInternalError(fmt.Errorf("File System not present for pool"))
					log.Error(err, "Failed to find file system", "Pool.Id", pool.Id)
					return ctrl.Result{}, err
				}

				log.Info("Creating file share...", "FileSystem.Id", fs.Id)
				sh, err := conn.CreateFileShare(fs, s, server.Path)
				if err != nil {
					log.Error(err, "Failed to create file share")
					return ctrl.Result{}, err
				}

				log.Info("Created file share", "FileShare.Id", sh.Id)
				server.ShareId = sh.Id
				r.Status().Update(ctx, storage)
			}
		}

		log.Info("Initialized storage on node", "Node.Name", nodeName)
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NnfStorageReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&nnfv1alpha1.NnfStorage{}).
		Complete(r)
}
