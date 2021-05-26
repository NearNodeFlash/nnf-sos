package ec

import (
	"fmt"
	"net/http"
)

type ControllerError struct {
	StatusCode   int
	ErrorDetails string
}

func NewControllerError(sc int) *ControllerError {
	return &ControllerError{
		StatusCode:   sc,
		ErrorDetails: ""}
}

func (err *ControllerError) Error() string {
	errorString := fmt.Sprintf("Error %d: %s", err.StatusCode, http.StatusText(err.StatusCode))
	if err.ErrorDetails != "" {
		return fmt.Sprintf("%s, %s", errorString, err.ErrorDetails)
	}
	return errorString
}

func (err *ControllerError) Details(s string) {
	err.ErrorDetails = s
}

var (
	ErrNotFound            = NewControllerError(http.StatusNotFound)
	ErrBadRequest          = NewControllerError(http.StatusBadRequest)
	ErrNotAcceptable       = NewControllerError(http.StatusNotAcceptable)
	ErrInternalServerError = NewControllerError(http.StatusInternalServerError)
)
