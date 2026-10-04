package academic

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/shule360/api/internal/academic/assessment"
	"github.com/shule360/api/internal/academic/attendance"
	"github.com/shule360/api/internal/academic/curriculum"
	"github.com/shule360/api/internal/middleware"
	"github.com/shule360/api/pkg/apperr"
	"github.com/shule360/api/pkg/httputil"
)

// Handler contains the HTTP handlers for the academic API.
type Handler struct {
	curriculumSvc   *curriculum.Service
	assessmentSvc   *assessment.Service
	attendanceSvc   *attendance.Service
	absenceAlertSvc *attendance.AbsenceAlertService
}

// NewHandler creates a new academic handler.
func NewHandler(
	curriculumSvc *curriculum.Service,
	assessmentSvc *assessment.Service,
	attendanceSvc *attendance.Service,
	absenceAlertSvc *attendance.AbsenceAlertService,
) *Handler {
	return &Handler{
		curriculumSvc:   curriculumSvc,
		assessmentSvc:   assessmentSvc,
		attendanceSvc:   attendanceSvc,
		absenceAlertSvc: absenceAlertSvc,
	}
}

// Mount registers all academic routes under the provided router.
func (h *Handler) Mount(r chi.Router) {
	r.Route("/curriculum", func(r chi.Router) {
		r.Get("/learning-areas", h.listLearningAreas)
		r.Post("/learning-areas", h.createLearningArea)
		r.Post("/learning-areas/defaults", h.addDefaultLearningAreas)
		r.Get("/learning-areas/{id}", h.getLearningArea)
		r.Put("/learning-areas/{id}", h.updateLearningArea)
		r.Delete("/learning-areas/{id}", h.deleteLearningArea)
		r.Get("/learning-areas/{id}/strands", h.listStrands)
		r.Post("/strands", h.createStrand)
		r.Put("/strands/{id}", h.updateStrand)
		r.Delete("/strands/{id}", h.deleteStrand)
		r.Get("/strands/{id}/sub-strands", h.listSubStrands)
		r.Post("/sub-strands", h.createSubStrand)
		r.Put("/sub-strands/{id}", h.updateSubStrand)
		r.Delete("/sub-strands/{id}", h.deleteSubStrand)
		r.Get("/sub-strands/{id}/learning-outcomes", h.listLearningOutcomes)
		r.Post("/learning-outcomes", h.createLearningOutcome)
		r.Get("/core-competencies", h.listCoreCompetencies)
		r.Post("/core-competencies", h.createCoreCompetency)
		r.Put("/core-competencies/{id}", h.updateCoreCompetency)
		r.Delete("/core-competencies/{id}", h.deleteCoreCompetency)
		r.Get("/values", h.listValues)
		r.Post("/values", h.createValue)
		r.Put("/values/{id}", h.updateValue)
		r.Delete("/values/{id}", h.deleteValue)
	})

	r.Route("/assessments", func(r chi.Router) {
		r.Post("/", h.createAssessment)
		r.Get("/{id}", h.getAssessment)
		r.Get("/learner/{learnerId}", h.listAssessmentsByLearner)
		r.Get("/term-summary", h.listTermSummaries)
		// Static segments are declared before the {id} wildcard for readability;
		// chi matches them first regardless of order.
		r.Get("/term-observations", h.listAssessmentsByTerm)
		r.Get("/competency-distribution", h.competencyDistribution)
		r.Delete("/{id}", h.deleteAssessment)
	})

	r.Route("/attendance", func(r chi.Router) {
		r.Post("/", h.markAttendance)
		r.Post("/bulk", h.markAttendanceBulk)
		r.Get("/date", h.listAttendanceByDate)
		r.Get("/summary", h.attendanceSummary)
		r.Get("/chronic", h.listChronicAbsenteeism)
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
	if !decodeBody(w, r, &la) {
		return
	}
	if !requireName(w, la.Name) {
		return
	}

	result, err := h.curriculumSvc.CreateLearningArea(r.Context(), tenantID, &la)
	if err != nil {
		respondCurriculumErr(w, err, "learning area", la.KICDCode)
		return
	}
	httputil.RespondCreated(w, result)
}

// addDefaultLearningAreas adds the usual learning areas for one grade.
func (h *Handler) addDefaultLearningAreas(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	var req struct {
		GradeLevel string `json:"grade_level"`
	}
	if !decodeBody(w, r, &req) {
		return
	}
	added, err := h.curriculumSvc.AddDefaultLearningAreas(r.Context(), tenantID, req.GradeLevel)
	if err != nil {
		apperr.Respond(w, err)
		return
	}
	httputil.RespondOK(w, map[string]any{"added": added, "grade_level": req.GradeLevel})
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
	if !decodeBody(w, r, &st) {
		return
	}
	if !requireName(w, st.Name) {
		return
	}

	result, err := h.curriculumSvc.CreateStrand(r.Context(), tenantID, &st)
	if err != nil {
		respondCurriculumErr(w, err, "strand", st.KICDCode)
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
	if !decodeBody(w, r, &ss) {
		return
	}
	if !requireName(w, ss.Name) {
		return
	}

	result, err := h.curriculumSvc.CreateSubStrand(r.Context(), tenantID, &ss)
	if err != nil {
		respondCurriculumErr(w, err, "sub-strand", ss.KICDCode)
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
	if strings.TrimSpace(lo.Description) == "" {
		httputil.RespondBadRequest(w, "DESCRIPTION_REQUIRED",
			"Write the learning outcome this sub-strand covers.")
		return
	}

	result, err := h.curriculumSvc.CreateLearningOutcome(r.Context(), tenantID, &lo)
	if err != nil {
		httputil.RespondWriteError(w, err,
			"That learning outcome already exists on this sub-strand.",
			"That sub-strand is not in your curriculum.")
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
	if !decodeBody(w, r, &cc) {
		return
	}
	if !requireName(w, cc.Name) {
		return
	}

	result, err := h.curriculumSvc.CreateCoreCompetency(r.Context(), tenantID, &cc)
	if err != nil {
		respondCurriculumErr(w, err, "core", cc.KICDCode)
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
	if !decodeBody(w, r, &v) {
		return
	}
	if !requireName(w, v.Name) {
		return
	}

	result, err := h.curriculumSvc.CreateValue(r.Context(), tenantID, &v)
	if err != nil {
		respondCurriculumErr(w, err, "value", v.KICDCode)
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
	if !decodeBody(w, r, &req) {
		return
	}
	// Who observed is whoever is signed in, never a field of the request.
	req.TeacherID, _ = middleware.GetStaffID(r)
	if req.Term == 0 {
		req.Term = currentTerm()
	}
	if req.Year == 0 {
		req.Year = currentYear()
	}

	result, err := h.assessmentSvc.Create(r.Context(), tenantID, req)
	if err != nil {
		apperr.Respond(w, err)
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
		apperr.Respond(w, err)
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

	term, year := termParams(r)

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

	term, year := termParams(r)

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

	staffID, _ := middleware.GetStaffID(r)
	role, _ := middleware.GetStaffRole(r)
	manager := role == "principal" || role == "super_admin"
	if err := h.assessmentSvc.Delete(r.Context(), tenantID, id, staffID, manager); err != nil {
		apperr.Respond(w, err)
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
	if !decodeBody(w, r, &req) {
		return
	}
	req.MarkedBy, _ = middleware.GetStaffID(r)

	result, err := h.attendanceSvc.MarkAttendance(r.Context(), tenantID, req)
	if err != nil {
		respondAttendanceErr(w, err)
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
		apperr.Respond(w, err)
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
