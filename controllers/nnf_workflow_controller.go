/*
Copyright 2021 Hewlett Packard Enterprise Development LP
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

	dwsv1alpha1 "github.hpe.com/hpe/hpc-dpm-dws-operator/api/v1alpha1"
	"github.hpe.com/hpe/hpc-dpm-dws-operator/utils/dwdparse"
	lusv1alpha1 "github.hpe.com/hpe/hpc-rabsw-lustre-fs-operator/api/v1alpha1"
	nnfv1alpha1 "github.hpe.com/hpe/hpc-rabsw-nnf-sos/api/v1alpha1"
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
	resourceExist nnfResourceState = iota
	resourceDeleted
)

var nnfResourceStateStrings = [...]string{
	"exist",
	"deleted",
}

// NnfWorkflowReconciler contains the pieces used by the reconciler
type NnfWorkflowReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *kruntime.Scheme
}

//+kubebuilder:rbac:groups=dws.cray.hpe.com,resources=workflows,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups=dws.cray.hpe.com,resources=workflows/finalizers,verbs=update
//+kubebuilder:rbac:groups=dws.cray.hpe.com,resources=directivebreakdowns,verbs=get;create;list;watch;update;patch
//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=directivebreakdowns/status,verbs=get;list;watch
//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfaccesses,verbs=get;create;list;watch;update;patch;delete
//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfdatamovements,verbs=get;create;list;watch;update;patch;delete
//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfdatamovementworkflows,verbs=get;create;list;watch;update;patch;delete
//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfjobstorageinstances,verbs=get;create;list;watch;update;patch;delete
//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfpersistentstorageinstances,verbs=get;create;list;watch;update;patch
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

	// Check if the object is being deleted
	if !workflow.GetDeletionTimestamp().IsZero() {
		log.Info("Deleting workflow...")

		if !controllerutil.ContainsFinalizer(workflow, finalizerNnfWorkflow) {
			return ctrl.Result{}, nil
		}

		// TODO: Teardown workflow
		/*
			if err := r.teardownWorkflow(ctx, workflow); err != nil {
				return ctrl.Result{}, err
			}

			controllerutil.RemoveFinalizer(workflow, finalizerNnfWorkflow)
			if err := r.Update(ctx, workflow); err != nil {
				return ctrl.Result{}, err
			}
		*/

		return ctrl.Result{}, nil
	}

	driverID := os.Getenv("DWS_DRIVER_ID")

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
		return r.failDriverState(ctx, workflow, driverID, err.Error())
	}

	var dbList []v1.ObjectReference
	for dwIndex, dwDirective := range workflow.Spec.DWDirectives {
		_, directiveBreakdown, err := r.generateDirectiveBreakdown(ctx, dwDirective, dwIndex, workflow, log)
		if err != nil {
			return ctrl.Result{}, err
		}
		if directiveBreakdown != nil {
			dbList = append(dbList, v1.ObjectReference{Name: directiveBreakdown.Name, Namespace: directiveBreakdown.Namespace})
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

// Validate the workflow and return any error found
func (r *NnfWorkflowReconciler) validateWorkflow(ctx context.Context, wf *dwsv1alpha1.Workflow) error {

	for _, directive := range wf.Spec.DWDirectives {
		if strings.HasPrefix(directive, "#DW copy_in") || strings.HasPrefix(directive, "#DW copy_out") {
			if err := r.validateStagingDirective(ctx, wf, directive); err != nil {
				return err
			}
		}
	}

	return nil
}

// Validate staging directives copy_in/copy_out directives
func (r *NnfWorkflowReconciler) validateStagingDirective(ctx context.Context, wf *dwsv1alpha1.Workflow, directive string) error {
	// Validate staging directive of the form...
	//   #DW copy_in source=[SOURCE] destination=[DESTINATION]
	//   #DW copy_out source=[SOURCE] destination=[DESTINATION]

	// For each copy_in/copy_out directive
	//   Make sure all $JOB_DW_ references point to job storage instance names
	//   Make sure all $PERSISTENT_DW references a new or existing create_persistent instance
	//   Otherwise, make sure source/destination is prefixed with a valid global lustre file system
	validateStagingArgument := func(arg string) error {
		name, _ := splitStagingArgumentIntoNameAndPath(arg)
		if strings.HasPrefix(arg, "$JOB_DW_") {
			if findDirectiveIndexByName(wf, name) == -1 {
				return fmt.Errorf("Job storage instance '%s' not found", name)
			}
		} else if strings.HasPrefix(arg, "$PERSISTENT_DW_") {
			if findDirectiveIndexByName(wf, name) == -1 {
				return fmt.Errorf("Persistent storage instance '%s' not found", name)
				// TODO: This may be defined by another job; we need to query the list of NNF Persistent Storage Instances
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

func (r *NnfWorkflowReconciler) generateDirectiveBreakdown(ctx context.Context, directive string, dwIndex int, workflow *dwsv1alpha1.Workflow, log logr.Logger) (result controllerutil.OperationResult, directiveBreakdown *dwsv1alpha1.DirectiveBreakdown, err error) {

	// DWDirectives that we need to generate directiveBreakdowns for look like this:
	//  #DW command            arguments...
	//  --- -----------------  -------------------------------------------
	// "#DW jobdw              type=raw    capacity=9TB name=thisIsReallyRaw"
	// "#DW jobdw              type=xfs    capacity=9TB name=thisIsXFS"
	// "#DW jobdw              type=lustre capacity=9TB name=thisIsLustre"
	// "#DW create_persistent  type=raw    capacity=9TB name=thisIsPersistent"

	// #DW commands that require dwDirectiveBreakdowns because they create storage
	breakDownCommands := []string{
		"jobdw",
		"create_persistent",
	}

	cmdElements := strings.Fields(directive)
	directiveBreakdown = nil
	result = controllerutil.OperationResultNone

	for _, breakThisDown := range breakDownCommands {
		// We care about the commands that generate a breakdown
		if breakThisDown == cmdElements[1] {

			lifetime := "job"
			dwdName := fmt.Sprintf("%s-%d", workflow.Name, dwIndex)
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

					directiveBreakdown.Labels = map[string]string{
						dwsv1alpha1.WorkflowNameLabel:      workflow.Name,
						dwsv1alpha1.WorkflowNamespaceLabel: workflow.Namespace,
					}

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
				log.Info("Created breakdown", "directiveBreakdown", directiveBreakdown.Spec.DW.DWDirective)
			} else if result == controllerutil.OperationResultNone {
				// no change
			} else {
				log.Info("Updated breakdown", "directiveBreakdown", directiveBreakdown.Spec.DW.DWDirective)
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

	// Iterate through the directive breakdowns...
	for _, dbdRef := range workflow.Status.DirectiveBreakdowns {

		// Chain through the DirectiveBreakdown to the Servers object
		dbd := &dwsv1alpha1.DirectiveBreakdown{}
		err := r.Get(ctx, types.NamespacedName{Name: dbdRef.Name, Namespace: dbdRef.Namespace}, dbd)
		if err != nil {
			log.Error(err, "Unable to get directiveBreakdown", "dbd", types.NamespacedName{Name: dbdRef.Name, Namespace: dbdRef.Namespace})
			return ctrl.Result{}, err
		}

		s := &dwsv1alpha1.Servers{}
		err = r.Get(ctx, types.NamespacedName{Name: dbd.Status.Servers.Name, Namespace: dbd.Status.Servers.Namespace}, s)
		if err != nil {
			log.Error(err, "Unable to get servers", "servers", types.NamespacedName{Name: dbd.Status.Servers.Name, Namespace: dbd.Status.Servers.Namespace})
			return ctrl.Result{}, err
		}

		if _, present := os.LookupEnv("RABBIT_TEST_ENV_BYPASS_SERVER_STORAGE_CHECK"); !present {
			if len(dbd.Status.AllocationSet) != 0 && len(dbd.Status.AllocationSet) != len(s.Spec.AllocationSets) {
				return r.failDriverState(ctx, workflow, driverID, fmt.Sprintf("Servers resource does not meet storage requirements for directive '%s'", dbd.Spec.DW.DWDirective))
			}
		}

		log.Info("Creating storage instance", "Directive", dbd.Spec.DW)
		result, err := r.createStorageInstance(ctx, workflow, dbd, s, log)
		if err != nil {
			log.Error(err, "Failed to create nnf storage instance")
			return ctrl.Result{}, err
		} else if result.Requeue {
			return result, nil
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

// Create the storage instance for the provided directive breakdown. A Storage Instance represents the storage for a
// particular breakdown and has either a job or persistent lifetime. Storage Instances are provided to data movement
// API so it is aware of the file system type and other attributes used in data movement.
func (r *NnfWorkflowReconciler) createStorageInstance(ctx context.Context, workflow *dwsv1alpha1.Workflow, d *dwsv1alpha1.DirectiveBreakdown, s *dwsv1alpha1.Servers, log logr.Logger) (ctrl.Result, error) {

	switch d.Spec.Lifetime {
	case "job": // TODO: "job" and "persistent" should move to const definitions
		storageInstance := &nnfv1alpha1.NnfJobStorageInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-%d", workflow.Name, d.Spec.DW.DWDirectiveIndex),
				Namespace: workflow.Namespace,
			},
		}

		mutateFn := func() error {
			storageInstance.Labels = map[string]string{
				dwsv1alpha1.WorkflowNameLabel:      workflow.Name,
				dwsv1alpha1.WorkflowNamespaceLabel: workflow.Namespace,
			}

			storageInstance.Spec = nnfv1alpha1.NnfJobStorageInstanceSpec{
				Name:    d.Spec.Name,
				FsType:  d.Spec.Type,
				Servers: d.Status.Servers,
			}

			return ctrl.SetControllerReference(workflow, storageInstance, r.Scheme)
		}

		result, err := ctrl.CreateOrUpdate(ctx, r.Client, storageInstance, mutateFn)
		return ctrl.Result{Requeue: result != controllerutil.OperationResultNone}, err

	case "persistent":
		storageInstance := &nnfv1alpha1.NnfPersistentStorageInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name:      d.Spec.Name,
				Namespace: "nnf-system",
			},
		}

		mutateFn := func() error {
			storageInstance.Spec = nnfv1alpha1.NnfPersistentStorageInstanceSpec{
				Name:    d.Spec.Name,
				FsType:  d.Spec.Type,
				Servers: d.Status.Servers,
			}

			return nil
		}

		_, err := ctrl.CreateOrUpdate(ctx, r.Client, storageInstance, mutateFn)
		return ctrl.Result{}, err

	default:
		return ctrl.Result{}, fmt.Errorf("unsupported directive lifetime '%s'", d.Spec.Lifetime)
	}
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
		return nnfStorage, err
	}

	result, err := ctrl.CreateOrUpdate(ctx, r.Client, nnfStorage,
		func() error {

			nnfStorage.Labels = map[string]string{
				dwsv1alpha1.WorkflowNameLabel:      workflow.Name,
				dwsv1alpha1.WorkflowNamespaceLabel: workflow.Namespace,
			}

			// Need to remove all of the AllocationSets in the NnfStorage object before we begin
			nnfStorage.Spec.AllocationSets = []nnfv1alpha1.NnfStorageAllocationSetSpec{}

			// Iterate the Servers data elements to pull out the allocation sets for the server
			for i := range s.Spec.AllocationSets {
				nnfAllocSet := nnfv1alpha1.NnfStorageAllocationSetSpec{}

				nnfAllocSet.Name = s.Spec.AllocationSets[i].Label
				nnfAllocSet.FileSystemType = d.Spec.Type // DirectiveBreakdown contains the filesystem type (lustre, xfs, etc)
				nnfAllocSet.Capacity = s.Spec.AllocationSets[i].AllocationSize
				if nnfAllocSet.FileSystemType == "lustre" {
					nnfAllocSet.NnfStorageLustreSpec.TargetType = strings.ToUpper(s.Spec.AllocationSets[i].Label)
					nnfAllocSet.NnfStorageLustreSpec.BackFs = "zfs"
					charsWanted := 8
					if len(d.Spec.Name) < charsWanted {
						charsWanted = len(d.Spec.Name)
					}
					nnfAllocSet.NnfStorageLustreSpec.FileSystemName = d.Spec.Name[:charsWanted]
					if mgsNid, present := dwArgs["external_mgs"]; present {
						nnfAllocSet.NnfStorageLustreSpec.ExternalMgsNid = mgsNid
					}
				}

				// Create Nodes for this allocation set.
				for _, storage := range s.Spec.AllocationSets[i].Storage {
					node := nnfv1alpha1.NnfStorageAllocationNodes{Name: storage.Name, Count: storage.AllocationCount}
					nnfAllocSet.Nodes = append(nnfAllocSet.Nodes, node)
				}

				nnfStorage.Spec.AllocationSets = append(nnfStorage.Spec.AllocationSets, nnfAllocSet)
			}

			// The Workflow object owns the NnfStorage object, so updates to the nnfStorage trigger a watch
			// in the workflow's reconciler
			return ctrl.SetControllerReference(workflow, nnfStorage, r.Scheme)
		})

	if err != nil {
		log.Error(err, "Failed to create or update NnfStorage", "name", nnfStorage.Name)
		return nnfStorage, err
	}

	if result == controllerutil.OperationResultCreated {
		log.Info("Created nnfStorage", "name", nnfStorage.Name)
	} else if result == controllerutil.OperationResultNone {
		// no change
	} else {
		log.Info("Updated nnfStorage", "name", nnfStorage.Name)
	}

	return nnfStorage, nil
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
		if strings.HasPrefix(directive, "#DW copy_in") || strings.HasPrefix(directive, "#DW copy_out") {
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
					result, err := r.teardownDataMovementResource(ctx, dm)
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
	// copy_out is the same, but typically source and destination reversed

	// Prepare the provided staging parameter for data-movement. Param is the source/destination value from the #DW copy_in/copy_out directive; based
	// on the param prefix we determine the storage instance and access requirements for data movement.
	prepareStagingArgumentFn := func(param string) (*corev1.ObjectReference, *nnfv1alpha1.NnfAccess, string, *ctrl.Result, error) {
		var storage *corev1.ObjectReference
		var access *nnfv1alpha1.NnfAccess

		name, path := splitStagingArgumentIntoNameAndPath(param)

		// Job or Persistent DWs will have a corresponding NNF Job Storage Instance or NNF Persistent Storage Instance, respectively. Both
		// require we setup an NNF Access for the data-movement resource to ensure the data is accessible on the NNF Server while data movement
		// executes.
		if strings.HasPrefix(param, "$JOB_DW_") || strings.HasPrefix(param, "$PERSISTENT_DW_") {

			servers := &corev1.ObjectReference{}
			if strings.HasPrefix(param, "$JOB_DW_") {

				storageInstance, err := r.findNnfJobStorageInstance(ctx, name, workflow)
				if err != nil {
					return nil, nil, path, nil, err
				}

				servers = &storageInstance.Spec.Servers

				storage = &corev1.ObjectReference{
					Kind:      reflect.TypeOf(nnfv1alpha1.NnfJobStorageInstance{}).Name(),
					Name:      storageInstance.Name,
					Namespace: storageInstance.Namespace,
				}
			} else if strings.HasPrefix(param, "$PERSISTENT_DW_") {

				storageInstance, err := r.findNnfPersistentStorageInstance(ctx, name)
				if err != nil {
					return nil, nil, path, nil, err
				}

				servers = &storageInstance.Spec.Servers

				storage = &corev1.ObjectReference{
					Kind:      reflect.TypeOf(nnfv1alpha1.NnfPersistentStorageInstance{}).Name(),
					Name:      storageInstance.Name,
					Namespace: storageInstance.Namespace,
				}
			} else {
				return nil, nil, path, nil, fmt.Errorf("Unsupported parameter '%s'", param)
			}

			var result controllerutil.OperationResult
			access, result, err = r.setupNnfAccessForServers(ctx, servers, workflow, name, dwIndex)
			if err != nil {
				return storage, access, path, nil, err
			} else if result != controllerutil.OperationResultNone {
				return storage, access, path, &ctrl.Result{Requeue: true}, nil
			}

		} else if lustre := r.findLustreFileSystemForPath(ctx, param, r.Log); lustre != nil {
			storage = &corev1.ObjectReference{
				Kind:      reflect.TypeOf(lusv1alpha1.LustreFileSystem{}).Name(),
				Name:      lustre.Name,
				Namespace: lustre.Namespace,
			}
		} else {
			return nil, nil, "", nil, fmt.Errorf("Parameter '%s' invalid", param)
		}

		return storage, access, path, nil, nil
	}

	sourceInstance, sourceAccess, sourcePath, result, err := prepareStagingArgumentFn(parameters["source"])
	if err != nil {
		return ctrl.Result{}, err
	} else if result != nil {
		return *result, nil
	}

	destinationInstance, destinationAccess, destinationPath, result, err := prepareStagingArgumentFn(parameters["destination"])
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

	dm.Labels = map[string]string{
		dwsv1alpha1.WorkflowNameLabel:      workflow.Name,
		dwsv1alpha1.WorkflowNamespaceLabel: workflow.Namespace,
	}

	dm.Spec = nnfv1alpha1.NnfDataMovementSpec{
		Source: nnfv1alpha1.NnfDataMovementSpecSourceDestination{
			Path:            sourcePath,
			StorageInstance: sourceInstance,
			Access:          accessToObjectReference(sourceAccess),
		},
		Destination: nnfv1alpha1.NnfDataMovementSpecSourceDestination{
			Path:            destinationPath,
			StorageInstance: destinationInstance,
			Access:          accessToObjectReference(destinationAccess),
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

func (r *NnfWorkflowReconciler) findNnfJobStorageInstance(ctx context.Context, name string, workflow *dwsv1alpha1.Workflow) (*nnfv1alpha1.NnfJobStorageInstance, error) {
	index := findDirectiveIndexByName(workflow, name)
	jsi := &nnfv1alpha1.NnfJobStorageInstance{}
	return jsi, r.Get(ctx, types.NamespacedName{Name: fmt.Sprintf("%s-%d", workflow.Name, index), Namespace: workflow.Namespace}, jsi)
}

func (r *NnfWorkflowReconciler) findNnfPersistentStorageInstance(ctx context.Context, name string) (*nnfv1alpha1.NnfPersistentStorageInstance, error) {
	psi := &nnfv1alpha1.NnfPersistentStorageInstance{}
	return psi, r.Get(ctx, types.NamespacedName{Name: name, Namespace: "nnf-system"}, psi)
}

func (r *NnfWorkflowReconciler) setupNnfAccessForServers(ctx context.Context, serversReference *corev1.ObjectReference, workflow *dwsv1alpha1.Workflow, name string, directiveIndex int) (*nnfv1alpha1.NnfAccess, controllerutil.OperationResult, error) {

	// NNF Storage is Namespaced Name to the servers object, make sure we can find it

	servers := &dwsv1alpha1.Servers{}
	if err := r.Get(ctx, types.NamespacedName{Name: serversReference.Name, Namespace: serversReference.Namespace}, servers); err != nil {
		return nil, controllerutil.OperationResultNone, err
	}

	storage := &nnfv1alpha1.NnfStorage{}
	if err := r.Get(ctx, client.ObjectKeyFromObject(servers), storage); err != nil {
		return nil, controllerutil.OperationResultNone, err
	}

	access := &nnfv1alpha1.NnfAccess{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%d", workflow.Name, directiveIndex),
			Namespace: workflow.Namespace,
		},
	}

	result, err := ctrl.CreateOrUpdate(ctx, r.Client, access,
		func() error {
			access.Labels = map[string]string{
				dwsv1alpha1.WorkflowNameLabel:      workflow.Name,
				dwsv1alpha1.WorkflowNamespaceLabel: workflow.Namespace,
			}

			access.Spec = nnfv1alpha1.NnfAccessSpec{
				DesiredState:    "mounted",
				Target:          "all",
				MountPathPrefix: fmt.Sprintf("/mnt/nnf/%d/job/%s", workflow.Spec.JobID, name),
				StorageReference: corev1.ObjectReference{
					Kind:      reflect.TypeOf(nnfv1alpha1.NnfStorage{}).Name(),
					Name:      storage.Name,
					Namespace: storage.Namespace,
				},
			}

			return ctrl.SetControllerReference(workflow, access, r.Scheme)
		})

	if err != nil || result != controllerutil.OperationResultNone {
		return nil, result, err
	}

	if err := r.Get(ctx, client.ObjectKeyFromObject(access), access); err != nil {
		return nil, result, err
	}

	return access, controllerutil.OperationResultNone, nil
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
func (r *NnfWorkflowReconciler) teardownDataMovementResource(ctx context.Context, dm *nnfv1alpha1.NnfDataMovement) (*ctrl.Result, error) {
	unmountAccess := func(ref *corev1.ObjectReference) (*ctrl.Result, error) {
		if ref == nil {
			return nil, nil
		}

		access := &nnfv1alpha1.NnfAccess{}
		if err := r.Get(ctx, types.NamespacedName{Name: ref.Name, Namespace: ref.Namespace}, access); err != nil {
			return nil, err
		}

		if access.Spec.DesiredState != "unmounted" {
			access.Spec.DesiredState = "unmounted"
			if err := r.Update(ctx, access); err != nil {
				if apierrors.IsConflict(err) {
					return &ctrl.Result{Requeue: true}, nil
				}
				return nil, err
			}
		}

		return nil, nil
	}

	if result, err := unmountAccess(dm.Spec.Source.Access); !result.IsZero() || err != nil {
		return result, err
	}

	if result, err := unmountAccess(dm.Spec.Destination.Access); !result.IsZero() || err != nil {
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

		// Create a companion NNF Data Movement Workflow resource that manages concurrent data movement
		// that is started during the job by the compute resources. It is expected that the customer
		// will clean-up data movement resources that it created, but this is implemented to ensure no
		// resources are left behind by the job.
		{
			dmw := &nnfv1alpha1.NnfDataMovementWorkflow{
				ObjectMeta: metav1.ObjectMeta{
					Name:      workflow.Name,
					Namespace: workflow.Namespace,
				},
			}

			result, err := ctrl.CreateOrUpdate(ctx, r.Client, dmw, func() error {
				return ctrl.SetControllerReference(workflow, dmw, r.Scheme)
			})
			if err != nil {
				return ctrl.Result{}, err
			}

			if result == controllerutil.OperationResultCreated {
				log.Info("Created NnfDatamovementWorkflow", "name", dmw.Name)
			}
		}

		// Parse the #DW line
		args, err := dwdparse.BuildArgsMap(workflow.Spec.DWDirectives[driverStatus.DWDIndex])
		if err != nil {
			return ctrl.Result{}, err
		}

		access := &nnfv1alpha1.NnfAccess{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-%d", workflow.Name, driverStatus.DWDIndex),
				Namespace: workflow.Namespace,
			},
		}

		// Create an NNFAccess for the compute clients
		mountPath := buildMountPath(workflow, args["name"], args["command"])
		result, err := ctrl.CreateOrUpdate(ctx, r.Client, access,
			func() error {
				access.Labels = map[string]string{
					dwsv1alpha1.WorkflowNameLabel:      workflow.Name,
					dwsv1alpha1.WorkflowNamespaceLabel: workflow.Namespace,
				}

				access.Spec.DesiredState = "mounted"
				access.Spec.Target = "single"
				access.Spec.MountPath = mountPath
				access.Spec.ClientReference = corev1.ObjectReference{
					Name:      workflow.Name,
					Namespace: workflow.Namespace,
					Kind:      "Computes",
				}
				access.Spec.StorageReference = corev1.ObjectReference{
					Name:      access.Name,
					Namespace: access.Namespace,
					Kind:      "NnfStorage",
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
			continue
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
		case "create_persistent":
			envName = "DW_PERSISTENT_" + args["name"]
		default:
			return ctrl.Result{}, fmt.Errorf("Unexpected #DW command %s", args["command"])
		}
		workflow.Status.Env[envName] = mountPath
	}

	if complete == false {
		return ctrl.Result{}, nil
	}

	r.completeDriverState(ctx, workflow, driverID, log)

	return ctrl.Result{}, nil
}

func buildMountPath(workflow *dwsv1alpha1.Workflow, name string, command string) string {
	switch command {
	case "jobdw":
		return fmt.Sprintf("/mnt/nnf/%d/job/%s", workflow.Spec.JobID, name)
	case "create_persistent":
		return fmt.Sprintf("/mnt/nnf/%d/persistent/%s", workflow.Spec.JobID, name)
	default:
	}
	return ""
}

func (r *NnfWorkflowReconciler) deleteNnfAccesses(ctx context.Context, workflow *dwsv1alpha1.Workflow, driverID string, log logr.Logger) (nnfResourceState, error) {
	exists := resourceDeleted

	for i := range workflow.Status.Drivers {
		driverStatus := &workflow.Status.Drivers[i]

		// Check if the driver status is for the post_run state. NnfAccess deletion
		// may happen in teardown state instead, but a post run driver status indicates
		// that an NnfAccess was created.
		if driverStatus.WatchState != dwsv1alpha1.StatePostRun.String() {
			continue
		}

		// Only look for driver status for the NNF driver
		if driverStatus.DriverID != driverID {
			continue
		}

		access := &nnfv1alpha1.NnfAccess{}
		namespacedName := types.NamespacedName{
			Name:      fmt.Sprintf("%s-%d", workflow.Name, driverStatus.DWDIndex),
			Namespace: workflow.Namespace,
		}

		err := r.Get(ctx, namespacedName, access)
		if err != nil {
			if !apierrors.IsNotFound(err) {
				return exists, err
			}
		} else {
			exists = resourceExist
			if !access.GetDeletionTimestamp().IsZero() {
				continue
			}
		}

		access.Name = namespacedName.Name
		access.Namespace = namespacedName.Namespace

		err = r.Delete(ctx, access)
		if err != nil {
			if !apierrors.IsNotFound(err) {
				return exists, err
			}
		} else {
			exists = resourceExist
		}
	}

	return exists, nil
}

func (r *NnfWorkflowReconciler) handlePostRunState(ctx context.Context, workflow *dwsv1alpha1.Workflow, driverID string, log logr.Logger) (ctrl.Result, error) {
	log.Info(workflow.Status.State)
	retry := false

	// Set the desired state of NnfAccess to "unmounted" before deleting. This keeps
	// the ClientMount resources around until post run is completed. That way the WLM can
	// check which clients were successfully unmounted if some of the clients have errors
	for i := range workflow.Status.Drivers {
		driverStatus := &workflow.Status.Drivers[i]

		// Check if the driver status is for the post_run state
		if driverStatus.WatchState != dwsv1alpha1.StatePostRun.String() {
			continue
		}

		// Only look for driver status for the NNF driver
		if driverStatus.DriverID != driverID {
			continue
		}

		// Delete the companion NNF Data Movement Workflow resource that manages concurrent data movement
		// executed during the jobs run state by compute resources. It is expected that the customer
		// will clean-up data movement resources that it created, but this is implemented to ensure no
		// resource are left behind by the job.
		{
			dmw := &nnfv1alpha1.NnfDataMovementWorkflow{
				ObjectMeta: metav1.ObjectMeta{
					Name:      workflow.Name,
					Namespace: workflow.Namespace,
				},
			}

			if err := r.Delete(ctx, dmw); err != nil {
				if !apierrors.IsNotFound(err) {
					return ctrl.Result{}, err
				}
			}
		}

		access := &nnfv1alpha1.NnfAccess{}
		namespacedName := types.NamespacedName{
			Name:      fmt.Sprintf("%s-%d", workflow.Name, driverStatus.DWDIndex),
			Namespace: workflow.Namespace,
		}

		err := r.Get(ctx, namespacedName, access)
		if err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			return ctrl.Result{}, err

		}

		if access.Status.State == "unmounted" {
			if access.Status.Ready == false {

				retry = true
			}
			continue
		}

		access.Spec.DesiredState = "unmounted"
		err = r.Update(ctx, access)
		if err != nil {
			return ctrl.Result{}, err
		}

		retry = true
	}

	if retry == true {
		return ctrl.Result{}, nil
	}

	// Delete the NnfAccess before completing post run
	exists, err := r.deleteNnfAccesses(ctx, workflow, driverID, log)
	if err != nil {
		return ctrl.Result{}, err
	}

	if exists == resourceExist {
		return ctrl.Result{}, nil
	}

	r.completeDriverState(ctx, workflow, driverID, log)

	return ctrl.Result{}, nil
}

func (r *NnfWorkflowReconciler) handleTeardownState(ctx context.Context, workflow *dwsv1alpha1.Workflow, driverID string, log logr.Logger) (ctrl.Result, error) {
	log.Info(workflow.Status.State)

	exists, err := r.deleteNnfAccesses(ctx, workflow, driverID, log)
	if err != nil {
		return ctrl.Result{}, err
	}

	if exists == resourceExist {
		return ctrl.Result{}, nil
	}

	exists, err = r.teardownStorage(ctx, workflow)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Wait for all the nnfStorages to finish deleting before completing the
	// teardown state.
	if exists == resourceExist {
		return ctrl.Result{}, err
	}

	// Complete state in the drivers
	return r.completeDriverState(ctx, workflow, driverID, log)
}

// Delete all the child NnfStorage resources. Don't trust the client cache
// We may have created children that aren't in the cache
func (r *NnfWorkflowReconciler) teardownStorage(ctx context.Context, wf *dwsv1alpha1.Workflow) (nnfResourceState, error) {
	log := r.Log.WithValues("Workflow", types.NamespacedName{Name: wf.Name, Namespace: wf.Namespace})
	var firstErr error
	state := resourceDeleted

	// Iterate through the directive breakdowns to determine the names for the NnfStorage
	// objects we need to delete.
	// There will be 1 NnfStorage object
	for _, dbdRef := range wf.Status.DirectiveBreakdowns {
		nameSpacedName := types.NamespacedName{Name: dbdRef.Name, Namespace: dbdRef.Namespace}

		// Do a Get on the nnfStorage first to check if it's already been marked
		// for deletion.
		nnfStorage := &nnfv1alpha1.NnfStorage{}
		err := r.Get(ctx, nameSpacedName, nnfStorage)
		if err != nil {
			if !apierrors.IsNotFound(err) {
				if firstErr == nil {
					firstErr = err
				}
				log.Info("Unable to get NnfStorage", "Error", err, "NnfStorage", nameSpacedName)
			}
		} else {
			state = resourceExist
			if !nnfStorage.GetDeletionTimestamp().IsZero() {
				continue
			}
		}

		nnfStorage = &nnfv1alpha1.NnfStorage{
			ObjectMeta: metav1.ObjectMeta{
				Name:      nameSpacedName.Name,
				Namespace: nameSpacedName.Namespace,
			},
		}

		err = r.Delete(ctx, nnfStorage)
		if err != nil {
			if !apierrors.IsNotFound(err) {
				if firstErr == nil {
					firstErr = err
				}
				log.Info("Unable to delete NnfStorage", "Error", err, "NnfStorage", nnfStorage)
			}
		} else {
			state = resourceExist
		}
	}

	return state, firstErr
}

// SetupWithManager sets up the controller with the Manager.
func (r *NnfWorkflowReconciler) SetupWithManager(mgr ctrl.Manager) error {
	maxReconciles := runtime.GOMAXPROCS(0)
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(controller.Options{MaxConcurrentReconciles: maxReconciles}).
		For(&dwsv1alpha1.Workflow{}).
		Owns(&nnfv1alpha1.NnfStorage{}).
		Owns(&nnfv1alpha1.NnfAccess{}).
		Owns(&nnfv1alpha1.NnfDataMovement{}).
		Owns(&nnfv1alpha1.NnfJobStorageInstance{}).
		Owns(&dwsv1alpha1.DirectiveBreakdown{}).
		Complete(r)
}
