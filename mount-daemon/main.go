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
	"strings"
	"time"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	dwsv1alpha2 "github.com/DataWorkflowServices/dws/api/v1alpha2"
	nnfcontroller "github.com/NearNodeFlash/nnf-sos/internal/controller"

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
	scheme    = kruntime.NewScheme()
	setupLog  = ctrl.Log.WithName("setup")
	clientLog = setupLog
)

const (
	finalizerClientMount = "nnf.cray.hpe.com/clientmount"
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(dwsv1alpha2.AddToScheme(scheme))
}

type managerConfig struct {
	config       *rest.Config
	namespace    string
	requeueDelay time.Duration
}

type options struct {
	host         string
	port         string
	name         string
	tokenFile    string
	certFile     string
	requeueDelay time.Duration
}

func getOptions() (*options, *rest.Config, error) {
	// Define our own flag set so we don't inherit one that is polluted by the
	// libraries that we've imported.
	//cflags := flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	opts := options{
		host:         os.Getenv("KUBERNETES_SERVICE_HOST"),
		port:         os.Getenv("KUBERNETES_SERVICE_PORT"),
		name:         os.Getenv("NODE_NAME"),
		tokenFile:    os.Getenv("DWS_CLIENT_MOUNT_SERVICE_TOKEN_FILE"),
		certFile:     os.Getenv("DWS_CLIENT_MOUNT_SERVICE_CERT_FILE"),
		requeueDelay: 10 * time.Second,
	}

	flag.StringVar(&opts.host, "kubernetes-service-host", opts.host, "Kubernetes service host address")
	flag.StringVar(&opts.port, "kubernetes-service-port", opts.port, "Kubernetes service port number")
	flag.StringVar(&opts.name, "node-name", opts.name, "Name of this compute resource")
	flag.StringVar(&opts.tokenFile, "service-token-file", opts.tokenFile, "Path to the DWS client mount service token")
	flag.StringVar(&opts.certFile, "service-cert-file", opts.certFile, "Path to the DWS client mount service certificate")
	flag.DurationVar(&opts.requeueDelay, "requeue-delay", opts.requeueDelay, "Delay between reconciler passes")

	zapOptions := zap.Options{
		Development: true,
	}

	zapOptions.BindFlags(flag.CommandLine) //cflags)

	flag.Parse() // os.Args[1:])

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&zapOptions)))

	config, err := clientcmd.BuildConfigFromFlags(opts.host, "")
	if err != nil {
		return nil, nil, fmt.Errorf("%w: failed BuildConfigFromFlags", err)
	}

	return &opts, config, nil
}

func populateRestConfig(opts *options, restConfig *rest.Config) (*managerConfig, error) {
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

	if len(opts.name) == 0 {
		longName, err := os.Hostname()
		if err != nil {
			return nil, err
		}
		parts := strings.Split(longName, ".")
		opts.name = parts[0]
		setupLog.Info("Using system hostname", "name", opts.name)
	}

	return &managerConfig{config: restConfig, namespace: opts.name, requeueDelay: opts.requeueDelay}, nil
}

func statusUpdate(clnt client.Client, clientMount *dwsv1alpha2.ClientMount, log logr.Logger) error {
	if err := clnt.Status().Update(context.TODO(), clientMount); err != nil {
		return err
	}
	log.Info("updated status")
	return nil
}

