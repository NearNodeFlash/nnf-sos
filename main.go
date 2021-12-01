/*
Copyright 2021 Hewlett Packard Enterprise Development LP
*/

package main

import (
	"flag"
	"os"
	"runtime"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.

	_ "k8s.io/client-go/plugin/pkg/client/auth"

	kruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	nnfv1alpha1 "stash.us.cray.com/RABSW/nnf-sos/api/v1alpha1"
	"stash.us.cray.com/RABSW/nnf-sos/controllers"
	dwsv1alpha1 "stash.us.cray.com/dpm/dws-operator/api/v1alpha1"

	//+kubebuilder:scaffold:imports

	nnf "stash.us.cray.com/rabsw/nnf-ec/pkg"
	ec "stash.us.cray.com/rabsw/nnf-ec/pkg/ec"
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
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)

	nnfopts := nnf.BindFlags(flag.CommandLine)

	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	setupLog.Info("GOMAXPROCS", "value", runtime.GOMAXPROCS(0))

	nnfCtrl := newNnfControllerInitializer(controller)
	if nnfCtrl == nil {
		setupLog.Info("unsupported controller type", "controller", controller)
		os.Exit(1)
	}

	if err := nnfCtrl.Start(nnfopts); err != nil {
		setupLog.Error(err, "failed to start nnf controller")
		os.Exit(1)
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "6cf68fa5.cray.com",
		Namespace:              nnfCtrl.GetNamespace(),
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err := nnfCtrl.SetupReconciler(mgr); err != nil {
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
	GetNamespace() string
	Start(*nnf.Options) error
	SetupReconciler(manager.Manager) error
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

func (*nodeLocalController) GetType() string      { return NodeLocalController }
func (*nodeLocalController) GetNamespace() string { return os.Getenv("NNF_NODE_NAME") }

func (*nodeLocalController) Start(opts *nnf.Options) error {
	c := nnf.NewController(opts)
	if err := c.Init(&ec.Options{
		Http:    true,
		Port:    nnf.Port,
		Log:     false,
		Verbose: false,
	}); err != nil {
		return err
	}

	go c.Run()

	return nil
}

func (c *nodeLocalController) SetupReconciler(mgr manager.Manager) error {
	if err := (&controllers.NnfNodeReconciler{
		Client:         mgr.GetClient(),
		Log:            ctrl.Log.WithName("controllers").WithName("NnfNode"),
		Scheme:         mgr.GetScheme(),
		NamespacedName: types.NamespacedName{Name: controllers.NnfNlcResourceName, Namespace: c.GetNamespace()},
	}).SetupWithManager(mgr); err != nil {
		return err
	}

	return (&controllers.NnfNodeStorageReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("NnfNodeStorage"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr)
}

// Type StorageController defines the initializer for the NNF Storage Controller
type storageController struct{}

func (*storageController) GetType() string          { return StorageController }
func (*storageController) GetNamespace() string     { return "" }
func (*storageController) Start(*nnf.Options) error { return nil }

func (c *storageController) SetupReconciler(mgr manager.Manager) error {
	if err := (&controllers.NnfNodeSLCReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("NnfNode"),
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

	if err := (&controllers.DWSServersReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("Servers"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		return err
	}

	return (&controllers.NnfStorageReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("NnfStorage"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr)
}
