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
	"strings"
	"sync"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	dwsv1alpha1 "github.com/HewlettPackard/dws/api/v1alpha1"
	dwparse "github.com/HewlettPackard/dws/utils/dwdparse"
	lusv1alpha1 "github.com/NearNodeFlash/lustre-fs-operator/api/v1alpha1"
	nnfv1alpha1 "github.com/NearNodeFlash/nnf-sos/api/v1alpha1"
)

var _ = Describe("Integration Test", func() {

	const (
		psiOwnedByWorkflow    = true
		psiNotownedByWorkflow = false
		nnfStoragePresent     = true
		nnfStorageDeleted     = false
	)

	controller := true
	blockOwnerDeletion := true

	var (
		workflow           *dwsv1alpha1.Workflow
		persistentInstance *dwsv1alpha1.PersistentStorageInstance
		nodeNames          []string
		setup              sync.Once
		storageProfile     *nnfv1alpha1.NnfStorageProfile
	)

	advanceState := func(state dwsv1alpha1.WorkflowState, w *dwsv1alpha1.Workflow, testStackOffset int) {
		By(fmt.Sprintf("Advancing to %s state, wf %s", state, w.Name))
		Eventually(func() error {
			Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(w), w)).WithOffset(testStackOffset).To(Succeed())
			w.Spec.DesiredState = state.String()
			return k8sClient.Update(context.TODO(), w)
		}).WithOffset(testStackOffset).Should(Succeed(), fmt.Sprintf("Advancing to %s state", state))

		By(fmt.Sprintf("Waiting on state %s", state))
		Eventually(func() string {
			Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(w), w)).WithOffset(testStackOffset).To(Succeed())
			return w.Status.State
		}).WithOffset(testStackOffset).Should(Equal(state.String()), fmt.Sprintf("Waiting on state %s", state))
	}

	advanceStateAndCheckReady := func(state dwsv1alpha1.WorkflowState, w *dwsv1alpha1.Workflow) {
		By("advanceStateAndCheckReady: advance workflow state")

		// If this method fails, have the test results report where it was called from rather
		// than where it fails in this method.
		// If you'd rather see why this method is failing, set this to 0.
		// If you'd rather see why advanceState() is failing, set this to -1.
		testStackOffset := 1

		// If advanceState fails, we are more interested in what the outer test method
		// was doing as it tried to advanceState versus, why advanceState fails.
		advanceState(state, w, testStackOffset+1)

		// If we're currently in a staging state, ensure the data movement status is marked as finished so
		// we can successfully transition out of that state.
		if state == dwsv1alpha1.StateDataIn || state == dwsv1alpha1.StateDataOut || state == dwsv1alpha1.StatePostRun {

			findDataMovementDirectiveIndex := func() int {
				for idx, directive := range w.Spec.DWDirectives {
					if state == dwsv1alpha1.StateDataIn && strings.HasPrefix(directive, "#DW copy_in") {
						return idx
					}
					if state == dwsv1alpha1.StateDataOut && strings.HasPrefix(directive, "#DW copy_out") {
						return idx
					}
				}

				return -1
			}

			workflowExpectsDatamovementInPostRun := func() bool {
				for _, directive := range w.Spec.DWDirectives {
					if strings.HasPrefix(directive, "#DW create_persistent") {
						return false
					}
					if strings.HasPrefix(directive, "#DW destroy_persistent") {
						return false
					}
				}

				return true
			}

			dm := &nnfv1alpha1.NnfDataMovement{
				ObjectMeta: metav1.ObjectMeta{
					Name:      w.Name,
					Namespace: w.Namespace,
				},
			}

			if state != dwsv1alpha1.StatePostRun {

				if findDataMovementDirectiveIndex() >= 0 {
					dm.ObjectMeta = metav1.ObjectMeta{
						Name:      fmt.Sprintf("%s-%d", w.Name, findDataMovementDirectiveIndex()),
						Namespace: w.Namespace,
					}
				} else {
					dm = nil
				}
			} else if !workflowExpectsDatamovementInPostRun() {
				dm = nil
			}

			if dm != nil {

				Eventually(func() error {
					return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(dm), dm)
				}).WithOffset(testStackOffset).Should(Succeed())

				Eventually(func() error {
					Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(dm), dm)).WithOffset(testStackOffset).Should(Succeed())

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
				}).WithOffset(testStackOffset).Should(Succeed())
			}
		}

		By(fmt.Sprintf("Waiting to go ready, wf '%s'", w.Name))
		Eventually(func() bool {
			Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(w), w)).WithOffset(testStackOffset).To(Succeed())
			return w.Status.Ready
		}).WithOffset(testStackOffset).Should(BeTrue(), fmt.Sprintf("Waiting on ready status state %s", state))

		if w.Status.State == dwsv1alpha1.StateSetup.String() {
			for dwIndex, directive := range w.Spec.DWDirectives {
				dwArgs, err := dwparse.BuildArgsMap(directive)
				Expect(err).WithOffset(testStackOffset).To(Succeed())
				if dwArgs["command"] != "jobdw" && dwArgs["command"] != "create_persistent" {
					continue
				}
				By("Verify that the NnfStorage now owns the pinned profile")
				commonName, commonNamespace := getStorageReferenceNameFromWorkflowActual(w, dwIndex)
				nnfStorage := &nnfv1alpha1.NnfStorage{}
				err = k8sClient.Get(context.TODO(), types.NamespacedName{Name: commonName, Namespace: commonNamespace}, nnfStorage)
				Expect(verifyPinnedProfile(context.TODO(), k8sClient, commonNamespace, commonName)).WithOffset(testStackOffset).To(Succeed())
			}
		}
	} // advanceStateAndCheckReady(state dwsv1alpha1.WorkflowState, w *dwsv1alpha1.Workflow)

	checkPSIToServerMapping := func(psiOwnedByWorkflow bool, storageName string, w *dwsv1alpha1.Workflow) {
		workFlowOwnerRef := metav1.OwnerReference{
			Kind:               reflect.TypeOf(dwsv1alpha1.Workflow{}).Name(),
			APIVersion:         dwsv1alpha1.GroupVersion.String(),
			UID:                w.GetUID(),
			Name:               w.GetName(),
			Controller:         &controller,
			BlockOwnerDeletion: &blockOwnerDeletion,
		}

		persistentInstance = &dwsv1alpha1.PersistentStorageInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name:      storageName,
				Namespace: w.Namespace,
			},
		}

		By(fmt.Sprintf("Retrieving PSI '%s'", storageName))
		Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(persistentInstance), persistentInstance)).To(Succeed(), "PersistentStorageInstance created")
		if psiOwnedByWorkflow {
			Expect(persistentInstance.ObjectMeta.OwnerReferences).To(ContainElement(workFlowOwnerRef), "PSI owned by workflow")
		} else {
			Expect(persistentInstance.ObjectMeta.OwnerReferences).ToNot(ContainElement(workFlowOwnerRef), "PSI NOT owned by workflow")
		}

		// Expect the persistentStorageInstance has owner reference to the servers resource;
		// this verifies the garbage collection chain is set up, but recall that GC is not
		// running in the testenv so we can't prove it is deleted on teardown.
		// See https://book.kubebuilder.io/reference/envtest.html#testing-considerations
		psiOwnerRef := metav1.OwnerReference{
			Kind:               reflect.TypeOf(dwsv1alpha1.PersistentStorageInstance{}).Name(),
			APIVersion:         dwsv1alpha1.GroupVersion.String(),
			UID:                persistentInstance.GetUID(),
			Name:               persistentInstance.GetName(),
			Controller:         &controller,
			BlockOwnerDeletion: &blockOwnerDeletion,
		}

		By("Checking DW Directive has Servers resource, named from the PSI")
		servers := &dwsv1alpha1.Servers{}
		Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(persistentInstance), servers)).To(Succeed())
		Expect(servers.ObjectMeta.OwnerReferences).To(ContainElement(psiOwnerRef), "Servers owned by PSI")

		By("Checking PersistentStorageInstance has reference to its Servers resource now that DirectiveBreakdown controller has finished")
		Expect(persistentInstance.Status.Servers.Kind).To(Equal(reflect.TypeOf(dwsv1alpha1.Servers{}).Name()))
		Expect(persistentInstance.Status.Servers.Name).To(Equal(persistentInstance.Name))
		Expect(persistentInstance.Status.Servers.Namespace).To(Equal(persistentInstance.Namespace))
	}

	checkServersToNnfStorageMapping := func(nnfStoragePresent bool) {
		servers := &dwsv1alpha1.Servers{}
		Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(persistentInstance), servers)).To(Succeed(), "Fetch Servers")

		persistentStorageOwnerRef := metav1.OwnerReference{
			Kind:               reflect.TypeOf(dwsv1alpha1.PersistentStorageInstance{}).Name(),
			APIVersion:         dwsv1alpha1.GroupVersion.String(),
			UID:                persistentInstance.GetUID(),
			Name:               persistentInstance.GetName(),
			Controller:         &controller,
			BlockOwnerDeletion: &blockOwnerDeletion,
		}

		nnfStorage := &nnfv1alpha1.NnfStorage{}
		if nnfStoragePresent {
			Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(persistentInstance), nnfStorage)).To(Succeed(), "Fetch NnfStorage matching PersistentStorageInstance")
			Expect(nnfStorage.ObjectMeta.OwnerReferences).To(ContainElement(persistentStorageOwnerRef), "NnfStorage owned by PersistentStorageInstance")
		} else {
			Eventually(func() error { // Delete can still return the cached object. Wait until the object is no longer present
				return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(servers), nnfStorage)
			}).ShouldNot(Succeed(), "NnfStorage should be deleted")
		}
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
		Expect(workflow).ToNot(BeNil(), "Did you comment out all wfTests below?")
		Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(workflow), workflow)).To(Succeed())
		Expect(k8sClient.Delete(context.TODO(), workflow)).To(Succeed())

		Eventually(func() error {
			return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(workflow), workflow)
		}).ShouldNot(Succeed())

		workflow = nil

		Expect(k8sClient.Delete(context.TODO(), storageProfile)).To(Succeed())
		profExpected := &nnfv1alpha1.NnfStorageProfile{}
		Eventually(func() error { // Delete can still return the cached object. Wait until the object is no longer present
			return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(storageProfile), profExpected)
		}).ShouldNot(Succeed())
	})

	It("Testing DWS directives", func() {
		type wfTestConfiguration struct {
			directive                   string
			expectedDirectiveBreakdowns int
			hasComputeBreakdown         bool
			hasStorageBreakdown         bool
			expectedAllocationSets      int
		}

		var wfTests = []wfTestConfiguration{
			{"#DW jobdw name=jobdw-raw    type=raw    capacity=1GiB", 1, true, true, 1},

			{"#DW jobdw name=jobdw-xfs    type=xfs    capacity=1GiB", 1, true, true, 1},
			{"#DW jobdw name=jobdw-gfs2   type=gfs2   capacity=1GiB", 1, true, true, 1},
			{"#DW jobdw name=jobdw-lustre type=lustre capacity=1GiB", 1, true, true, 3},
			{"#DW jobdw name=jobdw-lustre type=lustre combined_mgtmdt capacity=1GiB", 1, true, true, 2},
			{"#DW jobdw name=jobdw-lustre type=lustre external_mgs=localhost@tcp capacity=1GiB", 1, true, true, 2},

			{"#DW create_persistent name=createpersistent-xfs    type=xfs    capacity=1GiB", 1, false, true, 1},
			{"#DW create_persistent name=createpersistent-gfs2   type=gfs2   capacity=1GiB", 1, false, true, 1},
			{"#DW create_persistent name=createpersistent-lustre type=lustre capacity=1GiB", 1, false, true, 3},

			{"#DW persistentdw name=createpersistent-xfs", 1, true, false, 0},
			{"#DW persistentdw name=createpersistent-gfs2", 1, true, false, 0},
			{"#DW persistentdw name=createpersistent-lustre", 1, true, false, 0},

			{"#DW destroy_persistent name=doesnotexist", 0, false, false, 0},
			{"#DW destroy_persistent name=createpersistent-xfs   ", 0, false, false, 0},
			{"#DW destroy_persistent name=createpersistent-gfs2  ", 0, false, false, 0},
			{"#DW destroy_persistent name=createpersistent-lustre", 0, false, false, 0},
		}

		for idx := range wfTests {
			directive := wfTests[idx].directive

			By("Parsing the #DW")
			By(fmt.Sprintf("Directive %d: %s", idx, directive))
			dwArgs, err := dwparse.BuildArgsMap(directive)
			Expect(err).Should(BeNil())
			storageDirective, ok := dwArgs["command"]
			Expect(ok).To(BeTrue())

			// For persistentdw, we don't care what the fsType is, there are no allocations required
			fsType, ok := dwArgs["type"]
			if !ok {
				fsType = "unknown"
			}

			storageDirectiveName := strings.Replace(storageDirective, "_", "", 1) // Workflow names cannot include '_'
			storageName := fmt.Sprintf("%s-%s", storageDirectiveName, fsType)
			expectedAllocationSets := wfTests[idx].expectedAllocationSets
			wfid := uuid.NewString()[0:8]

			By(fmt.Sprintf("Testing directive '%s' filesystem '%s'", storageDirective, fsType))
			workflow = &dwsv1alpha1.Workflow{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("%s-%s-%s", storageDirectiveName, fsType, wfid),
					Namespace: corev1.NamespaceDefault,
				},
				Spec: dwsv1alpha1.WorkflowSpec{
					DesiredState: dwsv1alpha1.StateProposal.String(),
					JobID:        idx,
					WLMID:        "Test WLMID",
					DWDirectives: []string{
						directive,
					},
				},
			}

			By(fmt.Sprintf("Creating workflow '%s'", workflow.Name))
			Expect(k8sClient.Create(context.TODO(), workflow)).To(Succeed())

			By("Retrieving created workflow")
			Eventually(func() error {
				return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(workflow), workflow)
			}).Should(Succeed())

			// Store ownership reference to workflow - this is checked for many of the created objects
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
			Expect(workflow.Status.DirectiveBreakdowns).To(HaveLen(wfTests[idx].expectedDirectiveBreakdowns))
			for _, dbdRef := range workflow.Status.DirectiveBreakdowns {
				dbd := &dwsv1alpha1.DirectiveBreakdown{}
				Expect(dbdRef.Kind).To(Equal(reflect.TypeOf(dwsv1alpha1.DirectiveBreakdown{}).Name()))

				By("DW Directive Breakdown should go ready")
				Eventually(func() bool {
					Expect(k8sClient.Get(context.TODO(), types.NamespacedName{Name: dbdRef.Name, Namespace: dbdRef.Namespace}, dbd)).To(Succeed())
					return dbd.Status.Ready
				}).Should(BeTrue())

				By("Checking DW Directive Breakdown")
				Expect(k8sClient.Get(context.TODO(), types.NamespacedName{Name: dbdRef.Name, Namespace: dbdRef.Namespace}, dbd)).To(Succeed())
				Expect(dbd.ObjectMeta.OwnerReferences).To(ContainElement(ownerRef))
				if storageDirective == "jobdw" || storageDirective == "create_persistent" {
					By("Verify that pinned profiles have been created")
					pName, pNamespace := getStorageReferenceNameFromDBD(dbd)
					Expect(verifyPinnedProfile(context.TODO(), k8sClient, pNamespace, pName)).To(Succeed())
				}

				if wfTests[idx].hasComputeBreakdown {
					Expect(dbd.Status.Compute).NotTo(BeNil())
					Expect(dbd.Status.Compute.Constraints.Location).To(HaveLen(1))

					for _, location := range dbd.Status.Compute.Constraints.Location {
						servers := &dwsv1alpha1.Servers{}
						Expect(location.Reference.Kind).To(Equal(reflect.TypeOf(dwsv1alpha1.Servers{}).Name()))
						Expect(k8sClient.Get(context.TODO(), types.NamespacedName{Name: location.Reference.Name, Namespace: location.Reference.Namespace}, servers)).To(Succeed())
					}
				} else {
					Expect(dbd.Status.Compute).To(BeNil())
				}

				if !wfTests[idx].hasStorageBreakdown {
					Expect(dbd.Status.Storage).To(BeNil())
					continue
				}

				Expect(dbd.Status.Storage).NotTo(BeNil())
				Expect(dbd.Status.Storage.AllocationSets).To(HaveLen(expectedAllocationSets))

				By("DW Directive has Servers resource accessible from the DirectiveBreakdown")
				servers := &dwsv1alpha1.Servers{}
				Expect(dbd.Status.Storage.Reference.Kind).To(Equal(reflect.TypeOf(dwsv1alpha1.Servers{}).Name()))
				Expect(k8sClient.Get(context.TODO(), types.NamespacedName{Name: dbd.Status.Storage.Reference.Name, Namespace: dbd.Status.Storage.Reference.Namespace}, servers)).To(Succeed())

				By("DW Directive verifying servers resource")
				switch storageDirective {
				case "jobdw":
					Expect(servers.ObjectMeta.OwnerReferences).To(ContainElement(metav1.OwnerReference{
						Kind:               reflect.TypeOf(dwsv1alpha1.DirectiveBreakdown{}).Name(),
						APIVersion:         dwsv1alpha1.GroupVersion.String(),
						UID:                dbd.GetUID(),
						Name:               dbd.GetName(),
						Controller:         &controller,
						BlockOwnerDeletion: &blockOwnerDeletion,
					}))

				case "create_persistent":
					checkPSIToServerMapping(psiNotownedByWorkflow, storageName, workflow)
				}

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

					Expect(dbd.Status.Storage.AllocationSets).To(HaveLen(1))
					allocSet := &dbd.Status.Storage.AllocationSets[0]

					servers.Spec.AllocationSets = make([]dwsv1alpha1.ServersSpecAllocationSet, 1)
					servers.Spec.AllocationSets[0] = dwsv1alpha1.ServersSpecAllocationSet{
						AllocationSize: allocSet.MinimumCapacity,
						Label:          allocSet.Label,
						Storage:        storage,
					}

				} else {
					// If lustre, allocate one node per allocation set
					Expect(len(nodeNames) >= len(dbd.Status.Storage.AllocationSets)).To(BeTrue())
					servers.Spec.AllocationSets = make([]dwsv1alpha1.ServersSpecAllocationSet, len(dbd.Status.Storage.AllocationSets))
					for idx, allocset := range dbd.Status.Storage.AllocationSets {
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

			advanceStateAndCheckReady(dwsv1alpha1.StateSetup, workflow)

			By("Checking Setup state")
			switch storageDirective {
			case "jobdw":
			default:
				checkServersToNnfStorageMapping(nnfStoragePresent)
			}

			// TODO

			/**************************** Data In ****************************/

			advanceStateAndCheckReady(dwsv1alpha1.StateDataIn, workflow)

			By("Checking Data In state")
			// TODO

			/**************************** Pre Run ****************************/

			advanceStateAndCheckReady(dwsv1alpha1.StatePreRun, workflow)

			By("Checking Pre Run state")

			switch storageDirective {
			default:
				for _, dbdRef := range workflow.Status.DirectiveBreakdowns {
					dbd := &dwsv1alpha1.DirectiveBreakdown{}
					Expect(k8sClient.Get(context.TODO(), types.NamespacedName{Name: dbdRef.Name, Namespace: dbdRef.Namespace}, dbd)).To(Succeed())

					By("Check for an NNF Access describing the computes")
					access := &nnfv1alpha1.NnfAccess{
						ObjectMeta: metav1.ObjectMeta{
							Name:      fmt.Sprintf("%s-%s", dbd.Name, "computes"),
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

					// NnfStorage name is different for jobdw vs. persistentdw
					storageName := dbd.Name
					if storageDirective == "persistentdw" {
						storageName = dwArgs["name"]
					}
					By("Checking NNF Access storage reference exists")
					Expect(access.Spec.StorageReference).To(MatchFields(IgnoreExtras, Fields{
						"Name":      Equal(storageName),
						"Namespace": Equal(workflow.Namespace), // Namespace is the same as the workflow
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
						Expect(clientMount.Labels["dws.cray.hpe.com/workflow.name"]).To(Equal(workflow.Name))
						Expect(clientMount.Labels["dws.cray.hpe.com/workflow.namespace"]).To(Equal(workflow.Namespace))
						Expect(clientMount.Status.Mounts[0].Ready).To(BeTrue())
					}

					// For shared file systems, there should also be a NNF Access for the Rabbit as well as corresponding Client Mounts per Rabbit
					if fsType == "gfs2" {
						By("Checking for an NNF Access describing the servers")
						access := &nnfv1alpha1.NnfAccess{
							ObjectMeta: metav1.ObjectMeta{
								Name:      fmt.Sprintf("%s-%s", dbd.Name, "servers"),
								Namespace: workflow.Namespace,
							},
						}
						Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(access), access)).To(Succeed())
						Expect(access.ObjectMeta.OwnerReferences).To(ContainElement(ownerRef))

						// CONTINUE HERE WITH CHECKS FOR storage, status

						// CONTINUE HERE WITH CHECKS FOR Client Mounts on all the Rabbits
					}
				}
			case "create_persistent":
			}

			/*************************** Post Run ****************************/

			advanceStateAndCheckReady(dwsv1alpha1.StatePostRun, workflow)

			By("Checking Post Run state")

			switch storageDirective {
			default:
				for _, dbdRef := range workflow.Status.DirectiveBreakdowns {
					dbd := &dwsv1alpha1.DirectiveBreakdown{}
					Expect(k8sClient.Get(context.TODO(), types.NamespacedName{Name: dbdRef.Name, Namespace: dbdRef.Namespace}, dbd)).To(Succeed())

					By("Check that NNF Access describing computes is not present")
					access := &nnfv1alpha1.NnfAccess{
						ObjectMeta: metav1.ObjectMeta{
							Name:      fmt.Sprintf("%s-%s", dbd.Name, "computes"),
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
								Name:      fmt.Sprintf("%s-%s", dbd.Name, "servers"),
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
			case "create_persistent":
			}

			/**************************** Data Out ****************************/

			advanceStateAndCheckReady(dwsv1alpha1.StateDataOut, workflow)

			By("Checking Data Out state")
			// TODO

			/**************************** Teardown ****************************/

			advanceStateAndCheckReady(dwsv1alpha1.StateTeardown, workflow)

			By("Checking Teardown state")

			for _, dbdRef := range workflow.Status.DirectiveBreakdowns {
				dbd := &dwsv1alpha1.DirectiveBreakdown{}
				Expect(k8sClient.Get(context.TODO(), types.NamespacedName{Name: dbdRef.Name, Namespace: dbdRef.Namespace}, dbd)).To(Succeed())

				switch storageDirective {
				case "create_persistent":

					By("Check that the servers resource still exists")
					servers := &dwsv1alpha1.Servers{}
					Expect(dbd.Status.Storage.Reference.Kind).To(Equal(reflect.TypeOf(dwsv1alpha1.Servers{}).Name()))
					Expect(k8sClient.Get(context.TODO(), types.NamespacedName{Name: dbd.Status.Storage.Reference.Name, Namespace: dbd.Status.Storage.Reference.Namespace}, servers)).To(Succeed())

					By("NNFStorages for persistentStorageInstance should NOT be deleted")
					nnfStorage := &nnfv1alpha1.NnfStorage{}
					Consistently(func() error {
						return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(servers), nnfStorage)
					}).Should(Succeed(), "NnfStorage should continue to exist")

					By("PSI Not owned by workflow so it won't be deleted")
					checkPSIToServerMapping(psiNotownedByWorkflow, storageName, workflow)

				case "jobdw":
					By("Check that the servers resource still exists")
					servers := &dwsv1alpha1.Servers{}
					Expect(dbd.Status.Storage.Reference.Kind).To(Equal(reflect.TypeOf(dwsv1alpha1.Servers{}).Name()))
					Expect(k8sClient.Get(context.TODO(), types.NamespacedName{Name: dbd.Status.Storage.Reference.Name, Namespace: dbd.Status.Storage.Reference.Namespace}, servers)).To(Succeed())

					By("NNFStorages associated with jobdw should be deleted")
					nnfStorage := &nnfv1alpha1.NnfStorage{}
					Eventually(func() error { // Delete can still return the cached object. Wait until the object is no longer present
						return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(servers), nnfStorage)
					}).ShouldNot(Succeed(), "NnfStorage should be deleted")
				default:
				}
			}
		} // for idx := range wfTests
	}) // It(Testing DWS directives)

	Describe("Test workflow with no #DW directives", func() {
		It("Testing Lifecycle of workflow with no #DW directives", func() {
			const workflowName = "no-directives"

			workflow = &dwsv1alpha1.Workflow{
				ObjectMeta: metav1.ObjectMeta{
					Name:      workflowName,
					Namespace: corev1.NamespaceDefault,
				},
				Spec: dwsv1alpha1.WorkflowSpec{
					DesiredState: dwsv1alpha1.StateProposal.String(),
					WLMID:        "854973",
					DWDirectives: []string{}, // Empty
				},
			}
			Expect(k8sClient.Create(context.TODO(), workflow)).Should(Succeed())

			Eventually(func() error {
				return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(workflow), workflow)
			}).Should(Succeed())

			/*************************** Proposal ****************************/
			By("Verifying proposal state and ready")
			Eventually(func() bool {
				Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(workflow), workflow)).To(Succeed())
				return workflow.Status.State == dwsv1alpha1.StateProposal.String() && workflow.Status.Ready
			}).Should(BeTrue())

			By("Verifying it has no directiveBreakdowns")
			directiveBreakdown := &dwsv1alpha1.DirectiveBreakdown{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("%s-%d", workflowName, 0),
					Namespace: corev1.NamespaceDefault,
				},
			}
			Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(directiveBreakdown), directiveBreakdown)).ShouldNot(Succeed())

			By("Verifying it has no servers")
			servers := &dwsv1alpha1.Servers{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("%s-%d", workflowName, 0),
					Namespace: corev1.NamespaceDefault,
				},
			}
			Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(servers), servers)).ShouldNot(Succeed())

			// Store ownership reference to workflow - this is checked for many of the created objects
			ownerRef := metav1.OwnerReference{
				Kind:               reflect.TypeOf(dwsv1alpha1.Workflow{}).Name(),
				APIVersion:         dwsv1alpha1.GroupVersion.String(),
				UID:                workflow.GetUID(),
				Name:               workflow.GetName(),
				Controller:         &controller,
				BlockOwnerDeletion: &blockOwnerDeletion,
			}

			By("Verifying it has Computes resource")
			computes := &dwsv1alpha1.Computes{
				ObjectMeta: metav1.ObjectMeta{
					Name:      workflow.Status.Computes.Name,
					Namespace: workflow.Status.Computes.Namespace,
				},
			}
			Expect(workflow.Status.Computes.Kind).To(Equal(reflect.TypeOf(dwsv1alpha1.Computes{}).Name()))
			Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(computes), computes)).To(Succeed())
			Expect(computes.ObjectMeta.OwnerReferences).To(ContainElement(ownerRef))

			/***************************** Teardown *****************************/
			advanceStateAndCheckReady(dwsv1alpha1.StateTeardown, workflow)
		})
	})

	// Here we test the various data movement directives. This uses a workflow
	// per test where the workflow is brought to data_in stage. This is not
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
			}).Should(Succeed())

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

				Expect(dbd.Status.Storage.AllocationSets).To(HaveLen(1))
				allocSet := &dbd.Status.Storage.AllocationSets[0]

				By("Assigning storage")
				servers := &dwsv1alpha1.Servers{}
				Expect(k8sClient.Get(context.TODO(), types.NamespacedName{Name: dbd.Status.Storage.Reference.Name, Namespace: dbd.Status.Storage.Reference.Namespace}, servers)).To(Succeed())

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

			advanceStateAndCheckReady(dwsv1alpha1.StateSetup, workflow)

			/**************************** Data In ****************************/

			advanceStateAndCheckReady(dwsv1alpha1.StateDataIn, workflow)
		})

		AfterEach(func() {

			/**************************** Teardown ****************************/

			advanceStateAndCheckReady(dwsv1alpha1.StateTeardown, workflow)

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
					MgsNids:   []string{"172.0.0.1@tcp"},
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
				advanceStateAndCheckReady(dwsv1alpha1.StatePreRun, workflow)
				advanceStateAndCheckReady(dwsv1alpha1.StatePostRun, workflow)
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
				advanceStateAndCheckReady(dwsv1alpha1.StatePreRun, workflow)

				By("Validate NNF Access is created, with deletion in data-out")
				validateNnfAccessHasCorrectTeardownState(dwsv1alpha1.StateDataOut)

				By("Advancing to post run, ensure NNF Access is still set for deletion in data-out")
				advanceStateAndCheckReady(dwsv1alpha1.StatePostRun, workflow)
				validateNnfAccessHasCorrectTeardownState(dwsv1alpha1.StateDataOut)

				By("Advancing to data-out, ensure NNF Access is deleted")
				advanceStateAndCheckReady(dwsv1alpha1.StateDataOut, workflow)
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

				advanceStateAndCheckReady(dwsv1alpha1.StatePreRun, workflow)
				advanceStateAndCheckReady(dwsv1alpha1.StatePostRun, workflow)

				By("Advancing to data-out, ensure NNF Access is deleted")
				advanceStateAndCheckReady(dwsv1alpha1.StateDataOut, workflow)
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

				advanceStateAndCheckReady(dwsv1alpha1.StatePreRun, workflow)

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
				advanceState(dwsv1alpha1.StatePostRun, workflow, 1)

				By("Checking the driver has an error present")
				Eventually(func() *dwsv1alpha1.WorkflowDriverStatus {
					Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(workflow), workflow)).To(Succeed())

					driverID := os.Getenv("DWS_DRIVER_ID")
					for _, driver := range workflow.Status.Drivers {
						if driver.DriverID == driverID {
							if driver.Reason == "error" {
								return &driver
							}
						}
					}

					return nil
				}).ShouldNot(BeNil())
			})
		})

	})

	Describe("Test NnfStorageProfile Lustre profile merge with DW directives", func() {

		var (
			intendedDirective     string
			profileExternalMGS    *nnfv1alpha1.NnfStorageProfile
			profileCombinedMGTMDT *nnfv1alpha1.NnfStorageProfile

			directiveMgsNid string
			profileMgsNid   string

			dbd       *dwsv1alpha1.DirectiveBreakdown
			dbdServer *dwsv1alpha1.Servers

			externalMgsProfileName    string
			combinedMgtMdtProfileName string
		)

		BeforeEach(func() {
			directiveMgsNid = "directive-mgs@tcp"
			profileMgsNid = "profile-mgs@tcp"

			dbd = &dwsv1alpha1.DirectiveBreakdown{}
			dbdServer = &dwsv1alpha1.Servers{}

			externalMgsProfileName = "has-external-mgs"
			combinedMgtMdtProfileName = "has-combined-mgtmdt"
		})

		// Create some custom storage profiles.
		BeforeEach(func() {
			By("BeforeEach create some custom storage profiles")
			profileExternalMGS = basicNnfStorageProfile(externalMgsProfileName)
			profileExternalMGS.Data.LustreStorage.ExternalMGS = []string{
				profileMgsNid,
			}
			profileCombinedMGTMDT = basicNnfStorageProfile(combinedMgtMdtProfileName)
			profileCombinedMGTMDT.Data.LustreStorage.CombinedMGTMDT = true
			Expect(createNnfStorageProfile(profileExternalMGS, true)).ToNot(BeNil())
			Expect(createNnfStorageProfile(profileCombinedMGTMDT, true)).ToNot(BeNil())
		})

		// Destroy the custom storage profiles.
		AfterEach(func() {
			By("AfterEach destroy the custom storage profiles")
			Expect(k8sClient.Delete(context.TODO(), profileExternalMGS)).To(Succeed())
			Expect(k8sClient.Delete(context.TODO(), profileCombinedMGTMDT)).To(Succeed())

		})

		// Setup a basic workflow; each test is expected to fill in the
		// DWDirectives in its BeforeEach() clause.
		BeforeEach(func() {
			By("BeforeEach setup a basic workflow resource")
			wfid := uuid.NewString()[0:8]
			workflow = &dwsv1alpha1.Workflow{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("profile-%s", wfid),
					Namespace: corev1.NamespaceDefault,
				},
				Spec: dwsv1alpha1.WorkflowSpec{
					DesiredState: dwsv1alpha1.StateProposal.String(),
					JobID:        0,
					WLMID:        "Test WLMID",
				},
			}
		})

		AfterEach(func() {
			By("AfterEach advance the workflow state to teardown")
			advanceStateAndCheckReady(dwsv1alpha1.StateTeardown, workflow)
		})

		// Create the workflow for the Ginkgo specs.
		JustBeforeEach(func() {
			By("JustBeforeEach create the workflow")
			workflow.Spec.DWDirectives = []string{
				intendedDirective,
			}
			Expect(k8sClient.Create(context.TODO(), workflow)).To(Succeed())
			Eventually(func(g Gomega) error {
				g.Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(workflow), workflow)).To(Succeed())
				if (workflow.Status.Ready == true) && (workflow.Status.State == dwsv1alpha1.StateProposal.String()) {
					return nil
				}
				return fmt.Errorf("ready state not achieved")
			}).Should(Succeed(), "achieved ready state")

			By("Verify that one DirectiveBreakdown was created")
			Expect(workflow.Status.DirectiveBreakdowns).To(HaveLen(1))
			By("Get the DirectiveBreakdown resource")
			dbdRef := workflow.Status.DirectiveBreakdowns[0]
			Expect(k8sClient.Get(context.TODO(), types.NamespacedName{Name: dbdRef.Name, Namespace: dbdRef.Namespace}, dbd)).To(Succeed())

			pName, pNamespace := getStorageReferenceNameFromDBD(dbd)
			Expect(verifyPinnedProfile(context.TODO(), k8sClient, pNamespace, pName)).To(Succeed())

			By("Get the Servers resource")
			Expect(k8sClient.Get(context.TODO(), types.NamespacedName{Name: dbd.Status.Storage.Reference.Name, Namespace: dbd.Status.Storage.Reference.Namespace}, dbdServer)).To(Succeed())
		})

		assignStorageForMDTOST := func() {
			dbdServer.Spec.AllocationSets = []dwsv1alpha1.ServersSpecAllocationSet{
				{
					AllocationSize: 1,
					Label:          "mdt",
					Storage: []dwsv1alpha1.ServersSpecStorage{
						{AllocationCount: 1,
							Name: "rabbit-test-node-0"},
					},
				},
				{
					AllocationSize: 1,
					Label:          "ost",
					Storage: []dwsv1alpha1.ServersSpecStorage{
						{AllocationCount: 1,
							Name: "rabbit-test-node-1"},
					},
				},
			}
		}

		// verifyExternalMgsNid checks that an external MGS NID is used.
		verifyExternalMgsNid := func(getNidVia, desiredNid string) {
			Expect(dbd.Spec.Directive).To(Equal(intendedDirective))

			By("Verify that it does not allocate an MGT device")
			Expect(dbd.Status.Storage.AllocationSets).To(HaveLen(2))
			for _, comp := range dbd.Status.Storage.AllocationSets {
				Expect(comp.Label).To(Or(Equal("mdt"), Equal("ost")))
			}

			// Go to 'setup' state and verify the MGS NID is
			// propagated to the NnfStorage resource.
			By("Assign storage")
			assignStorageForMDTOST()
			Expect(k8sClient.Update(context.TODO(), dbdServer)).To(Succeed())
			By(fmt.Sprintf("Verify that the MGS NID %s is used by the filesystem", getNidVia))
			advanceStateAndCheckReady(dwsv1alpha1.StateSetup, workflow)
			// The NnfStorage's name matches the Server resource's name.
			nnfstorage := &nnfv1alpha1.NnfStorage{}
			Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(dbdServer), nnfstorage)).To(Succeed())
			for _, comp := range nnfstorage.Spec.AllocationSets {
				Expect(comp.ExternalMgsNid).To(Equal(desiredNid))
			}
		}

		// verifyCombinedMgtMdt checks that a single device is used for the MGT and MDT.
		verifyCombinedMgtMdt := func() {
			Expect(dbd.Spec.Directive).To(Equal(intendedDirective))

			By("Verify that it allocates an MGTMDT device rather than an MGT device")
			Expect(dbd.Status.Storage.AllocationSets).To(HaveLen(2))
			for _, comp := range dbd.Status.Storage.AllocationSets {
				Expect(comp.Label).To(Or(Equal("mgtmdt"), Equal("ost")))
			}
		}

		When("using external_mgs in directive", func() {
			BeforeEach(func() {
				intendedDirective = fmt.Sprintf("#DW jobdw type=lustre external_mgs=%s capacity=5GB name=directive-mgs", directiveMgsNid)
			})

			It("Uses external_mgs via the directive", func() {
				verifyExternalMgsNid("via directive", directiveMgsNid)
			})
		})

		When("using external_mgs in profile", func() {
			BeforeEach(func() {
				intendedDirective = fmt.Sprintf("#DW jobdw type=lustre profile=%s capacity=5GB name=profile-mgs", externalMgsProfileName)
			})

			It("Uses external_mgs via the profile", func() {
				verifyExternalMgsNid("via profile", profileMgsNid)
			})
		})

		When("using external_mgs in directive and profile", func() {
			BeforeEach(func() {
				intendedDirective = fmt.Sprintf("#DW jobdw type=lustre profile=%s external_mgs=%s capacity=5GB name=both-mgs", externalMgsProfileName, directiveMgsNid)
			})

			It("Uses external_mgs via the directive", func() {
				verifyExternalMgsNid("via directive", directiveMgsNid)
			})
		})

		When("using combined_mgtmdt in directive", func() {
			BeforeEach(func() {
				intendedDirective = fmt.Sprintf("#DW jobdw type=lustre combined_mgtmdt capacity=5GB name=directive-mgtmdt")
			})

			It("Uses combined_mgtmdt via the directive", func() {
				verifyCombinedMgtMdt()
			})
		})

		When("using combined_mgtmdt in profile", func() {
			BeforeEach(func() {
				intendedDirective = fmt.Sprintf("#DW jobdw type=lustre profile=%s capacity=5GB name=profile-mgtmdt", combinedMgtMdtProfileName)
			})

			It("Uses combined_mgtmdt via the profile", func() {
				verifyCombinedMgtMdt()
			})
		})

		When("using combined_mgtmdt from directive when external_mgs is in profile", func() {
			BeforeEach(func() {
				intendedDirective = fmt.Sprintf("#DW jobdw type=lustre profile=%s combined_mgtmdt capacity=5GB name=profile-mgtmdt", externalMgsProfileName)
			})

			It("Uses combined_mgtmdt via the directive", func() {
				verifyCombinedMgtMdt()
			})
		})

		When("using external_mgs from directive when combined_mgtmdt is in profile", func() {
			BeforeEach(func() {
				intendedDirective = fmt.Sprintf("#DW jobdw type=lustre profile=%s external_mgs=%s capacity=5GB name=profile-mgtmdt", combinedMgtMdtProfileName, directiveMgsNid)
			})

			It("Uses external_mgs via the directive", func() {
				verifyExternalMgsNid("via directive", directiveMgsNid)
			})
		})
	})

	Describe("Test failure cases for various directives", func() {

		var (
			directives []string
		)

		// Create a basic workflow; each test is expected to fill in the DWDirectives
		// in its BeforeEach() clause.
		BeforeEach(func() {
			wfid := uuid.NewString()[0:8]
			workflow = &dwsv1alpha1.Workflow{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("persistent-%s", wfid),
					Namespace: corev1.NamespaceDefault,
				},
				Spec: dwsv1alpha1.WorkflowSpec{
					DesiredState: dwsv1alpha1.StateProposal.String(),
					JobID:        -1,
					WLMID:        "Test WLMID",
				},
			}
		})

		AfterEach(func() {
			By("AfterEach advance the workflow state to teardown")
			advanceStateAndCheckReady(dwsv1alpha1.StateTeardown, workflow)
		})

		// Create the workflow for the Ginkgo specs.
		JustBeforeEach(func() {
			By("JustBeforeEach create the workflow")
			workflow.Spec.DWDirectives = directives
			Expect(k8sClient.Create(context.TODO(), workflow)).To(Succeed())
		})

		// verifyErrorIsPresent checks that the workflow has stopped due to a driver error
		verifyErrorIsPresent := func() {
			By("Checking the driver has an error present")
			Eventually(func(g Gomega) *dwsv1alpha1.WorkflowDriverStatus {
				g.Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(workflow), workflow)).To(Succeed())
				g.Expect(workflow.Status.Ready == false)
				g.Expect(workflow.Status.State == dwsv1alpha1.StateProposal.String())

				driverID := os.Getenv("DWS_DRIVER_ID")
				for _, driver := range workflow.Status.Drivers {
					if driver.DriverID == driverID {
						if driver.Reason == "error" {
							return &driver
						}
					}
				}

				return nil
			}).ShouldNot(BeNil())
		}

		// TODO: Make this a table driven set of tests to easily allow more combinations
		When("using jobdw and create_persistent", func() {
			BeforeEach(func() {
				directives = []string{
					"#DW jobdw name=jobdw type=xfs capacity=1GiB",
					"#DW create_persistent name=p2 type=xfs capacity=1GiB",
				}
			})

			It("using jobdw and create_persistent", func() {
				verifyErrorIsPresent()
			})
		})

		When("using create_persistent with persistentdw", func() {
			BeforeEach(func() {
				directives = []string{
					"#DW persistentdw name=jobdw",
					"#DW create_persistent name=p2 type=xfs capacity=1GiB",
				}
			})

			It("using persistentdw and create_persistent", func() {
				verifyErrorIsPresent()
			})
		})

		When("using more than one create_persistent directive in workflow", func() {
			BeforeEach(func() {
				directives = []string{
					"#DW create_persistent name=p1 type=xfs capacity=1GiB",
					"#DW create_persistent name=p2 type=xfs capacity=1GiB",
				}
			})

			It("Fails with more than 1 create_persistent", func() {
				verifyErrorIsPresent()
			})
		})

		When("using more than one destroy_persistent directive in workflow", func() {
			BeforeEach(func() {
				directives = []string{
					"#DW destroy_persistent name=p1",
					"#DW destroy_persistent name=p2",
				}
			})

			It("Fails with more than 1 destroy_persistent", func() {
				verifyErrorIsPresent()
			})
		})

		When("using 1 create_persistent and 1 destroy_persistent directive in workflow", func() {
			BeforeEach(func() {
				directives = []string{
					"#DW create_persistent name=p1 type=xfs capacity=1GiB",
					"#DW destroy_persistent name=p1",
				}
			})

			It("Fails with 1 create_persistent and 1 destroy_persistent", func() {
				verifyErrorIsPresent()
			})
		})

		When("using persistentdw for non-existent persistent storage", func() {
			BeforeEach(func() {
				directives = []string{
					"#DW persistentdw name=p1",
				}
			})

			It("Fails with persistentdw naming non-existent persistent storage", func() {
				verifyErrorIsPresent()
			})
		})

		When("using copy_in without jobdw", func() {
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
			BeforeEach(func() {
				directives = []string{
					"#DW copy_in source=/lus/maui/my-file.in destination=$JOB_DW_notThere/my-file.out",
				}
			})

			AfterEach(func() {
				Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(lustre), lustre)).To(Succeed())
				Expect(k8sClient.Delete(context.TODO(), lustre)).To(Succeed())
			})

			It("Fails with copy_in and no jobdw", func() {
				verifyErrorIsPresent()
			})
		})

		When("using copy_in without persistentdw", func() {
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

			BeforeEach(func() {
				directives = []string{
					"#DW copy_in source=/lus/maui/my-file.in destination=$PERSISTENT_DW_mpi/my-persistent-file.out",
				}
			})

			AfterEach(func() {
				Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(lustre), lustre)).To(Succeed())
				Expect(k8sClient.Delete(context.TODO(), lustre)).To(Succeed())
			})

			It("Fails with copy_in and no persistentdw", func() {
				verifyErrorIsPresent()
			})
		})

	})

})
