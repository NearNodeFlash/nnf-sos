/*
 * Copyright 2021, 2022 Hewlett Packard Enterprise Development LP
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

// NnfResourceStatus provides common fields that are included in all NNF Resources
type NnfResourceStatus struct {
	// ID reflects the NNF Node unique identifier for this NNF Server resource.
	ID string `json:"id,omitempty"`

	// Name reflects the common name of this NNF Server resource.
	Name string `json:"name,omitempty"`

	Status NnfResourceStatusType `json:"status,omitempty"`

	Health NnfResourceHealthType `json:"health,omitempty"`
}
