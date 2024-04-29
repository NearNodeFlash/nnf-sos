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
	"context"
	"flag"
	"fmt"
	"math"
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
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	kruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	certutil "k8s.io/client-go/util/cert"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	dwsv1alpha2 "github.com/DataWorkflowServices/dws/api/v1alpha2"
	nnfv1alpha1 "github.com/NearNodeFlash/nnf-sos/api/v1alpha1"
	"github.com/NearNodeFlash/nnf-sos/mount-daemon/version"
	//+kubebuilder:scaffold:imports
)

const (
	name        = "roehrich-clientmountd"
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

	opts, restConfig, err := getOptions()
	if err != nil {
		setupLog.Error(err, "failed getOptions")
		os.Exit(1)
	}

	config, err := populateRestConfig(opts, restConfig)
	if err != nil {
		setupLog.Error(err, "Unable to create manager")
		os.Exit(1)
	}

	setupLog.Info("Client Mount Daemon", "Version", version.BuildVersion())

	// Set up channel on which to send signal notifications; must use a buffered
	// channel or risk missing the signal if we're not setup to receive the signal
	// when it is sent.
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	go startManager(opts, config)

	killSignal := <-interrupt
	setupLog.Info("Daemon was killed", "signal", killSignal)
	return "Exited", nil
}

type managerConfig struct {
	config    *rest.Config
	namespace string
}

type options struct {
	host      string
	port      string
	name      string
	tokenFile string
	certFile  string
	step1     bool
	step2     bool
}

func getOptions() (*options, *rest.Config, error) {
	cflags := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	opts := options{
		host:      os.Getenv("KUBERNETES_SERVICE_HOST"),
		port:      os.Getenv("KUBERNETES_SERVICE_PORT"),
		name:      os.Getenv("NODE_NAME"),
		tokenFile: os.Getenv("NNF_CLIENT_MOUNT_SERVICE_TOKEN_FILE"),
		certFile:  os.Getenv("NNF_CLIENT_MOUNT_SERVICE_CERT_FILE"),
		step1:     false,
		step2:     false,
	}

	cflags.StringVar(&opts.host, "kubernetes-service-host", opts.host, "Kubernetes service host address")
	cflags.StringVar(&opts.port, "kubernetes-service-port", opts.port, "Kubernetes service port number")
	cflags.StringVar(&opts.name, "node-name", opts.name, "Name of this compute resource")
	cflags.StringVar(&opts.tokenFile, "service-token-file", opts.tokenFile, "Path to the DWS client mount service token")
	cflags.StringVar(&opts.certFile, "service-cert-file", opts.certFile, "Path to the DWS client mount service certificate")
	cflags.BoolVar(&opts.step1, "step1", opts.step1, "Run query")
	cflags.BoolVar(&opts.step2, "step2", opts.step2, "Run query")

	zapOptions := zap.Options{
		Development: true,
	}
	zapOptions.BindFlags(cflags)

	cflags.Parse(os.Args[1:])

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&zapOptions)))

	config, err := clientcmd.BuildConfigFromFlags(opts.host, "")
	if err != nil {
		return nil, nil, fmt.Errorf("%w: failed BuildConfigFromFlags", err)
	}

	return &opts, config, nil
}

func populateRestConfig(opts *options, restConfig *rest.Config) (*managerConfig, error) {
	if opts.host != "" {
		setupLog.Info("Creating rest configuration")

		if len(opts.host) == 0 || len(opts.port) == 0 {
			return nil, fmt.Errorf("kubernetes service host/port not defined")
		}

		if len(opts.tokenFile) == 0 {
			return nil, fmt.Errorf("DWS client mount service token not defined")
		}

		token, err := os.ReadFile(opts.tokenFile)
		if err != nil {
			return nil, fmt.Errorf("%w: DWS client mount service token failed to read", err)
		}

		if len(opts.certFile) == 0 {
			return nil, fmt.Errorf("DWS client mount service certificate file not defined")
		}

		if _, err := certutil.NewPool(opts.certFile); err != nil {
			return nil, fmt.Errorf("%w: DWS client mount service certificate invalid", err)
		}

		tlsClientConfig := rest.TLSClientConfig{}
		tlsClientConfig.CAFile = opts.certFile

		restConfig.Host = "https://" + net.JoinHostPort(opts.host, opts.port)
		restConfig.TLSClientConfig = tlsClientConfig
		restConfig.BearerToken = string(token)
		restConfig.BearerTokenFile = opts.tokenFile

	}
	if len(opts.name) == 0 {
		longName, err := os.Hostname()
		if err != nil {
			return nil, err
		}
		parts := strings.Split(longName, ".")
		opts.name = parts[0]
		setupLog.Info("Using system hostname", "name", opts.name)
	}

	return &managerConfig{config: restConfig, namespace: opts.name}, nil
}

func blockForever() {
	setupLog.Info("block forever")
	<-time.After(time.Duration(math.MaxInt64))
}

func startManager(opts *options, config *managerConfig) {
	clnt, err := client.New(config.config, client.Options{Scheme: scheme})
	if err != nil {
		setupLog.Error(err, "failed client.New")
		os.Exit(1)
	}

	if opts.step1 {
		namespacedName := types.NamespacedName{Name: "dean", Namespace: opts.name}
		clientLog := ctrl.Log.WithValues("ClientMount", namespacedName)
		clientMount := &dwsv1alpha2.ClientMount{}

		for {
			if err = clnt.Get(context.TODO(), namespacedName, clientMount); err != nil {
				clientLog.Error(err, "failed client.Get")
			} else {
				clientLog.Info("Found ClientMount", "resourceVersion", clientMount.ResourceVersion)
			}
			if opts.step2 {
				time.Sleep(10 * time.Second)
			} else {
				blockForever()
			}
		}
	} else {
		blockForever()
	}
}

func main() {

	if len(os.Args) > 1 && os.Args[1] == "version" {
		fmt.Println("DeanVersion", version.BuildVersion())
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
