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
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
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
)

const (
	// finalizerNnfWorkflow defines the key used in identifying the
	// storage object as being owned by this NNF Storage Reconciler. This
	// prevents the system from deleting the custom resource until the
	// reconciler has finished in using the resource.
	finalizerNnfWorkflow = "nnf.cray.hpe.com/nnf_workflow"

	// directiveIndexLabel is a label applied to child objects of the workflow
	// to show which directive they were created for. This is useful during deletion
	// to filter the child objects by the directive index and only delete the
	// resources for the directive being processed
	directiveIndexLabel = "nnf.cray.hpe.com/directive_index"

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
//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfdatamovementworkflows,verbs=get;create;list;watch;update;patch;delete;deletecollection
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
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	// Add the finalizer if it doesn't exist
	if !controllerutil.ContainsFinalizer(workflow, finalizerNnfWorkflow) {
		controllerutil.AddFinalizer(workflow, finalizerNnfWorkflow)
		if err := r.Update(ctx, workflow); err != nil {
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
			e, ok := err.(*nnfv1alpha1.WorkflowError)
			if ok {
				driverStatus.Message = e.GetMessage()
				if e.GetRecoverable() {
					driverStatus.Reason = "running"
				} else {
					driverStatus.Reason = "error"
				}
			} else {
				driverStatus.Reason = "error"
				driverStatus.Message = "Internal error: " + err.Error()
			}

			log.Info("Start error", "State", workflow.Status.State, "index", driverStatus.DWDIndex, "Message", err.Error())
			return ctrl.Result{}, err
		}

		driverStatus.Reason = "running"
		driverStatus.Message = ""

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
			e, ok := err.(*nnfv1alpha1.WorkflowError)
			if ok {
				driverStatus.Message = e.GetMessage()
				if e.GetRecoverable() {
					driverStatus.Reason = "running"
				} else {
					driverStatus.Reason = "error"
				}
			} else {
				driverStatus.Reason = "error"
				driverStatus.Message = "Internal error: " + err.Error()
			}
			log.Info("Finish error", "State", workflow.Status.State, "index", driverStatus.DWDIndex, "Message", err.Error())

			return ctrl.Result{}, err
		}

		driverStatus.Reason = "running"
		driverStatus.Message = ""

		if result != nil {
			log.Info("Finish wait", "State", workflow.Status.State, "index", driverStatus.DWDIndex)
			return *result, nil
		}

		log.Info("Finish done", "State", workflow.Status.State, "index", driverStatus.DWDIndex)

		ts := metav1.NowMicro()
		driverStatus.Reason = "completed"
		driverStatus.Message = ""
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
		return nil, err
	}

	// only jobdw, persistentdw, and create_persistent need a directive breakdown
	if dwArgs["command"] != "jobdw" && dwArgs["command"] != "persistentdw" && dwArgs["command"] != "create_persistent" {
		return nil, nil
	}

	directiveBreakdown, err := r.generateDirectiveBreakdown(ctx, index, workflow, log)
	if err != nil {
		return nil, err
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
		log.Error(err, "Failed to get DirectiveBreakdown", "name", directiveBreakdown.GetName())
		return nil, err
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
				return err
			}

		case "create_persistent":
			createPersistentCount++

		case "destroy_persistent":
			deletePersistentCount++

		case "persistentdw":
			if err := r.validatePersistentInstanceDirective(ctx, wf, directive); err != nil {
				return err
			}
		}
	}

	if directiveCount > 1 {
		// Ensure create_persistent or destroy_persistent are singletons in the workflow
		if createPersistentCount+deletePersistentCount > 0 {
			return fmt.Errorf("a single create_persistent or destroy_persistent directive allowed per workflow")
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
				return fmt.Errorf("Job storage instance '%s' not found", name)
			}
		} else if strings.HasPrefix(arg, "$PERSISTENT_DW_") {
			if err := r.validatePersistentInstanceByName(ctx, name, wf.Namespace); err != nil {
				return fmt.Errorf("Persistent storage instance '%s' not found", name)
			}
			if findDirectiveIndexByName(wf, name) == -1 {
				return fmt.Errorf("persistentdw directive mentioning '%s' not found", name)
			}
		} else {
			if r.findLustreFileSystemForPath(ctx, arg, r.Log) == nil {
				return fmt.Errorf("Global Lustre file system containing '%s' not found", arg)
			}
		}

		return nil
	}

	args, err := dwdparse.BuildArgsMap(directive)
	if err != nil {
		return err
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
		return err
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
		return fmt.Errorf("persistent storage instance '%s' is deleting", name)
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
func (r *NnfWorkflowReconciler) generateDirectiveBreakdown(ctx context.Context, dwIndex int, workflow *dwsv1alpha1.Workflow, log logr.Logger) (directiveBreakdown *dwsv1alpha1.DirectiveBreakdown, err error) {

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

	// Initialize return parameters
	directiveBreakdown = nil
	err = nil

	directive := workflow.Spec.DWDirectives[dwIndex]
	result := controllerutil.OperationResultNone

	dwArgs, _ := dwdparse.BuildArgsMap(directive)
	for _, breakThisDown := range breakDownCommands {
		// We care about the commands that generate a breakdown
		if breakThisDown == dwArgs["command"] {
			dwdName := indexedResourceName(workflow, dwIndex)
			directiveBreakdown = &dwsv1alpha1.DirectiveBreakdown{
				ObjectMeta: metav1.ObjectMeta{
					Name:      dwdName,
					Namespace: workflow.Namespace,
				},
			}

			result, err = ctrl.CreateOrUpdate(ctx, r.Client, directiveBreakdown,
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
				return
			}

			if result == controllerutil.OperationResultCreated {
				log.Info("Created DirectiveBreakdown", "name", directiveBreakdown.Name)
			} else if result == controllerutil.OperationResultNone {
				// no change
			} else {
				log.Info("Updated DirectiveBreakdown", "name", directiveBreakdown.Name)
			}

			// The directive's command has been matched, no need to look at the other breakDownCommands
			return
		}
	}

	return
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
		return &ctrl.Result{}, err
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
		return &ctrl.Result{}, err
	}

	if _, present := os.LookupEnv("RABBIT_TEST_ENV_BYPASS_SERVER_STORAGE_CHECK"); !present {
		if len(dbd.Status.Storage.AllocationSets) != 0 && len(dbd.Status.Storage.AllocationSets) != len(s.Spec.AllocationSets) {
			return nil, fmt.Errorf("Servers resource does not meet storage requirements for directive '%s'", dbd.Spec.Directive)
		}
	}

	_, err = r.createNnfStorage(ctx, workflow, s, index, log)
	if err != nil {
		if apierrors.IsConflict(err) {
			return &ctrl.Result{Requeue: true}, nil
		}

		log.Info("Failed to create nnf storage", "Message", err)
		return nil, err
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
		return nil, err
	}

	// If the Status section has not been filled in yet, exit and wait.
	if len(nnfStorage.Status.AllocationSets) != len(nnfStorage.Spec.AllocationSets) {
		// RequeueAfter is necessary for persistent storage that isn't owned by this workflow
		return &ctrl.Result{RequeueAfter: time.Second * time.Duration(2)}, nil
	}

	// Status section should be usable now, check for Ready
	for _, set := range nnfStorage.Status.AllocationSets {
		if set.Status != "Ready" {
			// RequeueAfter is necessary for persistent storage that isn't owned by this workflow
			return &ctrl.Result{RequeueAfter: time.Second * time.Duration(2)}, nil
		}
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
		return nil, err
	}

	pinnedName, pinnedNamespace := getStorageReferenceNameFromWorkflowActual(workflow, index)
	nnfStorageProfile, err := findPinnedProfile(ctx, r.Client, pinnedNamespace, pinnedName)
	if err != nil {
		log.Error(err, "Unable to find pinned NnfStorageProfile", "name", pinnedName)
		return nil, err
	}

	var owner metav1.Object = workflow
	if dwArgs["command"] == "create_persistent" {
		psi, err := r.findPersistentInstance(ctx, workflow, dwArgs["name"])
		if err != nil {
			return nil, err
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
						nnfAllocSet.NnfStorageLustreSpec.ExternalMgsNid = lustreData.ExternalMGS[0]
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
		return nnfStorage, err
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
	dm := &nnfv1alpha1.NnfDataMovement{
		ObjectMeta: metav1.ObjectMeta{
			Name:      indexedResourceName(workflow, index),
			Namespace: workflow.Namespace,
		},
	}

	if err := r.Get(ctx, client.ObjectKeyFromObject(dm), dm); err != nil {
		if !apierrors.IsNotFound(err) {
			return nil, err
		}
	} else {
		// If the NnfDataMovement resource is already created, then we're done.
		return nil, nil
	}

	parameters, err := dwdparse.BuildArgsMap(workflow.Spec.DWDirectives[index])
	if err != nil {
		return nil, err
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
	prepareStagingArgumentFn := func(param string) (*corev1.ObjectReference, *nnfv1alpha1.NnfAccess, string, *ctrl.Result, error) {
		var storageReference *corev1.ObjectReference
		var access *nnfv1alpha1.NnfAccess

		name, path := splitStagingArgumentIntoNameAndPath(param)

		// If param refers to a Job or Persistent storage type, find the NNF Storage that is backing
		// this directive line.
		if strings.HasPrefix(param, "$JOB_DW_") || strings.HasPrefix(param, "$PERSISTENT_DW_") {
			var result controllerutil.OperationResult
			var fsType string

			directiveIdx := findDirectiveIndexByName(workflow, name)
			if directiveIdx < 0 {
				return nil, nil, "", nil, fmt.Errorf("No directive matching '%s' found in workflow", name)
			}

			// If directive specifies a persistent storage instance, `name` will be the nnfStorageName
			// Otherwise it matches the workflow with the directive index as a suffix
			var nnfStorageName string
			if strings.HasPrefix(param, "$PERSISTENT_DW_") {
				nnfStorageName = name
			} else {
				nnfStorageName = indexedResourceName(workflow, directiveIdx)
			}

			storage := &nnfv1alpha1.NnfStorage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      nnfStorageName,
					Namespace: workflow.Namespace,
				},
			}

			if err := r.Get(ctx, client.ObjectKeyFromObject(storage), storage); err != nil {
				return nil, nil, "", nil, err
			}

			storageReference = &corev1.ObjectReference{
				Kind:      reflect.TypeOf(nnfv1alpha1.NnfStorage{}).Name(),
				Name:      storage.Name,
				Namespace: storage.Namespace,
			}

			fsType = storage.Spec.FileSystemType

			if fsType == "lustre" {
				return storageReference, nil, path, nil, nil
			}

			// Find the desired workflow teardown state for the NNF Access. This instructs the workflow
			// when to teardown an NNF Access for the servers
			teardownState := ""
			if parameters["command"] == "copy_in" {
				teardownState = dwsv1alpha1.StateDataIn.String()

				if fsType == "gfs2" {
					teardownState = dwsv1alpha1.StatePostRun.String()

					if findCopyOutDirectiveIndexByName(workflow, name) >= 0 {
						teardownState = dwsv1alpha1.StateDataOut.String()
					}
				}
			} else if parameters["command"] == "copy_out" {
				teardownState = dwsv1alpha1.StateDataOut.String()
			}

			// Setup NNF Access for the NNF Servers so we can run data movement on them.
			access, result, err = r.setupNnfAccessForServers(ctx, storage, workflow, directiveIdx, teardownState)
			if err != nil {
				return storageReference, access, path, nil, err
			} else if result != controllerutil.OperationResultNone {
				return storageReference, access, path, &ctrl.Result{Requeue: true}, nil
			}

		} else if lustre := r.findLustreFileSystemForPath(ctx, param, r.Log); lustre != nil {
			storageReference = &corev1.ObjectReference{
				Kind:      reflect.TypeOf(lusv1alpha1.LustreFileSystem{}).Name(),
				Name:      lustre.Name,
				Namespace: lustre.Namespace,
			}
		} else {
			return nil, nil, "", nil, fmt.Errorf("Parameter '%s' invalid", param)
		}

		return storageReference, access, path, nil, nil
	}

	sourceStorage, sourceAccess, sourcePath, result, err := prepareStagingArgumentFn(parameters["source"])
	if err != nil {
		return nil, err
	} else if result != nil {
		return result, nil
	}

	destinationStorage, destinationAccess, destinationPath, result, err := prepareStagingArgumentFn(parameters["destination"])
	if err != nil {
		return nil, err
	} else if result != nil {
		return result, nil
	}

	// Wait for accesses to go ready
	for _, access := range []*nnfv1alpha1.NnfAccess{sourceAccess, destinationAccess} {
		if access != nil {
			if err := r.Get(ctx, client.ObjectKeyFromObject(access), access); err != nil {
				return nil, err
			}

			if access.Status.Ready == false {
				return &ctrl.Result{}, nil
			}
		}
	}

	accessToObjectReference := func(access *nnfv1alpha1.NnfAccess) *corev1.ObjectReference {
		if access != nil {
			return &corev1.ObjectReference{
				Kind:      reflect.TypeOf(nnfv1alpha1.NnfAccess{}).Name(),
				Name:      access.Name,
				Namespace: access.Namespace,
			}
		}

		return nil
	}

	dm = &nnfv1alpha1.NnfDataMovement{
		ObjectMeta: metav1.ObjectMeta{
			Name:      indexedResourceName(workflow, index),
			Namespace: workflow.Namespace,
		},
	}

	dwsv1alpha1.AddWorkflowLabels(dm, workflow)
	dwsv1alpha1.AddOwnerLabels(dm, workflow)
	addDirectiveIndexLabel(dm, index)

	dm.Spec = nnfv1alpha1.NnfDataMovementSpec{
		Source: &nnfv1alpha1.NnfDataMovementSpecSourceDestination{
			Path:    sourcePath,
			Storage: sourceStorage,
			Access:  accessToObjectReference(sourceAccess),
		},
		Destination: &nnfv1alpha1.NnfDataMovementSpecSourceDestination{
			Path:    destinationPath,
			Storage: destinationStorage,
			Access:  accessToObjectReference(destinationAccess),
		},
		UserId:  workflow.Spec.UserID,
		GroupId: workflow.Spec.GroupID,
	}

	if err := ctrl.SetControllerReference(workflow, dm, r.Scheme); err != nil {
		return nil, err
	}

	if err := r.Create(ctx, dm); err != nil {
		return nil, err
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

func (r *NnfWorkflowReconciler) setupNnfAccessForServers(ctx context.Context, storage *nnfv1alpha1.NnfStorage, workflow *dwsv1alpha1.Workflow, directiveIdx int, teardownState string) (*nnfv1alpha1.NnfAccess, controllerutil.OperationResult, error) {

	params, err := dwdparse.BuildArgsMap(workflow.Spec.DWDirectives[directiveIdx])
	if err != nil {
		return nil, controllerutil.OperationResultNone, err
	}

	access := &nnfv1alpha1.NnfAccess{
		ObjectMeta: metav1.ObjectMeta{
			Name:      indexedResourceName(workflow, directiveIdx) + "-servers",
			Namespace: workflow.Namespace,
		},
	}

	result, err := ctrl.CreateOrUpdate(ctx, r.Client, access,
		func() error {
			dwsv1alpha1.AddWorkflowLabels(access, workflow)
			dwsv1alpha1.AddOwnerLabels(access, workflow)
			addDirectiveIndexLabel(access, directiveIdx)
			addTeardownStateLabel(access, teardownState)

			access.Spec = nnfv1alpha1.NnfAccessSpec{
				DesiredState:    "mounted",
				TeardownState:   teardownState,
				Target:          "all",
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

	return access, result, err
}

// Monitor a data movement resource for completion
func (r *NnfWorkflowReconciler) finishDataInOutState(ctx context.Context, workflow *dwsv1alpha1.Workflow, index int) (*ctrl.Result, error) {
	dm := &nnfv1alpha1.NnfDataMovement{
		ObjectMeta: metav1.ObjectMeta{
			Name:      indexedResourceName(workflow, index),
			Namespace: workflow.Namespace,
		},
	}

	if err := r.Get(ctx, client.ObjectKeyFromObject(dm), dm); err != nil {
		return nil, err
	}

	for _, condition := range dm.Status.Conditions {
		if condition.Type == nnfv1alpha1.DataMovementConditionTypeFinished {
			if err := r.teardownDataMovementResource(ctx, dm, workflow.Status.State); err != nil {
				return nil, err
			}

			return nil, nil
		}
	}

	return &ctrl.Result{}, nil
}

// Teardown a data movement resource and its subresources. This prepares the resource for deletion but does not delete it.
func (r *NnfWorkflowReconciler) teardownDataMovementResource(ctx context.Context, dm *nnfv1alpha1.NnfDataMovement, workflowState string) error {
	deleteAccess := func(ref *corev1.ObjectReference) error {
		if ref == nil {
			return nil
		}

		access := &nnfv1alpha1.NnfAccess{}
		if err := r.Get(ctx, types.NamespacedName{Name: ref.Name, Namespace: ref.Namespace}, access); err != nil {
			return client.IgnoreNotFound(err)
		}

		if workflowState != access.Spec.TeardownState {
			return nil
		}

		if err := r.Delete(ctx, access); err != nil {
			return client.IgnoreNotFound(err)
		}

		return nil
	}

	if err := deleteAccess(dm.Spec.Source.Access); err != nil {
		return err
	}

	if err := deleteAccess(dm.Spec.Destination.Access); err != nil {
		return err
	}

	return nil
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
			return "", err
		}

		return nnfStorage.Spec.FileSystemType, nil
	default:
		return "", fmt.Errorf("Invalid directive '%s' to get file system type", workflow.Spec.DWDirectives[index])
	}
}

func (r *NnfWorkflowReconciler) startPreRunState(ctx context.Context, workflow *dwsv1alpha1.Workflow, index int) (*ctrl.Result, error) {
	log := r.Log.WithValues("Workflow", client.ObjectKeyFromObject(workflow), "Index", index)
	dwArgs, _ := dwdparse.BuildArgsMap(workflow.Spec.DWDirectives[index])

	// Create a companion NNF Data Movement resource that manages concurrent data movement started
	// during the job by the compute resources. It is expected that the customer will clean-up data
	// movement resources that it created, but this is implemented to ensure no resources are left
	// behind by the job and we report all errors that might have been missed.

	dm := &nnfv1alpha1.NnfDataMovement{
		ObjectMeta: metav1.ObjectMeta{
			Name:      workflow.Name,
			Namespace: workflow.Namespace,
		},
	}

	result, err := ctrl.CreateOrUpdate(ctx, r.Client, dm, func() error {
		dwsv1alpha1.AddWorkflowLabels(dm, workflow)
		dwsv1alpha1.AddOwnerLabels(dm, workflow)
		addDirectiveIndexLabel(dm, index)

		dm.Spec.Monitor = true

		return ctrl.SetControllerReference(workflow, dm, r.Scheme)
	})
	if err != nil {
		return nil, err
	}

	if result == controllerutil.OperationResultCreated {
		log.Info("Created NnfDatamovement", "name", dm.Name)
	}

	access := &nnfv1alpha1.NnfAccess{
		ObjectMeta: metav1.ObjectMeta{
			Name:      indexedResourceName(workflow, index) + "-computes",
			Namespace: workflow.Namespace,
		},
	}

	// Create an NNFAccess for the compute clients
	result, err = ctrl.CreateOrUpdate(ctx, r.Client, access,
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
		return nil, err
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
		return nil, err
	}

	if fsType == "gfs2" {
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

		access, result, err := r.setupNnfAccessForServers(ctx, storage, workflow, index, teardownState)
		if err != nil {
			return nil, err
		} else if result == controllerutil.OperationResultCreated {
			log.Info("Created NnfAccess", "name", access.Name)
		} else if result == controllerutil.OperationResultUpdated {
			log.Info("Updated NnfAccess", "name", access.Name)
		}
	}

	return nil, nil
}

func (r *NnfWorkflowReconciler) finishPreRunState(ctx context.Context, workflow *dwsv1alpha1.Workflow, index int) (*ctrl.Result, error) {
	dwArgs, _ := dwdparse.BuildArgsMap(workflow.Spec.DWDirectives[index])

	accessSuffixes := []string{"-computes"}
	fsType, err := r.getDirectiveFileSystemType(ctx, workflow, index)
	if err != nil {
		return nil, err
	}

	if fsType == "gfs2" {
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
			if apierrors.IsNotFound(err) {
				continue
			}

			return nil, err
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
		return nil, fmt.Errorf("Unexpected #DW command %s", dwArgs["command"])
	}

	workflow.Status.Env[envName] = buildMountPath(workflow, dwArgs["name"], dwArgs["command"])

	return nil, nil
}

// TODO: Can this function be changed to accept a workflow and directive index? That way it can be used
// in the data movement code setupNnfAccessForServers()
func buildMountPath(workflow *dwsv1alpha1.Workflow, name string, command string) string {
	switch command {
	case "jobdw":
		return fmt.Sprintf("/mnt/nnf/%d/job/%s", workflow.Spec.JobID, name)
	case "persistentdw":
		return fmt.Sprintf("/mnt/nnf/%d/persistent/%s", workflow.Spec.JobID, name)
	default:
	}
	return ""
}

func (r *NnfWorkflowReconciler) startPostRunState(ctx context.Context, workflow *dwsv1alpha1.Workflow, index int) (*ctrl.Result, error) {
	// Unmount the NnfAccess from the compute nodes. This will free the compute nodes to be used
	// in a different job even if there is data movement happening on the Rabbits
	access := &nnfv1alpha1.NnfAccess{
		ObjectMeta: metav1.ObjectMeta{
			Name:      indexedResourceName(workflow, index) + "-computes",
			Namespace: workflow.Namespace,
		},
	}

	if err := r.Get(ctx, client.ObjectKeyFromObject(access), access); err != nil {
		return nil, nil
	}

	if access.Spec.DesiredState != "unmounted" {
		access.Spec.DesiredState = "unmounted"

		if err := r.Update(ctx, access); err != nil {
			if !apierrors.IsConflict(err) {
				return nil, err
			}

			return &ctrl.Result{}, nil
		}
	}

	// Delete the companion NNF Data Movement Workflow resource that manages concurrent data movement
	// executed during the jobs run state by compute resources. It is expected that the customer will
	// clean-up data movement resources that they created, but this is implemented to ensure no resource
	// are left behind by the job and that the rsync status is known to the workflow on failure
	dm := &nnfv1alpha1.NnfDataMovement{
		ObjectMeta: metav1.ObjectMeta{
			Name:      workflow.Name,
			Namespace: workflow.Namespace,
		},
	}

	if err := r.Get(ctx, client.ObjectKeyFromObject(dm), dm); err != nil {
		if !apierrors.IsNotFound(err) {
			return nil, err
		}
	} else {
		// Stop the data movement resource from monitoring subresources. This will permit the
		// data movement resource to reach a finished state where the status fields are valid.
		if dm.Spec.Monitor == true {
			dm.Spec.Monitor = false
			if err := r.Update(ctx, dm); err != nil {
				return nil, err
			}

			return &ctrl.Result{}, nil
		}
	}

	return nil, nil
}

func (r *NnfWorkflowReconciler) finishPostRunState(ctx context.Context, workflow *dwsv1alpha1.Workflow, index int) (*ctrl.Result, error) {
	dm := &nnfv1alpha1.NnfDataMovement{
		ObjectMeta: metav1.ObjectMeta{
			Name:      workflow.Name,
			Namespace: workflow.Namespace,
		},
	}

	if err := r.Get(ctx, client.ObjectKeyFromObject(dm), dm); err != nil {
		if !apierrors.IsNotFound(err) {
			return nil, err
		}
	} else {
		finished := false
		for _, condition := range dm.Status.Conditions {
			if condition.Type == nnfv1alpha1.DataMovementConditionTypeFinished {
				finished = true
			}
		}

		// Data movement is still in progress; return here so the NnfAccess is not destroyed
		// until after all data movement resources are finished.
		if !finished {
			return &ctrl.Result{}, nil
		}

		if len(dm.Status.Conditions) != 0 {
			condition := &dm.Status.Conditions[len(dm.Status.Conditions)-1]
			if condition.Reason == nnfv1alpha1.DataMovementConditionReasonFailed {
				return nil, fmt.Errorf("%s", condition.Message)
			}
		}

		childObjects := []dwsv1alpha1.ObjectList{
			&nnfv1alpha1.NnfAccessList{},
		}

		// Retrieve the NNF Accesses for this workflow, unmount and delete them if necessary. Each
		// access has a TeardownState defined in the specification that tells this logic when to
		// teardown the NNF Access. Filter on NNF Accesses matching the target workflow and in the
		// Post-Run teardown state.
		deleteStatus, err := dwsv1alpha1.DeleteChildrenWithLabels(ctx, r.Client, childObjects, workflow, client.MatchingLabels{teardownStateLabel: dwsv1alpha1.StatePostRun.String()})
		if err != nil {
			return nil, err
		}

		if deleteStatus == dwsv1alpha1.DeleteRetry {
			return &ctrl.Result{Requeue: true}, nil
		}

		if err := r.Delete(ctx, dm); err != nil {
			if !apierrors.IsNotFound(err) {
				return nil, err
			}
		}
	}

	access := &nnfv1alpha1.NnfAccess{
		ObjectMeta: metav1.ObjectMeta{
			Name:      indexedResourceName(workflow, index) + "-computes",
			Namespace: workflow.Namespace,
		},
	}

	if err := r.Get(ctx, client.ObjectKeyFromObject(access), access); err == nil {
		if access.Status.State != "unmounted" {
			return &ctrl.Result{Requeue: true}, nil
		}
	}

	if err := r.Delete(ctx, access); err != nil {
		if !apierrors.IsNotFound(err) {
			return nil, err
		}

		return nil, nil
	}

	return nil, nil
}

func (r *NnfWorkflowReconciler) startTeardownState(ctx context.Context, workflow *dwsv1alpha1.Workflow, index int) (*ctrl.Result, error) {
	// Delete the NnfDataMovement resource that was created during pre-run. This is named after
	// the workflow, and it does not have a directive index in the name due to the way client initiated
	// data movement works.
	dataMovement := &nnfv1alpha1.NnfDataMovement{
		ObjectMeta: metav1.ObjectMeta{
			Name:      workflow.GetName(),
			Namespace: workflow.GetNamespace(),
		},
	}

	if err := r.Get(ctx, client.ObjectKeyFromObject(dataMovement), dataMovement); err != nil {
		if !apierrors.IsNotFound(err) {
			return nil, err
		}
	} else {
		if err := r.Delete(ctx, dataMovement); err != nil {
			if !apierrors.IsNotFound(err) {
				return nil, err
			}
		}
	}

	// Delete the NnfDataMovement and NnfAccess for this directive before removing the NnfStorage.
	// copy_in/out directives can reference NnfStorage from a different directive, so all the NnfAccesses
	// need to be removed first.
	childObjects := []dwsv1alpha1.ObjectList{
		&nnfv1alpha1.NnfDataMovementList{},
		&nnfv1alpha1.NnfAccessList{},
	}

	deleteStatus, err := dwsv1alpha1.DeleteChildrenWithLabels(ctx, r.Client, childObjects, workflow, client.MatchingLabels{directiveIndexLabel: strconv.Itoa(index)})
	if err != nil {
		return nil, err
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
			return nil, err
		}

		persistentStorage.SetOwnerReferences([]metav1.OwnerReference{})
		dwsv1alpha1.RemoveOwnerLabels(persistentStorage)
		labels := persistentStorage.GetLabels()
		delete(labels, directiveIndexLabel)
		persistentStorage.SetLabels(labels)

		err = r.Update(ctx, persistentStorage)
		if err != nil {
			return nil, err
		}
		log.Info("Removed owner reference from persistent storage", "psi", persistentStorage)
	case "destroy_persistent":
		persistentStorage, err := r.findPersistentInstance(ctx, workflow, dwArgs["name"])
		if err != nil {
			return nil, client.IgnoreNotFound(err)
		}

		dwsv1alpha1.AddOwnerLabels(persistentStorage, workflow)
		addDirectiveIndexLabel(persistentStorage, index)

		if err := controllerutil.SetControllerReference(workflow, persistentStorage, r.Scheme); err != nil {
			log.Info("Unable to assign workflow as owner of persistentInstance", "psi", persistentStorage)
			return nil, err
		}

		err = r.Update(ctx, persistentStorage)
		if err != nil {
			return nil, err
		}
		log.Info("Add owner reference for persistent storage for deletion", "psi", persistentStorage)
	default:
	}

	childObjects := []dwsv1alpha1.ObjectList{
		&nnfv1alpha1.NnfStorageList{},
		&dwsv1alpha1.PersistentStorageInstanceList{},
	}

	deleteStatus, err := dwsv1alpha1.DeleteChildrenWithLabels(ctx, r.Client, childObjects, workflow, client.MatchingLabels{directiveIndexLabel: strconv.Itoa(index)})
	if err != nil {
		return nil, err
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
		Owns(&nnfv1alpha1.NnfDataMovement{}).
		Owns(&dwsv1alpha1.DirectiveBreakdown{}).
		Owns(&dwsv1alpha1.PersistentStorageInstance{}).
		Watches(&source.Kind{Type: &nnfv1alpha1.NnfStorage{}}, handler.EnqueueRequestsFromMapFunc(dwsv1alpha1.OwnerLabelMapFunc)).
		Complete(r)
}
