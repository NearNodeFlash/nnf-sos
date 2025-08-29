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

package controller

import (
	"context"
	"reflect"
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	dwsv1alpha6 "github.com/DataWorkflowServices/dws/api/v1alpha6"
	nnfv1alpha8 "github.com/NearNodeFlash/nnf-sos/api/v1alpha8"
)

var _ = Describe("NnfSystemStorage Controller Test", func() {

	nodeNames := []string{
		"rabbit-systemstorage-node-1",
		"rabbit-systemstorage-node-2"}

	nnfNodes := [2]*nnfv1alpha8.NnfNode{}
	nodes := [2]*corev1.Node{}

	var systemConfiguration *dwsv1alpha6.SystemConfiguration
	var storageProfile *nnfv1alpha8.NnfStorageProfile
	var setup sync.Once

	BeforeEach(func() {
		setup.Do(func() {
			for _, nodeName := range nodeNames {
				ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: nodeName}}
				Expect(k8sClient.Create(context.TODO(), ns)).To(Succeed(), "Create Namespace")
			}
		})

		systemConfiguration = &dwsv1alpha6.SystemConfiguration{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "default",
				Namespace: corev1.NamespaceDefault,
			},
			Spec: dwsv1alpha6.SystemConfigurationSpec{
				StorageNodes: []dwsv1alpha6.SystemConfigurationStorageNode{
					{
						Type: "Rabbit",
						Name: nodeNames[0],
						ComputesAccess: []dwsv1alpha6.SystemConfigurationComputeNodeReference{
							{
								Name:  "0-0",
								Index: 0,
							},
							{
								Name:  "0-1",
								Index: 1,
							},
							{
								Name:  "0-2",
								Index: 2,
							},
							{
								Name:  "0-3",
								Index: 3,
							},
							{
								Name:  "0-4",
								Index: 4,
							},
							{
								Name:  "0-5",
								Index: 5,
							},
							{
								Name:  "0-6",
								Index: 6,
							},
							{
								Name:  "0-7",
								Index: 7,
							},
							{
								Name:  "0-8",
								Index: 8,
							},
							{
								Name:  "0-9",
								Index: 9,
							},
							{
								Name:  "0-10",
								Index: 10,
							},
							{
								Name:  "0-11",
								Index: 11,
							},
							{
								Name:  "0-12",
								Index: 12,
							},
							{
								Name:  "0-13",
								Index: 13,
							},
							{
								Name:  "0-14",
								Index: 14,
							},
							{
								Name:  "0-15",
								Index: 15,
							},
						},
					},
					{
						Type: "Rabbit",
						Name: nodeNames[1],
						ComputesAccess: []dwsv1alpha6.SystemConfigurationComputeNodeReference{
							{
								Name:  "1-0",
								Index: 0,
							},
							{
								Name:  "1-1",
								Index: 1,
							},
							{
								Name:  "1-2",
								Index: 2,
							},
							{
								Name:  "1-3",
								Index: 3,
							},
							{
								Name:  "1-4",
								Index: 4,
							},
							{
								Name:  "1-5",
								Index: 5,
							},
							{
								Name:  "1-6",
								Index: 6,
							},
							{
								Name:  "1-7",
								Index: 7,
							},
							{
								Name:  "1-8",
								Index: 8,
							},
							{
								Name:  "1-9",
								Index: 9,
							},
							{
								Name:  "1-10",
								Index: 10,
							},
							{
								Name:  "1-11",
								Index: 11,
							},
							{
								Name:  "1-12",
								Index: 12,
							},
							{
								Name:  "1-13",
								Index: 13,
							},
							{
								Name:  "1-14",
								Index: 14,
							},
							{
								Name:  "1-15",
								Index: 15,
							},
						},
					},
				},
			},
		}

		Expect(k8sClient.Create(context.TODO(), systemConfiguration)).To(Succeed())
		for i, nodeName := range nodeNames {
			// Create the node - set it to up as ready
			nodes[i] = &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: nodeName,
					Labels: map[string]string{
						nnfv1alpha8.RabbitNodeSelectorLabel: "true",
					},
				},
				Status: corev1.NodeStatus{
					Conditions: []corev1.NodeCondition{
						{
							Status: corev1.ConditionTrue,
							Type:   corev1.NodeReady,
						},
					},
				},
			}

			Expect(k8sClient.Create(context.TODO(), nodes[i])).To(Succeed())

			nnfNodes[i] = &nnfv1alpha8.NnfNode{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "nnf-nlc",
					Namespace: nodeName,
				},
				Spec: nnfv1alpha8.NnfNodeSpec{
					State: nnfv1alpha8.ResourceEnable,
				},
			}
			Expect(k8sClient.Create(context.TODO(), nnfNodes[i])).To(Succeed())

			Eventually(func(g Gomega) error {
				g.Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(nnfNodes[i]), nnfNodes[i])).To(Succeed())
				nnfNodes[i].Status.LNetNid = "1.2.3.4@tcp0"
				return k8sClient.Update(context.TODO(), nnfNodes[i])
			}).Should(Succeed(), "set LNet Nid in NnfNode")

			storage := &dwsv1alpha6.Storage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      nodeName,
					Namespace: corev1.NamespaceDefault,
				},
			}

			Eventually(func() error { // wait until the SystemConfiguration controller creates the storage
				return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(storage), storage)
			}).Should(Succeed())
		}

		// Create a pinned NnfStorageProfile for the unit tests.
		storageProfile = createBasicPinnedNnfStorageProfile()
	})

	AfterEach(func() {
		Expect(k8sClient.Delete(context.TODO(), storageProfile)).To(Succeed())
		profExpected := &nnfv1alpha8.NnfStorageProfile{}
		Eventually(func() error { // Delete can still return the cached object. Wait until the object is no longer present
			return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(storageProfile), profExpected)
		}).ShouldNot(Succeed())

		for i := range nodeNames {
			Expect(k8sClient.Delete(context.TODO(), nnfNodes[i])).To(Succeed())
			tempNnfNode := &nnfv1alpha8.NnfNode{}
			Eventually(func() error { // Delete can still return the cached object. Wait until the object is no longer present
				return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(nnfNodes[i]), tempNnfNode)
			}).ShouldNot(Succeed())

			Expect(k8sClient.Delete(context.TODO(), nodes[i])).To(Succeed())
			tempNode := &corev1.Node{}
			Eventually(func() error { // Delete can still return the cached object. Wait until the object is no longer present
				return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(nodes[i]), tempNode)
			}).ShouldNot(Succeed())
		}

		Expect(k8sClient.Delete(context.TODO(), systemConfiguration)).To(Succeed())
		tempConfig := &dwsv1alpha6.SystemConfiguration{}
		Eventually(func() error { // Delete can still return the cached object. Wait until the object is no longer present
			return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(systemConfiguration), tempConfig)
		}).ShouldNot(Succeed())
	})

	Describe("Create NnfSystemStorage", func() {
		It("Creates basic system storage", func() {
			nnfSystemStorage := &nnfv1alpha8.NnfSystemStorage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "nnf-system-storage",
					Namespace: corev1.NamespaceDefault,
				},
				Spec: nnfv1alpha8.NnfSystemStorageSpec{
					Type:             "raw",
					ComputesTarget:   nnfv1alpha8.ComputesTargetAll,
					MakeClientMounts: false,
					Shared:           true,
					Capacity:         1073741824,
					StorageProfile: corev1.ObjectReference{
						Name:      storageProfile.GetName(),
						Namespace: storageProfile.GetNamespace(),
						Kind:      reflect.TypeOf(nnfv1alpha8.NnfStorageProfile{}).Name(),
					},
				},
			}

			By("Creating the NnfSystemStorage")
			Expect(k8sClient.Create(context.TODO(), nnfSystemStorage)).To(Succeed(), "Create NNF System Storage")
			Eventually(func(g Gomega) bool {
				g.Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(nnfSystemStorage), nnfSystemStorage)).To(Succeed())
				return nnfSystemStorage.Status.Ready
			}).Should(BeTrue())

			servers := &dwsv1alpha6.Servers{
				ObjectMeta: metav1.ObjectMeta{
					Name:      nnfSystemStorage.GetName(),
					Namespace: nnfSystemStorage.GetNamespace(),
				},
			}
			Eventually(func(g Gomega) error {
				return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(servers), servers)
			}).Should(Succeed())

			Expect(servers.Spec.AllocationSets).To(HaveLen(1))
			Expect(servers.Spec.AllocationSets[0].Storage).To(HaveLen(2))

			computes := &dwsv1alpha6.Computes{
				ObjectMeta: metav1.ObjectMeta{
					Name:      nnfSystemStorage.GetName(),
					Namespace: nnfSystemStorage.GetNamespace(),
				},
			}
			Eventually(func(g Gomega) error {
				return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(computes), computes)
			}).Should(Succeed())

			Expect(computes.Data).To(HaveLen(32))

			By("Deleting the NnfSystemStorage")
			Expect(k8sClient.Delete(context.TODO(), nnfSystemStorage)).To(Succeed())

			Eventually(func() error {
				return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(nnfSystemStorage), nnfSystemStorage)
			}).ShouldNot(Succeed())
		})

		It("Creates even system storage", func() {
			nnfSystemStorage := &nnfv1alpha8.NnfSystemStorage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "nnf-system-storage",
					Namespace: corev1.NamespaceDefault,
				},
				Spec: nnfv1alpha8.NnfSystemStorageSpec{
					Type:             "raw",
					ComputesTarget:   nnfv1alpha8.ComputesTargetEven,
					MakeClientMounts: false,
					Shared:           true,
					Capacity:         1073741824,
					StorageProfile: corev1.ObjectReference{
						Name:      storageProfile.GetName(),
						Namespace: storageProfile.GetNamespace(),
						Kind:      reflect.TypeOf(nnfv1alpha8.NnfStorageProfile{}).Name(),
					},
				},
			}

			By("Creating the NnfSystemStorage")
			Expect(k8sClient.Create(context.TODO(), nnfSystemStorage)).To(Succeed(), "Create NNF System Storage")
			Eventually(func(g Gomega) bool {
				g.Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(nnfSystemStorage), nnfSystemStorage)).To(Succeed())
				return nnfSystemStorage.Status.Ready
			}).Should(BeTrue())

			servers := &dwsv1alpha6.Servers{
				ObjectMeta: metav1.ObjectMeta{
					Name:      nnfSystemStorage.GetName(),
					Namespace: nnfSystemStorage.GetNamespace(),
				},
			}
			Eventually(func(g Gomega) error {
				return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(servers), servers)
			}).Should(Succeed())

			Expect(servers.Spec.AllocationSets).To(HaveLen(1))
			Expect(servers.Spec.AllocationSets[0].Storage).To(HaveLen(2))

			computes := &dwsv1alpha6.Computes{
				ObjectMeta: metav1.ObjectMeta{
					Name:      nnfSystemStorage.GetName(),
					Namespace: nnfSystemStorage.GetNamespace(),
				},
			}
			Eventually(func(g Gomega) error {
				return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(computes), computes)
			}).Should(Succeed())

			Expect(computes.Data).To(HaveLen(16))

			By("Deleting the NnfSystemStorage")
			Expect(k8sClient.Delete(context.TODO(), nnfSystemStorage)).To(Succeed())

			Eventually(func() error {
				return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(nnfSystemStorage), nnfSystemStorage)
			}).ShouldNot(Succeed())
		})

		It("Creates system storage with index map", func() {
			nnfSystemStorage := &nnfv1alpha8.NnfSystemStorage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "nnf-system-storage",
					Namespace: corev1.NamespaceDefault,
				},
				Spec: nnfv1alpha8.NnfSystemStorageSpec{
					Type:             "raw",
					ComputesTarget:   nnfv1alpha8.ComputesTargetPattern,
					ComputesPattern:  []int{0, 1, 2, 3, 4},
					MakeClientMounts: false,
					Shared:           true,
					Capacity:         1073741824,
					StorageProfile: corev1.ObjectReference{
						Name:      storageProfile.GetName(),
						Namespace: storageProfile.GetNamespace(),
						Kind:      reflect.TypeOf(nnfv1alpha8.NnfStorageProfile{}).Name(),
					},
				},
			}

			By("Creating the NnfSystemStorage")
			Expect(k8sClient.Create(context.TODO(), nnfSystemStorage)).To(Succeed(), "Create NNF System Storage")
			Eventually(func(g Gomega) bool {
				g.Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(nnfSystemStorage), nnfSystemStorage)).To(Succeed())
				return nnfSystemStorage.Status.Ready
			}).Should(BeTrue())

			servers := &dwsv1alpha6.Servers{
				ObjectMeta: metav1.ObjectMeta{
					Name:      nnfSystemStorage.GetName(),
					Namespace: nnfSystemStorage.GetNamespace(),
				},
			}
			Eventually(func(g Gomega) error {
				return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(servers), servers)
			}).Should(Succeed())

			Expect(servers.Spec.AllocationSets).To(HaveLen(1))
			Expect(servers.Spec.AllocationSets[0].Storage).To(HaveLen(2))

			computes := &dwsv1alpha6.Computes{
				ObjectMeta: metav1.ObjectMeta{
					Name:      nnfSystemStorage.GetName(),
					Namespace: nnfSystemStorage.GetNamespace(),
				},
			}
			Eventually(func(g Gomega) error {
				return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(computes), computes)
			}).Should(Succeed())

			Expect(computes.Data).To(HaveLen(10))

			By("Deleting the NnfSystemStorage")
			Expect(k8sClient.Delete(context.TODO(), nnfSystemStorage)).To(Succeed())

			Eventually(func() error {
				return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(nnfSystemStorage), nnfSystemStorage)
			}).ShouldNot(Succeed())
		})

		It("Creates system storage with excluded Rabbits and computes", func() {
			nnfSystemStorage := &nnfv1alpha8.NnfSystemStorage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "nnf-system-storage",
					Namespace: corev1.NamespaceDefault,
				},
				Spec: nnfv1alpha8.NnfSystemStorageSpec{
					Type:             "raw",
					ComputesTarget:   nnfv1alpha8.ComputesTargetAll,
					ExcludeRabbits:   []string{nodeNames[0]},
					ExcludeComputes:  []string{"1-4", "1-5", "1-6"},
					MakeClientMounts: false,
					Shared:           true,
					Capacity:         1073741824,
					StorageProfile: corev1.ObjectReference{
						Name:      storageProfile.GetName(),
						Namespace: storageProfile.GetNamespace(),
						Kind:      reflect.TypeOf(nnfv1alpha8.NnfStorageProfile{}).Name(),
					},
				},
			}

			By("Creating the NnfSystemStorage")
			Expect(k8sClient.Create(context.TODO(), nnfSystemStorage)).To(Succeed(), "Create NNF System Storage")
			Eventually(func(g Gomega) bool {
				g.Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(nnfSystemStorage), nnfSystemStorage)).To(Succeed())
				return nnfSystemStorage.Status.Ready
			}).Should(BeTrue())

			servers := &dwsv1alpha6.Servers{
				ObjectMeta: metav1.ObjectMeta{
					Name:      nnfSystemStorage.GetName(),
					Namespace: nnfSystemStorage.GetNamespace(),
				},
			}
			Eventually(func(g Gomega) error {
				return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(servers), servers)
			}).Should(Succeed())

			Expect(servers.Spec.AllocationSets).To(HaveLen(1))
			Expect(servers.Spec.AllocationSets[0].Storage).To(HaveLen(1))
			Expect(servers.Spec.AllocationSets[0].Storage[0]).To(MatchAllFields(Fields{
				"Name":            Equal(nodeNames[1]),
				"AllocationCount": Equal(1),
			}))

			computes := &dwsv1alpha6.Computes{
				ObjectMeta: metav1.ObjectMeta{
					Name:      nnfSystemStorage.GetName(),
					Namespace: nnfSystemStorage.GetNamespace(),
				},
			}
			Eventually(func(g Gomega) error {
				return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(computes), computes)
			}).Should(Succeed())

			Expect(computes.Data).To(HaveLen(13))

			By("Deleting the NnfSystemStorage")
			Expect(k8sClient.Delete(context.TODO(), nnfSystemStorage)).To(Succeed())

			Eventually(func() error {
				return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(nnfSystemStorage), nnfSystemStorage)
			}).ShouldNot(Succeed())
		})
	})
})
