/*
Copyright 2021 Hewlett Packard Enterprise Development LP
*/

package controllers

import (
	"context"
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	nnfv1alpha1 "stash.us.cray.com/RABSW/nnf-sos/api/v1alpha1"
	dwsv1alpha1 "stash.us.cray.com/dpm/dws-operator/api/v1alpha1"
)

var _ = Describe("Integration Test", func() {

	const (
		WorkflowName      = "test-workflow"
		WorkflowNamespace = "default"
		WorkflowID        = "test"
	)

	const timeout = time.Second * 10
	const interval = time.Second * 1

	Describe("Creating a Workflow Resource", func() {

		for _, fs := range []string{"raw", "lvm", "xfs", "gfs2", "lustre"} {

			// TODO: Remove this as more file systems are supported and tested
			BeforeEach(func() {
				if fs != "raw" {
					Skip("File System Not Supported")
				}
			})

			Context(fmt.Sprintf("%s File System", strings.ToTitle(fs)), func() {

				var workflow *dwsv1alpha1.Workflow
				var storage *nnfv1alpha1.NnfStorage

				It("Should create successfully", func() {
					workflow = &dwsv1alpha1.Workflow{
						ObjectMeta: metav1.ObjectMeta{
							Name:      WorkflowName,
							Namespace: WorkflowNamespace,
						},
						Spec: dwsv1alpha1.WorkflowSpec{
							DesiredState: "proposal", // TODO: This should be defined somewhere
							WLMID:        WorkflowID,
							DWDirectives: []string{
								fmt.Sprintf("#DW name=test filesystem=%s capacity=10GB", fs), // TODO: Should be a MakeDW method
							},
						},
					}

					Expect(k8sClient.Create(context.Background(), workflow)).Should(Succeed())
				})

				It("Should reach ready status and proposed state", func() {
					Eventually(func() bool {
						k8sClient.Get(context.Background(), types.NamespacedName{Name: WorkflowName, Namespace: WorkflowNamespace}, workflow)
						return workflow.Status.Ready
					}, timeout, interval).Should(BeTrue())

					Expect(workflow.Status.State).Should(Equal("proposal"))
				})

				PIt("Should have a single DW breakdown", func() {
					Expect(workflow.Status.DWDirectiveBreakdowns).Should(HaveLen(1))
				})

				PIt("Should have a NNF Storage reference in the DW breakdown", func() {
					//TODO
				})

				PIt("Should have created the storage allocation", func() {
					err := k8sClient.Get(context.Background(), types.NamespacedName{Name: "", Namespace: ""}, storage)
					Expect(err).ShouldNot(HaveOccurred())
				})

				PContext("Assigning NNF Node Names to NNF Storage", func() {

					var nodes *nnfv1alpha1.NnfNodeList
					It("Should find NNF Nodes", func() {
						err := k8sClient.List(context.Background(), nodes, &client.ListOptions{})
						Expect(err).ShouldNot(HaveOccurred())
					})

				})
			})
		}
	})
})
