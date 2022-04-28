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
	"math"
	"reflect"
	"regexp"
	"runtime"
	"strconv"

	"github.com/go-logr/logr"

	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	dwsv1alpha1 "github.hpe.com/hpe/hpc-dpm-dws-operator/api/v1alpha1"
	"github.hpe.com/hpe/hpc-dpm-dws-operator/utils/dwdparse"
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

type lustreComponentType struct {
	strategy      string
	cap           int64
	labelsStr     string
	colocationKey string
}

//+kubebuilder:rbac:groups=dws.cray.hpe.com,resources=directivebreakdowns,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=dws.cray.hpe.com,resources=directivebreakdowns/status,verbs=get;list;watch;create;update;patch
//+kubebuilder:rbac:groups=dws.cray.hpe.com,resources=directivebreakdowns/finalizers,verbs=update
//+kubebuilder:rbac:groups=dws.cray.hpe.com,resources=servers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=dws.cray.hpe.com,resources=persistentstorageinstance,verbs=get;list;watch;create;update;patch;delete

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

	// Default Servers name is the same as the DirectiveBreakdown
	serversName := dbd.Name
	serversNamespace := dbd.Namespace
	var persistentInstance *dwsv1alpha1.PersistentStorageInstance
	var serversOwner metav1.Object

	// Default owner for non-persistent storage is the DirectiveBreakdown
	serversOwner = dbd

	// If the lifetime of the storage is persistent, create PersistentStorageInstance
	if dbd.Spec.Lifetime == dwsv1alpha1.DirectiveLifetimePersistent {
		// Servers, NnfStorage, and NnfNodeStorage resources will be named according to the PersistentStorageInstance
		serversName = dbd.Spec.Name

		var result controllerutil.OperationResult
		result, persistentInstance, err = r.createOrUpdatePersistentStorageInstance(ctx, dbd, dbd.Spec.Name, &dbd.Status.Servers, log)
		if err != nil {
			return ctrl.Result{}, err
		}

		// If we failed to create the persistent instance or we created it this pass, requeue.
		if persistentInstance == nil || result == controllerutil.OperationResultCreated {
			return ctrl.Result{Requeue: true}, nil
		}

		// Since this is a persistent instance, the persistentInstance becomes the owner of the server
		serversOwner = persistentInstance
	}

	// Create the corresponding Servers object
	servers, err := r.createServers(ctx, serversName, serversNamespace, serversOwner, log)
	if err != nil {
		return ctrl.Result{}, err
	}

	if servers == nil {
		return ctrl.Result{Requeue: true}, nil
	}

	// Add Servers reference to persistentInstance
	if persistentInstance != nil {
		_, persistentInstance, err = r.createOrUpdatePersistentStorageInstance(
			ctx,
			dbd,
			dbd.Spec.Name,
			&v1.ObjectReference{
				Kind:      reflect.TypeOf(dwsv1alpha1.Servers{}).Name(),
				Name:      servers.Name,
				Namespace: servers.Namespace,
			},
			log)
	}

	dbd.Status.Servers = v1.ObjectReference{
		Kind:      reflect.TypeOf(dwsv1alpha1.Servers{}).Name(),
		Name:      servers.Name,
		Namespace: servers.Namespace,
	}

	result, err := r.populateDirectiveBreakdown(ctx, dbd, log)
	if err != nil {
		return ctrl.Result{}, err
	}

	if result == updated {
		log.V(1).Info("Updated directiveBreakdown", "name", dbd.Name)

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

func (r *DirectiveBreakdownReconciler) createOrUpdatePersistentStorageInstance(ctx context.Context, dbd *dwsv1alpha1.DirectiveBreakdown, name string, serversRef *v1.ObjectReference, log logr.Logger) (controllerutil.OperationResult, *dwsv1alpha1.PersistentStorageInstance, error) {

	psi := &dwsv1alpha1.PersistentStorageInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: dbd.Namespace,
		},
	}

	result, err := ctrl.CreateOrUpdate(ctx, r.Client, psi,
		func() error {
			psi.Spec.Name = dbd.Spec.Name
			psi.Spec.FsType = dbd.Spec.Type
			psi.Spec.DWDirective = dbd.Spec.DW.DWDirective

			if serversRef != nil {
				psi.Spec.Servers = *serversRef
			}
			return nil
		})

	if err != nil {
		if apierrors.IsConflict(err) {
			return result, nil, nil
		}

		log.Error(err, "Failed to create or update PersistentStorageInstance", "name", psi.Name)
		return result, nil, err
	}

	switch result {
	case controllerutil.OperationResultCreated:
		log.Info("Created persistentInstance", "name", psi.Name, "PIName", psi.Spec.Name)
	case controllerutil.OperationResultNone:
		// Empty
	case controllerutil.OperationResultUpdated:
		log.Info("Updated persistentInstance", "name", psi.Name, "PIName", psi.Spec.Name)
	}

	return result, psi, err
}

