/*
Copyright 2021 Hewlett Packard Enterprise Development LP
*/

package controllers

import (
	"context"
	"fmt"
	"math"
	"reflect"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"github.com/go-logr/logr"

	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	dwsv1alpha1 "stash.us.cray.com/dpm/dws-operator/api/v1alpha1"
)

// Define condition values
const (
	ConditionTrue  bool = true
	ConditionFalse bool = false
)

const (
	// finalizerWorkflow defines the key used in identifying the
	// storage object as being owned by this NNF Storage Reconciler. This
	// prevents the system from deleting the custom resource until the
	// reconciler has finished in using the resource.
	finalizerDirectiveBreakdown = "nnf.cray.hpe.com/directiveBreakdown"
)

type breakdownPopulateResult string

const ( // They should complete the sentence "populateDirectiveBreakdown ____ the directiveBreakdown  "
	updated  breakdownPopulateResult = "updated"
	verified breakdownPopulateResult = "verified"
	skipped  breakdownPopulateResult = "skipped"
)

// DirectiveBreakdownReconciler reconciles a DirectiveBreakdown object
type DirectiveBreakdownReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *kruntime.Scheme
}

//+kubebuilder:rbac:groups=dws.cray.hpe.com,resources=directivebreakdowns,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=dws.cray.hpe.com,resources=directivebreakdowns/status,verbs=get;list;watch;create;update;patch
//+kubebuilder:rbac:groups=dws.cray.hpe.com,resources=directivebreakdowns/finalizers,verbs=update
//+kubebuilder:rbac:groups=dws.cray.hpe.com,resources=servers,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the directiveBreakdown closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.9.2/pkg/reconcile
func (r *DirectiveBreakdownReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("DirectiveBreakdown", req.NamespacedName)

	dbd := &dwsv1alpha1.DirectiveBreakdown{}

	err := r.Get(ctx, req.NamespacedName, dbd)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Check if the object is being deleted
	if !dbd.GetDeletionTimestamp().IsZero() {
		log.Info("Deleting directiveBreakdown...")

		if !controllerutil.ContainsFinalizer(dbd, finalizerDirectiveBreakdown) {
			return ctrl.Result{}, nil
		}

		// TODO: Teardown directiveBreakdown
		/*
			if err := r.teardownDirectiveBreakdown(ctx, dbd); err != nil {
				return ctrl.Result{}, err
			}

			controllerutil.RemoveFinalizer(dbd, finalizerDirectiveBreakdown)
			if err := r.Update(ctx, dbd); err != nil {
				return ctrl.Result{}, err
			}
		*/

		return ctrl.Result{}, nil
	}

	// Create the corresponding Servers object
	name := dbd.Name
	servers, err := r.createServers(ctx, dbd, name, log)
	if err != nil {
		return ctrl.Result{}, err
	}

	if servers == nil {
		return ctrl.Result{Requeue: true}, nil
	}

	sref := v1.ObjectReference{Name: servers.Name, Namespace: servers.Namespace}
	if dbd.Status.Servers != sref {
		dbd.Status.Servers = sref
	}

	result, err := r.populateDirectiveBreakdown(ctx, dbd, log)
	if err != nil {
		return ctrl.Result{}, err
	}

	if result == updated {
		log.Info("Updated directiveBreakdown", "name", dbd.Name)

		err = r.Status().Update(ctx, dbd)
		if err != nil {
			// Ignore conflict errors
			if apierrors.IsConflict(err) {
				return ctrl.Result{Requeue: true}, nil
			}

			log.Error(err, "failed to update status")
		}
	}

	return ctrl.Result{}, err
}

