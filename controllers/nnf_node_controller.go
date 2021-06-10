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

	ec "stash.us.cray.com/rabsw/nnf-ec/pkg"
	nnf "stash.us.cray.com/rabsw/nnf-ec/pkg/manager-nnf"
	sf "stash.us.cray.com/rabsw/rfsf-openapi/pkg/models"

	nnfv1alpha1 "stash.us.cray.com/RABSW/nnf-sos/api/v1alpha1"
)

// NnfNodeReconciler reconciles a NnfNode object
type NnfNodeReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme

	types.NamespacedName
}

//+kubebuilder:rbac:groups=nnf.cray.com,resources=nnfnodes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=nnf.cray.com,resources=nnfnodes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=nnf.cray.com,resources=nnfnodes/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=namespaces,verbs=get;list;watch;create
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;update

// Start is called upon starting the component manager and will create the Namespace for controlling the
// NNF Node CRD that is representiative of this particular NNF Node.
func (r *NnfNodeReconciler) Start(ctx context.Context) error {
	log := r.Log.WithValues("Node", "Start")

	log.Info("Starting Node...", "Node.NamespacedName", r.NamespacedName)

	// Create a namespace unique to this node based on the node's x-name.
	namespace := &corev1.Namespace{}
	if err := r.Get(ctx, types.NamespacedName{Name: r.Namespace}, namespace); err != nil {
		if errors.IsNotFound(err) {
			log.Info("Creating Namespace...", "Namespace.NamespacedName", r.NamespacedName)
			namespace = r.createNamespace()

			if err := r.Create(ctx, namespace); err != nil {
				log.Error(err, "Create Namespace failed")
				return err
			}

			log.Info("Created Namespace", "Namespace.NamespacedName", r.NamespacedName)
		} else if !errors.IsAlreadyExists(err) {
			log.Error(err, "Get Namespace failed", "Namespace.NamespacedName", r.NamespacedName)
			return err
		}
	}

	node := &nnfv1alpha1.NnfNode{}
	if err := r.Get(ctx, r.NamespacedName, node); err != nil {
		if errors.IsNotFound(err) {
			log.Info("Creating NNF Node...", "Node.NamespacedName", r.NamespacedName)
			node = r.createNode()

			if err := r.Create(ctx, node); err != nil {
				log.Error(err, "Create NNF Node failed")
				return err
			}

			log.Info("Created Node", "Node.NamespacedName", r.NamespacedName)
		} else if !errors.IsAlreadyExists(err) {
			log.Error(err, "Get NNF-Node failed", "Node.NamespacedName", r.NamespacedName)
			return err
		}
	}

	log.Info("NNF Node Started")
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
func (r *NnfNodeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, err error) {
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
			log.Info("Creating Service...", "Service.NamespacedName", types.NamespacedName{Name: serviceName, Namespace: req.Namespace})

			service = r.createService(node)
			if err := r.Create(ctx, service); err != nil {
				log.Error(err, "Create service failed", "Service.NamespacedName", types.NamespacedName{Name: service.ObjectMeta.Name, Namespace: service.ObjectMeta.Namespace})
				return ctrl.Result{}, err
			}

			// Allow plenty of time for the service to start and resolve the DNS name for DP-API
			log.Info("Created Service", "Service.NamespacedName", types.NamespacedName{Name: service.ObjectMeta.Name, Namespace: service.ObjectMeta.Namespace})
			return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
		}

		log.Error(err, "Failed to get service", "Service.NamespacedName", types.NamespacedName{Name: serviceName, Namespace: req.Namespace})
		return ctrl.Result{}, err
	}

	// Prepare to update the node's status
	status := NewStatusUpdater(node)

	// Use the defer logic to submit a final update to the node's status, if required.
	// This modifies the return err on failure, such that it is automatically retried
	// by the controller if non-nil error is returned.
	defer func(c context.Context, r *NnfNodeReconciler, s *statusUpdater) {
		if s.requiresUpdate {
			if err = r.Status().Update(c, s.node); err != nil { // NOTE: err here is the named returned value
				r.Log.Info(fmt.Sprintf("Failed to update status with error %s", err))
			}
		}
	}(ctx, r, status)

	// Access the the default storage service running in the NNF Element
	// Controller. Check for any State/Health change.
	ss := nnf.NewDefaultStorageService()

	storageService := &sf.StorageServiceV150StorageService{}
	if err := ss.StorageServiceIdGet(ss.Id(), storageService); err != nil {
		log.Error(err, "Failed to retrieve Storage Service")
		return ctrl.Result{}, err
	}

	if node.Status.State != string(storageService.Status.State) ||
		node.Status.Health != string(storageService.Status.Health) {
		status.Update(func(s *nnfv1alpha1.NnfNodeStatus) {
			s.State = string(storageService.Status.State)
			s.Health = string(storageService.Status.Health)
		})
	}

	// Update the capacity and capacity allocated to reflect the current
	// values.
	capacitySource := &sf.CapacityCapacitySource{}
	if err := ss.StorageServiceIdCapacitySourceGet(ss.Id(), capacitySource); err != nil {
		log.Error(err, "Failed to retrieve Storage Service Capacity")
		return ctrl.Result{}, err
	}

	if node.Status.Capacity != capacitySource.ProvidedCapacity.Data.GuaranteedBytes ||
		node.Status.CapacityAllocated != capacitySource.ProvidedCapacity.Data.AllocatedBytes {
		status.Update(func(s *nnfv1alpha1.NnfNodeStatus) {
			s.Capacity = capacitySource.ProvidedCapacity.Data.GuaranteedBytes
			s.CapacityAllocated = capacitySource.ProvidedCapacity.Data.AllocatedBytes
		})
	}

	// Update the server status' with the current values
	serverEndpointCollection := &sf.EndpointCollectionEndpointCollection{}
	if err := ss.StorageServiceIdEndpointsGet(ss.Id(), serverEndpointCollection); err != nil {
		log.Error(err, "Failed to retrieve Storage Service Endpoints")
		return ctrl.Result{}, err
	}

	if len(node.Status.Servers) < len(serverEndpointCollection.Members) {
		status.Update(func(s *nnfv1alpha1.NnfNodeStatus) {
			s.Servers = make([]nnfv1alpha1.NnfServerStatus, len(serverEndpointCollection.Members))
		})
	}

	// Iterate over the server endpoints to ensure we've reflected
	// the status of each server (Compute & Rabbit)
	for idx, serverEndpoint := range serverEndpointCollection.Members {

		id := serverEndpoint.OdataId[strings.LastIndex(serverEndpoint.OdataId, "/")+1:]
		serverEndpoint := &sf.EndpointV150Endpoint{}
		if err := ss.StorageServiceIdEndpointIdGet(ss.Id(), id, serverEndpoint); err != nil {
			log.Error(err, fmt.Sprintf("Failed to retrieve Storage Service Endpoint %s", id))
			return ctrl.Result{}, err
		}

		if node.Status.Servers[idx].Id != serverEndpoint.Id || node.Status.Servers[idx].Name != serverEndpoint.Name {
			status.Update(func(s *nnfv1alpha1.NnfNodeStatus) {
				s.Servers[idx].Id = serverEndpoint.Id
				s.Servers[idx].Name = serverEndpoint.Name
			})
		}

		if node.Status.Servers[idx].State != string(serverEndpoint.Status.State) || node.Status.Servers[idx].Health != string(serverEndpoint.Status.Health) {
			status.Update(func(s *nnfv1alpha1.NnfNodeStatus) {
				s.Servers[idx].State = string(serverEndpoint.Status.State)
				s.Servers[idx].Health = string(serverEndpoint.Status.Health)
			})
		}
	}

	// TODO: Storage is allocated by updates to the .spec. Here code will reside
	// that reads the current .spec file and ensures that all resources pertaining
	// to the .spec are resolved. There should be a compainion .status field that
	// reflects the status of the .spec.

	return ctrl.Result{}, nil
}

