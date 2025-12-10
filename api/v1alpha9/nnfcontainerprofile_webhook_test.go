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

package v1alpha9

import (
	"context"
	"os"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.openly.dev/pointy"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// These tests are written in BDD-style using Ginkgo framework. Refer to
// http://onsi.github.io/ginkgo to learn more.

var _ = Describe("NnfContainerProfile Webhook", func() {
	var (
		namespaceName      = os.Getenv("NNF_CONTAINER_PROFILE_NAMESPACE")
		otherNamespaceName string
		otherNamespace     *corev1.Namespace

		pinnedResourceName string
		nnfProfile         *NnfContainerProfile
		newProfile         *NnfContainerProfile
	)

	BeforeEach(func() {
		pinnedResourceName = "test-pinned-" + uuid.NewString()[:8]

		nnfProfile = &NnfContainerProfile{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-" + uuid.NewString()[:8],
				Namespace: namespaceName,
			},
			Data: NnfContainerProfileData{
				NnfSpec: &NnfPodSpec{
					Containers: []NnfContainer{
						{Name: "test", Image: "test:latest", Command: []string{"test"}},
					},
				},
				Storages: []NnfContainerProfileStorage{
					{Name: "DW_JOB_storage", Optional: true},
					{Name: "DW_PERSISTENT_storage", Optional: true},
					{Name: "DW_GLOBAL_storage", Optional: true},
				},
			},
		}

		newProfile = &NnfContainerProfile{}
	})

	BeforeEach(func() {
		otherNamespaceName = "other-" + uuid.NewString()[:8]

		otherNamespace = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: otherNamespaceName,
			},
		}
		Expect(k8sClient.Create(context.TODO(), otherNamespace)).To(Succeed())
	})

	AfterEach(func() {
		Expect(k8sClient.Delete(context.TODO(), otherNamespace)).To(Succeed())
	})

	AfterEach(func() {
		if nnfProfile != nil {
			Expect(k8sClient.Delete(context.TODO(), nnfProfile)).To(Succeed())
			profExpected := &NnfContainerProfile{}
			Eventually(func() error { // Delete can still return the cached object. Wait until the object is no longer present
				return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(nnfProfile), profExpected)
			}).ShouldNot(Succeed())
		}
	})

	It("should accept system profiles in the designated namespace", func() {
		Expect(k8sClient.Create(context.TODO(), nnfProfile)).To(Succeed())
	})

	It("should not accept system profiles that are not in the designated namespace", func() {
		nnfProfile.ObjectMeta.Namespace = otherNamespaceName
		err := k8sClient.Create(context.TODO(), nnfProfile)
		Expect(err.Error()).To(MatchRegexp("webhook .* denied the request: incorrect namespace"))
		nnfProfile = nil
	})

	It("Should not allow a negative retryLimit", func() {
		nnfProfile.Data.RetryLimit = -1
		Expect(k8sClient.Create(context.TODO(), nnfProfile)).ToNot(Succeed())
		nnfProfile = nil
	})

	It("Should allow a zero retryLimit", func() {
		nnfProfile.Data.RetryLimit = 0
		Expect(k8sClient.Create(context.TODO(), nnfProfile)).To(Succeed())
	})

	It("Should not allow a negative postRunTimeoutSeconds", func() {
		nnfProfile.Data.PostRunTimeoutSeconds = pointy.Int64(-1)
		Expect(k8sClient.Create(context.TODO(), nnfProfile)).ToNot(Succeed())
		nnfProfile = nil
	})

	It("Should not allow setting both NnfSpec and NnfMPISpec", func() {
		nnfProfile.Data.NnfSpec = &NnfPodSpec{}
		nnfProfile.Data.NnfMPISpec = &NnfMPISpec{}
		Expect(k8sClient.Create(context.TODO(), nnfProfile)).ToNot(Succeed())
		nnfProfile = nil
	})

	It("Should fail when both NnfSpec and NnfMPISpec are unset", func() {
		nnfProfile.Data.NnfSpec = nil
		nnfProfile.Data.NnfMPISpec = nil
		Expect(k8sClient.Create(context.TODO(), nnfProfile)).ToNot(Succeed())
		nnfProfile = nil
	})

	DescribeTable("Should allow a user to set PreRunTimeoutSeconds",

		func(timeout, expected *int64, succeed bool) {
			nnfProfile.Data.NnfSpec = &NnfPodSpec{
				Containers: []NnfContainer{
					{Name: "test", Image: "alpine:latest", Command: []string{"test"}},
				},
			}
			nnfProfile.Data.NnfMPISpec = nil

			nnfProfile.Data.PreRunTimeoutSeconds = timeout
			if succeed {
				Expect(k8sClient.Create(context.TODO(), nnfProfile)).To(Succeed())
				Expect(nnfProfile.Data.PreRunTimeoutSeconds).To(Equal(expected))
			} else {
				Expect(k8sClient.Create(context.TODO(), nnfProfile)).ToNot(Succeed())
				nnfProfile = nil
			}

		},
		Entry("to 0", pointy.Int64(0), pointy.Int64(0), true),
		Entry("to 45", pointy.Int64(45), pointy.Int64(45), true),
		Entry("to nil and get the default(300)", nil, pointy.Int64(300), true),
		Entry("to -1 and fail", pointy.Int64(-1), nil, false),
	)

	DescribeTable("Should allow a user to set PostRunTimeoutSeconds",

		func(timeout, expected *int64, succeed bool) {
			nnfProfile.Data.NnfSpec = &NnfPodSpec{
				Containers: []NnfContainer{
					{Name: "test", Image: "alpine:latest", Command: []string{"test"}},
				},
			}
			nnfProfile.Data.NnfMPISpec = nil

			nnfProfile.Data.PostRunTimeoutSeconds = timeout
			if succeed {
				Expect(k8sClient.Create(context.TODO(), nnfProfile)).To(Succeed())
				Expect(nnfProfile.Data.PostRunTimeoutSeconds).To(Equal(expected))
			} else {
				Expect(k8sClient.Create(context.TODO(), nnfProfile)).ToNot(Succeed())
				nnfProfile = nil
			}

		},
		Entry("to 0", pointy.Int64(0), pointy.Int64(0), true),
		Entry("to 45", pointy.Int64(45), pointy.Int64(45), true),
		Entry("to nil and get the default(300)", nil, pointy.Int64(300), true),
		Entry("to -1 and fail", pointy.Int64(-1), nil, false),
	)

	It("Should allow a zero postRunTimeoutSeconds", func() {
		nnfProfile.Data.PostRunTimeoutSeconds = pointy.Int64(0)
		Expect(k8sClient.Create(context.TODO(), nnfProfile)).To(Succeed())
	})

	It("Should not allow modification of Data in a pinned resource", func() {
		nnfProfile.ObjectMeta.Name = pinnedResourceName
		nnfProfile.ObjectMeta.Namespace = otherNamespaceName
		nnfProfile.Data.Pinned = true
		Expect(k8sClient.Create(context.TODO(), nnfProfile)).To(Succeed())
		Eventually(func() error {
			return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(nnfProfile), nnfProfile)
		}).Should(Succeed())

		// Verify pinned
		Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(nnfProfile), newProfile)).To(Succeed())
		Expect(newProfile.Data.Pinned).To(BeTrue())

		// Try to update Data and fail
		newProfile.Data.RetryLimit = 10
		Expect(k8sClient.Update(context.TODO(), newProfile)).ToNot(Succeed())
	})

	It("Should allow modification of Meta in a pinned resource", func() {
		nnfProfile.ObjectMeta.Name = pinnedResourceName
		nnfProfile.ObjectMeta.Namespace = otherNamespaceName
		nnfProfile.Data.Pinned = true
		Expect(k8sClient.Create(context.TODO(), nnfProfile)).To(Succeed())
		Eventually(func() error {
			return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(nnfProfile), nnfProfile)
		}).Should(Succeed())

		// Verify pinned
		Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(nnfProfile), newProfile)).To(Succeed())
		Expect(newProfile.Data.Pinned).To(BeTrue())

		// Try to update metadata and succeed. A finalizer or ownerRef will interfere with deletion,
		// so set a label, instead.
		labels := newProfile.GetLabels()
		if labels == nil {
			labels = make(map[string]string)
		}
		labels["profile-label"] = "profile-label"
		newProfile.SetLabels(labels)
		Expect(k8sClient.Update(context.TODO(), newProfile)).To(Succeed())
	})

	It("Should not allow an unpinned profile to become pinned", func() {
		Expect(k8sClient.Create(context.TODO(), nnfProfile)).To(Succeed())
		Eventually(func() error {
			return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(nnfProfile), newProfile)
		}).Should(Succeed())

		newProfile.Data.Pinned = true
		Expect(k8sClient.Update(context.TODO(), newProfile)).ToNot(Succeed())
	})

	It("Should not allow a pinned profile to become unpinned", func() {
		nnfProfile.ObjectMeta.Name = pinnedResourceName
		nnfProfile.ObjectMeta.Namespace = otherNamespaceName
		nnfProfile.Data.Pinned = true

		Expect(k8sClient.Create(context.TODO(), nnfProfile)).To(Succeed())
		Eventually(func() error {
			return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(nnfProfile), newProfile)
		}).Should(Succeed())

		newProfile.Data.Pinned = false
		Expect(k8sClient.Update(context.TODO(), newProfile)).ToNot(Succeed())
	})

	DescribeTable("when modes are set for storages on creation",
		func(storageName string, mode corev1.PersistentVolumeAccessMode, result bool) {
			for i, storage := range nnfProfile.Data.Storages {
				if storage.Name == storageName && mode != "" {
					nnfProfile.Data.Storages[i].PVCMode = mode
				}
			}
			if result {
				Expect(k8sClient.Create(context.TODO(), nnfProfile)).To(Succeed())
			} else {
				Expect(k8sClient.Create(context.TODO(), nnfProfile)).ToNot(Succeed())
				nnfProfile = nil
			}
		},
		// Only nil modes should pass for JOB/PERSISTENT
		Entry("should pass when DW_JOB has no mode", "DW_JOB_storage", corev1.PersistentVolumeAccessMode(""), true),
		Entry("should fail when DW_JOB has a mode", "DW_JOB_storage", corev1.ReadWriteMany, false),
		Entry("should pass when DW_PERSISTENT has no mode", "DW_PERSISTENT_storage", corev1.PersistentVolumeAccessMode(""), true),
		Entry("should fail when DW_PERSISTENT has a mode", "DW_PERSISTENT_storage", corev1.ReadWriteMany, false),
		// Both should pass
		Entry("should pass when DW_GLOBAL has no mode (defaults)", "DW_GLOBAL_storage", corev1.PersistentVolumeAccessMode(""), true),
		Entry("should pass when DW_GLOBAL has a mode", "DW_GLOBAL_storage", corev1.ReadWriteMany, true),
	)

	DescribeTable("MPI Containers with copy offload in the name should have the copy offload flag set",
		func(cName string, copyOffload bool, shouldPass bool) {
			nnfProfile.Data.NnfSpec = nil
			nnfProfile.Data.NnfMPISpec = &NnfMPISpec{
				Launcher: NnfPodSpec{
					Containers: []NnfContainer{
						{Name: cName, Image: "test-image"},
					},
				},
				Worker: NnfPodSpec{
					Containers: []NnfContainer{
						{Name: "test", Image: "test-image"},
					},
				},
				CopyOffload: copyOffload,
			}
			if shouldPass {
				Expect(k8sClient.Create(context.TODO(), nnfProfile)).To(Succeed())
			} else {
				Expect(k8sClient.Create(context.TODO(), nnfProfile)).ToNot(Succeed())
				nnfProfile = nil
			}
		},
		Entry("should pass when the container name suggests copy offload and the flag is set", "copy-offload-server", true, true),
		Entry("should pass when the container name suggests copy offload and the flag is set", "copy-offload", true, true),
		Entry("should pass when the container name suggests copy offload and the flag is set", "copyoffload", true, true),
		Entry("should fail when the container name suggests copy offload and flag is not set", "copy offload", false, false),

		Entry("should pass when the container name does not suggest copy offload and the flag is not set", "offload", false, true),
		Entry("should pass when the container name does not suggest copy offload and the flag is not set", "copy", false, true),
		Entry("should fail when the container name does not suggest copy offload and the flag is set", "sawbill", true, false),
	)

	DescribeTable("non-mpi containers with copy offload in the name should fail",
		func(cName string) {
			nnfProfile.Data.NnfMPISpec = nil
			nnfProfile.Data.NnfSpec = &NnfPodSpec{
				Containers: []NnfContainer{
					{Name: cName, Image: cName, Command: []string{"test"}},
				},
			}

			Expect(k8sClient.Create(context.TODO(), nnfProfile)).ToNot(Succeed())
			nnfProfile = nil
		},
		Entry("when the container name suggests copy offload", "copy-offload-server"),
		Entry("when the container name suggests copy offload", "copy-offload"),
		Entry("when the container name suggests copy offload", "copyoffload"),
		Entry("when the container name suggests copy offload", "copy offload"),
	)

	DescribeTable("MPI Containers validation",
		func(launcherContainers, workerContainers []NnfContainer, shouldPass bool) {
			nnfProfile.Data.NnfSpec = nil
			nnfProfile.Data.NnfMPISpec = &NnfMPISpec{
				Launcher: NnfPodSpec{
					Containers: launcherContainers,
				},
				Worker: NnfPodSpec{
					Containers: workerContainers,
				},
			}
			if shouldPass {
				Expect(k8sClient.Create(context.TODO(), nnfProfile)).To(Succeed())
			} else {
				Expect(k8sClient.Create(context.TODO(), nnfProfile)).ToNot(Succeed())
				nnfProfile = nil
			}
		},
		Entry("should pass when both Launcher and Worker have containers",
			[]NnfContainer{{Name: "launcher", Image: "test-image"}},
			[]NnfContainer{{Name: "worker", Image: "test-image"}},
			true),
		Entry("should fail when Launcher has no containers",
			[]NnfContainer{},
			[]NnfContainer{{Name: "worker", Image: "test-image"}},
			false),
		Entry("should fail when Worker has no containers",
			[]NnfContainer{{Name: "launcher", Image: "test-image"}},
			[]NnfContainer{},
			false),
		Entry("should fail when both Launcher and Worker have no containers",
			[]NnfContainer{},
			[]NnfContainer{},
			false),
	)

	DescribeTable("NnfSpec Containers validation",
		func(containers []NnfContainer, shouldPass bool) {
			nnfProfile.Data.NnfMPISpec = nil
			nnfProfile.Data.NnfSpec = &NnfPodSpec{
				Containers: containers,
			}
			if shouldPass {
				Expect(k8sClient.Create(context.TODO(), nnfProfile)).To(Succeed())
			} else {
				Expect(k8sClient.Create(context.TODO(), nnfProfile)).ToNot(Succeed())
				nnfProfile = nil
			}
		},
		Entry("should pass when NnfSpec has containers",
			[]NnfContainer{{Name: "test", Image: "test-image"}},
			true),
		Entry("should fail when NnfSpec has no containers",
			[]NnfContainer{},
			false),
	)
})
