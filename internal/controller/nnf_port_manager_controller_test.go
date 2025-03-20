/*
 * Copyright 2023-2025 Hewlett Packard Enterprise Development LP
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
	"fmt"
	"reflect"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	dwsv1alpha3 "github.com/DataWorkflowServices/dws/api/v1alpha3"
	nnfv1alpha6 "github.com/NearNodeFlash/nnf-sos/api/v1alpha6"
)

var _ = Context("NNF Port Manager Controller Setup", Ordered, func() {

	var r = &NnfPortManagerReconciler{} // use this to access private reconciler methods

	const portStart = 20
	const portEnd = 29
	portTotal := portEnd - portStart + 1

	Describe("NNF Port Manager Controller Test", func() {
		var cfg *dwsv1alpha3.SystemConfiguration
		var mgr *nnfv1alpha6.NnfPortManager
		portCooldown := 1

		JustBeforeEach(func() {
			cfg = &dwsv1alpha3.SystemConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "default",
					Namespace: corev1.NamespaceDefault,
				},
				Spec: dwsv1alpha3.SystemConfigurationSpec{
					Ports: []intstr.IntOrString{
						intstr.FromString(fmt.Sprintf("%d-%d", portStart, portEnd)),
					},
					PortsCooldownInSeconds: portCooldown,
				},
			}
			Expect(k8sClient.Create(ctx, cfg)).To(Succeed())
			DeferCleanup(func() {
				if cfg != nil {
					Expect(k8sClient.Delete(ctx, cfg)).To(Succeed())
					Eventually(func() error {
						return k8sClient.Get(ctx, client.ObjectKeyFromObject(cfg), cfg)
					}).ShouldNot(Succeed())
				}
			})

			mgr = &nnfv1alpha6.NnfPortManager{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "nnf-port-manager",
					Namespace: corev1.NamespaceDefault,
				},
				Spec: nnfv1alpha6.NnfPortManagerSpec{
					SystemConfiguration: corev1.ObjectReference{
						Name:      cfg.Name,
						Namespace: cfg.Namespace,
						Kind:      reflect.TypeOf(*cfg).Name(),
					},
					Allocations: make([]nnfv1alpha6.NnfPortManagerAllocationSpec, 0),
				},
			}
			Expect(k8sClient.Create(ctx, mgr)).To(Succeed())
			DeferCleanup(func() {
				if mgr != nil {
					Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(mgr), mgr))
					mgr.SetFinalizers([]string{})
					Expect(k8sClient.Update(ctx, mgr)).To(Succeed())
					Expect(k8sClient.Delete(ctx, mgr)).To(Succeed())
					Eventually(func() error {
						return k8sClient.Get(ctx, client.ObjectKeyFromObject(mgr), mgr)
					}).ShouldNot(Succeed())
				}
			})
		})

		// Submit an allocation and verify it has been accounted for - this doesn't mean the ports
		// were successfully allocated, however.
		allocatePorts := func(mgr *nnfv1alpha6.NnfPortManager, name string, count int) []uint16 {
			By(fmt.Sprintf("Reserving %d ports for '%s'", count, name))

			allocation := nnfv1alpha6.NnfPortManagerAllocationSpec{
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
			return status.Ports
		}

		// Submit an allocation and expect it to be successfully allocated (i.e. ports InUse)
		reservePorts := func(mgr *nnfv1alpha6.NnfPortManager, name string, count int) []uint16 {
			ports := allocatePorts(mgr, name, count)

			allocation := nnfv1alpha6.NnfPortManagerAllocationSpec{
				Requester: corev1.ObjectReference{Name: name},
				Count:     count,
			}

			status := r.findAllocationStatus(mgr, allocation)
			Expect(status).ToNot(BeNil())
			Expect(status.Ports).To(HaveLen(allocation.Count))
			Expect(status.Status).To(Equal(nnfv1alpha6.NnfPortManagerAllocationStatusInUse))

			return ports
		}

		reservePortsAllowFail := func(mgr *nnfv1alpha6.NnfPortManager, name string, count int) []uint16 {
			return allocatePorts(mgr, name, count)
		}

		releasePorts := func(mgr *nnfv1alpha6.NnfPortManager, name string) {
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

		// Simple way to fire the reconciler to test the cooldown handling
		// without having to reserve new ports. This is just to limit the scope
		// of the test.
		kickPortManager := func(mgr *nnfv1alpha6.NnfPortManager) {
			By("Kicking port manager to force reconcile")

			Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(mgr), mgr)).To(Succeed())
			finalizers := mgr.GetFinalizers()
			finalizers = append(finalizers, "test-"+uuid.NewString())
			mgr.SetFinalizers(finalizers)
			Eventually(func() error {
				return k8sClient.Update(ctx, mgr)
			}).Should(Succeed())
		}

		// Verify the number of allocations in the status allocation list that are InUse
		verifyNumAllocations := func(mgr *nnfv1alpha6.NnfPortManager, status nnfv1alpha6.NnfPortManagerAllocationStatusStatus, count int) {
			By(fmt.Sprintf("Verifying there are %d allocations with Status %s in the status allocation list", count, status))

			Eventually(func() int {
				statusCount := 0
				Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(mgr), mgr)).To(Succeed())
				for _, a := range mgr.Status.Allocations {
					if a.Status == status {
						statusCount++
					}
				}
				return statusCount
			}).Should(Equal(count))
		}

		verifyNumAllocationsInUse := func(mgr *nnfv1alpha6.NnfPortManager, count int) {
			verifyNumAllocations(mgr, nnfv1alpha6.NnfPortManagerAllocationStatusInUse, count)
		}

		verifyNumAllocationsCooldown := func(mgr *nnfv1alpha6.NnfPortManager, count int) {
			verifyNumAllocations(mgr, nnfv1alpha6.NnfPortManagerAllocationStatusCooldown, count)
		}

		verifyNumAllocationsInsuffientResources := func(mgr *nnfv1alpha6.NnfPortManager, count int) {
			verifyNumAllocations(mgr, nnfv1alpha6.NnfPortManagerAllocationStatusInsufficientResources, count)
		}

		waitForCooldown := func(extra int) {
			By(fmt.Sprintf("Waiting for cooldown (%ds)to expire", portCooldown))
			time.Sleep(time.Duration(portCooldown+extra) * time.Second)

		}

		When("the system configuration is missing", func() {
			It("should have a status that indicates system configuration is not found", func() {
				Expect(k8sClient.Delete(ctx, cfg)).To(Succeed())
				Eventually(func() error {
					return k8sClient.Get(ctx, client.ObjectKeyFromObject(cfg), cfg)
				}).ShouldNot(Succeed())
				cfg = nil

				kickPortManager(mgr)

				Eventually(func() nnfv1alpha6.NnfPortManagerStatusStatus {
					k8sClient.Get(ctx, client.ObjectKeyFromObject(mgr), mgr)
					return mgr.Status.Status
				}).Should(Equal(nnfv1alpha6.NnfPortManagerStatusSystemConfigurationNotFound))
			})
		})

		When("reserving ports with portCooldown", func() {

			BeforeEach(func() {
				portCooldown = 2
			})

			When("a single port is reserved and removed", func() {
				It("should cooldown and then free up", func() {
					const name = "single"
					ports := reservePorts(mgr, name, 1)
					Expect(ports[0]).To(BeEquivalentTo(portStart))
					verifyNumAllocationsInUse(mgr, 1)
					releasePorts(mgr, name)
					verifyNumAllocationsInUse(mgr, 0)
					verifyNumAllocationsCooldown(mgr, 1)

					waitForCooldown(0)
					kickPortManager(mgr)
					verifyNumAllocationsInUse(mgr, 0)
					verifyNumAllocationsCooldown(mgr, 0)
				})
			})

			When("reserving and releasing multiple ports, one after another", func() {
				It("should use the next port since the first is still in cooldown", func() {
					first := "first"
					ports := reservePorts(mgr, first, 1)
					Expect(ports[0]).To(BeEquivalentTo(portStart))
					verifyNumAllocationsInUse(mgr, 1)

					second := "second"
					ports = reservePorts(mgr, second, 1)
					Expect(ports[0]).To(BeEquivalentTo(portStart + 1))
					verifyNumAllocationsInUse(mgr, 2)

					releasePorts(mgr, first)
					verifyNumAllocationsInUse(mgr, 1)
					verifyNumAllocationsCooldown(mgr, 1)

					releasePorts(mgr, second)
					verifyNumAllocationsInUse(mgr, 0)
					verifyNumAllocationsCooldown(mgr, 2)
				})
			})

			When("reserving and releasing multiple ports, one at a time", func() {
				It("should use the next port since the first is still in cooldown", func() {
					first := "first"
					ports := reservePorts(mgr, first, 1)
					firstPort := ports[0]
					Expect(ports[0]).To(BeEquivalentTo(portStart))
					verifyNumAllocationsInUse(mgr, 1)
					releasePorts(mgr, first)
					verifyNumAllocationsInUse(mgr, 0)
					verifyNumAllocationsCooldown(mgr, 1)

					second := "second"
					ports = reservePorts(mgr, second, 1)
					Expect(ports[0]).To(BeEquivalentTo(firstPort + 1))
					verifyNumAllocationsInUse(mgr, 1)
					verifyNumAllocationsCooldown(mgr, 1)

					releasePorts(mgr, second)
					verifyNumAllocationsInUse(mgr, 0)
					verifyNumAllocationsCooldown(mgr, 2)
				})
			})

			When("reserving all the ports in 1 allocation", func() {
				It("should reserve and cooldown successfully", func() {
					const name = "all"
					reservePorts(mgr, name, portEnd-portStart+1)
					verifyNumAllocationsInUse(mgr, 1)
					verifyNumAllocationsCooldown(mgr, 0)
					releasePorts(mgr, name)
					verifyNumAllocationsInUse(mgr, 0)
					verifyNumAllocationsCooldown(mgr, 1)
				})
			})

			XIt("Reserves from free list", func() {
				const single = "single"
				reservePorts(mgr, single, 1)

				const remaining = "remaining"
				count := portEnd - portStart
				reservePorts(mgr, remaining, count)

				releasePorts(mgr, single)
				verifyNumAllocationsInUse(mgr, 1)

				reservePorts(mgr, "free", 1)

				verifyNumAllocationsInUse(mgr, 2)
			})

			When("all ports are already reserved", func() {
				It("fails with insufficient resources", func() {
					const name = "all"
					reservePorts(mgr, name, portEnd-portStart+1)

					allocation := nnfv1alpha6.NnfPortManagerAllocationSpec{
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
					Expect(status.Status).To(Equal(nnfv1alpha6.NnfPortManagerAllocationStatusInsufficientResources))
				})
			})

			When("a single port is reserved and released", func() {
				It("expires and is removed from allocations after the cooldown period", func() {
					const name = "single"
					ports := reservePorts(mgr, name, 1)
					Expect(ports[0]).To(BeEquivalentTo(portStart))
					verifyNumAllocationsInUse(mgr, 1)
					verifyNumAllocationsCooldown(mgr, 0)

					releasePorts(mgr, name)
					verifyNumAllocationsInUse(mgr, 0)
					verifyNumAllocationsCooldown(mgr, 1)

					waitForCooldown(0)
					kickPortManager(mgr)
					verifyNumAllocationsInUse(mgr, 0)
					verifyNumAllocationsCooldown(mgr, 0)
				})
			})

			When("all ports are already reserved and another allocation is requested", func() {
				It("should eventually free up the cooldown ports and successfully reserve", func() {
					By("Reserving all available ports")
					for i := 0; i < portTotal; i++ {
						ports := reservePorts(mgr, fmt.Sprintf("test-%d", i), 1)
						verifyNumAllocationsInUse(mgr, i+1)
						Expect(ports[0]).To(BeEquivalentTo(portStart + i))
					}
					verifyNumAllocationsInUse(mgr, portTotal)

					By("Attempting to reserve an additional port and failing")
					ports := reservePortsAllowFail(mgr, "waiting", 1)
					allocation := nnfv1alpha6.NnfPortManagerAllocationSpec{Requester: corev1.ObjectReference{Name: "waiting"}, Count: 1}
					status := r.findAllocationStatus(mgr, allocation)

					Expect(ports).To(HaveLen(0))
					Expect(status).ToNot(BeNil())
					Expect(status.Status).To(Equal(nnfv1alpha6.NnfPortManagerAllocationStatusInsufficientResources))
					verifyNumAllocationsInUse(mgr, portTotal)
					verifyNumAllocationsInsuffientResources(mgr, 1)

					By("Releasing one of the original ports to make room for previous request")
					releasePorts(mgr, "test-0")
					verifyNumAllocationsInUse(mgr, portTotal-1)
					verifyNumAllocationsCooldown(mgr, 1)
					verifyNumAllocationsInsuffientResources(mgr, 1)

					By("Verifying that the cooldown expired and the new reservation is now InUse")
					waitForCooldown(0)
					verifyNumAllocationsCooldown(mgr, 0)
					verifyNumAllocationsInsuffientResources(mgr, 0)
					verifyNumAllocationsInUse(mgr, portTotal)
				})
			})
		})

		When("reserving ports with portCooldown", func() {

			BeforeEach(func() {
				portCooldown = 0
			})

			When("reserving and releasing multiple ports, one at a time", func() {
				It("should use the same port since the first has no cooldown", func() {
					first := "first"
					ports := reservePorts(mgr, first, 1)
					firstPort := ports[0]
					Expect(ports[0]).To(BeEquivalentTo(portStart))
					verifyNumAllocationsInUse(mgr, 1)
					releasePorts(mgr, first)
					verifyNumAllocationsInUse(mgr, 0)
					verifyNumAllocationsCooldown(mgr, 0)

					second := "second"
					ports = reservePorts(mgr, second, 1)
					Expect(ports[0]).To(BeEquivalentTo(firstPort))
					verifyNumAllocationsInUse(mgr, 1)
					verifyNumAllocationsCooldown(mgr, 0)

					releasePorts(mgr, second)
					verifyNumAllocationsInUse(mgr, 0)
					verifyNumAllocationsCooldown(mgr, 0)
				})
			})
		})
	})
})
