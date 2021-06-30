/*
Copyright 2021 Hewlett Packard Enterprise Development LP
*/

package controllers

import (
	"context"
	"fmt"
	"reflect"

	"github.com/go-logr/logr"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

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

//+kubebuilder:rbac:groups=nnf.cray.com,resources=nnfstorages,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=nnf.cray.com,resources=nnfstorages/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=nnf.cray.com,resources=nnfstorages/finalizers,verbs=update

// The Storage Controller will list and make modifications to individual NNF Nodes, so include the
// RBAC policy for nnfnodes. This isn't strictly necessary since the same ClusterRole is shared for
// both controllers, but we include it here for completeness
//+kubebuilder:rbac:groups=nnf.cray.com,resources=nnfnodes,verbs=get;list;watch;update;patch

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

		// TODO: Teardown storage
		/*
			if err := r.teardownStorage(ctx, storage); err != nil {
				return ctrl.Result{}, err
			}
		*/

		log.Info("Finalizing storage", "Storage.NamespacedName", types.NamespacedName{Name: storage.Name, Namespace: storage.Namespace})
		controllerutil.RemoveFinalizer(storage, finalizer)
		if err := r.Update(ctx, storage); err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	// First time setup adds the finalizer to the storage object and prepares
	// the status fields with initial data reflecting the present specification
	// values.
	if !controllerutil.ContainsFinalizer(storage, finalizer) {

		// Prepare the status fields; For each NNF Node in specification, there
		// must be a companion NNF Node Status.
		if storage.Status.Status != nnfv1alpha1.ResourceStarting {
			storage.Status.Status = nnfv1alpha1.ResourceStarting
			storage.Status.Health = nnfv1alpha1.ResourceOkay
			storage.Status.Nodes = make([]nnfv1alpha1.NnfStorageNodeStatus, len(storage.Spec.Nodes))
			if err := r.Status().Update(ctx, storage); err != nil {
				return ctrl.Result{}, err
			}

			// Force a requeue after applying changes. Otherwise this results in
			// future updates to be unfullfilled.
			return ctrl.Result{Requeue: true}, nil
		}

		// Ensure the storage specification supplied correct indicies
		for idx := range storage.Spec.Nodes {
			storage.Spec.Nodes[idx].Index = idx
		}

		controllerutil.AddFinalizer(storage, finalizer)
		if err := r.Update(ctx, storage); err != nil {
			return ctrl.Result{}, err
		}

		// Now that we've prepared the spec & status fields, force a requeue
		// to ensure Update() calls are clear of conflicts
		return ctrl.Result{Requeue: true}, nil
	}

	// At this point we've received a request to provision storage based on the list
	// of NNF Node Controllers supplied in the specification. For each NNF Node Controller,
	// we connect to the controller and allocate the required storage. This is done by fanning
	// out the required work to all NNF Nodes, and aggregating those results in the response
	// channel.

	for nodeIdx := range storage.Status.Nodes {
		if err := r.reconcileNodeController(ctx, storage.Spec, &storage.Spec.Nodes[nodeIdx], &storage.Status.Nodes[nodeIdx]); err != nil {
			//log.Error(err, "Failed to reconcile node")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *NnfStorageReconciler) reconcileNodeController(ctx context.Context, spec nnfv1alpha1.NnfStorageSpec, nodeSpec *nnfv1alpha1.NnfStorageNodeSpec, nodeStatus *nnfv1alpha1.NnfStorageNodeStatus) error {
	log := r.Log.WithValues("node", nodeSpec.Node)

	node := &nnfv1alpha1.NnfNode{}
	if err := r.Get(ctx, types.NamespacedName{Name: "nnf-nlc", Namespace: nodeSpec.Node}, node); err != nil {
		log.Error(err, "Unable to get NNF Node", "NamespacedName", types.NamespacedName{Name: "nnf-nlc", Namespace: nodeSpec.Node})
		return err
	}

	var storageSpec *nnfv1alpha1.NnfNodeStorageSpec = nil
	var storageStatus *nnfv1alpha1.NnfNodeStorageStatus = nil
	for idx := range node.Spec.Storage {
		if node.Spec.Storage[idx].Uuid == fmt.Sprintf("%s:%d", spec.Uuid, nodeSpec.Index) {
			storageSpec = &node.Spec.Storage[idx]
			storageStatus = &node.Status.Storage[idx]
		}
	}

	if storageSpec == nil {
		// Prepare the node storage specification
		storageSpec := nnfv1alpha1.NnfNodeStorageSpec{
			Uuid:       fmt.Sprintf("%s:%d", spec.Uuid, nodeSpec.Index),
			Capacity:   spec.Capacity,
			FileSystem: spec.FileSystem,
			State:      nnfv1alpha1.ResourceCreate,
			Servers:    make([]nnfv1alpha1.NnfNodeStorageServerSpec, len(nodeSpec.Servers)),
		}

		for idx := range nodeSpec.Servers {
			nodeSpec.Servers[idx].NnfNodeStorageServerSpec.DeepCopyInto(&storageSpec.Servers[idx])
		}

		node.Spec.Storage = append(node.Spec.Storage, storageSpec)

		if err := r.Update(ctx, node); err != nil {
			log.Error(err, "Failed to update NNF Node Storage spec")
			return err
		}

		log.Info("Created node storage spec")
		return nil
	}

	// Update the status of the node to reflect the current values
	// TODO: We should only be updating the status elements if changed
	//       then call r.Status().Update()
	nodeStatus.Status = node.Status.Status
	nodeStatus.Health = node.Status.Health

	// Update the status of the storage with the current values
	if !reflect.DeepEqual(storageStatus, &nodeStatus.Storage) {
		storageStatus.DeepCopyInto(&nodeStatus.Storage)

		//r.Status().Update(ctx, )
	}

	return nil
}

// teardownStorage will process the storage object in reverse of setupStorage, deleting
// all the objects that were created as part of setup. The bulk of this operation is handled
// by the NNF Element Controller, which supports deletion of the master Storage Pool object
// which will delete the tree of objects attached to the pool as well as the pool itself.
func (r *NnfStorageReconciler) teardownStorage(ctx context.Context, storage *nnfv1alpha1.NnfStorage) error {
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NnfStorageReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&nnfv1alpha1.NnfStorage{}).
		Complete(r)
}

type result struct{}

type nodeController struct {
	index  int
	spec   *nnfv1alpha1.NnfStorageNodeSpec
	status *nnfv1alpha1.NnfStorageNodeStatus
}
