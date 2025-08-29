/*
 * Copyright 2021-2025 Hewlett Packard Enterprise Development LP
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

package v1alpha8

import (
	sf "github.com/NearNodeFlash/nnf-ec/pkg/rfsf/pkg/models"
)

// NnfResourceHealthType defines the health of an NNF resource.
type NnfResourceHealthType string

const (
	// ResourceOkay is SF health OK
	ResourceOkay NnfResourceHealthType = NnfResourceHealthType(sf.OK_RH)

	// ResourceWarning is SF health WARNING
	ResourceWarning = NnfResourceHealthType(sf.WARNING_RH)

	// ResourceCritical is SF health CRITICAL
	ResourceCritical = NnfResourceHealthType(sf.CRITICAL_RH)
)

// ResourceHealth maps a SF ResourceStatus to an NNFResourceHealthType
func ResourceHealth(s sf.ResourceStatus) NnfResourceHealthType {
	switch s.Health {
	case sf.OK_RH:
		return ResourceOkay
	case sf.WARNING_RH:
		return ResourceWarning
	case sf.CRITICAL_RH:
		return ResourceCritical
	}

	panic("Unknown Resource Health " + string(s.Health))
}

// UpdateIfWorseThan examines the input health type and update the health if it is worse
// than the stored value
func (rht NnfResourceHealthType) UpdateIfWorseThan(health *NnfResourceHealthType) {
	switch rht {
	case ResourceWarning:
		if *health == ResourceOkay {
			*health = ResourceWarning
		}
	case ResourceCritical:
		if *health != ResourceCritical {
			*health = ResourceCritical
		}
	default:
	}
}
