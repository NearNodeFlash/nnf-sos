/*
 * Copyright 2021-2025 Hewlett Packard Enterprise Development LP
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
	"strconv"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	corev1 "k8s.io/api/core/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	zapcr "sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	dwsv1alpha2 "github.com/DataWorkflowServices/dws/api/v1alpha2"
	lusv1beta1 "github.com/NearNodeFlash/lustre-fs-operator/api/v1beta1"

	mpiv2beta1 "github.com/kubeflow/mpi-operator/pkg/apis/kubeflow/v2beta1"

	controllers "github.com/NearNodeFlash/nnf-sos/internal/controller"

	nnfv1alpha2 "github.com/NearNodeFlash/nnf-sos/api/v1alpha2"
	nnfv1alpha3 "github.com/NearNodeFlash/nnf-sos/api/v1alpha3"

	nnfv1alpha4 "github.com/NearNodeFlash/nnf-sos/api/v1alpha4"
	nnfv1alpha5 "github.com/NearNodeFlash/nnf-sos/api/v1alpha5"

	nnfv1alpha6 "github.com/NearNodeFlash/nnf-sos/api/v1alpha6"
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

	utilruntime.Must(dwsv1alpha2.AddToScheme(scheme))
	utilruntime.Must(lusv1beta1.AddToScheme(scheme))

	utilruntime.Must(mpiv2beta1.AddToScheme(scheme))
	utilruntime.Must(nnfv1alpha2.AddToScheme(scheme))
	utilruntime.Must(nnfv1alpha3.AddToScheme(scheme))
	utilruntime.Must(nnfv1alpha4.AddToScheme(scheme))
	utilruntime.Must(nnfv1alpha5.AddToScheme(scheme))
	utilruntime.Must(nnfv1alpha6.AddToScheme(scheme))
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

	flag.Parse()

	encoder := zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())
	zaplogger := zapcr.New(zapcr.Encoder(encoder), zapcr.UseDevMode(true))

	ctrl.SetLogger(zaplogger)

	setupLog.Info("GOMAXPROCS", "value", runtime.GOMAXPROCS(0))

	nnfCtrl := newNnfControllerInitializer(controller)
	if nnfCtrl == nil {
		setupLog.Info("unsupported controller type", "controller", controller)
		os.Exit(1)
	}

	options := ctrl.Options{
		Scheme:                 scheme,
		Metrics:                metricsserver.Options{BindAddress: metricsAddr},
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "6cf68fa5.cray.hpe.com",
	}

	nnfCtrl.SetNamespaces(&options)

	config := ctrl.GetConfigOrDie()
	qpsString, found := os.LookupEnv("NNF_REST_CONFIG_QPS")
	if found {
		qps, err := strconv.ParseFloat(qpsString, 32)
		if err != nil {
			setupLog.Error(err, "invalid value for NNF_REST_CONFIG_QPS")
			os.Exit(1)
		}
		config.QPS = float32(qps)
	}

	burstString, found := os.LookupEnv("NNF_REST_CONFIG_BURST")
	if found {
		burst, err := strconv.Atoi(burstString)
		if err != nil {
			setupLog.Error(err, "invalid value for NNF_REST_CONFIG_BURST")
			os.Exit(1)
		}
		config.Burst = burst
	}

	mgr, err := ctrl.NewManager(config, options)
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err := nnfCtrl.SetupReconcilers(mgr, nnfopts); err != nil {
		setupLog.Error(err, "unable to create nnf controller reconciler", "controller", nnfCtrl.GetType())
		os.Exit(1)
	}

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
	namespaceCache := make(map[string]cache.Config)
	namespaceCache[corev1.NamespaceDefault] = cache.Config{}
	namespaceCache[os.Getenv("NNF_NODE_NAME")] = cache.Config{}
	namespaceCache[os.Getenv("NNF_POD_NAMESPACE")] = cache.Config{}
	options.Cache = cache.Options{DefaultNamespaces: namespaceCache}
}

func (c *nodeLocalController) SetupReconcilers(mgr manager.Manager, opts *nnf.Options) error {
	if err := (&controllers.NnfLustreMGTReconciler{
		Client:         mgr.GetClient(),
		Log:            ctrl.Log.WithName("controllers").WithName("NnfLustreMgt"),
		Scheme:         mgr.GetScheme(),
		ControllerType: controllers.ControllerRabbit,
	}).SetupWithManager(mgr); err != nil {
		return err
	}

	// Coordinate the startup of the NLC controllers that use EC.

	semNnfNodeDone := make(chan struct{})
	if err := (&controllers.NnfNodeReconciler{
		Client:           mgr.GetClient(),
		Log:              ctrl.Log.WithName("controllers").WithName("NnfNode"),
		Scheme:           mgr.GetScheme(),
		Events:           make(chan event.GenericEvent),
		SemaphoreForDone: semNnfNodeDone,
		NamespacedName:   types.NamespacedName{Name: controllers.NnfNlcResourceName, Namespace: os.Getenv("NNF_NODE_NAME")},
		Options:          opts,
	}).SetupWithManager(mgr); err != nil {
		return err
	}

	semNnfNodeECDone := make(chan struct{})
	if err := (&controllers.NnfNodeECDataReconciler{
		Client:            mgr.GetClient(),
		Scheme:            mgr.GetScheme(),
		NamespacedName:    types.NamespacedName{Name: controllers.NnfNodeECDataResourceName, Namespace: os.Getenv("NNF_NODE_NAME")},
		SemaphoreForStart: semNnfNodeDone,
		SemaphoreForDone:  semNnfNodeECDone,
		Options:           opts,
		RawLog:            ctrl.Log,
	}).SetupWithManager(mgr); err != nil {
		return err
	}

	semNnfNodeBlockStorageDone := make(chan struct{})
	if err := (&controllers.NnfNodeBlockStorageReconciler{
		Client:            mgr.GetClient(),
		Log:               ctrl.Log.WithName("controllers").WithName("NnfNodeBlockStorage"),
		Scheme:            mgr.GetScheme(),
		Events:            make(chan event.GenericEvent),
		SemaphoreForStart: semNnfNodeECDone,
		SemaphoreForDone:  semNnfNodeBlockStorageDone,
		Options:           opts,
	}).SetupWithManager(mgr); err != nil {
		return err
	}

	semNnfNodeStorageDone := make(chan struct{})
	if err := (&controllers.NnfNodeStorageReconciler{
		Client:            mgr.GetClient(),
		Log:               ctrl.Log.WithName("controllers").WithName("NnfNodeStorage"),
		Scheme:            mgr.GetScheme(),
		SemaphoreForStart: semNnfNodeBlockStorageDone,
		SemaphoreForDone:  semNnfNodeStorageDone,
	}).SetupWithManager(mgr); err != nil {
		return err
	}

	// The NLC controllers relying on the readiness of EC.
	if err := (&controllers.NnfClientMountReconciler{
		Client:            mgr.GetClient(),
		Log:               ctrl.Log.WithName("controllers").WithName("NnfClientMount"),
		Scheme:            mgr.GetScheme(),
		SemaphoreForStart: semNnfNodeStorageDone,
		ClientType:        controllers.ClientRabbit,
	}).SetupWithManager(mgr); err != nil {
		return err
	}

	return nil
}

// Type StorageController defines the initializer for the NNF Storage Controller
type storageController struct{}

func (*storageController) GetType() string             { return StorageController }
func (*storageController) SetNamespaces(*ctrl.Options) { /* nop */ }

