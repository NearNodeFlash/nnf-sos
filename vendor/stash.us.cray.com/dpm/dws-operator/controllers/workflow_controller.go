/*
Copyright 2021 Hewlett Packard Enterprise Development LP
*/

package controllers

import (
	"context"
	myerror "errors"
	"strings"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	dwsv1alpha1 "stash.us.cray.com/dpm/dws-operator/api/v1alpha1"
)

// Define condition values
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
	log := r.Log.WithValues("Workflow", req.NamespacedName)
	log.Info("Reconciling Workflow")

	// Fetch the Workflow instance
	instance := &dwsv1alpha1.Workflow{}

	err := r.Get(context.TODO(), req.NamespacedName, instance)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
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
			instance.Status.DesiredStateChange = metav1.Now()
			updateNeeded = true
		}
		// Set Ready/Reason based on driverDone condition
		// All drivers achieving the current desiredStatus means we've achieved the desired state
		if driverDone == ConditionTrue {
			if instance.Status.Ready != ConditionTrue {
				instance.Status.Ready = ConditionTrue
				instance.Status.ReadyChange = metav1.Now()
				instance.Status.Reason = "Completed"
				instance.Status.Message = "Workflow " + instance.Status.State + " completed successfully"
				log.Info("Workflow transitioning to ready state " + instance.Status.State)
				updateNeeded = true
			}
		} else {
			// Driver not ready, update Status if not already in DriverWait
			if instance.Status.Reason != "DriverWait" {
				instance.Status.Ready = ConditionFalse
				instance.Status.Reason = "DriverWait"
				instance.Status.Message = "Workflow " + instance.Status.State + " waiting for driver completion"
				log.Info("Workflow state=" + instance.Status.State + " waiting for driver completion")
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
