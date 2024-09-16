/*
 * Copyright 2021-2024 Hewlett Packard Enterprise Development LP
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

package v1alpha2

// NnfResourceStateType defines valid states that a user can configure an NNF resource
type NnfResourceStateType string

const (
	//
	// Below reflects the current status of a static resource
	//

	// ResourceEnable means this static NNF resource should be enabled.
	ResourceEnable NnfResourceStateType = "Enable"

	// ResourceDisable means this static NNF resource should be disabled. Not all static resources can be disabled.
	ResourceDisable = "Disable"

	//
	// Below reflects the current status of a managed (user created) resource
	//

	// ResourceCreate means the resource should be created and enabled for operation. For a newly
	// created resource, the default state is create.
	ResourceCreate NnfResourceStateType = "Create"

	// ResourceDestroy means the resource should be released from the allocated resource pool, and
	// this resource and all child resources will be released to the free resource pools
	// managed by the system.
	ResourceDestroy = "Destroy"
)
