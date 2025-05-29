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
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"

	dwsv1alpha4 "github.com/DataWorkflowServices/dws/api/v1alpha4"
	nnfv1alpha7 "github.com/NearNodeFlash/nnf-sos/api/v1alpha7"
	nnftoken "github.com/NearNodeFlash/nnf-sos/pkg/token"
	"github.com/go-logr/logr"
	mpicommonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	mpiv2beta1 "github.com/kubeflow/mpi-operator/pkg/apis/kubeflow/v2beta1"
	"go.openly.dev/pointy"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type nnfUserContainer struct {
	workflow    *dwsv1alpha4.Workflow
	profile     *nnfv1alpha7.NnfContainerProfile
	nnfNodes    []string
	volumes     []nnfContainerVolume
	secrets     []nnfContainerSecret
	username    string
	uid, gid    int64
	client      client.Client
	log         logr.Logger
	scheme      *runtime.Scheme
	ctx         context.Context
	index       int
	copyOffload bool
}

// This struct contains all the necessary information for mounting container storages
type nnfContainerVolume struct {
	name           string
	command        string
	directiveName  string
	directiveIndex int
	mountPath      string
	envVarName     string
	pvcName        string
}

// This struct contains all the necessary information for mounting a secret into a container
type nnfContainerSecret struct {
	name       string
	mountPath  string
	secretName string
	// The files to mount from that secret and the env var to use for each one.
	envVarsToFileNames map[string]string
}

const (
	requiresContainerAuth = "container-auth"
	requiresCopyOffload   = "copy-offload"

	// The name of the secret that holds the TLS certificate and signing key for
	// containers that use "requiresContainerAuth" or "requiresCopyOffload".
	userContainerTLSSecretName      = "nnf-dm-usercontainer-server-tls"
	userContainerTLSSecretNamespace = corev1.NamespaceDefault

	// The version of the copy offload service that this controller supports. This is used to send
	// shutdown messages to user containers.
	copyOffloadAPIVersion = "1.0"
)

