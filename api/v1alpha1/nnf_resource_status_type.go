/*
 * Copyright 2021-2023 Hewlett Packard Enterprise Development LP
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
	dwsv1alpha2 "github.com/DataWorkflowServices/dws/api/v1alpha2"

	sf "github.com/NearNodeFlash/nnf-ec/pkg/rfsf/pkg/models"
)

// NnfResourceStatusType is the string that indicates the resource's status
type NnfResourceStatusType string

const (
	//
	// Below reflects the current status of a static resource
	//

	// ResourceEnabled means the static NNF resource is enabled and ready to fullfil requests for
	// managed resources.
	ResourceEnabled NnfResourceStatusType = NnfResourceStatusType(sf.ENABLED_RST)

	// ResourceDisabled means the static NNF resource is present but disabled and not available for use
	ResourceDisabled = NnfResourceStatusType(sf.DISABLED_RST)

	// ResourceNotPresent means the static NNF resource is not found; likely because it is disconnected
	// or in a powered down state.
	ResourceNotPresent = "NotPresent"

	// ResourceOffline means the static NNF resource is offline and the NNF Node cannot communicate with
	// the resource. This differs from a NotPresent status in that the device is known to exist.
	ResourceOffline = "Offline"

	//
	// Below reflects the current status of a managed (user created) resource
	//

	// ResourceStarting means the NNF resource is currently in the process of starting - resources
	// are being prepared for transition to an Active state.
	ResourceStarting = NnfResourceStatusType(sf.STARTING_RST)

	// ResourceDeleting means the NNF resource is currently in the process of being deleted - the resource
	// and all child resources are being returned to the NNF node's free resources. Upon a successful
	// deletion, the resource will be removed from the list of managed NNF resources
	ResourceDeleting = "Deleting"

	// ResourceDeleted means the NNF resource was deleted. This reflects the state where the NNF resource does
	// not exist in the NNF space, but the resource might still exist in Kubernetes. A resource in
	// this state suggests that Kubernetes is unable to delete the object.
	ResourceDeleted = "Deleted"

	// ResourceReady means the NNF resource is ready for use.
	ResourceReady = "Ready"

	// ResourceFailed means the NNF resource has failed during startup or execution. A failed state is
	// an unrecoverable condition. Additional information about the Failed cause can be found by
	// looking at the owning resource's Conditions field. A failed resource can only be removed
	// by transition to a Delete state.
	ResourceFailed = "Failed"

	// ResourceInvalid means the NNF resource configuration is invalid due to an improper format or arrangement
	// of listed resource parameters.
	ResourceInvalid = "Invalid"
)

// UpdateIfWorseThan updates the stored status of the resource if the new status is worse than what was stored
func (rst NnfResourceStatusType) UpdateIfWorseThan(status *NnfResourceStatusType) {
	switch rst {
	case ResourceStarting:
		if *status == ResourceReady {
			*status = ResourceStarting
		}
	case ResourceFailed:
		if *status != ResourceFailed {
			*status = ResourceFailed
		}
	default:
	}
}

func (rst NnfResourceStatusType) ConvertToDWSResourceStatus() dwsv1alpha2.ResourceStatus {
	switch rst {
	case ResourceStarting:
		return dwsv1alpha2.StartingStatus
	case ResourceReady:
		return dwsv1alpha2.ReadyStatus
	case ResourceDisabled:
		return dwsv1alpha2.DisabledStatus
	case ResourceNotPresent:
		return dwsv1alpha2.NotPresentStatus
	case ResourceOffline:
		return dwsv1alpha2.OfflineStatus
	case ResourceFailed:
		return dwsv1alpha2.FailedStatus
	default:
		return dwsv1alpha2.UnknownStatus
	}
}

// StaticResourceStatus will convert a Swordfish ResourceStatus to the NNF Resource Status.
func StaticResourceStatus(s sf.ResourceStatus) NnfResourceStatusType {
	switch s.State {
	case sf.STARTING_RST:
		return ResourceStarting
	case sf.ENABLED_RST:
		return ResourceReady
	case sf.DISABLED_RST:
		return ResourceDisabled
	case sf.ABSENT_RST:
		return ResourceNotPresent
	case sf.UNAVAILABLE_OFFLINE_RST:
		return ResourceOffline
	}

	panic("Unknown Resource State " + string(s.State))
}

// ResourceStatus will convert a Swordfish ResourceStatus to the NNF Resource Status.
func ResourceStatus(s sf.ResourceStatus) NnfResourceStatusType {
	switch s.State {
	case sf.STARTING_RST:
		return ResourceStarting
	case sf.ENABLED_RST:
		return ResourceReady
	case sf.DISABLED_RST:
		return ResourceDisabled
	case sf.ABSENT_RST:
		return ResourceNotPresent
	case sf.UNAVAILABLE_OFFLINE_RST:
		return ResourceOffline

	default:
		return ResourceFailed
	}
}
