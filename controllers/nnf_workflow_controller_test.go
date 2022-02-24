package controllers

import (
	"context"
	"fmt"
	"os"
	"reflect"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"

	"sigs.k8s.io/controller-runtime/pkg/client"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	dwsv1alpha1 "github.hpe.com/hpe/hpc-dpm-dws-operator/api/v1alpha1"
	lusv1alpha1 "github.hpe.com/hpe/hpc-rabsw-lustre-fs-operator/api/v1alpha1"
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
		}).Should(Succeed(), "teardown")

		Eventually(func() bool {
			Expect(k8sClient.Get(context.TODO(), key, workflow)).To(Succeed())
			return workflow.Status.State == workflow.Spec.DesiredState
		}).Should(BeTrue(), "reach desired teardown state")

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

	When("Using bad copy_in directives", func() {

		getErroredDriverStatus := func(workflow *dwsv1alpha1.Workflow) *dwsv1alpha1.WorkflowDriverStatus {
			driverID := os.Getenv("DWS_DRIVER_ID")
			for _, driver := range workflow.Status.Drivers {
				if driver.DriverID == driverID {
					if driver.Reason == "error" {
						return &driver
					}
				}
			}
			return nil
		}

		It("Fails missing or malformed job-dw reference", func() {
			workflow.Spec.DWDirectives = []string{
				"#DW jobdw name=test type=lustre capacity=1GiB",
				"#DW copy_in source=/lus/maui/my-file.in destination=$JOB_DW_INCORRECT/my-file.out",
			}

			Expect(k8sClient.Create(context.TODO(), workflow)).To(Succeed(), "create workflow")

			Eventually(func() *dwsv1alpha1.WorkflowDriverStatus {
				expected := &dwsv1alpha1.Workflow{}
				k8sClient.Get(context.TODO(), key, expected)
				return getErroredDriverStatus(expected)
			}).ShouldNot(BeNil(), "have an error present")

		})

		PIt("Fails missing or malformed persistent-dw reference", func() {

		})

		It("Fails missing or malformed global lustre reference", func() {
			workflow.Spec.DWDirectives = []string{
				"#DW jobdw name=test type=lustre capacity=1GiB",
				"#DW copy_in source=/lus/INCORRECT/my-file.in destination=$JOB_DW_test/my-file.out",
			}

			Expect(k8sClient.Create(context.TODO(), workflow)).To(Succeed(), "create workflow")

			Eventually(func() *dwsv1alpha1.WorkflowDriverStatus {
				expected := &dwsv1alpha1.Workflow{}
				k8sClient.Get(context.TODO(), key, expected)
				return getErroredDriverStatus(expected)
			}).ShouldNot(BeNil(), "have an error present")
		})
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

		// Create a fake global lustre file system.
		var (
			lustre *lusv1alpha1.LustreFileSystem
		)

		BeforeEach(func() {
			lustre = &lusv1alpha1.LustreFileSystem{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "maui",
					Namespace: corev1.NamespaceDefault,
				},
				Spec: lusv1alpha1.LustreFileSystemSpec{
					MountRoot: "/lus/maui",
				},
			}
			Expect(k8sClient.Create(context.TODO(), lustre)).To(Succeed())
		})

		AfterEach(func() {
			Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(lustre), lustre)).To(Succeed())
			Expect(k8sClient.Delete(context.TODO(), lustre)).To(Succeed())
		})

		When("using $JOB_DW_ references", func() {

			BeforeEach(func() {
				workflow.Spec.DWDirectives = []string{
					"#DW jobdw name=test type=lustre capacity=1GiB",
					"#DW copy_in source=/lus/maui/my-file.in destination=$JOB_DW_test/my-file.out",
				}
			})

			It("transition to data movement", func() {

				By("creates valid job storage instance")
				storageInstance := &nnfv1alpha1.NnfJobStorageInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name:      nnfv1alpha1.NnfJobStorageInstanceMakeName(workflow.Spec.JobID, workflow.Spec.WLMID, 0),
						Namespace: key.Namespace,
					},
				}

				Eventually(func() error {
					return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(storageInstance), storageInstance)
				}, "3s").Should(Succeed(), "get job storage instance")

				// Expect the workflow has owner reference to the job storage instance; this verifies the garbage collection
				// chain is set up, but recall that GC is not running in the testenv so we can't prove it is deleted on teardown.
				// See https://book.kubebuilder.io/reference/envtest.html#testing-considerations
				controller := true
				blockOwnerDeletion := true
				ownerRef := metav1.OwnerReference{
					Kind:               "Workflow",
					APIVersion:         dwsv1alpha1.GroupVersion.String(),
					UID:                workflow.GetUID(),
					Name:               workflow.GetName(),
					Controller:         &controller,
					BlockOwnerDeletion: &blockOwnerDeletion,
				}

				Expect(storageInstance.ObjectMeta.OwnerReferences).To(ContainElement(ownerRef))

				By("transition to data in state")
				Eventually(func() error {
					Expect(k8sClient.Get(context.TODO(), key, workflow)).To(Succeed())
					workflow.Spec.DesiredState = dwsv1alpha1.StateDataIn.String()
					return k8sClient.Update(context.TODO(), workflow)
				}).Should(Succeed(), "update to data_in")

				By("creates the data movement resource")
				dm := &nnfv1alpha1.NnfDataMovement{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("%s-%d", workflow.Name, 1),
						Namespace: workflow.Namespace,
					},
				}

				Eventually(func() error {
					return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(dm), dm)
				}).Should(Succeed(), "expect data movement resource")

				Expect(dm.ObjectMeta.OwnerReferences).To(ContainElement(ownerRef))

				Expect(dm.Spec.Source.Path).To(Equal(lustre.Spec.MountRoot + "/my-file.in"))
				Expect(*dm.Spec.Source.StorageInstance).To(MatchFields(IgnoreExtras,
					Fields{
						"Kind":      Equal(reflect.TypeOf(lusv1alpha1.LustreFileSystem{}).Name()),
						"Name":      Equal(lustre.ObjectMeta.Name),
						"Namespace": Equal(lustre.Namespace),
					}))

				Expect(dm.Spec.Destination.Path).To(Equal("/my-file.out"))
				Expect(*dm.Spec.Destination.StorageInstance).To(MatchFields(IgnoreExtras,
					Fields{
						"Kind": Equal(reflect.TypeOf(nnfv1alpha1.NnfJobStorageInstance{}).Name()),
						"Name": HaveSuffix("-0"), // Should reference the #DW that generated the job storage instance
					}))
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
