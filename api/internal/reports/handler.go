package reports

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/shule360/api/internal/middleware"
	"github.com/shule360/api/pkg/apperr"
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
		r.Post("/generate", h.generateReportCard)
		r.Post("/generate-class", h.generateForClass)
		r.Get("/{id}", h.getReportCard)
		r.Patch("/{id}", h.updateReportCard)
		r.Delete("/{id}", h.deleteReportCard)
		r.Get("/{id}/pdf", h.reportCardPDF)
		r.Post("/{id}/publish", h.publishReportCard)
		// Undoing a publication is for whoever runs the school.
		r.With(middleware.RequireRole("principal", "super_admin")).Post("/{id}/reopen", h.reopenReportCard)
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

// session returns the school and the signed-in member of staff.
func session(w http.ResponseWriter, r *http.Request) (tenantID, staffID uuid.UUID, ok bool) {
	tenantID, ok = middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	staffID, _ = middleware.GetStaffID(r)
	return
}

func cardID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondNotFound(w, "NOT_FOUND", "Not found")
		return uuid.Nil, false
	}
	return id, true
}

func decode(w http.ResponseWriter, r *http.Request, into any) bool {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<18))
	dec.DisallowUnknownFields()
	if err := dec.Decode(into); err != nil && !errors.Is(err, io.EOF) {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "The request could not be read.")
		return false
	}
	return true
}

func (h *Handler) listReportCards(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := session(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()
	term, _ := strconv.Atoi(q.Get("term"))
	year, _ := strconv.Atoi(q.Get("year"))
	cards, err := h.service.ListReportCards(r.Context(), tenantID, CardFilter{
		LearnerID: q.Get("learner_id"), Grade: q.Get("grade"), Stream: q.Get("stream"),
		Status: q.Get("status"), Term: term, Year: year,
	})
	if err != nil {
		apperr.Respond(w, err)
		return
	}
	httputil.RespondOK(w, cards)
}

func (h *Handler) getReportCard(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := session(w, r)
	if !ok {
		return
	}
	id, ok := cardID(w, r)
	if !ok {
		return
	}
	card, err := h.service.GetReportCard(r.Context(), tenantID, id)
	if err != nil {
		apperr.Respond(w, err)
		return
	}
	httputil.RespondOK(w, card)
}

// generateReportCard builds one learner's draft card. The learner and term
// may be given in the body or, as before, in the query string.
func (h *Handler) generateReportCard(w http.ResponseWriter, r *http.Request) {
	tenantID, staffID, ok := session(w, r)
	if !ok {
		return
	}
	var req struct {
		LearnerID string `json:"learner_id"`
		Term      int    `json:"term"`
		Year      int    `json:"year"`
		CardInput
	}
	if !decode(w, r, &req) {
		return
	}
	q := r.URL.Query()
	if req.LearnerID == "" {
		req.LearnerID = q.Get("learner_id")
	}
	if req.Term == 0 {
		req.Term, _ = strconv.Atoi(q.Get("term"))
	}
	if req.Year == 0 {
		req.Year, _ = strconv.Atoi(q.Get("year"))
	}
	learnerID, err := uuid.Parse(req.LearnerID)
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID", "Choose the learner to make a report card for.")
		return
	}
	card, err := h.service.GenerateReportCard(r.Context(), tenantID, learnerID, req.Term, req.Year, staffID, req.CardInput)
	if err != nil {
		apperr.Respond(w, err)
		return
	}
	httputil.RespondOK(w, card)
}

func (h *Handler) generateForClass(w http.ResponseWriter, r *http.Request) {
	tenantID, staffID, ok := session(w, r)
	if !ok {
		return
	}
	var req struct {
		Grade  string `json:"grade"`
		Stream string `json:"stream"`
		Term   int    `json:"term"`
		Year   int    `json:"year"`
	}
	if !decode(w, r, &req) {
		return
	}
	result, err := h.service.GenerateForClass(r.Context(), tenantID, req.Grade, req.Stream, req.Term, req.Year, staffID)
	if err != nil {
		apperr.Respond(w, err)
		return
	}
	httputil.RespondOK(w, result)
}

func (h *Handler) updateReportCard(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := session(w, r)
	if !ok {
		return
	}
	id, ok := cardID(w, r)
	if !ok {
		return
	}
	var req CardInput
	if !decode(w, r, &req) {
		return
	}
	card, err := h.service.UpdateReportCard(r.Context(), tenantID, id, req)
	if err != nil {
		apperr.Respond(w, err)
		return
	}
	httputil.RespondOK(w, card)
}

func (h *Handler) publishReportCard(w http.ResponseWriter, r *http.Request) {
	tenantID, staffID, ok := session(w, r)
	if !ok {
		return
	}
	id, ok := cardID(w, r)
	if !ok {
		return
	}
	card, err := h.service.PublishReportCard(r.Context(), tenantID, id, staffID)
	if err != nil {
		apperr.Respond(w, err)
		return
	}
	httputil.RespondOK(w, card)
}

func (h *Handler) reopenReportCard(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := session(w, r)
	if !ok {
		return
	}
	id, ok := cardID(w, r)
	if !ok {
		return
	}
	card, err := h.service.ReopenReportCard(r.Context(), tenantID, id)
	if err != nil {
		apperr.Respond(w, err)
		return
	}
	httputil.RespondOK(w, card)
}

func (h *Handler) deleteReportCard(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := session(w, r)
	if !ok {
		return
	}
	id, ok := cardID(w, r)
	if !ok {
		return
	}
	if err := h.service.DeleteReportCard(r.Context(), tenantID, id); err != nil {
		apperr.Respond(w, err)
		return
	}
	httputil.RespondNoContent(w)
}

// reportCardPDF answers with the document itself.
func (h *Handler) reportCardPDF(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := session(w, r)
	if !ok {
		return
	}
	id, ok := cardID(w, r)
	if !ok {
		return
	}
	pdf, name, err := h.service.ReportCardPDF(r.Context(), tenantID, id)
	if err != nil {
		apperr.Respond(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", `attachment; filename="`+name+`"`)
	w.Header().Set("Content-Length", strconv.Itoa(len(pdf)))
	// A learner's report is personal: it is not to be kept by a shared cache.
	w.Header().Set("Cache-Control", "private, no-store")
	_, _ = w.Write(pdf)
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
