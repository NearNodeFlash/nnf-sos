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

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/kubernetes/pkg/util/taints"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	dwsv1alpha2 "github.com/DataWorkflowServices/dws/api/v1alpha2"
	"github.com/DataWorkflowServices/dws/utils/updater"
	"github.com/NearNodeFlash/nnf-sos/api/v1alpha1"
	"github.com/NearNodeFlash/nnf-sos/internal/controller/metrics"
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

//+kubebuilder:rbac:groups=dataworkflowservices.github.io,resources=systemconfigurations,verbs=get;list;watch;update
//+kubebuilder:rbac:groups=dataworkflowservices.github.io,resources=systemconfigurations/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=dataworkflowservices.github.io,resources=systemconfigurations/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=namespaces,verbs=get;list;watch;create;update;delete
//+kubebuilder:rbac:groups=core,resources=nodes,verbs=get;list;watch;update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *NnfSystemConfigurationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, err error) {
	metrics.NnfSystemConfigurationReconcilesTotal.Inc()

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

	// Handle cleanup if the resource is being deleted
	if !systemConfiguration.GetDeletionTimestamp().IsZero() {
		if !controllerutil.ContainsFinalizer(systemConfiguration, finalizerNnfSystemConfiguration) {
			return ctrl.Result{}, nil
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
	for _, name := range systemConfiguration.Computes() {
		validNamespaces[*name] = struct{}{}
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

	// Make a map of the rabbit node names.
	validRabbits := make(map[string]struct{})
	for _, storageNode := range systemConfiguration.Spec.StorageNodes {
		validRabbits[storageNode.Name] = struct{}{}
	}

	// Add labels and taints to rabbit nodes.
	if requeue, err := r.labelsAndTaints(ctx, validRabbits); requeue || (err != nil) {
		return ctrl.Result{Requeue: requeue}, err
	}

	systemConfiguration.Status.Ready = true

	return ctrl.Result{}, nil
}

// deleteNamespaces looks for namespaces owned by the SystemConfiguration resource (by looking
// at the namespace's labels) and deletes them if it isn't a namespace listed in the validNamespaces
// map. Only the first error encountered is returned.
func (r *NnfSystemConfigurationReconciler) deleteNamespaces(ctx context.Context, config *dwsv1alpha2.SystemConfiguration, validNamespaces map[string]struct{}) error {
	listOptions := []client.ListOption{
		dwsv1alpha2.MatchingOwner(config),
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
func (r *NnfSystemConfigurationReconciler) createNamespaces(ctx context.Context, config *dwsv1alpha2.SystemConfiguration, validNamespaces map[string]struct{}) error {
	for name := range validNamespaces {
		namespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
		}
		_, err := ctrl.CreateOrUpdate(ctx, r.Client, namespace,
			func() error {
				dwsv1alpha2.AddOwnerLabels(namespace, config)
				return nil
			})
		if err != nil {
			return err
		}
	}

	return nil
}

// labelsAndTaints will ensure that all rabbits are labeled and it will
// apply appropriate taints to the rabbits.
//
// The first time through it will ensure that each rabbit is labeled and that
// it has the NoSchedule taint.
// The second time through it will add the NoExecute taint.
// The third time through it will remove the NoExecute taint.
//
// This does not apply the NoExecute taint until we're certain that every
// rabbit already has the NoSchedule taint, so we don't play musical chairs
// with the pods that we're evicting, chasing them from rabbit to rabbit.
// A breadcrumb (a label) is left on the node to remind us when we've already
// done the add/remove of the NoExecute taint, so we don't repeat that cycle.
func (r *NnfSystemConfigurationReconciler) labelsAndTaints(ctx context.Context, validRabbits map[string]struct{}) (bool, error) {
	var err error

	// Pass 1: apply cray.nnf.node label and NoSchedule taint.
	// Pass 2: apply NoExecute taint.
	// Pass 3: apply cray.nnf.node.cleared label and clear the NoExecute taint.
	var clearNoExecute corev1.TaintEffect = "clear-no-execute"
	taintEffectPerPass := []corev1.TaintEffect{
		corev1.TaintEffectNoSchedule,
		corev1.TaintEffectNoExecute,
		clearNoExecute,
	}

	updatedNode := false
	for _, effect := range taintEffectPerPass {

		if updatedNode {
			// The previous pass must be a no-op before we can run the next pass.
			break
		}

		for name := range validRabbits {
			doUpdate := false
			node := &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: name,
				},
			}
			log := r.Log.WithValues("Node", client.ObjectKeyFromObject(node))
			if err = r.Get(ctx, client.ObjectKeyFromObject(node), node); err != nil {
				// Maybe it's been removed for administrative purposes and the
				// SystemConfiguration hasn't been updated.
				log.Info("unable to find node", "error", err)
				continue
			}
			labels := node.GetLabels()
			if labels == nil {
				labels = make(map[string]string)
			}

			if _, present := labels[v1alpha1.TaintsAndLabelsCompletedLabel]; present {
				continue
			}

			taint := &corev1.Taint{
				Key:    v1alpha1.RabbitNodeTaintKey,
				Value:  "true",
				Effect: effect,
			}

			if effect == clearNoExecute {
				// Remove the NoExecute taint.
				taint.Effect = corev1.TaintEffectNoExecute
				node, _, err = taints.RemoveTaint(node, taint)
				if err != nil {
					log.Error(err, "unable to clear taint", "key", taint.Key, "effect", taint.Effect)
					return false, err
				}
				// All passes completed on this node.
				labels[v1alpha1.TaintsAndLabelsCompletedLabel] = "true"
				doUpdate = true
				node.SetLabels(labels)
			} else {
				// Add the taint.
				node, doUpdate, err = taints.AddOrUpdateTaint(node, taint)
				if err != nil {
					log.Error(err, "unable to add taint to spec", "key", taint.Key, "effect", taint.Effect)
					return false, err
				}
			}

			// Add the label.
			if _, present := labels[v1alpha1.RabbitNodeSelectorLabel]; !present {
				labels[v1alpha1.RabbitNodeSelectorLabel] = "true"
				doUpdate = true
				node.SetLabels(labels)
			}

			if doUpdate {
				updatedNode = true
				if err := r.Update(ctx, node); err != nil {
					log.Error(err, "unable to update taints and/or labels")
					return false, err
				}
			}
		} // for name := range validRabbits
	} // for pass, effect := range taintEffectPerPass

	return true, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NnfSystemConfigurationReconciler) SetupWithManager(mgr ctrl.Manager) error {

	// Setup to watch the Kubernetes Node resource
	nodeMapFunc := func(ctx context.Context, o client.Object) []reconcile.Request {
		return []reconcile.Request{{NamespacedName: types.NamespacedName{
			// The SystemConfiguration resource is always default/default.
			Name:      "default",
			Namespace: "default",
		}}}
	}

	builder := ctrl.NewControllerManagedBy(mgr).
		WithOptions(controller.Options{MaxConcurrentReconciles: 1}).
		Watches(&corev1.Namespace{}, handler.EnqueueRequestsFromMapFunc(dwsv1alpha2.OwnerLabelMapFunc)).
		Watches(&corev1.Node{}, handler.EnqueueRequestsFromMapFunc(nodeMapFunc)).
		For(&dwsv1alpha2.SystemConfiguration{})

	return builder.Complete(r)
}
