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
package v1alpha8

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"
)

var _ = Describe("NnfPodSpec", func() {
	Describe("ToCorePodSpec", func() {
		var (
			sourceSpec *NnfPodSpec
			targetSpec *corev1.PodSpec
		)
		rPolicy := corev1.ContainerRestartPolicyAlways

		BeforeEach(func() {
			sourceSpec = &NnfPodSpec{
				Containers: []NnfContainer{
					{
						Name:                     "test-container",
						Image:                    "test-image",
						Command:                  []string{"echo", "hello"},
						Args:                     []string{"arg1", "arg2"},
						Env:                      []corev1.EnvVar{{Name: "ENV_VAR", Value: "value"}},
						EnvFrom:                  []corev1.EnvFromSource{{ConfigMapRef: &corev1.ConfigMapEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: "test-configmap"}}}},
						VolumeMounts:             []corev1.VolumeMount{{Name: "test-volume", MountPath: "/mnt"}},
						WorkingDir:               "/workdir",
						RestartPolicy:            &rPolicy,
						LivenessProbe:            &corev1.Probe{TimeoutSeconds: 5},
						ReadinessProbe:           &corev1.Probe{TimeoutSeconds: 6},
						StartupProbe:             &corev1.Probe{TimeoutSeconds: 7},
						TerminationMessagePath:   "/dev/termination-log",
						TerminationMessagePolicy: corev1.TerminationMessageReadFile,
						ImagePullPolicy:          corev1.PullIfNotPresent,
					},
				},
				InitContainers: []NnfContainer{
					{
						Name:    "init-container",
						Image:   "init-image",
						Command: []string{"init-command"},
					},
				},
				Volumes: []corev1.Volume{
					{
						Name: "test-volume",
						VolumeSource: corev1.VolumeSource{
							EmptyDir: &corev1.EmptyDirVolumeSource{},
						},
					},
				},
				TerminationGracePeriodSeconds: pointer.Int64(30),
				ShareProcessNamespace:         pointer.Bool(true),
				ImagePullSecrets:              []corev1.LocalObjectReference{{Name: "test-secret"}},
				AutomountServiceAccountToken:  pointer.Bool(true),
			}
			targetSpec = nil
		})

		It("should copy containers correctly", func() {
			targetSpec = sourceSpec.ToCorePodSpec()

			Expect(targetSpec.Containers).To(HaveLen(1))
			Expect(targetSpec.Containers[0].Name).To(Equal("test-container"))
			Expect(targetSpec.Containers[0].Image).To(Equal("test-image"))
			Expect(targetSpec.Containers[0].Command).To(Equal([]string{"echo", "hello"}))
			Expect(targetSpec.Containers[0].Args).To(Equal([]string{"arg1", "arg2"}))
			Expect(targetSpec.Containers[0].Env).To(HaveLen(1))
			Expect(targetSpec.Containers[0].Env[0].Name).To(Equal("ENV_VAR"))
			Expect(targetSpec.Containers[0].Env[0].Value).To(Equal("value"))
			Expect(targetSpec.Containers[0].EnvFrom).To(HaveLen(1))
			Expect(targetSpec.Containers[0].EnvFrom[0].ConfigMapRef.Name).To(Equal("test-configmap"))
			Expect(targetSpec.Containers[0].VolumeMounts).To(HaveLen(1))
			Expect(targetSpec.Containers[0].VolumeMounts[0].Name).To(Equal("test-volume"))
			Expect(targetSpec.Containers[0].VolumeMounts[0].MountPath).To(Equal("/mnt"))
			Expect(targetSpec.Containers[0].WorkingDir).To(Equal("/workdir"))
			Expect(*targetSpec.Containers[0].RestartPolicy).To(Equal(corev1.ContainerRestartPolicyAlways))
			Expect(targetSpec.Containers[0].LivenessProbe.TimeoutSeconds).To(Equal(int32(5)))
			Expect(targetSpec.Containers[0].ReadinessProbe.TimeoutSeconds).To(Equal(int32(6)))
			Expect(targetSpec.Containers[0].StartupProbe.TimeoutSeconds).To(Equal(int32(7)))
			Expect(targetSpec.Containers[0].TerminationMessagePath).To(Equal("/dev/termination-log"))
			Expect(targetSpec.Containers[0].TerminationMessagePolicy).To(Equal(corev1.TerminationMessageReadFile))
			Expect(targetSpec.Containers[0].ImagePullPolicy).To(Equal(corev1.PullIfNotPresent))
		})

		It("should copy multiple containers correctly", func() {
			sourceSpec.Containers = []NnfContainer{
				{
					Name:  "container-1",
					Image: "image-1",
				},
				{
					Name:  "container-2",
					Image: "image-2",
				},
			}
			targetSpec = sourceSpec.ToCorePodSpec()

			Expect(targetSpec.Containers).To(HaveLen(2))
			Expect(targetSpec.Containers[0].Name).To(Equal("container-1"))
			Expect(targetSpec.Containers[0].Image).To(Equal("image-1"))
			Expect(targetSpec.Containers[1].Name).To(Equal("container-2"))
			Expect(targetSpec.Containers[1].Image).To(Equal("image-2"))
		})

		It("should copy init containers correctly", func() {
			targetSpec = sourceSpec.ToCorePodSpec()

			Expect(targetSpec.InitContainers).To(HaveLen(1))
			Expect(targetSpec.InitContainers[0].Name).To(Equal("init-container"))
			Expect(targetSpec.InitContainers[0].Image).To(Equal("init-image"))
			Expect(targetSpec.InitContainers[0].Command).To(Equal([]string{"init-command"}))
		})

		It("should copy multiple init containers correctly", func() {
			sourceSpec.InitContainers = []NnfContainer{
				{
					Name:    "init-container-1",
					Image:   "init-image-1",
					Command: []string{"init-command-1"},
				},
				{
					Name:    "init-container-2",
					Image:   "init-image-2",
					Command: []string{"init-command-2"},
				},
			}
			targetSpec = sourceSpec.ToCorePodSpec()

			Expect(targetSpec.InitContainers).To(HaveLen(2))
			Expect(targetSpec.InitContainers[0].Name).To(Equal("init-container-1"))
			Expect(targetSpec.InitContainers[0].Image).To(Equal("init-image-1"))
			Expect(targetSpec.InitContainers[0].Command).To(Equal([]string{"init-command-1"}))
			Expect(targetSpec.InitContainers[1].Name).To(Equal("init-container-2"))
			Expect(targetSpec.InitContainers[1].Image).To(Equal("init-image-2"))
			Expect(targetSpec.InitContainers[1].Command).To(Equal([]string{"init-command-2"}))
		})

		It("should copy volumes correctly", func() {
			targetSpec = sourceSpec.ToCorePodSpec()

			Expect(targetSpec.Volumes).To(HaveLen(1))
			Expect(targetSpec.Volumes[0].Name).To(Equal("test-volume"))
			Expect(targetSpec.Volumes[0].VolumeSource.EmptyDir).ToNot(BeNil())
		})

		It("should copy multiple volumes correctly", func() {
			sourceSpec.Volumes = []corev1.Volume{
				{
					Name: "volume-1",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
				{
					Name: "volume-2",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{LocalObjectReference: corev1.LocalObjectReference{Name: "configmap-1"}},
					},
				},
			}
			targetSpec = sourceSpec.ToCorePodSpec()

			Expect(targetSpec.Volumes).To(HaveLen(2))
			Expect(targetSpec.Volumes[0].Name).To(Equal("volume-1"))
			Expect(targetSpec.Volumes[0].VolumeSource.EmptyDir).ToNot(BeNil())
			Expect(targetSpec.Volumes[1].Name).To(Equal("volume-2"))
			Expect(targetSpec.Volumes[1].VolumeSource.ConfigMap).ToNot(BeNil())
			Expect(targetSpec.Volumes[1].VolumeSource.ConfigMap.LocalObjectReference.Name).To(Equal("configmap-1"))
		})

		It("should copy pod spec fields correctly", func() {
			targetSpec = sourceSpec.ToCorePodSpec()

			Expect(targetSpec.TerminationGracePeriodSeconds).To(Equal(sourceSpec.TerminationGracePeriodSeconds))
			Expect(targetSpec.ShareProcessNamespace).To(Equal(sourceSpec.ShareProcessNamespace))
			Expect(targetSpec.ImagePullSecrets).To(HaveLen(1))
			Expect(targetSpec.ImagePullSecrets[0].Name).To(Equal("test-secret"))
			Expect(targetSpec.AutomountServiceAccountToken).To(Equal(sourceSpec.AutomountServiceAccountToken))
		})

		It("should handle empty optional fields correctly", func() {
			sourceSpec = &NnfPodSpec{
				Containers: []NnfContainer{
					{
						Name:  "test-container",
						Image: "test-image",
					},
				},
				InitContainers:   nil,
				Volumes:          nil,
				ImagePullSecrets: nil,
			}
			targetSpec = sourceSpec.ToCorePodSpec()

			Expect(targetSpec.InitContainers).To(BeEmpty())
			Expect(targetSpec.Volumes).To(BeEmpty())
			Expect(targetSpec.ImagePullSecrets).To(BeEmpty())
		})

		It("should copy probes correctly", func() {
			sourceSpec = &NnfPodSpec{
				Containers: []NnfContainer{
					{
						Name:          "test-container",
						LivenessProbe: &corev1.Probe{TimeoutSeconds: 5},
					},
				},
			}
			targetSpec = sourceSpec.ToCorePodSpec()

			Expect(targetSpec.Containers[0].LivenessProbe).ToNot(BeNil())
			Expect(targetSpec.Containers[0].LivenessProbe.TimeoutSeconds).To(Equal(int32(5)))
		})
	})

	Describe("FromCorePodSpec", func() {
		var (
			sourceSpec *corev1.PodSpec
			targetSpec *NnfPodSpec
		)
		rPolicy := corev1.ContainerRestartPolicyAlways

		BeforeEach(func() {
			sourceSpec = &corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:                     "test-container",
						Image:                    "test-image",
						Command:                  []string{"echo", "hello"},
						Args:                     []string{"arg1", "arg2"},
						Env:                      []corev1.EnvVar{{Name: "ENV_VAR", Value: "value"}},
						EnvFrom:                  []corev1.EnvFromSource{{ConfigMapRef: &corev1.ConfigMapEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: "test-configmap"}}}},
						VolumeMounts:             []corev1.VolumeMount{{Name: "test-volume", MountPath: "/mnt"}},
						WorkingDir:               "/workdir",
						RestartPolicy:            &rPolicy,
						LivenessProbe:            &corev1.Probe{TimeoutSeconds: 5},
						ReadinessProbe:           &corev1.Probe{TimeoutSeconds: 6},
						StartupProbe:             &corev1.Probe{TimeoutSeconds: 7},
						TerminationMessagePath:   "/dev/termination-log",
						TerminationMessagePolicy: corev1.TerminationMessageReadFile,
						ImagePullPolicy:          corev1.PullIfNotPresent,
					},
				},
				InitContainers: []corev1.Container{
					{
						Name:    "init-container",
						Image:   "init-image",
						Command: []string{"init-command"},
					},
				},
				Volumes: []corev1.Volume{
					{
						Name: "test-volume",
						VolumeSource: corev1.VolumeSource{
							EmptyDir: &corev1.EmptyDirVolumeSource{},
						},
					},
				},
				TerminationGracePeriodSeconds: pointer.Int64(30),
				ShareProcessNamespace:         pointer.Bool(true),
				ImagePullSecrets:              []corev1.LocalObjectReference{{Name: "test-secret"}},
				AutomountServiceAccountToken:  pointer.Bool(true),
			}
			targetSpec = &NnfPodSpec{}
		})

		It("should copy containers correctly", func() {
			targetSpec.FromCorePodSpec(sourceSpec)

			Expect(targetSpec.Containers).To(HaveLen(1))
			Expect(targetSpec.Containers[0].Name).To(Equal("test-container"))
			Expect(targetSpec.Containers[0].Image).To(Equal("test-image"))
			Expect(targetSpec.Containers[0].Command).To(Equal([]string{"echo", "hello"}))
			Expect(targetSpec.Containers[0].Args).To(Equal([]string{"arg1", "arg2"}))
			Expect(targetSpec.Containers[0].Env).To(HaveLen(1))
			Expect(targetSpec.Containers[0].Env[0].Name).To(Equal("ENV_VAR"))
			Expect(targetSpec.Containers[0].Env[0].Value).To(Equal("value"))
			Expect(targetSpec.Containers[0].EnvFrom).To(HaveLen(1))
			Expect(targetSpec.Containers[0].EnvFrom[0].ConfigMapRef.Name).To(Equal("test-configmap"))
			Expect(targetSpec.Containers[0].VolumeMounts).To(HaveLen(1))
			Expect(targetSpec.Containers[0].VolumeMounts[0].Name).To(Equal("test-volume"))
			Expect(targetSpec.Containers[0].VolumeMounts[0].MountPath).To(Equal("/mnt"))
			Expect(targetSpec.Containers[0].WorkingDir).To(Equal("/workdir"))
			Expect(*targetSpec.Containers[0].RestartPolicy).To(Equal(corev1.ContainerRestartPolicyAlways))
			Expect(targetSpec.Containers[0].LivenessProbe.TimeoutSeconds).To(Equal(int32(5)))
			Expect(targetSpec.Containers[0].ReadinessProbe.TimeoutSeconds).To(Equal(int32(6)))
			Expect(targetSpec.Containers[0].StartupProbe.TimeoutSeconds).To(Equal(int32(7)))
			Expect(targetSpec.Containers[0].TerminationMessagePath).To(Equal("/dev/termination-log"))
			Expect(targetSpec.Containers[0].TerminationMessagePolicy).To(Equal(corev1.TerminationMessageReadFile))
			Expect(targetSpec.Containers[0].ImagePullPolicy).To(Equal(corev1.PullIfNotPresent))
		})

		It("should copy init containers correctly", func() {
			targetSpec.FromCorePodSpec(sourceSpec)

			Expect(targetSpec.InitContainers).To(HaveLen(1))
			Expect(targetSpec.InitContainers[0].Name).To(Equal("init-container"))
			Expect(targetSpec.InitContainers[0].Image).To(Equal("init-image"))
			Expect(targetSpec.InitContainers[0].Command).To(Equal([]string{"init-command"}))
		})

		It("should copy multiple init containers correctly", func() {
			sourceSpec.InitContainers = []corev1.Container{
				{
					Name:    "init-container-1",
					Image:   "init-image-1",
					Command: []string{"init-command-1"},
				},
				{
					Name:    "init-container-2",
					Image:   "init-image-2",
					Command: []string{"init-command-2"},
				},
			}
			targetSpec.FromCorePodSpec(sourceSpec)

			Expect(targetSpec.InitContainers).To(HaveLen(2))
			Expect(targetSpec.InitContainers[0].Name).To(Equal("init-container-1"))
			Expect(targetSpec.InitContainers[0].Image).To(Equal("init-image-1"))
			Expect(targetSpec.InitContainers[0].Command).To(Equal([]string{"init-command-1"}))
			Expect(targetSpec.InitContainers[1].Name).To(Equal("init-container-2"))
			Expect(targetSpec.InitContainers[1].Image).To(Equal("init-image-2"))
			Expect(targetSpec.InitContainers[1].Command).To(Equal([]string{"init-command-2"}))
		})

		It("should copy volumes correctly", func() {
			targetSpec.FromCorePodSpec(sourceSpec)

			Expect(targetSpec.Volumes).To(HaveLen(1))
			Expect(targetSpec.Volumes[0].Name).To(Equal("test-volume"))
			Expect(targetSpec.Volumes[0].VolumeSource.EmptyDir).ToNot(BeNil())
		})

		It("should copy multiple volumes correctly", func() {
			sourceSpec.Volumes = []corev1.Volume{
				{
					Name: "volume-1",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
				{
					Name: "volume-2",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{LocalObjectReference: corev1.LocalObjectReference{Name: "configmap-1"}},
					},
				},
			}
			targetSpec.FromCorePodSpec(sourceSpec)

			Expect(targetSpec.Volumes).To(HaveLen(2))
			Expect(targetSpec.Volumes[0].Name).To(Equal("volume-1"))
			Expect(targetSpec.Volumes[0].VolumeSource.EmptyDir).ToNot(BeNil())
			Expect(targetSpec.Volumes[1].Name).To(Equal("volume-2"))
			Expect(targetSpec.Volumes[1].VolumeSource.ConfigMap).ToNot(BeNil())
			Expect(targetSpec.Volumes[1].VolumeSource.ConfigMap.LocalObjectReference.Name).To(Equal("configmap-1"))
		})

		It("should copy pod spec fields correctly", func() {
			targetSpec.FromCorePodSpec(sourceSpec)

			Expect(targetSpec.TerminationGracePeriodSeconds).To(Equal(sourceSpec.TerminationGracePeriodSeconds))
			Expect(targetSpec.ShareProcessNamespace).To(Equal(sourceSpec.ShareProcessNamespace))
			Expect(targetSpec.ImagePullSecrets).To(HaveLen(1))
			Expect(targetSpec.ImagePullSecrets[0].Name).To(Equal("test-secret"))
			Expect(targetSpec.AutomountServiceAccountToken).To(Equal(sourceSpec.AutomountServiceAccountToken))
		})

		It("should handle nil sourceSpec gracefully", func() {
			sourceSpec = nil
			targetSpec = &NnfPodSpec{}
			targetSpec.FromCorePodSpec(sourceSpec)

			Expect(targetSpec.Containers).To(BeEmpty())
			Expect(targetSpec.InitContainers).To(BeEmpty())
			Expect(targetSpec.Volumes).To(BeEmpty())
			Expect(targetSpec.TerminationGracePeriodSeconds).To(BeNil())
			Expect(targetSpec.ShareProcessNamespace).To(BeNil())
			Expect(targetSpec.ImagePullSecrets).To(BeEmpty())
			Expect(targetSpec.AutomountServiceAccountToken).To(BeNil())
		})

		It("should deeply copy volumes", func() {
			sourceSpec = &corev1.PodSpec{
				Volumes: []corev1.Volume{
					{
						Name: "test-volume",
						VolumeSource: corev1.VolumeSource{
							EmptyDir: &corev1.EmptyDirVolumeSource{},
						},
					},
				},
			}
			targetSpec = &NnfPodSpec{}
			targetSpec.FromCorePodSpec(sourceSpec)

			Expect(targetSpec.Volumes).To(HaveLen(1))
			Expect(targetSpec.Volumes[0].Name).To(Equal("test-volume"))
			Expect(targetSpec.Volumes[0].VolumeSource.EmptyDir).ToNot(BeNil())

			// Modify source and ensure target is unaffected
			sourceSpec.Volumes[0].Name = "modified-volume"
			Expect(targetSpec.Volumes[0].Name).To(Equal("test-volume"))
		})
	})
})
