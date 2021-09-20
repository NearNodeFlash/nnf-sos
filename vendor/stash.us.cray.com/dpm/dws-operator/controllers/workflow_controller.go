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
	myerror "errors"
	"strings"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	dwsv1alpha1 "stash.us.cray.com/dpm/dws-operator/api/v1alpha1"
)

// Define condtion values
const (
	ConditionTrue  bool = true
	ConditionFalse bool = false
)

// WorkflowReconciler reconciles a Workflow object
type WorkflowReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
}

// checkDriverStatus returns true if all registered drivers for the current state completed successfully
func checkDriverStatus(instance *dwsv1alpha1.Workflow) (bool, error) {
	for _, d := range instance.Status.Drivers {
		if d.WatchState == instance.Spec.DesiredState {
			if strings.ToLower(d.Reason) == "error" {
				// Return errors
				return ConditionTrue, myerror.New(d.Message)
			}
			if d.Completed == ConditionFalse {
				// Return not ready
				return ConditionFalse, nil
			}
		}
	}
	return ConditionTrue, nil
}

//+kubebuilder:rbac:groups=dws.cray.hpe.com,resources=workflows,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups=dws.cray.hpe.com,resources=workflows/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=dws.cray.hpe.com,resources=workflows/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Workflow object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.9.2/pkg/reconcile
func (r *WorkflowReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("storage", req.NamespacedName)
	log.Info("Reconciling Workflow")

	// Fetch the Workflow instance
	instance := &dwsv1alpha1.Workflow{}

	err := r.Get(context.TODO(), req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Error(err, "Workflow instance not found")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Could not get instance Workflow")
		return ctrl.Result{}, nil
	}

	updateNeeded := false
	driverDone, err := checkDriverStatus(instance)
	if err != nil {
		// Update Status only if not already in an ERROR state
		if instance.Status.State != instance.Spec.DesiredState ||
			instance.Status.Ready != ConditionFalse ||
			instance.Status.Reason != "ERROR" {
			log.Info("Workflow state transitioning to " + "ERROR")
			instance.Status.State = instance.Spec.DesiredState
			instance.Status.Ready = ConditionFalse
			instance.Status.Reason = "ERROR"
			instance.Status.Message = err.Error()
			updateNeeded = true
		}
	} else {
		// Update Status.State if needed
		if instance.Status.State != instance.Spec.DesiredState {
			log.Info("Workflow state transitioning to " + instance.Spec.DesiredState)
			instance.Status.State = instance.Spec.DesiredState
			updateNeeded = true
		}
		// Set Ready/Reason based on driverDone condition
		if driverDone == ConditionTrue {
			instance.Status.Ready = ConditionTrue
			instance.Status.Reason = "Completed"
			instance.Status.Message = "Workflow " + instance.Status.State + " completed successfully"
			log.Info("Workflow " + instance.Name + " transitioning to ready state " + instance.Status.State)
			updateNeeded = true
		} else {
			// Driver not ready, update Status if not already in DriverWait
			if instance.Status.Reason != "DriverWait" {
				instance.Status.Ready = ConditionFalse
				instance.Status.Reason = "DriverWait"
				instance.Status.Message = "Workflow " + instance.Status.State + " waiting for driver completion"
				log.Info("Workflow " + instance.Name + " State=" + instance.Status.State + " waiting for driver completion")
				updateNeeded = true
			}
		}
	}
	if updateNeeded {
		err = r.Update(context.TODO(), instance)
		if err != nil {
			log.Error(err, "Failed to update Workflow state")
			return ctrl.Result{}, nil
		}
		log.Info("Status was updated", "State", instance.Status.State)
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *WorkflowReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&dwsv1alpha1.Workflow{}).
		Complete(r)
}
