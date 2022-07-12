/*
 * Copyright 2022 Hewlett Packard Enterprise Development LP
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

	dwsv1alpha1 "github.com/HewlettPackard/dws/api/v1alpha1"
	"github.com/HewlettPackard/dws/utils/dwdparse"
	nnfv1alpha1 "github.com/NearNodeFlash/nnf-sos/api/v1alpha1"
)

const (
	// finalizerPersistentStorage is the finalizer stirng used by this controller
	finalizerPersistentStorage = "nnf.cray.hpe.com/persistentStorage"
)

// DirectiveBreakdownReconciler reconciles a DirectiveBreakdown object
type PersistentStorageReconciler struct {
	client.Client
	Log          logr.Logger
	Scheme       *kruntime.Scheme
	ChildObjects []dwsv1alpha1.ObjectList
}

//+kubebuilder:rbac:groups=dws.cray.hpe.com,resources=persistentstorageinstances,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=dws.cray.hpe.com,resources=persistentstorageinstances/status,verbs=get;list;watch;create;update;patch
//+kubebuilder:rbac:groups=dws.cray.hpe.com,resources=servers,verbs=get;list;watch;create;update;patch;delete;deletecollection
//+kubebuilder:rbac:groups=dws.cray.hpe.com,resources=nnfstorage,verbs=get;list;watch;update;delete;deletecollection

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the directiveBreakdown closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.9.2/pkg/reconcile
func (r *PersistentStorageReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, err error) {
	log := r.Log.WithValues("PersistentStorage", req.NamespacedName)

	persistentStorage := &dwsv1alpha1.PersistentStorageInstance{}
	if err := r.Get(ctx, req.NamespacedName, persistentStorage); err != nil {
		// ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	statusUpdater := newPersistentStorageStatusUpdater(persistentStorage)
	defer func() {
		if err == nil {
			err = statusUpdater.close(ctx, r)
		}
	}()

	if !persistentStorage.GetDeletionTimestamp().IsZero() {
		log.Info("Deleting")
		if !controllerutil.ContainsFinalizer(persistentStorage, finalizerPersistentStorage) {
			return ctrl.Result{}, nil
		}

		// Delete all NnfStorage and Servers children that are owned by this PersistentStorage.
		deleteStatus, err := dwsv1alpha1.DeleteChildren(ctx, r.Client, r.ChildObjects, persistentStorage)
		if err != nil {
			return ctrl.Result{}, err
		}

		if deleteStatus == dwsv1alpha1.DeleteRetry {
			return ctrl.Result{}, nil
		}

		controllerutil.RemoveFinalizer(persistentStorage, finalizerPersistentStorage)
		if err := r.Update(ctx, persistentStorage); err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	// Add the finalizer if it doesn't exist
	if !controllerutil.ContainsFinalizer(persistentStorage, finalizerPersistentStorage) {
		controllerutil.AddFinalizer(persistentStorage, finalizerPersistentStorage)
		if err := r.Update(ctx, persistentStorage); err != nil {
			return ctrl.Result{Requeue: true}, nil
		}

		return ctrl.Result{}, nil
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

	// Create the Servers resource
	servers, err := r.createServers(ctx, persistentStorage)
	if err != nil {
		return ctrl.Result{}, err
	}

	if servers == nil {
		return ctrl.Result{Requeue: true}, nil
	}

	persistentStorage.Status.Servers = v1.ObjectReference{
		Kind:      reflect.TypeOf(dwsv1alpha1.Servers{}).Name(),
		Name:      servers.Name,
		Namespace: servers.Namespace,
	}

	return ctrl.Result{}, err
}

func (r *PersistentStorageReconciler) createServers(ctx context.Context, persistentStorage *dwsv1alpha1.PersistentStorageInstance) (*dwsv1alpha1.Servers, error) {
	log := r.Log.WithValues("PersistentStorage", client.ObjectKeyFromObject(persistentStorage))
	server := &dwsv1alpha1.Servers{
		ObjectMeta: metav1.ObjectMeta{
			Name:      persistentStorage.Name,
			Namespace: persistentStorage.Namespace,
		},
	}

	// Create the Servers resource with owner labels and PersistentStorage labels
	result, err := ctrl.CreateOrUpdate(ctx, r.Client, server,
		func() error {
			dwsv1alpha1.AddOwnerLabels(server, persistentStorage)
			dwsv1alpha1.AddPersistentStorageLabels(server, persistentStorage)

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

type persistentStorageStatusUpdater struct {
	persistentStorage *dwsv1alpha1.PersistentStorageInstance
	existingStatus    dwsv1alpha1.PersistentStorageInstanceStatus
}

func newPersistentStorageStatusUpdater(p *dwsv1alpha1.PersistentStorageInstance) *persistentStorageStatusUpdater {
	return &persistentStorageStatusUpdater{
		persistentStorage: p,
		existingStatus:    (*p.DeepCopy()).Status,
	}
}

func (p *persistentStorageStatusUpdater) close(ctx context.Context, r *PersistentStorageReconciler) error {
	if !reflect.DeepEqual(p.persistentStorage.Status, p.existingStatus) {
		err := r.Status().Update(ctx, p.persistentStorage)
		if !apierrors.IsConflict(err) {
			return err
		}
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PersistentStorageReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.ChildObjects = []dwsv1alpha1.ObjectList{
		&nnfv1alpha1.NnfStorageList{},
		&dwsv1alpha1.ServersList{},
		&nnfv1alpha1.NnfStorageProfileList{},
	}

	maxReconciles := runtime.GOMAXPROCS(0)
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(controller.Options{MaxConcurrentReconciles: maxReconciles}).
		For(&dwsv1alpha1.PersistentStorageInstance{}).
		Owns(&dwsv1alpha1.Servers{}).
		Owns(&nnfv1alpha1.NnfStorageProfile{}).
		Complete(r)
}
