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
	"fmt"
	"reflect"
	"runtime"
	"time"

	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	dwsv1alpha1 "github.com/HewlettPackard/dws/api/v1alpha1"
	"github.com/HewlettPackard/dws/controllers/metrics"
	"github.com/HewlettPackard/dws/utils/updater"
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
func (r *WorkflowReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, err error) {
	log := r.Log.WithValues("Workflow", req.NamespacedName)
	log.Info("Reconciling Workflow")

	metrics.DwsReconcilesTotal.Inc()

	// Fetch the Workflow workflow
	workflow := &dwsv1alpha1.Workflow{}
	if err := r.Get(ctx, req.NamespacedName, workflow); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Create a status updater that handles the call to r.Update() if any of the fields
	// in workflow.Status{} change. This is necessary since Status is not a subresource
	// of the workflow.
	statusUpdater := updater.NewStatusUpdater[*dwsv1alpha1.WorkflowStatus](workflow)
	defer func() { err = statusUpdater.CloseWithUpdate(ctx, r.Client, err) }()

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
		DeleteStatus, err := dwsv1alpha1.DeleteChildren(ctx, r.Client, r.ChildObjects, workflow)
		if err != nil {
			return ctrl.Result{}, err
		}

		if !DeleteStatus.Complete() {
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
		log.Info("Workflow state transitioning", "state", workflow.Spec.DesiredState)
		workflow.Status.State = workflow.Spec.DesiredState
		workflow.Status.Ready = ConditionFalse
		workflow.Status.Status = dwsv1alpha1.StatusDriverWait
		workflow.Status.Message = ""
		ts := metav1.NowMicro()
		workflow.Status.DesiredStateChange = &ts

		return ctrl.Result{}, nil
	}

	// We must create Computes during proposal state
	if workflow.Spec.DesiredState == dwsv1alpha1.StateProposal {
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

	// If the workflow has already been marked as complete for this state, then
	// we don't need to check the drivers. The drivers can't transition from complete
	// to not complete
	if workflow.Status.Ready == true {
		return ctrl.Result{}, nil
	}

	workflow.Status.Ready = true
	workflow.Status.Status = dwsv1alpha1.StatusCompleted
	workflow.Status.Message = ""

	// Loop through the driver status array and update the workflow
	// status as necessary
	for _, driver := range workflow.Status.Drivers {
		if driver.WatchState != workflow.Status.State {
			continue
		}

		if driver.Completed == false {
			workflow.Status.Ready = false
			workflow.Status.Status = dwsv1alpha1.StatusDriverWait
		}

		if driver.Message != "" {
			workflow.Status.Message = fmt.Sprintf("DW Directive %d: %s", driver.DWDIndex, driver.Message)
		}

		if driver.Status == dwsv1alpha1.StatusError {
			workflow.Status.Status = dwsv1alpha1.StatusError
			break
		}
	}

	if workflow.Status.Ready == true {
		ts := metav1.NowMicro()
		workflow.Status.ReadyChange = &ts
		workflow.Status.ElapsedTimeLastState = ts.Time.Sub(workflow.Status.DesiredStateChange.Time).Round(time.Microsecond).String()
		log.Info("Workflow transitioning to ready", "state", workflow.Status.State)
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

type workflowStatusUpdater struct {
	workflow       *dwsv1alpha1.Workflow
	existingStatus dwsv1alpha1.WorkflowStatus
}

func newWorkflowStatusUpdater(w *dwsv1alpha1.Workflow) *workflowStatusUpdater {
	return &workflowStatusUpdater{
		workflow:       w,
		existingStatus: (*w.DeepCopy()).Status,
	}
}

func (w *workflowStatusUpdater) close(ctx context.Context, r *WorkflowReconciler) error {
	if !reflect.DeepEqual(w.workflow.Status, w.existingStatus) {
		err := r.Update(ctx, w.workflow)
		if !apierrors.IsConflict(err) {
			return err
		}
	}

	return nil
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
