package attendance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/shule360/api/internal/comms"
	"github.com/shule360/api/pkg/apperr"
)

// MessageSender is the part of Communications an absence alert needs: record
// a message and its recipients, and leave the sending to the dispatcher.
type MessageSender interface {
	CreateAndSend(ctx context.Context, tenantID uuid.UUID, actor comms.Actor, req comms.CreateMessageRequest) (*comms.Message, error)
}

// AbsenceAlertService texts the parents of learners marked absent.
//
// An alert is an ordinary message: it appears under Communications with its
// recipients and their delivery, it respects a parent's opt-out, and it is
// sent with the school's own SMS account. The register only says a parent was
// told once that message exists.
type AbsenceAlertService struct {
	pool   *pgxpool.Pool
	sender MessageSender
}

func NewAbsenceAlertService(pool *pgxpool.Pool, sender MessageSender) *AbsenceAlertService {
	return &AbsenceAlertService{pool: pool, sender: sender}
}

// AlertResult says what was done about the day's absences.
type AlertResult struct {
	// Sent is how many learners' parents now have a message on its way.
	Sent int `json:"sent"`
	// Skipped lists the absent learners whose parents were not texted, and why.
	Skipped []AlertSkip `json:"skipped"`
}

// AlertSkip is one absent learner nobody was texted about.
type AlertSkip struct {
	LearnerID   uuid.UUID `json:"learner_id"`
	LearnerName string    `json:"learner_name"`
	Reason      string    `json:"reason"`
}

// AlertAbsences texts the parents of every learner marked absent on date who
// have not been told yet. One message per learner, so a parent with two
// children at the school is told about the one who is absent.
//
// A learner is never alerted about twice for the same day: the message carries
// a key made of the learner and the date, and a repeat returns the first one.
func (s *AbsenceAlertService) AlertAbsences(ctx context.Context, tenantID, staffID uuid.UUID, date time.Time) (*AlertResult, error) {
	if s.sender == nil {
		return nil, apperr.Invalid("Text messages are not set up, so parents cannot be told about absences.")
	}
	day := date.Format(dateOnlyLayout)
	if day != Today().Format(dateOnlyLayout) {
		return nil, apperr.Invalid("Parents are only texted about today's register, not an earlier day's.")
	}

	var school string
	var hasComms bool
	err := s.pool.QueryRow(ctx,
		`SELECT name, modules IS NULL OR 'communications' = ANY(modules) FROM tenants WHERE id = $1`, tenantID).
		Scan(&school, &hasComms)
	if err != nil {
		return nil, fmt.Errorf("load school: %w", err)
	}
	if !hasComms {
		return nil, apperr.Invalid("This school does not have Communications, so parents cannot be texted.")
	}

	rows, err := s.pool.Query(ctx, `
		SELECT a.id, l.id, l.full_name, l.guardian_ids::text[]
		FROM attendance a
		JOIN learners l ON l.id = a.learner_id AND l.tenant_id = a.tenant_id
		WHERE a.tenant_id = $1 AND a.date = $2 AND a.status = 'absent' AND a.sms_notified = false
		ORDER BY l.full_name
	`, tenantID, day)
	if err != nil {
		return nil, fmt.Errorf("query absences: %w", err)
	}
	type absence struct {
		markID, learnerID uuid.UUID
		name              string
		guardians         []string
	}
	var absences []absence
	for rows.Next() {
		var a absence
		if err := rows.Scan(&a.markID, &a.learnerID, &a.name, &a.guardians); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan absence: %w", err)
		}
		absences = append(absences, a)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	result := &AlertResult{Skipped: []AlertSkip{}}
	actor := comms.Actor{}
	if staffID != uuid.Nil {
		actor.StaffID = &staffID
	}
	for _, a := range absences {
		skip := func(reason string) {
			result.Skipped = append(result.Skipped, AlertSkip{LearnerID: a.learnerID, LearnerName: a.name, Reason: reason})
		}
		if len(a.guardians) == 0 {
			skip("No parent is recorded for this learner.")
			continue
		}
		filter, _ := json.Marshal(map[string][]string{"guardian_ids": a.guardians})
		msg, err := s.sender.CreateAndSend(ctx, tenantID, actor, comms.CreateMessageRequest{
			Channel:        "sms",
			AudienceType:   "custom",
			AudienceFilter: filter,
			Content: fmt.Sprintf("%s: %s was marked absent today, %s. If this is unexpected, please contact the school office.",
				school, a.name, date.Format("2 Jan 2006")),
			IdempotencyKey: "absence:" + a.learnerID.String() + ":" + day,
		})
		if err != nil {
			// Nobody to text (opted out, no usable number): say so and carry on
			// with the other learners. Anything else is a fault.
			var refusal *comms.ValidationError
			if errors.As(err, &refusal) {
				skip(refusal.Message)
				continue
			}
			return nil, fmt.Errorf("alert for learner %s: %w", a.learnerID, err)
		}
		if _, err := s.pool.Exec(ctx, `
			UPDATE attendance SET sms_notified = true, alert_message_id = $3, updated_at = now()
			WHERE tenant_id = $1 AND id = $2
		`, tenantID, a.markID, msg.ID); err != nil {
			return nil, fmt.Errorf("record alert: %w", err)
		}
		result.Sent++
	}
	return result, nil
}
