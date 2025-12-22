/*
 * Copyright 2023-2025 Hewlett Packard Enterprise Development LP
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
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	nnfec "github.com/NearNodeFlash/nnf-ec/pkg"
	ec "github.com/NearNodeFlash/nnf-ec/pkg/ec"
	nnfevent "github.com/NearNodeFlash/nnf-ec/pkg/manager-event"
	msgreg "github.com/NearNodeFlash/nnf-ec/pkg/manager-message-registry/registries"
	nnf "github.com/NearNodeFlash/nnf-ec/pkg/manager-nnf"
	nnfnvme "github.com/NearNodeFlash/nnf-ec/pkg/manager-nvme"
	openapi "github.com/NearNodeFlash/nnf-ec/pkg/rfsf/pkg/common"
	sf "github.com/NearNodeFlash/nnf-ec/pkg/rfsf/pkg/models"

	dwsv1alpha7 "github.com/DataWorkflowServices/dws/api/v1alpha7"
	"github.com/DataWorkflowServices/dws/utils/updater"
	nnfv1alpha10 "github.com/NearNodeFlash/nnf-sos/api/v1alpha10"
	"github.com/NearNodeFlash/nnf-sos/internal/controller/metrics"
	"github.com/NearNodeFlash/nnf-sos/pkg/blockdevice/nvme"
	"github.com/NearNodeFlash/nnf-sos/pkg/command"
	"github.com/NearNodeFlash/nnf-sos/pkg/fence"
)

const (
	// finalizerNnfNodeBlockStorage defines the key used in identifying the
	// storage object as being owned by this NNF Storage Reconciler. This
	// prevents the system from deleting the custom resource until the
	// reconciler has finished using the resource.
	finalizerNnfNodeBlockStorage = "nnf.cray.hpe.com/nnf_node_block_storage"
)

// NnfNodeBlockStorageReconciler contains the elements needed during reconciliation for NnfNodeBlockStorage
type NnfNodeBlockStorageReconciler struct {
	client.Client
	Log               logr.Logger
	Scheme            *kruntime.Scheme
	SemaphoreForStart chan struct{}
	SemaphoreForDone  chan struct{}
	Options           *nnfec.Options

	types.NamespacedName

	sync.Mutex
	Events          chan event.GenericEvent
	FenceEvents     chan event.GenericEvent // Receives fence events from NnfNode
	started         bool
	reconcilerAwake bool
}

// EventHandler implements event.Subscription. Every Upstream or Downstream event triggers a watch
// on all the NnfNodeBlockStorages. This is needed to create the StorageGroup for a compute node that
// was powered off when the Access list was updated.
func (r *NnfNodeBlockStorageReconciler) EventHandler(e nnfevent.Event) error {
	log := r.Log.WithValues("nnf-ec event", "node-up/node-down")

	// Upstream link events
	upstreamLinkEstablished := e.Is(msgreg.UpstreamLinkEstablishedFabric("", "")) || e.Is(msgreg.DegradedUpstreamLinkEstablishedFabric("", ""))
	upstreamLinkDropped := e.Is(msgreg.UpstreamLinkDroppedFabric("", ""))

	// Downstream link events
	downstreamLinkEstablished := e.Is(msgreg.DownstreamLinkEstablishedFabric("", "")) || e.Is(msgreg.DegradedDownstreamLinkEstablishedFabric("", ""))
	downstreamLinkDropped := e.Is(msgreg.DownstreamLinkDroppedFabric("", ""))

	// Check if the event is one that we care about
	if !upstreamLinkEstablished && !upstreamLinkDropped && !downstreamLinkEstablished && !downstreamLinkDropped {
		return nil
	}

	log.V(1).Info("triggering watch")

	r.Events <- event.GenericEvent{Object: &nnfv1alpha10.NnfNodeBlockStorage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nnf-ec-event",
			Namespace: "nnf-ec-event",
		},
	}}

	return nil
}

func (r *NnfNodeBlockStorageReconciler) Start(ctx context.Context) error {
	log := r.Log.WithValues("State", "Start")

	// Subscribe to the NNF Event Manager
	nnfevent.EventManager.Subscribe(r)

	<-r.SemaphoreForStart

	log.Info("Ready to start")

	r.Lock()
	r.started = true
	r.Unlock()

	close(r.SemaphoreForDone)
	return nil
}

//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfnodeblockstorages,verbs=get;list;watch;create;update;patch;delete;deletecollection
//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfnodeblockstorages/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfnodeblockstorages/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.2/pkg/reconcile
func (r *NnfNodeBlockStorageReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, err error) {
	log := r.Log.WithValues("NnfNodeBlockStorage", req.NamespacedName)
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

	metrics.NnfNodeBlockStorageReconcilesTotal.Inc()

	nodeBlockStorage := &nnfv1alpha10.NnfNodeBlockStorage{}
	if err := r.Get(ctx, req.NamespacedName, nodeBlockStorage); err != nil {
		// ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Ensure the NNF Storage Service is running prior to taking any action.
	ss := nnf.NewDefaultStorageService(r.Options.DeleteUnknownVolumes(), r.Options.ReplaceMissingVolumes())
	storageService := &sf.StorageServiceV150StorageService{}
	if err := ss.StorageServiceIdGet(ss.Id(), storageService); err != nil {
		return ctrl.Result{}, err
	}

	if storageService.Status.State != sf.ENABLED_RST {
		return ctrl.Result{RequeueAfter: 1 * time.Second}, nil
	}

	statusUpdater := updater.NewStatusUpdater[*nnfv1alpha10.NnfNodeBlockStorageStatus](nodeBlockStorage)
	defer func() { err = statusUpdater.CloseWithStatusUpdate(ctx, r.Client.Status(), err) }()
	defer func() { nodeBlockStorage.Status.SetResourceErrorAndLog(err, log) }()

	// If the NnfNodeStorage hasn't removed the finalizer from this NnfNodeBlockStorage, then don't start
	// the deletion process. We still might need the /dev paths updated for the NnfNodeStorage to properly
	// teardown
	if !nodeBlockStorage.GetDeletionTimestamp().IsZero() && !controllerutil.ContainsFinalizer(nodeBlockStorage, finalizerNnfNodeStorage) {
		if !controllerutil.ContainsFinalizer(nodeBlockStorage, finalizerNnfNodeBlockStorage) {
			return ctrl.Result{}, nil
		}

		for i := range nodeBlockStorage.Spec.Allocations {
			// Release physical storage
			result, err := r.deleteStorage(nodeBlockStorage, i)
			if err != nil {
				return ctrl.Result{Requeue: true}, nil
			}
			if result != nil {
				return *result, nil
			}
		}

		controllerutil.RemoveFinalizer(nodeBlockStorage, finalizerNnfNodeBlockStorage)
		if err := r.Update(ctx, nodeBlockStorage); err != nil {
			if !apierrors.IsConflict(err) {
				return ctrl.Result{}, err
			}

			return ctrl.Result{Requeue: true}, nil
		}

		return ctrl.Result{}, nil
	}

	// Add the finalizer if it doesn't exist yet
	if !controllerutil.ContainsFinalizer(nodeBlockStorage, finalizerNnfNodeBlockStorage) {
		controllerutil.AddFinalizer(nodeBlockStorage, finalizerNnfNodeBlockStorage)
		if err := r.Update(ctx, nodeBlockStorage); err != nil {
			if !apierrors.IsConflict(err) {
				return ctrl.Result{}, err
			}

			return ctrl.Result{Requeue: true}, nil
		}

		return ctrl.Result{}, nil
	}

	// Initialize the status section with empty allocation statuses.
	if len(nodeBlockStorage.Status.Allocations) == 0 {
		nodeBlockStorage.Status.Allocations = make([]nnfv1alpha10.NnfNodeBlockStorageAllocationStatus, len(nodeBlockStorage.Spec.Allocations))
		for i := range nodeBlockStorage.Status.Allocations {
			nodeBlockStorage.Status.Allocations[i].Accesses = make(map[string]nnfv1alpha10.NnfNodeBlockStorageAccessStatus)
		}

		return ctrl.Result{}, nil
	}

	// Loop through each allocation and create the storage
	for i := range nodeBlockStorage.Spec.Allocations {
		// Allocate physical storage
		result, err := r.allocateStorage(nodeBlockStorage, i)
		if err != nil {
			return ctrl.Result{}, dwsv1alpha7.NewResourceError("unable to allocate NVMe namespaces for allocation %v", i).WithError(err).WithMajor()
		}
		if result != nil {
			return *result, nil
		}

		// Create a block device in /dev that is accessible on the Rabbit node
		result, err = r.createBlockDevice(ctx, nodeBlockStorage, i)
		if err != nil {
			return ctrl.Result{}, dwsv1alpha7.NewResourceError("unable to attach NVMe namespace to node for allocation %v", i).WithError(err).WithMajor()
		}
		if result != nil {
			return *result, nil
		}
	}

	if _, found := os.LookupEnv("NNF_TEST_ENVIRONMENT"); !found {
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      os.Getenv("NNF_POD_NAME"),
				Namespace: os.Getenv("NNF_POD_NAMESPACE"),
			},
		}

		if err := r.Get(ctx, client.ObjectKeyFromObject(pod), pod); err != nil {
			return ctrl.Result{}, dwsv1alpha7.NewResourceError("could not get pod: %v", client.ObjectKeyFromObject(pod)).WithError(err)
		}

		// Set the start time of the pod that did the reconcile. This allows us to detect when the Rabbit node has
		// been rebooted and the /dev information is stale
		for _, container := range pod.Status.ContainerStatuses {
			if container.Name != "manager" {
				continue
			}

			if container.State.Running == nil {
				return ctrl.Result{}, dwsv1alpha7.NewResourceError("pod not in state running: %v", client.ObjectKeyFromObject(pod)).WithError(err).WithMajor()
			}

			nodeBlockStorage.Status.PodStartTime = container.State.Running.StartedAt
		}
	}

	nodeBlockStorage.Status.Ready = true

	// Process any pending fence requests for compute nodes with access to this storage.
	// This runs after normal operations complete so storage is fully configured.
	computeNodes := r.getComputeNodesWithAccess(nodeBlockStorage)
	if len(computeNodes) > 0 {
		fenceRequests, err := r.scanFenceRequests(computeNodes, log)
		if err != nil {
			log.Error(err, "Failed to scan fence requests")
			// Don't fail the reconcile, continue
		} else if len(fenceRequests) > 0 {
			log.Info("Fence requests detected for compute nodes with access",
				"blockStorage", nodeBlockStorage.Name,
				"computeNodes", computeNodes,
				"requestCount", len(fenceRequests))

			// Process fence requests - delete storage groups to fence the compute nodes
			if err := r.processFenceRequests(ctx, fenceRequests, log); err != nil {
				log.Error(err, "Failed to process fence requests")
				// Don't fail the reconcile, just log the error
			}
		}
	}

	return ctrl.Result{}, nil
}

func (r *NnfNodeBlockStorageReconciler) allocateStorage(nodeBlockStorage *nnfv1alpha10.NnfNodeBlockStorage, index int) (*ctrl.Result, error) {
	log := r.Log.WithValues("NnfNodeBlockStorage", types.NamespacedName{Name: nodeBlockStorage.Name, Namespace: nodeBlockStorage.Namespace})

	ss := nnf.NewDefaultStorageService(r.Options.DeleteUnknownVolumes(), r.Options.ReplaceMissingVolumes())
	nvmeSS := nnfnvme.NewDefaultStorageService()

	allocationStatus := &nodeBlockStorage.Status.Allocations[index]

	storagePoolID := getStoragePoolID(nodeBlockStorage, index)
	sp, err := r.createStoragePool(ss, storagePoolID, nodeBlockStorage.Spec.Allocations[index].Capacity)
	if err != nil {
		return nil, dwsv1alpha7.NewResourceError("could not create storage pool").WithError(err).WithMajor()
	}

	vc := &sf.VolumeCollectionVolumeCollection{}
	if err := ss.StorageServiceIdStoragePoolIdCapacitySourceIdProvidingVolumesGet(ss.Id(), storagePoolID, "0", vc); err != nil {
		return nil, err
	}

	if len(allocationStatus.Devices) == 0 {
		allocationStatus.Devices = make([]nnfv1alpha10.NnfNodeBlockStorageDeviceStatus, len(vc.Members))
	}

	if len(allocationStatus.Devices) != len(vc.Members) {
		return nil, dwsv1alpha7.NewResourceError("unexpected number of namespaces").WithFatal()
	}

	for i, member := range vc.Members {
		components := strings.Split(member.OdataId, "/")
		storageId := components[4]
		volumeId := components[6]

		storage := &sf.StorageV190Storage{}
		if err := nvmeSS.StorageIdGet(storageId, storage); err != nil {
			return nil, err
		}

		volume := &sf.VolumeV161Volume{}
		if err := nvmeSS.StorageIdVolumeIdGet(storageId, volumeId, volume); err != nil {
			return nil, err
		}

		allocationStatus.Devices[i].NQN = strings.Replace(storage.Identifiers[0].DurableName, "\u0000", "", -1)
		allocationStatus.Devices[i].NamespaceId = volume.NVMeNamespaceProperties.NamespaceId
		allocationStatus.Devices[i].CapacityAllocated = volume.CapacityBytes
	}

	allocationStatus.CapacityAllocated = sp.CapacityBytes

	// If the SF ID is empty then we just created the resource. Save the ID in the NnfNodeBlockStorage
	if len(allocationStatus.StoragePoolId) == 0 {
		log.Info("Created storage pool", "Id", sp.Id)
		allocationStatus.StoragePoolId = sp.Id
	}

	return nil, nil
}

func (r *NnfNodeBlockStorageReconciler) createBlockDevice(ctx context.Context, nodeBlockStorage *nnfv1alpha10.NnfNodeBlockStorage, index int) (*ctrl.Result, error) {
	log := r.Log.WithValues("NnfNodeBlockStorage", types.NamespacedName{Name: nodeBlockStorage.Name, Namespace: nodeBlockStorage.Namespace})
	ss := nnf.NewDefaultStorageService(r.Options.DeleteUnknownVolumes(), r.Options.ReplaceMissingVolumes())

	allocationStatus := &nodeBlockStorage.Status.Allocations[index]

	// Create a Storage Group if none is currently present. Recall that a Storage Group
	// is a mapping from the Storage Pool to a Server Endpoint. Establishing a Storage
	// Group makes block storage available on the server, which itself is a prerequisite to
	// any file system built on top of the block storage.

	// Retrieve the collection of endpoints for us to map
	serverEndpointCollection := &sf.EndpointCollectionEndpointCollection{}
	if err := ss.StorageServiceIdEndpointsGet(ss.Id(), serverEndpointCollection); err != nil {
		return nil, dwsv1alpha7.NewResourceError("could not get service endpoint").WithError(err).WithFatal()
	}

	// Get the Storage resource to map between compute node name and
	// endpoint index.
	namespacedName := types.NamespacedName{
		Name:      nodeBlockStorage.Namespace, // The namespace tells us which Rabbit we are dealing with
		Namespace: "default",
	}

	storage := &dwsv1alpha7.Storage{}
	err := r.Get(ctx, namespacedName, storage)
	if err != nil {
		return nil, dwsv1alpha7.NewResourceError("could not read storage resource").WithError(err)
	}

	// Build a list of all nodes with access to the storage
	clients := []string{}
	for _, server := range storage.Status.Access.Servers {
		clients = append(clients, server.Name)
	}

	for _, compute := range storage.Status.Access.Computes {
		clients = append(clients, compute.Name)
	}

	// Make a list of all the endpoints and set whether they need a storage group based
	// on the list of clients specified in the ClientEndpoints array
	accessList := make([]string, len(serverEndpointCollection.Members))
	for _, nodeName := range nodeBlockStorage.Spec.Allocations[index].Access {
		for i, clientName := range clients {
			if nodeName == clientName {
				accessList[i] = nodeName
			}
		}
	}

	// Loop through the list of endpoints and delete the StorageGroup for endpoints where
	// access==false, and create the StorageGroup for endpoints where access==true
	for clientIndex, nodeName := range accessList {
		endpointRef := serverEndpointCollection.Members[clientIndex]
		endpointID := endpointRef.OdataId[strings.LastIndex(endpointRef.OdataId, "/")+1:]
		storageGroupId := fmt.Sprintf("%s-%d-%s", nodeBlockStorage.Name, index, endpointID)

		// If the endpoint doesn't need a storage group, remove one if it exists
		if nodeName == "" {
			if _, err := r.getStorageGroup(ss, storageGroupId); err == nil {
				if err := r.deleteStorageGroup(ss, storageGroupId); err != nil {
					return nil, dwsv1alpha7.NewResourceError("could not delete storage group").WithError(err).WithMajor()
				}
				log.Info("Deleted storage group", "storageGroupId", storageGroupId)
			}

			for oldNodeName, accessStatus := range allocationStatus.Accesses {
				if accessStatus.StorageGroupId == storageGroupId {
					delete(allocationStatus.Accesses, oldNodeName)
				}
			}

		} else {
			// Skip fenced compute nodes for GFS2 to avoid re-creating storage groups
			allocationSet := nodeBlockStorage.Labels["nnf.cray.hpe.com/allocationset"]
			if allocationSet == "gfs2" {
				fenced, err := r.checkComputeFenced(nodeName, log)
				if err != nil {
					log.Error(err, "Error checking for fence response", "compute", nodeName)
					// Continue despite error to avoid blocking reconciliation
				} else if fenced {
					log.Info("Skipping storage group creation for fenced compute node", "compute", nodeName, "storageGroupId", storageGroupId)
					continue
				}
			}

			// The kind environment doesn't support endpoints beyond the Rabbit
			if os.Getenv("ENVIRONMENT") == "kind" && endpointID != os.Getenv("RABBIT_NODE") {
				allocationStatus.Accesses[nodeName] = nnfv1alpha10.NnfNodeBlockStorageAccessStatus{StorageGroupId: storageGroupId}
				continue
			}

			endPoint, err := r.getEndpoint(ss, endpointID)
			if err != nil {
				return nil, dwsv1alpha7.NewResourceError("could not get endpoint").WithError(err).WithFatal()
			}

			// Skip the endpoints that are not ready
			if nnfv1alpha10.StaticResourceStatus(endPoint.Status) != nnfv1alpha10.ResourceReady {
				continue
			}

			sg, err := r.createStorageGroup(ss, storageGroupId, allocationStatus.StoragePoolId, endpointID)
			if err != nil {
				return nil, dwsv1alpha7.NewResourceError("could not create storage group").WithError(err).WithMajor()
			}

			if allocationStatus.Accesses == nil {
				allocationStatus.Accesses = make(map[string]nnfv1alpha10.NnfNodeBlockStorageAccessStatus)
			}

			// If the access status doesn't exist then we just created the resource. Save the ID in the NnfNodeBlockStorage
			if _, ok := allocationStatus.Accesses[nodeName]; !ok {
				log.Info("Created storage group", "Id", storageGroupId)
				allocationStatus.Accesses[nodeName] = nnfv1alpha10.NnfNodeBlockStorageAccessStatus{StorageGroupId: sg.Id}
			}

			// The device paths are discovered below. This is only relevant for the Rabbit node access
			if nodeName != clients[0] {
				continue
			}

			// Bail out if this is kind
			_, found := os.LookupEnv("NNF_TEST_ENVIRONMENT")
			if found || os.Getenv("ENVIRONMENT") == "kind" {
				continue
			}

			// Initialize the path array if it doesn't exist yet
			if len(allocationStatus.Accesses[nodeName].DevicePaths) != len(allocationStatus.Devices) {
				if access, ok := allocationStatus.Accesses[nodeName]; ok {
					access.DevicePaths = make([]string, len(allocationStatus.Devices))
					allocationStatus.Accesses[nodeName] = access
				}
			}

			foundDevices, err := nvme.NvmeListDevices(log)
			if err != nil {
				return nil, err
			}

			for i, allocatedDevice := range allocationStatus.Devices {
				findMatchingNvmeDevice := func() string {
					for _, foundDevice := range foundDevices {
						if allocatedDevice.NQN == foundDevice.NQN && allocatedDevice.NamespaceId == strconv.FormatUint(uint64(foundDevice.NSID), 10) {
							return foundDevice.DevicePath
						}
					}

					return ""
				}

				path := findMatchingNvmeDevice()
				if path == "" {
					err := nvme.NvmeRescanDevices(log)
					if err != nil {
						return nil, dwsv1alpha7.NewResourceError("could not rescan devices after failing to find device path for %v", allocatedDevice).WithError(err).WithMajor()
					}

					return nil, dwsv1alpha7.NewResourceError("could not find device path for %v", allocatedDevice).WithMajor()
				}

				allocationStatus.Accesses[nodeName].DevicePaths[i] = path
			}
		}
	}

	return nil, nil

}

func (r *NnfNodeBlockStorageReconciler) deleteStorage(nodeBlockStorage *nnfv1alpha10.NnfNodeBlockStorage, index int) (*ctrl.Result, error) {
	log := r.Log.WithValues("NnfNodeBlockStorage", types.NamespacedName{Name: nodeBlockStorage.Name, Namespace: nodeBlockStorage.Namespace})

	ss := nnf.NewDefaultStorageService(r.Options.DeleteUnknownVolumes(), r.Options.ReplaceMissingVolumes())

	storagePoolID := getStoragePoolID(nodeBlockStorage, index)
	log.Info("Deleting storage pool", "Id", storagePoolID)

	err := r.deleteStoragePool(ss, storagePoolID)
	if err != nil {
		ecErr, ok := err.(*ec.ControllerError)

		// If the error is from a 404 error, then there's nothing to clean up and we
		// assume everything has been deleted
		if !ok || ecErr.StatusCode() != http.StatusNotFound {
			nodeBlockStorage.Status.Error = dwsv1alpha7.NewResourceError("could not delete storage pool").WithError(err).WithFatal()
			log.Info(nodeBlockStorage.Status.Error.Error())

			return &ctrl.Result{Requeue: true}, nil
		}
	}

	return nil, nil
}

func getStoragePoolID(nodeBlockStorage *nnfv1alpha10.NnfNodeBlockStorage, index int) string {
	return fmt.Sprintf("%s-%d", nodeBlockStorage.Name, index)
}

func (r *NnfNodeBlockStorageReconciler) createStoragePool(ss nnf.StorageServiceApi, id string, capacity int64) (*sf.StoragePoolV150StoragePool, error) {
	sp := &sf.StoragePoolV150StoragePool{
		Id:            id,
		CapacityBytes: capacity,
		Oem: openapi.MarshalOem(nnf.AllocationPolicyOem{
			Policy:     nnf.SpareAllocationPolicyType,
			Compliance: nnf.StrictAllocationComplianceType,
		}),
	}

	if err := ss.StorageServiceIdStoragePoolIdPut(ss.Id(), id, sp); err != nil {
		resourceErr := dwsv1alpha7.NewResourceError("could not allocate storage pool").WithError(err)
		ecErr, ok := err.(*ec.ControllerError)
		if ok {
			switch ecErr.Cause() {
			case "Insufficient capacity available":
				// log which VGs and zpools exist to make it easier to tell why we ran out of space
				log := r.Log.WithValues("StoragePool ID", id)

				vgsOutput, err := command.Run("vgs -o vg_name,vg_tags,vg_size,vg_attr,pv_count,lv_count, --reportformat json", log)
				if err != nil {
					log.Info("vgs failed", "error", err)
				}
				zpoolOutput, err := command.Run("zfs list -H -o space,nnf:jobid", log)
				if err != nil {
					log.Info("zfs list failed", "error", err)
				}

				log.Info("insufficient capacity", "LVM volume groups", vgsOutput, "zfs datasets", zpoolOutput)

				return nil, resourceErr.WithUserMessage("%s: insufficient capacity available", os.Getenv("NNF_NODE_NAME")).WithWLM().WithFatal()
			default:
				return nil, resourceErr
			}
		}

		return nil, resourceErr
	}

	return sp, nil
}

func (r *NnfNodeBlockStorageReconciler) getStoragePool(ss nnf.StorageServiceApi, id string) (*sf.StoragePoolV150StoragePool, error) {
	sp := &sf.StoragePoolV150StoragePool{}

	if err := ss.StorageServiceIdStoragePoolIdGet(ss.Id(), id, sp); err != nil {
		return nil, err
	}

	return sp, nil
}

func (r *NnfNodeBlockStorageReconciler) deleteStoragePool(ss nnf.StorageServiceApi, id string) error {
	if err := ss.StorageServiceIdStoragePoolIdDelete(ss.Id(), id); err != nil {
		return err
	}

	return nil
}

func (r *NnfNodeBlockStorageReconciler) getEndpoint(ss nnf.StorageServiceApi, id string) (*sf.EndpointV150Endpoint, error) {
	ep := &sf.EndpointV150Endpoint{}

	if err := ss.StorageServiceIdEndpointIdGet(ss.Id(), id, ep); err != nil {
		return nil, err
	}

	return ep, nil
}

func (r *NnfNodeBlockStorageReconciler) createStorageGroup(ss nnf.StorageServiceApi, id string, spID string, epID string) (*sf.StorageGroupV150StorageGroup, error) {
	sp, err := r.getStoragePool(ss, spID)
	if err != nil {
		return nil, err
	}

	ep, err := r.getEndpoint(ss, epID)
	if err != nil {
		return nil, err
	}

	sg := &sf.StorageGroupV150StorageGroup{
		Id: id,
		Links: sf.StorageGroupV150Links{
			StoragePool:    sf.OdataV4IdRef{OdataId: sp.OdataId},
			ServerEndpoint: sf.OdataV4IdRef{OdataId: ep.OdataId},
		},
	}

	if err := ss.StorageServiceIdStorageGroupIdPut(ss.Id(), id, sg); err != nil {
		return nil, err
	}

	return sg, nil
}

func (r *NnfNodeBlockStorageReconciler) getStorageGroup(ss nnf.StorageServiceApi, id string) (*sf.StorageGroupV150StorageGroup, error) {
	sg := &sf.StorageGroupV150StorageGroup{}

	if err := ss.StorageServiceIdStorageGroupIdGet(ss.Id(), id, sg); err != nil {
		return nil, err
	}

	return sg, nil
}

func (r *NnfNodeBlockStorageReconciler) deleteStorageGroup(ss nnf.StorageServiceApi, id string) error {
	err := ss.StorageServiceIdStorageGroupIdDelete(ss.Id(), id)
	if err != nil {
		ecErr, ok := err.(*ec.ControllerError)
		// If the error is from a 404 error, treat as success (idempotent deletion)
		if ok && ecErr.StatusCode() == http.StatusNotFound {
			return nil
		}
	}
	return err
}

// getComputeNodesWithAccess returns a deduplicated list of compute node names
// that have access to this NnfNodeBlockStorage.
func (r *NnfNodeBlockStorageReconciler) getComputeNodesWithAccess(blockStorage *nnfv1alpha10.NnfNodeBlockStorage) []string {
	computeNodes := make(map[string]bool)

	for _, allocation := range blockStorage.Status.Allocations {
		if allocation.Accesses != nil {
			for computeNode := range allocation.Accesses {
				computeNodes[computeNode] = true
			}
		}
	}

	result := make([]string, 0, len(computeNodes))
	for computeNode := range computeNodes {
		result = append(result, computeNode)
	}
	return result
}

// scanFenceRequests scans the fence request directory for requests targeting any of the given compute nodes
func (r *NnfNodeBlockStorageReconciler) scanFenceRequests(computeNodes []string, log logr.Logger) ([]*fence.FenceRequest, error) {
	// Check if fence request directory exists
	if _, err := os.Stat(fence.RequestDir); os.IsNotExist(err) {
		return nil, nil
	}

	// Create a map for fast lookup
	computeNodeMap := make(map[string]bool)
	for _, node := range computeNodes {
		computeNodeMap[node] = true
	}

	// Read all files in the request directory
	entries, err := os.ReadDir(fence.RequestDir)
	if err != nil {
		return nil, err
	}

	var fenceRequests []*fence.FenceRequest

	// Check each request file. Errors reading individual files are logged but don't
	// prevent processing other valid requests.
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		requestFile := filepath.Join(fence.RequestDir, entry.Name())
		data, err := os.ReadFile(requestFile)
		if err != nil {
			log.Error(err, "Failed to read fence request file", "file", requestFile)
			continue
		}

		var request fence.FenceRequest
		if err := json.Unmarshal(data, &request); err != nil {
			log.Error(err, "Failed to parse fence request file", "file", requestFile)
			continue
		}

		// Store the file path for later cleanup
		request.FilePath = requestFile

		// If this request is for one of our compute nodes, check if already processed
		if computeNodeMap[request.TargetNode] {
			// Check if response already exists (handle crash between response write and request delete)
			responseFilename := filepath.Base(requestFile)
			responseFile := filepath.Join(fence.ResponseDir, responseFilename)
			if _, err := os.Stat(responseFile); err == nil {
				// Response exists but request wasn't deleted - clean up the orphaned request file
				log.Info("Response already exists for fence request, cleaning up orphaned request file",
					"requestID", request.RequestID,
					"computeNode", request.TargetNode,
					"requestFile", requestFile,
					"responseFile", responseFile)
				if err := os.Remove(requestFile); err != nil {
					log.Error(err, "Failed to remove orphaned fence request file", "file", requestFile)
				}
				continue
			}

			fenceRequests = append(fenceRequests, &request)
		}
	}

	return fenceRequests, nil
}

// processFenceRequests deletes storage groups for fenced compute nodes and writes response files.
func (r *NnfNodeBlockStorageReconciler) processFenceRequests(ctx context.Context, fenceRequests []*fence.FenceRequest, log logr.Logger) error {
	// Build deduplicated set of compute nodes from the fence requests
	computeNodeMap := make(map[string]bool)
	for _, request := range fenceRequests {
		computeNodeMap[request.TargetNode] = true
	}

	// List all NnfNodeBlockStorage resources in this rabbit node's namespace
	nnfNodeBlockStorageList := &nnfv1alpha10.NnfNodeBlockStorageList{}
	listOptions := []client.ListOption{
		client.InNamespace(r.Namespace),
	}
	if err := r.List(ctx, nnfNodeBlockStorageList, listOptions...); err != nil {
		return err
	}

	storageGroupsToDelete := []string{}

	for _, blockStorage := range nnfNodeBlockStorageList.Items {
		// Only GFS2 requires STONITH fencing
		// TODO: Consider a "requires-fencing" label based on filesystem type and profile config
		allocationSet, hasLabel := blockStorage.Labels["nnf.cray.hpe.com/allocationset"]
		if !hasLabel || allocationSet != "gfs2" {
			log.V(1).Info("Skipping non-GFS2 block storage",
				"blockStorage", blockStorage.Name,
				"allocationset", allocationSet)
			continue
		}

		// Check each allocation for accesses by any of the fenced compute nodes
		for _, allocation := range blockStorage.Status.Allocations {
			if allocation.Accesses == nil {
				continue
			}

			// Check if any fenced compute node has access to this allocation
			for computeNode := range computeNodeMap {
				if access, found := allocation.Accesses[computeNode]; found {
					storageGroupID := access.StorageGroupId
					if storageGroupID != "" {
						log.Info("Found GFS2 storage group for fenced node",
							"computeNode", computeNode,
							"storageGroupId", storageGroupID,
							"blockStorage", blockStorage.Name,
							"namespace", blockStorage.Namespace,
							"allocationset", allocationSet)
						storageGroupsToDelete = append(storageGroupsToDelete, storageGroupID)
					}
				}
			}
		}
	}

	// Determine success and message
	success := false
	message := ""
	actionPerformed := "off"

	// Get list of compute nodes for logging
	computeNodes := make([]string, 0, len(computeNodeMap))
	for node := range computeNodeMap {
		computeNodes = append(computeNodes, node)
	}

	// Delete the storage groups to fence the nodes
	if len(storageGroupsToDelete) > 0 {
		log.Info("Deleting GFS2 storage groups to fence nodes",
			"computeNodes", computeNodes,
			"storageGroupIds", storageGroupsToDelete,
			"count", len(storageGroupsToDelete))

		// Get the storage service
		ss := nnf.NewDefaultStorageService(r.Options.DeleteUnknownVolumes(), r.Options.ReplaceMissingVolumes())

		deletedCount := 0
		var deleteErrors []string

		// Delete each storage group
		for _, storageGroupID := range storageGroupsToDelete {
			if err := r.deleteStorageGroup(ss, storageGroupID); err != nil {
				log.Error(err, "Failed to delete storage group",
					"storageGroupId", storageGroupID,
					"computeNodes", computeNodes)
				deleteErrors = append(deleteErrors, fmt.Sprintf("%s: %v", storageGroupID, err))
			} else {
				log.Info("Successfully deleted storage group",
					"storageGroupId", storageGroupID,
					"computeNodes", computeNodes)
				deletedCount++
			}
		}

		// Set success based on whether we deleted all storage groups
		if deletedCount == len(storageGroupsToDelete) {
			success = true
			actionPerformed = "off"
			message = fmt.Sprintf("Successfully fenced node by deleting %d GFS2 storage groups", deletedCount)
		} else if deletedCount > 0 {
			success = false
			message = fmt.Sprintf("Partially fenced node: deleted %d of %d storage groups. Errors: %s",
				deletedCount, len(storageGroupsToDelete), strings.Join(deleteErrors, "; "))
		} else {
			success = false
			message = fmt.Sprintf("Failed to fence node: could not delete any storage groups. Errors: %s",
				strings.Join(deleteErrors, "; "))
		}
	} else {
		log.Info("No GFS2 storage groups found for compute nodes", "computeNodes", computeNodes)
		success = true
		actionPerformed = "off"
		message = fmt.Sprintf("No GFS2 storage groups found for compute nodes %v (already fenced or no GFS2 access)", computeNodes)
	}

	// Write response files. On failure, Pacemaker retries based on pcmk_off_retries.
	for _, request := range fenceRequests {
		// Write response file
		if err := r.writeFenceResponse(request, success, message, actionPerformed, log); err != nil {
			log.Error(err, "Failed to write fence response", "requestID", request.RequestID)
			// Don't delete the request file if we couldn't write the response
			// The fence agent needs the response to complete the fencing operation
			continue
		}

		// Clean up request file. checkComputeFenced() prevents re-creating storage groups
		// for fenced nodes, and workflow teardown cleans up any orphans.
		if request.FilePath != "" {
			if err := os.Remove(request.FilePath); err != nil {
				log.Error(err, "Failed to remove fence request file", "file", request.FilePath)
			} else {
				log.Info("Removed processed fence request file", "file", request.FilePath)
			}
		}
	}

	return nil
}

// writeFenceResponse writes a response file for the fence agent using atomic rename.
// Writes to a temp dot-file first, then renames to the final filename.
func (r *NnfNodeBlockStorageReconciler) writeFenceResponse(request *fence.FenceRequest, success bool, message string, actionPerformed string, log logr.Logger) error {
	// Ensure response directory exists
	if err := os.MkdirAll(fence.ResponseDir, 0755); err != nil {
		return err
	}

	// Use the same filename as the request file for consistency
	requestFilename := filepath.Base(request.FilePath)
	responseFile := filepath.Join(fence.ResponseDir, requestFilename)
	tempFile := filepath.Join(fence.ResponseDir, "."+requestFilename+".tmp")

	// Include all fields from the request in the response
	response := map[string]interface{}{
		"request_id":       request.RequestID,
		"timestamp":        request.Timestamp,
		"action":           request.Action,
		"target_node":      request.TargetNode,
		"recorder_node":    request.RecorderNode,
		"success":          success,
		"message":          message,
		"action_performed": actionPerformed,
	}

	data, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return err
	}

	// Write to temp file first
	if err := os.WriteFile(tempFile, data, 0644); err != nil {
		return err
	}

	// Atomic rename to final filename
	if err := os.Rename(tempFile, responseFile); err != nil {
		os.Remove(tempFile) // Clean up temp file on rename failure
		return err
	}

	log.Info("Wrote fence response", "requestID", request.RequestID, "file", responseFile, "success", success)
	return nil
}

// Enqueue all the NnfNodeBlockStorage resources after an nnf-ec node-up/node-down event. If we
// can't List() the NnfNodeBlockStorages, trigger the watch again after 10 seconds.
func (r *NnfNodeBlockStorageReconciler) NnfEcEventEnqueueHandler(ctx context.Context, o client.Object) []reconcile.Request {
	log := r.Log.WithValues("Event", "Enqueue")

	requests := []reconcile.Request{}

	// Find all the NnfNodeBlockStorage resources for this Rabbit so we can reconcile them.
	listOptions := []client.ListOption{
		client.InNamespace(os.Getenv("NNF_NODE_NAME")),
	}

	nnfNodeBlockStorageList := &nnfv1alpha10.NnfNodeBlockStorageList{}
	if err := r.List(context.TODO(), nnfNodeBlockStorageList, listOptions...); err != nil {
		log.Error(err, "Could not list block storages")

		// Wait ten seconds and trigger the watch again to retry
		go func() {
			time.Sleep(time.Second * 10)

			log.Info("triggering watch after List() error")
			r.Events <- event.GenericEvent{Object: &nnfv1alpha10.NnfNodeBlockStorage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "nnf-ec-event",
					Namespace: "nnf-ec-event",
				},
			}}
		}()

		return requests
	}

	for _, nnfNodeBlockStorage := range nnfNodeBlockStorageList.Items {
		requests = append(requests, reconcile.Request{NamespacedName: types.NamespacedName{Name: nnfNodeBlockStorage.GetName(), Namespace: nnfNodeBlockStorage.GetNamespace()}})
	}

	log.Info("Enqueuing resources", "requests", requests)

	return requests
}

// checkComputeFenced checks if a fence response file exists for the given compute node,
// indicating it has been fenced and should not have storage groups restored
func (r *NnfNodeBlockStorageReconciler) checkComputeFenced(computeName string, log logr.Logger) (bool, error) {
	// Check if fence response directory exists
	if _, err := os.Stat(fence.ResponseDir); os.IsNotExist(err) {
		return false, nil
	}

	// Read all files in the response directory
	entries, err := os.ReadDir(fence.ResponseDir)
	if err != nil {
		return false, err
	}

	// Check each response file to see if it's for this compute node
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		responseFile := filepath.Join(fence.ResponseDir, entry.Name())
		data, err := os.ReadFile(responseFile)
		if err != nil {
			log.Error(err, "Failed to read fence response file", "file", responseFile)
			continue
		}

		var response struct {
			TargetNode string `json:"target_node"`
			Success    bool   `json:"success"`
		}

		if err := json.Unmarshal(data, &response); err != nil {
			log.Error(err, "Failed to parse fence response file", "file", responseFile)
			continue
		}

		// If this response is for our compute node and the fence was successful, it's fenced
		if response.TargetNode == computeName && response.Success {
			return true, nil
		}
	}

	return false, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NnfNodeBlockStorageReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.Add(r); err != nil {
		return err
	}

	// nnf-ec is not thread safe, so we are limited to a single reconcile thread.
	builder := ctrl.NewControllerManagedBy(mgr).
		WithOptions(controller.Options{MaxConcurrentReconciles: 1}).
		For(&nnfv1alpha10.NnfNodeBlockStorage{}).
		WatchesRawSource(&source.Channel{Source: r.Events}, handler.EnqueueRequestsFromMapFunc(r.NnfEcEventEnqueueHandler))

	// Watch fence events to trigger reconciliation for storage group deletion
	if r.FenceEvents != nil {
		r.Log.Info("Watching FenceEvents channel")
		builder = builder.WatchesRawSource(&source.Channel{Source: r.FenceEvents}, handler.EnqueueRequestsFromMapFunc(r.NnfEcEventEnqueueHandler))
	}

	return builder.Complete(r)
}
