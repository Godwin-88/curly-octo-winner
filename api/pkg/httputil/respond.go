package httputil

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// RespondOK writes a 200 OK response with the given payload as JSON.
func RespondOK(w http.ResponseWriter, payload any) {
	writeJSON(w, http.StatusOK, payload)
}

// RespondCreated writes a 201 Created response with the given payload as JSON.
func RespondCreated(w http.ResponseWriter, payload any) {
	writeJSON(w, http.StatusCreated, payload)
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

// RespondInternalError writes a 500 error and logs the underlying error.
func RespondInternalError(w http.ResponseWriter, err error) {
	slog.Error("internal server error", "error", err)
	message := "An internal error occurred"
	if err != nil {
		message = err.Error()
	}
	RespondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", message)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		slog.Error("failed to encode JSON response", "error", err)
	}
}
