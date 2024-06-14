/*
 * Copyright 2022-2023 Hewlett Packard Enterprise Development LP
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

import (
	"fmt"
	"strings"

	"github.com/go-logr/logr"
)

type ResourceErrorSeverity string
type ResourceErrorType string

const (
	// Minor errors are very likely to eventually succeed (e.g., errors caused by a stale cache)
	// The WLM doesn't see these errors directly. The workflow stays in the DriverWait state, and
	// the error string is put in workflow.Status.Message.
	SeverityMinor ResourceErrorSeverity = "Minor"

	// Major errors may or may not succeed. These are transient errors that could be persistent
	// due to an underlying problem (e.g., errors from OS calls)
	SeverityMajor ResourceErrorSeverity = "Major"

	// Fatal errors will never succeed. This is for situations where we can guarantee that retrying
	// will not fix the error (e.g., a DW directive that is not valid)
	SeverityFatal ResourceErrorSeverity = "Fatal"
)

const (
	// Internal errors are due to an error in the DWS/driver code
	TypeInternal ResourceErrorType = "Internal"

	// WLM errors are due to an error with the input from the WLM
	TypeWLM ResourceErrorType = "WLM"

	// User errors are due to an error with the input from a user
	TypeUser ResourceErrorType = "User"
)

type ResourceErrorInfo struct {
	// Optional user facing message if the error is relevant to an end user
	UserMessage string `json:"userMessage,omitempty"`

	// Internal debug message for the error
	DebugMessage string `json:"debugMessage"`

	// Internal or user error
	// +kubebuilder:validation:Enum=Internal;User;WLM
	Type ResourceErrorType `json:"type"`

	// Indication of how severe the error is. Minor will likely succeed, Major may
	// succeed, and Fatal will never succeed.
	// +kubebuilder:validation:Enum=Minor;Major;Fatal
	Severity ResourceErrorSeverity `json:"severity"`
}

type ResourceError struct {
	// Error information
	Error *ResourceErrorInfo `json:"error,omitempty"`
}

func NewResourceError(format string, a ...any) *ResourceErrorInfo {
	return &ResourceErrorInfo{
		Type:         TypeInternal,
		Severity:     SeverityMinor,
		DebugMessage: fmt.Sprintf(format, a...),
	}
}

// A resource error can have an optional user message that is displayed in the workflow.Status.Message
// field. The user message of the lowest level error is all that's displayed.
func (e *ResourceErrorInfo) WithUserMessage(format string, a ...any) *ResourceErrorInfo {
	// Only set the user message if it's empty. This prevents upper layers
	// from overriding a user message set by a lower layer
	if e.UserMessage == "" {
		e.UserMessage = fmt.Sprintf(format, a...)
	}

	return e
}

func (e *ResourceErrorInfo) WithError(err error) *ResourceErrorInfo {
	if err == nil {
		return e
	}

	// Concatenate the parent and child debug messages
	debugMessageList := []string{}
	if e.DebugMessage != "" {
		debugMessageList = append(debugMessageList, e.DebugMessage)
	}

	childError, ok := err.(*ResourceErrorInfo)
	if ok {
		// Inherit the severity and the user message if the child error is a ResourceError
		e.Severity = childError.Severity
		e.UserMessage = childError.UserMessage
		e.Type = childError.Type

		// If the child resource error doesn't have a debug message, use the user message instead
		if childError.DebugMessage == "" {
			debugMessageList = append(debugMessageList, childError.UserMessage)
		} else {
			debugMessageList = append(debugMessageList, childError.DebugMessage)
		}
	} else {
		debugMessageList = append(debugMessageList, err.Error())
	}

	e.DebugMessage = strings.Join(debugMessageList, ": ")

	return e
}

func (e *ResourceErrorInfo) WithFatal() *ResourceErrorInfo {
	e.Severity = SeverityFatal
	return e
}

func (e *ResourceErrorInfo) WithMajor() *ResourceErrorInfo {
	if e.Severity != SeverityFatal {
		e.Severity = SeverityMajor
	}
	return e
}

func (e *ResourceErrorInfo) WithMinor() *ResourceErrorInfo {
	if e.Severity != SeverityFatal && e.Severity != SeverityMajor {
		e.Severity = SeverityMinor
	}
	return e
}

func (e *ResourceErrorInfo) WithInternal() *ResourceErrorInfo {
	e.Type = TypeInternal
	return e
}

func (e *ResourceErrorInfo) WithWLM() *ResourceErrorInfo {
	e.Type = TypeWLM
	return e
}

func (e *ResourceErrorInfo) WithUser() *ResourceErrorInfo {
	e.Type = TypeUser
	return e
}

func (e *ResourceErrorInfo) Error() string {
	message := ""
	if e.DebugMessage == "" {
		message = e.UserMessage
	} else {
		message = e.DebugMessage
	}
	return fmt.Sprintf("%s error: %s", strings.ToLower(string(e.Type)), message)
}

func (e *ResourceErrorInfo) GetUserMessage() string {
	return fmt.Sprintf("%s error: %s", string(e.Type), e.UserMessage)
}

func (e *ResourceError) SetResourceErrorAndLog(err error, log logr.Logger) {
	e.SetResourceError(err)
	if err == nil {
		return
	}

	childError, ok := err.(*ResourceErrorInfo)
	if ok {
		if childError.Severity == SeverityFatal {
			log.Error(err, "Fatal error")
			return
		}

		log.Info("Recoverable Error", "Severity", childError.Severity, "Message", err.Error())
		return
	}

	log.Info("Recoverable Error", "Message", err.Error())
}

func (e *ResourceError) SetResourceError(err error) {
	if err == nil {
		e.Error = nil
	} else {
		e.Error = NewResourceError("").WithError(err)
	}
}
