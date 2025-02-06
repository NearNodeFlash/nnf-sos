/*
 * Copyright 2021-2025 Hewlett Packard Enterprise Development LP
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
	"path"
	"reflect"
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	dwsv1alpha2 "github.com/DataWorkflowServices/dws/api/v1alpha2"
	nnfv1alpha5 "github.com/NearNodeFlash/nnf-sos/api/v1alpha5"
)

var _ = Describe("Access Controller Test", func() {

	nodeNames := []string{
		"rabbit-nnf-access-test-node-1",
		"rabbit-nnf-access-test-node-2"}

	nnfNodes := [2]*nnfv1alpha5.NnfNode{}
	nodes := [2]*corev1.Node{}

	var systemConfiguration *dwsv1alpha2.SystemConfiguration
	var storageProfile *nnfv1alpha5.NnfStorageProfile
	var setup sync.Once

	BeforeEach(func() {
		setup.Do(func() {
			for _, nodeName := range nodeNames {
				ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: nodeName}}
				Expect(k8sClient.Create(context.TODO(), ns)).To(Succeed(), "Create Namespace")
			}
		})

		systemConfiguration = &dwsv1alpha2.SystemConfiguration{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "default",
				Namespace: corev1.NamespaceDefault,
			},
			Spec: dwsv1alpha2.SystemConfigurationSpec{
				StorageNodes: []dwsv1alpha2.SystemConfigurationStorageNode{
					{
						Type: "Rabbit",
						Name: "rabbit-nnf-access-test-node-1",
					},
					{
						Type: "Rabbit",
						Name: "rabbit-nnf-access-test-node-2",
					},
				},
			},
		}

		for i, nodeName := range nodeNames {
			// Create the node - set it to up as ready
			nodes[i] = &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: nodeName,
					Labels: map[string]string{
						nnfv1alpha5.RabbitNodeSelectorLabel: "true",
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

			nnfNodes[i] = &nnfv1alpha5.NnfNode{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "nnf-nlc",
					Namespace: nodeName,
				},
				Spec: nnfv1alpha5.NnfNodeSpec{
					State: nnfv1alpha5.ResourceEnable,
				},
			}
			Expect(k8sClient.Create(context.TODO(), nnfNodes[i])).To(Succeed())

			Eventually(func(g Gomega) error {
				g.Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(nnfNodes[i]), nnfNodes[i])).To(Succeed())
				nnfNodes[i].Status.LNetNid = "1.2.3.4@tcp0"
				return k8sClient.Update(context.TODO(), nnfNodes[i])
			}).Should(Succeed(), "set LNet Nid in NnfNode")

			storage := &dwsv1alpha2.Storage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      nodeName,
					Namespace: corev1.NamespaceDefault,
				},
			}

			Eventually(func() error { // wait until the SystemConfiguration controller creates the storage
				return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(storage), storage)
			}).ShouldNot(Succeed())
		}

		Expect(k8sClient.Create(context.TODO(), systemConfiguration)).To(Succeed())

		// Create a pinned NnfStorageProfile for the unit tests.
		storageProfile = createBasicPinnedNnfStorageProfile()
	})

	AfterEach(func() {
		Expect(k8sClient.Delete(context.TODO(), storageProfile)).To(Succeed())
		profExpected := &nnfv1alpha5.NnfStorageProfile{}
		Eventually(func() error { // Delete can still return the cached object. Wait until the object is no longer present
			return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(storageProfile), profExpected)
		}).ShouldNot(Succeed())

		for i := range nodeNames {
			Expect(k8sClient.Delete(context.TODO(), nnfNodes[i])).To(Succeed())
			tempNnfNode := &nnfv1alpha5.NnfNode{}
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
		tempConfig := &dwsv1alpha2.SystemConfiguration{}
		Eventually(func() error { // Delete can still return the cached object. Wait until the object is no longer present
			return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(systemConfiguration), tempConfig)
		}).ShouldNot(Succeed())
	})

	Describe("Create Client Mounts", func() {

		It("Creates Lustre Client Mount", func() {
			allocationNodes := make([]nnfv1alpha5.NnfStorageAllocationNodes, len(nodeNames))
			for idx, nodeName := range nodeNames {
				allocationNodes[idx] = nnfv1alpha5.NnfStorageAllocationNodes{
					Count: 1,
					Name:  nodeName,
				}
			}

			storage := &nnfv1alpha5.NnfStorage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "nnf-access-test-storage-lustre",
					Namespace: corev1.NamespaceDefault,
				},
				Spec: nnfv1alpha5.NnfStorageSpec{
					FileSystemType: "lustre",
					AllocationSets: []nnfv1alpha5.NnfStorageAllocationSetSpec{
						{
							Name:     "mgtmdt",
							Capacity: 50000000000,
							NnfStorageLustreSpec: nnfv1alpha5.NnfStorageLustreSpec{
								TargetType: "mgtmdt",
							},
							Nodes: []nnfv1alpha5.NnfStorageAllocationNodes{
								{
									Count: 1,
									Name:  nodeNames[0],
								},
							},
						},
						{
							Name:     "ost",
							Capacity: 50000000000,
							NnfStorageLustreSpec: nnfv1alpha5.NnfStorageLustreSpec{
								TargetType: "ost",
							},
							Nodes: allocationNodes,
						},
					},
				},
			}

			verifyClientMount(storage, storageProfile, nodeNames)
		})

		It("Creates XFS Client Mount", func() {

			storage := &nnfv1alpha5.NnfStorage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "nnf-access-test-storage-xfs",
					Namespace: corev1.NamespaceDefault,
				},
				Spec: nnfv1alpha5.NnfStorageSpec{
					FileSystemType: "xfs",
					AllocationSets: []nnfv1alpha5.NnfStorageAllocationSetSpec{
						{
							Name:     "xfs",
							Capacity: 50000000000,
							Nodes: []nnfv1alpha5.NnfStorageAllocationNodes{
								{
									Count: 1,
									Name:  nodeNames[0],
								},
								{
									Count: 1,
									Name:  nodeNames[1],
								},
							},
						},
					},
				},
			}

			verifyClientMount(storage, storageProfile, nodeNames)
		})

		It("Creates GFS2 Client Mount", func() {

			storage := &nnfv1alpha5.NnfStorage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "nnf-access-test-storage-gfs2",
					Namespace: corev1.NamespaceDefault,
				},
				Spec: nnfv1alpha5.NnfStorageSpec{
					FileSystemType: "gfs2",
					AllocationSets: []nnfv1alpha5.NnfStorageAllocationSetSpec{
						{
							Name:     "gfs2",
							Capacity: 50000000000,
							Nodes: []nnfv1alpha5.NnfStorageAllocationNodes{
								{
									Count: 1,
									Name:  nodeNames[0],
								},
								{
									Count: 1,
									Name:  nodeNames[1],
								},
							},
						},
					},
				},
			}

			verifyClientMount(storage, storageProfile, nodeNames)
		})
	})
})

func verifyClientMount(storage *nnfv1alpha5.NnfStorage, storageProfile *nnfv1alpha5.NnfStorageProfile, nodeNames []string) {
	Expect(k8sClient.Create(context.TODO(), storage)).To(Succeed(), "Create NNF Storage")
	Eventually(func(g Gomega) error {
		g.Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(storage), storage)).To(Succeed())
		storage.Status.MgsAddress = "127.0.0.1@tcp"
		return k8sClient.Status().Update(context.TODO(), storage)
	}).Should(Succeed())

	mountPath := "/mnt/nnf/12345-0/"
	access := &nnfv1alpha5.NnfAccess{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nnf-access-test-access-" + storage.Spec.FileSystemType,
			Namespace: corev1.NamespaceDefault,
		},
		Spec: nnfv1alpha5.NnfAccessSpec{

			DesiredState:     "mounted",
			TeardownState:    dwsv1alpha2.StatePreRun,
			Target:           "all",
			ClientReference:  corev1.ObjectReference{},
			MakeClientMounts: true,
			MountPath:        mountPath,
			MountPathPrefix:  mountPath,

			StorageReference: corev1.ObjectReference{
				Kind:      reflect.TypeOf(nnfv1alpha5.NnfStorage{}).Name(),
				Name:      storage.Name,
				Namespace: storage.Namespace,
			},
		},
	}

	addPinnedStorageProfileLabel(access, storageProfile)

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

		// If not lustre, verify the index mount directory
		if storage.Spec.FileSystemType != "lustre" {
			p := path.Join(mountPath, nodeName+"-0")
			Expect(mount.Spec.Mounts[0].MountPath).To(Equal(p))
		}

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

	By("Deleting the NNF Storage")
	Expect(k8sClient.Delete(context.TODO(), storage)).To(Succeed())

	Eventually(func() error {
		return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(storage), storage)
	}).ShouldNot(Succeed())
}
