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
		wfDirective       = "#DW jobdw name=test type=%s capacity=10GiB"
	)
	const timeout = time.Second * 5
	const interval = time.Millisecond * 100

	var savedWorkflow *dwsv1alpha1.Workflow
	var savedDirectiveBreakdown *dwsv1alpha1.DirectiveBreakdown

	type fsToTest struct {
		fsName      string
		isSupported bool
	}
	var filesystems = []fsToTest{
		{"raw", true},
		{"xfs", true},
		{"lustre", true},

		// The following are not yet supported
		// {"lvm", false},
		// {"gfs2", false},
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

				// Save a copy of the workflow for use during deletion
				savedWorkflow = wf

				Eventually(func() (string, error) {
					err := k8sClient.Get(context.Background(), client.ObjectKey{Namespace: WorkflowNamespace, Name: workflowName}, wf)
					if err != nil {
						return "", err
					}
					return wf.Status.State, nil
				}).Should(Equal("proposal"))
			})

			It("Should have a single Computes that is owned by the workflow", func() {
				computes := &dwsv1alpha1.Computes{}
				name := workflowName
				namespace := WorkflowNamespace

				Eventually(func() error {
					err := k8sClient.Get(context.Background(), client.ObjectKey{Name: name, Namespace: namespace}, computes)
					return err
				}, timeout, interval).Should(Succeed())

				// From https://book.kubebuilder.io/reference/envtest.html
				// Unless you’re using an existing cluster, keep in mind that no built-in controllers
				// are running in the test context. In some ways, the test control plane will behave
				// differently from “real” clusters, and that might have an impact on how you write tests.
				// One common example is garbage collection; because there are no controllers monitoring
				// built-in resources, objects do not get deleted, even if an OwnerReference is set up.
				//
				// To test that the deletion lifecycle works, test the ownership instead of asserting on existence.
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

			It("Should have a directiveBreakdown that is owned by the workflow", func() {
				directiveBreakdown := &dwsv1alpha1.DirectiveBreakdown{}
				name := workflowName + "-" + strconv.Itoa(0)
				namespace := WorkflowNamespace

				Eventually(func() error {
					err := k8sClient.Get(context.Background(), client.ObjectKey{Name: name, Namespace: namespace}, directiveBreakdown)
					return err
				}, timeout, interval).Should(Succeed())

				// From https://book.kubebuilder.io/reference/envtest.html
				// Unless you’re using an existing cluster, keep in mind that no built-in controllers
				// are running in the test context. In some ways, the test control plane will behave
				// differently from “real” clusters, and that might have an impact on how you write tests.
				// One common example is garbage collection; because there are no controllers monitoring
				// built-in resources, objects do not get deleted, even if an OwnerReference is set up.
				//
				// To test that the deletion lifecycle works, test the ownership instead of asserting on existence.
				t := true
				expectedOwnerReference := metav1.OwnerReference{
					APIVersion:         savedWorkflow.APIVersion,
					Kind:               savedWorkflow.Kind,
					Name:               savedWorkflow.Name,
					UID:                savedWorkflow.UID,
					Controller:         &t,
					BlockOwnerDeletion: &t,
				}

				Expect(directiveBreakdown.ObjectMeta.OwnerReferences).Should(ContainElement(expectedOwnerReference))
				savedDirectiveBreakdown = directiveBreakdown
			})

			It("Should have a single Servers that is owned by the directiveBreakdown", func() {
				servers := &dwsv1alpha1.Servers{}
				name := workflowName + "-" + strconv.Itoa(0)
				namespace := WorkflowNamespace

				Eventually(func() error {
					err := k8sClient.Get(context.Background(), client.ObjectKey{Name: name, Namespace: namespace}, servers)
					return err
				}, timeout, interval).Should(Succeed())

				// From https://book.kubebuilder.io/reference/envtest.html
				// Unless you’re using an existing cluster, keep in mind that no built-in controllers
				// are running in the test context. In some ways, the test control plane will behave
				// differently from “real” clusters, and that might have an impact on how you write tests.
				// One common example is garbage collection; because there are no controllers monitoring
				// built-in resources, objects do not get deleted, even if an OwnerReference is set up.
				//
				// To test that the deletion lifecycle works, test the ownership instead of asserting on existence.
				t := true
				expectedOwnerReference := metav1.OwnerReference{
					APIVersion:         savedDirectiveBreakdown.APIVersion,
					Kind:               savedDirectiveBreakdown.Kind,
					Name:               savedDirectiveBreakdown.Name,
					UID:                savedDirectiveBreakdown.UID,
					Controller:         &t,
					BlockOwnerDeletion: &t,
				}

				Expect(servers.ObjectMeta.OwnerReferences).Should(ContainElement(expectedOwnerReference))
			})

			// Once all of the objects above have been created, the workflow should achieve "proposal"
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

		Describe(fmt.Sprintf("Deleting workflow %s for file system %s", workflowName, f.fsName), func() {
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
				Eventually(func() error {
					return k8sClient.Get(context.Background(), client.ObjectKey{Namespace: WorkflowNamespace, Name: workflowName}, wf)
				}).ShouldNot(Succeed())
			})
		})
	}
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
