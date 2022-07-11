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

//+kubebuilder:object:generate=false
type WorkflowError struct {
	message     string
	recoverable bool
	err         error
}

func NewWorkflowError(message string, recoverable bool) *WorkflowError {
	return &WorkflowError{
		message:     message,
		recoverable: recoverable,
	}
}

func (e *WorkflowError) GetError() error {
	return e.err
}

func (e *WorkflowError) GetMessage() string {
	return e.message
}

func (e *WorkflowError) GetRecoverable() bool {
	return e.recoverable
}

func (e *WorkflowError) Error() string {
	return e.message + ": " + e.err.Error()
}

func (e *WorkflowError) Unwrap() error {
	return e.err
}

func (e *WorkflowError) WithError(err error) *WorkflowError {
	e.err = err
	return e
}