// MPI container workflow. In this model, we use mpi-operator to create an MPIJob, which creates
// a job for the launcher (to run mpirun) and a replicaset for the worker pods. The worker nodes
// run an ssh server tn listen for mpirun operations from the launcher pod.
func (c *nnfUserContainer) createMPIJob() error {

	if c.profile.Data.NnfMPISpec == nil {
		return fmt.Errorf("MPI spec is nil")
	}

	// Create a new MPIJob object
	cleanPodPolicy := mpiv2beta1.CleanPodPolicyRunning
	mpiJob := &mpiv2beta1.MPIJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.workflow.Name,
			Namespace: c.workflow.Namespace,
		},
		Spec: mpiv2beta1.MPIJobSpec{
			MPIReplicaSpecs: map[mpiv2beta1.MPIReplicaType]*mpicommonv1.ReplicaSpec{
				mpiv2beta1.MPIReplicaTypeLauncher: {},
				mpiv2beta1.MPIReplicaTypeWorker:   {},
			},
			RunPolicy: mpiv2beta1.RunPolicy{
				CleanPodPolicy: &cleanPodPolicy,            // Don't keep failed pods around
				BackoffLimit:   &c.profile.Data.RetryLimit, // Retry limit for the launcher
			},
			SlotsPerWorker: c.profile.Data.NnfMPISpec.SlotsPerWorker,
		},
	}
	launcher := mpiJob.Spec.MPIReplicaSpecs[mpiv2beta1.MPIReplicaTypeLauncher]
	worker := mpiJob.Spec.MPIReplicaSpecs[mpiv2beta1.MPIReplicaTypeWorker]

	// Copy the NNFContainerProfile spec into the MPIJob spec
	launcher.Template.Spec = *c.profile.Data.NnfMPISpec.Launcher.ToCorePodSpec()
	worker.Template.Spec = *c.profile.Data.NnfMPISpec.Worker.ToCorePodSpec()
	launcherSpec := &launcher.Template.Spec
	workerSpec := &worker.Template.Spec

	c.username = nnfv1alpha7.ContainerMPIUser

	if err := c.applyLabels(&mpiJob.ObjectMeta, true /* applyOwner */); err != nil {
		return err
	}

	// Add workflow labels to the launcher and worker pods themselves
	if err := c.applyLabels(&launcher.Template.ObjectMeta, false /* applyOwner */); err != nil {
		return err
	}
	if err := c.applyLabels(&worker.Template.ObjectMeta, false /* applyOwner */); err != nil {
		return err
	}

	// Keep failed pods around for log inspection
	launcher.RestartPolicy = mpicommonv1.RestartPolicyNever
	worker.RestartPolicy = mpicommonv1.RestartPolicyNever

	// Add NNF node tolerations
	c.applyTolerations(launcherSpec)
	c.applyTolerations(workerSpec)

	// Run the launcher on the first NNF node
	launcherSpec.NodeSelector = map[string]string{"kubernetes.io/hostname": c.nnfNodes[0]}

	// Target all the NNF nodes for the workers
	replicas := int32(len(c.nnfNodes))
	worker.Replicas = &replicas
	workerSpec.Affinity = &corev1.Affinity{
		// Ensure we run a worker on every NNF node
		NodeAffinity: &corev1.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
				NodeSelectorTerms: []corev1.NodeSelectorTerm{{
					MatchExpressions: []corev1.NodeSelectorRequirement{{
						Key:      "kubernetes.io/hostname",
						Operator: corev1.NodeSelectorOpIn,
						Values:   c.nnfNodes,
					}},
				}},
			},
		},
		// But make sure it's only 1 per node
		PodAntiAffinity: &corev1.PodAntiAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{{
				TopologyKey: "kubernetes.io/hostname",
				LabelSelector: &metav1.LabelSelector{
					MatchExpressions: []metav1.LabelSelectorRequirement{
						{
							Key:      "training.kubeflow.org/job-name",
							Operator: metav1.LabelSelectorOpIn,
							Values:   []string{c.workflow.Name},
						},
						{
							Key:      "training.kubeflow.org/job-role",
							Operator: metav1.LabelSelectorOpIn,
							Values:   []string{"worker"},
						},
					},
				}},
			},
		},
	}

	// Set the appropriate permissions (UID/GID) from the workflow
	c.applyPermissions(launcherSpec, &mpiJob.Spec, false)
	c.applyPermissions(workerSpec, &mpiJob.Spec, true)

	// Use an Init Container to test the waters for mpi - ensure it can contact the workers before
	// the launcher tries it. Since this runs as the UID/GID, this needs to happen after the
	// passwd Init Container.
	c.addInitContainerWorkerWait(launcherSpec, len(c.nnfNodes))

	// Get the ports from the port manager and map them to each container
	ports, err := c.getHostPorts()
	if err != nil {
		return err
	}
	portMap, err := mapPorts(launcherSpec, ports)
	if err != nil {
		return err
	}

	// Add host ports for only the launcher's containers
	if err := addHostPorts(launcherSpec, portMap); err != nil {
		return err
	}

	// Add env variables for those ports to both launcher and worker containers
	addPortsEnvVars(*c.workflow, launcherSpec, ports, portMap)
	addPortsEnvVars(*c.workflow, workerSpec, ports, portMap)

	c.addNnfVolumes(launcherSpec)
	c.addNnfVolumes(workerSpec)
	c.addEnvVars(*c.workflow, launcherSpec, true)
	c.addEnvVars(*c.workflow, workerSpec, true)

	// Any server that uses TLS/token when communicating with the compute node
	// will be in the launcher, so mount any secrets there.
	c.addSecrets(launcherSpec)

	// Copy offload containers need to use the copy offload service account
	if c.profile.Data.NnfMPISpec.CopyOffload {
		launcherSpec.ServiceAccountName = nnfv1alpha7.CopyOffloadServiceAccountName
	}

	err = c.client.Create(c.ctx, mpiJob)
	if err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return err
		}
	} else {
		c.log.Info("Created MPIJob", "name", mpiJob.Name, "namespace", mpiJob.Namespace)
	}

	return nil
}

// Non-MPI container workflow. In this model, a job is created for each NNF node which ensures
// that a pod is executed successfully (or the backOffLimit) is hit. Each container in this model
// runs the same image.
func (c *nnfUserContainer) createNonMPIJob() error {

	if c.profile.Data.NnfSpec == nil {
		return fmt.Errorf("NNF spec is nil")
	}

	// Use one job that we'll use as a base to create all jobs. Each NNF node will get its own job.
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: c.workflow.Namespace,
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: &c.profile.Data.RetryLimit,
		},
	}

	// Copy the NNFContainerProfile spec into the Job spec
	job.Spec.Template.Spec = *c.profile.Data.NnfSpec.ToCorePodSpec()
	podSpec := &job.Spec.Template.Spec

	if err := c.applyLabels(&job.ObjectMeta, true /* applyOwner */); err != nil {
		return err
	}

	// Use the same labels as the job for the pods
	job.Spec.Template.Labels = job.DeepCopy().Labels

	podSpec.RestartPolicy = corev1.RestartPolicyNever
	podSpec.Subdomain = c.workflow.Name // service name == workflow name

	// Get the ports from the port manager and map them to container
	ports, err := c.getHostPorts()
	if err != nil {
		return err
	}
	portMap, err := mapPorts(podSpec, ports)
	if err != nil {
		return err
	}

	if err := addHostPorts(podSpec, portMap); err != nil {
		return err
	}
	addPortsEnvVars(*c.workflow, podSpec, ports, portMap)

	c.applyTolerations(podSpec)
	c.applyPermissions(podSpec, nil, false)
	c.addNnfVolumes(podSpec)
	c.addSecrets(podSpec)
	c.addEnvVars(*c.workflow, podSpec, false)

	// Using the base job, create a job for each nnfNode. Only the name, hostname, and node selector is different for each node
	for _, nnfNode := range c.nnfNodes {
		job.ObjectMeta.Name = c.workflow.Name + "-" + nnfNode
		podSpec.Hostname = nnfNode

		// In our case, the target is only 1 node for the job, so a restartPolicy of Never
		// is ok because any retry (i.e. new pod) will land on the same node.
		podSpec.NodeSelector = map[string]string{"kubernetes.io/hostname": nnfNode}

		newJob := &batchv1.Job{}
		job.DeepCopyInto(newJob)

		err := c.client.Create(c.ctx, newJob)
		if err != nil {
			if !apierrors.IsAlreadyExists(err) {
				return err
			}
		} else {
			c.log.Info("Created non-MPI job", "name", newJob.Name, "namespace", newJob.Namespace)
		}
	}

	return nil
}

