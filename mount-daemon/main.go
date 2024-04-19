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
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	//_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"k8s.io/client-go/util/homedir"

	dwsv1alpha2 "github.com/DataWorkflowServices/dws/api/v1alpha2"
	nnfv1alpha1 "github.com/NearNodeFlash/nnf-sos/api/v1alpha1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	certutil "k8s.io/client-go/util/cert"

	"github.com/NearNodeFlash/nnf-sos/mount-daemon/version"
)

var (
	scheme   = kruntime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(dwsv1alpha2.AddToScheme(scheme))
	utilruntime.Must(nnfv1alpha1.AddToScheme(scheme))
}

type managerConfig struct {
	config    *rest.Config
	namespace string
	mock      bool
	timeout   time.Duration
}

type options struct {
	kubeconfig *string
	host       string
	port       string
	name       string
	tokenFile  string
	certFile   string
	mock       bool
	timeout    time.Duration
}

func getOptions() (*options, *rest.Config, error) {
	fmt.Println("Enter getOptions")

	// Define our own flag set so we don't inherit one that is polluted by the
	// libraries that we've imported.
	cflags := flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	opts := options{
		host:      os.Getenv("KUBERNETES_SERVICE_HOST"),
		port:      os.Getenv("KUBERNETES_SERVICE_PORT"),
		name:      os.Getenv("NODE_NAME"),
		tokenFile: os.Getenv("DWS_CLIENT_MOUNT_SERVICE_TOKEN_FILE"),
		certFile:  os.Getenv("DWS_CLIENT_MOUNT_SERVICE_CERT_FILE"),
		mock:      false,
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
	cflags.BoolVar(&opts.mock, "mock", opts.mock, "Run in mock mode where no client mount operations take place")
	cflags.DurationVar(&opts.timeout, "command-timeout", opts.timeout, "Timeout value before subcommands are killed")

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

	fmt.Printf("Enter populateRestConfig 1: kubeconfig(%s) %#v\n", *opts.kubeconfig, opts)

	if opts.host != "" {
		fmt.Println("Enter populateRestConfig 2")

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
		fmt.Println("Enter populateRestConfig 3")

		longName, err := os.Hostname()
		if err != nil {
			return nil, err
		}
		parts := strings.Split(longName, ".")
		opts.name = parts[0]
		setupLog.Info("Using system hostname", "name", opts.name)
	}

	return &managerConfig{config: restConfig, namespace: opts.name, mock: opts.mock, timeout: opts.timeout}, nil
}

func main() {

	if len(os.Args) > 1 && os.Args[1] == "version" {
		fmt.Println("Version", version.BuildVersion())
		os.Exit(0)
	}

	fmt.Println("Enter main 1")

	fmt.Println("Enter main 2")

	opts, restConfig, err := getOptions()
	if err != nil {
		fmt.Println("%w: Failed getOptions\n", err)
		os.Exit(1)
	}
	fmt.Println("Enter main 3")

	config, err := populateRestConfig(opts, restConfig)
	if err != nil {
		setupLog.Error(err, "Unable to create manager")
		os.Exit(1)
	}

	fmt.Println("Enter main 4")

	clientset, err := kubernetes.NewForConfig(config.config)
	if err != nil {
		setupLog.Error(err, "failed NewForConfig")
		os.Exit(1)
	}

	client, err := client.New(config.config, client.Options{Scheme: scheme})
	if err != nil {
		setupLog.Error(err, "failed client.New")
		os.Exit(1)
	}

	nnfstorageprofile := &nnfv1alpha1.NnfStorageProfile{}
	if err = client.Get(context.TODO(), types.NamespacedName{Name: "placeholder", Namespace: "nnf-system"}, nnfstorageprofile); err != nil {
		setupLog.Error(err, "failed client.Get")
		os.Exit(1)
	}
	fmt.Printf("NnfStorageProfile Data.Default before: %#v\n", nnfstorageprofile.Data.Default)
	nnfstorageprofile.Data.Default = !nnfstorageprofile.Data.Default
	if err = client.Update(context.TODO(), nnfstorageprofile); err != nil {
		setupLog.Error(err, "failed client.Update")
		os.Exit(1)
	}

	for {
		pods, err := clientset.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			setupLog.Error(err, "failed pods.list")
			os.Exit(1)
		}
		fmt.Printf("There are %d pods in the cluster\n", len(pods.Items))

		time.Sleep(10 * time.Second)
	}
}
