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

var _ = Describe("NnfPodSpec", func() {
	Describe("ToCorePodSpec", func() {
		var (
			sourceSpec *NnfPodSpec
			targetSpec *corev1.PodSpec
		)

		BeforeEach(func() {
			sourceSpec = &NnfPodSpec{
				Containers: []NnfContainer{
					{
						Name:    "test-container",
						Image:   "test-image",
						Command: []string{"echo", "hello"},
						Args:    []string{"arg1", "arg2"},
						Env: []corev1.EnvVar{
							{Name: "ENV_VAR", Value: "value"},
						},
						EnvFrom: []corev1.EnvFromSource{{
							ConfigMapRef: &corev1.ConfigMapEnvSource{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: "test-configmap",
								},
							}},
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
		})

		It("should copy init containers correctly", func() {
			targetSpec = sourceSpec.ToCorePodSpec()

			Expect(targetSpec.InitContainers).To(HaveLen(1))
			Expect(targetSpec.InitContainers[0].Name).To(Equal("init-container"))
			Expect(targetSpec.InitContainers[0].Image).To(Equal("init-image"))
			Expect(targetSpec.InitContainers[0].Command).To(Equal([]string{"init-command"}))
		})

		It("should copy volumes correctly", func() {
			targetSpec = sourceSpec.ToCorePodSpec()

			Expect(targetSpec.Volumes).To(HaveLen(1))
			Expect(targetSpec.Volumes[0].Name).To(Equal("test-volume"))
			Expect(targetSpec.Volumes[0].VolumeSource.EmptyDir).ToNot(BeNil())
		})
	})

	Describe("FromCorePodSpec", func() {
		var (
			sourceSpec *corev1.PodSpec
			targetSpec *NnfPodSpec
		)

		BeforeEach(func() {
			sourceSpec = &corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:    "test-container",
						Image:   "test-image",
						Command: []string{"echo", "hello"},
						Args:    []string{"arg1", "arg2"},
						Env: []corev1.EnvVar{
							{Name: "ENV_VAR", Value: "value"},
						},
						EnvFrom: []corev1.EnvFromSource{{
							ConfigMapRef: &corev1.ConfigMapEnvSource{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: "test-configmap",
								},
							}},
						},
						VolumeMounts: []corev1.VolumeMount{
							{Name: "test-volume", MountPath: "/mnt"},
						},
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
		})

		It("should copy init containers correctly", func() {
			targetSpec.FromCorePodSpec(sourceSpec)

			Expect(targetSpec.InitContainers).To(HaveLen(1))
			Expect(targetSpec.InitContainers[0].Name).To(Equal("init-container"))
			Expect(targetSpec.InitContainers[0].Image).To(Equal("init-image"))
			Expect(targetSpec.InitContainers[0].Command).To(Equal([]string{"init-command"}))
		})

		It("should copy volumes correctly", func() {
			targetSpec.FromCorePodSpec(sourceSpec)

			Expect(targetSpec.Volumes).To(HaveLen(1))
			Expect(targetSpec.Volumes[0].Name).To(Equal("test-volume"))
			Expect(targetSpec.Volumes[0].VolumeSource.EmptyDir).ToNot(BeNil())
		})
	})
})
