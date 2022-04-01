/*
Copyright 2021 Hewlett Packard Enterprise Development LP
*/

package controllers

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"

	dwsv1alpha1 "github.hpe.com/hpe/hpc-dpm-dws-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("Integration Test", func() {

	const timeout = time.Second * 10
	const interval = time.Millisecond * 100

	type fsToTest struct {
		fsType                 string
		expectedAllocationSets int
	}

	var filesystems = []fsToTest{
		{"raw", 1},
		//{"lvm", 1},
		{"xfs", 1},
		{"gfs2", 1},
		{"lustre", 3},
	}

	var workflow *dwsv1alpha1.Workflow

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
						fmt.Sprintf("#DW jobdw name=%s type=%s capacity=1GiB", "test-1", fsType),
					},
				},
			}

			By("Creating the workflow")
			Expect(k8sClient.Create(context.TODO(), workflow)).Should(Succeed())

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

			By("Checking proposal state and ready")
			Eventually(func() bool {
				Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(workflow), workflow)).To(Succeed())
				return workflow.Status.State == dwsv1alpha1.StateProposal.String() && workflow.Status.Ready
			}).Should(BeTrue())

			By("Checking for Computes resource")
			computes := &dwsv1alpha1.Computes{}
			Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(workflow), computes)).To(Succeed())
			Expect(computes.ObjectMeta.OwnerReferences).To(ContainElement(ownerRef))

			By("Checking various DW Directive Breakdowns")
			for _, dbdRef := range workflow.Status.DirectiveBreakdowns {
				dbd := &dwsv1alpha1.DirectiveBreakdown{}
				// TODO: Write up a bug and assign to Tony
				//Expect(dbdRef.Kind).To(Equal(reflect.TypeOf(dwsv1alpha1.DirectiveBreakdown{}).Name()))

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
				//Expect(servers.Spec.AllocationSets).To(HaveLen(len(dbd.Status.AllocationSet)))
				// TODO: Check with Tony why this doesn't work
				Expect(servers.ObjectMeta.OwnerReferences).To(ContainElement(metav1.OwnerReference{
					Kind:               reflect.TypeOf(dwsv1alpha1.DirectiveBreakdown{}).Name(),
					APIVersion:         dwsv1alpha1.GroupVersion.String(),
					UID:                dbd.GetUID(),
					Name:               dbd.GetName(),
					Controller:         &controller,
					BlockOwnerDeletion: &blockOwnerDeletion,
				}))

				By("Assigning storage")
				// TODO

			}

			By("Assigning computes")
			// TODO

			advanceStateAndCheckReady := func(state dwsv1alpha1.WorkflowState) {
				By(fmt.Sprintf("Advancing to %s state", state))
				Eventually(func() error {
					Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(workflow), workflow)).To(Succeed())
					workflow.Spec.DesiredState = state.String()
					return k8sClient.Update(context.TODO(), workflow)
				}).Should(Succeed())

				By(fmt.Sprintf("Waiting on state %s and ready", state))
				Eventually(func() bool {
					Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(workflow), workflow)).To(Succeed())
					return workflow.Status.State == state.String() && workflow.Status.Ready
				}).Should(BeTrue())
			}

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
			// TODO

			/*************************** Post Run ****************************/

			advanceStateAndCheckReady(dwsv1alpha1.StatePostRun)

			By("Checking Post Run state")
			// TODO

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

})

var _ = Describe("Empty #DW List Test", func() {
	const (
		WorkflowNamespace = "default"
		WorkflowID        = "test"
	)
	const timeout = time.Second * 10
	const interval = time.Millisecond * 100
	const workflowName = "no-storage"

	var savedWorkflow *dwsv1alpha1.Workflow

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
