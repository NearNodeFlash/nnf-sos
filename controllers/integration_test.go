/*
Copyright 2021 Hewlett Packard Enterprise Development LP
*/

package controllers

import (
	"context"
	"fmt"

	// "strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	//	"k8s.io/apimachinery/pkg/types"

	dwsv1alpha1 "stash.us.cray.com/dpm/dws-operator/api/v1alpha1"
)

var _ = Describe("Integration Test", func() {
	const (
		WorkflowName      = "test-workflow"
		WorkflowNamespace = "default"
		WorkflowID        = "test"
		wfDirective       = "#DW jobdw name=test filesystem=%s capacity=10GB"
	)

	const timeout = time.Second * 5
	const interval = time.Millisecond * 100

	Describe("Creating a Workflow Resource", func() {

		type fsToTest struct {
			fsName      string
			isSupported bool
		}
		var filesystems = []fsToTest{
			{"raw", true},
			// {"lvm", false},
			// {"xfs", false},
			// {"gfs2", false},
			// {"lustre", false},
		}

		// Spin through the supported file systems
		for _, f := range filesystems {

			Describe(fmt.Sprintf("Testing File System %s", f.fsName), func() {

				BeforeEach(func(f fsToTest) func() {
					return func() {
						if !f.isSupported {
							Skip(fmt.Sprintf("File System %s Not Supported", f.fsName))
						}
					}
				}(f))

				Context(fmt.Sprintf("%s File System", strings.ToTitle(f.fsName)), func() {

					var workflow *dwsv1alpha1.Workflow

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
									fmt.Sprintf(wfDirective, f.fsName),
								},
							},
						}

						fmt.Printf("workflow %v", workflow)

						Expect(k8sClient.Create(context.Background(), workflow)).Should(Succeed())
						fmt.Printf("workflow %v", workflow)
					})

					// // TODO: Because the mutating and admission webhooks create the Status section of the workflow,
					// //       we need to find a way to enable the webhooks. dws-operator's work there is non-trivial
					// //       because it also sets up the drivers array which is an important component to test.

					// It("Should be in proposal state", func() {
					// 	for i := 0; i < 10; i++ {
					// 		Eventually(func() error {
					// 			err := k8sClient.Get(context.Background(), types.NamespacedName{Name: WorkflowName, Namespace: WorkflowNamespace}, workflow)
					// 			return err
					// 		}, timeout, interval).Should(Succeed())
					// 		fmt.Printf("workflow %v", workflow)
					// 		time.Sleep(time.Second * 1)
					// 	}

					// 	// Expect(workflow.Status.State).To(Equal("proposal"))
					// })

					// It("Should eventually have a single directiveBreakdown", func() {
					// 	var directiveBreakdown *dwsv1alpha1.DirectiveBreakdown
					// 	dbdName := WorkflowName + strconv.Itoa(0)
					// 	dbdNamespace := WorkflowNamespace

					// 	Eventually(func() error {
					// 		err := k8sClient.Get(context.Background(), types.NamespacedName{Name: dbdName, Namespace: dbdNamespace}, directiveBreakdown)
					// 		return err
					// 	}, timeout, interval).Should(Succeed())

					// 	// Expect(directiveBreakdown.Spec.DWDirectiveBreakdown.DW.DWDirective).Should(Equal(wfDirective))
					// })

					// It("Should reach ready status and proposed state", func() {
					// 	Eventually(func() bool {
					// 		k8sClient.Get(context.Background(), types.NamespacedName{Name: WorkflowName, Namespace: WorkflowNamespace}, workflow)
					// 		return workflow.Status.Ready
					// 	}, timeout, interval).Should(BeTrue())

					// 	Expect(workflow.Status.State).Should(Equal("proposal"))
					// })

					// var servers *dwsv1alpha1.Servers
					// var computes *dwsv1alpha1.Computes

					// PIt("Should have a single DW breakdown", func() {
					// 	Expect(workflow.Status.DirectiveBreakdowns).Should(HaveLen(1))
					// })

					// PIt("Should have a NNF Storage reference in the DW breakdown", func() {
					// 	//TODO
					// })

					// PIt("Should have created the storage allocation", func() {
					// 	err := k8sClient.Get(context.Background(), types.NamespacedName{Name: "", Namespace: ""}, storage)
					// 	Expect(err).ShouldNot(HaveOccurred())
					// })

					// PContext("Assigning NNF Node Names to NNF Storage", func() {

					// 	var nodes *nnfv1alpha1.NnfNodeList
					// 	It("Should find NNF Nodes", func() {
					// 		err := k8sClient.List(context.Background(), nodes, &client.ListOptions{})
					// 		Expect(err).ShouldNot(HaveOccurred())
					// 	})

					// })
				})
			})
		}
	})
})