func reconcile(clnt client.Client, namespacedName types.NamespacedName) (ctrl.Result, bool, error) {
	var err error

	clientMount := &dwsv1alpha2.ClientMount{}
	if err = clnt.Get(context.TODO(), namespacedName, clientMount); err != nil {
		clientLog.Error(err, "failed client.Get")
		return ctrl.Result{}, false, nil
	}

	if !clientMount.GetDeletionTimestamp().IsZero() {
		clientLog.Info("resource is being deleted")
		if !controllerutil.ContainsFinalizer(clientMount, finalizerClientMount) {
			return ctrl.Result{}, false, nil
		}

		clientLog.Info("unmounting all file systems due to resource deletion")
		if _, err = nnfcontroller.ChangeMountAll(context.TODO(), clnt, clientMount, dwsv1alpha2.ClientMountStateUnmounted, clientLog); err != nil {
			resourceError := dwsv1alpha2.NewResourceError("unmount failed during deletion").WithError(err)
			clientMount.Status.SetResourceErrorAndLog(resourceError, clientLog)
			if suErr := statusUpdate(clnt, clientMount, clientLog); suErr != nil {
				clientLog.Error(suErr, "unable to update status after unmount attempt during deletion")
			}
			return ctrl.Result{Requeue: true}, false, nil
		}

		clientLog.Info("removing finalizer")
		controllerutil.RemoveFinalizer(clientMount, finalizerClientMount)
		if err = clnt.Update(context.TODO(), clientMount); err != nil {
			if !apierrors.IsConflict(err) {
				clientLog.Error(err, "unable to remove finalizer")
			} else {
				clientLog.Error(err, "conflict while removing finalizer")
			}
			clientMount.Status.SetResourceErrorAndLog(err, clientLog)
			if suErr := statusUpdate(clnt, clientMount, clientLog); suErr != nil {
				clientLog.Error(suErr, "unable to update status after removing finalizer during deletion")
			}
			return ctrl.Result{Requeue: true}, false, nil
		}

		return ctrl.Result{}, false, nil
	}

	if updated := nnfcontroller.InitializeClientMountStatus(clientMount); updated {
		if err = statusUpdate(clnt, clientMount, clientLog); err != nil {
			clientLog.Error(err, "unable to update status after initializing the clientmount status")
		}
		return ctrl.Result{Requeue: true}, false, nil
	}

	// Add finalizer if it doesn't exist
	if !controllerutil.ContainsFinalizer(clientMount, finalizerClientMount) {
		controllerutil.AddFinalizer(clientMount, finalizerClientMount)
		if err := clnt.Update(context.TODO(), clientMount); err != nil {
			if apierrors.IsConflict(err) {
				clientLog.Error(err, "conflict while setting finalizer")
			} else {
				clientLog.Error(err, "unable to set finalizer")
			}
			clientMount.Status.SetResourceErrorAndLog(err, clientLog)
			if suErr := statusUpdate(clnt, clientMount, clientLog); suErr != nil {
				clientLog.Error(suErr, "unable to update finalizer error status")
			}
			return ctrl.Result{Requeue: true}, false, nil
		}
		clientLog.Info("updated finalizer")
		return ctrl.Result{Requeue: true}, false, nil
	}

	mountChanged := false
	nnfcontroller.ChangeMountAllPre(clientMount)
	if mountChanged, err = nnfcontroller.ChangeMountAll(context.TODO(), clnt, clientMount, clientMount.Spec.DesiredState, setupLog); err != nil {
		resourceError := dwsv1alpha2.NewResourceError("mount/unmount failed").WithError(err)
		clientMount.Status.SetResourceErrorAndLog(resourceError, setupLog)
	} else {
		nnfcontroller.ChangeMountAllPost(clientMount)
	}
	if err = statusUpdate(clnt, clientMount, clientLog); err != nil {
		clientLog.Error(err, "unable to update status after ChangeMountAll")
		return ctrl.Result{Requeue: true}, mountChanged, nil
	}

	return ctrl.Result{}, mountChanged, nil
}

func reconcileAll(clnt client.Client, namespace string) (ctrl.Result, error) {
	clientMountList := &dwsv1alpha2.ClientMountList{}
	listOptions := []client.ListOption{
		client.InNamespace(namespace),
	}
	if err := clnt.List(context.TODO(), clientMountList, listOptions...); err != nil {
		setupLog.Error(err, "unable to list ClientMount resources")
		return ctrl.Result{}, err
	}

	var savedErr error
	wantRequeue := false
	for _, clientMount := range clientMountList.Items {
		namespacedName := types.NamespacedName{Name: clientMount.Name, Namespace: namespace}
		clientLog = ctrl.Log.WithValues("ClientMount", namespacedName)
		result, mountChanged, err := reconcile(clnt, namespacedName)
		// Save the first error.
		if err != nil && savedErr == nil {
			savedErr = err
		}
		// Remember whether anyone is asking for a requeue.
		if result.Requeue || mountChanged {
			wantRequeue = true
		}
	}

	return ctrl.Result{Requeue: wantRequeue}, savedErr
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

	var result ctrl.Result
	lastChance := false
	namespace := opts.name // The node's namespace is the same as the node's name.
	for {
		result, err = reconcileAll(clnt, namespace)
		if !result.Requeue {
			if lastChance {
				// We've done an extra round and found no more work.
				break
			}
			// Try one more time for any late-comers.
			lastChance = true
		} else {
			// Keep going as long as we're doing work.
			lastChance = false
		}
		time.Sleep(config.requeueDelay)
	}
	if err != nil {
		os.Exit(1)
	}
	setupLog.Info("ClientMount actions completed")
	os.Exit(0)
}
