/*
 * Copyright 2023-2024 Hewlett Packard Enterprise Development LP
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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/util/taints"
	"sigs.k8s.io/controller-runtime/pkg/client"

	dwsv1alpha2 "github.com/DataWorkflowServices/dws/api/v1alpha2"
	nnfv1alpha1 "github.com/NearNodeFlash/nnf-sos/api/v1alpha1"
)

var _ = Describe("NnfSystemconfigurationController", func() {
	var sysCfg *dwsv1alpha2.SystemConfiguration

	AfterEach(func() {
		Expect(k8sClient.Delete(context.TODO(), sysCfg)).To(Succeed())
		Eventually(func() error { // Delete can still return the cached object. Wait until the object is no longer present
			return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(sysCfg), sysCfg)
		}).ShouldNot(Succeed())
	})

	When("creating a SystemConfiguration", func() {
		sysCfg = &dwsv1alpha2.SystemConfiguration{
			ObjectMeta: v1.ObjectMeta{
				Name:      "default",
				Namespace: corev1.NamespaceDefault,
			},
			Spec: dwsv1alpha2.SystemConfigurationSpec{
				StorageNodes: []dwsv1alpha2.SystemConfigurationStorageNode{
					{
						Type: "Rabbit",
						Name: "rabbit1",
						ComputesAccess: []dwsv1alpha2.SystemConfigurationComputeNodeReference{
							{
								Name:  "test-compute-0",
								Index: 0,
							},
						},
					},
				},
			},
		}

		It("should go ready", func() {
			Expect(k8sClient.Create(context.TODO(), sysCfg)).To(Succeed())
			Eventually(func(g Gomega) bool {
				g.Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(sysCfg), sysCfg)).To(Succeed())
				return sysCfg.Status.Ready
			}).Should(BeFalse())
		})
	})
})

var _ = Describe("Adding taints and labels to nodes", func() {
	var sysCfg *dwsv1alpha2.SystemConfiguration

	taintNoSchedule := &corev1.Taint{
		Key:    nnfv1alpha1.RabbitNodeTaintKey,
		Value:  "true",
		Effect: corev1.TaintEffectNoSchedule,
	}
	taintNoExecute := &corev1.Taint{
		Key:    nnfv1alpha1.RabbitNodeTaintKey,
		Value:  "true",
		Effect: corev1.TaintEffectNoExecute,
	}

	makeNode := func(nodeName string) *corev1.Node {
		node := &corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: nodeName,
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
		return node
	}

	BeforeEach(func() {
		sysCfg = &dwsv1alpha2.SystemConfiguration{
			ObjectMeta: v1.ObjectMeta{
				Name:      "default",
				Namespace: corev1.NamespaceDefault,
			},
			Spec: dwsv1alpha2.SystemConfigurationSpec{
				StorageNodes: []dwsv1alpha2.SystemConfigurationStorageNode{
					{
						Type: "Rabbit",
						Name: "rabbit1",
					},
					{
						Type: "Rabbit",
						Name: "rabbit2",
					},
					{
						Type: "Rabbit",
						Name: "rabbit3",
					},
				},
			},
		}
		Expect(k8sClient.Create(context.TODO(), sysCfg)).To(Succeed())
	})

	AfterEach(func() {
		Expect(k8sClient.Delete(context.TODO(), sysCfg)).To(Succeed())
		Eventually(func() error { // Delete can still return the cached object. Wait until the object is no longer present
			return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(sysCfg), sysCfg)
		}).ShouldNot(Succeed())
	})

	// Wait until the Node has attained the desired labels and taints, then
	// return its metadata.resourceVersion representing that state.
	verifyTaintsAndLabels := func(node *corev1.Node) string {
		tnode := &corev1.Node{}
		Eventually(func(g Gomega) {
			g.Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(node), tnode))
			labels := tnode.GetLabels()
			g.Expect(labels).To(HaveKeyWithValue(nnfv1alpha1.RabbitNodeSelectorLabel, "true"))
			g.Expect(labels).To(HaveKeyWithValue(nnfv1alpha1.TaintsAndLabelsCompletedLabel, "true"))
			g.Expect(taints.TaintExists(tnode.Spec.Taints, taintNoSchedule)).To(BeTrue())
			g.Expect(taints.TaintExists(tnode.Spec.Taints, taintNoExecute)).To(BeFalse())
		}).Should(Succeed(), "verify failed for node %s", node.Name)
		return tnode.ObjectMeta.ResourceVersion
	}

	When("a node is added", func() {
		It("should be properly tainted and labeled", func() {
			node1 := makeNode("rabbit1")
			Expect(k8sClient.Create(context.TODO(), node1)).To(Succeed())
			By("verifying node1")
			node1ResVer1 := verifyTaintsAndLabels(node1)

			// Remove the "cleared" label from node1.
			Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(node1), node1))
			labels := node1.GetLabels()
			delete(labels, nnfv1alpha1.TaintsAndLabelsCompletedLabel)
			node1.SetLabels(labels)
			Expect(k8sClient.Update(context.TODO(), node1)).To(Succeed())
			By("verifying node1 is repaired")
			node1ResVer2 := verifyTaintsAndLabels(node1)
			Expect(node1ResVer1).ToNot(Equal(node1ResVer2))

			// Add two more nodes.
			node2 := makeNode("rabbit2")
			node3 := makeNode("rabbit3")
			Expect(k8sClient.Create(context.TODO(), node2)).To(Succeed())
			Expect(k8sClient.Create(context.TODO(), node3)).To(Succeed())

			By("verifying node2")
			_ = verifyTaintsAndLabels(node2)
			By("verifying node3")
			_ = verifyTaintsAndLabels(node3)

			By("verifying node1 was not touched when nodes 2 and 3 were added")
			node1ResVer3 := verifyTaintsAndLabels(node1)
			Expect(node1ResVer3).To(Equal(node1ResVer2))
		})

		It("should not taint and label a non-Rabbit node", func() {
			node4 := makeNode("somethingelse")
			Expect(k8sClient.Create(context.TODO(), node4)).To(Succeed())
			Consistently(func(g Gomega) {
				tnode := &corev1.Node{}
				g.Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(node4), tnode))
				labels := tnode.GetLabels()
				g.Expect(labels).ToNot(HaveKey(nnfv1alpha1.RabbitNodeSelectorLabel))
				g.Expect(labels).ToNot(HaveKey(nnfv1alpha1.TaintsAndLabelsCompletedLabel))
				g.Expect(taints.TaintExists(tnode.Spec.Taints, taintNoSchedule)).To(BeFalse())
				g.Expect(taints.TaintExists(tnode.Spec.Taints, taintNoExecute)).To(BeFalse())
			}).Should(Succeed(), "verify failed for node %s", node4.Name)
		})
	})
})
