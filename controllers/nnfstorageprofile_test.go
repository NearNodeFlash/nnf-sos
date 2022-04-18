package controllers

import (
	"context"

	. "github.com/onsi/gomega"

	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	nnfv1alpha1 "github.hpe.com/hpe/hpc-rabsw-nnf-sos/api/v1alpha1"
)

// createNnfStorageProfile creates the given profile in the "default" namespace.
func createNnfStorageProfile(storageProfile *nnfv1alpha1.NnfStorageProfile) *nnfv1alpha1.NnfStorageProfile {
	// Place NnfStorageProfiles in "default" for the test environment.
	storageProfile.ObjectMeta.Namespace = corev1.NamespaceDefault

	profKey := client.ObjectKeyFromObject(storageProfile)
	profExpected := &nnfv1alpha1.NnfStorageProfile{}
	Expect(k8sClient.Get(context.TODO(), profKey, profExpected)).ToNot(Succeed())

	Expect(k8sClient.Create(context.TODO(), storageProfile)).To(Succeed(), "create nnfstorageprofile")
	Eventually(func() error {
		return k8sClient.Get(context.TODO(), profKey, profExpected)
	}, "3s", "1s").Should(Succeed(), "wait for create of NnfStorageProfile")

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
	return createNnfStorageProfile(storageProfile)
}
