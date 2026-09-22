package teacher

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/shule360/api/pkg/httputil"
)

type Handler struct {
	pool *pgxpool.Pool
}

func NewHandler(pool *pgxpool.Pool) *Handler {
	return &Handler{pool: pool}
}

func (h *Handler) Mount(r chi.Router) {
	r.Route("/teacher", func(r chi.Router) {
		r.Get("/dashboard", h.dashboard)
		r.Get("/classes", h.listClasses)
		r.Get("/classes/learners", h.listClassLearners)
		r.Get("/classes/attendance", h.getClassAttendance)
		r.Post("/classes/attendance", h.markClassAttendance)
		r.Get("/classes/assessments", h.getClassAssessments)
		r.Post("/classes/assessments", h.createAssessment)
		r.Get("/learners/{learnerId}/portfolio", h.getLearnerPortfolio)
	})
}

func (h *Handler) dashboard(w http.ResponseWriter, r *http.Request) {
	staffID, _ := uuid.Parse(r.Header.Get("X-Staff-ID"))
	if staffID == uuid.Nil {
		staffID, _ = uuid.Parse(r.URL.Query().Get("staff_id"))
	}
	if staffID == uuid.Nil {
		httputil.RespondBadRequest(w, "INVALID_STAFF", "staff_id is required")
		return
	}

	rows, err := h.pool.Query(r.Context(), `
		SELECT grade, stream, COUNT(*) AS learner_count
		FROM learners
		WHERE tenant_id = $1 AND class_teacher_id = $2 AND is_active = true
		GROUP BY grade, stream
	`, r.Header.Get("X-Tenant-ID"), staffID)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	defer rows.Close()

	type ClassSummary struct {
		Grade       string `json:"grade"`
		Stream      string `json:"stream"`
		LearnerCount int64 `json:"learner_count"`
	}
	var classes []ClassSummary
	for rows.Next() {
		var c ClassSummary
		if err := rows.Scan(&c.Grade, &c.Stream, &c.LearnerCount); err != nil {
			httputil.RespondInternalError(w, err)
			return
		}
		classes = append(classes, c)
	}
	httputil.RespondOK(w, map[string]any{
		"classes": classes,
		"message": "Teacher dashboard",
	})
}

func (h *Handler) listClasses(w http.ResponseWriter, r *http.Request) {
	staffID, _ := uuid.Parse(r.Header.Get("X-Staff-ID"))
	if staffID == uuid.Nil {
		staffID, _ = uuid.Parse(r.URL.Query().Get("staff_id"))
	}
	if staffID == uuid.Nil {
		httputil.RespondBadRequest(w, "INVALID_STAFF", "staff_id is required")
		return
	}

	rows, err := h.pool.Query(r.Context(), `
		SELECT grade, stream, COUNT(*) AS learner_count
		FROM learners
		WHERE tenant_id = $1 AND class_teacher_id = $2 AND is_active = true
		GROUP BY grade, stream
	`, r.Header.Get("X-Tenant-ID"), staffID)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	defer rows.Close()

	type ClassSummary struct {
		Grade       string `json:"grade"`
		Stream      string `json:"stream"`
		LearnerCount int64 `json:"learner_count"`
	}
	var classes []ClassSummary
	for rows.Next() {
		var c ClassSummary
		if err := rows.Scan(&c.Grade, &c.Stream, &c.LearnerCount); err != nil {
			httputil.RespondInternalError(w, err)
			return
		}
		classes = append(classes, c)
	}
	httputil.RespondOK(w, classes)
}

func (h *Handler) listClassLearners(w http.ResponseWriter, r *http.Request) {
	grade := r.URL.Query().Get("grade")
	stream := r.URL.Query().Get("stream")
	if grade == "" || stream == "" {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "grade and stream are required")
		return
	}

	rows, err := h.pool.Query(r.Context(), `
		SELECT id, full_name, upi
		FROM learners
		WHERE tenant_id = $1 AND grade = $2 AND stream = $3 AND is_active = true
		ORDER BY full_name
	`, r.Header.Get("X-Tenant-ID"), grade, stream)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	defer rows.Close()

	type LearnerBrief struct {
		ID       uuid.UUID `json:"id"`
		FullName string    `json:"full_name"`
		UPI      string    `json:"upi"`
	}
	var learners []LearnerBrief
	for rows.Next() {
		var l LearnerBrief
		if err := rows.Scan(&l.ID, &l.FullName, &l.UPI); err != nil {
			httputil.RespondInternalError(w, err)
			return
		}
		learners = append(learners, l)
	}
	httputil.RespondOK(w, learners)
}