func (c *nnfUserContainer) applyLabels(obj metav1.Object, applyOwner bool) error {
	// Apply Job Labels/Owners
	dwsv1alpha4.InheritParentLabels(obj, c.workflow)
	dwsv1alpha4.AddWorkflowLabels(obj, c.workflow)
	if applyOwner {
		dwsv1alpha4.AddOwnerLabels(obj, c.workflow)
	}

	labels := obj.GetLabels()
	labels[nnfv1alpha7.ContainerLabel] = c.workflow.Name
	labels[nnfv1alpha7.PinnedContainerProfileLabelName] = c.profile.GetName()
	labels[nnfv1alpha7.PinnedContainerProfileLabelNameSpace] = c.profile.GetNamespace()
	labels[nnfv1alpha7.DirectiveIndexLabel] = strconv.Itoa(c.index)
	obj.SetLabels(labels)

	if applyOwner {
		if err := ctrl.SetControllerReference(c.workflow, obj, c.scheme); err != nil {
			return err
		}
	}

	return nil
}

func (c *nnfUserContainer) applyTolerations(spec *corev1.PodSpec) {
	spec.Tolerations = append(spec.Tolerations, corev1.Toleration{
		Effect:   corev1.TaintEffectNoSchedule,
		Key:      nnfv1alpha7.RabbitNodeTaintKey,
		Operator: corev1.TolerationOpEqual,
		Value:    "true",
	})
}

func (c *nnfUserContainer) addInitContainerPasswd(spec *corev1.PodSpec, image string) {
	// This script creates an entry in /etc/passwd to map the user to the given UID/GID using an
	// InitContainer. This is necessary for mpirun because it uses ssh to communicate with the
	// worker nodes. ssh itself requires that the UID is tied to a username in the container.
	// Since the launcher container is running as non-root, we need to make use of an InitContainer
	// to edit /etc/passwd and copy it to a volume which can then be mounted into the non-root
	// container to replace /etc/passwd.
	script := `# tie the UID/GID to the user
sed -i '/^$USER/d' /etc/passwd
echo "$USER:x:$UID:$GID::/home/$USER:/bin/sh" >> /etc/passwd
cp /etc/passwd /config/
exit 0
`
	// Replace the user and UID/GID
	script = strings.ReplaceAll(script, "$USER", c.username)
	script = strings.ReplaceAll(script, "$UID", fmt.Sprintf("%d", c.uid))
	script = strings.ReplaceAll(script, "$GID", fmt.Sprintf("%d", c.gid))

	spec.InitContainers = append(spec.InitContainers, corev1.Container{
		Name:  fmt.Sprintf("%s-init-passwd", c.username),
		Image: image,
		Command: []string{
			"/bin/sh",
			"-c",
			script,
		},
		VolumeMounts: []corev1.VolumeMount{
			{Name: "passwd", MountPath: "/config"},
		},
	})
}

func (c *nnfUserContainer) addInitContainerWorkerWait(spec *corev1.PodSpec, numWorkers int) {
	// Add an initContainer to ensure that the worker pods are up and discoverable via mpirun.
	script := `# use mpirun to contact workers
echo "contacting $HOSTS..."
for i in $(seq 1 100); do
   sleep 1
   echo "attempt $i of 100..."
   echo "mpirun -H $HOSTS hostname"
   mpirun -H $HOSTS hostname
   if [ $? -eq 0 ]; then
      echo "successfully contacted $HOSTS; done"
      exit 0
   fi
done
echo "failed to contact $HOSTS"
exit 1
`
	// Build a slice of the workers' hostname.domain (e.g. nnf-container-example-worker-0.nnf-container-example.default.svc)
	// This hostname comes from mpi-operator.
	workers := []string{}
	for i := 0; i < numWorkers; i++ {
		host := strings.ToLower(fmt.Sprintf(
			"%s-worker-%d.%s.%s.svc", c.workflow.Name, i, c.workflow.Name, c.workflow.Namespace))
		workers = append(workers, host)
	}
	// mpirun takes a comma separated list of hosts (-H)
	script = strings.ReplaceAll(script, "$HOSTS", strings.Join(workers, ","))

	spec.InitContainers = append(spec.InitContainers, corev1.Container{
		Name:  fmt.Sprintf("mpi-wait-for-%d-workers", numWorkers),
		Image: spec.Containers[0].Image,
		Command: []string{
			"/bin/sh",
			"-c",
			script,
		},
		// mpirun needs this environment variable to use DNS hostnames
		Env: []corev1.EnvVar{{Name: "OMPI_MCA_orte_keep_fqdn_hostnames", Value: "true"}},
		// Run this initContainer as the same UID/GID as the launcher
		SecurityContext: &corev1.SecurityContext{
			RunAsUser:    &c.uid,
			RunAsGroup:   &c.gid,
			RunAsNonRoot: pointy.Bool(true),
		},
		// And use the necessary volumes to support the UID/GID
		VolumeMounts: []corev1.VolumeMount{
			{MountPath: "/etc/passwd", Name: "passwd", SubPath: "passwd"},
			{MountPath: "/home/mpiuser/.ssh", Name: "ssh-auth"},
		},
	})
}

