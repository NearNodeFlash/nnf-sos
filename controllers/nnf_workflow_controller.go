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
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"

	dwsv1alpha1 "github.com/HewlettPackard/dws/api/v1alpha1"
	"github.com/HewlettPackard/dws/utils/dwdparse"
	lusv1alpha1 "github.com/NearNodeFlash/lustre-fs-operator/api/v1alpha1"
	nnfv1alpha1 "github.com/NearNodeFlash/nnf-sos/api/v1alpha1"
	"github.com/NearNodeFlash/nnf-sos/controllers/metrics"
)

const (
	// finalizerNnfWorkflow defines the key used in identifying the
	// storage object as being owned by this NNF Storage Reconciler. This
	// prevents the system from deleting the custom resource until the
	// reconciler has finished in using the resource.
	finalizerNnfWorkflow = "nnf.cray.hpe.com/nnf_workflow"

	// teardownStateLabel is a label applied to NnfAccess resources to determine when
	// they should be deleted. The delete code filters by the current workflow state
	// to only delete the correct NnfAccess resources
	teardownStateLabel = "nnf.cray.hpe.com/teardown_state"
)

type nnfResourceState int

// State enumerations
const (
	resourceExists nnfResourceState = iota
	resourceDeleted
)

var nnfResourceStateStrings = [...]string{
	"exists",
	"deleted",
}

// NnfWorkflowReconciler contains the pieces used by the reconciler
type NnfWorkflowReconciler struct {
	client.Client
	Log          logr.Logger
	Scheme       *kruntime.Scheme
	ChildObjects []dwsv1alpha1.ObjectList
}

