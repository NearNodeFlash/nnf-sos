/*
 * Copyright 2021, 2022 Hewlett Packard Enterprise Development LP
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

package controllers

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	dwsv1alpha1 "github.hpe.com/hpe/hpc-dpm-dws-operator/api/v1alpha1"
	lusv1alpha1 "github.hpe.com/hpe/hpc-rabsw-lustre-fs-operator/api/v1alpha1"
	nnfv1alpha1 "github.hpe.com/hpe/hpc-rabsw-nnf-sos/api/v1alpha1"
)

var _ = Describe("Integration Test", func() {

	const timeout = 10 * time.Second
	const interval = 100 * time.Millisecond

	type fsToTest struct {
		fsType                 string
		expectedAllocationSets int
	}

	var filesystems = []fsToTest{
		//{"raw", 1}, // RABSW-843: Raw device support. Matt's working on this and once resolved this test can be enabled
		//{"lvm", 1},
		{"xfs", 1},
		{"gfs2", 1},
		{"lustre", 3},
	}

	var (
		workflow       *dwsv1alpha1.Workflow
		nodeNames      []string
		setup          sync.Once
		storageProfile *nnfv1alpha1.NnfStorageProfile
	)

	advanceState := func(state dwsv1alpha1.WorkflowState) {
		By(fmt.Sprintf("Advancing to %s state", state))
		Eventually(func() error {
			Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(workflow), workflow)).To(Succeed())
			workflow.Spec.DesiredState = state.String()
			return k8sClient.Update(context.TODO(), workflow)
		}).Should(Succeed(), fmt.Sprintf("Advancing to %s state", state))

		By(fmt.Sprintf("Waiting on state %s", state))
		Eventually(func() string {
			Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(workflow), workflow)).To(Succeed())
			return workflow.Status.State
		}).WithTimeout(timeout).Should(Equal(state.String()), fmt.Sprintf("Waiting on state %s", state))
	}

	advanceStateAndCheckReady := func(state dwsv1alpha1.WorkflowState) {

		advanceState(state)

		// If we're currently in a staging state, ensure the data movement status is marked as finished so
		// we can successfully transition out of that state.
		if state == dwsv1alpha1.StateDataIn || state == dwsv1alpha1.StateDataOut || state == dwsv1alpha1.StatePostRun {

			findDataMovementDirectiveIndex := func() int {
				for idx, directive := range workflow.Spec.DWDirectives {
					if state == dwsv1alpha1.StateDataIn && strings.HasPrefix(directive, "#DW copy_in") {
						return idx
					}
					if state == dwsv1alpha1.StateDataOut && strings.HasPrefix(directive, "#DW copy_out") {
						return idx
					}
				}

				return -1
			}

			dm := &nnfv1alpha1.NnfDataMovement{
				ObjectMeta: metav1.ObjectMeta{
					Name:      workflow.Name,
					Namespace: workflow.Namespace,
				},
			}

			if state != dwsv1alpha1.StatePostRun {

				if findDataMovementDirectiveIndex() >= 0 {
					dm.ObjectMeta = metav1.ObjectMeta{
						Name:      fmt.Sprintf("%s-%d", workflow.Name, findDataMovementDirectiveIndex()),
						Namespace: workflow.Namespace,
					}
				} else {
					dm = nil
				}
			}

			if dm != nil {

				Eventually(func() error {
					return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(dm), dm)
				}).WithTimeout(timeout).Should(Succeed())

				Eventually(func() error {
					Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(dm), dm)).Should(Succeed())

					dm.Status.Conditions = []metav1.Condition{
						{
							Status:             metav1.ConditionTrue,
							Reason:             nnfv1alpha1.DataMovementConditionReasonSuccess,
							Type:               nnfv1alpha1.DataMovementConditionTypeFinished,
							LastTransitionTime: metav1.Now(),
							Message:            "",
						},
					}

					return k8sClient.Status().Update(context.TODO(), dm)
				}).Should(Succeed())
			}
		}

		By("Waiting to go ready")
		Eventually(func() bool {
			Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(workflow), workflow)).To(Succeed())
			return workflow.Status.Ready
		}).WithTimeout(timeout).Should(BeTrue(), fmt.Sprintf("Waiting on ready status state %s", state))
	}

	BeforeEach(func() {

		setup.Do(func() {

			// Initialize node names - currently set to three to satisify the lustre requirement of single MDT, MGT, OST
			// NOTE: Node names require the "rabbit" prefix to ensure client mounts occur on the correct controller
			nodeNames = []string{
				"rabbit-test-node-0",
				"rabbit-test-node-1",
				"rabbit-test-node-2",
			}

			// Build the config map that ties everything together; this also
			// creates a namespace for each compute node which is required for
			// client mount.
			computeNameGeneratorFunc := func() func() []dwsv1alpha1.SystemConfigurationComputeNodeReference {
				nextComputeIndex := 0
				return func() []dwsv1alpha1.SystemConfigurationComputeNodeReference {
					computes := make([]dwsv1alpha1.SystemConfigurationComputeNodeReference, 16)
					for i := 0; i < 16; i++ {
						name := fmt.Sprintf("compute%d", i+nextComputeIndex)

						computes[i].Name = name
						computes[i].Index = i
					}
					nextComputeIndex += 16
					return computes
				}
			}

			generator := computeNameGeneratorFunc()
			configSpec := dwsv1alpha1.SystemConfigurationSpec{}
			for _, nodeName := range nodeNames {
				storageNode := dwsv1alpha1.SystemConfigurationStorageNode{
					Type: "Rabbit",
					Name: nodeName,
				}

				storageNode.ComputesAccess = generator()
				configSpec.StorageNodes = append(configSpec.StorageNodes, storageNode)
				for _, computeAccess := range storageNode.ComputesAccess {
					compute := dwsv1alpha1.SystemConfigurationComputeNode{Name: computeAccess.Name}
					configSpec.ComputeNodes = append(configSpec.ComputeNodes, compute)
				}
			}

			config := &dwsv1alpha1.SystemConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "default",
					Namespace: corev1.NamespaceDefault,
				},
				Spec: configSpec,
			}

			Expect(k8sClient.Create(context.TODO(), config)).To(Succeed())

			// Each node gets a namespace, a node, and an NNF Node. Node would typically be handled
			// by kubernetes and then an NNF Node & Namespace are started by the NLC; but for test
			// we have to bootstrap all that.
			for _, nodeName := range nodeNames {
				// Create the namespace
				ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
					Name: nodeName,
				}}

				Expect(k8sClient.Create(context.TODO(), ns)).To(Succeed())

				// Create the node - set it to up as ready
				node := &corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name:      nodeName,
						Namespace: corev1.NamespaceDefault,
						Labels: map[string]string{
							"cray.nnf.node": "true",
						},
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

				Expect(k8sClient.Create(context.TODO(), node)).To(Succeed())

				// Create the NNF Node
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

				// Check that the DWS storage resource was updated with the compute node information
				storage := &dwsv1alpha1.Storage{}
				namespacedName := types.NamespacedName{
					Name:      nodeName,
					Namespace: corev1.NamespaceDefault,
				}

				Eventually(func() error {
					return k8sClient.Get(context.TODO(), namespacedName, storage)
				}).Should(Succeed())

				Eventually(func() bool {
					Expect(k8sClient.Get(context.TODO(), namespacedName, storage)).To(Succeed())
					return len(storage.Data.Access.Computes) == 16
				}).Should(BeTrue())

				// Check that a namespace was created for each compute node
				for i := 0; i < len(nodeNames)*16; i++ {
					namespace := &corev1.Namespace{}
					Eventually(func() error {
						return k8sClient.Get(context.TODO(), types.NamespacedName{Name: fmt.Sprintf("compute%d", i)}, namespace)
					}).Should(Succeed())
				}
			}
		}) // once

		// Create a default NnfStorageProfile for the unit tests.
		storageProfile = createBasicDefaultNnfStorageProfile()
	})

	AfterEach(func() {
		Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(workflow), workflow)).To(Succeed())
		Expect(k8sClient.Delete(context.TODO(), workflow)).To(Succeed())

		Eventually(func() error {
			return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(workflow), workflow)
		}).WithTimeout(timeout).ShouldNot(Succeed())

		workflow = nil

		Expect(k8sClient.Delete(context.TODO(), storageProfile)).To(Succeed())
	})

	for idx := range filesystems {
		fsType := filesystems[idx].fsType
		expectedAllocationSets := filesystems[idx].expectedAllocationSets
		wfid := uuid.NewString()[0:8]

		It(fmt.Sprintf("Testing file system '%s'", fsType), func() {

			workflow = &dwsv1alpha1.Workflow{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("%s-%s", fsType, wfid),
					Namespace: corev1.NamespaceDefault,
				},
				Spec: dwsv1alpha1.WorkflowSpec{
					DesiredState: dwsv1alpha1.StateProposal.String(),
					JobID:        idx,
					WLMID:        "Test WLMID",
					DWDirectives: []string{
						fmt.Sprintf("#DW jobdw name=%s type=%s capacity=1GiB", "test-0", fsType),
						//fmt.Sprintf("#DW jobdw name=%s type=%s capacity=1GiB", "test-1", fsType),
					},
				},
			}

			By("Creating the workflow")
			Expect(k8sClient.Create(context.TODO(), workflow)).To(Succeed())

			By("Retrieving created workflow")
			Eventually(func() error {
				return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(workflow), workflow)
			}).Should(Succeed())

			// Store ownership reference to workflow - this is checked for many of the created objects
			controller := true
			blockOwnerDeletion := true
			ownerRef := metav1.OwnerReference{
				Kind:               reflect.TypeOf(dwsv1alpha1.Workflow{}).Name(),
				APIVersion:         dwsv1alpha1.GroupVersion.String(),
				UID:                workflow.GetUID(),
				Name:               workflow.GetName(),
				Controller:         &controller,
				BlockOwnerDeletion: &blockOwnerDeletion,
			}

			/*************************** Proposal ****************************/

			By("Checking proposal state and ready")
			Eventually(func() bool {
				Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(workflow), workflow)).To(Succeed())
				return workflow.Status.State == dwsv1alpha1.StateProposal.String() && workflow.Status.Ready
			}).Should(BeTrue())

			By("Checking for Computes resource")
			computes := &dwsv1alpha1.Computes{
				ObjectMeta: metav1.ObjectMeta{
					Name:      workflow.Status.Computes.Name,
					Namespace: workflow.Status.Computes.Namespace,
				},
			}
			Expect(workflow.Status.Computes.Kind).To(Equal(reflect.TypeOf(dwsv1alpha1.Computes{}).Name()))
			Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(computes), computes)).To(Succeed())
			Expect(computes.ObjectMeta.OwnerReferences).To(ContainElement(ownerRef))

			By("Checking various DW Directive Breakdowns")
			for _, dbdRef := range workflow.Status.DirectiveBreakdowns {
				dbd := &dwsv1alpha1.DirectiveBreakdown{}
				Expect(dbdRef.Kind).To(Equal(reflect.TypeOf(dwsv1alpha1.DirectiveBreakdown{}).Name()))

				By("Checking DW Directive Breakdown")
				Expect(k8sClient.Get(context.TODO(), types.NamespacedName{Name: dbdRef.Name, Namespace: dbdRef.Namespace}, dbd)).To(Succeed())
				Expect(dbd.ObjectMeta.OwnerReferences).To(ContainElement(ownerRef))
				Expect(dbd.Status.AllocationSet).To(HaveLen(expectedAllocationSets))

				By("DW Directive Breakdown should go ready")
				Eventually(func() bool {
					Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(dbd), dbd)).To(Succeed())
					return dbd.Status.Ready
				}).Should(BeTrue())

				By("Checking DW Directive has Servers resource")
				servers := &dwsv1alpha1.Servers{}
				Expect(dbd.Status.Servers.Kind).To(Equal(reflect.TypeOf(dwsv1alpha1.Servers{}).Name()))
				Expect(k8sClient.Get(context.TODO(), types.NamespacedName{Name: dbd.Status.Servers.Name, Namespace: dbd.Status.Servers.Namespace}, servers)).To(Succeed())

				By("Checking servers resource")
				Expect(servers.ObjectMeta.OwnerReferences).To(ContainElement(metav1.OwnerReference{
					Kind:               reflect.TypeOf(dwsv1alpha1.DirectiveBreakdown{}).Name(),
					APIVersion:         dwsv1alpha1.GroupVersion.String(),
					UID:                dbd.GetUID(),
					Name:               dbd.GetName(),
					Controller:         &controller,
					BlockOwnerDeletion: &blockOwnerDeletion,
				}))

				By("Assigning storage")
				if fsType != "lustre" {
					// If non-lustre, allocate storage on all the Rabbit nodes in test.
					storage := make([]dwsv1alpha1.ServersSpecStorage, 0, len(nodeNames))
					for _, nodeName := range nodeNames {
						storage = append(storage, dwsv1alpha1.ServersSpecStorage{
							AllocationCount: 1,
							Name:            nodeName,
						})
					}

					Expect(dbd.Status.AllocationSet).To(HaveLen(1))
					allocSet := &dbd.Status.AllocationSet[0]

					servers.Spec.AllocationSets = make([]dwsv1alpha1.ServersSpecAllocationSet, 1)
					servers.Spec.AllocationSets[0] = dwsv1alpha1.ServersSpecAllocationSet{
						AllocationSize: allocSet.MinimumCapacity,
						Label:          allocSet.Label,
						Storage:        storage,
					}

				} else {
					// If lustre, allocate one node per allocation set
					Expect(len(nodeNames) >= len(dbd.Status.AllocationSet)).To(BeTrue())
					servers.Spec.AllocationSets = make([]dwsv1alpha1.ServersSpecAllocationSet, len(dbd.Status.AllocationSet))
					for idx, allocset := range dbd.Status.AllocationSet {
						servers.Spec.AllocationSets[idx] = dwsv1alpha1.ServersSpecAllocationSet{
							AllocationSize: allocset.MinimumCapacity,
							Label:          allocset.Label,
							Storage: []dwsv1alpha1.ServersSpecStorage{
								{
									AllocationCount: 1,
									Name:            nodeNames[idx],
								},
							},
						}
					}
				}

				Expect(k8sClient.Update(context.TODO(), servers)).To(Succeed())
			}

			By("Assigning computes")
			Expect(computes.Data).To(HaveLen(0))
			computes.Data = make([]dwsv1alpha1.ComputesData, 0, len(nodeNames))
			for idx := range nodeNames {
				computes.Data = append(computes.Data, dwsv1alpha1.ComputesData{Name: fmt.Sprintf("compute%d", idx*16)})
			}
			Expect(k8sClient.Update(context.TODO(), computes)).To(Succeed())

			/***************************** Setup *****************************/

			advanceStateAndCheckReady(dwsv1alpha1.StateSetup)

			By("Checking Setup state")
			// TODO

			/**************************** Data In ****************************/

			advanceStateAndCheckReady(dwsv1alpha1.StateDataIn)

			By("Checking Data In state")
			// TODO

			/**************************** Pre Run ****************************/

			advanceStateAndCheckReady(dwsv1alpha1.StatePreRun)

			By("Checking Pre Run state")

			for _, dbdRef := range workflow.Status.DirectiveBreakdowns {
				dbd := &dwsv1alpha1.DirectiveBreakdown{}
				Expect(k8sClient.Get(context.TODO(), types.NamespacedName{Name: dbdRef.Name, Namespace: dbdRef.Namespace}, dbd)).To(Succeed())

				By("Check for an NNF Access describing the computes")
				access := &nnfv1alpha1.NnfAccess{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("%s-%d-%s", workflow.Name, dbd.Spec.DW.DWDirectiveIndex, "computes"),
						Namespace: workflow.Namespace,
					},
				}
				Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(access), access)).To(Succeed())
				Expect(access.ObjectMeta.OwnerReferences).To(ContainElement(ownerRef))
				Expect(access.Spec).To(MatchFields(IgnoreExtras, Fields{
					"TeardownState": Equal(dwsv1alpha1.StatePostRun.String()),
					"DesiredState":  Equal("mounted"),
					"Target":        Equal("single"),
				}))
				Expect(access.Status.State).To(Equal("mounted"))
				Expect(access.Status.Ready).To(BeTrue())

				By("Checking NNF Access computes reference exists")
				Expect(access.Spec.ClientReference).To(MatchFields(IgnoreExtras, Fields{
					"Name":      Equal(workflow.Name),
					"Namespace": Equal(workflow.Namespace),
					"Kind":      Equal(reflect.TypeOf(dwsv1alpha1.Computes{}).Name()),
				}))
				computes := &dwsv1alpha1.Computes{
					ObjectMeta: metav1.ObjectMeta{
						Name:      access.Spec.ClientReference.Name,
						Namespace: access.Spec.ClientReference.Namespace,
					},
				}
				Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(computes), computes)).To(Succeed())

				By("Checking NNF Access storage reference exists")
				Expect(access.Spec.StorageReference).To(MatchFields(IgnoreExtras, Fields{
					"Name":      Equal(dbd.Name),      // Servers is created from the DBD, and NNFStorage is created from servers
					"Namespace": Equal(dbd.Namespace), // Servers is created from the DBD, and NNFStorage is created from servers
					"Kind":      Equal(reflect.TypeOf(nnfv1alpha1.NnfStorage{}).Name()),
				}))
				storage := &nnfv1alpha1.NnfStorage{
					ObjectMeta: metav1.ObjectMeta{
						Name:      access.Spec.StorageReference.Name,
						Namespace: access.Spec.StorageReference.Namespace,
					},
				}
				Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(storage), storage)).To(Succeed())

				By("Checking for a Client Mount on each compute")
				for _, compute := range computes.Data {
					clientMount := &dwsv1alpha1.ClientMount{
						ObjectMeta: metav1.ObjectMeta{
							Name:      clientMountName(access),
							Namespace: compute.Name,
						},
					}
					Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(clientMount), clientMount)).To(Succeed())
					Expect(clientMount.Status.Mounts).To(HaveLen(1))
					Expect(clientMount.Status.Mounts[0].Ready).To(BeTrue())
				}

				// For shared file systems, there should also be a NNF Access for the Rabbit as well as corresponding Client Mounts per Rabbit
				if fsType == "gfs2" {
					By("Checking for an NNF Access describing the servers")
					access := &nnfv1alpha1.NnfAccess{
						ObjectMeta: metav1.ObjectMeta{
							Name:      fmt.Sprintf("%s-%d-%s", workflow.Name, dbd.Spec.DW.DWDirectiveIndex, "servers"),
							Namespace: workflow.Namespace,
						},
					}
					Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(access), access)).To(Succeed())
					Expect(access.ObjectMeta.OwnerReferences).To(ContainElement(ownerRef))

					// CONTINUE HERE WITH CHECKS FOR storage, status

					// CONTINUE HERE WITH CHECKS FOR Client Mounts on all the Rabbits
				}
			}

			/*************************** Post Run ****************************/

			advanceStateAndCheckReady(dwsv1alpha1.StatePostRun)

			By("Checking Post Run state")

			for _, dbdRef := range workflow.Status.DirectiveBreakdowns {
				dbd := &dwsv1alpha1.DirectiveBreakdown{}
				Expect(k8sClient.Get(context.TODO(), types.NamespacedName{Name: dbdRef.Name, Namespace: dbdRef.Namespace}, dbd)).To(Succeed())

				By("Check that NNF Access describing computes is not present")
				access := &nnfv1alpha1.NnfAccess{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("%s-%d-%s", workflow.Name, dbd.Spec.DW.DWDirectiveIndex, "computes"),
						Namespace: workflow.Namespace,
					},
				}
				Eventually(func() error {
					err := k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(access), access)
					return client.IgnoreNotFound(err)
				}).Should(Succeed())

				By("Check that all Client Mounts for computes are removed")
				// TODO

				if fsType == "gfs2" {
					By("Check that NNF Access describing computes is not present")
					access := &nnfv1alpha1.NnfAccess{
						ObjectMeta: metav1.ObjectMeta{
							Name:      fmt.Sprintf("%s-%d-%s", workflow.Name, dbd.Spec.DW.DWDirectiveIndex, "servers"),
							Namespace: workflow.Namespace,
						},
					}
					Eventually(func() error {
						err := k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(access), access)
						return client.IgnoreNotFound(err)
					}).Should(Succeed())

					By("Check that all Client Mounts for servers are removed")
					// TODO
				}
			}

			/**************************** Data Out ****************************/

			advanceStateAndCheckReady(dwsv1alpha1.StateDataOut)

			By("Checking Data Out state")
			// TODO

			/**************************** Teardown ****************************/

			advanceStateAndCheckReady(dwsv1alpha1.StateTeardown)

			By("Checking Teardown state")
			// TODO: Check that all the objects are deleted

		}) // It(fmt.Sprintf("Testing file system '%s'", fsType)

	} // for idx := range filesystems

	// Here we test the various data movement directives. This uses a workflow
	// per test where the workflow is brought to data in stage. This is not
	// expected to be a full workflow test; but singles out the data movement
	// directives (copy_in and copy_out).
	Describe("Test with data movement directives", func() {

		// Setup a basic workflow; each test is expected to fill in the DWDirectives
		// in its BeforeEach() clause.
		BeforeEach(func() {
			wfid := uuid.NewString()[0:8]
			workflow = &dwsv1alpha1.Workflow{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("gfs2-%s", wfid),
					Namespace: corev1.NamespaceDefault,
				},
				Spec: dwsv1alpha1.WorkflowSpec{
					DesiredState: dwsv1alpha1.StateProposal.String(),
					JobID:        -1,
					WLMID:        "Test WLMID",
				},
			}
		})

		// Bring the workflow up to Data In; assign the necessary servers and computes
		// resource. This isn't meant to be a vigorous test of the proposal and setup
		// stages; that is provided by the topmost integration test.
		JustBeforeEach(func() {
			By("Create the workflow")
			Expect(k8sClient.Create(context.TODO(), workflow)).To(Succeed())

			By("Checking for workflow creation")
			Eventually(func() error {
				return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(workflow), workflow)
			}).WithTimeout(timeout).Should(Succeed())

			/*************************** Proposal ****************************/

			By("Checking for proposal state and ready")
			Eventually(func() bool {
				Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(workflow), workflow)).To(Succeed())
				return workflow.Status.State == dwsv1alpha1.StateProposal.String() && workflow.Status.Ready
			}).Should(BeTrue())

			for _, dbdRef := range workflow.Status.DirectiveBreakdowns {
				By("Checking DW Directive Breakdown")
				dbd := &dwsv1alpha1.DirectiveBreakdown{}
				Expect(k8sClient.Get(context.TODO(), types.NamespacedName{Name: dbdRef.Name, Namespace: dbdRef.Namespace}, dbd)).To(Succeed())

				Expect(dbd.Status.AllocationSet).To(HaveLen(1))
				allocSet := &dbd.Status.AllocationSet[0]

				By("Assigning storage")
				servers := &dwsv1alpha1.Servers{}
				Expect(k8sClient.Get(context.TODO(), types.NamespacedName{Name: dbd.Status.Servers.Name, Namespace: dbd.Status.Servers.Namespace}, servers)).To(Succeed())

				storage := make([]dwsv1alpha1.ServersSpecStorage, 0, len(nodeNames))
				for _, nodeName := range nodeNames {
					storage = append(storage, dwsv1alpha1.ServersSpecStorage{
						AllocationCount: 1,
						Name:            nodeName,
					})
				}

				servers.Spec.AllocationSets = []dwsv1alpha1.ServersSpecAllocationSet{
					{
						AllocationSize: allocSet.MinimumCapacity,
						Label:          allocSet.Label,
						Storage:        storage,
					},
				}

				Expect(k8sClient.Update(context.TODO(), servers)).To(Succeed())
			} // for _, dbdRef := range workflow.Status.DirectiveBreakdowns

			/***************************** Setup *****************************/

			advanceStateAndCheckReady(dwsv1alpha1.StateSetup)

			/**************************** Data In ****************************/

			advanceStateAndCheckReady(dwsv1alpha1.StateDataIn)
		})

		AfterEach(func() {

			/**************************** Teardown ****************************/

			advanceStateAndCheckReady(dwsv1alpha1.StateTeardown)

		})

		var lustre *lusv1alpha1.LustreFileSystem

		BeforeEach(func() {
			lustre = &lusv1alpha1.LustreFileSystem{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "lustre-test",
					Namespace: corev1.NamespaceDefault,
				},
				Spec: lusv1alpha1.LustreFileSystemSpec{
					Name:      "lustre",
					MgsNid:    "172.0.0.1@tcp",
					MountRoot: "/lus/maui",
				},
			}

			Expect(k8sClient.Create(context.TODO(), lustre)).To(Succeed())
		})

		AfterEach(func() {
			Expect(k8sClient.Delete(context.TODO(), lustre)).To(Succeed())
		})

		validateNnfAccessHasCorrectTeardownState := func(state dwsv1alpha1.WorkflowState) {
			Expect(workflow.Status.DirectiveBreakdowns).To(HaveLen(1))

			access := &nnfv1alpha1.NnfAccess{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("%s-%d-%s", workflow.Name, 0, "servers"),
					Namespace: workflow.Namespace,
				},
			}
			Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(access), access)).To(Succeed())

			By("Validate access specification is correct")
			Expect(access.Spec).To(MatchFields(IgnoreExtras, Fields{
				"TeardownState": Equal(state.String()),
				"DesiredState":  Equal("mounted"),
				"Target":        Equal("all"),
			}))

			By("Validate access status is correct")
			Expect(access.Status.State).To(Equal("mounted"))
			Expect(access.Status.Ready).To(BeTrue())
		}

		validateNnfAccessIsNotFound := func() {
			Expect(workflow.Status.DirectiveBreakdowns).To(HaveLen(1))
			access := &nnfv1alpha1.NnfAccess{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("%s-%d-%s", workflow.Name, 0, "servers"),
					Namespace: workflow.Namespace,
				},
			}

			Eventually(func() error {
				err := k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(access), access)
				return client.IgnoreNotFound(err)
			}).Should(Succeed())
		}

		// If there is only a copy_in directive, then expect the lifetime of NNF Access setup
		// in data_in and deleted in post-run.
		Describe("Test with copy_in directive", func() {
			BeforeEach(func() {
				workflow.Spec.DWDirectives = []string{
					"#DW jobdw name=test-dm-0 type=gfs2 capacity=1GiB",
					"#DW copy_in source=/lus/maui/file destination=$JOB_DW_test-dm-0/file",
				}
			})

			It("Validates Workflow", func() {
				Expect(workflow.Status.State).To(Equal(dwsv1alpha1.StateDataIn.String()))

				By("Check that NNF Access is created, with deletion in post-run")
				validateNnfAccessHasCorrectTeardownState(dwsv1alpha1.StatePostRun)

				By("Advancing to post run, ensure NNF Access is deleted")
				advanceStateAndCheckReady(dwsv1alpha1.StatePreRun)
				advanceStateAndCheckReady(dwsv1alpha1.StatePostRun)
				validateNnfAccessIsNotFound()
			})
		})

		// If there is only a copy_out directive, NNF Access is not needed until pre-run
		// and should remain until post-run.
		Describe("Test with copy_out directive", func() {
			BeforeEach(func() {
				workflow.Spec.DWDirectives = []string{
					"#DW jobdw name=test-dm-0 type=gfs2 capacity=1GiB",
					"#DW copy_out source=$JOB_DW_test-dm-0/file destination=/lus/maui/file",
				}
			})

			It("Validates Workflow", func() {
				Expect(workflow.Status.State).To(Equal(dwsv1alpha1.StateDataIn.String()))

				validateNnfAccessIsNotFound()
				advanceStateAndCheckReady(dwsv1alpha1.StatePreRun)

				By("Validate NNF Access is created, with deletion in data-out")
				validateNnfAccessHasCorrectTeardownState(dwsv1alpha1.StateDataOut)

				By("Advancing to post run, ensure NNF Access is still set for deletion in data-out")
				advanceStateAndCheckReady(dwsv1alpha1.StatePostRun)
				validateNnfAccessHasCorrectTeardownState(dwsv1alpha1.StateDataOut)

				By("Advancing to data-out, ensure NNF Access is deleted")
				advanceStateAndCheckReady(dwsv1alpha1.StateDataOut)
				validateNnfAccessIsNotFound()
			})
		})

		Describe("Test with copy_in and copy_out directives", func() {
			BeforeEach(func() {
				workflow.Spec.DWDirectives = []string{
					"#DW jobdw name=test-dm-0 type=gfs2 capacity=1GiB",
					"#DW copy_in source=/lus/maui/file destination=$JOB_DW_test-dm-0/file",
					"#DW copy_out source=$JOB_DW_test-dm-0/file destination=/lus/maui/file",
				}
			})

			It("Validates Workflow", func() {
				Expect(workflow.Status.State).To(Equal(dwsv1alpha1.StateDataIn.String()))

				By("Validate NNF Access is created, with deletion in data-out")
				validateNnfAccessHasCorrectTeardownState(dwsv1alpha1.StateDataOut)

				advanceStateAndCheckReady(dwsv1alpha1.StatePreRun)
				advanceStateAndCheckReady(dwsv1alpha1.StatePostRun)

				By("Advancing to data-out, ensure NNF Access is deleted")
				advanceStateAndCheckReady(dwsv1alpha1.StateDataOut)
				validateNnfAccessIsNotFound()
			})

		})

		Describe("Test with failure injection of error encountered during run", func() {
			BeforeEach(func() {
				workflow.Spec.DWDirectives = []string{
					"#DW jobdw name=test-dm-0 type=gfs2 capacity=1GiB",
				}
			})

			It("Validates error propgates to workflow", func() {
				Expect(workflow.Status.State).To(Equal(dwsv1alpha1.StateDataIn.String()))

				advanceStateAndCheckReady(dwsv1alpha1.StatePreRun)

				By("Injecting an error in the data movement resource")
				dm := &nnfv1alpha1.NnfDataMovement{}

				Eventually(func() error {
					return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(workflow), dm)
				}).Should(Succeed())

				dm.Status.Conditions = []metav1.Condition{{
					Status:             metav1.ConditionTrue,
					Reason:             nnfv1alpha1.DataMovementConditionReasonFailed,
					Type:               nnfv1alpha1.DataMovementConditionTypeFinished,
					LastTransitionTime: metav1.Now(),
					Message:            "Error Injection",
				}}

				Expect(k8sClient.Status().Update(context.TODO(), dm)).To(Succeed())

				By("Advancing to post-run")
				advanceState(dwsv1alpha1.StatePostRun)

				By("Checking the driver has an error present")
				Eventually(func() *dwsv1alpha1.WorkflowDriverStatus {
					Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(workflow), workflow)).To(Succeed())

					driverId := os.Getenv("DWS_DRIVER_ID")
					for _, driver := range workflow.Status.Drivers {
						if driver.DriverID == driverId {
							if driver.Reason == "error" {
								return &driver
							}
						}
					}

					return nil
				}).WithTimeout(timeout).ShouldNot(BeNil())
			})
		})

	})
})

