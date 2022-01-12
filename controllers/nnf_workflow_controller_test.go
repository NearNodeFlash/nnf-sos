package controllers

import (
	"context"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	dwsv1alpha1 "github.hpe.com/hpe/hpc-dpm-dws-operator/api/v1alpha1"
	nnfv1alpha1 "github.hpe.com/hpe/hpc-rabsw-nnf-sos/api/v1alpha1"
)

// TODO:
// BeforeEach - initialize the workflow
// AfterEach - destroy the workflow

var _ = Describe("NNF Workflow Unit Tests", func() {

	var (
		key      types.NamespacedName
		workflow *dwsv1alpha1.Workflow
	)

	BeforeEach(func() {
		key = types.NamespacedName{
			Name:      "nnf-workflow",
			Namespace: corev1.NamespaceDefault,
		}

		workflow = &dwsv1alpha1.Workflow{
			ObjectMeta: metav1.ObjectMeta{
				Name:      key.Name,
				Namespace: key.Namespace,
			},
			Spec: dwsv1alpha1.WorkflowSpec{
				DesiredState: dwsv1alpha1.StateProposal.String(),
				JobID:        0,
				WLMID:        uuid.NewString(),
			},
		}

		expected := &dwsv1alpha1.Workflow{}
		Expect(k8sClient.Get(context.TODO(), key, expected)).ToNot(Succeed())
	})

	AfterEach(func() {
		Eventually(func() error {
			Expect(k8sClient.Get(context.TODO(), key, workflow)).To(Succeed())
			workflow.Spec.DesiredState = dwsv1alpha1.StateTeardown.String()
			return k8sClient.Update(context.TODO(), workflow)
		}).Should(Succeed())

		Eventually(func() bool {
			Expect(k8sClient.Get(context.TODO(), key, workflow)).To(Succeed())
			return workflow.Status.State == workflow.Spec.DesiredState
		}).Should(BeTrue())

		Eventually(func() bool {
			Expect(k8sClient.Get(context.TODO(), key, workflow)).To(Succeed())
			return workflow.Status.Ready
		}).Should(BeTrue())

		Expect(k8sClient.Delete(context.TODO(), workflow)).To(Succeed())

		expected := &dwsv1alpha1.Workflow{}
		Eventually(func() error { // Delete can still return the cached object. Wait until the object is no longer present
			return k8sClient.Get(context.TODO(), key, expected)
		}).ShouldNot(Succeed())
	})

	When("Using copy_in directives", func() {

		JustBeforeEach(func() {
			Expect(k8sClient.Create(context.TODO(), workflow)).To(Succeed(), "create workflow")

			// Kubernetes isn't always returning the object right away; we don't fully understand why at this point; need to wait until it responds with a valid object
			Eventually(func() error {
				return k8sClient.Get(context.TODO(), key, workflow)
			}, "3s", "1s").Should(Succeed(), "wait for create to occurr")

			Eventually(func() bool {
				Expect(k8sClient.Get(context.TODO(), key, workflow)).To(Succeed())
				return workflow.Status.Ready
			}, "3s").Should(BeTrue(), "waiting for ready after create")

			workflow.Spec.DesiredState = dwsv1alpha1.StateSetup.String()
			Expect(k8sClient.Update(context.TODO(), workflow)).To(Succeed())
			Eventually(func() bool {
				Expect(k8sClient.Get(context.TODO(), key, workflow)).To(Succeed())
				return workflow.Status.Ready && workflow.Status.State == workflow.Spec.DesiredState
			}).Should(BeTrue(), "transition through setup")
		})

		When("using $JOB_DW_ references", func() {
			BeforeEach(func() {
				workflow.Spec.DWDirectives = []string{
					"#DW jobdw name=test type=lustre capacity=1GiB",
					"#DW copy_in source=/lus/maui/my-file destination=$JOB_DW_test",
				}
			})

			It("creates valid job storage instance", func() {
				Eventually(func() error {
					storageInstanceKey := types.NamespacedName{
						Name:      nnfv1alpha1.NnfJobStorageInstanceMakeName(workflow.Spec.JobID, workflow.Spec.WLMID, 0),
						Namespace: key.Namespace,
					}
					storageInstance := &nnfv1alpha1.NnfJobStorageInstance{}
					return k8sClient.Get(context.TODO(), storageInstanceKey, storageInstance)
				}, "3s").Should(Succeed(), "get job storage instance")

				// TODO: Expect the workflow has owner reference to the job storage instance; this verifies the garbage collection
				// chain is set up, but recall that GC is not running in the testenv so we can't prove it is deleted on teardown.
				// See https://book.kubebuilder.io/reference/envtest.html#testing-considerations

			})

			PIt("transitions through data_in", func() {
				Eventually(func() error {
					Expect(k8sClient.Get(context.TODO(), key, workflow)).To(Succeed())
					workflow.Spec.DesiredState = dwsv1alpha1.StateDataIn.String()
					return k8sClient.Update(context.TODO(), workflow)
				}).Should(Succeed(), "update to data_in")

				Eventually(func() bool {
					Expect(k8sClient.Get(context.TODO(), key, workflow)).To(Succeed())
					return workflow.Status.Ready && workflow.Status.State == workflow.Spec.DesiredState
				}, "3s").Should(BeTrue(), "go ready")
			})
		})

		PWhen("using $PERSISTENT_DW_ references", func() {
			BeforeEach(func() {
				workflow.Spec.DWDirectives = []string{
					"#DW create_persistent TODO",
					"#DW copy_in source=/lus/maui/ destination=$PERSISTENT_DW_TODO",
				}
			})
		})
	})
})