//+kubebuilder:rbac:groups=dws.cray.hpe.com,resources=workflows,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups=dws.cray.hpe.com,resources=workflows/finalizers,verbs=update
//+kubebuilder:rbac:groups=dws.cray.hpe.com,resources=directivebreakdowns,verbs=get;create;list;watch;update;patch;delete;deletecollection
//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=directivebreakdowns/status,verbs=get;list;watch
//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfstorages,verbs=get;create;list;watch;update;patch;delete;deletecollection
//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfaccesses,verbs=get;create;list;watch;update;patch;delete;deletecollection
//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfdatamovements,verbs=get;create;list;watch;update;patch;delete;deletecollection
//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfstorageprofiles,verbs=get;create;list;watch;update;patch;delete;deletecollection
//+kubebuilder:rbac:groups=dws.cray.hpe.com,resources=persistentstorageinstances,verbs=get;create;list;watch;update;patch;delete;deletecollection
//+kubebuilder:rbac:groups=dws.cray.hpe.com,resources=servers,verbs=get;create;list;watch;update;patch
//+kubebuilder:rbac:groups=dws.cray.hpe.com,resources=computes,verbs=get;create;list;watch;update;patch
//+kubebuilder:rbac:groups=cray.hpe.com,resources=lustrefilesystems,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Workflow object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.9.2/pkg/reconcile
func (r *NnfWorkflowReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, err error) {
	log := r.Log.WithValues("Workflow", req.NamespacedName)

	metrics.NnfWorkflowReconcilesTotal.Inc()

	// Fetch the Workflow instance
	workflow := &dwsv1alpha1.Workflow{}

	if err := r.Get(ctx, req.NamespacedName, workflow); err != nil {
		// ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	statusUpdater := newWorkflowStatusUpdater(workflow)
	defer func() {
		updateErr := statusUpdater.close(ctx, r)
		if err == nil && updateErr != nil {
			err = updateErr
		}
	}()

	driverID := os.Getenv("DWS_DRIVER_ID")

	// Check if the object is being deleted
	if !workflow.GetDeletionTimestamp().IsZero() {
		log.Info("Deleting workflow...")

		if !controllerutil.ContainsFinalizer(workflow, finalizerNnfWorkflow) {
			return ctrl.Result{}, nil
		}

		deleteStatus, err := dwsv1alpha1.DeleteChildren(ctx, r.Client, r.ChildObjects, workflow)
		if err != nil {
			return ctrl.Result{}, err
		}

		if deleteStatus == dwsv1alpha1.DeleteRetry {
			return ctrl.Result{}, nil
		}

		controllerutil.RemoveFinalizer(workflow, finalizerNnfWorkflow)
		if err := r.Update(ctx, workflow); err != nil {
			if !apierrors.IsConflict(err) {
				return ctrl.Result{}, err
			}

			return ctrl.Result{Requeue: true}, nil
		}

		return ctrl.Result{}, nil
	}

	// Add the finalizer if it doesn't exist
	if !controllerutil.ContainsFinalizer(workflow, finalizerNnfWorkflow) {
		controllerutil.AddFinalizer(workflow, finalizerNnfWorkflow)
		if err := r.Update(ctx, workflow); err != nil {
			if !apierrors.IsConflict(err) {
				return ctrl.Result{}, err
			}

			return ctrl.Result{Requeue: true}, nil
		}

		return ctrl.Result{}, nil
	}

	// If the dws-operator has yet to set the Status.State, just return and wait for it.
	if workflow.Status.State == "" || workflow.Status.State != workflow.Spec.DesiredState {
		return ctrl.Result{}, nil
	}

	// If the workflow has already achieved its desired state, then there's no work to do
	if workflow.Status.Ready {
		return ctrl.Result{}, nil
	}

	// Create a list of the driverStatus array elements that correspond to the current state
	// of the workflow and are targeted for the Rabbit driver
	driverList := []*dwsv1alpha1.WorkflowDriverStatus{}

	for i := range workflow.Status.Drivers {
		driverStatus := &workflow.Status.Drivers[i]

		if driverStatus.DriverID != driverID {
			continue
		}
		if workflow.Status.State != driverStatus.WatchState {
			continue
		}
		if driverStatus.Completed {
			continue
		}

		driverList = append(driverList, driverStatus)
	}

	startFunctions := map[string]func(*NnfWorkflowReconciler, context.Context, *dwsv1alpha1.Workflow, int) (*ctrl.Result, error){
		dwsv1alpha1.StateProposal.String(): (*NnfWorkflowReconciler).startProposalState,
		dwsv1alpha1.StateSetup.String():    (*NnfWorkflowReconciler).startSetupState,
		dwsv1alpha1.StateDataIn.String():   (*NnfWorkflowReconciler).startDataInOutState,
		dwsv1alpha1.StatePreRun.String():   (*NnfWorkflowReconciler).startPreRunState,
		dwsv1alpha1.StatePostRun.String():  (*NnfWorkflowReconciler).startPostRunState,
		dwsv1alpha1.StateDataOut.String():  (*NnfWorkflowReconciler).startDataInOutState,
		dwsv1alpha1.StateTeardown.String(): (*NnfWorkflowReconciler).startTeardownState,
	}

	// Call the correct "start" function based on workflow state for each directive that has registered for
	// it. The "start" function does the initial work of setting up and creating the appropriate child resources.
	for _, driverStatus := range driverList {
		log.Info("Start", "State", workflow.Status.State, "index", driverStatus.DWDIndex, "directive", workflow.Spec.DWDirectives[driverStatus.DWDIndex])
		result, err := startFunctions[workflow.Status.State](r, ctx, workflow, driverStatus.DWDIndex)
		if err != nil {
			handleWorkflowError(err, driverStatus)

			log.Info("Start error", "State", workflow.Status.State, "index", driverStatus.DWDIndex, "Message", err.Error())
			return ctrl.Result{}, err
		}

		driverStatus.Status = dwsv1alpha1.StatusRunning
		driverStatus.Message = ""
		driverStatus.Error = ""

		if result != nil {
			log.Info("Start wait", "State", workflow.Status.State, "index", driverStatus.DWDIndex)
			return *result, nil
		}

		log.Info("Start done", "State", workflow.Status.State, "index", driverStatus.DWDIndex)
	}

	finishFunctions := map[string]func(*NnfWorkflowReconciler, context.Context, *dwsv1alpha1.Workflow, int) (*ctrl.Result, error){
		dwsv1alpha1.StateProposal.String(): (*NnfWorkflowReconciler).finishProposalState,
		dwsv1alpha1.StateSetup.String():    (*NnfWorkflowReconciler).finishSetupState,
		dwsv1alpha1.StateDataIn.String():   (*NnfWorkflowReconciler).finishDataInOutState,
		dwsv1alpha1.StatePreRun.String():   (*NnfWorkflowReconciler).finishPreRunState,
		dwsv1alpha1.StatePostRun.String():  (*NnfWorkflowReconciler).finishPostRunState,
		dwsv1alpha1.StateDataOut.String():  (*NnfWorkflowReconciler).finishDataInOutState,
		dwsv1alpha1.StateTeardown.String(): (*NnfWorkflowReconciler).finishTeardownState,
	}

	// Call the correct "finish" function based on workflow state for each directive that has registered for
	// it. The "finish" functions wait for the child resources to complete all their work and do any teardown
	// necessary.
	for _, driverStatus := range driverList {
		log.Info("Finish", "State", workflow.Status.State, "index", driverStatus.DWDIndex, "directive", workflow.Spec.DWDirectives[driverStatus.DWDIndex])
		result, err := finishFunctions[workflow.Status.State](r, ctx, workflow, driverStatus.DWDIndex)
		if err != nil {
			handleWorkflowError(err, driverStatus)

			log.Info("Finish error", "State", workflow.Status.State, "index", driverStatus.DWDIndex, "Message", err.Error())

			return ctrl.Result{}, err
		}

		driverStatus.Status = dwsv1alpha1.StatusRunning
		driverStatus.Message = ""
		driverStatus.Error = ""

		if result != nil {
			log.Info("Finish wait", "State", workflow.Status.State, "index", driverStatus.DWDIndex)
			return *result, nil
		}

		log.Info("Finish done", "State", workflow.Status.State, "index", driverStatus.DWDIndex)

		ts := metav1.NowMicro()
		driverStatus.Status = dwsv1alpha1.StatusCompleted
		driverStatus.Message = ""
		driverStatus.Error = ""
		driverStatus.CompleteTime = &ts
		driverStatus.Completed = true
	}

	return ctrl.Result{}, nil
}

func (r *NnfWorkflowReconciler) startProposalState(ctx context.Context, workflow *dwsv1alpha1.Workflow, index int) (*ctrl.Result, error) {
	log := r.Log.WithValues("Workflow", client.ObjectKeyFromObject(workflow), "Index", index)
	dwArgs, _ := dwdparse.BuildArgsMap(workflow.Spec.DWDirectives[index])

	if err := r.validateWorkflow(ctx, workflow); err != nil {
		log.Error(err, "Unable to validate workflow")
		return nil, nnfv1alpha1.NewWorkflowError("Unable to validate DW directives").WithFatal().WithError(err)
	}

	// only jobdw, persistentdw, and create_persistent need a directive breakdown
	if dwArgs["command"] != "jobdw" && dwArgs["command"] != "persistentdw" && dwArgs["command"] != "create_persistent" {
		return nil, nil
	}

	directiveBreakdown, err := r.generateDirectiveBreakdown(ctx, index, workflow, log)
	if err != nil {
		return nil, nnfv1alpha1.NewWorkflowError("Unable to start parsing DW directive").WithError(err)
	}

	if directiveBreakdown == nil {
		return &ctrl.Result{Requeue: true}, nil
	}

	directiveBreakdownReference := v1.ObjectReference{
		Kind:      reflect.TypeOf(dwsv1alpha1.DirectiveBreakdown{}).Name(),
		Name:      directiveBreakdown.Name,
		Namespace: directiveBreakdown.Namespace,
	}

	found := false
	for _, reference := range workflow.Status.DirectiveBreakdowns {
		if reference == directiveBreakdownReference {
			found = true
		}
	}

	if !found {
		workflow.Status.DirectiveBreakdowns = append(workflow.Status.DirectiveBreakdowns, directiveBreakdownReference)
	}

	return nil, nil
}

func (r *NnfWorkflowReconciler) finishProposalState(ctx context.Context, workflow *dwsv1alpha1.Workflow, index int) (*ctrl.Result, error) {
	log := r.Log.WithValues("Workflow", client.ObjectKeyFromObject(workflow), "Index", index)
	dwArgs, _ := dwdparse.BuildArgsMap(workflow.Spec.DWDirectives[index])

	// only jobdw, persistentdw, and create_persistent need a directive breakdown
	if dwArgs["command"] != "jobdw" && dwArgs["command"] != "persistentdw" && dwArgs["command"] != "create_persistent" {
		return nil, nil
	}

	directiveBreakdown := &dwsv1alpha1.DirectiveBreakdown{
		ObjectMeta: metav1.ObjectMeta{
			Name:      indexedResourceName(workflow, index),
			Namespace: workflow.GetNamespace(),
		},
	}

	err := r.Get(ctx, client.ObjectKeyFromObject(directiveBreakdown), directiveBreakdown)
	if err != nil {
		log.Info("Failed to get DirectiveBreakdown", "name", directiveBreakdown.GetName(), "error", err.Error())
		return nil, nnfv1alpha1.NewWorkflowError("Unable to finish parsing DW directive").WithError(err)
	}

	// Wait for the breakdown to be ready
	if directiveBreakdown.Status.Ready != ConditionTrue {
		return &ctrl.Result{}, nil
	}

	return nil, nil
}

// Validate the workflow and return any error found
func (r *NnfWorkflowReconciler) validateWorkflow(ctx context.Context, wf *dwsv1alpha1.Workflow) error {

	var createPersistentCount, deletePersistentCount, directiveCount int
	for _, directive := range wf.Spec.DWDirectives {

		directiveCount++
		// The webhook validated directives, ignore errors
		dwArgs, _ := dwdparse.BuildArgsMap(directive)
		command := dwArgs["command"]

		switch command {
		case "jobdw":

		case "copy_in", "copy_out":
			if err := r.validateStagingDirective(ctx, wf, directive); err != nil {
				return nnfv1alpha1.NewWorkflowError("Invalid staging Directive: " + directive).WithFatal().WithError(err)
			}

		case "create_persistent":
			createPersistentCount++

		case "destroy_persistent":
			deletePersistentCount++

		case "persistentdw":
			if err := r.validatePersistentInstanceDirective(ctx, wf, directive); err != nil {
				return nnfv1alpha1.NewWorkflowError("Could not validate persistent instance: " + directive).WithFatal().WithError(err)
			}
		}
	}

	if directiveCount > 1 {
		// Ensure create_persistent or destroy_persistent are singletons in the workflow
		if createPersistentCount+deletePersistentCount > 0 {
			return nnfv1alpha1.NewWorkflowError("Only a single create_persistent or destroy_persistent directive is allowed per workflow").WithFatal()
		}
	}

	return nil
}

// validateStagingDirective validates the staging copy_in/copy_out directives.
func (r *NnfWorkflowReconciler) validateStagingDirective(ctx context.Context, wf *dwsv1alpha1.Workflow, directive string) error {
	// Validate staging directive of the form...
	//   #DW copy_in source=[SOURCE] destination=[DESTINATION]
	//   #DW copy_out source=[SOURCE] destination=[DESTINATION]

	// For each copy_in/copy_out directive
	//   Make sure all $JOB_DW_ references point to job storage instance names
	//   Make sure all $PERSISTENT_DW references an existing persistentdw instance
	//   Otherwise, make sure source/destination is prefixed with a valid global lustre file system
	validateStagingArgument := func(arg string) error {
		name, _ := splitStagingArgumentIntoNameAndPath(arg)
		if strings.HasPrefix(arg, "$JOB_DW_") {
			if findDirectiveIndexByName(wf, name) == -1 {
				return nnfv1alpha1.NewWorkflowError(fmt.Sprintf("Job storage instance '%s' not found", name)).WithFatal()
			}
		} else if strings.HasPrefix(arg, "$PERSISTENT_DW_") {
			if err := r.validatePersistentInstanceByName(ctx, name, wf.Namespace); err != nil {
				return nnfv1alpha1.NewWorkflowError(fmt.Sprintf("Persistent storage instance '%s' not found", name)).WithFatal()
			}
			if findDirectiveIndexByName(wf, name) == -1 {
				return nnfv1alpha1.NewWorkflowError(fmt.Sprintf("persistentdw directive mentioning '%s' not found", name)).WithFatal()
			}
		} else {
			if r.findLustreFileSystemForPath(ctx, arg, r.Log) == nil {
				return nnfv1alpha1.NewWorkflowError(fmt.Sprintf("Global Lustre file system containing '%s' not found", arg)).WithFatal()
			}
		}

		return nil
	}

	args, err := dwdparse.BuildArgsMap(directive)
	if err != nil {
		return nnfv1alpha1.NewWorkflowError("Invalid DW directive: " + directive).WithFatal()
	}

	if err := validateStagingArgument(args["source"]); err != nil {
		return err
	}

	if err := validateStagingArgument(args["destination"]); err != nil {
		return err
	}

	return nil
}

// validatePersistentInstance validates the persistentdw directive.
func (r *NnfWorkflowReconciler) validatePersistentInstanceDirective(ctx context.Context, wf *dwsv1alpha1.Workflow, directive string) error {
	// Validate that the persistent instance is available and not in the process of being deleted
	args, err := dwdparse.BuildArgsMap(directive)
	if err != nil {
		return nnfv1alpha1.NewWorkflowError("Invalid DW directive: " + directive).WithFatal()
	}

	return r.validatePersistentInstanceByName(ctx, args["name"], wf.Namespace)
}

// validatePersistentInstance validates the persistentdw directive.
func (r *NnfWorkflowReconciler) validatePersistentInstanceByName(ctx context.Context, name string, namespace string) error {
	psi, err := r.getPersistentStorageInstance(ctx, name, namespace)
	if err != nil {
		return err
	}

	if !psi.DeletionTimestamp.IsZero() {
		return nnfv1alpha1.NewWorkflowError("Persistent storage instance " + name + " is deleting").WithFatal()
	}

	return nil
}

// Retrieve the persistent storage instance with the specified name
func (r *NnfWorkflowReconciler) getPersistentStorageInstance(ctx context.Context, name string, namespace string) (*dwsv1alpha1.PersistentStorageInstance, error) {
	psi := &dwsv1alpha1.PersistentStorageInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	err := r.Get(ctx, client.ObjectKeyFromObject(psi), psi)
	return psi, err
}

// generateDirectiveBreakdown creates a DirectiveBreakdown for any #DW directive that needs to specify storage
// or compute node information to the WLM (jobdw, create_persistent, persistentdw)
func (r *NnfWorkflowReconciler) generateDirectiveBreakdown(ctx context.Context, dwIndex int, workflow *dwsv1alpha1.Workflow, log logr.Logger) (*dwsv1alpha1.DirectiveBreakdown, error) {

	// DWDirectives that we need to generate directiveBreakdowns for look like this:
	//  #DW command            arguments...
	//  --- -----------------  -------------------------------------------
	// "#DW jobdw              type=raw    capacity=9TB name=thisIsReallyRaw"
	// "#DW jobdw              type=xfs    capacity=9TB name=thisIsXFS"
	// "#DW jobdw              type=gfs2   capacity=9GB name=thisIsGfs2"
	// "#DW jobdw              type=lustre capacity=9TB name=thisIsLustre"
	// "#DW create_persistent  type=lustre capacity=9TB name=thisIsPersistent"
	// "#DW persistentdw       name=thisIsPersistent"

	// #DW commands that require a dwDirectiveBreakdown
	breakDownCommands := []string{
		"jobdw",
		"create_persistent",
		"persistentdw",
	}

	directive := workflow.Spec.DWDirectives[dwIndex]
	dwArgs, _ := dwdparse.BuildArgsMap(directive)
	for _, breakThisDown := range breakDownCommands {
		// We care about the commands that generate a breakdown
		if breakThisDown == dwArgs["command"] {
			dwdName := indexedResourceName(workflow, dwIndex)
			directiveBreakdown := &dwsv1alpha1.DirectiveBreakdown{
				ObjectMeta: metav1.ObjectMeta{
					Name:      dwdName,
					Namespace: workflow.Namespace,
				},
			}

			result, err := ctrl.CreateOrUpdate(ctx, r.Client, directiveBreakdown,
				// Mutate function to fill in a directiveBreakdown
				func() error {
					dwsv1alpha1.AddWorkflowLabels(directiveBreakdown, workflow)
					dwsv1alpha1.AddOwnerLabels(directiveBreakdown, workflow)
					addDirectiveIndexLabel(directiveBreakdown, dwIndex)

					directiveBreakdown.Spec.Directive = directive

					// Link the directive breakdown to the workflow
					return ctrl.SetControllerReference(workflow, directiveBreakdown, r.Scheme)
				})

			if err != nil {
				log.Error(err, "failed to create or update DirectiveBreakdown", "name", directiveBreakdown.Name)
				return nil, fmt.Errorf("CreateOrUpdate failed for DirectiveBreakdown %v: %w", client.ObjectKeyFromObject(directiveBreakdown), err)
			}

			if result == controllerutil.OperationResultCreated {
				log.Info("Created DirectiveBreakdown", "name", directiveBreakdown.Name)
			} else if result == controllerutil.OperationResultNone {
				// no change
			} else {
				log.Info("Updated DirectiveBreakdown", "name", directiveBreakdown.Name)
			}

			// The directive's command has been matched, no need to look at the other breakDownCommands
			return directiveBreakdown, nil
		}
	}

	return nil, nil
}

func (r *NnfWorkflowReconciler) startSetupState(ctx context.Context, workflow *dwsv1alpha1.Workflow, index int) (*ctrl.Result, error) {
	log := r.Log.WithValues("Workflow", client.ObjectKeyFromObject(workflow), "Index", index)
	dwArgs, _ := dwdparse.BuildArgsMap(workflow.Spec.DWDirectives[index])

	// Only jobdw and create_persistent need to create an NnfStorage resource
	if dwArgs["command"] != "create_persistent" && dwArgs["command"] != "jobdw" {
		return nil, nil
	}

	// Chain through the DirectiveBreakdown to the Servers object
	dbd := &dwsv1alpha1.DirectiveBreakdown{
		ObjectMeta: metav1.ObjectMeta{
			Name:      indexedResourceName(workflow, index),
			Namespace: workflow.Namespace,
		},
	}
	err := r.Get(ctx, client.ObjectKeyFromObject(dbd), dbd)
	if err != nil {
		log.Info("Unable to get directiveBreakdown", "dbd", client.ObjectKeyFromObject(dbd), "Message", err)
		err = fmt.Errorf("Unable to get DirectiveBreakdown %v: %w", client.ObjectKeyFromObject(dbd), err)
		return nil, nnfv1alpha1.NewWorkflowError("Could not read allocation request").WithError(err)
	}

	s := &dwsv1alpha1.Servers{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dbd.Status.Storage.Reference.Name,
			Namespace: dbd.Status.Storage.Reference.Namespace,
		},
	}
	err = r.Get(ctx, client.ObjectKeyFromObject(s), s)
	if err != nil {
		log.Info("Unable to get servers", "servers", client.ObjectKeyFromObject(s), "Message", err)
		err = fmt.Errorf("Unable to get Servers %v: %w", client.ObjectKeyFromObject(s), err)
		return nil, nnfv1alpha1.NewWorkflowError("Could not read allocation request").WithError(err)
	}

	if _, present := os.LookupEnv("RABBIT_TEST_ENV_BYPASS_SERVER_STORAGE_CHECK"); !present {
		if len(dbd.Status.Storage.AllocationSets) != 0 && len(dbd.Status.Storage.AllocationSets) != len(s.Spec.AllocationSets) {
			err := fmt.Errorf("Servers resource does not meet storage requirements for directive '%s'", dbd.Spec.Directive)
			return nil, nnfv1alpha1.NewWorkflowError("Allocation request does not meet directive requirements").WithFatal().WithError(err)
		}

		for _, breakdownAllocationSet := range dbd.Status.Storage.AllocationSets {
			found := false
			for _, serverAllocationsSet := range s.Spec.AllocationSets {
				if breakdownAllocationSet.Label != serverAllocationsSet.Label {
					continue
				}

				found = true

				if serverAllocationsSet.AllocationSize < breakdownAllocationSet.MinimumCapacity {
					err := fmt.Errorf("Allocation set %s specified insufficient capacity", breakdownAllocationSet.Label)
					return nil, nnfv1alpha1.NewWorkflowError("Allocation request does not meet directive requirements").WithFatal().WithError(err)
				}
			}

			if found == false {
				err := fmt.Errorf("Allocation set %s not found in Servers resource", breakdownAllocationSet.Label)
				return nil, nnfv1alpha1.NewWorkflowError("Allocation request does not meet directive requirements").WithFatal().WithError(err)
			}
		}
	}

	_, err = r.createNnfStorage(ctx, workflow, s, index, log)
	if err != nil {
		if apierrors.IsConflict(err) {
			return &ctrl.Result{Requeue: true}, nil
		}

		log.Info("Failed to create nnf storage", "Message", err)
		err = fmt.Errorf("Could not create NnfStorage %w", err)
		return nil, nnfv1alpha1.NewWorkflowError("Could not create allocation").WithError(err)
	}

	return nil, nil
}

