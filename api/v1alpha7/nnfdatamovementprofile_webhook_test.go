/*
 * Copyright 2024-2025 Hewlett Packard Enterprise Development LP
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
	"context"
	"os"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("NnfDataMovementProfile Webhook", func() {
	var (
		namespaceName      = os.Getenv("NNF_DM_PROFILE_NAMESPACE")
		otherNamespaceName string
		otherNamespace     *corev1.Namespace

		pinnedResourceName string
		nnfProfile         *NnfDataMovementProfile
	)

	BeforeEach(func() {
		pinnedResourceName = "test-pinned-" + uuid.NewString()[:8]

		nnfProfile = &NnfDataMovementProfile{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-" + uuid.NewString()[:8],
				Namespace: namespaceName,
			},
		}
	})

	BeforeEach(func() {
		otherNamespaceName = "other-" + uuid.NewString()[:8]

		otherNamespace = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: otherNamespaceName,
			},
		}
		Expect(k8sClient.Create(context.TODO(), otherNamespace)).To(Succeed())
	})

	AfterEach(func() {
		Expect(k8sClient.Delete(context.TODO(), otherNamespace)).To(Succeed())
	})

	AfterEach(func() {
		if nnfProfile != nil {
			Expect(k8sClient.Delete(context.TODO(), nnfProfile)).To(Succeed())
			profExpected := &NnfDataMovementProfile{}
			Eventually(func() error { // Delete can still return the cached object. Wait until the object is no longer present
				return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(nnfProfile), profExpected)
			}).ShouldNot(Succeed())
		}
	})

	It("should accept system profiles in the designated namespace", func() {
		Expect(k8sClient.Create(context.TODO(), nnfProfile)).To(Succeed())
	})

	It("should not accept system profiles that are not in the designated namespace", func() {
		nnfProfile.ObjectMeta.Namespace = otherNamespaceName
		err := k8sClient.Create(context.TODO(), nnfProfile)
		Expect(err.Error()).To(MatchRegexp("webhook .* denied the request: incorrect namespace"))
		nnfProfile = nil
	})

	It("should accept default=true", func() {
		nnfProfile.Data.Default = true
		Expect(k8sClient.Create(context.TODO(), nnfProfile)).To(Succeed())
		Eventually(func() error {
			return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(nnfProfile), nnfProfile)
		}).Should(Succeed())
		Expect(nnfProfile.Data.Default).To(BeTrue())
	})
	It("should accept default=false", func() {
		nnfProfile.Data.Default = false
		Expect(k8sClient.Create(context.TODO(), nnfProfile)).To(Succeed())
		Eventually(func() error {
			return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(nnfProfile), nnfProfile)
		}).Should(Succeed())
		Expect(nnfProfile.Data.Default).ToNot(BeTrue())
	})

	It("Should not allow a default resource to be pinned", func() {
		nnfProfile.Data.Default = true
		nnfProfile.Data.Pinned = true

		Expect(k8sClient.Create(context.TODO(), nnfProfile)).ToNot(Succeed())
		nnfProfile = nil
	})

	It("Should not allow modification of Data in a pinned resource", func() {
		nnfProfile.ObjectMeta.Name = pinnedResourceName
		nnfProfile.ObjectMeta.Namespace = otherNamespaceName
		nnfProfile.Data.Pinned = true

		Expect(k8sClient.Create(context.TODO(), nnfProfile)).To(Succeed())
		Eventually(func() error {
			return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(nnfProfile), nnfProfile)
		}).Should(Succeed())

		Expect(nnfProfile.Data.Pinned).To(BeTrue())
		nnfProfile.Data.Pinned = false
		Expect(k8sClient.Update(context.TODO(), nnfProfile)).ToNot(Succeed())
	})

	It("Should allow modification of Meta in a pinned resource", func() {
		nnfProfile.ObjectMeta.Name = pinnedResourceName
		nnfProfile.ObjectMeta.Namespace = otherNamespaceName
		nnfProfile.Data.Pinned = true

		Expect(k8sClient.Create(context.TODO(), nnfProfile)).To(Succeed())
		Eventually(func() error {
			return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(nnfProfile), nnfProfile)
		}).Should(Succeed())

		Expect(nnfProfile.Data.Pinned).To(BeTrue())
		// A finalizer or ownerRef will interfere with deletion,
		// so set a label, instead.
		labels := nnfProfile.GetLabels()
		if labels == nil {
			labels = make(map[string]string)
		}
		labels["profile-label"] = "profile-label"
		nnfProfile.SetLabels(labels)
		Expect(k8sClient.Update(context.TODO(), nnfProfile)).To(Succeed())
	})

	It("Should not allow an unpinned profile to become pinned", func() {
		Expect(k8sClient.Create(context.TODO(), nnfProfile)).To(Succeed())
		Eventually(func() error {
			return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(nnfProfile), nnfProfile)
		}).Should(Succeed())

		nnfProfile.Data.Pinned = true
		Expect(k8sClient.Update(context.TODO(), nnfProfile)).ToNot(Succeed())
	})

	It("Should not allow a pinned profile to become unpinned", func() {
		nnfProfile.ObjectMeta.Name = pinnedResourceName
		nnfProfile.ObjectMeta.Namespace = otherNamespaceName
		nnfProfile.Data.Pinned = true

		Expect(k8sClient.Create(context.TODO(), nnfProfile)).To(Succeed())
		Eventually(func() error {
			return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(nnfProfile), nnfProfile)
		}).Should(Succeed())

		nnfProfile.Data.Pinned = false
		Expect(k8sClient.Update(context.TODO(), nnfProfile)).ToNot(Succeed())
	})

	It("Should allow slots/maxSlots set to 0", func() {
		nnfProfile.Data.Slots = 0
		nnfProfile.Data.MaxSlots = 0

		Expect(k8sClient.Create(context.TODO(), nnfProfile)).To(Succeed())
	})

	It("Should allow slots/maxSlots set", func() {
		nnfProfile.Data.Slots = 16
		nnfProfile.Data.MaxSlots = 16

		Expect(k8sClient.Create(context.TODO(), nnfProfile)).To(Succeed())
	})

	It("Should allow slots set to 0", func() {
		nnfProfile.Data.Slots = 0
		nnfProfile.Data.MaxSlots = 16

		Expect(k8sClient.Create(context.TODO(), nnfProfile)).To(Succeed())
	})

	It("Should allow maxSlots set to 0", func() {
		nnfProfile.Data.Slots = 16
		nnfProfile.Data.MaxSlots = 0

		Expect(k8sClient.Create(context.TODO(), nnfProfile)).To(Succeed())
	})

	It("Should not allow slots to be set more than maxSlots", func() {
		nnfProfile.Data.Slots = 16
		nnfProfile.Data.MaxSlots = 8

		Expect(k8sClient.Create(context.TODO(), nnfProfile)).ToNot(Succeed())
		nnfProfile = nil
	})

})
