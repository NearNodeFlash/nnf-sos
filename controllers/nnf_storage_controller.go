/*
Copyright 2021 Hewlett Packard Enterprise Development LP
*/

package controllers

import (
	"context"
	"fmt"
	"reflect"

	"github.com/go-logr/logr"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
// TODO: UPDATE THIS COMMENT
//+kubebuilder:rbac:groups=nnf.cray.com,resources=nnfnodestorages,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Storage object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.2/pkg/reconcile
func (r *NnfStorageReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, err error) {
	log := r.Log.WithValues("storage", req.NamespacedName)

	storage := &nnfv1alpha1.NnfStorage{}
	if err := r.Get(ctx, req.NamespacedName, storage); err != nil {
		log.Error(err, "Storage not found")
		return ctrl.Result{}, err
	}

	// Check if the object is being deleted
	if !storage.GetDeletionTimestamp().IsZero() {
		log.Info("Deleting storage...")

		if !controllerutil.ContainsFinalizer(storage, finalizer) {
			return ctrl.Result{}, nil
		}

		// TODO: Teardown storage
		/*
			if err := r.teardownStorage(ctx, storage); err != nil {
				return ctrl.Result{}, err
			}
		*/

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

		// Initialize the status fields if necessary. Each storage node defined in the specification
		// should have a corresponding storage node status field.
		if storage.Status.Status != nnfv1alpha1.ResourceStarting {
			storage.Status.Status = nnfv1alpha1.ResourceStarting
			storage.Status.Nodes = make([]nnfv1alpha1.NnfStorageNodeStatus, len(storage.Spec.Nodes))

			for nodeIdx, nodeSpec := range storage.Spec.Nodes {
				storage.Status.Nodes[nodeIdx].Node = nodeSpec.Node
			}

			if err := r.Status().Update(ctx, storage); err != nil {
				log.Error(err, "Failed to update storage status")
				return ctrl.Result{}, err
			}

			// Force a requeue after applying changes. Otherwise this results in
			// future updates to be unfullfilled.
			return ctrl.Result{Requeue: true}, nil
		}

		controllerutil.AddFinalizer(storage, finalizer)
		if err := r.Update(ctx, storage); err != nil {
			log.Error(err, "Failed to update finalizer")
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

	// Create an updater for the entire node. This will handle calls to r.Status().Update() such
	// that we can repeatedly make calls to the internal update method, with the final update
	// occuring on the on function exit.
	statusUpdater := NewStorageStatusUpdater(storage)
	defer func() {
		if err == nil {
			if err = statusUpdater.Close(r, ctx); err != nil {
				log.Info(fmt.Sprintf("Failed to update status with error %s", err))
			}
		}
	}()

	for nodeIdx := range storage.Status.Nodes {
		nodeStatusUpdater := NewStorageNodeStatusUpdater(statusUpdater, &storage.Status.Nodes[nodeIdx])

		res, err := r.reconcileNodeController(ctx, storage.Spec, &storage.Spec.Nodes[nodeIdx], nodeStatusUpdater)
		if err != nil || res.Requeue || res.RequeueAfter != 0 {
			return res, err
		}
	}

	return ctrl.Result{}, nil
}

// Reconcile the lower-level node controller Node Storage resource.
func (r *NnfStorageReconciler) reconcileNodeController(ctx context.Context, spec nnfv1alpha1.NnfStorageSpec, nodeSpec *nnfv1alpha1.NnfStorageNodeSpec, statusUpdater *storageNodeStatusUpdater) (ctrl.Result, error) {
	log := r.Log.WithValues("node", nodeSpec.Node)

	node := &nnfv1alpha1.NnfNodeStorage{}
	if err := r.Get(ctx, types.NamespacedName{Name: spec.Uuid, Namespace: nodeSpec.Node}, node); err != nil {

		if errors.IsNotFound(err) {

			node = &nnfv1alpha1.NnfNodeStorage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      spec.Uuid,
					Namespace: nodeSpec.Node,
				},
			}

			// Populate the global storage values into the local per-node specifications,
			// then copy the local specifications into the new NNF Node Storage.
			nodeSpec.NnfNodeStorageSpec.Capacity = spec.Capacity
			nodeSpec.NnfNodeStorageSpec.FileSystem = spec.FileSystem
			nodeSpec.NnfNodeStorageSpec.State = nnfv1alpha1.ResourceCreate
			nodeSpec.NnfNodeStorageSpec.DeepCopyInto(&node.Spec)

			if err := r.Create(ctx, node); err != nil {
				log.Error(err, "Failed to create NNF Node Storage")
				return ctrl.Result{}, err
			}

			return ctrl.Result{Requeue: true}, nil
		}

		return ctrl.Result{}, err
	}

	if !reflect.DeepEqual(node.Status, statusUpdater.Status()) {
		statusUpdater.Update(func(s *nnfv1alpha1.NnfStorageNodeStatus) {
			node.Status.DeepCopyInto(&s.NnfNodeStorageStatus)
		})
	}

	return ctrl.Result{}, nil
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

// Storage Status Updater handles finalizing of updates to the storage object for updates to a Storage object.
// Kubernete's requires only one such update per generation, with successive generations requiring a requeue.
// This object allows repeated calls to Update(), with the final call to reconciler.Status().Update() occurring
// only when the updater is Closed()
type storageStatusUpdater struct {
	storage     *nnfv1alpha1.NnfStorage
	needsUpdate bool
}

func NewStorageStatusUpdater(s *nnfv1alpha1.NnfStorage) *storageStatusUpdater {
	return &storageStatusUpdater{
		storage:     s,
		needsUpdate: false,
	}
}

func (u *storageStatusUpdater) Close(r *NnfStorageReconciler, ctx context.Context) error {
	defer func() { u.needsUpdate = false }()
	if u.needsUpdate {
		return r.Status().Update(ctx, u.storage)
	}

	return nil
}

// StorageNodeStatusUpdater defines a mechanism for updating a particular node's status values.
// It attaches to the parent StorageStatusUpdater to ensure calls to r.Status().Update() occur
// only one.
type storageNodeStatusUpdater struct {
	updater *storageStatusUpdater
	status  *nnfv1alpha1.NnfStorageNodeStatus
}

func NewStorageNodeStatusUpdater(u *storageStatusUpdater, s *nnfv1alpha1.NnfStorageNodeStatus) *storageNodeStatusUpdater {
	return &storageNodeStatusUpdater{
		updater: u,
		status:  s,
	}
}

func (u *storageNodeStatusUpdater) Status() *nnfv1alpha1.NnfStorageNodeStatus { return u.status }

func (u *storageNodeStatusUpdater) Update(update func(s *nnfv1alpha1.NnfStorageNodeStatus)) {
	update(u.status)
	u.updater.needsUpdate = true
}
