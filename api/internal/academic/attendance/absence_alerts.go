package attendance

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/shule360/api/internal/comms/sms"
)

// SMSSender is the interface for sending SMS messages.
type SMSSender interface {
	SendBulk(ctx context.Context, req sms.BulkSMSRequest) ([]sms.SMSResult, error)
}

// AbsenceAlertService handles sending SMS alerts for unexcused absences.
type AbsenceAlertService struct {
	pool      *pgxpool.Pool
	smsSender SMSSender
}

func NewAbsenceAlertService(pool *pgxpool.Pool, smsSender SMSSender) *AbsenceAlertService {
	return &AbsenceAlertService{pool: pool, smsSender: smsSender}
}

// CheckAndAlert finds unexcused absences for a date/class and sends SMS alerts to guardians.
func (s *AbsenceAlertService) CheckAndAlert(ctx context.Context, tenantID uuid.UUID, date time.Time, classGrade, classStream string) error {
	rows, err := s.pool.Query(ctx, `
		SELECT a.id, a.learner_id, l.full_name, g.phone_primary, g.full_name AS guardian_name
		FROM attendance a
		JOIN learners l ON l.id = a.learner_id AND l.tenant_id = a.tenant_id
		LEFT JOIN guardians g ON g.tenant_id = a.tenant_id AND g.id = ANY(l.guardian_ids)
		WHERE a.tenant_id = $1
		  AND a.date = $2
		  AND a.status = 'absent'
		  AND (a.reason IS NULL OR a.reason = '')
		  AND ($3 = '' OR l.grade = $3)
		  AND ($4 = '' OR l.stream = $4)
	`, tenantID, date.Format("2006-01-02"), classGrade, classStream)
	if err != nil {
		return fmt.Errorf("query absences: %w", err)
	}
	defer rows.Close()

	type absence struct {
		id           uuid.UUID
		learnerID    uuid.UUID
		learnerName  string
		phone        string
		guardianName string
	}
	var absences []absence
	for rows.Next() {
		var a absence
		if err := rows.Scan(&a.id, &a.learnerID, &a.learnerName, &a.phone, &a.guardianName); err != nil {
			return fmt.Errorf("scan absence: %w", err)
		}
		if a.phone != "" && a.phone != "null" {
			absences = append(absences, a)
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if len(absences) == 0 {
		return nil
	}

	phones := make([]string, 0, len(absences))
	for _, a := range absences {
		normalized, err := sms.NormalizeKenyanPhone(a.phone)
		if err != nil {
			continue
		}
		phones = append(phones, normalized)
	}

	if len(phones) == 0 {
		return nil
	}

	message := fmt.Sprintf("Dear parent, %s was absent from school today. Please contact the school office if this is unexpected.", "")
	_, _ = s.smsSender.SendBulk(ctx, sms.BulkSMSRequest{
		To:      phones,
		Message: message,
	})

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	for _, a := range absences {
		_, _ = tx.Exec(ctx, `
			UPDATE attendance
			SET sms_notified = true, updated_at = now()
			WHERE tenant_id = $1 AND id = $2
		`, tenantID, a.id)
	}

	return tx.Commit(ctx)
}

func scanAttendanceRow(row pgx.Row) (*Attendance, error) {
	var a Attendance
	err := row.Scan(
		&a.ID, &a.TenantID, &a.LearnerID, &a.Date, &a.Status,
		&a.MarkedBy, &a.Reason, &a.SMSNotified, &a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &a, nil
}
