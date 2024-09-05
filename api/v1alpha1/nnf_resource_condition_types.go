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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Types define the condition type that is recorded by the system. Each storage resource
// defines an array of conditions as state transitions. Entry into and out of the state
// is recorded by the metav1.ConditionStatus. Order must be preserved and consistent between
// the Index and string values.
const (
	ConditionIndexCreateStoragePool = iota
	ConditionIndexDeleteStoragePool
	ConditionIndexCreateStorageGroup
	ConditionIndexCreateFileSystem
	ConditionIndexCreateFileShare
	ConditionIndexGetResource
	ConditionIndexInvalidResource
	// INSERT NEW ITEMS HERE - Ensure Condition string is at same index

	numConditions

	ConditionCreateStoragePool  = "CreateStoragePool"
	ConditionDeleteStoragePool  = "DeleteStoragePool"
	ConditionCreateStorageGroup = "CreateStorageGroup"
	ConditionCreateFileSystem   = "CreateFileSystem"
	ConditionCreateFileShare    = "CreateFileShare"
	ConditionGetResource        = "GetResource"
	ConditionInvalidResource    = "InvalidResource"
	// INSERT NEW ITEMS HERE - Ensure NewConditions() is updated to contain item and correct ordering
)

// NewConditions generates a new conditions array for NNFNodeStorage
func NewConditions() []metav1.Condition {

	types := []string{
		ConditionCreateStoragePool,
		ConditionDeleteStoragePool,
		ConditionCreateStorageGroup,
		ConditionCreateFileSystem,
		ConditionCreateFileShare,
		ConditionGetResource,
		ConditionInvalidResource,
	}

	if numConditions != len(types) {
		panic("Did you forget to include the condition in the types array?")
	}

	c := make([]metav1.Condition, len(types))
	for idx := range c {
		c[idx] = metav1.Condition{
			Type:               types[idx],
			Status:             metav1.ConditionUnknown,
			Reason:             ConditionUnknown,
			LastTransitionTime: metav1.Now(),
		}
	}

	c[ConditionIndexCreateStoragePool].Status = metav1.ConditionTrue
	c[ConditionIndexCreateStoragePool].LastTransitionTime = metav1.Now()

	return c

}

// SetGetResourceFailureCondition sets/gets the specified condition to failed
func SetGetResourceFailureCondition(c []metav1.Condition, err error) {
	c[ConditionIndexGetResource] = metav1.Condition{
		Type:               ConditionGetResource,
		Reason:             ConditionFailed,
		Status:             metav1.ConditionTrue,
		Message:            err.Error(),
		LastTransitionTime: metav1.Now(),
	}
}

// SetResourceInvalidCondition sets/gets the specified condition to invalid
func SetResourceInvalidCondition(c []metav1.Condition, err error) {
	c[ConditionIndexInvalidResource] = metav1.Condition{
		Type:               ConditionInvalidResource,
		Reason:             ConditionInvalid,
		Status:             metav1.ConditionTrue,
		Message:            err.Error(),
		LastTransitionTime: metav1.Now(),
	}
}

// Reason implements the Reason field of a metav1.Condition. In accordance with the metav1.Condition,
// the value should be a CamelCase string and may not be empty.
const (
	ConditionUnknown = "Unknown"
	ConditionFailed  = "Failed"
	ConditionInvalid = "Invalid"
	ConditionSuccess = "Success"
)
