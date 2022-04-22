/*
 * Copyright 2022 Hewlett Packard Enterprise Development LP
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

package v1alpha1

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// These tests are written in BDD-style using Ginkgo framework. Refer to
// http://onsi.github.io/ginkgo to learn more.

var _ = Describe("NnfStorageProfile Webhook", func() {
	var (
		nnfProfile *NnfStorageProfile = nil
	)

	BeforeEach(func() {
		nnfProfile = &NnfStorageProfile{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: metav1.NamespaceDefault,
			},
		}
	})

	AfterEach(func() {
		if nnfProfile != nil {
			Expect(k8sClient.Delete(context.TODO(), nnfProfile)).To(Succeed())
		}
	})

	It("should accept default=true", func() {
		nnfProfile.Data.Default = true
		Expect(k8sClient.Create(context.TODO(), nnfProfile)).To(Succeed())
		Expect(nnfProfile.Data.Default).To(BeTrue())
	})
	It("should accept default=false", func() {
		nnfProfile.Data.Default = false
		Expect(k8sClient.Create(context.TODO(), nnfProfile)).To(Succeed())
		Expect(nnfProfile.Data.Default).ToNot(BeTrue())
	})

	It("should accept externalMgs", func() {
		nnfProfile.Data.LustreStorage = &NnfStorageProfileLustreData{
			ExternalMGS: []string{
				"10.0.0.1@tcp",
			},
		}
		Expect(k8sClient.Create(context.TODO(), nnfProfile)).To(Succeed())
		Expect(nnfProfile.Data.Default).ToNot(BeTrue())
	})

	It("should accept combinedMgtMdt", func() {
		nnfProfile.Data.LustreStorage = &NnfStorageProfileLustreData{
			CombinedMGTMDT: true,
		}
		Expect(k8sClient.Create(context.TODO(), nnfProfile)).To(Succeed())
		Expect(nnfProfile.Data.Default).ToNot(BeTrue())
	})

	It("should not accept combinedMgtMdt with externalMgs", func() {
		nnfProfile.Data.LustreStorage = &NnfStorageProfileLustreData{
			CombinedMGTMDT: true,
			ExternalMGS: []string{
				"10.0.0.1@tcp",
			},
		}
		Expect(k8sClient.Create(context.TODO(), nnfProfile)).ToNot(Succeed())
		nnfProfile = nil
	})

})
