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
package v1alpha7

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("NnfContainerSpec", func() {
	Describe("DeepCopyIntoCore", func() {
		var (
			sourceSpec *NnfContainerSpec
			targetSpec *corev1.PodSpec
		)

		BeforeEach(func() {
			sourceSpec = &NnfContainerSpec{
				Containers: []NnfContainer{
					{
						Name:    "test-container",
						Image:   "test-image",
						Command: []string{"echo", "hello"},
						Args:    []string{"arg1", "arg2"},
						Env: []corev1.EnvVar{
							{Name: "ENV_VAR", Value: "value"},
						},
						VolumeMounts: []corev1.VolumeMount{
							{Name: "test-volume", MountPath: "/mnt"},
						},
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
			}
			targetSpec = &corev1.PodSpec{}
		})

		It("should copy containers correctly", func() {
			sourceSpec.ToCorePodSpec(targetSpec)

			Expect(targetSpec.Containers).To(HaveLen(1))
			Expect(targetSpec.Containers[0].Name).To(Equal("test-container"))
			Expect(targetSpec.Containers[0].Image).To(Equal("test-image"))
			Expect(targetSpec.Containers[0].Command).To(Equal([]string{"echo", "hello"}))
			Expect(targetSpec.Containers[0].Args).To(Equal([]string{"arg1", "arg2"}))
			Expect(targetSpec.Containers[0].Env).To(HaveLen(1))
			Expect(targetSpec.Containers[0].Env[0].Name).To(Equal("ENV_VAR"))
			Expect(targetSpec.Containers[0].Env[0].Value).To(Equal("value"))
			Expect(targetSpec.Containers[0].VolumeMounts).To(HaveLen(1))
			Expect(targetSpec.Containers[0].VolumeMounts[0].Name).To(Equal("test-volume"))
			Expect(targetSpec.Containers[0].VolumeMounts[0].MountPath).To(Equal("/mnt"))
		})

		It("should copy init containers correctly", func() {
			sourceSpec.ToCorePodSpec(targetSpec)

			Expect(targetSpec.InitContainers).To(HaveLen(1))
			Expect(targetSpec.InitContainers[0].Name).To(Equal("init-container"))
			Expect(targetSpec.InitContainers[0].Image).To(Equal("init-image"))
			Expect(targetSpec.InitContainers[0].Command).To(Equal([]string{"init-command"}))
		})

		It("should copy volumes correctly", func() {
			sourceSpec.ToCorePodSpec(targetSpec)

			Expect(targetSpec.Volumes).To(HaveLen(1))
			Expect(targetSpec.Volumes[0].Name).To(Equal("test-volume"))
			Expect(targetSpec.Volumes[0].VolumeSource.EmptyDir).ToNot(BeNil())
		})
	})
})
