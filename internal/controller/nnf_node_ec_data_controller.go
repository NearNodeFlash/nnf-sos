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
	"os"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	nnfec "github.com/NearNodeFlash/nnf-ec/pkg"
	ec "github.com/NearNodeFlash/nnf-ec/pkg/ec"
	"github.com/NearNodeFlash/nnf-ec/pkg/persistent"
	nnfv1alpha1 "github.com/NearNodeFlash/nnf-sos/api/v1alpha1"
	"github.com/NearNodeFlash/nnf-sos/internal/controller/metrics"
	"github.com/NearNodeFlash/nnf-sos/pkg/blockdevice"
	"github.com/go-logr/logr"
)

const (
	NnfNodeECDataResourceName = "ec-data"
)

// NnfNodeECDataReconciler reconciles a NnfNodeECData object
type NnfNodeECDataReconciler struct {
	client.Client
	Scheme  *runtime.Scheme
	Options *nnfec.Options
	types.NamespacedName
	SemaphoreForStart chan struct{}
	SemaphoreForDone  chan struct{}

	RawLog logr.Logger // RawLog, as opposed to Log, is the un-edited controller logger
}

// Start implements manager.Runnable
func (r *NnfNodeECDataReconciler) Start(ctx context.Context) error {
	log := r.RawLog.WithName("NnfNodeECData").WithValues("State", "Start")

	<-r.SemaphoreForStart

	log.Info("Ready to start")

	_, testing := os.LookupEnv("NNF_TEST_ENVIRONMENT")

	// During testing, the reconciler is started before kubeapi-server runs, so any calls will fail with
	// 'connection refused'. The test code instead bootstraps this resource manually
	if !testing {

		// Create the resource if necessary
		data := nnfv1alpha1.NnfNodeECData{}
		if err := r.Get(ctx, r.NamespacedName, &data); err != nil {

			if !errors.IsNotFound(err) {
				return err
			}

			data := nnfv1alpha1.NnfNodeECData{
				ObjectMeta: metav1.ObjectMeta{
					Name:      r.Name,
					Namespace: r.Namespace,
				},
			}

			if err := r.Create(ctx, &data); err != nil {
				return err
			}
		}

		// This resource acts as the storage provider for the NNF Element Controller
		persistent.StorageProvider = r
	}

	// NNF Element Controller logger provides 4 verbosity levels. By default, nnf-ec
	// will use the raw logger provided. If you are using zap defaults, this will
	// only include the INFO and DEBUG levels. You can increase the defaults, by
	// manually setting the zap.Options.Level, or using the -zap-log-level flag.
	//
	// You can decrease the log level of nnf-ec by adding verbosity levels to the
	// logger. For example, `logger = logger.V(1)` would reduce the logging level
	// by 1 in all nnf-ec logs. Unfortunately, the inverse is not true; one cannot
	// increase the log level of nnf-ec as the `logger.V` method does not support
	// negative values.
	logger := r.RawLog.WithName("ec")

	// Start the NNF Element Controller
	c := nnfec.NewController(r.Options).WithLogger(logger)

	if err := c.Init(&ec.Options{Http: !testing, Port: nnfec.Port}); err != nil {
		return err
	}

	if !testing {
		// Starts the http server - generally not needed but is sometimes handy to query the state of the element controller from inside a pod.
		go c.Run()
	}

	// import zpools
	ran, err := blockdevice.ZpoolImportAll(log)
	if err != nil {
		log.Error(err, "failed to import zpools")
		return err
	}
	if ran {
		log.Info("Imported all available zpools")
	}

	log.Info("Allow others to start")
	close(r.SemaphoreForDone)

	return nil
}

//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfnodeecdata,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfnodeecdata/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfnodeecdata/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the NnfNodeECData object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile
func (r *NnfNodeECDataReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// This reconciler does nothing with EC. If it ever does, then it should
	// have a lock to coordinate with the completion of Start() above.

	metrics.NnfNodeECDataReconcilesTotal.Inc()

	// your logic here

	return ctrl.Result{}, nil
}

func (r *NnfNodeECDataReconciler) NewPersistentStorageInterface(name string, readOnly bool) (persistent.PersistentStorageApi, error) {
	return &crdPersistentStorageInterface{reconciler: r, name: name}, nil
}

// Implementation of Persistent Storage API for CRD Persistent Storage
type crdPersistentStorageInterface struct {
	reconciler *NnfNodeECDataReconciler
	name       string
}

func (*crdPersistentStorageInterface) Close() error {
	return nil
}

func (psi *crdPersistentStorageInterface) View(fn func(persistent.PersistentStorageTransactionApi) error) error {
	data := nnfv1alpha1.NnfNodeECData{}
	if err := psi.reconciler.Get(context.TODO(), psi.reconciler.NamespacedName, &data); err != nil {
		return err
	}

	return fn(persistent.NewBase64PersistentStorageTransaction(data.Status.Data[psi.name]))
}

func (psi *crdPersistentStorageInterface) Update(fn func(persistent.PersistentStorageTransactionApi) error) error {

Retry:

	data := nnfv1alpha1.NnfNodeECData{}
	if err := psi.reconciler.Get(context.TODO(), psi.reconciler.NamespacedName, &data); err != nil {
		return err
	}

	if data.Status.Data == nil {
		data.Status.Data = make(map[string]nnfv1alpha1.NnfNodeECPrivateData)
	}

	if _, found := data.Status.Data[psi.name]; !found {
		data.Status.Data[psi.name] = make(nnfv1alpha1.NnfNodeECPrivateData)
	}

	if err := fn(persistent.NewBase64PersistentStorageTransaction(data.Status.Data[psi.name])); err != nil {
		return err
	}

	if err := psi.reconciler.Status().Update(context.TODO(), &data); err != nil {
		if errors.IsConflict(err) {
			goto Retry
		}

		return err
	}

	return nil
}

func (psi *crdPersistentStorageInterface) Delete(key string) error {

Retry:

	data := nnfv1alpha1.NnfNodeECData{}
	if err := psi.reconciler.Get(context.TODO(), psi.reconciler.NamespacedName, &data); err != nil {
		return err
	}

	delete(data.Status.Data[psi.name], key)

	if err := psi.reconciler.Status().Update(context.TODO(), &data); err != nil {
		if errors.IsConflict(err) {
			goto Retry
		}

		return err
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NnfNodeECDataReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.Add(r); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&nnfv1alpha1.NnfNodeECData{}).
		Complete(r)
}
