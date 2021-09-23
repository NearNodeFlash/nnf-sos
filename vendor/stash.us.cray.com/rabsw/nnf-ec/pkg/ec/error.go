package ec

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type ControllerError struct {
	statusCode int
	cause      string
	err        error
	Event      interface{}
}

func NewControllerError(sc int) *ControllerError {
	return &ControllerError{statusCode: sc}
}

func (e *ControllerError) Error() string {
	errorString := fmt.Sprintf("Error %d: %s", e.statusCode, http.StatusText(e.statusCode))
	if len(e.cause) != 0 {
		return fmt.Sprintf("%s, %s", errorString, e.cause)
	}
	return errorString
}

func (e *ControllerError) Unwrap() error {
	return e.err
}

func (e *ControllerError) WithError(err error) *ControllerError {
	e.err = err
	return e
}

func (e *ControllerError) WithCause(cause string) *ControllerError {
	e.cause = cause
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
