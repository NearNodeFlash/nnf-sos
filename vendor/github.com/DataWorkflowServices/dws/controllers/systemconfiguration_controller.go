/*
 * Copyright 2024 Hewlett Packard Enterprise Development LP
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

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	dwsv1alpha2 "github.com/DataWorkflowServices/dws/api/v1alpha2"
	"github.com/DataWorkflowServices/dws/internal/controller/metrics"
	"github.com/DataWorkflowServices/dws/utils/updater"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	// finalizerDWSSystemConfiguration defines the finalizer name that this controller
	// uses on the SystemConfiguration resource. This prevents the SystemConfiguration resource
	// from being fully deleted until this controller removes the finalizer.
	finalizerDWSSystemConfiguration = "dataworkflowservices.github.io/systemconfiguration"
)

// SystemConfigurationReconciler reconciles a SystemConfiguration object
type SystemConfigurationReconciler struct {
	client.Client
	Log          logr.Logger
	Scheme       *runtime.Scheme
	ChildObjects []dwsv1alpha2.ObjectList
}

// +kubebuilder:rbac:groups=dataworkflowservices.github.io,resources=systemconfigurations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=dataworkflowservices.github.io,resources=systemconfigurations/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=dataworkflowservices.github.io,resources=systemconfigurations/finalizers,verbs=update
// +kubebuilder:rbac:groups=dataworkflowservices.github.io,resources=storages,verbs=get;list;watch;create;update;patch;delete;deletecollection

func (r *SystemConfigurationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, err error) {
	log := r.Log.WithValues("SystemConfiguration", req.NamespacedName)

	metrics.DwsReconcilesTotal.Inc()

	systemConfiguration := &dwsv1alpha2.SystemConfiguration{}
	if err := r.Get(ctx, req.NamespacedName, systemConfiguration); err != nil {
		// ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Create a status updater that handles the call to r.Status().Update() if any of the fields
	// in systemConfiguration.Status{} change
	statusUpdater := updater.NewStatusUpdater[*dwsv1alpha2.SystemConfigurationStatus](systemConfiguration)
	defer func() { err = statusUpdater.CloseWithStatusUpdate(ctx, r.Client.Status(), err) }()
	defer func() { systemConfiguration.Status.SetResourceErrorAndLog(err, log) }()

	// Handle cleanup if the resource is being deleted
	if !systemConfiguration.GetDeletionTimestamp().IsZero() {
		if !controllerutil.ContainsFinalizer(systemConfiguration, finalizerDWSSystemConfiguration) {
			return ctrl.Result{}, nil
		}

		deleteStatus, err := dwsv1alpha2.DeleteChildren(ctx, r.Client, r.ChildObjects, systemConfiguration)
		if err != nil {
			return ctrl.Result{}, err
		}

		if !deleteStatus.Complete() {
			return ctrl.Result{}, nil
		}

		controllerutil.RemoveFinalizer(systemConfiguration, finalizerDWSSystemConfiguration)
		if err := r.Update(ctx, systemConfiguration); err != nil {
			if !apierrors.IsConflict(err) {
				return ctrl.Result{}, err
			}

			return ctrl.Result{Requeue: true}, nil
		}

		return ctrl.Result{}, nil
	}

	// Add finalizer if it doesn't exist
	if !controllerutil.ContainsFinalizer(systemConfiguration, finalizerDWSSystemConfiguration) {
		controllerutil.AddFinalizer(systemConfiguration, finalizerDWSSystemConfiguration)
		if err := r.Update(ctx, systemConfiguration); err != nil {
			if !apierrors.IsConflict(err) {
				return ctrl.Result{}, err
			}

			return ctrl.Result{Requeue: true}, nil
		}

		return ctrl.Result{}, nil
	}

	for _, storageNode := range systemConfiguration.Spec.StorageNodes {
		// Create a storage resource for each storage node listed in the system configuration
		storage := &dwsv1alpha2.Storage{
			ObjectMeta: metav1.ObjectMeta{
				Name:      storageNode.Name,
				Namespace: corev1.NamespaceDefault,
			},
		}

		result, err := ctrl.CreateOrUpdate(ctx, r.Client, storage,
			func() error {
				dwsv1alpha2.AddOwnerLabels(storage, systemConfiguration)
				labels := storage.GetLabels()
				labels[dwsv1alpha2.StorageTypeLabel] = storageNode.Type
				storage.SetLabels(labels)

				return ctrl.SetControllerReference(systemConfiguration, storage, r.Scheme)
			})

		if err != nil {
			return ctrl.Result{}, dwsv1alpha2.NewResourceError("CreateOrUpdate failed for storage: %v", client.ObjectKeyFromObject(storage)).WithError(err)
		}

		if result == controllerutil.OperationResultCreated {
			log.Info("Created storage", "name", storage.Name)
		} else if result == controllerutil.OperationResultNone {
			// no change
		} else {
			log.Info("Updated storage", "name", storage.Name)
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SystemConfigurationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.ChildObjects = []dwsv1alpha2.ObjectList{
		&dwsv1alpha2.StorageList{},
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&dwsv1alpha2.SystemConfiguration{}).
		Owns(&dwsv1alpha2.Storage{}).
		Complete(r)
}