func (c *nnfUserContainer) applyPermissions(spec *corev1.PodSpec, mpiJobSpec *mpiv2beta1.MPIJobSpec, worker bool) {

	// Add volume for /etc/passwd to map user to UID/GID
	spec.Volumes = append(spec.Volumes, corev1.Volume{
		Name: "passwd",
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	})

	if !worker {
		// Add SecurityContext if necessary
		if spec.SecurityContext == nil {
			spec.SecurityContext = &corev1.PodSecurityContext{}
		}

		// Add spec level security context to apply FSGroup to all containers. This keeps the
		// volumes safe from root actions.
		spec.SecurityContext.FSGroup = &c.gid

		// Set the ssh key path for non-root users. Defaults to root.
		if mpiJobSpec != nil {
			mpiJobSpec.SSHAuthMountPath = fmt.Sprintf("/home/%s/.ssh", c.username)
		}
	}

	// Add user permissions to each container. This needs to be done for each container because
	// we do not want these permissions on the init container.
	for idx := range spec.Containers {
		container := &spec.Containers[idx]

		// Add an InitContainer to map the user to the provided uid/gid using /etc/passwd. This only
		// needs to happen once.
		if idx == 0 {
			c.addInitContainerPasswd(spec, container.Image)
		}

		// Add a mount to copy the modified /etc/passwd to
		container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{
			Name:      "passwd",
			MountPath: "/etc/passwd",
			SubPath:   "passwd",
		})

		// Create SecurityContext if necessary
		if container.SecurityContext == nil {
			container.SecurityContext = &corev1.SecurityContext{}
		}

		// Add non-root permissions from the workflow's user/group ID for the launcher, but not
		// the worker. The worker needs to run an ssh daemon, which requires root. Commands on
		// the worker are executed via the launcher as the `mpiuser` and not root.
		if !worker {
			container.SecurityContext.RunAsUser = &c.uid
			container.SecurityContext.RunAsGroup = &c.gid
			container.SecurityContext.RunAsNonRoot = pointy.Bool(true)
			container.SecurityContext.AllowPrivilegeEscalation = pointy.Bool(false)
		} else {
			// For the worker nodes, we need to ensure we have the appropriate linux capabilities to
			// allow for ssh access for mpirun. Drop all capabilities and only add what is
			// necessary. Only do this if the Capabilities have not been set by the user.
			container.SecurityContext.AllowPrivilegeEscalation = pointy.Bool(true)
			if container.SecurityContext.Capabilities == nil {
				container.SecurityContext.Capabilities = &corev1.Capabilities{
					Drop: []corev1.Capability{"ALL"},
					Add:  []corev1.Capability{"NET_BIND_SERVICE", "SYS_CHROOT", "AUDIT_WRITE", "SETUID", "SETGID"},
				}
			}
		}
	}
}

func (c *nnfUserContainer) getHostPorts() ([]uint16, error) {
	ports := []uint16{}

	// Each container gets NumPorts ports
	expectedPorts := countContainersInProfile(c.profile) * int(c.profile.Data.NumPorts)

	if expectedPorts <= 0 {
		return ports, nil
	}

	pm, err := getContainerPortManager(c.ctx, c.client)
	if err != nil {
		return nil, err
	}

	// Get the ports from the port manager for this workflow
	for _, alloc := range pm.Status.Allocations {
		if alloc.Requester != nil && alloc.Requester.UID == c.workflow.UID && alloc.Status == nnfv1alpha7.NnfPortManagerAllocationStatusInUse {
			ports = append(ports, alloc.Ports...)
		}
	}

	// Make sure we found the number of ports in the port manager that we expect
	if len(ports) != expectedPorts {
		return nil, dwsv1alpha4.NewResourceError(
			"number of ports found in NnfPortManager's allocation (%d) does not equal the profile's requested ports (%d)",
			len(ports), expectedPorts).
			WithUserMessage("requested ports do not meet the number of allocated ports").WithFatal()
	}

	return ports, nil
}

