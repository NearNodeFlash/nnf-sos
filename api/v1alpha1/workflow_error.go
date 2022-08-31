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

import (
	dwsv1alpha1 "github.com/HewlettPackard/dws/api/v1alpha1"
)

// +kubebuilder:object:generate=false
type WorkflowError struct {
	message     string
	recoverable bool
	err         error
}

func NewWorkflowError(message string) *WorkflowError {
	return &WorkflowError{
		message:     message,
		recoverable: true,
	}
}

func (e *WorkflowError) GetMessage() string {
	return e.message
}

func (e *WorkflowError) GetRecoverable() bool {
	return e.recoverable
}

func (e *WorkflowError) GetError() error {
	return e.err
}

func (e *WorkflowError) Error() string {
	if e.err == nil {
		return e.message
	}

	return e.message + ": " + e.err.Error()
}

func (e *WorkflowError) Unwrap() error {
	return e.err
}

func (e *WorkflowError) Inject(driverStatus *dwsv1alpha1.WorkflowDriverStatus) {
	driverStatus.Message = e.GetMessage()
	if e.GetRecoverable() {
		driverStatus.Status = dwsv1alpha1.StatusRunning
	} else {
		driverStatus.Status = dwsv1alpha1.StatusError
	}

	if e.Unwrap() != nil {
		driverStatus.Error = e.Unwrap().Error()
	} else {
		driverStatus.Error = e.Error()
	}
}

func (e *WorkflowError) WithFatal() *WorkflowError {
	e.recoverable = false
	return e
}

func (e *WorkflowError) WithError(err error) *WorkflowError {
	// if the error is already a WorkflowError, then return it unmodified
	workflowError, ok := err.(*WorkflowError)
	if ok {
		return workflowError
	}

	resourceError, ok := err.(*dwsv1alpha1.ResourceError)
	if ok {
		e.message = resourceError.UserMessage
		e.recoverable = resourceError.Recoverable
	}

	e.err = err
	return e
}
