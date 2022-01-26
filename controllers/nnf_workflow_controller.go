/*
Copyright 2021 Hewlett Packard Enterprise Development LP
*/

package controllers

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"runtime"
	"strconv"
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
	nnfv1alpha1 "github.hpe.com/hpe/hpc-rabsw-nnf-sos/api/v1alpha1"
)

const (
	// finalizerNnfWorkflow defines the key used in identifying the
	// storage object as being owned by this NNF Storage Reconciler. This
	// prevents the system from deleting the custom resource until the
	// reconciler has finished in using the resource.
	finalizerNnfWorkflow = "nnf.cray.hpe.com/nnf_workflow"
)

type nnfStoragesState int

// State enumerations
const (
	nnfStoragesExist nnfStoragesState = iota
	nnfStoragesDeleted
)

var nnfStorageStateStrings = [...]string{
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
//+kubebuilder:rbac:groups=dws.cray.hpe.com,resources=directivebreakdowns,verbs=get;create;list;watch;update;patch
//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=directivebreakdowns/status,verbs=get;list;watch
//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfjobstorageinstances,verbs=get;create;list;watch;update;patch;delete
//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfjobstorageinstances/status,verbs=get;create;list;watch;update;patch
//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfjobstorageinstances/finalizers,verbs=get;create;list;watch;update;patch
//+kubebuilder:rbac:groups=dws.cray.hpe.com,resources=servers,verbs=get;create;list;watch;update;patch
//+kubebuilder:rbac:groups=dws.cray.hpe.com,resources=computes,verbs=get;create;list;watch;update;patch
//+kubebuilder:rbac:groups=dws.cray.hpe.com,resources=workflows/finalizers,verbs=update
//+kubebuilder:rbac:groups=dm.cray.hpe.com,resources=datamovements,verbs=get;list;watch

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

func (r *NnfWorkflowReconciler) handleUnsupportedState(ctx context.Context, workflow *dwsv1alpha1.Workflow, driverID string, log logr.Logger) (ctrl.Result, error) {
	log.Info(workflow.Status.State)

	log.Info("Unsupported Status.State", "state", workflow.Status.State)

	// TODO: This should return an error, but for now unhandled states simply succeed to allow Flux to exercise workflow...
	// Complete state in the drivers
	return r.completeDriverState(ctx, workflow, driverID, log)
}

func (r *NnfWorkflowReconciler) handleProposalState(ctx context.Context, workflow *dwsv1alpha1.Workflow, driverID string, log logr.Logger) (ctrl.Result, error) {
	log.Info(workflow.Status.State)

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

		// Wait for all directiveBreakdowns to become ready
		if directiveBreakdown.Status.Ready != ConditionTrue {
			return ctrl.Result{}, nil
		}
	}

	// TODO: PROPOSAL STAGE - Validate that...
	// Extract the job instance or persistent storage instance names
	// For each copy_in directive
	//   Make sure all $JOB_DW_ references point to storage instance names
	//   Make sure all $PERSISTENT_DW references a new or existing create_persistent instance
	//   Make sure source is prefixed with a valid lustre file system

	// Complete state in the drivers
	return r.completeDriverState(ctx, workflow, driverID, log)
}

