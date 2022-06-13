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

	dwsv1alpha1 "github.com/HewlettPackard/dws/api/v1alpha1"
	"github.com/HewlettPackard/dws/utils/dwdparse"
)

// Define condition values
const (
	ConditionTrue  bool = true
	ConditionFalse bool = false
)

const (
	// finalizerWorkflow defines the key used in identifying the
	// storage object as being owned by this directive breakdown reconciler. This
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
//+kubebuilder:rbac:groups=dws.cray.hpe.com,resources=persistentstorageinstances,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the directiveBreakdown closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.9.2/pkg/reconcile
func (r *DirectiveBreakdownReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, err error) {
	log := r.Log.WithValues("DirectiveBreakdown", req.NamespacedName)

	dbd := &dwsv1alpha1.DirectiveBreakdown{}
	err = r.Get(ctx, req.NamespacedName, dbd)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	statusUpdater := newDirectiveBreakdownStatusUpdater(dbd)
	defer func() {
		if err == nil {
			err = statusUpdater.close(ctx, r)
		}
	}()

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

	argsMap, err := dwdparse.BuildArgsMap(dbd.Spec.Directive)
	if err != nil {
		return ctrl.Result{}, err
	}

	commonResourceName, commonResourceNamespace := getStorageReferenceNameFromDBD(dbd)

	switch argsMap["command"] {
	case "create_persistent":
		var result controllerutil.OperationResult
		result, persistentStorage, err := r.createOrUpdatePersistentStorageInstance(ctx, dbd, commonResourceName, argsMap)
		if err != nil {
			return ctrl.Result{}, err
		}

		// If we failed to create the persistent instance or we created it this pass, requeue.
		if persistentStorage == nil || result == controllerutil.OperationResultCreated {
			return ctrl.Result{}, nil
		}

		// Wait for the ObjectReference to the Servers resource to be filled in
		if persistentStorage.Status.Servers == (v1.ObjectReference{}) {
			return ctrl.Result{}, nil
		}

		dbd.Status.Storage = &dwsv1alpha1.StorageBreakdown{
			Lifetime:  dwsv1alpha1.StorageLifetimePersistent,
			Reference: persistentStorage.Status.Servers,
		}

		err = r.populateStorageBreakdown(ctx, dbd, commonResourceName, argsMap)
		if err != nil {
			return ctrl.Result{}, err
		}
	case "persistentdw":
		// Find the peristentStorageInstance that the persistentdw is referencing
		persistentStorage := &dwsv1alpha1.PersistentStorageInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name:      commonResourceName,
				Namespace: commonResourceNamespace,
			},
		}

		err := r.Get(ctx, client.ObjectKeyFromObject(persistentStorage), persistentStorage)
		if err != nil {
			return ctrl.Result{}, err
		}

		// Create a location constraint for the compute nodes based on what type of file system
		// the persistent storage is using. For Lustre, any compute node with network access to
		// the storage is acceptable. All other file systems need a physical connection to the storage.
		constraint := dwsv1alpha1.ComputeLocationConstraint{
			Reference: persistentStorage.Status.Servers,
		}

		if persistentStorage.Spec.FsType == "lustre" {
			constraint.Type = dwsv1alpha1.ComputeLocationNetwork
		} else {
			constraint.Type = dwsv1alpha1.ComputeLocationPhysical
		}

		dbd.Status.Compute = &dwsv1alpha1.ComputeBreakdown{
			Constraints: dwsv1alpha1.ComputeConstraints{
				Location: []dwsv1alpha1.ComputeLocationConstraint{constraint},
			},
		}
	case "jobdw":
		// Create the corresponding Servers object
		servers, err := r.createServers(ctx, commonResourceName, commonResourceNamespace, dbd)
		if err != nil {
			return ctrl.Result{}, err
		}

		if servers == nil {
			return ctrl.Result{Requeue: true}, nil
		}

		serversReference := v1.ObjectReference{
			Kind:      reflect.TypeOf(dwsv1alpha1.Servers{}).Name(),
			Name:      servers.Name,
			Namespace: servers.Namespace,
		}

		dbd.Status.Storage = &dwsv1alpha1.StorageBreakdown{
			Lifetime:  dwsv1alpha1.StorageLifetimeJob,
			Reference: serversReference,
		}

		// Create a location constraint for the compute nodes based on what type of file system
		// will be created. For Lustre, any compute node with network access to the storage is
		// acceptable. All other file systems need a physical connection to the storage.
		constraint := dwsv1alpha1.ComputeLocationConstraint{
			Reference: serversReference,
		}

		if argsMap["type"] == "lustre" {
			constraint.Type = dwsv1alpha1.ComputeLocationNetwork
		} else {
			constraint.Type = dwsv1alpha1.ComputeLocationPhysical
		}

		dbd.Status.Compute = &dwsv1alpha1.ComputeBreakdown{
			Constraints: dwsv1alpha1.ComputeConstraints{
				Location: []dwsv1alpha1.ComputeLocationConstraint{constraint},
			},
		}

		err = r.populateStorageBreakdown(ctx, dbd, commonResourceName, argsMap)
		if err != nil {
			return ctrl.Result{}, err
		}
	default:
	}

	dbd.Status.Ready = ConditionTrue

	return ctrl.Result{}, nil
}

