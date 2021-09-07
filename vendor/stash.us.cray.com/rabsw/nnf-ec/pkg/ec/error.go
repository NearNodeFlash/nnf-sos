package ec

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type controllerError struct {
	statusCode int
	cause      string
	err        error
}

func NewControllerError(sc int) *controllerError {
	return &controllerError{statusCode: sc}
}

func (e *controllerError) Error() string {
	errorString := fmt.Sprintf("Error %d: %s", e.statusCode, http.StatusText(e.statusCode))
	if len(e.cause) != 0 {
		return fmt.Sprintf("%s, %s", errorString, e.cause)
	}
	return errorString
}

func (e *controllerError) Unwrap() error {
	return e.err
}

func (e *controllerError) WithError(err error) *controllerError {
	e.err = err
	return e
}

func (e *controllerError) WithCause(cause string) *controllerError {
	e.cause = cause
	return e
}

// NewErr** Functions will allocate a controller error for the specific type of error.
// Error details can be added through the use of WithError(err) and WithCause(string) methods

func NewErrNotFound() *controllerError {
	return NewControllerError(http.StatusNotFound)
}

func NewErrBadRequest() *controllerError {
	return NewControllerError(http.StatusBadRequest)
}

func NewErrNotAcceptable() *controllerError {
	return NewControllerError(http.StatusNotAcceptable)
}

func NewErrInternalServerError() *controllerError {
	return NewControllerError(http.StatusInternalServerError)
}

func NewErrNotImplemented() *controllerError {
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
func NewErrorResponse(e *controllerError, v interface{}) (s interface{}) {

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
