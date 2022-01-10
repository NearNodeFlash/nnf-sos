package controllers

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	nnf "github.hpe.com/hpe/hpc-rabsw-nnf-ec/pkg"
	nnfv1alpha1 "github.hpe.com/hpe/hpc-rabsw-nnf-sos/api/v1alpha1"
)

var _ = PDescribe("NNF Node Storage Controller Test", func() {
	var (
		key     types.NamespacedName
		storage *nnfv1alpha1.NnfNodeStorage
	)

	BeforeEach(func() {
		c := nnf.NewController(nnf.NewMockOptions())
		Expect(c.Init(nil)).To(Succeed())

		// TODO: Eventually the controller should go ready; convert this to a poll once the API is available
		time.Sleep(1 * time.Second)
	})

	BeforeEach(func() {
		key = types.NamespacedName{
			Name:      "nnf-node-storage",
			Namespace: corev1.NamespaceDefault,
		}

		storage = &nnfv1alpha1.NnfNodeStorage{
			ObjectMeta: metav1.ObjectMeta{
				Name:      key.Name,
				Namespace: key.Namespace,
			},
			Spec: nnfv1alpha1.NnfNodeStorageSpec{
				Count:    1,
				Capacity: 1024 * 1024 * 1024,
			},
		}
	})

	JustBeforeEach(func() {
		Expect(k8sClient.Create(context.TODO(), storage)).To(Succeed())

		Eventually(func() error {
			expected := &nnfv1alpha1.NnfNodeStorage{}
			return k8sClient.Get(context.TODO(), key, expected)
		}, "3s", "1s").Should(Succeed(), "expected return after create. key: "+key.String())
	})

	AfterEach(func() {
		expected := &nnfv1alpha1.NnfNodeStorage{}
		Expect(k8sClient.Get(context.TODO(), key, expected)).To(Succeed())
		Expect(k8sClient.Delete(context.TODO(), expected)).To(Succeed())
	})

	When("creating xfs storage", func() {
		BeforeEach(func() {
			storage.Spec.FileSystemType = "xfs"
		})

		It("is successful", func() {
			expected := &nnfv1alpha1.NnfNodeStorage{}
			Expect(k8sClient.Get(context.TODO(), key, expected)).To(Succeed())
		})
	})

	When("creating lustre storage", func() {
		BeforeEach(func() {
			storage.Spec.FileSystemType = "lustre"

			storage.Spec.LustreStorage = nnfv1alpha1.LustreStorageSpec{
				FileSystemName: "test",
				StartIndex:     0,
				MgsNode:        "test",
				TargetType:     "MGT",
				BackFs:         "zfs",
			}
		})

		It("is successful", func() {
			expected := &nnfv1alpha1.NnfNodeStorage{}
			Expect(k8sClient.Get(context.TODO(), key, expected)).To(Succeed())
		})
	})
})
