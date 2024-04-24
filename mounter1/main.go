/*
 * Copyright 2021-2024 Hewlett Packard Enterprise Development LP
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
	"path/filepath"
	"strings"
	"time"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	//_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"k8s.io/client-go/util/homedir"

	kruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	certutil "k8s.io/client-go/util/cert"

	"github.com/NearNodeFlash/nnf-sos/mount-daemon/version"

	dwsv1alpha2 "github.com/DataWorkflowServices/dws/api/v1alpha2"
)

var (
	scheme   = kruntime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(dwsv1alpha2.AddToScheme(scheme))
}

type managerConfig struct {
	config    *rest.Config
	namespace string
	timeout   time.Duration
}

type options struct {
	kubeconfig *string
	host       string
	port       string
	name       string
	tokenFile  string
	certFile   string
	step1      bool
	timeout    time.Duration
}

func getOptions() (*options, *rest.Config, error) {
	// Define our own flag set so we don't inherit one that is polluted by the
	// libraries that we've imported.
	cflags := flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	opts := options{
		host:      os.Getenv("KUBERNETES_SERVICE_HOST"),
		port:      os.Getenv("KUBERNETES_SERVICE_PORT"),
		name:      os.Getenv("NODE_NAME"),
		tokenFile: os.Getenv("DWS_CLIENT_MOUNT_SERVICE_TOKEN_FILE"),
		certFile:  os.Getenv("DWS_CLIENT_MOUNT_SERVICE_CERT_FILE"),
		step1:     false,
		timeout:   time.Minute,
	}

	if kconfig := os.Getenv("KUBECONFIG"); kconfig != "" {
		opts.kubeconfig = cflags.String("kubeconfig", kconfig, "absolute path to the kubeconfig file")
	} else {
		if home := homedir.HomeDir(); home != "" {
			opts.kubeconfig = cflags.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
		} else {
			opts.kubeconfig = cflags.String("kubeconfig", "", "absolute path to the kubeconfig file")
		}
	}

	cflags.StringVar(&opts.host, "kubernetes-service-host", opts.host, "Kubernetes service host address")
	cflags.StringVar(&opts.port, "kubernetes-service-port", opts.port, "Kubernetes service port number")
	cflags.StringVar(&opts.name, "node-name", opts.name, "Name of this compute resource")
	cflags.StringVar(&opts.tokenFile, "service-token-file", opts.tokenFile, "Path to the DWS client mount service token")
	cflags.StringVar(&opts.certFile, "service-cert-file", opts.certFile, "Path to the DWS client mount service certificate")
	cflags.DurationVar(&opts.timeout, "command-timeout", opts.timeout, "Timeout value before subcommands are killed")
	cflags.BoolVar(&opts.step1, "step1", opts.step1, "Run query")

	zapOptions := zap.Options{
		Development: true,
	}

	zapOptions.BindFlags(cflags)

	cflags.Parse(os.Args[1:])

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&zapOptions)))

	config, err := clientcmd.BuildConfigFromFlags(opts.host, *opts.kubeconfig)
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

	return &managerConfig{config: restConfig, namespace: opts.name, timeout: opts.timeout}, nil
}

func blockForever() {
	setupLog.Info("block forever")
	<-time.After(time.Duration(math.MaxInt64))
}

func main() {

	if len(os.Args) > 1 && os.Args[1] == "version" {
		fmt.Println("Version", version.BuildVersion())
		os.Exit(0)
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
			time.Sleep(10 * time.Second)
		}
	} else {
		blockForever()
	}

	os.Exit(0)
}