// mapPorts distributes a given list of ports across all containers in the PodSpec. Each container
// is assigned an equal number of ports from the list. If the number of ports is not evenly
// divisible by the number of containers, an error is returned. The keys in the map are the
// container names.
func mapPorts(spec *corev1.PodSpec, ports []uint16) (map[string][]uint16, error) {
	portMap := make(map[string][]uint16)

	if len(ports) == 0 {
		return portMap, nil
	}

	numContainers := len(spec.Containers)
	if numContainers == 0 {
		return nil, fmt.Errorf("no containers found in the PodSpec")
	}
	if len(ports)%numContainers != 0 {
		return nil, fmt.Errorf("number of ports (%d) must be a multiple of the number of containers (%d)", len(ports), numContainers)
	}

	// Distribute ports to each container
	portsPerContainer := len(ports) / numContainers
	for idx, container := range spec.Containers {
		// Assign the next slice of ports to the current container
		startIdx := idx * portsPerContainer
		endIdx := startIdx + portsPerContainer
		portMap[container.Name] = make([]uint16, portsPerContainer)

		for i, port := range ports[startIdx:endIdx] {
			portMap[container.Name][i] = port
		}
	}

	return portMap, nil
}

// addHostPorts adds HostPort entries to all containers in a PodSpec using map of container names to
// a list of ports.
func addHostPorts(spec *corev1.PodSpec, portMap map[string][]uint16) error {

	// Add the ports to the containers based on the portMap
	for idx := range spec.Containers {
		container := &spec.Containers[idx]

		// Get the ports assigned to this container from the portMap
		containerPorts, exists := portMap[container.Name]
		if !exists {
			continue // Skip if no ports are assigned to this container
		}

		// Assign the ports to the container
		for _, port := range containerPorts {
			container.Ports = append(container.Ports, corev1.ContainerPort{
				ContainerPort: int32(port),
				HostPort:      int32(port),
			})
		}
	}

	return nil
}

// Given a list of ports, convert it into an environment variable name and comma separated value
func getContainerPortsList(ports []uint16) string {
	portStr := []string{}
	for _, port := range ports {
		portStr = append(portStr, strconv.Itoa(int(port)))
	}

	return strings.Join(portStr, ",")
}

// Add environment variables for the container ports to all containers in a PodSpec.
// Ex:
// NNF_CONTAINER_PORTS - all the ports assigned to all containers
// NNF_CONTAINER_PORTS_my_container1 - ports assigned to container named my_container1
// NNF_CONTAINER_PORTS_my_container2 - ports assigned to container named my_container2
func addPortsEnvVars(workflow dwsv1alpha4.Workflow, spec *corev1.PodSpec, ports []uint16, portMap map[string][]uint16) {
	if len(ports) < 1 {
		return
	}

	// Add port environment variable to containers
	for idx := range spec.Containers {
		container := &spec.Containers[idx]

		// Add env var for all the ports
		env := corev1.EnvVar{
			Name:  "NNF_CONTAINER_PORTS",
			Value: getContainerPortsList(ports),
		}
		container.Env = append(container.Env, env)
		workflow.Status.Env[env.Name] = env.Value

		// Add env var for each container name
		for containerName, containerPorts := range portMap {
			env := corev1.EnvVar{
				Name:  "NNF_CONTAINER_PORTS_" + strings.Replace(containerName, "-", "_", -1),
				Value: getContainerPortsList(containerPorts),
			}
			container.Env = append(container.Env, env)
			workflow.Status.Env[env.Name] = env.Value
		}
	}
}

// Look in the PodSpec and count the number of containers. For MPI containers, only count Launcher
// containers
func countContainersInProfile(profile *nnfv1alpha7.NnfContainerProfile) int {
	if profile.Data.NnfMPISpec != nil {
		return len(profile.Data.NnfMPISpec.Launcher.Containers)
	} else if profile.Data.NnfSpec != nil {
		return len(profile.Data.NnfSpec.Containers)
	}

	return 0
}

func (c *nnfUserContainer) addNnfVolumes(spec *corev1.PodSpec) {
	for _, vol := range c.volumes {

		var volSource corev1.VolumeSource

		// If global lustre, use a PVC, otherwise use a HostPath on the rabbit to the mounts that
		// already exist.
		if vol.command == "globaldw" {
			volSource = corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: vol.pvcName,
				},
			}
		} else {
			hostPathType := corev1.HostPathDirectory
			volSource = corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: vol.mountPath,
					Type: &hostPathType,
				},
			}
		}
		spec.Volumes = append(spec.Volumes, corev1.Volume{Name: vol.name, VolumeSource: volSource})

		// Add VolumeMounts and Volume environment variables for all containers
		for idx := range spec.Containers {
			container := &spec.Containers[idx]

			container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{
				Name:      vol.name,
				MountPath: vol.mountPath,
			})

			if vol.envVarName != "" {
				container.Env = append(container.Env, corev1.EnvVar{
					Name:  vol.envVarName,
					Value: vol.mountPath,
				})
			}
		}
	}
}

