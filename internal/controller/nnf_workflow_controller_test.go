/*
 * Copyright 2023-2025 Hewlett Packard Enterprise Development LP
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
	"fmt"
	"os"
	"reflect"
	"strings"
	"sync"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"

	"sigs.k8s.io/controller-runtime/pkg/client"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"

	dwsv1alpha7 "github.com/DataWorkflowServices/dws/api/v1alpha7"
	lusv1beta1 "github.com/NearNodeFlash/lustre-fs-operator/api/v1beta1"
	nnfv1alpha10 "github.com/NearNodeFlash/nnf-sos/api/v1alpha10"
)

var (
	baseWorkflowUserID  uint32 = 1042
	baseWorkflowGroupID uint32 = 1043

	altWorkflowUserID  uint32 = 1044
	altWorkflowGroupID uint32 = 1045
)

func makeUserContainerTLSSecret() *corev1.Secret {
	// Just an empty opaque secret. We're only interested in its existence,
	// not its content.
	someSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      userContainerTLSSecretName,
			Namespace: userContainerTLSSecretNamespace,
		},
		Type: corev1.SecretTypeOpaque,
		Data: make(map[string][]byte),
	}
	Expect(k8sClient.Create(context.TODO(), someSecret)).To(Succeed())
	return someSecret
}

var _ = Describe("NNF Workflow Unit Tests", func() {

	var (
		key                   types.NamespacedName
		workflow              *dwsv1alpha7.Workflow
		setup                 sync.Once
		storageProfile        *nnfv1alpha10.NnfStorageProfile
		dmProfile             *nnfv1alpha10.NnfDataMovementProfile
		nnfNode               *nnfv1alpha10.NnfNode
		namespace             *corev1.Namespace
		persistentStorageName string
	)

	BeforeEach(func() {
		setup.Do(func() {
			namespace = &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
				Name: "rabbit-node",
			}}

			Expect(k8sClient.Create(context.TODO(), namespace)).To(Succeed())

			nnfNode = &nnfv1alpha10.NnfNode{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "nnf-nlc",
					Namespace: "rabbit-node",
				},
				Spec: nnfv1alpha10.NnfNodeSpec{
					State: nnfv1alpha10.ResourceEnable,
				},
			}
			Expect(k8sClient.Create(context.TODO(), nnfNode)).To(Succeed())

			Eventually(func(g Gomega) error {
				g.Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(nnfNode), nnfNode)).To(Succeed())
				nnfNode.Status.LNetNid = "1.2.3.4@tcp0"
				return k8sClient.Update(context.TODO(), nnfNode)
			}).Should(Succeed(), "set LNet Nid in NnfNode")
		})
		wfid := uuid.NewString()[0:8]
		persistentStorageName = "persistent-" + uuid.NewString()[:8]

		key = types.NamespacedName{
			Name:      "nnf-workflow-" + wfid,
			Namespace: corev1.NamespaceDefault,
		}

		workflow = &dwsv1alpha7.Workflow{
			ObjectMeta: metav1.ObjectMeta{
				Name:      key.Name,
				Namespace: key.Namespace,
			},
			Spec: dwsv1alpha7.WorkflowSpec{
				DesiredState: dwsv1alpha7.StateProposal,
				JobID:        intstr.FromString("job 1244"),
				WLMID:        uuid.NewString(),
				UserID:       baseWorkflowUserID,
				GroupID:      baseWorkflowGroupID,
			},
		}

		expected := &dwsv1alpha7.Workflow{}
		Expect(k8sClient.Get(context.TODO(), key, expected)).ToNot(Succeed())

		// Create a default NnfStorageProfile for the unit tests.
		storageProfile = createBasicDefaultNnfStorageProfile()

		// Create a default NnfDataMovementProfile for the unit tests.
		dmProfile = createBasicDefaultNnfDataMovementProfile()

		DeferCleanup(os.Setenv, "RABBIT_TEST_ENV_BYPASS_SERVER_STORAGE_CHECK", os.Getenv("RABBIT_TEST_ENV_BYPASS_SERVER_STORAGE_CHECK"))
	})

	AfterEach(func() {
		Eventually(func(g Gomega) error {
			g.Expect(k8sClient.Get(context.TODO(), key, workflow)).To(Succeed())
			workflow.Spec.DesiredState = dwsv1alpha7.StateTeardown
			return k8sClient.Update(context.TODO(), workflow)
		}).Should(Succeed(), "teardown")

		Eventually(func(g Gomega) bool {
			g.Expect(k8sClient.Get(context.TODO(), key, workflow)).To(Succeed())
			return workflow.Status.Ready && workflow.Status.State == dwsv1alpha7.StateTeardown
		}).Should(BeTrue(), "reach desired teardown state")

		Expect(k8sClient.Delete(context.TODO(), workflow)).To(Succeed())

		Eventually(func() error { // Delete can still return the cached object. Wait until the object is no longer present
			expected := &dwsv1alpha7.Workflow{}
			return k8sClient.Get(context.TODO(), key, expected)
		}).ShouldNot(Succeed())

		Expect(k8sClient.Delete(context.TODO(), storageProfile)).To(Succeed())
		profExpected := &nnfv1alpha10.NnfStorageProfile{}
		Eventually(func() error { // Delete can still return the cached object. Wait until the object is no longer present
			return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(storageProfile), profExpected)
		}).ShouldNot(Succeed())

		Expect(k8sClient.Delete(context.TODO(), dmProfile)).To(Succeed())
		dmProfExpected := &nnfv1alpha10.NnfDataMovementProfile{}
		Eventually(func() error { // Delete can still return the cached object. Wait until the object is no longer present
			return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(dmProfile), dmProfExpected)
		}).ShouldNot(Succeed())
	})

	getErroredDriverStatus := func(workflow *dwsv1alpha7.Workflow) *dwsv1alpha7.WorkflowDriverStatus {
		driverID := os.Getenv("DWS_DRIVER_ID")
		for _, driver := range workflow.Status.Drivers {
			if driver.DriverID == driverID {
				if driver.Status == dwsv1alpha7.StatusError {
					return &driver
				}
			}
		}
		return nil
	}

	createPersistentStorageInstance := func(name, fsType string) {
		By("Fabricate the persistent storage instance")

		// Create a persistent storage instance to be found
		psi := &dwsv1alpha7.PersistentStorageInstance{
			TypeMeta:   metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: workflow.Namespace},
			Spec: dwsv1alpha7.PersistentStorageInstanceSpec{
				Name:   name,
				FsType: fsType,
				// DWDirective: workflow.Spec.DWDirectives[0],
				DWDirective: "#DW create_persistent capacity=1GB name=" + name,
				State:       dwsv1alpha7.PSIStateActive,
			},
		}
		Expect(k8sClient.Create(context.TODO(), psi)).To(Succeed())

		// persistentdw directive checks that the nnfStorage associated with the
		// PersistentStorageInstance (by name) is present and its Status is 'Ready'
		// For this test, create such an nnfStorage so the nnfdatamovement pieces can
		// operate.
		// An alternative is to create a workflow with 'create_persistent'
		// as its directive and actually create the full-blown persistent instance.. (painful)
		nnfStorage := &nnfv1alpha10.NnfStorage{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: workflow.Namespace,
			},
			Spec: nnfv1alpha10.NnfStorageSpec{
				FileSystemType: fsType,
				AllocationSets: []nnfv1alpha10.NnfStorageAllocationSetSpec{},
			},
		}

		nnfStorageProfile := &nnfv1alpha10.NnfStorageProfile{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: workflow.Namespace,
			},
		}

		addPinnedStorageProfileLabel(nnfStorage, nnfStorageProfile)
		Expect(k8sClient.Create(context.TODO(), nnfStorage)).To(Succeed())
	}

	deletePersistentStorageInstance := func(name string) {
		By("delete persistent storage instance")
		psi := &dwsv1alpha7.PersistentStorageInstance{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: workflow.Namespace},
		}
		Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(psi), psi)).To(Succeed())
		Expect(k8sClient.Delete(context.TODO(), psi)).Should(Succeed())

		nnfStorage := &nnfv1alpha10.NnfStorage{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: workflow.Namespace},
		}

		Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(nnfStorage), nnfStorage)).To(Succeed())
		Expect(k8sClient.Delete(context.TODO(), nnfStorage)).Should(Succeed())
	}

	When("Negative tests for storage profiles", func() {
		It("Fails to achieve proposal state when the named profile cannot be found", func() {
			workflow.Spec.DWDirectives = []string{
				"#DW jobdw name=test profile=none-such type=lustre capacity=1GiB",
			}

			Expect(k8sClient.Create(context.TODO(), workflow)).To(Succeed(), "create workflow")
			// We need to be able to pass errors through the PersistentStorageInstance and DirectiveBreakdown
			// before we can test this
			/*
				Eventually(func() *dwsv1alpha7.WorkflowDriverStatus {
					expected := &dwsv1alpha7.Workflow{}
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
				Eventually(func() *dwsv1alpha7.WorkflowDriverStatus {
					expected := &dwsv1alpha7.Workflow{}
					k8sClient.Get(context.TODO(), key, expected)
					return getErroredDriverStatus(expected)
				}).ShouldNot(BeNil(), "have an error present")
			*/
		})

		When("More than one default profile", func() {

			var storageProfile2 *nnfv1alpha10.NnfStorageProfile

			BeforeEach(func() {
				// The second profile will get a different name via the call to uuid.
				// Then we'll have two that are default.
				storageProfile2 = createBasicDefaultNnfStorageProfile()
			})

			AfterEach(func() {
				Expect(k8sClient.Delete(context.TODO(), storageProfile2)).To(Succeed())
				profExpected := &nnfv1alpha10.NnfStorageProfile{}
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
					Eventually(func() *dwsv1alpha7.WorkflowDriverStatus {
						expected := &dwsv1alpha7.Workflow{}
						k8sClient.Get(context.TODO(), key, expected)
						return getErroredDriverStatus(expected)
					}).ShouldNot(BeNil(), "have an error present")
				*/
			})
		})
	})

	When("Positive tests for storage profiles", func() {

		profiles := []*nnfv1alpha10.NnfStorageProfile{}
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

			workflowAfter := &dwsv1alpha7.Workflow{}
			Eventually(func(g Gomega) error {
				g.Expect(k8sClient.Get(context.TODO(), key, workflowAfter)).To(Succeed())
				if (workflowAfter.Status.Ready == true) && (workflowAfter.Status.State == dwsv1alpha7.StateProposal) && (getErroredDriverStatus(workflowAfter) == nil) {
					return nil
				}
				return fmt.Errorf("ready state not achieved")
			}).Should(Succeed(), "achieve ready state")

			By("Verify that one DirectiveBreakdown was created")
			Expect(workflowAfter.Status.DirectiveBreakdowns).To(HaveLen(1))
			By("Verify empty requires in Workflow")
			Expect(workflowAfter.Status.Requires).To(HaveLen(0))
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

	When("Negative tests for data movement profiles", func() {
		var lustre *lusv1beta1.LustreFileSystem

		BeforeEach(func() {
			lustre = &lusv1beta1.LustreFileSystem{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "maui",
					Namespace: corev1.NamespaceDefault,
				},
				Spec: lusv1beta1.LustreFileSystemSpec{
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

		It("Fails to achieve proposal state when the named dm profile cannot be found", func() {
			workflow.Spec.DWDirectives = []string{
				"#DW jobdw name=test type=lustre capacity=1GiB",
				"#DW copy_in profile=none-such source=/lus/maui/my-file.in destination=$DW_JOB_test",
			}

			Expect(k8sClient.Create(context.TODO(), workflow)).To(Succeed(), "create workflow")

			workflowAfter := &dwsv1alpha7.Workflow{}
			Eventually(func(g Gomega) error {
				g.Expect(k8sClient.Get(context.TODO(), key, workflowAfter)).To(Succeed())
				if (workflowAfter.Status.Ready == false) && (workflowAfter.Status.State == dwsv1alpha7.StateProposal) && (getErroredDriverStatus(workflowAfter) != nil) {
					return nil
				}
				return fmt.Errorf("error state not achieved")
			}).Should(Succeed(), "achieve error state")
		})

		It("Fails to achieve proposal state when a default profile cannot be found", func() {
			dmProfile.Data.Default = false
			Expect(k8sClient.Update(context.TODO(), dmProfile)).To(Succeed(), "remove default flag")

			workflow.Spec.DWDirectives = []string{
				"#DW jobdw name=test type=lustre capacity=1GiB",
				"#DW copy_in source=/lus/maui/my-file.in destination=$DW_JOB_test",
			}

			Expect(k8sClient.Create(context.TODO(), workflow)).To(Succeed(), "create workflow")

			workflowAfter := &dwsv1alpha7.Workflow{}
			Eventually(func(g Gomega) error {
				g.Expect(k8sClient.Get(context.TODO(), key, workflowAfter)).To(Succeed())
				if (workflowAfter.Status.Ready == false) && (workflowAfter.Status.State == dwsv1alpha7.StateProposal) && (getErroredDriverStatus(workflowAfter) != nil) {
					return nil
				}
				return fmt.Errorf("error state not achieved")
			}).Should(Succeed(), "achieve error state")
		})

		When("More than one default profile", func() {

			var dmProfile2 *nnfv1alpha10.NnfDataMovementProfile

			BeforeEach(func() {
				// The second profile will get a different name via the call to uuid.
				// Then we'll have two that are default.
				dmProfile2 = createBasicDefaultNnfDataMovementProfile()
			})

			AfterEach(func() {
				Expect(k8sClient.Delete(context.TODO(), dmProfile2)).To(Succeed())
				profExpected := &nnfv1alpha10.NnfDataMovementProfile{}
				Eventually(func() error { // Delete can still return the cached object. Wait until the object is no longer present
					return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(dmProfile2), profExpected)
				}).ShouldNot(Succeed())
			})

			It("Fails to achieve proposal state when more than one default profile exists", func() {
				workflow.Spec.DWDirectives = []string{
					"#DW jobdw name=test type=lustre capacity=1GiB",
					"#DW copy_in source=/lus/maui/my-file.in destination=$DW_JOB_test",
				}

				Expect(k8sClient.Create(context.TODO(), workflow)).To(Succeed(), "create workflow")

				workflowAfter := &dwsv1alpha7.Workflow{}
				Eventually(func(g Gomega) error {
					g.Expect(k8sClient.Get(context.TODO(), key, workflowAfter)).To(Succeed())
					if (workflowAfter.Status.Ready == false) && (workflowAfter.Status.State == dwsv1alpha7.StateProposal) && (getErroredDriverStatus(workflowAfter) != nil) {
						return nil
					}
					return fmt.Errorf("error state not achieved")
				}).Should(Succeed(), "achieve error state")
			})
		})
	})

	When("Positive tests for data movement profiles", func() {

		profiles := []*nnfv1alpha10.NnfDataMovementProfile{}
		profNames := []string{}
		var lustre *lusv1beta1.LustreFileSystem

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
				prof := basicNnfDataMovementProfile(pn)
				prof = createNnfDataMovementProfile(prof, true)
				profiles = append(profiles, prof)
			}

			lustre = &lusv1beta1.LustreFileSystem{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "maui",
					Namespace: corev1.NamespaceDefault,
				},
				Spec: lusv1beta1.LustreFileSystemSpec{
					Name:      "maui",
					MountRoot: "/lus/maui",
					MgsNids:   "10.0.0.1@tcp",
				},
			}
			Expect(k8sClient.Create(context.TODO(), lustre)).To(Succeed())

		})

		AfterEach(func() {
			for _, prof := range profiles {
				Expect(k8sClient.Delete(context.TODO(), prof)).To(Succeed())
			}

			Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(lustre), lustre)).To(Succeed())
			Expect(k8sClient.Delete(context.TODO(), lustre)).To(Succeed())
		})

		JustAfterEach(func() {
			By("Verify workflow achieves proposal state with pinned profile")

			Expect(k8sClient.Create(context.TODO(), workflow)).To(Succeed(), "create workflow")

			workflowAfter := &dwsv1alpha7.Workflow{}
			Eventually(func(g Gomega) error {
				g.Expect(k8sClient.Get(context.TODO(), key, workflowAfter)).To(Succeed())
				if (workflowAfter.Status.Ready == true) && (workflowAfter.Status.State == dwsv1alpha7.StateProposal) && (getErroredDriverStatus(workflowAfter) == nil) {
					return nil
				}
				return fmt.Errorf("ready state not achieved")
			}).Should(Succeed(), "achieve ready state")

			By("Verify that one DirectiveBreakdowns was created")
			Expect(workflowAfter.Status.DirectiveBreakdowns).To(HaveLen(1))
			By("Verify empty requires in Workflow")
			Expect(workflowAfter.Status.Requires).To(HaveLen(0))
			By("Verify its pinned dm profile")
			pinnedName, pinnedNamespace := getStorageReferenceNameFromWorkflowActual(workflowAfter, 1)
			// The profile begins life with the workflow as the owner.
			Expect(verifyPinnedDMProfile(context.TODO(), k8sClient, pinnedNamespace, pinnedName)).To(Succeed())
		})

		It("Implicit use of default profile", func() {
			workflow.Spec.DWDirectives = []string{
				"#DW jobdw name=test type=lustre capacity=1GiB",
				"#DW copy_in source=/lus/maui/my-file.in destination=$DW_JOB_test",
			}
		})

		It("Named profile, which happens to also be the default", func() {
			workflow.Spec.DWDirectives = []string{
				"#DW jobdw name=test type=lustre capacity=1GiB",
				fmt.Sprintf("#DW copy_in source=/lus/maui/my-file.in destination=$DW_JOB_test profile=%s", dmProfile.GetName()),
			}
		})

		It("Named profile", func() {
			workflow.Spec.DWDirectives = []string{
				"#DW jobdw name=test type=lustre capacity=1GiB",
				fmt.Sprintf("#DW copy_in source=/lus/maui/my-file.in destination=$DW_JOB_test profile=%s", profNames[0]),
			}
		})
	})

	When("Using bad copy_in directives", func() {

		It("Fails missing or malformed jobdw reference", func() {
			workflow.Spec.DWDirectives = []string{
				"#DW jobdw name=test type=lustre capacity=1GiB",
				"#DW copy_in source=/lus/maui/my-file.in destination=$DW_JOB_INCORRECT/my-file.out",
			}

			Expect(k8sClient.Create(context.TODO(), workflow)).To(Succeed(), "create workflow")

			Eventually(func() *dwsv1alpha7.WorkflowDriverStatus {
				expected := &dwsv1alpha7.Workflow{}
				k8sClient.Get(context.TODO(), key, expected)
				return getErroredDriverStatus(expected)
			}).ShouldNot(BeNil(), "have an error present")

		})

		It("Fails missing or malformed persistentdw reference", func() {
			workflow.Spec.DWDirectives = []string{
				"#DW create_persistent name=test type=lustre capacity=1GiB",
				"#DW copy_in source=/lus/maui/my-file.in destination=$DW_PERSISTENT_INCORRECT/my-file.out",
			}

			Expect(k8sClient.Create(context.TODO(), workflow)).To(Succeed(), "create workflow")

			Eventually(func() *dwsv1alpha7.WorkflowDriverStatus {
				expected := &dwsv1alpha7.Workflow{}
				k8sClient.Get(context.TODO(), key, expected)
				return getErroredDriverStatus(expected)
			}).ShouldNot(BeNil(), "have an error present")
		})

		It("Fails missing or malformed global lustre reference", func() {
			workflow.Spec.DWDirectives = []string{
				"#DW jobdw name=test type=lustre capacity=1GiB",
				"#DW copy_in source=/lus/INCORRECT/my-file.in destination=$DW_JOB_test/my-file.out",
			}

			Expect(k8sClient.Create(context.TODO(), workflow)).To(Succeed(), "create workflow")

			Eventually(func() *dwsv1alpha7.WorkflowDriverStatus {
				expected := &dwsv1alpha7.Workflow{}
				k8sClient.Get(context.TODO(), key, expected)
				return getErroredDriverStatus(expected)
			}).ShouldNot(BeNil(), "have an error present")
		})
	})

	When("Using bad copy_out directives", func() {

		It("Fails missing or malformed jobdw reference", func() {
			workflow.Spec.DWDirectives = []string{
				"#DW jobdw name=test type=lustre capacity=1GiB",
				"#DW copy_out source=$DW_JOB_INCORRECT/my-file.in destination=/lus/maui/my-file.out",
			}

			Expect(k8sClient.Create(context.TODO(), workflow)).To(Succeed(), "create workflow")

			Eventually(func() *dwsv1alpha7.WorkflowDriverStatus {
				expected := &dwsv1alpha7.Workflow{}
				k8sClient.Get(context.TODO(), key, expected)
				return getErroredDriverStatus(expected)
			}).ShouldNot(BeNil(), "have an error present")

		})

		It("Fails missing or malformed persistentdw reference", func() {
			workflow.Spec.DWDirectives = []string{
				"#DW create_persistent name=test type=lustre capacity=1GiB",
				"#DW copy_out source=$DW_PERSISTENT_INCORRECT/my-file.int destination=/lus/maui/my-file.out",
			}

			Expect(k8sClient.Create(context.TODO(), workflow)).To(Succeed(), "create workflow")

			Eventually(func() *dwsv1alpha7.WorkflowDriverStatus {
				expected := &dwsv1alpha7.Workflow{}
				k8sClient.Get(context.TODO(), key, expected)
				return getErroredDriverStatus(expected)
			}).ShouldNot(BeNil(), "have an error present")
		})

		It("Fails missing or malformed global lustre reference", func() {
			workflow.Spec.DWDirectives = []string{
				"#DW jobdw name=test type=lustre capacity=1GiB",
				"#DW copy_out source=$DW_JOB_test/my-file.int destination=/lus/INCORRECT/my-file.out",
			}

			Expect(k8sClient.Create(context.TODO(), workflow)).To(Succeed(), "create workflow")

			Eventually(func() *dwsv1alpha7.WorkflowDriverStatus {
				expected := &dwsv1alpha7.Workflow{}
				k8sClient.Get(context.TODO(), key, expected)
				return getErroredDriverStatus(expected)
			}).ShouldNot(BeNil(), "have an error present")
		})
	})

	When("Using copy_in directives", func() {
		var (
			dmm *nnfv1alpha10.NnfDataMovementManager
		)

		JustBeforeEach(func() {
			Expect(k8sClient.Create(context.TODO(), workflow)).To(Succeed(), "create workflow")

			Eventually(func(g Gomega) bool {
				g.Expect(k8sClient.Get(context.TODO(), key, workflow)).To(Succeed())
				return workflow.Status.Ready
			}).Should(BeTrue(), "waiting for ready after create")

			workflow.Spec.DesiredState = dwsv1alpha7.StateSetup
			Expect(k8sClient.Update(context.TODO(), workflow)).To(Succeed())

			Eventually(func(g Gomega) bool {
				g.Expect(k8sClient.Get(context.TODO(), key, workflow)).To(Succeed())
				return workflow.Status.Ready && workflow.Status.State == dwsv1alpha7.StateSetup
			}).Should(BeTrue(), "transition through setup")
		})

		// Create a fake global lustre file system.
		var (
			lustre *lusv1beta1.LustreFileSystem
		)

		BeforeEach(func() {
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: nnfv1alpha10.DataMovementNamespace,
				},
			}

			k8sClient.Create(context.TODO(), ns) // Ignore errors as namespace may be created from other tests

			dmm = &nnfv1alpha10.NnfDataMovementManager{
				ObjectMeta: metav1.ObjectMeta{
					Name:      nnfv1alpha10.DataMovementManagerName,
					Namespace: nnfv1alpha10.DataMovementNamespace,
				},
				Spec: nnfv1alpha10.NnfDataMovementManagerSpec{
					PodSpec: nnfv1alpha10.NnfPodSpec{
						Containers: []nnfv1alpha10.NnfContainer{{
							Name:  "name-manager-dummy",
							Image: "nginx",
						}},
					},
				},
				Status: nnfv1alpha10.NnfDataMovementManagerStatus{
					Ready: true,
				},
			}
			Expect(k8sClient.Create(ctx, dmm)).To(Succeed())
			WaitForDMMReady(dmm)
		})

		BeforeEach(func() {
			lustre = &lusv1beta1.LustreFileSystem{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "maui",
					Namespace: corev1.NamespaceDefault,
				},
				Spec: lusv1beta1.LustreFileSystemSpec{
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

			Expect(k8sClient.Delete(context.TODO(), dmm)).To(Succeed())
			Eventually(func() error { // Delete can still return the cached object. Wait until the object is no longer present
				return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(dmm), dmm)
			}).ShouldNot(Succeed())
		})

		When("using $DW_JOB_ references", func() {

			BeforeEach(func() {
				workflow.Spec.DWDirectives = []string{
					"#DW jobdw name=test type=lustre capacity=1GiB",
					"#DW copy_in source=/lus/maui/my-file.in destination=$DW_JOB_test/my-file.out",
				}
			})

			It("transition to data movement", func() {

				By("transition to data in state")
				Eventually(func(g Gomega) error {
					g.Expect(k8sClient.Get(context.TODO(), key, workflow)).To(Succeed())
					workflow.Spec.DesiredState = dwsv1alpha7.StateDataIn
					return k8sClient.Update(context.TODO(), workflow)
				}).Should(Succeed(), "update to DataIn")

				By("creates the data movement resource")
				dm := &nnfv1alpha10.NnfDataMovement{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("%s-%d", workflow.Name, 1),
						Namespace: nnfv1alpha10.DataMovementNamespace,
					},
				}

				Eventually(func() error {
					return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(dm), dm)
				}).Should(Succeed(), "expect data movement resource")

				Expect(dm.Spec.Source.Path).To(Equal(lustre.Spec.MountRoot + "/my-file.in"))
				Expect(dm.Spec.Source.StorageReference).ToNot(BeNil())
				Expect(dm.Spec.Source.StorageReference).To(MatchFields(IgnoreExtras,
					Fields{
						"Kind":      Equal(reflect.TypeOf(lusv1beta1.LustreFileSystem{}).Name()),
						"Name":      Equal(lustre.ObjectMeta.Name),
						"Namespace": Equal(lustre.Namespace),
					}))

				Expect(dm.Spec.Destination.Path).To(Equal(buildComputeMountPath(workflow, 0) + "/my-file.out"))
				Expect(dm.Spec.Destination.StorageReference).ToNot(BeNil())
				Expect(dm.Spec.Destination.StorageReference).To(MatchFields(IgnoreExtras,
					Fields{
						"Kind":      Equal(reflect.TypeOf(nnfv1alpha10.NnfStorage{}).Name()),
						"Name":      Equal(fmt.Sprintf("%s-%d", workflow.Name, 0)),
						"Namespace": Equal(workflow.Namespace),
					}))

				Expect(dm.Spec.ProfileReference).To(MatchFields(IgnoreExtras,
					Fields{
						"Kind":      Equal(reflect.TypeOf(nnfv1alpha10.NnfDataMovementProfile{}).Name()),
						"Name":      Equal(indexedResourceName(workflow, 1)),
						"Namespace": Equal(corev1.NamespaceDefault),
					},
				))
				Expect(dm.GetLabels()[nnfv1alpha10.DataMovementInitiatorLabel]).To(Equal("copy_in"))
			})
		})

		When("using $DW_PERSISTENT_ references", func() {
			dmProfile := basicNnfDataMovementProfile("test")

			BeforeEach(func() {
				workflow.Spec.DWDirectives = []string{
					fmt.Sprintf("#DW persistentdw name=%s", persistentStorageName),
					fmt.Sprintf("#DW copy_in source=/lus/maui/my-file.in profile=%s destination=$DW_PERSISTENT_%s/my-persistent-file.out", dmProfile.Name, strings.ReplaceAll(persistentStorageName, "-", "_")),
				}

				createPersistentStorageInstance(persistentStorageName, "lustre")
				createNnfDataMovementProfile(dmProfile, true)
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
				deletePersistentStorageInstance(persistentStorageName)

				Expect(k8sClient.Delete(context.TODO(), dmProfile)).Should(Succeed())
				Expect(k8sClient.Delete(context.TODO(), ns)).Should(Succeed())
			})

			It("transitions to data movement", func() {

				Eventually(func(g Gomega) error {
					g.Expect(k8sClient.Get(context.TODO(), key, workflow)).To(Succeed())
					workflow.Spec.DesiredState = dwsv1alpha7.StateDataIn
					return k8sClient.Update(context.TODO(), workflow)
				}).Should(Succeed(), "transition desired state to DataIn")

				dm := &nnfv1alpha10.NnfDataMovement{
					ObjectMeta: metav1.ObjectMeta{
						Name:      indexedResourceName(workflow, 1),
						Namespace: nnfv1alpha10.DataMovementNamespace,
					},
				}

				Eventually(func() error {
					return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(dm), dm)
				}).Should(Succeed(), "data movement resource created")

				Expect(dm.Spec.Source.Path).To(Equal(lustre.Spec.MountRoot + "/my-file.in"))
				Expect(dm.Spec.Source.StorageReference).ToNot(BeNil())
				Expect(dm.Spec.Source.StorageReference).To(MatchFields(IgnoreExtras,
					Fields{
						"Kind":      Equal(reflect.TypeOf(lusv1beta1.LustreFileSystem{}).Name()),
						"Name":      Equal(lustre.ObjectMeta.Name),
						"Namespace": Equal(lustre.Namespace),
					}))

				Expect(dm.Spec.Destination.Path).To(Equal(buildComputeMountPath(workflow, 0) + "/my-persistent-file.out"))
				Expect(dm.Spec.Destination.StorageReference).ToNot(BeNil())
				Expect(dm.Spec.Destination.StorageReference).To(MatchFields(IgnoreExtras,
					Fields{
						"Kind":      Equal(reflect.TypeOf(nnfv1alpha10.NnfStorage{}).Name()),
						"Name":      Equal(persistentStorageName),
						"Namespace": Equal(workflow.Namespace),
					}))
				Expect(dm.Spec.ProfileReference).To(MatchFields(IgnoreExtras,
					Fields{
						"Kind":      Equal(reflect.TypeOf(nnfv1alpha10.NnfDataMovementProfile{}).Name()),
						"Name":      Equal(indexedResourceName(workflow, 1)),
						"Namespace": Equal(corev1.NamespaceDefault),
					},
				))
				Expect(dm.GetLabels()[nnfv1alpha10.DataMovementInitiatorLabel]).To(Equal("copy_in"))

			})
		})
	}) // When("Using copy_in directives", func()

	When("Using server allocation input", func() {
		const nodeName = "rabbit-node"

		var (
			storage            *dwsv1alpha7.Storage
			directiveBreakdown *dwsv1alpha7.DirectiveBreakdown
			servers            *dwsv1alpha7.Servers
		)

		JustBeforeEach(func() {
			Expect(k8sClient.Update(context.TODO(), storageProfile)).To(Succeed())

			Expect(k8sClient.Create(context.TODO(), workflow)).To(Succeed(), "create workflow")

			Eventually(func(g Gomega) bool {
				g.Expect(k8sClient.Get(context.TODO(), key, workflow)).To(Succeed())
				return workflow.Status.Ready
			}).Should(BeTrue(), "waiting for ready after create")

			directiveBreakdown = &dwsv1alpha7.DirectiveBreakdown{
				ObjectMeta: metav1.ObjectMeta{
					Name:      workflow.Status.DirectiveBreakdowns[0].Name,
					Namespace: workflow.Status.DirectiveBreakdowns[0].Namespace,
				},
			}
			Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(directiveBreakdown), directiveBreakdown)).To(Succeed())

			servers = &dwsv1alpha7.Servers{
				ObjectMeta: metav1.ObjectMeta{
					Name:      directiveBreakdown.Status.Storage.Reference.Name,
					Namespace: directiveBreakdown.Status.Storage.Reference.Namespace,
				},
			}
			Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(servers), servers)).To(Succeed())

			for _, directiveAllocationSet := range directiveBreakdown.Status.Storage.AllocationSets {
				allocationSet := dwsv1alpha7.ServersSpecAllocationSet{
					Label:          directiveAllocationSet.Label,
					AllocationSize: directiveAllocationSet.MinimumCapacity,
					Storage: []dwsv1alpha7.ServersSpecStorage{
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
			storage = &dwsv1alpha7.Storage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      nodeName,
					Namespace: corev1.NamespaceDefault,
				},
			}

			Expect(k8sClient.Create(context.TODO(), storage)).To(Succeed())

			Eventually(func(g Gomega) error {
				g.Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(storage), storage)).To(Succeed())

				storage.Status = dwsv1alpha7.StorageStatus{
					Capacity: 100000000000,
					Access: dwsv1alpha7.StorageAccess{
						Protocol: dwsv1alpha7.PCIe,
						Servers: []dwsv1alpha7.Node{
							{
								Name:   nodeName,
								Status: dwsv1alpha7.ReadyStatus,
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
					workflow.Spec.DesiredState = dwsv1alpha7.StateSetup
					return k8sClient.Update(context.TODO(), workflow)
				}).Should(Succeed(), "update to Setup")

				Eventually(func(g Gomega) bool {
					g.Expect(k8sClient.Get(context.TODO(), key, workflow)).To(Succeed())
					return workflow.Status.Ready && workflow.Status.State == dwsv1alpha7.StateSetup
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
					workflow.Spec.DesiredState = dwsv1alpha7.StateSetup
					return k8sClient.Update(context.TODO(), workflow)
				}).Should(Succeed(), "update to Setup")

				Eventually(func(g Gomega) bool {
					g.Expect(k8sClient.Get(context.TODO(), key, workflow)).To(Succeed())
					return workflow.Status.Ready && workflow.Status.State == dwsv1alpha7.StateSetup
				}).Should(BeTrue(), "waiting for ready after setup")
			})

			It("Succeeds with multiple mdts", func() {
				Eventually(func(g Gomega) error {
					g.Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(servers), servers)).To(Succeed())
					if len(servers.Spec.AllocationSets) == 0 {
						return fmt.Errorf("Waiting for cache to update")
					}
					for i := range servers.Spec.AllocationSets {
						if servers.Spec.AllocationSets[i].Label != "mdt" {
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
					workflow.Spec.DesiredState = dwsv1alpha7.StateSetup
					return k8sClient.Update(context.TODO(), workflow)
				}).Should(Succeed(), "update to Setup")

				Eventually(func(g Gomega) bool {
					g.Expect(k8sClient.Get(context.TODO(), key, workflow)).To(Succeed())
					return workflow.Status.Ready && workflow.Status.State == dwsv1alpha7.StateSetup
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
					workflow.Spec.DesiredState = dwsv1alpha7.StateSetup
					return k8sClient.Update(context.TODO(), workflow)
				}).Should(Succeed(), "update to Setup")

				Eventually(func(g Gomega) bool {
					g.Expect(k8sClient.Get(context.TODO(), key, workflow)).To(Succeed())
					return workflow.Status.State == dwsv1alpha7.StateSetup && workflow.Status.Status == dwsv1alpha7.StatusError
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
					workflow.Spec.DesiredState = dwsv1alpha7.StateSetup
					return k8sClient.Update(context.TODO(), workflow)
				}).Should(Succeed(), "update to Setup")

				Eventually(func(g Gomega) bool {
					g.Expect(k8sClient.Get(context.TODO(), key, workflow)).To(Succeed())
					return workflow.Status.State == dwsv1alpha7.StateSetup && workflow.Status.Status == dwsv1alpha7.StatusError
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
					workflow.Spec.DesiredState = dwsv1alpha7.StateSetup
					return k8sClient.Update(context.TODO(), workflow)
				}).Should(Succeed(), "update to Setup")

				Eventually(func(g Gomega) bool {
					g.Expect(k8sClient.Get(context.TODO(), key, workflow)).To(Succeed())
					return workflow.Status.State == dwsv1alpha7.StateSetup && workflow.Status.Status == dwsv1alpha7.StatusError
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
					workflow.Spec.DesiredState = dwsv1alpha7.StateSetup
					return k8sClient.Update(context.TODO(), workflow)
				}).Should(Succeed(), "update to Setup")

				Eventually(func(g Gomega) bool {
					g.Expect(k8sClient.Get(context.TODO(), key, workflow)).To(Succeed())
					return workflow.Status.State == dwsv1alpha7.StateSetup && workflow.Status.Status == dwsv1alpha7.StatusError
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
					workflow.Spec.DesiredState = dwsv1alpha7.StateSetup
					return k8sClient.Update(context.TODO(), workflow)
				}).Should(Succeed(), "update to Setup")

				Eventually(func(g Gomega) bool {
					g.Expect(k8sClient.Get(context.TODO(), key, workflow)).To(Succeed())
					return workflow.Status.State == dwsv1alpha7.StateSetup && workflow.Status.Status == dwsv1alpha7.StatusError
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
					workflow.Spec.DesiredState = dwsv1alpha7.StateSetup
					return k8sClient.Update(context.TODO(), workflow)
				}).Should(Succeed(), "update to Setup")

				Eventually(func(g Gomega) bool {
					g.Expect(k8sClient.Get(context.TODO(), key, workflow)).To(Succeed())
					return workflow.Status.State == dwsv1alpha7.StateSetup && workflow.Status.Status == dwsv1alpha7.StatusError
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
					workflow.Spec.DesiredState = dwsv1alpha7.StateSetup
					return k8sClient.Update(context.TODO(), workflow)
				}).Should(Succeed(), "update to Setup")

				Eventually(func(g Gomega) bool {
					g.Expect(k8sClient.Get(context.TODO(), key, workflow)).To(Succeed())
					return workflow.Status.State == dwsv1alpha7.StateSetup && workflow.Status.Status == dwsv1alpha7.StatusError
				}).Should(BeTrue(), "waiting for setup state to fail")
			})

		})

		When("Using a Lustre file system with combined mgtmdt", func() {
			BeforeEach(func() {
				workflow.Spec.DWDirectives = []string{
					"#DW jobdw name=test type=lustre capacity=1GiB",
				}

				storageProfile.Data.LustreStorage.CombinedMGTMDT = true
			})

			It("Succeeds with multiple mdts", func() {
				Eventually(func(g Gomega) error {
					g.Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(servers), servers)).To(Succeed())
					if len(servers.Spec.AllocationSets) == 0 {
						return fmt.Errorf("Waiting for cache to update")
					}
					for i := range servers.Spec.AllocationSets {
						if servers.Spec.AllocationSets[i].Label != "mgtmdt" {
							continue
						}

						// Make four allocations, each with one quarter the size. 3 allocations are on the first storage
						// and one is on a second storage
						servers.Spec.AllocationSets[i].AllocationSize /= 4
						servers.Spec.AllocationSets[i].Storage = append(servers.Spec.AllocationSets[i].Storage, servers.Spec.AllocationSets[i].Storage[0])
						servers.Spec.AllocationSets[i].Storage[0].AllocationCount = 3
					}
					return k8sClient.Update(context.TODO(), servers)
				}).Should(Succeed(), "Set multiple allocations")

				Eventually(func(g Gomega) error {
					g.Expect(k8sClient.Get(context.TODO(), key, workflow)).To(Succeed())
					workflow.Spec.DesiredState = dwsv1alpha7.StateSetup
					return k8sClient.Update(context.TODO(), workflow)
				}).Should(Succeed(), "update to Setup")

				Eventually(func(g Gomega) bool {
					g.Expect(k8sClient.Get(context.TODO(), key, workflow)).To(Succeed())
					return workflow.Status.Ready && workflow.Status.State == dwsv1alpha7.StateSetup
				}).Should(BeTrue(), "waiting for ready after setup")

				nnfStorage := &nnfv1alpha10.NnfStorage{
					ObjectMeta: metav1.ObjectMeta{
						Name:      servers.Name,
						Namespace: servers.Namespace,
					},
				}

				Eventually(func(g Gomega) bool {
					g.Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(nnfStorage), nnfStorage)).To(Succeed())
					for _, allocationSet := range nnfStorage.Spec.AllocationSets {
						if allocationSet.Name != "mgtmdt" {
							continue
						}

						// Check that there are 3 nodes listed in the mgtmdt allocation set. The first node should
						// only have a single allocation.
						return len(allocationSet.Nodes) == 3 && allocationSet.Nodes[0].Count == 1
					}

					return false
				}).Should(BeTrue(), "waiting for ready after setup")
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
					workflow.Spec.DesiredState = dwsv1alpha7.StateSetup
					return k8sClient.Update(context.TODO(), workflow)
				}).Should(Succeed(), "update to Setup")

				Eventually(func(g Gomega) bool {
					g.Expect(k8sClient.Get(context.TODO(), key, workflow)).To(Succeed())
					return workflow.Status.Ready && workflow.Status.State == dwsv1alpha7.StateSetup
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
					workflow.Spec.DesiredState = dwsv1alpha7.StateSetup
					return k8sClient.Update(context.TODO(), workflow)
				}).Should(Succeed(), "update to Setup")

				Eventually(func(g Gomega) bool {
					g.Expect(k8sClient.Get(context.TODO(), key, workflow)).To(Succeed())
					return workflow.Status.State == dwsv1alpha7.StateSetup && workflow.Status.Status == dwsv1alpha7.StatusError
				}).Should(BeTrue(), "waiting for setup state to fail")
			})

		})
	})

	When("Using container directives", func() {
		var (
			ns *corev1.Namespace

			createPersistent     bool
			createPersistentType string

			createGlobalLustre bool
			globalLustre       *lusv1beta1.LustreFileSystem

			containerProfile         *nnfv1alpha10.NnfContainerProfile
			containerProfileStorages []nnfv1alpha10.NnfContainerProfileStorage
			createContainerProfile   bool

			userContainerTLSSecret *corev1.Secret
		)

		BeforeEach(func() {
			createPersistent = true
			createPersistentType = "lustre"
			createGlobalLustre = false
			containerProfile = nil
			containerProfileStorages = nil
			createContainerProfile = true

			// Create/Delete the "nnf-system" namespace as part of the test life-cycle; the persistent storage instances are
			// placed in the "nnf-system" namespace so it must be present.
			// EnvTest does not support namespace deletion, so this could already exist. Ignore any errors.
			ns = &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "nnf-system",
				},
			}
			k8sClient.Create(context.TODO(), ns)

		})

		JustBeforeEach(func() {
			// containerProfileStorages is configurable, so this must happen in JustBeforeEach()
			if createContainerProfile {
				containerProfile = createBasicNnfContainerProfile(containerProfileStorages)
			}

			if createPersistent {
				createPersistentStorageInstance(persistentStorageName, createPersistentType)
			}

			if createGlobalLustre {
				globalLustre = &lusv1beta1.LustreFileSystem{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "sawbill",
						Namespace: corev1.NamespaceDefault,
					},
					Spec: lusv1beta1.LustreFileSystemSpec{
						Name:      "sawbill",
						MountRoot: "/lus/sawbill",
						MgsNids:   "10.0.0.2@tcp",
					},
				}
				Expect(k8sClient.Create(context.TODO(), globalLustre)).To(Succeed())
			}
		})

		AfterEach(func() {
			if containerProfile != nil {
				By("delete NnfContainerProfile")
				Expect(k8sClient.Delete(context.TODO(), containerProfile)).Should(Succeed())
				Eventually(func() error {
					return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(containerProfile), containerProfile)
				}).ShouldNot(Succeed())
				containerProfile = nil
			}

			if userContainerTLSSecret != nil {
				By("delete user container TLS secret")
				Expect(k8sClient.Delete(context.TODO(), userContainerTLSSecret)).Should(Succeed())
				Eventually(func() error {
					return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(userContainerTLSSecret), userContainerTLSSecret)
				}).ShouldNot(Succeed())
				userContainerTLSSecret = nil
			}

			if createPersistent {
				deletePersistentStorageInstance(persistentStorageName)
			}

			if createGlobalLustre {
				Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(globalLustre), globalLustre)).To(Succeed())
				Expect(k8sClient.Delete(context.TODO(), globalLustre)).To(Succeed())
			}
		})

		Context("with container restrictions", func() {
			BeforeEach(func() {
				createContainerProfile = false // We'll make a custom version.
				createGlobalLustre = true
			})

			// buildRestrictedContainerProfile will create a NnfContainerProfile that
			// is restricted to a specific user ID or group ID.
			buildRestrictedContainerProfile := func(userID *uint32, groupID *uint32) {
				By("Create a restricted NnfContainerProfile")
				tempProfile := basicNnfContainerProfile("restricted-"+uuid.NewString()[:8], containerProfileStorages)
				if userID != nil {
					tempProfile.Data.UserID = userID
				}
				if groupID != nil {
					tempProfile.Data.GroupID = groupID
				}

				containerProfile = createNnfContainerProfile(tempProfile, true)
			}

			buildWorkflowWithCorrectDirectives := func(requiresList string) {
				By("creating the workflow")
				jobdw := "#DW jobdw name=container-storage type=gfs2 capacity=1GB"
				if len(requiresList) > 0 {
					jobdw = fmt.Sprintf("%s requires=%s", jobdw, requiresList)
				}
				workflow.Spec.DWDirectives = []string{
					jobdw,
					"#DW persistentdw name=" + persistentStorageName,
					fmt.Sprintf("#DW container name=container profile=%s "+
						"DW_JOB_foo_local_storage=container-storage "+
						"DW_PERSISTENT_foo_persistent_storage=%s "+
						"DW_GLOBAL_foo_global_lustre=%s",
						containerProfile.Name, persistentStorageName, globalLustre.Spec.MountRoot),
				}
				Expect(k8sClient.Create(context.TODO(), workflow)).Should(Succeed())
			}

			DescribeTable("should go to Proposal Ready when everything is in order",
				func(containerUserID *uint32, containerGroupID *uint32) {
					buildRestrictedContainerProfile(containerUserID, containerGroupID)
					buildWorkflowWithCorrectDirectives("")
					Eventually(func(g Gomega) bool {
						g.Expect(k8sClient.Get(context.TODO(), key, workflow)).To(Succeed())
						return workflow.Status.Ready && workflow.Status.State == dwsv1alpha7.StateProposal
					}).Should(BeTrue(), "reach desired Proposal state")
					Expect(verifyPinnedContainerProfile(context.TODO(), k8sClient, workflow, 2)).To(Succeed())
					Expect(workflow.Status.Requires).To(HaveLen(0))
				},
				Entry("when not restricted to a user ID or group ID", nil, nil),
				Entry("when restricted to a matching user ID", &baseWorkflowUserID, nil),
				Entry("when restricted to a matching group ID", nil, &baseWorkflowGroupID),
				Entry("when restricted to a matching user ID and group ID", &baseWorkflowUserID, &baseWorkflowGroupID),
			)

			DescribeTable("should not go to Proposal Ready when profile restriction is not satisfied",
				func(containerUserID *uint32, containerGroupID *uint32) {
					buildRestrictedContainerProfile(containerUserID, containerGroupID)
					buildWorkflowWithCorrectDirectives("")
					Eventually(func(g Gomega) bool {
						g.Expect(k8sClient.Get(context.TODO(), key, workflow)).To(Succeed())
						return workflow.Status.Status == dwsv1alpha7.StatusError && strings.Contains(workflow.Status.Message, "container profile") && strings.Contains(workflow.Status.Message, "is restricted to")
					}).Should(BeTrue(), "does not reach desired Proposal state")
				},
				Entry("when restricted to non-matching user ID", &altWorkflowUserID, nil),
				Entry("when restricted to non-matching group ID", nil, &altWorkflowGroupID),
				Entry("when restricted to non-matching user ID and group ID", &altWorkflowUserID, &altWorkflowGroupID),
			)

			DescribeTable("should go to Proposal Ready with interpreted Requires in the workflow",
				func(requiresList string, wantList []string, createSecret bool) {
					buildRestrictedContainerProfile(nil, nil)
					if createSecret {
						userContainerTLSSecret = makeUserContainerTLSSecret()
					}
					buildWorkflowWithCorrectDirectives(requiresList)
					Eventually(func(g Gomega) bool {
						g.Expect(k8sClient.Get(context.TODO(), key, workflow)).To(Succeed())
						return workflow.Status.Ready && workflow.Status.State == dwsv1alpha7.StateProposal
					}).Should(BeTrue(), "reach desired Proposal state")
					if len(wantList) == 0 {
						Expect(workflow.Status.Requires).To(HaveLen(0))
					} else {
						Expect(workflow.Status.Requires).Should(ContainElements(wantList))
						Expect(workflow.Status.Requires).To(HaveLen(len(wantList)))
					}
				},
				// The 'requiresList' content is constrained by config/dws/nnf-ruleset.yaml, while the
				// 'wantList' content is not.
				// In these first two cases, we don't care whether or not the
				// TLS secret exists because we're not using it anyway. The workflow
				// still has to get to Proposal-ready.
				Entry("when requires list is empty", "", []string{}, true),
				Entry("when requires list is empty and user-container TLS secret does not exist", "", []string{}, false),
				Entry("when requires list has one", "user-container-auth", []string{requiresContainerAuth}, true),
				// copy-offload adds two words: one for itself and one for container auth
				Entry("when requires list has copy-offload", "copy-offload", []string{requiresContainerAuth, requiresCopyOffload}, true),
				Entry("when requires list has multiple matches", "copy-offload,user-container-auth", []string{requiresContainerAuth, requiresCopyOffload}, true),
			)

			DescribeTable("when missing TLS secret, should not go to Proposal Ready when asking for user container auth",
				func(requiresList string, wantList []string) {
					buildRestrictedContainerProfile(nil, nil)
					buildWorkflowWithCorrectDirectives(requiresList)
					Eventually(func(g Gomega) bool {
						g.Expect(k8sClient.Get(context.TODO(), key, workflow)).To(Succeed())
						return workflow.Status.Ready == false && workflow.Status.Status == dwsv1alpha7.StatusError && workflow.Status.State == dwsv1alpha7.StateProposal
					}).Should(BeTrue(), "did not reach desired Proposal state")
					Expect(workflow.Status.Drivers[0].Error).Should(ContainSubstring("administrator must configure the user container TLS secret"))
				},
				// The 'requiresList' content is constrained by config/dws/nnf-ruleset.yaml, while the
				// 'wantList' content is not.
				Entry("when requires list has one", "user-container-auth", []string{requiresContainerAuth}),
				Entry("when requires list has copy-offload", "copy-offload", []string{requiresContainerAuth, requiresCopyOffload}),
			)
		})

		Context("when an optional storage in the container profile is not present in the container arguments", func() {
			BeforeEach(func() {
				containerProfileStorages = []nnfv1alpha10.NnfContainerProfileStorage{
					{Name: "DW_JOB_foo_local_storage", Optional: false},
					{Name: "DW_PERSISTENT_foo_persistent_storage", Optional: true},
					{Name: "DW_GLOBAL_foo_global_lustre", Optional: true},
				}
			})

			It("should go to Proposal Ready", func() {
				workflow.Spec.DWDirectives = []string{
					"#DW jobdw name=container-storage type=gfs2 capacity=1GB",
					fmt.Sprintf("#DW container name=container profile=%s "+
						"DW_JOB_foo_local_storage=container-storage ",
						containerProfile.Name),
				}
				Expect(k8sClient.Create(context.TODO(), workflow)).Should(Succeed())

				Eventually(func(g Gomega) bool {
					g.Expect(k8sClient.Get(context.TODO(), key, workflow)).To(Succeed())
					return workflow.Status.Ready && workflow.Status.State == dwsv1alpha7.StateProposal
				}).Should(BeTrue(), "reach desired Proposal state")
				Expect(verifyPinnedContainerProfile(context.TODO(), k8sClient, workflow, 1)).To(Succeed())
			})
		})

		Context("when a required storage in the container profile is not present in the directives", func() {
			It("should go to error", func() {
				workflow.Spec.DWDirectives = []string{
					// persistent storage is missing
					"#DW jobdw name=container-storage type=gfs2 capacity=1GB",
					fmt.Sprintf("#DW container name=container profile=%s "+
						"DW_JOB_foo_local_storage=container-storage "+
						"DW_PERSISTENT_foo_persistent_storage=%s",
						containerProfile.Name, persistentStorageName),
				}
				Expect(k8sClient.Create(context.TODO(), workflow)).Should(Succeed())

				Eventually(func(g Gomega) bool {
					g.Expect(k8sClient.Get(context.TODO(), key, workflow)).To(Succeed())
					return !workflow.Status.Ready && workflow.Status.Status == dwsv1alpha7.StatusError
				}).Should(BeTrue(), "be in error state")
			})
		})

		Context("when a required storage in the container profile is not present in the arguments", func() {
			BeforeEach(func() {
				containerProfileStorages = []nnfv1alpha10.NnfContainerProfileStorage{
					{Name: "DW_JOB_foo_local_storage", Optional: false},
					{Name: "DW_PERSISTENT_foo_persistent_storage", Optional: true},
				}
			})
			It("should go to error", func() {
				workflow.Spec.DWDirectives = []string{
					"#DW jobdw name=container-storage type=gfs2 capacity=1GB",
					"#DW persistentdw name=" + persistentStorageName,
					fmt.Sprintf("#DW container name=container profile=%s "+
						// local storage is missing
						"DW_PERSISTENT_foo_persistent_storage=%s",
						containerProfile.Name, persistentStorageName),
				}
				Expect(k8sClient.Create(context.TODO(), workflow)).Should(Succeed())

				Eventually(func(g Gomega) bool {
					g.Expect(k8sClient.Get(context.TODO(), key, workflow)).To(Succeed())
					return !workflow.Status.Ready && workflow.Status.Status == dwsv1alpha7.StatusError
				}).Should(BeTrue(), "be in error state")
			})
		})

		Context("when an argument is present in the container directive but not in the container profile", func() {
			var storageArgsList []string
			localStorageName := "local-storage"

			BeforeEach(func() {
				createContainerProfile = false // We'll make a custom version.
				createGlobalLustre = true
			})

			JustBeforeEach(func() {
				// Build a list of storage arguments for the test. This is necessary because things
				// like persistentStorageName are not initialized until the parent's BeforeEach()
				// block, and the Entry() in the DescribeTable() will be translated well before
				// then. So create a list of canned directive arguments for use in the Entries.
				storageArgsList = []string{
					fmt.Sprintf("DW_JOB_foo_local_storage=%s", localStorageName),
					fmt.Sprintf("DW_PERSISTENT_foo_persistent_storage=%s", persistentStorageName),
					fmt.Sprintf("DW_GLOBAL_foo_global_lustre=%s", globalLustre.Spec.MountRoot),
				}
			})

			buildContainerProfile := func(storages []nnfv1alpha10.NnfContainerProfileStorage) {
				By("Creating a profile with specific storages")
				tempProfile := basicNnfContainerProfile("restricted-"+uuid.NewString()[:8], storages)
				containerProfile = createNnfContainerProfile(tempProfile, true)
			}

			buildContainerWorkflowWithArgs := func(args string) {
				By("creating the workflow")
				workflow.Spec.DWDirectives = []string{
					fmt.Sprintf("#DW jobdw name=%s type=gfs2 capacity=1GB", localStorageName),
					fmt.Sprintf("#DW persistentdw name=%s", persistentStorageName),
					fmt.Sprintf("#DW container name=container profile=%s %s", containerProfile.Name, args),
				}
				Expect(k8sClient.Create(context.TODO(), workflow)).Should(Succeed())
			}

			DescribeTable("should not go to Proposal Ready",
				func(argIdx int, storages []nnfv1alpha10.NnfContainerProfileStorage) {
					buildContainerProfile(storages)
					buildContainerWorkflowWithArgs(storageArgsList[argIdx])
					Eventually(func(g Gomega) bool {
						g.Expect(k8sClient.Get(context.TODO(), key, workflow)).To(Succeed())
						return workflow.Status.Status == dwsv1alpha7.StatusError &&
							strings.Contains(workflow.Status.Message, "not found in container profile")
					}).Should(BeTrue(), "does not reach desired Proposal state")
				},

				Entry("when DW_JOB_ not present in the container profile", 0,
					[]nnfv1alpha10.NnfContainerProfileStorage{
						{Name: "DW_PERSISTENT_foo_persistent_storage", Optional: true},
						{Name: "DW_GLOBAL_foo_global_lustre", Optional: true},
					},
				),
				Entry("when DW_PERSISTENT_ not present in the container profile", 1,
					[]nnfv1alpha10.NnfContainerProfileStorage{
						{Name: "DW_JOB_foo_local_storage", Optional: true},
						{Name: "DW_GLOBAL_foo_global_lustre", Optional: true},
					},
				),
				Entry("when DW_GLOBAL_ not present in the container profile", 2,
					[]nnfv1alpha10.NnfContainerProfileStorage{
						{Name: "DW_JOB_foo_local_storage", Optional: true},
						{Name: "DW_PERSISTENT_foo_persistent_storage", Optional: true},
					},
				),
			)
		})

		Context("when an unsupported jobdw container filesystem type is specified", func() {
			localStorageName := "local-storage"

			buildContainerWorkflowWithJobDWType := func(fsType string) {
				By("creating the workflow")
				workflow.Spec.DWDirectives = []string{
					fmt.Sprintf("#DW jobdw name=%s type=%s capacity=1GB", localStorageName, fsType),
					fmt.Sprintf("#DW container name=container profile=%s DW_JOB_foo_local_storage=%s",
						containerProfile.Name, localStorageName),
				}
				Expect(k8sClient.Create(context.TODO(), workflow)).Should(Succeed())
			}

			DescribeTable("should reach the desired Proposal state",
				func(fsType string, shouldError bool) {
					buildContainerWorkflowWithJobDWType(fsType)
					Eventually(func(g Gomega) bool {
						g.Expect(k8sClient.Get(context.TODO(), key, workflow)).To(Succeed())
						if shouldError {
							return workflow.Status.Status == dwsv1alpha7.StatusError &&
								strings.Contains(workflow.Status.Message, "unsupported container filesystem")
						} else {
							return workflow.Status.Ready == true
						}
					}).Should(BeTrue(), "should reach desired Proposal state")

				},
				Entry("when gfs2 jobdw storage is used", "gfs2", false),
				Entry("when lustre jobdw storage is used", "lustre", false),
				Entry("when xfs jobdw storage is used", "xfs", true),
				Entry("when raw jobdw storage is used", "raw", true),
			)
		})

		Context("when an unsupported persistentdw container filesystem type is specified", func() {

			BeforeEach(func() {
				createPersistent = false
			})

			buildContainerWorkflowWithPersistentDWType := func() {
				By("creating the workflow")
				workflow.Spec.DWDirectives = []string{
					fmt.Sprintf("#DW persistentdw name=%s", persistentStorageName),
					fmt.Sprintf("#DW container name=container profile=%s DW_PERSISTENT_foo_persistent_storage=%s",
						containerProfile.Name, persistentStorageName),
				}
				Expect(k8sClient.Create(context.TODO(), workflow)).Should(Succeed())
			}

			DescribeTable("should reach the desired Proposal state",
				func(fsType string, shouldError bool) {
					createPersistentStorageInstance(persistentStorageName, fsType)
					buildContainerWorkflowWithPersistentDWType()
					Eventually(func(g Gomega) bool {
						g.Expect(k8sClient.Get(context.TODO(), key, workflow)).To(Succeed())
						if shouldError {
							return workflow.Status.Status == dwsv1alpha7.StatusError &&
								strings.Contains(workflow.Status.Message, "unsupported container filesystem: "+fsType)
						} else {
							return workflow.Status.Ready == true
						}
					}).Should(BeTrue(), "should reach desired Proposal state")

				},
				Entry("when gfs2 persistentdw storage is used", "gfs2", false),
				Entry("when lustre persistentdw storage is used", "lustre", false),
				Entry("when xfs persistentdw storage is used", "xfs", true),
				Entry("when raw persistentdw storage is used", "raw", true),
			)
		})
	})
})

