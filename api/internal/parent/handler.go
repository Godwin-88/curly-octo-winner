package parent

import (
	"net/http"
	"time"

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
	r.Route("/parent", func(r chi.Router) {
		r.Get("/dashboard", h.dashboard)
		r.Get("/learners", h.listLearners)
		r.Get("/results", h.listResults)
		r.Get("/fees", h.listFees)
		r.Get("/transport", h.listTransport)
	})
}

func (h *Handler) dashboard(w http.ResponseWriter, r *http.Request) {
	guardianID, err := uuid.Parse(r.Header.Get("X-Guardian-ID"))
	if err != nil {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Guardian ID not found")
		return
	}

	tenantID, err := uuid.Parse(r.Header.Get("X-Tenant-ID"))
	if err != nil {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}

	rows, err := h.pool.Query(r.Context(), `
		SELECT l.id, l.full_name, l.grade, l.stream
		FROM learners l
		WHERE l.tenant_id = $1 AND $2 = ANY(l.guardian_ids)
	`, tenantID, guardianID)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	defer rows.Close()

	type LearnerBrief struct {
		ID       uuid.UUID `json:"id"`
		FullName string    `json:"full_name"`
		Grade    string    `json:"grade"`
		Stream   string    `json:"stream"`
	}
	var learners []LearnerBrief
	for rows.Next() {
		var l LearnerBrief
		if err := rows.Scan(&l.ID, &l.FullName, &l.Grade, &l.Stream); err != nil {
			httputil.RespondInternalError(w, err)
			return
		}
		learners = append(learners, l)
	}

	httputil.RespondOK(w, map[string]any{
		"learners": learners,
		"message":  "Welcome to the parent portal",
	})
}

func (h *Handler) listLearners(w http.ResponseWriter, r *http.Request) {
	guardianID, err := uuid.Parse(r.Header.Get("X-Guardian-ID"))
	if err != nil {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Guardian ID not found")
		return
	}
	tenantID, err := uuid.Parse(r.Header.Get("X-Tenant-ID"))
	if err != nil {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}

	rows, err := h.pool.Query(r.Context(), `
		SELECT l.id, l.full_name, l.grade, l.stream, l.upi
		FROM learners l
		WHERE l.tenant_id = $1 AND $2 = ANY(l.guardian_ids)
	`, tenantID, guardianID)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	defer rows.Close()

	type LearnerBrief struct {
		ID       uuid.UUID `json:"id"`
		FullName string    `json:"full_name"`
		Grade    string    `json:"grade"`
		Stream   string    `json:"stream"`
		UPI      string    `json:"upi"`
	}
	var learners []LearnerBrief
	for rows.Next() {
		var l LearnerBrief
		if err := rows.Scan(&l.ID, &l.FullName, &l.Grade, &l.Stream, &l.UPI); err != nil {
			httputil.RespondInternalError(w, err)
			return
		}
		learners = append(learners, l)
	}
	httputil.RespondOK(w, learners)
}

func (h *Handler) listResults(w http.ResponseWriter, r *http.Request) {
	guardianID, err := uuid.Parse(r.Header.Get("X-Guardian-ID"))
	if err != nil {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Guardian ID not found")
		return
	}
	tenantID, err := uuid.Parse(r.Header.Get("X-Tenant-ID"))
	if err != nil {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}

	learnerID := r.URL.Query().Get("learner_id")
	if learnerID == "" {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "learner_id is required")
		return
	}

	rows, err := h.pool.Query(r.Context(), `
		SELECT rc.id, rc.learner_id, l.full_name, rc.term, rc.year, rc.status, rc.overall_rating, rc.generated_at
		FROM report_cards rc
		JOIN learners l ON l.id = rc.learner_id AND l.tenant_id = rc.tenant_id
		WHERE rc.tenant_id = $1 AND rc.learner_id = $2 AND $3 = ANY(l.guardian_ids)
		  AND rc.status = 'final'
		ORDER BY rc.year DESC, rc.term DESC
	`, tenantID, learnerID, guardianID)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	defer rows.Close()

	type ReportCardBrief struct {
		ID            uuid.UUID `json:"id"`
		LearnerID     uuid.UUID `json:"learner_id"`
		LearnerName   string    `json:"learner_name"`
		Term          int       `json:"term"`
		Year          int       `json:"year"`
		Status        string    `json:"status"`
		OverallRating *int      `json:"overall_rating,omitempty"`
		GeneratedAt   time.Time `json:"generated_at"`
	}
	var cards []ReportCardBrief
	for rows.Next() {
		var c ReportCardBrief
		if err := rows.Scan(&c.ID, &c.LearnerID, &c.LearnerName, &c.Term, &c.Year, &c.Status, &c.OverallRating, &c.GeneratedAt); err != nil {
			httputil.RespondInternalError(w, err)
			return
		}
		cards = append(cards, c)
	}
	httputil.RespondOK(w, cards)
}