func (r *NnfWorkflowReconciler) generateDirectiveBreakdown(ctx context.Context, directive string, dwIndex int, wf *dwsv1alpha1.Workflow, log logr.Logger) (result controllerutil.OperationResult, directiveBreakdown *dwsv1alpha1.DirectiveBreakdown, err error) {

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
			dwdName := wf.Name + "-" + strconv.Itoa(dwIndex)
			switch cmdElements[1] {
			case "create_persistent":
				lifetime = "persistent"
			}

			directiveBreakdown = &dwsv1alpha1.DirectiveBreakdown{
				ObjectMeta: metav1.ObjectMeta{
					Name:      dwdName,
					Namespace: wf.Namespace,
				},
			}

			result, err = ctrl.CreateOrUpdate(ctx, r.Client, directiveBreakdown,
				// Mutate function to fill in a directiveBreakdown
				func() error {

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
					return ctrl.SetControllerReference(wf, directiveBreakdown, r.Scheme)
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
func (r *NnfWorkflowReconciler) createStorageInstance(ctx context.Context, wf *dwsv1alpha1.Workflow, d *dwsv1alpha1.DirectiveBreakdown, s *dwsv1alpha1.Servers, log logr.Logger) (ctrl.Result, error) {

	switch d.Spec.Lifetime {
	case "job": // TODO: "job" and "persistent" should move to const definitions
		storageInstance := &nnfv1alpha1.NnfJobStorageInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name:      nnfv1alpha1.NnfJobStorageInstanceMakeName(wf.Spec.JobID, wf.Spec.WLMID, d.Spec.DW.DWDirectiveIndex),
				Namespace: wf.Namespace,
			},
		}

		mutateFn := func() error {
			storageInstance.Spec = nnfv1alpha1.NnfJobStorageInstanceSpec{
				Name:    d.Spec.Name,
				FsType:  d.Spec.Type,
				Servers: d.Status.Servers,
			}

			return ctrl.SetControllerReference(wf, storageInstance, r.Scheme)
		}

		result, err := ctrl.CreateOrUpdate(ctx, r.Client, storageInstance, mutateFn)
		log.V(4).Info("Job Storage Instance", "result", result, "error", err)
		return ctrl.Result{Requeue: result != controllerutil.OperationResultNone}, err

	case "persistent":

	}

	return ctrl.Result{}, nil
}

func (r *NnfWorkflowReconciler) createNnfStorage(ctx context.Context, wf *dwsv1alpha1.Workflow, d *dwsv1alpha1.DirectiveBreakdown, s *dwsv1alpha1.Servers, log logr.Logger) (*nnfv1alpha1.NnfStorage, error) {

	// If the Servers object is empty, no work to do we're finished.
	if len(s.Spec.AllocationSets) == 0 {
		return nil, nil
	}

	nnfStorage := &nnfv1alpha1.NnfStorage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      s.Name,
			Namespace: s.Namespace,
		},
	}

	result, err := ctrl.CreateOrUpdate(ctx, r.Client, nnfStorage,
		func() error {

			// Need to remove all of the AllocationSets in the NnfStorage object before we begin
			nnfStorage.Spec.AllocationSets = nil

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
			return ctrl.SetControllerReference(wf, nnfStorage, r.Scheme)
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

	hasCopyInDirective := false
	copyInDirectivesFinished := true
	for directiveIdx, directive := range workflow.Spec.DWDirectives {
		if strings.HasPrefix(directive, "#DW copy_in") {
			hasCopyInDirective = true
			_, parameters, err := parseDirective(directive)
			if err != nil {
				workflow.Status.Message = fmt.Sprintf("Stage %s failed: %v", workflow.Spec.DesiredState, err)
				// TODO: Return with error
			}

			// NOTE: We don't need to check for the occurrence of a source or destination parameters since these are required fields and validated through the webhook
			// NOTE: We don't need to validate destination has $JOB_DW_ since this is done during the proposal stage

			// Split the source string into a file system name and a companion path
			// i.e. $JOB_DW_my-file-system-name/path/to/a/file into "my-file-system-name" and "/path/to/a/file"
			splitIntoNameAndPath := func(s string) (string, string) {
				var name = ""
				if strings.Contains(s, "$JOB_DW_") {
					name = strings.SplitN(strings.Replace(s, "$JOB_DW_", "", 1), "/", 1)[0]

				} else if strings.Contains(s, "$PERSISTENT_DW_") {
					name = strings.SplitN(strings.Replace(s, "$PERSISTENT_DW_", "", 1), "/", 1)[0]
				}
				var path = "/"
				if strings.Count(s, "/") > 1 {
					path = "/" + strings.SplitN(s, "/", 1)[1]
				}
				return name, path
			}

			findDirectiveIndexByName := func(name string) int {
				for idx, directive := range workflow.Spec.DWDirectives {
					_, parameters, _ := parseDirective(directive)
					if parameters["name"] == name {
						return idx
					}
				}
				return -1
			}

			// copy_in needs to handle four cases
			// 1. Lustre to JobStorageInstance                              #DW copy_in source=[path] destination=$JOB_DW_[name]/[path]
			// 2. Lustre to PersistentStorageInstance                       #DW copy_in source=[path] destination=$PERSISTENT_DW_[name]/[path]
			// 3. PersistentStorageInstance to JobStorageInstance           #DW copy_in source=$PERSISTENT_DW_[name]/[path] destination=$JOB_DW_[name]/[path]
			// 4. PersistentStorageInstance to PersistentStorageInstance    #DW copy_in source=$PERSISTENT_DW_[name]/[path] destination=$PERSISTENT_DW_[name]/[path]

			name := fmt.Sprintf("%d-%d", workflow.Spec.JobID, directiveIdx) // TODO: Should this move to a MakeName()?
			dm := &nnfv1alpha1.NnfDataMovement{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: workflow.Namespace,
				},
			}

			mutateFn := func() error {

				var sourceInstance *corev1.ObjectReference
				sourceName, sourcePath := splitIntoNameAndPath(parameters["source"])
				if strings.Contains(parameters["source"], "$JOB_DW_") {
					sourceInstance = &corev1.ObjectReference{
						Kind:      "JobStorageInstance",
						Name:      nnfv1alpha1.NnfJobStorageInstanceMakeName(workflow.Spec.JobID, workflow.Spec.WLMID, findDirectiveIndexByName(sourceName)),
						Namespace: workflow.Namespace,
					}
				} else if strings.Contains(parameters["source"], "$PERSISTENT_DW_") {
					sourceInstance = &corev1.ObjectReference{
						Kind:      "PersistentStorageInstance",
						Name:      sourceName,
						Namespace: workflow.Namespace,
					}
				}

				var destinationInstance *corev1.ObjectReference
				destinationName, destinationPath := splitIntoNameAndPath(parameters["destination"])
				if strings.Contains(parameters["destination"], "$JOB_DW_") {
					destinationInstance = &corev1.ObjectReference{
						Kind:      "JobStorageInstance",
						Name:      nnfv1alpha1.NnfJobStorageInstanceMakeName(workflow.Spec.JobID, workflow.Spec.WLMID, findDirectiveIndexByName(destinationName)),
						Namespace: workflow.Namespace,
					}
				} else if strings.Contains(parameters["destination"], "$PERSISTENT_DW_") {
					destinationInstance = &corev1.ObjectReference{
						Kind:      "PersistentStorageInstance",
						Name:      destinationName,
						Namespace: workflow.Namespace,
					}
				}

				dm.Spec = nnfv1alpha1.NnfDataMovementSpec{
					Source: nnfv1alpha1.NnfDataMovementSpecSourceDestination{
						Path:            sourcePath,
						StorageInstance: sourceInstance,
					},
					Destination: nnfv1alpha1.NnfDataMovementSpecSourceDestination{
						Path:            destinationPath,
						StorageInstance: destinationInstance,
					},
				}

				return ctrl.SetControllerReference(workflow, dm, r.Scheme)
			}

			result, err := ctrl.CreateOrUpdate(ctx, r.Client, dm, mutateFn)
			if err != nil {
				log.Error(err, "DataMovement CreateOrUpdate failed", "name", name)
				return ctrl.Result{}, err
			}

			log.V(4).Info("DataMovement CreateOrUpdate", "result", result, "error", err)
			switch result {
			case controllerutil.OperationResultCreated, controllerutil.OperationResultUpdated:
				return ctrl.Result{Requeue: true}, nil
			case controllerutil.OperationResultNone:
				// The copy_in directive was already handled; retrieve the data-movement request and check
				// if it has finished
				dm := &nnfv1alpha1.NnfDataMovement{}
				if err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: workflow.Namespace}, dm); err != nil {
					log.Error(err, "Failed to retrieve DataMovement resource", "Name", name)
					return ctrl.Result{}, err
				}

				finished := false
				for _, condition := range dm.Status.Conditions {
					if condition.Type == nnfv1alpha1.DataMovementConditionTypeFinished {
						finished = true
						// TODO: Finished could be in error - check for an error message
					}
				}

				if !finished {
					copyInDirectivesFinished = false
				}
			}

		} // if strings.HasPrefix(directive, "#DW copy_in")
	}

	if !hasCopyInDirective {
		return r.completeDriverState(ctx, workflow, driverID, log)
	}

	if copyInDirectivesFinished {
		return r.completeDriverState(ctx, workflow, driverID, log)
	}

	return ctrl.Result{}, nil
}

