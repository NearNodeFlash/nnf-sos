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

package v1alpha5

import (
	. "github.com/onsi/ginkgo/v2"
)

var _ = Describe("NnfAccess Webhook", func() {

	// We already have api/<spoke_ver>/conversion_test.go that is
	// digging deep into the conversion routines, and we have
	// internal/controllers/conversion_test.go that is verifying that the
	// conversion webhook is hooked up to those routines.

	Context("When creating NnfAccess under Conversion Webhook", func() {
		It("Should get the converted version of NnfAccess", func() {

			// TODO(user): Add your logic here

		})
	})

})