func (r *NnfNodeReconciler) createNamespace() *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: r.Namespace,
			Labels: map[string]string{ // TODO: Check if this is necessary
				"control-plane": "controller-manager",
			},
		},
	}
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
			State:    "Starting",
			Capacity: 0,
		},
	}
}

func ServiceName(nodeName string) string {
	// A DNS-1035 label must consist of lower case alphanumeric characters or
	// '-', start with an alphabetic character, and end with an alphanumeric
	// character (e.g. 'my-name',  or 'abc-123', regex used for validation is
	// '[a-z]([-a-z0-9]*[a-z0-9])?')"

	return "nnf-ec"
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
				"cray.nnf.node": "true",
				//"cray.nnf.x-name": node.Name,
			},
			Type: corev1.ServiceTypeClusterIP,
			Ports: []corev1.ServicePort{
				{
					Name:       "nnf-ec",
					Protocol:   corev1.ProtocolTCP,
					Port:       ec.Port,
					TargetPort: intstr.FromInt(ec.Port),
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
		Owns(&corev1.Namespace{}). // The node will create a namespace for itself, so it can watch changes to the NNF Node custom resource
		Owns(&corev1.Service{}).   // The Node will create a service for the corresponding x-name
		Complete(r)
}

type statusUpdater struct {
	node           *nnfv1alpha1.NnfNode
	requiresUpdate bool
}

func NewStatusUpdater(node *nnfv1alpha1.NnfNode) *statusUpdater {
	return &statusUpdater{
		node:           node,
		requiresUpdate: false,
	}
}

func (s *statusUpdater) Update(update func(status *nnfv1alpha1.NnfNodeStatus)) {
	update(&s.node.Status)
	s.requiresUpdate = true
}
