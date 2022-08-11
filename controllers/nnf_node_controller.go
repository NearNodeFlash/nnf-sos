/*
 * Copyright 2021, 2022 Hewlett Packard Enterprise Development LP
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

package controllers

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-logr/logr"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	nnfec "github.com/NearNodeFlash/nnf-ec/pkg"
	"github.com/NearNodeFlash/nnf-ec/pkg/manager-event"
	msgreg "github.com/NearNodeFlash/nnf-ec/pkg/manager-message-registry/registries"
	nnf "github.com/NearNodeFlash/nnf-ec/pkg/manager-nnf"
	nvme "github.com/NearNodeFlash/nnf-ec/pkg/manager-nvme"
	sf "github.com/NearNodeFlash/nnf-ec/pkg/rfsf/pkg/models"

	dwsv1alpha1 "github.com/HewlettPackard/dws/api/v1alpha1"
	nnfv1alpha1 "github.com/NearNodeFlash/nnf-sos/api/v1alpha1"
)

const (
	// NnfNlcResourceName is the name of the NNF Node Local Controller resource.
	NnfNlcResourceName = "nnf-nlc"
)

// NnfNodeReconciler reconciles a NnfNode object
type NnfNodeReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *kruntime.Scheme

	Options *nnfec.Options

	types.NamespacedName
}

//+kubebuilder:rbac:groups=dws.cray.hpe.com,resources=systemconfigurations,verbs=get;list;watch
//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfnodes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfnodes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfnodes/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=namespaces,verbs=get;list;watch;create
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;update

// Start is called upon starting the component manager and will create the Namespace for controlling the
// NNF Node CRD that is representiative of this particular NNF Node.
func (r *NnfNodeReconciler) Start(ctx context.Context) error {
	log := r.Log.WithValues("NnfNode", r.NamespacedName, "State", "Start")

	// During testing, the NNF Node Reconciler is started before the kubeapi-server runs, so any Get() will
	// fail with 'connection refused'. The test code will instead bootstrap some nodes using the k8s test client.
	if r.NamespacedName.String() != string(types.Separator) {

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
	}

	// Subscribe to the NNF Event Manager
	event.EventManager.Subscribe(r)

	return nil
}

// EventHandler implements event.Subscription. Every Upstream or Downstream event runs the reconciler
// so all the NNF Node server/drive status stays current.
func (r *NnfNodeReconciler) EventHandler(e event.Event) error {

	// Upstream link events
	linkEstablished := e.Is(msgreg.UpstreamLinkEstablishedFabric("", "")) || e.Is(msgreg.DegradedUpstreamLinkEstablishedFabric("", ""))
	linkDropped := e.Is(msgreg.UpstreamLinkDroppedFabric("", ""))

	if linkEstablished || linkDropped {
		r.Reconcile(context.TODO(), ctrl.Request{NamespacedName: r.NamespacedName})
	}

	// Downstream link events
	linkEstablished = e.Is(msgreg.DownstreamLinkEstablishedFabric("", "")) || e.Is(msgreg.DegradedDownstreamLinkEstablishedFabric("", ""))
	linkDropped = e.Is(msgreg.DownstreamLinkDroppedFabric("", ""))

	if linkEstablished || linkDropped {
		r.Reconcile(context.TODO(), ctrl.Request{NamespacedName: r.NamespacedName})
	}

	return nil
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.2/pkg/reconcile
func (r *NnfNodeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, err error) {
	log := r.Log.WithValues("NnfNode", req.NamespacedName)

	node := &nnfv1alpha1.NnfNode{}
	if err := r.Get(ctx, req.NamespacedName, node); err != nil {
		// ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Prepare to update the node's status
	statusUpdater := newStatusUpdater(node)

	// Use the defer logic to submit a final update to the node's status, if required.
	// This modifies the return err on failure, such that it is automatically retried
	// by the controller if non-nil error is returned.
	defer func() {
		if err == nil {
			if err = statusUpdater.Close(ctx, r); err != nil { // NOTE: err here is the named returned value
				r.Log.Info(fmt.Sprintf("Failed to update status with error %s", err))
			}
		}
	}()

	// Access the default storage service running in the NNF Element
	// Controller. Check for any State/Health change.
	ss := nnf.NewDefaultStorageService()

	storageService := &sf.StorageServiceV150StorageService{}
	if err := ss.StorageServiceIdGet(ss.Id(), storageService); err != nil {
		log.Error(err, "Failed to retrieve Storage Service")
		return ctrl.Result{}, err
	}

	if storageService.Status.State != sf.ENABLED_RST {
		return ctrl.Result{RequeueAfter: 1 * time.Second}, nil
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

	if err := updateServers(node, statusUpdater, log); err != nil {
		return ctrl.Result{}, err
	}

	if err := updateDrives(node, statusUpdater, log); err != nil {
		return ctrl.Result{}, err
	}

	systemConfig := &dwsv1alpha1.SystemConfiguration{}
	if err := r.Get(ctx, types.NamespacedName{Name: "default", Namespace: corev1.NamespaceDefault}, systemConfig); err != nil {
		log.Info("Could not get system configuration")
		return ctrl.Result{}, nil
	}

	// Look at the storage nodes in the system config and find the
	// one corresponding to this Rabbit. Add the hostnames to the NnfNode
	// resource for the Rabbit and compute nodes.
	for _, storageNode := range systemConfig.Spec.StorageNodes {
		if storageNode.Name != req.NamespacedName.Namespace {
			continue
		}

		if storageNode.Type != "Rabbit" {
			continue
		}

		// For each of the servers in the NnfNode resource, find the
		// corresponding entry in the storage node section of the SystemConfiguration
		// resource and get the hostname.
		for i := range node.Status.Servers {
			server := &node.Status.Servers[i]
			if server.ID == "0" {
				statusUpdater.Update(func(s *nnfv1alpha1.NnfNodeStatus) {
					s.Servers[i].Hostname = storageNode.Name
				})
				continue
			}

			for _, compute := range storageNode.ComputesAccess {
				if server.ID != strconv.Itoa(compute.Index+1) {
					continue
				}

				statusUpdater.Update(func(s *nnfv1alpha1.NnfNodeStatus) {
					s.Servers[i].Hostname = compute.Name
				})
			}
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

// Update the Servers status of the NNF Node if necessary
func updateServers(node *nnfv1alpha1.NnfNode, statusUpdater *statusUpdater, log logr.Logger) error {

	ss := nnf.NewDefaultStorageService()

	// Update the server status' with the current values
	serverEndpointCollection := &sf.EndpointCollectionEndpointCollection{}
	if err := ss.StorageServiceIdEndpointsGet(ss.Id(), serverEndpointCollection); err != nil {
		log.Error(err, "Failed to retrieve Storage Service Endpoints")
		return err
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
			return err
		}

		if node.Status.Servers[idx].ID != serverEndpoint.Id || node.Status.Servers[idx].Name != serverEndpoint.Name {
			statusUpdater.Update(func(s *nnfv1alpha1.NnfNodeStatus) {
				s.Servers[idx].ID = serverEndpoint.Id
				s.Servers[idx].Name = serverEndpoint.Name
			})
		}

		if node.Status.Servers[idx].Status != nnfv1alpha1.ResourceStatus(serverEndpoint.Status) || node.Status.Servers[idx].Health != nnfv1alpha1.ResourceHealth(serverEndpoint.Status) {
			statusUpdater.Update(func(s *nnfv1alpha1.NnfNodeStatus) {
				s.Servers[idx].Status = nnfv1alpha1.ResourceStatus(serverEndpoint.Status)
				s.Servers[idx].Health = nnfv1alpha1.ResourceHealth(serverEndpoint.Status)
			})
		}
	}

	return nil
}

// Update the Drives status of the NNF Node if necessary
func updateDrives(node *nnfv1alpha1.NnfNode, statusUpdater *statusUpdater, log logr.Logger) error {
	storageService := nvme.NewDefaultStorageService()

	storageCollection := &sf.StorageCollectionStorageCollection{}
	if err := storageService.Get(storageCollection); err != nil {
		log.Error(err, "Failed to retrieve storage collection")
		return err
	}

	if len(node.Status.Drives) < len(storageCollection.Members) {
		statusUpdater.Update(func(s *nnfv1alpha1.NnfNodeStatus) {
			s.Drives = make([]nnfv1alpha1.NnfDriveStatus, len(storageCollection.Members))
		})
	}

	// Iterate over the storage devices and controllers to ensure we've reflected
	// the status of each drive.
	for idx, storageEndpoint := range storageCollection.Members {
		drive := &node.Status.Drives[idx]

		storageId := storageEndpoint.OdataId[strings.LastIndex(storageEndpoint.OdataId, "/")+1:]
		storage := &sf.StorageV190Storage{}
		if err := storageService.StorageIdGet(storageId, storage); err != nil {
			log.Error(err, fmt.Sprintf("Failed to retrive Storage %s", storageId))
			return err
		}

		if drive.ID != storage.Id || drive.Name != storage.Name {
			statusUpdater.Update(func(*nnfv1alpha1.NnfNodeStatus) {
				drive.ID = storage.Id
				drive.Name = storage.Name
			})
		}

		if drive.Status != nnfv1alpha1.ResourceStatus(storage.Status) || drive.Health != nnfv1alpha1.ResourceHealth(storage.Status) {
			statusUpdater.Update(func(*nnfv1alpha1.NnfNodeStatus) {
				drive.Status = nnfv1alpha1.ResourceStatus(storage.Status)
				drive.Health = nnfv1alpha1.ResourceHealth(storage.Status)
			})
		}

		if storage.Status.State == sf.ENABLED_RST {
			// The Swordfish architecture keeps very little information in the Storage object, instead it is nested in
			// the Storage Controllers. For our purposes, we only need to pull the information off one Storage Controller,
			// since all the desired information is replicated amongst all Storage Controllers - so use the first one returned.
			// This is only valid if the Storage object is Enabled.
			storageControllers := &sf.StorageControllerCollectionStorageControllerCollection{}
			if err := storageService.StorageIdControllersGet(storage.Id, storageControllers); err != nil {
				log.Error(err, fmt.Sprintf("Storage %s: Failed to retrieve Storage Controllers", storage.Id))
				return err
			}

			if len(storageControllers.Members) > 0 {
				storageControllerId := storageControllers.Members[0].OdataId[strings.LastIndex(storageControllers.Members[0].OdataId, "/")+1:]
				storageController := &sf.StorageControllerV100StorageController{}
				if err := storageService.StorageIdControllersControllerIdGet(storage.Id, storageControllerId, storageController); err != nil {
					log.Error(err, fmt.Sprintf("Storage %s: Failed to retrieve Storage Controller %s", storage.Id, storageControllerId))
					return err
				}

				if drive.Model != storageController.Model || drive.WearLevel != int64(storageController.NVMeControllerProperties.NVMeSMARTPercentageUsage) {
					statusUpdater.Update(func(*nnfv1alpha1.NnfNodeStatus) {
						drive.Model = storageController.Model
						drive.WearLevel = int64(storageController.NVMeControllerProperties.NVMeSMARTPercentageUsage)
					})
				}
			}

			// The Swordfish architecture places capacity information in a Storage device's Storage Pools. For our implementation,
			// we are guarenteed one Storage Pool per Storage device. This is only valid if the Storage object is Enabled.
			storagePool := &sf.StoragePoolV150StoragePool{}
			if err := storageService.StorageIdStoragePoolsStoragePoolIdGet(storage.Id, nvme.DefaultStoragePoolId, storagePool); err != nil {
				log.Error(err, fmt.Sprintf("Storage %s: Failed to retrieve Storage Pool %s", storage.Id, nvme.DefaultStoragePoolId))
				return err
			}

			if drive.Capacity != storagePool.CapacityBytes {
				statusUpdater.Update(func(*nnfv1alpha1.NnfNodeStatus) {
					drive.Capacity = storagePool.CapacityBytes
				})
			}
		}
	}

	return nil
}

type statusUpdater struct {
	node        *nnfv1alpha1.NnfNode
	needsUpdate bool
}

func newStatusUpdater(node *nnfv1alpha1.NnfNode) *statusUpdater {
	return &statusUpdater{
		node:        node,
		needsUpdate: false,
	}
}

func (s *statusUpdater) Update(update func(status *nnfv1alpha1.NnfNodeStatus)) {
	update(&s.node.Status)
	s.needsUpdate = true
}

func (s *statusUpdater) Close(ctx context.Context, r *NnfNodeReconciler) error {
	defer func() { s.needsUpdate = false }()
	if s.needsUpdate {
		return r.Status().Update(ctx, s.node)
	}
	return nil
}

// If the SystemConfiguration resource changes, generate a reconcile.Request
// for the NnfNode resource that this pod created.
func systemConfigurationMapFunc(o client.Object) []reconcile.Request {
	return []reconcile.Request{
		{NamespacedName: types.NamespacedName{
			Name:      NnfNlcResourceName,
			Namespace: os.Getenv("NNF_NODE_NAME"),
		}},
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *NnfNodeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Causes NnfNodeReconciler.Start() to be called.
	if err := mgr.Add(r); err != nil {
		return err
	}

	// There can be only one NnfNode resource for this controller to
	// manage, so we don't set MaxConcurrentReconciles.
	return ctrl.NewControllerManagedBy(mgr).
		For(&nnfv1alpha1.NnfNode{}).
		Owns(&corev1.Namespace{}). // The node will create a namespace for itself, so it can watch changes to the NNF Node custom resource
		Watches(&source.Kind{Type: &dwsv1alpha1.SystemConfiguration{}}, handler.EnqueueRequestsFromMapFunc(systemConfigurationMapFunc)).
		Complete(r)
}
