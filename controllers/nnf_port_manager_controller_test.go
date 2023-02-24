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

package controllers

import (
	"fmt"
	"reflect"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	dwsv1alpha1 "github.com/HewlettPackard/dws/api/v1alpha1"
	nnfv1alpha1 "github.com/NearNodeFlash/nnf-sos/api/v1alpha1"
)

var _ = Context("NNF Port Manager Controller Setup", Ordered, func() {

	var cfg *dwsv1alpha1.SystemConfiguration
	const portStart = 20
	const portEnd = 29

	BeforeAll(func() {
		cfg = &dwsv1alpha1.SystemConfiguration{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "port-manager-system-config",
				Namespace: corev1.NamespaceDefault,
			},
			Spec: dwsv1alpha1.SystemConfigurationSpec{
				Ports: []intstr.IntOrString{
					intstr.FromString(fmt.Sprintf("%d-%d", portStart, portEnd)),
				},
			},
		}

		Expect(k8sClient.Create(ctx, cfg)).To(Succeed())
		DeferCleanup(func() { Expect(k8sClient.Delete(ctx, cfg)).To(Succeed()) })
	})

	Describe("NNF Port Manager Controller Test", func() {

		var mgr *nnfv1alpha1.NnfPortManager
		var r = &NnfPortManagerReconciler{} // use this to access private reconciler methods

		BeforeEach(func() {
			mgr = &nnfv1alpha1.NnfPortManager{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "nnf-port-manager",
					Namespace: corev1.NamespaceDefault,
				},
				Spec: nnfv1alpha1.NnfPortManagerSpec{
					SystemConfiguration: corev1.ObjectReference{
						Name:      cfg.Name,
						Namespace: cfg.Namespace,
						Kind:      reflect.TypeOf(*cfg).Name(),
					},
					Allocations: make([]nnfv1alpha1.NnfPortManagerAllocationSpec, 0),
				},
			}
		})

		JustBeforeEach(func() {
			Expect(k8sClient.Create(ctx, mgr)).To(Succeed())
			DeferCleanup(func() { Expect(k8sClient.Delete(ctx, mgr)).To(Succeed()) })
		})

		reservePorts := func(mgr *nnfv1alpha1.NnfPortManager, name string, count int) {
			By(fmt.Sprintf("Reserving %d ports for '%s'", count, name))

			allocation := nnfv1alpha1.NnfPortManagerAllocationSpec{
				Requester: corev1.ObjectReference{Name: name},
				Count:     count,
			}

			Eventually(func(g Gomega) error {
				g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(mgr), mgr)).To(Succeed())

				mgr.Spec.Allocations = append(mgr.Spec.Allocations, allocation)

				return k8sClient.Update(ctx, mgr)
			}).Should(Succeed())

			Eventually(func() bool {
				Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(mgr), mgr)).To(Succeed())
				return r.isAllocated(mgr, allocation)
			}).Should(BeTrue())

			status := r.findAllocationStatus(mgr, allocation)
			Expect(status).ToNot(BeNil())
			Expect(status.Ports).To(HaveLen(allocation.Count))
			Expect(status.Status).To(Equal(nnfv1alpha1.NnfPortManagerAllocationStatusInUse))
		}

		releasePorts := func(mgr *nnfv1alpha1.NnfPortManager, name string) {
			By(fmt.Sprintf("Releasing ports for '%s'", name))

			requester := corev1.ObjectReference{Name: name}

			Eventually(func() error {
				Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(mgr), mgr)).To(Succeed())

				for idx, allocation := range mgr.Spec.Allocations {
					if allocation.Requester == requester {
						mgr.Spec.Allocations = append(mgr.Spec.Allocations[:idx], mgr.Spec.Allocations[idx+1:]...)
					}
				}

				return k8sClient.Update(ctx, mgr)
			}).Should(Succeed())
		}

		It("Reserves & removes a single port", func() {
			const name = "single"
			reservePorts(mgr, name, 1)
			releasePorts(mgr, name)
		})

		It("Reserves & removes all ports", func() {
			const name = "all"
			reservePorts(mgr, name, portEnd-portStart+1)
			releasePorts(mgr, name)
		})

		It("Reserves from free list", func() {
			const single = "single"
			reservePorts(mgr, single, 1)

			const remaining = "remaining"
			reservePorts(mgr, remaining, portEnd-portStart)

			releasePorts(mgr, single)

			reservePorts(mgr, "free", 1)
		})

		It("Fails with insufficient resources", func() {
			const name = "all"
			reservePorts(mgr, name, portEnd-portStart+1)

			allocation := nnfv1alpha1.NnfPortManagerAllocationSpec{
				Requester: corev1.ObjectReference{Name: "insufficient-resources"},
				Count:     1,
			}

			Eventually(func() error {
				Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(mgr), mgr)).To(Succeed())
				mgr.Spec.Allocations = append(mgr.Spec.Allocations, allocation)
				return k8sClient.Update(ctx, mgr)
			}).Should(Succeed())

			Eventually(func() bool {
				Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(mgr), mgr)).To(Succeed())
				return r.isAllocated(mgr, allocation)
			}).Should(BeTrue())

			status := r.findAllocationStatus(mgr, allocation)
			Expect(status).ToNot(BeNil())
			Expect(status.Ports).To(BeEmpty())
			Expect(status.Status).To(Equal(nnfv1alpha1.NnfPortManagerAllocationStatusInsufficientResources))
		})
	})

})
