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
	myerror "errors"
	"reflect"
	"runtime"
	"strings"
	"time"

	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	dwsv1alpha1 "github.com/HewlettPackard/dws/api/v1alpha1"
)

const (
	// finalizerDwsWorkflow is the finalizer string used by this controller
	finalizerDwsWorkflow = "dws.cray.hpe.com/workflow"
)

// Define condition values
const (
	ConditionTrue  bool = true
	ConditionFalse bool = false
)

// WorkflowReconciler reconciles a Workflow object
type WorkflowReconciler struct {
	client.Client
	Scheme       *kruntime.Scheme
	Log          logr.Logger
	ChildObjects []dwsv1alpha1.ObjectList
}

// checkDriverStatus returns true if all registered drivers for the current state completed successfully
func checkDriverStatus(instance *dwsv1alpha1.Workflow) (bool, error) {
	for _, d := range instance.Status.Drivers {
		if d.WatchState == instance.Status.State {
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
//+kubebuilder:rbac:groups=dws.cray.hpe.com,resources=computes,verbs=get;create;list;watch;update;patch;delete;deletecollection

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

	// Fetch the Workflow workflow
	workflow := &dwsv1alpha1.Workflow{}

	err := r.Get(ctx, req.NamespacedName, workflow)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Check if the object is being deleted
	if !workflow.GetDeletionTimestamp().IsZero() {
		if !controllerutil.ContainsFinalizer(workflow, finalizerDwsWorkflow) {
			return ctrl.Result{}, nil
		}

		// Wait for all other finalizers to be removed
		if len(workflow.GetFinalizers()) != 1 {
			return ctrl.Result{}, nil
		}

		// Delete all the Computes resources owned by the workflow
		deleteStatus, err := dwsv1alpha1.DeleteChildren(ctx, r.Client, r.ChildObjects, workflow)
		if err != nil {
			return ctrl.Result{}, err
		}

		if deleteStatus == dwsv1alpha1.DeleteRetry {
			return ctrl.Result{}, nil
		}

		controllerutil.RemoveFinalizer(workflow, finalizerDwsWorkflow)
		if err := r.Update(ctx, workflow); err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	// Add the finalizer if it doesn't exist
	if !controllerutil.ContainsFinalizer(workflow, finalizerDwsWorkflow) {
		controllerutil.AddFinalizer(workflow, finalizerDwsWorkflow)
		if err := r.Update(ctx, workflow); err != nil {
			return ctrl.Result{Requeue: true}, nil
		}

		return ctrl.Result{}, nil
	}

	// Need to set Status.State first because the webhook validates this.
	if workflow.Status.State != workflow.Spec.DesiredState {
		log.Info("Workflow state transitioning to " + workflow.Spec.DesiredState)
		workflow.Status.State = workflow.Spec.DesiredState
		workflow.Status.Ready = ConditionFalse
		workflow.Status.Reason = ""
		workflow.Status.Message = ""
		ts := metav1.NowMicro()
		workflow.Status.DesiredStateChange = &ts

		err = r.Update(ctx, workflow)
		if err != nil {
			log.Error(err, "Failed to update Workflow state")
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	// We must create Computes during proposal state
	if workflow.Spec.DesiredState == dwsv1alpha1.StateProposal.String() {
		computes, err := r.createComputes(ctx, workflow, workflow.Name, log)
		if err != nil {
			return ctrl.Result{}, err
		}

		// Ensure the computes reference is set
		cref := v1.ObjectReference{
			Kind:      reflect.TypeOf(dwsv1alpha1.Computes{}).Name(),
			Name:      computes.Name,
			Namespace: computes.Namespace,
		}
		if workflow.Status.Computes != cref {
			log.Info("Updating workflow with Computes")
			workflow.Status.Computes = cref

			err = r.Update(ctx, workflow)
			if err != nil {
				log.Error(err, "Failed to add computes reference")
			}
			return ctrl.Result{}, err
		}
	}

	updateNeeded := false
	driverDone, err := checkDriverStatus(workflow)
	if err != nil {
		// Update Status only if not already in an ERROR state
		if workflow.Status.State != workflow.Spec.DesiredState ||
			workflow.Status.Ready != ConditionFalse ||
			workflow.Status.Reason != "ERROR" {
			log.Info("Workflow state transitioning to " + "ERROR")
			workflow.Status.State = workflow.Spec.DesiredState
			workflow.Status.Ready = ConditionFalse
			workflow.Status.Reason = "ERROR"
			workflow.Status.Message = err.Error()
			updateNeeded = true
		}
	} else {
		// Set Ready/Reason based on driverDone condition
		// All drivers achieving the current desiredStatus means we've achieved the desired state
		if driverDone == ConditionTrue {
			if workflow.Status.Ready != ConditionTrue {
				workflow.Status.Ready = ConditionTrue
				ts := metav1.NowMicro()
				workflow.Status.ReadyChange = &ts
				workflow.Status.ElapsedTimeLastState = ts.Time.Sub(workflow.Status.DesiredStateChange.Time).Round(time.Microsecond).String()
				workflow.Status.Reason = "Completed"
				workflow.Status.Message = "Workflow " + workflow.Status.State + " completed successfully"
				log.Info("Workflow transitioning to ready state " + workflow.Status.State)
				updateNeeded = true
			}
		} else {
			// Driver not ready, update Status if not already in DriverWait
			if workflow.Status.Reason != "DriverWait" {
				workflow.Status.Ready = ConditionFalse
				workflow.Status.Reason = "DriverWait"
				workflow.Status.Message = "Workflow " + workflow.Status.State + " waiting for driver completion"
				log.Info("Workflow state=" + workflow.Status.State + " waiting for driver completion")
				updateNeeded = true
			}
		}
	}
	if updateNeeded {
		err = r.Update(ctx, workflow)
		if err != nil {
			log.Error(err, "Failed to update Workflow state")
			return ctrl.Result{}, nil
		}
		log.Info("Status was updated", "State", workflow.Status.State)
	}

	return ctrl.Result{}, nil
}

func (r *WorkflowReconciler) createComputes(ctx context.Context, wf *dwsv1alpha1.Workflow, name string, log logr.Logger) (*dwsv1alpha1.Computes, error) {

	computes := &dwsv1alpha1.Computes{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: wf.Namespace,
		},
	}

	result, err := ctrl.CreateOrUpdate(ctx, r.Client, computes,
		func() error {
			dwsv1alpha1.AddWorkflowLabels(computes, wf)
			dwsv1alpha1.AddOwnerLabels(computes, wf)

			// Link the Computes to the workflow
			return ctrl.SetControllerReference(wf, computes, r.Scheme)
		})

	if err != nil {
		log.Error(err, "Failed to create or update Computes", "name", computes.Name)
		return nil, err
	}
	if result == controllerutil.OperationResultCreated {
		log.Info("Created Computes", "name", computes.Name)
	} else if result == controllerutil.OperationResultNone {
		// no change
	} else {
		log.Info("Updated Computes", "name", computes.Name)
	}

	return computes, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *WorkflowReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.ChildObjects = []dwsv1alpha1.ObjectList{
		&dwsv1alpha1.ComputesList{},
	}

	maxReconciles := runtime.GOMAXPROCS(0)
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(controller.Options{MaxConcurrentReconciles: maxReconciles}).
		For(&dwsv1alpha1.Workflow{}).
		Owns(&dwsv1alpha1.Computes{}).
		Complete(r)
}