var _ = Describe("NnfContainerProfile Webhook test", func() {
	// The nnfcontainer_webhook_test.go covers testing of the webhook.
	// This spec exists only to verify that the webhook is also running for
	// the controller tests.
	It("fails to create an invalid profile to verify that the webhook is installed", func() {
		profileInvalid := basicNnfContainerProfile("invalid-"+uuid.NewString()[:8], nil)
		profileInvalid.Data.NnfSpec = nil
		profileInvalid.Data.NnfMPISpec = nil
		Expect(createNnfContainerProfile(profileInvalid, false)).To(BeNil())
	})
})

var _ = Describe("NnfStorageProfile Webhook test", func() {
	// The nnfstorageprofile_webhook_test.go covers testing of the webhook.
	// This spec exists only to verify that the webhook is also running for
	// the controller tests.
	It("fails to create an invalid profile to verify that the webhook is installed", func() {
		profileInvalid := basicNnfStorageProfile("invalid-" + uuid.NewString()[:8])
		profileInvalid.Data.LustreStorage.MgtOptions.ExternalMGS = "10.0.0.1@tcp"
		profileInvalid.Data.LustreStorage.CombinedMGTMDT = true
		Expect(createNnfStorageProfile(profileInvalid, false)).To(BeNil())
	})
})

func WaitForDMMReady(dmm *nnfv1alpha10.NnfDataMovementManager) {
	Eventually(func(g Gomega) bool {
		g.Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(dmm), dmm)).To(Succeed())
		if !dmm.Status.Ready {
			dmm.Status.Ready = true
			g.Expect(k8sClient.Status().Update(context.TODO(), dmm)).To(Succeed())
		}
		return dmm.Status.Ready
	}).Should(BeTrue())
}