func (h *Handler) getClassAttendance(w http.ResponseWriter, r *http.Request) {
	grade := r.URL.Query().Get("grade")
	stream := r.URL.Query().Get("stream")
	dateStr := r.URL.Query().Get("date")
	if grade == "" || stream == "" || dateStr == "" {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "grade, stream, and date are required")
		return
	}

	rows, err := h.pool.Query(r.Context(), `
		SELECT a.id, a.learner_id, l.full_name, a.status, a.reason, a.sms_notified
		FROM attendance a
		JOIN learners l ON l.id = a.learner_id AND l.tenant_id = a.tenant_id
		WHERE a.tenant_id = $1 AND a.date = $2 AND l.grade = $3 AND l.stream = $4
	`, r.Header.Get("X-Tenant-ID"), dateStr, grade, stream)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	defer rows.Close()

	type AttendanceItem struct {
		ID          uuid.UUID `json:"id"`
		LearnerID   uuid.UUID `json:"learner_id"`
		LearnerName string    `json:"learner_name"`
		Status      string    `json:"status"`
		Reason      string    `json:"reason,omitempty"`
		SMSNotified bool      `json:"sms_notified"`
	}
	var items []AttendanceItem
	for rows.Next() {
		var a AttendanceItem
		if err := rows.Scan(&a.ID, &a.LearnerID, &a.LearnerName, &a.Status, &a.Reason, &a.SMSNotified); err != nil {
			httputil.RespondInternalError(w, err)
			return
		}
		items = append(items, a)
	}
	httputil.RespondOK(w, items)
}

func (h *Handler) markClassAttendance(w http.ResponseWriter, r *http.Request) {
	tenantID := uuid.MustParse(r.Header.Get("X-Tenant-ID"))
	staffID, _ := uuid.Parse(r.Header.Get("X-Staff-ID"))
	if staffID == uuid.Nil {
		staffID, _ = uuid.Parse(r.URL.Query().Get("staff_id"))
	}

	var req struct {
		Grade   string             `json:"grade"`
		Stream  string             `json:"stream"`
		Date    string             `json:"date"`
		Records []struct {
			LearnerID uuid.UUID `json:"learner_id"`
			Status    string     `json:"status"`
			Reason    string     `json:"reason"`
		} `json:"records"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "Invalid request body: "+err.Error())
		return
	}

	tx, err := h.pool.Begin(r.Context())
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	defer tx.Rollback(r.Context())

	for _, rec := range req.Records {
		_, _ = tx.Exec(r.Context(), `
			INSERT INTO attendance (tenant_id, learner_id, date, status, marked_by, reason, sms_notified)
			VALUES ($1, $2, $3, $4, $5, $6, false)
			ON CONFLICT (tenant_id, learner_id, date)
			DO UPDATE SET status = EXCLUDED.status, marked_by = EXCLUDED.marked_by, reason = EXCLUDED.reason, updated_at = now()
		`, tenantID, rec.LearnerID, req.Date, rec.Status, staffID, rec.Reason)
	}

	if err := tx.Commit(r.Context()); err != nil {
		httputil.RespondInternalError(w, err)
		return
	}

	httputil.RespondOK(w, map[string]string{"status": "marked"})
}

func (h *Handler) getClassAssessments(w http.ResponseWriter, r *http.Request) {
	grade := r.URL.Query().Get("grade")
	stream := r.URL.Query().Get("stream")
	term, _ := strconv.Atoi(r.URL.Query().Get("term"))
	year, _ := strconv.Atoi(r.URL.Query().Get("year"))
	if grade == "" || stream == "" || term < 1 || year <= 0 {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "grade, stream, term, and year are required")
		return
	}

	rows, err := h.pool.Query(r.Context(), `
		SELECT a.id, a.learner_id, l.full_name, a.sub_strand_id, s.name AS sub_strand_name,
		       str.name AS strand_name, la.name AS learning_area, a.rubric_level, a.note, a.term, a.year
		FROM assessments a
		JOIN learners l ON l.id = a.learner_id AND l.tenant_id = a.tenant_id
		JOIN sub_strands s ON s.id = a.sub_strand_id AND s.tenant_id = a.tenant_id
		JOIN strands str ON str.id = s.strand_id AND str.tenant_id = a.tenant_id
		JOIN learning_areas la ON la.id = str.learning_area_id AND la.tenant_id = a.tenant_id
		WHERE a.tenant_id = $1 AND l.grade = $2 AND l.stream = $3 AND a.term = $4 AND a.year = $5
	`, r.Header.Get("X-Tenant-ID"), grade, stream, term, year)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	defer rows.Close()

	type AssessmentItem struct {
		ID           uuid.UUID `json:"id"`
		LearnerID    uuid.UUID `json:"learner_id"`
		LearnerName  string    `json:"learner_name"`
		SubStrandID  uuid.UUID `json:"sub_strand_id"`
		SubStrandName string   `json:"sub_strand_name"`
		StrandName   string    `json:"strand_name"`
		LearningArea string    `json:"learning_area"`
		RubricLevel  int       `json:"rubric_level"`
		Note         string    `json:"note"`
		Term         int       `json:"term"`
		Year         int       `json:"year"`
	}
	var items []AssessmentItem
	for rows.Next() {
		var a AssessmentItem
		if err := rows.Scan(&a.ID, &a.LearnerID, &a.LearnerName, &a.SubStrandID, &a.SubStrandName,
			&a.StrandName, &a.LearningArea, &a.RubricLevel, &a.Note, &a.Term, &a.Year); err != nil {
			httputil.RespondInternalError(w, err)
			return
		}
		items = append(items, a)
	}
	httputil.RespondOK(w, items)
}

func (h *Handler) createAssessment(w http.ResponseWriter, r *http.Request) {
	tenantID := uuid.MustParse(r.Header.Get("X-Tenant-ID"))
	staffID, _ := uuid.Parse(r.Header.Get("X-Staff-ID"))
	if staffID == uuid.Nil {
		staffID, _ = uuid.Parse(r.URL.Query().Get("staff_id"))
	}

	var req struct {
		LearnerID   uuid.UUID `json:"learner_id"`
		SubStrandID uuid.UUID `json:"sub_strand_id"`
		RubricLevel int       `json:"rubric_level"`
		Note        string    `json:"note"`
		Term        int       `json:"term"`
		Year        int       `json:"year"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "Invalid request body: "+err.Error())
		return
	}

	var id uuid.UUID
	err := h.pool.QueryRow(r.Context(), `
		INSERT INTO assessments (tenant_id, learner_id, sub_strand_id, rubric_level, note, teacher_id, term, year)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`, tenantID, req.LearnerID, req.SubStrandID, req.RubricLevel, req.Note, staffID, req.Term, req.Year).Scan(&id)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondCreated(w, map[string]any{"id": id})
}