func (h *Handler) listFees(w http.ResponseWriter, r *http.Request) {
	guardianID, err := uuid.Parse(r.Header.Get("X-Guardian-ID"))
	if err != nil {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Guardian ID not found")
		return
	}
	tenantID, err := uuid.Parse(r.Header.Get("X-Tenant-ID"))
	if err != nil {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}

	rows, err := h.pool.Query(r.Context(), `
		SELECT i.id, i.learner_id, l.full_name, i.invoice_number, i.term, i.year,
		       i.total_cents, i.paid_cents, i.balance_cents, i.status, i.due_date
		FROM invoices i
		JOIN learners l ON l.id = i.learner_id
		WHERE i.tenant_id = $1 AND $2 = ANY(l.guardian_ids)
	`, tenantID, guardianID)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	defer rows.Close()

	type InvoiceBrief struct {
		ID            uuid.UUID `json:"id"`
		LearnerID     uuid.UUID `json:"learner_id"`
		LearnerName   string    `json:"learner_name"`
		InvoiceNumber string    `json:"invoice_number"`
		Term          int       `json:"term"`
		Year          int       `json:"year"`
		TotalCents    int64     `json:"total_cents"`
		PaidCents     int64     `json:"paid_cents"`
		BalanceCents  int64     `json:"balance_cents"`
		Status        string    `json:"status"`
		DueDate       *string   `json:"due_date,omitempty"`
	}
	var invoices []InvoiceBrief
	for rows.Next() {
		var inv InvoiceBrief
		if err := rows.Scan(&inv.ID, &inv.LearnerID, &inv.LearnerName, &inv.InvoiceNumber,
			&inv.Term, &inv.Year, &inv.TotalCents, &inv.PaidCents, &inv.BalanceCents,
			&inv.Status, &inv.DueDate); err != nil {
			httputil.RespondInternalError(w, err)
			return
		}
		invoices = append(invoices, inv)
	}
	httputil.RespondOK(w, invoices)
}

func (h *Handler) listTransport(w http.ResponseWriter, r *http.Request) {
	guardianID, err := uuid.Parse(r.Header.Get("X-Guardian-ID"))
	if err != nil {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Guardian ID not found")
		return
	}
	tenantID, err := uuid.Parse(r.Header.Get("X-Tenant-ID"))
	if err != nil {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}

	rows, err := h.pool.Query(r.Context(), `
		SELECT t.id, t.route_id, r.name AS route_name, t.direction, t.status,
		       t.scheduled_departure, t.actual_departure, t.boarded_count
		FROM trips t
		JOIN routes r ON r.id = t.route_id
		WHERE t.tenant_id = $1 AND t.status IN ('scheduled', 'in_progress')
		  AND EXISTS (
		      SELECT 1 FROM assignments a
		      JOIN learners l ON l.id = a.learner_id
		      WHERE a.route_id = t.route_id AND $2 = ANY(l.guardian_ids)
		  )
	`, tenantID, guardianID)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	defer rows.Close()

	type TripBrief struct {
		ID                 uuid.UUID `json:"id"`
		RouteID            uuid.UUID `json:"route_id"`
		RouteName          string    `json:"route_name"`
		Direction          string    `json:"direction"`
		Status             string    `json:"status"`
		ScheduledDeparture string    `json:"scheduled_departure"`
		ActualDeparture    *string   `json:"actual_departure,omitempty"`
		BoardedCount       int       `json:"boarded_count"`
	}
	var trips []TripBrief
	for rows.Next() {
		var t TripBrief
		if err := rows.Scan(&t.ID, &t.RouteID, &t.RouteName, &t.Direction, &t.Status,
			&t.ScheduledDeparture, &t.ActualDeparture, &t.BoardedCount); err != nil {
			httputil.RespondInternalError(w, err)
			return
		}
		trips = append(trips, t)
	}
	httputil.RespondOK(w, trips)
}