var _ = Describe("Empty #DW List Test", func() { // TODO: Re-enable this
	const (
		WorkflowNamespace = "default"
		WorkflowID        = "test"
	)
	const timeout = time.Second * 10
	const interval = time.Millisecond * 100
	const workflowName = "no-storage"

	var (
		savedWorkflow  *dwsv1alpha1.Workflow
		storageProfile *nnfv1alpha1.NnfStorageProfile
	)

	BeforeEach(func() {
		// Create a default NnfStorageProfile for the unit tests.
		storageProfile = createBasicDefaultNnfStorageProfile()
	})

	AfterEach(func() {
		Expect(k8sClient.Delete(context.TODO(), storageProfile)).To(Succeed())
	})

	Describe(fmt.Sprintf("Creating workflow %s with no #DWs", workflowName), func() {

		It("Should create successfully", func() {
			workflow := &dwsv1alpha1.Workflow{
				ObjectMeta: metav1.ObjectMeta{
					Name:      workflowName,
					Namespace: WorkflowNamespace,
				},
				Spec: dwsv1alpha1.WorkflowSpec{
					DesiredState: "proposal", // TODO: This should be defined somewhere
					WLMID:        WorkflowID,
					DWDirectives: []string{}, // Empty
				},
			}
			Expect(k8sClient.Create(context.Background(), workflow)).Should(Succeed())
		})

		It("Should be in proposal state", func() {
			wf := &dwsv1alpha1.Workflow{}
			Eventually(func() error {
				err := k8sClient.Get(context.Background(), client.ObjectKey{Namespace: WorkflowNamespace, Name: workflowName}, wf)
				return err
			}).Should(Succeed())

			Eventually(func() (string, error) {
				err := k8sClient.Get(context.Background(), client.ObjectKey{Namespace: WorkflowNamespace, Name: workflowName}, wf)
				if err != nil {
					return "", err
				}
				return wf.Status.State, nil
			}).Should(Equal("proposal"))

			savedWorkflow = wf
		})

		It("Should have no directiveBreakdowns", func() {
			directiveBreakdown := &dwsv1alpha1.DirectiveBreakdown{}
			name := workflowName + "-" + strconv.Itoa(0)
			namespace := WorkflowNamespace

			Eventually(func() error {
				return k8sClient.Get(context.Background(), client.ObjectKey{Name: name, Namespace: namespace}, directiveBreakdown)
			}).ShouldNot(Succeed())
		})

		It("Should have a single Computes that is owned by the workflow", func() {
			computes := &dwsv1alpha1.Computes{}
			name := workflowName
			namespace := WorkflowNamespace

			Eventually(func() error {
				return k8sClient.Get(context.Background(), client.ObjectKey{Name: name, Namespace: namespace}, computes)
			}, timeout, interval).Should(Succeed())

			t := true
			expectedOwnerReference := metav1.OwnerReference{
				APIVersion:         savedWorkflow.APIVersion,
				Kind:               savedWorkflow.Kind,
				Name:               savedWorkflow.Name,
				UID:                savedWorkflow.UID,
				Controller:         &t,
				BlockOwnerDeletion: &t,
			}

			Expect(computes.ObjectMeta.OwnerReferences).Should(ContainElement(expectedOwnerReference))
		})

		It("Should have no Servers", func() {
			servers := &dwsv1alpha1.Servers{}
			name := workflowName + "-" + strconv.Itoa(0)
			namespace := WorkflowNamespace

			Eventually(func() error {
				return k8sClient.Get(context.Background(), client.ObjectKey{Name: name, Namespace: namespace}, servers)
			}, timeout, interval).ShouldNot(Succeed())
		})

		It("Should complete proposal state", func() {
			wf := &dwsv1alpha1.Workflow{}
			Eventually(func() (bool, error) {
				err := k8sClient.Get(context.Background(), client.ObjectKey{Namespace: WorkflowNamespace, Name: workflowName}, wf)
				if err != nil {
					return false, err
				}
				return wf.Status.Ready, nil
			}, timeout, interval).Should(BeTrue())
		})
	})

	Describe(fmt.Sprintf("Deleting workflow %s", workflowName), func() {

		It("Should delete successfully", func() {
			workflow := &dwsv1alpha1.Workflow{
				ObjectMeta: metav1.ObjectMeta{
					Name:      workflowName,
					Namespace: WorkflowNamespace,
				},
			}
			Expect(k8sClient.Delete(context.Background(), workflow)).Should(Succeed())
		})

		It("Should be deleted", func() {
			wf := &dwsv1alpha1.Workflow{}
			Eventually(func() error {
				return k8sClient.Get(context.Background(), client.ObjectKey{Namespace: WorkflowNamespace, Name: workflowName}, wf)
			}).ShouldNot(Succeed())
		})
	})
})
