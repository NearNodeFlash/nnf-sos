/*
Copyright 2021 Hewlett Packard Enterprise Development LP
*/

package controllers

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/go-logr/logr"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	ec "stash.us.cray.com/rabsw/nnf-ec/pkg"
	nnf "stash.us.cray.com/rabsw/nnf-ec/pkg/manager-nnf"
	sf "stash.us.cray.com/rabsw/nnf-ec/pkg/rfsf/pkg/models"

	nnfv1alpha1 "stash.us.cray.com/RABSW/nnf-sos/api/v1alpha1"
)

const (
	// The name of the NNF Node Local Controller resource.
	NnfNlcResourceName = "nnf-nlc"
)

// NnfNodeReconciler reconciles a NnfNode object
type NnfNodeReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme

	types.NamespacedName
}

//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfnodes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfnodes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfnodes/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=namespaces,verbs=get;list;watch;create
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=endpoints,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;update

// Start is called upon starting the component manager and will create the Namespace for controlling the
// NNF Node CRD that is representiative of this particular NNF Node.
func (r *NnfNodeReconciler) Start(ctx context.Context) error {
	log := r.Log.WithValues("Node", r.NamespacedName, "State", "Start")

	// During testing, the NNF Node Reconciler is started before the kubeapi-server runs, so any Get() will
	// fail with 'connection refused'. The test code will instead bootstrap some nodes using the k8s test client.
	if r.NamespacedName.String() == string(types.Separator) {
		return nil
	}

	// Create a namespace unique to this node based on the node's x-name.
	namespace := &corev1.Namespace{}
	if err := r.Get(ctx, types.NamespacedName{Name: r.Namespace}, namespace); err != nil {
		if errors.IsNotFound(err) {
			log.Info("Creating Namespace...")
			namespace = r.createNamespace()

			if err := r.Create(ctx, namespace); err != nil {
				log.Error(err, "Create Namespace failed")
				return err
			}

			log.Info("Created Namespace")
		} else if !errors.IsAlreadyExists(err) {
			log.Error(err, "Get Namespace failed")
			return err
		}
	}

	node := &nnfv1alpha1.NnfNode{}
	if err := r.Get(ctx, r.NamespacedName, node); err != nil {
		if errors.IsNotFound(err) {
			log.Info("Creating NNF Node...")
			node = r.createNode()

			if err := r.Create(ctx, node); err != nil {
				log.Error(err, "Create Node failed")
				return err
			}

			log.Info("Created Node")
		} else if !errors.IsAlreadyExists(err) {
			log.Error(err, "Get Node failed")
			return err
		}
	} else {
		// If the pod were to crash and restart, the NNF Node resource will persist
		// but the pod name will change. Ensure the pod name is current.
		if node.Spec.Pod != os.Getenv("NNF_POD_NAME") {
			node.Spec.Pod = os.Getenv("NNF_POD_NAME")
			if err := r.Update(ctx, node); err != nil {
				return err
			}
		}
	}

	log.Info("Started")
	return nil
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.2/pkg/reconcile
func (r *NnfNodeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, err error) {
	log := r.Log.WithValues("Node", req.NamespacedName)

	node := &nnfv1alpha1.NnfNode{}
	if err := r.Get(ctx, req.NamespacedName, node); err != nil {
		// ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Configure a kubernetes service for access to the NNF Element Controller. This
	// allows developers to query the full state of the NNF EC using the established
	// Redfish / Swordfish API.
	if res, err := r.configureElementControllerService(ctx, node); !res.IsZero() || err != nil {
		return res, err
	}

	// Prepare to update the node's status
	statusUpdater := NewStatusUpdater(node)

	// Use the defer logic to submit a final update to the node's status, if required.
	// This modifies the return err on failure, such that it is automatically retried
	// by the controller if non-nil error is returned.
	defer func() {
		if err == nil {
			if err = statusUpdater.Close(r, ctx); err != nil { // NOTE: err here is the named returned value
				r.Log.Info(fmt.Sprintf("Failed to update status with error %s", err))
			}
		}
	}()

	// Access the the default storage service running in the NNF Element
	// Controller. Check for any State/Health change.
	ss := nnf.NewDefaultStorageService()

	storageService := &sf.StorageServiceV150StorageService{}
	if err := ss.StorageServiceIdGet(ss.Id(), storageService); err != nil {
		log.Error(err, "Failed to retrieve Storage Service")
		return ctrl.Result{}, err
	}

	if node.Status.Status != nnfv1alpha1.ResourceStatus(storageService.Status) ||
		node.Status.Health != nnfv1alpha1.ResourceHealth(storageService.Status) {
		statusUpdater.Update(func(s *nnfv1alpha1.NnfNodeStatus) {
			s.Status = nnfv1alpha1.ResourceStatus(storageService.Status)
			s.Health = nnfv1alpha1.ResourceHealth(storageService.Status)
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
		statusUpdater.Update(func(s *nnfv1alpha1.NnfNodeStatus) {
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
		statusUpdater.Update(func(s *nnfv1alpha1.NnfNodeStatus) {
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
			statusUpdater.Update(func(s *nnfv1alpha1.NnfNodeStatus) {
				s.Servers[idx].Id = serverEndpoint.Id
				s.Servers[idx].Name = serverEndpoint.Name
			})
		}

		if node.Status.Servers[idx].Status != nnfv1alpha1.ResourceStatus(storageService.Status) || node.Status.Servers[idx].Health != nnfv1alpha1.ResourceHealth(serverEndpoint.Status) {
			statusUpdater.Update(func(s *nnfv1alpha1.NnfNodeStatus) {
				s.Servers[idx].Status = nnfv1alpha1.ResourceStatus(storageService.Status)
				s.Servers[idx].Health = nnfv1alpha1.ResourceHealth(serverEndpoint.Status)
			})
		}
	}

	return ctrl.Result{}, nil
}

func (r *NnfNodeReconciler) createNamespace() *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: r.Namespace,
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
			Name:  r.Namespace,               // Note the conversion here from namespace to name, each NNF Node is given a unique namespace, which then becomes how the NLC is controlled.
			Pod:   os.Getenv("NNF_POD_NAME"), // Providing the podname gives users quick means to query the pod for a particular NNF Node
			State: nnfv1alpha1.ResourceEnable,
		},
		Status: nnfv1alpha1.NnfNodeStatus{
			Status:   nnfv1alpha1.ResourceStarting,
			Capacity: 0,
		},
	}
}

// Configure the NNF Element Controller service & endpoint for access to the Redfish / Swordfish API
// This uses a service without a selector so the NNF EC can be accessed through the NNF namespace.
//   i.e. curl nnf-ec.THE-NNF-NODE-NAMESPACE.svc.cluster.local/redfish/v1/StorageServices
// This requires us to manually associate an endpoint to the service for this particular pod.
func (r *NnfNodeReconciler) configureElementControllerService(ctx context.Context, node *nnfv1alpha1.NnfNode) (ctrl.Result, error) {

	// A DNS-1035 label must consist of lower case alphanumeric characters or
	// '-', start with an alphabetic character, and end with an alphanumeric
	// character (e.g. 'my-name',  or 'abc-123', regex used for validation is
	// '[a-z]([-a-z0-9]*[a-z0-9])?')"
	name := "nnf-ec"

	log := r.Log.WithValues("Service.Name", name, "Service.Namespace", node.Namespace)

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: node.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:       name,
					Protocol:   corev1.ProtocolTCP,
					Port:       80,
					TargetPort: intstr.FromInt(ec.Port),
				},
			},
		},
	}

	nodeName := os.Getenv("NNF_NODE_NAME")
	endpoints := &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: node.Namespace,
		},
		Subsets: []corev1.EndpointSubset{
			{
				Addresses: []corev1.EndpointAddress{
					{
						IP:       os.Getenv("NNF_POD_IP"),
						NodeName: &nodeName,
					},
				},
				Ports: []corev1.EndpointPort{
					{
						Name: name,
						Port: ec.Port,
					},
				},
			},
		},
	}

	for _, object := range []client.Object{service, endpoints} {
		res, err := ctrl.CreateOrUpdate(ctx, r.Client, object, func() error {
			return ctrl.SetControllerReference(node, object, r.Scheme)
		})

		if err != nil {
			log.Error(err, "Failed to create object", "Object.Kind", object.GetObjectKind())
			return ctrl.Result{}, err
		}

		if res != controllerutil.OperationResultNone {
			log.Info("Operation succeeded", "Result", res, "Object.Kind", object.GetObjectKind())
			return ctrl.Result{Requeue: true}, err
		}
	}

	return ctrl.Result{}, nil
}

type statusUpdater struct {
	node        *nnfv1alpha1.NnfNode
	needsUpdate bool
}

func NewStatusUpdater(node *nnfv1alpha1.NnfNode) *statusUpdater {
	return &statusUpdater{
		node:        node,
		needsUpdate: false,
	}
}

func (s *statusUpdater) Update(update func(status *nnfv1alpha1.NnfNodeStatus)) {
	update(&s.node.Status)
	s.needsUpdate = true
}

func (s *statusUpdater) Close(r *NnfNodeReconciler, ctx context.Context) error {
	defer func() { s.needsUpdate = false }()
	if s.needsUpdate {
		return r.Status().Update(ctx, s.node)
	}
	return nil
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
		Owns(&corev1.Endpoints{}).
		Complete(r)
}
