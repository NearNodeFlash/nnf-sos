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
	"reflect"
	"runtime"
	"strings"
	"time"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	nnf "github.com/NearNodeFlash/nnf-ec/pkg/manager-nnf"

	dwsv1alpha1 "github.com/HewlettPackard/dws/api/v1alpha1"
	sf "github.com/NearNodeFlash/nnf-ec/pkg/rfsf/pkg/models"
	nnfv1alpha1 "github.com/NearNodeFlash/nnf-sos/api/v1alpha1"
	"github.com/NearNodeFlash/nnf-sos/controllers/metrics"
)

const (
	// finalizerNnfClientMount defines the finalizer name that this controller
	// uses on the ClientMount resource. This prevents the ClientMount resource
	// from being fully deleted until this controller removes the finalizer.
	finalizerNnfClientMount = "nnf.cray.hpe.com/nnf_clientmount"
)

// NnfClientMountReconciler contains the pieces used by the reconciler
type NnfClientMountReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *kruntime.Scheme
}

//+kubebuilder:rbac:groups=dws.cray.hpe.com,resources=clientmounts,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=dws.cray.hpe.com,resources=clientmounts/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=dws.cray.hpe.com,resources=clientmounts/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *NnfClientMountReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, err error) {
	log := r.Log.WithValues("ClientMount", req.NamespacedName)

	metrics.NnfClientMountReconcilesTotal.Inc()

	clientMount := &dwsv1alpha1.ClientMount{}
	if err := r.Get(ctx, req.NamespacedName, clientMount); err != nil {
		// ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Create a status updater that handles the call to status().Update() if any of the fields
	// in clientMount.Status change
	statusUpdater := newClientMountStatusUpdater(clientMount)
	defer func() {
		if err == nil {
			err = statusUpdater.close(ctx, r)
		}
	}()

	// Handle cleanup if the resource is being deleted
	if !clientMount.GetDeletionTimestamp().IsZero() {
		if !controllerutil.ContainsFinalizer(clientMount, finalizerNnfClientMount) {
			return ctrl.Result{}, nil
		}

		// Unmount everything before removing the finalizer
		log.Info("Unmounting all file systems due to resource deletion")
		if err := r.changeMountAll(ctx, clientMount, dwsv1alpha1.ClientMountStateUnmounted); err != nil {
			return ctrl.Result{}, err
		}

		controllerutil.RemoveFinalizer(clientMount, finalizerNnfClientMount)
		if err := r.Update(ctx, clientMount); err != nil {
			if !apierrors.IsConflict(err) {
				return ctrl.Result{}, err
			}

			return ctrl.Result{Requeue: true}, nil
		}

		return ctrl.Result{}, nil
	}

	// Create the status section if it doesn't exist yet
	if len(clientMount.Status.Mounts) != len(clientMount.Spec.Mounts) {
		clientMount.Status.Mounts = make([]dwsv1alpha1.ClientMountInfoStatus, len(clientMount.Spec.Mounts))
	}

	// Initialize the status section if the desired state doesn't match the status state
	if clientMount.Status.Mounts[0].State != clientMount.Spec.DesiredState {
		for i := 0; i < len(clientMount.Status.Mounts); i++ {
			clientMount.Status.Mounts[i].State = clientMount.Spec.DesiredState
			clientMount.Status.Mounts[i].Ready = false
			clientMount.Status.Mounts[i].Message = ""
		}

		return ctrl.Result{}, nil
	}

	// Add finalizer if it doesn't exist
	if !controllerutil.ContainsFinalizer(clientMount, finalizerNnfClientMount) {
		controllerutil.AddFinalizer(clientMount, finalizerNnfClientMount)
		if err := r.Update(ctx, clientMount); err != nil {
			if !apierrors.IsConflict(err) {
				return ctrl.Result{}, err
			}

			return ctrl.Result{Requeue: true}, nil
		}

		return ctrl.Result{}, nil
	}

	if err := r.changeMountAll(ctx, clientMount, clientMount.Spec.DesiredState); err != nil {
		log.Info("Change mount error", "Error", err.Error())
		return ctrl.Result{RequeueAfter: time.Second * time.Duration(10)}, nil
	}

	return ctrl.Result{}, nil
}

// changeMmountAll mounts or unmounts all the file systems listed in the spec.Mounts list
func (r *NnfClientMountReconciler) changeMountAll(ctx context.Context, clientMount *dwsv1alpha1.ClientMount, state dwsv1alpha1.ClientMountState) error {
	log := r.Log.WithValues("ClientMount", types.NamespacedName{Name: clientMount.Name, Namespace: clientMount.Namespace})

	var firstError error
	for i, mount := range clientMount.Spec.Mounts {
		var err error

		switch state {
		case dwsv1alpha1.ClientMountStateMounted:
			err = r.changeMount(ctx, mount, true, log)
		case dwsv1alpha1.ClientMountStateUnmounted:
			err = r.changeMount(ctx, mount, false, log)
		default:
			return fmt.Errorf("Invalid desired state %s", state)
		}

		if err != nil {
			if firstError == nil {
				firstError = err
			}
			clientMount.Status.Mounts[i].Message = err.Error()
			clientMount.Status.Mounts[i].Ready = false
		} else {
			clientMount.Status.Mounts[i].Message = ""
			clientMount.Status.Mounts[i].Ready = true
		}
	}

	return firstError
}

