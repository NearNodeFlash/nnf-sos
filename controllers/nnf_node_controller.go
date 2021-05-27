/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-logr/logr"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	nnf "stash.us.cray.com/rabsw/nnf-ec/pkg"

	nnfv1alpha1 "stash.us.cray.com/RABSW/nnf-sos/api/v1alpha1"
)

// NnfNodeReconciler reconciles a NnfNode object
type NnfNodeReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme

	types.NamespacedName
}

//+kubebuilder:rbac:groups=nnf.cray.com,namespace=nnf-system,resources=nnfnodes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=nnf.cray.com,namespace=nnf-system,resources=nnfnodes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=nnf.cray.com,namespace=nnf-system,resources=nnfnodes/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,namespace=nnf-system,resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,namespace=nnf-system,resources=pods,verbs=get;list;update

// Start is called upon starting the component manager and will create the NNF Node CRD
// that is representiative of this particular NNF Node.
func (r *NnfNodeReconciler) Start(ctx context.Context) error {
	log := r.Log.WithValues("Node", "Start")

	log.Info("Starting Node...", "Node.NamespacedName", r.NamespacedName)

	node := &nnfv1alpha1.NnfNode{}
	err := r.Get(ctx, r.NamespacedName, node)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("Creating Node...", "Node.NamespacedName", r.NamespacedName)
			node = r.createNode()

			if err := r.Create(ctx, node); err != nil {
				log.Error(err, "Create Node failed")
				return err
			}

			if err := r.Update(ctx, node); err != nil {
				log.Error(err, "Update Node failed")
				return err
			}

			if err := r.Status().Update(ctx, node); err != nil {
				log.Error(err, "Update Node status failed")
				return err
			}

			log.Info("Created Node", "Node.NamespacedName", r.NamespacedName)
			return nil
		}

		log.Error(err, "Start Node failed", "Node.NamespacedName", r.NamespacedName)
		return err
	}

	log.Info("Node Present", "Node.NamespacedName", r.NamespacedName)
	return nil
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Node object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.2/pkg/reconcile
func (r *NnfNodeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("Node", req.NamespacedName)

	node := &nnfv1alpha1.NnfNode{}
	if err := r.Get(ctx, req.NamespacedName, node); err != nil {
		log.Error(err, "Failed to get node", "Request.NamespacedName", req.NamespacedName)
		return ctrl.Result{}, err
	}

	// Create the Service for contacting DP-API
	service := &corev1.Service{}
	serviceName := ServiceName(node.Name)
	if err := r.Get(ctx, types.NamespacedName{Name: serviceName, Namespace: req.Namespace}, service); err != nil {
		if errors.IsNotFound(err) {
			log.Info("Creating service...", "Service.NamespacedName", types.NamespacedName{Name: serviceName, Namespace: req.Namespace})

			// Create the correct label for this pod.
			pod := &corev1.Pod{}
			podName := os.Getenv("NNF_POD_NAME")
			if err := r.Get(ctx, types.NamespacedName{Name: podName, Namespace: req.Namespace}, pod); err != nil {
				log.Error(err, "Failed to get pod", "Pod.NamespacedName", types.NamespacedName{Name: podName, Namespace: req.Namespace})
				return ctrl.Result{}, err
			}

			if _, ok := pod.Labels["cray.nnf.x-name"]; !ok {
				log.Info("Updating pod with x-name...", "Pod.NamespacedName", types.NamespacedName{Name: podName, Namespace: req.Namespace})

				pod.Labels["cray.nnf.x-name"] = node.Name

				if err := r.Update(ctx, pod); err != nil {
					log.Error(err, "Failed to update pod", "Pod.NamespacedName", types.NamespacedName{Name: podName, Namespace: req.Namespace})
					return ctrl.Result{}, err
				}
			}

			service = r.createService(node)
			if err := r.Create(ctx, service); err != nil {
				log.Error(err, "Create service failed", "Service.NamespacedName", types.NamespacedName{Name: service.ObjectMeta.Name, Namespace: service.ObjectMeta.Namespace})
				return ctrl.Result{}, err
			}

			// Allow plenty of time for the service to start and resolve the DNS name for DP-API
			log.Info("Created service", "Service.NamespacedName", types.NamespacedName{Name: service.ObjectMeta.Name, Namespace: service.ObjectMeta.Namespace})
			return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
		}

		log.Error(err, "Failed to get service", "Service.NamespacedName", types.NamespacedName{Name: serviceName, Namespace: req.Namespace})
		return ctrl.Result{}, err
	}

	addr := fmt.Sprintf("%s.%s", service.ObjectMeta.Name, service.ObjectMeta.Namespace)
	port := fmt.Sprintf("%d", service.Spec.Ports[0].Port)

	log.Info("Connecting to service", "Service.Addr", addr, "Service.Port", port)
	conn, err := nnf.NewStorageServiceConnection(addr, port)

	/*
		I can't for the life of me figure out a valid error that distiguishes a DNS or Connnection Refused error before the
		service is up and running. So I'm just going to default to handling all connection errors as retryable
		if err != nil {
			log.Error(err, "Failed to create connection to storage service", "StorageService.Address", address, "StorageService.Port", port)
			return ctrl.Result{}, err
		}
	*/

	if conn == nil {
		requeueAfter := 10 * time.Second
		log.Info("Failed to make connection to storage service - this may be because the service is not running", "Service.Addr", addr, "Service.Port", port, "Requeue.After", requeueAfter)
		return ctrl.Result{RequeueAfter: requeueAfter}, nil
	}

	status := &node.Status
	defer r.Status().Update(ctx, node)

	ss, err := conn.Get()
	if err != nil {
		log.Error(err, "Failed to get storage service")
		return ctrl.Result{}, err
	}

	log.Info("Retrieve storage service", "StorageService.Id", ss.Id, "StorageService.Status.Health", string(ss.Status.Health), "StorgeService.Status.State", string(ss.Status.State))
	status.State = string(ss.Status.State)
	status.Health = string(ss.Status.Health)

	cap, err := conn.GetCapacity()
	if err != nil {
		log.Error(err, "Failed to get storge service capacity")
		return ctrl.Result{}, err
	}

	status.Capacity = cap.ProvidedCapacity.Data.GuaranteedBytes
	status.CapacityAllocated = cap.ProvidedCapacity.Data.AllocatedBytes

	servers, err := conn.GetServers()
	if err != nil {
		log.Error(err, "Failed to get storage service servers")
		return ctrl.Result{}, err
	}

	status.Servers = make([]nnfv1alpha1.NnfServerStatus, len(servers))
	for idx, server := range servers {
		status.Servers[idx] = nnfv1alpha1.NnfServerStatus{
			Id:     server.OdataId,
			Name:   server.Name,
			Health: string(server.Status.Health),
			State:  string(server.Status.State),
		}
	}

	return ctrl.Result{}, nil
}

