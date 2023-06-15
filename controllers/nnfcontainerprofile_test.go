/*
 * Copyright 2023 Hewlett Packard Enterprise Development LP
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

	. "github.com/onsi/gomega"

	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	dwsv1alpha2 "github.com/HewlettPackard/dws/api/v1alpha2"
	nnfv1alpha1 "github.com/NearNodeFlash/nnf-sos/api/v1alpha1"
)

// createNnfContainerProfile creates the given profile in the "default" namespace.
func createNnfContainerProfile(containerProfile *nnfv1alpha1.NnfContainerProfile, expectSuccess bool) *nnfv1alpha1.NnfContainerProfile {
	// Place NnfContainerProfiles in "default" for the test environment.
	containerProfile.ObjectMeta.Namespace = corev1.NamespaceDefault

	profKey := client.ObjectKeyFromObject(containerProfile)
	profExpected := &nnfv1alpha1.NnfContainerProfile{}
	Expect(k8sClient.Get(context.TODO(), profKey, profExpected)).ToNot(Succeed())

	if expectSuccess {
		Expect(k8sClient.Create(context.TODO(), containerProfile)).To(Succeed(), "create nnfcontainerprofile")
		//err := k8sClient.Create(context.TODO(), containerProfile)
		Eventually(func(g Gomega) {
			g.Expect(k8sClient.Get(context.TODO(), profKey, profExpected)).To(Succeed())
		}, "3s", "1s").Should(Succeed(), "wait for create of NnfContainerProfile")
	} else {
		Expect(k8sClient.Create(context.TODO(), containerProfile)).ToNot(Succeed(), "expect to fail to create nnfcontainerprofile")
		containerProfile = nil
	}

	return containerProfile
}

// basicNnfContainerProfile creates a simple NnfContainerProfile struct.
func basicNnfContainerProfile(name string, storages []nnfv1alpha1.NnfContainerProfileStorage) *nnfv1alpha1.NnfContainerProfile {

	// default storages if not supplied, optional by default
	if len(storages) == 0 {
		storages = []nnfv1alpha1.NnfContainerProfileStorage{
			{Name: "DW_JOB_foo_local_storage", Optional: true},
			{Name: "DW_PERSISTENT_foo_persistent_storage", Optional: true},
		}
	}

	containerProfile := &nnfv1alpha1.NnfContainerProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Data: nnfv1alpha1.NnfContainerProfileData{
			Pinned:   false,
			Storages: storages,
			Spec: &corev1.PodSpec{
				Containers: []corev1.Container{
					{Name: "test", Image: "alpine:latest", Command: []string{"true"}},
				},
			},
		},
	}

	return containerProfile
}

// createBasicNnfContainerProfile creates a simple default container profile.
func createBasicNnfContainerProfile(storages []nnfv1alpha1.NnfContainerProfileStorage) *nnfv1alpha1.NnfContainerProfile {
	containerProfile := basicNnfContainerProfile("sample-"+uuid.NewString()[:8], storages)
	return createNnfContainerProfile(containerProfile, true)
}

func verifyPinnedContainerProfile(ctx context.Context, clnt client.Client, workflow *dwsv1alpha2.Workflow, index int) error {

	nnfContainerProfile, err := findPinnedContainerProfile(ctx, clnt, workflow, index)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	ExpectWithOffset(1, nnfContainerProfile.Data.Pinned).To(BeTrue())
	refs := nnfContainerProfile.GetOwnerReferences()
	ExpectWithOffset(1, refs).To(HaveLen(1))
	return nil
}
