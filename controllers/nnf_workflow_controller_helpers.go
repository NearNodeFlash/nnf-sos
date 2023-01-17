/*
 * Copyright 2022-2023 Hewlett Packard Enterprise Development LP
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
	"reflect"
	"strconv"
	"strings"
	"time"

	dwsv1alpha1 "github.com/HewlettPackard/dws/api/v1alpha1"
	"github.com/HewlettPackard/dws/utils/dwdparse"
	lusv1alpha1 "github.com/NearNodeFlash/lustre-fs-operator/api/v1alpha1"
	nnfv1alpha1 "github.com/NearNodeFlash/nnf-sos/api/v1alpha1"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// NNF Workflow stages all return a `result` structure when successful that describes
// the result of an operation at each stage of the workflow.
type result struct {
	ctrl.Result
	reason       string
	object       client.Object
	deleteStatus *dwsv1alpha1.DeleteStatus
}

// When workflow stages cannot advance they return a Requeue result with a particular reason.
func Requeue(reason string) *result {
	return &result{Result: ctrl.Result{}, reason: reason}
}

func (r *result) complete() bool {
	return r.IsZero()
}

func (r *result) after(duration time.Duration) *result {
	r.Result.RequeueAfter = duration
	return r
}

func (r *result) withObject(object client.Object) *result {
	r.object = object
	return r
}

func (r *result) withDeleteStatus(d dwsv1alpha1.DeleteStatus) *result {
	r.deleteStatus = &d
	return r
}

func (r *result) info() []interface{} {
	args := make([]interface{}, 0)
	if r == nil {
		return args
	}

	if len(r.reason) != 0 {
		args = append(args, "reason", r.reason)
	}

	if r.object != nil {
		args = append(args, "object", client.ObjectKeyFromObject(r.object).String())
		args = append(args, "kind", r.object.GetObjectKind().GroupVersionKind().Kind)
	}

	if r.deleteStatus != nil {
		args = append(args, r.deleteStatus.Info()...)
	}

	return args
}

// Validate the workflow and return any error found
func (r *NnfWorkflowReconciler) validateWorkflow(ctx context.Context, wf *dwsv1alpha1.Workflow) error {

	var createPersistentCount, deletePersistentCount, directiveCount int
	for _, directive := range wf.Spec.DWDirectives {

		directiveCount++
		// The webhook validated directives, ignore errors
		dwArgs, _ := dwdparse.BuildArgsMap(directive)
		command := dwArgs["command"]

		switch command {
		case "jobdw":

		case "copy_in", "copy_out":
			if err := r.validateStagingDirective(ctx, wf, directive); err != nil {
				return nnfv1alpha1.NewWorkflowError("Invalid staging Directive: " + directive).WithFatal().WithError(err)
			}

		case "create_persistent":
			createPersistentCount++

		case "destroy_persistent":
			deletePersistentCount++

		case "persistentdw":
			if err := r.validatePersistentInstanceDirective(ctx, wf, directive); err != nil {
				return nnfv1alpha1.NewWorkflowError("Could not validate persistent instance: " + directive).WithFatal().WithError(err)
			}

		case "container":
			if err := r.validateContainerDirective(ctx, wf, directive); err != nil {
				return nnfv1alpha1.NewWorkflowError("Could not validate container directive: " + directive).WithFatal().WithError(err)
			}
		}
	}

	if directiveCount > 1 {
		// Ensure create_persistent or destroy_persistent are singletons in the workflow
		if createPersistentCount+deletePersistentCount > 0 {
			return nnfv1alpha1.NewWorkflowError("Only a single create_persistent or destroy_persistent directive is allowed per workflow").WithFatal()
		}
	}

	return nil
}

// validateStagingDirective validates the staging copy_in/copy_out directives.
func (r *NnfWorkflowReconciler) validateStagingDirective(ctx context.Context, wf *dwsv1alpha1.Workflow, directive string) error {
	// Validate staging directive of the form...
	//   #DW copy_in source=[SOURCE] destination=[DESTINATION]
	//   #DW copy_out source=[SOURCE] destination=[DESTINATION]

	// For each copy_in/copy_out directive
	//   Make sure all $DW_JOB references point to job storage instance names
	//   Make sure all $DW_PERSISTENT references an existing persistentdw instance
	//   Otherwise, make sure source/destination is prefixed with a valid global lustre file system
	validateStagingArgument := func(arg string) error {
		name, _ := splitStagingArgumentIntoNameAndPath(arg)
		if strings.HasPrefix(arg, "$DW_JOB_") {
			if findDirectiveIndexByName(wf, name) == -1 {
				return nnfv1alpha1.NewWorkflowError(fmt.Sprintf("jobdw directive mentioning '%s' not found", name)).WithFatal()
			}
		} else if strings.HasPrefix(arg, "$DW_PERSISTENT_") {
			if err := r.validatePersistentInstanceByName(ctx, name, wf.Namespace); err != nil {
				return nnfv1alpha1.NewWorkflowError(fmt.Sprintf("persistent storage instance '%s' not found", name)).WithFatal()
			}
			if findDirectiveIndexByName(wf, name) == -1 {
				return nnfv1alpha1.NewWorkflowError(fmt.Sprintf("persistentdw directive mentioning '%s' not found", name)).WithFatal()
			}
		} else {
			if r.findLustreFileSystemForPath(ctx, arg, r.Log) == nil {
				return nnfv1alpha1.NewWorkflowError(fmt.Sprintf("global Lustre file system containing '%s' not found", arg)).WithFatal()
			}
		}

		return nil
	}

	args, err := dwdparse.BuildArgsMap(directive)
	if err != nil {
		return nnfv1alpha1.NewWorkflowError("Invalid DW directive: " + directive).WithFatal()
	}

	if err := validateStagingArgument(args["source"]); err != nil {
		return err
	}

	if err := validateStagingArgument(args["destination"]); err != nil {
		return err
	}

	return nil
}

// validateContainerDirective validates the container directive.
func (r *NnfWorkflowReconciler) validateContainerDirective(ctx context.Context, wf *dwsv1alpha1.Workflow, directive string) error {
	args, err := dwdparse.BuildArgsMap(directive)
	if err != nil {
		return nnfv1alpha1.NewWorkflowError("invalid DW directive: " + directive).WithFatal()
	}

	// Ensure the supplied profile exists or use the default
	profile, err := findContainerProfileToUse(ctx, r.Client, args)
	if err != nil {
		return nnfv1alpha1.NewWorkflowError(err.Error()).WithFatal()
	}

	// Check to see if the container storage argument is in the list of storages in the container profile
	checkStorageIsInProfile := func(storageName string) error {
		for _, storage := range profile.Data.Storages {
			if storage.Name == storageName {
				return nil
			}
		}
		return fmt.Errorf("storage '%s' not found in container profile '%s'", storageName, profile.Name)
	}

	// Find the wildcard JOB_DW and PERSISTENT_DW arguments. Verify that they exist in the
	// preceeding directives and in the container profile. For persistent arguments, ensure there
	// is a persistent storage instance.
	var suppliedStorageArguments []string // use this list later to validate non-optional storages
	for arg, storageName := range args {
		switch arg {
		case "command", "name", "profile":
		default:
			if strings.HasPrefix(arg, "JOB_DW_") {
				if findDirectiveIndexByName(wf, storageName) == -1 {
					return nnfv1alpha1.NewWorkflowError(fmt.Sprintf("jobdw directive mentioning '%s' not found", storageName)).WithFatal()
				}
				if err := checkStorageIsInProfile(arg); err != nil {
					return nnfv1alpha1.NewWorkflowError(err.Error()).WithFatal()
				}
				suppliedStorageArguments = append(suppliedStorageArguments, arg)
			} else if strings.HasPrefix(arg, "PERSISTENT_DW_") {
				if err := r.validatePersistentInstanceByName(ctx, storageName, wf.Namespace); err != nil {
					return nnfv1alpha1.NewWorkflowError(fmt.Sprintf("persistent storage instance '%s' not found", storageName)).WithFatal()
				}
				if findDirectiveIndexByName(wf, storageName) == -1 {
					return nnfv1alpha1.NewWorkflowError(fmt.Sprintf("persistentdw directive mentioning '%s' not found", storageName)).WithFatal()
				}
				if err := checkStorageIsInProfile(arg); err != nil {
					return nnfv1alpha1.NewWorkflowError(err.Error()).WithFatal()
				}
				suppliedStorageArguments = append(suppliedStorageArguments, arg)
			} else {
				return nnfv1alpha1.NewWorkflowError(fmt.Sprintf("unrecognized container argument: %s", arg)).WithFatal()
			}
		}
	}

	// Ensure that any storage in the profile that is non-optional is present in the supplied storage arguments
	checkNonOptionalStorages := func(storageArguments []string) error {
		findInStorageArguments := func(name string) bool {
			for _, directive := range storageArguments {
				if name == directive {
					return true
				}
			}
			return false
		}

		for _, storage := range profile.Data.Storages {
			if !storage.Optional {
				if !findInStorageArguments(storage.Name) {
					return fmt.Errorf("storage '%s' in container profile '%s' is not optional: storage argument not found in the supplied arguments",
						storage.Name, profile.Name)
				}
			}
		}

		return nil
	}

	if err := checkNonOptionalStorages(suppliedStorageArguments); err != nil {
		return nnfv1alpha1.NewWorkflowError(err.Error()).WithFatal()
	}

	return nil
}

// validatePersistentInstance validates the persistentdw directive.
func (r *NnfWorkflowReconciler) validatePersistentInstanceDirective(ctx context.Context, wf *dwsv1alpha1.Workflow, directive string) error {
	// Validate that the persistent instance is available and not in the process of being deleted
	args, err := dwdparse.BuildArgsMap(directive)
	if err != nil {
		return nnfv1alpha1.NewWorkflowError("Invalid DW directive: " + directive).WithFatal()
	}

	return r.validatePersistentInstanceByName(ctx, args["name"], wf.Namespace)
}

// validatePersistentInstance validates the persistentdw directive.
func (r *NnfWorkflowReconciler) validatePersistentInstanceByName(ctx context.Context, name string, namespace string) error {
	psi, err := r.getPersistentStorageInstance(ctx, name, namespace)
	if err != nil {
		return err
	}

	if !psi.DeletionTimestamp.IsZero() {
		return nnfv1alpha1.NewWorkflowError("Persistent storage instance " + name + " is deleting").WithFatal()
	}

	return nil
}

// Retrieve the persistent storage instance with the specified name
func (r *NnfWorkflowReconciler) getPersistentStorageInstance(ctx context.Context, name string, namespace string) (*dwsv1alpha1.PersistentStorageInstance, error) {
	psi := &dwsv1alpha1.PersistentStorageInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	err := r.Get(ctx, client.ObjectKeyFromObject(psi), psi)
	return psi, err
}

// generateDirectiveBreakdown creates a DirectiveBreakdown for any #DW directive that needs to specify storage
// or compute node information to the WLM (jobdw, create_persistent, persistentdw)
func (r *NnfWorkflowReconciler) generateDirectiveBreakdown(ctx context.Context, dwIndex int, workflow *dwsv1alpha1.Workflow, log logr.Logger) (*dwsv1alpha1.DirectiveBreakdown, error) {

	// DWDirectives that we need to generate directiveBreakdowns for look like this:
	//  #DW command            arguments...
	//  --- -----------------  -------------------------------------------
	// "#DW jobdw              type=raw    capacity=9TB name=thisIsReallyRaw"
	// "#DW jobdw              type=xfs    capacity=9TB name=thisIsXFS"
	// "#DW jobdw              type=gfs2   capacity=9GB name=thisIsGfs2"
	// "#DW jobdw              type=lustre capacity=9TB name=thisIsLustre"
	// "#DW create_persistent  type=lustre capacity=9TB name=thisIsPersistent"
	// "#DW persistentdw       name=thisIsPersistent"
	//
	// Container directives need to be accompanied by jobdw and/or persistentdw directives prior
	// to the container directive. The names of those jobdw/persistentdw directives must match
	// what is then supplied to the container directive as key value arguments.
	//
	// #DW jobdw name=my-gfs2 type=gfs2 capacity=1TB
	// #DW persistentdw name=some-lustre
	// #DW container name=my-foo profile=foo
	//     JOB_DW_foo-local-storage=my-gfs2
	//     PERSISTENT_DW_foo-persistent-storage=some-lustre

	// #DW commands that require a dwDirectiveBreakdown
	breakDownCommands := []string{
		"jobdw",
		"create_persistent",
		"persistentdw",
		"container",
	}

	directive := workflow.Spec.DWDirectives[dwIndex]
	dwArgs, _ := dwdparse.BuildArgsMap(directive)
	for _, breakThisDown := range breakDownCommands {
		// We care about the commands that generate a breakdown
		if breakThisDown == dwArgs["command"] {
			dwdName := indexedResourceName(workflow, dwIndex)
			directiveBreakdown := &dwsv1alpha1.DirectiveBreakdown{
				ObjectMeta: metav1.ObjectMeta{
					Name:      dwdName,
					Namespace: workflow.Namespace,
				},
			}

			result, err := ctrl.CreateOrUpdate(ctx, r.Client, directiveBreakdown,
				// Mutate function to fill in a directiveBreakdown
				func() error {
					dwsv1alpha1.AddWorkflowLabels(directiveBreakdown, workflow)
					dwsv1alpha1.AddOwnerLabels(directiveBreakdown, workflow)
					addDirectiveIndexLabel(directiveBreakdown, dwIndex)

					directiveBreakdown.Spec.Directive = directive
					directiveBreakdown.Spec.UserID = workflow.Spec.UserID

					// Link the directive breakdown to the workflow
					return ctrl.SetControllerReference(workflow, directiveBreakdown, r.Scheme)
				})

			if err != nil {
				log.Error(err, "failed to create or update DirectiveBreakdown", "name", directiveBreakdown.Name)
				return nil, fmt.Errorf("CreateOrUpdate failed for DirectiveBreakdown %v: %w", client.ObjectKeyFromObject(directiveBreakdown), err)
			}

			if result == controllerutil.OperationResultCreated {
				log.Info("Created DirectiveBreakdown", "name", directiveBreakdown.Name)
			} else if result == controllerutil.OperationResultNone {
				// no change
			} else {
				log.Info("Updated DirectiveBreakdown", "name", directiveBreakdown.Name)
			}

			// The directive's command has been matched, no need to look at the other breakDownCommands
			return directiveBreakdown, nil
		}
	}

	return nil, nil
}

func (r *NnfWorkflowReconciler) validateServerAllocations(ctx context.Context, dbd *dwsv1alpha1.DirectiveBreakdown, servers *dwsv1alpha1.Servers) error {
	if len(dbd.Status.Storage.AllocationSets) != 0 && len(dbd.Status.Storage.AllocationSets) != len(servers.Spec.AllocationSets) {
		err := fmt.Errorf("Servers resource does not meet storage requirements for directive '%s'", dbd.Spec.Directive)
		return nnfv1alpha1.NewWorkflowError("Allocation request does not meet directive requirements").WithFatal().WithError(err)
	}

	for _, breakdownAllocationSet := range dbd.Status.Storage.AllocationSets {
		found := false
		for _, serverAllocationSet := range servers.Spec.AllocationSets {
			if breakdownAllocationSet.Label != serverAllocationSet.Label {
				continue
			}

			found = true

			if breakdownAllocationSet.AllocationStrategy == dwsv1alpha1.AllocateSingleServer {
				if len(serverAllocationSet.Storage) != 1 || serverAllocationSet.Storage[0].AllocationCount != 1 {
					err := fmt.Errorf("Allocation set %s expected single allocation", breakdownAllocationSet.Label)
					return nnfv1alpha1.NewWorkflowError("Allocation request does not meet directive requirements").WithFatal().WithError(err)
				}
			}

			var totalCapacity int64 = 0

			if breakdownAllocationSet.AllocationStrategy == dwsv1alpha1.AllocateAcrossServers {
				for _, serverAllocation := range serverAllocationSet.Storage {
					totalCapacity += serverAllocationSet.AllocationSize * int64(serverAllocation.AllocationCount)
				}
			} else {
				totalCapacity = serverAllocationSet.AllocationSize
			}

			if totalCapacity < breakdownAllocationSet.MinimumCapacity {
				err := fmt.Errorf("Allocation set %s specified insufficient capacity", breakdownAllocationSet.Label)
				return nnfv1alpha1.NewWorkflowError("Allocation request does not meet directive requirements").WithFatal().WithError(err)
			}

			// Look up each of the storages specified to make sure they exist
			for _, serverAllocation := range serverAllocationSet.Storage {
				storage := &dwsv1alpha1.Storage{
					ObjectMeta: metav1.ObjectMeta{
						Name:      serverAllocation.Name,
						Namespace: corev1.NamespaceDefault,
					},
				}

				if err := r.Get(ctx, client.ObjectKeyFromObject(storage), storage); err != nil {
					if apierrors.IsNotFound(err) {
						return nnfv1alpha1.NewWorkflowError("Allocation request did not specify valid storage").WithFatal().WithError(err)
					}

					return nnfv1alpha1.NewWorkflowError("Could not validate allocation request").WithError(err)
				}
			}
		}

		if found == false {
			err := fmt.Errorf("Allocation set %s not found in Servers resource", breakdownAllocationSet.Label)
			return nnfv1alpha1.NewWorkflowError("Allocation request does not meet directive requirements").WithFatal().WithError(err)
		}
	}

	return nil

}

func (r *NnfWorkflowReconciler) createNnfStorage(ctx context.Context, workflow *dwsv1alpha1.Workflow, s *dwsv1alpha1.Servers, index int, log logr.Logger) (*nnfv1alpha1.NnfStorage, error) {
	nnfStorage := &nnfv1alpha1.NnfStorage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      s.Name,
			Namespace: s.Namespace,
		},
	}

	dwArgs, err := dwdparse.BuildArgsMap(workflow.Spec.DWDirectives[index])
	if err != nil {
		return nil, nnfv1alpha1.NewWorkflowError("Invalid DW directive: " + workflow.Spec.DWDirectives[index]).WithFatal()
	}

	pinnedName, pinnedNamespace := getStorageReferenceNameFromWorkflowActual(workflow, index)
	nnfStorageProfile, err := findPinnedProfile(ctx, r.Client, pinnedNamespace, pinnedName)
	if err != nil {
		log.Error(err, "Unable to find pinned NnfStorageProfile", "name", pinnedName)
		return nil, fmt.Errorf("Could not find pinned NnfStorageProfile %v: %w", types.NamespacedName{Name: pinnedName, Namespace: pinnedNamespace}, err)
	}

	var owner metav1.Object = workflow
	if dwArgs["command"] == "create_persistent" {
		psi, err := r.findPersistentInstance(ctx, workflow, dwArgs["name"])
		if err != nil {
			return nil, fmt.Errorf("Could not find PersistentStorageInstance %v for 'create_persistent' directive: %w", dwArgs["name"], err)
		}

		owner = psi
	}

	result, err := ctrl.CreateOrUpdate(ctx, r.Client, nnfStorage,
		func() error {
			dwsv1alpha1.AddWorkflowLabels(nnfStorage, workflow)
			dwsv1alpha1.AddOwnerLabels(nnfStorage, owner)
			addDirectiveIndexLabel(nnfStorage, index)
			addPinnedStorageProfileLabel(nnfStorage, nnfStorageProfile)

			nnfStorage.Spec.FileSystemType = dwArgs["type"]
			nnfStorage.Spec.UserID = workflow.Spec.UserID
			nnfStorage.Spec.GroupID = workflow.Spec.GroupID

			// Need to remove all of the AllocationSets in the NnfStorage object before we begin
			nnfStorage.Spec.AllocationSets = []nnfv1alpha1.NnfStorageAllocationSetSpec{}

			// Iterate the Servers data elements to pull out the allocation sets for the server
			for i := range s.Spec.AllocationSets {
				nnfAllocSet := nnfv1alpha1.NnfStorageAllocationSetSpec{}

				nnfAllocSet.Name = s.Spec.AllocationSets[i].Label
				nnfAllocSet.Capacity = s.Spec.AllocationSets[i].AllocationSize
				if dwArgs["type"] == "lustre" {
					nnfAllocSet.NnfStorageLustreSpec.TargetType = strings.ToUpper(s.Spec.AllocationSets[i].Label)
					nnfAllocSet.NnfStorageLustreSpec.BackFs = "zfs"
					nnfAllocSet.NnfStorageLustreSpec.FileSystemName = "z" + string(s.GetUID())[:7]
					if len(nnfStorageProfile.Data.LustreStorage.ExternalMGS) > 0 {
						nnfAllocSet.NnfStorageLustreSpec.ExternalMgsNid = nnfStorageProfile.Data.LustreStorage.ExternalMGS
					}
				}

				// Create Nodes for this allocation set.
				for k, storage := range s.Spec.AllocationSets[i].Storage {
					if s.Spec.AllocationSets[i].Label == "mgtmdt" && k == 0 && storage.AllocationCount > 1 {
						// If there are multiple allocations on the first MGTMDT node, split it out into two seperate
						// node entries. The first is a single allocation that will be used for the MGTMDT. The remaining
						// allocations on the node will be MDTs only.
						node := nnfv1alpha1.NnfStorageAllocationNodes{Name: storage.Name, Count: 1}
						nnfAllocSet.Nodes = append(nnfAllocSet.Nodes, node)
						node = nnfv1alpha1.NnfStorageAllocationNodes{Name: storage.Name, Count: storage.AllocationCount - 1}
						nnfAllocSet.Nodes = append(nnfAllocSet.Nodes, node)
					} else {
						node := nnfv1alpha1.NnfStorageAllocationNodes{Name: storage.Name, Count: storage.AllocationCount}
						nnfAllocSet.Nodes = append(nnfAllocSet.Nodes, node)
					}
				}

				nnfStorage.Spec.AllocationSets = append(nnfStorage.Spec.AllocationSets, nnfAllocSet)
			}

			// The Servers object owns the NnfStorage object, so it will be garbage collected when the
			// Server object is deleted.
			return ctrl.SetControllerReference(owner, nnfStorage, r.Scheme)
		})

	if err != nil {
		log.Error(err, "Failed to create or update NnfStorage", "name", nnfStorage.Name)
		return nnfStorage, fmt.Errorf("CreateOrUpdate failed for NnfStorage %v: %w", client.ObjectKeyFromObject(nnfStorage), err)
	}

	if result == controllerutil.OperationResultCreated {
		log.Info("Created NnfStorage", "name", nnfStorage.Name)
	} else if result == controllerutil.OperationResultNone {
		// no change
	} else {
		log.Info("Updated NnfStorage", "name", nnfStorage.Name)
	}

	return nnfStorage, nil
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

func (r *NnfWorkflowReconciler) setupNnfAccessForServers(ctx context.Context, storage *nnfv1alpha1.NnfStorage, workflow *dwsv1alpha1.Workflow, index int, parentDwIndex int, teardownState dwsv1alpha1.WorkflowState, log logr.Logger) (*nnfv1alpha1.NnfAccess, error) {
	access := &nnfv1alpha1.NnfAccess{
		ObjectMeta: metav1.ObjectMeta{
			Name:      indexedResourceName(workflow, parentDwIndex) + "-servers",
			Namespace: workflow.Namespace,
		},
	}

	result, err := ctrl.CreateOrUpdate(ctx, r.Client, access,
		func() error {
			dwsv1alpha1.AddWorkflowLabels(access, workflow)
			dwsv1alpha1.AddOwnerLabels(access, workflow)
			nnfv1alpha1.AddDataMovementTeardownStateLabel(access, teardownState)
			addDirectiveIndexLabel(access, index)

			access.Spec = nnfv1alpha1.NnfAccessSpec{
				DesiredState:    "mounted",
				TeardownState:   teardownState,
				Target:          "all",
				MountPath:       buildMountPath(workflow, index),
				MountPathPrefix: buildMountPath(workflow, index),

				// NNF Storage is Namespaced Name to the servers object
				StorageReference: corev1.ObjectReference{
					Kind:      reflect.TypeOf(nnfv1alpha1.NnfStorage{}).Name(),
					Name:      storage.Name,
					Namespace: storage.Namespace,
				},
			}

			return ctrl.SetControllerReference(workflow, access, r.Scheme)
		})

	if err != nil {
		return nil, fmt.Errorf("CreateOrUpdate failed for NnfAccess %v: %w", client.ObjectKeyFromObject(access), err)
	}

	if result == controllerutil.OperationResultCreated {
		log.Info("Created NnfAccess", "name", access.Name, "teardown", teardownState)
	} else if result == controllerutil.OperationResultUpdated {
		log.Info("Updated NnfAccess", "name", access.Name, "teardown", teardownState)
	}

	return access, nil
}

func (r *NnfWorkflowReconciler) getDirectiveFileSystemType(ctx context.Context, workflow *dwsv1alpha1.Workflow, index int) (string, error) {
	dwArgs, _ := dwdparse.BuildArgsMap(workflow.Spec.DWDirectives[index])
	switch dwArgs["command"] {
	case "jobdw":
		return dwArgs["type"], nil
	case "persistentdw":
		name, namespace := getStorageReferenceNameFromWorkflowActual(workflow, index)
		nnfStorage := &nnfv1alpha1.NnfStorage{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
		}

		if err := r.Get(ctx, client.ObjectKeyFromObject(nnfStorage), nnfStorage); err != nil {
			return "", fmt.Errorf("Could not get persistent NnfStorage %v to determine file system type: %w", client.ObjectKeyFromObject(nnfStorage), err)
		}

		return nnfStorage.Spec.FileSystemType, nil
	default:
		return "", fmt.Errorf("Invalid directive '%s' to get file system type", workflow.Spec.DWDirectives[index])
	}
}

func buildMountPath(workflow *dwsv1alpha1.Workflow, index int) string {
	return fmt.Sprintf("/mnt/nnf/%s-%d", workflow.UID, index)
}

func (r *NnfWorkflowReconciler) findPersistentInstance(ctx context.Context, wf *dwsv1alpha1.Workflow, psiName string) (*dwsv1alpha1.PersistentStorageInstance, error) {
	log := r.Log.WithValues("Workflow", types.NamespacedName{Name: wf.Name, Namespace: wf.Namespace})

	psi := &dwsv1alpha1.PersistentStorageInstance{}
	psiNamedNamespace := types.NamespacedName{Name: psiName, Namespace: wf.Namespace}
	err := r.Get(ctx, psiNamedNamespace, psi)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			log.Error(err, "Unable to get PersistentStorageInstance", "name", psiName, "error", err)
		}

		return nil, err
	}

	return psi, err
}

func handleWorkflowError(err error, driverStatus *dwsv1alpha1.WorkflowDriverStatus) {
	e, ok := err.(*nnfv1alpha1.WorkflowError)
	if ok {
		e.Inject(driverStatus)
	} else {
		driverStatus.Status = dwsv1alpha1.StatusError
		driverStatus.Message = "Internal error: " + err.Error()
		driverStatus.Error = err.Error()
	}
}

// Returns the directive index with the 'name' argument matching name, or -1 if not found
func findDirectiveIndexByName(workflow *dwsv1alpha1.Workflow, name string) int {
	for idx, directive := range workflow.Spec.DWDirectives {
		parameters, _ := dwdparse.BuildArgsMap(directive)
		if parameters["name"] == name {
			return idx
		}
	}
	return -1
}

// Returns the directive index matching the copy_out directive whose source field references
// the provided name argument, or -1 if not found.
func findCopyOutDirectiveIndexByName(workflow *dwsv1alpha1.Workflow, name string) int {
	for idx, directive := range workflow.Spec.DWDirectives {
		if strings.HasPrefix(directive, "#DW copy_out") {
			parameters, _ := dwdparse.BuildArgsMap(directive) // ignore error, directives are validated in proposal

			srcName, _ := splitStagingArgumentIntoNameAndPath(parameters["source"]) // i.e. source=$DW_JOB_[name]
			if srcName == name {
				return idx
			}
		}
	}

	return -1
}

// Returns a <name, path> pair for the given staging argument (typically source or destination)
// i.e. $DW_JOB_my-file-system-name/path/to/a/file into "my-file-system-name" and "/path/to/a/file"
func splitStagingArgumentIntoNameAndPath(arg string) (string, string) {

	var name = ""
	if strings.HasPrefix(arg, "$DW_JOB_") {
		name = strings.SplitN(strings.Replace(arg, "$DW_JOB_", "", 1), "/", 2)[0]
	} else if strings.HasPrefix(arg, "$DW_PERSISTENT_") {
		name = strings.SplitN(strings.Replace(arg, "$DW_PERSISTENT_", "", 1), "/", 2)[0]
	}
	var path = "/"
	if strings.Count(arg, "/") >= 1 {
		path = "/" + strings.SplitN(arg, "/", 2)[1]
	}
	return name, path

}

// indexedResourceName returns a name for a workflow child resource based on the index of the #DW directive
func indexedResourceName(workflow *dwsv1alpha1.Workflow, dwIndex int) string {
	return fmt.Sprintf("%s-%d", workflow.Name, dwIndex)
}

// Returns the <name, namespace> pair for the #DW directive at the specified index
func getStorageReferenceNameFromWorkflowActual(workflow *dwsv1alpha1.Workflow, dwdIndex int) (string, string) {

	directive := workflow.Spec.DWDirectives[dwdIndex]
	p, _ := dwdparse.BuildArgsMap(directive) // ignore error, directives were validated in proposal

	var name, namespace string

	switch p["command"] {
	case "persistentdw", "create_persistent", "destroy_persistent":
		name = p["name"]
		namespace = workflow.Namespace
	default:
		name = indexedResourceName(workflow, dwdIndex)
		namespace = workflow.Namespace
	}

	return name, namespace
}

// Returns the <name, namespace> pair for the #DW directive in the given DirectiveBreakdown
func getStorageReferenceNameFromDBD(dbd *dwsv1alpha1.DirectiveBreakdown) (string, string) {

	var name string
	namespace := dbd.Namespace

	p, _ := dwdparse.BuildArgsMap(dbd.Spec.Directive) // ignore error, directives were validated in proposal
	switch p["command"] {
	case "persistentdw", "create_persistent", "destroy_persistent":
		name = p["name"]
	default:
		name = dbd.Name
	}
	return name, namespace
}

func addDirectiveIndexLabel(object metav1.Object, index int) {
	labels := object.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}

	labels[nnfv1alpha1.DirectiveIndexLabel] = strconv.Itoa(index)
	object.SetLabels(labels)
}

func (r *NnfWorkflowReconciler) unmountNnfAccessIfNecessary(ctx context.Context, workflow *dwsv1alpha1.Workflow, index int, accessSuffix string) (*result, error) {
	if !(accessSuffix == "computes" || accessSuffix == "servers") {
		panic(fmt.Sprint("unhandled NnfAccess suffix", accessSuffix))
	}

	access := &nnfv1alpha1.NnfAccess{
		ObjectMeta: metav1.ObjectMeta{
			Name:      indexedResourceName(workflow, index) + "-" + accessSuffix,
			Namespace: workflow.Namespace,
		},
	}

	if err := r.Get(ctx, client.ObjectKeyFromObject(access), access); err != nil {
		err = fmt.Errorf("Could not get NnfAccess %v: %w", client.ObjectKeyFromObject(access), err)
		return nil, nnfv1alpha1.NewWorkflowError("Unable to find compute node mount information").WithError(err)
	}

	teardownState, found := access.Labels[nnfv1alpha1.DataMovementTeardownStateLabel]
	if !found || dwsv1alpha1.WorkflowState(teardownState) == workflow.Status.State {
		if access.Spec.DesiredState != "unmounted" {
			access.Spec.DesiredState = "unmounted"

			if err := r.Update(ctx, access); err != nil {
				if !apierrors.IsConflict(err) {
					err = fmt.Errorf("Could not update NnfAccess %v: %w", client.ObjectKeyFromObject(access), err)
					return nil, nnfv1alpha1.NewWorkflowError("Unable to request compute node unmount").WithError(err)
				}

				return Requeue("conflict").withObject(access), nil
			}
		}
	}

	return nil, nil
}

// Wait on the NnfAccesses for this workflow-index to reach the provided state.
func (r *NnfWorkflowReconciler) waitForNnfAccessStateAndReady(ctx context.Context, workflow *dwsv1alpha1.Workflow, index int, state string) (*result, error) {

	accessSuffixes := []string{"-computes"}

	// Check if we should also wait on the NnfAccess for the servers
	fsType, err := r.getDirectiveFileSystemType(ctx, workflow, index)
	if err != nil {
		return nil, nnfv1alpha1.NewWorkflowError("Unable to determine directive file system type").WithError(err)
	}

	if fsType == "gfs2" || fsType == "lustre" {
		accessSuffixes = append(accessSuffixes, "-servers")
	}

	for _, suffix := range accessSuffixes {

		access := &nnfv1alpha1.NnfAccess{
			ObjectMeta: metav1.ObjectMeta{
				Name:      indexedResourceName(workflow, index) + suffix,
				Namespace: workflow.Namespace,
			},
		}

		if err := r.Get(ctx, client.ObjectKeyFromObject(access), access); err != nil {
			err = fmt.Errorf("Could not get NnfAccess %s: %w", client.ObjectKeyFromObject(access).String(), err)
			return nil, nnfv1alpha1.NewWorkflowError("Could not access file system on nodes").WithError(err)
		}

		if access.Status.Error != nil {
			err = fmt.Errorf("Error on NnfAccess %s: %w", client.ObjectKeyFromObject(access).String(), access.Status.Error)
			return nil, nnfv1alpha1.NewWorkflowError("Could not access file system on nodes").WithError(err)
		}

		if state == "mounted" {
			// When mounting, we must always go ready regardless of workflow state
			if access.Status.State != "mounted" || !access.Status.Ready {
				return Requeue("pending mount").withObject(access), nil
			}
		} else {
			// When unmounting, we are conditionally dependent on the workflow state matching the
			// state of the teardown label, if found.
			teardownState, found := access.Labels[nnfv1alpha1.DataMovementTeardownStateLabel]
			if !found || dwsv1alpha1.WorkflowState(teardownState) == workflow.Status.State {
				if access.Status.State != "unmounted" || !access.Status.Ready {
					return Requeue("pending unmount").withObject(access), nil
				}

			}
		}
	}

	return nil, nil
}

func (r *NnfWorkflowReconciler) addPersistentStorageReference(ctx context.Context, workflow *dwsv1alpha1.Workflow, index int) error {
	dwArgs, _ := dwdparse.BuildArgsMap(workflow.Spec.DWDirectives[index])

	persistentStorage, err := r.findPersistentInstance(ctx, workflow, dwArgs["name"])
	if err != nil {
		return err
	}

	if persistentStorage.Status.State != dwsv1alpha1.PSIStateActive {
		return fmt.Errorf("PersistentStorage is not active")
	}

	// Add a consumer reference to the persistent storage for this directive
	reference := corev1.ObjectReference{
		Name:      indexedResourceName(workflow, index),
		Namespace: workflow.Namespace,
	}

	found := false
	for _, existingReference := range persistentStorage.Spec.ConsumerReferences {
		if existingReference == reference {
			found = true
		}
	}

	if !found {
		persistentStorage.Spec.ConsumerReferences = append(persistentStorage.Spec.ConsumerReferences, reference)

		return r.Update(ctx, persistentStorage)
	}

	return nil
}

func (r *NnfWorkflowReconciler) removePersistentStorageReference(ctx context.Context, workflow *dwsv1alpha1.Workflow, index int) error {
	dwArgs, _ := dwdparse.BuildArgsMap(workflow.Spec.DWDirectives[index])

	persistentStorage, err := r.findPersistentInstance(ctx, workflow, dwArgs["name"])
	if err != nil {
		return client.IgnoreNotFound(err)
	}

	// remove the consumer reference on the persistent storage for this directive
	reference := corev1.ObjectReference{
		Name:      indexedResourceName(workflow, index),
		Namespace: workflow.Namespace,
	}

	for i, existingReference := range persistentStorage.Spec.ConsumerReferences {
		if existingReference == reference {
			persistentStorage.Spec.ConsumerReferences = append(persistentStorage.Spec.ConsumerReferences[:i], persistentStorage.Spec.ConsumerReferences[i+1:]...)
			return r.Update(ctx, persistentStorage)
		}
	}

	return nil
}

func (r *NnfWorkflowReconciler) removeAllPersistentStorageReferences(ctx context.Context, workflow *dwsv1alpha1.Workflow) error {
	for i, directive := range workflow.Spec.DWDirectives {
		dwArgs, _ := dwdparse.BuildArgsMap(directive)
		if dwArgs["command"] == "persistentdw" {
			err := r.removePersistentStorageReference(ctx, workflow, i)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (r *NnfWorkflowReconciler) createOrUpdateContainerDaemonSetIfNecessary(ctx context.Context, workflow *dwsv1alpha1.Workflow) error {
	log := log.FromContext(ctx)

	// TODO: check the directive or the workflow itself to determine if containers are requested
	// I would imagine a change to dws (workflow type) would be necessary here but perhaps we can
	// skirt around it with the directivebreakdown?

	// TODO move this to helpers and use pinned profiles like storage profiles
	// TODO: name is hardcoded
	container := &nnfv1alpha1.NnfContainerProfile{}
	if err := r.Client.Get(ctx, types.NamespacedName{Namespace: "default", Name: "nnfcontainerprofile-sample"}, container); err != nil {
		return err
	}

	ds := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nnfcontainerprofile-sample-12345",
			Namespace: "nnf-system",
		},
	}

	mutateFn := func() error {
		podTemplateSpec := container.Data.Template.DeepCopy()
		// podTemplateSpec.Labels = manager.Spec.Selector.DeepCopy().MatchLabels

		// if podTemplateSpec.Labels == nil {
		// 	podTemplateSpec.Labels = make(map[string]string)
		// }
		// podTemplateSpec.Labels[dmv1alpha1.DataMovementWorkerLabel] = "true"
		podTemplateSpec.Labels = map[string]string{
			"cray.nnf.node": "true",
		}

		podSpec := &podTemplateSpec.Spec
		// podSpec.NodeSelector = manager.Spec.Selector.MatchLabels
		podSpec.NodeSelector = map[string]string{"cray.nnf.node": "true"}
		// podSpec.Subdomain = serviceName

		// setupSSHAuthVolumes(manager, podSpec)
		// setupLustreVolumes(ctx, manager, podSpec, filesystems.Items)

		ds.Spec = appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{
				"cray.nnf.node": "true",
			}},
			Template: *podTemplateSpec,
		}
		// ds.Spec = appsv1.DaemonSetSpec{
		// 	Selector: &manager.Spec.Selector,
		// 	Template: *podTemplateSpec,
		// }

		dwsv1alpha1.InheritParentLabels(ds, workflow)
		dwsv1alpha1.AddOwnerLabels(ds, workflow)
		// labels := ds.GetLabels()
		// labels[nnfv1alpha1.AllocationSetLabel] = allocationSet.Name
		// nnfNodeStorage.SetLabels(labels)

		// TODO: add Volumes to DS, append if customer specifies; check for conflicts
		vol := corev1.Volume{
			Name: "foo-local-storage",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/path/to/thing"}}}
		podTemplateSpec.Spec.Volumes = append(podTemplateSpec.Spec.Volumes, vol)
		// TODO: get mounts from workflow

		return nil
	}

	result, err := ctrl.CreateOrUpdate(ctx, r.Client, ds, mutateFn)
	if err != nil {
		return err
	}

	if result == controllerutil.OperationResultCreated {
		log.Info("Created DaemonSet", "object", client.ObjectKeyFromObject(ds).String())
	} else if result == controllerutil.OperationResultUpdated {
		log.Info("Updated DaemonSet", "object", client.ObjectKeyFromObject(ds).String())
	}

	return nil
}
