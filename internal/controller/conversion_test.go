/*
 * Copyright 2024-2026 Hewlett Packard Enterprise Development LP
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

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	nnfv1alpha10 "github.com/NearNodeFlash/nnf-sos/api/v1alpha10"
	nnfv1alpha8 "github.com/NearNodeFlash/nnf-sos/api/v1alpha8"
	nnfv1alpha9 "github.com/NearNodeFlash/nnf-sos/api/v1alpha9"
	utilconversion "github.com/NearNodeFlash/nnf-sos/github/cluster-api/util/conversion"
)

var _ = Describe("Conversion Webhook Test", func() {

	// Don't get deep into verifying the conversion.
	// We have api/<spoke_ver>/conversion_test.go that is digging deep.
	// We're just verifying that the conversion webhook is hooked up.

	// Note: if a resource is accessed by its spoke API, then it should
	// have the utilconversion.DataAnnotation annotation.  It will not
	// have that annotation when it is accessed by its hub API.

	Context("NnfAccess", func() {
		var resHub *nnfv1alpha10.NnfAccess

		BeforeEach(func() {
			id := uuid.NewString()[0:8]
			resHub = &nnfv1alpha10.NnfAccess{
				ObjectMeta: metav1.ObjectMeta{
					Name:      id,
					Namespace: corev1.NamespaceDefault,
				},
				Spec: nnfv1alpha10.NnfAccessSpec{
					DesiredState:  "mounted",
					TeardownState: "Teardown",
					Target:        "all",
					UserID:        1001,
					GroupID:       2002,
				},
			}

			Expect(k8sClient.Create(context.TODO(), resHub)).To(Succeed())
		})

		AfterEach(func() {
			if resHub != nil {
				Expect(k8sClient.Delete(context.TODO(), resHub)).To(Succeed())
				expected := &nnfv1alpha10.NnfAccess{}
				Eventually(func() error { // Delete can still return the cached object. Wait until the object is no longer present.
					return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(resHub), expected)
				}).ShouldNot(Succeed())
			}
		})

		It("is unable to read NnfAccess resource via spoke v1alpha8", func() {
			resSpoke := &nnfv1alpha8.NnfAccess{}
			Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(resHub), resSpoke)).ToNot(Succeed())
		})

		It("reads NnfAccess resource via hub and via spoke v1alpha9", func() {
			// Spoke should have annotation.
			resSpoke := &nnfv1alpha9.NnfAccess{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(resHub), resSpoke)).To(Succeed())
				anno := resSpoke.GetAnnotations()
				g.Expect(anno).To(HaveLen(1))
				g.Expect(anno).Should(HaveKey(utilconversion.DataAnnotation))
			}).Should(Succeed())

			// Hub should not have annotation.
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(resHub), resHub)).To(Succeed())
				anno := resHub.GetAnnotations()
				g.Expect(anno).To(HaveLen(0))
			}).Should(Succeed())
		})

		// +crdbumper:scaffold:spoketest="nnf.NnfAccess"
	})

	Context("NnfContainerProfile", func() {
		var resHub *nnfv1alpha10.NnfContainerProfile

		BeforeEach(func() {
			id := uuid.NewString()[0:8]
			resHub = &nnfv1alpha10.NnfContainerProfile{
				ObjectMeta: metav1.ObjectMeta{
					Name:      id,
					Namespace: corev1.NamespaceDefault,
				},
				Data: nnfv1alpha10.NnfContainerProfileData{
					NnfSpec: &nnfv1alpha10.NnfPodSpec{
						Containers: []nnfv1alpha10.NnfContainer{
							{Name: "one", Image: "nginx:latest", Command: []string{"echo", "hello"}},
						},
					},
				},
			}

			Expect(k8sClient.Create(context.TODO(), resHub)).To(Succeed())
		})

		AfterEach(func() {
			if resHub != nil {
				Expect(k8sClient.Delete(context.TODO(), resHub)).To(Succeed())
				expected := &nnfv1alpha10.NnfContainerProfile{}
				Eventually(func() error { // Delete can still return the cached object. Wait until the object is no longer present.
					return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(resHub), expected)
				}).ShouldNot(Succeed())
			}
		})

		It("is unable to read NnfContainerProfile resource via spoke v1alpha8", func() {
			resSpoke := &nnfv1alpha8.NnfContainerProfile{}
			Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(resHub), resSpoke)).ToNot(Succeed())
		})

		It("reads NnfContainerProfile resource via hub and via spoke v1alpha9", func() {
			// Spoke should have annotation.
			resSpoke := &nnfv1alpha9.NnfContainerProfile{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(resHub), resSpoke)).To(Succeed())
				anno := resSpoke.GetAnnotations()
				g.Expect(anno).To(HaveLen(1))
				g.Expect(anno).Should(HaveKey(utilconversion.DataAnnotation))
			}).Should(Succeed())

			// Hub should not have annotation.
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(resHub), resHub)).To(Succeed())
				anno := resHub.GetAnnotations()
				g.Expect(anno).To(HaveLen(0))
			}).Should(Succeed())
		})

		// +crdbumper:scaffold:spoketest="nnf.NnfContainerProfile"
	})

	Context("NnfDataMovement", func() {
		var resHub *nnfv1alpha10.NnfDataMovement

		BeforeEach(func() {
			id := uuid.NewString()[0:8]
			resHub = &nnfv1alpha10.NnfDataMovement{
				ObjectMeta: metav1.ObjectMeta{
					Name:      id,
					Namespace: corev1.NamespaceDefault,
				},
				Spec: nnfv1alpha10.NnfDataMovementSpec{},
			}

			Expect(k8sClient.Create(context.TODO(), resHub)).To(Succeed())
		})

		AfterEach(func() {
			if resHub != nil {
				Expect(k8sClient.Delete(context.TODO(), resHub)).To(Succeed())
				expected := &nnfv1alpha10.NnfDataMovement{}
				Eventually(func() error { // Delete can still return the cached object. Wait until the object is no longer present.
					return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(resHub), expected)
				}).ShouldNot(Succeed())
			}
		})

		It("is unable to read NnfDataMovement resource via spoke v1alpha8", func() {
			resSpoke := &nnfv1alpha8.NnfDataMovement{}
			Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(resHub), resSpoke)).ToNot(Succeed())
		})

		It("reads NnfDataMovement resource via hub and via spoke v1alpha9", func() {
			// Spoke should have annotation.
			resSpoke := &nnfv1alpha9.NnfDataMovement{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(resHub), resSpoke)).To(Succeed())
				anno := resSpoke.GetAnnotations()
				g.Expect(anno).To(HaveLen(1))
				g.Expect(anno).Should(HaveKey(utilconversion.DataAnnotation))
			}).Should(Succeed())

			// Hub should not have annotation.
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(resHub), resHub)).To(Succeed())
				anno := resHub.GetAnnotations()
				g.Expect(anno).To(HaveLen(0))
			}).Should(Succeed())
		})

		// +crdbumper:scaffold:spoketest="nnf.NnfDataMovement"
	})

	Context("NnfDataMovementManager", func() {
		var resHub *nnfv1alpha10.NnfDataMovementManager

		BeforeEach(func() {
			id := uuid.NewString()[0:8]
			resHub = &nnfv1alpha10.NnfDataMovementManager{
				ObjectMeta: metav1.ObjectMeta{
					Name:      id,
					Namespace: corev1.NamespaceDefault,
				},
				Spec: nnfv1alpha10.NnfDataMovementManagerSpec{
					PodSpec: nnfv1alpha10.NnfPodSpec{
						Containers: []nnfv1alpha10.NnfContainer{{
							Name:  "dm-worker-dummy",
							Image: "nginx",
						}},
					},
				},
			}

			Expect(k8sClient.Create(context.TODO(), resHub)).To(Succeed())
		})

		AfterEach(func() {
			if resHub != nil {
				Expect(k8sClient.Delete(context.TODO(), resHub)).To(Succeed())
				expected := &nnfv1alpha10.NnfDataMovementManager{}
				Eventually(func() error { // Delete can still return the cached object. Wait until the object is no longer present.
					return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(resHub), expected)
				}).ShouldNot(Succeed())
			}
		})

		It("is unable to read NnfDataMovementManager resource via spoke v1alpha8", func() {
			resSpoke := &nnfv1alpha8.NnfDataMovementManager{}
			Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(resHub), resSpoke)).ToNot(Succeed())
		})

		It("reads NnfDataMovementManager resource via hub and via spoke v1alpha9", func() {
			// Spoke should have annotation.
			resSpoke := &nnfv1alpha9.NnfDataMovementManager{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(resHub), resSpoke)).To(Succeed())
				anno := resSpoke.GetAnnotations()
				g.Expect(anno).To(HaveLen(1))
				g.Expect(anno).Should(HaveKey(utilconversion.DataAnnotation))
			}).Should(Succeed())

			// Hub should not have annotation.
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(resHub), resHub)).To(Succeed())
				anno := resHub.GetAnnotations()
				g.Expect(anno).To(HaveLen(0))
			}).Should(Succeed())
		})

		// +crdbumper:scaffold:spoketest="nnf.NnfDataMovementManager"
	})

	Context("NnfDataMovementProfile", func() {
		var resHub *nnfv1alpha10.NnfDataMovementProfile

		BeforeEach(func() {
			id := uuid.NewString()[0:8]
			resHub = &nnfv1alpha10.NnfDataMovementProfile{
				ObjectMeta: metav1.ObjectMeta{
					Name:      id,
					Namespace: corev1.NamespaceDefault,
				},
				Data: nnfv1alpha10.NnfDataMovementProfileData{},
			}

			Expect(k8sClient.Create(context.TODO(), resHub)).To(Succeed())
		})

		AfterEach(func() {
			if resHub != nil {
				Expect(k8sClient.Delete(context.TODO(), resHub)).To(Succeed())
				expected := &nnfv1alpha10.NnfDataMovementProfile{}
				Eventually(func() error { // Delete can still return the cached object. Wait until the object is no longer present.
					return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(resHub), expected)
				}).ShouldNot(Succeed())
			}
		})

		It("is unable to read NnfDataMovementProfile resource via spoke v1alpha8", func() {
			resSpoke := &nnfv1alpha8.NnfDataMovementProfile{}
			Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(resHub), resSpoke)).ToNot(Succeed())
		})

		It("reads NnfDataMovementProfile resource via hub and via spoke v1alpha9", func() {
			// Spoke should have annotation.
			resSpoke := &nnfv1alpha9.NnfDataMovementProfile{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(resHub), resSpoke)).To(Succeed())
				anno := resSpoke.GetAnnotations()
				g.Expect(anno).To(HaveLen(1))
				g.Expect(anno).Should(HaveKey(utilconversion.DataAnnotation))
			}).Should(Succeed())

			// Hub should not have annotation.
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(resHub), resHub)).To(Succeed())
				anno := resHub.GetAnnotations()
				g.Expect(anno).To(HaveLen(0))
			}).Should(Succeed())
		})

		// +crdbumper:scaffold:spoketest="nnf.NnfDataMovementProfile"
	})

	Context("NnfLustreMGT", func() {
		var resHub *nnfv1alpha10.NnfLustreMGT

		BeforeEach(func() {
			id := uuid.NewString()[0:8]
			resHub = &nnfv1alpha10.NnfLustreMGT{
				ObjectMeta: metav1.ObjectMeta{
					Name:      id,
					Namespace: corev1.NamespaceDefault,
				},
				Spec: nnfv1alpha10.NnfLustreMGTSpec{
					Addresses: []string{"rabbit-1@tcp", "rabbit-2@tcp"},
				},
			}

			Expect(k8sClient.Create(context.TODO(), resHub)).To(Succeed())
		})

		AfterEach(func() {
			if resHub != nil {
				Expect(k8sClient.Delete(context.TODO(), resHub)).To(Succeed())
				expected := &nnfv1alpha10.NnfLustreMGT{}
				Eventually(func() error { // Delete can still return the cached object. Wait until the object is no longer present.
					return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(resHub), expected)
				}).ShouldNot(Succeed())
			}
		})

		It("is unable to read NnfLustreMGT resource via spoke v1alpha8", func() {
			resSpoke := &nnfv1alpha8.NnfLustreMGT{}
			Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(resHub), resSpoke)).ToNot(Succeed())
		})

		It("reads NnfLustreMGT resource via hub and via spoke v1alpha9", func() {
			// Spoke should have annotation.
			resSpoke := &nnfv1alpha9.NnfLustreMGT{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(resHub), resSpoke)).To(Succeed())
				anno := resSpoke.GetAnnotations()
				g.Expect(anno).To(HaveLen(1))
				g.Expect(anno).Should(HaveKey(utilconversion.DataAnnotation))
			}).Should(Succeed())

			// Hub should not have annotation.
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(resHub), resHub)).To(Succeed())
				anno := resHub.GetAnnotations()
				g.Expect(anno).To(HaveLen(0))
			}).Should(Succeed())
		})

		// +crdbumper:scaffold:spoketest="nnf.NnfLustreMGT"
	})

	Context("NnfNode", func() {
		var resHub *nnfv1alpha10.NnfNode

		BeforeEach(func() {
			id := uuid.NewString()[0:8]
			resHub = &nnfv1alpha10.NnfNode{
				ObjectMeta: metav1.ObjectMeta{
					Name:      id,
					Namespace: corev1.NamespaceDefault,
				},
				Spec: nnfv1alpha10.NnfNodeSpec{
					State: "Enable",
				},
			}

			Expect(k8sClient.Create(context.TODO(), resHub)).To(Succeed())
		})

		AfterEach(func() {
			if resHub != nil {
				Expect(k8sClient.Delete(context.TODO(), resHub)).To(Succeed())
				expected := &nnfv1alpha10.NnfNode{}
				Eventually(func() error { // Delete can still return the cached object. Wait until the object is no longer present.
					return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(resHub), expected)
				}).ShouldNot(Succeed())
			}
		})

		It("is unable to read NnfNode resource via spoke v1alpha8", func() {
			resSpoke := &nnfv1alpha8.NnfNode{}
			Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(resHub), resSpoke)).ToNot(Succeed())
		})

		It("reads NnfNode resource via hub and via spoke v1alpha9", func() {
			// Spoke should have annotation.
			resSpoke := &nnfv1alpha9.NnfNode{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(resHub), resSpoke)).To(Succeed())
				anno := resSpoke.GetAnnotations()
				g.Expect(anno).To(HaveLen(1))
				g.Expect(anno).Should(HaveKey(utilconversion.DataAnnotation))
			}).Should(Succeed())

			// Hub should not have annotation.
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(resHub), resHub)).To(Succeed())
				anno := resHub.GetAnnotations()
				g.Expect(anno).To(HaveLen(0))
			}).Should(Succeed())
		})

		// +crdbumper:scaffold:spoketest="nnf.NnfNode"
	})

	Context("NnfNodeBlockStorage", func() {
		var resHub *nnfv1alpha10.NnfNodeBlockStorage

		BeforeEach(func() {
			id := uuid.NewString()[0:8]
			resHub = &nnfv1alpha10.NnfNodeBlockStorage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      id,
					Namespace: corev1.NamespaceDefault,
				},
				Spec: nnfv1alpha10.NnfNodeBlockStorageSpec{},
			}

			Expect(k8sClient.Create(context.TODO(), resHub)).To(Succeed())
		})

		AfterEach(func() {
			if resHub != nil {
				Expect(k8sClient.Delete(context.TODO(), resHub)).To(Succeed())
				expected := &nnfv1alpha10.NnfNodeBlockStorage{}
				Eventually(func() error { // Delete can still return the cached object. Wait until the object is no longer present.
					return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(resHub), expected)
				}).ShouldNot(Succeed())
			}
		})

		It("is unable to read NnfNodeBlockStorage resource via spoke v1alpha8", func() {
			resSpoke := &nnfv1alpha8.NnfNodeBlockStorage{}
			Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(resHub), resSpoke)).ToNot(Succeed())
		})

		It("reads NnfNodeBlockStorage resource via hub and via spoke v1alpha9", func() {
			// Spoke should have annotation.
			resSpoke := &nnfv1alpha9.NnfNodeBlockStorage{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(resHub), resSpoke)).To(Succeed())
				anno := resSpoke.GetAnnotations()
				g.Expect(anno).To(HaveLen(1))
				g.Expect(anno).Should(HaveKey(utilconversion.DataAnnotation))
			}).Should(Succeed())

			// Hub should not have annotation.
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(resHub), resHub)).To(Succeed())
				anno := resHub.GetAnnotations()
				g.Expect(anno).To(HaveLen(0))
			}).Should(Succeed())
		})

		// +crdbumper:scaffold:spoketest="nnf.NnfNodeBlockStorage"
	})

	Context("NnfNodeECData", func() {
		var resHub *nnfv1alpha10.NnfNodeECData

		BeforeEach(func() {
			id := uuid.NewString()[0:8]
			resHub = &nnfv1alpha10.NnfNodeECData{
				ObjectMeta: metav1.ObjectMeta{
					Name:      id,
					Namespace: corev1.NamespaceDefault,
				},
				Spec: nnfv1alpha10.NnfNodeECDataSpec{},
			}

			Expect(k8sClient.Create(context.TODO(), resHub)).To(Succeed())
		})

		AfterEach(func() {
			if resHub != nil {
				Expect(k8sClient.Delete(context.TODO(), resHub)).To(Succeed())
				expected := &nnfv1alpha10.NnfNodeECData{}
				Eventually(func() error { // Delete can still return the cached object. Wait until the object is no longer present.
					return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(resHub), expected)
				}).ShouldNot(Succeed())
			}
		})

		It("is unable to read NnfNodeECData resource via spoke v1alpha8", func() {
			resSpoke := &nnfv1alpha8.NnfNodeECData{}
			Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(resHub), resSpoke)).ToNot(Succeed())
		})

		It("reads NnfNodeECData resource via hub and via spoke v1alpha9", func() {
			// Spoke should have annotation.
			resSpoke := &nnfv1alpha9.NnfNodeECData{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(resHub), resSpoke)).To(Succeed())
				anno := resSpoke.GetAnnotations()
				g.Expect(anno).To(HaveLen(1))
				g.Expect(anno).Should(HaveKey(utilconversion.DataAnnotation))
			}).Should(Succeed())

			// Hub should not have annotation.
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(resHub), resHub)).To(Succeed())
				anno := resHub.GetAnnotations()
				g.Expect(anno).To(HaveLen(0))
			}).Should(Succeed())
		})

		// +crdbumper:scaffold:spoketest="nnf.NnfNodeECData"
	})

	Context("NnfNodeStorage", func() {
		var resHub *nnfv1alpha10.NnfNodeStorage

		BeforeEach(func() {
			id := uuid.NewString()[0:8]
			resHub = &nnfv1alpha10.NnfNodeStorage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      id,
					Namespace: corev1.NamespaceDefault,
				},
				Spec: nnfv1alpha10.NnfNodeStorageSpec{},
			}

			Expect(k8sClient.Create(context.TODO(), resHub)).To(Succeed())
		})

		AfterEach(func() {
			if resHub != nil {
				Expect(k8sClient.Delete(context.TODO(), resHub)).To(Succeed())
				expected := &nnfv1alpha10.NnfNodeStorage{}
				Eventually(func() error { // Delete can still return the cached object. Wait until the object is no longer present.
					return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(resHub), expected)
				}).ShouldNot(Succeed())
			}
		})

		It("is unable to read NnfNodeStorage resource via spoke v1alpha8", func() {
			resSpoke := &nnfv1alpha8.NnfNodeStorage{}
			Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(resHub), resSpoke)).ToNot(Succeed())
		})

		It("reads NnfNodeStorage resource via hub and via spoke v1alpha9", func() {
			// Spoke should have annotation.
			resSpoke := &nnfv1alpha9.NnfNodeStorage{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(resHub), resSpoke)).To(Succeed())
				anno := resSpoke.GetAnnotations()
				g.Expect(anno).To(HaveLen(1))
				g.Expect(anno).Should(HaveKey(utilconversion.DataAnnotation))
			}).Should(Succeed())

			// Hub should not have annotation.
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(resHub), resHub)).To(Succeed())
				anno := resHub.GetAnnotations()
				g.Expect(anno).To(HaveLen(0))
			}).Should(Succeed())
		})

		// +crdbumper:scaffold:spoketest="nnf.NnfNodeStorage"
	})

	Context("NnfPortManager", func() {
		var resHub *nnfv1alpha10.NnfPortManager

		BeforeEach(func() {
			id := uuid.NewString()[0:8]
			resHub = &nnfv1alpha10.NnfPortManager{
				ObjectMeta: metav1.ObjectMeta{
					Name:      id,
					Namespace: corev1.NamespaceDefault,
				},
				Spec: nnfv1alpha10.NnfPortManagerSpec{
					Allocations: make([]nnfv1alpha10.NnfPortManagerAllocationSpec, 0),
				},
			}

			Expect(k8sClient.Create(context.TODO(), resHub)).To(Succeed())
		})

		AfterEach(func() {
			if resHub != nil {
				Expect(k8sClient.Delete(context.TODO(), resHub)).To(Succeed())
				expected := &nnfv1alpha10.NnfPortManager{}
				Eventually(func() error { // Delete can still return the cached object. Wait until the object is no longer present.
					return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(resHub), expected)
				}).ShouldNot(Succeed())
			}
		})

		It("is unable to read NnfPortManager resource via spoke v1alpha8", func() {
			resSpoke := &nnfv1alpha8.NnfPortManager{}
			Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(resHub), resSpoke)).ToNot(Succeed())
		})

		It("reads NnfPortManager resource via hub and via spoke v1alpha9", func() {
			// Spoke should have annotation.
			resSpoke := &nnfv1alpha9.NnfPortManager{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(resHub), resSpoke)).To(Succeed())
				anno := resSpoke.GetAnnotations()
				g.Expect(anno).To(HaveLen(1))
				g.Expect(anno).Should(HaveKey(utilconversion.DataAnnotation))
			}).Should(Succeed())

			// Hub should not have annotation.
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(resHub), resHub)).To(Succeed())
				anno := resHub.GetAnnotations()
				g.Expect(anno).To(HaveLen(0))
			}).Should(Succeed())
		})

		// +crdbumper:scaffold:spoketest="nnf.NnfPortManager"
	})

	Context("NnfStorage", func() {
		var resHub *nnfv1alpha10.NnfStorage

		BeforeEach(func() {
			id := uuid.NewString()[0:8]
			resHub = &nnfv1alpha10.NnfStorage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      id,
					Namespace: corev1.NamespaceDefault,
				},
				Spec: nnfv1alpha10.NnfStorageSpec{
					AllocationSets: []nnfv1alpha10.NnfStorageAllocationSetSpec{},
				},
			}

			Expect(k8sClient.Create(context.TODO(), resHub)).To(Succeed())
		})

		AfterEach(func() {
			if resHub != nil {
				Expect(k8sClient.Delete(context.TODO(), resHub)).To(Succeed())
				expected := &nnfv1alpha10.NnfStorage{}
				Eventually(func() error { // Delete can still return the cached object. Wait until the object is no longer present.
					return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(resHub), expected)
				}).ShouldNot(Succeed())
			}
		})

		It("is unable to read NnfStorage resource via spoke v1alpha8", func() {
			resSpoke := &nnfv1alpha8.NnfStorage{}
			Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(resHub), resSpoke)).ToNot(Succeed())
		})

		It("reads NnfStorage resource via hub and via spoke v1alpha9", func() {
			// Spoke should have annotation.
			resSpoke := &nnfv1alpha9.NnfStorage{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(resHub), resSpoke)).To(Succeed())
				anno := resSpoke.GetAnnotations()
				g.Expect(anno).To(HaveLen(1))
				g.Expect(anno).Should(HaveKey(utilconversion.DataAnnotation))
			}).Should(Succeed())

			// Hub should not have annotation.
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(resHub), resHub)).To(Succeed())
				anno := resHub.GetAnnotations()
				g.Expect(anno).To(HaveLen(0))
			}).Should(Succeed())
		})

		// +crdbumper:scaffold:spoketest="nnf.NnfStorage"
	})

	Context("NnfStorageProfile", func() {
		var resHub *nnfv1alpha10.NnfStorageProfile

		BeforeEach(func() {
			id := uuid.NewString()[0:8]
			resHub = &nnfv1alpha10.NnfStorageProfile{
				ObjectMeta: metav1.ObjectMeta{
					Name:      id,
					Namespace: corev1.NamespaceDefault,
				},
				Data: nnfv1alpha10.NnfStorageProfileData{},
			}

			Expect(k8sClient.Create(context.TODO(), resHub)).To(Succeed())
		})

		AfterEach(func() {
			if resHub != nil {
				Expect(k8sClient.Delete(context.TODO(), resHub)).To(Succeed())
				expected := &nnfv1alpha10.NnfStorageProfile{}
				Eventually(func() error { // Delete can still return the cached object. Wait until the object is no longer present.
					return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(resHub), expected)
				}).ShouldNot(Succeed())
			}
		})

		It("is unable to read NnfStorageProfile resource via spoke v1alpha8", func() {
			resSpoke := &nnfv1alpha8.NnfStorageProfile{}
			Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(resHub), resSpoke)).ToNot(Succeed())
		})

		It("reads NnfStorageProfile resource via hub and via spoke v1alpha9", func() {
			// Spoke should have annotation.
			resSpoke := &nnfv1alpha9.NnfStorageProfile{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(resHub), resSpoke)).To(Succeed())
				anno := resSpoke.GetAnnotations()
				g.Expect(anno).To(HaveLen(1))
				g.Expect(anno).Should(HaveKey(utilconversion.DataAnnotation))
			}).Should(Succeed())

			// Hub should not have annotation.
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(resHub), resHub)).To(Succeed())
				anno := resHub.GetAnnotations()
				g.Expect(anno).To(HaveLen(0))
			}).Should(Succeed())
		})

		// +crdbumper:scaffold:spoketest="nnf.NnfStorageProfile"
	})

	Context("NnfSystemStorage", func() {
		var resHub *nnfv1alpha10.NnfSystemStorage

		BeforeEach(func() {
			id := uuid.NewString()[0:8]
			resHub = &nnfv1alpha10.NnfSystemStorage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      id,
					Namespace: corev1.NamespaceDefault,
				},
				Spec: nnfv1alpha10.NnfSystemStorageSpec{},
			}

			Expect(k8sClient.Create(context.TODO(), resHub)).To(Succeed())
		})

		AfterEach(func() {
			if resHub != nil {
				Expect(k8sClient.Delete(context.TODO(), resHub)).To(Succeed())
				expected := &nnfv1alpha10.NnfSystemStorage{}
				Eventually(func() error { // Delete can still return the cached object. Wait until the object is no longer present.
					return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(resHub), expected)
				}).ShouldNot(Succeed())
			}
		})

		It("is unable to read NnfSystemStorage resource via spoke v1alpha8", func() {
			resSpoke := &nnfv1alpha8.NnfSystemStorage{}
			Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(resHub), resSpoke)).ToNot(Succeed())
		})

		It("reads NnfSystemStorage resource via hub and via spoke v1alpha9", func() {
			// Spoke should have annotation.
			resSpoke := &nnfv1alpha9.NnfSystemStorage{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(resHub), resSpoke)).To(Succeed())
				anno := resSpoke.GetAnnotations()
				g.Expect(anno).To(HaveLen(1))
				g.Expect(anno).Should(HaveKey(utilconversion.DataAnnotation))
			}).Should(Succeed())

			// Hub should not have annotation.
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(resHub), resHub)).To(Succeed())
				anno := resHub.GetAnnotations()
				g.Expect(anno).To(HaveLen(0))
			}).Should(Succeed())
		})

		// +crdbumper:scaffold:spoketest="nnf.NnfSystemStorage"
	})

	// +crdbumper:scaffold:webhooksuitetest
})
