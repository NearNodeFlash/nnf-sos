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
	"reflect"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	nnfv1alpha2 "github.com/NearNodeFlash/nnf-sos/api/v1alpha2"
)

var _ = Describe("NnfLustreMGT Controller Test", func() {
	It("Verifies a single fsname consumer", func() {
		nnfLustreMgt := &nnfv1alpha2.NnfLustreMGT{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-mgt",
				Namespace: corev1.NamespaceDefault,
			},
			Spec: nnfv1alpha2.NnfLustreMGTSpec{
				Addresses:   []string{"1.1.1.1@tcp"},
				FsNameStart: "bbbbbbbb",
			},
		}
		Expect(k8sClient.Create(context.TODO(), nnfLustreMgt)).To(Succeed())

		By("Verify that the Status.FsNameNext field gets set from the value in Spec.FsNameStart")
		Eventually(func(g Gomega) bool {
			g.Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(nnfLustreMgt), nnfLustreMgt)).To(Succeed())
			return nnfLustreMgt.Status.FsNameNext != nnfLustreMgt.Spec.FsNameStart
		}).Should(BeTrue())

		By("Add a claim to the NnfLustreMGT")
		Eventually(func(g Gomega) error {
			g.Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(nnfLustreMgt), nnfLustreMgt)).To(Succeed())
			nnfLustreMgt.Spec.ClaimList = append(nnfLustreMgt.Spec.ClaimList, corev1.ObjectReference{Name: "fake-claim", Namespace: "fake-namespace"})
			return k8sClient.Update(context.TODO(), nnfLustreMgt)
		}).Should(Succeed())

		By("Verify that an fsname is assigned")
		Eventually(func(g Gomega) bool {
			g.Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(nnfLustreMgt), nnfLustreMgt)).To(Succeed())
			if len(nnfLustreMgt.Status.ClaimList) != 1 {
				return false
			}

			if nnfLustreMgt.Status.ClaimList[0].Reference.Name != "fake-claim" || nnfLustreMgt.Status.ClaimList[0].Reference.Namespace != "fake-namespace" {
				return false
			}

			if nnfLustreMgt.Status.ClaimList[0].FsName != "bbbbbbbb" {
				return false
			}
			return true
		}).Should(BeTrue())

		By("Deleting the NnfLustreMGT")
		Expect(k8sClient.Delete(context.TODO(), nnfLustreMgt)).To(Succeed())

		Eventually(func() error {
			return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(nnfLustreMgt), nnfLustreMgt)
		}).ShouldNot(Succeed())
	})

	It("Verifies two fsname consumers with fsname wrap", func() {
		nnfLustreMgt := &nnfv1alpha2.NnfLustreMGT{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-mgt",
				Namespace: corev1.NamespaceDefault,
			},
			Spec: nnfv1alpha2.NnfLustreMGTSpec{
				Addresses:   []string{"1.1.1.1@tcp"},
				FsNameStart: "zzzzzzzz",
			},
		}
		Expect(k8sClient.Create(context.TODO(), nnfLustreMgt)).To(Succeed())

		By("Verify that the Status.FsNameNext field gets set from the value in Spec.FsNameStart")
		Eventually(func(g Gomega) bool {
			g.Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(nnfLustreMgt), nnfLustreMgt)).To(Succeed())
			return nnfLustreMgt.Status.FsNameNext != nnfLustreMgt.Spec.FsNameStart
		}).Should(BeTrue())

		By("Add a claim to the NnfLustreMGT")
		Eventually(func(g Gomega) error {
			g.Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(nnfLustreMgt), nnfLustreMgt)).To(Succeed())
			nnfLustreMgt.Spec.ClaimList = append(nnfLustreMgt.Spec.ClaimList, corev1.ObjectReference{Name: "fake-claim1", Namespace: "fake-namespace1"})
			return k8sClient.Update(context.TODO(), nnfLustreMgt)
		}).Should(Succeed())

		By("Add another claim to the NnfLustreMGT")
		Eventually(func(g Gomega) error {
			g.Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(nnfLustreMgt), nnfLustreMgt)).To(Succeed())
			nnfLustreMgt.Spec.ClaimList = append(nnfLustreMgt.Spec.ClaimList, corev1.ObjectReference{Name: "fake-claim2", Namespace: "fake-namespace2"})
			return k8sClient.Update(context.TODO(), nnfLustreMgt)
		}).Should(Succeed())

		By("Verify that both fsnames are assigned")
		Eventually(func(g Gomega) bool {
			g.Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(nnfLustreMgt), nnfLustreMgt)).To(Succeed())
			if len(nnfLustreMgt.Status.ClaimList) != 2 {
				return false
			}

			if nnfLustreMgt.Status.ClaimList[0].Reference.Name != "fake-claim1" || nnfLustreMgt.Status.ClaimList[0].Reference.Namespace != "fake-namespace1" {
				return false
			}

			if nnfLustreMgt.Status.ClaimList[1].Reference.Name != "fake-claim2" || nnfLustreMgt.Status.ClaimList[1].Reference.Namespace != "fake-namespace2" {
				return false
			}

			if nnfLustreMgt.Status.ClaimList[0].FsName != "zzzzzzzz" {
				return false
			}

			if nnfLustreMgt.Status.ClaimList[1].FsName != "aaaaaaaa" {
				return false
			}
			return true
		}).Should(BeTrue())

		By("Deleting the NnfLustreMGT")
		Expect(k8sClient.Delete(context.TODO(), nnfLustreMgt)).To(Succeed())

		Eventually(func() error {
			return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(nnfLustreMgt), nnfLustreMgt)
		}).ShouldNot(Succeed())
	})

	It("Verifies a FsNameStart is set appropriately when a configmap is used", func() {
		configMap := &corev1.ConfigMap{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-mgt",
				Namespace: corev1.NamespaceDefault,
			},
			Data: map[string]string{"NextFsName": "abcdefgh"},
		}
		Expect(k8sClient.Create(context.TODO(), configMap)).To(Succeed())

		nnfLustreMgt := &nnfv1alpha2.NnfLustreMGT{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-mgt",
				Namespace: corev1.NamespaceDefault,
			},
			Spec: nnfv1alpha2.NnfLustreMGTSpec{
				Addresses:   []string{"1.1.1.1@tcp"},
				FsNameStart: "bbbbbbbb",
				FsNameStartReference: corev1.ObjectReference{
					Name:      "test-mgt",
					Namespace: corev1.NamespaceDefault,
					Kind:      reflect.TypeOf(corev1.ConfigMap{}).Name(),
				},
			},
		}
		Expect(k8sClient.Create(context.TODO(), nnfLustreMgt)).To(Succeed())

		By("Verify that the Status.FsNameNext field gets set from the value in Spec.FsNameStart")
		Eventually(func(g Gomega) bool {
			g.Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(nnfLustreMgt), nnfLustreMgt)).To(Succeed())
			return nnfLustreMgt.Status.FsNameNext != "abcdefgh"
		}).Should(BeTrue())

		By("Deleting the NnfLustreMGT")
		Expect(k8sClient.Delete(context.TODO(), nnfLustreMgt)).To(Succeed())

		Eventually(func() error {
			return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(nnfLustreMgt), nnfLustreMgt)
		}).ShouldNot(Succeed())
	})
})