func (r *NnfWorkflowReconciler) finishSetupState(ctx context.Context, workflow *dwsv1alpha1.Workflow, index int) (*ctrl.Result, error) {
	name, namespace := getStorageReferenceNameFromWorkflowActual(workflow, index)

	// Check whether the NnfStorage has finished creating the storage.
	nnfStorage := &nnfv1alpha1.NnfStorage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	if err := r.Get(ctx, client.ObjectKeyFromObject(nnfStorage), nnfStorage); err != nil {
		err = fmt.Errorf("Could not get NnfStorage %v: %w", client.ObjectKeyFromObject(nnfStorage), err)
		return nil, nnfv1alpha1.NewWorkflowError("Could not create allocation").WithError(err)
	}

	// If the Status section has not been filled in yet, exit and wait.
	if len(nnfStorage.Status.AllocationSets) != len(nnfStorage.Spec.AllocationSets) {
		// RequeueAfter is necessary for persistent storage that isn't owned by this workflow
		return &ctrl.Result{RequeueAfter: time.Second * time.Duration(2)}, nil
	}

	var complete bool = true
	// Status section should be usable now, check for Ready
	for _, set := range nnfStorage.Status.AllocationSets {
		if set.Status != "Ready" {
			complete = false
		}
	}

	if nnfStorage.Status.Error != nil {
		return nil, nnfv1alpha1.NewWorkflowError("Could not create allocation").WithError(nnfStorage.Status.Error)
	}

	if !complete {
		// RequeueAfter is necessary for persistent storage that isn't owned by this workflow
		return &ctrl.Result{RequeueAfter: time.Second * time.Duration(2)}, nil
	}

	return nil, nil
}

