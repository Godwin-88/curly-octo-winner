package academic

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/shule360/api/internal/academic/assessment"
	"github.com/shule360/api/internal/academic/attendance"
	"github.com/shule360/api/internal/academic/curriculum"
	"github.com/shule360/api/internal/middleware"
	"github.com/shule360/api/pkg/httputil"
)

// Handler contains the HTTP handlers for the academic API.
type Handler struct {
	curriculumSvc *curriculum.Service
	assessmentSvc *assessment.Service
	attendanceSvc *attendance.Service
}

// NewHandler creates a new academic handler.
func NewHandler(
	curriculumSvc *curriculum.Service,
	assessmentSvc *assessment.Service,
	attendanceSvc *attendance.Service,
) *Handler {
	return &Handler{
		curriculumSvc: curriculumSvc,
		assessmentSvc: assessmentSvc,
		attendanceSvc: attendanceSvc,
	}
}

// Mount registers all academic routes under the provided router.
func (h *Handler) Mount(r chi.Router) {
	r.Route("/curriculum", func(r chi.Router) {
		r.Get("/learning-areas", h.listLearningAreas)
		r.Post("/learning-areas", h.createLearningArea)
		r.Get("/learning-areas/{id}", h.getLearningArea)
		r.Get("/learning-areas/{id}/strands", h.listStrands)
		r.Post("/strands", h.createStrand)
		r.Get("/strands/{id}/sub-strands", h.listSubStrands)
		r.Post("/sub-strands", h.createSubStrand)
		r.Get("/sub-strands/{id}/learning-outcomes", h.listLearningOutcomes)
		r.Post("/learning-outcomes", h.createLearningOutcome)
		r.Get("/core-competencies", h.listCoreCompetencies)
		r.Post("/core-competencies", h.createCoreCompetency)
		r.Get("/values", h.listValues)
		r.Post("/values", h.createValue)
	})

	r.Route("/assessments", func(r chi.Router) {
		r.Post("/", h.createAssessment)
		r.Get("/{id}", h.getAssessment)
		r.Get("/learner/{learnerId}", h.listAssessmentsByLearner)
		r.Get("/term-summary", h.listTermSummaries)
		r.Delete("/{id}", h.deleteAssessment)
	})

	r.Route("/attendance", func(r chi.Router) {
		r.Post("/", h.markAttendance)
		r.Get("/date", h.listAttendanceByDate)
		r.Get("/learner/{learnerId}", h.listAttendanceByLearner)
		r.Get("/{id}", h.getAttendance)
		r.Delete("/{id}", h.deleteAttendance)
	})
}

// --- Curriculum Handlers ---

func (h *Handler) listLearningAreas(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}

	areas, err := h.curriculumSvc.ListLearningAreas(r.Context(), tenantID)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondOK(w, areas)
}

func (h *Handler) createLearningArea(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}

	var la curriculum.LearningArea
	if err := json.NewDecoder(r.Body).Decode(&la); err != nil {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "Invalid request body: "+err.Error())
		return
	}

	result, err := h.curriculumSvc.CreateLearningArea(r.Context(), tenantID, &la)
	if err != nil {
		httputil.RespondBadRequest(w, "CREATE_FAILED", err.Error())
		return
	}
	httputil.RespondCreated(w, result)
}

func (h *Handler) getLearningArea(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid learning area ID")
		return
	}

	result, err := h.curriculumSvc.GetLearningArea(r.Context(), tenantID, id)
	if err != nil {
		httputil.RespondNotFound(w, "NOT_FOUND", err.Error())
		return
	}
	httputil.RespondOK(w, result)
}

func (h *Handler) listStrands(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}

	learningAreaID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid learning area ID")
		return
	}

	strands, err := h.curriculumSvc.ListStrands(r.Context(), tenantID, learningAreaID)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondOK(w, strands)
}

func (h *Handler) createStrand(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}

	var st curriculum.Strand
	if err := json.NewDecoder(r.Body).Decode(&st); err != nil {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "Invalid request body: "+err.Error())
		return
	}

	result, err := h.curriculumSvc.CreateStrand(r.Context(), tenantID, &st)
	if err != nil {
		httputil.RespondBadRequest(w, "CREATE_FAILED", err.Error())
		return
	}
	httputil.RespondCreated(w, result)
}

func (h *Handler) listSubStrands(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}

	strandID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid strand ID")
		return
	}

	subStrands, err := h.curriculumSvc.ListSubStrands(r.Context(), tenantID, strandID)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondOK(w, subStrands)
}

func (h *Handler) createSubStrand(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}

	var ss curriculum.SubStrand
	if err := json.NewDecoder(r.Body).Decode(&ss); err != nil {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "Invalid request body: "+err.Error())
		return
	}

	result, err := h.curriculumSvc.CreateSubStrand(r.Context(), tenantID, &ss)
	if err != nil {
		httputil.RespondBadRequest(w, "CREATE_FAILED", err.Error())
		return
	}
	httputil.RespondCreated(w, result)
}

func (h *Handler) listLearningOutcomes(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}

	subStrandID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid sub-strand ID")
		return
	}

	outcomes, err := h.curriculumSvc.ListLearningOutcomes(r.Context(), tenantID, subStrandID)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondOK(w, outcomes)
}

