/*
 * Copyright 2021-2024 Hewlett Packard Enterprise Development LP
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

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	dwsv1alpha2 "github.com/DataWorkflowServices/dws/api/v1alpha2"
	dwparse "github.com/DataWorkflowServices/dws/utils/dwdparse"
	lusv1beta1 "github.com/NearNodeFlash/lustre-fs-operator/api/v1beta1"
	"github.com/NearNodeFlash/nnf-sos/api/v1alpha1"
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
		workflow           *dwsv1alpha2.Workflow
		persistentInstance *dwsv1alpha2.PersistentStorageInstance
		nodeNames          []string
		setup              sync.Once
		storageProfile     *nnfv1alpha1.NnfStorageProfile
		dmm                *nnfv1alpha1.NnfDataMovementManager
	)

	advanceState := func(state dwsv1alpha2.WorkflowState, w *dwsv1alpha2.Workflow, testStackOffset int) {
		By(fmt.Sprintf("Advancing to %s state, wf %s", state, w.Name))
		Eventually(func() error {
			Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(w), w)).WithOffset(testStackOffset).To(Succeed())
			w.Spec.DesiredState = state
			return k8sClient.Update(context.TODO(), w)
		}).WithOffset(testStackOffset).Should(Succeed(), fmt.Sprintf("Advancing to %s state", state))

		By(fmt.Sprintf("Waiting on state %s", state))
		Eventually(func() dwsv1alpha2.WorkflowState {
			Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(w), w)).WithOffset(testStackOffset).To(Succeed())
			return w.Status.State
		}).WithOffset(testStackOffset).Should(Equal(state), fmt.Sprintf("Waiting on state %s", state))
	}

	verifyNnfNodeStoragesHaveStorageProfileLabel := func(nnfStorage *nnfv1alpha1.NnfStorage) {
		for allocationSetIndex := range nnfStorage.Spec.AllocationSets {
			allocationSet := nnfStorage.Spec.AllocationSets[allocationSetIndex]
			for i, node := range allocationSet.Nodes {
				// Per Rabbit namespace.
				nnfNodeStorage := &nnfv1alpha1.NnfNodeStorage{
					ObjectMeta: metav1.ObjectMeta{
						Name:      nnfNodeStorageName(nnfStorage, allocationSetIndex, i),
						Namespace: node.Name,
					},
				}

				Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(nnfNodeStorage), nnfNodeStorage)).To(Succeed())
				By("Verify that the NnfNodeStorage has a label for the pinned profile")
				_, err := getPinnedStorageProfileFromLabel(context.TODO(), k8sClient, nnfNodeStorage)
				Expect(err).ShouldNot(HaveOccurred())
			}

		}
	}

	advanceStateAndCheckReady := func(state dwsv1alpha2.WorkflowState, w *dwsv1alpha2.Workflow) {
		By(fmt.Sprintf("advanceStateAndCheckReady: advance workflow state %s", state))

		// If this method fails, have the test results report where it was called from rather
		// than where it fails in this method.
		// If you'd rather see why this method is failing, set this to 0.
		// If you'd rather see why advanceState() is failing, set this to -1.
		testStackOffset := 1

		// If advanceState fails, we are more interested in what the outer test method
		// was doing as it tried to advanceState versus, why advanceState fails.
		advanceState(state, w, testStackOffset+1)

		By(fmt.Sprintf("Waiting to go ready, wf '%s'", w.Name))
		Eventually(func(g Gomega) bool {

			// If we're currently in a staging state, ensure the data movement status is marked as finished so
			// we can successfully transition out of that state.
			if w.Status.State == dwsv1alpha2.StateDataIn || w.Status.State == dwsv1alpha2.StateDataOut {

				findDataMovementDirectiveIndex := func() int {
					for idx, directive := range w.Spec.DWDirectives {
						if state == dwsv1alpha2.StateDataIn && strings.HasPrefix(directive, "#DW copy_in") {
							return idx
						}
						if state == dwsv1alpha2.StateDataOut && strings.HasPrefix(directive, "#DW copy_out") {
							return idx
						}
					}

					return -1
				}

				if findDataMovementDirectiveIndex() >= 0 {

					dms := &nnfv1alpha1.NnfDataMovementList{}
					Expect(k8sClient.List(context.TODO(), dms)).To(Succeed())

					for _, dm := range dms.Items {
						dm := dm
						g.Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(&dm), &dm)).To(Succeed())
						dm.Status.State = nnfv1alpha1.DataMovementConditionTypeFinished
						dm.Status.Status = nnfv1alpha1.DataMovementConditionReasonSuccess
						g.Expect(k8sClient.Status().Update(context.TODO(), &dm)).To(Succeed())
					}
				}
			}

			g.Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(w), w)).WithOffset(testStackOffset).To(Succeed())
			return w.Status.Ready
		}).WithOffset(testStackOffset).Should(BeTrue(), fmt.Sprintf("Waiting on ready status state %s", state))

		if w.Status.State == dwsv1alpha2.StateSetup {
			for dwIndex, directive := range w.Spec.DWDirectives {
				dwArgs, err := dwparse.BuildArgsMap(directive)
				Expect(err).WithOffset(testStackOffset).To(Succeed())
				if dwArgs["command"] != "jobdw" && dwArgs["command"] != "create_persistent" {
					continue
				}
				By("Verify that the NnfStorage now owns the pinned profile")
				commonName, commonNamespace := getStorageReferenceNameFromWorkflowActual(w, dwIndex)
				nnfStorage := &nnfv1alpha1.NnfStorage{}
				Expect(k8sClient.Get(context.TODO(), types.NamespacedName{Name: commonName, Namespace: commonNamespace}, nnfStorage)).To(Succeed())
				Expect(verifyPinnedProfile(context.TODO(), k8sClient, commonNamespace, commonName)).WithOffset(testStackOffset).To(Succeed())

				By("Verify that the NnfStorage has a label for the pinned profile")
				_, err = getPinnedStorageProfileFromLabel(context.TODO(), k8sClient, nnfStorage)
				Expect(err).ShouldNot(HaveOccurred())

				By("Verify that the NnfNodeStorages have a label for the pinned profile")
				verifyNnfNodeStoragesHaveStorageProfileLabel(nnfStorage)
			}
		}
	} // advanceStateAndCheckReady(state dwsv1alpha2.WorkflowState, w *dwsv1alpha2.Workflow)

	checkPSIConsumerReference := func(storageName string, w *dwsv1alpha2.Workflow) {
		persistentInstance = &dwsv1alpha2.PersistentStorageInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name:      storageName,
				Namespace: w.Namespace,
			},
		}

		By(fmt.Sprintf("Retrieving PSI '%s'", storageName))
		Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(persistentInstance), persistentInstance)).To(Succeed(), "PersistentStorageInstance created")

		By("Checking PSI has the correct consumer reference")
		Expect(persistentInstance.Spec.ConsumerReferences).To(HaveLen(1))
		Expect(persistentInstance.Spec.ConsumerReferences[0].Name).To(Equal(indexedResourceName(w, 0)))
		Expect(persistentInstance.Spec.ConsumerReferences[0].Namespace).To(Equal(w.Namespace))
	}

	checkPSIToServerMapping := func(psiOwnedByWorkflow bool, storageName string, w *dwsv1alpha2.Workflow) {
		workFlowOwnerRef := metav1.OwnerReference{
			Kind:               reflect.TypeOf(dwsv1alpha2.Workflow{}).Name(),
			APIVersion:         dwsv1alpha2.GroupVersion.String(),
			UID:                w.GetUID(),
			Name:               w.GetName(),
			Controller:         &controller,
			BlockOwnerDeletion: &blockOwnerDeletion,
		}

		persistentInstance = &dwsv1alpha2.PersistentStorageInstance{
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
			Kind:               reflect.TypeOf(dwsv1alpha2.PersistentStorageInstance{}).Name(),
			APIVersion:         dwsv1alpha2.GroupVersion.String(),
			UID:                persistentInstance.GetUID(),
			Name:               persistentInstance.GetName(),
			Controller:         &controller,
			BlockOwnerDeletion: &blockOwnerDeletion,
		}

		By("Checking DW Directive has Servers resource, named from the PSI")
		servers := &dwsv1alpha2.Servers{}
		Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(persistentInstance), servers)).To(Succeed())
		Expect(servers.ObjectMeta.OwnerReferences).To(ContainElement(psiOwnerRef), "Servers owned by PSI")

		By("Checking PersistentStorageInstance has reference to its Servers resource now that DirectiveBreakdown controller has finished")
		Expect(persistentInstance.Status.Servers.Kind).To(Equal(reflect.TypeOf(dwsv1alpha2.Servers{}).Name()))
		Expect(persistentInstance.Status.Servers.Name).To(Equal(persistentInstance.Name))
		Expect(persistentInstance.Status.Servers.Namespace).To(Equal(persistentInstance.Namespace))
	}

	checkServersToNnfStorageMapping := func(nnfStoragePresent bool) {
		servers := &dwsv1alpha2.Servers{}
		Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(persistentInstance), servers)).To(Succeed(), "Fetch Servers")

		persistentStorageOwnerRef := metav1.OwnerReference{
			Kind:               reflect.TypeOf(dwsv1alpha2.PersistentStorageInstance{}).Name(),
			APIVersion:         dwsv1alpha2.GroupVersion.String(),
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

		// Initialize node names - currently set to three to satisify the lustre requirement of single MDT, MGT, OST
		// NOTE: Node names require the "rabbit" prefix to ensure client mounts occur on the correct controller
		nodeNames = []string{
			"rabbit-test-node-0",
			"rabbit-test-node-1",
			"rabbit-test-node-2",
		}

		setup.Do(func() {
			for _, nodeName := range nodeNames {
				// Create the namespace
				ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
					Name: nodeName,
				}}

				Expect(k8sClient.Create(context.TODO(), ns)).To(Succeed())
			}
		}) // once

		// Build the config map that ties everything together; this also
		// creates a namespace for each compute node which is required for
		// client mount.
		computeNameGeneratorFunc := func() func() []string {
			nextComputeIndex := 0
			return func() []string {
				computes := make([]string, 16)
				for i := 0; i < 16; i++ {
					name := fmt.Sprintf("compute%d", i+nextComputeIndex)
					computes[i] = name
				}
				nextComputeIndex += 16
				return computes
			}
		}

		generator := computeNameGeneratorFunc()
		configSpec := dwsv1alpha2.SystemConfigurationSpec{}
		for _, nodeName := range nodeNames {
			storageNode := dwsv1alpha2.SystemConfigurationStorageNode{
				Type: "Rabbit",
				Name: nodeName,
			}

			computeNames := generator()
			storageNode.ComputesAccess = make([]dwsv1alpha2.SystemConfigurationComputeNodeReference, 0)
			for idx, name := range computeNames {
				computesAccess := dwsv1alpha2.SystemConfigurationComputeNodeReference{
					Name:  name,
					Index: idx,
				}
				storageNode.ComputesAccess = append(storageNode.ComputesAccess, computesAccess)
			}
			configSpec.StorageNodes = append(configSpec.StorageNodes, storageNode)
		}

		config := &dwsv1alpha2.SystemConfiguration{
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
			// Create the node - set it to up as ready
			node := &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: nodeName,
					Labels: map[string]string{
						v1alpha1.RabbitNodeSelectorLabel: "true",
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

			// Create the NNF Node resource
			nnfNode := &nnfv1alpha1.NnfNode{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "nnf-nlc",
					Namespace: nodeName,
				},
				Spec: nnfv1alpha1.NnfNodeSpec{
					Name:  nodeName,
					State: nnfv1alpha1.ResourceEnable,
				},
				Status: nnfv1alpha1.NnfNodeStatus{},
			}

			Expect(k8sClient.Create(context.TODO(), nnfNode)).To(Succeed())

			// Create the DWS Storage resource
			storage := &dwsv1alpha2.Storage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      nodeName,
					Namespace: corev1.NamespaceDefault,
				},
			}

			Expect(k8sClient.Create(context.TODO(), storage)).To(Succeed())

			// Check that the DWS storage resource was updated with the compute node information

			Eventually(func() error {
				return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(storage), storage)
			}).Should(Succeed())

			Eventually(func() bool {
				Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(storage), storage)).To(Succeed())
				return len(storage.Status.Access.Computes) == 16
			}).Should(BeTrue())

			// Check that a namespace was created for each compute node
			for i := 0; i < len(nodeNames)*16; i++ {
				namespace := &corev1.Namespace{}
				Eventually(func() error {
					return k8sClient.Get(context.TODO(), types.NamespacedName{Name: fmt.Sprintf("compute%d", i)}, namespace)
				}).Should(Succeed())
			}
		}

		// Create a default NnfStorageProfile for the unit tests.
		storageProfile = createBasicDefaultNnfStorageProfile()

		DeferCleanup(os.Setenv, "RABBIT_TEST_ENV_BYPASS_SERVER_STORAGE_CHECK", os.Getenv("RABBIT_TEST_ENV_BYPASS_SERVER_STORAGE_CHECK"))
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

		for _, nodeName := range nodeNames {
			storage := &dwsv1alpha2.Storage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      nodeName,
					Namespace: corev1.NamespaceDefault,
				},
			}
			Expect(k8sClient.Delete(context.TODO(), storage)).To(Succeed())
			tempStorage := &dwsv1alpha2.Storage{}
			Eventually(func() error { // Delete can still return the cached object. Wait until the object is no longer present
				return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(storage), tempStorage)
			}).ShouldNot(Succeed())

			nnfNode := &nnfv1alpha1.NnfNode{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "nnf-nlc",
					Namespace: nodeName,
				},
			}
			Expect(k8sClient.Delete(context.TODO(), nnfNode)).To(Succeed())
			tempNnfNode := &nnfv1alpha1.NnfNode{}
			Eventually(func() error { // Delete can still return the cached object. Wait until the object is no longer present
				return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(nnfNode), tempNnfNode)
			}).ShouldNot(Succeed())

			node := &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name:      nodeName,
					Namespace: corev1.NamespaceDefault,
				},
			}
			Expect(k8sClient.Delete(context.TODO(), node)).To(Succeed())
			tempNode := &corev1.Node{}
			Eventually(func() error { // Delete can still return the cached object. Wait until the object is no longer present
				return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(node), tempNode)
			}).ShouldNot(Succeed())

		}

		config := &dwsv1alpha2.SystemConfiguration{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "default",
				Namespace: corev1.NamespaceDefault,
			},
		}

		Expect(k8sClient.Delete(context.TODO(), config)).To(Succeed())
		tempConfig := &dwsv1alpha2.SystemConfiguration{}
		Eventually(func() error { // Delete can still return the cached object. Wait until the object is no longer present
			return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(config), tempConfig)
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
			{"#DW jobdw name=jobdw-xfs2   type=xfs    capacity=1GiB requires=copy-offload", 1, true, true, 1},
			{"#DW jobdw name=jobdw-gfs2   type=gfs2   capacity=1GiB", 1, true, true, 1},
			{"#DW jobdw name=jobdw-lustre type=lustre capacity=1GiB", 1, true, true, 3},

			{"#DW create_persistent name=createpersistent-xfs    type=xfs    capacity=1GiB", 1, false, true, 1},
			{"#DW create_persistent name=createpersistent-xfs2   type=xfs    capacity=1GiB", 1, false, true, 1},
			{"#DW create_persistent name=createpersistent-gfs2   type=gfs2   capacity=1GiB", 1, false, true, 1},
			{"#DW create_persistent name=createpersistent-lustre type=lustre capacity=1GiB", 1, false, true, 3},

			{"#DW persistentdw name=createpersistent-xfs", 1, true, false, 0},
			{"#DW persistentdw name=createpersistent-xfs2 requires=copy-offload", 1, true, false, 0},
			{"#DW persistentdw name=createpersistent-gfs2", 1, true, false, 0},
			{"#DW persistentdw name=createpersistent-lustre", 1, true, false, 0},

			{"#DW destroy_persistent name=doesnotexist", 0, false, false, 0},
			{"#DW destroy_persistent name=createpersistent-xfs   ", 0, false, false, 0},
			{"#DW destroy_persistent name=createpersistent-xfs2  ", 0, false, false, 0},
			{"#DW destroy_persistent name=createpersistent-gfs2  ", 0, false, false, 0},
			{"#DW destroy_persistent name=createpersistent-lustre", 0, false, false, 0},
		}

		err := os.Unsetenv("RABBIT_TEST_ENV_BYPASS_SERVER_STORAGE_CHECK")
		Expect(err).NotTo(HaveOccurred())

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
			workflow = &dwsv1alpha2.Workflow{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("%s-%s-%s", storageDirectiveName, fsType, wfid),
					Namespace: corev1.NamespaceDefault,
				},
				Spec: dwsv1alpha2.WorkflowSpec{
					DesiredState: dwsv1alpha2.StateProposal,
					JobID:        intstr.FromInt(idx),
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
				Kind:               reflect.TypeOf(dwsv1alpha2.Workflow{}).Name(),
				APIVersion:         dwsv1alpha2.GroupVersion.String(),
				UID:                workflow.GetUID(),
				Name:               workflow.GetName(),
				Controller:         &controller,
				BlockOwnerDeletion: &blockOwnerDeletion,
			}

			/*************************** Proposal ****************************/

			By("Checking proposal state and ready")
			Eventually(func() bool {
				Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(workflow), workflow)).To(Succeed())
				return workflow.Status.State == dwsv1alpha2.StateProposal && workflow.Status.Ready
			}).Should(BeTrue())

			By("Checking for Computes resource")
			computes := &dwsv1alpha2.Computes{
				ObjectMeta: metav1.ObjectMeta{
					Name:      workflow.Status.Computes.Name,
					Namespace: workflow.Status.Computes.Namespace,
				},
			}
			Expect(workflow.Status.Computes.Kind).To(Equal(reflect.TypeOf(dwsv1alpha2.Computes{}).Name()))
			Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(computes), computes)).To(Succeed())
			Expect(computes.ObjectMeta.OwnerReferences).To(ContainElement(ownerRef))

			By("Checking various DW Directive Breakdowns")
			Expect(workflow.Status.DirectiveBreakdowns).To(HaveLen(wfTests[idx].expectedDirectiveBreakdowns))
			for _, dbdRef := range workflow.Status.DirectiveBreakdowns {
				dbd := &dwsv1alpha2.DirectiveBreakdown{}
				Expect(dbdRef.Kind).To(Equal(reflect.TypeOf(dwsv1alpha2.DirectiveBreakdown{}).Name()))

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

				if requiresList, requiresDaemons := dwArgs["requires"]; requiresDaemons {
					requires := strings.Split(requiresList, ",")
					Expect(dbd.Status.RequiredDaemons).Should(ConsistOf(requires), directive)
				} else {
					Expect(dbd.Status.RequiredDaemons).Should(BeEmpty(), directive)
				}

				if wfTests[idx].hasComputeBreakdown {
					Expect(dbd.Status.Compute).NotTo(BeNil())
					Expect(dbd.Status.Compute.Constraints.Location).ToNot(BeEmpty())

					for _, location := range dbd.Status.Compute.Constraints.Location {
						servers := &dwsv1alpha2.Servers{}
						Expect(location.Reference.Kind).To(Equal(reflect.TypeOf(dwsv1alpha2.Servers{}).Name()))
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
				servers := &dwsv1alpha2.Servers{}
				Expect(dbd.Status.Storage.Reference.Kind).To(Equal(reflect.TypeOf(dwsv1alpha2.Servers{}).Name()))
				Expect(k8sClient.Get(context.TODO(), types.NamespacedName{Name: dbd.Status.Storage.Reference.Name, Namespace: dbd.Status.Storage.Reference.Namespace}, servers)).To(Succeed())

				By("DW Directive verifying servers resource")
				switch storageDirective {
				case "jobdw":
					Expect(servers.ObjectMeta.OwnerReferences).To(ContainElement(metav1.OwnerReference{
						Kind:               reflect.TypeOf(dwsv1alpha2.DirectiveBreakdown{}).Name(),
						APIVersion:         dwsv1alpha2.GroupVersion.String(),
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
					storage := make([]dwsv1alpha2.ServersSpecStorage, 0, len(nodeNames))
					for _, nodeName := range nodeNames {
						storage = append(storage, dwsv1alpha2.ServersSpecStorage{
							AllocationCount: 1,
							Name:            nodeName,
						})
					}

					Expect(dbd.Status.Storage.AllocationSets).To(HaveLen(1))
					allocSet := &dbd.Status.Storage.AllocationSets[0]

					servers.Spec.AllocationSets = make([]dwsv1alpha2.ServersSpecAllocationSet, 1)
					servers.Spec.AllocationSets[0] = dwsv1alpha2.ServersSpecAllocationSet{
						AllocationSize: allocSet.MinimumCapacity,
						Label:          allocSet.Label,
						Storage:        storage,
					}

				} else {
					// If lustre, allocate one node per allocation set
					Expect(len(nodeNames) >= len(dbd.Status.Storage.AllocationSets)).To(BeTrue())
					servers.Spec.AllocationSets = make([]dwsv1alpha2.ServersSpecAllocationSet, len(dbd.Status.Storage.AllocationSets))
					for idx, allocset := range dbd.Status.Storage.AllocationSets {
						servers.Spec.AllocationSets[idx] = dwsv1alpha2.ServersSpecAllocationSet{
							AllocationSize: allocset.MinimumCapacity,
							Label:          allocset.Label,
							Storage: []dwsv1alpha2.ServersSpecStorage{
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
			computes.Data = make([]dwsv1alpha2.ComputesData, 0, len(nodeNames))
			for idx := range nodeNames {
				computes.Data = append(computes.Data, dwsv1alpha2.ComputesData{Name: fmt.Sprintf("compute%d", idx*16)})
			}
			Expect(k8sClient.Update(context.TODO(), computes)).To(Succeed())

			/***************************** Setup *****************************/

			advanceStateAndCheckReady(dwsv1alpha2.StateSetup, workflow)

			By("Checking Setup state")
			switch storageDirective {
			case "jobdw":
			case "persistendw":
				checkPSIConsumerReference(storageName, workflow)
				checkServersToNnfStorageMapping(nnfStoragePresent)
			default:
				checkServersToNnfStorageMapping(nnfStoragePresent)
			}

			// TODO

			/**************************** Data In ****************************/

			advanceStateAndCheckReady(dwsv1alpha2.StateDataIn, workflow)

			By("Checking Data In state")
			// TODO

			/**************************** Pre Run ****************************/

			advanceStateAndCheckReady(dwsv1alpha2.StatePreRun, workflow)

			By("Checking Pre Run state")

			switch storageDirective {
			default:
				for _, dbdRef := range workflow.Status.DirectiveBreakdowns {
					dbd := &dwsv1alpha2.DirectiveBreakdown{}
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
						"TeardownState": Equal(dwsv1alpha2.StatePostRun),
						"DesiredState":  Equal("mounted"),
						"Target":        Equal("single"),
					}))
					Expect(access.Status.State).To(Equal("mounted"))
					Eventually(func(g Gomega) {
						g.Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(access), access)).To(Succeed())
						g.Expect(access.Status.Ready).To(BeTrue())
					}).Should(Succeed())

					By("Checking NNF Access computes reference exists")
					Expect(access.Spec.ClientReference).To(MatchFields(IgnoreExtras, Fields{
						"Name":      Equal(workflow.Name),
						"Namespace": Equal(workflow.Namespace),
						"Kind":      Equal(reflect.TypeOf(dwsv1alpha2.Computes{}).Name()),
					}))
					computes := &dwsv1alpha2.Computes{
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
						clientMount := &dwsv1alpha2.ClientMount{
							ObjectMeta: metav1.ObjectMeta{
								Name:      clientMountName(access),
								Namespace: compute.Name,
							},
						}
						Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(clientMount), clientMount)).To(Succeed())
						Expect(clientMount.Status.Mounts).To(HaveLen(1))
						Expect(clientMount.Labels[dwsv1alpha2.WorkflowNameLabel]).To(Equal(workflow.Name))
						Expect(clientMount.Labels[dwsv1alpha2.WorkflowNamespaceLabel]).To(Equal(workflow.Namespace))
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

			advanceStateAndCheckReady(dwsv1alpha2.StatePostRun, workflow)

			By("Checking Post Run state")

			switch storageDirective {
			default:
				for _, dbdRef := range workflow.Status.DirectiveBreakdowns {
					dbd := &dwsv1alpha2.DirectiveBreakdown{}
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

			advanceStateAndCheckReady(dwsv1alpha2.StateDataOut, workflow)

			By("Checking Data Out state")
			// TODO

			/**************************** Teardown ****************************/

			advanceStateAndCheckReady(dwsv1alpha2.StateTeardown, workflow)

			By("Checking Teardown state")

			for _, dbdRef := range workflow.Status.DirectiveBreakdowns {
				dbd := &dwsv1alpha2.DirectiveBreakdown{}
				Expect(k8sClient.Get(context.TODO(), types.NamespacedName{Name: dbdRef.Name, Namespace: dbdRef.Namespace}, dbd)).To(Succeed())

				switch storageDirective {
				case "create_persistent":

					By("Check that the servers resource still exists")
					servers := &dwsv1alpha2.Servers{}
					Expect(dbd.Status.Storage.Reference.Kind).To(Equal(reflect.TypeOf(dwsv1alpha2.Servers{}).Name()))
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
					servers := &dwsv1alpha2.Servers{}
					Expect(dbd.Status.Storage.Reference.Kind).To(Equal(reflect.TypeOf(dwsv1alpha2.Servers{}).Name()))
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

			workflow = &dwsv1alpha2.Workflow{
				ObjectMeta: metav1.ObjectMeta{
					Name:      workflowName,
					Namespace: corev1.NamespaceDefault,
				},
				Spec: dwsv1alpha2.WorkflowSpec{
					DesiredState: dwsv1alpha2.StateProposal,
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
				return workflow.Status.State == dwsv1alpha2.StateProposal && workflow.Status.Ready
			}).Should(BeTrue())

			By("Verifying it has no directiveBreakdowns")
			directiveBreakdown := &dwsv1alpha2.DirectiveBreakdown{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("%s-%d", workflowName, 0),
					Namespace: corev1.NamespaceDefault,
				},
			}
			Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(directiveBreakdown), directiveBreakdown)).ShouldNot(Succeed())

			By("Verifying it has no servers")
			servers := &dwsv1alpha2.Servers{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("%s-%d", workflowName, 0),
					Namespace: corev1.NamespaceDefault,
				},
			}
			Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(servers), servers)).ShouldNot(Succeed())

			// Store ownership reference to workflow - this is checked for many of the created objects
			ownerRef := metav1.OwnerReference{
				Kind:               reflect.TypeOf(dwsv1alpha2.Workflow{}).Name(),
				APIVersion:         dwsv1alpha2.GroupVersion.String(),
				UID:                workflow.GetUID(),
				Name:               workflow.GetName(),
				Controller:         &controller,
				BlockOwnerDeletion: &blockOwnerDeletion,
			}

			By("Verifying it has Computes resource")
			computes := &dwsv1alpha2.Computes{
				ObjectMeta: metav1.ObjectMeta{
					Name:      workflow.Status.Computes.Name,
					Namespace: workflow.Status.Computes.Namespace,
				},
			}
			Expect(workflow.Status.Computes.Kind).To(Equal(reflect.TypeOf(dwsv1alpha2.Computes{}).Name()))
			Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(computes), computes)).To(Succeed())
			Expect(computes.ObjectMeta.OwnerReferences).To(ContainElement(ownerRef))

			/***************************** Teardown *****************************/
			advanceStateAndCheckReady(dwsv1alpha2.StateTeardown, workflow)
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
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: nnfv1alpha1.DataMovementNamespace,
				},
			}

			k8sClient.Create(context.TODO(), ns)

			wfid := uuid.NewString()[0:8]
			workflow = &dwsv1alpha2.Workflow{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("gfs2-%s", wfid),
					Namespace: corev1.NamespaceDefault,
				},
				Spec: dwsv1alpha2.WorkflowSpec{
					DesiredState: dwsv1alpha2.StateProposal,
					JobID:        intstr.FromString("a job id"),
					WLMID:        "Test WLMID",
				},
			}

			dmm = &nnfv1alpha1.NnfDataMovementManager{
				ObjectMeta: metav1.ObjectMeta{
					Name:      nnfv1alpha1.DataMovementManagerName,
					Namespace: nnfv1alpha1.DataMovementNamespace,
				},
				Spec: nnfv1alpha1.NnfDataMovementManagerSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{{
								Name:  "dm-worker-dummy",
								Image: "nginx",
							}},
						},
					},
				},
				Status: nnfv1alpha1.NnfDataMovementManagerStatus{
					Ready: true,
				},
			}

		})

		// Bring the workflow up to Data In; assign the necessary servers and computes
		// resource. This isn't meant to be a vigorous test of the proposal and setup
		// stages; that is provided by the topmost integration test.
		JustBeforeEach(func() {
			By("Creating the workflow")
			Expect(k8sClient.Create(context.TODO(), workflow)).To(Succeed())

			By("Checking for workflow creation")
			Eventually(func() error {
				return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(workflow), workflow)
			}).Should(Succeed())

			By("Creating the NnfDataMovementManager")
			Expect(k8sClient.Create(context.TODO(), dmm)).To(Succeed())
			WaitForDMMReady(dmm)

			/*************************** Proposal ****************************/

			By("Checking for proposal state and ready")
			Eventually(func() bool {
				Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(workflow), workflow)).To(Succeed())
				return workflow.Status.State == dwsv1alpha2.StateProposal && workflow.Status.Ready
			}).Should(BeTrue())

			for _, dbdRef := range workflow.Status.DirectiveBreakdowns {
				By("Checking DW Directive Breakdown")
				dbd := &dwsv1alpha2.DirectiveBreakdown{}
				Expect(k8sClient.Get(context.TODO(), types.NamespacedName{Name: dbdRef.Name, Namespace: dbdRef.Namespace}, dbd)).To(Succeed())

				Expect(dbd.Status.Storage.AllocationSets).To(HaveLen(1))
				allocSet := &dbd.Status.Storage.AllocationSets[0]

				By("Assigning storage")
				servers := &dwsv1alpha2.Servers{}
				Expect(k8sClient.Get(context.TODO(), types.NamespacedName{Name: dbd.Status.Storage.Reference.Name, Namespace: dbd.Status.Storage.Reference.Namespace}, servers)).To(Succeed())

				storage := make([]dwsv1alpha2.ServersSpecStorage, 0, len(nodeNames))
				for _, nodeName := range nodeNames {
					storage = append(storage, dwsv1alpha2.ServersSpecStorage{
						AllocationCount: 1,
						Name:            nodeName,
					})
				}

				servers.Spec.AllocationSets = []dwsv1alpha2.ServersSpecAllocationSet{
					{
						AllocationSize: allocSet.MinimumCapacity,
						Label:          allocSet.Label,
						Storage:        storage,
					},
				}

				Expect(k8sClient.Update(context.TODO(), servers)).To(Succeed())
			} // for _, dbdRef := range workflow.Status.DirectiveBreakdowns

			/***************************** Setup *****************************/

			advanceStateAndCheckReady(dwsv1alpha2.StateSetup, workflow)

			/**************************** Data In ****************************/

			advanceStateAndCheckReady(dwsv1alpha2.StateDataIn, workflow)
		})

		AfterEach(func() {

			/**************************** Teardown ****************************/

			advanceStateAndCheckReady(dwsv1alpha2.StateTeardown, workflow)

			Expect(k8sClient.Delete(context.TODO(), dmm)).To(Succeed())
			Eventually(func() error { // Delete can still return the cached object. Wait until the object is no longer present
				return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(dmm), dmm)
			}).ShouldNot(Succeed())
		})

		var lustre *lusv1beta1.LustreFileSystem

		BeforeEach(func() {
			lustre = &lusv1beta1.LustreFileSystem{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "lustre-test",
					Namespace: corev1.NamespaceDefault,
				},
				Spec: lusv1beta1.LustreFileSystemSpec{
					Name:      "lustre",
					MgsNids:   "172.0.0.1@tcp",
					MountRoot: "/lus/maui",
				},
			}

			Expect(k8sClient.Create(context.TODO(), lustre)).To(Succeed())
		})

		AfterEach(func() {
			Expect(k8sClient.Delete(context.TODO(), lustre)).To(Succeed())
		})

		validateNnfAccessHasCorrectTeardownState := func(state dwsv1alpha2.WorkflowState) {
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
				"TeardownState": Equal(state),
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
					"#DW copy_in source=/lus/maui/file destination=$DW_JOB_test_dm_0/file",
				}
			})

			It("Validates Workflow", func() {
				Expect(workflow.Status.State).To(Equal(dwsv1alpha2.StateDataIn))

				By("Check that NNF Access is created, with deletion in post-run")
				validateNnfAccessHasCorrectTeardownState(dwsv1alpha2.StatePostRun)

				By("Advancing to post run, ensure NNF Access is deleted")
				advanceStateAndCheckReady(dwsv1alpha2.StatePreRun, workflow)
				advanceStateAndCheckReady(dwsv1alpha2.StatePostRun, workflow)
				validateNnfAccessIsNotFound()
			})
		})

		// If there is only a copy_out directive, NNF Access is not needed until pre-run
		// and should remain until post-run.
		Describe("Test with copy_out directive", func() {
			BeforeEach(func() {
				workflow.Spec.DWDirectives = []string{
					"#DW jobdw name=test-dm-0 type=gfs2 capacity=1GiB",
					"#DW copy_out source=$DW_JOB_test_dm_0/file destination=/lus/maui/file",
				}
			})

			It("Validates Workflow", func() {
				Expect(workflow.Status.State).To(Equal(dwsv1alpha2.StateDataIn))

				validateNnfAccessIsNotFound()
				advanceStateAndCheckReady(dwsv1alpha2.StatePreRun, workflow)

				By("Validate NNF Access is created, with deletion in data-out")
				validateNnfAccessHasCorrectTeardownState(dwsv1alpha2.StateTeardown)

				By("Advancing to post run, ensure NNF Access is still set for deletion in data-out")
				advanceStateAndCheckReady(dwsv1alpha2.StatePostRun, workflow)
				validateNnfAccessHasCorrectTeardownState(dwsv1alpha2.StateTeardown)

				By("Advancing to data-out, ensure NNF Access is deleted")
				advanceStateAndCheckReady(dwsv1alpha2.StateDataOut, workflow)
				validateNnfAccessIsNotFound()
			})
		})

		Describe("Test with copy_in and copy_out directives", func() {
			BeforeEach(func() {
				workflow.Spec.DWDirectives = []string{
					"#DW jobdw name=test-dm-0 type=gfs2 capacity=1GiB",
					"#DW copy_in source=/lus/maui/file destination=$DW_JOB_test_dm_0/file",
					"#DW copy_out source=$DW_JOB_test_dm_0/file destination=/lus/maui/file",
				}
			})

			It("Validates Workflow", func() {
				Expect(workflow.Status.State).To(Equal(dwsv1alpha2.StateDataIn))

				By("Validate NNF Access is created, with deletion in data-out")
				validateNnfAccessHasCorrectTeardownState(dwsv1alpha2.StateTeardown)

				advanceStateAndCheckReady(dwsv1alpha2.StatePreRun, workflow)
				advanceStateAndCheckReady(dwsv1alpha2.StatePostRun, workflow)

				By("Advancing to data-out, ensure NNF Access is deleted")
				advanceStateAndCheckReady(dwsv1alpha2.StateDataOut, workflow)
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
				Expect(workflow.Status.State).To(Equal(dwsv1alpha2.StateDataIn))

				advanceStateAndCheckReady(dwsv1alpha2.StatePreRun, workflow)

				By("Injecting an error in the data movement resource")

				dm := &nnfv1alpha1.NnfDataMovement{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "failed-data-movement",
						Namespace: nnfv1alpha1.DataMovementNamespace,
					},
				}
				dwsv1alpha2.AddWorkflowLabels(dm, workflow)
				dwsv1alpha2.AddOwnerLabels(dm, workflow)
				nnfv1alpha1.AddDataMovementTeardownStateLabel(dm, dwsv1alpha2.StatePostRun)

				Expect(k8sClient.Create(context.TODO(), dm)).To(Succeed())

				Eventually(func() error {
					return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(dm), dm)
				}).Should(Succeed())

				dm.Status.State = nnfv1alpha1.DataMovementConditionTypeFinished
				dm.Status.Status = nnfv1alpha1.DataMovementConditionReasonFailed

				Expect(k8sClient.Status().Update(context.TODO(), dm)).To(Succeed())

				By("Advancing to post-run")
				advanceState(dwsv1alpha2.StatePostRun, workflow, 1)

				By("Checking the driver has an error present")
				Eventually(func() *dwsv1alpha2.WorkflowDriverStatus {
					Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(workflow), workflow)).To(Succeed())

					driverID := os.Getenv("DWS_DRIVER_ID")
					for _, driver := range workflow.Status.Drivers {
						if driver.DriverID == driverID {
							if driver.Status == dwsv1alpha2.StatusError {
								return &driver
							}
						}
					}

					return nil
				}).ShouldNot(BeNil())
			})
		})

	})

	Describe("Test with container directives", func() {
		var (
			containerProfile *nnfv1alpha1.NnfContainerProfile
		)

		BeforeEach(func() {
			containerProfile = createBasicNnfContainerProfile(nil)
			Expect(containerProfile).ToNot(BeNil())

			wfName := "container-test"
			workflow = &dwsv1alpha2.Workflow{
				ObjectMeta: metav1.ObjectMeta{
					Name:      wfName,
					Namespace: corev1.NamespaceDefault,
				},
				Spec: dwsv1alpha2.WorkflowSpec{
					DesiredState: dwsv1alpha2.StateProposal,
					JobID:        intstr.FromString("job 1234"),
					WLMID:        "Test WLMID",
					DWDirectives: []string{
						fmt.Sprintf("#DW container name=%s profile=%s", wfName, containerProfile.Name),
					},
				},
			}
			Expect(k8sClient.Create(context.TODO(), workflow)).To(Succeed())

			Eventually(func(g Gomega) error {
				return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(workflow), workflow)
			}).Should(Succeed())

			advanceStateAndCheckReady("Proposal", workflow)
			Expect(verifyPinnedContainerProfile(context.TODO(), k8sClient, workflow, 0)).To(Succeed())
		})

		AfterEach(func() {
			if workflow != nil {
				advanceStateAndCheckReady("Teardown", workflow)
				Expect(k8sClient.Delete(context.TODO(), workflow)).To(Succeed())
			}

			if containerProfile != nil {
				Expect(k8sClient.Delete(ctx, containerProfile)).To(Succeed())
			}
		})

		When("compute nodes are selected for container directives", func() {
			It("it should target the local NNF nodes for the container jobs", func() {
				By("assigning the container directive to compute nodes")

				computes := &dwsv1alpha2.Computes{
					ObjectMeta: metav1.ObjectMeta{
						Name:      workflow.Status.Computes.Name,
						Namespace: workflow.Status.Computes.Namespace,
					},
				}

				// Add 2 computes across 2 NNF nodes. For containers, this means we should see 2 jobs
				// These computes are defined in the SystemConfiguration
				Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(computes), computes)).To(Succeed())
				data := []dwsv1alpha2.ComputesData{
					{Name: "compute0"},
					{Name: "compute32"},
				}
				computes.Data = append(computes.Data, data...)
				Expect(k8sClient.Update(context.TODO(), computes)).To(Succeed())

				By("advancing to the PreRun state")
				advanceStateAndCheckReady("Setup", workflow)
				advanceStateAndCheckReady("DataIn", workflow)
				advanceState("PreRun", workflow, 1)

				By("verifying the number of targeted NNF nodes for the container jobs")
				matchLabels := dwsv1alpha2.MatchingWorkflow(workflow)
				matchLabels[nnfv1alpha1.DirectiveIndexLabel] = "0"

				jobList := &batchv1.JobList{}
				Eventually(func() int {
					Expect(k8sClient.List(context.TODO(), jobList, matchLabels)).To(Succeed())
					return len(jobList.Items)
				}).Should(Equal(2))

				By("verifying the node names of targeted NNF nodes for the container jobs")
				nodes := []string{}
				for _, job := range jobList.Items {
					nodeName, ok := job.Spec.Template.Spec.NodeSelector["kubernetes.io/hostname"]
					Expect(ok).To(BeTrue())
					nodes = append(nodes, nodeName)
				}
				Expect(nodes).To(ContainElements("rabbit-test-node-0", "rabbit-test-node-2"))
			})
		})
	})

	Describe("Test NnfStorageProfile Lustre profile merge with DW directives", func() {

		var (
			intendedDirective     string
			profileExternalMGS    *nnfv1alpha1.NnfStorageProfile
			profileCombinedMGTMDT *nnfv1alpha1.NnfStorageProfile
			nnfLustreMgt          *nnfv1alpha1.NnfLustreMGT

			profileMgsNid string

			dbd       *dwsv1alpha2.DirectiveBreakdown
			dbdServer *dwsv1alpha2.Servers

			externalMgsProfileName    string
			combinedMgtMdtProfileName string
		)

		BeforeEach(func() {
			profileMgsNid = "profile-mgs@tcp"

			dbd = &dwsv1alpha2.DirectiveBreakdown{}
			dbdServer = &dwsv1alpha2.Servers{}

			externalMgsProfileName = "has-external-mgs"
			combinedMgtMdtProfileName = "has-combined-mgtmdt"
		})

		// Create some custom storage profiles.
		BeforeEach(func() {
			By("BeforeEach create some custom storage profiles")
			profileExternalMGS = basicNnfStorageProfile(externalMgsProfileName)
			profileExternalMGS.Data.LustreStorage.ExternalMGS = profileMgsNid
			profileCombinedMGTMDT = basicNnfStorageProfile(combinedMgtMdtProfileName)
			profileCombinedMGTMDT.Data.LustreStorage.CombinedMGTMDT = true
			Expect(createNnfStorageProfile(profileExternalMGS, true)).ToNot(BeNil())
			Expect(createNnfStorageProfile(profileCombinedMGTMDT, true)).ToNot(BeNil())

			nnfLustreMgt = &nnfv1alpha1.NnfLustreMGT{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "profile-mgs",
					Namespace: corev1.NamespaceDefault,
				},
				Spec: nnfv1alpha1.NnfLustreMGTSpec{
					Addresses:   []string{profileMgsNid},
					FsNameStart: "dddddddd",
				},
			}
			Expect(k8sClient.Create(context.TODO(), nnfLustreMgt)).To(Succeed())
		})

		// Destroy the custom storage profiles.
		AfterEach(func() {
			By("AfterEach destroy the custom storage profiles")
			Expect(k8sClient.Delete(context.TODO(), profileExternalMGS)).To(Succeed())
			Expect(k8sClient.Delete(context.TODO(), profileCombinedMGTMDT)).To(Succeed())
			Expect(k8sClient.Delete(context.TODO(), nnfLustreMgt)).To(Succeed())

		})

		// Setup a basic workflow; each test is expected to fill in the
		// DWDirectives in its BeforeEach() clause.
		BeforeEach(func() {
			By("BeforeEach setup a basic workflow resource")
			wfid := uuid.NewString()[0:8]
			workflow = &dwsv1alpha2.Workflow{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("profile-%s", wfid),
					Namespace: corev1.NamespaceDefault,
				},
				Spec: dwsv1alpha2.WorkflowSpec{
					DesiredState: dwsv1alpha2.StateProposal,
					JobID:        intstr.FromString("some job 234"),
					WLMID:        "Test WLMID",
				},
			}
		})

		AfterEach(func() {
			By("AfterEach advance the workflow state to teardown")
			advanceStateAndCheckReady(dwsv1alpha2.StateTeardown, workflow)
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
				if (workflow.Status.Ready == true) && (workflow.Status.State == dwsv1alpha2.StateProposal) {
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
			dbdServer.Spec.AllocationSets = []dwsv1alpha2.ServersSpecAllocationSet{
				{
					AllocationSize: 1,
					Label:          "mdt",
					Storage: []dwsv1alpha2.ServersSpecStorage{
						{AllocationCount: 1,
							Name: "rabbit-test-node-0"},
					},
				},
				{
					AllocationSize: 1,
					Label:          "ost",
					Storage: []dwsv1alpha2.ServersSpecStorage{
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
			advanceStateAndCheckReady(dwsv1alpha2.StateSetup, workflow)
			// The NnfStorage's name matches the Server resource's name.
			nnfstorage := &nnfv1alpha1.NnfStorage{}
			Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(dbdServer), nnfstorage)).To(Succeed())
			for _, comp := range nnfstorage.Spec.AllocationSets {
				Expect(comp.MgsAddress).To(Equal(desiredNid))
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

		When("using external_mgs in profile", func() {
			BeforeEach(func() {
				intendedDirective = fmt.Sprintf("#DW jobdw type=lustre profile=%s capacity=5GB name=profile-mgs", externalMgsProfileName)
			})

			It("Uses external_mgs via the profile", func() {
				verifyExternalMgsNid("via profile", profileMgsNid)
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
	})

	Describe("Test failure cases for various directives", func() {

		var (
			directives []string
		)

		// Create a basic workflow; each test is expected to fill in the DWDirectives
		// in its BeforeEach() clause.
		BeforeEach(func() {
			wfid := uuid.NewString()[0:8]
			workflow = &dwsv1alpha2.Workflow{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("persistent-%s", wfid),
					Namespace: corev1.NamespaceDefault,
				},
				Spec: dwsv1alpha2.WorkflowSpec{
					DesiredState: dwsv1alpha2.StateProposal,
					JobID:        intstr.FromString("job 2222"),
					WLMID:        "Test WLMID",
				},
			}
		})

		AfterEach(func() {
			By("AfterEach advance the workflow state to teardown")
			advanceStateAndCheckReady(dwsv1alpha2.StateTeardown, workflow)
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
			Eventually(func(g Gomega) *dwsv1alpha2.WorkflowDriverStatus {
				g.Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(workflow), workflow)).To(Succeed())
				g.Expect(workflow.Status.Ready == false)
				g.Expect(workflow.Status.State == dwsv1alpha2.StateProposal)

				driverID := os.Getenv("DWS_DRIVER_ID")
				for _, driver := range workflow.Status.Drivers {
					if driver.DriverID == driverID {
						if driver.Status == dwsv1alpha2.StatusError {
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
				lustre *lusv1beta1.LustreFileSystem
			)

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
			BeforeEach(func() {
				directives = []string{
					"#DW copy_in source=/lus/maui/my-file.in destination=$DW_JOB_not_there/my-file.out",
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
				lustre *lusv1beta1.LustreFileSystem
			)

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

			BeforeEach(func() {
				directives = []string{
					"#DW copy_in source=/lus/maui/my-file.in destination=$DW_PERSISTENT_mpi/my-persistent-file.out",
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
