/*
 * Copyright 2024-2025 Hewlett Packard Enterprise Development LP
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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	nnfv1alpha6 "github.com/NearNodeFlash/nnf-sos/api/v1alpha6"
)

var _ = Describe("NNFStorage Controller Test", func() {

	It("It should correctly create a human-readable lustre mapping for NnfStorage", func() {
		s := nnfv1alpha6.NnfStorage{
			Spec: nnfv1alpha6.NnfStorageSpec{
				AllocationSets: []nnfv1alpha6.NnfStorageAllocationSetSpec{
					{Name: "ost", Nodes: []nnfv1alpha6.NnfStorageAllocationNodes{
						{Name: "rabbit-node-1", Count: 2},
						{Name: "rabbit-node-2", Count: 1}},
					},
					// throw another OST on rabbit-node-2
					{Name: "ost", Nodes: []nnfv1alpha6.NnfStorageAllocationNodes{
						{Name: "rabbit-node-2", Count: 1}},
					},
					{Name: "mdt", Nodes: []nnfv1alpha6.NnfStorageAllocationNodes{
						{Name: "rabbit-node-3", Count: 1},
						{Name: "rabbit-node-4", Count: 1},
						{Name: "rabbit-node-8", Count: 1}},
					},
					{Name: "mgt", Nodes: []nnfv1alpha6.NnfStorageAllocationNodes{
						{Name: "rabbit-node-3", Count: 1}},
					},
					{Name: "mgtmdt", Nodes: []nnfv1alpha6.NnfStorageAllocationNodes{
						{Name: "rabbit-node-4", Count: 1}},
					},
				},
			},
		}

		Expect(s.Spec.AllocationSets).To(HaveLen(5))
		m := getLustreMappingFromStorage(&s)
		Expect(m).To(HaveLen(5)) // should have keys for 4 lustre components (i.e. ost, mdt, mgt, mgtmdt) + rabbits

		Expect(m["ost"]).To(HaveLen(4))
		Expect(m["ost"]).Should(ContainElements("rabbit-node-1", "rabbit-node-1", "rabbit-node-2", "rabbit-node-2"))

		Expect(m["mdt"]).To(HaveLen(3))
		Expect(m["mdt"]).Should(ContainElements("rabbit-node-3", "rabbit-node-4", "rabbit-node-8"))

		Expect(m["mgt"]).To(HaveLen(1))
		Expect(m["mgt"]).Should(ContainElements("rabbit-node-3"))

		Expect(m["mgtmdt"]).To(HaveLen(1))
		Expect(m["mgtmdt"]).Should(ContainElements("rabbit-node-4"))

		Expect(m["nnfNode"]).To(HaveLen(5))
		Expect(m["nnfNode"]).Should(ContainElements("rabbit-node-1", "rabbit-node-2", "rabbit-node-3", "rabbit-node-4", "rabbit-node-8"))
	})
})
