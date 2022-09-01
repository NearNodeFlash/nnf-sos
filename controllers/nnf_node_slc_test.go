package controllers

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	dwsv1alpha1 "github.com/HewlettPackard/dws/api/v1alpha1"
	nnfv1alpha1 "github.com/NearNodeFlash/nnf-sos/api/v1alpha1"
)

var _ = Describe("System Level Controller Test", func() {
	It("Creates a Storage object for down node", func() {

		nodeName := "nnf-slc-test-node-0"

		By("Creating the Kubernetes Node")
		node := &corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name:      nodeName,
				Namespace: corev1.NamespaceDefault,
				Labels: map[string]string{
					"cray.nnf.node": "true",
				},
			},
			// By default, a node lacking any status block is not ready
		}

		Expect(k8sClient.Create(context.TODO(), node)).To(Succeed())

		By("Creating the Node's Namespace")
		namespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: nodeName,
			},
		}

		Expect(k8sClient.Create(context.TODO(), namespace)).To(Succeed())

		By("Creating the NNF Node")
		nnfNode := &nnfv1alpha1.NnfNode{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "nnf-nlc",
				Namespace: nodeName,
			},
			Spec: nnfv1alpha1.NnfNodeSpec{
				Name:  nodeName,
				State: "Enable",
			},
		}

		Expect(k8sClient.Create(context.TODO(), nnfNode)).To(Succeed())

		storage := &dwsv1alpha1.Storage{}
		Eventually(func() error {
			return k8sClient.Get(context.TODO(), types.NamespacedName{Name: nnfNode.Namespace, Namespace: corev1.NamespaceDefault}, storage)
		}).Should(Succeed(), "Create the DWS Storage object")

		Expect(storage.Data.Status).To(Equal("Offline"))

		By("Deleting the NNF Node")
		Expect(k8sClient.Delete(context.TODO(), nnfNode)).To(Succeed())

		Eventually(func() error {
			return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(storage), storage)
		}).ShouldNot(Succeed(), "Deletes the DWS Storage object")

		// Test cleanup
		Expect(k8sClient.Delete(context.TODO(), node)).To(Succeed())
		Expect(k8sClient.Delete(context.TODO(), namespace)).To(Succeed())
	})
})
