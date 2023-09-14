/*
 * Copyright 2023 Hewlett Packard Enterprise Development LP
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
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	ec "github.com/NearNodeFlash/nnf-ec/pkg/ec"
	nnf "github.com/NearNodeFlash/nnf-ec/pkg/manager-nnf"
	nnfnvme "github.com/NearNodeFlash/nnf-ec/pkg/manager-nvme"
	openapi "github.com/NearNodeFlash/nnf-ec/pkg/rfsf/pkg/common"
	sf "github.com/NearNodeFlash/nnf-ec/pkg/rfsf/pkg/models"

	dwsv1alpha2 "github.com/DataWorkflowServices/dws/api/v1alpha2"
	"github.com/DataWorkflowServices/dws/utils/updater"
	nnfv1alpha1 "github.com/NearNodeFlash/nnf-sos/api/v1alpha1"
	"github.com/NearNodeFlash/nnf-sos/internal/controller/metrics"
	"github.com/NearNodeFlash/nnf-sos/pkg/blockdevice/nvme"
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
	Log    logr.Logger
	Scheme *kruntime.Scheme

	types.NamespacedName
}

//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfnodeblockstorages,verbs=get;list;watch;create;update;patch;delete;deletecollection
//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfnodeblockstorages/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=nnf.cray.hpe.com,resources=nnfnodeblockstorages/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.2/pkg/reconcile
func (r *NnfNodeBlockStorageReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, err error) {
	log := r.Log.WithValues("NnfNodeBlockStorage", req.NamespacedName)
	metrics.NnfNodeBlockStorageReconcilesTotal.Inc()

	nodeBlockStorage := &nnfv1alpha1.NnfNodeBlockStorage{}
	if err := r.Get(ctx, req.NamespacedName, nodeBlockStorage); err != nil {
		// ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Ensure the NNF Storage Service is running prior to taking any action.
	ss := nnf.NewDefaultStorageService()
	storageService := &sf.StorageServiceV150StorageService{}
	if err := ss.StorageServiceIdGet(ss.Id(), storageService); err != nil {
		return ctrl.Result{}, err
	}

	if storageService.Status.State != sf.ENABLED_RST {
		return ctrl.Result{RequeueAfter: 1 * time.Second}, nil
	}

	statusUpdater := updater.NewStatusUpdater[*nnfv1alpha1.NnfNodeBlockStorageStatus](nodeBlockStorage)
	defer func() { err = statusUpdater.CloseWithStatusUpdate(ctx, r.Client.Status(), err) }()
	defer func() { nodeBlockStorage.Status.SetResourceErrorAndLog(err, log) }()

	if !nodeBlockStorage.GetDeletionTimestamp().IsZero() {
		if !controllerutil.ContainsFinalizer(nodeBlockStorage, finalizerNnfNodeBlockStorage) {
			return ctrl.Result{}, nil
		}

		for i := range nodeBlockStorage.Status.Allocations {
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
		nodeBlockStorage.Status.Allocations = make([]nnfv1alpha1.NnfNodeBlockStorageAllocationStatus, len(nodeBlockStorage.Spec.Allocations))
		for i := range nodeBlockStorage.Status.Allocations {
			nodeBlockStorage.Status.Allocations[i].Accesses = make(map[string]nnfv1alpha1.NnfNodeBlockStorageAccessStatus)
		}

		return ctrl.Result{}, nil
	}

	// Loop through each allocation and create the storage
	for i := range nodeBlockStorage.Spec.Allocations {
		// Allocate physical storage
		result, err := r.allocateStorage(nodeBlockStorage, i)
		if err != nil {
			return ctrl.Result{}, dwsv1alpha2.NewResourceError("unable to allocate NVMe namespaces for allocation %v", i).WithError(err).WithMajor()
		}
		if result != nil {
			return *result, nil
		}

		// Create a block device in /dev that is accessible on the Rabbit node
		result, err = r.createBlockDevice(ctx, nodeBlockStorage, i)
		if err != nil {
			return ctrl.Result{}, dwsv1alpha2.NewResourceError("unable to attache NVMe namespace to node for allocation %v", i).WithError(err).WithMajor()
		}
		if result != nil {
			return *result, nil
		}
	}

	nodeBlockStorage.Status.Ready = true

	return ctrl.Result{}, nil
}

func (r *NnfNodeBlockStorageReconciler) allocateStorage(nodeBlockStorage *nnfv1alpha1.NnfNodeBlockStorage, index int) (*ctrl.Result, error) {
	log := r.Log.WithValues("NnfNodeBlockStorage", types.NamespacedName{Name: nodeBlockStorage.Name, Namespace: nodeBlockStorage.Namespace})

	ss := nnf.NewDefaultStorageService()
	nvmeSS := nnfnvme.NewDefaultStorageService()

	allocationStatus := &nodeBlockStorage.Status.Allocations[index]

	storagePoolID := fmt.Sprintf("%s-%d", nodeBlockStorage.Name, index)
	sp, err := r.createStoragePool(ss, storagePoolID, nodeBlockStorage.Spec.Allocations[index].Capacity)
	if err != nil {
		return &ctrl.Result{}, dwsv1alpha2.NewResourceError("could not create storage pool").WithError(err).WithMajor()

	}

	vc := &sf.VolumeCollectionVolumeCollection{}
	if err := ss.StorageServiceIdStoragePoolIdCapacitySourceIdProvidingVolumesGet(ss.Id(), storagePoolID, "0", vc); err != nil {
		return nil, err
	}

	if len(allocationStatus.Devices) == 0 {
		allocationStatus.Devices = make([]nnfv1alpha1.NnfNodeBlockStorageDeviceStatus, len(vc.Members))
	}

	if len(allocationStatus.Devices) != len(vc.Members) {
		return &ctrl.Result{}, dwsv1alpha2.NewResourceError("unexpected number of namespaces").WithFatal()
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

		return &ctrl.Result{}, nil
	}

	return nil, nil
}

func (r *NnfNodeBlockStorageReconciler) createBlockDevice(ctx context.Context, nodeBlockStorage *nnfv1alpha1.NnfNodeBlockStorage, index int) (*ctrl.Result, error) {
	log := r.Log.WithValues("NnfNodeBlockStorage", types.NamespacedName{Name: nodeBlockStorage.Name, Namespace: nodeBlockStorage.Namespace})
	ss := nnf.NewDefaultStorageService()

	allocationStatus := &nodeBlockStorage.Status.Allocations[index]

	// Create a Storage Group if none is currently present. Recall that a Storage Group
	// is a mapping from the Storage Pool to a Server Endpoint. Establishing a Storage
	// Group makes block storage available on the server, which itself is a prerequisite to
	// any file system built on top of the block storage.

	// Retrieve the collection of endpoints for us to map
	serverEndpointCollection := &sf.EndpointCollectionEndpointCollection{}
	if err := ss.StorageServiceIdEndpointsGet(ss.Id(), serverEndpointCollection); err != nil {
		return nil, dwsv1alpha2.NewResourceError("could not get service endpoint").WithError(err).WithFatal()
	}

	// Get the Storage resource to map between compute node name and
	// endpoint index.
	namespacedName := types.NamespacedName{
		Name:      nodeBlockStorage.Namespace,
		Namespace: "default",
	}

	storage := &dwsv1alpha2.Storage{}
	err := r.Get(ctx, namespacedName, storage)
	if err != nil {
		return nil, dwsv1alpha2.NewResourceError("could not read storage resource").WithError(err)
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
			if _, err := r.getStorageGroup(ss, storageGroupId); err != nil {
				continue
			}

			if err := r.deleteStorageGroup(ss, storageGroupId); err != nil {
				return nil, dwsv1alpha2.NewResourceError("could not delete storage group").WithError(err).WithMajor()
			}

			delete(allocationStatus.Accesses, nodeName)

			log.Info("Deleted storage group", "storageGroupId", storageGroupId)
		} else {
			// The kind environment doesn't support endpoints beyond the Rabbit
			if os.Getenv("ENVIRONMENT") == "kind" && endpointID != os.Getenv("RABBIT_NODE") {
				continue
			}

			endPoint, err := r.getEndpoint(ss, endpointID)
			if err != nil {
				return nil, dwsv1alpha2.NewResourceError("could not get endpoint").WithError(err).WithFatal()
			}

			// Skip the endpoints that are not ready
			if nnfv1alpha1.StaticResourceStatus(endPoint.Status) != nnfv1alpha1.ResourceReady {
				continue
			}

			sg, err := r.createStorageGroup(ss, storageGroupId, allocationStatus.StoragePoolId, endpointID)
			if err != nil {
				return &ctrl.Result{}, dwsv1alpha2.NewResourceError("could not create storage group").WithError(err).WithMajor()
			}

			if allocationStatus.Accesses == nil {
				allocationStatus.Accesses = make(map[string]nnfv1alpha1.NnfNodeBlockStorageAccessStatus)
			}

			// If the access status doesn't exist then we just created the resource. Save the ID in the NnfNodeBlockStorage
			if _, ok := allocationStatus.Accesses[nodeName]; !ok {
				log.Info("Created storage group", "Id", storageGroupId)
				allocationStatus.Accesses[nodeName] = nnfv1alpha1.NnfNodeBlockStorageAccessStatus{StorageGroupId: sg.Id}

				return &ctrl.Result{}, nil
			}

			// The device paths are discovered below. This is only relevant for the Rabbit node access
			if nodeName != clients[0] {
				return nil, nil
			}

			//
			_, found := os.LookupEnv("NNF_TEST_ENVIRONMENT")
			if found || os.Getenv("ENVIRONMENT") == "kind" {
				return nil, nil
			}

			// Initialize the path array if it doesn't exist yet
			if len(allocationStatus.Accesses[nodeName].DevicePaths) != len(allocationStatus.Devices) {
				if access, ok := allocationStatus.Accesses[nodeName]; ok {
					access.DevicePaths = make([]string, len(allocationStatus.Devices))
					allocationStatus.Accesses[nodeName] = access
				}
			}

			foundDevices, err := nvme.NvmeListDevices()
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
					err := nvme.NvmeRescanDevices()
					if err != nil {
						return nil, dwsv1alpha2.NewResourceError("could not rescan devices after failing to find device path for %v", allocatedDevice).WithError(err).WithMajor()
					}

					return nil, dwsv1alpha2.NewResourceError("could not find device path for %v", allocatedDevice).WithMajor()
				}

				allocationStatus.Accesses[nodeName].DevicePaths[i] = path
			}
		}
	}

	return nil, nil

}

func (r *NnfNodeBlockStorageReconciler) deleteStorage(nodeBlockStorage *nnfv1alpha1.NnfNodeBlockStorage, index int) (*ctrl.Result, error) {
	log := r.Log.WithValues("NnfNodeBlockStorage", types.NamespacedName{Name: nodeBlockStorage.Name, Namespace: nodeBlockStorage.Namespace})

	ss := nnf.NewDefaultStorageService()

	allocationStatus := &nodeBlockStorage.Status.Allocations[index]
	if allocationStatus.StoragePoolId == "" {
		return nil, nil
	}

	log.Info("Deleting storage pool", "Id", allocationStatus.StoragePoolId)

	err := r.deleteStoragePool(ss, allocationStatus.StoragePoolId)
	if err != nil {
		ecErr, ok := err.(*ec.ControllerError)

		// If the error is from a 404 error, then there's nothing to clean up and we
		// assume everything has been deleted
		if !ok || ecErr.StatusCode() != http.StatusNotFound {
			nodeBlockStorage.Status.Error = dwsv1alpha2.NewResourceError("could not delete storage pool").WithError(err).WithFatal()
			log.Info(nodeBlockStorage.Status.Error.Error())

			return &ctrl.Result{Requeue: true}, nil
		}
	}

	return nil, nil
}

func (r *NnfNodeBlockStorageReconciler) createStoragePool(ss nnf.StorageServiceApi, id string, capacity int64) (*sf.StoragePoolV150StoragePool, error) {
	sp := &sf.StoragePoolV150StoragePool{
		Id:            id,
		CapacityBytes: capacity,
		Oem: openapi.MarshalOem(nnf.AllocationPolicyOem{
			Policy:     nnf.SpareAllocationPolicyType,
			Compliance: nnf.RelaxedAllocationComplianceType,
		}),
	}

	if err := ss.StorageServiceIdStoragePoolIdPut(ss.Id(), id, sp); err != nil {
		resourceErr := dwsv1alpha2.NewResourceError("could not allocate storage pool").WithError(err)
		ecErr, ok := err.(*ec.ControllerError)
		if ok {
			switch ecErr.Cause() {
			case "Insufficient capacity available":
				return nil, resourceErr.WithUserMessage("insufficient capacity available").WithWLM().WithFatal()
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
	return ss.StorageServiceIdStorageGroupIdDelete(ss.Id(), id)
}

// SetupWithManager sets up the controller with the Manager.
func (r *NnfNodeBlockStorageReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// nnf-ec is not thread safe, so we are limited to a single reconcile thread.
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(controller.Options{MaxConcurrentReconciles: 1}).
		For(&nnfv1alpha1.NnfNodeBlockStorage{}).
		Complete(r)
}