func (r *DirectiveBreakdownReconciler) populateDirectiveBreakdown(ctx context.Context, dbd *dwsv1alpha1.DirectiveBreakdown, log logr.Logger) (breakdownPopulateResult, error) {

	result := verified
	cmdElements := strings.Fields(dbd.Spec.DW.DWDirective)

	// Construct a map of the arguments within the directive
	m := make(map[string]string)
	for _, pair := range cmdElements[2:] {
		if arg := strings.Split(pair, "="); len(arg) > 1 {
			m[arg[0]] = arg[1]
		}
	}

	filesystem := m["type"]
	capacity := m["capacity"]

	breakdownCapacity, _ := getCapacityInBytes(capacity)

	// allocationSet represents the goal. We build it and then
	// check to see if the directiveBreakdown's AllocationSet matches. If so, we don't change it.
	var allocationSet []dwsv1alpha1.AllocationSetComponents

	// Depending on the #DW's filesystem (#DW type=<>) , we have different work to do
	switch filesystem {
	case "raw", "xfs", "gfs2":
		component := dwsv1alpha1.AllocationSetComponents{}
		populateAllocationSetComponents(&component, "AllocatePerCompute", breakdownCapacity, filesystem, "")

		log.Info("allocationSet", "comp", component)

		allocationSet = append(allocationSet, component)

	case "lustre":
		mdtCapacity, _ := getCapacityInBytes("1TB")
		mgtCapacity, _ := getCapacityInBytes("1GB")

		// We need 3 distinct components for Lustre, ost, mdt, and mgt
		var lustreComponents = []struct {
			strategy      string
			cap           int64
			labelsStr     string
			colocationKey string
		}{
			{"AllocateAcrossServers", breakdownCapacity, "ost", ""},
			{"AllocateSingleServer", mdtCapacity, "mdt", ""},           // NOTE: hardcoded size
			{"AllocateSingleServer", mgtCapacity, "mgt", "lustre-mgt"}, // NOTE: hardcoded size
		}

		for _, i := range lustreComponents {
			component := dwsv1alpha1.AllocationSetComponents{}
			populateAllocationSetComponents(&component, i.strategy, i.cap, i.labelsStr, i.colocationKey)

			allocationSet = append(allocationSet, component)
		}

	default:
		err := fmt.Errorf("failed to populate directiveBreakdown")
		log.Error(err, "populate directiveBreakdown", "directiveBreakdown", dbd.Name, "filesystem", filesystem)
		result = skipped
		return result, err
	}

	// If the dbd is missing the correct allocation set, assign it.
	if !reflect.DeepEqual(dbd.Status.AllocationSet, allocationSet) {
		result = updated
		dbd.Status.AllocationSet = allocationSet
		dbd.Status.Ready = ConditionTrue
	}

	if dbd.Status.Ready != ConditionTrue {
		result = updated
		dbd.Status.Ready = ConditionTrue
	}

	return result, nil
}

func getCapacityInBytes(capacity string) (int64, error) {

	// Matcher for capacity string
	multMatcher := regexp.MustCompile(`(\d+(\.\d*)?|\.\d+)([kKMGTP]i?)?B?$`)

	matches := multMatcher.FindStringSubmatch(capacity)
	if matches == nil {
		return 0, fmt.Errorf("invalid capacity string, %s", capacity)
	}

	var powers = map[string]float64{
		"":   1, // No units -> bytes, nothing to multiply
		"K":  math.Pow10(3),
		"M":  math.Pow10(6),
		"G":  math.Pow10(9),
		"T":  math.Pow10(12),
		"P":  math.Pow10(15),
		"Ki": math.Pow(2, 10),
		"Mi": math.Pow(2, 20),
		"Gi": math.Pow(2, 30),
		"Ti": math.Pow(2, 40),
		"Pi": math.Pow(2, 50)}

	// matches[0] is the entire string, we want the parts.
	val, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return 0, fmt.Errorf("invalid capacity string, %s", capacity)
	}

	return int64(math.Round(val * powers[matches[3]])), nil
}

func populateAllocationSetComponents(a *dwsv1alpha1.AllocationSetComponents, strategy string, cap int64, labelStr string, constraintKey string) {
	a.AllocationStrategy = strategy
	a.Label = labelStr
	a.MinimumCapacity = cap
	a.Constraints.Labels = []string{"dws.cray.hpe.com/storage=Rabbit"}
	if len(constraintKey) > 0 {
		a.Constraints.Colocation = []dwsv1alpha1.AllocationSetColocationConstraint{{
			Type: "exclusive",
			Key:  constraintKey,
		}}
	}
}

func (r *DirectiveBreakdownReconciler) createServers(ctx context.Context, dbd *dwsv1alpha1.DirectiveBreakdown, serverName string, log logr.Logger) (*dwsv1alpha1.Servers, error) {

	server := &dwsv1alpha1.Servers{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serverName,
			Namespace: dbd.Namespace,
		},
	}

	result, err := ctrl.CreateOrUpdate(ctx, r.Client, server,
		func() error {
			// Link the Servers to the DirectiveBreakdown
			return ctrl.SetControllerReference(dbd, server, r.Scheme)
		})

	if err != nil {
		if apierrors.IsConflict(err) {
			return nil, nil
		}

		log.Error(err, "Failed to create or update Servers", "name", server.Name)
		return nil, err
	}

	switch result {
	case controllerutil.OperationResultCreated:
		log.Info("Created server", "name", server.Name)
	case controllerutil.OperationResultNone:
		// Empty
	case controllerutil.OperationResultUpdated:
		log.Info("Updated server", "name", server.Name)
	}

	return server, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *DirectiveBreakdownReconciler) SetupWithManager(mgr ctrl.Manager) error {
	maxReconciles := runtime.GOMAXPROCS(0)
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(controller.Options{MaxConcurrentReconciles: maxReconciles}).
		For(&dwsv1alpha1.DirectiveBreakdown{}).
		Owns(&dwsv1alpha1.Servers{}).
		Complete(r)
}
