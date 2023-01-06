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
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/source"

	dwsv1alpha1 "github.com/HewlettPackard/dws/api/v1alpha1"
	"github.com/HewlettPackard/dws/utils/dwdparse"
	"github.com/HewlettPackard/dws/utils/updater"
	lusv1alpha1 "github.com/NearNodeFlash/lustre-fs-operator/api/v1alpha1"
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
	ChildObjects []dwsv1alpha1.ObjectList
}

//+kubebuilder:rbac:groups=dws.cray.hpe.com,resources=workflows,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups=dws.cray.hpe.com,resources=workflows/finalizers,verbs=update
//+kubebuilder:rbac:groups=dws.cray.hpe.com,resources=directivebreakdowns,verbs=get;create;list;watch;update;patch;delete;deletecollection
//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=directivebreakdowns/status,verbs=get;list;watch
//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfstorages,verbs=get;create;list;watch;update;patch;delete;deletecollection
//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfaccesses,verbs=get;create;list;watch;update;patch;delete;deletecollection
//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfdatamovements,verbs=get;create;list;watch;update;patch;delete;deletecollection
//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfstorageprofiles,verbs=get;create;list;watch;update;patch;delete;deletecollection
//+kubebuilder:rbac:groups=dws.cray.hpe.com,resources=persistentstorageinstances,verbs=get;create;list;watch;update;patch;delete;deletecollection
//+kubebuilder:rbac:groups=dws.cray.hpe.com,resources=servers,verbs=get;create;list;watch;update;patch
//+kubebuilder:rbac:groups=dws.cray.hpe.com,resources=computes,verbs=get;create;list;watch;update;patch
//+kubebuilder:rbac:groups=cray.hpe.com,resources=lustrefilesystems,verbs=get;list;watch

