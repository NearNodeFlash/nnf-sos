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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	nnf "github.com/NearNodeFlash/nnf-ec/pkg"
	nnfv1alpha5 "github.com/NearNodeFlash/nnf-sos/api/v1alpha5"
)

var _ = PDescribe("NNF Node Storage Controller Test", func() {
	var (
		key     types.NamespacedName
		storage *nnfv1alpha5.NnfNodeStorage
	)

	BeforeEach(func() {
		enablePersistence := false
		c := nnf.NewController(nnf.NewMockOptions(enablePersistence))
		Expect(c.Init(nil)).To(Succeed())

		// TODO: Eventually the controller should go ready; convert this to a poll once the API is available
		time.Sleep(1 * time.Second)
	})

	BeforeEach(func() {
		key = types.NamespacedName{
			Name:      "nnf-node-storage",
			Namespace: corev1.NamespaceDefault,
		}

		storage = &nnfv1alpha5.NnfNodeStorage{
			ObjectMeta: metav1.ObjectMeta{
				Name:      key.Name,
				Namespace: key.Namespace,
			},
			Spec: nnfv1alpha5.NnfNodeStorageSpec{
				Count: 1,
			},
		}
	})

	JustBeforeEach(func() {
		Expect(k8sClient.Create(context.TODO(), storage)).To(Succeed())

		Eventually(func() error {
			expected := &nnfv1alpha5.NnfNodeStorage{}
			return k8sClient.Get(context.TODO(), key, expected)
		}, "3s", "1s").Should(Succeed(), "expected return after create. key: "+key.String())
	})

	AfterEach(func() {
		expected := &nnfv1alpha5.NnfNodeStorage{}
		Expect(k8sClient.Get(context.TODO(), key, expected)).To(Succeed())
		Expect(k8sClient.Delete(context.TODO(), expected)).To(Succeed())
	})

	When("creating xfs storage", func() {
		BeforeEach(func() {
			storage.Spec.FileSystemType = "xfs"
		})

		It("is successful", func() {
			expected := &nnfv1alpha5.NnfNodeStorage{}
			Expect(k8sClient.Get(context.TODO(), key, expected)).To(Succeed())
		})
	})

	When("creating lustre storage", func() {
		BeforeEach(func() {
			storage.Spec.FileSystemType = "lustre"

			storage.Spec.LustreStorage = nnfv1alpha5.LustreStorageSpec{
				FileSystemName: "test",
				StartIndex:     0,
				MgsAddress:     "test",
				TargetType:     "mgt",
				BackFs:         "zfs",
			}
		})

		It("is successful", func() {
			expected := &nnfv1alpha5.NnfNodeStorage{}
			Expect(k8sClient.Get(context.TODO(), key, expected)).To(Succeed())
		})
	})
})