func (c *nnfUserContainer) addSecrets(spec *corev1.PodSpec) {
	for _, secretSpec := range c.secrets {
		volSource := corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: secretSpec.secretName,
			},
		}
		spec.Volumes = append(spec.Volumes, corev1.Volume{Name: secretSpec.name, VolumeSource: volSource})

		// Add VolumeMounts and Volume environment variables for all containers
		for idx := range spec.Containers {
			container := &spec.Containers[idx]

			container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{
				Name:      secretSpec.name,
				MountPath: secretSpec.mountPath,
				ReadOnly:  true,
			})

			for key, value := range secretSpec.envVarsToFileNames {
				container.Env = append(container.Env, corev1.EnvVar{
					Name:  key,
					Value: fmt.Sprintf("%s/%s", secretSpec.mountPath, value),
				})
			}
		}
	}
}

func (c *nnfUserContainer) addEnvVars(workflow dwsv1alpha4.Workflow, spec *corev1.PodSpec, mpi bool) {
	// Add in non-volume environment variables for all containers
	for idx := range spec.Containers {
		container := &spec.Containers[idx]

		// Jobs/hostnames are named differently based on mpi or not. For MPI, there are
		// launcher/worker pods. For non-MPI, the jobs are named after the rabbit node.
		subdomain := c.workflow.Name // service name == workflow name
		domain := c.workflow.Namespace + ".svc.cluster.local"
		hosts := []string{}

		if mpi {
			launcher := c.workflow.Name + "-launcher"
			worker := c.workflow.Name + "-worker"

			hosts = append(hosts, launcher)
			for i := range c.nnfNodes {
				hosts = append(hosts, fmt.Sprintf("%s-%d", worker, i))
			}
		} else {
			hosts = append(hosts, c.nnfNodes...)
		}

		// Add env variables to workflow
		workflow.Status.Env["NNF_CONTAINER_SUBDOMAIN"] = subdomain
		workflow.Status.Env["NNF_CONTAINER_DOMAIN"] = domain
		workflow.Status.Env["NNF_CONTAINER_HOSTNAMES"] = strings.Join(hosts, ",")

		// Add env variables to container
		container.Env = append(container.Env,
			corev1.EnvVar{Name: "NNF_CONTAINER_SUBDOMAIN", Value: subdomain},
			corev1.EnvVar{Name: "NNF_CONTAINER_DOMAIN", Value: domain},
			corev1.EnvVar{Name: "NNF_CONTAINER_HOSTNAMES", Value: strings.Join(hosts, ",")},
			corev1.EnvVar{Name: "DW_WORKFLOW_NAME", Value: workflow.Name},
			corev1.EnvVar{Name: "DW_WORKFLOW_NAMESPACE", Value: workflow.Namespace},
			corev1.EnvVar{Name: "ENVIRONMENT", Value: os.Getenv("ENVIRONMENT")},
			corev1.EnvVar{
				Name: "NNF_NODE_NAME",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						APIVersion: "v1",
						FieldPath:  "spec.nodeName",
					},
				},
			})
	}
}

func verifyUserContainerTLSSecretName(clnt client.Client, ctx context.Context) error {
	// Check if the user container TLS secret exists.
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      userContainerTLSSecretName,
			Namespace: userContainerTLSSecretNamespace,
		},
	}
	if err := clnt.Get(ctx, client.ObjectKeyFromObject(secret), secret); err != nil {
		return dwsv1alpha4.NewResourceError("the administrator must configure the user container TLS secret for the system. See the copy-offload docs").WithError(err).WithFatal()
	}
	return nil
}

func getUserContainerCACert(clnt client.Client, ctx context.Context) ([]byte, error) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      userContainerTLSSecretName,
			Namespace: userContainerTLSSecretNamespace,
		},
	}
	if err := clnt.Get(ctx, client.ObjectKeyFromObject(secret), secret); err != nil {
		return nil, fmt.Errorf("failed to get secret %s: %w", secret.Name, err)
	}
	key := "tls.crt"
	caCert, ok := secret.Data[key]
	if !ok {
		return nil, fmt.Errorf("'%s 'not found in secret %s", key, secret.Name)
	}
	return caCert, nil
}

func (r *NnfWorkflowReconciler) setupContainerAuth(ctx context.Context, workflow *dwsv1alpha4.Workflow, log logr.Logger) (*dwsv1alpha4.WorkflowTokenSecret, error) {
	privKey, err := r.createContainerTokenKey(ctx, workflow, log)
	if err != nil {
		return nil, err
	}
	workflowToken, err := r.createContainerToken(ctx, workflow, privKey, log)
	if err != nil {
		return nil, err
	}
	return workflowToken, nil
}

func makeWorkflowTokenName(workflow *dwsv1alpha4.Workflow) (*dwsv1alpha4.WorkflowTokenSecret, string) {
	workflowToken := workflow.Status.WorkflowToken
	if workflowToken == nil {
		workflowToken = &dwsv1alpha4.WorkflowTokenSecret{
			SecretName:      workflow.GetName() + "-token",
			SecretNamespace: workflow.GetNamespace(),
		}
	}
	serversSecretName := workflowToken.SecretName + "-server"
	return workflowToken, serversSecretName
}