func (r *NnfWorkflowReconciler) createNnfStorage(ctx context.Context, workflow *dwsv1alpha1.Workflow, s *dwsv1alpha1.Servers, index int, log logr.Logger) (*nnfv1alpha1.NnfStorage, error) {
	nnfStorage := &nnfv1alpha1.NnfStorage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      s.Name,
			Namespace: s.Namespace,
		},
	}

	dwArgs, err := dwdparse.BuildArgsMap(workflow.Spec.DWDirectives[index])
	if err != nil {
		return nil, nnfv1alpha1.NewWorkflowError("Invalid DW directive: " + workflow.Spec.DWDirectives[index]).WithFatal()
	}

	pinnedName, pinnedNamespace := getStorageReferenceNameFromWorkflowActual(workflow, index)
	nnfStorageProfile, err := findPinnedProfile(ctx, r.Client, pinnedNamespace, pinnedName)
	if err != nil {
		log.Error(err, "Unable to find pinned NnfStorageProfile", "name", pinnedName)
		return nil, fmt.Errorf("Could not find pinned NnfStorageProfile %v: %w", types.NamespacedName{Name: pinnedName, Namespace: pinnedNamespace}, err)
	}

	var owner metav1.Object = workflow
	if dwArgs["command"] == "create_persistent" {
		psi, err := r.findPersistentInstance(ctx, workflow, dwArgs["name"])
		if err != nil {
			return nil, fmt.Errorf("Could not find PersistentStorageInstance %v for 'create_persistent' directive: %w", dwArgs["name"], err)
		}

		owner = psi
	}

	result, err := ctrl.CreateOrUpdate(ctx, r.Client, nnfStorage,
		func() error {
			dwsv1alpha1.AddWorkflowLabels(nnfStorage, workflow)
			dwsv1alpha1.AddOwnerLabels(nnfStorage, owner)
			addDirectiveIndexLabel(nnfStorage, index)

			nnfStorage.Spec.FileSystemType = dwArgs["type"]

			// Need to remove all of the AllocationSets in the NnfStorage object before we begin
			nnfStorage.Spec.AllocationSets = []nnfv1alpha1.NnfStorageAllocationSetSpec{}

			// Iterate the Servers data elements to pull out the allocation sets for the server
			for i := range s.Spec.AllocationSets {
				nnfAllocSet := nnfv1alpha1.NnfStorageAllocationSetSpec{}

				nnfAllocSet.Name = s.Spec.AllocationSets[i].Label
				nnfAllocSet.Capacity = s.Spec.AllocationSets[i].AllocationSize
				if dwArgs["type"] == "lustre" {
					nnfAllocSet.NnfStorageLustreSpec.TargetType = strings.ToUpper(s.Spec.AllocationSets[i].Label)
					nnfAllocSet.NnfStorageLustreSpec.BackFs = "zfs"
					charsWanted := 8
					if len(dwArgs["name"]) < charsWanted {
						charsWanted = len(dwArgs["name"])
					}
					nnfAllocSet.NnfStorageLustreSpec.FileSystemName = dwArgs["name"][:charsWanted]
					lustreData := mergeLustreStorageDirectiveAndProfile(dwArgs, nnfStorageProfile)
					if len(lustreData.ExternalMGS) > 0 {
						nnfAllocSet.NnfStorageLustreSpec.ExternalMgsNid = lustreData.ExternalMGS
					}
				}

				// Create Nodes for this allocation set.
				for _, storage := range s.Spec.AllocationSets[i].Storage {
					node := nnfv1alpha1.NnfStorageAllocationNodes{Name: storage.Name, Count: storage.AllocationCount}
					nnfAllocSet.Nodes = append(nnfAllocSet.Nodes, node)
				}

				nnfStorage.Spec.AllocationSets = append(nnfStorage.Spec.AllocationSets, nnfAllocSet)
			}

			// The Servers object owns the NnfStorage object, so it will be garbage collected when the
			// Server object is deleted.
			return ctrl.SetControllerReference(owner, nnfStorage, r.Scheme)
		})

	if err != nil {
		log.Error(err, "Failed to create or update NnfStorage", "name", nnfStorage.Name)
		return nnfStorage, fmt.Errorf("CreateOrUpdate failed for NnfStorage %v: %w", client.ObjectKeyFromObject(nnfStorage), err)
	}

	if result == controllerutil.OperationResultCreated {
		log.Info("Created NnfStorage", "name", nnfStorage.Name)
	} else if result == controllerutil.OperationResultNone {
		// no change
	} else {
		log.Info("Updated NnfStorage", "name", nnfStorage.Name)
	}

	return nnfStorage, nil
}

