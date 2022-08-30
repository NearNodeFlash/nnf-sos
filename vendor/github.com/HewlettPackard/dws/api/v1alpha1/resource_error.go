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

type ResourceError struct {
	// Optional user facing message if the error is relevant to an end user
	UserMessage string `json:"userMessage,omitempty"`

	// Internal debug message for the error
	DebugMessage string `json:"debugMessage"`

	// Indication if the error is likely recoverable or not
	// +kubebuilder:validation:Default=true
	Recoverable bool `json:"recoverable"`
}

func NewResourceError(message string, err error) *ResourceError {
	resourceError := &ResourceError{
		Recoverable: true,
	}

	if err != nil {
		// If the error provided is already a ResourceError, use it and concatenate
		// the debug messages
		_, ok := err.(*ResourceError)
		if ok {
			resourceError = err.(*ResourceError)
		}

		if message == "" {
			message = err.Error()
		} else {
			message = message + ": " + err.Error()
		}
	}

	resourceError.DebugMessage = message

	return resourceError
}

func (e *ResourceError) WithFatal() *ResourceError {
	e.Recoverable = false
	return e
}

func (e *ResourceError) WithUserMessage(message string) *ResourceError {
	// Only set the user message if it's empty. This prevents upper layers
	// from overriding a user message set by a lower layer
	if e.UserMessage == "" {
		e.UserMessage = message
	}

	return e
}

func (e *ResourceError) Error() string {
	return e.DebugMessage
}
