/*
Copyright 2021 Hewlett Packard Enterprise Development LP
*/

package controllers

import (
	"context"
	"fmt"
	"reflect"

	"github.com/go-logr/logr"

	corev1 "k8s.io/api/core/v1"
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
	finalizer = "nnf.cray.hpe.com/finalizer"
)

//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfstorages,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfstorages/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfstorages/finalizers,verbs=update

// The Storage Controller will list and make modifications to individual NNF Nodes, so include the
// RBAC policy for nnfnodes. This isn't strictly necessary since the same ClusterRole is shared for
// both controllers, but we include it here for completeness
// TODO: UPDATE THIS COMMENT
//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfnodestorages,verbs=get;list;watch;create;update;patch;delete

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
		// ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		log.Error(err, "Failed to get instance")
		return ctrl.Result{}, client.IgnoreNotFound(err)
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

	// Add a finalizer to ensure all NNF Node Storage resources created by this resource are properly deleted.
	if !controllerutil.ContainsFinalizer(storage, finalizer) {

		controllerutil.AddFinalizer(storage, finalizer)
		if err := r.Update(ctx, storage); err != nil {
			log.Error(err, "Failed to update finalizer")
			return ctrl.Result{}, err
		}

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
			res, err = statusUpdater.Close(r, ctx)
			if err != nil {
				log.Info(fmt.Sprintf("Failed to update status with error '%s'", err))
			}
		}
	}()

	for nodeIdx := range storage.Spec.Nodes {

		nodeSpec := &storage.Spec.Nodes[nodeIdx]
		var nodeStatus *nnfv1alpha1.NnfStorageNodeStatus = nil
		for statusIdx := range storage.Status.Nodes {
			if nodeSpec.Node == storage.Status.Nodes[statusIdx].Node {
				nodeStatus = &storage.Status.Nodes[statusIdx]
				break
			}
		}

		if nodeStatus == nil {
			log.Info("Node Status Not Found. Creating...")
			statusUpdater.Update(func(s *nnfv1alpha1.NnfStorageStatus) {
				s.Nodes = append(s.Nodes, nnfv1alpha1.NnfStorageNodeStatus{
					Node: nodeSpec.Node,
				})
			})

			return ctrl.Result{Requeue: true}, nil
		}

		nodeStatusUpdater := NewStorageNodeStatusUpdater(statusUpdater, &storage.Status.Nodes[nodeIdx])

		res, err := r.reconcileNodeController(ctx, storage, nodeSpec, nodeStatusUpdater)
		if err != nil || !res.IsZero() {
			return res, err
		}
	}

	health := nnfv1alpha1.ResourceOkay
	for nodeIdx := range storage.Status.Nodes {
		if storage.Status.Nodes[nodeIdx].Health.IsWorseThan(health) {
			health = storage.Status.Nodes[nodeIdx].Health
		}
	}

	if health != storage.Status.Health {
		statusUpdater.Update(func(s *nnfv1alpha1.NnfStorageStatus) {
			s.Health = health
		})
	}

	return ctrl.Result{}, nil
}

// Reconcile the lower-level node controller Node Storage resource.
func (r *NnfStorageReconciler) reconcileNodeController(ctx context.Context, storage *nnfv1alpha1.NnfStorage, nodeSpec *nnfv1alpha1.NnfStorageNodeSpec, statusUpdater *storageNodeStatusUpdater) (ctrl.Result, error) {
	log := r.Log.WithValues("node", nodeSpec.Node)

	spec := &storage.Spec

	node := &nnfv1alpha1.NnfNodeStorage{}
	if err := r.Get(ctx, types.NamespacedName{Name: spec.Uuid, Namespace: nodeSpec.Node}, node); err != nil {

		if errors.IsNotFound(err) {

			log.Info("Creating NNF Node Storage...")
			node = &nnfv1alpha1.NnfNodeStorage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      spec.Uuid,
					Namespace: nodeSpec.Node,
				},
			}

			// First perform a deep copy - this will ensure the servers specified in the local node
			// specification get copied to the NNF Node Storage specification.
			nodeSpec.NnfNodeStorageSpec.DeepCopyInto(&node.Spec)

			// Next, fill in the global settings from the parent NNF Storage specification
			node.Spec.Capacity = spec.Capacity

			// Configure the owner reference - this does not work via controllerutil because cross-namespace
			// owner references are disallowed. Instead create a soft reference and make the delete logic
			// robust enough to handle ownership.
			//controllerutil.SetOwnerReference(storage, node, r.Scheme)
			node.Spec.Owner = corev1.ObjectReference{
				Name:       storage.Name,
				Namespace:  storage.Namespace,
				Kind:       storage.Kind,
				APIVersion: storage.APIVersion,
				UID:        storage.UID,
			}

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

		return ctrl.Result{Requeue: true}, nil
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
		Owns(&nnfv1alpha1.NnfNodeStorage{}).
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

func (u *storageStatusUpdater) Update(update func(s *nnfv1alpha1.NnfStorageStatus)) {
	update(&u.storage.Status)
	u.needsUpdate = true
}

func (u *storageStatusUpdater) Close(r *NnfStorageReconciler, ctx context.Context) (ctrl.Result, error) {
	defer func() { u.needsUpdate = false }()
	if u.needsUpdate {
		return ctrl.Result{Requeue: true}, r.Status().Update(ctx, u.storage)
	}

	return ctrl.Result{}, nil
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
