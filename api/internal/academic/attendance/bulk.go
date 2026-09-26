package attendance

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// BulkMarkRequest marks a whole register in one call.
//
// A register is 30-45 learners; asking the browser to fire 45 separate POSTs
// and then discover the 20th failed leaves the teacher with a half-saved
// register and no way to tell which half. One transaction either records the
// entire register or none of it.
type BulkMarkRequest struct {
	Date   Date                      `json:"date"`
	Marks  []CreateAttendanceRequest `json:"marks"`
	Notify bool                      `json:"notify_guardians"`
}

// BulkMarkResult reports what a bulk save did.
//
// There is no "skipped" list because the write is a single atomic statement:
// either every mark in the register is stored or none of them are, so there is
// no partial outcome left to report.
type BulkMarkResult struct {
	Saved int    `json:"saved"`
	Date  string `json:"date"`
}

// MarkBulk records every mark in a single statement.
//
// The first implementation looped over the marks issuing one INSERT each, which
// cost a network round trip per learner: a 50-learner class took 10.8 seconds
// to save. A teacher staring at a spinner for ten seconds assumes it failed and
// clicks again, so the register is now written by one unnest INSERT with a
// CTE that reports how many rows it touched.
func (s *Service) MarkBulk(ctx context.Context, tenantID uuid.UUID, req BulkMarkRequest) (*BulkMarkResult, error) {
	if len(req.Marks) == 0 {
		return nil, fmt.Errorf("no marks supplied")
	}
	if req.Date.IsZero() {
		return nil, fmt.Errorf("date is required")
	}

	// Reject the whole batch if any mark is invalid, rather than saving 30 valid
	// rows and dropping one — a partially saved register is a lie.
	seen := make(map[uuid.UUID]bool, len(req.Marks))
	learnerIDs := make([]string, 0, len(req.Marks))
	statuses := make([]string, 0, len(req.Marks))
	reasons := make([]string, 0, len(req.Marks))

	for i, m := range req.Marks {
		if m.LearnerID == uuid.Nil {
			return nil, fmt.Errorf("mark %d is missing learner_id", i+1)
		}
		if !validStatuses[m.Status] {
			return nil, fmt.Errorf("mark %d: %w", i+1, ErrInvalidStatus)
		}
		if seen[m.LearnerID] {
			return nil, fmt.Errorf("learner %s appears twice in the same register", m.LearnerID)
		}
		seen[m.LearnerID] = true
		learnerIDs = append(learnerIDs, m.LearnerID.String())
		statuses = append(statuses, string(m.Status))
		reasons = append(reasons, m.Reason)
	}

	// The marker is the same authenticated staff member across the register, so
	// it is passed once rather than as a parallel array.
	markedBy := req.Marks[0].MarkedBy

	var saved int
	err := s.pool.QueryRow(ctx, `
WITH written AS (
INSERT INTO attendance (tenant_id, learner_id, date, status, marked_by, reason, sms_notified)
SELECT $1, u.learner_id::uuid, $2, u.status, $3, u.reason, $4
FROM unnest($5::text[], $6::text[], $7::text[])
AS u(learner_id, status, reason)
ON CONFLICT (tenant_id, learner_id, date)
DO UPDATE SET status = EXCLUDED.status, marked_by = EXCLUDED.marked_by,
              reason = EXCLUDED.reason, sms_notified = EXCLUDED.sms_notified,
              updated_at = now()
RETURNING 1
)
SELECT COUNT(*) FROM written
`, tenantID, req.Date.Time, markedByArg(markedBy), req.Notify,
		learnerIDs, statuses, reasons).Scan(&saved)
	if err != nil {
		// A learner id outside this tenant violates the foreign key. The single
		// statement is atomic, so a bad roster cannot half-save a register.
		return nil, fmt.Errorf("save register: %w", err)
	}

	return &BulkMarkResult{Saved: saved, Date: req.Date.Format(dateOnlyLayout)}, nil
}
