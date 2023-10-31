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
package controller

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	dwsv1alpha2 "github.com/DataWorkflowServices/dws/api/v1alpha2"
)

var _ = Describe("NnfSystemconfigurationController", func() {
	var sysCfg *dwsv1alpha2.SystemConfiguration

	AfterEach(func() {
		Expect(k8sClient.Delete(context.TODO(), sysCfg)).To(Succeed())
		Eventually(func() error { // Delete can still return the cached object. Wait until the object is no longer present
			return k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(sysCfg), sysCfg)
		}).ShouldNot(Succeed())
	})

	When("creating a SystemConfiguration", func() {
		sysCfg = &dwsv1alpha2.SystemConfiguration{
			ObjectMeta: v1.ObjectMeta{
				Name:      "test",
				Namespace: corev1.NamespaceDefault,
			},
			Spec: dwsv1alpha2.SystemConfigurationSpec{
				ComputeNodes: []dwsv1alpha2.SystemConfigurationComputeNode{
					{Name: "test-compute-0"},
				},
			},
		}

		It("should go ready", func() {
			Expect(k8sClient.Create(context.TODO(), sysCfg)).To(Succeed())
			Eventually(func(g Gomega) bool {
				g.Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(sysCfg), sysCfg)).To(Succeed())
				return sysCfg.Status.Ready
			}).Should(BeFalse())
		})
	})
})
