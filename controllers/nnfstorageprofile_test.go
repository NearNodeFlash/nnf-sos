package controllers

import (
	"context"

	. "github.com/onsi/gomega"

	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	nnfv1alpha1 "github.com/NearNodeFlash/nnf-sos/api/v1alpha1"
)

// createNnfStorageProfile creates the given profile in the "default" namespace.
func createNnfStorageProfile(storageProfile *nnfv1alpha1.NnfStorageProfile, expectSuccess bool) *nnfv1alpha1.NnfStorageProfile {
	// Place NnfStorageProfiles in "default" for the test environment.
	storageProfile.ObjectMeta.Namespace = corev1.NamespaceDefault

	profKey := client.ObjectKeyFromObject(storageProfile)
	profExpected := &nnfv1alpha1.NnfStorageProfile{}
	Expect(k8sClient.Get(context.TODO(), profKey, profExpected)).ToNot(Succeed())

	if expectSuccess {
		Expect(k8sClient.Create(context.TODO(), storageProfile)).To(Succeed(), "create nnfstorageprofile")
		//err := k8sClient.Create(context.TODO(), storageProfile)
		Eventually(func(g Gomega) {
			g.Expect(k8sClient.Get(context.TODO(), profKey, profExpected)).To(Succeed())
		}, "3s", "1s").Should(Succeed(), "wait for create of NnfStorageProfile")
	} else {
		Expect(k8sClient.Create(context.TODO(), storageProfile)).ToNot(Succeed(), "expect to fail to create nnfstorageprofile")
		storageProfile = nil
	}

	return storageProfile
}

// basicNnfStorageProfile creates a simple NnfStorageProfile struct.
func basicNnfStorageProfile(name string) *nnfv1alpha1.NnfStorageProfile {
	storageProfile := &nnfv1alpha1.NnfStorageProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	return storageProfile
}

// createBasicDefaultNnfStorageProfile creates a simple default storage profile.
func createBasicDefaultNnfStorageProfile() *nnfv1alpha1.NnfStorageProfile {
	storageProfile := basicNnfStorageProfile("durable-" + uuid.NewString()[:8])
	storageProfile.Data.Default = true
	return createNnfStorageProfile(storageProfile, true)
}

func verifyPinnedProfile(ctx context.Context, clnt client.Client, namespace string, profileName string, expectedOwner metav1.Object) error {

	nnfStorageProfile, err := findPinnedProfile(ctx, clnt, namespace, profileName)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	ExpectWithOffset(1, nnfStorageProfile.Data.Pinned).To(BeTrue())
	refs := nnfStorageProfile.GetOwnerReferences()
	ExpectWithOffset(1, refs).To(HaveLen(1))
	ExpectWithOffset(1, refs[0].UID).To(Equal(expectedOwner.GetUID()))
	return nil
}
