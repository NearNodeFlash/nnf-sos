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

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"

	dwsv1alpha1 "github.com/HewlettPackard/dws/api/v1alpha1"
	"github.com/NearNodeFlash/nnf-sos/controllers/metrics"
)

const (
	// finalizerNnfSystemConfiguration defines the finalizer name that this controller
	// uses on the SystemConfiguration resource. This prevents the SystemConfiguration resource
	// from being fully deleted until this controller removes the finalizer.
	finalizerNnfSystemConfiguration = "nnf.cray.hpe.com/nnf_systemconfiguration"
)

// NnfSystemConfigurationReconciler contains the pieces used by the reconciler
type NnfSystemConfigurationReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *kruntime.Scheme
}

//+kubebuilder:rbac:groups=dws.cray.hpe.com,resources=systemconfigurations,verbs=get;list;watch;update
//+kubebuilder:rbac:groups=dws.cray.hpe.com,resources=systemconfigurations/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=dws.cray.hpe.com,resources=systemconfigurations/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=namespaces,verbs=get;list;watch;create;update;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *NnfSystemConfigurationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("SystemConfiguration", req.NamespacedName)

	metrics.NnfSystemConfigurationReconcilesTotal.Inc()

	systemConfiguration := &dwsv1alpha1.SystemConfiguration{}
	if err := r.Get(ctx, req.NamespacedName, systemConfiguration); err != nil {
		// ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Handle cleanup if the resource is being deleted
	if !systemConfiguration.GetDeletionTimestamp().IsZero() {
		if !controllerutil.ContainsFinalizer(systemConfiguration, finalizerNnfSystemConfiguration) {
			return ctrl.Result{}, nil
		}

		log.Info("Removing all compute node namespaces due to object deletion")
		if err := r.deleteNamespaces(ctx, systemConfiguration, map[string]struct{}{}); err != nil {
			return ctrl.Result{}, err
		}

		controllerutil.RemoveFinalizer(systemConfiguration, finalizerNnfSystemConfiguration)
		if err := r.Update(ctx, systemConfiguration); err != nil {
			if !apierrors.IsConflict(err) {
				return ctrl.Result{}, err
			}

			return ctrl.Result{Requeue: true}, nil
		}

		return ctrl.Result{}, nil
	}

	// Add finalizer if it doesn't exist
	if !controllerutil.ContainsFinalizer(systemConfiguration, finalizerNnfSystemConfiguration) {
		controllerutil.AddFinalizer(systemConfiguration, finalizerNnfSystemConfiguration)
		if err := r.Update(ctx, systemConfiguration); err != nil {
			if !apierrors.IsConflict(err) {
				return ctrl.Result{}, err
			}

			return ctrl.Result{Requeue: true}, nil
		}

		return ctrl.Result{}, nil
	}

	// make a map of compute node names that need a namespace. The map only contains
	// keys and empty values, but it makes it easy to search the names.
	validNamespaces := make(map[string]struct{})
	for _, compute := range systemConfiguration.Spec.ComputeNodes {
		validNamespaces[compute.Name] = struct{}{}
	}

	// Delete any namespaces owned by this systemConfiguration resource that aren't included
	// in the validNamespaces map
	if err := r.deleteNamespaces(ctx, systemConfiguration, validNamespaces); err != nil {
		return ctrl.Result{}, err
	}

	// Create the namespaces in the validNamespaces map if they don't exist yet.
	if err := r.createNamespaces(ctx, systemConfiguration, validNamespaces); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// deleteNamespaces looks for namespaces owned by the SystemConfiguration resource (by looking
// at the namespace's labels) and deletes them if it isn't a namespace listed in the validNamespaces
// map. Only the first error encountered is returned.
func (r *NnfSystemConfigurationReconciler) deleteNamespaces(ctx context.Context, config *dwsv1alpha1.SystemConfiguration, validNamespaces map[string]struct{}) error {
	listOptions := []client.ListOption{
		dwsv1alpha1.MatchingOwner(config),
	}

	namespaces := &corev1.NamespaceList{}
	if err := r.List(ctx, namespaces, listOptions...); err != nil {
		return err
	}

	var firstError error
	for _, namespace := range namespaces.Items {
		if _, ok := validNamespaces[namespace.Name]; ok {
			continue
		}

		if err := r.Delete(ctx, &namespace); err != nil {
			if !apierrors.IsNotFound(err) {
				if firstError == nil {
					firstError = err
				}
			}
		}
	}

	return firstError
}

// createNamespaces creates a namespace for each entry in the validNamespaces map. The
// namespaces have a "name" and "namespace" label for the SystemConfiguration owner.
func (r *NnfSystemConfigurationReconciler) createNamespaces(ctx context.Context, config *dwsv1alpha1.SystemConfiguration, validNamespaces map[string]struct{}) error {
	for name := range validNamespaces {
		namespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
		}
		_, err := ctrl.CreateOrUpdate(ctx, r.Client, namespace,
			func() error {
				dwsv1alpha1.AddOwnerLabels(namespace, config)
				return nil
			})
		if err != nil {
			return err
		}
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NnfSystemConfigurationReconciler) SetupWithManager(mgr ctrl.Manager) error {

	builder := ctrl.NewControllerManagedBy(mgr).
		WithOptions(controller.Options{MaxConcurrentReconciles: 1}).
		Watches(&source.Kind{Type: &corev1.Namespace{}}, handler.EnqueueRequestsFromMapFunc(dwsv1alpha1.OwnerLabelMapFunc)).
		For(&dwsv1alpha1.SystemConfiguration{})

	return builder.Complete(r)
}
