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
				DesiredState: dwsv1alpha1.StateProposal,
				JobID:        0,
				WLMID:        uuid.NewString(),
			},
		}

		expected := &dwsv1alpha1.Workflow{}
		Expect(k8sClient.Get(context.TODO(), key, expected)).ToNot(Succeed())

		// Create a default NnfStorageProfile for the unit tests.
		storageProfile = createBasicDefaultNnfStorageProfile()

		DeferCleanup(os.Setenv, "RABBIT_TEST_ENV_BYPASS_SERVER_STORAGE_CHECK", os.Getenv("RABBIT_TEST_ENV_BYPASS_SERVER_STORAGE_CHECK"))
	})

	AfterEach(func() {
		Eventually(func(g Gomega) error {
			g.Expect(k8sClient.Get(context.TODO(), key, workflow)).To(Succeed())
			workflow.Spec.DesiredState = dwsv1alpha1.StateTeardown
			return k8sClient.Update(context.TODO(), workflow)
		}).Should(Succeed(), "teardown")

		Eventually(func(g Gomega) bool {
			g.Expect(k8sClient.Get(context.TODO(), key, workflow)).To(Succeed())
			return workflow.Status.Ready && workflow.Status.State == dwsv1alpha1.StateTeardown
		}).Should(BeTrue(), "reach desired teardown state")

		Expect(k8sClient.Delete(context.TODO(), workflow)).To(Succeed())

		Eventually(func() error { // Delete can still return the cached object. Wait until the object is no longer present
			expected := &dwsv1alpha1.Workflow{}
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
				if driver.Status == dwsv1alpha1.StatusError {
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
				if (workflowAfter.Status.Ready == true) && (workflowAfter.Status.State == dwsv1alpha1.StateProposal) && (getErroredDriverStatus(workflowAfter) == nil) {
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

			Eventually(func(g Gomega) bool {
				g.Expect(k8sClient.Get(context.TODO(), key, workflow)).To(Succeed())
				return workflow.Status.Ready
			}).Should(BeTrue(), "waiting for ready after create")

			workflow.Spec.DesiredState = dwsv1alpha1.StateSetup
			Expect(k8sClient.Update(context.TODO(), workflow)).To(Succeed())

			Eventually(func(g Gomega) bool {
				g.Expect(k8sClient.Get(context.TODO(), key, workflow)).To(Succeed())
				return workflow.Status.Ready && workflow.Status.State == dwsv1alpha1.StateSetup
			}).Should(BeTrue(), "transition through setup")
		})

		// Create a fake global lustre file system.
		var (
			lustre *lusv1alpha1.LustreFileSystem
		)

		BeforeEach(func() {
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: nnfv1alpha1.DataMovementNamespace,
				},
			}

			k8sClient.Create(context.TODO(), ns) // Ignore errors as namespace may be created from other tests
		})

		BeforeEach(func() {
			lustre = &lusv1alpha1.LustreFileSystem{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "maui",
					Namespace: corev1.NamespaceDefault,
				},
				Spec: lusv1alpha1.LustreFileSystemSpec{
					Name:      "maui",
					MountRoot: "/lus/maui",
					MgsNids:   "10.0.0.1@tcp",
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

				By("transition to data in state")
				Eventually(func(g Gomega) error {
					g.Expect(k8sClient.Get(context.TODO(), key, workflow)).To(Succeed())
					workflow.Spec.DesiredState = dwsv1alpha1.StateDataIn
					return k8sClient.Update(context.TODO(), workflow)
				}).Should(Succeed(), "update to DataIn")

				By("creates the data movement resource")
				dm := &nnfv1alpha1.NnfDataMovement{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("%s-%d", workflow.Name, 1),
						Namespace: nnfv1alpha1.DataMovementNamespace,
					},
				}

				Eventually(func() error {
					return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(dm), dm)
				}).Should(Succeed(), "expect data movement resource")

				Expect(dm.Spec.Source.Path).To(Equal(lustre.Spec.MountRoot + "/my-file.in"))
				Expect(dm.Spec.Source.StorageReference).ToNot(BeNil())
				Expect(dm.Spec.Source.StorageReference).To(MatchFields(IgnoreExtras,
					Fields{
						"Kind":      Equal(reflect.TypeOf(lusv1alpha1.LustreFileSystem{}).Name()),
						"Name":      Equal(lustre.ObjectMeta.Name),
						"Namespace": Equal(lustre.Namespace),
					}))

				Expect(dm.Spec.Destination.Path).To(Equal(buildMountPath(workflow, 1) + "/my-file.out"))
				Expect(dm.Spec.Destination.StorageReference).ToNot(BeNil())
				Expect(dm.Spec.Destination.StorageReference).To(MatchFields(IgnoreExtras,
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
						Name:        persistentStorageName,
						FsType:      "lustre",
						DWDirective: workflow.Spec.DWDirectives[0],
						State:       dwsv1alpha1.PSIStateActive,
					},
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
						FileSystemType: "lustre",
						AllocationSets: []nnfv1alpha1.NnfStorageAllocationSetSpec{},
					},
					Status: nnfv1alpha1.NnfStorageStatus{
						MgsNode: "",
						AllocationSets: []nnfv1alpha1.NnfStorageAllocationSetStatus{{
							Status:          "Ready",
							Health:          "OK",
							Error:           "",
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
				workflow.Spec.DWDirectives = []string{
					fmt.Sprintf("#DW persistentdw name=%s", persistentStorageName),
					fmt.Sprintf("#DW copy_in source=/lus/maui/my-file.in destination=$PERSISTENT_DW_%s/my-persistent-file.out", persistentStorageName),
				}

				createPersistentStorageInstance()
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

				Eventually(func(g Gomega) error {
					g.Expect(k8sClient.Get(context.TODO(), key, workflow)).To(Succeed())
					workflow.Spec.DesiredState = dwsv1alpha1.StateDataIn
					return k8sClient.Update(context.TODO(), workflow)
				}).Should(Succeed(), "transition desired state to DataIn")

				dm := &nnfv1alpha1.NnfDataMovement{
					ObjectMeta: metav1.ObjectMeta{
						Name:      indexedResourceName(workflow, 1),
						Namespace: nnfv1alpha1.DataMovementNamespace,
					},
				}

				Eventually(func() error {
					return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(dm), dm)
				}).Should(Succeed(), "data movement resource created")

				Expect(dm.Spec.Source.Path).To(Equal(lustre.Spec.MountRoot + "/my-file.in"))
				Expect(dm.Spec.Source.StorageReference).ToNot(BeNil())
				Expect(dm.Spec.Source.StorageReference).To(MatchFields(IgnoreExtras,
					Fields{
						"Kind":      Equal(reflect.TypeOf(lusv1alpha1.LustreFileSystem{}).Name()),
						"Name":      Equal(lustre.ObjectMeta.Name),
						"Namespace": Equal(lustre.Namespace),
					}))

				Expect(dm.Spec.Destination.Path).To(Equal(buildMountPath(workflow, 1) + "/my-persistent-file.out"))
				Expect(dm.Spec.Destination.StorageReference).ToNot(BeNil())
				Expect(dm.Spec.Destination.StorageReference).To(MatchFields(IgnoreExtras,
					Fields{
						"Kind":      Equal(reflect.TypeOf(nnfv1alpha1.NnfStorage{}).Name()),
						"Name":      Equal(persistentStorageName),
						"Namespace": Equal(workflow.Namespace),
					}))
			})
		})
	}) // When("Using copy_in directives", func()

	When("Using server allocation input", func() {
		const nodeName = "rabbit-node"

		var (
			storage            *dwsv1alpha1.Storage
			directiveBreakdown *dwsv1alpha1.DirectiveBreakdown
			servers            *dwsv1alpha1.Servers
		)

		JustBeforeEach(func() {
			Expect(k8sClient.Create(context.TODO(), workflow)).To(Succeed(), "create workflow")

			Eventually(func(g Gomega) bool {
				g.Expect(k8sClient.Get(context.TODO(), key, workflow)).To(Succeed())
				return workflow.Status.Ready
			}).Should(BeTrue(), "waiting for ready after create")

			directiveBreakdown = &dwsv1alpha1.DirectiveBreakdown{
				ObjectMeta: metav1.ObjectMeta{
					Name:      workflow.Status.DirectiveBreakdowns[0].Name,
					Namespace: workflow.Status.DirectiveBreakdowns[0].Namespace,
				},
			}
			Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(directiveBreakdown), directiveBreakdown)).To(Succeed())

			servers = &dwsv1alpha1.Servers{
				ObjectMeta: metav1.ObjectMeta{
					Name:      directiveBreakdown.Status.Storage.Reference.Name,
					Namespace: directiveBreakdown.Status.Storage.Reference.Namespace,
				},
			}
			Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(servers), servers)).To(Succeed())

			for _, directiveAllocationSet := range directiveBreakdown.Status.Storage.AllocationSets {
				allocationSet := dwsv1alpha1.ServersSpecAllocationSet{
					Label:          directiveAllocationSet.Label,
					AllocationSize: directiveAllocationSet.MinimumCapacity,
					Storage: []dwsv1alpha1.ServersSpecStorage{
						{
							Name:            nodeName,
							AllocationCount: 1,
						},
					},
				}
				servers.Spec.AllocationSets = append(servers.Spec.AllocationSets, allocationSet)
			}

			Expect(k8sClient.Update(context.TODO(), servers)).To(Succeed())

			err := os.Unsetenv("RABBIT_TEST_ENV_BYPASS_SERVER_STORAGE_CHECK")
			Expect(err).NotTo(HaveOccurred())
		})

		BeforeEach(func() {
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: nodeName,
				},
			}

			k8sClient.Create(context.TODO(), ns) // Ignore errors as namespace may be created from other tests
		})

		BeforeEach(func() {
			storage = &dwsv1alpha1.Storage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      nodeName,
					Namespace: corev1.NamespaceDefault,
				},
			}

			Expect(k8sClient.Create(context.TODO(), storage)).To(Succeed())

			Eventually(func(g Gomega) error {
				g.Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(storage), storage)).To(Succeed())

				storage.Status = dwsv1alpha1.StorageStatus{
					Capacity: 100000000000,
					Access: dwsv1alpha1.StorageAccess{
						Protocol: dwsv1alpha1.PCIe,
						Servers: []dwsv1alpha1.Node{
							{
								Name:   nodeName,
								Status: dwsv1alpha1.ReadyStatus,
							},
						},
					},
				}

				return k8sClient.Status().Update(context.TODO(), storage)
			}).Should(Succeed())
		})

		AfterEach(func() {
			Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(storage), storage)).To(Succeed())
			Expect(k8sClient.Delete(context.TODO(), storage)).To(Succeed())
		})

		When("Using a Lustre file system", func() {
			BeforeEach(func() {
				workflow.Spec.DWDirectives = []string{
					"#DW jobdw name=test type=lustre capacity=1GiB",
				}
			})

			It("Succeeds with one allocation per allocation set", func() {
				Eventually(func(g Gomega) error {
					g.Expect(k8sClient.Get(context.TODO(), key, workflow)).To(Succeed())
					workflow.Spec.DesiredState = dwsv1alpha1.StateSetup
					return k8sClient.Update(context.TODO(), workflow)
				}).Should(Succeed(), "update to Setup")

				Eventually(func(g Gomega) bool {
					g.Expect(k8sClient.Get(context.TODO(), key, workflow)).To(Succeed())
					return workflow.Status.Ready && workflow.Status.State == dwsv1alpha1.StateSetup
				}).Should(BeTrue(), "waiting for ready after setup")
			})

			It("Succeeds with multiple allocations in an allocation set", func() {
				Eventually(func(g Gomega) error {
					g.Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(servers), servers)).To(Succeed())
					if len(servers.Spec.AllocationSets) == 0 {
						return fmt.Errorf("Waiting for cache to update")
					}
					for i := range servers.Spec.AllocationSets {
						if servers.Spec.AllocationSets[i].Label != "ost" {
							continue
						}

						// Make four allocations, each with one quarter the size
						servers.Spec.AllocationSets[i].AllocationSize /= 4
						servers.Spec.AllocationSets[i].Storage = append(servers.Spec.AllocationSets[i].Storage, servers.Spec.AllocationSets[i].Storage[0])
						servers.Spec.AllocationSets[i].Storage[1].AllocationCount = 3
					}
					return k8sClient.Update(context.TODO(), servers)
				}).Should(Succeed(), "Set multiple allocations")

				Eventually(func(g Gomega) error {
					g.Expect(k8sClient.Get(context.TODO(), key, workflow)).To(Succeed())
					workflow.Spec.DesiredState = dwsv1alpha1.StateSetup
					return k8sClient.Update(context.TODO(), workflow)
				}).Should(Succeed(), "update to Setup")

				Eventually(func(g Gomega) bool {
					g.Expect(k8sClient.Get(context.TODO(), key, workflow)).To(Succeed())
					return workflow.Status.Ready && workflow.Status.State == dwsv1alpha1.StateSetup
				}).Should(BeTrue(), "waiting for ready after setup")
			})

			It("Fails when allocation is too small", func() {
				Eventually(func(g Gomega) error {
					g.Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(servers), servers)).To(Succeed())
					if len(servers.Spec.AllocationSets) == 0 {
						return fmt.Errorf("Waiting for cache to update")
					}
					servers.Spec.AllocationSets[0].AllocationSize -= 1
					return k8sClient.Update(context.TODO(), servers)
				}).Should(Succeed(), "Set allocation size too small")

				Eventually(func(g Gomega) error {
					g.Expect(k8sClient.Get(context.TODO(), key, workflow)).To(Succeed())
					workflow.Spec.DesiredState = dwsv1alpha1.StateSetup
					return k8sClient.Update(context.TODO(), workflow)
				}).Should(Succeed(), "update to Setup")

				Eventually(func(g Gomega) bool {
					g.Expect(k8sClient.Get(context.TODO(), key, workflow)).To(Succeed())
					return workflow.Status.State == dwsv1alpha1.StateSetup && workflow.Status.Status == dwsv1alpha1.StatusError
				}).Should(BeTrue(), "waiting for setup state to fail")
			})

			It("Fails when using the incorrect labels", func() {
				Eventually(func(g Gomega) error {
					g.Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(servers), servers)).To(Succeed())
					if len(servers.Spec.AllocationSets) == 0 {
						return fmt.Errorf("Waiting for cache to update")
					}
					servers.Spec.AllocationSets[0].Label = "bad"
					return k8sClient.Update(context.TODO(), servers)
				}).Should(Succeed(), "Set incorrect allocation set label")

				Eventually(func(g Gomega) error {
					g.Expect(k8sClient.Get(context.TODO(), key, workflow)).To(Succeed())
					workflow.Spec.DesiredState = dwsv1alpha1.StateSetup
					return k8sClient.Update(context.TODO(), workflow)
				}).Should(Succeed(), "update to Setup")

				Eventually(func(g Gomega) bool {
					g.Expect(k8sClient.Get(context.TODO(), key, workflow)).To(Succeed())
					return workflow.Status.State == dwsv1alpha1.StateSetup && workflow.Status.Status == dwsv1alpha1.StatusError
				}).Should(BeTrue(), "waiting for setup state to fail")
			})

			It("Fails when an allocation set is duplicated", func() {
				Eventually(func(g Gomega) error {
					g.Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(servers), servers)).To(Succeed())
					if len(servers.Spec.AllocationSets) == 0 {
						return fmt.Errorf("Waiting for cache to update")
					}
					servers.Spec.AllocationSets = append(servers.Spec.AllocationSets, servers.Spec.AllocationSets[0])
					return k8sClient.Update(context.TODO(), servers)
				}).Should(Succeed(), "Set incorrect allocation set label")

				Eventually(func(g Gomega) error {
					g.Expect(k8sClient.Get(context.TODO(), key, workflow)).To(Succeed())
					workflow.Spec.DesiredState = dwsv1alpha1.StateSetup
					return k8sClient.Update(context.TODO(), workflow)
				}).Should(Succeed(), "update to Setup")

				Eventually(func(g Gomega) bool {
					g.Expect(k8sClient.Get(context.TODO(), key, workflow)).To(Succeed())
					return workflow.Status.State == dwsv1alpha1.StateSetup && workflow.Status.Status == dwsv1alpha1.StatusError
				}).Should(BeTrue(), "waiting for setup state to fail")
			})

			It("Fails when using a non-existent storage", func() {
				Eventually(func(g Gomega) error {
					g.Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(servers), servers)).To(Succeed())
					if len(servers.Spec.AllocationSets) == 0 {
						return fmt.Errorf("Waiting for cache to update")
					}
					servers.Spec.AllocationSets[0].Storage[0].Name = "no-such-rabbit"
					return k8sClient.Update(context.TODO(), servers)
				}).Should(Succeed(), "Set incorrect allocation set label")

				Eventually(func(g Gomega) error {
					g.Expect(k8sClient.Get(context.TODO(), key, workflow)).To(Succeed())
					workflow.Spec.DesiredState = dwsv1alpha1.StateSetup
					return k8sClient.Update(context.TODO(), workflow)
				}).Should(Succeed(), "update to Setup")

				Eventually(func(g Gomega) bool {
					g.Expect(k8sClient.Get(context.TODO(), key, workflow)).To(Succeed())
					return workflow.Status.State == dwsv1alpha1.StateSetup && workflow.Status.Status == dwsv1alpha1.StatusError
				}).Should(BeTrue(), "waiting for setup state to fail")
			})

			It("Fails with multiple allocations totaling not enough capacity", func() {
				Eventually(func(g Gomega) error {
					g.Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(servers), servers)).To(Succeed())
					if len(servers.Spec.AllocationSets) == 0 {
						return fmt.Errorf("Waiting for cache to update")
					}
					for i := range servers.Spec.AllocationSets {
						if servers.Spec.AllocationSets[i].Label != "ost" {
							continue
						}

						// Make four allocations, each with one eigth the size
						servers.Spec.AllocationSets[i].AllocationSize /= 8
						servers.Spec.AllocationSets[i].Storage = append(servers.Spec.AllocationSets[i].Storage, servers.Spec.AllocationSets[i].Storage[0])
						servers.Spec.AllocationSets[i].Storage[1].AllocationCount = 3
					}
					return k8sClient.Update(context.TODO(), servers)
				}).Should(Succeed(), "Set multiple allocations sized too small")

				Eventually(func(g Gomega) error {
					g.Expect(k8sClient.Get(context.TODO(), key, workflow)).To(Succeed())
					workflow.Spec.DesiredState = dwsv1alpha1.StateSetup
					return k8sClient.Update(context.TODO(), workflow)
				}).Should(Succeed(), "update to Setup")

				Eventually(func(g Gomega) bool {
					g.Expect(k8sClient.Get(context.TODO(), key, workflow)).To(Succeed())
					return workflow.Status.State == dwsv1alpha1.StateSetup && workflow.Status.Status == dwsv1alpha1.StatusError
				}).Should(BeTrue(), "waiting for setup state to fail")
			})

			It("Fails with multiple allocations specified for AllocationSingleServer", func() {
				Eventually(func(g Gomega) error {
					g.Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(servers), servers)).To(Succeed())
					if len(servers.Spec.AllocationSets) == 0 {
						return fmt.Errorf("Waiting for cache to update")
					}
					for i := range servers.Spec.AllocationSets {
						if servers.Spec.AllocationSets[i].Label != "mgt" {
							continue
						}

						servers.Spec.AllocationSets[i].Storage[0].AllocationCount = 2
					}
					return k8sClient.Update(context.TODO(), servers)
				}).Should(Succeed(), "Set multiple allocations on the same server")

				Eventually(func(g Gomega) error {
					g.Expect(k8sClient.Get(context.TODO(), key, workflow)).To(Succeed())
					workflow.Spec.DesiredState = dwsv1alpha1.StateSetup
					return k8sClient.Update(context.TODO(), workflow)
				}).Should(Succeed(), "update to Setup")

				Eventually(func(g Gomega) bool {
					g.Expect(k8sClient.Get(context.TODO(), key, workflow)).To(Succeed())
					return workflow.Status.State == dwsv1alpha1.StateSetup && workflow.Status.Status == dwsv1alpha1.StatusError
				}).Should(BeTrue(), "waiting for setup state to fail")
			})

			It("Fails with multiple allocations specified for AllocationSingleServer", func() {
				Eventually(func(g Gomega) error {
					g.Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(servers), servers)).To(Succeed())
					for i := range servers.Spec.AllocationSets {
						if servers.Spec.AllocationSets[i].Label != "mgt" {
							continue
						}

						servers.Spec.AllocationSets[i].Storage = append(servers.Spec.AllocationSets[i].Storage, servers.Spec.AllocationSets[i].Storage[0])
					}
					return k8sClient.Update(context.TODO(), servers)
				}).Should(Succeed(), "Set multiple allocations on two servers")

				Eventually(func(g Gomega) error {
					g.Expect(k8sClient.Get(context.TODO(), key, workflow)).To(Succeed())
					workflow.Spec.DesiredState = dwsv1alpha1.StateSetup
					return k8sClient.Update(context.TODO(), workflow)
				}).Should(Succeed(), "update to Setup")

				Eventually(func(g Gomega) bool {
					g.Expect(k8sClient.Get(context.TODO(), key, workflow)).To(Succeed())
					return workflow.Status.State == dwsv1alpha1.StateSetup && workflow.Status.Status == dwsv1alpha1.StatusError
				}).Should(BeTrue(), "waiting for setup state to fail")
			})

		})

		When("Using an XFS file system", func() {
			BeforeEach(func() {
				workflow.Spec.DWDirectives = []string{
					"#DW jobdw name=test type=xfs capacity=1GiB",
				}
			})

			It("Succeeds with one allocation per allocation set", func() {
				Eventually(func(g Gomega) error {
					g.Expect(k8sClient.Get(context.TODO(), key, workflow)).To(Succeed())
					workflow.Spec.DesiredState = dwsv1alpha1.StateSetup
					return k8sClient.Update(context.TODO(), workflow)
				}).Should(Succeed(), "update to Setup")

				Eventually(func(g Gomega) bool {
					g.Expect(k8sClient.Get(context.TODO(), key, workflow)).To(Succeed())
					return workflow.Status.Ready && workflow.Status.State == dwsv1alpha1.StateSetup
				}).Should(BeTrue(), "waiting for ready after setup")
			})

			It("Fails with multiple allocations with total capacity == minimum capacity", func() {
				Eventually(func(g Gomega) error {
					g.Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(servers), servers)).To(Succeed())
					if len(servers.Spec.AllocationSets) == 0 {
						return fmt.Errorf("Waiting for cache to update")
					}

					// Make two  allocations, each with one half the size
					servers.Spec.AllocationSets[0].AllocationSize /= 2
					servers.Spec.AllocationSets[0].Storage = append(servers.Spec.AllocationSets[0].Storage, servers.Spec.AllocationSets[0].Storage[0])
					return k8sClient.Update(context.TODO(), servers)
				}).Should(Succeed(), "Set multiple allocations sized too small")

				Eventually(func(g Gomega) error {
					g.Expect(k8sClient.Get(context.TODO(), key, workflow)).To(Succeed())
					workflow.Spec.DesiredState = dwsv1alpha1.StateSetup
					return k8sClient.Update(context.TODO(), workflow)
				}).Should(Succeed(), "update to Setup")

				Eventually(func(g Gomega) bool {
					g.Expect(k8sClient.Get(context.TODO(), key, workflow)).To(Succeed())
					return workflow.Status.State == dwsv1alpha1.StateSetup && workflow.Status.Status == dwsv1alpha1.StatusError
				}).Should(BeTrue(), "waiting for setup state to fail")
			})

		})
	})
})

var _ = Describe("NnfStorageProfile Webhook test", func() {
	// The nnfstorageprofile_webhook_test.go covers testing of the webhook.
	// This spec exists only to verify that the webhook is also running for
	// the controller tests.
	It("Fails to create an invalid profile, to verify that the webhook is installed", func() {
		profileInvalid := basicNnfStorageProfile("an-invalid-profile")
		profileInvalid.Data.LustreStorage.ExternalMGS = "10.0.0.1@tcp"
		profileInvalid.Data.LustreStorage.CombinedMGTMDT = true
		Expect(createNnfStorageProfile(profileInvalid, false)).To(BeNil())
	})
})
