package reports

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/shule360/api/internal/middleware"
	"github.com/shule360/api/pkg/httputil"
)

// Handler contains the HTTP handlers for the reports & analytics API.
type Handler struct {
	service *Service
}

// NewHandler creates a new reports handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Mount registers all reports routes under the provided router.
func (h *Handler) Mount(r chi.Router) {
	r.Route("/reports", func(r chi.Router) {
		r.Get("/", h.listReportCards)
		r.Get("/{id}", h.getReportCard)
		r.Patch("/{id}", h.updateReportCard)
		r.Delete("/{id}", h.deleteReportCard)
		r.Post("/generate", h.generateReportCard)
	})

	r.Route("/analytics", func(r chi.Router) {
		r.Get("/overview", h.schoolOverview)
		r.Get("/strand-coverage", h.strandCoverage)
		r.Get("/competency-distribution", h.competencyDistribution)
		r.Get("/teacher-velocity", h.teacherVelocity)
		r.Get("/learner-portfolio", h.learnerPortfolio)
		r.Get("/at-risk", h.atRiskLearners)
		r.Get("/learners/{learnerId}/performance", h.learningAreaPerformance)
	})
}

// --- Report card handlers ---

func (h *Handler) listReportCards(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	learnerID := r.URL.Query().Get("learner_id")
	term, _ := strconv.Atoi(r.URL.Query().Get("term"))
	year, _ := strconv.Atoi(r.URL.Query().Get("year"))

	cards, err := h.service.ListReportCards(r.Context(), tenantID, learnerID, term, year)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondOK(w, cards)
}

func (h *Handler) getReportCard(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid report card ID")
		return
	}
	card, err := h.service.GetReportCard(r.Context(), tenantID, id)
	if err != nil {
		httputil.RespondNotFound(w, "NOT_FOUND", err.Error())
		return
	}
	httputil.RespondOK(w, card)
}

func (h *Handler) generateReportCard(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	learnerID, err := uuid.Parse(r.URL.Query().Get("learner_id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "learner_id parameter is required")
		return
	}
	term, _ := strconv.Atoi(r.URL.Query().Get("term"))
	year, _ := strconv.Atoi(r.URL.Query().Get("year"))
	if term < 1 || term > 3 || year <= 0 {
		httputil.RespondBadRequest(w, "INVALID_PARAM", "term (1-3) and year are required")
		return
	}

	var req GenerateReportCardRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "Invalid request body: "+err.Error())
		return
	}

	card, err := h.service.GenerateReportCard(r.Context(), tenantID, learnerID, term, year, req)
	if err != nil {
		httputil.RespondBadRequest(w, "GENERATE_FAILED", err.Error())
		return
	}
	httputil.RespondOK(w, card)
}

func (h *Handler) updateReportCard(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid report card ID")
		return
	}
	var req UpdateReportCardRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "Invalid request body: "+err.Error())
		return
	}
	card, err := h.service.UpdateReportCard(r.Context(), tenantID, id, req)
	if err != nil {
		httputil.RespondBadRequest(w, "UPDATE_FAILED", err.Error())
		return
	}
	httputil.RespondOK(w, card)
}

func (h *Handler) deleteReportCard(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid report card ID")
		return
	}
	if err := h.service.DeleteReportCard(r.Context(), tenantID, id); err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondNoContent(w)
}

// --- Analytics handlers ---

func (h *Handler) schoolOverview(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	ov, err := h.service.SchoolOverview(r.Context(), tenantID)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondOK(w, ov)
}

func (h *Handler) strandCoverage(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	grade := r.URL.Query().Get("grade")
	stream := r.URL.Query().Get("stream")
	term, _ := strconv.Atoi(r.URL.Query().Get("term"))
	year, _ := strconv.Atoi(r.URL.Query().Get("year"))

	rows, err := h.service.StrandCoverage(r.Context(), tenantID, grade, stream, term, year)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondOK(w, rows)
}

func (h *Handler) competencyDistribution(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	strandID := r.URL.Query().Get("strand_id")
	grade := r.URL.Query().Get("grade")
	stream := r.URL.Query().Get("stream")
	term, _ := strconv.Atoi(r.URL.Query().Get("term"))
	year, _ := strconv.Atoi(r.URL.Query().Get("year"))

	rows, err := h.service.CompetencyDistribution(r.Context(), tenantID, strandID, grade, stream, term, year)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondOK(w, rows)
}

func (h *Handler) teacherVelocity(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	term, _ := strconv.Atoi(r.URL.Query().Get("term"))
	year, _ := strconv.Atoi(r.URL.Query().Get("year"))

	rows, err := h.service.TeacherVelocity(r.Context(), tenantID, term, year)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondOK(w, rows)
}

func (h *Handler) learnerPortfolio(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	grade := r.URL.Query().Get("grade")
	stream := r.URL.Query().Get("stream")
	term, _ := strconv.Atoi(r.URL.Query().Get("term"))
	year, _ := strconv.Atoi(r.URL.Query().Get("year"))

	rows, err := h.service.LearnerPortfolio(r.Context(), tenantID, grade, stream, term, year)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondOK(w, rows)
}

func (h *Handler) atRiskLearners(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	term, _ := strconv.Atoi(r.URL.Query().Get("term"))
	year, _ := strconv.Atoi(r.URL.Query().Get("year"))

	rows, err := h.service.AtRiskLearners(r.Context(), tenantID, term, year)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondOK(w, rows)
}

func (h *Handler) learningAreaPerformance(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	learnerID, err := uuid.Parse(chi.URLParam(r, "learnerId"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid learner ID")
		return
	}
	term, _ := strconv.Atoi(r.URL.Query().Get("term"))
	year, _ := strconv.Atoi(r.URL.Query().Get("year"))

	rows, err := h.service.LearningAreaPerformance(r.Context(), tenantID, learnerID, term, year)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondOK(w, rows)
}