func (r *DirectiveBreakdownReconciler) createOrUpdatePersistentStorageInstance(ctx context.Context, dbd *dwsv1alpha1.DirectiveBreakdown, name string, argsMap map[string]string) (controllerutil.OperationResult, *dwsv1alpha1.PersistentStorageInstance, error) {
	log := r.Log.WithValues("DirectiveBreakdown", client.ObjectKeyFromObject(dbd))

	psi := &dwsv1alpha1.PersistentStorageInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: dbd.Namespace,
		},
	}

	result, err := ctrl.CreateOrUpdate(ctx, r.Client, psi,
		func() error {
			// Only set the owner references during the create. The workflow controller
			// will remove the reference after setup phase has completed
			if psi.Spec.Name == "" {
				dwsv1alpha1.AddOwnerLabels(psi, dbd)
				err := ctrl.SetControllerReference(dbd, psi, r.Scheme)
				if err != nil {
					return err
				}
			}

			psi.Spec.Name = argsMap["name"]
			psi.Spec.FsType = argsMap["type"]
			psi.Spec.DWDirective = dbd.Spec.Directive

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
		log.Info("Created PersistentStorageInstance", "name", psi.Name, "PIName", psi.Spec.Name)
	case controllerutil.OperationResultNone:
		// Empty
	case controllerutil.OperationResultUpdated:
		log.Info("Updated PersistentStorageInstance", "name", psi.Name, "PIName", psi.Spec.Name)
	}

	return result, psi, err
}

func (r *DirectiveBreakdownReconciler) createServers(ctx context.Context, serversName string, serversNamespace string, dbd *dwsv1alpha1.DirectiveBreakdown) (*dwsv1alpha1.Servers, error) {
	log := r.Log.WithValues("DirectiveBreakdown", client.ObjectKeyFromObject(dbd))

	server := &dwsv1alpha1.Servers{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serversName,
			Namespace: serversNamespace,
		},
	}

	result, err := ctrl.CreateOrUpdate(ctx, r.Client, server,
		func() error {
			dwsv1alpha1.InheritParentLabels(server, dbd)
			dwsv1alpha1.AddOwnerLabels(server, dbd)

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
		log.Info("Created Server", "name", server.Name)
	case controllerutil.OperationResultNone:
		// Empty
	case controllerutil.OperationResultUpdated:
		log.Info("Updated Server", "name", server.Name)
	}

	return server, err
}

// populateDirectiveBreakdown parses the #DW to pull out the relevant information for the WLM to see.
func (r *DirectiveBreakdownReconciler) populateStorageBreakdown(ctx context.Context, dbd *dwsv1alpha1.DirectiveBreakdown, commonResourceName string, argsMap map[string]string) error {
	log := r.Log.WithValues("DirectiveBreakdown", client.ObjectKeyFromObject(dbd))

	// The pinned profile will be named for the NnfStorage.
	nnfStorageProfile, err := findPinnedProfile(ctx, r.Client, dbd.GetNamespace(), commonResourceName)
	if err != nil {
		log.Error(err, "Unable to find pinned NnfStorageProfile", "name", commonResourceName)
		return err
	}

	// The directive has been validated by the webhook, so we can assume the pieces we need are in the map.
	filesystem := argsMap["type"]
	capacity := argsMap["capacity"]

	breakdownCapacity, _ := getCapacityInBytes(capacity)

	// allocationSets represents the result we need to produce.
	// We build it then check to see if the directiveBreakdown's
	// AllocationSet matches. If so, we don't change it.
	var allocationSets []dwsv1alpha1.StorageAllocationSet

	// Depending on the #DW's filesystem (#DW type=<>) , we have different work to do
	switch filesystem {
	case "raw", "xfs", "gfs2":
		component := dwsv1alpha1.StorageAllocationSet{}
		populateStorageAllocationSet(&component, "AllocatePerCompute", breakdownCapacity, filesystem, "")

		log.Info("allocationSets", "comp", component)

		allocationSets = append(allocationSets, component)

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
			component := dwsv1alpha1.StorageAllocationSet{}
			populateStorageAllocationSet(&component, i.strategy, i.cap, i.labelsStr, i.colocationKey)

			allocationSets = append(allocationSets, component)
		}

	default:
		err := fmt.Errorf("failed to populate directiveBreakdown")
		log.Error(err, "populate directiveBreakdown", "directiveBreakdown", dbd.Name, "filesystem", filesystem)
		return err
	}

	if dbd.Status.Storage == nil {
		dbd.Status.Storage = &dwsv1alpha1.StorageBreakdown{}
	}

	dbd.Status.Storage.AllocationSets = allocationSets
	return nil
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

func populateStorageAllocationSet(a *dwsv1alpha1.StorageAllocationSet, strategy string, cap int64, labelStr string, constraintKey string) {
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

type directiveBreakdownStatusUpdater struct {
	directiveBreakdown *dwsv1alpha1.DirectiveBreakdown
	existingStatus     dwsv1alpha1.DirectiveBreakdownStatus
}

func newDirectiveBreakdownStatusUpdater(d *dwsv1alpha1.DirectiveBreakdown) *directiveBreakdownStatusUpdater {
	return &directiveBreakdownStatusUpdater{
		directiveBreakdown: d,
		existingStatus:     (*d.DeepCopy()).Status,
	}
}

func (d *directiveBreakdownStatusUpdater) close(ctx context.Context, r *DirectiveBreakdownReconciler) error {
	if !reflect.DeepEqual(d.directiveBreakdown.Status, d.existingStatus) {
		err := r.Status().Update(ctx, d.directiveBreakdown)
		if !apierrors.IsConflict(err) {
			return err
		}
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DirectiveBreakdownReconciler) SetupWithManager(mgr ctrl.Manager) error {
	maxReconciles := runtime.GOMAXPROCS(0)
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(controller.Options{MaxConcurrentReconciles: maxReconciles}).
		For(&dwsv1alpha1.DirectiveBreakdown{}).
		Owns(&dwsv1alpha1.Servers{}).
		Owns(&dwsv1alpha1.PersistentStorageInstance{}).
		Complete(r)
}