func (c *storageController) SetupReconcilers(mgr manager.Manager, opts *nnf.Options) error {
	if err := (&controllers.NnfSystemConfigurationReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("NnfSystemConfiguration"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		return err
	}

	if err := (&controllers.NnfPortManagerReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		// Note: Log is built into controller-runtime
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

	if err := (&controllers.NnfLustreMGTReconciler{
		Client:         mgr.GetClient(),
		Log:            ctrl.Log.WithName("controllers").WithName("NnfLustreMgt"),
		Scheme:         mgr.GetScheme(),
		ControllerType: controllers.ControllerWorker,
	}).SetupWithManager(mgr); err != nil {
		return err
	}

	if err := (&controllers.NnfSystemStorageReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("NnfSystemStorage"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		return err
	}

	var err error

	if os.Getenv("ENABLE_WEBHOOKS") != "false" {
		if err = (&nnfv1alpha6.NnfStorageProfile{}).SetupWebhookWithManager(mgr); err != nil {
			ctrl.Log.Error(err, "unable to create webhook", "webhook", "NnfStorageProfile")
			return err
		}
	}

	if os.Getenv("ENABLE_WEBHOOKS") != "false" {
		if err = (&nnfv1alpha6.NnfContainerProfile{}).SetupWebhookWithManager(mgr); err != nil {
			ctrl.Log.Error(err, "unable to create webhook", "webhook", "NnfContainerProfile")
			return err
		}
	}

	if os.Getenv("ENABLE_WEBHOOKS") != "false" {
		if err = (&nnfv1alpha6.NnfDataMovementProfile{}).SetupWebhookWithManager(mgr); err != nil {
			ctrl.Log.Error(err, "unable to create webhook", "webhook", "NnfDataMovementProfile")
			return err
		}
	}

	if os.Getenv("ENABLE_WEBHOOKS") != "false" {
		if err = (&nnfv1alpha6.NnfAccess{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "NnfAccess")
			os.Exit(1)
		}
	}
	if os.Getenv("ENABLE_WEBHOOKS") != "false" {
		if err = (&nnfv1alpha6.NnfDataMovement{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "NnfDataMovement")
			os.Exit(1)
		}
	}
	if os.Getenv("ENABLE_WEBHOOKS") != "false" {
		if err = (&nnfv1alpha6.NnfDataMovementManager{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "NnfDataMovementManager")
			os.Exit(1)
		}
	}
	if os.Getenv("ENABLE_WEBHOOKS") != "false" {
		if err = (&nnfv1alpha6.NnfLustreMGT{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "NnfLustreMGT")
			os.Exit(1)
		}
	}
	if os.Getenv("ENABLE_WEBHOOKS") != "false" {
		if err = (&nnfv1alpha6.NnfNode{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "NnfNode")
			os.Exit(1)
		}
	}
	if os.Getenv("ENABLE_WEBHOOKS") != "false" {
		if err = (&nnfv1alpha6.NnfNodeBlockStorage{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "NnfNodeBlockStorage")
			os.Exit(1)
		}
	}
	if os.Getenv("ENABLE_WEBHOOKS") != "false" {
		if err = (&nnfv1alpha6.NnfNodeECData{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "NnfNodeECData")
			os.Exit(1)
		}
	}
	if os.Getenv("ENABLE_WEBHOOKS") != "false" {
		if err = (&nnfv1alpha6.NnfNodeStorage{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "NnfNodeStorage")
			os.Exit(1)
		}
	}
	if os.Getenv("ENABLE_WEBHOOKS") != "false" {
		if err = (&nnfv1alpha6.NnfPortManager{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "NnfPortManager")
			os.Exit(1)
		}
	}
	if os.Getenv("ENABLE_WEBHOOKS") != "false" {
		if err = (&nnfv1alpha6.NnfStorage{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "NnfStorage")
			os.Exit(1)
		}
	}
	if os.Getenv("ENABLE_WEBHOOKS") != "false" {
		if err = (&nnfv1alpha6.NnfSystemStorage{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "NnfSystemStorage")
			os.Exit(1)
		}
	}
	//+kubebuilder:scaffold:builder

	return nil
}