func (r *NnfWorkflowReconciler) startDataInOutState(ctx context.Context, workflow *dwsv1alpha1.Workflow, index int) (*ctrl.Result, error) {
	log := r.Log.WithValues("Workflow", client.ObjectKeyFromObject(workflow), "Index", index)

	dwArgs, err := dwdparse.BuildArgsMap(workflow.Spec.DWDirectives[index])
	if err != nil {
		return nil, nnfv1alpha1.NewWorkflowError("Invalid DW directive: " + workflow.Spec.DWDirectives[index]).WithFatal()
	}

	// NOTE: We don't need to check for the occurrence of a source or destination parameters since these are required fields and validated through the webhook
	// NOTE: We don't need to validate destination has $JOB_DW_ since this is done during the proposal stage

	// copy_in needs to handle four cases
	// 1. Lustre to JobStorageInstance                              #DW copy_in source=[path] destination=$JOB_DW_[name]/[path]
	// 2. Lustre to PersistentStorageInstance                       #DW copy_in source=[path] destination=$PERSISTENT_DW_[name]/[path]
	// 3. PersistentStorageInstance to JobStorageInstance           #DW copy_in source=$PERSISTENT_DW_[name]/[path] destination=$JOB_DW_[name]/[path]
	// 4. PersistentStorageInstance to PersistentStorageInstance    #DW copy_in source=$PERSISTENT_DW_[name]/[path] destination=$PERSISTENT_DW_[name]/[path]
	// copy_out is the same, but typically source and destination are reversed

	// Prepare the provided staging parameter for data-movement. Param is the source/destination value from the #DW copy_in/copy_out directive; based
	// on the param prefix we determine the storage instance and access requirements for data movement.
	prepareStagingArgumentFn := func(param string) (*corev1.ObjectReference, *nnfv1alpha1.NnfAccess, *ctrl.Result, error) {
		var storageReference *corev1.ObjectReference

		name, _ := splitStagingArgumentIntoNameAndPath(param)

		// If param refers to a Job or Persistent storage type, find the NNF Storage that is backing
		// this directive line.
		if strings.HasPrefix(param, "$JOB_DW_") || strings.HasPrefix(param, "$PERSISTENT_DW_") {

			// Find the parent directive index that corresponds to this copy_in/copy_out directive
			parentDwIndex := findDirectiveIndexByName(workflow, name)
			if parentDwIndex < 0 {
				return nil, nil, nil, nnfv1alpha1.NewWorkflowError("No directive matching '" + name + "' found in workflow").WithFatal()
			}

			// If directive specifies a persistent storage instance, `name` will be the nnfStorageName
			// Otherwise it matches the workflow with the directive index as a suffix
			var nnfStorageName string
			if strings.HasPrefix(param, "$PERSISTENT_DW_") {
				nnfStorageName = name
			} else {
				nnfStorageName = indexedResourceName(workflow, parentDwIndex)
			}

			storage := &nnfv1alpha1.NnfStorage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      nnfStorageName,
					Namespace: workflow.Namespace,
				},
			}

			if err := r.Get(ctx, client.ObjectKeyFromObject(storage), storage); err != nil {
				return nil, nil, nil, fmt.Errorf("Could not get NnfStorage %v: %w", client.ObjectKeyFromObject(storage), err)
			}

			storageReference = &corev1.ObjectReference{
				Kind:      reflect.TypeOf(nnfv1alpha1.NnfStorage{}).Name(),
				Name:      storage.Name,
				Namespace: storage.Namespace,
			}

			fsType := storage.Spec.FileSystemType

			// Find the desired workflow teardown state for the NNF Access. This instructs the workflow
			// when to teardown an NNF Access for the servers
			teardownState := ""
			if dwArgs["command"] == "copy_in" {
				teardownState = dwsv1alpha1.StateDataIn.String()

				if fsType == "gfs2" || fsType == "lustre" {
					teardownState = dwsv1alpha1.StatePostRun.String()

					if findCopyOutDirectiveIndexByName(workflow, name) >= 0 {
						teardownState = dwsv1alpha1.StateDataOut.String()
					}
				}
			} else if dwArgs["command"] == "copy_out" {
				teardownState = dwsv1alpha1.StateDataOut.String()
			}

			// Setup NNF Access for the NNF Servers so we can run data movement on them.
			access, err := r.setupNnfAccessForServers(ctx, storage, workflow, index, parentDwIndex, teardownState, log)
			if err != nil {
				return storageReference, access, nil, nnfv1alpha1.NewWorkflowError("Could not create data movement mount points").WithError(err)
			}

			// Wait for accesses to go ready
			if access.Status.Ready == false {
				return nil, access, &ctrl.Result{}, nil
			}

			return storageReference, access, nil, nil

		} else if lustre := r.findLustreFileSystemForPath(ctx, param, r.Log); lustre != nil {
			storageReference = &corev1.ObjectReference{
				Kind:      reflect.TypeOf(lusv1alpha1.LustreFileSystem{}).Name(),
				Name:      lustre.Name,
				Namespace: lustre.Namespace,
			}

			return storageReference, nil, nil, nil
		}

		return nil, nil, nil, nnfv1alpha1.NewWorkflowError(fmt.Sprintf("Staging parameter '%s' is invalid", param)).WithFatal()
	}

	sourceStorage, sourceAccess, result, err := prepareStagingArgumentFn(dwArgs["source"])
	if err != nil {
		return nil, nnfv1alpha1.NewWorkflowError("Could not prepare data movement resources").WithError(err)
	} else if result != nil {
		return result, nil
	}

	destStorage, destAccess, result, err := prepareStagingArgumentFn(dwArgs["destination"])
	if err != nil {
		return nil, nnfv1alpha1.NewWorkflowError("Could not prepare data movement resources").WithError(err)
	} else if result != nil {
		return result, nil
	}

	// Wait for accesses to go ready
	for _, access := range []*nnfv1alpha1.NnfAccess{sourceAccess, destAccess} {
		if access != nil {
			if err := r.Get(ctx, client.ObjectKeyFromObject(access), access); err != nil {
				return nil, fmt.Errorf("Could not get NnfAccess %v: %w", client.ObjectKeyFromObject(access), err)
			}

			if access.Status.Ready == false {
				return &ctrl.Result{}, nil
			}
		}
	}

	// Retrieve the target storage that is to perform the data movement.
	// For copy_in, the destination is the Rabbit and therefore the target
	// For copy_out, the source is the Rabbit and therefore the target

	var targetStorageRef *corev1.ObjectReference
	if workflow.Spec.DesiredState == dwsv1alpha1.StateDataIn.String() {
		targetStorageRef = destStorage
	} else {
		targetStorageRef = sourceStorage
	}

	targetStorage := &nnfv1alpha1.NnfStorage{}
	if err := r.Get(ctx, types.NamespacedName{Name: targetStorageRef.Name, Namespace: targetStorageRef.Namespace}, targetStorage); err != nil {
		return nil, nnfv1alpha1.NewWorkflowError("Data Movement: Failed to retrieve NNF Storage").WithError(err)
	}

	_, source := splitStagingArgumentIntoNameAndPath(dwArgs["source"])
	_, dest := splitStagingArgumentIntoNameAndPath(dwArgs["destination"])

	fsType := targetStorage.Spec.FileSystemType

	getRabbitRelativePath := func(storageRef *corev1.ObjectReference, access *nnfv1alpha1.NnfAccess, path string, index int) string {

		if storageRef.Kind == reflect.TypeOf(nnfv1alpha1.NnfStorage{}).Name() {
			switch fsType {
			case "xfs", "gfs2":
				return filepath.Join(access.Spec.MountPathPrefix, strconv.Itoa(index), path)
			case "lustre":
				return access.Spec.MountPath + path
			}
		}

		return path
	}

	switch fsType {
	case "xfs", "gfs2":

		// XFS & GFS2 require the individual rabbit nodes are performing the data movement.

		if len(targetStorage.Spec.AllocationSets) != 1 {
			msg := fmt.Sprintf("Data Movement: File System %s has unexpected allocation sets %d", fsType, len(targetStorage.Spec.AllocationSets))
			return nil, nnfv1alpha1.NewWorkflowError(msg).WithFatal()
		}

		nodes := targetStorage.Spec.AllocationSets[0].Nodes

		for _, node := range nodes {

			for i := 0; i < node.Count; i++ {
				dm := &nnfv1alpha1.NnfDataMovement{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("%s-%d", indexedResourceName(workflow, index), i),
						Namespace: node.Name,
					},
					Spec: nnfv1alpha1.NnfDataMovementSpec{
						Source: &nnfv1alpha1.NnfDataMovementSpecSourceDestination{
							Path:             getRabbitRelativePath(sourceStorage, sourceAccess, source, i),
							StorageReference: *sourceStorage,
						},
						Destination: &nnfv1alpha1.NnfDataMovementSpecSourceDestination{
							Path:             getRabbitRelativePath(destStorage, destAccess, dest, i),
							StorageReference: *destStorage,
						},
						UserId:  workflow.Spec.UserID,
						GroupId: workflow.Spec.GroupID,
					},
				}

				dwsv1alpha1.AddWorkflowLabels(dm, workflow)
				dwsv1alpha1.AddOwnerLabels(dm, workflow)
				addDirectiveIndexLabel(dm, index)
				addTeardownStateLabel(dm, workflow.Status.State)

				log.Info("Creating NNF Data Movement", "name", client.ObjectKeyFromObject(dm).String())
				if err := r.Create(ctx, dm); err != nil {
					if !errors.IsAlreadyExists(err) {
						return nil, nnfv1alpha1.NewWorkflowError("Data Movement failed to create").WithError(err)
					}
				}
			}

		}

	case "lustre":

		dm := &nnfv1alpha1.NnfDataMovement{
			ObjectMeta: metav1.ObjectMeta{
				Name:      indexedResourceName(workflow, index),
				Namespace: nnfv1alpha1.DataMovementNamespace,
			},
			Spec: nnfv1alpha1.NnfDataMovementSpec{
				Source: &nnfv1alpha1.NnfDataMovementSpecSourceDestination{
					Path:             getRabbitRelativePath(sourceStorage, sourceAccess, source, 0),
					StorageReference: *sourceStorage,
				},
				Destination: &nnfv1alpha1.NnfDataMovementSpecSourceDestination{
					Path:             getRabbitRelativePath(destStorage, destAccess, dest, 0),
					StorageReference: *destStorage,
				},
				UserId:  workflow.Spec.UserID,
				GroupId: workflow.Spec.GroupID,
			},
		}

		dwsv1alpha1.AddWorkflowLabels(dm, workflow)
		dwsv1alpha1.AddOwnerLabels(dm, workflow)
		addDirectiveIndexLabel(dm, index)
		addTeardownStateLabel(dm, workflow.Status.State)

		log.Info("Creating NNF Data Movement", "name", client.ObjectKeyFromObject(dm).String())
		if err := r.Create(ctx, dm); err != nil {
			if !errors.IsAlreadyExists(err) {
				return nil, nnfv1alpha1.NewWorkflowError("Data Movement failed to create").WithError(err)
			}
		}
	}

	return nil, nil
}