func parseDirective(directive string) (command string, parameters map[string]string, err error) {
	if !strings.HasPrefix(directive, "#DW") {
		return "", nil, fmt.Errorf("Malformed directive: No #DW prefix")
	}

	re := regexp.MustCompile("^#DW\\s+(\\w+)\\s+(.*)")
	fields := re.FindAllString(directive, -1)
	if len(fields) == 0 {
		return "", nil, fmt.Errorf("Malformed directive: Failed regexp match")
	}
	if len(fields) == 1 {
		return "", nil, fmt.Errorf("Malformed directive: Missing arguments")
	}

	command = fields[0]

	params := strings.Fields(fields[1])
	for _, param := range params {
		if arg := strings.Split(param, "="); len(arg) == 2 {
			parameters[arg[0]] = arg[1]
		}
	}

	return
}

func (r *NnfWorkflowReconciler) handleTeardownState(ctx context.Context, workflow *dwsv1alpha1.Workflow, driverID string, log logr.Logger) (ctrl.Result, error) {
	log.Info(workflow.Status.State)

	exists, err := r.teardownStorage(ctx, workflow)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Wait for all the nnfStorages to finish deleting before completing the
	// teardown state.
	if exists == nnfStoragesExist {
		return ctrl.Result{}, err
	}

	// Complete state in the drivers
	return r.completeDriverState(ctx, workflow, driverID, log)
}

// Delete all the child NnfStorage resources. Don't trust the client cache
// We may have created children that aren't in the cache
func (r *NnfWorkflowReconciler) teardownStorage(ctx context.Context, wf *dwsv1alpha1.Workflow) (nnfStoragesState, error) {
	log := r.Log.WithValues("Workflow", types.NamespacedName{Name: wf.Name, Namespace: wf.Namespace})
	var firstErr error
	state := nnfStoragesDeleted

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
			state = nnfStoragesExist
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
			state = nnfStoragesExist
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
		Owns(&nnfv1alpha1.NnfJobStorageInstance{}).
		Owns(&dwsv1alpha1.DirectiveBreakdown{}).
		//Owns(&dmv1alpha1.DataMovement{}). Enable once data movement is fully deployable
		Complete(r)
}
