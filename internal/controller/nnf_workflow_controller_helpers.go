/*
 * Copyright 2022-2024 Hewlett Packard Enterprise Development LP
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
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	dwsv1alpha2 "github.com/DataWorkflowServices/dws/api/v1alpha2"
	"github.com/DataWorkflowServices/dws/utils/dwdparse"
	lusv1beta1 "github.com/NearNodeFlash/lustre-fs-operator/api/v1beta1"
	nnfv1alpha1 "github.com/NearNodeFlash/nnf-sos/api/v1alpha1"

	"github.com/go-logr/logr"
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

// When workflow stages cannot advance they return a Requeue result with a particular reason.
func Requeue(reason string) *result {
	return &result{Result: ctrl.Result{Requeue: true}, reason: reason}
}

func (r *result) complete() bool {
	return r.IsZero()
}

func (r *result) after(duration time.Duration) *result {
	r.Result.Requeue = false
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
				return dwsv1alpha2.NewResourceError("").WithError(err).WithUserMessage("invalid staging Directive: '%v'", directive)
			}

		case "create_persistent":
			createPersistentCount++

		case "destroy_persistent":
			deletePersistentCount++

		case "persistentdw":
			if err := r.validatePersistentInstanceDirective(ctx, wf, directive); err != nil {
				return dwsv1alpha2.NewResourceError("").WithError(err).WithUserMessage("could not validate persistent instance: '%s'", directive)
			}

		case "container":
			containerCount++

			if err := r.validateContainerDirective(ctx, wf, index); err != nil {
				return dwsv1alpha2.NewResourceError("").WithError(err).WithUserMessage("could not validate container directive: '%s'", directive)
			}
		}
	}

	if directiveCount > 1 {
		// Ensure create_persistent or destroy_persistent are singletons in the workflow
		if createPersistentCount+deletePersistentCount > 0 {
			return dwsv1alpha2.NewResourceError("").WithUserMessage("only a single create_persistent or destroy_persistent directive is allowed per workflow").WithFatal().WithUser()
		}

		// Only allow 1 container directive (for now)
		if containerCount > 1 {
			return dwsv1alpha2.NewResourceError("").WithUserMessage("only a single container directive is supported per workflow").WithFatal().WithUser()
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
				return dwsv1alpha2.NewResourceError("").WithUserMessage("job storage instance '%s' not found", name).WithFatal().WithUser()
			}

			args, err := dwdparse.BuildArgsMap(wf.Spec.DWDirectives[index])
			if err != nil {
				return dwsv1alpha2.NewResourceError("").WithUserMessage("invalid DW directive: '%s'", wf.Spec.DWDirectives[index]).WithFatal()
			}

			fsType, exists := args["type"]
			if !exists {
				return dwsv1alpha2.NewResourceError("").WithUserMessage("invalid DW directive match for staging argument").WithFatal()
			}

			if fsType == "raw" {
				return dwsv1alpha2.NewResourceError("").WithUserMessage("data movement can not be used with raw allocations").WithFatal().WithUser()
			}
		} else if strings.HasPrefix(arg, "$DW_PERSISTENT_") {
			if err := r.validatePersistentInstanceForStaging(ctx, name, wf.Namespace); err != nil {
				return dwsv1alpha2.NewResourceError("").WithUserMessage("persistent storage instance '%s' not found", name).WithFatal().WithUser()
			}
			if findDirectiveIndexByName(wf, name, "persistentdw") == -1 {
				return dwsv1alpha2.NewResourceError("").WithUserMessage("persistentdw directive mentioning '%s' not found", name).WithFatal().WithUser()
			}
		} else {
			if r.findLustreFileSystemForPath(ctx, arg, r.Log) == nil {
				return dwsv1alpha2.NewResourceError("").WithUserMessage("global Lustre file system containing '%s' not found", arg).WithFatal().WithUser()
			}
		}

		return nil
	}

	args, err := dwdparse.BuildArgsMap(directive)
	if err != nil {
		return dwsv1alpha2.NewResourceError("").WithUserMessage("invalid DW directive: '%s'", directive).WithFatal()
	}

	if err := validateStagingArgument(args["source"]); err != nil {
		return dwsv1alpha2.NewResourceError("Invalid source argument: '%s'", args["source"]).WithError(err)
	}

	if err := validateStagingArgument(args["destination"]); err != nil {
		return dwsv1alpha2.NewResourceError("Invalid destination argument: '%s'", args["destination"]).WithError(err)
	}

	return nil
}

// validateContainerDirective validates the container directive.
func (r *NnfWorkflowReconciler) validateContainerDirective(ctx context.Context, workflow *dwsv1alpha2.Workflow, index int) error {
	args, err := dwdparse.BuildArgsMap(workflow.Spec.DWDirectives[index])
	if err != nil {
		return dwsv1alpha2.NewResourceError("").WithUserMessage("invalid DW directive: '%s'", workflow.Spec.DWDirectives[index]).WithFatal()
	}

	// Ensure the supplied profile exists
	profile, err := findContainerProfile(ctx, r.Client, workflow, index)
	if err != nil {
		return dwsv1alpha2.NewResourceError("").WithError(err).WithUserMessage("no valid container profile found").WithError(err).WithFatal()
	}

	// Check to see if the container storage argument is in the list of storages in the container profile
	checkStorageIsInProfile := func(storageName string) error {
		for _, storage := range profile.Data.Storages {
			if storage.Name == storageName {
				return nil
			}
		}
		return dwsv1alpha2.NewResourceError("").WithUserMessage("storage '%s' not found in container profile '%s'", storageName, profile.Name).WithFatal().WithUser()
	}

	checkContainerFs := func(idx int) error {
		getDirectiveFsType := func(idx int) (string, error) {
			args, _ := dwdparse.BuildArgsMap(workflow.Spec.DWDirectives[idx])

			// For persistent instance
			if args["command"] == "persistentdw" {
				psi, err := r.getPersistentStorageInstance(ctx, args["name"], workflow.Namespace)
				if err != nil {
					return "", fmt.Errorf("could not retrieve persistent instance %s for container directive: %v", args["name"], err)
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
			return dwsv1alpha2.NewResourceError("").WithUserMessage("unsupported container filesystem: %s", t).WithFatal().WithUser()
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
					return dwsv1alpha2.NewResourceError("").WithUserMessage("jobdw directive mentioning '%s' not found", storageName).WithFatal().WithUser()
				}
				if err := checkContainerFs(idx); err != nil {
					return err
				}
				if err := checkStorageIsInProfile(arg); err != nil {
					return err
				}
				suppliedStorageArguments = append(suppliedStorageArguments, arg)
			} else if strings.HasPrefix(arg, "DW_PERSISTENT_") {
				if err := r.validatePersistentInstance(ctx, storageName, workflow.Namespace); err != nil {
					return dwsv1alpha2.NewResourceError("").WithError(err).WithUserMessage("persistent storage instance '%s' not found", storageName).WithFatal()
				}
				idx := findDirectiveIndexByName(workflow, storageName, "persistentdw")
				if idx == -1 {
					return dwsv1alpha2.NewResourceError("").WithUserMessage("persistentdw directive mentioning '%s' not found", storageName).WithFatal().WithUser()
				}
				if err := checkContainerFs(idx); err != nil {
					return err
				}
				if err := checkStorageIsInProfile(arg); err != nil {
					return err
				}
				suppliedStorageArguments = append(suppliedStorageArguments, arg)
			} else if strings.HasPrefix(arg, "DW_GLOBAL_") {
				// Look up the global lustre fs by path rather than LustreFilesystem name
				if globalLustre := r.findLustreFileSystemForPath(ctx, storageName, r.Log); globalLustre == nil {
					return dwsv1alpha2.NewResourceError("").WithUserMessage("global Lustre file system containing '%s' not found", storageName).WithFatal().WithUser()
				}
				if err := checkStorageIsInProfile(arg); err != nil {
					return dwsv1alpha2.NewResourceError("").WithError(err).WithUserMessage("storage '%s' is not present in the container profile", arg).WithUser().WithFatal()
				}
				suppliedStorageArguments = append(suppliedStorageArguments, arg)
			} else {
				return dwsv1alpha2.NewResourceError("").WithUserMessage("unrecognized container argument: %s", arg).WithFatal().WithUser()
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
					return dwsv1alpha2.NewResourceError("").WithUserMessage("storage '%s' in container profile '%s' is not optional: storage argument not found in the supplied arguments",
						storage.Name, profile.Name).WithUser().WithFatal()
				}
			}
		}

		return nil
	}

	if err := checkNonOptionalStorages(suppliedStorageArguments); err != nil {
		return err
	}

	return nil
}

// validatePersistentInstance validates the persistentdw directive.
func (r *NnfWorkflowReconciler) validatePersistentInstanceForStaging(ctx context.Context, name string, namespace string) error {
	psi, err := r.getPersistentStorageInstance(ctx, name, namespace)
	if err != nil {
		return dwsv1alpha2.NewResourceError("").WithError(err).WithUserMessage("could not get PersistentStorageInstance '%s'", name).WithFatal().WithUser()
	}

	if psi.Spec.FsType == "raw" {
		return dwsv1alpha2.NewResourceError("").WithUserMessage("data movement can not be used with raw allocations").WithFatal().WithUser()
	}

	if !psi.DeletionTimestamp.IsZero() {
		return dwsv1alpha2.NewResourceError("").WithUserMessage("Persistent storage instance '%s' is deleting", name).WithUser().WithFatal()
	}

	return nil
}

// validatePersistentInstance validates the persistentdw directive.
func (r *NnfWorkflowReconciler) validatePersistentInstance(ctx context.Context, name string, namespace string) error {
	psi, err := r.getPersistentStorageInstance(ctx, name, namespace)
	if err != nil {
		return dwsv1alpha2.NewResourceError("").WithError(err).WithUserMessage("could not get PersistentStorageInstance %s", name).WithFatal().WithUser()
	}

	if !psi.DeletionTimestamp.IsZero() {
		return dwsv1alpha2.NewResourceError("").WithUserMessage("Persistent storage instance '%s' is deleting", name).WithUser().WithFatal()
	}

	return nil
}

// validatePersistentInstance validates the persistentdw directive.
func (r *NnfWorkflowReconciler) validatePersistentInstanceDirective(ctx context.Context, wf *dwsv1alpha2.Workflow, directive string) error {
	// Validate that the persistent instance is available and not in the process of being deleted
	args, err := dwdparse.BuildArgsMap(directive)
	if err != nil {
		return dwsv1alpha2.NewResourceError("invalid DW directive: %s", directive).WithFatal()
	}

	psi, err := r.getPersistentStorageInstance(ctx, args["name"], wf.Namespace)
	if err != nil {
		return dwsv1alpha2.NewResourceError("").WithError(err).WithUserMessage("could not get PersistentStorageInstance '%s'", args["name"]).WithFatal().WithUser()
	}

	if !psi.DeletionTimestamp.IsZero() {
		return dwsv1alpha2.NewResourceError("").WithUserMessage("Persistent storage instance '%s' is deleting", args["name"]).WithUser().WithFatal()
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
				return nil, dwsv1alpha2.NewResourceError("CreateOrUpdate failed for DirectiveBreakdown: %v", client.ObjectKeyFromObject(directiveBreakdown)).WithError(err)
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
		return dwsv1alpha2.NewResourceError("Servers resource does not meet storage requirements for directive '%s'", dbd.Spec.Directive).WithUserMessage("Allocation request does not meet directive requirements").WithWLM().WithFatal()
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
					return dwsv1alpha2.NewResourceError("allocation set %s expected single allocation", breakdownAllocationSet.Label).WithUserMessage("storage directive requirements were not satisfied").WithWLM().WithFatal()
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
				return dwsv1alpha2.NewResourceError("allocation set %s specified insufficient capacity", breakdownAllocationSet.Label).WithUserMessage("storage directive requirements were not satisfied").WithWLM().WithFatal()
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
					return dwsv1alpha2.NewResourceError("could not get storage: %s", client.ObjectKeyFromObject(storage)).WithError(err).WithUserMessage("storage directive requirements were not satisfied").WithFatal()
				}
			}
		}

		if !found {
			return dwsv1alpha2.NewResourceError("allocation set %s not found in Servers resource", breakdownAllocationSet.Label).WithUserMessage("storage directive requirements were not satisfied").WithWLM().WithFatal()
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
		return nil, dwsv1alpha2.NewResourceError("").WithUserMessage("invalid DW directive: %s", workflow.Spec.DWDirectives[index]).WithFatal().WithUser()
	}

	pinnedName, pinnedNamespace := getStorageReferenceNameFromWorkflowActual(workflow, index)
	nnfStorageProfile, err := findPinnedProfile(ctx, r.Client, pinnedNamespace, pinnedName)
	if err != nil {
		return nil, dwsv1alpha2.NewResourceError("could not find pinned NnfStorageProfile: %v", types.NamespacedName{Name: pinnedName, Namespace: pinnedNamespace}).WithError(err).WithFatal()
	}

	var owner metav1.Object = workflow
	if dwArgs["command"] == "create_persistent" {
		psi, err := r.findPersistentInstance(ctx, workflow, dwArgs["name"])
		if err != nil {
			return nil, dwsv1alpha2.NewResourceError("could not find PersistentStorageInstance: %v", dwArgs["name"]).WithError(err).WithFatal()
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

			// determine the NID of the external MGS if necessary
			mgsNid := ""
			persistentMgsReference := corev1.ObjectReference{}

			if dwArgs["type"] == "lustre" && len(nnfStorageProfile.Data.LustreStorage.ExternalMGS) > 0 {
				// If the prefix on the ExternalMGS field is "pool:", then this is pool name instead of a NID.
				if strings.HasPrefix(nnfStorageProfile.Data.LustreStorage.ExternalMGS, "pool:") {
					// Copy the existing PersistentStorageInstance data if present to prevent picking a different
					// MGS
					for _, allocationSet := range nnfStorage.Spec.AllocationSets {
						mgsNid = allocationSet.NnfStorageLustreSpec.MgsAddress
						persistentMgsReference = allocationSet.NnfStorageLustreSpec.PersistentMgsReference
						break
					}

					// If no MGS was picked yet, pick one randomly from the pool of PersistentStorageInstances with the right label
					if mgsNid == "" {
						persistentMgsReference, mgsNid, err = r.getLustreMgsFromPool(ctx, strings.TrimPrefix(nnfStorageProfile.Data.LustreStorage.ExternalMGS, "pool:"))
						if err != nil {
							return err
						}
					}

				} else {
					// Take the MGS address as given in the NnfStorageProfile ExternalMgs field and remove the "0" for any
					// addresses using LNet 0. This is needed because Lustre trims the "0" off internally, so the MGS address
					// given in the "mount" command output is trimmed. We need the "mount" command output to match our view of
					// the MGS address so we can verify if the file system is mounted correctly.
					// Examples:
					// 25@kfi0:26@kfi0 -> 25@kfi:26@kfi
					// 25@o2ib10 -> 25@o2ib10
					// 10.1.1.113@tcp0 -> 10.1.1.113@tcp
					// 25@kfi0,25@kfi1:26@kfi0,26@kfi1 -> 25@kfi,25@kfi1:26@kfi,26@kfi1
					re := regexp.MustCompile("(@[A-Za-z0-9]+[A-Za-z])0")
					mgsNid = re.ReplaceAllString(nnfStorageProfile.Data.LustreStorage.ExternalMGS, "$1")
				}
			}

			sharedAllocation := false
			if dwArgs["type"] == "gfs2" && nnfStorageProfile.Data.GFS2Storage.CmdLines.SharedVg {
				sharedAllocation = true
			}
			if dwArgs["type"] == "xfs" && nnfStorageProfile.Data.XFSStorage.CmdLines.SharedVg {
				sharedAllocation = true
			}
			if dwArgs["type"] == "raw" && nnfStorageProfile.Data.RawStorage.CmdLines.SharedVg {
				sharedAllocation = true
			}

			// Need to remove all of the AllocationSets in the NnfStorage object before we begin
			nnfStorage.Spec.AllocationSets = []nnfv1alpha1.NnfStorageAllocationSetSpec{}

			// Iterate the Servers data elements to pull out the allocation sets for the server
			for i := range s.Spec.AllocationSets {
				nnfAllocSet := nnfv1alpha1.NnfStorageAllocationSetSpec{}

				nnfAllocSet.Name = s.Spec.AllocationSets[i].Label
				nnfAllocSet.Capacity = s.Spec.AllocationSets[i].AllocationSize
				nnfAllocSet.SharedAllocation = sharedAllocation
				if dwArgs["type"] == "lustre" {
					nnfAllocSet.NnfStorageLustreSpec.TargetType = s.Spec.AllocationSets[i].Label
					nnfAllocSet.NnfStorageLustreSpec.BackFs = "zfs"
					if len(mgsNid) > 0 {
						nnfAllocSet.NnfStorageLustreSpec.MgsAddress = mgsNid
						nnfAllocSet.NnfStorageLustreSpec.PersistentMgsReference = persistentMgsReference
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
		return nil, dwsv1alpha2.NewResourceError("CreateOrUpdate failed for NnfStorage: %v", client.ObjectKeyFromObject(nnfStorage)).WithError(err)
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

func (r *NnfWorkflowReconciler) getLustreMgsFromPool(ctx context.Context, pool string) (corev1.ObjectReference, string, error) {
	persistentStorageList := &dwsv1alpha2.PersistentStorageInstanceList{}
	if err := r.List(ctx, persistentStorageList, client.MatchingLabels(map[string]string{nnfv1alpha1.StandaloneMGTLabel: pool})); err != nil {
		return corev1.ObjectReference{}, "", err
	}

	// Choose an MGS at random from the list of persistent storages
	if len(persistentStorageList.Items) == 0 {
		return corev1.ObjectReference{}, "", dwsv1alpha2.NewResourceError("").WithUserMessage("no MGSs found for pool: %s", pool).WithFatal().WithUser()
	}

	persistentStorage := persistentStorageList.Items[rand.Intn(len(persistentStorageList.Items))]

	// Find the NnfStorage for the PersistentStorage so we can get the LNid
	nnfStorage := &nnfv1alpha1.NnfStorage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      persistentStorage.Name,
			Namespace: persistentStorage.Namespace,
		},
	}

	if err := r.Get(ctx, client.ObjectKeyFromObject(nnfStorage), nnfStorage); err != nil {
		return corev1.ObjectReference{}, "", dwsv1alpha2.NewResourceError("could not get persistent NnfStorage %v for MGS", client.ObjectKeyFromObject(nnfStorage)).WithError(err)
	}

	if nnfStorage.Spec.FileSystemType != "lustre" {
		return corev1.ObjectReference{}, "", dwsv1alpha2.NewResourceError("invalid file systems type '%s' for persistent MGS", nnfStorage.Spec.FileSystemType).WithFatal()
	}

	if len(nnfStorage.Spec.AllocationSets) != 1 {
		return corev1.ObjectReference{}, "", dwsv1alpha2.NewResourceError("unexpected number of allocation sets '%d' for persistent MGS", len(nnfStorage.Spec.AllocationSets)).WithFatal()
	}

	if len(nnfStorage.Status.MgsAddress) == 0 {
		return corev1.ObjectReference{}, "", dwsv1alpha2.NewResourceError("no LNid listed for persistent MGS").WithFatal()
	}

	return corev1.ObjectReference{
		Kind:      reflect.TypeOf(dwsv1alpha2.PersistentStorageInstance{}).Name(),
		Name:      persistentStorage.Name,
		Namespace: persistentStorage.Namespace,
	}, nnfStorage.Status.MgsAddress, nil
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
	pinnedName, pinnedNamespace := getStorageReferenceNameFromWorkflowActual(workflow, parentDwIndex)
	nnfStorageProfile, err := findPinnedProfile(ctx, r.Client, pinnedNamespace, pinnedName)
	if err != nil {
		return nil, dwsv1alpha2.NewResourceError("could not find pinned NnfStorageProfile: %v", types.NamespacedName{Name: pinnedName, Namespace: pinnedNamespace}).WithError(err).WithFatal()
	}

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
			addPinnedStorageProfileLabel(access, nnfStorageProfile)
			addDirectiveIndexLabel(access, index)
			nnfv1alpha1.AddDataMovementTeardownStateLabel(access, teardownState)

			access.Spec = nnfv1alpha1.NnfAccessSpec{
				DesiredState:    "mounted",
				TeardownState:   teardownState,
				Target:          "all",
				UserID:          workflow.Spec.UserID,
				GroupID:         workflow.Spec.GroupID,
				MountPath:       buildServerMountPath(workflow, parentDwIndex),
				MountPathPrefix: buildServerMountPath(workflow, parentDwIndex),

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
		return nil, dwsv1alpha2.NewResourceError("CreateOrUpdate failed for NnfAccess: %v", client.ObjectKeyFromObject(access)).WithError(err)
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
			return "", dwsv1alpha2.NewResourceError("could not get persistent NnfStorage %v to determine file system type", client.ObjectKeyFromObject(nnfStorage)).WithError(err)
		}

		return nnfStorage.Spec.FileSystemType, nil
	default:
		return "", dwsv1alpha2.NewResourceError("invalid directive '%s' to get file system type", workflow.Spec.DWDirectives[index]).WithFatal()
	}
}

func buildComputeMountPath(workflow *dwsv1alpha2.Workflow, index int) string {
	prefix := os.Getenv("COMPUTE_MOUNT_PREFIX")
	if len(prefix) == 0 {
		prefix = "/mnt/nnf"
	}
	return filepath.Clean(fmt.Sprintf("/%s/%s-%d", prefix, workflow.UID, index))
}

func buildServerMountPath(workflow *dwsv1alpha2.Workflow, index int) string {
	prefix := os.Getenv("SERVER_MOUNT_PREFIX")
	if len(prefix) == 0 {
		prefix = "/mnt/nnf"
	}
	return filepath.Clean(fmt.Sprintf("/%s/%s-%d", prefix, workflow.UID, index))
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

	return psi, nil
}

func handleWorkflowError(err error, driverStatus *dwsv1alpha2.WorkflowDriverStatus) {
	e, ok := err.(*dwsv1alpha2.ResourceErrorInfo)
	if ok {
		status, err := e.Severity.ToStatus()
		if err != nil {
			driverStatus.Status = dwsv1alpha2.StatusError
			driverStatus.Message = "Internal error: " + err.Error()
			driverStatus.Error = err.Error()
		} else {
			driverStatus.Status = status
			driverStatus.Message = e.GetUserMessage()
			driverStatus.Error = e.Error()
		}
	} else {
		driverStatus.Status = dwsv1alpha2.StatusError
		driverStatus.Message = "Internal error: " + err.Error()
		driverStatus.Error = err.Error()
	}
}

func handleWorkflowErrorByIndex(err error, workflow *dwsv1alpha2.Workflow, index int) {
	// Create a list of the driverStatus array elements that correspond to the current state
	// of the workflow and are targeted for the Rabbit driver
	driverList := []*dwsv1alpha2.WorkflowDriverStatus{}
	driverID := os.Getenv("DWS_DRIVER_ID")

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

		driverList = append(driverList, driverStatus)
	}

	for _, driverStatus := range driverList {
		if driverStatus.DWDIndex != index {
			continue
		}

		handleWorkflowError(err, driverStatus)

		return
	}

	panic(index)
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
	var path = ""
	if strings.Count(arg, "/") >= 1 {
		path = "/" + strings.SplitN(arg, "/", 2)[1]
	}
	return name, path
}

func getRabbitRelativePath(fsType string, storageRef *corev1.ObjectReference, access *nnfv1alpha1.NnfAccess, path, namespace string, index int) string {
	relPath := path

	if storageRef.Kind == reflect.TypeOf(nnfv1alpha1.NnfStorage{}).Name() {
		switch fsType {
		case "xfs", "gfs2":
			idxMount := getIndexMountDir(namespace, index)
			relPath = filepath.Join(access.Spec.MountPathPrefix, idxMount, path)

		case "lustre":
			relPath = filepath.Join(access.Spec.MountPath, path)
		}

		// Join does a Clean, which removes the trailing slash. Put it back.
		if strings.HasSuffix(path, "/") {
			relPath += "/"
		}
	}

	return relPath
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

func getDirectiveIndexLabel(object metav1.Object) string {
	labels := object.GetLabels()
	if labels == nil {
		return ""
	}

	return labels[nnfv1alpha1.DirectiveIndexLabel]
}

func setTargetOwnerUIDLabel(object metav1.Object, value string) {
	labels := object.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}

	labels[nnfv1alpha1.TargetOwnerUidLabel] = value
	object.SetLabels(labels)
}

func getTargetOwnerUIDLabel(object metav1.Object) string {
	labels := object.GetLabels()
	if labels == nil {
		return ""
	}

	return labels[nnfv1alpha1.TargetOwnerUidLabel]
}

func setTargetDirectiveIndexLabel(object metav1.Object, value string) {
	labels := object.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}

	labels[nnfv1alpha1.TargetDirectiveIndexLabel] = value
	object.SetLabels(labels)
}

func getTargetDirectiveIndexLabel(object metav1.Object) string {
	labels := object.GetLabels()
	if labels == nil {
		return ""
	}

	return labels[nnfv1alpha1.TargetDirectiveIndexLabel]
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
					return nil, dwsv1alpha2.NewResourceError("could not update NnfAccess: %v", client.ObjectKeyFromObject(access)).WithError(err)
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
		return nil, dwsv1alpha2.NewResourceError("unable to determine directive file system type").WithError(err).WithFatal()
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
			return nil, dwsv1alpha2.NewResourceError("could not get NnfAccess: %v", client.ObjectKeyFromObject(access)).WithError(err)
		}

		if access.Status.Error != nil {
			handleWorkflowErrorByIndex(access.Status.Error, workflow, index)

			return Requeue("mount/unmount error").withObject(access), nil
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
		return dwsv1alpha2.NewResourceError("").WithUserMessage("PersistentStorage '%v' not found", dwArgs["name"]).WithMajor().WithUser()
	}

	if persistentStorage.Status.State != dwsv1alpha2.PSIStateActive {
		return dwsv1alpha2.NewResourceError("").WithUserMessage("PersistentStorage is not active").WithFatal().WithUser()
	}

	// Add a consumer reference to the persistent storage for this directive
	reference := corev1.ObjectReference{
		Name:      indexedResourceName(workflow, index),
		Namespace: workflow.Namespace,
		Kind:      reflect.TypeOf(dwsv1alpha2.Workflow{}).Name(),
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
		Kind:      reflect.TypeOf(dwsv1alpha2.Workflow{}).Name(),
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

func (r *NnfWorkflowReconciler) userContainerHandler(ctx context.Context, workflow *dwsv1alpha2.Workflow, dwArgs map[string]string, index int, log logr.Logger) (*result, error) {
	profile, err := getContainerProfile(ctx, r.Client, workflow, index)
	if err != nil {
		return nil, err
	}
	mpiJob := profile.Data.MPISpec != nil

	// Get the targeted NNF nodes for the container jobs
	nnfNodes, err := r.getNnfNodesFromComputes(ctx, workflow)
	if err != nil || len(nnfNodes) <= 0 {
		return nil, dwsv1alpha2.NewResourceError("error obtaining the target NNF nodes for containers").WithError(err)
	}

	// Get the NNF volumes to mount into the containers
	volumes, result, err := r.getContainerVolumes(ctx, workflow, dwArgs, profile)
	if err != nil {
		return nil, dwsv1alpha2.NewResourceError("could not determine the list of volumes needed to create container job for workflow: %s", workflow.Name).WithError(err)
	}
	if result != nil { // a requeue can be returned, so make sure that happens
		return result, nil
	}

	c := nnfUserContainer{
		workflow: workflow,
		profile:  profile,
		nnfNodes: nnfNodes,
		volumes:  volumes,
		username: nnfv1alpha1.ContainerUser,
		uid:      int64(workflow.Spec.UserID),
		gid:      int64(workflow.Spec.GroupID),
		index:    index,
		client:   r.Client,
		log:      r.Log,
		scheme:   r.Scheme,
		ctx:      ctx,
	}

	if mpiJob {
		if err := c.createMPIJob(); err != nil {
			return nil, dwsv1alpha2.NewResourceError("unable to create/update MPIJob").WithMajor().WithError(err)
		}
	} else {
		// For non-MPI jobs, we need to create a service ourselves
		if err := r.createContainerService(ctx, workflow); err != nil {
			return nil, dwsv1alpha2.NewResourceError("unable to create/update Container Service").WithMajor().WithError(err)
		}

		if err := c.createNonMPIJob(); err != nil {
			return nil, dwsv1alpha2.NewResourceError("unable to create/update Container Jobs").WithMajor().WithError(err)
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
		return ret, dwsv1alpha2.NewResourceError("could not find Computes resource for workflow")
	}

	// Build the list of computes
	for _, c := range computes.Data {
		computeNodes = append(computeNodes, c.Name)
	}
	if len(computeNodes) == 0 {
		return computeNodes, dwsv1alpha2.NewResourceError("the Computes resources does not specify any compute nodes").WithWLM().WithFatal()
	}

	systemConfig := &dwsv1alpha2.SystemConfiguration{}
	if err := r.Get(ctx, types.NamespacedName{Name: "default", Namespace: corev1.NamespaceDefault}, systemConfig); err != nil {
		return ret, dwsv1alpha2.NewResourceError("could not get system configuration").WithFatal()
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
			return ret, dwsv1alpha2.NewResourceError("supplied compute node '%s' not found in SystemConfiguration", c).WithFatal()
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
	profile, err := getContainerProfile(ctx, r.Client, workflow, index)
	if err != nil {
		return nil, err
	}
	isMPIJob := profile.Data.MPISpec != nil

	// Timeouts - If the containers don't start after PreRunTimeoutSeconds, we need to send an error
	// up to the workflow in every one of our return cases. Each return path will check for
	// timeoutElapsed and bubble up a fatal error.
	// We must also set the Jobs' activeDeadline timeout so that the containers are stopped once the
	// timeout is hit. This needs to be handled slightly differently depending on if the job is MPI
	// or not. Once set, k8s will take care of stopping the pods for us.
	timeoutElapsed := false
	timeout := time.Duration(0)
	if profile.Data.PreRunTimeoutSeconds != nil {
		timeout = time.Duration(*profile.Data.PreRunTimeoutSeconds) * time.Second
	}
	timeoutMessage := fmt.Sprintf("user container(s) failed to start after %d seconds", int(timeout.Seconds()))

	// Check if PreRunTimeoutSeconds has elapsed and set the flag. The logic will check once more to
	// see if it started or not. If not, then the job(s) activeDeadline will be set to stop the
	// jobs/pods.
	if timeout > 0 && metav1.Now().Sub(workflow.Status.DesiredStateChange.Time) >= timeout {
		timeoutElapsed = true
	}

	if isMPIJob {
		mpiJob, result := r.getMPIJobConditions(ctx, workflow, index, 1)
		if result != nil {
			// If timeout, don't allow requeue and return an error
			if timeoutElapsed {
				return nil, dwsv1alpha2.NewResourceError("could not retrieve MPIJobs to set timeout").
					WithUserMessage(timeoutMessage).WithFatal()
			}
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

		// Jobs are not running. Check to see if timeout elapsed and have k8s stop the jobs for us.
		// If no timeout, then just requeue.
		if !running {
			if timeoutElapsed {
				r.Log.Info("container prerun timeout occurred, attempting to set MPIJob activeDeadlineSeconds")
				if err := r.setMPIJobTimeout(ctx, workflow, mpiJob, time.Duration(1*time.Millisecond)); err != nil {
					return nil, dwsv1alpha2.NewResourceError("could not set timeout on MPIJobs").
						WithUserMessage(timeoutMessage).WithError(err).WithFatal()
				} else {
					return nil, dwsv1alpha2.NewResourceError("MPIJob timeout set").WithUserMessage(timeoutMessage).WithFatal()
				}
			}
			return Requeue(fmt.Sprintf("pending MPIJob start for workflow '%s', index: %d", workflow.Name, index)).after(2 * time.Second), nil
		}
	} else {
		jobList, err := r.getContainerJobs(ctx, workflow, index)
		if err != nil {
			if timeoutElapsed {
				return nil, dwsv1alpha2.NewResourceError("could not retrieve Jobs to set timeout").
					WithUserMessage(timeoutMessage).WithFatal().WithError(err)
			}
			return nil, err
		}

		// Jobs may not be queryable yet, so requeue
		if len(jobList.Items) < 1 {
			// If timeout, don't allow a requeue and return an error
			if timeoutElapsed {
				return nil, dwsv1alpha2.NewResourceError("no Jobs found in JobList to set timeout").
					WithUserMessage(timeoutMessage).WithFatal()
			}
			return Requeue(fmt.Sprintf("pending job creation for workflow '%s', index: %d", workflow.Name, index)).after(2 * time.Second), nil
		}

		for _, job := range jobList.Items {

			// Attempt to set the timeout on all the Jobs in the list
			if timeoutElapsed {
				r.Log.Info("container prerun timeout occurred, attempting to set Job activeDeadlineSeconds")
				if err := r.setJobTimeout(ctx, job, time.Duration(1*time.Millisecond)); err != nil {
					return nil, dwsv1alpha2.NewResourceError("could not set timeout on MPIJobs").
						WithUserMessage(timeoutMessage).WithError(err).WithFatal()
				} else {
					continue
				}
			}

			// If we have any conditions, the job already finished
			if len(job.Status.Conditions) > 0 {
				continue
			}

			// Ready should be non-zero to indicate the a pod is running for the job
			if job.Status.Ready == nil || *job.Status.Ready < 1 {
				return Requeue(fmt.Sprintf("pending container start for job '%s'", job.Name)).after(2 * time.Second), nil
			}
		}

		// Report the timeout error
		if timeoutElapsed {
			return nil, dwsv1alpha2.NewResourceError("job(s) timeout set").WithUserMessage(timeoutMessage).WithFatal()
		}
	}

	return nil, nil
}

func (r *NnfWorkflowReconciler) deleteContainers(ctx context.Context, workflow *dwsv1alpha2.Workflow, index int) (*result, error) {
	doneMpi := false
	doneNonMpi := false

	// Set the delete propagation
	policy := metav1.DeletePropagationBackground
	deleteAllOptions := &client.DeleteAllOfOptions{
		DeleteOptions: client.DeleteOptions{
			PropagationPolicy: &policy,
		},
	}
	// Add workflow matchLabels + directive index (if desired)
	matchLabels := dwsv1alpha2.MatchingWorkflow(workflow)
	if index >= 0 {
		matchLabels[nnfv1alpha1.DirectiveIndexLabel] = strconv.Itoa(index)
	}

	// Delete MPIJobs
	mpiJobList, err := r.getMPIJobs(ctx, workflow, index)
	if err != nil {
		if strings.Contains(err.Error(), "no kind is registered for the type") || apierrors.IsNotFound(err) {
			doneMpi = true
		} else {
			return nil, dwsv1alpha2.NewResourceError("could not delete container MPIJob(s)").WithError(err).WithMajor().WithInternal()
		}
	} else if len(mpiJobList.Items) > 0 {
		if err := r.DeleteAllOf(ctx, &mpiJobList.Items[0], client.InNamespace(workflow.Namespace), matchLabels, deleteAllOptions); err != nil {
			if !apierrors.IsNotFound(err) {
				return nil, dwsv1alpha2.NewResourceError("could not delete container MPIJob(s)").WithError(err).WithMajor().WithInternal()
			}
		}
	} else {
		doneMpi = true
	}

	// Delete non-MPI Jobs
	jobList, err := r.getContainerJobs(ctx, workflow, index)
	if err != nil {
		if apierrors.IsNotFound(err) {
			doneNonMpi = true
		} else {
			return nil, dwsv1alpha2.NewResourceError("could not delete container Job(s)").WithError(err).WithMajor().WithInternal()
		}
	} else if len(jobList.Items) > 0 {
		if err := r.DeleteAllOf(ctx, &jobList.Items[0], client.InNamespace(workflow.Namespace), matchLabels, deleteAllOptions); err != nil {
			if !apierrors.IsNotFound(err) {
				return nil, dwsv1alpha2.NewResourceError("could not delete container Job(s)").WithError(err).WithMajor().WithInternal()
			}
		}
	} else {
		doneNonMpi = true
	}

	if doneMpi && doneNonMpi {
		return nil, nil
	}

	return Requeue("pending container deletion"), nil
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

func (r *NnfWorkflowReconciler) setJobTimeout(ctx context.Context, job batchv1.Job, timeout time.Duration) error {
	// If desired, set the ActiveDeadline on the job to kill pods. Use the job's creation
	// timestamp to determine how long the job/pod has been running at this point. Then, add
	// the desired timeout to that value. k8s Job's ActiveDeadLineSeconds will then
	// terminate the pods once the deadline is hit.
	if timeout > 0 && job.Spec.ActiveDeadlineSeconds == nil {
		var deadline int64
		deadline = int64((metav1.Now().Sub(job.CreationTimestamp.Time) + timeout).Seconds())

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
			return dwsv1alpha2.NewResourceError("error updating job '%s' activeDeadlineSeconds:", job.Name)
		}
	}

	return nil
}

func (r *NnfWorkflowReconciler) setMPIJobTimeout(ctx context.Context, workflow *dwsv1alpha2.Workflow, mpiJob *mpiv2beta1.MPIJob, timeout time.Duration) error {
	// Set the ActiveDeadLineSeconds on each of the k8s jobs created by MPIJob/mpi-operator. We
	// need to retrieve the jobs in a different way than non-MPI jobs since the jobs are created
	// by the MPIJob.
	jobList, err := r.getMPIJobChildrenJobs(ctx, workflow, mpiJob)
	if err != nil {
		return dwsv1alpha2.NewResourceError("setMPIJobTimeout: no MPIJob JobList found for workflow '%s'", workflow.Name).WithMajor()
	}

	if len(jobList.Items) < 1 {
		return dwsv1alpha2.NewResourceError("setMPIJobTimeout: no MPIJob jobs found for workflow '%s'", workflow.Name).WithMajor()
	}

	for _, job := range jobList.Items {
		if err := r.setJobTimeout(ctx, job, timeout); err != nil {
			return err
		}
	}

	return nil
}

func (r *NnfWorkflowReconciler) waitForContainersToFinish(ctx context.Context, workflow *dwsv1alpha2.Workflow, index int) (*result, error) {
	// Get profile to determine container job type (MPI or not)
	profile, err := getContainerProfile(ctx, r.Client, workflow, index)
	if err != nil {
		return nil, err
	}
	isMPIJob := profile.Data.MPISpec != nil

	timeout := time.Duration(0)
	if profile.Data.PostRunTimeoutSeconds != nil {
		timeout = time.Duration(*profile.Data.PostRunTimeoutSeconds) * time.Second
	}

	if isMPIJob {
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
			if err := r.setMPIJobTimeout(ctx, workflow, mpiJob, timeout); err != nil {
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
			return nil, dwsv1alpha2.NewResourceError("waitForContainersToFinish: no container jobs found for workflow '%s', index: %d", workflow.Name, index).WithMajor()
		}

		// Ensure all the jobs are done running before we check the conditions.
		for _, job := range jobList.Items {
			// Jobs will have conditions when finished
			if len(job.Status.Conditions) <= 0 {
				if err := r.setJobTimeout(ctx, job, timeout); err != nil {
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
	profile, err := getContainerProfile(ctx, r.Client, workflow, index)
	if err != nil {
		return nil, err
	}
	isMPIJob := profile.Data.MPISpec != nil

	timeout := time.Duration(0)
	if profile.Data.PostRunTimeoutSeconds != nil {
		timeout = time.Duration(*profile.Data.PostRunTimeoutSeconds) * time.Second
	}
	timeoutMessage := fmt.Sprintf("user container(s) failed to complete after %d seconds", int(timeout.Seconds()))

	if isMPIJob {
		mpiJob, result := r.getMPIJobConditions(ctx, workflow, index, 2)
		if result != nil {
			return result, nil
		}

		for _, c := range mpiJob.Status.Conditions {
			if c.Type == mpiv2beta1.JobFailed {
				if c.Reason == "DeadlineExceeded" {
					return nil, dwsv1alpha2.NewResourceError("container MPIJob %s (%s): %s", c.Type, c.Reason, c.Message).WithFatal().
						WithUserMessage(timeoutMessage)
				}
				return nil, dwsv1alpha2.NewResourceError("container MPIJob %s (%s): %s", c.Type, c.Reason, c.Message).WithFatal().
					WithUserMessage("user container(s) failed to run successfully after %d attempts", profile.Data.RetryLimit+1)
			}
		}
	} else {
		jobList, err := r.getContainerJobs(ctx, workflow, index)
		if err != nil {
			return nil, err
		}

		if len(jobList.Items) < 1 {
			return nil, dwsv1alpha2.NewResourceError("checkContainersResults: no container jobs found for workflow '%s', index: %d", workflow.Name, index).WithMajor()
		}

		for _, job := range jobList.Items {
			for _, condition := range job.Status.Conditions {
				if condition.Type != batchv1.JobComplete {
					if condition.Reason == "DeadlineExceeded" {
						return nil, dwsv1alpha2.NewResourceError("container job %s (%s): %s", condition.Type, condition.Reason, condition.Message).WithFatal().WithUserMessage(timeoutMessage)
					}
					return nil, dwsv1alpha2.NewResourceError("container job %s (%s): %s", condition.Type, condition.Reason, condition.Message).WithFatal()
				}
			}
		}
	}

	return nil, nil
}

// Given an MPIJob, return a list of all the k8s Jobs owned by the MPIJob
func (r *NnfWorkflowReconciler) getMPIJobChildrenJobs(ctx context.Context, workflow *dwsv1alpha2.Workflow, mpiJob *mpiv2beta1.MPIJob) (*batchv1.JobList, error) {
	// The k8s jobs that are spawned off by MPIJob do not have labels tied to the workflow.
	// Therefore, we need to get the k8s jobs manually. To do this, we can query the jobs by the
	// name of the MPIJob. However, this doesn't account for the namespace. We need another way.
	matchLabels := client.MatchingLabels(map[string]string{
		"app": workflow.Name,
	})

	jobList := &batchv1.JobList{}
	if err := r.List(ctx, jobList, matchLabels); err != nil {
		return nil, dwsv1alpha2.NewResourceError("could not retrieve Jobs for MPIJob %s", mpiJob.Name).WithError(err).WithMajor()
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

func (r *NnfWorkflowReconciler) getMPIJobs(ctx context.Context, workflow *dwsv1alpha2.Workflow, index int) (*mpiv2beta1.MPIJobList, error) {
	// Get the MPIJobs for this workflow and directive index
	matchLabels := dwsv1alpha2.MatchingWorkflow(workflow)
	if index >= 0 {
		matchLabels[nnfv1alpha1.DirectiveIndexLabel] = strconv.Itoa(index)
	}

	jobList := &mpiv2beta1.MPIJobList{}
	if err := r.List(ctx, jobList, matchLabels); err != nil {
		return nil, dwsv1alpha2.NewResourceError("could not retrieve MPIJobs").WithError(err).WithMajor()
	}

	return jobList, nil
}

func (r *NnfWorkflowReconciler) getContainerJobs(ctx context.Context, workflow *dwsv1alpha2.Workflow, index int) (*batchv1.JobList, error) {
	// Get the jobs for this workflow and directive index
	matchLabels := dwsv1alpha2.MatchingWorkflow(workflow)
	if index >= 0 {
		matchLabels[nnfv1alpha1.DirectiveIndexLabel] = strconv.Itoa(index)
	}

	jobList := &batchv1.JobList{}
	if err := r.List(ctx, jobList, matchLabels); err != nil {
		return nil, dwsv1alpha2.NewResourceError("could not retrieve Jobs").WithError(err).WithMajor()
	}

	return jobList, nil
}

// Create a list of volumes to be mounted inside of the containers based on the DW_JOB/DW_PERSISTENT arguments
func (r *NnfWorkflowReconciler) getContainerVolumes(ctx context.Context, workflow *dwsv1alpha2.Workflow, dwArgs map[string]string, profile *nnfv1alpha1.NnfContainerProfile) ([]nnfContainerVolume, *result, error) {
	volumes := []nnfContainerVolume{}

	for arg, val := range dwArgs {
		volName, cmd := "", ""

		// Find any DW_(JOB|PERSISTENT) arguments
		if strings.HasPrefix(arg, "DW_JOB_") {
			volName = strings.TrimPrefix(arg, "DW_JOB_")
			cmd = "jobdw"
		} else if strings.HasPrefix(arg, "DW_PERSISTENT_") {
			volName = strings.TrimPrefix(arg, "DW_PERSISTENT_")
			cmd = "persistentdw"
		} else if strings.HasPrefix(arg, "DW_GLOBAL_") {
			volName = strings.TrimPrefix(arg, "DW_GLOBAL_")
			cmd = "globaldw"
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

		// For global lustre, a namespace that matches the workflow's namespace must be present in
		// the LustreFilesystem's Spec.Namespaces list. This results in a matching PVC that can
		// then be mounted into containers in that namespace.
		if cmd == "globaldw" {
			globalLustre := r.findLustreFileSystemForPath(ctx, val, r.Log)
			if globalLustre == nil {
				return nil, nil, dwsv1alpha2.NewResourceError("").WithUserMessage("global Lustre file system containing '%s' not found", val).WithUser().WithFatal()
			}

			ns, nsFound := globalLustre.Spec.Namespaces[workflow.Namespace]
			if !nsFound || len(ns.Modes) < 1 {
				return nil, nil, dwsv1alpha2.NewResourceError("").WithUserMessage("global Lustre file system containing '%s' is not configured for the '%s' namespace", val, workflow.Namespace).WithUser().WithFatal()
			}

			// Retrieve the desired PVC mode from the container profile. Default to readwritemany.
			modeStr := strings.ToLower(string(corev1.ReadWriteMany))
			if profile != nil {
				for _, storage := range profile.Data.Storages {
					if storage.Name == arg && storage.PVCMode != "" {
						modeStr = strings.ToLower(string(storage.PVCMode))
					}
				}
			}

			// e.g. PVC name: global-default-readwritemany-pvc
			vol.pvcName = strings.ToLower(fmt.Sprintf("%s-%s-%s-pvc", globalLustre.Name, globalLustre.Namespace, modeStr))
			vol.mountPath = globalLustre.Spec.MountRoot
		} else {
			// Find the directive index for the given name so we can retrieve its NnfAccess
			vol.directiveIndex = findDirectiveIndexByName(workflow, vol.directiveName, vol.command)
			if vol.directiveIndex < 0 {
				return nil, nil, dwsv1alpha2.NewResourceError("could not retrieve the directive breakdown for '%s'", vol.directiveName).WithMajor()
			}

			nnfAccess := &nnfv1alpha1.NnfAccess{
				ObjectMeta: metav1.ObjectMeta{
					Name:      workflow.Name + "-" + strconv.Itoa(vol.directiveIndex) + "-servers",
					Namespace: workflow.Namespace,
				},
			}
			if err := r.Get(ctx, client.ObjectKeyFromObject(nnfAccess), nnfAccess); err != nil {
				return nil, nil, dwsv1alpha2.NewResourceError("could not retrieve the NnfAccess '%s'", nnfAccess.Name).WithMajor()
			}

			if !nnfAccess.Status.Ready {
				return nil, Requeue(fmt.Sprintf("NnfAccess '%s' is not ready to be mounted into container", nnfAccess.Name)).after(2 * time.Second), nil
			}

			vol.mountPath = nnfAccess.Spec.MountPath
		}
		volumes = append(volumes, vol)
	}

	return volumes, nil, nil
}

// Use the container profile to determine how many ports are needed and request them from the default NnfPortManager
func (r *NnfWorkflowReconciler) getContainerPorts(ctx context.Context, workflow *dwsv1alpha2.Workflow, index int) (*result, error) {
	profile, err := getContainerProfile(ctx, r.Client, workflow, index)
	if err != nil {
		return nil, err
	}

	// Nothing to do here if ports are not requested
	if profile.Data.NumPorts > 0 {
		pm, err := getContainerPortManager(ctx, r.Client)
		if err != nil {
			return nil, err
		}

		// Check to see if we've already made an allocation
		for _, alloc := range pm.Spec.Allocations {
			if alloc.Requester.UID == workflow.UID {
				return nil, nil
			}
		}

		// Add a port allocation request to the manager for the number of ports specified by the
		// container profile
		pm.Spec.Allocations = append(pm.Spec.Allocations, nnfv1alpha1.NnfPortManagerAllocationSpec{
			Requester: corev1.ObjectReference{
				Name:      workflow.Name,
				Namespace: workflow.Namespace,
				Kind:      reflect.TypeOf(dwsv1alpha2.Workflow{}).Name(),
				UID:       workflow.UID,
			},
			Count: int(profile.Data.NumPorts),
		})

		if err := r.Update(ctx, pm); err != nil {
			if !apierrors.IsConflict(err) {
				return nil, err
			}
			return Requeue("update port manager allocation"), nil
		}

		r.Log.Info("Ports Requested", "numPorts", profile.Data.NumPorts)
	}

	return nil, nil
}

// Ensure that the default NnfPortManager has assigned the appropriate number of requested ports
func (r *NnfWorkflowReconciler) checkContainerPorts(ctx context.Context, workflow *dwsv1alpha2.Workflow, index int) (*result, error) {

	profile, err := getContainerProfile(ctx, r.Client, workflow, index)
	if err != nil {
		return nil, err
	}

	// Nothing to do here if ports are not requested
	r.Log.Info("Checking for requested ports", "numPorts", profile.Data.NumPorts)
	if profile.Data.NumPorts > 0 {
		pm, err := getContainerPortManager(ctx, r.Client)
		if err != nil {
			return nil, err
		}

		for _, alloc := range pm.Status.Allocations {
			if alloc.Requester != nil && alloc.Requester.UID == workflow.UID {
				if alloc.Status == nnfv1alpha1.NnfPortManagerAllocationStatusInUse && len(alloc.Ports) == int(profile.Data.NumPorts) {
					// Add workflow env var for the ports
					name, val := getContainerPortsEnvVar(alloc.Ports)
					workflow.Status.Env[name] = val
					return nil, nil // done
				} else if alloc.Status == nnfv1alpha1.NnfPortManagerAllocationStatusInvalidConfiguration {
					return nil, dwsv1alpha2.NewResourceError("").WithUserMessage("could not request ports for container workflow: Invalid NnfPortManager configuration").WithFatal().WithUser()
				} else if alloc.Status == nnfv1alpha1.NnfPortManagerAllocationStatusInsufficientResources {
					return nil, dwsv1alpha2.NewResourceError("").WithUserMessage("could not request ports for container workflow: InsufficientResources").WithFatal()
				}
			}
		}

		return Requeue("NnfPortManager allocation not ready").after(2 * time.Second).withObject(pm), nil
	}

	return nil, nil
}

// Retrieve the default NnfPortManager for user containers. Allow a client to be passed in as this
// is meant to be used by reconcilers or container helpers.
func getContainerPortManager(ctx context.Context, cl client.Client) (*nnfv1alpha1.NnfPortManager, error) {
	portManagerName := os.Getenv("NNF_PORT_MANAGER_NAME")
	portManagerNamespace := os.Getenv("NNF_PORT_MANAGER_NAMESPACE")

	pm := &nnfv1alpha1.NnfPortManager{
		ObjectMeta: metav1.ObjectMeta{
			Name:      portManagerName,
			Namespace: portManagerNamespace,
		},
	}
	if err := cl.Get(ctx, client.ObjectKeyFromObject(pm), pm); err != nil {
		return nil, err
	}

	return pm, nil
}

// Tell the NnfPortManager that the ports are no longer needed
// func (r *NnfWorkflowReconciler) releaseContainerPorts(ctx context.Context, workflow *dwsv1alpha2.Workflow, index int) (*result, error) {
func (r *NnfWorkflowReconciler) releaseContainerPorts(ctx context.Context, workflow *dwsv1alpha2.Workflow) (*result, error) {
	found := false

	pm, err := getContainerPortManager(ctx, r.Client)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		} else {
			return nil, err
		}
	}

	// Find the allocation in the Status
	for _, alloc := range pm.Status.Allocations {
		if alloc.Requester.UID == workflow.UID && alloc.Status == nnfv1alpha1.NnfPortManagerAllocationStatusInUse {
			found = true
			break
		}
	}

	if found {
		// Remove the allocation request from the Spec
		// TODO: For cooldowns, change the status to cooldown/time_wait rather than delete. Can we
		// even do that from here?
		for idx, alloc := range pm.Spec.Allocations {
			if alloc.Requester.UID == workflow.UID {
				pm.Spec.Allocations = append(pm.Spec.Allocations[:idx], pm.Spec.Allocations[idx+1:]...)
			}
		}

		if err := r.Update(ctx, pm); err != nil {
			if !apierrors.IsConflict(err) {
				return nil, err
			}
		}

		return Requeue("pending port de-allocation"), nil
	} else {
		return nil, nil
	}
}
