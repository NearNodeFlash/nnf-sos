/*
 * Copyright 2021-2023 Hewlett Packard Enterprise Development LP
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
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"

	dwsv1alpha2 "github.com/DataWorkflowServices/dws/api/v1alpha2"
	"github.com/DataWorkflowServices/dws/utils/dwdparse"
	"github.com/DataWorkflowServices/dws/utils/updater"
	lusv1beta1 "github.com/NearNodeFlash/lustre-fs-operator/api/v1beta1"
	nnfv1alpha1 "github.com/NearNodeFlash/nnf-sos/api/v1alpha1"
	"github.com/NearNodeFlash/nnf-sos/controllers/metrics"
)

const (
	// finalizerNnfWorkflow defines the key used in identifying the
	// storage object as being owned by this NNF Storage Reconciler. This
	// prevents the system from deleting the custom resource until the
	// reconciler has finished in using the resource.
	finalizerNnfWorkflow = "nnf.cray.hpe.com/nnf_workflow"
)

// NnfWorkflowReconciler contains the pieces used by the reconciler
type NnfWorkflowReconciler struct {
	client.Client
	Log          logr.Logger
	Scheme       *kruntime.Scheme
	ChildObjects []dwsv1alpha2.ObjectList
}

//+kubebuilder:rbac:groups=dataworkflowservices.github.io,resources=workflows,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups=dataworkflowservices.github.io,resources=workflows/finalizers,verbs=update
//+kubebuilder:rbac:groups=dataworkflowservices.github.io,resources=directivebreakdowns,verbs=get;create;list;watch;update;patch;delete;deletecollection
//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=directivebreakdowns/status,verbs=get;list;watch
//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfstorages,verbs=get;create;list;watch;update;patch;delete;deletecollection
//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfaccesses,verbs=get;create;list;watch;update;patch;delete;deletecollection
//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfdatamovements,verbs=get;create;list;watch;update;patch;delete;deletecollection
//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfstorageprofiles,verbs=get;create;list;watch;update;patch;delete;deletecollection
//+kubebuilder:rbac:groups=dataworkflowservices.github.io,resources=persistentstorageinstances,verbs=get;create;list;watch;update;patch;delete;deletecollection
//+kubebuilder:rbac:groups=dataworkflowservices.github.io,resources=servers,verbs=get;create;list;watch;update;patch
//+kubebuilder:rbac:groups=dataworkflowservices.github.io,resources=computes,verbs=get;create;list;watch;update;patch
//+kubebuilder:rbac:groups=lus.cray.hpe.com,resources=lustrefilesystems,verbs=get;list;watch

//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfcontainerprofiles,verbs=get;create;list;watch;update;patch;delete;deletecollection
//+kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;create;watch;update;patch;delete;deletecollection
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;create;update;patch;delete;deletecollection
//+kubebuilder:rbac:groups=kubeflow.org,resources=mpijobs,verbs=get;list;create;watch;update;patch;delete;deletecollection

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.9.2/pkg/reconcile
func (r *NnfWorkflowReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, err error) {
	log := r.Log.WithValues("Workflow", req.NamespacedName)

	metrics.NnfWorkflowReconcilesTotal.Inc()

	// Fetch the Workflow instance
	workflow := &dwsv1alpha2.Workflow{}

	if err := r.Get(ctx, req.NamespacedName, workflow); err != nil {
		// ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	statusUpdater := updater.NewStatusUpdater[*dwsv1alpha2.WorkflowStatus](workflow)
	defer func() { err = statusUpdater.CloseWithUpdate(ctx, r, err) }()

	driverID := os.Getenv("DWS_DRIVER_ID")

	// Check if the object is being deleted
	if !workflow.GetDeletionTimestamp().IsZero() {
		log.Info("Deleting workflow...")

		if !controllerutil.ContainsFinalizer(workflow, finalizerNnfWorkflow) {
			return ctrl.Result{}, nil
		}

		if err := r.removeAllPersistentStorageReferences(ctx, workflow); err != nil {
			return ctrl.Result{}, err
		}

		// Delete containers and unallocate port
		containerRes, err := r.deleteContainers(ctx, workflow, -1)
		if err != nil {
			return ctrl.Result{}, err
		} else if containerRes != nil {
			return containerRes.Result, nil
		}
		containerRes, err = r.releaseContainerPorts(ctx, workflow)
		if err != nil {
			return ctrl.Result{}, err
		} else if containerRes != nil {
			return containerRes.Result, nil
		}

		deleteStatus, err := dwsv1alpha2.DeleteChildren(ctx, r.Client, r.ChildObjects, workflow)
		if err != nil {
			return ctrl.Result{}, err
		}

		if !deleteStatus.Complete() {
			return ctrl.Result{}, nil
		}

		controllerutil.RemoveFinalizer(workflow, finalizerNnfWorkflow)
		if err := r.Update(ctx, workflow); err != nil {
			if !apierrors.IsConflict(err) {
				return ctrl.Result{}, err
			}

			return ctrl.Result{Requeue: true}, nil
		}

		return ctrl.Result{}, nil
	}

	// Add the finalizer if it doesn't exist
	if !controllerutil.ContainsFinalizer(workflow, finalizerNnfWorkflow) {
		controllerutil.AddFinalizer(workflow, finalizerNnfWorkflow)
		if err := r.Update(ctx, workflow); err != nil {
			if !apierrors.IsConflict(err) {
				return ctrl.Result{}, err
			}

			return ctrl.Result{Requeue: true}, nil
		}

		return ctrl.Result{}, nil
	}

	// If the dws has yet to set the Status.State, just return and wait for it.
	if workflow.Status.State == "" || workflow.Status.State != workflow.Spec.DesiredState {
		return ctrl.Result{}, nil
	}

	// If the workflow has already achieved its desired state, then there's no work to do
	if workflow.Status.Ready {
		return ctrl.Result{}, nil
	}

	// Create a list of the driverStatus array elements that correspond to the current state
	// of the workflow and are targeted for the Rabbit driver
	driverList := []*dwsv1alpha2.WorkflowDriverStatus{}

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

	startFunctions := map[dwsv1alpha2.WorkflowState]func(*NnfWorkflowReconciler, context.Context, *dwsv1alpha2.Workflow, int) (*result, error){
		dwsv1alpha2.StateProposal: (*NnfWorkflowReconciler).startProposalState,
		dwsv1alpha2.StateSetup:    (*NnfWorkflowReconciler).startSetupState,
		dwsv1alpha2.StateDataIn:   (*NnfWorkflowReconciler).startDataInOutState,
		dwsv1alpha2.StatePreRun:   (*NnfWorkflowReconciler).startPreRunState,
		dwsv1alpha2.StatePostRun:  (*NnfWorkflowReconciler).startPostRunState,
		dwsv1alpha2.StateDataOut:  (*NnfWorkflowReconciler).startDataInOutState,
		dwsv1alpha2.StateTeardown: (*NnfWorkflowReconciler).startTeardownState,
	}

	// Call the correct "start" function based on workflow state for each directive that has registered for
	// it. The "start" function does the initial work of setting up and creating the appropriate child resources.
	for _, driverStatus := range driverList {
		log := log.WithValues("state", workflow.Status.State, "index", driverStatus.DWDIndex)
		log.Info("Start", "directive", workflow.Spec.DWDirectives[driverStatus.DWDIndex])

		driverStatus.Status = dwsv1alpha2.StatusRunning
		driverStatus.Message = ""
		driverStatus.Error = ""

		result, err := startFunctions[workflow.Status.State](r, ctx, workflow, driverStatus.DWDIndex)
		if err != nil {
			handleWorkflowError(err, driverStatus)

			log.Info("Start error", "Message", err.Error())
			return ctrl.Result{}, err
		}

		if result != nil {
			log.Info("Start wait", result.info()...)
			driverStatus.Message = result.reason
			return result.Result, nil
		}

		log.Info("Start done")
	}

	finishFunctions := map[dwsv1alpha2.WorkflowState]func(*NnfWorkflowReconciler, context.Context, *dwsv1alpha2.Workflow, int) (*result, error){
		dwsv1alpha2.StateProposal: (*NnfWorkflowReconciler).finishProposalState,
		dwsv1alpha2.StateSetup:    (*NnfWorkflowReconciler).finishSetupState,
		dwsv1alpha2.StateDataIn:   (*NnfWorkflowReconciler).finishDataInOutState,
		dwsv1alpha2.StatePreRun:   (*NnfWorkflowReconciler).finishPreRunState,
		dwsv1alpha2.StatePostRun:  (*NnfWorkflowReconciler).finishPostRunState,
		dwsv1alpha2.StateDataOut:  (*NnfWorkflowReconciler).finishDataInOutState,
		dwsv1alpha2.StateTeardown: (*NnfWorkflowReconciler).finishTeardownState,
	}

	// Call the correct "finish" function based on workflow state for each directive that has registered for
	// it. The "finish" functions wait for the child resources to complete all their work and do any teardown
	// necessary.
	for _, driverStatus := range driverList {
		log := log.WithValues("state", workflow.Status.State, "index", driverStatus.DWDIndex)
		log.Info("Finish", "directive", workflow.Spec.DWDirectives[driverStatus.DWDIndex])

		driverStatus.Status = dwsv1alpha2.StatusRunning
		driverStatus.Message = ""
		driverStatus.Error = ""

		result, err := finishFunctions[workflow.Status.State](r, ctx, workflow, driverStatus.DWDIndex)
		if err != nil {
			handleWorkflowError(err, driverStatus)

			log.Info("Finish error", "Message", err.Error())

			return ctrl.Result{}, err
		}

		if result != nil {
			log.Info("Finish wait", result.info()...)
			if driverStatus.Message == "" {
				driverStatus.Message = result.reason
			}
			return result.Result, nil
		}

		log.Info("Finish done")

		ts := metav1.NowMicro()
		driverStatus.Status = dwsv1alpha2.StatusCompleted
		driverStatus.Message = ""
		driverStatus.Error = ""
		driverStatus.CompleteTime = &ts
		driverStatus.Completed = true
	}

	return ctrl.Result{}, nil
}

func (r *NnfWorkflowReconciler) startProposalState(ctx context.Context, workflow *dwsv1alpha2.Workflow, index int) (*result, error) {
	log := r.Log.WithValues("Workflow", client.ObjectKeyFromObject(workflow), "Index", index)
	dwArgs, _ := dwdparse.BuildArgsMap(workflow.Spec.DWDirectives[index])

	if err := r.validateWorkflow(ctx, workflow); err != nil {
		return nil, dwsv1alpha2.NewResourceError("").WithError(err).WithUserMessage("unable to validate DW directives")
	}

	// only jobdw, persistentdw, and create_persistent need a directive breakdown
	switch dwArgs["command"] {
	case "container":
		return nil, createPinnedContainerProfileIfNecessary(ctx, r.Client, r.Scheme, workflow, index, r.Log)
	case "jobdw", "persistentdw", "create_persistent":
		break
	default:
		return nil, nil
	}

	directiveBreakdown, err := r.generateDirectiveBreakdown(ctx, index, workflow, log)
	if err != nil {
		return nil, dwsv1alpha2.NewResourceError("could not generate DirectiveBreakdown").WithError(err).WithUserMessage("unable to start parsing DW directive")
	}

	if directiveBreakdown == nil {
		return Requeue("no breakdown"), nil
	}

	directiveBreakdownReference := v1.ObjectReference{
		Kind:      reflect.TypeOf(dwsv1alpha2.DirectiveBreakdown{}).Name(),
		Name:      directiveBreakdown.Name,
		Namespace: directiveBreakdown.Namespace,
	}

	found := false
	for _, reference := range workflow.Status.DirectiveBreakdowns {
		if reference == directiveBreakdownReference {
			found = true
		}
	}

	if !found {
		workflow.Status.DirectiveBreakdowns = append(workflow.Status.DirectiveBreakdowns, directiveBreakdownReference)
	}

	return nil, nil
}

func (r *NnfWorkflowReconciler) finishProposalState(ctx context.Context, workflow *dwsv1alpha2.Workflow, index int) (*result, error) {
	dwArgs, _ := dwdparse.BuildArgsMap(workflow.Spec.DWDirectives[index])

	// only jobdw, persistentdw, and create_persistent have a directive breakdown
	switch dwArgs["command"] {
	case "jobdw", "persistentdw", "create_persistent":
		break
	default:
		return nil, nil
	}

	directiveBreakdown := &dwsv1alpha2.DirectiveBreakdown{
		ObjectMeta: metav1.ObjectMeta{
			Name:      indexedResourceName(workflow, index),
			Namespace: workflow.GetNamespace(),
		},
	}

	err := r.Get(ctx, client.ObjectKeyFromObject(directiveBreakdown), directiveBreakdown)
	if err != nil {
		return nil, dwsv1alpha2.NewResourceError("could not get DirectiveBreakdown: %v", client.ObjectKeyFromObject(directiveBreakdown)).WithError(err).WithUserMessage("unable to finish parsing DW directive")
	}

	if directiveBreakdown.Status.Error != nil {
		handleWorkflowErrorByIndex(directiveBreakdown.Status.Error, workflow, index)

		return Requeue("error").withObject(directiveBreakdown), nil
	}

	// Wait for the breakdown to be ready
	if directiveBreakdown.Status.Ready != ConditionTrue {
		return Requeue("status pending").withObject(directiveBreakdown), nil
	}

	return nil, nil
}

func (r *NnfWorkflowReconciler) startSetupState(ctx context.Context, workflow *dwsv1alpha2.Workflow, index int) (*result, error) {
	log := r.Log.WithValues("Workflow", client.ObjectKeyFromObject(workflow), "Index", index)
	dwArgs, _ := dwdparse.BuildArgsMap(workflow.Spec.DWDirectives[index])

	switch dwArgs["command"] {
	default:
		return nil, nil
	case "persistentdw":
		return nil, r.addPersistentStorageReference(ctx, workflow, index)
	case "jobdw", "create_persistent":
		// Chain through the DirectiveBreakdown to the Servers object
		dbd := &dwsv1alpha2.DirectiveBreakdown{
			ObjectMeta: metav1.ObjectMeta{
				Name:      indexedResourceName(workflow, index),
				Namespace: workflow.Namespace,
			},
		}
		err := r.Get(ctx, client.ObjectKeyFromObject(dbd), dbd)
		if err != nil {
			return nil, dwsv1alpha2.NewResourceError("unable to get DirectiveBreakdown: %v", client.ObjectKeyFromObject(dbd)).WithError(err).WithUserMessage("could not read allocation request")
		}

		s := &dwsv1alpha2.Servers{
			ObjectMeta: metav1.ObjectMeta{
				Name:      dbd.Status.Storage.Reference.Name,
				Namespace: dbd.Status.Storage.Reference.Namespace,
			},
		}
		err = r.Get(ctx, client.ObjectKeyFromObject(s), s)
		if err != nil {
			return nil, dwsv1alpha2.NewResourceError("unable to get Servers: %v", client.ObjectKeyFromObject(s)).WithError(err).WithUserMessage("could not read allocation request")
		}

		if _, present := os.LookupEnv("RABBIT_TEST_ENV_BYPASS_SERVER_STORAGE_CHECK"); !present {
			if err := r.validateServerAllocations(ctx, dbd, s); err != nil {
				return nil, dwsv1alpha2.NewResourceError("invalid Rabbit allocations for servers: %v", client.ObjectKeyFromObject(s)).WithError(err).WithUserMessage("invalid Rabbit allocations")
			}
		}

		if storage, err := r.createNnfStorage(ctx, workflow, s, index, log); err != nil {
			if apierrors.IsConflict(err) {
				return Requeue("conflict").withObject(storage), nil
			}

			return nil, dwsv1alpha2.NewResourceError("could not create NnfStorage").WithError(err).WithUserMessage("could not create allocation")
		}
	case "container":
		return r.getContainerPorts(ctx, workflow, index)
	}

	return nil, nil
}

func (r *NnfWorkflowReconciler) finishSetupState(ctx context.Context, workflow *dwsv1alpha2.Workflow, index int) (*result, error) {
	dwArgs, _ := dwdparse.BuildArgsMap(workflow.Spec.DWDirectives[index])

	switch dwArgs["command"] {
	case "container":
		return r.checkContainerPorts(ctx, workflow, index)
	default:
		name, namespace := getStorageReferenceNameFromWorkflowActual(workflow, index)

		// Check whether the NnfStorage has finished creating the storage.
		nnfStorage := &nnfv1alpha1.NnfStorage{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
		}
		if err := r.Get(ctx, client.ObjectKeyFromObject(nnfStorage), nnfStorage); err != nil {
			return nil, dwsv1alpha2.NewResourceError("could not get NnfStorage: %v", client.ObjectKeyFromObject(nnfStorage)).WithError(err).WithUserMessage("could not allocate storage")
		}

		// If the Status section has not been filled in yet, exit and wait.
		if len(nnfStorage.Status.AllocationSets) != len(nnfStorage.Spec.AllocationSets) {
			// RequeueAfter is necessary for persistent storage that isn't owned by this workflow
			return Requeue("allocation").after(2 * time.Second).withObject(nnfStorage), nil
		}

		if nnfStorage.Status.Error != nil {
			handleWorkflowErrorByIndex(dwsv1alpha2.NewResourceError("storage resource error: %v", client.ObjectKeyFromObject(nnfStorage)).WithError(nnfStorage.Status.Error).WithUserMessage("could not allocate storage"), workflow, index)
			return Requeue("error").withObject(nnfStorage), nil
		}

		if nnfStorage.Status.Status != nnfv1alpha1.ResourceReady {
			// RequeueAfter is necessary for persistent storage that isn't owned by this workflow
			return Requeue("allocation set not ready").after(2 * time.Second).withObject(nnfStorage), nil
		}
	}

	return nil, nil
}

func (r *NnfWorkflowReconciler) startDataInOutState(ctx context.Context, workflow *dwsv1alpha2.Workflow, index int) (*result, error) {
	log := r.Log.WithValues("Workflow", client.ObjectKeyFromObject(workflow), "Index", index)

	dwArgs, err := dwdparse.BuildArgsMap(workflow.Spec.DWDirectives[index])
	if err != nil {
		return nil, dwsv1alpha2.NewResourceError("").WithUserMessage("Invalid DW directive: %v", workflow.Spec.DWDirectives[index]).WithFatal().WithUser()
	}

	// NOTE: We don't need to check for the occurrence of a source or destination parameters since these are required fields and validated through the webhook
	// NOTE: We don't need to validate destination has $DW_JOB_ since this is done during the proposal stage

	// copy_in needs to handle four cases
	// 1. Lustre to JobStorageInstance                              #DW copy_in source=[path] destination=$DW_JOB_[name]/[path]
	// 2. Lustre to PersistentStorageInstance                       #DW copy_in source=[path] destination=$DW_PERSISTENT_[name]/[path]
	// 3. PersistentStorageInstance to JobStorageInstance           #DW copy_in source=$DW_PERSISTENT_[name]/[path] destination=$DW_JOB_[name]/[path]
	// 4. PersistentStorageInstance to PersistentStorageInstance    #DW copy_in source=$DW_PERSISTENT_[name]/[path] destination=$DW_PERSISTENT_[name]/[path]
	// copy_out is the same, but typically source and destination are reversed

	// Prepare the provided staging parameter for data-movement. Param is the source/destination value from the #DW copy_in/copy_out directive; based
	// on the param prefix we determine the storage instance and access requirements for data movement.
	prepareStagingArgumentFn := func(param string) (*corev1.ObjectReference, *nnfv1alpha1.NnfAccess, *result, error) {
		var storageReference *corev1.ObjectReference

		name, _ := splitStagingArgumentIntoNameAndPath(param)

		// If param refers to a Job or Persistent storage type, find the NNF Storage that is backing
		// this directive line.
		if strings.HasPrefix(param, "$DW_JOB_") || strings.HasPrefix(param, "$DW_PERSISTENT_") {

			// Find the parent directive index that corresponds to this copy_in/copy_out directive
			parentDwIndex := 0

			if strings.HasPrefix(param, "$DW_PERSISTENT_") {
				parentDwIndex = findDirectiveIndexByName(workflow, name, "persistentdw")
			} else {
				parentDwIndex = findDirectiveIndexByName(workflow, name, "jobdw")
			}

			if parentDwIndex < 0 {
				return nil, nil, nil, dwsv1alpha2.NewResourceError("").WithUserMessage("no directive matching '%v' found in workflow", name).WithFatal().WithUser()
			}

			// If directive specifies a persistent storage instance, `name` will be the nnfStorageName
			// Otherwise it matches the workflow with the directive index as a suffix
			var nnfStorageName string
			if strings.HasPrefix(param, "$DW_PERSISTENT_") {
				nnfStorageName = name
			} else {
				nnfStorageName = indexedResourceName(workflow, parentDwIndex)
			}

			storage := &nnfv1alpha1.NnfStorage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      nnfStorageName,
					Namespace: workflow.Namespace,
				},
			}

			if err := r.Get(ctx, client.ObjectKeyFromObject(storage), storage); err != nil {
				return nil, nil, nil, dwsv1alpha2.NewResourceError("could not get NnfStorage %v", client.ObjectKeyFromObject(storage)).WithError(err).WithUserMessage("could not find storage allocation")
			}

			storageReference = &corev1.ObjectReference{
				Kind:      reflect.TypeOf(nnfv1alpha1.NnfStorage{}).Name(),
				Name:      storage.Name,
				Namespace: storage.Namespace,
			}

			fsType := storage.Spec.FileSystemType

			// Find the desired workflow teardown state for the NNF Access. This instructs the workflow
			// when to teardown an NNF Access for the servers
			var teardownState dwsv1alpha2.WorkflowState
			if dwArgs["command"] == "copy_in" {
				teardownState = dwsv1alpha2.StatePreRun

				if fsType == "gfs2" || fsType == "lustre" {
					teardownState = dwsv1alpha2.StatePostRun

					if findCopyOutDirectiveIndexByName(workflow, name) >= 0 {
						teardownState = dwsv1alpha2.StateTeardown
					}
				}
			} else if dwArgs["command"] == "copy_out" {
				teardownState = dwsv1alpha2.StateTeardown
			}

			// Setup NNF Access for the NNF Servers so we can run data movement on them.
			access, err := r.setupNnfAccessForServers(ctx, storage, workflow, index, parentDwIndex, teardownState, log)
			if err != nil {
				return storageReference, access, nil, dwsv1alpha2.NewResourceError("").WithError(err).WithUserMessage("could not create data movement mount points")
			}

			// Wait for accesses to go ready
			if access.Status.Ready == false {
				return nil, access, Requeue("status pending").withObject(access), nil
			}

			return storageReference, access, nil, nil

		} else if lustre := r.findLustreFileSystemForPath(ctx, param, r.Log); lustre != nil {
			storageReference = &corev1.ObjectReference{
				Kind:      reflect.TypeOf(lusv1beta1.LustreFileSystem{}).Name(),
				Name:      lustre.Name,
				Namespace: lustre.Namespace,
			}

			return storageReference, nil, nil, nil
		}

		return nil, nil, nil, dwsv1alpha2.NewResourceError("").WithUserMessage("Staging parameter '%s' is invalid", param).WithFatal().WithUser()
	}

	sourceStorage, sourceAccess, result, err := prepareStagingArgumentFn(dwArgs["source"])
	if err != nil {
		return nil, dwsv1alpha2.NewResourceError("").WithError(err).WithUserMessage("could not prepare data movement resources")
	} else if result != nil {
		return result, nil
	}

	destStorage, destAccess, result, err := prepareStagingArgumentFn(dwArgs["destination"])
	if err != nil {
		return nil, dwsv1alpha2.NewResourceError("").WithError(err).WithUserMessage("Could not prepare data movement resources")
	} else if result != nil {
		return result, nil
	}

	// Wait for accesses to go ready
	for _, access := range []*nnfv1alpha1.NnfAccess{sourceAccess, destAccess} {
		if access != nil {
			if err := r.Get(ctx, client.ObjectKeyFromObject(access), access); err != nil {
				return nil, dwsv1alpha2.NewResourceError("could not get NnfAccess %v", client.ObjectKeyFromObject(access)).WithError(err).WithUserMessage("could not create data movement mount points")
			}

			if access.Status.State != "mounted" || !access.Status.Ready {
				return Requeue("pending mount").withObject(access), nil
			}
		}
	}

	// Retrieve the target storage that is to perform the data movement.
	// For copy_in, the destination is the Rabbit and therefore the target
	// For copy_out, the source is the Rabbit and therefore the target

	var targetStorageRef *corev1.ObjectReference
	if workflow.Spec.DesiredState == dwsv1alpha2.StateDataIn {
		targetStorageRef = destStorage
	} else {
		targetStorageRef = sourceStorage
	}

	targetStorage := &nnfv1alpha1.NnfStorage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      targetStorageRef.Name,
			Namespace: targetStorageRef.Namespace,
		},
	}
	if err := r.Get(ctx, client.ObjectKeyFromObject(targetStorage), targetStorage); err != nil {
		return nil, dwsv1alpha2.NewResourceError("could not get NnfStorage: %v", client.ObjectKeyFromObject(targetStorage)).WithError(err).WithUserMessage("could not find storage allocations")
	}

	_, source := splitStagingArgumentIntoNameAndPath(dwArgs["source"])
	_, dest := splitStagingArgumentIntoNameAndPath(dwArgs["destination"])

	fsType := targetStorage.Spec.FileSystemType

	getRabbitRelativePath := func(storageRef *corev1.ObjectReference, access *nnfv1alpha1.NnfAccess, path string, index int) string {

		if storageRef.Kind == reflect.TypeOf(nnfv1alpha1.NnfStorage{}).Name() {
			switch fsType {
			case "xfs", "gfs2":
				return filepath.Join(access.Spec.MountPathPrefix, strconv.Itoa(index), path)
			case "lustre":
				return access.Spec.MountPath + path
			}
		}

		return path
	}

	switch fsType {
	case "xfs", "gfs2":

		// XFS & GFS2 require the individual rabbit nodes are performing the data movement.

		if len(targetStorage.Spec.AllocationSets) != 1 {
			return nil, dwsv1alpha2.NewResourceError("file system %s has unexpected allocation sets %d", fsType, len(targetStorage.Spec.AllocationSets)).WithUserMessage("unexpected allocation count").WithFatal()
		}

		nodes := targetStorage.Spec.AllocationSets[0].Nodes

		for _, node := range nodes {

			for i := 0; i < node.Count; i++ {
				dm := &nnfv1alpha1.NnfDataMovement{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("%s-%d", indexedResourceName(workflow, index), i),
						Namespace: node.Name,
					},
					Spec: nnfv1alpha1.NnfDataMovementSpec{
						Source: &nnfv1alpha1.NnfDataMovementSpecSourceDestination{
							Path:             getRabbitRelativePath(sourceStorage, sourceAccess, source, i),
							StorageReference: *sourceStorage,
						},
						Destination: &nnfv1alpha1.NnfDataMovementSpecSourceDestination{
							Path:             getRabbitRelativePath(destStorage, destAccess, dest, i),
							StorageReference: *destStorage,
						},
						UserId:  workflow.Spec.UserID,
						GroupId: workflow.Spec.GroupID,
					},
				}

				dwsv1alpha2.AddWorkflowLabels(dm, workflow)
				dwsv1alpha2.AddOwnerLabels(dm, workflow)
				nnfv1alpha1.AddDataMovementTeardownStateLabel(dm, workflow.Status.State)
				addDirectiveIndexLabel(dm, index)

				log.Info("Creating NNF Data Movement", "name", client.ObjectKeyFromObject(dm).String())
				if err := r.Create(ctx, dm); err != nil {
					if !errors.IsAlreadyExists(err) {
						return nil, dwsv1alpha2.NewResourceError("could not create DataMovement: %v", client.ObjectKeyFromObject(dm)).WithError(err).WithUserMessage("could not start data movement")
					}
				}
			}

		}

	case "lustre":

		dm := &nnfv1alpha1.NnfDataMovement{
			ObjectMeta: metav1.ObjectMeta{
				Name:      indexedResourceName(workflow, index),
				Namespace: nnfv1alpha1.DataMovementNamespace,
			},
			Spec: nnfv1alpha1.NnfDataMovementSpec{
				Source: &nnfv1alpha1.NnfDataMovementSpecSourceDestination{
					Path:             getRabbitRelativePath(sourceStorage, sourceAccess, source, 0),
					StorageReference: *sourceStorage,
				},
				Destination: &nnfv1alpha1.NnfDataMovementSpecSourceDestination{
					Path:             getRabbitRelativePath(destStorage, destAccess, dest, 0),
					StorageReference: *destStorage,
				},
				UserId:  workflow.Spec.UserID,
				GroupId: workflow.Spec.GroupID,
			},
		}

		dwsv1alpha2.AddWorkflowLabels(dm, workflow)
		dwsv1alpha2.AddOwnerLabels(dm, workflow)
		nnfv1alpha1.AddDataMovementTeardownStateLabel(dm, workflow.Status.State)
		addDirectiveIndexLabel(dm, index)

		log.Info("Creating NNF Data Movement", "name", client.ObjectKeyFromObject(dm).String())
		if err := r.Create(ctx, dm); err != nil {
			if !errors.IsAlreadyExists(err) {
				return nil, dwsv1alpha2.NewResourceError("could not create DataMovement: %v", client.ObjectKeyFromObject(dm)).WithError(err).WithUserMessage("could not start data movement")
			}
		}
	}

	return nil, nil
}

// Monitor a data movement resource for completion
func (r *NnfWorkflowReconciler) finishDataInOutState(ctx context.Context, workflow *dwsv1alpha2.Workflow, index int) (*result, error) {

	// Wait for data movement resources to complete

	matchingLabels := dwsv1alpha2.MatchingOwner(workflow)
	matchingLabels[nnfv1alpha1.DirectiveIndexLabel] = strconv.Itoa(index)
	matchingLabels[nnfv1alpha1.DataMovementTeardownStateLabel] = string(workflow.Status.State)

	dataMovementList := &nnfv1alpha1.NnfDataMovementList{}
	if err := r.List(ctx, dataMovementList, matchingLabels); err != nil {
		return nil, dwsv1alpha2.NewResourceError("could not list DataMovements with labels: %v", matchingLabels).WithError(err).WithUserMessage("could not find data movement information")
	}

	// Since the Finish state is only called when copy_in / copy_out directives are present - the lack of any items
	// implies that the data movement operations are only just creating and the cache hasn't been updated yet.
	if len(dataMovementList.Items) == 0 {
		return Requeue("pending data movement").after(2 * time.Second), nil
	}

	for _, dm := range dataMovementList.Items {
		if dm.Status.State != nnfv1alpha1.DataMovementConditionTypeFinished {
			return Requeue("pending data movement").withObject(&dm), nil
		}
	}

	// Check results of data movement operations
	// TODO: Detailed Fail Message?
	for _, dm := range dataMovementList.Items {
		if dm.Status.Status != nnfv1alpha1.DataMovementConditionReasonSuccess {
			handleWorkflowErrorByIndex(dwsv1alpha2.NewResourceError("").WithUserMessage("data movement operation failed").WithFatal(), workflow, index)
			return Requeue("error").withObject(&dm), nil
		}
	}

	return nil, nil
}

func (r *NnfWorkflowReconciler) startPreRunState(ctx context.Context, workflow *dwsv1alpha2.Workflow, index int) (*result, error) {
	log := r.Log.WithValues("Workflow", client.ObjectKeyFromObject(workflow), "Index", index)
	dwArgs, _ := dwdparse.BuildArgsMap(workflow.Spec.DWDirectives[index])

	// If there's an NnfAccess for the servers that needs to be unmounted, do it now. This is
	// required for XFS mounts where the compute node and Rabbit can't be mounted at the same
	// time.
	unmountResult, err := r.unmountNnfAccessIfNecessary(ctx, workflow, index, "servers")
	if err != nil {
		return nil, dwsv1alpha2.NewResourceError("could not unmount NnfAccess index: %v", index).WithError(err).WithUserMessage("could not unmount on Rabbit nodes")
	}

	if unmountResult != nil {
		return unmountResult, nil
	}

	access := &nnfv1alpha1.NnfAccess{
		ObjectMeta: metav1.ObjectMeta{
			Name:      indexedResourceName(workflow, index) + "-computes",
			Namespace: workflow.Namespace,
		},
	}

	// Create container service and jobs
	if dwArgs["command"] == "container" {
		result, err := r.userContainerHandler(ctx, workflow, dwArgs, index, log)

		if err != nil {
			return nil, dwsv1alpha2.NewResourceError("").WithError(err).WithFatal().WithUserMessage("unable to create/update Container Jobs")
		}
		if result != nil {
			return result, nil
		}

		return nil, nil
	}

	// Create an NNFAccess for the compute clients
	result, err := ctrl.CreateOrUpdate(ctx, r.Client, access,
		func() error {
			dwsv1alpha2.AddWorkflowLabels(access, workflow)
			dwsv1alpha2.AddOwnerLabels(access, workflow)
			addDirectiveIndexLabel(access, index)

			access.Spec.TeardownState = dwsv1alpha2.StatePostRun
			access.Spec.DesiredState = "mounted"
			access.Spec.UserID = workflow.Spec.UserID
			access.Spec.GroupID = workflow.Spec.GroupID
			access.Spec.Target = "single"
			access.Spec.MountPath = buildMountPath(workflow, index)
			access.Spec.ClientReference = corev1.ObjectReference{
				Name:      workflow.Name,
				Namespace: workflow.Namespace,
				Kind:      "Computes",
			}

			// Determine the name/namespace to use based on the directive
			name, namespace := getStorageReferenceNameFromWorkflowActual(workflow, index)

			access.Spec.StorageReference = corev1.ObjectReference{
				// Directive Breakdowns share the same NamespacedName with the Servers it creates, which shares the same name with the NNFStorage.
				Name:      name,
				Namespace: namespace,
				Kind:      reflect.TypeOf(nnfv1alpha1.NnfStorage{}).Name(),
			}

			return ctrl.SetControllerReference(workflow, access, r.Scheme)
		})
	if err != nil {
		return nil, dwsv1alpha2.NewResourceError("Could not CreateOrUpdate compute node NnfAccess: %v", client.ObjectKeyFromObject(access)).WithError(err).WithUserMessage("could not mount file system on compute nodes")
	}

	if result == controllerutil.OperationResultCreated {
		log.Info("Created NnfAccess", "name", access.Name)
	} else if result == controllerutil.OperationResultNone {
		// no change
	} else {
		log.Info("Updated NnfAccess", "name", access.Name)
	}

	// Create an NnfAccess for the servers resources if necessary. Shared storage like
	// that of GFS2 provides access to the Rabbit for the lifetime of the user's job.
	// NnfAccess may already be present if a data_in directive was specified for the
	// particular $DW_JOB_[name]; in this case we only need to recreate the resource

	fsType, err := r.getDirectiveFileSystemType(ctx, workflow, index)
	if err != nil {
		return nil, dwsv1alpha2.NewResourceError("").WithError(err).WithFatal().WithUser().WithUserMessage("Unable to determine directive file system type")
	}

	if fsType == "gfs2" || fsType == "lustre" {
		name, namespace := getStorageReferenceNameFromWorkflowActual(workflow, index)

		storage := &nnfv1alpha1.NnfStorage{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
		}

		// Set the teardown state to post run. If there is a copy_out or container directive that
		// uses this storage instance, set the teardown state so NNF Access is preserved up until
		// Teardown
		teardownState := dwsv1alpha2.StatePostRun
		if findCopyOutDirectiveIndexByName(workflow, dwArgs["name"]) >= 0 || findContainerDirectiveIndexByName(workflow, dwArgs["name"]) >= 0 {
			teardownState = dwsv1alpha2.StateTeardown
		}

		_, err := r.setupNnfAccessForServers(ctx, storage, workflow, index, index, teardownState, log)
		if err != nil {
			return nil, dwsv1alpha2.NewResourceError("could not setup NNF Access in state %s", workflow.Status.State).WithError(err).WithUserMessage("could not mount file system on Rabbit nodes")
		}
	}

	return nil, nil
}

func (r *NnfWorkflowReconciler) finishPreRunState(ctx context.Context, workflow *dwsv1alpha2.Workflow, index int) (*result, error) {

	dwArgs, _ := dwdparse.BuildArgsMap(workflow.Spec.DWDirectives[index])

	// Add an environment variable to the workflow status section for the location of the
	// mount point on the clients.
	if workflow.Status.Env == nil {
		workflow.Status.Env = make(map[string]string)
	}

	envName := ""
	switch dwArgs["command"] {
	case "jobdw":
		envName = "DW_JOB_" + strings.ReplaceAll(dwArgs["name"], "-", "_")
	case "persistentdw":
		envName = "DW_PERSISTENT_" + strings.ReplaceAll(dwArgs["name"], "-", "_")
	case "container":
		return r.waitForContainersToStart(ctx, workflow, index)
	default:
		return nil, dwsv1alpha2.NewResourceError("unexpected directive: %v", dwArgs["command"]).WithFatal().WithUserMessage("could not mount file system on compute nodes")
	}

	workflow.Status.Env[envName] = buildMountPath(workflow, index)

	// Containers do not have NNFAccesses, so only do this after r.waitForContainersToStart() would have returned
	result, err := r.waitForNnfAccessStateAndReady(ctx, workflow, index, "mounted")
	if err != nil {
		return nil, dwsv1alpha2.NewResourceError("could not mount rabbit NnfAccess for index %v", index).WithError(err).WithUserMessage("could not mount file system on compute nodes")
	} else if result != nil {
		return result, nil
	}

	return nil, nil
}

func (r *NnfWorkflowReconciler) startPostRunState(ctx context.Context, workflow *dwsv1alpha2.Workflow, index int) (*result, error) {

	dwArgs, _ := dwdparse.BuildArgsMap(workflow.Spec.DWDirectives[index])

	if dwArgs["command"] == "container" {
		return r.waitForContainersToFinish(ctx, workflow, index)
	}

	// Unmount the NnfAccess for the compute nodes. This will free the compute nodes to be used
	// in a different job even if there is data movement happening on the Rabbits.
	if result, err := r.unmountNnfAccessIfNecessary(ctx, workflow, index, "computes"); result != nil || err != nil {
		if err != nil {
			return nil, dwsv1alpha2.NewResourceError("").WithError(err).WithUserMessage("could not unmount file system from compute nodes")
		}

		return result, nil
	}

	// Wait for data movement resources to complete
	matchingLabels := dwsv1alpha2.MatchingOwner(workflow)
	matchingLabels[nnfv1alpha1.DataMovementTeardownStateLabel] = string(dwsv1alpha2.StatePostRun)

	dataMovementList := &nnfv1alpha1.NnfDataMovementList{}
	if err := r.List(ctx, dataMovementList, matchingLabels); err != nil {
		return nil, dwsv1alpha2.NewResourceError("could not list DataMovements with labels: %v", matchingLabels).WithError(err).WithUserMessage("could not find data movement information")
	}

	for _, dm := range dataMovementList.Items {
		if dm.Status.State != nnfv1alpha1.DataMovementConditionTypeFinished {
			return Requeue("pending data movement").withObject(&dm), nil
		}
	}

	// Unmount the NnfAccess for the servers resource if necessary.
	fsType, err := r.getDirectiveFileSystemType(ctx, workflow, index)
	if err != nil {
		return nil, dwsv1alpha2.NewResourceError("").WithError(err).WithFatal().WithUser().WithUserMessage("Unable to determine directive file system type")
	}

	if fsType == "gfs2" || fsType == "lustre" {
		if result, err := r.unmountNnfAccessIfNecessary(ctx, workflow, index, "servers"); result != nil || err != nil {
			return result, err
		}
	}

	return nil, nil
}

func (r *NnfWorkflowReconciler) finishPostRunState(ctx context.Context, workflow *dwsv1alpha2.Workflow, index int) (*result, error) {
	dwArgs, _ := dwdparse.BuildArgsMap(workflow.Spec.DWDirectives[index])

	if dwArgs["command"] == "container" {
		return r.checkContainersResults(ctx, workflow, index)
	}

	result, err := r.waitForNnfAccessStateAndReady(ctx, workflow, index, "unmounted")
	if err != nil {
		return nil, dwsv1alpha2.NewResourceError("could not unmount compute NnfAccess for index %v", index).WithError(err).WithUserMessage("could not unmount file system on compute nodes")
	} else if result != nil {
		return result, nil
	}

	// Any user created copy-offload data movement requests created during run must report any errors to the workflow.
	// TODO: Customer asked if this could be optional
	matchingLabels := dwsv1alpha2.MatchingOwner(workflow)
	matchingLabels[nnfv1alpha1.DataMovementTeardownStateLabel] = string(dwsv1alpha2.StatePostRun)

	dataMovementList := &nnfv1alpha1.NnfDataMovementList{}
	if err := r.List(ctx, dataMovementList, matchingLabels); err != nil {
		return nil, dwsv1alpha2.NewResourceError("could not list DataMovements with labels: %v", matchingLabels).WithError(err).WithUserMessage("could not find data movement information")
	}

	for _, dm := range dataMovementList.Items {
		if dm.Status.State != nnfv1alpha1.DataMovementConditionTypeFinished {
			return Requeue("pending data movement").withObject(&dm), nil
		}

		if dm.Status.Status == nnfv1alpha1.DataMovementConditionReasonFailed {
			handleWorkflowErrorByIndex(dwsv1alpha2.NewResourceError("data movement %v failed", client.ObjectKeyFromObject(&dm)).WithUserMessage("data movement failed").WithFatal(), workflow, index)
			return Requeue("error").withObject(&dm), nil
		}
	}

	return nil, nil
}

func (r *NnfWorkflowReconciler) startTeardownState(ctx context.Context, workflow *dwsv1alpha2.Workflow, index int) (*result, error) {
	dwArgs, _ := dwdparse.BuildArgsMap(workflow.Spec.DWDirectives[index])

	switch dwArgs["command"] {
	case "container":
		res, err := r.deleteContainers(ctx, workflow, index)
		if res != nil || err != nil {
			return res, err
		}
	default:
		// Delete the NnfDataMovement and NnfAccess for this directive before removing the NnfStorage.
		// copy_in/out directives can reference NnfStorage from a different directive, so all the NnfAccesses
		// need to be removed first.
		childObjects := []dwsv1alpha2.ObjectList{
			&nnfv1alpha1.NnfDataMovementList{},
			&nnfv1alpha1.NnfAccessList{},
		}

		deleteStatus, err := dwsv1alpha2.DeleteChildrenWithLabels(ctx, r.Client, childObjects, workflow, client.MatchingLabels{nnfv1alpha1.DirectiveIndexLabel: strconv.Itoa(index)})
		if err != nil {
			return nil, dwsv1alpha2.NewResourceError("could not delete NnfDataMovement and NnfAccess children").WithError(err).WithUserMessage("could not stop data movement and unmount file systems")
		}

		if !deleteStatus.Complete() {
			return Requeue("delete").withDeleteStatus(deleteStatus), nil
		}
	}

	return nil, nil
}

func (r *NnfWorkflowReconciler) finishTeardownState(ctx context.Context, workflow *dwsv1alpha2.Workflow, index int) (*result, error) {
	log := r.Log.WithValues("Workflow", client.ObjectKeyFromObject(workflow), "Index", index)
	dwArgs, _ := dwdparse.BuildArgsMap(workflow.Spec.DWDirectives[index])

	switch dwArgs["command"] {
	case "create_persistent":
		for _, driverStatus := range workflow.Status.Drivers {
			if driverStatus.WatchState == dwsv1alpha2.StateTeardown {
				continue
			}

			if !driverStatus.Completed {
				return nil, nil
			}
		}

		persistentStorage, err := r.findPersistentInstance(ctx, workflow, dwArgs["name"])
		if err != nil {
			return nil, dwsv1alpha2.NewResourceError("").WithError(err).WithUserMessage("could not find persistent storage %v", dwArgs["name"])
		}

		persistentStorage.SetOwnerReferences([]metav1.OwnerReference{})
		dwsv1alpha2.RemoveOwnerLabels(persistentStorage)
		labels := persistentStorage.GetLabels()
		delete(labels, nnfv1alpha1.DirectiveIndexLabel)
		persistentStorage.SetLabels(labels)

		err = r.Update(ctx, persistentStorage)
		if err != nil {
			return nil, dwsv1alpha2.NewResourceError("could not update PersistentStorage: %v", client.ObjectKeyFromObject(persistentStorage)).WithError(err).WithUserMessage("could not finalize peristent storage")
		}
		log.Info("Removed owner reference from persistent storage", "psi", persistentStorage)
	case "destroy_persistent":
		persistentStorage, err := r.findPersistentInstance(ctx, workflow, dwArgs["name"])
		if err != nil {
			if !apierrors.IsNotFound(err) {
				return nil, dwsv1alpha2.NewResourceError("").WithError(err).WithFatal().WithUser().WithUserMessage("could not find peristent storage %v", dwArgs["name"])
			}

			return nil, nil
		}

		if persistentStorage.Spec.UserID != workflow.Spec.UserID {
			return nil, dwsv1alpha2.NewResourceError("Existing persistent storage user ID %v does not match user ID %v", persistentStorage.Spec.UserID, workflow.Spec.UserID).WithError(err).WithUserMessage("user ID does not match existing persistent storage").WithFatal().WithUser()
		}

		if len(persistentStorage.Spec.ConsumerReferences) != 0 {
			err = fmt.Errorf("PersistentStorage cannot be deleted with %v consumers", len(persistentStorage.Spec.ConsumerReferences))
			log.Info(err.Error())
			return nil, dwsv1alpha2.NewResourceError("persistent storage cannot be deleted with %v consumers", len(persistentStorage.Spec.ConsumerReferences)).WithError(err).WithUserMessage("persistent storage cannot be deleted while in use").WithFatal().WithUser()
		}

		persistentStorage.Spec.State = dwsv1alpha2.PSIStateDestroying

		dwsv1alpha2.AddOwnerLabels(persistentStorage, workflow)
		addDirectiveIndexLabel(persistentStorage, index)

		if err := controllerutil.SetControllerReference(workflow, persistentStorage, r.Scheme); err != nil {
			return nil, dwsv1alpha2.NewResourceError("could not assign workflow as owner of PersistentInstance: %v", client.ObjectKeyFromObject(persistentStorage)).WithError(err).WithUserMessage("could not delete persistent storage %v", dwArgs["name"])
		}

		err = r.Update(ctx, persistentStorage)
		if err != nil {
			return nil, dwsv1alpha2.NewResourceError("could not update PersistentInstance: %v", client.ObjectKeyFromObject(persistentStorage)).WithError(err).WithUserMessage("could not delete persistent storage %v", dwArgs["name"])
		}
		log.Info("Add owner reference for persistent storage for deletion", "psi", persistentStorage)
	case "persistentdw":
		err := r.removePersistentStorageReference(ctx, workflow, index)
		if err != nil {
			return nil, dwsv1alpha2.NewResourceError("").WithError(err).WithUserMessage("Could not remove persistent storage reference")
		}
	case "container":
		// Release container ports
		res, err := r.releaseContainerPorts(ctx, workflow)
		if res != nil || err != nil {
			return res, err
		}
	default:
	}

	childObjects := []dwsv1alpha2.ObjectList{
		&nnfv1alpha1.NnfStorageList{},
		&dwsv1alpha2.PersistentStorageInstanceList{},
	}

	deleteStatus, err := dwsv1alpha2.DeleteChildrenWithLabels(ctx, r.Client, childObjects, workflow, client.MatchingLabels{nnfv1alpha1.DirectiveIndexLabel: strconv.Itoa(index)})
	if err != nil {
		return nil, dwsv1alpha2.NewResourceError("could not delete NnfStorage and PersistentStorageInstance children").WithError(err).WithUserMessage("could not delete storage allocations")
	}

	if !deleteStatus.Complete() {
		return Requeue("delete").withDeleteStatus(deleteStatus), nil
	}

	return nil, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NnfWorkflowReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.ChildObjects = []dwsv1alpha2.ObjectList{
		&nnfv1alpha1.NnfDataMovementList{},
		&nnfv1alpha1.NnfAccessList{},
		&nnfv1alpha1.NnfStorageList{},
		&dwsv1alpha2.PersistentStorageInstanceList{},
		&dwsv1alpha2.DirectiveBreakdownList{},
	}

	maxReconciles := runtime.GOMAXPROCS(0)
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(controller.Options{MaxConcurrentReconciles: maxReconciles}).
		For(&dwsv1alpha2.Workflow{}).
		Owns(&nnfv1alpha1.NnfAccess{}).
		Owns(&dwsv1alpha2.DirectiveBreakdown{}).
		Owns(&dwsv1alpha2.PersistentStorageInstance{}).
		Watches(&nnfv1alpha1.NnfDataMovement{}, handler.EnqueueRequestsFromMapFunc(dwsv1alpha2.OwnerLabelMapFunc)).
		Watches(&nnfv1alpha1.NnfStorage{}, handler.EnqueueRequestsFromMapFunc(dwsv1alpha2.OwnerLabelMapFunc)).
		Complete(r)
}
