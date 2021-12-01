/*
Copyright 2021 Hewlett Packard Enterprise Development LP
*/

package controllers

import (
	"context"
	"runtime"

	"github.com/go-logr/logr"

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
	"sigs.k8s.io/controller-runtime/pkg/source"

	nnfv1alpha1 "stash.us.cray.com/RABSW/nnf-sos/api/v1alpha1"
	dwsv1alpha1 "stash.us.cray.com/dpm/dws-operator/api/v1alpha1"
)

// NnfNodeSLCReconciler reconciles a NnfNode object
type NnfNodeSLCReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *kruntime.Scheme
}

const (
	// finalizerNnfNodeSLC defines the key used to prevent the system from deleting the
	// resource until this reconciler has finished doing clean up
	finalizerNnfNodeSLC = "nnf.cray.hpe.com/nnf-node-slc"
)

//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfnodes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfnodes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfnodes/finalizers,verbs=update
//+kubebuilder:rbac:groups=dws.cray.hpe.com,resources=storages,verbs=get;create;list;watch;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=nodes,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the NnfNodeSLC to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.9.2/pkg/reconcile
func (r *NnfNodeSLCReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, err error) {
	log := r.Log.WithValues("NnfNodeSLC", req.NamespacedName)

	nnfNode := &nnfv1alpha1.NnfNode{}
	if err := r.Get(ctx, req.NamespacedName, nnfNode); err != nil {
		// ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Check if the object is being deleted
	if !nnfNode.GetDeletionTimestamp().IsZero() {
		if !controllerutil.ContainsFinalizer(nnfNode, finalizerNnfNodeSLC) {
			return ctrl.Result{}, nil
		}

		if err := r.deleteStorage(ctx, nnfNode); err != nil {
			return ctrl.Result{}, err
		}

		controllerutil.RemoveFinalizer(nnfNode, finalizerNnfNodeSLC)
		if err := r.Update(ctx, nnfNode); err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	// Add a finalizer to ensure the DWS Storage resource created by this resource is properly deleted.
	if !controllerutil.ContainsFinalizer(nnfNode, finalizerNnfNodeSLC) {

		controllerutil.AddFinalizer(nnfNode, finalizerNnfNodeSLC)
		if err := r.Update(ctx, nnfNode); err != nil {
			return ctrl.Result{Requeue: true}, nil
		}

		return ctrl.Result{}, nil
	}

	// Get the kubernetes node resource corresponding to the same node as the nnfNode resource.
	// The kubelet has a heartbeat mechanism, so we can determine node failures from this resource.
	node := &corev1.Node{}
	if err := r.Get(ctx, types.NamespacedName{Name: nnfNode.Namespace}, node); err != nil {
		return ctrl.Result{}, err
	}

	// Look through the node conditions to determine if the node is up
	nodeStatus := "NotReady"
	for _, condition := range node.Status.Conditions {
		if condition.Type == "Ready" && condition.Status == "True" {
			nodeStatus = "Ready"
			break
		}
	}

	// Create a DWS Storage resource based on the information from the NNFNode and Node resources
	storage := &dwsv1alpha1.Storage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nnfNode.Namespace,
			Namespace: "default",
		},
	}

	result, err := ctrl.CreateOrUpdate(ctx, r.Client, storage,
		func() error {
			storage.Data.Type = "NVMe"
			storage.Data.Capacity = nnfNode.Status.Capacity
			if nodeStatus == "Ready" {
				storage.Data.Status = string(nnfNode.Status.Status)
			} else {
				storage.Data.Status = nodeStatus
			}

			// Add a label for a storage type of Rabbit. Don't overwrite the
			// existing labels map since there may be user applied labels.
			labels := storage.GetLabels()
			if labels == nil {
				labels = make(map[string]string)
			}
			labels[dwsv1alpha1.StorageTypeLabel] = "Rabbit"
			storage.SetLabels(labels)

			// Access information for how to get to the storage
			storage.Data.Access.Protocol = "PCIe"

			// Wait until the servers array has been filled in with the Rabbit info
			if len(nnfNode.Status.Servers) == 0 {
				return nil
			}

			storage.Data.Access.Servers = []dwsv1alpha1.Node{{
				// The Rabbit node is always server 0 in NNFNode
				Name:   nnfNode.Status.Servers[0].Name,
				Status: string(nnfNode.Status.Servers[0].Status),
			}}

			// Wait until the servers array has been filled in with the compute node info
			if len(nnfNode.Status.Servers) == 1 {
				return nil
			}

			computes := []dwsv1alpha1.Node{}
			for _, c := range nnfNode.Status.Servers[1:] {
				compute := dwsv1alpha1.Node{
					Name:   c.Name,
					Status: string(c.Status),
				}
				computes = append(computes, compute)
			}
			storage.Data.Access.Computes = computes

			return nil
		})
	if err != nil {
		return ctrl.Result{}, err
	}

	if result == controllerutil.OperationResultCreated {
		log.Info("Created DWS Storage resource", "name", storage.Name)
	} else if result == controllerutil.OperationResultNone {
		// no change
	} else {
		log.Info("Updated DWS Storage", "name", storage.Name)
	}

	return ctrl.Result{}, nil
}

func (r *NnfNodeSLCReconciler) deleteStorage(ctx context.Context, nnfNode *nnfv1alpha1.NnfNode) error {
	log := r.Log.WithValues("NnfNode", types.NamespacedName{Name: nnfNode.Name, Namespace: nnfNode.Namespace})

	storage := &dwsv1alpha1.Storage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nnfNode.Namespace,
			Namespace: "default",
		},
	}

	err := r.Delete(ctx, storage)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
	} else {
		log.Info("Deleted DWS Storage resource")
	}

	return nil
}

// Map a DWS storage resource to a NnfNode resource. This is a cross-namespace
// mapping, so we can't use owner references
func nnfStorageMapFunc(o client.Object) []reconcile.Request {
	return []reconcile.Request{
		{NamespacedName: types.NamespacedName{
			Name:      "nnf-nlc",
			Namespace: o.GetName(),
		}},
	}
}

// Map a kubernetes Node resource to a NnfNode resource
func nnfNodeMapFunc(o client.Object) []reconcile.Request {
	return []reconcile.Request{
		{NamespacedName: types.NamespacedName{
			Name:      "nnf-nlc",
			Namespace: o.GetName(),
		}},
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *NnfNodeSLCReconciler) SetupWithManager(mgr ctrl.Manager) error {
	maxReconciles := runtime.GOMAXPROCS(0)
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(controller.Options{MaxConcurrentReconciles: maxReconciles}).
		For(&nnfv1alpha1.NnfNode{}).
		Watches(&source.Kind{Type: &dwsv1alpha1.Storage{}}, handler.EnqueueRequestsFromMapFunc(nnfStorageMapFunc)).
		Watches(&source.Kind{Type: &corev1.Node{}}, handler.EnqueueRequestsFromMapFunc(nnfNodeMapFunc)).
		Complete(r)
}
