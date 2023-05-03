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

package v1alpha2

type ResourceErrorInfo struct {
	// Optional user facing message if the error is relevant to an end user
	UserMessage string `json:"userMessage,omitempty"`

	// Internal debug message for the error
	DebugMessage string `json:"debugMessage"`

	// Indication if the error is likely recoverable or not
	Recoverable bool `json:"recoverable"`
}

type ResourceError struct {
	// Error information
	Error *ResourceErrorInfo `json:"error,omitempty"`
}

func NewResourceError(message string, err error) *ResourceErrorInfo {
	resourceError := &ResourceErrorInfo{
		Recoverable: true,
	}

	if err != nil {
		// If the error provided is already a ResourceError, use it and concatenate
		// the debug messages
		_, ok := err.(*ResourceErrorInfo)
		if ok {
			resourceError = err.(*ResourceErrorInfo)
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

func (e *ResourceErrorInfo) WithFatal() *ResourceErrorInfo {
	e.Recoverable = false
	return e
}

func (e *ResourceErrorInfo) WithUserMessage(message string) *ResourceErrorInfo {
	// Only set the user message if it's empty. This prevents upper layers
	// from overriding a user message set by a lower layer
	if e.UserMessage == "" {
		e.UserMessage = message
	}

	return e
}

func (e *ResourceErrorInfo) Error() string {
	return e.DebugMessage
}

func (e *ResourceError) SetResourceError(err error) {
	if err == nil {
		e.Error = nil
	} else {
		e.Error = NewResourceError("", err)
	}
}
