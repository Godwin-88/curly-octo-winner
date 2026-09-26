package academic

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/shule360/api/internal/middleware"
	"github.com/shule360/api/pkg/httputil"
)

// listAssessmentsByTerm returns the term's observations with learner and
// curriculum context — the list a teacher reviews before a parents' day.
func (h *Handler) listAssessmentsByTerm(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	term, year := termParams(r)

	rows, err := h.assessmentSvc.ListSummariesByTermYear(r.Context(), tenantID, term, year)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondOK(w, rows)
}

// competencyDistribution counts rubric levels for one sub-strand, so the
// assessments page can show real numbers instead of four hard-coded zeros.
func (h *Handler) competencyDistribution(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	subStrandID, err := uuid.Parse(r.URL.Query().Get("sub_strand_id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "sub_strand_id is required")
		return
	}
	term, year := termParams(r)

	dist, err := h.assessmentSvc.CompetencyDistribution(r.Context(), tenantID, subStrandID, term, year)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondOK(w, dist)
}