func (r *NnfWorkflowReconciler) findLustreFileSystemForPath(ctx context.Context, path string, log logr.Logger) *lusv1alpha1.LustreFileSystem {
	lustres := &lusv1alpha1.LustreFileSystemList{}
	if err := r.List(ctx, lustres); err != nil {
		log.Error(err, "Failed to list lustre file systems")
		return nil
	}

	for _, lustre := range lustres.Items {
		if strings.HasPrefix(path, lustre.Spec.MountRoot) {
			return &lustre
		}
	}

	return nil
}

func (r *NnfWorkflowReconciler) setupNnfAccessForServers(ctx context.Context, storage *nnfv1alpha1.NnfStorage, workflow *dwsv1alpha1.Workflow, index int, parentDwIndex int, teardownState string, log logr.Logger) (*nnfv1alpha1.NnfAccess, error) {

	params, err := dwdparse.BuildArgsMap(workflow.Spec.DWDirectives[parentDwIndex])
	if err != nil {
		return nil, fmt.Errorf("Unable to parse directive %s: %w", workflow.Spec.DWDirectives[parentDwIndex], err)
	}

	access := &nnfv1alpha1.NnfAccess{
		ObjectMeta: metav1.ObjectMeta{
			Name:      indexedResourceName(workflow, parentDwIndex) + "-servers",
			Namespace: workflow.Namespace,
		},
	}

	result, err := ctrl.CreateOrUpdate(ctx, r.Client, access,
		func() error {
			dwsv1alpha1.AddWorkflowLabels(access, workflow)
			dwsv1alpha1.AddOwnerLabels(access, workflow)
			addDirectiveIndexLabel(access, index)
			addTeardownStateLabel(access, teardownState)

			access.Spec = nnfv1alpha1.NnfAccessSpec{
				DesiredState:    "mounted",
				TeardownState:   teardownState,
				Target:          "all",
				MountPath:       buildMountPath(workflow, params["name"], params["command"]),
				MountPathPrefix: buildMountPath(workflow, params["name"], params["command"]),

				// NNF Storage is Namespaced Name to the servers object
				StorageReference: corev1.ObjectReference{
					Kind:      reflect.TypeOf(nnfv1alpha1.NnfStorage{}).Name(),
					Name:      storage.Name,
					Namespace: storage.Namespace,
				},
			}

			return ctrl.SetControllerReference(workflow, access, r.Scheme)
		})

	if err != nil {
		return nil, fmt.Errorf("CreateOrUpdate failed for NnfAccess %v: %w", client.ObjectKeyFromObject(access), err)
	}

	if result == controllerutil.OperationResultCreated {
		log.Info("Created NnfAccess", "name", access.Name, "teardown", teardownState)
	} else if result == controllerutil.OperationResultUpdated {
		log.Info("Updated NnfAccess", "name", access.Name, "teardown", teardownState)
	}

	return access, nil
}

// Monitor a data movement resource for completion
func (r *NnfWorkflowReconciler) finishDataInOutState(ctx context.Context, workflow *dwsv1alpha1.Workflow, index int) (*ctrl.Result, error) {
	log := r.Log.WithValues("Workflow", client.ObjectKeyFromObject(workflow), "Index", index)

	// Wait for data movement resources to complete

	matchingLabels := dwsv1alpha1.MatchingOwner(workflow)
	matchingLabels[nnfv1alpha1.DirectiveIndexLabel] = strconv.Itoa(index)
	matchingLabels[teardownStateLabel] = workflow.Status.State

	dataMovementList := &nnfv1alpha1.NnfDataMovementList{}
	if err := r.List(ctx, dataMovementList, matchingLabels); err != nil {
		return nil, nnfv1alpha1.NewWorkflowError("Could not retrieve data movements").WithError(err)
	}

	// Since the Finish state is only called when copy_in / copy_out directives are present - the lack of any items
	// implies that the data movement operations are only just creating and the cache hasn't been updated yet.
	if len(dataMovementList.Items) == 0 {
		return &ctrl.Result{RequeueAfter: time.Second}, nil
	}

	for _, dm := range dataMovementList.Items {
		log.Info("Processing data movement", "name", client.ObjectKeyFromObject(&dm).String(), "state", dm.Status.State)
		if dm.Status.State != nnfv1alpha1.DataMovementConditionTypeFinished {
			return &ctrl.Result{}, nil
		}

		// TODO: If one fails they all should fail.
	}

	// Check results of data movement operations
	// TODO: Detailed Fail Message?
	for _, dm := range dataMovementList.Items {
		log.Info("Processing data movement", "name", client.ObjectKeyFromObject(&dm).String(), "status", dm.Status.Status)
		if dm.Status.Status != nnfv1alpha1.DataMovementConditionReasonSuccess {
			return nil, nnfv1alpha1.NewWorkflowError(fmt.Sprintf("Staging operation failed")).WithFatal()
		}
	}

	// Delete the NnfAccess resources if necessary
	childObjects := []dwsv1alpha1.ObjectList{
		&nnfv1alpha1.NnfAccessList{},
	}

	deleteStatus, err := dwsv1alpha1.DeleteChildrenWithLabels(ctx, r.Client, childObjects, workflow, matchingLabels)
	if err != nil {
		err = fmt.Errorf("Could not delete NnfAccess children: %w", err)
		return nil, nnfv1alpha1.NewWorkflowError("Could not stop data movement").WithError(err)
	}

	if deleteStatus == dwsv1alpha1.DeleteRetry {
		return &ctrl.Result{}, nil
	}

	return nil, nil
}

func (r *NnfWorkflowReconciler) getDirectiveFileSystemType(ctx context.Context, workflow *dwsv1alpha1.Workflow, index int) (string, error) {
	dwArgs, _ := dwdparse.BuildArgsMap(workflow.Spec.DWDirectives[index])
	switch dwArgs["command"] {
	case "jobdw":
		return dwArgs["type"], nil
	case "persistentdw":
		name, namespace := getStorageReferenceNameFromWorkflowActual(workflow, index)
		nnfStorage := &nnfv1alpha1.NnfStorage{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
		}

		if err := r.Get(ctx, client.ObjectKeyFromObject(nnfStorage), nnfStorage); err != nil {
			return "", fmt.Errorf("Could not get persistent NnfStorage %v to determine file system type: %w", client.ObjectKeyFromObject(nnfStorage), err)
		}

		return nnfStorage.Spec.FileSystemType, nil
	default:
		return "", fmt.Errorf("Invalid directive '%s' to get file system type", workflow.Spec.DWDirectives[index])
	}
}

