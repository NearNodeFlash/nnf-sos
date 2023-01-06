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

	dwsv1alpha1 "github.com/HewlettPackard/dws/api/v1alpha1"
	"github.com/HewlettPackard/dws/utils/updater"
	nnfv1alpha1 "github.com/NearNodeFlash/nnf-sos/api/v1alpha1"
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
//+kubebuilder:rbac:groups=dws.cray.hpe.com,resources=storages/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core,resources=nodes,verbs=get;list;watch
//+kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch

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
			if !apierrors.IsConflict(err) {
				return ctrl.Result{}, err
			}

			return ctrl.Result{Requeue: true}, nil
		}

		return ctrl.Result{}, nil
	}

	// Add a finalizer to ensure the DWS Storage resource created by this resource is properly deleted.
	if !controllerutil.ContainsFinalizer(nnfNode, finalizerNnfNodeSLC) {

		controllerutil.AddFinalizer(nnfNode, finalizerNnfNodeSLC)
		if err := r.Update(ctx, nnfNode); err != nil {
			if !apierrors.IsConflict(err) {
				return ctrl.Result{}, err
			}

			return ctrl.Result{Requeue: true}, nil
		}

		return ctrl.Result{}, nil
	}

	// Create a DWS Storage resource based on the information from the NNFNode and Node resources
	storage := &dwsv1alpha1.Storage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nnfNode.Namespace,
			Namespace: corev1.NamespaceDefault,
		},
	}

	if err := r.Get(ctx, client.ObjectKeyFromObject(storage), storage); err != nil {
		if !apierrors.IsNotFound(err) {
			return ctrl.Result{}, err
		}

		storage.Spec = dwsv1alpha1.StorageSpec{
			State: dwsv1alpha1.EnabledState,
		}

		if err := r.Create(ctx, storage); err != nil {
			return ctrl.Result{}, err
		}

		log.Info("Created DWS Storage resource", "name", storage.Name)

		return ctrl.Result{Requeue: true}, nil
	}

	// Create a pair of updaters for updating the NNF Node status and Storage status
	nnfNodeStatusUpdater := updater.NewStatusUpdater[*nnfv1alpha1.NnfNodeStatus](nnfNode)
	defer func() { err = nnfNodeStatusUpdater.CloseWithStatusUpdate(ctx, r, err) }()

	storageStatusUpdater := updater.NewStatusUpdater[*dwsv1alpha1.StorageStatus](storage)
	defer func() { err = storageStatusUpdater.CloseWithStatusUpdate(ctx, r, err) }()

	// Populate the status of the storage resource.
	storage.Status.Type = dwsv1alpha1.NVMe
	storage.Status.Capacity = nnfNode.Status.Capacity

	switch storage.Spec.State {
	case dwsv1alpha1.EnabledState:
		// Clear the fenced status if the node is enabled from a disabled status
		if storage.Status.Status == dwsv1alpha1.DisabledStatus {
			nnfNode.Status.Fenced = false

			storage.Status.RebootRequired = false
			storage.Status.Message = ""

			// TODO: Fencing Agent Phase #2: Resume Rabbit NLC pods, wait for the pods to
			//       resume, then change Node Status to Enabled
		}

		if nnfNode.Status.Fenced {
			storage.Status.Status = dwsv1alpha1.DegradedStatus
			storage.Status.RebootRequired = true
			storage.Status.Message = "Storage node requires reboot to recover from STONITH event"
		} else {

			// Check if the kubernetes node itself is offline
			ready, err := r.isKubernetesNodeReady(ctx, nnfNode)
			if err != nil {
				return ctrl.Result{}, err
			}

			if !ready {
				storage.Status.Status = dwsv1alpha1.OfflineStatus
				storage.Status.Message = "Kubernetes node is not ready"
			} else {
				storage.Status.Status = nnfNode.Status.Status.ConvertToDWSResourceStatus()
			}
		}
	case dwsv1alpha1.DisabledState:
		// TODO: Fencing Agent Phase #2: Pause Rabbit NLC pods, wait for pods to be
		//       removed, then change Node Status to Disabled

		storage.Status.Status = dwsv1alpha1.DisabledStatus
		storage.Status.Message = "Storage node was manually disabled"
	default:
		log.Info("Unhandled storage state", "state", storage.Spec.State)
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
	storage.Status.Access.Protocol = dwsv1alpha1.PCIe

	// Wait until the servers array has been filled in with the Rabbit info
	if len(nnfNode.Status.Servers) == 0 {
		return ctrl.Result{}, nil
	}

	storage.Status.Access.Servers = []dwsv1alpha1.Node{{
		Name:   nnfNode.Namespace, // The Rabbit node is the name of the nnfNode namespace
		Status: nnfNode.Status.Servers[0].Status.ConvertToDWSResourceStatus(),
	}}

	computes := []dwsv1alpha1.Node{}
	for _, c := range nnfNode.Status.Servers[1:] {
		if c.Hostname == "" {
			continue
		}

		compute := dwsv1alpha1.Node{
			Name:   c.Hostname,
			Status: c.Status.ConvertToDWSResourceStatus(),
		}
		computes = append(computes, compute)
	}
	storage.Status.Access.Computes = computes

	devices := []dwsv1alpha1.StorageDevice{}
	for _, d := range nnfNode.Status.Drives {
		if d.Status == nnfv1alpha1.ResourceOffline {
			continue
		}

		wearLevel := d.WearLevel
		device := dwsv1alpha1.StorageDevice{
			Model:           d.Model,
			SerialNumber:    d.SerialNumber,
			FirmwareVersion: d.FirmwareVersion,
			Capacity:        d.Capacity,
			Status:          d.Status.ConvertToDWSResourceStatus(),
			WearLevel:       &wearLevel,
			Slot:            d.Slot,
		}
		devices = append(devices, device)
	}
	storage.Status.Devices = devices

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

	if err := r.Delete(ctx, storage); err != nil {
		return client.IgnoreNotFound(err)
	}

	log.Info("Deleted DWS Storage resource", "name", storage.Name)

	return nil
}

func (r *NnfNodeSLCReconciler) isKubernetesNodeReady(ctx context.Context, nnfNode *nnfv1alpha1.NnfNode) (bool, error) {
	// Get the kubernetes node resource corresponding to the same node as the nnfNode resource.
	// The kubelet has a heartbeat mechanism, so we can determine node failures from this resource.
	node := &corev1.Node{}
	if err := r.Get(ctx, types.NamespacedName{Name: nnfNode.Namespace}, node); err != nil {
		return false, err
	}

	// Look through the node conditions to determine if the node is up.
	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady && condition.Status == corev1.ConditionTrue {
			return true, nil
		}
	}

	return false, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NnfNodeSLCReconciler) SetupWithManager(mgr ctrl.Manager) error {

	maxReconciles := runtime.GOMAXPROCS(0)

	mapFunc := func(o client.Object) []reconcile.Request {
		return []reconcile.Request{{NamespacedName: types.NamespacedName{
			Name:      "nnf-nlc",
			Namespace: o.GetName(),
		}}}
	}

	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(controller.Options{MaxConcurrentReconciles: maxReconciles}).
		For(&nnfv1alpha1.NnfNode{}).
		Watches(&source.Kind{Type: &dwsv1alpha1.Storage{}}, handler.EnqueueRequestsFromMapFunc(mapFunc)).
		Watches(&source.Kind{Type: &corev1.Node{}}, handler.EnqueueRequestsFromMapFunc(mapFunc)).
		Complete(r)
}
