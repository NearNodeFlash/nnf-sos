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
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	dwsv1alpha1 "github.com/HewlettPackard/dws/api/v1alpha1"
	"github.com/HewlettPackard/dws/utils/dwdparse"
	lusv1alpha1 "github.com/NearNodeFlash/lustre-fs-operator/api/v1alpha1"
	nnfv1alpha1 "github.com/NearNodeFlash/nnf-sos/api/v1alpha1"

	"github.com/go-logr/logr"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// NNF Workflow stages all return a `result` structure when successful that describes
// the result of an operation at each stage of the workflow.
type result struct {
	ctrl.Result
	reason       string
	object       client.Object
	deleteStatus *dwsv1alpha1.DeleteStatus
}

// This struct contains all the necessary information for mounting container storages
type nnfContainerVolume struct {
	name           string
	command        string
	directiveName  string
	directiveIndex int
	mountPath      string
	envVarName     string
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

	var createPersistentCount, deletePersistentCount, directiveCount, containerCount int
	for index, directive := range wf.Spec.DWDirectives {

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
			containerCount++

			if err := r.validateContainerDirective(ctx, wf, index); err != nil {
				return nnfv1alpha1.NewWorkflowError("Could not validate container directive: " + directive).WithFatal().WithError(err)
			}
		}
	}

	if directiveCount > 1 {
		// Ensure create_persistent or destroy_persistent are singletons in the workflow
		if createPersistentCount+deletePersistentCount > 0 {
			return nnfv1alpha1.NewWorkflowError("Only a single create_persistent or destroy_persistent directive is allowed per workflow").WithFatal()
		}

		// Only allow 1 container directive (for now)
		if containerCount > 1 {
			return nnfv1alpha1.NewWorkflowError("Only a single container directive is supported per workflow").WithFatal()
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
			index := findDirectiveIndexByName(wf, name, "jobdw")
			if index == -1 {
				return nnfv1alpha1.NewWorkflowError(fmt.Sprintf("Job storage instance '%s' not found", name)).WithFatal()
			}

			args, err := dwdparse.BuildArgsMap(wf.Spec.DWDirectives[index])
			if err != nil {
				return nnfv1alpha1.NewWorkflowError("Invalid DW directive: " + wf.Spec.DWDirectives[index]).WithFatal()
			}

			fsType, exists := args["type"]
			if !exists {
				return nnfv1alpha1.NewWorkflowError("Invalid DW directive match for staging argument")
			}

			if fsType == "raw" {
				return nnfv1alpha1.NewWorkflowError("Data movement can not be used with raw allocations").WithFatal()
			}
		} else if strings.HasPrefix(arg, "$DW_PERSISTENT_") {
			if err := r.validatePersistentInstanceForStaging(ctx, name, wf.Namespace); err != nil {
				return nnfv1alpha1.NewWorkflowError(fmt.Sprintf("Persistent storage instance '%s' not found", name)).WithFatal()
			}
			if findDirectiveIndexByName(wf, name, "persistentdw") == -1 {
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
func (r *NnfWorkflowReconciler) validateContainerDirective(ctx context.Context, workflow *dwsv1alpha1.Workflow, index int) error {
	args, err := dwdparse.BuildArgsMap(workflow.Spec.DWDirectives[index])
	if err != nil {
		return nnfv1alpha1.NewWorkflowError("invalid DW directive: " + workflow.Spec.DWDirectives[index]).WithFatal()
	}

	// Ensure the supplied profile exists or use the default
	profile, err := r.findContainerProfile(ctx, workflow, index)
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

	checkContainerFs := func(idx int) error {
		getDirectiveFsType := func(idx int) (string, error) {
			args, _ := dwdparse.BuildArgsMap(workflow.Spec.DWDirectives[idx])

			// For persistent instance
			if args["command"] == "persistentdw" {
				psi, err := r.getPersistentStorageInstance(ctx, args["name"], workflow.Namespace)
				if err != nil {
					return "", fmt.Errorf("could not retrieve persistent instance '%s' for container directive: %s", args["name"], err)
				}

				return psi.Spec.FsType, nil
			}

			return args["type"], nil
		}

		t, err := getDirectiveFsType(idx)
		if err != nil {
			return err
		}

		if strings.ToLower(t) != "lustre" && strings.ToLower(t) != "gfs2" {
			return fmt.Errorf("unsupported container filesystem: %s", t)
		}

		return nil
	}

	// Find the wildcard DW_JOB and DW_PERSISTENT arguments. Verify that they exist in the
	// preceding directives and in the container profile. For persistent arguments, ensure there
	// is a persistent storage instance.
	var suppliedStorageArguments []string // use this list later to validate non-optional storages
	for arg, storageName := range args {
		switch arg {
		case "command", "name", "profile":
		default:
			if strings.HasPrefix(arg, "DW_JOB_") {
				idx := findDirectiveIndexByName(workflow, storageName, "jobdw")
				if idx == -1 {
					return nnfv1alpha1.NewWorkflowError(fmt.Sprintf("jobdw directive mentioning '%s' not found", storageName)).WithFatal()
				}
				if err := checkContainerFs(idx); err != nil {
					return nnfv1alpha1.NewWorkflowError(err.Error()).WithFatal()
				}
				if err := checkStorageIsInProfile(arg); err != nil {
					return nnfv1alpha1.NewWorkflowError(err.Error()).WithFatal()
				}
				suppliedStorageArguments = append(suppliedStorageArguments, arg)
			} else if strings.HasPrefix(arg, "DW_PERSISTENT_") {
				if err := r.validatePersistentInstanceForStaging(ctx, storageName, workflow.Namespace); err != nil {
					return nnfv1alpha1.NewWorkflowError(fmt.Sprintf("persistent storage instance '%s' not found", storageName)).WithFatal()
				}
				idx := findDirectiveIndexByName(workflow, storageName, "persistentdw")
				if idx == -1 {
					return nnfv1alpha1.NewWorkflowError(fmt.Sprintf("persistentdw directive mentioning '%s' not found", storageName)).WithFatal()
				}
				if err := checkContainerFs(idx); err != nil {
					return nnfv1alpha1.NewWorkflowError(err.Error()).WithFatal()
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
func (r *NnfWorkflowReconciler) validatePersistentInstanceForStaging(ctx context.Context, name string, namespace string) error {
	psi, err := r.getPersistentStorageInstance(ctx, name, namespace)
	if err != nil {
		return err
	}

	if psi.Spec.FsType == "raw" {
		return nnfv1alpha1.NewWorkflowError("Data movement can not be used with raw allocations").WithFatal()
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

	psi, err := r.getPersistentStorageInstance(ctx, args["name"], wf.Namespace)
	if err != nil {
		return err
	}

	if !psi.DeletionTimestamp.IsZero() {
		return nnfv1alpha1.NewWorkflowError("Persistent storage instance " + args["name"] + " is deleting").WithFatal()
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
	//     DW_JOB_foo-local-storage=my-gfs2
	//     DW_PERSISTENT_foo-persistent-storage=some-lustre

	// #DW commands that require a dwDirectiveBreakdown
	breakDownCommands := []string{
		"jobdw",
		"create_persistent",
		"persistentdw",
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

		if !found {
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
				UserID:          workflow.Spec.UserID,
				GroupID:         workflow.Spec.GroupID,
				MountPath:       buildMountPath(workflow, parentDwIndex),
				MountPathPrefix: buildMountPath(workflow, parentDwIndex),

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
func findDirectiveIndexByName(workflow *dwsv1alpha1.Workflow, name string, command string) int {
	for idx, directive := range workflow.Spec.DWDirectives {
		parameters, _ := dwdparse.BuildArgsMap(directive)
		if parameters["name"] == name && parameters["command"] == command {
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

// Returns the directive index matching the container directive whose storage field(s) reference
// the provided name argument, or -1 if not found.
func findContainerDirectiveIndexByName(workflow *dwsv1alpha1.Workflow, name string) int {
	for idx, directive := range workflow.Spec.DWDirectives {
		parameters, _ := dwdparse.BuildArgsMap(directive) // ignore error, directives are validated in proposal
		if parameters["command"] == "container" {
			for k, v := range parameters {
				if strings.HasPrefix(k, "DW_JOB_") || strings.HasPrefix(k, "DW_PERSISTENT_") {
					if v == name {
						return idx
					}
				}
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

	// Check if the NnfAccess exists. If it doesn't, then we don't need to unmount anything. If we didn't
	// find it in the cache and it really does exist, we'll eventually get an event from the API server and
	// do the unmount then.
	if err := r.Get(ctx, client.ObjectKeyFromObject(access), access); err != nil {
		return nil, client.IgnoreNotFound(err)
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

		if access.Status.State != "unmounted" || !access.Status.Ready {
			return Requeue("pending unmount").withObject(access), nil
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

func (r *NnfWorkflowReconciler) createContainerService(ctx context.Context, workflow *dwsv1alpha1.Workflow) error {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      workflow.Name,
			Namespace: workflow.Namespace,
		},
	}

	service.Spec.Selector = map[string]string{
		nnfv1alpha1.ContainerLabel: workflow.Name,
	}

	service.Spec.ClusterIP = corev1.ClusterIPNone

	if err := ctrl.SetControllerReference(workflow, service, r.Scheme); err != nil {
		return fmt.Errorf("setting Service controller reference failed for '%s': %w", service.Name, err)
	}

	err := r.Create(ctx, service)
	if err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return err
		}
	}

	return nil
}

func (r *NnfWorkflowReconciler) createContainerJobs(ctx context.Context, workflow *dwsv1alpha1.Workflow, dwArgs map[string]string, index int) (*result, error) {
	profile, err := r.getContainerProfile(ctx, workflow, index)
	if err != nil {
		return nil, err
	}

	// Create one master job that will be used for all the jobs on all the NnfNodes. Most of the Job's Data will be the same.
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: workflow.Namespace,
		},
	}

	// Apply Job Labels/Owners
	dwsv1alpha1.InheritParentLabels(job, workflow)
	dwsv1alpha1.AddOwnerLabels(job, workflow)
	dwsv1alpha1.AddWorkflowLabels(job, workflow)

	labels := job.GetLabels()
	labels[nnfv1alpha1.ContainerLabel] = workflow.Name
	labels[nnfv1alpha1.PinnedContainerProfileLabelName] = profile.GetName()
	labels[nnfv1alpha1.PinnedContainerProfileLabelNameSpace] = profile.GetNamespace()
	labels[nnfv1alpha1.DirectiveIndexLabel] = strconv.Itoa(index)
	job.SetLabels(labels)

	if err := ctrl.SetControllerReference(workflow, job, r.Scheme); err != nil {
		return nil, nnfv1alpha1.NewWorkflowErrorf("setting Job controller reference failed for '%s':", job.Name).WithError(err)
	}

	// This defaults to 6 and is the maximum number of pod retries before considering the
	// job failed. I don't believe this can be turned off.
	// See the comments below regarding the restartPolicy and also see
	// https://github.com/NearNodeFlash/NearNodeFlash.github.io/pull/26#discussion_r1089460308.
	job.Spec.BackoffLimit = &profile.Data.RetryLimit

	// Copy the container template from the profile
	profile.Data.Template.DeepCopyInto(&job.Spec.Template)

	// Use the same labels as the job for the pods
	job.Spec.Template.Labels = job.DeepCopy().Labels

	podSpec := &job.Spec.Template.Spec

	// We want to keep the restart policy to Never. This way, any pods that failed are
	// kept around for inspection. The job attempts to retry pods until the number of
	// completions are hit or the number of max retries (BackoffLimit) have been hit.
	// A retry with a restart policy of Never will not restart the pod, but spin up a new one
	// with a new IP (DNS will have to be used to reach pods). A retry is not the same as a restart.
	// If we set this to OnFailure, the pods will truly restart but we will lose any log history
	// outside of (kubectl logs --previous).
	// See https://github.com/NearNodeFlash/NearNodeFlash.github.io/pull/26#discussion_r1089460308
	podSpec.RestartPolicy = corev1.RestartPolicyNever

	podSpec.Subdomain = workflow.Name // service name == workflow name

	podSpec.Tolerations = []corev1.Toleration{
		{
			Effect:   "NoSchedule",
			Key:      "cray.nnf.node",
			Operator: "Equal",
			Value:    "true",
		},
	}

	// Add non-root permissions from the workflow's user/group ID
	if podSpec.SecurityContext == nil {
		podSpec.SecurityContext = &corev1.PodSecurityContext{}
	}
	uid := int64(workflow.Spec.UserID)
	gid := int64(workflow.Spec.GroupID)
	podSpec.SecurityContext.RunAsUser = &uid
	podSpec.SecurityContext.RunAsGroup = &gid
	nonRoot := false
	if uid != 0 {
		nonRoot = true
	}
	podSpec.SecurityContext.RunAsNonRoot = &nonRoot

	// Get the volumes to mount into the containers
	volumes, result, err := r.getContainerVolumes(ctx, workflow, dwArgs)
	if err != nil {
		return nil, nnfv1alpha1.NewWorkflowErrorf("could not determine the list of volumes need to create container job %s:", job.Name).WithError(err).WithFatal()
	}
	if result != nil {
		return result, nil
	}

	// Add Volumes/VolumeMounts
	for _, vol := range volumes {
		// Volumes
		hostPathType := corev1.HostPathDirectory
		podSpec.Volumes = append(podSpec.Volumes, corev1.Volume{
			Name: vol.name,
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: vol.mountPath,
					Type: &hostPathType,
				},
			},
		})

		// Add VolumeMounts and Volume environment variables for all containers
		for idx := range podSpec.Containers {
			container := &podSpec.Containers[idx]

			container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{
				Name:      vol.name,
				MountPath: vol.mountPath,
			})

			container.Env = append(container.Env, corev1.EnvVar{
				Name:  vol.envVarName,
				Value: vol.mountPath,
			})
		}
	}

	// Get the targeted NNF nodes for the container jobs
	nnfNodes, err := r.getNnfNodesFromComputes(ctx, workflow)
	if err != nil {
		return nil, nnfv1alpha1.NewWorkflowError("error obtaining the target NNF nodes for containers:").WithError(err).WithFatal()
	}

	// Add in non-volume environment variables for all containers
	for idx := range podSpec.Containers {
		container := &podSpec.Containers[idx]

		container.Env = append(container.Env,
			corev1.EnvVar{Name: "NNF_CONTAINER_SUBDOMAIN", Value: podSpec.Subdomain},
			corev1.EnvVar{Name: "NNF_CONTAINER_DOMAIN", Value: workflow.Namespace + ".svc.cluster.local"},
			corev1.EnvVar{Name: "NNF_CONTAINER_HOSTNAMES", Value: strings.Join(nnfNodes, " ")})
	}

	// Finally, create a job for each nnfNode. Only the name, hostname, and node selector is different for each node
	for _, nnfNode := range nnfNodes {
		job.ObjectMeta.Name = workflow.Name + "-" + nnfNode
		podSpec.Hostname = nnfNode

		// In our case, the target is only 1 node for the job, so a restartPolicy of Never
		// is ok because any retry (i.e. new pod) will land on the same node.
		podSpec.NodeSelector = map[string]string{"kubernetes.io/hostname": nnfNode}

		newJob := &batchv1.Job{}
		job.DeepCopyInto(newJob)

		err = r.Create(ctx, newJob)
		if err != nil {
			if !apierrors.IsAlreadyExists(err) {
				return nil, err
			}
		}
	}

	return nil, nil
}

// Retrieve the computes for the workflow and find their local nnf nodes
func (r *NnfWorkflowReconciler) getNnfNodesFromComputes(ctx context.Context, workflow *dwsv1alpha1.Workflow) ([]string, error) {

	var nnfNodes []string
	var computeNodes []string

	// Get the compute resources
	computes := dwsv1alpha1.Computes{
		ObjectMeta: metav1.ObjectMeta{
			Name:      workflow.Name,
			Namespace: workflow.Namespace,
		},
	}
	if err := r.Get(ctx, client.ObjectKeyFromObject(&computes), &computes); err != nil {
		return nnfNodes, nnfv1alpha1.NewWorkflowError("could not find Computes resource for workflow")
	}

	// Build the list of computes
	for _, c := range computes.Data {
		computeNodes = append(computeNodes, c.Name)
	}
	if len(computeNodes) == 0 {
		return computeNodes, nnfv1alpha1.NewWorkflowError("the Computes resources does not specify any compute nodes")
	}

	systemConfig := &dwsv1alpha1.SystemConfiguration{}
	if err := r.Get(ctx, types.NamespacedName{Name: "default", Namespace: corev1.NamespaceDefault}, systemConfig); err != nil {
		return nnfNodes, nnfv1alpha1.NewWorkflowError("could not get system configuration")
	}

	// The SystemConfiguration is organized by rabbit. Make a map of computes:rabbit for easy lookup.
	computeMap := make(map[string]string)
	for _, storageNode := range systemConfig.Spec.StorageNodes {
		if storageNode.Type != "Rabbit" {
			continue
		}
		for _, ca := range storageNode.ComputesAccess {
			computeMap[ca.Name] = storageNode.Name
		}
	}

	// For each supplied compute, use the map to lookup its local Rabbit
	for _, c := range computeNodes {
		nnfNode, found := computeMap[c]
		if !found {
			return nnfNodes, nnfv1alpha1.NewWorkflowErrorf("supplied compute node '%s' not found in SystemConfiguration", c)
		}
		nnfNodes = append(nnfNodes, nnfNode)
	}

	return nnfNodes, nil
}

func (r *NnfWorkflowReconciler) waitForContainersToStart(ctx context.Context, workflow *dwsv1alpha1.Workflow, index int) (*result, error) {

	jobList, err := r.getContainerJobs(ctx, workflow, index)
	if err != nil {
		return nil, err
	}

	// Jobs may not be queryable yet, so requeue
	if len(jobList.Items) < 1 {
		return Requeue(fmt.Sprintf("pending job creation for workflow '%s', index: %d", workflow.Name, index)).after(2 * time.Second), nil
	}

	for _, job := range jobList.Items {
		// If we have any conditions, the job already finished
		if len(job.Status.Conditions) > 0 {
			continue
		}

		// Ready should be non-zero to indicate the a pod is running for the job
		if job.Status.Ready == nil || *job.Status.Ready < 1 {
			return Requeue(fmt.Sprintf("pending container start for job '%s'", job.Name)).after(2 * time.Second), nil
		}
	}

	return nil, nil
}

func (r *NnfWorkflowReconciler) waitForContainersToFinish(ctx context.Context, workflow *dwsv1alpha1.Workflow, index int) (*result, error) {
	jobList, err := r.getContainerJobs(ctx, workflow, index)
	if err != nil {
		return nil, err
	}

	if len(jobList.Items) < 1 {
		return nil, nnfv1alpha1.NewWorkflowErrorf("waitForContainersToFinish: no container jobs found for workflow '%s', index: %d", workflow.Name, index)
	}

	// Retrieve the profile to extract the PostRun timeout
	profile, err := r.getContainerProfile(ctx, workflow, index)
	if err != nil {
		return nil, err
	}
	timeout := time.Duration(profile.Data.PostRunTimeoutSeconds) * time.Second

	// Ensure all the jobs are done running before we check the conditions.
	for _, job := range jobList.Items {
		// Jobs will have conditions when finished
		if len(job.Status.Conditions) <= 0 {

			// If desired, set the ActiveDeadline on the job to kill pods. Use the job's creation
			// timestamp to determine how long the job/pod has been running at this point. Then, add
			// the desired timeout to that value. k8s Job's ActiveDeadLineSeconds will then
			// terminate the pods once the deadline is hit.
			if timeout > 0 && job.Spec.ActiveDeadlineSeconds == nil {
				deadline := int64((metav1.Now().Sub(job.CreationTimestamp.Time) + timeout).Seconds())

				// Update the job with the deadline
				err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
					j := &batchv1.Job{ObjectMeta: metav1.ObjectMeta{Name: job.Name, Namespace: job.Namespace}}
					if err := r.Get(ctx, client.ObjectKeyFromObject(j), j); err != nil {
						return client.IgnoreNotFound(err)
					}

					j.Spec.ActiveDeadlineSeconds = &deadline
					return r.Update(ctx, j)
				})

				if err != nil {
					return nil, nnfv1alpha1.NewWorkflowErrorf("error updating job '%s' activeDeadlineSeconds:", job.Name)
				}
			}

			return Requeue("pending container finish").after(2 * time.Second).withObject(&job), nil
		}
	}

	return nil, nil
}

func (r *NnfWorkflowReconciler) checkContainersResults(ctx context.Context, workflow *dwsv1alpha1.Workflow, index int) (*result, error) {
	jobList, err := r.getContainerJobs(ctx, workflow, index)
	if err != nil {
		return nil, err
	}

	if len(jobList.Items) < 1 {
		return nil, nnfv1alpha1.NewWorkflowErrorf("checkContainersResults: no container jobs found for workflow '%s', index: %d", workflow.Name, index)
	}

	for _, job := range jobList.Items {
		for _, condition := range job.Status.Conditions {
			if condition.Type != batchv1.JobComplete {
				return nil, nnfv1alpha1.NewWorkflowErrorf("container job %s (%s): %s", condition.Type, condition.Reason, condition.Message)
			}
		}
	}

	return nil, nil
}

func (r *NnfWorkflowReconciler) getContainerJobs(ctx context.Context, workflow *dwsv1alpha1.Workflow, index int) (*batchv1.JobList, error) {
	// Get the jobs for this workflow and directive index
	matchLabels := dwsv1alpha1.MatchingWorkflow(workflow)
	matchLabels[nnfv1alpha1.DirectiveIndexLabel] = strconv.Itoa(index)

	jobList := &batchv1.JobList{}
	if err := r.List(ctx, jobList, matchLabels); err != nil {
		return nil, nnfv1alpha1.NewWorkflowErrorf("could not retrieve Jobs for index %d", index).WithError(err)
	}

	return jobList, nil
}

func (r *NnfWorkflowReconciler) getContainerProfile(ctx context.Context, workflow *dwsv1alpha1.Workflow, index int) (*nnfv1alpha1.NnfContainerProfile, error) {
	profile, err := r.findPinnedContainerProfile(ctx, workflow, index)
	if err != nil {
		return nil, err
	}

	if profile == nil {
		return nil, nnfv1alpha1.NewWorkflowErrorf("container profile '%s' not found", indexedResourceName(workflow, index)).WithFatal()
	}

	if !profile.Data.Pinned {
		return nil, nnfv1alpha1.NewWorkflowErrorf("expected pinned container profile '%s'", indexedResourceName(workflow, index)).WithFatal()
	}

	return profile, nil
}

func (r *NnfWorkflowReconciler) findPinnedContainerProfile(ctx context.Context, workflow *dwsv1alpha1.Workflow, index int) (*nnfv1alpha1.NnfContainerProfile, error) {
	profile := &nnfv1alpha1.NnfContainerProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      indexedResourceName(workflow, index),
			Namespace: workflow.Namespace,
		},
	}

	if err := r.Get(ctx, client.ObjectKeyFromObject(profile), profile); err != nil {
		return nil, err
	}

	return profile, nil
}

func (r *NnfWorkflowReconciler) findContainerProfile(ctx context.Context, workflow *dwsv1alpha1.Workflow, index int) (*nnfv1alpha1.NnfContainerProfile, error) {
	args, err := dwdparse.BuildArgsMap(workflow.Spec.DWDirectives[index])
	if err != nil {
		return nil, err
	}

	name, found := args["profile"]
	if !found {
		return nil, fmt.Errorf("container directive '%s' has no profile key", workflow.Spec.DWDirectives[index])
	}

	profile := &nnfv1alpha1.NnfContainerProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: os.Getenv("NNF_CONTAINER_PROFILE_NAMESPACE"),
		},
	}

	if err := r.Get(ctx, client.ObjectKeyFromObject(profile), profile); err != nil {
		return nil, err
	}

	return profile, nil
}

func (r *NnfWorkflowReconciler) createPinnedContainerProfileIfNecessary(ctx context.Context, workflow *dwsv1alpha1.Workflow, index int) error {
	profile, err := r.findPinnedContainerProfile(ctx, workflow, index)
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	if profile != nil {
		if !profile.Data.Pinned {
			return fmt.Errorf("expected pinned container profile, but it was not pinned: %s", profile.Name)
		}

		return nil
	}

	profile, err = r.findContainerProfile(ctx, workflow, index)
	if err != nil {
		return err
	}

	pinnedProfile := &nnfv1alpha1.NnfContainerProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      indexedResourceName(workflow, index),
			Namespace: workflow.Namespace,
		},
	}

	profile.Data.DeepCopyInto(&pinnedProfile.Data)

	pinnedProfile.Data.Pinned = true

	dwsv1alpha1.AddOwnerLabels(pinnedProfile, workflow)

	if err := controllerutil.SetControllerReference(workflow, pinnedProfile, r.Scheme); err != nil {
		r.Log.Error(err, "failed to set controller reference on profile", "profile", pinnedProfile)
		return fmt.Errorf("failed to set controller reference on profile %s", client.ObjectKeyFromObject(pinnedProfile))
	}

	r.Log.Info("Creating pinned container profile", "resource", client.ObjectKeyFromObject(pinnedProfile))
	if err := r.Create(ctx, pinnedProfile); err != nil {
		return err
	}

	return nil
}

// Create a list of volumes to be mounted inside of the containers based on the DW_JOB/DW_PERSISTENT arguments
func (r *NnfWorkflowReconciler) getContainerVolumes(ctx context.Context, workflow *dwsv1alpha1.Workflow, dwArgs map[string]string) ([]nnfContainerVolume, *result, error) {
	volumes := []nnfContainerVolume{}

	// TODO: ssh is necessary for mpi see setupSSHAuthVolumes(manager, podSpec) in nnf-dm

	for arg, val := range dwArgs {
		volName, cmd := "", ""

		// Find any DW_(JOB|PERSISTENT) arguments
		if strings.HasPrefix(arg, "DW_JOB_") {
			volName = strings.TrimPrefix(arg, "DW_JOB_")
			cmd = "jobdw"
		} else if strings.HasPrefix(arg, "DW_PERSISTENT_") {
			volName = strings.TrimPrefix(arg, "DW_PERSISTENT_")
			cmd = "persistentdw"
		} else {
			continue
		}

		vol := nnfContainerVolume{
			name:           volName,
			command:        cmd,
			directiveName:  val,
			directiveIndex: -1,
			envVarName:     strings.ReplaceAll(arg, "-", "_"), // env vars can't have hyphens
		}

		// Find the directive index for the given name so we can retrieve its NnfAccess
		vol.directiveIndex = findDirectiveIndexByName(workflow, vol.directiveName, vol.command)
		if vol.directiveIndex < 0 {
			return nil, nil, nnfv1alpha1.NewWorkflowErrorf("could not retrieve the directive breakdown for '%s'", vol.directiveName)
		}

		nnfAccess := &nnfv1alpha1.NnfAccess{
			ObjectMeta: metav1.ObjectMeta{
				Name:      workflow.Name + "-" + strconv.Itoa(vol.directiveIndex) + "-servers",
				Namespace: workflow.Namespace,
			},
		}
		if err := r.Get(ctx, client.ObjectKeyFromObject(nnfAccess), nnfAccess); err != nil {
			return nil, nil, nnfv1alpha1.NewWorkflowErrorf("could not retrieve the NnfAccess '%s'", nnfAccess.Name)
		}

		if !nnfAccess.Status.Ready {
			return nil, Requeue(fmt.Sprintf("NnfAccess '%s' is not ready to be mounted into container", nnfAccess.Name)).after(2 * time.Second), nil
		}

		vol.mountPath = nnfAccess.Spec.MountPath
		volumes = append(volumes, vol)
	}

	return volumes, nil, nil
}