func (r *NnfWorkflowReconciler) createContainerTokenKey(ctx context.Context, workflow *dwsv1alpha4.Workflow, log logr.Logger) ([]byte, error) {
	immutable := true
	workflowToken, serversSecretName := makeWorkflowTokenName(workflow)
	tokenKeySecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serversSecretName,
			Namespace: workflowToken.SecretNamespace,
		},
		Type:      corev1.SecretTypeOpaque,
		Immutable: &immutable,
		Data:      make(map[string][]byte),
	}
	errGroup := fmt.Sprintf("could not create secret: %v", client.ObjectKeyFromObject(tokenKeySecret))
	userMessageError := "could not create workflow's server-token secret"

	keyBytes, pemKey, err := nnftoken.CreateKeyForTokens()
	if err != nil {
		return []byte(""), dwsv1alpha4.NewResourceError("%s: %s", errGroup, "CreateKeyForTokens").WithError(err).WithUserMessage("%s", userMessageError)
	}
	tokenKeySecret.Data["token.key"] = pemKey
	dwsv1alpha4.AddWorkflowLabels(tokenKeySecret, workflow)
	dwsv1alpha4.AddOwnerLabels(tokenKeySecret, workflow)
	if err := ctrl.SetControllerReference(workflow, tokenKeySecret, r.Scheme); err != nil {
		return []byte(""), dwsv1alpha4.NewResourceError("%s: %s", errGroup, "SetControllerReference").WithError(err).WithUserMessage("%s", userMessageError)
	}

	if err := r.Create(ctx, tokenKeySecret); err != nil {
		if apierrors.IsAlreadyExists(err) {
			// Get the key from it so we can use it to create a token.
			prevTokenKeySecret := &corev1.Secret{}
			if err = r.Get(ctx, client.ObjectKeyFromObject(tokenKeySecret), prevTokenKeySecret); err != nil {
				return []byte(""), dwsv1alpha4.NewResourceError("%s: %s", errGroup, "Get").WithError(err).WithUserMessage("%s", userMessageError)
			}
			pemKey = prevTokenKeySecret.Data["token.key"]
			if keyBytes, err = nnftoken.GetKeyFromPEM(pemKey); err != nil {
				return []byte(""), dwsv1alpha4.NewResourceError("%s: %s", errGroup, "GetKeyFromPEM").WithError(err).WithUserMessage("%s", userMessageError)
			}
			log.Info("using existing key", "secret", client.ObjectKeyFromObject(prevTokenKeySecret))
		} else {
			return []byte(""), dwsv1alpha4.NewResourceError("%s: %s", errGroup, "Create").WithError(err).WithUserMessage("%s", userMessageError)
		}
	} else {
		log.Info("created key", "secret", client.ObjectKeyFromObject(tokenKeySecret))
	}
	return keyBytes, nil
}

func (r *NnfWorkflowReconciler) createContainerToken(ctx context.Context, workflow *dwsv1alpha4.Workflow, keyBytes []byte, log logr.Logger) (*dwsv1alpha4.WorkflowTokenSecret, error) {
	immutable := true
	workflowToken, _ := makeWorkflowTokenName(workflow)
	tokenSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      workflowToken.SecretName,
			Namespace: workflowToken.SecretNamespace,
		},
		Type:       corev1.SecretTypeOpaque,
		Immutable:  &immutable,
		StringData: make(map[string]string),
	}
	errGroup := fmt.Sprintf("could not create secret: %v", client.ObjectKeyFromObject(tokenSecret))
	userMessageError := "could not create or update workflow's client-token secret"

	tokenString, err := nnftoken.CreateTokenFromKey(keyBytes, "user-container")
	if err != nil {
		return nil, dwsv1alpha4.NewResourceError("%s: %s", errGroup, "CreateTokenFromKey").WithError(err).WithUserMessage("%s", userMessageError)
	}
	tokenSecret.StringData["token"] = tokenString
	dwsv1alpha4.AddWorkflowLabels(tokenSecret, workflow)
	dwsv1alpha4.AddOwnerLabels(tokenSecret, workflow)
	if err := ctrl.SetControllerReference(workflow, tokenSecret, r.Scheme); err != nil {
		return nil, dwsv1alpha4.NewResourceError("%s: %s", errGroup, "SetControllerReference").WithError(err).WithUserMessage("%s", userMessageError)
	}

	if err := r.Create(ctx, tokenSecret); err != nil {
		if apierrors.IsAlreadyExists(err) {
			// Get the token to verify it against our key.
			// In this case, the user container may already be running and using
			// the already-existing token.
			// If the token doesn't verify with the key then we cannot create
			// a new token&secret; we have to bail.
			prevTokenSecret := &corev1.Secret{}
			if err = r.Get(ctx, client.ObjectKeyFromObject(tokenSecret), prevTokenSecret); err != nil {
				return nil, dwsv1alpha4.NewResourceError("%s: %s", errGroup, "Get").WithError(err).WithUserMessage("%s", userMessageError)
			}
			// Note: we read from Secret.Data, rather than Secret.StringData,
			// per the instructions in corev1.Secret.
			tokenStringBytes := prevTokenSecret.Data["token"]
			if err = nnftoken.VerifyToken(string(tokenStringBytes), keyBytes); err != nil {
				return nil, dwsv1alpha4.NewResourceError("%s: %s", errGroup, "VerifyToken").WithError(err).WithUserMessage("%s", userMessageError)
			}
		} else {
			return nil, dwsv1alpha4.NewResourceError("%s: %s", errGroup, "Create").WithError(err).WithUserMessage("%s", userMessageError)
		}
	} else {
		log.Info("created token", "secret", client.ObjectKeyFromObject(tokenSecret))
	}
	return workflowToken, nil
}

