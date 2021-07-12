/*
Copyright 2021 Hewlett Packard Enterprise Development LP
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

	ec "stash.us.cray.com/rabsw/nnf-ec/pkg"
	nnf "stash.us.cray.com/rabsw/nnf-ec/pkg/manager-nnf"
	nnfserver "stash.us.cray.com/rabsw/nnf-ec/pkg/manager-server"

	openapi "stash.us.cray.com/rabsw/rfsf-openapi/pkg/common"
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

	if node.Status.Status != nnfv1alpha1.ResourceStatus(storageService.Status) ||
		node.Status.Health != nnfv1alpha1.ResourceHealth(storageService.Status) {
		status.Update(func(s *nnfv1alpha1.NnfNodeStatus) {
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

		if node.Status.Servers[idx].Status != nnfv1alpha1.ResourceStatus(storageService.Status) || node.Status.Servers[idx].Health != nnfv1alpha1.ResourceHealth(serverEndpoint.Status) {
			status.Update(func(s *nnfv1alpha1.NnfNodeStatus) {
				s.Servers[idx].Status = nnfv1alpha1.ResourceStatus(storageService.Status)
				s.Servers[idx].Health = nnfv1alpha1.ResourceHealth(serverEndpoint.Status)
			})
		}
	}

	// Iterate over the storage specifications for this NNF Node; try to resolve
	// the desired state of each storage spec, and bring the corresponding storage
	// status up to date.
StorageSpecLoop:
	for storageIdx, storage := range node.Spec.Storage {

		// Each storage spec should have a corresponding status
		var storageStatus *nnfv1alpha1.NnfNodeStorageStatus = nil
		var storageConditions []metav1.Condition = nil
		for statusIdx := range node.Status.Storage {
			if storage.Uuid == node.Status.Storage[statusIdx].Uuid {
				storageStatus = &node.Status.Storage[statusIdx]
				storageConditions = storageStatus.Conditions
				break
			}
		}

		if storageStatus == nil {

			status.Update(func(s *nnfv1alpha1.NnfNodeStatus) {
				creationTime := metav1.Now()
				s.Storage = append(s.Storage, nnfv1alpha1.NnfNodeStorageStatus{
					Uuid:              storage.Uuid,
					CreationTime:      &creationTime,
					Status:            nnfv1alpha1.ResourceStarting,
					CapacityAllocated: 0,
					CapacityUsed:      0,
					Conditions:        nnfv1alpha1.NewConditions(),
				})
			})

			storageStatus = &node.Status.Storage[len(node.Status.Storage)-1]
			storageConditions = storageStatus.Conditions

			storagePool := &sf.StoragePoolV150StoragePool{
				CapacityBytes: storage.Capacity,
				Oem: openapi.MarshalOem(nnf.AllocationPolicyOem{
					Policy:     nnf.SpareAllocationPolicyType,
					Compliance: nnf.RelaxedAllocationComplianceType,
				}),
			}

			condition := &storageConditions[nnfv1alpha1.ConditionIndexCreateStoragePool]

			if err := ss.StorageServiceIdStoragePoolsPost(ss.Id(), storagePool); err != nil {
				status.Update(func(*nnfv1alpha1.NnfNodeStatus) {
					storageStatus.Status = nnfv1alpha1.ResourceFailed

					condition.Reason = nnfv1alpha1.ConditionFailed
					condition.Message = err.Error()
				})

				continue StorageSpecLoop
			}

			status.Update(func(*nnfv1alpha1.NnfNodeStatus) {
				storageStatus.Id = storagePool.Id
				storageStatus.Status = nnfv1alpha1.ResourceStatus(storagePool.Status)
				storageStatus.Health = nnfv1alpha1.ResourceHealth(storagePool.Status)
				storageStatus.CapacityAllocated = storagePool.CapacityBytes

				storageStatus.Servers = make([]nnfv1alpha1.NnfNodeStorageServerStatus, len(storage.Servers))
				for serverIdx := range storage.Servers {
					serverStatus := &storageStatus.Servers[serverIdx]
					serverStatus.Status = nnfv1alpha1.ResourceStarting
					storage.Servers[serverIdx].DeepCopyInto(&serverStatus.NnfNodeStorageServerSpec)
				}

				condition.Status = metav1.ConditionFalse
				condition.Reason = nnfv1alpha1.ConditionSuccess
			})

		} // if storageStatus == nil

		if storageStatus.Status == nnfv1alpha1.ResourceFailed && storage.State != nnfv1alpha1.ResourceDestroy {
			log.Info("Skipping Storage Pool %s: Status Failed", storageStatus.Id)
			continue StorageSpecLoop
		}

		sp := sf.StoragePoolV150StoragePool{}
		if err := ss.StorageServiceIdStoragePoolIdGet(ss.Id(), storageStatus.Id, &sp); err != nil {
			status.Update(func(*nnfv1alpha1.NnfNodeStatus) {
				storageStatus.Status = nnfv1alpha1.ResourceFailed
				nnfv1alpha1.SetGetResourceFailureCondition(storageConditions, err)
			})

			continue StorageSpecLoop
		}

		// Check if the resource is to be destroyed
		if storage.State == nnfv1alpha1.ResourceDestroy {
			if storageStatus.Status == nnfv1alpha1.ResourceDeleting {
				continue StorageSpecLoop
			}

			condition := &storageConditions[nnfv1alpha1.ConditionIndexDeleteStoragePool]

			status.Update(func(*nnfv1alpha1.NnfNodeStatus) {
				storageStatus.Status = nnfv1alpha1.ResourceDeleting

				condition.Status = metav1.ConditionTrue
				condition.LastTransitionTime = metav1.Now()
			})

			if err := ss.StorageServiceIdStoragePoolIdDelete(ss.Id(), sp.Id); err != nil {

				status.Update(func(*nnfv1alpha1.NnfNodeStatus) {
					storageStatus.Status = nnfv1alpha1.ResourceFailed

					condition.Reason = nnfv1alpha1.ConditionFailed
					condition.Message = err.Error()
				})

				continue StorageSpecLoop
			}

			// Remove this storage and storage status from the list of managed resources.
			// This will cause the correspoding arrays to shift in order, so we can no
			// longer rely on the processing loop; thus we need to return with a requeue
			// to continue processing the other elements in correct order.

			node.Spec.Storage = append(node.Spec.Storage[:storageIdx], node.Spec.Storage[storageIdx+1:]...)
			node.Status.Storage = append(node.Status.Storage[:storageIdx], node.Status.Storage[storageIdx+1:]...)

			// Perform the updates required to update the node spec/status. Note that the
			// status is updated via the defer method above.
			// TODO: We will need to handle failure here where the Spec/Status fields get out-of-sync.
			if err := r.Update(ctx, node); err != nil {
				log.Error(err, "Failed to update Node", "Node.Name", node.Name, "Node.Namespace", node.Namespace)
				return ctrl.Result{}, err
			}

			return ctrl.Result{Requeue: true}, nil
		} // if storage.State == nnfv1alpha1.ResourceDestroy

		// Check if the storage status or health need to change to reflect the
		// current storage pool status.
		if storageStatus.Status != nnfv1alpha1.ResourceStatus(sp.Status) || storageStatus.Health != nnfv1alpha1.ResourceHealth(sp.Status) {
			status.Update(func(*nnfv1alpha1.NnfNodeStatus) {
				storageStatus.Status = nnfv1alpha1.ResourceStatus(sp.Status)
				storageStatus.Health = nnfv1alpha1.ResourceHealth(sp.Status)
			})
		}

		// Check if a file system is defined
		fs := sf.FileSystemV122FileSystem{}
		if len(storage.FileSystem) != 0 {
			if len(sp.Links.FileSystem.OdataId) == 0 {

				fs = sf.FileSystemV122FileSystem{
					Links: sf.FileSystemV122Links{
						StoragePool: sf.OdataV4IdRef{
							OdataId: sp.OdataId,
						},
					},
					Oem: openapi.MarshalOem(nnfserver.FileSystemOem{
						Name: storage.FileSystem,
					}),
				}

				condition := &storageConditions[nnfv1alpha1.ConditionIndexCreateFileSystem]

				status.Update(func(*nnfv1alpha1.NnfNodeStatus) {
					condition.Status = metav1.ConditionTrue
					condition.LastTransitionTime = metav1.Now()
				})

				if err := ss.StorageServiceIdFileSystemsPost(ss.Id(), &fs); err != nil {
					status.Update(func(*nnfv1alpha1.NnfNodeStatus) {
						storageStatus.Status = nnfv1alpha1.ResourceFailed

						condition.Reason = nnfv1alpha1.ConditionFailed
						condition.Message = err.Error()
					})

					continue StorageSpecLoop
				}

				status.Update(func(*nnfv1alpha1.NnfNodeStatus) {
					condition.Status = metav1.ConditionFalse
					condition.Reason = nnfv1alpha1.ConditionSuccess
				})

			} else {

				fsid := sp.Links.FileSystem.OdataId[strings.LastIndex(sp.Links.FileSystem.OdataId, "/")+1:]
				if err := ss.StorageServiceIdFileSystemIdGet(ss.Id(), fsid, &fs); err != nil {
					status.Update(func(*nnfv1alpha1.NnfNodeStatus) {
						storageStatus.Status = nnfv1alpha1.ResourceFailed
						nnfv1alpha1.SetGetResourceFailureCondition(storageConditions, err)
					})

					continue StorageSpecLoop
				}
			}
		}

		// Iterate over all the attached servers from the storage spec and reconcile
		// the server storage attributes as requested. The first step is done by mapping
		// the NVMe Namespaces that make up the Storage Pool to an individual server - this
		// is a so called Storage Group. The second step is optional and will establish
		// the file system onto the desired server.
		for serverIdx := range storageStatus.Servers {
			storageServer := &storage.Servers[serverIdx]
			storageServerStatus := &storageStatus.Servers[serverIdx]

			// Make sure the client hasn't rearranged the server spec and status
			// fields. We could sort them, but in reality they should never be
			// rearranging them - so we just fail if a mismatch occurs.
			if storageServer.Id != storageServerStatus.Id {
				status.Update(func(*nnfv1alpha1.NnfNodeStatus) {
					storageStatus.Status = nnfv1alpha1.ResourceInvalid

					nnfv1alpha1.SetResourceInvalidCondition(
						storageConditions,
						fmt.Errorf("Server Id %s does not match Server Status Id %s", storageServer.Id, storageServerStatus.Id),
					)
				})

				continue StorageSpecLoop
			}

			// Retrieve the server to confirm it is up and available for storage
			srvr := sf.EndpointV150Endpoint{}
			if err := ss.StorageServiceIdEndpointIdGet(ss.Id(), storageServer.Id, &srvr); err != nil {

				// TODO: We should differentiate between a bad request (bad server id/name) and a failed
				// request (could not get status)

				status.Update(func(*nnfv1alpha1.NnfNodeStatus) {
					storageStatus.Status = nnfv1alpha1.ResourceFailed
					nnfv1alpha1.SetGetResourceFailureCondition(storageConditions, err)
				})

				continue StorageSpecLoop
			}

			sgid := storageServerStatus.StorageGroup.Id
			shid := storageServerStatus.FileShare.Id

			if len(sgid) == 0 {

				condition := &storageConditions[nnfv1alpha1.ConditionIndexCreateStorageGroup]

				status.Update(func(*nnfv1alpha1.NnfNodeStatus) {
					condition.Status = metav1.ConditionTrue
					condition.ObservedGeneration = int64(serverIdx)
					condition.LastTransitionTime = metav1.Now()
				})

				sg := sf.StorageGroupV150StorageGroup{
					Links: sf.StorageGroupV150Links{
						StoragePool:    sf.OdataV4IdRef{OdataId: sp.OdataId},
						ServerEndpoint: sf.OdataV4IdRef{OdataId: srvr.OdataId},
					},
				}

				if err := ss.StorageServiceIdStorageGroupPost(ss.Id(), &sg); err != nil {
					status.Update(func(*nnfv1alpha1.NnfNodeStatus) {
						storageServerStatus.Status = nnfv1alpha1.ResourceFailed

						condition.Reason = nnfv1alpha1.ResourceFailed
						condition.Message = err.Error()
					})

					continue StorageSpecLoop
				}

				status.Update(func(*nnfv1alpha1.NnfNodeStatus) {
					storageServerStatus.StorageGroup.Id = sg.Id
					storageServerStatus.StorageGroup.Status = nnfv1alpha1.ResourceStatus(sg.Status)
					storageServerStatus.StorageGroup.Health = nnfv1alpha1.ResourceHealth(sg.Status)

					condition.Status = metav1.ConditionFalse
					condition.Reason = nnfv1alpha1.ConditionSuccess
				})

			} else {

				sg := sf.StorageGroupV150StorageGroup{}
				if err := ss.StorageServiceIdStorageGroupIdGet(ss.Id(), sgid, &sg); err != nil {
					status.Update(func(*nnfv1alpha1.NnfNodeStatus) {
						storageServerStatus.Status = nnfv1alpha1.ResourceFailed
						nnfv1alpha1.SetGetResourceFailureCondition(storageConditions, err)
					})

					continue StorageSpecLoop
				}

				if storageServerStatus.StorageGroup.Status != nnfv1alpha1.ResourceStatus(sg.Status) || storageServerStatus.StorageGroup.Health != nnfv1alpha1.ResourceHealth(sg.Status) {
					status.Update(func(*nnfv1alpha1.NnfNodeStatus) {
						storageServerStatus.StorageGroup.Status = nnfv1alpha1.ResourceStatus(sg.Status)
						storageServerStatus.StorageGroup.Health = nnfv1alpha1.ResourceHealth(sg.Status)
					})
				}
			}

			if len(storageServer.Path) != 0 {

				if len(shid) == 0 {

					condition := &storageConditions[nnfv1alpha1.ConditionIndexCreateFileShare]

					status.Update(func(*nnfv1alpha1.NnfNodeStatus) {
						condition.Status = metav1.ConditionTrue
						condition.ObservedGeneration = int64(serverIdx)
						condition.LastTransitionTime = metav1.Now()
					})

					sh := sf.FileShareV120FileShare{
						FileSharePath: storageServer.Path,
						Links: sf.FileShareV120Links{
							FileSystem: sf.OdataV4IdRef{OdataId: fs.OdataId},
							Endpoint:   sf.OdataV4IdRef{OdataId: srvr.OdataId},
						},
					}

					if err := ss.StorageServiceIdFileSystemIdExportedSharesPost(ss.Id(), fs.Id, &sh); err != nil {
						status.Update(func(*nnfv1alpha1.NnfNodeStatus) {
							storageServerStatus.Status = nnfv1alpha1.ResourceFailed

							condition.Reason = nnfv1alpha1.ConditionFailed
							condition.Message = err.Error()
						})

						continue StorageSpecLoop
					}

					status.Update(func(*nnfv1alpha1.NnfNodeStatus) {
						storageServerStatus.FileShare.Id = sh.Id
						storageServerStatus.FileShare.Status = nnfv1alpha1.ResourceStatus(sh.Status)
						storageServerStatus.FileShare.Health = nnfv1alpha1.ResourceHealth(sh.Status)

						condition.Status = metav1.ConditionFalse
						condition.Reason = nnfv1alpha1.ConditionSuccess
					})

				} else {

					sh := sf.FileShareV120FileShare{}
					if err := ss.StorageServiceIdFileSystemIdExportedShareIdGet(ss.Id(), fs.Id, shid, &sh); err != nil {
						status.Update(func(*nnfv1alpha1.NnfNodeStatus) {
							storageServerStatus.Status = nnfv1alpha1.ResourceFailed
							nnfv1alpha1.SetGetResourceFailureCondition(storageConditions, err)
						})

						continue StorageSpecLoop
					}

					if storageServerStatus.FileShare.Status != nnfv1alpha1.ResourceStatus(sh.Status) || storageServerStatus.FileShare.Health != nnfv1alpha1.ResourceHealth(sh.Status) {
						status.Update(func(*nnfv1alpha1.NnfNodeStatus) {
							storageServerStatus.FileShare.Status = nnfv1alpha1.ResourceStatus(sh.Status)
							storageServerStatus.FileShare.Health = nnfv1alpha1.ResourceHealth(sh.Status)
						})
					}
				}
			} // if len(storageServer.Path) != 0
		} // for serverIdx := range storageStatus.Servers

		// At this point, any storage resource has fully initialized
		// and is ready for use - transition to the Ready state.
		if storageStatus.Status == nnfv1alpha1.ResourceStarting {
			status.Update(func(status *nnfv1alpha1.NnfNodeStatus) {
				storageStatus.Status = nnfv1alpha1.ResourceReady
			})
		}

	} // for _, storage := range node.Spec.Storage

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
			Name:  r.Namespace, // Note the conversion here from namespace to name, each NNF Node is given a unique namespace, which then becomes how the NLC is controlled.
			Pod:   os.ExpandEnv("NNF_POD_NAME"), // Providing the podname gives users quick means to query the pod for a particular NNF Node
			State: nnfv1alpha1.ResourceEnable,
		},
		Status: nnfv1alpha1.NnfNodeStatus{
			Status:   nnfv1alpha1.ResourceStarting,
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
