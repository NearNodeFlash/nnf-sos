/*
 * Copyright 2024 Hewlett Packard Enterprise Development LP
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

	nnfv1alpha1 "github.com/NearNodeFlash/nnf-sos/api/v1alpha1"
	nnfv1alpha2 "github.com/NearNodeFlash/nnf-sos/api/v1alpha2"
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
		var resHub *nnfv1alpha2.NnfAccess

		BeforeEach(func() {
			id := uuid.NewString()[0:8]
			resHub = &nnfv1alpha2.NnfAccess{
				ObjectMeta: metav1.ObjectMeta{
					Name:      id,
					Namespace: corev1.NamespaceDefault,
				},
				Spec: nnfv1alpha2.NnfAccessSpec{
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
				expected := &nnfv1alpha2.NnfAccess{}
				Eventually(func() error { // Delete can still return the cached object. Wait until the object is no longer present.
					return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(resHub), expected)
				}).ShouldNot(Succeed())
			}
		})

		It("reads NnfAccess resource via hub and via spoke v1alpha1", func() {
			// Spoke should have annotation.
			resSpoke := &nnfv1alpha1.NnfAccess{}
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
		var resHub *nnfv1alpha2.NnfContainerProfile

		BeforeEach(func() {
			id := uuid.NewString()[0:8]
			resHub = &nnfv1alpha2.NnfContainerProfile{
				ObjectMeta: metav1.ObjectMeta{
					Name:      id,
					Namespace: corev1.NamespaceDefault,
				},
				Data: nnfv1alpha2.NnfContainerProfileData{
					Spec: &corev1.PodSpec{
						NodeName:   "rabbit-1",
						Containers: []corev1.Container{{Name: "one"}},
					},
				},
			}

			Expect(k8sClient.Create(context.TODO(), resHub)).To(Succeed())
		})

		AfterEach(func() {
			if resHub != nil {
				Expect(k8sClient.Delete(context.TODO(), resHub)).To(Succeed())
				expected := &nnfv1alpha2.NnfContainerProfile{}
				Eventually(func() error { // Delete can still return the cached object. Wait until the object is no longer present.
					return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(resHub), expected)
				}).ShouldNot(Succeed())
			}
		})

		It("reads NnfContainerProfile resource via hub and via spoke v1alpha1", func() {
			// Spoke should have annotation.
			resSpoke := &nnfv1alpha1.NnfContainerProfile{}
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
		var resHub *nnfv1alpha2.NnfDataMovement

		BeforeEach(func() {
			id := uuid.NewString()[0:8]
			resHub = &nnfv1alpha2.NnfDataMovement{
				ObjectMeta: metav1.ObjectMeta{
					Name:      id,
					Namespace: corev1.NamespaceDefault,
				},
				Spec: nnfv1alpha2.NnfDataMovementSpec{},
			}

			Expect(k8sClient.Create(context.TODO(), resHub)).To(Succeed())
		})

		AfterEach(func() {
			if resHub != nil {
				Expect(k8sClient.Delete(context.TODO(), resHub)).To(Succeed())
				expected := &nnfv1alpha2.NnfDataMovement{}
				Eventually(func() error { // Delete can still return the cached object. Wait until the object is no longer present.
					return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(resHub), expected)
				}).ShouldNot(Succeed())
			}
		})

		It("reads NnfDataMovement resource via hub and via spoke v1alpha1", func() {
			// Spoke should have annotation.
			resSpoke := &nnfv1alpha1.NnfDataMovement{}
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
		var resHub *nnfv1alpha2.NnfDataMovementManager

		BeforeEach(func() {
			id := uuid.NewString()[0:8]
			resHub = &nnfv1alpha2.NnfDataMovementManager{
				ObjectMeta: metav1.ObjectMeta{
					Name:      id,
					Namespace: corev1.NamespaceDefault,
				},
				Spec: nnfv1alpha2.NnfDataMovementManagerSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{{
								Name:  "dm-worker-dummy",
								Image: "nginx",
							}},
						},
					},
				},
			}

			Expect(k8sClient.Create(context.TODO(), resHub)).To(Succeed())
		})

		AfterEach(func() {
			if resHub != nil {
				Expect(k8sClient.Delete(context.TODO(), resHub)).To(Succeed())
				expected := &nnfv1alpha2.NnfDataMovementManager{}
				Eventually(func() error { // Delete can still return the cached object. Wait until the object is no longer present.
					return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(resHub), expected)
				}).ShouldNot(Succeed())
			}
		})

		It("reads NnfDataMovementManager resource via hub and via spoke v1alpha1", func() {
			// Spoke should have annotation.
			resSpoke := &nnfv1alpha1.NnfDataMovementManager{}
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
		var resHub *nnfv1alpha2.NnfDataMovementProfile

		BeforeEach(func() {
			id := uuid.NewString()[0:8]
			resHub = &nnfv1alpha2.NnfDataMovementProfile{
				ObjectMeta: metav1.ObjectMeta{
					Name:      id,
					Namespace: corev1.NamespaceDefault,
				},
				Data: nnfv1alpha2.NnfDataMovementProfileData{},
			}

			Expect(k8sClient.Create(context.TODO(), resHub)).To(Succeed())
		})

		AfterEach(func() {
			if resHub != nil {
				Expect(k8sClient.Delete(context.TODO(), resHub)).To(Succeed())
				expected := &nnfv1alpha2.NnfDataMovementProfile{}
				Eventually(func() error { // Delete can still return the cached object. Wait until the object is no longer present.
					return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(resHub), expected)
				}).ShouldNot(Succeed())
			}
		})

		It("reads NnfDataMovementProfile resource via hub and via spoke v1alpha1", func() {
			// Spoke should have annotation.
			resSpoke := &nnfv1alpha1.NnfDataMovementProfile{}
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
		var resHub *nnfv1alpha2.NnfLustreMGT

		BeforeEach(func() {
			id := uuid.NewString()[0:8]
			resHub = &nnfv1alpha2.NnfLustreMGT{
				ObjectMeta: metav1.ObjectMeta{
					Name:      id,
					Namespace: corev1.NamespaceDefault,
				},
				Spec: nnfv1alpha2.NnfLustreMGTSpec{
					Addresses: []string{"rabbit-1@tcp", "rabbit-2@tcp"},
				},
			}

			Expect(k8sClient.Create(context.TODO(), resHub)).To(Succeed())
		})

		AfterEach(func() {
			if resHub != nil {
				Expect(k8sClient.Delete(context.TODO(), resHub)).To(Succeed())
				expected := &nnfv1alpha2.NnfLustreMGT{}
				Eventually(func() error { // Delete can still return the cached object. Wait until the object is no longer present.
					return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(resHub), expected)
				}).ShouldNot(Succeed())
			}
		})

		It("reads NnfLustreMGT resource via hub and via spoke v1alpha1", func() {
			// Spoke should have annotation.
			resSpoke := &nnfv1alpha1.NnfLustreMGT{}
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
		var resHub *nnfv1alpha2.NnfNode

		BeforeEach(func() {
			id := uuid.NewString()[0:8]
			resHub = &nnfv1alpha2.NnfNode{
				ObjectMeta: metav1.ObjectMeta{
					Name:      id,
					Namespace: corev1.NamespaceDefault,
				},
				Spec: nnfv1alpha2.NnfNodeSpec{
					State: "Enable",
				},
			}

			Expect(k8sClient.Create(context.TODO(), resHub)).To(Succeed())
		})

		AfterEach(func() {
			if resHub != nil {
				Expect(k8sClient.Delete(context.TODO(), resHub)).To(Succeed())
				expected := &nnfv1alpha2.NnfNode{}
				Eventually(func() error { // Delete can still return the cached object. Wait until the object is no longer present.
					return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(resHub), expected)
				}).ShouldNot(Succeed())
			}
		})

		It("reads NnfNode resource via hub and via spoke v1alpha1", func() {
			// Spoke should have annotation.
			resSpoke := &nnfv1alpha1.NnfNode{}
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
		var resHub *nnfv1alpha2.NnfNodeBlockStorage

		BeforeEach(func() {
			id := uuid.NewString()[0:8]
			resHub = &nnfv1alpha2.NnfNodeBlockStorage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      id,
					Namespace: corev1.NamespaceDefault,
				},
				Spec: nnfv1alpha2.NnfNodeBlockStorageSpec{},
			}

			Expect(k8sClient.Create(context.TODO(), resHub)).To(Succeed())
		})

		AfterEach(func() {
			if resHub != nil {
				Expect(k8sClient.Delete(context.TODO(), resHub)).To(Succeed())
				expected := &nnfv1alpha2.NnfNodeBlockStorage{}
				Eventually(func() error { // Delete can still return the cached object. Wait until the object is no longer present.
					return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(resHub), expected)
				}).ShouldNot(Succeed())
			}
		})

		It("reads NnfNodeBlockStorage resource via hub and via spoke v1alpha1", func() {
			// Spoke should have annotation.
			resSpoke := &nnfv1alpha1.NnfNodeBlockStorage{}
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
		var resHub *nnfv1alpha2.NnfNodeECData

		BeforeEach(func() {
			id := uuid.NewString()[0:8]
			resHub = &nnfv1alpha2.NnfNodeECData{
				ObjectMeta: metav1.ObjectMeta{
					Name:      id,
					Namespace: corev1.NamespaceDefault,
				},
				Spec: nnfv1alpha2.NnfNodeECDataSpec{},
			}

			Expect(k8sClient.Create(context.TODO(), resHub)).To(Succeed())
		})

		AfterEach(func() {
			if resHub != nil {
				Expect(k8sClient.Delete(context.TODO(), resHub)).To(Succeed())
				expected := &nnfv1alpha2.NnfNodeECData{}
				Eventually(func() error { // Delete can still return the cached object. Wait until the object is no longer present.
					return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(resHub), expected)
				}).ShouldNot(Succeed())
			}
		})

		It("reads NnfNodeECData resource via hub and via spoke v1alpha1", func() {
			// Spoke should have annotation.
			resSpoke := &nnfv1alpha1.NnfNodeECData{}
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
		var resHub *nnfv1alpha2.NnfNodeStorage

		BeforeEach(func() {
			id := uuid.NewString()[0:8]
			resHub = &nnfv1alpha2.NnfNodeStorage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      id,
					Namespace: corev1.NamespaceDefault,
				},
				Spec: nnfv1alpha2.NnfNodeStorageSpec{},
			}

			Expect(k8sClient.Create(context.TODO(), resHub)).To(Succeed())
		})

		AfterEach(func() {
			if resHub != nil {
				Expect(k8sClient.Delete(context.TODO(), resHub)).To(Succeed())
				expected := &nnfv1alpha2.NnfNodeStorage{}
				Eventually(func() error { // Delete can still return the cached object. Wait until the object is no longer present.
					return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(resHub), expected)
				}).ShouldNot(Succeed())
			}
		})

		It("reads NnfNodeStorage resource via hub and via spoke v1alpha1", func() {
			// Spoke should have annotation.
			resSpoke := &nnfv1alpha1.NnfNodeStorage{}
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
		var resHub *nnfv1alpha2.NnfPortManager

		BeforeEach(func() {
			id := uuid.NewString()[0:8]
			resHub = &nnfv1alpha2.NnfPortManager{
				ObjectMeta: metav1.ObjectMeta{
					Name:      id,
					Namespace: corev1.NamespaceDefault,
				},
				Spec: nnfv1alpha2.NnfPortManagerSpec{
					Allocations: make([]nnfv1alpha2.NnfPortManagerAllocationSpec, 0),
				},
			}

			Expect(k8sClient.Create(context.TODO(), resHub)).To(Succeed())
		})

		AfterEach(func() {
			if resHub != nil {
				Expect(k8sClient.Delete(context.TODO(), resHub)).To(Succeed())
				expected := &nnfv1alpha2.NnfPortManager{}
				Eventually(func() error { // Delete can still return the cached object. Wait until the object is no longer present.
					return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(resHub), expected)
				}).ShouldNot(Succeed())
			}
		})

		It("reads NnfPortManager resource via hub and via spoke v1alpha1", func() {
			// Spoke should have annotation.
			resSpoke := &nnfv1alpha1.NnfPortManager{}
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
		var resHub *nnfv1alpha2.NnfStorage

		BeforeEach(func() {
			id := uuid.NewString()[0:8]
			resHub = &nnfv1alpha2.NnfStorage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      id,
					Namespace: corev1.NamespaceDefault,
				},
				Spec: nnfv1alpha2.NnfStorageSpec{
					AllocationSets: []nnfv1alpha2.NnfStorageAllocationSetSpec{},
				},
			}

			Expect(k8sClient.Create(context.TODO(), resHub)).To(Succeed())
		})

		AfterEach(func() {
			if resHub != nil {
				Expect(k8sClient.Delete(context.TODO(), resHub)).To(Succeed())
				expected := &nnfv1alpha2.NnfStorage{}
				Eventually(func() error { // Delete can still return the cached object. Wait until the object is no longer present.
					return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(resHub), expected)
				}).ShouldNot(Succeed())
			}
		})

		It("reads NnfStorage resource via hub and via spoke v1alpha1", func() {
			// Spoke should have annotation.
			resSpoke := &nnfv1alpha1.NnfStorage{}
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
		var resHub *nnfv1alpha2.NnfStorageProfile

		BeforeEach(func() {
			id := uuid.NewString()[0:8]
			resHub = &nnfv1alpha2.NnfStorageProfile{
				ObjectMeta: metav1.ObjectMeta{
					Name:      id,
					Namespace: corev1.NamespaceDefault,
				},
				Data: nnfv1alpha2.NnfStorageProfileData{},
			}

			Expect(k8sClient.Create(context.TODO(), resHub)).To(Succeed())
		})

		AfterEach(func() {
			if resHub != nil {
				Expect(k8sClient.Delete(context.TODO(), resHub)).To(Succeed())
				expected := &nnfv1alpha2.NnfStorageProfile{}
				Eventually(func() error { // Delete can still return the cached object. Wait until the object is no longer present.
					return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(resHub), expected)
				}).ShouldNot(Succeed())
			}
		})

		It("reads NnfStorageProfile resource via hub and via spoke v1alpha1", func() {
			// Spoke should have annotation.
			resSpoke := &nnfv1alpha1.NnfStorageProfile{}
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
		var resHub *nnfv1alpha2.NnfSystemStorage

		BeforeEach(func() {
			id := uuid.NewString()[0:8]
			resHub = &nnfv1alpha2.NnfSystemStorage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      id,
					Namespace: corev1.NamespaceDefault,
				},
				Spec: nnfv1alpha2.NnfSystemStorageSpec{},
			}

			Expect(k8sClient.Create(context.TODO(), resHub)).To(Succeed())
		})

		AfterEach(func() {
			if resHub != nil {
				Expect(k8sClient.Delete(context.TODO(), resHub)).To(Succeed())
				expected := &nnfv1alpha2.NnfSystemStorage{}
				Eventually(func() error { // Delete can still return the cached object. Wait until the object is no longer present.
					return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(resHub), expected)
				}).ShouldNot(Succeed())
			}
		})

		It("reads NnfSystemStorage resource via hub and via spoke v1alpha1", func() {
			// Spoke should have annotation.
			resSpoke := &nnfv1alpha1.NnfSystemStorage{}
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
