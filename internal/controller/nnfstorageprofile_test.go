/*
 * Copyright 2022-2025 Hewlett Packard Enterprise Development LP
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

	. "github.com/onsi/gomega"

	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	nnfv1alpha9 "github.com/NearNodeFlash/nnf-sos/api/v1alpha9"
)

// createNnfStorageProfile creates the given profile in the "default" namespace.
// When expectSuccess=false, we expect to find that it was failed by the webhook.
func createNnfStorageProfile(storageProfile *nnfv1alpha9.NnfStorageProfile, expectSuccess bool) *nnfv1alpha9.NnfStorageProfile {
	// Place NnfStorageProfiles in "default" for the test environment.
	storageProfile.ObjectMeta.Namespace = corev1.NamespaceDefault

	profKey := client.ObjectKeyFromObject(storageProfile)
	profExpected := &nnfv1alpha9.NnfStorageProfile{}
	err := k8sClient.Get(context.TODO(), profKey, profExpected)
	Expect(err).ToNot(BeNil())
	Expect(apierrors.IsNotFound(err)).To(BeTrue())

	if expectSuccess {
		Expect(k8sClient.Create(context.TODO(), storageProfile)).To(Succeed(), "create nnfstorageprofile")
		Eventually(func(g Gomega) {
			g.Expect(k8sClient.Get(context.TODO(), profKey, profExpected)).To(Succeed())
		}, "3s", "1s").Should(Succeed(), "wait for create of NnfStorageProfile")
	} else {
		err = k8sClient.Create(context.TODO(), storageProfile)
		Expect(err).ToNot(BeNil())
		Expect(err.Error()).To(MatchRegexp("webhook .* denied the request"))
		storageProfile = nil
	}

	return storageProfile
}

// basicNnfStorageProfile creates a simple NnfStorageProfile struct.
func basicNnfStorageProfile(name string) *nnfv1alpha9.NnfStorageProfile {
	storageProfile := &nnfv1alpha9.NnfStorageProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	return storageProfile
}

// createBasicDefaultNnfStorageProfile creates a simple default storage profile.
func createBasicDefaultNnfStorageProfile() *nnfv1alpha9.NnfStorageProfile {
	storageProfile := basicNnfStorageProfile("durable-" + uuid.NewString()[:8])
	storageProfile.Data.Default = true
	return createNnfStorageProfile(storageProfile, true)
}

// createBasicDefaultNnfStorageProfile creates a simple default storage profile.
func createBasicPinnedNnfStorageProfile() *nnfv1alpha9.NnfStorageProfile {
	storageProfile := basicNnfStorageProfile("durable-" + uuid.NewString()[:8])
	storageProfile.Data.Pinned = true
	return createNnfStorageProfile(storageProfile, true)
}

func verifyPinnedProfile(ctx context.Context, clnt client.Client, namespace string, profileName string) error {

	nnfStorageProfile, err := findPinnedProfile(ctx, clnt, namespace, profileName)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	ExpectWithOffset(1, nnfStorageProfile.Data.Pinned).To(BeTrue())
	refs := nnfStorageProfile.GetOwnerReferences()
	ExpectWithOffset(1, refs).To(HaveLen(1))
	return nil
}
