/*
 * Copyright 2023 Hewlett Packard Enterprise Development LP
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

	"github.com/go-logr/logr"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	dwsv1alpha1 "github.com/HewlettPackard/dws/api/v1alpha1"
	"github.com/HewlettPackard/dws/utils/updater"
	nnfv1alpha1 "github.com/NearNodeFlash/nnf-sos/api/v1alpha1"
)

type DWSStorageReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=dws.cray.hpe.com,resources=storages,verbs=get;create;list;watch;update;patch;delete
// +kubebuilder:rbac:groups=dws.cray.hpe.com,resources=storages/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfnodes,verbs=get;list;watch;
// +kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfnodes/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=nodes,verbs=get;list;watch

// Reconcile is part of the main Kubernetes reconciliation loop which aims to
// move the current state of the Storage resource to the desired state.
func (r *DWSStorageReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, err error) {
	log := r.Log.WithValues("resource", req.NamespacedName.String())

	storage := &dwsv1alpha1.Storage{}
	if err := r.Get(ctx, req.NamespacedName, storage); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Check if the object is being deleted
	if !storage.GetDeletionTimestamp().IsZero() {
		return ctrl.Result{}, nil
	}

	// Ensure the storage resource is updated with the latest NNF Node resource status
	node := &nnfv1alpha1.NnfNode{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nnf-nlc",
			Namespace: storage.GetName(),
		},
	}

	if err := r.Get(ctx, client.ObjectKeyFromObject(node), node); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Now that it is confirmed an NNF Node resource exists for this Storage resource,
	// ensure the proper labels are set on the resource
	labels := storage.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}

	const rabbitStorageType = "Rabbit"
	if label, found := labels[dwsv1alpha1.StorageTypeLabel]; !found || label != rabbitStorageType {
		labels[dwsv1alpha1.StorageTypeLabel] = rabbitStorageType
		storage.SetLabels(labels)

		if err := r.Update(ctx, storage); err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{Requeue: true}, nil
	}

	// Create a new status

	statusUpdater := updater.NewStatusUpdater[*dwsv1alpha1.StorageStatus](storage)
	defer func() { err = statusUpdater.CloseWithStatusUpdate(ctx, r.Client.Status(), err) }()

	storage.Status.Type = dwsv1alpha1.NVMe
	storage.Status.Capacity = node.Status.Capacity
	storage.Status.Access.Protocol = dwsv1alpha1.PCIe

	if len(node.Status.Servers) == 0 {
		return ctrl.Result{}, nil // Wait until severs array has been filed in with Rabbit info
	}

	// Populate server status' - Server 0 is reserved as the Rabbit node.
	storage.Status.Access.Servers = []dwsv1alpha1.Node{{
		Name:   storage.Name,
		Status: node.Status.Servers[0].Status.ConvertToDWSResourceStatus(),
	}}

	// Populate compute status'
	if len(node.Status.Servers) > 1 {
		storage.Status.Access.Computes = make([]dwsv1alpha1.Node, 0)
		for _, server := range node.Status.Servers[1:] /*Skip Rabbit*/ {

			// Servers that are unassigned in the system configuration will
			// not have a hostname and should be skipped.
			if len(server.Hostname) == 0 {
				continue
			}

			storage.Status.Access.Computes = append(storage.Status.Access.Computes,
				dwsv1alpha1.Node{
					Name:   server.Hostname,
					Status: server.Status.ConvertToDWSResourceStatus(),
				})
		}
	}

	// Populate storage status'
	storage.Status.Devices = make([]dwsv1alpha1.StorageDevice, len(node.Status.Drives))
	for idx, drive := range node.Status.Drives {
		device := &storage.Status.Devices[idx]

		device.Slot = drive.Slot
		device.Status = drive.Status.ConvertToDWSResourceStatus()

		if drive.Status == nnfv1alpha1.ResourceReady {
			wearLevel := drive.WearLevel
			device.Model = drive.Model
			device.SerialNumber = drive.SerialNumber
			device.FirmwareVersion = drive.FirmwareVersion
			device.Capacity = drive.Capacity
			device.WearLevel = &wearLevel
		} else {
			device.Model = ""
			device.SerialNumber = ""
			device.FirmwareVersion = ""
			device.Capacity = 0
			device.WearLevel = nil
		}
	}

	// Handle any state transitions
	switch storage.Spec.State {
	case dwsv1alpha1.EnabledState:

		// Clear the fence status if the storage resource is enabled from a disabled state
		if storage.Status.Status == dwsv1alpha1.DisabledStatus {

			log.WithValues("fenced", node.Status.Fenced).Info("resource disabled")
			if node.Status.Fenced {
				node.Status.Fenced = false

				if err := r.Status().Update(ctx, node); err != nil {
					return ctrl.Result{}, err
				}

				log.Info("fenced status cleared")
				return ctrl.Result{}, nil
			}

			// TODO: Fencing Agent Phase #2: Resume Rabbit NLC pods, wait for the pods to
			//       resume, then change Node Status to Enabled

			storage.Status.RebootRequired = false
			storage.Status.Message = ""
		}

		if node.Status.Fenced {
			storage.Status.Status = dwsv1alpha1.DegradedStatus
			storage.Status.RebootRequired = true
			storage.Status.Message = "Storage node requires reboot to recover from STONITH event"
		} else {

			ready, err := r.isKubernetesNodeReady(ctx, storage)
			if err != nil {
				return ctrl.Result{}, err
			}

			if !ready {
				log.Info("storage node is offline")
				storage.Status.Status = dwsv1alpha1.OfflineStatus
				storage.Status.Message = "Kubernetes node is offline"
			} else {
				storage.Status.Status = node.Status.Status.ConvertToDWSResourceStatus()
			}
		}

	case dwsv1alpha1.DisabledState:
		// TODO: Fencing Agent Phase #2: Pause Rabbit NLC pods, wait for pods to be
		//       removed, then change Node Status to Disabled

		storage.Status.Status = dwsv1alpha1.DisabledStatus
		storage.Status.Message = "Storage node manually disabled"
	}

	return ctrl.Result{}, nil
}

func (r *DWSStorageReconciler) isKubernetesNodeReady(ctx context.Context, storage *dwsv1alpha1.Storage) (bool, error) {
	// Get the kubernetes node resource corresponding to the same node as the nnfNode resource.
	// The kubelet has a heartbeat mechanism, so we can determine node failures from this resource.
	node := &corev1.Node{}
	if err := r.Get(ctx, types.NamespacedName{Name: storage.Name}, node); err != nil {
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

func (r *DWSStorageReconciler) SetupWithManager(mgr ctrl.Manager) error {

	// Setup to watch the NNF Node resource
	nnfNodeMapFunc := func(o client.Object) []reconcile.Request {
		return []reconcile.Request{{NamespacedName: types.NamespacedName{
			Name:      o.GetNamespace(),
			Namespace: corev1.NamespaceDefault,
		}}}
	}

	// Setup to watch the Kubernetes Node resource
	nodeMapFunc := func(o client.Object) []reconcile.Request {
		return []reconcile.Request{{NamespacedName: types.NamespacedName{
			Name: o.GetName(),
		}}}
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&dwsv1alpha1.Storage{}).
		Watches(&source.Kind{Type: &nnfv1alpha1.NnfNode{}}, handler.EnqueueRequestsFromMapFunc(nnfNodeMapFunc)).
		Watches(&source.Kind{Type: &corev1.Node{}}, handler.EnqueueRequestsFromMapFunc(nodeMapFunc)).
		Complete(r)
}