// sendContainerShutdown sends a shutdown request to the user container. This is done by sending a
// POST request to the `/shutdown` endpoint of the user container. This might fail if the user
// container does not implement the `/shutdown` endpoint, but we still want to try to send the
// request.
func (r *NnfWorkflowReconciler) sendContainerShutdown(ctx context.Context, workflow *dwsv1alpha4.Workflow, profile *nnfv1alpha7.NnfContainerProfile) error {
	isMPIJob := profile.Data.NnfMPISpec != nil

	hosts := []string{}
	ports := strings.Split(workflow.Status.Env["NNF_CONTAINER_PORTS"], ",")

	// Build the list of hosts to send the shutdown to based on the job type
	if isMPIJob {
		launcher := workflow.Status.Env["NNF_CONTAINER_LAUNCHER"]
		for _, port := range ports {
			hosts = append(hosts, fmt.Sprintf("%s:%s", launcher, port))
		}
	} else {
		// Get the targeted NNF nodes for the container jobs
		nnfNodes, err := r.getNnfNodesFromComputes(ctx, workflow)
		if err != nil || len(nnfNodes) <= 0 {
			return dwsv1alpha4.NewResourceError("error obtaining the target NNF nodes for containers").WithError(err)
		}
		for _, node := range nnfNodes {
			for _, port := range ports {
				hosts = append(hosts, fmt.Sprintf("%s:%s", node, port))
			}
		}
	}

	if len(hosts) > 0 {
		token, err := r.getWorkflowToken(ctx, workflow)
		if workflow.Status.WorkflowToken != nil && err != nil {
			return dwsv1alpha4.NewResourceError("could not get workflow token string").WithError(err)
		}

		// Send the requiest, but do not return an error if it fails. It's possible that the
		// container doesn't have this implemented, but we need to try to send the requests anyway.
		// The PostRunTimeout will eventually kill the container if it doesn't shut down gracefully.
		if err := r.sendShutdownToHosts(ctx, hosts, string(token)); err != nil {
			// Log the error but do not return it, as we want to continue with the workflow.
			r.Log.Error(err, "could not send shutdown to user container hosts", "hosts", hosts)
		}
	}

	return nil
}

// sendShutdownToHosts sends a shutdown request to each host in the provided list. This expects the
// user container to have implemented a `/shutdown` endpoint that accepts a POST request. If we get
// a connection refused error, we treat it as a success because the user container will no longer be
// running after the shutdown.
func (r *NnfWorkflowReconciler) sendShutdownToHosts(ctx context.Context, hosts []string, token string) error {

	failedRequests := map[string]int{}

	// Retrieve the CA cert for the user container and create a cert pool for the HTTP client.
	// This should be present whether the container is using tokens or not.
	caCert, err := getUserContainerCACert(r.Client, ctx)
	if err != nil {
		return err
	}
	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		return fmt.Errorf("failed to append CA cert")
	}

	// Create an HTTP client with the CA cert pool and a short timeout
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: caCertPool,
			},
		},
		Timeout: 1 * time.Second,
	}

	// Send a shutdown request to each host. The request is a POST to the /shutdown endpoint. If it fails,
	// just log the error and continue to the next host.
	for _, host := range hosts {
		url := "https://" + host + "/shutdown"
		body := []byte(`{"message": "shutdown"}`)

		req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")

		// This is required for copy offload containers
		req.Header.Set("Accepts-version", copyOffloadAPIVersion)

		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("X-Auth-Type", "XOAUTH2")
		}

		resp, err := client.Do(req)
		if err != nil {
			// Once the shutdown completes, the user container will no longer be running, so we can ignore
			// ECONNREFUSED errors
			var opErr *net.OpError
			if errors.As(err, &opErr) {
				if sysErr, ok := opErr.Err.(*os.SyscallError); ok && sysErr.Err == syscall.ECONNREFUSED {
					// Treat as success, do not add to failedRequests
					continue
				}
			}
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			failedRequests[host] = resp.StatusCode
		}
	}

	// If there are any failed requests, return an error with the details of the failed hosts
	if len(failedRequests) > 0 {
		failedRequestsStr := ""
		for host, status := range failedRequests {
			failedRequestsStr += fmt.Sprintf("%s: %d, ", host, status)
		}
		return fmt.Errorf("failed to send shutdown to some hosts: %s", failedRequestsStr)
	}

	return nil
}