func (r *DirectiveBreakdownReconciler) createServers(ctx context.Context, serversName string, serversNamespace string, serversOwner metav1.Object, log logr.Logger) (*dwsv1alpha1.Servers, error) {

	server := &dwsv1alpha1.Servers{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serversName,
			Namespace: serversNamespace,
		},
	}

	result, err := ctrl.CreateOrUpdate(ctx, r.Client, server,
		func() error {
			return ctrl.SetControllerReference(serversOwner, server, r.Scheme)
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

// populateDirectiveBreakdown parses the #DW to pull out the relevant information for the WLM to see.
func (r *DirectiveBreakdownReconciler) populateDirectiveBreakdown(ctx context.Context, dbd *dwsv1alpha1.DirectiveBreakdown, log logr.Logger) (breakdownPopulateResult, error) {

	result := verified
	argsMap, err := dwdparse.BuildArgsMap(dbd.Spec.DW.DWDirective)
	if err != nil {
		result = skipped
		return result, err
	}

	nnfStorageProfile, err := findProfileToUse(ctx, r.Client, argsMap)
	if err != nil {
		result = skipped
		return result, err
	}

	// The directive has been validated by the webhook, so we can assume the pieces we need are in the map.
	filesystem := argsMap["type"]
	capacity := argsMap["capacity"]

	breakdownCapacity, _ := getCapacityInBytes(capacity)

	// allocationSet represents the result we need to produce.
	// We build it then check to see if the directiveBreakdown's
	// AllocationSet matches. If so, we don't change it.
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

		lustreData := mergeLustreStorageDirectiveAndProfile(argsMap, nnfStorageProfile)

		// We need 3 distinct components for Lustre, ost, mdt, and mgt
		var lustreComponents []lustreComponentType
		lustreComponents = append(lustreComponents, lustreComponentType{"AllocateAcrossServers", breakdownCapacity, "ost", ""})

		if lustreData.CombinedMGTMDT {
			lustreComponents = append(lustreComponents, lustreComponentType{"AllocateSingleServer", mdtCapacity, "mgtmdt", "lustre-mgt"})
		} else if len(lustreData.ExternalMGS) > 0 {
			lustreComponents = append(lustreComponents, lustreComponentType{"AllocateSingleServer", mdtCapacity, "mdt", ""})
		} else {
			lustreComponents = append(lustreComponents, lustreComponentType{"AllocateSingleServer", mdtCapacity, "mdt", ""})
			lustreComponents = append(lustreComponents, lustreComponentType{"AllocateSingleServer", mgtCapacity, "mgt", "lustre-mgt"})
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

// SetupWithManager sets up the controller with the Manager.
func (r *DirectiveBreakdownReconciler) SetupWithManager(mgr ctrl.Manager) error {
	maxReconciles := runtime.GOMAXPROCS(0)
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(controller.Options{MaxConcurrentReconciles: maxReconciles}).
		For(&dwsv1alpha1.DirectiveBreakdown{}).
		Owns(&dwsv1alpha1.Servers{}).
		Complete(r)
}
