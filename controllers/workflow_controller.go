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
	uuid "github.com/google/uuid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	nnfv1alpha1 "stash.us.cray.com/RABSW/nnf-sos/api/v1alpha1"
	dwsv1alpha1 "stash.us.cray.com/dpm/dws-operator/api/v1alpha1"
)

// WorkflowReconciler reconciles a Workflow object
type WorkflowReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=dws.cray.hpe.com,resources=workflows,verbs=get;list;watch;update;patch
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

	driverID := os.Getenv("WORKFLOW_WEBHOOK_DRIVER_ID")

	// Ensure we've generated DWDirectiveBreakdowns
	if (workflow.Status.State == dwsv1alpha1.StateProposal) && needDirectiveBreakdowns(workflow) {
		log.Info("Generate dwDirectiveBreakdowns")

		err = r.generateDirectiveBreakdowns(workflow.Spec.DWDirectives, workflow, log)
		if err != nil {
			return ctrl.Result{}, err
		}

		err := r.Update(ctx, workflow)
		if err != nil {
			log.Error(err, "Failed to update Workflow state")
		}

		// We're not finished yet, we need to generate NnfStorage resources for each DWDirectiveBreakdown
		return ctrl.Result{Requeue: true}, err
	}

	// Create and link an NnfStorage CR's to each DWDirectiveBreakdown
	if workflow.Status.State == dwsv1alpha1.StateProposal {

		for i := range workflow.Status.DWDirectiveBreakdowns {
			b := &workflow.Status.DWDirectiveBreakdowns[i]

			filesystem := b.Type
			nnfName := workflow.Name + "." + strconv.Itoa(b.DW.DWDirectiveIndex) + "." + filesystem
			result, err := r.createNNFStorageResource(ctx, workflow, nnfName, b.AllocationSet[0].MinimumCapacity, filesystem, b, log)
			if err != nil {
				log.Error(err, "Failed to create or update NnfStorage")
				return ctrl.Result{}, err
			}

			// Check the next breakdown if the prior has its NnfStorage in place.
			if result == controllerutil.OperationResultNone {
				continue
			}

			err = r.Update(ctx, workflow)
			if err != nil {
				log.Error(err, "Failed to update Workflow state")
			}

			// Continue to reconcile until all DWDirectiveBreakdowns have been satisfied.
			return ctrl.Result{Requeue: true}, err
		}

		// No more NnfStorage objects required, We've achieved Proposal state,
		// mark driver status completed
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
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *WorkflowReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&dwsv1alpha1.Workflow{}).
		Complete(r)
}

func needDirectiveBreakdowns(workflow *dwsv1alpha1.Workflow) bool {
	return (len(workflow.Status.DWDirectiveBreakdowns) == 0)
}

func (r *WorkflowReconciler) generateDirectiveBreakdowns(directives []string, wf *dwsv1alpha1.Workflow, log logr.Logger) error {

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

	for dwIndex, dwd := range directives {
		cmdElements := strings.Fields(dwd)

	CommandLoop:
		for _, breakThisDown := range breakDownCommands {
			// We care about the commands that generate breakdowns
			if breakThisDown == cmdElements[1] {

				// Construct a map of the arguments within the directive
				m := make(map[string]string)
				for _, pair := range cmdElements[2:] {
					if arg := strings.Split(pair, "="); len(arg) > 1 {
						m[arg[0]] = arg[1]
					}
				}

				filesystem := m["type"]
				name := m["name"]
				capacity := m["capacity"]

				var lifetime string
				var dwdName string
				switch cmdElements[1] {
				case "jobdw":
					lifetime = "job"
					dwdName = strconv.Itoa(wf.Spec.JobID) + name
				case "create_persistent":
					lifetime = "persistent"
					dwdName = name
				default:
					lifetime = "job"
					dwdName = strconv.Itoa(wf.Spec.JobID) + name
				}

				directiveBreakdown := dwsv1alpha1.DWDirectiveBreakdown{
					DW:            dwsv1alpha1.DWRecord{},
					Type:          filesystem,
					Name:          dwdName,
					Lifetime:      lifetime,
					AllocationSet: []dwsv1alpha1.AllocationSetComponents{},
				}
				directiveBreakdown.DW.DWDirectiveIndex = dwIndex
				directiveBreakdown.DW.DWDirective = dwd
				breakdownCapacity, _ := getCapacityInBytes(capacity)

				// Depending on the #DW's type, we have different work to do
				switch filesystem {
				case "raw", "xfs", "gfs2":

					component := dwsv1alpha1.AllocationSetComponents{}
					populateAllocationSetComponents(&component, "AllocatePerCompute", breakdownCapacity, filesystem, "")

					log.Info("allocationSet", "comp", component)

					directiveBreakdown.AllocationSet = append(directiveBreakdown.AllocationSet, component)

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
						{"DivideAcrossRabbits", breakdownCapacity, "ost", ""},
						{"SingleRabbit", mdtCapacity, "mdt", ""},                      // NOTE: hardcoded size
						{"SingleRabbit", mgtCapacity, "mgt", "OnePerRabbitPerSystem"}, // NOTE: hardcoded size
					}

					for _, i := range lustreComponents {
						component := dwsv1alpha1.AllocationSetComponents{}
						populateAllocationSetComponents(&component, i.strategy, i.cap, i.labelsStr, i.constraintStr)

						directiveBreakdown.AllocationSet = append(directiveBreakdown.AllocationSet, component)
					}
				}

				wf.Status.DWDirectiveBreakdowns = append(wf.Status.DWDirectiveBreakdowns, directiveBreakdown)
				log.Info("added breakdown", "directiveBreakdown", directiveBreakdown.DW.DWDirective)

				// The directive's command has been matched, no need to look at the other breakDownCommands
				break CommandLoop
			}
		}
	}
	return nil
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

func (r *WorkflowReconciler) createNNFStorageResource(ctx context.Context, wf *dwsv1alpha1.Workflow, nnfName string, cap int64, filesys string, ddbd *dwsv1alpha1.DWDirectiveBreakdown, log logr.Logger) (controllerutil.OperationResult, error) {

	nnfStorage := &nnfv1alpha1.NnfStorage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nnfName,
			Namespace: wf.Namespace,
		},
	}
	log.Info("Trying to create NNFStorage", "name", nnfStorage.Name)

	result, err := ctrl.CreateOrUpdate(ctx, r.Client, nnfStorage,
		func() error {
			// Don't reset the Uuid if it has already been set
			if len(nnfStorage.Spec.Uuid) == 0 {
				nnfStorage.Spec.Uuid = uuid.New().String()
			}

			nnfStorage.Spec.Capacity = cap
			nnfStorage.Spec.WorkflowReference.Name = wf.Name
			nnfStorage.Spec.WorkflowReference.Namespace = wf.Namespace

			// For Job lifecycles, link the NnfStorage to the workflow so the garbage collector can clean it up
			// when the workflow is deleted if it hasn't already been linked.
			if ddbd.Lifetime == "job" {
				return ctrl.SetControllerReference(wf, nnfStorage, r.Scheme)
			}

			return nil
		})

	if err != nil {
		log.Error(err, "Failed to create or update nnf storage")
		return result, err
	}

	ddbd.StorageReference.Name = nnfStorage.GetName()
	ddbd.StorageReference.Namespace = nnfStorage.GetNamespace()
	ddbd.StorageReference.Kind = nnfStorage.Kind

	log.Info("done", "workflow", wf, "ddbd", ddbd)

	return result, nil
}
