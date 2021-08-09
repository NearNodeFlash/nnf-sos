/*
Copyright 2021 Hewlett Packard Enterprise Development LP
*/

package controllers

import (
	"context"
	"os"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	dwsv1alpha1 "stash.us.cray.com/dpm/dws-operator/api/v1alpha1"
)

// WorkflowReconciler reconciles a Workflow object
type WorkflowReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=dws.cray.hpe.com,resources=workflows,verbs=get;list;watch;update;patch
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

	// Fetch the Workflow instance
	workflow := &dwsv1alpha1.Workflow{}

	err := r.Get(ctx, req.NamespacedName, workflow)
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
		return ctrl.Result{Requeue: true}, nil
	}

	driverID := os.Getenv("WORKFLOW_WEBHOOK_DRIVER_ID")

	for i := range workflow.Status.Drivers {
		driverStatus := &workflow.Status.Drivers[i]

		if driverStatus.DriverID != driverID {
			continue
		}

		if workflow.Status.State != driverStatus.WatchState {
			continue
		}

		if workflow.Status.Ready == true {
			continue
		}

		if driverStatus.Completed == true {
			continue
		}

		driverStatus.Completed = true
	}

	err = r.Update(ctx, workflow)
	if err != nil {
		log.Error(err, "Failed to update Workflow state")
		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *WorkflowReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&dwsv1alpha1.Workflow{}).
		Complete(r)
}
