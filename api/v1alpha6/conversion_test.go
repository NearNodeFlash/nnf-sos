/*
 * Copyright 2025 Hewlett Packard Enterprise Development LP
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

package v1alpha6

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	fuzz "github.com/google/gofuzz"
	mpicommon "github.com/kubeflow/common/pkg/apis/common/v1"
	mpiv2beta1 "github.com/kubeflow/mpi-operator/pkg/apis/kubeflow/v2beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/apitesting/fuzzer"
	runtimeserializer "k8s.io/apimachinery/pkg/runtime/serializer"

	. "github.com/onsi/ginkgo/v2"

	nnfv1alpha7 "github.com/NearNodeFlash/nnf-sos/api/v1alpha7"
	utilconversion "github.com/NearNodeFlash/nnf-sos/github/cluster-api/util/conversion"
)

func TestFuzzyConversion(t *testing.T) {

	t.Run("for NnfAccess", utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Hub:   &nnfv1alpha7.NnfAccess{},
		Spoke: &NnfAccess{},
	}))

	t.Run("for NnfContainerProfile", utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Hub:         &nnfv1alpha7.NnfContainerProfile{},
		Spoke:       &NnfContainerProfile{},
		FuzzerFuncs: []fuzzer.FuzzerFuncs{NnfContainerProfileFuzzFunc},
	}))

	t.Run("for NnfDataMovement", utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Hub:   &nnfv1alpha7.NnfDataMovement{},
		Spoke: &NnfDataMovement{},
	}))

	t.Run("for NnfDataMovementManager", utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Hub:         &nnfv1alpha7.NnfDataMovementManager{},
		Spoke:       &NnfDataMovementManager{},
		FuzzerFuncs: []fuzzer.FuzzerFuncs{NnfDataMovementManagerFuzzFunc},
	}))

	t.Run("for NnfDataMovementProfile", utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Hub:   &nnfv1alpha7.NnfDataMovementProfile{},
		Spoke: &NnfDataMovementProfile{},
	}))

	t.Run("for NnfLustreMGT", utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Hub:   &nnfv1alpha7.NnfLustreMGT{},
		Spoke: &NnfLustreMGT{},
	}))

	t.Run("for NnfNode", utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Hub:   &nnfv1alpha7.NnfNode{},
		Spoke: &NnfNode{},
	}))

	t.Run("for NnfNodeBlockStorage", utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Hub:   &nnfv1alpha7.NnfNodeBlockStorage{},
		Spoke: &NnfNodeBlockStorage{},
	}))

	t.Run("for NnfNodeECData", utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Hub:   &nnfv1alpha7.NnfNodeECData{},
		Spoke: &NnfNodeECData{},
	}))

	t.Run("for NnfNodeStorage", utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Hub:   &nnfv1alpha7.NnfNodeStorage{},
		Spoke: &NnfNodeStorage{},
	}))

	t.Run("for NnfPortManager", utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Hub:   &nnfv1alpha7.NnfPortManager{},
		Spoke: &NnfPortManager{},
	}))

	t.Run("for NnfStorage", utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Hub:   &nnfv1alpha7.NnfStorage{},
		Spoke: &NnfStorage{},
	}))

	t.Run("for NnfStorageProfile", utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Hub:   &nnfv1alpha7.NnfStorageProfile{},
		Spoke: &NnfStorageProfile{},
	}))

	t.Run("for NnfSystemStorage", utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Hub:   &nnfv1alpha7.NnfSystemStorage{},
		Spoke: &NnfSystemStorage{},
	}))

}

// noFuzzPodSpec sets all the fields in a PodSpec that are not used in v1alpha7 and later to nil or default values.
func noFuzzPodSpec(spec *corev1.PodSpec) {
	// We use:
	// - Containers
	// - InitContainers
	// - Volumes
	// - TerminationGracePeriodSeconds
	// - AutomountServiceAccountToken
	// - ShareProcessNamespace
	// - ImagePullSecrets

	// Don't fuzz anything else:
	spec.EphemeralContainers = nil
	spec.ActiveDeadlineSeconds = nil
	spec.Affinity = nil
	spec.RestartPolicy = ""
	spec.DNSPolicy = ""
	spec.NodeSelector = nil
	spec.ServiceAccountName = ""
	spec.NodeName = ""
	spec.HostNetwork = false
	spec.HostPID = false
	spec.HostIPC = false
	spec.SecurityContext = nil
	spec.Hostname = ""
	spec.Subdomain = ""
	spec.Affinity = nil
	spec.SchedulerName = ""
	spec.Tolerations = nil
	spec.HostAliases = nil
	spec.PriorityClassName = ""
	spec.Priority = nil
	spec.DNSConfig = nil
	spec.ReadinessGates = nil
	spec.RuntimeClassName = nil
	spec.EnableServiceLinks = nil
	spec.PreemptionPolicy = nil
	spec.Overhead = nil
	spec.TopologySpreadConstraints = nil
	spec.DeprecatedServiceAccount = ""
	spec.SetHostnameAsFQDN = nil
	spec.OS = nil
	spec.SchedulingGates = nil
	spec.HostUsers = nil
	spec.ResourceClaims = nil

	noFuzzContainer(spec.Containers)
	noFuzzContainer(spec.InitContainers)

}

// noFuzzMPIPodSpec sets all the fields in a MPIJobSpec that are not used in v1alpha7 and later to nil or default values.
func noFuzzMPIPodSpec(mpiSpec *mpiv2beta1.MPIJobSpec) {

	// Similar to a PodSpec, all these fields are not used in v1alpha7 and later.
	mpiSpec.RunPolicy = mpiv2beta1.RunPolicy{}
	mpiSpec.SSHAuthMountPath = ""
	mpiSpec.MPIImplementation = ""
	mpiSpec.MPIReplicaSpecs = make(map[mpiv2beta1.MPIReplicaType]*mpicommon.ReplicaSpec)
	mpiSpec.MPIReplicaSpecs[mpiv2beta1.MPIReplicaTypeWorker] = &mpicommon.ReplicaSpec{}
	mpiSpec.MPIReplicaSpecs[mpiv2beta1.MPIReplicaTypeLauncher] = &mpicommon.ReplicaSpec{}
	for _, spec := range mpiSpec.MPIReplicaSpecs {
		spec.Template.Spec.ImagePullSecrets = nil
		spec.Template.Spec.Hostname = ""
		spec.Template.Spec.Subdomain = ""
		spec.Template.Spec.Affinity = nil
		spec.Template.Spec.SchedulerName = ""
		spec.Template.Spec.Tolerations = nil
		spec.Template.Spec.HostAliases = nil
		spec.Template.Spec.PriorityClassName = ""
		spec.Template.Spec.Priority = nil
		spec.Template.Spec.DNSConfig = nil
		spec.Template.Spec.ReadinessGates = nil
		spec.Template.Spec.RuntimeClassName = nil
		spec.Template.Spec.EnableServiceLinks = nil

		noFuzzContainer(spec.Template.Spec.Containers)
		noFuzzContainer(spec.Template.Spec.InitContainers)

	}

}

// noFuzzContainer sets all the fields in a Container that are not used in v1alpha7 and later to nil or default values.
func noFuzzContainer(containers []corev1.Container) {
	for i := range containers {
		// We use:
		// - Name
		// - Image
		// - Command
		// - Args
		// - Env
		// - EnvFrom
		// - VolumeMounts
		// - RestartPolicy
		// - ReadinessProbe
		// - StartupProbe
		// - LivenessProbe
		// - TerminationMessagePath
		// - TerminationMessagePolicy
		// - ImagePullPolicy
		// - WorkingDir

		// Don't fuzz anything else:
		containers[i].Resources = corev1.ResourceRequirements{}
		containers[i].ResizePolicy = nil
		containers[i].VolumeDevices = nil
		containers[i].Lifecycle = nil
		containers[i].SecurityContext = nil
		containers[i].Stdin = false
		containers[i].StdinOnce = false
		containers[i].TTY = false
		containers[i].Lifecycle = nil
		containers[i].VolumeDevices = nil
		containers[i].Ports = nil
	}
}

func NnfContainerProfileFuzzFunc(_ runtimeserializer.CodecFactory) []interface{} {
	return []interface{}{
		NnfContainerProfilev6Fuzzer,
	}
}

func NnfContainerProfilev6Fuzzer(in *NnfContainerProfileData, c fuzz.Continue) {
	// Tell the fuzzer to begin by fuzzing everything in the object.
	c.FuzzNoCustom(in)

	if in.Spec != nil {
		noFuzzPodSpec(in.Spec)
	}

	if in.MPISpec != nil {
		noFuzzMPIPodSpec(in.MPISpec)
	}
}

func NnfDataMovementManagerFuzzFunc(_ runtimeserializer.CodecFactory) []interface{} {
	return []interface{}{
		NnfDataMovementManagerv6Fuzzer,
	}
}

func NnfDataMovementManagerv6Fuzzer(in *NnfDataMovementManagerSpec, c fuzz.Continue) {
	// Tell the fuzzer to begin by fuzzing everything in the object.
	c.FuzzNoCustom(in)

	in.Template.ObjectMeta = metav1.ObjectMeta{}

	noFuzzPodSpec(&in.Template.Spec)
}

// Just touch ginkgo, so it's here to interpret any ginkgo args from
// "make test", so that doesn't fail on this test file.
var _ = BeforeSuite(func() {})