func (h *Handler) getLearnerPortfolio(w http.ResponseWriter, r *http.Request) {
	learnerID, err := uuid.Parse(chi.URLParam(r, "learnerId"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid learner ID")
		return
	}

	term, _ := strconv.Atoi(r.URL.Query().Get("term"))
	year, _ := strconv.Atoi(r.URL.Query().Get("year"))
	if term < 1 || year <= 0 {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "term and year are required")
		return
	}

	rows, err := h.pool.Query(r.Context(), `
		SELECT a.id, a.sub_strand_id, s.name, str.name AS strand_name, la.name AS learning_area,
		       a.rubric_level, a.note, a.term, a.year
		FROM assessments a
		JOIN sub_strands s ON s.id = a.sub_strand_id AND s.tenant_id = a.tenant_id
		JOIN strands str ON str.id = s.strand_id AND str.tenant_id = a.tenant_id
		JOIN learning_areas la ON la.id = str.learning_area_id AND la.tenant_id = a.tenant_id
		WHERE a.tenant_id = $1 AND a.learner_id = $2 AND a.term = $3 AND a.year = $4
	`, r.Header.Get("X-Tenant-ID"), learnerID, term, year)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	defer rows.Close()

	type PortfolioItem struct {
		ID           uuid.UUID `json:"id"`
		SubStrandID  uuid.UUID `json:"sub_strand_id"`
		SubStrandName string   `json:"sub_strand_name"`
		StrandName   string    `json:"strand_name"`
		LearningArea string    `json:"learning_area"`
		RubricLevel  int       `json:"rubric_level"`
		Note         string    `json:"note"`
		Term         int       `json:"term"`
		Year         int       `json:"year"`
	}
	var items []PortfolioItem
	for rows.Next() {
		var a PortfolioItem
		if err := rows.Scan(&a.ID, &a.SubStrandID, &a.SubStrandName, &a.StrandName, &a.LearningArea,
			&a.RubricLevel, &a.Note, &a.Term, &a.Year); err != nil {
			httputil.RespondInternalError(w, err)
			return
		}
		items = append(items, a)
	}
	httputil.RespondOK(w, items)
}
