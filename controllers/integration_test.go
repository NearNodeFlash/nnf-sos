/*
Copyright 2021 Hewlett Packard Enterprise Development LP
*/

package controllers

import (
	"context"
	"fmt"
	"strconv"

	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	dwsv1alpha1 "stash.us.cray.com/dpm/dws-operator/api/v1alpha1"
)

var _ = Describe("Integration Test", func() {
	const (
		WorkflowNamespace = "default"
		WorkflowID        = "test"
		wfDirective       = "#DW jobdw name=test type=%s capacity=10GB"
	)
	const timeout = time.Second * 10
	const interval = time.Millisecond * 100

	type fsToTest struct {
		fsName      string
		isSupported bool
	}
	var filesystems = []fsToTest{
		{"raw", true},
		{"xfs", true},
		{"lustre", true},

		// The following are not yet supported
		{"lvm", false},
		{"gfs2", false},
	}

	// Spin through the supported file systems
	for index := range filesystems {
		f := filesystems[index] // Ensure closure has the current value from the loop above.
		workflowName := "wf-" + f.fsName

		Describe(fmt.Sprintf("Creating workflow %s for file system %s", workflowName, f.fsName), func() {
			BeforeEach(func() {
				if !f.isSupported {
					Skip(fmt.Sprintf("File System %s Not Supported", f.fsName))
				}
			})

			It("Should create successfully", func() {
				workflow := &dwsv1alpha1.Workflow{
					ObjectMeta: metav1.ObjectMeta{
						Name:      workflowName,
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
			})

			It("Should have a single directiveBreakdown", func() {
				directiveBreakdown := &dwsv1alpha1.DirectiveBreakdown{}
				name := workflowName + "-" + strconv.Itoa(0)
				namespace := WorkflowNamespace

				Eventually(func() error {
					err := k8sClient.Get(context.Background(), client.ObjectKey{Name: name, Namespace: namespace}, directiveBreakdown)
					return err
				}, timeout, interval).Should(Succeed())
			})

			It("Should have a single Computes", func() {
				computes := &dwsv1alpha1.Computes{}
				name := workflowName
				namespace := WorkflowNamespace

				Eventually(func() error {
					err := k8sClient.Get(context.Background(), client.ObjectKey{Name: name, Namespace: namespace}, computes)
					return err
				}, timeout, interval).Should(Succeed())
			})

			It("Should have a single Servers", func() {
				servers := &dwsv1alpha1.Servers{}
				name := workflowName + "-" + strconv.Itoa(0)
				namespace := WorkflowNamespace

				Eventually(func() error {
					err := k8sClient.Get(context.Background(), client.ObjectKey{Name: name, Namespace: namespace}, servers)
					return err
				}, timeout, interval).Should(Succeed())
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

		PDescribe(fmt.Sprintf("Deleting workflow %s for file system %s", workflowName, f.fsName), func() {
			BeforeEach(func() {
				if !f.isSupported {
					Skip(fmt.Sprintf("File System %s Not Supported", f.fsName))
				}
			})

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
				Eventually(func() bool {
					err := k8sClient.Get(context.Background(), client.ObjectKey{Namespace: WorkflowNamespace, Name: workflowName}, wf)
					return err != nil
				}).Should(BeTrue())
			})

			It("DirectiveBreakdown should be deleted", func() {
				directiveBreakdown := &dwsv1alpha1.DirectiveBreakdown{}
				name := workflowName + "-" + strconv.Itoa(0)
				namespace := WorkflowNamespace

				Eventually(func() bool {
					err := k8sClient.Get(context.Background(), client.ObjectKey{Name: name, Namespace: namespace}, directiveBreakdown)
					return err != nil
				}, timeout, interval).Should(BeTrue())
			})

			It("Computes should be deleted", func() {
				computes := &dwsv1alpha1.Computes{}
				name := workflowName
				namespace := WorkflowNamespace

				Eventually(func() bool {
					err := k8sClient.Get(context.Background(), client.ObjectKey{Name: name, Namespace: namespace}, computes)
					return err != nil
				}, timeout, interval).Should(BeTrue())
			})

			It("Servers should be deleted", func() {
				servers := &dwsv1alpha1.Servers{}
				name := workflowName + "-" + strconv.Itoa(0)
				namespace := WorkflowNamespace

				Eventually(func() bool {
					err := k8sClient.Get(context.Background(), client.ObjectKey{Name: name, Namespace: namespace}, servers)
					return err != nil
				}, timeout, interval).Should(BeTrue())
			})
		})
	}
})
