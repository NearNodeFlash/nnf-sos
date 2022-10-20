package controllers

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	dwsv1alpha1 "github.com/HewlettPackard/dws/api/v1alpha1"
	nnfv1alpha1 "github.com/NearNodeFlash/nnf-sos/api/v1alpha1"
)

var _ = Describe("PersistentStorage test", func() {
	var (
		storageProfile *nnfv1alpha1.NnfStorageProfile
	)

	BeforeEach(func() {
		// Create a default NnfStorageProfile for the unit tests.
		storageProfile = createBasicDefaultNnfStorageProfile()
	})

	AfterEach(func() {
		Expect(k8sClient.Delete(context.TODO(), storageProfile)).To(Succeed())
		profExpected := &nnfv1alpha1.NnfStorageProfile{}
		Eventually(func() error { // Delete can still return the cached object. Wait until the object is no longer present
			return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(storageProfile), profExpected)
		}).ShouldNot(Succeed())
	})

	It("Creates a PersistentStorageInstance", func() {
		By("Creating a PersistentStorageInstance")
		persistentStorage := &dwsv1alpha1.PersistentStorageInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "persistent-test",
				Namespace: corev1.NamespaceDefault,
			},
			Spec: dwsv1alpha1.PersistentStorageInstanceSpec{
				Name:        "persistent_lustre",
				DWDirective: "#DW create_persistent name=persistent_lustre type=lustre capacity=1GiB",
				FsType:      "lustre",
				UserID:      999,
				State:       dwsv1alpha1.PSIStateActive,
			},
		}

		Expect(k8sClient.Create(context.TODO(), persistentStorage)).To(Succeed())
		Eventually(func(g Gomega) dwsv1alpha1.PersistentStorageInstanceState {
			g.Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(persistentStorage), persistentStorage)).To(Succeed())
			return persistentStorage.Status.State
		}).Should(Equal(dwsv1alpha1.PSIStateCreating))

		servers := &dwsv1alpha1.Servers{
			ObjectMeta: metav1.ObjectMeta{
				Name:      persistentStorage.GetName(),
				Namespace: persistentStorage.GetNamespace(),
			},
		}
		Eventually(func() error {
			return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(servers), servers)
		}).Should(Succeed(), "Create the DWS Servers Resource")

		pinnedStorageProfile := &nnfv1alpha1.NnfStorageProfile{
			ObjectMeta: metav1.ObjectMeta{
				Name:      persistentStorage.GetName(),
				Namespace: persistentStorage.GetNamespace(),
			},
		}
		Eventually(func() error {
			return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(pinnedStorageProfile), pinnedStorageProfile)
		}).Should(Succeed(), "Create the pinned StorageProfile Resource")

		By("Adding consumer reference to prevent destroying")
		Eventually(func(g Gomega) error {
			g.Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(persistentStorage), persistentStorage)).To(Succeed())
			persistentStorage.Spec.ConsumerReferences = []corev1.ObjectReference{{
				Name:      "Fake",
				Namespace: "Reference",
			}}

			return k8sClient.Update(context.TODO(), persistentStorage)
		}).Should(Succeed(), "Add fake consumer reference")

		By("Marking the persistentStorage as destroying")
		Eventually(func(g Gomega) error {
			g.Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(persistentStorage), persistentStorage)).To(Succeed())
			persistentStorage.Spec.State = dwsv1alpha1.PSIStateDestroying

			return k8sClient.Update(context.TODO(), persistentStorage)
		}).Should(Succeed(), "Set as destroying")

		Eventually(func(g Gomega) dwsv1alpha1.PersistentStorageInstanceState {
			g.Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(persistentStorage), persistentStorage)).To(Succeed())
			return persistentStorage.Status.State
		}).Should(Equal(dwsv1alpha1.PSIStateCreating))

		By("Removing consumer reference")
		Eventually(func(g Gomega) error {
			g.Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(persistentStorage), persistentStorage)).To(Succeed())
			persistentStorage.Spec.ConsumerReferences = []corev1.ObjectReference{}

			return k8sClient.Update(context.TODO(), persistentStorage)
		}).Should(Succeed(), "Remove fake consumer reference")

		Eventually(func(g Gomega) dwsv1alpha1.PersistentStorageInstanceState {
			g.Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(persistentStorage), persistentStorage)).To(Succeed())
			return persistentStorage.Status.State
		}).Should(Equal(dwsv1alpha1.PSIStateDestroying))

		By("Deleting the PersistentStorageInstance")
		Expect(k8sClient.Delete(context.TODO(), persistentStorage)).To(Succeed())
		Eventually(func() error {
			return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(pinnedStorageProfile), pinnedStorageProfile)
		}).ShouldNot(Succeed())

		Eventually(func() error {
			return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(servers), servers)
		}).ShouldNot(Succeed())
	})

})