//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfcontainerprofiles,verbs=get;create;list;watch;update;patch;delete;deletecollection
//+kubebuilder:rbac:groups=apps,resources=daemonsets,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.9.2/pkg/reconcile
func (r *NnfWorkflowReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, err error) {
	log := r.Log.WithValues("Workflow", req.NamespacedName)

	metrics.NnfWorkflowReconcilesTotal.Inc()

	// Fetch the Workflow instance
	workflow := &dwsv1alpha1.Workflow{}

	if err := r.Get(ctx, req.NamespacedName, workflow); err != nil {
		// ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	statusUpdater := updater.NewStatusUpdater[*dwsv1alpha1.WorkflowStatus](workflow)
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

		deleteStatus, err := dwsv1alpha1.DeleteChildren(ctx, r.Client, r.ChildObjects, workflow)
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

	// If the dws-operator has yet to set the Status.State, just return and wait for it.
	if workflow.Status.State == "" || workflow.Status.State != workflow.Spec.DesiredState {
		return ctrl.Result{}, nil
	}

	// If the workflow has already achieved its desired state, then there's no work to do
	if workflow.Status.Ready {
		return ctrl.Result{}, nil
	}

	// Create a list of the driverStatus array elements that correspond to the current state
	// of the workflow and are targeted for the Rabbit driver
	driverList := []*dwsv1alpha1.WorkflowDriverStatus{}

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

	startFunctions := map[dwsv1alpha1.WorkflowState]func(*NnfWorkflowReconciler, context.Context, *dwsv1alpha1.Workflow, int) (*result, error){
		dwsv1alpha1.StateProposal: (*NnfWorkflowReconciler).startProposalState,
		dwsv1alpha1.StateSetup:    (*NnfWorkflowReconciler).startSetupState,
		dwsv1alpha1.StateDataIn:   (*NnfWorkflowReconciler).startDataInOutState,
		dwsv1alpha1.StatePreRun:   (*NnfWorkflowReconciler).startPreRunState,
		dwsv1alpha1.StatePostRun:  (*NnfWorkflowReconciler).startPostRunState,
		dwsv1alpha1.StateDataOut:  (*NnfWorkflowReconciler).startDataInOutState,
		dwsv1alpha1.StateTeardown: (*NnfWorkflowReconciler).startTeardownState,
	}

	// Call the correct "start" function based on workflow state for each directive that has registered for
	// it. The "start" function does the initial work of setting up and creating the appropriate child resources.
	for _, driverStatus := range driverList {
		log := log.WithValues("state", workflow.Status.State, "index", driverStatus.DWDIndex)
		log.Info("Start", "directive", workflow.Spec.DWDirectives[driverStatus.DWDIndex])

		result, err := startFunctions[workflow.Status.State](r, ctx, workflow, driverStatus.DWDIndex)
		if err != nil {
			handleWorkflowError(err, driverStatus)

			log.Info("Start error", "Message", err.Error())
			return ctrl.Result{}, err
		}

		driverStatus.Status = dwsv1alpha1.StatusRunning
		driverStatus.Message = ""
		driverStatus.Error = ""

		if result != nil {
			log.Info("Start wait", result.info()...)
			return result.Result, nil
		}

		log.Info("Start done")
	}

	finishFunctions := map[dwsv1alpha1.WorkflowState]func(*NnfWorkflowReconciler, context.Context, *dwsv1alpha1.Workflow, int) (*result, error){
		dwsv1alpha1.StateProposal: (*NnfWorkflowReconciler).finishProposalState,
		dwsv1alpha1.StateSetup:    (*NnfWorkflowReconciler).finishSetupState,
		dwsv1alpha1.StateDataIn:   (*NnfWorkflowReconciler).finishDataInOutState,
		dwsv1alpha1.StatePreRun:   (*NnfWorkflowReconciler).finishPreRunState,
		dwsv1alpha1.StatePostRun:  (*NnfWorkflowReconciler).finishPostRunState,
		dwsv1alpha1.StateDataOut:  (*NnfWorkflowReconciler).finishDataInOutState,
		dwsv1alpha1.StateTeardown: (*NnfWorkflowReconciler).finishTeardownState,
	}

	// Call the correct "finish" function based on workflow state for each directive that has registered for
	// it. The "finish" functions wait for the child resources to complete all their work and do any teardown
	// necessary.
	for _, driverStatus := range driverList {
		log := log.WithValues("state", workflow.Status.State, "index", driverStatus.DWDIndex)
		log.Info("Finish", "directive", workflow.Spec.DWDirectives[driverStatus.DWDIndex])

		result, err := finishFunctions[workflow.Status.State](r, ctx, workflow, driverStatus.DWDIndex)
		if err != nil {
			handleWorkflowError(err, driverStatus)

			log.Info("Finish error", "Message", err.Error())

			return ctrl.Result{}, err
		}

		driverStatus.Status = dwsv1alpha1.StatusRunning
		driverStatus.Message = ""
		driverStatus.Error = ""

		if result != nil {
			log.Info("Finish wait", result.info()...)
			return result.Result, nil
		}

		log.Info("Finish done")

		ts := metav1.NowMicro()
		driverStatus.Status = dwsv1alpha1.StatusCompleted
		driverStatus.Message = ""
		driverStatus.Error = ""
		driverStatus.CompleteTime = &ts
		driverStatus.Completed = true
	}

	return ctrl.Result{}, nil
}

func (r *NnfWorkflowReconciler) startProposalState(ctx context.Context, workflow *dwsv1alpha1.Workflow, index int) (*result, error) {
	log := r.Log.WithValues("Workflow", client.ObjectKeyFromObject(workflow), "Index", index)
	dwArgs, _ := dwdparse.BuildArgsMap(workflow.Spec.DWDirectives[index])

	if err := r.validateWorkflow(ctx, workflow); err != nil {
		log.Error(err, "Unable to validate workflow")
		return nil, nnfv1alpha1.NewWorkflowError("Unable to validate DW directives").WithFatal().WithError(err)
	}

	// only jobdw, persistentdw, create_persistent, and container need a directive breakdown
	switch dwArgs["command"] {
	case "jobdw", "persistentdw", "create_persistent", "container":
		break
	default:
		return nil, nil
	}

	directiveBreakdown, err := r.generateDirectiveBreakdown(ctx, index, workflow, log)
	if err != nil {
		return nil, nnfv1alpha1.NewWorkflowError("Unable to start parsing DW directive").WithError(err)
	}

	if directiveBreakdown == nil {
		return Requeue("no breakdown"), nil
	}

	directiveBreakdownReference := v1.ObjectReference{
		Kind:      reflect.TypeOf(dwsv1alpha1.DirectiveBreakdown{}).Name(),
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

func (r *NnfWorkflowReconciler) finishProposalState(ctx context.Context, workflow *dwsv1alpha1.Workflow, index int) (*result, error) {
	log := r.Log.WithValues("Workflow", client.ObjectKeyFromObject(workflow), "Index", index)
	dwArgs, _ := dwdparse.BuildArgsMap(workflow.Spec.DWDirectives[index])

	// only jobdw, persistentdw, and create_persistent need a directive breakdown
	if dwArgs["command"] != "jobdw" && dwArgs["command"] != "persistentdw" && dwArgs["command"] != "create_persistent" {
		return nil, nil
	}

	directiveBreakdown := &dwsv1alpha1.DirectiveBreakdown{
		ObjectMeta: metav1.ObjectMeta{
			Name:      indexedResourceName(workflow, index),
			Namespace: workflow.GetNamespace(),
		},
	}

	err := r.Get(ctx, client.ObjectKeyFromObject(directiveBreakdown), directiveBreakdown)
	if err != nil {
		log.Info("Failed to get DirectiveBreakdown", "name", directiveBreakdown.GetName(), "error", err.Error())
		return nil, nnfv1alpha1.NewWorkflowError("Unable to finish parsing DW directive").WithError(err)
	}

	if directiveBreakdown.Status.Error != nil {
		return nil, nnfv1alpha1.NewWorkflowError("").WithError(directiveBreakdown.Status.Error)
	}

	// Wait for the breakdown to be ready
	if directiveBreakdown.Status.Ready != ConditionTrue {
		return Requeue("status pending").withObject(directiveBreakdown), nil
	}

	return nil, nil
}

func (r *NnfWorkflowReconciler) startSetupState(ctx context.Context, workflow *dwsv1alpha1.Workflow, index int) (*result, error) {
	log := r.Log.WithValues("Workflow", client.ObjectKeyFromObject(workflow), "Index", index)
	dwArgs, _ := dwdparse.BuildArgsMap(workflow.Spec.DWDirectives[index])

	if dwArgs["command"] == "persistentdw" {
		return nil, r.addPersistentStorageReference(ctx, workflow, index)
	}

	// Only jobdw and create_persistent need to create an NnfStorage resource
	if dwArgs["command"] != "create_persistent" && dwArgs["command"] != "jobdw" {
		return nil, nil
	}

	// Chain through the DirectiveBreakdown to the Servers object
	dbd := &dwsv1alpha1.DirectiveBreakdown{
		ObjectMeta: metav1.ObjectMeta{
			Name:      indexedResourceName(workflow, index),
			Namespace: workflow.Namespace,
		},
	}
	err := r.Get(ctx, client.ObjectKeyFromObject(dbd), dbd)
	if err != nil {
		log.Info("Unable to get directiveBreakdown", "dbd", client.ObjectKeyFromObject(dbd), "Message", err)
		err = fmt.Errorf("Unable to get DirectiveBreakdown %v: %w", client.ObjectKeyFromObject(dbd), err)
		return nil, nnfv1alpha1.NewWorkflowError("Could not read allocation request").WithError(err)
	}

	s := &dwsv1alpha1.Servers{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dbd.Status.Storage.Reference.Name,
			Namespace: dbd.Status.Storage.Reference.Namespace,
		},
	}
	err = r.Get(ctx, client.ObjectKeyFromObject(s), s)
	if err != nil {
		log.Info("Unable to get servers", "servers", client.ObjectKeyFromObject(s), "Message", err)
		err = fmt.Errorf("Unable to get Servers %v: %w", client.ObjectKeyFromObject(s), err)
		return nil, nnfv1alpha1.NewWorkflowError("Could not read allocation request").WithError(err)
	}

	if _, present := os.LookupEnv("RABBIT_TEST_ENV_BYPASS_SERVER_STORAGE_CHECK"); !present {
		if err := r.validateServerAllocations(ctx, dbd, s); err != nil {
			return nil, err
		}
	}

	if storage, err := r.createNnfStorage(ctx, workflow, s, index, log); err != nil {
		if apierrors.IsConflict(err) {
			return Requeue("conflict").withObject(storage), nil
		}

		log.Info("Failed to create nnf storage", "Message", err)
		err = fmt.Errorf("Could not create NnfStorage %w", err)
		return nil, nnfv1alpha1.NewWorkflowError("Could not create allocation").WithError(err)
	}

	return nil, nil
}

func (r *NnfWorkflowReconciler) finishSetupState(ctx context.Context, workflow *dwsv1alpha1.Workflow, index int) (*result, error) {
	name, namespace := getStorageReferenceNameFromWorkflowActual(workflow, index)

	// Check whether the NnfStorage has finished creating the storage.
	nnfStorage := &nnfv1alpha1.NnfStorage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	if err := r.Get(ctx, client.ObjectKeyFromObject(nnfStorage), nnfStorage); err != nil {
		err = fmt.Errorf("Could not get NnfStorage %v: %w", client.ObjectKeyFromObject(nnfStorage), err)
		return nil, nnfv1alpha1.NewWorkflowError("Could not create allocation").WithError(err)
	}

	// If the Status section has not been filled in yet, exit and wait.
	if len(nnfStorage.Status.AllocationSets) != len(nnfStorage.Spec.AllocationSets) {
		// RequeueAfter is necessary for persistent storage that isn't owned by this workflow
		return Requeue("allocation").after(2 * time.Second).withObject(nnfStorage), nil
	}

	if nnfStorage.Status.Error != nil {
		return nil, nnfv1alpha1.NewWorkflowError("Could not create allocation").WithError(nnfStorage.Status.Error)
	}

	if nnfStorage.Status.Status != nnfv1alpha1.ResourceReady {
		// RequeueAfter is necessary for persistent storage that isn't owned by this workflow
		return Requeue("allocation set not ready").after(2 * time.Second).withObject(nnfStorage), nil
	}

	return nil, nil
}

func (r *NnfWorkflowReconciler) startDataInOutState(ctx context.Context, workflow *dwsv1alpha1.Workflow, index int) (*result, error) {
	log := r.Log.WithValues("Workflow", client.ObjectKeyFromObject(workflow), "Index", index)

	dwArgs, err := dwdparse.BuildArgsMap(workflow.Spec.DWDirectives[index])
	if err != nil {
		return nil, nnfv1alpha1.NewWorkflowError("Invalid DW directive: " + workflow.Spec.DWDirectives[index]).WithFatal()
	}

	// NOTE: We don't need to check for the occurrence of a source or destination parameters since these are required fields and validated through the webhook
	// NOTE: We don't need to validate destination has $JOB_DW_ since this is done during the proposal stage

	// copy_in needs to handle four cases
	// 1. Lustre to JobStorageInstance                              #DW copy_in source=[path] destination=$JOB_DW_[name]/[path]
	// 2. Lustre to PersistentStorageInstance                       #DW copy_in source=[path] destination=$PERSISTENT_DW_[name]/[path]
	// 3. PersistentStorageInstance to JobStorageInstance           #DW copy_in source=$PERSISTENT_DW_[name]/[path] destination=$JOB_DW_[name]/[path]
	// 4. PersistentStorageInstance to PersistentStorageInstance    #DW copy_in source=$PERSISTENT_DW_[name]/[path] destination=$PERSISTENT_DW_[name]/[path]
	// copy_out is the same, but typically source and destination are reversed

	// Prepare the provided staging parameter for data-movement. Param is the source/destination value from the #DW copy_in/copy_out directive; based
	// on the param prefix we determine the storage instance and access requirements for data movement.
	prepareStagingArgumentFn := func(param string) (*corev1.ObjectReference, *nnfv1alpha1.NnfAccess, *result, error) {
		var storageReference *corev1.ObjectReference

		name, _ := splitStagingArgumentIntoNameAndPath(param)

		// If param refers to a Job or Persistent storage type, find the NNF Storage that is backing
		// this directive line.
		if strings.HasPrefix(param, "$JOB_DW_") || strings.HasPrefix(param, "$PERSISTENT_DW_") {

			// Find the parent directive index that corresponds to this copy_in/copy_out directive
			parentDwIndex := findDirectiveIndexByName(workflow, name)
			if parentDwIndex < 0 {
				return nil, nil, nil, nnfv1alpha1.NewWorkflowError("No directive matching '" + name + "' found in workflow").WithFatal()
			}

			// If directive specifies a persistent storage instance, `name` will be the nnfStorageName
			// Otherwise it matches the workflow with the directive index as a suffix
			var nnfStorageName string
			if strings.HasPrefix(param, "$PERSISTENT_DW_") {
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
				return nil, nil, nil, fmt.Errorf("Could not get NnfStorage %v: %w", client.ObjectKeyFromObject(storage), err)
			}

			storageReference = &corev1.ObjectReference{
				Kind:      reflect.TypeOf(nnfv1alpha1.NnfStorage{}).Name(),
				Name:      storage.Name,
				Namespace: storage.Namespace,
			}

			fsType := storage.Spec.FileSystemType

			// Find the desired workflow teardown state for the NNF Access. This instructs the workflow
			// when to teardown an NNF Access for the servers
			var teardownState dwsv1alpha1.WorkflowState
			if dwArgs["command"] == "copy_in" {
				teardownState = dwsv1alpha1.StateDataIn

				if fsType == "gfs2" || fsType == "lustre" {
					teardownState = dwsv1alpha1.StatePostRun

					if findCopyOutDirectiveIndexByName(workflow, name) >= 0 {
						teardownState = dwsv1alpha1.StateDataOut
					}
				}
			} else if dwArgs["command"] == "copy_out" {
				teardownState = dwsv1alpha1.StateDataOut
			}

			// Setup NNF Access for the NNF Servers so we can run data movement on them.
			access, err := r.setupNnfAccessForServers(ctx, storage, workflow, index, parentDwIndex, teardownState, log)
			if err != nil {
				return storageReference, access, nil, nnfv1alpha1.NewWorkflowError("Could not create data movement mount points").WithError(err)
			}

			// Wait for accesses to go ready
			if access.Status.Ready == false {
				return nil, access, Requeue("status pending").withObject(access), nil
			}

			return storageReference, access, nil, nil

		} else if lustre := r.findLustreFileSystemForPath(ctx, param, r.Log); lustre != nil {
			storageReference = &corev1.ObjectReference{
				Kind:      reflect.TypeOf(lusv1alpha1.LustreFileSystem{}).Name(),
				Name:      lustre.Name,
				Namespace: lustre.Namespace,
			}

			return storageReference, nil, nil, nil
		}

		return nil, nil, nil, nnfv1alpha1.NewWorkflowError(fmt.Sprintf("Staging parameter '%s' is invalid", param)).WithFatal()
	}

	sourceStorage, sourceAccess, result, err := prepareStagingArgumentFn(dwArgs["source"])
	if err != nil {
		return nil, nnfv1alpha1.NewWorkflowError("Could not prepare data movement resources").WithError(err)
	} else if result != nil {
		return result, nil
	}

	destStorage, destAccess, result, err := prepareStagingArgumentFn(dwArgs["destination"])
	if err != nil {
		return nil, nnfv1alpha1.NewWorkflowError("Could not prepare data movement resources").WithError(err)
	} else if result != nil {
		return result, nil
	}

	// Wait for accesses to go ready
	for _, access := range []*nnfv1alpha1.NnfAccess{sourceAccess, destAccess} {
		if access != nil {
			if err := r.Get(ctx, client.ObjectKeyFromObject(access), access); err != nil {
				return nil, fmt.Errorf("Could not get NnfAccess %v: %w", client.ObjectKeyFromObject(access), err)
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
	if workflow.Spec.DesiredState == dwsv1alpha1.StateDataIn {
		targetStorageRef = destStorage
	} else {
		targetStorageRef = sourceStorage
	}

	targetStorage := &nnfv1alpha1.NnfStorage{}
	if err := r.Get(ctx, types.NamespacedName{Name: targetStorageRef.Name, Namespace: targetStorageRef.Namespace}, targetStorage); err != nil {
		return nil, nnfv1alpha1.NewWorkflowError("Data Movement: Failed to retrieve NNF Storage").WithError(err)
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
			msg := fmt.Sprintf("Data Movement: File System %s has unexpected allocation sets %d", fsType, len(targetStorage.Spec.AllocationSets))
			return nil, nnfv1alpha1.NewWorkflowError(msg).WithFatal()
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

				dwsv1alpha1.AddWorkflowLabels(dm, workflow)
				dwsv1alpha1.AddOwnerLabels(dm, workflow)
				nnfv1alpha1.AddDataMovementTeardownStateLabel(dm, workflow.Status.State)
				addDirectiveIndexLabel(dm, index)

				log.Info("Creating NNF Data Movement", "name", client.ObjectKeyFromObject(dm).String())
				if err := r.Create(ctx, dm); err != nil {
					if !errors.IsAlreadyExists(err) {
						return nil, nnfv1alpha1.NewWorkflowError("Data Movement failed to create").WithError(err)
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

		dwsv1alpha1.AddWorkflowLabels(dm, workflow)
		dwsv1alpha1.AddOwnerLabels(dm, workflow)
		nnfv1alpha1.AddDataMovementTeardownStateLabel(dm, workflow.Status.State)
		addDirectiveIndexLabel(dm, index)

		log.Info("Creating NNF Data Movement", "name", client.ObjectKeyFromObject(dm).String())
		if err := r.Create(ctx, dm); err != nil {
			if !errors.IsAlreadyExists(err) {
				return nil, nnfv1alpha1.NewWorkflowError("Data Movement failed to create").WithError(err)
			}
		}
	}

	return nil, nil
}

// Monitor a data movement resource for completion
func (r *NnfWorkflowReconciler) finishDataInOutState(ctx context.Context, workflow *dwsv1alpha1.Workflow, index int) (*result, error) {

	// Wait for data movement resources to complete

	matchingLabels := dwsv1alpha1.MatchingOwner(workflow)
	matchingLabels[nnfv1alpha1.DirectiveIndexLabel] = strconv.Itoa(index)
	matchingLabels[nnfv1alpha1.DataMovementTeardownStateLabel] = string(workflow.Status.State)

	dataMovementList := &nnfv1alpha1.NnfDataMovementList{}
	if err := r.List(ctx, dataMovementList, matchingLabels); err != nil {
		return nil, nnfv1alpha1.NewWorkflowError("Could not retrieve data movements").WithError(err)
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
			return nil, nnfv1alpha1.NewWorkflowError(fmt.Sprintf("Staging operation failed")).WithFatal()
		}
	}

	// Unmount any NNF Accesses that are no longer needed and cannot be shared (i.e. an XFS volume)
	// These NNF Accesses should have a teardown label with the current data_in/data_out state.
	dwArgs, err := dwdparse.BuildArgsMap(workflow.Spec.DWDirectives[index])
	if err != nil {
		return nil, nnfv1alpha1.NewWorkflowErrorf("Invalid DW directive index %d: %s", index, workflow.Spec.DWDirectives[index]).WithFatal()
	}

	unmountIfNecessary := func(param string) (*result, error) {
		name, _ := splitStagingArgumentIntoNameAndPath(param)

		if strings.HasPrefix(param, "$JOB_DW_") || strings.HasPrefix(param, "$PERSISTENT_DW_") {
			parentDwIndex := findDirectiveIndexByName(workflow, name)
			if parentDwIndex < 0 {
				return nil, nnfv1alpha1.NewWorkflowErrorf("No directive matching '%s' found in workflow", name).WithFatal()
			}

			return r.unmountNnfAccessIfNecessary(ctx, workflow, parentDwIndex, "servers")
		}

		return nil, nil
	}

	if result, err := unmountIfNecessary(dwArgs["source"]); result != nil || err != nil {
		return result, err
	}

	if result, err := unmountIfNecessary(dwArgs["destination"]); result != nil || err != nil {
		return result, err
	}

	return nil, nil
}

func (r *NnfWorkflowReconciler) startPreRunState(ctx context.Context, workflow *dwsv1alpha1.Workflow, index int) (*result, error) {
	log := r.Log.WithValues("Workflow", client.ObjectKeyFromObject(workflow), "Index", index)
	dwArgs, _ := dwdparse.BuildArgsMap(workflow.Spec.DWDirectives[index])

	access := &nnfv1alpha1.NnfAccess{
		ObjectMeta: metav1.ObjectMeta{
			Name:      indexedResourceName(workflow, index) + "-computes",
			Namespace: workflow.Namespace,
		},
	}

	// Create an NNFAccess for the compute clients
	result, err := ctrl.CreateOrUpdate(ctx, r.Client, access,
		func() error {
			dwsv1alpha1.AddWorkflowLabels(access, workflow)
			dwsv1alpha1.AddOwnerLabels(access, workflow)
			addDirectiveIndexLabel(access, index)

			access.Spec.TeardownState = dwsv1alpha1.StatePostRun
			access.Spec.DesiredState = "mounted"
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
		err = fmt.Errorf("Could not CreateOrUpdate compute node NnfAccess %v: %w", client.ObjectKeyFromObject(access), err)
		return nil, nnfv1alpha1.NewWorkflowError("Could not mount file system on compute nodes").WithError(err)
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
	// particular $JOB_DW_[name]; in this case we only need to recreate the resource

	fsType, err := r.getDirectiveFileSystemType(ctx, workflow, index)
	if err != nil {
		return nil, nnfv1alpha1.NewWorkflowError("Unable to determine directive file system type").WithError(err)
	}

	if fsType == "gfs2" || fsType == "lustre" {
		name, namespace := getStorageReferenceNameFromWorkflowActual(workflow, index)

		storage := &nnfv1alpha1.NnfStorage{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
		}

		// Set the teardown state to post run. If there is a copy_out directive that uses
		// this storage instance, set the teardown state so NNF Access is preserved through
		// DataOut.
		teardownState := dwsv1alpha1.StatePostRun
		if findCopyOutDirectiveIndexByName(workflow, dwArgs["name"]) >= 0 {
			teardownState = dwsv1alpha1.StateDataOut
		}

		_, err := r.setupNnfAccessForServers(ctx, storage, workflow, index, index, teardownState, log)
		if err != nil {
			return nil, nnfv1alpha1.NewWorkflowError(fmt.Sprintf("Could not setup NNF Access in state %s", workflow.Status.State)).WithError(err)
		}
	}

	readyResult, err := r.waitForNnfAccessStateAndReady(ctx, workflow, index, "mounted")
	if err != nil {
		return nil, nnfv1alpha1.NewWorkflowError("Failed to achieve NnfAccess 'mounted' state").WithError(err).WithFatal()
	} else if readyResult != nil {
		return readyResult, nil
	}

	// TODO: Create container DS or Job. Need to determine which is the best route to handle communication and termination of the container.
	if dwArgs["command"] == "container" {
		if err := r.createOrUpdateDaemonSetIfNecessary(ctx, workflow); err != nil {
			return nil, nnfv1alpha1.NewWorkflowError("Unable to create/update Container DaemonSet").WithError(err)
		}
	}

	return nil, nil
}

func (r *NnfWorkflowReconciler) finishPreRunState(ctx context.Context, workflow *dwsv1alpha1.Workflow, index int) (*result, error) {

	dwArgs, _ := dwdparse.BuildArgsMap(workflow.Spec.DWDirectives[index])

	// Add an environment variable to the workflow status section for the location of the
	// mount point on the clients.
	if workflow.Status.Env == nil {
		workflow.Status.Env = make(map[string]string)
	}

	envName := ""
	switch dwArgs["command"] {
	case "jobdw":
		envName = "DW_JOB_" + dwArgs["name"]
	case "persistentdw":
		envName = "DW_PERSISTENT_" + dwArgs["name"]
	default:
		return nil, nnfv1alpha1.NewWorkflowErrorf("Unexpected directive %v", dwArgs["command"])
	}

	workflow.Status.Env[envName] = buildMountPath(workflow, index)

	return nil, nil
}

func (r *NnfWorkflowReconciler) createOrUpdateDaemonSetIfNecessary(ctx context.Context, workflow *dwsv1alpha1.Workflow) error {
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
		podTemplateSpec := container.Spec.Template.DeepCopy()
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

func (r *NnfWorkflowReconciler) startPostRunState(ctx context.Context, workflow *dwsv1alpha1.Workflow, index int) (*result, error) {

	// Unmount the NnfAccess for the compute nodes. This will free the compute nodes to be used
	// in a different job even if there is data movement happening on the Rabbits.
	if result, err := r.unmountNnfAccessIfNecessary(ctx, workflow, index, "computes"); result != nil || err != nil {
		return result, err
	}

	// Wait for data movement resources to complete
	matchingLabels := dwsv1alpha1.MatchingOwner(workflow)
	matchingLabels[nnfv1alpha1.DataMovementTeardownStateLabel] = string(dwsv1alpha1.StatePostRun)

	dataMovementList := &nnfv1alpha1.NnfDataMovementList{}
	if err := r.List(ctx, dataMovementList, matchingLabels); err != nil {
		return nil, nnfv1alpha1.NewWorkflowError("Could not retrieve data movements").WithError(err)
	}

	for _, dm := range dataMovementList.Items {
		if dm.Status.State != nnfv1alpha1.DataMovementConditionTypeFinished {
			return Requeue("pending data movement").withObject(&dm), nil
		}
	}

	// Unmount the NnfAccess for the server nodes if necessary
	if result, err := r.unmountNnfAccessIfNecessary(ctx, workflow, index, "servers"); result != nil || err != nil {
		return result, err
	}

	return nil, nil
}

func (r *NnfWorkflowReconciler) finishPostRunState(ctx context.Context, workflow *dwsv1alpha1.Workflow, index int) (*result, error) {

	result, err := r.waitForNnfAccessStateAndReady(ctx, workflow, index, "unmounted")
	if err != nil {
		return nil, nnfv1alpha1.NewWorkflowError("Failed to achieve NnfAccess 'unmounted' state").WithError(err).WithFatal()
	} else if result != nil {
		return result, nil
	}

	// Any user created copy-offload data movement requests created during run must report any errors to the workflow.
	// TODO: Customer asked if this could be optional
	matchingLabels := dwsv1alpha1.MatchingOwner(workflow)
	matchingLabels[nnfv1alpha1.DataMovementTeardownStateLabel] = string(dwsv1alpha1.StatePostRun)

	dataMovementList := &nnfv1alpha1.NnfDataMovementList{}
	if err := r.List(ctx, dataMovementList, matchingLabels); err != nil {
		return nil, nnfv1alpha1.NewWorkflowError("Could not retrieve data movements").WithError(err)
	}

	for _, dm := range dataMovementList.Items {
		if dm.Status.State != nnfv1alpha1.DataMovementConditionTypeFinished {
			return Requeue("pending data movement").withObject(&dm), nil
		}

		if dm.Status.Status == nnfv1alpha1.DataMovementConditionReasonFailed {
			err := fmt.Errorf("Data movement %s failed", client.ObjectKeyFromObject(&dm).String())
			return nil, nnfv1alpha1.NewWorkflowError("Data movement unsuccessful").WithError(err).WithFatal()
		}
	}

	return nil, nil
}

func (r *NnfWorkflowReconciler) startTeardownState(ctx context.Context, workflow *dwsv1alpha1.Workflow, index int) (*result, error) {

	// Delete the NnfDataMovement and NnfAccess for this directive before removing the NnfStorage.
	// copy_in/out directives can reference NnfStorage from a different directive, so all the NnfAccesses
	// need to be removed first.
	childObjects := []dwsv1alpha1.ObjectList{
		&nnfv1alpha1.NnfDataMovementList{},
		&nnfv1alpha1.NnfAccessList{},
	}

	deleteStatus, err := dwsv1alpha1.DeleteChildrenWithLabels(ctx, r.Client, childObjects, workflow, client.MatchingLabels{nnfv1alpha1.DirectiveIndexLabel: strconv.Itoa(index)})
	if err != nil {
		err = fmt.Errorf("Could not delete NnfDataMovement and NnfAccess children: %w", err)
		return nil, nnfv1alpha1.NewWorkflowError("Could not stop data movement and unmount file systems").WithError(err)
	}

	if !deleteStatus.Complete() {
		return Requeue("delete").withDeleteStatus(deleteStatus), nil
	}

	return nil, nil
}

func (r *NnfWorkflowReconciler) finishTeardownState(ctx context.Context, workflow *dwsv1alpha1.Workflow, index int) (*result, error) {
	log := r.Log.WithValues("Workflow", client.ObjectKeyFromObject(workflow), "Index", index)
	dwArgs, _ := dwdparse.BuildArgsMap(workflow.Spec.DWDirectives[index])

	switch dwArgs["command"] {
	case "create_persistent":
		for _, driverStatus := range workflow.Status.Drivers {
			if driverStatus.WatchState == dwsv1alpha1.StateTeardown {
				continue
			}

			if !driverStatus.Completed {
				return nil, nil
			}
		}

		persistentStorage, err := r.findPersistentInstance(ctx, workflow, dwArgs["name"])
		if err != nil {
			return nil, nnfv1alpha1.NewWorkflowError(fmt.Sprintf("Could not find persistent storage %v", dwArgs["name"])).WithError(err)
		}

		persistentStorage.SetOwnerReferences([]metav1.OwnerReference{})
		dwsv1alpha1.RemoveOwnerLabels(persistentStorage)
		labels := persistentStorage.GetLabels()
		delete(labels, nnfv1alpha1.DirectiveIndexLabel)
		persistentStorage.SetLabels(labels)

		err = r.Update(ctx, persistentStorage)
		if err != nil {
			err = fmt.Errorf("Could not update PersistentStorage %v: %w", client.ObjectKeyFromObject(persistentStorage), err)
			return nil, nnfv1alpha1.NewWorkflowError("Could not finalize peristent storage").WithError(err)
		}
		log.Info("Removed owner reference from persistent storage", "psi", persistentStorage)
	case "destroy_persistent":
		persistentStorage, err := r.findPersistentInstance(ctx, workflow, dwArgs["name"])
		if err != nil {
			if !apierrors.IsNotFound(err) {
				return nil, nnfv1alpha1.NewWorkflowError(fmt.Sprintf("Could not find peristent storage %v", dwArgs["name"])).WithError(err)
			}

			return nil, nil
		}

		if persistentStorage.Spec.UserID != workflow.Spec.UserID {
			err = fmt.Errorf("Existing persistent storage user ID %v does not match user ID %v", persistentStorage.Spec.UserID, workflow.Spec.UserID)
			log.Info(err.Error())
			return nil, nnfv1alpha1.NewWorkflowError("user ID does not match existing persistent storage").WithError(err).WithFatal()
		}

		if len(persistentStorage.Spec.ConsumerReferences) != 0 {
			err = fmt.Errorf("PersistentStorage cannot be deleted with %v consumers", len(persistentStorage.Spec.ConsumerReferences))
			log.Info(err.Error())
			return nil, nnfv1alpha1.NewWorkflowError("PersistentStorage cannot be deleted while in use").WithError(err).WithFatal()
		}

		persistentStorage.Spec.State = dwsv1alpha1.PSIStateDestroying

		dwsv1alpha1.AddOwnerLabels(persistentStorage, workflow)
		addDirectiveIndexLabel(persistentStorage, index)

		if err := controllerutil.SetControllerReference(workflow, persistentStorage, r.Scheme); err != nil {
			log.Info("Unable to assign workflow as owner of persistentInstance", "psi", persistentStorage)
			err = fmt.Errorf("Could not assign workflow as owner of PersistentInstance %v: %w", client.ObjectKeyFromObject(persistentStorage), err)
			return nil, nnfv1alpha1.NewWorkflowError(fmt.Sprintf("Could not delete peristent storage %v", dwArgs["name"])).WithError(err)
		}

		err = r.Update(ctx, persistentStorage)
		if err != nil {
			err = fmt.Errorf("Could not update PersistentInstance %v: %w", client.ObjectKeyFromObject(persistentStorage), err)
			return nil, nnfv1alpha1.NewWorkflowError(fmt.Sprintf("Could not delete peristent storage %v", dwArgs["name"])).WithError(err)
		}
		log.Info("Add owner reference for persistent storage for deletion", "psi", persistentStorage)
	case "persistentdw":
		err := r.removePersistentStorageReference(ctx, workflow, index)
		if err != nil {
			return nil, nnfv1alpha1.NewWorkflowError("Could not remove persistent storage reference").WithError(err)
		}
	default:
	}

	childObjects := []dwsv1alpha1.ObjectList{
		&nnfv1alpha1.NnfStorageList{},
		&dwsv1alpha1.PersistentStorageInstanceList{},
	}

	deleteStatus, err := dwsv1alpha1.DeleteChildrenWithLabels(ctx, r.Client, childObjects, workflow, client.MatchingLabels{nnfv1alpha1.DirectiveIndexLabel: strconv.Itoa(index)})
	if err != nil {
		return nil, nnfv1alpha1.NewWorkflowError("Could not delete storage allocations").WithError(err)
	}

	if !deleteStatus.Complete() {
		return Requeue("delete").withDeleteStatus(deleteStatus), nil
	}

	return nil, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NnfWorkflowReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.ChildObjects = []dwsv1alpha1.ObjectList{
		&nnfv1alpha1.NnfDataMovementList{},
		&nnfv1alpha1.NnfAccessList{},
		&nnfv1alpha1.NnfStorageList{},
		&dwsv1alpha1.PersistentStorageInstanceList{},
		&dwsv1alpha1.DirectiveBreakdownList{},
	}

	maxReconciles := runtime.GOMAXPROCS(0)
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(controller.Options{MaxConcurrentReconciles: maxReconciles}).
		For(&dwsv1alpha1.Workflow{}).
		Owns(&nnfv1alpha1.NnfAccess{}).
		Owns(&dwsv1alpha1.DirectiveBreakdown{}).
		Owns(&dwsv1alpha1.PersistentStorageInstance{}).
		Watches(&source.Kind{Type: &nnfv1alpha1.NnfDataMovement{}}, handler.EnqueueRequestsFromMapFunc(dwsv1alpha1.OwnerLabelMapFunc)).
		Watches(&source.Kind{Type: &nnfv1alpha1.NnfStorage{}}, handler.EnqueueRequestsFromMapFunc(dwsv1alpha1.OwnerLabelMapFunc)).
		Complete(r)
}