func (r *NnfWorkflowReconciler) startPreRunState(ctx context.Context, workflow *dwsv1alpha1.Workflow, index int) (*ctrl.Result, error) {
	log := r.Log.WithValues("Workflow", client.ObjectKeyFromObject(workflow), "Index", index)
	dwArgs, _ := dwdparse.BuildArgsMap(workflow.Spec.DWDirectives[index])

	access := &nnfv1alpha1.NnfAccess{
		ObjectMeta: metav1.ObjectMeta{
			Name:      indexedResourceName(workflow, index) + "-computes",
			Namespace: workflow.Namespace,
		},
	}

	// Create an NNFAccess for the compute clients
	result, err := ctrl.CreateOrUpdate(ctx, r.Client, access,
		func() error {
			dwsv1alpha1.AddWorkflowLabels(access, workflow)
			dwsv1alpha1.AddOwnerLabels(access, workflow)
			addDirectiveIndexLabel(access, index)

			access.Spec.TeardownState = dwsv1alpha1.StatePostRun.String()
			access.Spec.DesiredState = "mounted"
			access.Spec.Target = "single"
			access.Spec.MountPath = buildMountPath(workflow, dwArgs["name"], dwArgs["command"])
			access.Spec.ClientReference = corev1.ObjectReference{
				Name:      workflow.Name,
				Namespace: workflow.Namespace,
				Kind:      "Computes",
			}

			// Determine the name/namespace to use based on the directive
			name, namespace := getStorageReferenceNameFromWorkflowActual(workflow, index)

			access.Spec.StorageReference = corev1.ObjectReference{
				// Directive Breakdowns share the same NamespacedName with the Servers it creates, which shares the same name with the NNFStorage.
				Name:      name,
				Namespace: namespace,
				Kind:      reflect.TypeOf(nnfv1alpha1.NnfStorage{}).Name(),
			}

			return ctrl.SetControllerReference(workflow, access, r.Scheme)
		})
	if err != nil {
		err = fmt.Errorf("Could not CreateOrUpdate compute node NnfAccess %v: %w", client.ObjectKeyFromObject(access), err)
		return nil, nnfv1alpha1.NewWorkflowError("Could not mount file system on compute nodes").WithError(err)
	}

	if result == controllerutil.OperationResultCreated {
		log.Info("Created NnfAccess", "name", access.Name)
	} else if result == controllerutil.OperationResultNone {
		// no change
	} else {
		log.Info("Updated NnfAccess", "name", access.Name)
	}

	// Create an NnfAccess for the servers resources if necessary. Shared storage like
	// that of GFS2 provides access to the Rabbit for the lifetime of the user's job.
	// NnfAccess may already be present if a data_in directive was specified for the
	// particular $JOB_DW_[name]; in this case we only need to recreate the resource

	fsType, err := r.getDirectiveFileSystemType(ctx, workflow, index)
	if err != nil {
		return nil, nnfv1alpha1.NewWorkflowError("Unable to determine directive file system type").WithError(err)
	}

	if fsType == "gfs2" || fsType == "lustre" {
		name, namespace := getStorageReferenceNameFromWorkflowActual(workflow, index)

		storage := &nnfv1alpha1.NnfStorage{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
		}

		// Set the teardown state to post run. If there is a copy_out directive that uses
		// this storage instance, set the teardown state so NNF Access is preserved through
		// data_out.
		teardownState := dwsv1alpha1.StatePostRun.String()
		if findCopyOutDirectiveIndexByName(workflow, dwArgs["name"]) >= 0 {
			teardownState = dwsv1alpha1.StateDataOut.String()
		}

		_, err := r.setupNnfAccessForServers(ctx, storage, workflow, index, index, teardownState, log)
		if err != nil {
			return nil, nnfv1alpha1.NewWorkflowError(fmt.Sprintf("Could not setup NNF Access in state %s", workflow.Status.State)).WithError(err)
		}
	}

	return nil, nil
}

func (r *NnfWorkflowReconciler) finishPreRunState(ctx context.Context, workflow *dwsv1alpha1.Workflow, index int) (*ctrl.Result, error) {
	dwArgs, _ := dwdparse.BuildArgsMap(workflow.Spec.DWDirectives[index])

	accessSuffixes := []string{"-computes"}
	fsType, err := r.getDirectiveFileSystemType(ctx, workflow, index)
	if err != nil {
		return nil, nnfv1alpha1.NewWorkflowError("Unable to determine directive file system type").WithError(err)
	}

	if fsType == "gfs2" || fsType == "lustre" {
		accessSuffixes = append(accessSuffixes, "-servers")
	}

	for _, suffix := range accessSuffixes {
		access := &nnfv1alpha1.NnfAccess{
			ObjectMeta: metav1.ObjectMeta{
				Name:      indexedResourceName(workflow, index) + suffix,
				Namespace: workflow.Namespace,
			},
		}

		if err := r.Get(ctx, client.ObjectKeyFromObject(access), access); err != nil {
			err = fmt.Errorf("Could not get NnfAccess %v: %w", client.ObjectKeyFromObject(access), err)
			return nil, nnfv1alpha1.NewWorkflowError("Could not mount file system on compute nodes").WithError(err)
		}

		if access.Status.Error != nil {
			return nil, nnfv1alpha1.NewWorkflowError("Could not mount file system on compute nodes").WithError(access.Status.Error)
		}

		if access.Status.State != access.Spec.DesiredState || access.Status.Ready == false {
			return &ctrl.Result{}, nil
		}
	}

	// Add an environment variable to the workflow status section for the location of the
	// mount point on the clients.
	if workflow.Status.Env == nil {
		workflow.Status.Env = make(map[string]string)
	}

	envName := ""
	switch dwArgs["command"] {
	case "jobdw":
		envName = "DW_JOB_" + dwArgs["name"]
	case "persistentdw":
		envName = "DW_PERSISTENT_" + dwArgs["name"]
	default:
		return nil, nnfv1alpha1.NewWorkflowError(fmt.Sprintf("Unexpected directive %v", dwArgs["command"]))
	}

	workflow.Status.Env[envName] = buildMountPath(workflow, dwArgs["name"], dwArgs["command"])

	return nil, nil
}

// TODO: Can this function be changed to accept a workflow and directive index? That way it can be used
// in the data movement code setupNnfAccessForServers()
func buildMountPath(workflow *dwsv1alpha1.Workflow, name string, command string) string {

	if len(name) == 0 {
		panic(fmt.Sprintf("Mount Path: Empty name"))
	}

	switch command {
	case "jobdw":
		return fmt.Sprintf("/mnt/nnf/%d/job/%s", workflow.Spec.JobID, name)
	case "persistentdw":
		return fmt.Sprintf("/mnt/nnf/%d/persistent/%s", workflow.Spec.JobID, name)
	}

	panic(fmt.Sprintf("Mount Path: Invalid command '%s'", command))
}

func (r *NnfWorkflowReconciler) startPostRunState(ctx context.Context, workflow *dwsv1alpha1.Workflow, index int) (*ctrl.Result, error) {

	// Wait for data movement resources to complete
	matchingLabels := dwsv1alpha1.MatchingOwner(workflow)
	matchingLabels[teardownStateLabel] = workflow.Status.State

	dataMovementList := &nnfv1alpha1.NnfDataMovementList{}
	if err := r.List(ctx, dataMovementList, matchingLabels); err != nil {
		return nil, nnfv1alpha1.NewWorkflowError("Could not retrieve data movements").WithError(err)
	}

	for _, dm := range dataMovementList.Items {
		if dm.Status.State != nnfv1alpha1.DataMovementConditionTypeFinished {
			return &ctrl.Result{}, nil
		}

		if dm.Status.Status != nnfv1alpha1.DataMovementConditionReasonSuccess {
			return nil, nnfv1alpha1.NewWorkflowError("Data movement unsuccessful").WithFatal()
		}
	}

	// TODO: Pass the data movement(s) failure message into the workflow

	// Unmount the NnfAccess from the compute nodes. This will free the compute nodes to be used
	// in a different job even if there is data movement happening on the Rabbits
	access := &nnfv1alpha1.NnfAccess{
		ObjectMeta: metav1.ObjectMeta{
			Name:      indexedResourceName(workflow, index) + "-computes",
			Namespace: workflow.Namespace,
		},
	}

	if err := r.Get(ctx, client.ObjectKeyFromObject(access), access); err != nil {
		err = fmt.Errorf("Could not get NnfAccess %v: %w", client.ObjectKeyFromObject(access), err)
		return nil, nnfv1alpha1.NewWorkflowError("Unable to find compute node mount information").WithError(err)
	}

	if access.Spec.DesiredState != "unmounted" {
		access.Spec.DesiredState = "unmounted"

		if err := r.Update(ctx, access); err != nil {
			if !apierrors.IsConflict(err) {
				err = fmt.Errorf("Could not update NnfAccess %v: %w", client.ObjectKeyFromObject(access), err)
				return nil, nnfv1alpha1.NewWorkflowError("Unable to request compute node unmount").WithError(err)
			}

			return &ctrl.Result{}, nil
		}
	}

	return nil, nil
}

