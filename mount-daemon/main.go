/*
 * Copyright 2021-2023 Hewlett Packard Enterprise Development LP
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
	"fmt"
	"net"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	"github.com/takama/daemon"
	corev1 "k8s.io/api/core/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"

	kruntime "k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	certutil "k8s.io/client-go/util/cert"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	dwsv1alpha2 "github.com/DataWorkflowServices/dws/api/v1alpha2"
	nnfv1alpha1 "github.com/NearNodeFlash/nnf-sos/api/v1alpha1"
	controllers "github.com/NearNodeFlash/nnf-sos/internal/controller"
	"github.com/NearNodeFlash/nnf-sos/mount-daemon/version"
	//+kubebuilder:scaffold:imports
)

const (
	name        = "clientmountd"
	description = "Data Workflow Service (DWS) Client Mount Service"
)

var (
	scheme   = kruntime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

type Service struct {
	daemon.Daemon
}

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(dwsv1alpha2.AddToScheme(scheme))
	utilruntime.Must(nnfv1alpha1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func (service *Service) Manage() (string, error) {

	if len(os.Args) > 1 {
		command := os.Args[1]
		switch command {
		case "install":
			return service.Install(os.Args[2:]...)
		case "remove":
			return service.Remove()
		case "start":
			return service.Start()
		case "stop":
			return service.Stop()
		case "status":
			return service.Status()
		}
	}

	opts := getOptions()

	setupLog.Info("Client Mount Daemon", "Version", version.BuildVersion())

	config, err := createManager(opts)
	if err != nil {
		return "Create", err
	}

	// Set up channel on which to send signal notifications; must use a buffered
	// channel or risk missing the signal if we're not setup to receive the signal
	// when it is sent.
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	go startManager(config)

	killSignal := <-interrupt
	setupLog.Info("Daemon was killed", "signal", killSignal)
	return "Exited", nil
}

type managerConfig struct {
	config    *rest.Config
	namespace string
	mock      bool
	timeout   time.Duration
}

type options struct {
	host      string
	port      string
	name      string
	tokenFile string
	certFile  string
	mock      bool
	timeout   time.Duration
}

func getOptions() *options {
	opts := options{
		host:      os.Getenv("KUBERNETES_SERVICE_HOST"),
		port:      os.Getenv("KUBERNETES_SERVICE_PORT"),
		name:      os.Getenv("NODE_NAME"),
		tokenFile: os.Getenv("DWS_CLIENT_MOUNT_SERVICE_TOKEN_FILE"),
		certFile:  os.Getenv("DWS_CLIENT_MOUNT_SERVICE_CERT_FILE"),
		mock:      false,
		timeout:   time.Minute,
	}

	flag.StringVar(&opts.host, "kubernetes-service-host", opts.host, "Kubernetes service host address")
	flag.StringVar(&opts.port, "kubernetes-service-port", opts.port, "Kubernetes service port number")
	flag.StringVar(&opts.name, "node-name", opts.name, "Name of this compute resource")
	flag.StringVar(&opts.tokenFile, "service-token-file", opts.tokenFile, "Path to the DWS client mount service token")
	flag.StringVar(&opts.certFile, "service-cert-file", opts.certFile, "Path to the DWS client mount service certificate")
	flag.BoolVar(&opts.mock, "mock", opts.mock, "Run in mock mode where no client mount operations take place")
	flag.DurationVar(&opts.timeout, "command-timeout", opts.timeout, "Timeout value before subcommands are killed")

	zapOptions := zap.Options{
		Development: true,
	}
	zapOptions.BindFlags(flag.CommandLine)

	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&zapOptions)))

	return &opts
}

func createManager(opts *options) (*managerConfig, error) {

	var config *rest.Config
	var err error

	if len(opts.name) == 0 {
		longName, err := os.Hostname()
		if err != nil {
			return nil, err
		}
		parts := strings.Split(longName, ".")
		opts.name = parts[0]
		setupLog.Info("Using system hostname", "name", opts.name)
	}

	if len(opts.host) == 0 && len(opts.port) == 0 {
		setupLog.Info("Using kubeconfig rest configuration")

		config, err = ctrl.GetConfig()
		if err != nil {
			return nil, err
		}

	} else {
		setupLog.Info("Using default rest configuration")

		if len(opts.host) == 0 || len(opts.port) == 0 {
			return nil, fmt.Errorf("kubernetes service host/port not defined")
		}

		if len(opts.tokenFile) == 0 {
			return nil, fmt.Errorf("DWS client mount service token not defined")
		}

		token, err := os.ReadFile(opts.tokenFile)
		if err != nil {
			return nil, fmt.Errorf("DWS client mount service token failed to read")
		}

		if len(opts.certFile) == 0 {
			return nil, fmt.Errorf("DWS client mount service certificate file not defined")
		}

		if _, err := certutil.NewPool(opts.certFile); err != nil {
			return nil, fmt.Errorf("DWS client mount service certificate invalid")
		}

		tlsClientConfig := rest.TLSClientConfig{}
		tlsClientConfig.CAFile = opts.certFile

		config = &rest.Config{
			Host:            "https://" + net.JoinHostPort(opts.host, opts.port),
			TLSClientConfig: tlsClientConfig,
			BearerToken:     string(token),
			BearerTokenFile: opts.tokenFile,
		}
	}

	return &managerConfig{config: config, namespace: opts.name, mock: opts.mock, timeout: opts.timeout}, nil
}

func startManager(config *managerConfig) {
	setupLog.Info("GOMAXPROCS", "value", runtime.GOMAXPROCS(0))

	namespaceCache := make(map[string]cache.Config)
	namespaceCache[config.namespace] = cache.Config{}
	namespaceCache[corev1.NamespaceDefault] = cache.Config{}

	mgr, err := ctrl.NewManager(config.config, ctrl.Options{
		Scheme:         scheme,
		LeaderElection: false,
		Cache:          cache.Options{DefaultNamespaces: namespaceCache},
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	semReady := make(chan struct{})
	close(semReady)
	if err = (&controllers.NnfClientMountReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("ClientMount"),
		//	Mock:    config.mock,
		//	Timeout: config.timeout,
		Scheme:            mgr.GetScheme(),
		SemaphoreForStart: semReady,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ClientMount")
		os.Exit(1)
	}

	//+kubebuilder:scaffold:builder

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func main() {

	if len(os.Args) > 1 && os.Args[1] == "version" {
		fmt.Println("Version", version.BuildVersion())
		os.Exit(0)
	}

	kindFn := func() daemon.Kind {
		if runtime.GOOS == "darwin" {
			return daemon.UserAgent
		}
		return daemon.SystemDaemon
	}

	d, err := daemon.New(name, description, kindFn(), "network-online.target")
	if err != nil {
		setupLog.Error(err, "Could not create daemon")
		os.Exit(1)
	}

	service := &Service{d}

	status, err := service.Manage()
	if err != nil {
		setupLog.Error(err, status)
		os.Exit(1)
	}

	fmt.Println(status)
}
