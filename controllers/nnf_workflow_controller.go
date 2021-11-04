/*
Copyright 2021 Hewlett Packard Enterprise Development LP
*/

package controllers

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	nnfv1alpha1 "stash.us.cray.com/RABSW/nnf-sos/api/v1alpha1"
	dwsv1alpha1 "stash.us.cray.com/dpm/dws-operator/api/v1alpha1"
)

const (
	// finalizerNnfWorkflow defines the key used in identifying the
	// storage object as being owned by this NNF Storage Reconciler. This
	// prevents the system from deleting the custom resource until the
	// reconciler has finished in using the resource.
	finalizerNnfWorkflow = "nnf.cray.hpe.com/nnf_workflow"
)

// NnfWorkflowReconciler contains the pieces used by the reconciler
type NnfWorkflowReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=dws.cray.hpe.com,resources=workflows,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups=dws.cray.hpe.com,resources=directivebreakdowns,verbs=get;create;list;watch;update;patch
//+kubebuilder:rbac:groups=nnf.cray.com,resources=directivebreakdowns/status,verbs=get;list;watch
//+kubebuilder:rbac:groups=dws.cray.hpe.com,resources=servers,verbs=get;create;list;watch;update;patch
//+kubebuilder:rbac:groups=dws.cray.hpe.com,resources=computes,verbs=get;create;list;watch;update;patch
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

	// Handle the state work required to achieve the desired State.
	// The model right now is that we continue to call the state handler until it completes the state
	// in which case the reconciler will no longer be called.
	// TODO: Look into have each handler return a "ImNotFinished" to cause a ctrl.Result{Request: repeat}
	//       until the state is finished.
	//       Once the state is finished, we can update the drivers in 1 spot. Right now, the drivers are updated
	//       at the end of each state handler.
	switch workflow.Spec.DesiredState {

	case dwsv1alpha1.StateProposal.String():
		return r.handleProposalState(ctx, workflow, driverID, log)
	case dwsv1alpha1.StateSetup.String():
		return r.handleSetupState(ctx, workflow, driverID, log)
	default:
		err = fmt.Errorf("desiredState %s not yet supported", workflow.Spec.DesiredState)
		log.Error(err, "Unsupported desiredState")
		return ctrl.Result{}, err
	}
}

func (r *NnfWorkflowReconciler) updateDriversStatusForStatusState(workflow *dwsv1alpha1.Workflow, driverID string, log logr.Logger) bool {

	log.Info("Drivers complete", "state", workflow.Status.State)

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
		err := r.Update(ctx, workflow)
		if err != nil {
			log.Error(err, "Failed to update Workflow state")
		}
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *NnfWorkflowReconciler) handleProposalState(ctx context.Context, workflow *dwsv1alpha1.Workflow, driverID string, log logr.Logger) (ctrl.Result, error) {

	log.Info("Proposal")

	var dbList []v1.ObjectReference
	for dwIndex, dwDirective := range workflow.Spec.DWDirectives {
		log.Info("DirectiveBreakdown", "dwDirective", dwDirective)

		_, directiveBreakdown, err := r.generateDirectiveBreakdown(ctx, dwDirective, dwIndex, workflow, log)
		if err != nil {
			return ctrl.Result{}, err
		}

		// Add the directiveBreakdown to the list
		dbList = append(dbList, v1.ObjectReference{Name: directiveBreakdown.Name, Namespace: directiveBreakdown.Namespace})
	}

	// If the workflow already has the correct list of directiveBreakdown references, keep going.
	if !reflect.DeepEqual(workflow.Status.DirectiveBreakdowns, dbList) {
		log.Info("Updating directiveBreakdown references")
		workflow.Status.DirectiveBreakdowns = dbList

		err := r.Update(ctx, workflow)
		if err != nil {
			// Ignore conflict errors
			if apierrors.IsConflict(err) {
				return ctrl.Result{Requeue: true}, nil
			}
			log.Error(err, "Failed to update Workflow DirectiveBreakdowns")
			return ctrl.Result{}, err
		}

		return ctrl.Result{Requeue: true}, nil
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
			return ctrl.Result{Requeue: true}, nil
		}
	}

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

	log.Info("Setup")

	// We don't start looking for NnfStorage to be "Ready" until we've created all of them.
	// In the following loop if we create anything, we requeue before we look at any
	// NnfStorage readiness.
	var nnfStorages []nnfv1alpha1.NnfStorage

	// Iterate through the directive breakdowns...
	for _, dbdRef := range workflow.Status.DirectiveBreakdowns {
		log.Info("DirectiveBreakdown", "name", dbdRef.Name)

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

		result, nnfStorage, err := r.createNnfStorage(ctx, workflow, dbd, s, log)
		if err != nil {
			return ctrl.Result{}, err
		}

		if result == controllerutil.OperationResultCreated {
			log.Info("Created nnfStorage", "name", nnfStorage.Name)
		} else if result == controllerutil.OperationResultNone {
			// no change
		} else {
			log.Info("Updated nnfStorage", "name", nnfStorage.Name)
		}

		nnfStorages = append(nnfStorages, nnfStorage)
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

func (r *NnfWorkflowReconciler) createNnfStorage(ctx context.Context, wf *dwsv1alpha1.Workflow, d *dwsv1alpha1.DirectiveBreakdown, s *dwsv1alpha1.Servers, log logr.Logger) (controllerutil.OperationResult, nnfv1alpha1.NnfStorage, error) {

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
		return result, *nnfStorage, err
	}

	if result == controllerutil.OperationResultCreated {
		log.Info("Created nnfStorage", "name", nnfStorage.Name)
	} else if result == controllerutil.OperationResultNone {
		// no change
	} else {
		log.Info("Updated nnfStorage", "name", nnfStorage.Name)
	}

	return result, *nnfStorage, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NnfWorkflowReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&dwsv1alpha1.Workflow{}).
		Owns(&nnfv1alpha1.NnfStorage{}).
		Complete(r)
}
