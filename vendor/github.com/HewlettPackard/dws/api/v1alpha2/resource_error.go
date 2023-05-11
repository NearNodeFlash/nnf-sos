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

import (
	"fmt"
	"strings"

	"github.com/go-logr/logr"
)

type ResourceErrorSeverity string
type ResourceErrorType string

const (
	SeverityMinor ResourceErrorSeverity = "Minor"
	SeverityMajor ResourceErrorSeverity = "Major"
	SeverityFatal ResourceErrorSeverity = "Fatal"
)

const (
	TypeInternal ResourceErrorType = "Internal"
	TypeUser     ResourceErrorType = "User"
)

type ResourceErrorInfo struct {
	// Optional user facing message if the error is relevant to an end user
	UserMessage string `json:"userMessage,omitempty"`

	// Internal debug message for the error
	DebugMessage string `json:"debugMessage"`

	// Internal or user error
	// +kubebuilder:validation:Enum=Internal;User
	Type ResourceErrorType `json:"type"`

	// Indication if the error is likely recoverable or not
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

func (e *ResourceErrorInfo) WithUserMessage(format string, a ...any) *ResourceErrorInfo {
	// Only set the user message if it's empty. This prevents upper layers
	// from overriding a user message set by a lower layer
	if e.UserMessage == "" {
		e.UserMessage = fmt.Sprintf(format, a...)
	}

	return e
}

func (e *ResourceErrorInfo) WithError(err error) *ResourceErrorInfo {
	debugMessageList := []string{}

	childError, ok := err.(*ResourceErrorInfo)
	if ok {
		// Inherit the severity and the user message if the child error is a ResourceError
		e.Severity = childError.Severity
		e.UserMessage = childError.UserMessage
		debugMessageList = append(debugMessageList, childError.DebugMessage)
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
	e.Severity = SeverityMajor
	return e
}

func (e *ResourceErrorInfo) WithMinor() *ResourceErrorInfo {
	e.Severity = SeverityMinor
	return e
}

func (e *ResourceErrorInfo) WithInternal() *ResourceErrorInfo {
	e.Type = TypeInternal
	return e
}

func (e *ResourceErrorInfo) WithUser() *ResourceErrorInfo {
	e.Type = TypeUser
	return e
}

func (e *ResourceErrorInfo) Error() string {
	return fmt.Sprintf("%s error: %s", strings.ToLower(string(e.Type)), e.DebugMessage)
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
