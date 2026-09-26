package httputil

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"reflect"
)

// RespondOK writes a 200 OK response with the given payload as JSON.
//
// A nil Go slice is written as [] rather than null. This is deliberate: list
// handlers naturally build `var rows []T` and append, so an empty result is a
// nil slice, and a nil slice marshals to JSON `null`. The web client then runs
// `rows.map(...)` and the page crashes with "Cannot read properties of null" —
// which a brand-new school hits on every empty list it opens. Normalising here
// fixes all of them (and any future one) instead of one handler at a time.
func RespondOK(w http.ResponseWriter, payload any) {
	writeJSON(w, http.StatusOK, emptySliceIfNil(payload))
}

// emptySliceIfNil replaces a nil slice with an empty slice of the same type.
// Everything else is passed through untouched.
func emptySliceIfNil(payload any) any {
	if payload == nil {
		return payload
	}
	v := reflect.ValueOf(payload)
	if v.Kind() == reflect.Slice && v.IsNil() {
		return reflect.MakeSlice(v.Type(), 0, 0).Interface()
	}
	return payload
}

// RespondCreated writes a 201 Created response with the given payload as JSON.
func RespondCreated(w http.ResponseWriter, payload any) {
	writeJSON(w, http.StatusCreated, emptySliceIfNil(payload))
}

// RespondNoContent writes a 204 No Content response.
func RespondNoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// RespondError writes a structured JSON error response.
// Format: {"error": "message", "code": "ERROR_CODE"}
func RespondError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]string{
		"error": message,
		"code":  code,
	})
}

// RespondBadRequest writes a 400 error.
func RespondBadRequest(w http.ResponseWriter, code, message string) {
	RespondError(w, http.StatusBadRequest, code, message)
}

// RespondUnauthorized writes a 401 error.
func RespondUnauthorized(w http.ResponseWriter, code, message string) {
	RespondError(w, http.StatusUnauthorized, code, message)
}

// RespondForbidden writes a 403 error.
func RespondForbidden(w http.ResponseWriter, code, message string) {
	RespondError(w, http.StatusForbidden, code, message)
}

// RespondNotFound writes a 404 error.
func RespondNotFound(w http.ResponseWriter, code, message string) {
	RespondError(w, http.StatusNotFound, code, message)
}

// RespondConflict writes a 409 Conflict error, used when a write clashes with
// existing data (e.g. a contact whose phone number is already saved). The
// client can offer "update the existing one instead" instead of failing.
func RespondConflict(w http.ResponseWriter, code, message string) {
	RespondError(w, http.StatusConflict, code, message)
}

// RespondInternalError writes a 500 error and logs the underlying error.
//
// The detail stays in the server log on purpose: the previous behaviour echoed
// err.Error() to the browser, which handed users raw Postgres text such as
// `column "name" does not exist` — table and column names are free
// reconnaissance, and a school user needs a sentence they can act on, not a
// query plan. The request id and the timestamp in the log line are enough to
// find the failure server-side.
func RespondInternalError(w http.ResponseWriter, err error) {
	slog.Error("internal server error", "error", err)
	RespondError(w, http.StatusInternalServerError, "INTERNAL_ERROR",
		"Something went wrong on our side. Please try again, or contact support if it keeps happening.")
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		slog.Error("failed to encode JSON response", "error", err)
	}
}
