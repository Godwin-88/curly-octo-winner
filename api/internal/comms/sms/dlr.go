package sms

import (
	"context"
	"crypto/subtle"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DLRHandler receives Africa's Talking delivery reports.
//
// Africa's Talking does not sign its callbacks, so the URL itself carries a
// secret: the route is /webhooks/sms/dlr/{token} and the token is compared in
// constant time. Without a configured token the endpoint answers 404, so an
// unauthenticated caller can never mark a school's messages delivered.
type DLRHandler struct {
	pool  *pgxpool.Pool
	token string
}

func NewDLRHandler(pool *pgxpool.Pool, token string) *DLRHandler {
	return &DLRHandler{pool: pool, token: strings.TrimSpace(token)}
}

// Mount registers the callback route.
func (h *DLRHandler) Mount(r chi.Router) {
	r.Post("/webhooks/sms/dlr/{token}", h.receive)
}

// DLRStatus maps a delivery-report status to the log status it sets.
// "" means the report is not final (Sent, Submitted, Buffered) and changes
// nothing.
func DLRStatus(reported string) string {
	switch strings.ToLower(strings.TrimSpace(reported)) {
	case "success":
		return "delivered"
	case "failed", "rejected":
		return "failed"
	default:
		return ""
	}
}

func (h *DLRHandler) receive(w http.ResponseWriter, r *http.Request) {
	given := chi.URLParam(r, "token")
	if h.token == "" || subtle.ConstantTimeCompare([]byte(given), []byte(h.token)) != 1 {
		http.NotFound(w, r)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	id := strings.TrimSpace(r.PostForm.Get("id"))
	reported := r.PostForm.Get("status")
	reason := strings.TrimSpace(r.PostForm.Get("failureReason"))

	if id == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}

	matched, err := h.Apply(r.Context(), id, reported, reason)
	if err != nil {
		// A 5xx makes Africa's Talking retry, which is what we want when the
		// database is briefly unavailable.
		slog.Error("sms dlr: update failed", "provider_message_id", id, "error", err)
		http.Error(w, "temporarily unavailable", http.StatusServiceUnavailable)
		return
	}
	if !matched {
		// Not ours, or already final. Answer 200 so it is not retried forever.
		slog.Info("sms dlr: no matching message", "provider_message_id", id, "status", reported)
	}
	w.WriteHeader(http.StatusOK)
}

// Apply records one delivery report. It returns false when no log row changed:
// the id is unknown, the status is not final, or the row is already delivered.
func (h *DLRHandler) Apply(ctx context.Context, providerMessageID, reported, failureReason string) (bool, error) {
	status := DLRStatus(reported)
	if status == "" {
		return false, nil
	}

	tx, err := h.pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after commit

	// `delivered` is terminal: a late or out-of-order "Failed" for a message the
	// handset already acknowledged must not undo it.
	rows, err := tx.Query(ctx, `
		UPDATE message_logs
		SET status = $2::text,
		    delivered_at = CASE WHEN $2::text = 'delivered' THEN now() ELSE delivered_at END,
		    error_code = CASE WHEN $2::text = 'failed' THEN 'DELIVERY_FAILED' ELSE NULL END,
		    error_message = CASE WHEN $2::text = 'failed' THEN NULLIF($3::text, '') ELSE NULL END,
		    updated_at = now()
		WHERE provider_message_id = $1
		  AND channel = 'sms'
		  AND status <> 'delivered'
		RETURNING message_id
	`, providerMessageID, status, failureReason)
	if err != nil {
		return false, err
	}
	var messageIDs []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return false, err
		}
		messageIDs = append(messageIDs, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return false, err
	}
	if len(messageIDs) == 0 {
		return false, nil
	}

	for _, id := range messageIDs {
		if _, err := tx.Exec(ctx, RefreshCountsSQL, id); err != nil {
			return false, err
		}
	}
	return true, tx.Commit(ctx)
}

// RefreshCountsSQL recomputes a message's counters from its delivery log, the
// only source of truth for them. $1 is the message id.
const RefreshCountsSQL = `
	UPDATE messages m
	SET delivered_count = c.delivered,
	    failed_count = c.failed,
	    updated_at = now()
	FROM (
		SELECT COUNT(*) FILTER (WHERE status IN ('delivered', 'read')) AS delivered,
		       COUNT(*) FILTER (WHERE status = 'failed') AS failed
		FROM message_logs
		WHERE message_id = $1
	) c
	WHERE m.id = $1
`
