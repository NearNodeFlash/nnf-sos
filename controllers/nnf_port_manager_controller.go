/*
 * Copyright 2022 Hewlett Packard Enterprise Development LP
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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	portsutil "github.com/HewlettPackard/dws/utils/ports"
	"github.com/HewlettPackard/dws/utils/updater"
	"github.com/go-logr/logr"

	dwsv1alpha1 "github.com/HewlettPackard/dws/api/v1alpha1"
	nnfv1alpha1 "github.com/NearNodeFlash/nnf-sos/api/v1alpha1"
)

// NnfPortManagerReconciler reconciles a NnfPortManager object
type NnfPortManagerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// type aliases for name shortening
type AllocationSpec = nnfv1alpha1.NnfPortManagerAllocationSpec
type AllocationStatus = nnfv1alpha1.NnfPortManagerAllocationStatus

//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfportmanagers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfportmanagers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfportmanagers/finalizers,verbs=update

// System Configuration provides the
//+kubebuilder:rbac:groups=dws.cray.hpe.com,resources=systemconfigurations,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the NnfPortManager object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *NnfPortManagerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, err error) {
	log := log.FromContext(ctx)

	mgr := &nnfv1alpha1.NnfPortManager{}
	if err := r.Get(ctx, req.NamespacedName, mgr); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Create a resource status updater to ensure the status subresource is updated.
	statusUpdater := updater.NewStatusUpdater[*nnfv1alpha1.NnfPortManagerStatus](mgr)
	defer func() { err = statusUpdater.CloseWithStatusUpdate(ctx, r, err) }()

	// Read in the system configuration which contains the available ports.
	config := &dwsv1alpha1.SystemConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mgr.Spec.SystemConfiguration.Name,
			Namespace: mgr.Spec.SystemConfiguration.Namespace,
		},
	}

	mgr.Status.Status = nnfv1alpha1.NnfPortManagerStatusReady
	if err := r.Get(ctx, client.ObjectKeyFromObject(config), config); err != nil {
		if !errors.IsNotFound(err) {
			return ctrl.Result{}, err
		}

		log.Info("System Configuration not found", "config", client.ObjectKeyFromObject(config).String())
		mgr.Status.Status = nnfv1alpha1.NnfPortManagerStatusSystemConfigurationNotFound
		res = ctrl.Result{Requeue: true} // Force a requeue - we want the manager to go ready even if there are zero allocations
	}

	// Free any unused allocations
	r.cleanupUnusedAllocations(log, mgr)

	// For each "requester" in the mgr.Spec.Allocations, try to satisfy the request by
	// allocating the desired ports.
	for _, spec := range mgr.Spec.Allocations {

		// If the specification is already included in the allocations, continue
		if r.isAllocated(mgr, spec) {
			continue
		}

		var ports []uint16
		var status nnfv1alpha1.NnfPortManagerAllocationStatusStatus
		if mgr.Status.Status != nnfv1alpha1.NnfPortManagerStatusReady {
			ports, status = nil, nnfv1alpha1.NnfPortManagerAllocationStatusInvalidConfiguration
		} else {
			ports, status = r.findFreePorts(log, mgr, config, spec)
		}

		log.Info("Allocation", "requester", spec.Requester, "count", spec.Count, "ports", ports, "status", status)
		allocationStatus := AllocationStatus{
			Requester: &corev1.ObjectReference{},
			Ports:     ports,
			Status:    status,
		}

		spec.Requester.DeepCopyInto(allocationStatus.Requester)

		if mgr.Status.Allocations == nil {
			mgr.Status.Allocations = make([]nnfv1alpha1.NnfPortManagerAllocationStatus, 0)
		}

		mgr.Status.Allocations = append(mgr.Status.Allocations, allocationStatus)
	}

	return res, nil
}

// isAllocationNeeded returns true if the provided Port Allocation Status has a matching value
// requester in the specification, and false otherwise.
func (r *NnfPortManagerReconciler) isAllocationNeeded(mgr *nnfv1alpha1.NnfPortManager, status *AllocationStatus) bool {
	if status.Status != nnfv1alpha1.NnfPortManagerAllocationStatusInUse {
		return false
	}

	if status.Requester == nil {
		return false
	}

	for _, desired := range mgr.Spec.Allocations {
		if *status.Requester == desired.Requester {
			return true
		}
	}

	return false
}

func (r *NnfPortManagerReconciler) cleanupUnusedAllocations(log logr.Logger, mgr *nnfv1alpha1.NnfPortManager) {

	// Free unused allocations. This will check if the Status.Allocations exist in
	// the list of desired allocations in the Spec field and mark any unused allocations
	// as freed.
	failedIndices := make([]int, 0)
	for idx := range mgr.Status.Allocations {
		status := &mgr.Status.Allocations[idx]

		if !r.isAllocationNeeded(mgr, status) {
			log.Info("Allocation unused", "requester", status.Requester, "status", status.Status)
			if status.Status == nnfv1alpha1.NnfPortManagerAllocationStatusInUse {
				status.Requester = nil
				status.Status = nnfv1alpha1.NnfPortManagerAllocationStatusFree
			} else if status.Status != nnfv1alpha1.NnfPortManagerAllocationStatusFree {
				failedIndices = append(failedIndices, idx)
			}
		}
	}

	for idx := range failedIndices {
		failedIdx := failedIndices[len(failedIndices)-1-idx] // remove in reverse order
		mgr.Status.Allocations = append(mgr.Status.Allocations[:failedIdx], mgr.Status.Allocations[failedIdx+1:]...)
	}
}

func (r *NnfPortManagerReconciler) findAllocationStatus(mgr *nnfv1alpha1.NnfPortManager, spec AllocationSpec) *AllocationStatus {
	for idx := range mgr.Status.Allocations {
		status := &mgr.Status.Allocations[idx]
		if status.Requester == nil {
			continue
		}

		if *status.Requester == spec.Requester {
			return status
		}
	}

	return nil
}

// isAllocated returns true if the provided specification is in the Port Manager's allocation
// status', and false otherwise.
func (r *NnfPortManagerReconciler) isAllocated(mgr *nnfv1alpha1.NnfPortManager, spec AllocationSpec) bool {
	return r.findAllocationStatus(mgr, spec) != nil
}

// Find free ports to satisfy the provided specification.
func (r *NnfPortManagerReconciler) findFreePorts(log logr.Logger, mgr *nnfv1alpha1.NnfPortManager, config *dwsv1alpha1.SystemConfiguration, spec AllocationSpec) ([]uint16, nnfv1alpha1.NnfPortManagerAllocationStatusStatus) {

	portsInUse := make([]uint16, 0)
	for _, status := range mgr.Status.Allocations {
		if status.Status == nnfv1alpha1.NnfPortManagerAllocationStatusInUse {
			portsInUse = append(portsInUse, status.Ports...)
		}
	}

	isPortInUse := func(port uint16) bool {
		for _, p := range portsInUse {
			if p == port {
				return true
			}
		}

		return false
	}

	count := spec.Count
	ports := make([]uint16, 0)

	// We first turn to the system configuration and try to allocate free ports from there.
	itr := portsutil.NewPortIterator(config.Spec.Ports)
	for port := itr.Next(); port != portsutil.InvalidPort; port = itr.Next() {

		if isPortInUse(port) {
			continue
		}

		ports = append(ports, port)

		if len(ports) >= count {
			break
		}
	}

	if len(ports) >= count {
		log.Info("Ports claimed from system configuration", "ports", ports)
		return ports[:count], nnfv1alpha1.NnfPortManagerAllocationStatusInUse
	}

	// If we still haven't found a sufficient number of free ports, free up unused allocations
	// in the status' until we have the desired number of ports.

	const (
		exhausted = iota
		more
	)

	// Search for a free allocation and claim ports from that allocation. Returns 'more' if there
	// are potentially more allocations available with free ports, and 'exhausted' otherwise.
	claimPortsFromFreeAllocation := func() int {
		for idx := range mgr.Status.Allocations {
			status := &mgr.Status.Allocations[idx]

			if status.Status == nnfv1alpha1.NnfPortManagerAllocationStatusFree {
				log.Info("Ports claimed from free list", "ports", status.Ports)

				// Append this values ports to the returned ports. We could over-allocate here, but
				// that is okay because we'll truncate the ports on return.
				ports = append(ports, status.Ports...)

				// Remove the allocation from the list.
				mgr.Status.Allocations = append(mgr.Status.Allocations[:idx], mgr.Status.Allocations[idx+1:]...)

				return more // because we just modified the list in place, must break from the for loop and force an additional search
			}
		}

		return exhausted
	}

	// Loop until we have all (or more) of the requested port count, or we've exhausted the search
	for len(ports) < count {
		switch claimPortsFromFreeAllocation() {
		case exhausted:
			return []uint16{}, nnfv1alpha1.NnfPortManagerAllocationStatusInsufficientResources
		case more:
			// loop again if needed
		}
	}

	return ports[:count], nnfv1alpha1.NnfPortManagerAllocationStatusInUse
}

// SetupWithManager sets up the controller with the Manager.
func (r *NnfPortManagerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&nnfv1alpha1.NnfPortManager{}).
		Complete(r)
}