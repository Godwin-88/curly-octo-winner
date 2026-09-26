package guardian_auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	appmiddleware "github.com/shule360/api/internal/middleware"
)

// TestSessionWithStaffSessionAnswersNullGuardian locks in the fix for the
// console error the web client logged on every page load: hydration probes
// GET /auth/guardian/me to find out whether the session belongs to a parent,
// and a staff session used to be rejected with 403 (RequireRole("guardian")),
// which the browser reported as a failed request every time.
//
// The contract now: 200 with a null guardian for a non-guardian session.
func TestSessionWithStaffSessionAnswersNullGuardian(t *testing.T) {
	h := &Handler{}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/guardian/me", nil)
	// A staff session: authenticated, but no guardian id in the context.
	staffID := uuid.MustParse("b0000000-0000-0000-0000-000000000001")
	req = req.WithContext(context.WithValue(req.Context(), appmiddleware.ContextKeyStaffID, staffID))

	rec := httptest.NewRecorder()
	h.session(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusOK, rec.Body.String())
	}

	var body struct {
		Guardian *Guardian `json:"guardian"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Guardian != nil {
		t.Fatalf("guardian = %+v, want null for a staff session", body.Guardian)
	}
}