// changeMount mount or unmounts a single mount point described in the ClientMountInfo object
func (r *NnfClientMountReconciler) changeMount(ctx context.Context, clientMountInfo dwsv1alpha1.ClientMountInfo, mount bool, log logr.Logger) error {

	if clientMountInfo.Device.Type != dwsv1alpha1.ClientMountDeviceTypeReference {
		return fmt.Errorf("Invalid device type %s", clientMountInfo.Device.Type)
	}

	if clientMountInfo.Device.DeviceReference.ObjectReference.Kind != reflect.TypeOf(nnfv1alpha1.NnfNodeStorage{}).Name() {
		return fmt.Errorf("Invalid device reference kind %s", clientMountInfo.Device.DeviceReference.ObjectReference.Kind)
	}

	namespacedName := types.NamespacedName{
		Name:      clientMountInfo.Device.DeviceReference.ObjectReference.Name,
		Namespace: clientMountInfo.Device.DeviceReference.ObjectReference.Namespace,
	}

	nodeStorage := &nnfv1alpha1.NnfNodeStorage{}
	if err := r.Get(ctx, namespacedName, nodeStorage); err != nil {
		return err
	}

	allocationStatus := nodeStorage.Status.Allocations[clientMountInfo.Device.DeviceReference.Data]
	fileShare, err := r.getFileShare(allocationStatus.FileShare.ID, allocationStatus.FileSystem.ID)
	if err != nil {
		return err
	}

	if mount {
		fileShare.FileSharePath = clientMountInfo.MountPath
	} else {
		fileShare.FileSharePath = ""
	}

	fileShare, err = r.updateFileShare(fileShare, allocationStatus.FileSystem.ID)
	if err != nil {
		return err
	}

	if mount {
		log.Info("Mounted file system", "Mount path", clientMountInfo.MountPath)
	} else {
		log.Info("Unmounted file system", "Mount path", clientMountInfo.MountPath)
	}

	return nil
}

func (r *NnfClientMountReconciler) updateFileShare(fileShare *sf.FileShareV120FileShare, fsID string) (*sf.FileShareV120FileShare, error) {
	ss := nnf.NewDefaultStorageService()

	if err := ss.StorageServiceIdFileSystemIdExportedShareIdPut(ss.Id(), fileShare.Id, fsID, fileShare); err != nil {
		return nil, err
	}

	return fileShare, nil
}

func (r *NnfClientMountReconciler) getFileShare(fsID string, id string) (*sf.FileShareV120FileShare, error) {
	ss := nnf.NewDefaultStorageService()
	sh := &sf.FileShareV120FileShare{}

	if err := ss.StorageServiceIdFileSystemIdExportedShareIdGet(ss.Id(), fsID, id, sh); err != nil {
		return nil, err
	}

	return sh, nil
}

type clientMountStatusUpdater struct {
	clientMount    *dwsv1alpha1.ClientMount
	existingStatus dwsv1alpha1.ClientMountStatus
}

func newClientMountStatusUpdater(c *dwsv1alpha1.ClientMount) *clientMountStatusUpdater {
	return &clientMountStatusUpdater{
		clientMount:    c,
		existingStatus: (*c.DeepCopy()).Status,
	}
}

func (c *clientMountStatusUpdater) close(ctx context.Context, r *NnfClientMountReconciler) error {
	if !reflect.DeepEqual(c.clientMount.Status, c.existingStatus) {
		err := r.Status().Update(ctx, c.clientMount)
		if !apierrors.IsConflict(err) {
			return err
		}
	}

	return nil
}

func filterByRabbitNamespacePrefixForTest() predicate.Predicate {
	return predicate.NewPredicateFuncs(func(object client.Object) bool {
		return strings.HasPrefix(object.GetNamespace(), "rabbit")
	})
}

// SetupWithManager sets up the controller with the Manager.
func (r *NnfClientMountReconciler) SetupWithManager(mgr ctrl.Manager) error {

	maxReconciles := runtime.GOMAXPROCS(0)
	builder := ctrl.NewControllerManagedBy(mgr).
		WithOptions(controller.Options{MaxConcurrentReconciles: maxReconciles}).
		For(&dwsv1alpha1.ClientMount{})

	if _, found := os.LookupEnv("NNF_TEST_ENVIRONMENT"); found {
		builder = builder.WithEventFilter(filterByRabbitNamespacePrefixForTest())
	}

	return builder.Complete(r)
}
