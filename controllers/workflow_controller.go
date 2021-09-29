/*
Copyright 2021 Hewlett Packard Enterprise Development LP
*/

package controllers

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/c2h5oh/datasize"
	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	nnfv1alpha1 "stash.us.cray.com/RABSW/nnf-sos/api/v1alpha1"
	dwsv1alpha1 "stash.us.cray.com/dpm/dws-operator/api/v1alpha1"
)

// WorkflowReconciler contains the pieces used by the reconciler
type WorkflowReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=dws.cray.hpe.com,resources=workflows,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups=dws.cray.hpe.com,resources=directivebreakdowns,verbs=get;create;list;watch;update;patch
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
func (r *WorkflowReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("storage", req.NamespacedName)

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

		if !controllerutil.ContainsFinalizer(workflow, finalizer) {
			return ctrl.Result{}, nil
		}

		// TODO: Teardown workflow
		/*
			if err := r.teardownWorkflow(ctx, workflow); err != nil {
				return ctrl.Result{}, err
			}

			controllerutil.RemoveFinalizer(workflow, finalizer)
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

func (r *WorkflowReconciler) updateDriversStatusForStatusState(workflow *dwsv1alpha1.Workflow, driverID string, log logr.Logger) bool {

	s := fmt.Sprintf("Marking drivers complete for %s state", workflow.Status.State)
	log.Info(s)

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

		driverStatus.Completed = true
		updateWorkflow = true
	}

	return updateWorkflow
}

func (r *WorkflowReconciler) completeDriverState(ctx context.Context, workflow *dwsv1alpha1.Workflow, driverID string, log logr.Logger) (ctrl.Result, error) {

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

func (r *WorkflowReconciler) handleProposalState(ctx context.Context, workflow *dwsv1alpha1.Workflow, driverID string, log logr.Logger) (ctrl.Result, error) {

	log.Info("Proposal")

	for dwIndex, dwDirective := range workflow.Spec.DWDirectives {
		log.Info("DirectiveBreakdown", "dwDirective", dwDirective)

		var directiveBreakdown *dwsv1alpha1.DirectiveBreakdown
		var err error

		_, directiveBreakdown, err = r.generateDirectiveBreakdown(ctx, dwDirective, dwIndex, workflow, log)
		if err != nil {
			return ctrl.Result{}, err
		}

		// Do we need to add object reference of dbd to the workflow's directiveBreakdown array?
		if needDirectiveBreakdownReference(directiveBreakdown, workflow) {
			// Add reference to directiveBreakdown to the workflow
			workflow.Status.DirectiveBreakdowns = append(workflow.Status.DirectiveBreakdowns, v1.ObjectReference{Name: directiveBreakdown.Name, Namespace: directiveBreakdown.Namespace})

			// Need to update the workflow because a DirectiveBreakdown was added or updated
			err := r.Update(ctx, workflow)
			if err != nil {
				log.Error(err, "Failed to update Workflow DirectiveBreakdowns")
			}

			// Until all DirectiveBreakdowns have been created, requeue and continue creating
			return ctrl.Result{Requeue: true}, err
		}
	}

	// Ensure Computes has been created
	name := workflow.Name
	computes, err := r.createComputes(ctx, workflow, name, log)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Ensure the computes reference is set
	cref := v1.ObjectReference{Name: computes.Name, Namespace: computes.Namespace}
	if workflow.Status.Computes != cref {
		log.Info("Updating workflow with Computes")
		workflow.Status.Computes = cref

		err = r.Update(ctx, workflow)
		if err != nil {
			log.Error(err, "Failed to add computes reference")
		}

		return ctrl.Result{}, err
	}

	for _, bdRef := range workflow.Status.DirectiveBreakdowns {
		log.Info("Generate Server", "breakdown", bdRef)

		directiveBreakdown := &dwsv1alpha1.DirectiveBreakdown{}

		err := r.Get(ctx, types.NamespacedName{Namespace: bdRef.Namespace, Name: bdRef.Name}, directiveBreakdown)
		if err != nil {
			// We expect the DirectiveBreakdown to be there since we checked in the loop above
			log.Error(err, "Failed to get DirectiveBreakdown", "name", bdRef.Name)
			return ctrl.Result{Requeue: true}, nil
		}

		name := directiveBreakdown.Name
		servers, err := r.createServers(ctx, workflow, directiveBreakdown, name, log)
		if err != nil {
			return ctrl.Result{}, err
		}

		sref := v1.ObjectReference{Name: servers.Name, Namespace: servers.Namespace}
		if directiveBreakdown.Spec.Servers != sref {
			directiveBreakdown.Spec.Servers = sref
			err = r.Update(ctx, directiveBreakdown)
			if err != nil {
				log.Error(err, "Failed to add servers reference", "dbd", directiveBreakdown.Name)
			}

			// Continue to reconcile until all DWDirectiveBreakdowns have a Servers CR associated with them.
			// Since we didn't update the workflow here, we need to request a requeue.
			return ctrl.Result{Requeue: true}, err
		}
	}

	// Complete state in the drivers
	return r.completeDriverState(ctx, workflow, driverID, log)
}

func (r *WorkflowReconciler) generateDirectiveBreakdown(ctx context.Context, directive string, dwIndex int, wf *dwsv1alpha1.Workflow, log logr.Logger) (result controllerutil.OperationResult, directiveBreakdown *dwsv1alpha1.DirectiveBreakdown, err error) {

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
			log.Info("Trying to create/update DirectiveBreakdown", "name", directiveBreakdown.Name)

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

					filesystem := m["type"]
					capacity := m["capacity"]

					directiveBreakdown.Spec.DW.DWDirectiveIndex = dwIndex
					directiveBreakdown.Spec.DW.DWDirective = directive
					directiveBreakdown.Spec.Name = m["name"]
					directiveBreakdown.Spec.Type = filesystem
					directiveBreakdown.Spec.Lifetime = lifetime
					breakdownCapacity, _ := getCapacityInBytes(capacity)

					// Ensure we are starting with an empty allocation set.
					directiveBreakdown.Spec.AllocationSet = nil

					// Depending on the #DW's filesystem (#DW type=<>) , we have different work to do
					switch filesystem {
					case "raw", "xfs", "gfs2":

						component := dwsv1alpha1.AllocationSetComponents{}
						populateAllocationSetComponents(&component, "AllocatePerCompute", breakdownCapacity, filesystem, "")

						log.Info("allocationSet", "comp", component)

						directiveBreakdown.Spec.AllocationSet = append(directiveBreakdown.Spec.AllocationSet, component)

					case "lustre":

						mdtCapacity, _ := getCapacityInBytes("1TB")
						mgtCapacity, _ := getCapacityInBytes("1GB")

						// We need 3 distinct components for Lustre, ost, mdt, and mgt
						var lustreComponents = []struct {
							strategy      string
							cap           int64
							labelsStr     string
							constraintStr string
						}{
							{"AllocateAcrossServers", breakdownCapacity, "ost", ""},
							{"AllocateSingleServer", mdtCapacity, "mdt", ""},               // NOTE: hardcoded size
							{"AllocateSingleServer", mgtCapacity, "mgt", "MayNotBeShared"}, // NOTE: hardcoded size
						}

						for _, i := range lustreComponents {
							component := dwsv1alpha1.AllocationSetComponents{}
							populateAllocationSetComponents(&component, i.strategy, i.cap, i.labelsStr, i.constraintStr)

							directiveBreakdown.Spec.AllocationSet = append(directiveBreakdown.Spec.AllocationSet, component)
						}
					}

					// Link the directive breakdown to the workflow
					return ctrl.SetControllerReference(wf, directiveBreakdown, r.Scheme)
				})

			if err != nil {
				log.Error(err, "Failed to create or update DirectiveBreakdown", "name", directiveBreakdown.Name)
				return
			}

			log.Info("added breakdown", "directiveBreakdown", directiveBreakdown.Spec.DW.DWDirective)

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

func getCapacityInBytes(capacity string) (int64, error) {
	var v datasize.ByteSize
	err := v.UnmarshalText([]byte(capacity))

	return int64(v.Bytes()), err
}

func populateAllocationSetComponents(a *dwsv1alpha1.AllocationSetComponents, strategy string, cap int64, labelStr string, labelConstraintStr string) {
	a.AllocationStrategy = strategy
	a.Label = labelStr
	a.Constraint = labelConstraintStr
	a.MinimumCapacity = cap
}

func (r *WorkflowReconciler) createComputes(ctx context.Context, wf *dwsv1alpha1.Workflow, name string, log logr.Logger) (computes *dwsv1alpha1.Computes, err error) {

	computes = &dwsv1alpha1.Computes{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: wf.Namespace,
		},
	}
	log.Info("Trying to create Computes", "name", computes.Name)

	_, err = ctrl.CreateOrUpdate(ctx, r.Client, computes,
		func() error {
			// Link the Computes to the workflow
			return ctrl.SetControllerReference(wf, computes, r.Scheme)
		})

	if err != nil {
		log.Error(err, "Failed to create or update Computes", "name", computes.Name)
		return nil, err
	}

	log.Info("done", "workflow", wf, "name", computes.Name)

	return computes, nil
}

func (r *WorkflowReconciler) createServers(ctx context.Context, wf *dwsv1alpha1.Workflow, dbd *dwsv1alpha1.DirectiveBreakdown, serverName string, log logr.Logger) (server *dwsv1alpha1.Servers, err error) {

	server = &dwsv1alpha1.Servers{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serverName,
			Namespace: wf.Namespace,
		},
	}
	log.Info("Trying to create/update Servers", "name", server.Name)

	_, err = ctrl.CreateOrUpdate(ctx, r.Client, server,
		func() error {
			// Link the Servers to the DirectiveBreakdown
			return ctrl.SetControllerReference(dbd, server, r.Scheme)
		})

	if err != nil {
		log.Error(err, "Failed to create or update Servers", "name", server.Name)
		return nil, err
	}

	log.Info("done", "workflow", wf, "name", server.Name)

	return server, err
}

func (r *WorkflowReconciler) handleSetupState(ctx context.Context, workflow *dwsv1alpha1.Workflow, driverID string, log logr.Logger) (ctrl.Result, error) {

	log.Info("Setup: Find each DirectiveBreakdown to locate its Servers object")

	// We don't start looking for NnfStorage to be "Ready" until we've created all of them.
	// In the following loop if we create anything, we requeue before we look at any
	// NnfStorage readiness.
	var nnfStorages []nnfv1alpha1.NnfStorage
	existingNnfStorage := true

	// Iterate through the directive breakdowns...
	for _, dbdRef := range workflow.Status.DirectiveBreakdowns {
		log.Info("DirectiveBreakdown", "name", dbdRef.Name)

		// Chain through the DirectiveBreakdown to the Servers object

		dbd := &dwsv1alpha1.DirectiveBreakdown{}
		err := r.Get(ctx, types.NamespacedName{Name: dbdRef.Name, Namespace: dbdRef.Namespace}, dbd)
		if err != nil {
			// TODO: Report error in workflow for missing DirectiveBreakdown
			return ctrl.Result{}, err
		}

		s := &dwsv1alpha1.Servers{}
		err = r.Get(ctx, types.NamespacedName{Name: dbd.Spec.Servers.Name, Namespace: dbd.Spec.Servers.Namespace}, s)
		if err != nil {
			// TODO: Report error in workflow for missing Server for the DirectiveBreakdown
			return ctrl.Result{}, err
		}

		nnfStorage, result, err := r.createNnfStorage(ctx, workflow, dbd, s, log)
		if err != nil {
			return ctrl.Result{}, err
		}

		if result == controllerutil.OperationResultCreated {
			existingNnfStorage = false
		}

		// If we've created all of the NnfStorages, accumulate them for examination in the next step.
		if existingNnfStorage {
			nnfStorages = append(nnfStorages, nnfStorage)
		}
	}

	// If we created any NnfStorage objects in the last loop, return and requeue before we check for completion.
	if !existingNnfStorage {
		return ctrl.Result{Requeue: true}, nil
	}

	// Walk the nnfStorages looking for them to be Ready. We exit the reconciler if
	// we encounter a non-ready AllocationSet indicating we haven't finished creating the storage.
	for i := range nnfStorages {
		for _, set := range nnfStorages[i].Status.AllocationSets {
			if set.Status != "Ready" {
				return ctrl.Result{}, nil
			}
		}
	}

	// Complete state in the drivers
	return r.completeDriverState(ctx, workflow, driverID, log)
}

func (r *WorkflowReconciler) createNnfStorage(ctx context.Context, wf *dwsv1alpha1.Workflow, d *dwsv1alpha1.DirectiveBreakdown, s *dwsv1alpha1.Servers, log logr.Logger) (nnfv1alpha1.NnfStorage, controllerutil.OperationResult, error) {

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
			for i := range s.Data {
				nnfAllocSet := nnfv1alpha1.NnfStorageAllocationSetSpec{}

				nnfAllocSet.Name = d.Spec.Name + "-" + s.Data[i].Label // Note: DirectiveBreakdown contains the name of the allocation
				nnfAllocSet.Capacity = s.Data[i].AllocationSize
				nnfAllocSet.FileSystemType = d.Spec.Type // DirectiveBreakdown contains the filesystem type (lustre, xfs, etc)
				if nnfAllocSet.FileSystemType == "lustre" {
					nnfAllocSet.NnfStorageLustreSpec.TargetType = strings.ToUpper(s.Data[i].Label)
					nnfAllocSet.NnfStorageLustreSpec.FileSystemName = d.Spec.Name[0:8]
				}

				// Create Nodes for this allocation set.
				for _, host := range s.Data[i].Hosts {
					node := nnfv1alpha1.NnfStorageAllocationNodes{Name: host.Name, Count: host.AllocationCount}
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
		return *nnfStorage, result, err
	}

	return *nnfStorage, result, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *WorkflowReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&dwsv1alpha1.Workflow{}).
		Owns(&nnfv1alpha1.NnfStorage{}).
		Complete(r)
}
