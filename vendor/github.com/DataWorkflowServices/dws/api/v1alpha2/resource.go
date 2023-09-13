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

package v1alpha2

// ResourceState is the enumeration of the state of a DWS resource
// +kubebuilder:validation:Enum:=Enabled;Disabled
type ResourceState string

const (
	// NOTE: Any new enumeration values must update the related kubebuilder validation

	// Enabled means the resource shall be enabled.
	EnabledState ResourceState = "Enabled"

	// Disabled means the resource shall be disabled.
	DisabledState ResourceState = "Disabled"
)

// ResourceStatus is the enumeration of the status of a DWS resource
// +kubebuilder:validation:Enum:=Starting;Ready;Disabled;NotPresent;Offline;Failed;Degraded;Unknown
type ResourceStatus string

const (
	// NOTE: Any new enumeration values must update the related kubebuilder validation

	// Starting means the resource is currently starting prior to becoming ready.
	StartingStatus ResourceStatus = "Starting"

	// Ready means the resource is fully operational and ready for use.
	ReadyStatus ResourceStatus = "Ready"

	// Disabled means the resource is present but disabled by an administrator or external
	// user.
	DisabledStatus ResourceStatus = "Disabled"

	// NotPresent means the resource is not present within the system, likely because
	// it is missing or powered down. This differs from the Offline state in that the
	// resource is not known to exist.
	NotPresentStatus ResourceStatus = "NotPresent"

	// Offline means the resource is offline and cannot be communicated with. This differs
	// fro the NotPresent state in that the resource is known to exist.
	OfflineStatus ResourceStatus = "Offline"

	// Failed means the resource has failed during startup or execution.
	FailedStatus ResourceStatus = "Failed"

	// Degraded means the resource is ready but operating in a degraded state. Certain
	// recovery actions may alleviate a degraded status.
	DegradedStatus ResourceStatus = "Degraded"

	// Unknown means the resource status is unknown.
	UnknownStatus ResourceStatus = "Unknown"
)
