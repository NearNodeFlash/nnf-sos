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
	"strings"

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
func (r *NnfWorkflowReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("Workflow", req.NamespacedName)

	// Fetch the Workflow instance
	workflow := &dwsv1alpha1.Workflow{}

	err := r.Get(ctx, req.NamespacedName, workflow)
	if err != nil {
		// ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	driverID := os.Getenv("DWS_DRIVER_ID")

	// Check if the object is being deleted
	if !workflow.GetDeletionTimestamp().IsZero() {
		log.Info("Deleting workflow...")

		if !controllerutil.ContainsFinalizer(workflow, finalizerNnfWorkflow) {
			return ctrl.Result{}, nil
		}

		// Run the teardown state logic in case the workflow was deleted without
		// transitioning to teardown first
		retry, err := r.handleTeardown(ctx, workflow, log)
		if err != nil {
			return ctrl.Result{}, err
		}

		if retry == true {
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

	// Handle the state work required to achieve the desired State.
	// The model right now is that we continue to call the state handler until it completes the state
	// in which case the reconciler will no longer be called.
	// TODO: Look into have each handler return a "ImNotFinished" to cause a ctrl.Result{Request: repeat}
	//       until the state is finished.
	//       Once the state is finished, we can update the drivers in 1 spot. Right now, the drivers are updated
	//       at the end of each state handler.

	switch workflow.Status.State {
	case dwsv1alpha1.StateProposal.String():
		return r.handleProposalState(ctx, workflow, driverID, log)
	case dwsv1alpha1.StateSetup.String():
		return r.handleSetupState(ctx, workflow, driverID, log)
	case dwsv1alpha1.StateDataIn.String():
		return r.handleDataInState(ctx, workflow, driverID, log)
	case dwsv1alpha1.StatePreRun.String():
		return r.handlePreRunState(ctx, workflow, driverID, log)
	case dwsv1alpha1.StatePostRun.String():
		return r.handlePostRunState(ctx, workflow, driverID, log)
	case dwsv1alpha1.StateDataOut.String():
		return r.handleDataOutState(ctx, workflow, driverID, log)
	case dwsv1alpha1.StateTeardown.String():
		return r.handleTeardownState(ctx, workflow, driverID, log)
	default:
		return r.handleUnsupportedState(ctx, workflow, driverID, log)
	}
}

func (r *NnfWorkflowReconciler) updateDriversStatusForStatusState(workflow *dwsv1alpha1.Workflow, driverID string, log logr.Logger) bool {

	updateWorkflow := false
	for i := range workflow.Status.Drivers {
		driverStatus := &workflow.Status.Drivers[i]

		if driverStatus.DriverID != driverID {
			continue
		}
		if workflow.Status.State != driverStatus.WatchState {
			continue
		}
		if workflow.Status.Ready {
			continue
		}
		if driverStatus.Completed {
			continue
		}

		ts := metav1.NowMicro()
		driverStatus.CompleteTime = &ts
		driverStatus.Completed = true
		updateWorkflow = true
	}

	return updateWorkflow
}

func (r *NnfWorkflowReconciler) completeDriverState(ctx context.Context, workflow *dwsv1alpha1.Workflow, driverID string, log logr.Logger) (ctrl.Result, error) {

	driverUpdate := r.updateDriversStatusForStatusState(workflow, driverID, log)
	if driverUpdate {
		if err := r.Update(ctx, workflow); err != nil {
			if apierrors.IsConflict(err) {
				return ctrl.Result{Requeue: true}, nil
			}

			log.Error(err, "Failed to update Workflow state")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *NnfWorkflowReconciler) failDriverState(ctx context.Context, workflow *dwsv1alpha1.Workflow, driverID string, message string) (ctrl.Result, error) {
	for i := range workflow.Status.Drivers {
		driver := &workflow.Status.Drivers[i]
		if driver.DriverID == driverID && workflow.Status.State == driver.WatchState {
			driver.Reason = "error"
			driver.Message = message
			break
		}
	}

	return r.completeDriverState(ctx, workflow, driverID, r.Log)
}

func (r *NnfWorkflowReconciler) handleUnsupportedState(ctx context.Context, workflow *dwsv1alpha1.Workflow, driverID string, log logr.Logger) (ctrl.Result, error) {
	log.Info(workflow.Status.State)

	log.Info("Unsupported Status.State", "state", workflow.Status.State)

	// TODO: This should return an error, but for now unhandled states simply succeed to allow Flux to exercise workflow...
	// Complete state in the drivers
	return r.completeDriverState(ctx, workflow, driverID, log)
}

func (r *NnfWorkflowReconciler) handleProposalState(ctx context.Context, workflow *dwsv1alpha1.Workflow, driverID string, log logr.Logger) (ctrl.Result, error) {
	log.Info(workflow.Status.State)

	if err := r.validateWorkflow(ctx, workflow); err != nil {
		log.Error(err, "Unable to validate workflow")
		return r.failDriverState(ctx, workflow, driverID, err.Error())
	}

	// For each directive breakdown that will require an NnfStorageProfile,
	// create a pinned copy of the profile.
	if pResult, err := r.generatePinnedStorageProfiles(ctx, workflow, log); err != nil {
		log.Error(err, "Unable to generate pinned NnfStorageProfile resources for new DirectiveBreakdown resources")
		return r.failDriverState(ctx, workflow, driverID, err.Error())
	} else if pResult.Requeue {
		return pResult, nil
	}

	// Generate a list of directive breakdowns from the #DWs present (if any)
	var dbList []v1.ObjectReference
	for dwIndex, dwDirective := range workflow.Spec.DWDirectives {
		_, directiveBreakdown, err := r.generateDirectiveBreakdown(ctx, dwDirective, dwIndex, workflow, log)
		if err != nil {
			return ctrl.Result{}, err
		}

		if directiveBreakdown != nil {
			dbList = append(dbList, v1.ObjectReference{
				Kind:      reflect.TypeOf(dwsv1alpha1.DirectiveBreakdown{}).Name(),
				Name:      directiveBreakdown.Name,
				Namespace: directiveBreakdown.Namespace})
		}
	}

	// If the workflow already has the correct list of directiveBreakdown references, keep going.
	if !reflect.DeepEqual(workflow.Status.DirectiveBreakdowns, dbList) {
		log.Info("Updating directiveBreakdown references")
		workflow.Status.DirectiveBreakdowns = dbList

		if err := r.Update(ctx, workflow); err != nil {
			if apierrors.IsConflict(err) {
				return ctrl.Result{Requeue: true}, nil
			}

			log.Error(err, "Failed to update Workflow DirectiveBreakdowns")
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	// Ensure all DirectiveBreakdowns are ready
	for _, bdRef := range workflow.Status.DirectiveBreakdowns {
		directiveBreakdown := &dwsv1alpha1.DirectiveBreakdown{}

		err := r.Get(ctx, types.NamespacedName{Namespace: bdRef.Namespace, Name: bdRef.Name}, directiveBreakdown)
		if err != nil {
			// We expect the DirectiveBreakdown to be there since we checked in the loop above
			log.Error(err, "Failed to get DirectiveBreakdown", "name", bdRef.Name)
			return ctrl.Result{Requeue: true}, nil
		}

		// Wait for the breakdown to be ready
		if directiveBreakdown.Status.Ready != ConditionTrue {
			return ctrl.Result{}, nil
		}
	}

	// Complete state in the drivers
	return r.completeDriverState(ctx, workflow, driverID, log)
}

func (r *NnfWorkflowReconciler) generatePinnedStorageProfiles(ctx context.Context, workflow *dwsv1alpha1.Workflow, log logr.Logger) (ctrl.Result, error) {

	result := ctrl.Result{}
	for dwIndex, directive := range workflow.Spec.DWDirectives {
		// The webhook validated directives, ignore errors
		dwArgs, _ := dwdparse.BuildArgsMap(directive)
		command := dwArgs["command"]

		if command != "jobdw" && command != "create_persistent" {
			continue
		}

		pinnedName, _ := getStorageReferenceNameFromWorkflowIntended(workflow, dwIndex)
		opResult, err := pinProfile(ctx, r.Client, r.Scheme, dwArgs, workflow, pinnedName)
		if err != nil {
			log.Error(err, "Failed to pin NnfStorageProfile")
			return result, err
		}
		if opResult != controllerutil.OperationResultNone {
			// If we created one, then ask for a requeue.
			log.Info("Created NnfStorageProfile", "name", pinnedName)
			result.Requeue = true
		}
	}
	return result, nil
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

		case "delete_persistent":
			deletePersistentCount++

		case "persistentdw":
			if err := r.validatePersistentInstanceDirective(ctx, wf, directive); err != nil {
				return err
			}
		}
	}

	if directiveCount > 1 {
		// Ensure create_persistent or delete_persistent are singletons in the workflow
		if createPersistentCount+deletePersistentCount > 0 {
			return fmt.Errorf("a single create_persistent or delete_persistent directive allowed per workflow")
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

// generateDirectiveBreakdown creates a DirectiveBreakdown for any #DW directive that causes storage to be allocated.
func (r *NnfWorkflowReconciler) generateDirectiveBreakdown(ctx context.Context, directive string, dwIndex int, workflow *dwsv1alpha1.Workflow, log logr.Logger) (result controllerutil.OperationResult, directiveBreakdown *dwsv1alpha1.DirectiveBreakdown, err error) {

	// DWDirectives that we need to generate directiveBreakdowns for look like this:
	//  #DW command            arguments...
	//  --- -----------------  -------------------------------------------
	// "#DW jobdw              type=raw    capacity=9TB name=thisIsReallyRaw"
	// "#DW jobdw              type=xfs    capacity=9TB name=thisIsXFS"
	// "#DW jobdw              type=gfs2   capacity=9GB name=thisIsGfs2"
	// "#DW jobdw              type=lustre capacity=9TB name=thisIsLustre"
	// "#DW create_persistent  type=lustre capacity=9TB name=thisIsPersistent"

	// #DW commands that require a dwDirectiveBreakdown because they create storage
	breakDownCommands := []string{
		"jobdw",
		"create_persistent",
	}

	// Initialize return parameters
	directiveBreakdown = nil
	result = controllerutil.OperationResultNone
	err = nil

	cmdElements := strings.Fields(directive)
	for _, breakThisDown := range breakDownCommands {
		// We care about the commands that generate a breakdown
		if breakThisDown == cmdElements[1] {

			lifetime := "job"
			dwdName := createDirectiveBreakdownName(workflow, dwIndex)
			switch cmdElements[1] {
			case "create_persistent":
				lifetime = "persistent"
			}

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

					// Construct a map of the arguments within the directive
					m := make(map[string]string)
					for _, pair := range cmdElements[2:] {
						if arg := strings.Split(pair, "="); len(arg) > 1 {
							m[arg[0]] = arg[1]
						}
					}

					directiveBreakdown.Spec.DW.DWDirectiveIndex = dwIndex
					directiveBreakdown.Spec.DW.DWDirective = directive
					directiveBreakdown.Spec.Name = m["name"]
					directiveBreakdown.Spec.Type = m["type"]
					directiveBreakdown.Spec.Lifetime = lifetime

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

func needDirectiveBreakdownReference(dbdNeeded *dwsv1alpha1.DirectiveBreakdown, workflow *dwsv1alpha1.Workflow) bool {
	for _, dbd := range workflow.Status.DirectiveBreakdowns {
		// If we already have a reference to the specified dbd, we don't need another one
		if (dbd.Name == dbdNeeded.ObjectMeta.Name) && (dbd.Namespace == dbdNeeded.ObjectMeta.Namespace) {
			return false
		}
	}

	// Did not find it, so we need to create
	return true
}

func (r *NnfWorkflowReconciler) handleSetupState(ctx context.Context, workflow *dwsv1alpha1.Workflow, driverID string, log logr.Logger) (ctrl.Result, error) {
	log.Info(workflow.Status.State)

	// We don't start looking for NnfStorage to be "Ready" until we've created all of them.
	// In the following loop if we create anything, we requeue before we look at any
	// NnfStorage readiness.
	var nnfStorages []nnfv1alpha1.NnfStorage

	// Iterate through the DirectiveBreakdowns
	for _, dbdRef := range workflow.Status.DirectiveBreakdowns {

		// Chain through the DirectiveBreakdown to the Servers object
		dbd := &dwsv1alpha1.DirectiveBreakdown{
			ObjectMeta: metav1.ObjectMeta{
				Name:      dbdRef.Name,
				Namespace: dbdRef.Namespace,
			},
		}
		err := r.Get(ctx, client.ObjectKeyFromObject(dbd), dbd)
		if err != nil {
			log.Error(err, "Unable to get directiveBreakdown", "dbd", types.NamespacedName{Name: dbdRef.Name, Namespace: dbdRef.Namespace})
			return ctrl.Result{}, err
		}

		s := &dwsv1alpha1.Servers{
			ObjectMeta: metav1.ObjectMeta{
				Name:      dbd.Status.Servers.Name,
				Namespace: dbd.Status.Servers.Namespace,
			},
		}
		err = r.Get(ctx, client.ObjectKeyFromObject(s), s)
		if err != nil {
			log.Error(err, "Unable to get servers", "servers", types.NamespacedName{Name: dbd.Status.Servers.Name, Namespace: dbd.Status.Servers.Namespace})
			return ctrl.Result{}, err
		}

		if _, present := os.LookupEnv("RABBIT_TEST_ENV_BYPASS_SERVER_STORAGE_CHECK"); !present {
			if len(dbd.Status.AllocationSet) != 0 && len(dbd.Status.AllocationSet) != len(s.Spec.AllocationSets) {
				return r.failDriverState(ctx, workflow, driverID, fmt.Sprintf("Servers resource does not meet storage requirements for directive '%s'", dbd.Spec.DW.DWDirective))
			}
		}

		nnfStorage, err := r.createNnfStorage(ctx, workflow, dbd, s, log)
		if err != nil {
			if apierrors.IsConflict(err) {
				return ctrl.Result{Requeue: true}, nil
			}

			log.Error(err, "Failed to create nnf storage")
			return ctrl.Result{}, err
		}

		// Allow an empty Servers object, only append if we have an nnfStorage object
		if nnfStorage != nil {
			nnfStorages = append(nnfStorages, *nnfStorage)
		}
	}

	// Add any nnfStorages associated with 'persistentdw' directives, so they can be checked for READY as well
	for _, directive := range workflow.Spec.DWDirectives {
		dwArgs, _ := dwdparse.BuildArgsMap(directive)
		command := dwArgs["command"]

		if command == "persistentdw" {
			nnfStorage, err := r.findNnfStorageForPersistentDw(ctx, dwArgs["name"], workflow.Namespace, log)
			if err != nil {
				// Error has already been logged
				return ctrl.Result{}, err
			}

			nnfStorages = append(nnfStorages, *nnfStorage)
		}
	}

	// Walk the nnfStorages looking for them to be Ready. We exit the reconciler if
	// - the status section of the NnfStorage has not yet been filled in
	// - the AllocationSet is not ready indicating we haven't finished creating the storage
	for i := range nnfStorages {
		// If the Status section has not been filled in yet, exit and wait.
		if len(nnfStorages[i].Status.AllocationSets) != len(nnfStorages[i].Spec.AllocationSets) {
			return ctrl.Result{Requeue: true}, nil
		}

		// Status section should be usable now, check for Ready
		for _, set := range nnfStorages[i].Status.AllocationSets {
			if set.Status != "Ready" {
				return ctrl.Result{Requeue: true}, nil
			}
		}
	}

	// Complete state in the drivers
	return r.completeDriverState(ctx, workflow, driverID, log)
}

func (r *NnfWorkflowReconciler) createNnfStorage(ctx context.Context, workflow *dwsv1alpha1.Workflow, d *dwsv1alpha1.DirectiveBreakdown, s *dwsv1alpha1.Servers, log logr.Logger) (*nnfv1alpha1.NnfStorage, error) {
	nnfStorage := &nnfv1alpha1.NnfStorage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      s.Name,
			Namespace: s.Namespace,
		},
	}

	dwArgs, err := dwdparse.BuildArgsMap(d.Spec.DW.DWDirective)
	if err != nil {
		return nil, err
	}

	pinnedName, pinnedNamespace := getStorageReferenceNameFromDBD(d)
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
			dwsv1alpha1.InheritParentLabels(nnfStorage, owner)
			dwsv1alpha1.AddOwnerLabels(nnfStorage, owner)
			nnfStorage.Spec.FileSystemType = d.Spec.Type

			// Need to remove all of the AllocationSets in the NnfStorage object before we begin
			nnfStorage.Spec.AllocationSets = []nnfv1alpha1.NnfStorageAllocationSetSpec{}

			// Iterate the Servers data elements to pull out the allocation sets for the server
			for i := range s.Spec.AllocationSets {
				nnfAllocSet := nnfv1alpha1.NnfStorageAllocationSetSpec{}

				nnfAllocSet.Name = s.Spec.AllocationSets[i].Label
				nnfAllocSet.Capacity = s.Spec.AllocationSets[i].AllocationSize
				if d.Spec.Type == "lustre" {
					nnfAllocSet.NnfStorageLustreSpec.TargetType = strings.ToUpper(s.Spec.AllocationSets[i].Label)
					nnfAllocSet.NnfStorageLustreSpec.BackFs = "zfs"
					charsWanted := 8
					if len(d.Spec.Name) < charsWanted {
						charsWanted = len(d.Spec.Name)
					}
					nnfAllocSet.NnfStorageLustreSpec.FileSystemName = d.Spec.Name[:charsWanted]
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

	// Let this NnfStorage have ownership over the NnfStorageProfile it is using.
	result, err = setPinnedProfileOwnership(ctx, nnfStorage, r.Client, r.Scheme, nnfStorageProfile)
	if err != nil {
		if !apierrors.IsConflict(err) {
			log.Error(err, "Unable to add new owner to NnfStorageProfile", "name", nnfStorageProfile.GetName())
		}
		return nil, err
	} else if result != controllerutil.OperationResultNone {
		log.Info("Updated NnfStorageProfile", "name", nnfStorageProfile.GetName())
	}

	return nnfStorage, nil
}

// Returns an NnfStorage object matching #DW persistentdw <name>
func (r *NnfWorkflowReconciler) findNnfStorageForPersistentDw(ctx context.Context, psiName string, psiNamespace string, log logr.Logger) (*nnfv1alpha1.NnfStorage, error) {

	nnfStorage := &nnfv1alpha1.NnfStorage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      psiName,
			Namespace: psiNamespace,
		},
	}

	err := r.Get(ctx, client.ObjectKeyFromObject(nnfStorage), nnfStorage)
	if err != nil {
		log.Error(err, "Unable to get nnfStorage", "nnfStorage", nnfStorage)
		return nil, err
	}

	return nnfStorage, err
}

func (r *NnfWorkflowReconciler) handleDataInState(ctx context.Context, workflow *dwsv1alpha1.Workflow, driverID string, log logr.Logger) (ctrl.Result, error) {
	return r.handleDataInOutState(ctx, workflow, driverID, log)
}

func (r *NnfWorkflowReconciler) handleDataOutState(ctx context.Context, workflow *dwsv1alpha1.Workflow, driverID string, log logr.Logger) (ctrl.Result, error) {
	return r.handleDataInOutState(ctx, workflow, driverID, log)
}

func (r *NnfWorkflowReconciler) handleDataInOutState(ctx context.Context, workflow *dwsv1alpha1.Workflow, driverID string, log logr.Logger) (ctrl.Result, error) {
	log.Info(workflow.Status.State)

	hasCopyInOutDirective := false
	copyInDirectivesFinished := true
	for directiveIdx, directive := range workflow.Spec.DWDirectives {

		shouldProcessDirective := false
		if workflow.Status.State == dwsv1alpha1.StateDataIn.String() && strings.HasPrefix(directive, "#DW copy_in") {
			shouldProcessDirective = true
		}
		if workflow.Status.State == dwsv1alpha1.StateDataOut.String() && strings.HasPrefix(directive, "#DW copy_out") {
			shouldProcessDirective = true
		}

		if shouldProcessDirective {
			hasCopyInOutDirective = true

			name := fmt.Sprintf("%s-%d", workflow.Name, directiveIdx) // TODO: Should this move to a MakeName()?
			dm := &nnfv1alpha1.NnfDataMovement{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: workflow.Namespace,
				},
			}

			if err := r.Get(ctx, client.ObjectKeyFromObject(dm), dm); err != nil {
				if !apierrors.IsNotFound(err) {
					return ctrl.Result{}, err
				}

				return r.createDataMovementResource(ctx, workflow, directiveIdx)
			}

			// Monitor data movement resource
			if finished, err := r.monitorDataMovementResource(ctx, dm); err != nil {
				return ctrl.Result{}, err
			} else {
				if finished {
					result, err := r.teardownDataMovementResource(ctx, dm, workflow.Status.State)
					if err != nil {
						return ctrl.Result{}, err
					} else if !result.IsZero() {
						return *result, nil
					}
				} else {
					copyInDirectivesFinished = false
				}
			}

		} // if strings.HasPrefix(directive, "#DW copy_in") || strings.HasPrefix(directive, "#DW copy_out")
	}

	if !hasCopyInOutDirective {
		return r.completeDriverState(ctx, workflow, driverID, log)
	}

	if copyInDirectivesFinished {
		return r.completeDriverState(ctx, workflow, driverID, log)
	}

	return ctrl.Result{}, nil
}

func (r *NnfWorkflowReconciler) createDataMovementResource(ctx context.Context, workflow *dwsv1alpha1.Workflow, dwIndex int) (ctrl.Result, error) {

	parameters, err := dwdparse.BuildArgsMap(workflow.Spec.DWDirectives[dwIndex])
	if err != nil {
		workflow.Status.Message = fmt.Sprintf("Stage %s failed: %v", workflow.Spec.DesiredState, err)
		return ctrl.Result{}, err
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
				nnfStorageName = fmt.Sprintf("%s-%d", workflow.Name, directiveIdx)
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
		return ctrl.Result{}, err
	} else if result != nil {
		return *result, nil
	}

	destinationStorage, destinationAccess, destinationPath, result, err := prepareStagingArgumentFn(parameters["destination"])
	if err != nil {
		return ctrl.Result{}, err
	} else if result != nil {
		return *result, nil
	}

	// Wait for accesses to go ready
	for _, access := range []*nnfv1alpha1.NnfAccess{sourceAccess, destinationAccess} {
		if access != nil {
			if err := r.Get(ctx, client.ObjectKeyFromObject(access), access); err != nil {
				return ctrl.Result{}, err
			}

			if access.Status.Ready == false {
				return ctrl.Result{}, nil
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

	name := fmt.Sprintf("%s-%d", workflow.Name, dwIndex) // TODO: Need a naming convention for all workflow derived objects
	dm := &nnfv1alpha1.NnfDataMovement{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: workflow.Namespace,
		},
	}

	dwsv1alpha1.AddWorkflowLabels(dm, workflow)
	dwsv1alpha1.AddOwnerLabels(dm, workflow)

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
		return ctrl.Result{}, err
	}

	if err := r.Create(ctx, dm); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
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
			Name:      fmt.Sprintf("%s-%d-%s", workflow.Name, directiveIdx, "servers"),
			Namespace: workflow.Namespace,
		},
	}

	result, err := ctrl.CreateOrUpdate(ctx, r.Client, access,
		func() error {
			dwsv1alpha1.AddWorkflowLabels(access, workflow)
			dwsv1alpha1.AddOwnerLabels(access, workflow)

			access.Spec = nnfv1alpha1.NnfAccessSpec{
				DesiredState:    "mounted",
				TeardownState:   teardownState,
				Target:          "all",
				MountPathPrefix: fmt.Sprintf("/mnt/nnf/%d/job/%s", workflow.Spec.JobID, params["name"]),

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
func (r *NnfWorkflowReconciler) monitorDataMovementResource(ctx context.Context, dm *nnfv1alpha1.NnfDataMovement) (bool, error) {

	for _, condition := range dm.Status.Conditions {
		if condition.Type == nnfv1alpha1.DataMovementConditionTypeFinished {
			return true, nil
		}
	}

	return false, nil
}

// Teardown a data movement resource and its subresources. This prepares the resource for deletion but does not delete it.
func (r *NnfWorkflowReconciler) teardownDataMovementResource(ctx context.Context, dm *nnfv1alpha1.NnfDataMovement, workflowState string) (*ctrl.Result, error) {
	deleteAccess := func(ref *corev1.ObjectReference) (*ctrl.Result, error) {
		if ref == nil {
			return nil, nil
		}

		access := &nnfv1alpha1.NnfAccess{}
		if err := r.Get(ctx, types.NamespacedName{Name: ref.Name, Namespace: ref.Namespace}, access); err != nil {
			return nil, client.IgnoreNotFound(err)
		}

		if workflowState != access.Spec.TeardownState {
			return nil, nil
		}

		if err := r.Delete(ctx, access); err != nil {
			if client.IgnoreNotFound(err) != nil {
				return nil, err
			}
		}

		return nil, nil
	}

	if result, err := deleteAccess(dm.Spec.Source.Access); !result.IsZero() || err != nil {
		return result, err
	}

	if result, err := deleteAccess(dm.Spec.Destination.Access); !result.IsZero() || err != nil {
		return result, err
	}

	return nil, nil
}

func (r *NnfWorkflowReconciler) handlePreRunState(ctx context.Context, workflow *dwsv1alpha1.Workflow, driverID string, log logr.Logger) (ctrl.Result, error) {
	log.Info(workflow.Status.State)
	complete := true

	for i := range workflow.Status.Drivers {
		driverStatus := &workflow.Status.Drivers[i]

		// Check if the driver status is for the pre_run state
		if driverStatus.WatchState != dwsv1alpha1.StatePreRun.String() {
			continue
		}

		// Only look for driver status for the NNF driver
		if driverStatus.DriverID != driverID {
			continue
		}

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
			dm.Spec.Monitor = true

			return ctrl.SetControllerReference(workflow, dm, r.Scheme)
		})
		if err != nil {
			return ctrl.Result{}, err
		}

		if result == controllerutil.OperationResultCreated {
			log.Info("Created NnfDatamovement", "name", dm.Name)
		}

		// Parse the #DW line
		args, err := dwdparse.BuildArgsMap(workflow.Spec.DWDirectives[driverStatus.DWDIndex])
		if err != nil {
			return ctrl.Result{}, err
		}

		access := &nnfv1alpha1.NnfAccess{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-%d-%s", workflow.Name, driverStatus.DWDIndex, "computes"),
				Namespace: workflow.Namespace,
			},
		}

		// Create an NNFAccess for the compute clients
		mountPath := buildMountPath(workflow, args["name"], args["command"])
		result, err = ctrl.CreateOrUpdate(ctx, r.Client, access,
			func() error {
				dwsv1alpha1.AddWorkflowLabels(access, workflow)
				dwsv1alpha1.AddOwnerLabels(access, workflow)

				access.Spec.TeardownState = dwsv1alpha1.StatePostRun.String()
				access.Spec.DesiredState = "mounted"
				access.Spec.Target = "single"
				access.Spec.MountPath = mountPath
				access.Spec.ClientReference = corev1.ObjectReference{
					Name:      workflow.Name,
					Namespace: workflow.Namespace,
					Kind:      "Computes",
				}

				// Determine the name/namespace to use based on the directive
				name, namespace := getStorageReferenceNameFromWorkflowActual(workflow, driverStatus.DWDIndex)

				access.Spec.StorageReference = corev1.ObjectReference{
					// Directive Breakdowns share the same NamespacedName with the Servers it creates, which shares the same name with the NNFStorage.
					Name:      name,
					Namespace: namespace,
					Kind:      reflect.TypeOf(nnfv1alpha1.NnfStorage{}).Name(),
				}

				return ctrl.SetControllerReference(workflow, access, r.Scheme)
			})
		if err != nil {
			return ctrl.Result{}, err
		}

		if result == controllerutil.OperationResultCreated {
			log.Info("Created NnfAccess", "name", access.Name)
		} else if result == controllerutil.OperationResultNone {
			// no change
		} else {
			log.Info("Updated NnfAccess", "name", access.Name)
		}

		if access.Status.State != access.Spec.DesiredState || access.Status.Ready == false {
			complete = false
		}

		// Add an environment variable to the workflow status section for the location of the
		// mount point on the clients.
		if workflow.Status.Env == nil {
			workflow.Status.Env = make(map[string]string)
		}

		envName := ""
		switch args["command"] {
		case "jobdw":
			envName = "DW_JOB_" + args["name"]
		case "persistentdw":
			envName = "DW_PERSISTENT_" + args["name"]
		default:
			return ctrl.Result{}, fmt.Errorf("Unexpected #DW command %s", args["command"])
		}
		workflow.Status.Env[envName] = mountPath

		// Create an NnfAccess for the servers resources if necessary. Shared storage like
		// that of GFS2 provides access to the Rabbit for the lifetime of the user's job.
		// NnfAccess may already be present if a data_in directive was specified for the
		// particular $JOB_DW_[name]; in this case we only don't need to recreate the
		// resource.

		if (args["command"] == "jobdw") || (args["command"] == "persistentdw") {
			if args["type"] == "gfs2" { // TODO: Should this include Lustre?

				storage := &nnfv1alpha1.NnfStorage{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("%s-%d", workflow.Name, driverStatus.DWDIndex),
						Namespace: workflow.Namespace,
					},
				}

				// Set the teardown state to post run. If there is a copy_out directive that uses
				// this storage instance, set the teardown state so NNF Access is preserved through
				// data_out.
				teardownState := dwsv1alpha1.StatePostRun.String()
				if findCopyOutDirectiveIndexByName(workflow, args["name"]) >= 0 {
					teardownState = dwsv1alpha1.StateDataOut.String()
				}

				access, result, err := r.setupNnfAccessForServers(ctx, storage, workflow, driverStatus.DWDIndex, teardownState)
				if err != nil {
					return ctrl.Result{}, err
				} else if result == controllerutil.OperationResultCreated {
					log.Info("Created NnfAccess", "name", access.Name)
				} else if result == controllerutil.OperationResultUpdated {
					log.Info("Updated NnfAccess", "name", access.Name)
				}

				if access.Status.State != access.Spec.DesiredState || access.Status.Ready == false {
					complete = false
				}
			}

		}
	}

	if complete == false {
		return ctrl.Result{}, nil
	}

	r.completeDriverState(ctx, workflow, driverID, log)

	return ctrl.Result{}, nil
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

func (r *NnfWorkflowReconciler) handlePostRunState(ctx context.Context, workflow *dwsv1alpha1.Workflow, driverID string, log logr.Logger) (ctrl.Result, error) {
	log.Info(workflow.Status.State)
	retry := false

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
			return ctrl.Result{}, err
		}

		// Other logic in this state might have requeued while the data movement resource was
		// successfully deleted. In this case, clear the pointer so it is not processed further.
		dm = nil
	}

	if dm != nil {

		// Stop the data movement resource from monitoring subresources. This will permit the
		// data movement resource to reach a finished state where the status fields are valid.
		if dm.Spec.Monitor == true {
			dm.Spec.Monitor = false
			if err := r.Update(ctx, dm); err != nil {
				return ctrl.Result{}, err
			}

			return ctrl.Result{Requeue: true}, nil
		}

		finished, err := r.monitorDataMovementResource(ctx, dm)
		if err != nil {
			return ctrl.Result{}, err
		}

		if finished && dm.DeletionTimestamp.IsZero() {

			if len(dm.Status.Conditions) != 0 {
				condition := &dm.Status.Conditions[len(dm.Status.Conditions)-1]
				if condition.Reason == nnfv1alpha1.DataMovementConditionReasonFailed {
					return r.failDriverState(ctx, workflow, driverID, condition.Message)
				}
			}

			if err := r.Delete(ctx, dm); err != nil {
				if !apierrors.IsNotFound(err) {
					return ctrl.Result{}, err
				}
			}

		}

		// Data movement is still in progress; return here so the NnfAccess is not destroyed
		// until after all data movement resources are finished.
		return ctrl.Result{}, nil
	}

	// Retrieve the NNF Accesses for this workflow, unmount and delete them if necessary. Each
	// access has a TeardownState defined in the specification that tells this logic when to
	// teardown the NNF Access. Filter on NNF Accesses matching the target workflow and in the
	// Post-Run teardown state.
	accesses := &nnfv1alpha1.NnfAccessList{}
	listOptions := []client.ListOption{
		dwsv1alpha1.MatchingOwner(workflow),
		// NJR Would be really sweet if this worked!
		//client.MatchingFields{
		//	".spec.teardownState": dwsv1alpha1.StatePostRun.String(),
		//},
	}

	if err := r.List(ctx, accesses, listOptions...); err != nil {
		return ctrl.Result{}, err
	}

	for _, access := range accesses.Items {

		if access.Spec.TeardownState != dwsv1alpha1.StatePostRun.String() {
			continue
		}

		if access.Status.State == "unmounted" {
			if access.Status.Ready == false {
				retry = true
			} else {

				if err := r.Delete(ctx, &access); err != nil {
					if !apierrors.IsNotFound(err) {
						return ctrl.Result{}, err
					}
				}

			}
		}

		access.Spec.DesiredState = "unmounted"
		if err := r.Update(ctx, &access); err != nil {
			if !apierrors.IsConflict(err) {
				return ctrl.Result{}, err
			}
		}

		retry = true
	}

	if retry == true {
		return ctrl.Result{}, nil
	}

	r.completeDriverState(ctx, workflow, driverID, log)

	return ctrl.Result{}, nil
}

func (r *NnfWorkflowReconciler) handleTeardown(ctx context.Context, workflow *dwsv1alpha1.Workflow, log logr.Logger) (bool, error) {
	// Add owner labels to the persistentStorageInstances named by a "delete_persistent" #DW.
	// They'll get deleted below in the call to deleteChildren
	err := r.teardownPersistentStorageInstances(ctx, workflow)
	if err != nil {
		return true, err
	}

	childObjects := []dwsv1alpha1.ObjectList{
		&nnfv1alpha1.NnfDataMovementList{},
		&nnfv1alpha1.NnfAccessList{},
		&nnfv1alpha1.NnfStorageList{},
		&dwsv1alpha1.PersistentStorageInstanceList{},
	}

	deleteStatus, err := dwsv1alpha1.DeleteChildren(ctx, r.Client, childObjects, workflow)
	if err != nil {
		return true, err
	}

	if deleteStatus == dwsv1alpha1.DeleteRetry {
		return true, nil
	}

	return false, nil
}

func (r *NnfWorkflowReconciler) handleTeardownState(ctx context.Context, workflow *dwsv1alpha1.Workflow, driverID string, log logr.Logger) (ctrl.Result, error) {
	retry, err := r.handleTeardown(ctx, workflow, log)
	if err != nil {
		return ctrl.Result{}, err
	}

	if retry == true {
		return ctrl.Result{}, nil
	}

	// Complete state in the drivers
	return r.completeDriverState(ctx, workflow, driverID, log)
}

// Delete persistent storage specified with delete_persistent directive
func (r *NnfWorkflowReconciler) teardownPersistentStorageInstances(ctx context.Context, wf *dwsv1alpha1.Workflow) error {
	log := r.Log.WithValues("Workflow", types.NamespacedName{Name: wf.Name, Namespace: wf.Namespace})
	var firstErr error

	// Iterate through the #DW directives looking for deletions
	for _, directive := range wf.Spec.DWDirectives {
		args, err := dwdparse.BuildArgsMap(directive)
		if err != nil {
			return err
		}

		if args["command"] == "delete_persistent" {
			psiName := args["name"]
			log.Info("Teardown PersistentStorageInstance", "name", psiName)

			psi, err := r.findPersistentInstance(ctx, wf, psiName)
			if err != nil {
				if !apierrors.IsNotFound(err) {
					if firstErr == nil {
						firstErr = err
					}
				}

				// Failed to find the PersistentStorageInstance, continue looking for more delete directives
				continue
			}

			err = r.assignWorkflowAsOwner(ctx, psi, wf)
			if err != nil {
				if firstErr == nil {
					firstErr = err
				}
				log.Info("Failed to assign workflow as owner", "name", psiName, "error", err)
			}
		}
	}

	return firstErr
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

func (r *NnfWorkflowReconciler) assignWorkflowAsOwner(ctx context.Context, psi *dwsv1alpha1.PersistentStorageInstance, wf *dwsv1alpha1.Workflow) error {
	log := r.Log.WithValues("Workflow", types.NamespacedName{Name: wf.Name, Namespace: wf.Namespace}, "persistentInstance", types.NamespacedName{Name: psi.Name, Namespace: psi.Namespace})

	dwsv1alpha1.AddOwnerLabels(psi, wf)

	if err := controllerutil.SetControllerReference(wf, psi, r.Scheme); err != nil {
		log.Error(err, "Unable to assign workflow as owner of persistentInstance")
		return err
	}

	return r.Update(ctx, psi)
}

// SetupWithManager sets up the controller with the Manager.
func (r *NnfWorkflowReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.ChildObjects = []dwsv1alpha1.ObjectList{
		&nnfv1alpha1.NnfDataMovementList{},
		&nnfv1alpha1.NnfAccessList{},
		&nnfv1alpha1.NnfStorageList{},
		&dwsv1alpha1.PersistentStorageInstanceList{},
		&dwsv1alpha1.DirectiveBreakdownList{},
		&nnfv1alpha1.NnfStorageProfileList{},
	}

	maxReconciles := runtime.GOMAXPROCS(0)
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(controller.Options{MaxConcurrentReconciles: maxReconciles}).
		For(&dwsv1alpha1.Workflow{}).
		Owns(&nnfv1alpha1.NnfAccess{}).
		Owns(&nnfv1alpha1.NnfDataMovement{}).
		Owns(&dwsv1alpha1.DirectiveBreakdown{}).
		Owns(&dwsv1alpha1.PersistentStorageInstance{}).
		Owns(&nnfv1alpha1.NnfStorageProfile{}).
		Watches(&source.Kind{Type: &nnfv1alpha1.NnfStorage{}}, handler.EnqueueRequestsFromMapFunc(dwsv1alpha1.OwnerLabelMapFunc)).
		Complete(r)
}
