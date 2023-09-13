/*
 * Copyright 2021-2023 Hewlett Packard Enterprise Development LP
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
	"context"
	"reflect"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	dwsv1alpha2 "github.com/DataWorkflowServices/dws/api/v1alpha2"
	nnfv1alpha1 "github.com/NearNodeFlash/nnf-sos/api/v1alpha1"
)

var _ = Describe("Access Controller Test", func() {

	nodeNames := []string{
		"rabbit-nnf-access-test-node-1",
		"rabbit-nnf-access-test-node-2"}

	BeforeEach(func() {
		for _, nodeName := range nodeNames {
			ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: nodeName}}
			Expect(k8sClient.Create(context.TODO(), ns)).To(Succeed(), "Create Namespace")
		}
	})

	Describe("Create Client Mounts", func() {

		allocationNodes := make([]nnfv1alpha1.NnfStorageAllocationNodes, len(nodeNames))
		for idx, nodeName := range nodeNames {
			allocationNodes[idx] = nnfv1alpha1.NnfStorageAllocationNodes{
				Count: 1,
				Name:  nodeName,
			}
		}

		It("Creates Lustre Client Mount", func() {

			storage := &nnfv1alpha1.NnfStorage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "nnf-access-test-storage",
					Namespace: corev1.NamespaceDefault,
				},
				Spec: nnfv1alpha1.NnfStorageSpec{
					FileSystemType: "lustre",
					AllocationSets: []nnfv1alpha1.NnfStorageAllocationSetSpec{
						{
							Name: "mgtmdt",
							NnfStorageLustreSpec: nnfv1alpha1.NnfStorageLustreSpec{
								FileSystemName: "MGTMDT",
								TargetType:     "MGTMDT",
							},
							Nodes: []nnfv1alpha1.NnfStorageAllocationNodes{
								{
									Count: 1,
									Name:  corev1.NamespaceDefault,
								},
							},
						},
						{
							Name: "ost",
							NnfStorageLustreSpec: nnfv1alpha1.NnfStorageLustreSpec{
								FileSystemName: "OST",
								TargetType:     "OST",
							},
							Nodes: allocationNodes,
						},
					},
				},
				Status: nnfv1alpha1.NnfStorageStatus{
					MgsNode: "127.0.0.1@tcp",
				},
			}

			Expect(k8sClient.Create(context.TODO(), storage)).To(Succeed(), "Create NNF Storage")
			Eventually(func(g Gomega) error {
				g.Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(storage), storage)).To(Succeed())
				storage.Status.MgsNode = "127.0.0.1@tcp"
				return k8sClient.Status().Update(context.TODO(), storage)
			}).Should(Succeed())

			access := &nnfv1alpha1.NnfAccess{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "nnf-access-test-access",
					Namespace: corev1.NamespaceDefault,
				},
				Spec: nnfv1alpha1.NnfAccessSpec{
					DesiredState:    "mounted",
					TeardownState:   dwsv1alpha2.StatePreRun,
					Target:          "all",
					ClientReference: corev1.ObjectReference{},
					MountPath:       "./",

					StorageReference: corev1.ObjectReference{
						Kind:      reflect.TypeOf(nnfv1alpha1.NnfStorage{}).Name(),
						Name:      storage.Name,
						Namespace: storage.Namespace,
					},
				},
			}

			Expect(k8sClient.Create(context.TODO(), access)).To(Succeed(), "Create NNF Access")

			By("Verify NNF Access goes Ready in mounted state")
			Eventually(func(g Gomega) bool {
				g.Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(access), access)).To(Succeed())
				return access.Status.Ready && access.Status.State == "mounted"
			}).Should(BeTrue())

			By("Verify Client Mounts")
			for _, nodeName := range nodeNames {
				mount := &dwsv1alpha2.ClientMount{
					ObjectMeta: metav1.ObjectMeta{
						Name:      clientMountName(access),
						Namespace: nodeName,
					},
				}
				Eventually(func(g Gomega) bool {
					g.Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(mount), mount)).To(Succeed())
					g.Expect(mount.Status.Mounts).ToNot(HaveLen(0))

					for _, mountStatus := range mount.Status.Mounts {
						if mountStatus.Ready == false {
							return false
						}
					}
					return true
				}).Should(BeTrue())

				Expect(mount.Spec).To(MatchFields(IgnoreExtras, Fields{
					"Node":         Equal(nodeName),
					"DesiredState": Equal(dwsv1alpha2.ClientMountStateMounted),
					"Mounts":       HaveLen(1),
				}))
				Expect(mount.Status.Error).To(BeNil())
				Expect(mount.Status.Mounts).To(HaveLen(1))
				Expect(mount.Status.Mounts[0]).To(MatchAllFields(Fields{
					"State": Equal(dwsv1alpha2.ClientMountStateMounted),
					"Ready": BeTrue(),
				}))
			}

			By("Set NNF Access Desired State to unmounted")
			Eventually(func(g Gomega) error {
				g.Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(access), access)).To(Succeed())
				access.Spec.DesiredState = "unmounted"
				return k8sClient.Update(context.TODO(), access)
			}).Should(Succeed())

			By("Verify NNF Access goes Ready in unmounted state")
			Eventually(func(g Gomega) bool {
				g.Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(access), access)).To(Succeed())
				return access.Status.Ready && access.Status.State == "unmounted"
			}).Should(BeTrue())

			By("Verify Client Mounts go unmounted")
			for _, nodeName := range nodeNames {
				mount := &dwsv1alpha2.ClientMount{
					ObjectMeta: metav1.ObjectMeta{
						Name:      clientMountName(access),
						Namespace: nodeName,
					},
				}
				Eventually(func(g Gomega) bool {
					g.Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(mount), mount)).To(Succeed())
					g.Expect(mount.Status.Mounts).ToNot(HaveLen(0))

					for _, mountStatus := range mount.Status.Mounts {
						if mountStatus.Ready == false {
							return false
						}
					}
					return true
				}).Should(BeTrue())

				Expect(mount.Spec).To(MatchFields(IgnoreExtras, Fields{
					"Node":         Equal(nodeName),
					"DesiredState": Equal(dwsv1alpha2.ClientMountStateUnmounted),
					"Mounts":       HaveLen(1),
				}))
				Expect(mount.Status.Error).To(BeNil())
				Expect(mount.Status.Mounts).To(HaveLen(1))
				Expect(mount.Status.Mounts[0]).To(MatchAllFields(Fields{
					"State": Equal(dwsv1alpha2.ClientMountStateUnmounted),
					"Ready": BeTrue(),
				}))
			}

			By("Deleting the NNF Access")
			Expect(k8sClient.Delete(context.TODO(), access)).To(Succeed())

			Eventually(func() error {
				return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(access), access)
			}).ShouldNot(Succeed())
		})
	})
})
