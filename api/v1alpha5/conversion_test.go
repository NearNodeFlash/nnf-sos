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

package v1alpha5

import (
	corev1 "k8s.io/api/core/v1"
	"testing"

	fuzz "github.com/google/gofuzz"
	common "github.com/kubeflow/common/pkg/apis/common/v1"
	mpiv2beta1 "github.com/kubeflow/mpi-operator/pkg/apis/kubeflow/v2beta1"
	. "github.com/onsi/ginkgo/v2"
	"k8s.io/apimachinery/pkg/api/apitesting/fuzzer"
	runtimeserializer "k8s.io/apimachinery/pkg/runtime/serializer"

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
		Hub:   &nnfv1alpha7.NnfDataMovementManager{},
		Spoke: &NnfDataMovementManager{},
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
		Hub:         &nnfv1alpha7.NnfStorageProfile{},
		Spoke:       &NnfStorageProfile{},
		FuzzerFuncs: []fuzzer.FuzzerFuncs{NnfStorageProfileFuzzFunc},
	}))

	t.Run("for NnfSystemStorage", utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Hub:   &nnfv1alpha7.NnfSystemStorage{},
		Spoke: &NnfSystemStorage{},
	}))

}

func NnfContainerProfileFuzzFunc(_ runtimeserializer.CodecFactory) []interface{} {
	return []interface{}{
		NnfContainerProfilev5Fuzzer,
	}
}

func NnfContainerProfilev5Fuzzer(in *NnfContainerProfileData, c fuzz.Continue) {
	// Tell the fuzzer to begin by fuzzing everything in the object.
	c.FuzzNoCustom(in)

	if in.Spec != nil {

		// All these fields are not used in v1alpha7 and later, so we set them to nil or default values. We only use:
		// - Containers
		// - InitContainers
		// - Volumes
		// Don't fuzz anything else:
		in.Spec.EphemeralContainers = nil
		in.Spec.ActiveDeadlineSeconds = nil
		in.Spec.TerminationGracePeriodSeconds = nil
		in.Spec.Affinity = nil
		in.Spec.RestartPolicy = ""
		in.Spec.DNSPolicy = ""
		in.Spec.NodeSelector = nil
		in.Spec.ServiceAccountName = ""
		in.Spec.AutomountServiceAccountToken = nil
		in.Spec.NodeName = ""
		in.Spec.HostNetwork = false
		in.Spec.HostPID = false
		in.Spec.HostIPC = false
		in.Spec.ShareProcessNamespace = nil
		in.Spec.SecurityContext = nil
		in.Spec.ImagePullSecrets = nil
		in.Spec.Hostname = ""
		in.Spec.Subdomain = ""
		in.Spec.Affinity = nil
		in.Spec.SchedulerName = ""
		in.Spec.Tolerations = nil
		in.Spec.HostAliases = nil
		in.Spec.PriorityClassName = ""
		in.Spec.Priority = nil
		in.Spec.DNSConfig = nil
		in.Spec.ReadinessGates = nil
		in.Spec.RuntimeClassName = nil
		in.Spec.EnableServiceLinks = nil
		in.Spec.PreemptionPolicy = nil
		in.Spec.Overhead = nil
		in.Spec.TopologySpreadConstraints = nil
		in.Spec.DeprecatedServiceAccount = ""
		in.Spec.SetHostnameAsFQDN = nil
		in.Spec.OS = nil
		in.Spec.SchedulingGates = nil
		in.Spec.HostUsers = nil
		in.Spec.ResourceClaims = nil

		noFuzzContainer := func(containers []corev1.Container) {
			for i := range containers {
				// Same as above. We only use the following fields:
				// - Name
				// - Image
				// - Command
				// - Args
				// - Env
				// - EnvFrom
				// - VolumeMounts
				containers[i].Resources = corev1.ResourceRequirements{}
				containers[i].RestartPolicy = nil
				containers[i].ResizePolicy = nil
				containers[i].VolumeDevices = nil
				containers[i].ReadinessProbe = nil
				containers[i].StartupProbe = nil
				containers[i].LivenessProbe = nil
				containers[i].Lifecycle = nil
				containers[i].TerminationMessagePath = ""
				containers[i].TerminationMessagePolicy = ""
				containers[i].ImagePullPolicy = ""
				containers[i].SecurityContext = nil
				containers[i].Stdin = false
				containers[i].StdinOnce = false
				containers[i].TTY = false
				containers[i].WorkingDir = ""
				containers[i].Lifecycle = nil
				containers[i].VolumeDevices = nil
				containers[i].Ports = nil
			}
		}

		noFuzzContainer(in.Spec.Containers)
		noFuzzContainer(in.Spec.InitContainers)
	}

	if in.MPISpec != nil {
		in.MPISpec.SlotsPerWorker = nil
		in.MPISpec.RunPolicy = mpiv2beta1.RunPolicy{}
		in.MPISpec.SSHAuthMountPath = ""
		in.MPISpec.MPIImplementation = ""
		in.MPISpec.MPIReplicaSpecs = make(map[mpiv2beta1.MPIReplicaType]*common.ReplicaSpec)
		in.MPISpec.MPIReplicaSpecs[mpiv2beta1.MPIReplicaTypeWorker] = &common.ReplicaSpec{}
		in.MPISpec.MPIReplicaSpecs[mpiv2beta1.MPIReplicaTypeLauncher] = &common.ReplicaSpec{}
		for _, spec := range in.MPISpec.MPIReplicaSpecs {
			spec.Template.Spec.Containers = nil
			spec.Template.Spec.InitContainers = nil
			spec.Template.Spec.Volumes = nil
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
		}
	}
}

func NnfStorageProfileFuzzFunc(_ runtimeserializer.CodecFactory) []interface{} {
	return []interface{}{
		NnfStorageProfilev5Fuzzer,
	}
}

func NnfStorageProfilev5Fuzzer(in *NnfStorageProfileData, c fuzz.Continue) {
	// Tell the fuzzer to begin by fuzzing everything in the object.
	c.FuzzNoCustom(in)

	// Remove any fuzz from the PostMount and PreUnmount fields in the MDT, MGT, and MGT/MDT
	// command lines. They aren't used, and they're removed starting in v1alpha6
	in.LustreStorage.MdtCmdLines.PostMount = []string{}
	in.LustreStorage.MdtCmdLines.PreUnmount = []string{}
	in.LustreStorage.MgtCmdLines.PostMount = []string{}
	in.LustreStorage.MgtCmdLines.PreUnmount = []string{}
	in.LustreStorage.MgtMdtCmdLines.PostMount = []string{}
	in.LustreStorage.MgtMdtCmdLines.PreUnmount = []string{}
}

// Just touch ginkgo, so it's here to interpret any ginkgo args from
// "make test", so that doesn't fail on this test file.
var _ = BeforeSuite(func() {})
