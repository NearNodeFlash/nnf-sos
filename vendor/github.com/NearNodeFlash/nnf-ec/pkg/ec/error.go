/*
 * Copyright 2020, 2021, 2022 Hewlett Packard Enterprise Development LP
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

package ec

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type ControllerError struct {
	statusCode   int
	cause        string
	resourceType string
	err          error
	Event        interface{}
}

func NewControllerError(sc int) *ControllerError {
	return &ControllerError{statusCode: sc}
}

func (e *ControllerError) Error() string {
	errorString := fmt.Sprintf("Error %d: %s", e.statusCode, http.StatusText(e.statusCode))
	if len(e.cause) != 0 {
		errorString += fmt.Sprintf(", Cause: %s", e.cause)
	}
	if e.err != nil {
		errorString += fmt.Sprintf(", Internal Error: %s", e.err)
	}
	return errorString
}

func (e *ControllerError) Unwrap() error {
	return e.err
}

// Getters

func (e *ControllerError) StatusCode() int {
	return e.statusCode
}

func (e *ControllerError) Cause() string {
	return e.cause
}

func (e *ControllerError) ResourceType() string {
	return e.resourceType
}

// Setters

func (e *ControllerError) WithError(err error) *ControllerError {
	e.err = err
	return e
}

func (e *ControllerError) WithCause(cause string) *ControllerError {
	e.cause = cause
	return e
}

func (e *ControllerError) WithResourceType(t string) *ControllerError {
	e.resourceType = t
	return e
}

func (e *ControllerError) WithEvent(event interface{}) *ControllerError {
	e.Event = &event
	return e
}

// NewErr** Functions will allocate a controller error for the specific type of error.
// Error details can be added through the use of WithError(err) and WithCause(string) methods

func NewErrNotFound() *ControllerError {
	return NewControllerError(http.StatusNotFound)
}

func NewErrBadRequest() *ControllerError {
	return NewControllerError(http.StatusBadRequest)
}

func NewErrNotAcceptable() *ControllerError {
	return NewControllerError(http.StatusNotAcceptable)
}

func NewErrInternalServerError() *ControllerError {
	return NewControllerError(http.StatusInternalServerError)
}

func NewErrNotImplemented() *ControllerError {
	return NewControllerError(http.StatusNotImplemented)
}

type ErrorResponse struct {
	Status  int    `json:"status"`
	Error   string `json:"error"`
	Cause   string `json:"cause,omitempty"`
	Details string `json:"details,omitempty"`
	Model   string `json:"model,omitempty"`
}

// New Error Response - Returns encoded byte stream for responding to
// an http request with an error. This provides a well defined response
// body for all unsuccessful element controller requests.
func NewErrorResponse(e *ControllerError, v interface{}) (s interface{}) {

	var details string
	if e.Unwrap() != nil {
		details = e.Unwrap().Error()
	}

	rsp := ErrorResponse{
		Status:  e.statusCode,
		Error:   http.StatusText(e.statusCode),
		Cause:   e.cause,
		Details: details,
	}

	if v != nil {
		model, _ := json.Marshal(v)
		rsp.Model = string(model)
	}

	return &rsp
}
