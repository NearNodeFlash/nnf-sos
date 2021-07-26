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

var (
	// Error Not Found - Returned when the requested URL path contains wildcards
	// not found or supported by the element controller.
	ErrNotFound = NewControllerError(http.StatusNotFound)

	// Error Bad Request - Used when the request body is illformed. Usually this
	// is in response to a POST request where the provided json is invalid, or
	// has values unsupported by the element controller
	ErrBadRequest = NewControllerError(http.StatusBadRequest)

	// Error Not Acceptable - Used when the request body contains fields that
	// are invalid to the request. Usually this is in response to a POST request
	// where the element controller is not in a position to handle the request.
	ErrNotAcceptable = NewControllerError(http.StatusNotAcceptable)

	// Error Internal Server Error - Used when the element controller cannot
	// service the request for some reason.
	ErrInternalServerError = NewControllerError(http.StatusInternalServerError)

	// Error Not Implemented - Used for rejecting requests the element controller
	// does not implement. Usually used for code stubs.
	ErrNotImplemented = NewControllerError(http.StatusNotImplemented)
)

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

	rsp := ErrorResponse{
		Status:  e.statusCode,
		Error:   http.StatusText(e.statusCode),
		Cause:   e.cause,
		Details: e.Unwrap().Error(),
	}

	if v != nil {
		model, _ := json.Marshal(v)
		rsp.Model = string(model)
	}

	return &rsp
}
