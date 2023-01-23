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

package main

import (
	"flag"
	"os"
	"runtime"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.

	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	zapcr "sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	dwsv1alpha1 "github.com/HewlettPackard/dws/api/v1alpha1"
	lusv1alpha1 "github.com/NearNodeFlash/lustre-fs-operator/api/v1alpha1"

	nnfv1alpha1 "github.com/NearNodeFlash/nnf-sos/api/v1alpha1"

	"github.com/NearNodeFlash/nnf-sos/controllers"

	//+kubebuilder:scaffold:imports

	nnf "github.com/NearNodeFlash/nnf-ec/pkg"
)

var (
	scheme   = kruntime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

const (
	NodeLocalController = "node"
	StorageController   = "storage"
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(nnfv1alpha1.AddToScheme(scheme))
	utilruntime.Must(dwsv1alpha1.AddToScheme(scheme))
	utilruntime.Must(lusv1alpha1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var controller string
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.StringVar(&controller, "controller", "node", "The controller type to run (node, storage)")

	nnfopts := nnf.BindFlags(flag.CommandLine)

	opts := zapcr.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	encoder := zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())
	zaplogger := zapcr.New(zapcr.WriteTo(os.Stdout), zapcr.Encoder(encoder))
	ctrl.SetLogger(zaplogger)

	setupLog.Info("GOMAXPROCS", "value", runtime.GOMAXPROCS(0))

	nnfCtrl := newNnfControllerInitializer(controller)
	if nnfCtrl == nil {
		setupLog.Info("unsupported controller type", "controller", controller)
		os.Exit(1)
	}

	options := ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "6cf68fa5.cray.hpe.com",
	}

	nnfCtrl.SetNamespaces(&options)

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), options)
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err := nnfCtrl.SetupReconcilers(mgr, nnfopts); err != nil {
		setupLog.Error(err, "unable to create nnf controller reconciler", "controller", nnfCtrl.GetType())
		os.Exit(1)
	}

	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

// Type NNF Controller Initializer defines an interface to initialize one of the
// NNF Controller types.
type nnfControllerInitializer interface {
	GetType() string
	SetNamespaces(*ctrl.Options)
	SetupReconcilers(manager.Manager, *nnf.Options) error
}

// Create a new NNF Controller Initializer for the given type, or nil if not found.
func newNnfControllerInitializer(typ string) nnfControllerInitializer {
	switch typ {
	case NodeLocalController:
		return &nodeLocalController{}
	case StorageController:
		return &storageController{}
	}
	return nil
}

// Type NodeLocalController defines initializer for the per NNF Node Controller
type nodeLocalController struct{}

func (*nodeLocalController) GetType() string { return NodeLocalController }
func (*nodeLocalController) SetNamespaces(options *ctrl.Options) {
	namespaces := []string{"default", os.Getenv("NNF_NODE_NAME")}

	options.NewCache = cache.MultiNamespacedCacheBuilder(namespaces)
}

func (c *nodeLocalController) SetupReconcilers(mgr manager.Manager, opts *nnf.Options) error {
	if err := (&controllers.NnfNodeReconciler{
		Client:         mgr.GetClient(),
		Log:            ctrl.Log.WithName("controllers").WithName("NnfNode"),
		Scheme:         mgr.GetScheme(),
		NamespacedName: types.NamespacedName{Name: controllers.NnfNlcResourceName, Namespace: os.Getenv("NNF_NODE_NAME")},
	}).SetupWithManager(mgr); err != nil {
		return err
	}

	if err := (&controllers.NnfNodeECDataReconciler{
		Client:         mgr.GetClient(),
		Scheme:         mgr.GetScheme(),
		NamespacedName: types.NamespacedName{Name: controllers.NnfNodeECDataResourceName, Namespace: os.Getenv("NNF_NODE_NAME")},
		Options:        opts,
	}).SetupWithManager(mgr); err != nil {
		return err
	}

	if os.Getenv("ENVIRONMENT") != "kind" {
		if err := (&controllers.NnfClientMountReconciler{
			Client: mgr.GetClient(),
			Log:    ctrl.Log.WithName("controllers").WithName("NnfClientMount"),
			Scheme: mgr.GetScheme(),
		}).SetupWithManager(mgr); err != nil {
			return err
		}
	}

	return (&controllers.NnfNodeStorageReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("NnfNodeStorage"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr)
}

// Type StorageController defines the initializer for the NNF Storage Controller
type storageController struct{}

func (*storageController) GetType() string                     { return StorageController }
func (*storageController) SetNamespaces(options *ctrl.Options) { options.Namespace = "" }

func (c *storageController) SetupReconcilers(mgr manager.Manager, opts *nnf.Options) error {
	if err := (&controllers.NnfSystemConfigurationReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("NnfSystemConfiguration"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		return err
	}

	if err := (&controllers.NnfWorkflowReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("NnfWorkflow"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		return err
	}

	if err := (&controllers.DirectiveBreakdownReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("DirectiveBreakdown"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		return err
	}

	if err := (&controllers.PersistentStorageReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("PersistentStorage"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		return err
	}

	if err := (&controllers.DWSStorageReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("Storage"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		return err
	}

	if err := (&controllers.DWSServersReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("Servers"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		return err
	}

	if err := (&controllers.NnfAccessReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("NnfAccess"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		return err
	}

	if err := (&controllers.NnfStorageReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("NnfStorage"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		return err
	}

	if err := (&nnfv1alpha1.NnfStorageProfile{}).SetupWebhookWithManager(mgr); err != nil {
		ctrl.Log.Error(err, "unable to create webhook", "webhook", "NnfStorageProfile")
		return err
	}

	return nil
}