func (h *Handler) createLearningOutcome(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}

	var lo curriculum.LearningOutcome
	if err := json.NewDecoder(r.Body).Decode(&lo); err != nil {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "Invalid request body: "+err.Error())
		return
	}

	result, err := h.curriculumSvc.CreateLearningOutcome(r.Context(), tenantID, &lo)
	if err != nil {
		httputil.RespondBadRequest(w, "CREATE_FAILED", err.Error())
		return
	}
	httputil.RespondCreated(w, result)
}

func (h *Handler) listCoreCompetencies(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}

	competencies, err := h.curriculumSvc.ListCoreCompetencies(r.Context(), tenantID)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondOK(w, competencies)
}

func (h *Handler) createCoreCompetency(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}

	var cc curriculum.CoreCompetency
	if err := json.NewDecoder(r.Body).Decode(&cc); err != nil {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "Invalid request body: "+err.Error())
		return
	}

	result, err := h.curriculumSvc.CreateCoreCompetency(r.Context(), tenantID, &cc)
	if err != nil {
		httputil.RespondBadRequest(w, "CREATE_FAILED", err.Error())
		return
	}
	httputil.RespondCreated(w, result)
}

func (h *Handler) listValues(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}

	values, err := h.curriculumSvc.ListValues(r.Context(), tenantID)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondOK(w, values)
}

func (h *Handler) createValue(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}

	var v curriculum.Value
	if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "Invalid request body: "+err.Error())
		return
	}

	result, err := h.curriculumSvc.CreateValue(r.Context(), tenantID, &v)
	if err != nil {
		httputil.RespondBadRequest(w, "CREATE_FAILED", err.Error())
		return
	}
	httputil.RespondCreated(w, result)
}

// --- Assessment Handlers ---

func (h *Handler) createAssessment(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}

	var req assessment.CreateAssessmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "Invalid request body: "+err.Error())
		return
	}

	result, err := h.assessmentSvc.Create(r.Context(), tenantID, req)
	if err != nil {
		httputil.RespondBadRequest(w, "CREATE_FAILED", err.Error())
		return
	}
	httputil.RespondCreated(w, result)
}

func (h *Handler) getAssessment(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid assessment ID")
		return
	}

	result, err := h.assessmentSvc.GetByID(r.Context(), tenantID, id)
	if err != nil {
		httputil.RespondNotFound(w, "NOT_FOUND", err.Error())
		return
	}
	httputil.RespondOK(w, result)
}

func (h *Handler) listAssessmentsByLearner(w http.ResponseWriter, r *http.Request) {
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
	if year == 0 {
		year = time.Now().Year()
	}

	results, err := h.assessmentSvc.ListByLearner(r.Context(), tenantID, learnerID, term, year)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondOK(w, results)
}

func (h *Handler) listTermSummaries(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}

	learnerID, err := uuid.Parse(r.URL.Query().Get("learner_id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid learner_id parameter")
		return
	}

	term, _ := strconv.Atoi(r.URL.Query().Get("term"))
	year, _ := strconv.Atoi(r.URL.Query().Get("year"))
	if year == 0 {
		year = time.Now().Year()
	}

	results, err := h.assessmentSvc.ListSummariesByLearner(r.Context(), tenantID, learnerID, term, year)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondOK(w, results)
}

func (h *Handler) deleteAssessment(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid assessment ID")
		return
	}

	if err := h.assessmentSvc.Delete(r.Context(), tenantID, id); err != nil {
		httputil.RespondBadRequest(w, "DELETE_FAILED", err.Error())
		return
	}
	httputil.RespondNoContent(w)
}

// --- Attendance Handlers ---

func (h *Handler) markAttendance(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}

	var req attendance.CreateAttendanceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "Invalid request body: "+err.Error())
		return
	}

	result, err := h.attendanceSvc.MarkAttendance(r.Context(), tenantID, req)
	if err != nil {
		httputil.RespondBadRequest(w, "MARK_FAILED", err.Error())
		return
	}
	httputil.RespondCreated(w, result)
}

func (h *Handler) listAttendanceByDate(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}

	dateStr := r.URL.Query().Get("date")
	if dateStr == "" {
		httputil.RespondBadRequest(w, "MISSING_PARAM", "date parameter is required (YYYY-MM-DD)")
		return
	}

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_DATE", "date must be in YYYY-MM-DD format")
		return
	}

	results, err := h.attendanceSvc.ListSummariesByDate(r.Context(), tenantID, date)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondOK(w, results)
}

func (h *Handler) listAttendanceByLearner(w http.ResponseWriter, r *http.Request) {
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

	results, err := h.attendanceSvc.ListByLearner(r.Context(), tenantID, learnerID)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondOK(w, results)
}

func (h *Handler) getAttendance(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid attendance ID")
		return
	}

	result, err := h.attendanceSvc.GetByID(r.Context(), tenantID, id)
	if err != nil {
		httputil.RespondNotFound(w, "NOT_FOUND", err.Error())
		return
	}
	httputil.RespondOK(w, result)
}

func (h *Handler) deleteAttendance(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid attendance ID")
		return
	}

	if err := h.attendanceSvc.Delete(r.Context(), tenantID, id); err != nil {
		httputil.RespondBadRequest(w, "DELETE_FAILED", err.Error())
		return
	}
	httputil.RespondNoContent(w)
}