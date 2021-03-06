package controllers

import (
	"context"
	"fmt"
	"os"
	"reflect"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"

	"sigs.k8s.io/controller-runtime/pkg/client"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	dwsv1alpha1 "github.com/HewlettPackard/dws/api/v1alpha1"
	lusv1alpha1 "github.com/NearNodeFlash/lustre-fs-operator/api/v1alpha1"
	nnfv1alpha1 "github.com/NearNodeFlash/nnf-sos/api/v1alpha1"
)

// TODO:
// BeforeEach - initialize the workflow
// AfterEach - destroy the workflow

var _ = Describe("NNF Workflow Unit Tests", func() {

	var (
		key            types.NamespacedName
		workflow       *dwsv1alpha1.Workflow
		storageProfile *nnfv1alpha1.NnfStorageProfile
	)

	BeforeEach(func() {
		wfid := uuid.NewString()[0:8]

		key = types.NamespacedName{
			Name:      "nnf-workflow-" + wfid,
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

		// Create a default NnfStorageProfile for the unit tests.
		storageProfile = createBasicDefaultNnfStorageProfile()
	})

	AfterEach(func() {
		Eventually(func(g Gomega) error {
			g.Expect(k8sClient.Get(context.TODO(), key, workflow)).To(Succeed())
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

		Expect(k8sClient.Delete(context.TODO(), storageProfile)).To(Succeed())
		profExpected := &nnfv1alpha1.NnfStorageProfile{}
		Eventually(func() error { // Delete can still return the cached object. Wait until the object is no longer present
			return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(storageProfile), profExpected)
		}).ShouldNot(Succeed())
	})

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

	When("Negative tests for storage profiles", func() {
		It("Fails to achieve proposal state when the named profile cannot be found", func() {
			workflow.Spec.DWDirectives = []string{
				"#DW jobdw name=test profile=noneSuch type=lustre capacity=1GiB",
			}

			Expect(k8sClient.Create(context.TODO(), workflow)).To(Succeed(), "create workflow")
			// We need to be able to pass errors through the PersistentStorageInstance and DirectiveBreakdown
			// before we can test this
			/*
				Eventually(func() *dwsv1alpha1.WorkflowDriverStatus {
					expected := &dwsv1alpha1.Workflow{}
					k8sClient.Get(context.TODO(), key, expected)
					return getErroredDriverStatus(expected)
				}).ShouldNot(BeNil(), "have an error present")
			*/
		})

		It("Fails to achieve proposal state when a default profile cannot be found", func() {
			storageProfile.Data.Default = false
			Expect(k8sClient.Update(context.TODO(), storageProfile)).To(Succeed(), "remove default flag")

			workflow.Spec.DWDirectives = []string{
				"#DW jobdw name=test type=lustre capacity=1GiB",
			}

			Expect(k8sClient.Create(context.TODO(), workflow)).To(Succeed(), "create workflow")

			// We need to be able to pass errors through the PersistentStorageInstance and DirectiveBreakdown
			// before we can test this
			/*
				Eventually(func() *dwsv1alpha1.WorkflowDriverStatus {
					expected := &dwsv1alpha1.Workflow{}
					k8sClient.Get(context.TODO(), key, expected)
					return getErroredDriverStatus(expected)
				}).ShouldNot(BeNil(), "have an error present")
			*/
		})

		When("More than one default profile", func() {

			var storageProfile2 *nnfv1alpha1.NnfStorageProfile

			BeforeEach(func() {
				// The second profile will get a different name via the call to uuid.
				// Then we'll have two that are default.
				storageProfile2 = createBasicDefaultNnfStorageProfile()
			})

			AfterEach(func() {
				Expect(k8sClient.Delete(context.TODO(), storageProfile2)).To(Succeed())
				profExpected := &nnfv1alpha1.NnfStorageProfile{}
				Eventually(func() error { // Delete can still return the cached object. Wait until the object is no longer present
					return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(storageProfile2), profExpected)
				}).ShouldNot(Succeed())
			})

			It("Fails to achieve proposal state when more than one default profile exists", func() {
				workflow.Spec.DWDirectives = []string{
					"#DW jobdw name=test type=lustre capacity=1GiB",
				}

				Expect(k8sClient.Create(context.TODO(), workflow)).To(Succeed(), "create workflow")

				// We need to be able to pass errors through the PersistentStorageInstance and DirectiveBreakdown
				// before we can test this
				/*
					Eventually(func() *dwsv1alpha1.WorkflowDriverStatus {
						expected := &dwsv1alpha1.Workflow{}
						k8sClient.Get(context.TODO(), key, expected)
						return getErroredDriverStatus(expected)
					}).ShouldNot(BeNil(), "have an error present")
				*/
			})
		})
	})

	When("Positive tests for storage profiles", func() {

		profiles := []*nnfv1alpha1.NnfStorageProfile{}
		profNames := []string{}

		BeforeEach(func() {
			// Keep the underlying array memory, so the captures in AfterEach() and It() see the new values.
			profNames = profNames[:0]
			// Names to use for a batch of profiles.
			profNames = append(profNames,
				"test-"+uuid.NewString()[:8],
				"test-"+uuid.NewString()[:8],
				"test-"+uuid.NewString()[:8],
			)

			// Keep the underlying array memory, so the captures in AfterEach() and It() see the new values.
			profiles = profiles[:0]
			// Create a batch of profiles.
			for _, pn := range profNames {
				prof := basicNnfStorageProfile(pn)
				prof = createNnfStorageProfile(prof, true)
				profiles = append(profiles, prof)
			}

		})

		AfterEach(func() {
			for _, prof := range profiles {
				Expect(k8sClient.Delete(context.TODO(), prof)).To(Succeed())
			}
		})

		JustAfterEach(func() {
			By("Verify workflow achieves proposal state with pinned profile")

			Expect(k8sClient.Create(context.TODO(), workflow)).To(Succeed(), "create workflow")

			workflowAfter := &dwsv1alpha1.Workflow{}
			Eventually(func(g Gomega) error {
				g.Expect(k8sClient.Get(context.TODO(), key, workflowAfter)).To(Succeed())
				if (workflowAfter.Status.Ready == true) && (workflowAfter.Status.State == dwsv1alpha1.StateProposal.String()) && (getErroredDriverStatus(workflowAfter) == nil) {
					return nil
				}
				return fmt.Errorf("ready state not achieved")
			}).Should(Succeed(), "achieve ready state")

			By("Verify that one DirectiveBreakdown was created")
			Expect(workflowAfter.Status.DirectiveBreakdowns).To(HaveLen(1))
			By("Verify its pinned profile")
			pinnedName, pinnedNamespace := getStorageReferenceNameFromWorkflowActual(workflowAfter, 0)
			// The profile begins life with the workflow as the owner.
			Expect(verifyPinnedProfile(context.TODO(), k8sClient, pinnedNamespace, pinnedName)).To(Succeed())
		})

		It("Implicit use of default profile", func() {
			workflow.Spec.DWDirectives = []string{
				"#DW jobdw name=test type=lustre capacity=1GiB",
			}
		})

		It("Named profile, which happens to also be the default", func() {
			workflow.Spec.DWDirectives = []string{
				fmt.Sprintf("#DW jobdw name=test profile=%s type=lustre capacity=1GiB", storageProfile.GetName()),
			}
		})

		It("Named profile", func() {
			workflow.Spec.DWDirectives = []string{
				fmt.Sprintf("#DW jobdw name=test profile=%s type=lustre capacity=1GiB", profNames[0]),
			}
		})
	})

	When("Using bad copy_in/copy_out directives", func() {

		It("Fails missing or malformed jobdw reference", func() {
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

		It("Fails missing or malformed persistentdw reference", func() {
			workflow.Spec.DWDirectives = []string{
				"#DW create_persistent name=test type=lustre capacity=1GiB",
				"#DW copy_in source=/lus/maui/my-file.in destination=$PERSISTENT_DW_INCORRECT/my-file.out",
			}

			Expect(k8sClient.Create(context.TODO(), workflow)).To(Succeed(), "create workflow")

			Eventually(func() *dwsv1alpha1.WorkflowDriverStatus {
				expected := &dwsv1alpha1.Workflow{}
				k8sClient.Get(context.TODO(), key, expected)
				return getErroredDriverStatus(expected)
			}).ShouldNot(BeNil(), "have an error present")
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
			}).Should(Succeed(), "wait for create to occur")

			Eventually(func() bool {
				Expect(k8sClient.Get(context.TODO(), key, workflow)).To(Succeed())
				return workflow.Status.Ready
			}).Should(BeTrue(), "waiting for ready after create")

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
					Name:      "maui",
					MountRoot: "/lus/maui",
					MgsNids:   []string{"10.0.0.1@tcp"},
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

				By("transition to data in state")
				Eventually(func(g Gomega) error {
					g.Expect(k8sClient.Get(context.TODO(), key, workflow)).To(Succeed())
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
				Expect(dm.Spec.Source.Storage).ToNot(BeNil())
				Expect(*dm.Spec.Source.Storage).To(MatchFields(IgnoreExtras,
					Fields{
						"Kind":      Equal(reflect.TypeOf(lusv1alpha1.LustreFileSystem{}).Name()),
						"Name":      Equal(lustre.ObjectMeta.Name),
						"Namespace": Equal(lustre.Namespace),
					}))

				Expect(dm.Spec.Destination.Path).To(Equal("/my-file.out"))
				Expect(dm.Spec.Destination.Storage).ToNot(BeNil())
				Expect(*dm.Spec.Destination.Storage).To(MatchFields(IgnoreExtras,
					Fields{
						"Kind":      Equal(reflect.TypeOf(nnfv1alpha1.NnfStorage{}).Name()),
						"Name":      Equal(fmt.Sprintf("%s-%d", workflow.Name, 0)),
						"Namespace": Equal(workflow.Namespace),
					}))
			})
		})

		When("using $PERSISTENT_DW_ references", func() {
			persistentStorageName := "my-persistent-storage"

			createPersistentStorageInstance := func() {
				By("Fabricate the persistent storage instance")

				// Create a persistent storage instance to be found
				psi := &dwsv1alpha1.PersistentStorageInstance{
					TypeMeta:   metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{Name: persistentStorageName, Namespace: workflow.Namespace},
					Spec: dwsv1alpha1.PersistentStorageInstanceSpec{
						Name:   persistentStorageName,
						FsType: "xfs",
						// DWDirective: "some directive",
					},
					Status: dwsv1alpha1.PersistentStorageInstanceStatus{},
				}
				Expect(k8sClient.Create(context.TODO(), psi)).To(Succeed())

				// persistentdw directive checks that the nnfStorage associated with the
				// PersistentStorageInstance (by name) is present and its Status is 'Ready'
				// For this test, create such an nnfStorage so the datamovement pieces can
				// operate.
				// An alternative is to create a workflow with 'create_persistent'
				// as its directive and actually create the full-blown persistent instance.. (painful)
				nnfStorage := &nnfv1alpha1.NnfStorage{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name:      persistentStorageName,
						Namespace: workflow.Namespace,
					},
					Spec: nnfv1alpha1.NnfStorageSpec{
						FileSystemType: "xfs",
						AllocationSets: []nnfv1alpha1.NnfStorageAllocationSetSpec{},
					},
					Status: nnfv1alpha1.NnfStorageStatus{
						MgsNode: "",
						AllocationSets: []nnfv1alpha1.NnfStorageAllocationSetStatus{{
							Status:          "Ready",
							Health:          "OK",
							Reason:          "",
							AllocationCount: 0,
						}},
					},
				}
				Expect(k8sClient.Create(context.TODO(), nnfStorage)).To(Succeed())
			}

			deletePersistentStorageInstance := func() {
				By("Fabricate the nnfStorage as it the persistent storage instance exists")

				// Delete persistent storage instance
				psi := &dwsv1alpha1.PersistentStorageInstance{
					ObjectMeta: metav1.ObjectMeta{Name: persistentStorageName, Namespace: workflow.Namespace},
				}
				Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(psi), psi)).To(Succeed())
				Expect(k8sClient.Delete(context.TODO(), psi)).Should(Succeed())

				nnfStorage := &nnfv1alpha1.NnfStorage{
					ObjectMeta: metav1.ObjectMeta{Name: persistentStorageName, Namespace: workflow.Namespace},
				}

				Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(nnfStorage), nnfStorage)).To(Succeed())
				Expect(k8sClient.Delete(context.TODO(), nnfStorage)).Should(Succeed())
			}

			BeforeEach(func() {
				createPersistentStorageInstance()

				workflow.Spec.DWDirectives = []string{
					fmt.Sprintf("#DW persistentdw name=%s", persistentStorageName),
					fmt.Sprintf("#DW copy_in source=/lus/maui/my-file.in destination=$PERSISTENT_DW_%s/my-persistent-file.out", persistentStorageName),
				}
			})

			// Create/Delete the "nnf-system" namespace as part of the test life-cycle; the persistent storage instances are
			// placed in the "nnf-system" namespace so it must be present.
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "nnf-system",
				},
			}

			BeforeEach(func() {
				Expect(k8sClient.Create(context.TODO(), ns)).Should(Succeed())
			})

			AfterEach(func() {
				deletePersistentStorageInstance()

				Expect(k8sClient.Delete(context.TODO(), ns)).Should(Succeed())
			})

			It("transitions to data movement", func() {

				Eventually(func() error {
					Expect(k8sClient.Get(context.TODO(), key, workflow)).To(Succeed())
					workflow.Spec.DesiredState = dwsv1alpha1.StateDataIn.String()
					return k8sClient.Update(context.TODO(), workflow)
				}).Should(Succeed(), "transition desired state to data_in")

				dm := &nnfv1alpha1.NnfDataMovement{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("%s-%d", workflow.Name, 1),
						Namespace: workflow.Namespace,
					},
				}

				Eventually(func() error {
					return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(dm), dm)
				}).Should(Succeed(), "data movement resource created")

				Expect(dm.Spec.Source.Path).To(Equal(lustre.Spec.MountRoot + "/my-file.in"))
				Expect(dm.Spec.Source.Storage).ToNot(BeNil())
				Expect(*dm.Spec.Source.Storage).To(MatchFields(IgnoreExtras,
					Fields{
						"Kind":      Equal(reflect.TypeOf(lusv1alpha1.LustreFileSystem{}).Name()),
						"Name":      Equal(lustre.ObjectMeta.Name),
						"Namespace": Equal(lustre.Namespace),
					}))

				Expect(dm.Spec.Destination.Path).To(Equal("/my-persistent-file.out"))
				Expect(dm.Spec.Destination.Storage).ToNot(BeNil())
				Expect(*dm.Spec.Destination.Storage).To(MatchFields(IgnoreExtras,
					Fields{
						"Kind":      Equal(reflect.TypeOf(nnfv1alpha1.NnfStorage{}).Name()),
						"Name":      Equal(persistentStorageName),
						"Namespace": Equal(workflow.Namespace),
					}))
			})
		})
	}) // When("Using copy_in directives", func()
})

var _ = Describe("NnfStorageProfile Webhook test", func() {
	// The nnfstorageprofile_webhook_test.go covers testing of the webhook.
	// This spec exists only to verify that the webhook is also running for
	// the controller tests.
	It("Fails to create an invalid profile, to verify that the webhook is installed", func() {
		profileInvalid := basicNnfStorageProfile("an-invalid-profile")
		profileInvalid.Data.LustreStorage.ExternalMGS = []string{
			"10.0.0.1@tcp",
		}
		profileInvalid.Data.LustreStorage.CombinedMGTMDT = true
		Expect(createNnfStorageProfile(profileInvalid, false)).To(BeNil())
	})
})
