/*
 * Copyright 2021-2025 Hewlett Packard Enterprise Development LP
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

package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/go-logr/logr"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	nnfec "github.com/NearNodeFlash/nnf-ec/pkg"
	nnfevent "github.com/NearNodeFlash/nnf-ec/pkg/manager-event"
	msgreg "github.com/NearNodeFlash/nnf-ec/pkg/manager-message-registry/registries"
	nnf "github.com/NearNodeFlash/nnf-ec/pkg/manager-nnf"
	nvme "github.com/NearNodeFlash/nnf-ec/pkg/manager-nvme"
	sf "github.com/NearNodeFlash/nnf-ec/pkg/rfsf/pkg/models"
	"github.com/NearNodeFlash/nnf-sos/pkg/command"
	"github.com/NearNodeFlash/nnf-sos/pkg/fence"

	dwsv1alpha7 "github.com/DataWorkflowServices/dws/api/v1alpha7"
	"github.com/DataWorkflowServices/dws/utils/updater"
	nnfv1alpha10 "github.com/NearNodeFlash/nnf-sos/api/v1alpha10"
	"github.com/NearNodeFlash/nnf-sos/internal/controller/metrics"
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

	Options          *nnfec.Options
	SemaphoreForDone chan struct{}
	types.NamespacedName

	sync.Mutex
	Events             chan event.GenericEvent
	started            bool
	reconcilerAwake    bool
	requestWatcher     *fsnotify.Watcher       // Watches request directory to trigger BlockStorage reconcile
	responseWatcher    *fsnotify.Watcher       // Watches response directory to trigger NnfNode reconcile
	blockStorageEvents chan event.GenericEvent // Channel to trigger BlockStorage reconciles
}

//+kubebuilder:rbac:groups=dataworkflowservices.github.io,resources=systemconfigurations,verbs=get;list;watch
//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfnodes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfnodes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfnodes/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=namespaces,verbs=get;list;watch;create
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;update

// Start is called upon starting the component manager and will create the Namespace for controlling the
// NNF Node CRD that is representiative of this particular NNF Node.
func (r *NnfNodeReconciler) Start(ctx context.Context) error {
	log := r.Log.WithValues("NnfNode", r.NamespacedName, "State", "Start")

	log.Info("Ready to start")

	_, testing := os.LookupEnv("NNF_TEST_ENVIRONMENT")

	// During testing, the NNF Node Reconciler is started before the kubeapi-server runs, so any Get() will
	// fail with 'connection refused'. The test code will instead bootstrap some nodes using the k8s test client.
	if !testing {

		// Create a namespace unique to this node based on the node's x-name.
		namespace := &corev1.Namespace{}
		if err := r.Get(ctx, types.NamespacedName{Name: r.Namespace}, namespace); err != nil {

			if !errors.IsNotFound(err) {
				log.Error(err, "get namespace failed")
				return err
			}

			log.Info("Creating Namespace...")
			namespace = r.createNamespace()

			if err := r.Create(ctx, namespace); err != nil {
				log.Error(err, "create namespace failed")
				return err
			}

			log.Info("Created Namespace")
		}

		node := &nnfv1alpha10.NnfNode{}
		if err := r.Get(ctx, r.NamespacedName, node); err != nil {

			if !errors.IsNotFound(err) {
				log.Error(err, "get node failed")
				return err
			}

			log.Info("Creating NNF Node...")
			node = r.createNode()

			if err := r.Create(ctx, node); err != nil {
				log.Error(err, "create node failed")
				return err
			}

			log.Info("Created NNF Node")

		} else {

			err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
				node := &nnfv1alpha10.NnfNode{}
				if err := r.Get(ctx, r.NamespacedName, node); err != nil {
					return err
				}

				// If the pod were to crash and restart, the NNF Node resource will persist
				// but the pod name will change. Ensure the pod name is current.
				if node.Spec.Pod != os.Getenv("NNF_POD_NAME") {
					node.Spec.Pod = os.Getenv("NNF_POD_NAME")

					if err := r.Update(ctx, node); err != nil {
						return err
					}
				}

				// Mark the node's status as starting
				if node.Status.Status != nnfv1alpha10.ResourceStarting {
					node.Status.Status = nnfv1alpha10.ResourceStarting

					if err := r.Status().Update(ctx, node); err != nil {
						return err
					}
				}

				return nil
			})

			if err != nil {
				log.Error(err, "failed to initialize node")
				return err
			}
		}
	}

	// Subscribe to the NNF Event Manager
	nnfevent.EventManager.Subscribe(r)

	// Initialize fsnotify watcher for fence request directory to trigger BlockStorage reconciles
	var err error
	r.requestWatcher, err = fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create fence request watcher: %w", err)
	}

	// Create fence request directory if it doesn't exist
	if err := os.MkdirAll(fence.RequestDir, 0755); err != nil {
		log.Error(err, "Failed to create fence request directory")
		return fmt.Errorf("failed to create fence request directory: %w", err)
	}

	// Add fence request directory to watcher
	if err := r.requestWatcher.Add(fence.RequestDir); err != nil {
		log.Error(err, "Failed to add fence request directory to watcher")
		return fmt.Errorf("failed to add fence request directory to watcher: %w", err)
	}

	// Start goroutine to watch for fence request files and trigger BlockStorage reconciles
	go r.watchFenceRequests(ctx, log)

	// Initialize fsnotify watcher for fence response directory to trigger NnfNode reconciles
	r.responseWatcher, err = fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create fence response watcher: %w", err)
	}

	// Create fence response directory if it doesn't exist
	if err := os.MkdirAll(fence.ResponseDir, 0755); err != nil {
		log.Error(err, "Failed to create fence response directory")
		return fmt.Errorf("failed to create fence response directory: %w", err)
	}

	// Add fence response directory to watcher
	if err := r.responseWatcher.Add(fence.ResponseDir); err != nil {
		log.Error(err, "Failed to add fence response directory to watcher")
		return fmt.Errorf("failed to add fence response directory to watcher: %w", err)
	}

	// Start goroutine to watch for fence response files and trigger NnfNode reconciles
	go r.watchFenceResponses(ctx, log)

	r.Lock()
	r.started = true
	r.Unlock()

	log.Info("Allow others to start")
	close(r.SemaphoreForDone)
	return nil
}

// EventHandler implements event.Subscription. Every Upstream, Downstream, or NvmeStateChange event runs the reconciler
// to keep NNF Node server/drive status current.
func (r *NnfNodeReconciler) EventHandler(e nnfevent.Event) error {
	log := r.Log.WithValues("nnf-ec event", "node-up/node-down")

	// Upstream link events
	upstreamLinkEstablished := e.Is(msgreg.UpstreamLinkEstablishedFabric("", "")) || e.Is(msgreg.DegradedUpstreamLinkEstablishedFabric("", ""))
	upstreamLinkDropped := e.Is(msgreg.UpstreamLinkDroppedFabric("", ""))

	// Downstream link events
	downstreamLinkEstablished := e.Is(msgreg.DownstreamLinkEstablishedFabric("", "")) || e.Is(msgreg.DegradedDownstreamLinkEstablishedFabric("", ""))
	downstreamLinkDropped := e.Is(msgreg.DownstreamLinkDroppedFabric("", ""))

	// Drive state change events
	nvmeStateChange := e.Is(msgreg.NvmeStateChangeNnf("", "", ""))

	// If we don't care about the event, exit
	if !upstreamLinkEstablished &&
		!upstreamLinkDropped &&
		!downstreamLinkEstablished &&
		!downstreamLinkDropped &&
		!nvmeStateChange {
		return nil
	}

	log.V(1).Info("triggering watch")

	r.Events <- event.GenericEvent{Object: &nnfv1alpha10.NnfNode{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.NamespacedName.Name,
			Namespace: r.NamespacedName.Namespace,
		},
	}}

	return nil
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.2/pkg/reconcile
func (r *NnfNodeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, err error) {
	log := r.Log.WithValues("NnfNode", req.NamespacedName)
	r.Lock()
	if !r.started {
		r.Unlock()
		return ctrl.Result{RequeueAfter: 2 * time.Second}, nil
	}
	if !r.reconcilerAwake {
		log.Info("Reconciler is awake")
		r.reconcilerAwake = true
	}
	r.Unlock()

	metrics.NnfNodeReconcilesTotal.Inc()

	node := &nnfv1alpha10.NnfNode{}
	if err := r.Get(ctx, req.NamespacedName, node); err != nil {
		// ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Prepare to update the node's status
	statusUpdater := updater.NewStatusUpdater[*nnfv1alpha10.NnfNodeStatus](node)
	defer func() { err = statusUpdater.CloseWithStatusUpdate(ctx, r.Client.Status(), err) }()

	// Access the default storage service running in the NNF Element
	// Controller. Check for any State/Health change.
	ss := nnf.NewDefaultStorageService(r.Options.DeleteUnknownVolumes(), r.Options.ReplaceMissingVolumes())

	storageService := &sf.StorageServiceV150StorageService{}
	if err := ss.StorageServiceIdGet(ss.Id(), storageService); err != nil {
		log.Error(err, "Failed to retrieve Storage Service")
		return ctrl.Result{}, err
	}

	node.Status.Status = nnfv1alpha10.ResourceStatus(storageService.Status)
	node.Status.Health = nnfv1alpha10.ResourceHealth(storageService.Status)

	if storageService.Status.State != sf.ENABLED_RST {
		return ctrl.Result{RequeueAfter: 1 * time.Second}, nil
	}

	// Update the capacity and capacity allocated to reflect the current values.
	capacitySource := &sf.CapacityCapacitySource{}
	if err := ss.StorageServiceIdCapacitySourceGet(ss.Id(), capacitySource); err != nil {
		log.Error(err, "Failed to retrieve Storage Service Capacity")
		return ctrl.Result{}, err
	}

	node.Status.Capacity = capacitySource.ProvidedCapacity.Data.GuaranteedBytes
	node.Status.CapacityAllocated = capacitySource.ProvidedCapacity.Data.AllocatedBytes

	if err := r.updateServers(node, log); err != nil {
		return ctrl.Result{}, err
	}

	if err := updateDrives(node, log); err != nil {
		return ctrl.Result{}, err
	}

	systemConfig := &dwsv1alpha7.SystemConfiguration{}
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
				node.Status.Servers[i].Hostname = storageNode.Name
				continue
			}

			for _, compute := range storageNode.ComputesAccess {
				if server.ID != strconv.Itoa(compute.Index+1) {
					continue
				}

				node.Status.Servers[i].Hostname = compute.Name
			}
		}
	}

	// Check for fence response files for compute nodes on this Rabbit
	// This must run after hostnames are populated above
	if err := r.checkFencedStatus(ctx, node, log); err != nil {
		log.Error(err, "Failed to check fenced status")
		// Don't fail the reconcile, just log the error and continue
	}

	_, found := os.LookupEnv("NNF_TEST_ENVIRONMENT")
	if found || os.Getenv("ENVIRONMENT") == "kind" {
		node.Status.LNetNid = "1.2.3.4@tcp"
		return ctrl.Result{}, nil
	}

	output, err := command.Run("lctl list_nids", log)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("Could not find local LNid: %w", err)
	}

	for _, nid := range strings.Split(string(output), "\n") {
		if strings.Contains(nid, "@") {
			node.Status.LNetNid = nid
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

func (r *NnfNodeReconciler) createNode() *nnfv1alpha10.NnfNode {
	return &nnfv1alpha10.NnfNode{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.Name,
			Namespace: r.Namespace,
		},
		Spec: nnfv1alpha10.NnfNodeSpec{
			Name:  r.Namespace,               // Note the conversion here from namespace to name, each NNF Node is given a unique namespace, which then becomes how the NLC is controlled.
			Pod:   os.Getenv("NNF_POD_NAME"), // Providing the podname gives users quick means to query the pod for a particular NNF Node
			State: nnfv1alpha10.ResourceEnable,
		},
		Status: nnfv1alpha10.NnfNodeStatus{
			Status:   nnfv1alpha10.ResourceStarting,
			Capacity: 0,
		},
	}
}

// Update the Servers status of the NNF Node if necessary
func (r *NnfNodeReconciler) updateServers(node *nnfv1alpha10.NnfNode, log logr.Logger) error {

	ss := nnf.NewDefaultStorageService(r.Options.DeleteUnknownVolumes(), r.Options.ReplaceMissingVolumes())

	// Update the server status' with the current values
	serverEndpointCollection := &sf.EndpointCollectionEndpointCollection{}
	if err := ss.StorageServiceIdEndpointsGet(ss.Id(), serverEndpointCollection); err != nil {
		log.Error(err, "Failed to retrieve Storage Service Endpoints")
		return err
	}

	if len(node.Status.Servers) < len(serverEndpointCollection.Members) {
		node.Status.Servers = make([]nnfv1alpha10.NnfServerStatus, len(serverEndpointCollection.Members))
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

		node.Status.Servers[idx].NnfResourceStatus = nnfv1alpha10.NnfResourceStatus{
			ID:     serverEndpoint.Id,
			Name:   serverEndpoint.Name,
			Status: nnfv1alpha10.ResourceStatus(serverEndpoint.Status),
			Health: nnfv1alpha10.ResourceHealth(serverEndpoint.Status),
		}
	}

	return nil
}

// checkFencedStatus checks for fence response files for any of the compute nodes
// attached to this Rabbit and marks those compute servers as offline.
func (r *NnfNodeReconciler) checkFencedStatus(ctx context.Context, node *nnfv1alpha10.NnfNode, log logr.Logger) error {
	// Check if the fence response directory exists
	if _, err := os.Stat(fence.ResponseDir); err != nil {
		if os.IsNotExist(err) {
			// Directory doesn't exist, no fenced computes
			return nil
		}
		// Some other error accessing directory
		return fmt.Errorf("error checking fence response directory: %w", err)
	}

	// Read all files in the response directory
	entries, err := os.ReadDir(fence.ResponseDir)
	if err != nil {
		return fmt.Errorf("error reading fence response directory: %w", err)
	}

	// Build a map of fenced compute nodes from fence response files
	fencedComputes := make(map[string]bool)

	// Check if any compute node attached to this Rabbit has a fence response file
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		// Read and parse the fence response file
		responsePath := filepath.Join(fence.ResponseDir, entry.Name())
		data, err := os.ReadFile(responsePath)
		if err != nil {
			log.Error(err, "Failed to read fence response file", "file", responsePath)
			continue
		}

		var response struct {
			TargetNode string `json:"target_node"`
			Success    bool   `json:"success"`
			Status     string `json:"status"`
		}

		if err := json.Unmarshal(data, &response); err != nil {
			log.Error(err, "Failed to parse fence response", "file", responsePath)
			continue
		}

		// Track fenced computes
		if response.Success || response.Status == "success" {
			fencedComputes[response.TargetNode] = true
		}
	}

	// Update all server statuses based on fence state
	for i := range node.Status.Servers {
		server := &node.Status.Servers[i]
		// Skip the Rabbit node itself (ID 0)
		if server.ID == "0" {
			continue
		}

		// Check if this compute has a fence response file
		if fencedComputes[server.Hostname] {
			// Mark as fenced only if not already marked
			if server.Status != nnfv1alpha10.ResourceFenced {
				log.Info("Compute node fenced", "compute", server.Hostname)
				node.Status.Servers[i].Status = nnfv1alpha10.ResourceFenced
				node.Status.Servers[i].Health = nnfv1alpha10.ResourceCritical
			}
		} else {
			// No fence file - clear fenced status if previously set
			if server.Status == nnfv1alpha10.ResourceFenced && server.Hostname != "" {
				log.Info("Compute node fence status cleared", "compute", server.Hostname)
				node.Status.Servers[i].Status = nnfv1alpha10.ResourceReady
			}
		}
	}

	return nil
}

// Update the Drives status of the NNF Node if necessary
func updateDrives(node *nnfv1alpha10.NnfNode, log logr.Logger) error {
	storageService := nvme.NewDefaultStorageService()

	storageCollection := &sf.StorageCollectionStorageCollection{}
	if err := storageService.Get(storageCollection); err != nil {
		log.Error(err, "Failed to retrieve storage collection")
		return err
	}

	if len(node.Status.Drives) < len(storageCollection.Members) {
		node.Status.Drives = make([]nnfv1alpha10.NnfDriveStatus, len(storageCollection.Members))
	}

	// Iterate over the storage devices and controllers to ensure we've reflected
	// the status of each drive.
	for idx, storageEndpoint := range storageCollection.Members {
		drive := &node.Status.Drives[idx]

		storageId := storageEndpoint.OdataId[strings.LastIndex(storageEndpoint.OdataId, "/")+1:]
		storage := &sf.StorageV190Storage{}
		if err := storageService.StorageIdGet(storageId, storage); err != nil {
			log.Error(err, fmt.Sprintf("Failed to retrieve Storage %s", storageId))
			return err
		}

		drive.Slot = fmt.Sprintf("%d", storage.Location.PartLocation.LocationOrdinalValue)
		drive.NnfResourceStatus = nnfv1alpha10.NnfResourceStatus{
			ID:     storage.Id,
			Name:   storage.Name,
			Status: nnfv1alpha10.ResourceStatus(storage.Status),
			Health: nnfv1alpha10.ResourceHealth(storage.Status),
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

				drive.Model = storageController.Model
				drive.WearLevel = int64(storageController.NVMeControllerProperties.NVMeSMARTPercentageUsage)
				drive.SerialNumber = storageController.SerialNumber
				drive.FirmwareVersion = storageController.FirmwareVersion
			}

			// The Swordfish architecture places capacity information in a Storage device's Storage Pools. For our implementation,
			// we are guarenteed one Storage Pool per Storage device. This is only valid if the Storage object is Enabled.
			storagePool := &sf.StoragePoolV150StoragePool{}
			if err := storageService.StorageIdStoragePoolsStoragePoolIdGet(storage.Id, nvme.DefaultStoragePoolId, storagePool); err != nil {
				log.Error(err, fmt.Sprintf("Storage %s: Failed to retrieve Storage Pool %s", storage.Id, nvme.DefaultStoragePoolId))
				return err
			}

			drive.Capacity = storagePool.CapacityBytes
		}
	}

	return nil
}

// watchFenceRequests monitors the fence request directory and triggers NnfNodeBlockStorage reconciles.
func (r *NnfNodeReconciler) watchFenceRequests(ctx context.Context, log logr.Logger) {
	log = log.WithValues("component", "fenceRequestWatcher")
	log.Info("Started watching fence request directory", "dir", fence.RequestDir)

	// Scan for existing fence requests on startup (ignore dot-files which are temp files)
	files, err := os.ReadDir(fence.RequestDir)
	if err != nil {
		log.Error(err, "Failed to read fence request directory")
	} else {
		for _, file := range files {
			if !file.IsDir() && !strings.HasPrefix(file.Name(), ".") {
				log.Info("Processing existing fence request", "file", file.Name())
				r.processFenceRequest(ctx, filepath.Join(fence.RequestDir, file.Name()), log)
			}
		}
	}

	for {
		select {
		case <-ctx.Done():
			log.Info("Stopping fence request watcher")
			r.requestWatcher.Close()
			return

		case fsEvent, ok := <-r.requestWatcher.Events:
			if !ok {
				return
			}

			// Process Create events for new fence requests. The fence agent writes to a
			// temp dot-file then renames it atomically, so Create indicates a complete file.
			if fsEvent.Op&fsnotify.Create == fsnotify.Create {
				// Ignore dot-files (temp files being written)
				if strings.HasPrefix(filepath.Base(fsEvent.Name), ".") {
					continue
				}
				r.processFenceRequest(ctx, fsEvent.Name, log)
			}

		case err, ok := <-r.requestWatcher.Errors:
			if !ok {
				return
			}
			log.Error(err, "Fence request watcher error")
		}
	}
}

// processFenceRequest reads a fence request file and triggers the necessary actions
func (r *NnfNodeReconciler) processFenceRequest(ctx context.Context, filePath string, log logr.Logger) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		log.Error(err, "Failed to read fence request file", "file", filePath)
		return
	}

	var request fence.FenceRequest
	if err := json.Unmarshal(data, &request); err != nil {
		log.Error(err, "Failed to parse fence request file", "file", filePath)
		return
	}

	log.Info("Detected fence request", "targetNode", request.TargetNode, "requestID", request.RequestID, "action", request.Action)

	// Find all NnfNodeBlockStorage resources with GFS2 allocations that have access
	// from this compute node, and trigger their reconciliation
	triggered := r.triggerBlockStorageReconciles(ctx, request.TargetNode, log)
	log.Info("Triggered block storage reconciles", "count", triggered)

	// If no block storage resources were found to fence, we must still respond to the fence request
	// to indicate success (nothing to fence is effectively fenced).
	if triggered == 0 {
		log.Info("No block storage resources found to fence, writing success response", "targetNode", request.TargetNode)
		response := fence.FenceResponse{
			RequestID:       request.RequestID,
			Status:          "success",
			Success:         true,
			Message:         "No block storage resources found to fence",
			Timestamp:       time.Now().Format(time.RFC3339),
			TargetNode:      request.TargetNode,
			Action:          request.Action,
			RecorderNode:    request.RecorderNode,
			ActionPerformed: "off",
		}

		responseData, err := json.Marshal(response)
		if err != nil {
			log.Error(err, "Failed to marshal fence response")
			return
		}

		// Use the same filename as the request file
		responseFile := filepath.Join(fence.ResponseDir, filepath.Base(filePath))
		if err := os.WriteFile(responseFile, responseData, 0644); err != nil {
			log.Error(err, "Failed to write fence response file", "file", responseFile)
			return
		}

		// Delete the request file since we've handled it
		if err := os.Remove(filePath); err != nil {
			log.Error(err, "Failed to remove fence request file", "file", filePath)
		} else {
			log.Info("Removed processed fence request file", "file", filePath)
		}
	}
}

// watchFenceResponses watches fence response files to update compute node fenced status.
func (r *NnfNodeReconciler) watchFenceResponses(ctx context.Context, log logr.Logger) {
	log = log.WithValues("component", "fenceResponseWatcher")
	log.Info("Started watching fence response directory", "dir", fence.ResponseDir)

	for {
		select {
		case <-ctx.Done():
			log.Info("Stopping fence response watcher")
			r.responseWatcher.Close()
			return

		case fsEvent, ok := <-r.responseWatcher.Events:
			if !ok {
				return
			}

			// Response write indicates fence operation complete, delete indicates unfencing
			if fsEvent.Op&(fsnotify.Write|fsnotify.Remove) != 0 {
				filename := filepath.Base(fsEvent.Name)
				action := "written"
				if fsEvent.Op&fsnotify.Remove != 0 {
					action = "removed"
				}
				log.Info("Fence response file event, triggering NnfNode reconcile", "file", filename, "action", action)

				// Trigger reconciliation of this NnfNode
				r.Events <- event.GenericEvent{Object: &nnfv1alpha10.NnfNode{
					ObjectMeta: metav1.ObjectMeta{
						Name:      r.Name,
						Namespace: r.Namespace,
					},
				}}
			}

		case err, ok := <-r.responseWatcher.Errors:
			if !ok {
				return
			}
			log.Error(err, "Fence response watcher error")
		}
	}
}

// triggerBlockStorageReconciles triggers reconciliation of GFS2 NnfNodeBlockStorage resources
// that grant access to the given compute node. The block storage reconciler owns storage group
// deletion and fence response handling.
func (r *NnfNodeReconciler) triggerBlockStorageReconciles(ctx context.Context, computeNode string, log logr.Logger) int {
	// List all NnfNodeBlockStorage resources in this namespace
	blockStorageList := &nnfv1alpha10.NnfNodeBlockStorageList{}
	if err := r.List(ctx, blockStorageList, client.InNamespace(r.Namespace)); err != nil {
		log.Error(err, "Failed to list NnfNodeBlockStorage resources")
		return 0
	}

	log.Info("Listing NnfNodeBlockStorage resources", "count", len(blockStorageList.Items))

	triggered := 0
	for _, nnfNodeBlockStorage := range blockStorageList.Items {
		// Only process GFS2 filesystems
		allocationSet, hasLabel := nnfNodeBlockStorage.Labels["nnf.cray.hpe.com/allocationset"]
		if !hasLabel || allocationSet != "gfs2" {
			log.Info("Skipping non-GFS2 resource", "name", nnfNodeBlockStorage.Name, "allocationSet", allocationSet)
			continue
		}

		// Check if this NnfNodeBlockStorage has allocations with access from the compute node
		hasAccess := false
		for _, allocation := range nnfNodeBlockStorage.Status.Allocations {
			if allocation.Accesses != nil {
				if _, found := allocation.Accesses[computeNode]; found {
					hasAccess = true
					break
				}
			}
		}

		if hasAccess {
			log.Info("Triggering reconcile for GFS2 NnfNodeBlockStorage with compute node access",
				"nnfNodeBlockStorage", nnfNodeBlockStorage.Name,
				"computeNode", computeNode)

			if r.blockStorageEvents == nil {
				log.Error(nil, "blockStorageEvents channel is nil!")
				continue
			}

			// Trigger reconciliation by sending an event
			r.blockStorageEvents <- event.GenericEvent{
				Object: &nnfv1alpha10.NnfNodeBlockStorage{
					ObjectMeta: metav1.ObjectMeta{
						Name:      nnfNodeBlockStorage.Name,
						Namespace: nnfNodeBlockStorage.Namespace,
					},
				},
			}
			triggered++
		} else {
			log.Info("Resource has no access for compute node", "name", nnfNodeBlockStorage.Name, "computeNode", computeNode)
		}
	}

	log.Info("Triggered NnfNodeBlockStorage reconciliations", "count", triggered, "computeNode", computeNode)
	return triggered
}

// GetBlockStorageEvents returns the channel for triggering BlockStorage reconciles
func (r *NnfNodeReconciler) GetBlockStorageEvents() chan event.GenericEvent {
	return r.blockStorageEvents
}

// InitializeBlockStorageEvents initializes the channel for triggering BlockStorage reconciles.
// The buffer prevents the fence request watcher from blocking when sending multiple events.
// Size 100 is arbitrary but sufficient since concurrent GFS2 workflows per Rabbit is limited.
func (r *NnfNodeReconciler) InitializeBlockStorageEvents() {
	if r.blockStorageEvents == nil {
		r.blockStorageEvents = make(chan event.GenericEvent, 100)
	}
	r.Log.Info("Initialized blockStorageEvents channel")
}

// If the SystemConfiguration resource changes, generate a reconcile.Request
// for the NnfNode resource that this pod created.
func systemConfigurationMapFunc(context.Context, client.Object) []reconcile.Request {
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
		For(&nnfv1alpha10.NnfNode{}).
		Owns(&corev1.Namespace{}). // The node will create a namespace for itself, so it can watch changes to the NNF Node custom resource
		Watches(&dwsv1alpha7.SystemConfiguration{}, handler.EnqueueRequestsFromMapFunc(systemConfigurationMapFunc)).
		WatchesRawSource(&source.Channel{Source: r.Events}, &handler.EnqueueRequestForObject{}).
		Complete(r)
}
