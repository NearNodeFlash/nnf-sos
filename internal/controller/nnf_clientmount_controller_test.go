/*
 * Copyright 2024 Hewlett Packard Enterprise Development LP
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

	dwsv1alpha2 "github.com/DataWorkflowServices/dws/api/v1alpha2"
)

var _ = Describe("Clientmount Controller Test", func() {

	It("It should correctly create a human-readable lustre mapping for Servers ", func() {
		s := dwsv1alpha2.Servers{
			Status: dwsv1alpha2.ServersStatus{
				AllocationSets: []dwsv1alpha2.ServersStatusAllocationSet{
					{Label: "ost", Storage: map[string]dwsv1alpha2.ServersStatusStorage{
						"rabbit-node-1": dwsv1alpha2.ServersStatusStorage{
							AllocationSize: 123345,
						},
						"rabbit-node-2": dwsv1alpha2.ServersStatusStorage{
							AllocationSize: 123345,
						},
					}},
					{Label: "mdt", Storage: map[string]dwsv1alpha2.ServersStatusStorage{
						"rabbit-node-3": dwsv1alpha2.ServersStatusStorage{
							AllocationSize: 123345,
						},
						"rabbit-node-4": dwsv1alpha2.ServersStatusStorage{
							AllocationSize: 123345,
						},
						"rabbit-node-8": dwsv1alpha2.ServersStatusStorage{
							AllocationSize: 123345,
						},
					}},
				},
			},
		}

		m := createLustreMapping(&s)
		Expect(m).To(HaveLen(2))
		Expect(m["ost"]).To(HaveLen(2))
		Expect(m["ost"]).Should(ContainElements("rabbit-node-1", "rabbit-node-2"))
		Expect(m["mdt"]).To(HaveLen(3))
		Expect(m["mdt"]).Should(ContainElements("rabbit-node-3", "rabbit-node-4", "rabbit-node-8"))
	})
})
