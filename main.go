/*
Copyright 2021 Hewlett Packard Enterprise Development LP
*/

package main

import (
	"flag"
	"os"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.

	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	nnfv1alpha1 "stash.us.cray.com/RABSW/nnf-sos/api/v1alpha1"
	"stash.us.cray.com/RABSW/nnf-sos/controllers"

	//+kubebuilder:scaffold:imports

	"stash.us.cray.com/rabsw/ec"
	nnf "stash.us.cray.com/rabsw/nnf-ec/pkg"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

const (
	NodeLocalController = "node"
	StorageController   = "storage"
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(nnfv1alpha1.AddToScheme(scheme))
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

	namespace := ""
	switch controller {
	case NodeLocalController:
		namespace = os.Getenv("NNF_NODE_NAME")

		// Initialize & Run the NNF Element Controller - this allows external entities
		// to access the Rabbit via Redfish/Swordfish APIs via HTTP server. Internally,
		// the APIs are accesses directly (not through HTTP).
		c := nnf.NewController(nnfopts)
		c.Init(&ec.Options{
			Http:    true,
			Port:    nnf.Port,
			Log:     false,
			Verbose: false,
		})

		go c.Run()

	case StorageController:
		namespace = "nnf-system"
	default:
		setupLog.Info("unsupported controller type", "controller", controller)
		os.Exit(1)
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "6cf68fa5.cray.com",
		Namespace:              namespace,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	switch controller {
	case NodeLocalController:
		if err = (&controllers.NnfNodeReconciler{
			Client: mgr.GetClient(),
			Log:    ctrl.Log.WithName("controllers").WithName("NnfNode"),
			Scheme: mgr.GetScheme(),
			NamespacedName: types.NamespacedName{
				Name:      "nnf-nlc",
				Namespace: namespace,
			},
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "Node")
			os.Exit(1)
		}
	case StorageController:
		if err = (&controllers.NnfStorageReconciler{
			Client: mgr.GetClient(),
			Log:    ctrl.Log.WithName("controllers").WithName("NnfStorage"),
			Scheme: mgr.GetScheme(),
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "Storage")
			os.Exit(1)
		}
	default:
		setupLog.Info("unsupported controller type", "controller", controller)
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
