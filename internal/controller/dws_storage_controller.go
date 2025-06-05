/*
 * Copyright 2023-2025 Hewlett Packard Enterprise Development LP
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

package controller

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-logr/logr"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	dwsv1alpha5 "github.com/DataWorkflowServices/dws/api/v1alpha5"
	"github.com/DataWorkflowServices/dws/utils/updater"
	nnfv1alpha7 "github.com/NearNodeFlash/nnf-sos/api/v1alpha7"
)

type DWSStorageReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

const taintCrayNnfNodeDrainPrefix string = "cray.nnf.node.drain"

type K8sNodeState struct {
	// nnfTaint is the name of any "cray.nnf.node.drain*" taint found in Node.Spec.Taints.
	nnfTaint string

	// nodeReady indicates whether Node.Status.Conditions shows Ready or NotReady.
	nodeReady bool
}

// +kubebuilder:rbac:groups=dataworkflowservices.github.io,resources=storages,verbs=get;create;list;watch;update;patch;delete
// +kubebuilder:rbac:groups=dataworkflowservices.github.io,resources=storages/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfnodes,verbs=get;list;watch;
// +kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfnodes/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=nodes,verbs=get;list;watch

// Reconcile is part of the main Kubernetes reconciliation loop which aims to
// move the current state of the Storage resource to the desired state.
func (r *DWSStorageReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, err error) {
	log := r.Log.WithValues("resource", req.NamespacedName.String())

	storage := &dwsv1alpha5.Storage{}
	if err := r.Get(ctx, req.NamespacedName, storage); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Only reconcile this Storage resource if it is marked as Rabbit Storage
	labels := storage.GetLabels()
	if labels == nil {
		return ctrl.Result{}, nil
	}

	if storageType := labels[dwsv1alpha5.StorageTypeLabel]; storageType != "Rabbit" {
		return ctrl.Result{}, nil
	}

	// Create the status updater to update the status section if any changes are made
	statusUpdater := updater.NewStatusUpdater[*dwsv1alpha5.StorageStatus](storage)
	defer func() { err = statusUpdater.CloseWithStatusUpdate(ctx, r.Client.Status(), err) }()

	// Check if the object is being deleted
	if !storage.GetDeletionTimestamp().IsZero() {
		return ctrl.Result{}, nil
	}

	if storage.Spec.State == dwsv1alpha5.DisabledState {
		storage.Status.Status = dwsv1alpha5.DisabledStatus
		storage.Status.Message = "Storage node manually disabled"
	}

	if storage.Spec.Mode != "Live" {
		log.Info("Reconciliation skipped, not in live mode")
		return ctrl.Result{}, nil
	}

	// Ensure the storage resource is updated with the latest NNF Node resource status
	nnfNode := &nnfv1alpha7.NnfNode{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nnf-nlc",
			Namespace: storage.GetName(),
		},
	}

	if err := r.Get(ctx, client.ObjectKeyFromObject(nnfNode), nnfNode); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	storage.Status.Type = dwsv1alpha5.NVMe
	storage.Status.Capacity = nnfNode.Status.Capacity
	storage.Status.Access.Protocol = dwsv1alpha5.PCIe

	if len(nnfNode.Status.Servers) == 0 {
		return ctrl.Result{}, nil // Wait until severs array has been filled in with Rabbit info
	}

	// Populate server status' - Server 0 is reserved as the Rabbit node.
	storage.Status.Access.Servers = []dwsv1alpha5.Node{{
		Name:   storage.Name,
		Status: dwsv1alpha5.ResourceStatus(nnfNode.Status.Servers[0].Status.ConvertToDWSResourceStatus()),
	}}

	// Populate compute status'
	if len(nnfNode.Status.Servers) > 1 {
		storage.Status.Access.Computes = make([]dwsv1alpha5.Node, 0)
		for _, server := range nnfNode.Status.Servers[1:] /*Skip Rabbit*/ {

			// Servers that are unassigned in the system configuration will
			// not have a hostname and should be skipped.
			if len(server.Hostname) == 0 {
				continue
			}

			storage.Status.Access.Computes = append(storage.Status.Access.Computes,
				dwsv1alpha5.Node{
					Name:   server.Hostname,
					Status: dwsv1alpha5.ResourceStatus(server.Status.ConvertToDWSResourceStatus()),
				})
		}
	}

	// Populate storage status'
	storage.Status.Devices = make([]dwsv1alpha5.StorageDevice, len(nnfNode.Status.Drives))
	for idx, drive := range nnfNode.Status.Drives {
		device := &storage.Status.Devices[idx]

		device.Slot = drive.Slot
		device.Status = dwsv1alpha5.ResourceStatus(drive.Status.ConvertToDWSResourceStatus())

		if drive.Status == nnfv1alpha7.ResourceReady {
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

	// If the Rabbit is disabled we don't have to check the fenced status
	if storage.Spec.State == dwsv1alpha5.DisabledState {
		return ctrl.Result{}, nil
	}

	// Clear the fence status if the storage resource is enabled from a disabled state
	if storage.Status.Status == dwsv1alpha5.DisabledStatus {

		if nnfNode.Status.Fenced {
			log.WithValues("fenced", nnfNode.Status.Fenced).Info("resource disabled")
			nnfNode.Status.Fenced = false

			if err := r.Status().Update(ctx, nnfNode); err != nil {
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

	if nnfNode.Status.Fenced {
		storage.Status.Status = dwsv1alpha5.DegradedStatus
		storage.Status.RebootRequired = true
		storage.Status.Message = "Storage node requires reboot to recover from STONITH event"
	} else {
		nodeState, err := r.coreNodeState(ctx, storage)
		if err != nil {
			return ctrl.Result{}, err
		}

		if !nodeState.nodeReady {
			log.Info("storage node is offline")
			storage.Status.Status = dwsv1alpha5.OfflineStatus
			storage.Status.Message = "Kubernetes node is offline"
		} else if len(nodeState.nnfTaint) > 0 {
			log.Info(fmt.Sprintf("storage node is tainted with %s", nodeState.nnfTaint))
			storage.Status.Status = dwsv1alpha5.DrainedStatus
			storage.Status.Message = fmt.Sprintf("Kubernetes node is tainted with %s", nodeState.nnfTaint)
		} else {
			storage.Status.Status = dwsv1alpha5.ResourceStatus(nnfNode.Status.Status.ConvertToDWSResourceStatus())
		}
	}

	return ctrl.Result{}, nil
}

func (r *DWSStorageReconciler) coreNodeState(ctx context.Context, storage *dwsv1alpha5.Storage) (K8sNodeState, error) {
	nodeState := K8sNodeState{}

	// Get the kubernetes node resource corresponding to the same node as the nnfNode resource.
	// The kubelet has a heartbeat mechanism, so we can determine node failures from this resource.
	node := &corev1.Node{}
	if err := r.Get(ctx, types.NamespacedName{Name: storage.Name}, node); err != nil {
		return nodeState, err
	}

	// Look through the node conditions to determine if the node is up.
	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady && condition.Status == corev1.ConditionTrue {
			nodeState.nodeReady = true
			break
		}
	}

	// Look for any "cray.nnf.node.drain*" taint.
	for _, taint := range node.Spec.Taints {
		if strings.HasPrefix(taint.Key, taintCrayNnfNodeDrainPrefix) {
			nodeState.nnfTaint = taint.Key
			break
		}
	}

	return nodeState, nil
}

func (r *DWSStorageReconciler) SetupWithManager(mgr ctrl.Manager) error {

	// Setup to watch the NNF Node resource
	nnfNodeMapFunc := func(ctx context.Context, o client.Object) []reconcile.Request {
		return []reconcile.Request{{NamespacedName: types.NamespacedName{
			Name:      o.GetNamespace(),
			Namespace: corev1.NamespaceDefault,
		}}}
	}

	// Setup to watch the Kubernetes Node resource
	nodeMapFunc := func(ctx context.Context, o client.Object) []reconcile.Request {
		return []reconcile.Request{{NamespacedName: types.NamespacedName{
			Name:      o.GetName(),
			Namespace: corev1.NamespaceDefault,
		}}}
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&dwsv1alpha5.Storage{}).
		Watches(&nnfv1alpha7.NnfNode{}, handler.EnqueueRequestsFromMapFunc(nnfNodeMapFunc)).
		Watches(&corev1.Node{}, handler.EnqueueRequestsFromMapFunc(nodeMapFunc)).
		Complete(r)
}
