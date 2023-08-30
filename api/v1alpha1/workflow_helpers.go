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

const (
	// DirectiveIndexLabel is a label applied to child objects of the workflow
	// to show which directive they were created for. This is useful during deletion
	// to filter the child objects by the directive index and only delete the
	// resources for the directive being processed
	DirectiveIndexLabel = "nnf.cray.hpe.com/directive_index"

	// PinnedStorageProfileLabelName is a label applied to NnfStorage objects to show
	// which pinned storage profile is being used.
	PinnedStorageProfileLabelName = "nnf.cray.hpe.com/pinned_storage_profile_name"

	// PinnedStorageProfileLabelNameSpace is a label applied to NnfStorage objects to show
	// which pinned storage profile is being used.
	PinnedStorageProfileLabelNameSpace = "nnf.cray.hpe.com/pinned_storage_profile_namespace"

	// PinnedContainerProfileLabelName is a label applied to NnfStorage objects to show
	// which pinned container profile is being used.
	PinnedContainerProfileLabelName = "nnf.cray.hpe.com/pinned_container_profile_name"

	// PinnedContainerProfileLabelNameSpace is a label applied to NnfStorage objects to show
	// which pinned container profile is being used.
	PinnedContainerProfileLabelNameSpace = "nnf.cray.hpe.com/pinned_container_profile_namespace"

	// StandaloneMGTLabel is a label applied to the PersistentStorageInstance to show that
	// it is for a Lustre MGT only. The value for the label is the pool name.
	StandaloneMGTLabel = "nnf.cray.hpe.com/standalone_mgt"
)
