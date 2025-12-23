/*
 * Copyright 2022-2025 Hewlett Packard Enterprise Development LP
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
	"reflect"
	"runtime"

	"github.com/go-logr/logr"

	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	dwsv1alpha7 "github.com/DataWorkflowServices/dws/api/v1alpha7"
	"github.com/DataWorkflowServices/dws/utils/dwdparse"
	"github.com/DataWorkflowServices/dws/utils/updater"
	nnfv1alpha10 "github.com/NearNodeFlash/nnf-sos/api/v1alpha10"
	"github.com/NearNodeFlash/nnf-sos/internal/controller/metrics"
)

const (
	// finalizerPersistentStorage is the finalizer stirng used by this controller
	finalizerPersistentStorage = "nnf.cray.hpe.com/persistentStorage"
)

// DirectiveBreakdownReconciler reconciles a DirectiveBreakdown object
type PersistentStorageReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *kruntime.Scheme
}

//+kubebuilder:rbac:groups=dataworkflowservices.github.io,resources=persistentstorageinstances,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=dataworkflowservices.github.io,resources=persistentstorageinstances/status,verbs=get;list;watch;create;update;patch
//+kubebuilder:rbac:groups=dataworkflowservices.github.io,resources=servers,verbs=get;list;watch;create;update;patch;delete;deletecollection
//+kubebuilder:rbac:groups=dataworkflowservices.github.io,resources=nnfstorage,verbs=get;list;watch;update;delete;deletecollection

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the directiveBreakdown closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.9.2/pkg/reconcile
func (r *PersistentStorageReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, err error) {
	log := r.Log.WithValues("PersistentStorage", req.NamespacedName)

	metrics.NnfPersistentStorageReconcilesTotal.Inc()

	persistentStorage := &dwsv1alpha7.PersistentStorageInstance{}
	if err := r.Get(ctx, req.NamespacedName, persistentStorage); err != nil {
		// ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	statusUpdater := updater.NewStatusUpdater[*dwsv1alpha7.PersistentStorageInstanceStatus](persistentStorage)
	defer func() { err = statusUpdater.CloseWithStatusUpdate(ctx, r.Client.Status(), err) }()
	defer func() { persistentStorage.Status.SetResourceError(err) }()

	if !persistentStorage.GetDeletionTimestamp().IsZero() {
		log.Info("Deleting")
		if !controllerutil.ContainsFinalizer(persistentStorage, finalizerPersistentStorage) {
			return ctrl.Result{}, nil
		}

		if len(persistentStorage.Spec.ConsumerReferences) != 0 {
			log.Info("Unable to delete persistent storage with consumer references")
			return ctrl.Result{}, nil
		}

		// Delete all NnfStorage and Servers children that are owned by this PersistentStorage.
		deleteStatus, err := dwsv1alpha7.DeleteChildren(ctx, r.Client, r.getChildObjects(), persistentStorage)
		if err != nil {
			return ctrl.Result{}, err
		}

		if !deleteStatus.Complete() {
			return ctrl.Result{}, nil
		}

		controllerutil.RemoveFinalizer(persistentStorage, finalizerPersistentStorage)
		if err := r.Update(ctx, persistentStorage); err != nil {
			if !apierrors.IsConflict(err) {
				return ctrl.Result{}, err
			}

			return ctrl.Result{Requeue: true}, nil
		}

		return ctrl.Result{}, nil
	}

	// Add the finalizer if it doesn't exist
	if !controllerutil.ContainsFinalizer(persistentStorage, finalizerPersistentStorage) {
		controllerutil.AddFinalizer(persistentStorage, finalizerPersistentStorage)
		if err := r.Update(ctx, persistentStorage); err != nil {
			if !apierrors.IsConflict(err) {
				return ctrl.Result{}, err
			}

			return ctrl.Result{Requeue: true}, nil
		}

		return ctrl.Result{}, nil
	}

	if persistentStorage.Status.State == "" {
		persistentStorage.Status.State = dwsv1alpha7.PSIStateCreating
	}

	argsMap, err := dwdparse.BuildArgsMap(persistentStorage.Spec.DWDirective)
	if err != nil {
		return ctrl.Result{}, err
	}

	pinnedProfile, err := createPinnedProfile(ctx, r.Client, r.Scheme, argsMap, persistentStorage, persistentStorage.GetName())
	if err != nil {
		return ctrl.Result{}, err
	}

	if pinnedProfile == nil {
		return ctrl.Result{Requeue: true}, nil
	}

	// If this PersistentStorageInstance is for a standalone MGT, add a label so it can be easily found
	if argsMap["type"] == "lustre" && len(pinnedProfile.Data.LustreStorage.StandaloneMGTPoolName) > 0 {
		if _, exists := argsMap["capacity"]; exists {
			return ctrl.Result{}, dwsv1alpha7.NewResourceError("").WithUserMessage("creating persistent MGT does not accept 'capacity' argument").WithFatal().WithUser()
		}
		labels := persistentStorage.GetLabels()
		if _, ok := labels[nnfv1alpha10.StandaloneMGTLabel]; !ok {
			labels[nnfv1alpha10.StandaloneMGTLabel] = pinnedProfile.Data.LustreStorage.StandaloneMGTPoolName
			persistentStorage.SetLabels(labels)
			if err := r.Update(ctx, persistentStorage); err != nil {
				if !apierrors.IsConflict(err) {
					return ctrl.Result{}, err
				}

				return ctrl.Result{Requeue: true}, nil
			}
		}
	} else {
		if _, exists := argsMap["capacity"]; !exists {
			return ctrl.Result{}, dwsv1alpha7.NewResourceError("").WithUserMessage("creating persistent storage requires 'capacity' argument").WithFatal().WithUser()
		}
	}

	// Create the Servers resource
	servers, err := r.createServers(ctx, persistentStorage)
	if err != nil {
		return ctrl.Result{}, err
	}

	if servers == nil {
		return ctrl.Result{Requeue: true}, nil
	}

	persistentStorage.Status.Servers = v1.ObjectReference{
		Kind:      reflect.TypeOf(dwsv1alpha7.Servers{}).Name(),
		Name:      servers.Name,
		Namespace: servers.Namespace,
	}

	if persistentStorage.Spec.State == dwsv1alpha7.PSIStateDestroying {
		if len(persistentStorage.Spec.ConsumerReferences) == 0 {
			persistentStorage.Status.State = dwsv1alpha7.PSIStateDestroying
		}
	} else if persistentStorage.Spec.State == dwsv1alpha7.PSIStateActive {
		// Wait for the NnfStorage to be ready before marking the persistent storage
		// state as "active"
		nnfStorage := &nnfv1alpha10.NnfStorage{}
		if err := r.Get(ctx, req.NamespacedName, nnfStorage); err != nil {
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}

		// If the Status section has not been filled in yet, exit and wait.
		if len(nnfStorage.Status.AllocationSets) != len(nnfStorage.Spec.AllocationSets) {
			return ctrl.Result{}, nil
		}

		var complete bool = true
		// Status section should be usable now, check for Ready
		for _, set := range nnfStorage.Status.AllocationSets {
			if set.Ready == false {
				complete = false
			}
		}

		if complete == true {
			persistentStorage.Status.State = dwsv1alpha7.PSIStateActive
			if nnfStorage.Status.Health != nnfv1alpha10.NnfStorageHealthHealthy {
				persistentStorage.Status.State = dwsv1alpha7.PSIStateDegraded
			}
		}
	}

	return ctrl.Result{}, err
}

func (r *PersistentStorageReconciler) createServers(ctx context.Context, persistentStorage *dwsv1alpha7.PersistentStorageInstance) (*dwsv1alpha7.Servers, error) {
	log := r.Log.WithValues("PersistentStorage", client.ObjectKeyFromObject(persistentStorage))
	server := &dwsv1alpha7.Servers{
		ObjectMeta: metav1.ObjectMeta{
			Name:      persistentStorage.Name,
			Namespace: persistentStorage.Namespace,
		},
	}

	// Create the Servers resource with owner labels and PersistentStorage labels
	result, err := ctrl.CreateOrUpdate(ctx, r.Client, server,
		func() error {
			dwsv1alpha7.AddOwnerLabels(server, persistentStorage)
			dwsv1alpha7.AddPersistentStorageLabels(server, persistentStorage)

			return ctrl.SetControllerReference(persistentStorage, server, r.Scheme)
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

func (r *PersistentStorageReconciler) getChildObjects() []dwsv1alpha7.ObjectList {
	return []dwsv1alpha7.ObjectList{
		&nnfv1alpha10.NnfStorageList{},
		&dwsv1alpha7.ServersList{},
		&nnfv1alpha10.NnfStorageProfileList{},
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *PersistentStorageReconciler) SetupWithManager(mgr ctrl.Manager) error {
	maxReconciles := runtime.GOMAXPROCS(0)
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(controller.Options{MaxConcurrentReconciles: maxReconciles}).
		For(&dwsv1alpha7.PersistentStorageInstance{}).
		Owns(&dwsv1alpha7.Servers{}).
		Owns(&nnfv1alpha10.NnfStorage{}).
		Owns(&nnfv1alpha10.NnfStorageProfile{}).
		Complete(r)
}
