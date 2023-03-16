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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
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
			// Containers cannot be null, so set it
			Data: NnfContainerProfileData{
				Template: v1.PodTemplateSpec{
					Spec: v1.PodSpec{
						Containers: []v1.Container{},
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

	It("Should not allow a negative postRunTimeoutSeconds", func() {
		nnfProfile.ObjectMeta.Name = pinnedResourceName
		nnfProfile.Data.PostRunTimeoutSeconds = -1
		Expect(k8sClient.Create(context.TODO(), nnfProfile)).ToNot(Succeed())
		nnfProfile = nil
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
