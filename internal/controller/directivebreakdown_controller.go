/*
 * Copyright 2021-2025 Hewlett Packard Enterprise Development LP
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

package controller

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

	dwsv1alpha5 "github.com/DataWorkflowServices/dws/api/v1alpha5"
	"github.com/DataWorkflowServices/dws/utils/dwdparse"
	"github.com/DataWorkflowServices/dws/utils/updater"
	nnfv1alpha7 "github.com/NearNodeFlash/nnf-sos/api/v1alpha7"
	"github.com/NearNodeFlash/nnf-sos/internal/controller/metrics"
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

// DirectiveBreakdownReconciler reconciles a DirectiveBreakdown object
type DirectiveBreakdownReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *kruntime.Scheme
}

type lustreComponentType struct {
	strategy      dwsv1alpha5.AllocationStrategy
	cap           int64
	labelsStr     string
	colocationKey *dwsv1alpha5.AllocationSetColocationConstraint
}

//+kubebuilder:rbac:groups=dataworkflowservices.github.io,resources=directivebreakdowns,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=dataworkflowservices.github.io,resources=directivebreakdowns/status,verbs=get;list;watch;create;update;patch
//+kubebuilder:rbac:groups=dataworkflowservices.github.io,resources=directivebreakdowns/finalizers,verbs=update
//+kubebuilder:rbac:groups=dataworkflowservices.github.io,resources=servers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=dataworkflowservices.github.io,resources=persistentstorageinstances,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the directiveBreakdown closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.9.2/pkg/reconcile
func (r *DirectiveBreakdownReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, err error) {
	log := r.Log.WithValues("DirectiveBreakdown", req.NamespacedName)

	metrics.NnfDirectiveBreakdownReconcilesTotal.Inc()

	dbd := &dwsv1alpha5.DirectiveBreakdown{}
	err = r.Get(ctx, req.NamespacedName, dbd)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	statusUpdater := updater.NewStatusUpdater[*dwsv1alpha5.DirectiveBreakdownStatus](dbd)
	defer func() { err = statusUpdater.CloseWithStatusUpdate(ctx, r.Client.Status(), err) }()
	defer func() { dbd.Status.SetResourceErrorAndLog(err, log) }()

	// Check if the object is being deleted
	if !dbd.GetDeletionTimestamp().IsZero() {
		log.Info("Deleting directiveBreakdown...")

		if !controllerutil.ContainsFinalizer(dbd, finalizerDirectiveBreakdown) {
			return ctrl.Result{}, nil
		}

		// Delete all children that are owned by this DirectiveBreakdown.
		deleteStatus, err := dwsv1alpha5.DeleteChildren(ctx, r.Client, r.getChildObjects(), dbd)
		if err != nil {
			return ctrl.Result{}, err
		}

		if !deleteStatus.Complete() {
			return ctrl.Result{}, nil
		}

		controllerutil.RemoveFinalizer(dbd, finalizerDirectiveBreakdown)
		if err := r.Update(ctx, dbd); err != nil {
			if !apierrors.IsConflict(err) {
				return ctrl.Result{}, err
			}

			return ctrl.Result{Requeue: true}, nil
		}

		return ctrl.Result{}, nil
	}

	// Add the finalizer if it doesn't exist
	if !controllerutil.ContainsFinalizer(dbd, finalizerDirectiveBreakdown) {
		controllerutil.AddFinalizer(dbd, finalizerDirectiveBreakdown)
		if err := r.Update(ctx, dbd); err != nil {
			if !apierrors.IsConflict(err) {
				return ctrl.Result{}, err
			}

			return ctrl.Result{Requeue: true}, nil
		}

		return ctrl.Result{}, nil
	}

	argsMap, err := dwdparse.BuildArgsMap(dbd.Spec.Directive)
	if err != nil {
		return ctrl.Result{}, dwsv1alpha5.NewResourceError("invalid DW directive: %s", dbd.Spec.Directive).WithError(err).WithUserMessage("invalid DW directive").WithFatal()
	}

	commonResourceName, commonResourceNamespace := getStorageReferenceNameFromDBD(dbd)

	switch argsMap["command"] {
	case "create_persistent":
		persistentStorage, err := r.createOrUpdatePersistentStorageInstance(ctx, dbd, commonResourceName, argsMap)
		if err != nil {
			return ctrl.Result{}, err
		}

		// Requeue if we failed to create the persistent instance
		if persistentStorage == nil {
			return ctrl.Result{Requeue: true}, nil
		}

		if persistentStorage.Status.Error != nil {
			return ctrl.Result{}, persistentStorage.Status.Error
		}

		// Wait for the ObjectReference to the Servers resource to be filled in
		if persistentStorage.Status.Servers == (v1.ObjectReference{}) {
			return ctrl.Result{}, nil
		}

		dbd.Status.Storage = &dwsv1alpha5.StorageBreakdown{
			Lifetime:  dwsv1alpha5.StorageLifetimePersistent,
			Reference: persistentStorage.Status.Servers,
		}

		err = r.populateStorageBreakdown(ctx, dbd, commonResourceName, argsMap)
		if err != nil {
			return ctrl.Result{}, err
		}
	case "persistentdw":
		// Find the peristentStorageInstance that the persistentdw is referencing
		persistentStorage := &dwsv1alpha5.PersistentStorageInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name:      commonResourceName,
				Namespace: commonResourceNamespace,
			},
		}

		err := r.Get(ctx, client.ObjectKeyFromObject(persistentStorage), persistentStorage)
		if err != nil {
			return ctrl.Result{}, err
		}

		servers := &dwsv1alpha5.Servers{
			ObjectMeta: metav1.ObjectMeta{
				Name:      persistentStorage.Status.Servers.Name,
				Namespace: persistentStorage.Status.Servers.Namespace,
			},
		}

		if err := r.Get(ctx, client.ObjectKeyFromObject(servers), servers); err != nil {
			return ctrl.Result{}, err
		}

		populateRequires(dbd, argsMap)

		// Create a location constraint for the compute nodes based on what type of file system
		// the persistent storage is using.
		dbd.Status.Compute = &dwsv1alpha5.ComputeBreakdown{
			Constraints: dwsv1alpha5.ComputeConstraints{},
		}

		for i := range servers.Spec.AllocationSets {
			constraint := dwsv1alpha5.ComputeLocationConstraint{
				Reference: v1.ObjectReference{
					Kind:      persistentStorage.Status.Servers.Kind,
					Name:      persistentStorage.Status.Servers.Name,
					Namespace: persistentStorage.Status.Servers.Namespace,
					FieldPath: fmt.Sprintf("servers.spec.allocationSets[%d]", i),
				},
			}

			if argsMap["type"] == "lustre" {
				// Lustre requires a network connection between compute and Rabbit
				constraint.Access = append(constraint.Access, dwsv1alpha5.ComputeLocationAccess{
					Type:     dwsv1alpha5.ComputeLocationNetwork,
					Priority: dwsv1alpha5.ComputeLocationPriorityMandatory,
				})
			} else if argsMap["type"] == "gfs2" {
				// GFS2 requires both PCIe and network connection between compute and Rabbit
				constraint.Access = append(constraint.Access, dwsv1alpha5.ComputeLocationAccess{
					Type:     dwsv1alpha5.ComputeLocationNetwork,
					Priority: dwsv1alpha5.ComputeLocationPriorityMandatory,
				})
				constraint.Access = append(constraint.Access, dwsv1alpha5.ComputeLocationAccess{
					Type:     dwsv1alpha5.ComputeLocationPhysical,
					Priority: dwsv1alpha5.ComputeLocationPriorityMandatory,
				})
			} else {
				// XFS and Raw only require PCIe connection between compute and Rabbit
				constraint.Access = append(constraint.Access, dwsv1alpha5.ComputeLocationAccess{
					Type:     dwsv1alpha5.ComputeLocationPhysical,
					Priority: dwsv1alpha5.ComputeLocationPriorityMandatory,
				})
			}
			dbd.Status.Compute.Constraints.Location = append(dbd.Status.Compute.Constraints.Location, constraint)
		}

	case "jobdw":
		pinnedProfile, err := createPinnedProfile(ctx, r.Client, r.Scheme, argsMap, dbd, commonResourceName)
		if err != nil {
			return ctrl.Result{}, err
		}

		if pinnedProfile == nil {
			return ctrl.Result{Requeue: true}, nil
		}

		// Create the corresponding Servers object
		servers, err := r.createServers(ctx, commonResourceName, commonResourceNamespace, dbd)
		if err != nil {
			return ctrl.Result{}, err
		}

		if servers == nil {
			return ctrl.Result{Requeue: true}, nil
		}

		serversReference := v1.ObjectReference{
			Kind:      reflect.TypeOf(dwsv1alpha5.Servers{}).Name(),
			Name:      servers.Name,
			Namespace: servers.Namespace,
		}

		dbd.Status.Storage = &dwsv1alpha5.StorageBreakdown{
			Lifetime:  dwsv1alpha5.StorageLifetimeJob,
			Reference: serversReference,
		}

		err = r.populateStorageBreakdown(ctx, dbd, commonResourceName, argsMap)
		if err != nil {
			return ctrl.Result{}, err
		}
		populateRequires(dbd, argsMap)

		// Create a location constraint for the compute nodes based on what type of file system
		// will be created.
		dbd.Status.Compute = &dwsv1alpha5.ComputeBreakdown{
			Constraints: dwsv1alpha5.ComputeConstraints{},
		}

		for i, allocationSet := range dbd.Status.Storage.AllocationSets {
			constraint := dwsv1alpha5.ComputeLocationConstraint{
				Reference: v1.ObjectReference{
					Kind:      reflect.TypeOf(dwsv1alpha5.Servers{}).Name(),
					Name:      servers.Name,
					Namespace: servers.Namespace,
					FieldPath: fmt.Sprintf("servers.spec.allocationSets[%d]", i),
				},
			}

			if argsMap["type"] == "lustre" {
				// Lustre requires a network connection between compute and Rabbit
				constraint.Access = append(constraint.Access, dwsv1alpha5.ComputeLocationAccess{
					Type:     dwsv1alpha5.ComputeLocationNetwork,
					Priority: dwsv1alpha5.ComputeLocationPriorityMandatory,
				})

				// If the "ColocateComputes" option is specified, force the computes to have a
				// physical connection to the storage to limit their placement
				targetOptions := pinnedProfile.GetLustreMiscOptions(allocationSet.Label)
				if targetOptions.ColocateComputes {
					constraint.Access = append(constraint.Access, dwsv1alpha5.ComputeLocationAccess{
						Type:     dwsv1alpha5.ComputeLocationPhysical,
						Priority: dwsv1alpha5.ComputeLocationPriorityBestEffort,
					})
				}
			} else if argsMap["type"] == "gfs2" {
				// GFS2 requires both PCIe and network connection between compute and Rabbit
				constraint.Access = append(constraint.Access, dwsv1alpha5.ComputeLocationAccess{
					Type:     dwsv1alpha5.ComputeLocationNetwork,
					Priority: dwsv1alpha5.ComputeLocationPriorityMandatory,
				})
				constraint.Access = append(constraint.Access, dwsv1alpha5.ComputeLocationAccess{
					Type:     dwsv1alpha5.ComputeLocationPhysical,
					Priority: dwsv1alpha5.ComputeLocationPriorityMandatory,
				})
			} else {
				// XFS and Raw only require PCIe connection between compute and Rabbit
				constraint.Access = append(constraint.Access, dwsv1alpha5.ComputeLocationAccess{
					Type:     dwsv1alpha5.ComputeLocationPhysical,
					Priority: dwsv1alpha5.ComputeLocationPriorityMandatory,
				})
			}

			dbd.Status.Compute.Constraints.Location = append(dbd.Status.Compute.Constraints.Location, constraint)
		}

	default:
	}

	dbd.Status.Ready = ConditionTrue

	return ctrl.Result{}, nil
}

func (r *DirectiveBreakdownReconciler) createOrUpdatePersistentStorageInstance(ctx context.Context, dbd *dwsv1alpha5.DirectiveBreakdown, name string, argsMap map[string]string) (*dwsv1alpha5.PersistentStorageInstance, error) {
	log := r.Log.WithValues("DirectiveBreakdown", client.ObjectKeyFromObject(dbd))

	psi := &dwsv1alpha5.PersistentStorageInstance{
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
				dwsv1alpha5.AddOwnerLabels(psi, dbd)
				err := ctrl.SetControllerReference(dbd, psi, r.Scheme)
				if err != nil {
					return err
				}
			} else {
				if psi.Spec.UserID != dbd.Spec.UserID {
					return dwsv1alpha5.NewResourceError("existing persistent storage user ID %v does not match user ID %v", psi.Spec.UserID, dbd.Spec.UserID).WithUserMessage("User ID does not match existing persistent storage").WithFatal().WithUser()
				}
			}

			psi.Spec.Name = argsMap["name"]
			psi.Spec.FsType = argsMap["type"]
			psi.Spec.DWDirective = dbd.Spec.Directive
			psi.Spec.UserID = dbd.Spec.UserID
			psi.Spec.State = dwsv1alpha5.PSIStateActive

			return nil
		})

	if err != nil {
		if apierrors.IsConflict(err) {
			return nil, nil
		}

		log.Error(err, "Failed to create or update PersistentStorageInstance", "name", psi.Name)
		return nil, err
	}

	switch result {
	case controllerutil.OperationResultCreated:
		log.Info("Created PersistentStorageInstance", "name", psi.Name, "PIName", psi.Spec.Name)
	case controllerutil.OperationResultNone:
		// Empty
	case controllerutil.OperationResultUpdated:
		log.Info("Updated PersistentStorageInstance", "name", psi.Name, "PIName", psi.Spec.Name)
	}

	return psi, err
}

func (r *DirectiveBreakdownReconciler) createServers(ctx context.Context, serversName string, serversNamespace string, dbd *dwsv1alpha5.DirectiveBreakdown) (*dwsv1alpha5.Servers, error) {
	log := r.Log.WithValues("DirectiveBreakdown", client.ObjectKeyFromObject(dbd))

	server := &dwsv1alpha5.Servers{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serversName,
			Namespace: serversNamespace,
		},
	}

	result, err := ctrl.CreateOrUpdate(ctx, r.Client, server,
		func() error {
			dwsv1alpha5.InheritParentLabels(server, dbd)
			dwsv1alpha5.AddOwnerLabels(server, dbd)

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
func (r *DirectiveBreakdownReconciler) populateStorageBreakdown(ctx context.Context, dbd *dwsv1alpha5.DirectiveBreakdown, commonResourceName string, argsMap map[string]string) error {
	log := r.Log.WithValues("DirectiveBreakdown", client.ObjectKeyFromObject(dbd))

	// The pinned profile will be named for the NnfStorage.
	nnfStorageProfile, err := findPinnedProfile(ctx, r.Client, dbd.GetNamespace(), commonResourceName)
	if err != nil {
		return dwsv1alpha5.NewResourceError("unable to find pinned NnfStorageProfile: %s/%s", commonResourceName, dbd.GetNamespace()).WithError(err).WithUserMessage("Unable to find pinned NnfStorageProfile").WithMajor()
	}

	// The directive has been validated by the webhook, so we can assume the pieces we need are in the map.
	filesystem := argsMap["type"]
	capacity, capacityExists := argsMap["capacity"]

	breakdownCapacity, _ := getCapacityInBytes(capacity)
	if breakdownCapacity == 0 && capacityExists {
		return dwsv1alpha5.NewResourceError("").WithUserMessage("'capacity' must be a non-zero value").WithFatal()
	}

	// allocationSets represents the result we need to produce.
	// We build it then check to see if the directiveBreakdown's
	// AllocationSet matches. If so, we don't change it.
	var allocationSets []dwsv1alpha5.StorageAllocationSet

	// Depending on the #DW's filesystem (#DW type=<>) , we have different work to do
	switch filesystem {
	case "raw":
		scalingFactor, err := strconv.ParseFloat(nnfStorageProfile.Data.RawStorage.CapacityScalingFactor, 64)
		if err != nil {
			return dwsv1alpha5.NewResourceError("").WithError(err).WithUserMessage("invalid capacityScalingFactor for raw allocation").WithFatal()
		}
		breakdownCapacity = int64(scalingFactor * float64(breakdownCapacity))

		component := dwsv1alpha5.StorageAllocationSet{}
		populateStorageAllocationSet(&component, dwsv1alpha5.AllocatePerCompute, breakdownCapacity, 0, 0, nnfStorageProfile.Data.RawStorage.StorageLabels, filesystem, nil)

		log.Info("allocationSets", "comp", component)

		allocationSets = append(allocationSets, component)
	case "xfs":
		scalingFactor, err := strconv.ParseFloat(nnfStorageProfile.Data.XFSStorage.CapacityScalingFactor, 64)
		if err != nil {
			return dwsv1alpha5.NewResourceError("").WithError(err).WithUserMessage("invalid capacityScalingFactor for xfs allocation").WithFatal()
		}
		breakdownCapacity = int64(scalingFactor * float64(breakdownCapacity))

		component := dwsv1alpha5.StorageAllocationSet{}
		populateStorageAllocationSet(&component, dwsv1alpha5.AllocatePerCompute, breakdownCapacity, 0, 0, nnfStorageProfile.Data.XFSStorage.StorageLabels, filesystem, nil)

		log.Info("allocationSets", "comp", component)

		allocationSets = append(allocationSets, component)
	case "gfs2":
		scalingFactor, err := strconv.ParseFloat(nnfStorageProfile.Data.GFS2Storage.CapacityScalingFactor, 64)
		if err != nil {
			return dwsv1alpha5.NewResourceError("").WithError(err).WithUserMessage("invalid capacityScalingFactor for gfs2 allocation").WithFatal()
		}
		breakdownCapacity = int64(scalingFactor * float64(breakdownCapacity))

		component := dwsv1alpha5.StorageAllocationSet{}
		populateStorageAllocationSet(&component, dwsv1alpha5.AllocatePerCompute, breakdownCapacity, 0, 0, nnfStorageProfile.Data.GFS2Storage.StorageLabels, filesystem, nil)

		log.Info("allocationSets", "comp", component)

		allocationSets = append(allocationSets, component)
	case "lustre":
		scalingFactor, err := strconv.ParseFloat(nnfStorageProfile.Data.LustreStorage.CapacityScalingFactor, 64)
		if err != nil {
			return dwsv1alpha5.NewResourceError("").WithError(err).WithUserMessage("invalid capacityScalingFactor for lustre allocation").WithFatal()
		}
		breakdownCapacity = int64(scalingFactor * float64(breakdownCapacity))
		mdtCapacity, _ := getCapacityInBytes(nnfStorageProfile.Data.LustreStorage.CapacityMDT)
		mgtCapacity, _ := getCapacityInBytes(nnfStorageProfile.Data.LustreStorage.CapacityMGT)

		// We need 3 distinct components for Lustre, ost, mdt, and mgt
		var lustreComponents []lustreComponentType

		lustreComponents = append(lustreComponents, lustreComponentType{dwsv1alpha5.AllocateAcrossServers, breakdownCapacity, "ost", nil})

		mgtKey := &dwsv1alpha5.AllocationSetColocationConstraint{Type: "exclusive", Key: "lustre-mgt"}
		var mdtKey *dwsv1alpha5.AllocationSetColocationConstraint
		if nnfStorageProfile.Data.LustreStorage.ExclusiveMDT {
			mdtKey = &dwsv1alpha5.AllocationSetColocationConstraint{Type: "exclusive"}
		}

		if nnfStorageProfile.Data.LustreStorage.CombinedMGTMDT {
			useKey := mgtKey
			// If both combinedMGTMDT and exclusiveMDT are specified, then exclusiveMDT wins.
			if mdtKey != nil {
				useKey = mdtKey
			}
			lustreComponents = append(lustreComponents, lustreComponentType{dwsv1alpha5.AllocateAcrossServers, mdtCapacity, "mgtmdt", useKey})
		} else if len(nnfStorageProfile.Data.LustreStorage.ExternalMGS) > 0 {
			lustreComponents = append(lustreComponents, lustreComponentType{dwsv1alpha5.AllocateAcrossServers, mdtCapacity, "mdt", mdtKey})
		} else if len(nnfStorageProfile.Data.LustreStorage.StandaloneMGTPoolName) > 0 {
			if argsMap["command"] != "create_persistent" {
				return dwsv1alpha5.NewResourceError("").WithUserMessage("standaloneMgtPoolName option can only be used with 'create_persistent' directive").WithFatal().WithUser()
			}

			lustreComponents = []lustreComponentType{{dwsv1alpha5.AllocateSingleServer, mgtCapacity, "mgt", mgtKey}}
		} else {
			lustreComponents = append(lustreComponents, lustreComponentType{dwsv1alpha5.AllocateAcrossServers, mdtCapacity, "mdt", mdtKey})
			lustreComponents = append(lustreComponents, lustreComponentType{dwsv1alpha5.AllocateSingleServer, mgtCapacity, "mgt", mgtKey})
		}

		for _, i := range lustreComponents {
			targetMiscOptions := nnfStorageProfile.GetLustreMiscOptions(i.labelsStr)
			component := dwsv1alpha5.StorageAllocationSet{}
			populateStorageAllocationSet(&component, i.strategy, i.cap, targetMiscOptions.Scale, targetMiscOptions.Count, targetMiscOptions.StorageLabels, i.labelsStr, i.colocationKey)

			allocationSets = append(allocationSets, component)
		}

	default:
		return dwsv1alpha5.NewResourceError("invalid DW directive file system type: %s", filesystem).WithUserMessage("invalid DW directive").WithFatal()
	}

	if dbd.Status.Storage == nil {
		dbd.Status.Storage = &dwsv1alpha5.StorageBreakdown{}
	}

	dbd.Status.Storage.AllocationSets = allocationSets
	return nil
}

func populateRequires(dbd *dwsv1alpha5.DirectiveBreakdown, argsMap map[string]string) {
	if wordList, present := argsMap["requires"]; present {
		dbd.Status.Requires = strings.Split(wordList, ",")
	}
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
		return 0, dwsv1alpha5.NewResourceError("invalid capacity string, %s", capacity)
	}

	return int64(math.Round(val * powers[matches[3]])), nil
}

func populateStorageAllocationSet(a *dwsv1alpha5.StorageAllocationSet, strategy dwsv1alpha5.AllocationStrategy, cap int64, scale int, count int, storageLabels []string, labelStr string, constraint *dwsv1alpha5.AllocationSetColocationConstraint) {
	a.AllocationStrategy = strategy
	a.Label = labelStr
	a.MinimumCapacity = cap
	a.Constraints.Labels = append(storageLabels, dwsv1alpha5.StorageTypeLabel+"=Rabbit")
	a.Constraints.Scale = scale
	a.Constraints.Count = count
	if constraint != nil {
		a.Constraints.Colocation = []dwsv1alpha5.AllocationSetColocationConstraint{*constraint}
	}
}

func (r *DirectiveBreakdownReconciler) getChildObjects() []dwsv1alpha5.ObjectList {
	return []dwsv1alpha5.ObjectList{
		&dwsv1alpha5.ServersList{},
		&nnfv1alpha7.NnfStorageProfileList{},
		&dwsv1alpha5.PersistentStorageInstanceList{},
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *DirectiveBreakdownReconciler) SetupWithManager(mgr ctrl.Manager) error {
	maxReconciles := runtime.GOMAXPROCS(0)
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(controller.Options{MaxConcurrentReconciles: maxReconciles}).
		For(&dwsv1alpha5.DirectiveBreakdown{}).
		Owns(&dwsv1alpha5.Servers{}).
		Owns(&dwsv1alpha5.PersistentStorageInstance{}).
		Owns(&nnfv1alpha7.NnfStorageProfile{}).
		Complete(r)
}
