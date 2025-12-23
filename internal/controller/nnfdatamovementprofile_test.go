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

	nnfv1alpha10 "github.com/NearNodeFlash/nnf-sos/api/v1alpha10"
)

// createNnfDataMovementProfile creates the given profile in the "default" namespace.
// When expectSuccess=false, we expect to find that it was failed by the webhook.
func createNnfDataMovementProfile(DataMovementProfile *nnfv1alpha10.NnfDataMovementProfile, expectSuccess bool) *nnfv1alpha10.NnfDataMovementProfile {
	// Place NnfDataMovementProfiles in "default" for the test environment.
	DataMovementProfile.ObjectMeta.Namespace = corev1.NamespaceDefault

	profKey := client.ObjectKeyFromObject(DataMovementProfile)
	profExpected := &nnfv1alpha10.NnfDataMovementProfile{}
	err := k8sClient.Get(context.TODO(), profKey, profExpected)
	Expect(err).ToNot(BeNil())
	Expect(apierrors.IsNotFound(err)).To(BeTrue())

	if expectSuccess {
		Expect(k8sClient.Create(context.TODO(), DataMovementProfile)).To(Succeed(), "create nnfDataMovementProfile")
		Eventually(func(g Gomega) {
			g.Expect(k8sClient.Get(context.TODO(), profKey, profExpected)).To(Succeed())
		}, "3s", "1s").Should(Succeed(), "wait for create of NnfDataMovementProfile")
	} else {
		err = k8sClient.Create(context.TODO(), DataMovementProfile)
		Expect(err).ToNot(BeNil())
		Expect(err.Error()).To(MatchRegexp("webhook .* denied the request"))
		DataMovementProfile = nil
	}

	return DataMovementProfile
}

// basicNnfDataMovementProfile creates a simple NnfDataMovementProfile struct.
func basicNnfDataMovementProfile(name string) *nnfv1alpha10.NnfDataMovementProfile {
	DataMovementProfile := &nnfv1alpha10.NnfDataMovementProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	return DataMovementProfile
}

// createBasicDefaultNnfDataMovementProfile creates a simple default storage profile.
func createBasicDefaultNnfDataMovementProfile() *nnfv1alpha10.NnfDataMovementProfile {
	DataMovementProfile := basicNnfDataMovementProfile("durable-" + uuid.NewString()[:8])
	DataMovementProfile.Data.Default = true
	return createNnfDataMovementProfile(DataMovementProfile, true)
}

// createBasicDefaultNnfDataMovementProfile creates a simple default storage profile.
func createBasicPinnedNnfDataMovementProfile() *nnfv1alpha10.NnfDataMovementProfile {
	DataMovementProfile := basicNnfDataMovementProfile("durable-" + uuid.NewString()[:8])
	DataMovementProfile.Data.Pinned = true
	return createNnfDataMovementProfile(DataMovementProfile, true)
}

func verifyPinnedDMProfile(ctx context.Context, clnt client.Client, namespace string, profileName string) error {
	nnfDataMovementProfile, err := findPinnedDMProfile(ctx, clnt, namespace, profileName)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	ExpectWithOffset(1, nnfDataMovementProfile.Data.Pinned).To(BeTrue())
	refs := nnfDataMovementProfile.GetOwnerReferences()
	ExpectWithOffset(1, refs).To(HaveLen(1))
	return nil
}