func (r *NnfWorkflowReconciler) finishPostRunState(ctx context.Context, workflow *dwsv1alpha1.Workflow, index int) (*ctrl.Result, error) {

	access := &nnfv1alpha1.NnfAccess{
		ObjectMeta: metav1.ObjectMeta{
			Name:      indexedResourceName(workflow, index) + "-computes",
			Namespace: workflow.Namespace,
		},
	}

	if err := r.Get(ctx, client.ObjectKeyFromObject(access), access); err == nil {
		if access.Status.Error != nil {
			return nil, nnfv1alpha1.NewWorkflowError("Could not unmount file system from compute nodes").WithError(access.Status.Error)
		}

		if access.Status.State != "unmounted" || access.Status.Ready != true {
			return &ctrl.Result{}, nil
		}
	}

	if err := r.Delete(ctx, access); err != nil {
		if !apierrors.IsNotFound(err) {
			err = fmt.Errorf("Could not delete compute NnfAccess: %w", err)
			return nil, nnfv1alpha1.NewWorkflowError("Could not unmount file system from compute nodes").WithError(err)
		}

		return nil, nil
	}

	// TODO: We should delete the NNF Access for the servers if it's no longer needed (if there are no copy_out directives)
	//       We should also delete all the user initiated offload operations.

	return nil, nil
}

func (r *NnfWorkflowReconciler) startTeardownState(ctx context.Context, workflow *dwsv1alpha1.Workflow, index int) (*ctrl.Result, error) {

	// Delete the NnfDataMovement and NnfAccess for this directive before removing the NnfStorage.
	// copy_in/out directives can reference NnfStorage from a different directive, so all the NnfAccesses
	// need to be removed first.
	childObjects := []dwsv1alpha1.ObjectList{
		&nnfv1alpha1.NnfDataMovementList{},
		&nnfv1alpha1.NnfAccessList{},
	}

	deleteStatus, err := dwsv1alpha1.DeleteChildrenWithLabels(ctx, r.Client, childObjects, workflow, client.MatchingLabels{nnfv1alpha1.DirectiveIndexLabel: strconv.Itoa(index)})
	if err != nil {
		err = fmt.Errorf("Could not delete NnfDataMovement and NnfAccess children: %w", err)
		return nil, nnfv1alpha1.NewWorkflowError("Could not stop data movement and unmount file systems").WithError(err)
	}

	if deleteStatus == dwsv1alpha1.DeleteRetry {
		return &ctrl.Result{}, nil
	}

	return nil, nil
}

func (r *NnfWorkflowReconciler) finishTeardownState(ctx context.Context, workflow *dwsv1alpha1.Workflow, index int) (*ctrl.Result, error) {
	log := r.Log.WithValues("Workflow", client.ObjectKeyFromObject(workflow), "Index", index)
	dwArgs, _ := dwdparse.BuildArgsMap(workflow.Spec.DWDirectives[index])

	switch dwArgs["command"] {
	case "create_persistent":
		for _, driverStatus := range workflow.Status.Drivers {
			if driverStatus.WatchState == dwsv1alpha1.StateTeardown.String() {
				continue
			}

			if !driverStatus.Completed {
				return nil, nil
			}
		}

		persistentStorage, err := r.findPersistentInstance(ctx, workflow, dwArgs["name"])
		if err != nil {
			return nil, nnfv1alpha1.NewWorkflowError(fmt.Sprintf("Could not find persistent storage %v", dwArgs["name"])).WithError(err)
		}

		persistentStorage.SetOwnerReferences([]metav1.OwnerReference{})
		dwsv1alpha1.RemoveOwnerLabels(persistentStorage)
		labels := persistentStorage.GetLabels()
		delete(labels, nnfv1alpha1.DirectiveIndexLabel)
		persistentStorage.SetLabels(labels)

		err = r.Update(ctx, persistentStorage)
		if err != nil {
			err = fmt.Errorf("Could not update PersistentStorage %v: %w", client.ObjectKeyFromObject(persistentStorage), err)
			return nil, nnfv1alpha1.NewWorkflowError("Could not finalize peristent storage").WithError(err)
		}
		log.Info("Removed owner reference from persistent storage", "psi", persistentStorage)
	case "destroy_persistent":
		persistentStorage, err := r.findPersistentInstance(ctx, workflow, dwArgs["name"])
		if err != nil {
			if !apierrors.IsNotFound(err) {
				return nil, nnfv1alpha1.NewWorkflowError(fmt.Sprintf("Could not find peristent storage %v", dwArgs["name"])).WithError(err)
			}

			return nil, nil
		}

		dwsv1alpha1.AddOwnerLabels(persistentStorage, workflow)
		addDirectiveIndexLabel(persistentStorage, index)

		if err := controllerutil.SetControllerReference(workflow, persistentStorage, r.Scheme); err != nil {
			log.Info("Unable to assign workflow as owner of persistentInstance", "psi", persistentStorage)
			err = fmt.Errorf("Could not assign workflow as owner of PersistentInstance %v: %w", client.ObjectKeyFromObject(persistentStorage), err)
			return nil, nnfv1alpha1.NewWorkflowError(fmt.Sprintf("Could not delete peristent storage %v", dwArgs["name"])).WithError(err)
		}

		err = r.Update(ctx, persistentStorage)
		if err != nil {
			err = fmt.Errorf("Could not update PersistentInstance %v: %w", client.ObjectKeyFromObject(persistentStorage), err)
			return nil, nnfv1alpha1.NewWorkflowError(fmt.Sprintf("Could not delete peristent storage %v", dwArgs["name"])).WithError(err)
		}
		log.Info("Add owner reference for persistent storage for deletion", "psi", persistentStorage)
	default:
	}

	childObjects := []dwsv1alpha1.ObjectList{
		&nnfv1alpha1.NnfStorageList{},
		&dwsv1alpha1.PersistentStorageInstanceList{},
	}

	deleteStatus, err := dwsv1alpha1.DeleteChildrenWithLabels(ctx, r.Client, childObjects, workflow, client.MatchingLabels{nnfv1alpha1.DirectiveIndexLabel: strconv.Itoa(index)})
	if err != nil {
		return nil, nnfv1alpha1.NewWorkflowError("Could not delete storage allocations").WithError(err)
	}

	if deleteStatus == dwsv1alpha1.DeleteRetry {
		return &ctrl.Result{}, nil
	}

	return nil, nil
}

func (r *NnfWorkflowReconciler) findPersistentInstance(ctx context.Context, wf *dwsv1alpha1.Workflow, psiName string) (*dwsv1alpha1.PersistentStorageInstance, error) {
	log := r.Log.WithValues("Workflow", types.NamespacedName{Name: wf.Name, Namespace: wf.Namespace})

	psi := &dwsv1alpha1.PersistentStorageInstance{}
	psiNamedNamespace := types.NamespacedName{Name: psiName, Namespace: wf.Namespace}
	err := r.Get(ctx, psiNamedNamespace, psi)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			log.Error(err, "Unable to get PersistentStorageInstance", "name", psiName, "error", err)
		}

		return nil, err
	}

	return psi, err
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

func (w *workflowStatusUpdater) close(ctx context.Context, r *NnfWorkflowReconciler) error {
	if !reflect.DeepEqual(w.workflow.Status, w.existingStatus) {
		err := r.Update(ctx, w.workflow)
		if !apierrors.IsConflict(err) {
			return err
		}
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NnfWorkflowReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.ChildObjects = []dwsv1alpha1.ObjectList{
		&nnfv1alpha1.NnfDataMovementList{},
		&nnfv1alpha1.NnfAccessList{},
		&nnfv1alpha1.NnfStorageList{},
		&dwsv1alpha1.PersistentStorageInstanceList{},
		&dwsv1alpha1.DirectiveBreakdownList{},
	}

	maxReconciles := runtime.GOMAXPROCS(0)
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(controller.Options{MaxConcurrentReconciles: maxReconciles}).
		For(&dwsv1alpha1.Workflow{}).
		Owns(&nnfv1alpha1.NnfAccess{}).
		Owns(&dwsv1alpha1.DirectiveBreakdown{}).
		Owns(&dwsv1alpha1.PersistentStorageInstance{}).
		Watches(&source.Kind{Type: &nnfv1alpha1.NnfDataMovement{}}, handler.EnqueueRequestsFromMapFunc(dwsv1alpha1.OwnerLabelMapFunc)).
		Watches(&source.Kind{Type: &nnfv1alpha1.NnfStorage{}}, handler.EnqueueRequestsFromMapFunc(dwsv1alpha1.OwnerLabelMapFunc)).
		Complete(r)
}
