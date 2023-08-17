package controllers

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	dwsv1alpha2 "github.com/HewlettPackard/dws/api/v1alpha2"
	nnfv1alpha1 "github.com/NearNodeFlash/nnf-sos/api/v1alpha1"
)

var _ = Describe("DirectiveBreakdown test", func() {
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

	It("Creates a DirectiveBreakdown with a jobdw", func() {
		By("Creating a DirectiveBreakdown")
		directiveBreakdown := &dwsv1alpha2.DirectiveBreakdown{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "jobdw-test",
				Namespace: corev1.NamespaceDefault,
			},
			Spec: dwsv1alpha2.DirectiveBreakdownSpec{
				Directive: "#DW jobdw name=jobdw-xfs type=xfs capacity=1GiB",
			},
		}

		Expect(k8sClient.Create(context.TODO(), directiveBreakdown)).To(Succeed())

		Eventually(func(g Gomega) bool {
			g.Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(directiveBreakdown), directiveBreakdown)).To(Succeed())
			return directiveBreakdown.Status.Ready
		}).Should(BeTrue())

		servers := &dwsv1alpha2.Servers{
			ObjectMeta: metav1.ObjectMeta{
				Name:      directiveBreakdown.GetName(),
				Namespace: directiveBreakdown.GetNamespace(),
			},
		}
		Eventually(func() error {
			return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(servers), servers)
		}).Should(Succeed(), "Create the DWS Servers Resource")

		pinnedStorageProfile := &nnfv1alpha1.NnfStorageProfile{
			ObjectMeta: metav1.ObjectMeta{
				Name:      directiveBreakdown.GetName(),
				Namespace: directiveBreakdown.GetNamespace(),
			},
		}
		Eventually(func() error {
			return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(pinnedStorageProfile), pinnedStorageProfile)
		}).Should(Succeed(), "Create the pinned StorageProfile Resource")

		By("Deleting the DirectiveBreakdown")
		Expect(k8sClient.Delete(context.TODO(), directiveBreakdown)).To(Succeed())
		Eventually(func() error {
			return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(pinnedStorageProfile), pinnedStorageProfile)
		}).ShouldNot(Succeed())

		Eventually(func() error {
			return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(servers), servers)
		}).ShouldNot(Succeed())
	})

	It("Verifies DirectiveBreakdowns with persistent storage", func() {
		By("Creating a DirectiveBreakdown with create_persistent")
		directiveBreakdownOne := &dwsv1alpha2.DirectiveBreakdown{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "create-persistent-test",
				Namespace: corev1.NamespaceDefault,
			},
			Spec: dwsv1alpha2.DirectiveBreakdownSpec{
				Directive: "#DW create_persistent name=persistent-storage type=xfs capacity=1GiB",
			},
		}

		Expect(k8sClient.Create(context.TODO(), directiveBreakdownOne)).To(Succeed())

		Eventually(func(g Gomega) bool {
			g.Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(directiveBreakdownOne), directiveBreakdownOne)).To(Succeed())
			return directiveBreakdownOne.Status.Ready
		}).Should(BeTrue())

		persistentStorage := &dwsv1alpha2.PersistentStorageInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "persistent-storage",
				Namespace: directiveBreakdownOne.GetNamespace(),
			},
		}
		Eventually(func() error {
			return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(persistentStorage), persistentStorage)
		}).Should(Succeed(), "Create the PersistentStorageInstance resource")

		By("Creating a DirectiveBreakdown with persistentdw")
		directiveBreakdownTwo := &dwsv1alpha2.DirectiveBreakdown{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "use-persistent-test",
				Namespace: corev1.NamespaceDefault,
			},
			Spec: dwsv1alpha2.DirectiveBreakdownSpec{
				Directive: "#DW persistentdw name=persistent-storage",
			},
		}

		Expect(k8sClient.Create(context.TODO(), directiveBreakdownTwo)).To(Succeed())

		Eventually(func(g Gomega) bool {
			g.Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(directiveBreakdownTwo), directiveBreakdownTwo)).To(Succeed())
			return directiveBreakdownTwo.Status.Ready
		}).Should(BeTrue())

		By("Deleting the DirectiveBreakdown with persistentdw")
		Expect(k8sClient.Delete(context.TODO(), directiveBreakdownTwo)).To(Succeed())

		By("Deleting the DirectiveBreakdown with create_persistent")
		Expect(k8sClient.Delete(context.TODO(), directiveBreakdownOne)).To(Succeed())
		Eventually(func() error {
			return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(persistentStorage), persistentStorage)
		}).ShouldNot(Succeed())
	})

	It("Creates a DirectiveBreakdown with a lustre jobdw and standaloneMgtPoolName", func() {
		By("Setting standaloneMgtPoolName in the storage profile")
		Eventually(func(g Gomega) error {
			g.Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(storageProfile), storageProfile)).To(Succeed())
			storageProfile.Data.LustreStorage.StandaloneMGTPoolName = "FakePool"
			return k8sClient.Update(context.TODO(), storageProfile)
		}).Should(Succeed())

		By("Creating a DirectiveBreakdown")
		directiveBreakdown := &dwsv1alpha2.DirectiveBreakdown{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "standalone-lustre-jobdw-test",
				Namespace: corev1.NamespaceDefault,
			},
			Spec: dwsv1alpha2.DirectiveBreakdownSpec{
				Directive: "#DW jobdw name=jobdw-lustre type=lustre capacity=1GiB",
			},
		}

		Expect(k8sClient.Create(context.TODO(), directiveBreakdown)).To(Succeed())

		Eventually(func(g Gomega) error {
			g.Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(directiveBreakdown), directiveBreakdown)).To(Succeed())
			return directiveBreakdown.Status.Error
		}).ShouldNot(BeNil())
	})

	It("Creates a DirectiveBreakdown with an xfs jobdw and standaloneMgtPoolName", func() {
		By("Setting standaloneMgtPoolName in the storage profile")
		Eventually(func(g Gomega) error {
			g.Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(storageProfile), storageProfile)).To(Succeed())
			storageProfile.Data.LustreStorage.StandaloneMGTPoolName = "FakePool"
			return k8sClient.Update(context.TODO(), storageProfile)
		}).Should(Succeed())

		By("Creating a DirectiveBreakdown")
		directiveBreakdown := &dwsv1alpha2.DirectiveBreakdown{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "standalone-xfs-jobdw-test",
				Namespace: corev1.NamespaceDefault,
			},
			Spec: dwsv1alpha2.DirectiveBreakdownSpec{
				Directive: "#DW jobdw name=jobdw-xfs type=xfs capacity=1GiB",
			},
		}

		Expect(k8sClient.Create(context.TODO(), directiveBreakdown)).To(Succeed())

		Eventually(func(g Gomega) bool {
			g.Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(directiveBreakdown), directiveBreakdown)).To(Succeed())
			return directiveBreakdown.Status.Ready
		}).Should(BeTrue())
	})

	It("Creates a DirectiveBreakdown with a create_persistent and standaloneMgtPoolName", func() {
		By("Setting standaloneMgtPoolName in the storage profile")
		Eventually(func(g Gomega) error {
			g.Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(storageProfile), storageProfile)).To(Succeed())
			storageProfile.Data.LustreStorage.StandaloneMGTPoolName = "FakePool"
			return k8sClient.Update(context.TODO(), storageProfile)
		}).Should(Succeed())

		By("Creating a DirectiveBreakdown")
		directiveBreakdown := &dwsv1alpha2.DirectiveBreakdown{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "standalone-lustre-persistent-test",
				Namespace: corev1.NamespaceDefault,
			},
			Spec: dwsv1alpha2.DirectiveBreakdownSpec{
				Directive: "#DW create_persistent name=persistent-lustre type=lustre capacity=1GiB",
			},
		}

		Expect(k8sClient.Create(context.TODO(), directiveBreakdown)).To(Succeed())

		Eventually(func(g Gomega) bool {
			g.Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(directiveBreakdown), directiveBreakdown)).To(Succeed())
			return directiveBreakdown.Status.Ready
		}).Should(BeTrue())
	})
})
