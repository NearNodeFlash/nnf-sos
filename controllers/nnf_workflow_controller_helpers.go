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

	dwsv1alpha2 "github.com/HewlettPackard/dws/api/v1alpha2"
	"github.com/HewlettPackard/dws/utils/dwdparse"
	lusv1beta1 "github.com/NearNodeFlash/lustre-fs-operator/api/v1beta1"
	nnfv1alpha1 "github.com/NearNodeFlash/nnf-sos/api/v1alpha1"

	"github.com/go-logr/logr"
	mpicommonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	mpiv2beta1 "github.com/kubeflow/mpi-operator/pkg/apis/kubeflow/v2beta1"
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
	deleteStatus *dwsv1alpha2.DeleteStatus
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

func (r *result) withDeleteStatus(d dwsv1alpha2.DeleteStatus) *result {
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
func (r *NnfWorkflowReconciler) validateWorkflow(ctx context.Context, wf *dwsv1alpha2.Workflow) error {

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
func (r *NnfWorkflowReconciler) validateStagingDirective(ctx context.Context, wf *dwsv1alpha2.Workflow, directive string) error {
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
func (r *NnfWorkflowReconciler) validateContainerDirective(ctx context.Context, workflow *dwsv1alpha2.Workflow, index int) error {
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
func (r *NnfWorkflowReconciler) validatePersistentInstanceDirective(ctx context.Context, wf *dwsv1alpha2.Workflow, directive string) error {
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
func (r *NnfWorkflowReconciler) getPersistentStorageInstance(ctx context.Context, name string, namespace string) (*dwsv1alpha2.PersistentStorageInstance, error) {
	psi := &dwsv1alpha2.PersistentStorageInstance{
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
func (r *NnfWorkflowReconciler) generateDirectiveBreakdown(ctx context.Context, dwIndex int, workflow *dwsv1alpha2.Workflow, log logr.Logger) (*dwsv1alpha2.DirectiveBreakdown, error) {

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
	//     DW_JOB_foo_local_storage=my-gfs2
	//     DW_PERSISTENT_foo_persistent_storage=some-lustre

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
			directiveBreakdown := &dwsv1alpha2.DirectiveBreakdown{
				ObjectMeta: metav1.ObjectMeta{
					Name:      dwdName,
					Namespace: workflow.Namespace,
				},
			}

			result, err := ctrl.CreateOrUpdate(ctx, r.Client, directiveBreakdown,
				// Mutate function to fill in a directiveBreakdown
				func() error {
					dwsv1alpha2.AddWorkflowLabels(directiveBreakdown, workflow)
					dwsv1alpha2.AddOwnerLabels(directiveBreakdown, workflow)
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

func (r *NnfWorkflowReconciler) validateServerAllocations(ctx context.Context, dbd *dwsv1alpha2.DirectiveBreakdown, servers *dwsv1alpha2.Servers) error {
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

			if breakdownAllocationSet.AllocationStrategy == dwsv1alpha2.AllocateSingleServer {
				if len(serverAllocationSet.Storage) != 1 || serverAllocationSet.Storage[0].AllocationCount != 1 {
					err := fmt.Errorf("Allocation set %s expected single allocation", breakdownAllocationSet.Label)
					return nnfv1alpha1.NewWorkflowError("Allocation request does not meet directive requirements").WithFatal().WithError(err)
				}
			}

			var totalCapacity int64 = 0

			if breakdownAllocationSet.AllocationStrategy == dwsv1alpha2.AllocateAcrossServers {
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
				storage := &dwsv1alpha2.Storage{
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

func (r *NnfWorkflowReconciler) createNnfStorage(ctx context.Context, workflow *dwsv1alpha2.Workflow, s *dwsv1alpha2.Servers, index int, log logr.Logger) (*nnfv1alpha1.NnfStorage, error) {
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
			dwsv1alpha2.AddWorkflowLabels(nnfStorage, workflow)
			dwsv1alpha2.AddOwnerLabels(nnfStorage, owner)
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

func (r *NnfWorkflowReconciler) findLustreFileSystemForPath(ctx context.Context, path string, log logr.Logger) *lusv1beta1.LustreFileSystem {
	lustres := &lusv1beta1.LustreFileSystemList{}
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

func (r *NnfWorkflowReconciler) setupNnfAccessForServers(ctx context.Context, storage *nnfv1alpha1.NnfStorage, workflow *dwsv1alpha2.Workflow, index int, parentDwIndex int, teardownState dwsv1alpha2.WorkflowState, log logr.Logger) (*nnfv1alpha1.NnfAccess, error) {
	access := &nnfv1alpha1.NnfAccess{
		ObjectMeta: metav1.ObjectMeta{
			Name:      indexedResourceName(workflow, parentDwIndex) + "-servers",
			Namespace: workflow.Namespace,
		},
	}

	result, err := ctrl.CreateOrUpdate(ctx, r.Client, access,
		func() error {
			dwsv1alpha2.AddWorkflowLabels(access, workflow)
			dwsv1alpha2.AddOwnerLabels(access, workflow)
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

func (r *NnfWorkflowReconciler) getDirectiveFileSystemType(ctx context.Context, workflow *dwsv1alpha2.Workflow, index int) (string, error) {
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

func buildMountPath(workflow *dwsv1alpha2.Workflow, index int) string {
	return fmt.Sprintf("/mnt/nnf/%s-%d", workflow.UID, index)
}

func (r *NnfWorkflowReconciler) findPersistentInstance(ctx context.Context, wf *dwsv1alpha2.Workflow, psiName string) (*dwsv1alpha2.PersistentStorageInstance, error) {
	log := r.Log.WithValues("Workflow", types.NamespacedName{Name: wf.Name, Namespace: wf.Namespace})

	psi := &dwsv1alpha2.PersistentStorageInstance{}
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

func handleWorkflowError(err error, driverStatus *dwsv1alpha2.WorkflowDriverStatus) {
	e, ok := err.(*nnfv1alpha1.WorkflowError)
	if ok {
		e.Inject(driverStatus)
	} else {
		driverStatus.Status = dwsv1alpha2.StatusError
		driverStatus.Message = "Internal error: " + err.Error()
		driverStatus.Error = err.Error()
	}
}

// Returns the directive index with the 'name' argument matching name, or -1 if not found
func findDirectiveIndexByName(workflow *dwsv1alpha2.Workflow, name string, command string) int {
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
func findCopyOutDirectiveIndexByName(workflow *dwsv1alpha2.Workflow, name string) int {
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
func findContainerDirectiveIndexByName(workflow *dwsv1alpha2.Workflow, name string) int {
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

// Returns a <name, path> pair for the given staging argument (typically source or destination).
// Convert underscores in the variable name to dashs in the FS name.
// i.e. $DW_JOB_my_file_system_name/path/to/a/file into:
//   - "my-file-system-name"
//   - "/path/to/a/file"
func splitStagingArgumentIntoNameAndPath(arg string) (string, string) {

	var varname = ""
	if strings.HasPrefix(arg, "$DW_JOB_") {
		varname = strings.SplitN(strings.Replace(arg, "$DW_JOB_", "", 1), "/", 2)[0]
	} else if strings.HasPrefix(arg, "$DW_PERSISTENT_") {
		varname = strings.SplitN(strings.Replace(arg, "$DW_PERSISTENT_", "", 1), "/", 2)[0]
	}
	name := strings.ReplaceAll(varname, "_", "-")
	var path = "/"
	if strings.Count(arg, "/") >= 1 {
		path = "/" + strings.SplitN(arg, "/", 2)[1]
	}
	return name, path
}

// indexedResourceName returns a name for a workflow child resource based on the index of the #DW directive
func indexedResourceName(workflow *dwsv1alpha2.Workflow, dwIndex int) string {
	return fmt.Sprintf("%s-%d", workflow.Name, dwIndex)
}

// Returns the <name, namespace> pair for the #DW directive at the specified index
func getStorageReferenceNameFromWorkflowActual(workflow *dwsv1alpha2.Workflow, dwdIndex int) (string, string) {

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
func getStorageReferenceNameFromDBD(dbd *dwsv1alpha2.DirectiveBreakdown) (string, string) {

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

func (r *NnfWorkflowReconciler) unmountNnfAccessIfNecessary(ctx context.Context, workflow *dwsv1alpha2.Workflow, index int, accessSuffix string) (*result, error) {
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
	if !found || dwsv1alpha2.WorkflowState(teardownState) == workflow.Status.State {
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
func (r *NnfWorkflowReconciler) waitForNnfAccessStateAndReady(ctx context.Context, workflow *dwsv1alpha2.Workflow, index int, state string) (*result, error) {

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
			if !found || dwsv1alpha2.WorkflowState(teardownState) == workflow.Status.State {
				if access.Status.State != "unmounted" || !access.Status.Ready {
					return Requeue("pending unmount").withObject(access), nil
				}

			}
		}
	}

	return nil, nil
}

func (r *NnfWorkflowReconciler) addPersistentStorageReference(ctx context.Context, workflow *dwsv1alpha2.Workflow, index int) error {
	dwArgs, _ := dwdparse.BuildArgsMap(workflow.Spec.DWDirectives[index])

	persistentStorage, err := r.findPersistentInstance(ctx, workflow, dwArgs["name"])
	if err != nil {
		return err
	}

	if persistentStorage.Status.State != dwsv1alpha2.PSIStateActive {
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

func (r *NnfWorkflowReconciler) removePersistentStorageReference(ctx context.Context, workflow *dwsv1alpha2.Workflow, index int) error {
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

func (r *NnfWorkflowReconciler) removeAllPersistentStorageReferences(ctx context.Context, workflow *dwsv1alpha2.Workflow) error {
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

func (r *NnfWorkflowReconciler) containerHandler(ctx context.Context, workflow *dwsv1alpha2.Workflow, dwArgs map[string]string, index int, log logr.Logger) (*result, error) {
	profile, err := r.getContainerProfile(ctx, workflow, index)
	if err != nil {
		return nil, err
	}
	mpiJob := profile.Data.MPISpec != nil

	username := nnfv1alpha1.ContainerUser

	// Get the targeted NNF nodes for the container jobs
	nnfNodes, err := r.getNnfNodesFromComputes(ctx, workflow)
	if err != nil || len(nnfNodes) <= 0 {
		return nil, nnfv1alpha1.NewWorkflowError("error obtaining the target NNF nodes for containers:").WithError(err).WithFatal()
	}

	// Get the NNF volumes to mount into the containers
	volumes, result, err := r.getContainerVolumes(ctx, workflow, dwArgs)
	if err != nil {
		return nil, nnfv1alpha1.NewWorkflowErrorf("could not determine the list of volumes need to create container job for workflow: %s", workflow.Name).WithError(err).WithFatal()
	}
	if result != nil {
		return result, nil
	}

	applyLabels := func(job metav1.Object) error {

		// Apply Job Labels/Owners
		dwsv1alpha2.InheritParentLabels(job, workflow)
		dwsv1alpha2.AddOwnerLabels(job, workflow)
		dwsv1alpha2.AddWorkflowLabels(job, workflow)

		labels := job.GetLabels()
		labels[nnfv1alpha1.ContainerLabel] = workflow.Name
		labels[nnfv1alpha1.PinnedContainerProfileLabelName] = profile.GetName()
		labels[nnfv1alpha1.PinnedContainerProfileLabelNameSpace] = profile.GetNamespace()
		labels[nnfv1alpha1.DirectiveIndexLabel] = strconv.Itoa(index)
		job.SetLabels(labels)

		if err := ctrl.SetControllerReference(workflow, job, r.Scheme); err != nil {
			return nnfv1alpha1.NewWorkflowErrorf("setting Job controller reference failed for '%s':", job.GetName()).WithError(err)
		}

		return nil
	}

	applyTolerations := func(spec *corev1.PodSpec) {
		spec.Tolerations = append(spec.Tolerations, corev1.Toleration{
			Effect:   corev1.TaintEffectNoSchedule,
			Key:      "cray.nnf.node",
			Operator: corev1.TolerationOpEqual,
			Value:    "true",
		})
	}

	addInitContainerPasswd := func(spec *corev1.PodSpec, user string, uid, gid int64, image string) {
		// This script creates an entry in /etc/passwd to map the user to the given UID/GID using an
		// InitContainer. This is necessary for mpirun because it uses ssh to communicate with the
		// worker nodes. ssh itself requires that the UID is tied to a username in the container.
		// Since the launcher container is running as non-root, we need to make use of an InitContainer
		// to edit /etc/passwd and copy it to a volume which can then be mounted into the non-root
		// container to replace /etc/passwd.
		script := `# tie the UID/GID to the user
sed -i '/^$USER/d' /etc/passwd
echo "$USER:x:$UID:$GID::/home/$USER:/bin/sh" >> /etc/passwd
cp /etc/passwd /config/
exit 0
`
		// Replace the user and UID/GID
		script = strings.ReplaceAll(script, "$USER", user)
		script = strings.ReplaceAll(script, "$UID", fmt.Sprintf("%d", uid))
		script = strings.ReplaceAll(script, "$GID", fmt.Sprintf("%d", gid))

		spec.InitContainers = append(spec.InitContainers, corev1.Container{
			Name:  "mpi-init-passwd",
			Image: image,
			Command: []string{
				"/bin/sh",
				"-c",
				script,
			},
			VolumeMounts: []corev1.VolumeMount{
				{Name: "passwd", MountPath: "/config"},
			},
		})
	}

	addInitContainerWorkerWait := func(spec *corev1.PodSpec, worker int) {
		// Add an initContainer to ensure that a worker pod is up and discoverable via dns. This
		// assumes nslookup is available in the container. The nnf-mfu image provides this.
		script := `# use nslookup to contact workers
echo "contacting $HOST..."
for i in $(seq 1 100); do
   sleep 1
   echo "attempt $i of 100..."
   nslookup $HOST
   if [ $? -eq 0 ]; then
      echo "successfully contacted $HOST; done"
      exit 0
   fi
done
echo "failed to contact $HOST"
exit 1
`
		// Build the worker's hostname.domain (e.g. nnf-container-example-worker-0.nnf-container-example-worker.default.svc)
		// This name comes from mpi-operator.
		host := strings.ToLower(fmt.Sprintf(
			"%s-worker-%d.%s-worker.%s.svc", workflow.Name, worker, workflow.Name, workflow.Namespace))
		script = strings.ReplaceAll(script, "$HOST", host)

		spec.InitContainers = append(spec.InitContainers, corev1.Container{
			Name:  fmt.Sprintf("mpi-wait-for-worker-%d", worker),
			Image: spec.Containers[0].Image,
			Command: []string{
				"/bin/sh",
				"-c",
				script,
			},
		})
	}

	applyPermissions := func(spec *corev1.PodSpec, mpiJobSpec *mpiv2beta1.MPIJobSpec, user string, worker bool) {
		uid := int64(workflow.Spec.UserID)
		gid := int64(workflow.Spec.GroupID)

		// Add volume for /etc/passwd to map user to UID/GID
		spec.Volumes = append(spec.Volumes, corev1.Volume{
			Name: "passwd",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		})

		if !worker {
			// Add SecurityContext if necessary
			if spec.SecurityContext == nil {
				spec.SecurityContext = &corev1.PodSecurityContext{}
			}

			// Add spec level security context to apply FSGroup to all containers. This keeps the
			// volumes safe from root actions.
			spec.SecurityContext.FSGroup = &gid

			// Set the ssh key path for non-root users. Defaults to root.
			if mpiJobSpec != nil {
				mpiJobSpec.SSHAuthMountPath = fmt.Sprintf("/home/%s/.ssh", username)
			}
		}

		// Add user permissions to each container. This needs to be done for each container because
		// we do not want these permissions on the init container.
		for idx := range spec.Containers {
			container := &spec.Containers[idx]

			// Add an InitContainer to map the user to the provided uid/gid using /etc/passwd
			addInitContainerPasswd(spec, user, uid, gid, container.Image)

			// Add a mount to copy the modified /etc/passwd to
			container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{
				Name:      "passwd",
				MountPath: "/etc/passwd",
				SubPath:   "passwd",
			})

			// Add non-root permissions from the workflow's user/group ID for the launcher, but not
			// the worker. The worker needs to run an ssh daemon, which requires root. Commands on
			// the worker are executed via the launcher as the `mpiuser` and not root.
			if !worker {
				if container.SecurityContext == nil {
					container.SecurityContext = &corev1.SecurityContext{}
				}
				container.SecurityContext.RunAsUser = &uid
				container.SecurityContext.RunAsGroup = &gid
				nonRoot := true
				container.SecurityContext.RunAsNonRoot = &nonRoot
				su := false
				container.SecurityContext.AllowPrivilegeEscalation = &su
			}
		}
	}

	addNNFVolumes := func(spec *corev1.PodSpec) {
		for _, vol := range volumes {
			// Volumes
			hostPathType := corev1.HostPathDirectory
			spec.Volumes = append(spec.Volumes, corev1.Volume{
				Name: vol.name,
				VolumeSource: corev1.VolumeSource{
					HostPath: &corev1.HostPathVolumeSource{
						Path: vol.mountPath,
						Type: &hostPathType,
					},
				},
			})

			// Add VolumeMounts and Volume environment variables for all containers
			for idx := range spec.Containers {
				container := &spec.Containers[idx]

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
	}

	addEnvVars := func(spec *corev1.PodSpec, mpi bool) {
		// Add in non-volume environment variables for all containers
		for idx := range spec.Containers {
			container := &spec.Containers[idx]

			// Jobs/hostnames and services/subdomains are named differently based on mpi or not. For
			// MPI, there are launcher/worker pods and the service is named after the worker. For
			// non-MPI, the jobs are named after the rabbit node.
			subdomain := ""
			domain := workflow.Namespace + ".svc.cluster.local"
			hosts := []string{}

			if mpi {
				launcher := workflow.Name + "-launcher"
				worker := workflow.Name + "-worker"
				subdomain = worker

				hosts = append(hosts, launcher)
				for i, _ := range nnfNodes {
					hosts = append(hosts, fmt.Sprintf("%s-%d", worker, i))
				}
			} else {
				subdomain = spec.Subdomain
				hosts = append(hosts, nnfNodes...)
			}

			container.Env = append(container.Env,
				corev1.EnvVar{Name: "NNF_CONTAINER_SUBDOMAIN", Value: subdomain},
				corev1.EnvVar{Name: "NNF_CONTAINER_DOMAIN", Value: domain},
				corev1.EnvVar{Name: "NNF_CONTAINER_HOSTNAMES", Value: strings.Join(hosts, " ")})
		}
	}

	// MPI container workflow. In this model, we use mpi-operator to create an MPIJob, which creates
	// a job for the launcher (to run mpirun) and a replicaset for the worker pods. The worker nodes
	// run an ssh server to listen for mpirun operations from the launcher pod.
	createMPIJob := func() error {
		mpiJob := &mpiv2beta1.MPIJob{
			ObjectMeta: metav1.ObjectMeta{
				Name:      workflow.Name,
				Namespace: workflow.Namespace,
			},
		}
		profile.Data.MPISpec.DeepCopyInto(&mpiJob.Spec)
		username = nnfv1alpha1.ContainerMPIUser

		if err := applyLabels(&mpiJob.ObjectMeta); err != nil {
			return err
		}

		// Use the profile's backoff limit if not set
		if mpiJob.Spec.RunPolicy.BackoffLimit == nil {
			mpiJob.Spec.RunPolicy.BackoffLimit = &profile.Data.RetryLimit
		}

		// MPIJobs have two pod specs: one for the launcher and one for the workers
		launcher := mpiJob.Spec.MPIReplicaSpecs[mpiv2beta1.MPIReplicaTypeLauncher]
		launcherSpec := &launcher.Template.Spec
		worker := mpiJob.Spec.MPIReplicaSpecs[mpiv2beta1.MPIReplicaTypeWorker]
		workerSpec := &worker.Template.Spec

		// Keep failed pods around for log inspection
		launcher.RestartPolicy = mpicommonv1.RestartPolicyNever
		worker.RestartPolicy = mpicommonv1.RestartPolicyNever

		// Add NNF node tolerations
		applyTolerations(launcherSpec)
		applyTolerations(workerSpec)

		// Run the launcher on the first NNF node
		launcherSpec.NodeSelector = map[string]string{"kubernetes.io/hostname": nnfNodes[0]}

		// Use initContainers to ensure the workers are up and discoverable before running the launcher command
		for i := range nnfNodes {
			addInitContainerWorkerWait(launcherSpec, i)
		}

		// Target all the NNF nodes for the workers
		replicas := int32(len(nnfNodes))
		worker.Replicas = &replicas
		workerSpec.Affinity = &corev1.Affinity{
			// Ensure we run a worker on every NNF node
			NodeAffinity: &corev1.NodeAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
					NodeSelectorTerms: []corev1.NodeSelectorTerm{{
						MatchExpressions: []corev1.NodeSelectorRequirement{{
							Key:      "kubernetes.io/hostname",
							Operator: corev1.NodeSelectorOpIn,
							Values:   nnfNodes,
						}},
					}},
				},
			},
			// But make sure it's only 1 per node
			PodAntiAffinity: &corev1.PodAntiAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{{
					TopologyKey: "kubernetes.io/hostname",
					LabelSelector: &metav1.LabelSelector{
						MatchExpressions: []metav1.LabelSelectorRequirement{
							{
								Key:      "training.kubeflow.org/job-name",
								Operator: metav1.LabelSelectorOpIn,
								Values:   []string{workflow.Name},
							},
							{
								Key:      "training.kubeflow.org/job-role",
								Operator: metav1.LabelSelectorOpIn,
								Values:   []string{"worker"},
							},
						},
					}},
				},
			},
		}

		// Set the appropriate permissions (UID/GID) from the workflow
		applyPermissions(launcherSpec, &mpiJob.Spec, username, false)
		applyPermissions(workerSpec, &mpiJob.Spec, username, true)

		addNNFVolumes(launcherSpec)
		addNNFVolumes(workerSpec)
		addEnvVars(launcherSpec, true)
		addEnvVars(workerSpec, true)

		err = r.Create(ctx, mpiJob)
		if err != nil {
			if !apierrors.IsAlreadyExists(err) {
				return err
			}
		} else {
			log.Info("Created MPIJob", "name", mpiJob.Name, "namespace", mpiJob.Namespace)
		}

		return nil
	}

	// Non-MPI container workflow. In this model, a job is created for each NNF node which ensures
	// that a pod is executed successfully (or the backOffLimit) is hit. Each container in this model
	// runs the same image.
	createNonMPIJob := func() error {
		// Use one job that we'll use as a base to create all jobs. Each NNF node will get its own job.
		job := &batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: workflow.Namespace,
			},
		}
		profile.Data.Spec.DeepCopyInto(&job.Spec.Template.Spec)
		podSpec := &job.Spec.Template.Spec

		if err := applyLabels(&job.ObjectMeta); err != nil {
			return err
		}

		// Use the same labels as the job for the pods
		job.Spec.Template.Labels = job.DeepCopy().Labels

		job.Spec.BackoffLimit = &profile.Data.RetryLimit

		podSpec.RestartPolicy = corev1.RestartPolicyNever
		podSpec.Subdomain = workflow.Name // service name == workflow name

		applyTolerations(podSpec)
		applyPermissions(podSpec, nil, username, false)
		addNNFVolumes(podSpec)
		addEnvVars(podSpec, false)

		// Using the base job, create a job for each nnfNode. Only the name, hostname, and node selector is different for each node
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
					return err
				}
			} else {
				log.Info("Created non-MPI job", "name", newJob.Name, "namespace", newJob.Namespace)
			}
		}

		return nil
	}

	if mpiJob {
		if err := createMPIJob(); err != nil {
			return nil, nnfv1alpha1.NewWorkflowError("Unable to create/update MPIJob").WithFatal().WithError(err)
		}
	} else {
		if err := r.createContainerService(ctx, workflow); err != nil {
			return nil, nnfv1alpha1.NewWorkflowError("Unable to create/update Container Service").WithFatal().WithError(err)
		}

		if err := createNonMPIJob(); err != nil {
			return nil, nnfv1alpha1.NewWorkflowError("Unable to create/update Container Jobs").WithFatal().WithError(err)
		}
	}

	return nil, nil
}

func (r *NnfWorkflowReconciler) createContainerService(ctx context.Context, workflow *dwsv1alpha2.Workflow) error {
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

// Retrieve the computes for the workflow and find their local nnf nodes
func (r *NnfWorkflowReconciler) getNnfNodesFromComputes(ctx context.Context, workflow *dwsv1alpha2.Workflow) ([]string, error) {

	ret := []string{}
	nnfNodes := make(map[string]struct{}) // use a empty struct map to store unique values
	var computeNodes []string

	// Get the compute resources
	computes := dwsv1alpha2.Computes{
		ObjectMeta: metav1.ObjectMeta{
			Name:      workflow.Name,
			Namespace: workflow.Namespace,
		},
	}
	if err := r.Get(ctx, client.ObjectKeyFromObject(&computes), &computes); err != nil {
		return ret, nnfv1alpha1.NewWorkflowError("could not find Computes resource for workflow")
	}

	// Build the list of computes
	for _, c := range computes.Data {
		computeNodes = append(computeNodes, c.Name)
	}
	if len(computeNodes) == 0 {
		return computeNodes, nnfv1alpha1.NewWorkflowError("the Computes resources does not specify any compute nodes")
	}

	systemConfig := &dwsv1alpha2.SystemConfiguration{}
	if err := r.Get(ctx, types.NamespacedName{Name: "default", Namespace: corev1.NamespaceDefault}, systemConfig); err != nil {
		return ret, nnfv1alpha1.NewWorkflowError("could not get system configuration")
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
			return ret, nnfv1alpha1.NewWorkflowErrorf("supplied compute node '%s' not found in SystemConfiguration", c)
		}

		// Add the node to the map
		if _, found := nnfNodes[nnfNode]; !found {
			nnfNodes[nnfNode] = struct{}{}
		}
	}

	// Turn the map keys into a slice to return
	for n, _ := range nnfNodes {
		ret = append(ret, n)
	}

	return ret, nil
}

func (r *NnfWorkflowReconciler) waitForContainersToStart(ctx context.Context, workflow *dwsv1alpha2.Workflow, index int) (*result, error) {
	// Get profile to determine container job type (MPI or not)
	profile, err := r.getContainerProfile(ctx, workflow, index)
	if err != nil {
		return nil, err
	}

	if profile.Data.MPISpec != nil {
		mpiJob, result := r.getMPIJobConditions(ctx, workflow, index, 1)
		if result != nil {
			return result, nil
		}

		// Expect a job condition of running or succeeded to signal the start
		running := false
		for _, c := range mpiJob.Status.Conditions {
			if (c.Type == mpiv2beta1.JobRunning || c.Type == mpiv2beta1.JobSucceeded) && c.Status == corev1.ConditionTrue {
				running = true
				break
			}
		}

		if !running {
			return Requeue(fmt.Sprintf("pending MPIJob start for workflow '%s', index: %d", workflow.Name, index)).after(2 * time.Second), nil
		}
	} else {
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
	}

	return nil, nil
}

func (r *NnfWorkflowReconciler) getMPIJobConditions(ctx context.Context, workflow *dwsv1alpha2.Workflow, index, expected int) (*mpiv2beta1.MPIJob, *result) {
	mpiJob := &mpiv2beta1.MPIJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      workflow.Name,
			Namespace: workflow.Namespace,
		},
	}

	if err := r.Client.Get(ctx, client.ObjectKeyFromObject(mpiJob), mpiJob); err != nil {
		return nil, Requeue(fmt.Sprintf("pending MPIJob creation for workflow '%s', index: %d", workflow.Name, index)).after(2 * time.Second)
	}

	// The job is really only useful when we have 1 (JobCreated) or more conditions (JobRunning, JobSucceeded)
	if len(mpiJob.Status.Conditions) < expected {
		return nil, Requeue(fmt.Sprintf("pending MPIJob conditions for workflow '%s', index: %d", workflow.Name, index)).after(2 * time.Second)
	}

	return mpiJob, nil
}

func (r *NnfWorkflowReconciler) waitForContainersToFinish(ctx context.Context, workflow *dwsv1alpha2.Workflow, index int) (*result, error) {
	// Get profile to determine container job type (MPI or not)
	profile, err := r.getContainerProfile(ctx, workflow, index)
	if err != nil {
		return nil, err
	}
	timeout := time.Duration(profile.Data.PostRunTimeoutSeconds) * time.Second

	setTimeout := func(job batchv1.Job) error {
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
				return nnfv1alpha1.NewWorkflowErrorf("error updating job '%s' activeDeadlineSeconds:", job.Name)
			}
		}

		return nil
	}

	setMPITimeout := func(mpiJob *mpiv2beta1.MPIJob) error {
		// Set the ActiveDeadLineSeconds on each of the k8s jobs created by MPIJob/mpi-operator. We
		// need to retrieve the jobs in a different way than non-MPI jobs since the jobs are created
		// by the MPIJob.
		jobList, err := r.getMPIJobList(ctx, workflow, mpiJob)
		if err != nil {
			return nnfv1alpha1.NewWorkflowErrorf("waitForContainersToFinish: no MPIJob JobList found for workflow '%s', index: %d", workflow.Name, index)
		}

		if len(jobList.Items) < 1 {
			return nnfv1alpha1.NewWorkflowErrorf("waitForContainersToFinish: no MPIJob jobs found for workflow '%s', index: %d", workflow.Name, index)
		}

		for _, job := range jobList.Items {
			if err := setTimeout(job); err != nil {
				return err
			}
		}

		return nil
	}

	if profile.Data.MPISpec != nil {
		// We should expect at least 2 conditions: created and running
		mpiJob, result := r.getMPIJobConditions(ctx, workflow, index, 2)
		if result != nil {
			return result, nil
		}

		finished := false
		for _, c := range mpiJob.Status.Conditions {
			// Job is finished when we have a pass or fail result
			if (c.Type == mpiv2beta1.JobSucceeded || c.Type == mpiv2beta1.JobFailed) && c.Status == corev1.ConditionTrue {
				finished = true
				break
			}
		}

		if !finished {
			if err := setMPITimeout(mpiJob); err != nil {
				return nil, err
			}
			return Requeue(fmt.Sprintf("pending MPIJob completion for workflow '%s', index: %d", workflow.Name, index)).after(2 * time.Second), nil
		}

	} else {
		jobList, err := r.getContainerJobs(ctx, workflow, index)
		if err != nil {
			return nil, err
		}

		if len(jobList.Items) < 1 {
			return nil, nnfv1alpha1.NewWorkflowErrorf("waitForContainersToFinish: no container jobs found for workflow '%s', index: %d", workflow.Name, index)
		}

		// Ensure all the jobs are done running before we check the conditions.
		for _, job := range jobList.Items {
			// Jobs will have conditions when finished
			if len(job.Status.Conditions) <= 0 {
				if err := setTimeout(job); err != nil {
					return nil, err
				}
				return Requeue("pending container finish").after(2 * time.Second).withObject(&job), nil
			}
		}
	}

	return nil, nil
}

func (r *NnfWorkflowReconciler) checkContainersResults(ctx context.Context, workflow *dwsv1alpha2.Workflow, index int) (*result, error) {
	// Get profile to determine container job type (MPI or not)
	profile, err := r.getContainerProfile(ctx, workflow, index)
	if err != nil {
		return nil, err
	}

	if profile.Data.MPISpec != nil {
		mpiJob, result := r.getMPIJobConditions(ctx, workflow, index, 2)
		if result != nil {
			return result, nil
		}

		for _, c := range mpiJob.Status.Conditions {
			if c.Type == mpiv2beta1.JobFailed {
				return nil, nnfv1alpha1.NewWorkflowErrorf("container MPIJob %s (%s): %s", c.Type, c.Reason, c.Message)
			}
		}
	} else {
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
	}

	return nil, nil
}

func (r *NnfWorkflowReconciler) getMPIJobList(ctx context.Context, workflow *dwsv1alpha2.Workflow, mpiJob *mpiv2beta1.MPIJob) (*batchv1.JobList, error) {
	// The k8s jobs that are spawned off by MPIJob do not have labels tied to the workflow.
	// Therefore, we need to get the k8s jobs manually. To do this, we can query the jobs by the
	// name of the MPIJob. However, this doesn't account for the namespace. We need another way.
	matchLabels := client.MatchingLabels(map[string]string{
		"app": workflow.Name,
	})

	jobList := &batchv1.JobList{}
	if err := r.List(ctx, jobList, matchLabels); err != nil {
		return nil, nnfv1alpha1.NewWorkflowErrorf("could not retrieve Jobs for MPIJob %s", mpiJob.Name).WithError(err)
	}

	// Create a new list so we don't alter the loop iterator
	items := []batchv1.Job{}

	// Once we have the job list of the matching MPIJob names, we can filter through these by
	// checking the OwnerReferences to find the UID of the MPIJob to ensure we have the right
	// one in case we have workflow/MPIJobs with the same names, but in different namespaces.
	for _, job := range jobList.Items {
		for _, ref := range job.OwnerReferences {
			if ref.UID == mpiJob.UID {
				items = append(items, job)
			}
		}
	}

	jobList.Items = items
	return jobList, nil
}

func (r *NnfWorkflowReconciler) getContainerJobs(ctx context.Context, workflow *dwsv1alpha2.Workflow, index int) (*batchv1.JobList, error) {
	// Get the jobs for this workflow and directive index
	matchLabels := dwsv1alpha2.MatchingWorkflow(workflow)
	matchLabels[nnfv1alpha1.DirectiveIndexLabel] = strconv.Itoa(index)

	jobList := &batchv1.JobList{}
	if err := r.List(ctx, jobList, matchLabels); err != nil {
		return nil, nnfv1alpha1.NewWorkflowErrorf("could not retrieve Jobs for index %d", index).WithError(err)
	}

	return jobList, nil
}

func (r *NnfWorkflowReconciler) getContainerProfile(ctx context.Context, workflow *dwsv1alpha2.Workflow, index int) (*nnfv1alpha1.NnfContainerProfile, error) {
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

func (r *NnfWorkflowReconciler) findPinnedContainerProfile(ctx context.Context, workflow *dwsv1alpha2.Workflow, index int) (*nnfv1alpha1.NnfContainerProfile, error) {
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

func (r *NnfWorkflowReconciler) findContainerProfile(ctx context.Context, workflow *dwsv1alpha2.Workflow, index int) (*nnfv1alpha1.NnfContainerProfile, error) {
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

func (r *NnfWorkflowReconciler) createPinnedContainerProfileIfNecessary(ctx context.Context, workflow *dwsv1alpha2.Workflow, index int) error {
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

	dwsv1alpha2.AddOwnerLabels(pinnedProfile, workflow)

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
func (r *NnfWorkflowReconciler) getContainerVolumes(ctx context.Context, workflow *dwsv1alpha2.Workflow, dwArgs map[string]string) ([]nnfContainerVolume, *result, error) {
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

		// k8s resources can't have underscores
		volName = strings.ReplaceAll(volName, "_", "-")

		vol := nnfContainerVolume{
			name:           volName,
			command:        cmd,
			directiveName:  val,
			directiveIndex: -1,
			// and env vars can't have hyphens
			envVarName: strings.ReplaceAll(arg, "-", "_"),
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
