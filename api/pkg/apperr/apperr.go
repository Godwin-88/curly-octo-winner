// Package apperr holds the refusals a service can give that the person at the
// screen can act on, and turns them into responses. Anything else is a fault
// and is answered as one.
package apperr

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/shule360/api/pkg/httputil"
)

// ErrNotFound is returned when a record does not exist in the caller's school.
var ErrNotFound = errors.New("not found")

// ValidationError is a refusal the caller can fix; it is shown as written.
type ValidationError struct{ Message string }

func (e *ValidationError) Error() string { return e.Message }

// ConflictError is a refusal because of what is already recorded.
type ConflictError struct{ Message string }

func (e *ConflictError) Error() string { return e.Message }

// Invalid builds a ValidationError.
func Invalid(format string, args ...any) error {
	return &ValidationError{Message: fmt.Sprintf(format, args...)}
}

// Conflict builds a ConflictError.
func Conflict(format string, args ...any) error {
	return &ConflictError{Message: fmt.Sprintf(format, args...)}
}

// Respond writes the response for an error returned by a service.
func Respond(w http.ResponseWriter, err error) {
	var validation *ValidationError
	var conflict *ConflictError
	switch {
	case errors.Is(err, ErrNotFound):
		httputil.RespondNotFound(w, "NOT_FOUND", "Not found")
	case errors.As(err, &validation):
		httputil.RespondBadRequest(w, "INVALID", validation.Message)
	case errors.As(err, &conflict):
		httputil.RespondConflict(w, "CONFLICT", conflict.Message)
	default:
		httputil.RespondInternalError(w, err)
	}
}
