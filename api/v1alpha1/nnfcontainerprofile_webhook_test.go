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

package v1alpha1

import (
	"context"
	"os"

	mpicommonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	mpiv2beta1 "github.com/kubeflow/mpi-operator/pkg/apis/kubeflow/v2beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// These tests are written in BDD-style using Ginkgo framework. Refer to
// http://onsi.github.io/ginkgo to learn more.

var _ = Describe("NnfContainerProfile Webhook", func() {
	var (
		namespaceName                           = os.Getenv("NNF_CONTAINER_PROFILE_NAMESPACE")
		pinnedResourceName                      = "test-pinned"
		nnfProfile         *NnfContainerProfile = nil
		newProfile         *NnfContainerProfile
	)

	BeforeEach(func() {
		nnfProfile = &NnfContainerProfile{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: namespaceName,
			},
			Data: NnfContainerProfileData{
				Spec: &corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "test"},
					},
				},
			},
		}

		newProfile = &NnfContainerProfile{}
	})

	AfterEach(func() {
		if nnfProfile != nil {
			Expect(k8sClient.Delete(context.TODO(), nnfProfile)).To(Succeed())
			profExpected := &NnfContainerProfile{}
			Eventually(func() error { // Delete can still return the cached object. Wait until the object is no longer present
				return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(nnfProfile), profExpected)
			}).ShouldNot(Succeed())
		}
	})

	It("Should not allow a negative retryLimit", func() {
		nnfProfile.ObjectMeta.Name = pinnedResourceName
		nnfProfile.Data.RetryLimit = -1
		Expect(k8sClient.Create(context.TODO(), nnfProfile)).ToNot(Succeed())
		nnfProfile = nil
	})

	It("Should allow a zero retryLimit", func() {
		nnfProfile.ObjectMeta.Name = pinnedResourceName
		nnfProfile.Data.RetryLimit = 0
		Expect(k8sClient.Create(context.TODO(), nnfProfile)).To(Succeed())
	})

	It("Should not allow a negative postRunTimeoutSeconds", func() {
		nnfProfile.ObjectMeta.Name = pinnedResourceName
		nnfProfile.Data.PostRunTimeoutSeconds = -1
		Expect(k8sClient.Create(context.TODO(), nnfProfile)).ToNot(Succeed())
		nnfProfile = nil
	})

	It("Should not allow setting both Spec and MPISpec", func() {
		nnfProfile.ObjectMeta.Name = pinnedResourceName
		nnfProfile.Data.Spec = &corev1.PodSpec{}
		nnfProfile.Data.MPISpec = &mpiv2beta1.MPIJobSpec{}
		Expect(k8sClient.Create(context.TODO(), nnfProfile)).ToNot(Succeed())
		nnfProfile = nil
	})

	It("Should fail when both Spec and MPISpec are unset", func() {
		nnfProfile.ObjectMeta.Name = pinnedResourceName
		nnfProfile.Data.Spec = nil
		nnfProfile.Data.MPISpec = nil
		Expect(k8sClient.Create(context.TODO(), nnfProfile)).ToNot(Succeed())
		nnfProfile = nil
	})

	It("Should not allow an empty MPIReplicaSpecs", func() {
		nnfProfile.ObjectMeta.Name = pinnedResourceName
		nnfProfile.Data.MPISpec = &mpiv2beta1.MPIJobSpec{
			MPIReplicaSpecs: map[mpiv2beta1.MPIReplicaType]*mpicommonv1.ReplicaSpec{},
		}
		Expect(k8sClient.Create(context.TODO(), nnfProfile)).ToNot(Succeed())
		nnfProfile = nil
	})

	It("Should not allow both an empty Launcher and Worker ReplicaSpecs", func() {
		nnfProfile.ObjectMeta.Name = pinnedResourceName
		nnfProfile.Data.MPISpec = &mpiv2beta1.MPIJobSpec{
			MPIReplicaSpecs: map[mpiv2beta1.MPIReplicaType]*mpicommonv1.ReplicaSpec{
				mpiv2beta1.MPIReplicaTypeLauncher: nil,
				mpiv2beta1.MPIReplicaTypeWorker:   nil,
			},
		}
		Expect(k8sClient.Create(context.TODO(), nnfProfile)).ToNot(Succeed())
		nnfProfile = nil
	})

	It("Should not allow an empty Launcher ReplicaSpec", func() {
		nnfProfile.ObjectMeta.Name = pinnedResourceName
		nnfProfile.Data.MPISpec = &mpiv2beta1.MPIJobSpec{
			MPIReplicaSpecs: map[mpiv2beta1.MPIReplicaType]*mpicommonv1.ReplicaSpec{
				mpiv2beta1.MPIReplicaTypeLauncher: nil,
				mpiv2beta1.MPIReplicaTypeWorker: {
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{},
					},
				},
			},
		}
		Expect(k8sClient.Create(context.TODO(), nnfProfile)).ToNot(Succeed())
		nnfProfile = nil
	})

	It("Should not allow an empty Worker ReplicaSpec", func() {
		nnfProfile.ObjectMeta.Name = pinnedResourceName
		nnfProfile.Data.MPISpec = &mpiv2beta1.MPIJobSpec{
			MPIReplicaSpecs: map[mpiv2beta1.MPIReplicaType]*mpicommonv1.ReplicaSpec{
				mpiv2beta1.MPIReplicaTypeLauncher: {
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{},
					},
				},
				mpiv2beta1.MPIReplicaTypeWorker: nil,
			},
		}
		Expect(k8sClient.Create(context.TODO(), nnfProfile)).ToNot(Succeed())
		nnfProfile = nil
	})

	It("Should not allow an empty Launcher and Worker PodSpecs", func() {
		nnfProfile.ObjectMeta.Name = pinnedResourceName
		nnfProfile.Data.MPISpec = &mpiv2beta1.MPIJobSpec{
			MPIReplicaSpecs: map[mpiv2beta1.MPIReplicaType]*mpicommonv1.ReplicaSpec{
				mpiv2beta1.MPIReplicaTypeLauncher: {
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{},
					},
				},
				mpiv2beta1.MPIReplicaTypeWorker: {
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{},
					},
				},
			},
		}
		Expect(k8sClient.Create(context.TODO(), nnfProfile)).ToNot(Succeed())
		nnfProfile = nil
	})

	It("Should not allow setting both PostRunTimeoutSeconds and MPISpec.RunPolicy.ActiveDeadlineSeconds", func() {
		nnfProfile.ObjectMeta.Name = pinnedResourceName
		nnfProfile.Data.Spec = nil
		nnfProfile.Data.MPISpec = &mpiv2beta1.MPIJobSpec{}

		timeout := int64(10)
		nnfProfile.Data.PostRunTimeoutSeconds = timeout
		nnfProfile.Data.MPISpec.RunPolicy.ActiveDeadlineSeconds = &timeout

		Expect(k8sClient.Create(context.TODO(), nnfProfile)).ToNot(Succeed())
		nnfProfile = nil
	})

	It("Should not allow setting both PostRunTimeoutSeconds and Spec.ActiveDeadlineSeconds", func() {
		nnfProfile.ObjectMeta.Name = pinnedResourceName

		timeout := int64(10)
		nnfProfile.Data.PostRunTimeoutSeconds = timeout
		nnfProfile.Data.Spec.ActiveDeadlineSeconds = &timeout

		Expect(k8sClient.Create(context.TODO(), nnfProfile)).ToNot(Succeed())
		nnfProfile = nil
	})

	It("Should not allow setting MPISpec.RunPolicy.BackoffLimit directly", func() {
		nnfProfile.ObjectMeta.Name = pinnedResourceName
		nnfProfile.Data.Spec = nil
		nnfProfile.Data.MPISpec = &mpiv2beta1.MPIJobSpec{}

		limit := int32(10)
		nnfProfile.Data.MPISpec.RunPolicy.BackoffLimit = &limit

		Expect(k8sClient.Create(context.TODO(), nnfProfile)).ToNot(Succeed())
		nnfProfile = nil
	})

	It("Should allow a zero postRunTimeoutSeconds", func() {
		nnfProfile.ObjectMeta.Name = pinnedResourceName
		nnfProfile.Data.PostRunTimeoutSeconds = 0
		Expect(k8sClient.Create(context.TODO(), nnfProfile)).To(Succeed())
	})

	It("Should not allow modification of Data in a pinned resource", func() {
		nnfProfile.ObjectMeta.Name = pinnedResourceName
		Expect(k8sClient.Create(context.TODO(), nnfProfile)).To(Succeed())
		Eventually(func() error {
			return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(nnfProfile), nnfProfile)
		}).Should(Succeed())

		// Set it as pinned with an Update
		nnfProfile.Data.Pinned = true
		Expect(k8sClient.Update(context.TODO(), nnfProfile)).To(Succeed())

		// Verify pinned
		Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(nnfProfile), newProfile)).To(Succeed())
		Expect(newProfile.Data.Pinned).To(BeTrue())

		// Try to update Data and fail
		newProfile.Data.RetryLimit = 10
		Expect(k8sClient.Update(context.TODO(), newProfile)).ToNot(Succeed())
	})

	It("Should allow modification of Meta in a pinned resource", func() {
		nnfProfile.ObjectMeta.Name = pinnedResourceName
		Expect(k8sClient.Create(context.TODO(), nnfProfile)).To(Succeed())
		Eventually(func() error {
			return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(nnfProfile), nnfProfile)
		}).Should(Succeed())

		// Set it as pinned with an Update
		nnfProfile.Data.Pinned = true
		Expect(k8sClient.Update(context.TODO(), nnfProfile)).To(Succeed())

		// Verify pinned
		Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(nnfProfile), newProfile)).To(Succeed())
		Expect(newProfile.Data.Pinned).To(BeTrue())

		// Try to update metadata and succeed. A finalizer or ownerRef will interfere with deletion,
		// so set a label, instead.
		labels := newProfile.GetLabels()
		if labels == nil {
			labels = make(map[string]string)
		}
		labels["profile-label"] = "profile-label"
		newProfile.SetLabels(labels)
		Expect(k8sClient.Update(context.TODO(), newProfile)).To(Succeed())
	})
})
