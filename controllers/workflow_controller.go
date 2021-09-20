/*
Copyright 2021 Hewlett Packard Enterprise Development LP
*/

package controllers

import (
	"context"
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

	// nnfv1alpha1 "stash.us.cray.com/RABSW/nnf-sos/api/v1alpha1"
	dwsv1alpha1 "stash.us.cray.com/dpm/dws-operator/api/v1alpha1"
)

// WorkflowReconciler reconciles a Workflow object
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
		log.Error(err, "Failed to get instance")
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
		*/

		controllerutil.RemoveFinalizer(workflow, finalizer)
		if err := r.Update(ctx, workflow); err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	driverID := os.Getenv("DWS_DRIVER_ID")

	// Proposal State work
	if workflow.Status.State == dwsv1alpha1.StateProposal.String() {
		log.Info("Generate DirectiveBreakdowns")

		for dwIndex, dwDirective := range workflow.Spec.DWDirectives {
			var result controllerutil.OperationResult
			var dbd *dwsv1alpha1.DirectiveBreakdown

			result, dbd, err = r.generateDirectiveBreakdown(ctx, dwDirective, dwIndex, workflow, log)
			if err != nil {
				log.Error(err, "Failed to create or update DirectiveBreakdown")
				return ctrl.Result{}, err
			}

			// If nothing happened the breakdown is set, continue looking for breakdowns to create
			if result == controllerutil.OperationResultNone {
				continue
			}

			// Do we need to add object reference of dbd to the workflow's directiveBreakdown array?
			if needDirectiveBreakdownReference(dbd, workflow) {
				// Add reference to directiveBreakdown to the workflow
				workflow.Status.DirectiveBreakdowns = append(workflow.Status.DirectiveBreakdowns, v1.ObjectReference{Name: dbd.Name, Namespace: dbd.Namespace})

				// Need to update the workflow because a DirectiveBreakdown was added or updated
				err := r.Update(ctx, workflow)
				if err != nil {
					log.Error(err, "Failed to update Workflow DirectiveBreakdowns")
				}

				// Until all DirectiveBreakdowns have been created, requeue and continue creating
				return ctrl.Result{}, err
			}
		}

		log.Info("Generate Computes")
		name := workflow.Name
		computes, err := r.createComputes(ctx, workflow, name, log)
		if err != nil {
			log.Error(err, "Failed to create or update Computes", "name", name)
			return ctrl.Result{}, err
		}

		// Ensure the computes reference is set
		cref := v1.ObjectReference{Name: computes.Name, Namespace: computes.Namespace}
		if workflow.Status.Computes != cref {
			workflow.Status.Computes = cref

			err = r.Update(ctx, workflow)
			if err != nil {
				log.Error(err, "Failed to add computes reference")
			}
		}

		log.Info("Generate Servers")
		for _, bdRef := range workflow.Status.DirectiveBreakdowns {
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
				log.Error(err, "Failed to create or update Servers", "name", name)
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

		// Check to see if we've completed the desiredState's work
		log.Info("Completing proposal state")
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

			log.Info("Updating driver", "id", driverStatus.DriverID, "dwd", driverStatus.DWDIndex)
			driverStatus.Completed = true
			err = r.Update(ctx, workflow)
			if err != nil {
				log.Error(err, "Failed to update Workflow state")
				return ctrl.Result{}, nil
			}
			return ctrl.Result{Requeue: true}, nil
		}

		return ctrl.Result{}, nil
	}

	// Check to see if we've completed the desiredState's work
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
		err = r.Update(ctx, workflow)
		if err != nil {
			log.Error(err, "Failed to update Workflow state")
			return ctrl.Result{}, nil
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *WorkflowReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&dwsv1alpha1.Workflow{}).
		Complete(r)
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
			dwdName := wf.Name + strconv.Itoa(dwIndex)
			switch cmdElements[1] {
			case "create_persistent":
				lifetime = "persistent"
				dwdName = wf.Name
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

					log.Info("mutating", "dbd", directiveBreakdown)
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
							{"AllocateSingleServer", mdtCapacity, "mdt", ""},                      // NOTE: hardcoded size
							{"AllocateSingleServer", mgtCapacity, "mgt", "OnePerRabbitPerSystem"}, // NOTE: hardcoded size
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

	// Did not find it.
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

	return server, nil
}

func (r *WorkflowReconciler) createComputes(ctx context.Context, wf *dwsv1alpha1.Workflow, name string, log logr.Logger) (computes *dwsv1alpha1.Computes, err error) {

	computes = &dwsv1alpha1.Computes{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: wf.Namespace,
		},
	}
	log.Info("Trying to create Servers", "name", computes.Name)

	_, err = ctrl.CreateOrUpdate(ctx, r.Client, computes,
		func() error {
			// Link the Computes to the workflow
			return ctrl.SetControllerReference(wf, computes, r.Scheme)
		})

	if err != nil {
		log.Error(err, "Failed to create or update Servers", "name", computes.Name)
		return nil, err
	}

	log.Info("done", "workflow", wf, "name", computes.Name)

	return computes, nil
}