func (r *NnfNodeReconciler) createNode() *nnfv1alpha1.NnfNode {
	return &nnfv1alpha1.NnfNode{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.Name,
			Namespace: r.Namespace,
		},
		Spec: nnfv1alpha1.NnfNodeSpec{
			Name:  r.Name,
			State: "Enabled",
		},
		Status: nnfv1alpha1.NnfNodeStatus{
			State: "Starting",
		},
	}
}

func ServiceName(nodeName string) string {
	// A DNS-1035 label must consist of lower case alphanumeric characters or
	// '-', start with an alphabetic character, and end with an alphanumeric
	// character (e.g. 'my-name',  or 'abc-123', regex used for validation is
	// '[a-z]([-a-z0-9]*[a-z0-9])?')"

	// We should use the xname here when available, otherwise we default to
	// a valid name for other setups.
	if strings.HasPrefix(nodeName, "x") {
		return nodeName + "-dpapi"
	}

	return "x-name-temp-nnf-" + nodeName + "-dpapi"
}

func (r *NnfNodeReconciler) createService(node *nnfv1alpha1.NnfNode) *corev1.Service {

	// TODO: In order for the service to target our particular pod,
	// the pod must have a unique label of type nnf.node.x-name=X-NAME

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ServiceName(node.Name),
			Namespace: r.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"cray.nnf.node":   "true",
				"cray.nnf.x-name": node.Name,
			},
			Type: corev1.ServiceTypeClusterIP,
			Ports: []corev1.ServicePort{
				{
					Name:       "nnf-ec",
					Protocol:   corev1.ProtocolTCP,
					Port:       nnf.Port,
					TargetPort: intstr.FromInt(nnf.Port),
				},
			},
		},
	}

	ctrl.SetControllerReference(node, service, r.Scheme)

	return service
}

// SetupWithManager sets up the controller with the Manager.
func (r *NnfNodeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.Add(r); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&nnfv1alpha1.NnfNode{}).
		Owns(&corev1.Service{}). // The Node will create a service for the corresponding x-name
		Complete(r)
}
