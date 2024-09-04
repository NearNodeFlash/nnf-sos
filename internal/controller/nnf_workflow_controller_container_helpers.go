/*
 * Copyright 2023-2024 Hewlett Packard Enterprise Development LP
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
	"strconv"
	"strings"

	dwsv1alpha2 "github.com/DataWorkflowServices/dws/api/v1alpha2"
	nnfv1alpha1 "github.com/NearNodeFlash/nnf-sos/api/v1alpha1"
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
	workflow *dwsv1alpha2.Workflow
	profile  *nnfv1alpha1.NnfContainerProfile
	nnfNodes []string
	volumes  []nnfContainerVolume
	username string
	uid, gid int64
	client   client.Client
	log      logr.Logger
	scheme   *runtime.Scheme
	ctx      context.Context
	index    int
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

// MPI container workflow. In this model, we use mpi-operator to create an MPIJob, which creates
// a job for the launcher (to run mpirun) and a replicaset for the worker pods. The worker nodes
// run an ssh server tn listen for mpirun operations from the launcher pod.
func (c *nnfUserContainer) createMPIJob() error {
	mpiJob := &mpiv2beta1.MPIJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.workflow.Name,
			Namespace: c.workflow.Namespace,
		},
	}

	c.profile.Data.MPISpec.DeepCopyInto(&mpiJob.Spec)
	c.username = nnfv1alpha1.ContainerMPIUser

	if err := c.applyLabels(&mpiJob.ObjectMeta); err != nil {
		return err
	}

	// Use the profile's backoff limit if not set
	if mpiJob.Spec.RunPolicy.BackoffLimit == nil {
		mpiJob.Spec.RunPolicy.BackoffLimit = &c.profile.Data.RetryLimit
	}

	// MPIJobs have two pod specs: one for the launcher and one for the workers. The webhook ensures
	// that the launcher/worker specs exist
	launcher := mpiJob.Spec.MPIReplicaSpecs[mpiv2beta1.MPIReplicaTypeLauncher]
	launcherSpec := &launcher.Template.Spec
	worker := mpiJob.Spec.MPIReplicaSpecs[mpiv2beta1.MPIReplicaTypeWorker]
	workerSpec := &worker.Template.Spec

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

	// Get the ports from the port manager
	ports, err := c.getHostPorts()
	if err != nil {
		return err
	}
	// Add the ports to the worker spec and add environment variable for both launcher/worker
	addHostPorts(workerSpec, ports)
	addPortsEnvVars(launcherSpec, ports)
	addPortsEnvVars(workerSpec, ports)

	c.addNnfVolumes(launcherSpec)
	c.addNnfVolumes(workerSpec)
	c.addEnvVars(launcherSpec, true)
	c.addEnvVars(workerSpec, true)

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
	// Use one job that we'll use as a base to create all jobs. Each NNF node will get its own job.
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: c.workflow.Namespace,
		},
	}
	c.profile.Data.Spec.DeepCopyInto(&job.Spec.Template.Spec)
	podSpec := &job.Spec.Template.Spec

	if err := c.applyLabels(&job.ObjectMeta); err != nil {
		return err
	}

	// Use the same labels as the job for the pods
	job.Spec.Template.Labels = job.DeepCopy().Labels

	job.Spec.BackoffLimit = &c.profile.Data.RetryLimit

	podSpec.RestartPolicy = corev1.RestartPolicyNever
	podSpec.Subdomain = c.workflow.Name // service name == workflow name

	// Get the ports from the port manager
	ports, err := c.getHostPorts()
	if err != nil {
		return err
	}
	addHostPorts(podSpec, ports)
	addPortsEnvVars(podSpec, ports)

	c.applyTolerations(podSpec)
	c.applyPermissions(podSpec, nil, false)
	c.addNnfVolumes(podSpec)
	c.addEnvVars(podSpec, false)

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

func (c *nnfUserContainer) applyLabels(job metav1.Object) error {
	// Apply Job Labels/Owners
	dwsv1alpha2.InheritParentLabels(job, c.workflow)
	dwsv1alpha2.AddOwnerLabels(job, c.workflow)
	dwsv1alpha2.AddWorkflowLabels(job, c.workflow)

	labels := job.GetLabels()
	labels[nnfv1alpha1.ContainerLabel] = c.workflow.Name
	labels[nnfv1alpha1.PinnedContainerProfileLabelName] = c.profile.GetName()
	labels[nnfv1alpha1.PinnedContainerProfileLabelNameSpace] = c.profile.GetNamespace()
	labels[nnfv1alpha1.DirectiveIndexLabel] = strconv.Itoa(c.index)
	job.SetLabels(labels)

	if err := ctrl.SetControllerReference(c.workflow, job, c.scheme); err != nil {
		return err
	}

	return nil
}

func (c *nnfUserContainer) applyTolerations(spec *corev1.PodSpec) {
	spec.Tolerations = append(spec.Tolerations, corev1.Toleration{
		Effect:   corev1.TaintEffectNoSchedule,
		Key:      nnfv1alpha1.RabbitNodeTaintKey,
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
		Name:  "mpi-init-passwd",
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
		Name:  fmt.Sprintf("mpi-wait-for-worker-%d", numWorkers),
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

		// Add an InitContainer to map the user to the provided uid/gid using /etc/passwd
		c.addInitContainerPasswd(spec, container.Image)

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
	expectedPorts := int(c.profile.Data.NumPorts)

	if expectedPorts < 1 {
		return ports, nil
	}

	pm, err := getContainerPortManager(c.ctx, c.client)
	if err != nil {
		return nil, err
	}

	// Get the ports from the port manager for this workflow
	for _, alloc := range pm.Status.Allocations {
		if alloc.Requester != nil && alloc.Requester.UID == c.workflow.UID && alloc.Status == nnfv1alpha1.NnfPortManagerAllocationStatusInUse {
			ports = append(ports, alloc.Ports...)
		}
	}

	// Make sure we found the number of ports in the port manager that we expect
	if len(ports) != expectedPorts {
		return nil, dwsv1alpha2.NewResourceError(
			"number of ports found in NnfPortManager's allocation (%d) does not equal the profile's requested ports (%d)",
			len(ports), expectedPorts).
			WithUserMessage("requested ports do not meet the number of allocated ports").WithFatal()
	}

	return ports, nil
}

// Given a list of ports, add HostPort entries for all containers in a PodSpec
func addHostPorts(spec *corev1.PodSpec, ports []uint16) {

	// Nothing to add
	if len(ports) < 1 {
		return
	}

	// Add the ports to the containers
	for idx := range spec.Containers {
		container := &spec.Containers[idx]

		for _, port := range ports {
			container.Ports = append(container.Ports, corev1.ContainerPort{
				ContainerPort: int32(port),
				HostPort:      int32(port),
			})
		}
	}
}

// Given a list of ports, convert it into an environment variable name and comma separated value
func getContainerPortsEnvVar(ports []uint16) (string, string) {
	portStr := []string{}
	for _, port := range ports {
		portStr = append(portStr, strconv.Itoa(int(port)))
	}

	return "NNF_CONTAINER_PORTS", strings.Join(portStr, ",")
}

// Add a environment variable for the container ports to all containers in a PodSpec
func addPortsEnvVars(spec *corev1.PodSpec, ports []uint16) {
	if len(ports) < 1 {
		return
	}

	// Add port environment variable to containers
	for idx := range spec.Containers {
		container := &spec.Containers[idx]

		name, val := getContainerPortsEnvVar(ports)
		container.Env = append(container.Env, corev1.EnvVar{
			Name:  name,
			Value: val,
		})
	}
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

			container.Env = append(container.Env, corev1.EnvVar{
				Name:  vol.envVarName,
				Value: vol.mountPath,
			})
		}
	}
}

func (c *nnfUserContainer) addEnvVars(spec *corev1.PodSpec, mpi bool) {
	// Add in non-volume environment variables for all containers
	for idx := range spec.Containers {
		container := &spec.Containers[idx]

		// Jobs/hostnames and services/subdomains are named differently based on mpi or not. For
		// MPI, there are launcher/worker pods and the service is named after the worker. For
		// non-MPI, the jobs are named after the rabbit node.
		subdomain := ""
		domain := c.workflow.Namespace + ".svc.cluster.local"
		hosts := []string{}

		if mpi {
			launcher := c.workflow.Name + "-launcher"
			worker := c.workflow.Name + "-worker"
			subdomain = worker

			hosts = append(hosts, launcher)
			for i, _ := range c.nnfNodes {
				hosts = append(hosts, fmt.Sprintf("%s-%d", worker, i))
			}
		} else {
			subdomain = spec.Subdomain
			hosts = append(hosts, c.nnfNodes...)
		}

		container.Env = append(container.Env,
			corev1.EnvVar{Name: "NNF_CONTAINER_SUBDOMAIN", Value: subdomain},
			corev1.EnvVar{Name: "NNF_CONTAINER_DOMAIN", Value: domain},
			corev1.EnvVar{Name: "NNF_CONTAINER_HOSTNAMES", Value: strings.Join(hosts, " ")})
	}
}
