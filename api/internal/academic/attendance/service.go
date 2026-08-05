package attendance

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AttendanceStatus represents the possible attendance states.
type AttendanceStatus string

const (
	AttendancePresent  AttendanceStatus = "present"
	AttendanceAbsent   AttendanceStatus = "absent"
	AttendanceLate     AttendanceStatus = "late"
	AttendanceExcused  AttendanceStatus = "excused"
)

// Attendance represents a single attendance record.
type Attendance struct {
	ID          uuid.UUID      `json:"id"`
	TenantID    uuid.UUID      `json:"tenant_id"`
	LearnerID   uuid.UUID      `json:"learner_id"`
	Date        time.Time      `json:"date"`
	Status      AttendanceStatus `json:"status"`
	MarkedBy    *uuid.UUID     `json:"marked_by,omitempty"`
	Reason      string         `json:"reason,omitempty"`
	SMSNotified bool           `json:"sms_notified"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

// CreateAttendanceRequest is the request payload for marking attendance.
type CreateAttendanceRequest struct {
	LearnerID   uuid.UUID
	Date        time.Time
	Status      AttendanceStatus
	MarkedBy    uuid.UUID
	Reason      string
	SMSNotified bool
}

// AttendanceSummary is a joined view with learner info.
type AttendanceSummary struct {
	ID         uuid.UUID      `json:"id"`
	LearnerID  uuid.UUID      `json:"learner_id"`
	LearnerName string        `json:"learner_name"`
	Grade      string         `json:"grade"`
	Stream     string         `json:"stream"`
	Date       time.Time      `json:"date"`
	Status     AttendanceStatus `json:"status"`
	Reason     string         `json:"reason,omitempty"`
	SMSNotified bool          `json:"sms_notified"`
	CreatedAt  time.Time      `json:"created_at"`
}

// Service handles attendance-related operations.
type Service struct {
	pool *pgxpool.Pool
}

// NewService creates a new attendance service.
func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

// MarkAttendance records or updates an attendance mark for a learner on a date.
func (s *Service) MarkAttendance(ctx context.Context, tenantID uuid.UUID, req CreateAttendanceRequest) (*Attendance, error) {
	var a Attendance
	err := s.pool.QueryRow(ctx, `
		INSERT INTO attendance (tenant_id, learner_id, date, status, marked_by, reason, sms_notified)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (tenant_id, learner_id, date)
		DO UPDATE SET status = EXCLUDED.status, marked_by = EXCLUDED.marked_by, reason = EXCLUDED.reason, sms_notified = EXCLUDED.sms_notified, updated_at = now()
		RETURNING id, tenant_id, learner_id, date, status, marked_by, reason, sms_notified, created_at, updated_at
	`, tenantID, req.LearnerID, req.Date, req.Status, req.MarkedBy, req.Reason, req.SMSNotified).Scan(
		&a.ID, &a.TenantID, &a.LearnerID, &a.Date, &a.Status,
		&a.MarkedBy, &a.Reason, &a.SMSNotified, &a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("mark attendance: %w", err)
	}
	return &a, nil
}

// ListByDate returns attendance records for a specific date.
func (s *Service) ListByDate(ctx context.Context, tenantID uuid.UUID, date time.Time) ([]Attendance, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, tenant_id, learner_id, date, status, marked_by, reason, sms_notified, created_at, updated_at
		FROM attendance
		WHERE tenant_id = $1 AND date = $2
		ORDER BY created_at
	`, tenantID, date)
	if err != nil {
		return nil, fmt.Errorf("query attendance: %w", err)
	}
	defer rows.Close()

	var records []Attendance
	for rows.Next() {
		var a Attendance
		if err := rows.Scan(
			&a.ID, &a.TenantID, &a.LearnerID, &a.Date, &a.Status,
			&a.MarkedBy, &a.Reason, &a.SMSNotified, &a.CreatedAt, &a.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan attendance: %w", err)
		}
		records = append(records, a)
	}
	return records, rows.Err()
}

// ListByLearner returns attendance records for a specific learner.
func (s *Service) ListByLearner(ctx context.Context, tenantID, learnerID uuid.UUID) ([]Attendance, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, tenant_id, learner_id, date, status, marked_by, reason, sms_notified, created_at, updated_at
		FROM attendance
		WHERE tenant_id = $1 AND learner_id = $2
		ORDER BY date DESC
	`, tenantID, learnerID)
	if err != nil {
		return nil, fmt.Errorf("query attendance: %w", err)
	}
	defer rows.Close()

	var records []Attendance
	for rows.Next() {
		var a Attendance
		if err := rows.Scan(
			&a.ID, &a.TenantID, &a.LearnerID, &a.Date, &a.Status,
			&a.MarkedBy, &a.Reason, &a.SMSNotified, &a.CreatedAt, &a.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan attendance: %w", err)
		}
		records = append(records, a)
	}
	return records, rows.Err()
}

// ListSummariesByDate returns attendance summaries joined with learner info for a date.
func (s *Service) ListSummariesByDate(ctx context.Context, tenantID uuid.UUID, date time.Time) ([]AttendanceSummary, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT
			a.id, a.learner_id, l.full_name AS learner_name, l.grade, l.stream,
			a.date, a.status, a.reason, a.sms_notified, a.created_at
		FROM attendance a
		JOIN learners l ON l.id = a.learner_id AND l.tenant_id = a.tenant_id
		WHERE a.tenant_id = $1 AND a.date = $2
		ORDER BY l.full_name
	`, tenantID, date)
	if err != nil {
		return nil, fmt.Errorf("query attendance summaries: %w", err)
	}
	defer rows.Close()

	var summaries []AttendanceSummary
	for rows.Next() {
		var as AttendanceSummary
		if err := rows.Scan(
			&as.ID, &as.LearnerID, &as.LearnerName, &as.Grade, &as.Stream,
			&as.Date, &as.Status, &as.Reason, &as.SMSNotified, &as.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan attendance summary: %w", err)
		}
		summaries = append(summaries, as)
	}
	return summaries, rows.Err()
}

// GetByID returns a single attendance record.
func (s *Service) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*Attendance, error) {
	var a Attendance
	err := s.pool.QueryRow(ctx, `
		SELECT id, tenant_id, learner_id, date, status, marked_by, reason, sms_notified, created_at, updated_at
		FROM attendance
		WHERE tenant_id = $1 AND id = $2
	`, tenantID, id).Scan(
		&a.ID, &a.TenantID, &a.LearnerID, &a.Date, &a.Status,
		&a.MarkedBy, &a.Reason, &a.SMSNotified, &a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("attendance record not found")
		}
		return nil, fmt.Errorf("query attendance: %w", err)
	}
	return &a, nil
}

// Delete removes an attendance record.
func (s *Service) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `
		DELETE FROM attendance
		WHERE tenant_id = $1 AND id = $2
	`, tenantID, id)
	if err != nil {
		return fmt.Errorf("delete attendance: %w", err)
	}
	return nil
}

// ChronicAbsenteeism returns learners with attendance below the threshold.
func (s *Service) ChronicAbsenteeism(ctx context.Context, tenantID uuid.UUID, threshold float64, term int, year int) ([]map[string]interface{}, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT
			l.id AS learner_id,
			l.full_name AS learner_name,
			l.grade,
			l.stream,
			COUNT(*) AS total_days,
			COUNT(*) FILTER (WHERE a.status = 'absent') AS absent_days,
			ROUND(COUNT(*) FILTER (WHERE a.status = 'absent')::numeric / NULLIF(COUNT(*), 0) * 100, 2) AS attendance_rate
		FROM attendance a
		JOIN learners l ON l.id = a.learner_id AND l.tenant_id = a.tenant_id
		WHERE a.tenant_id = $1
		  AND a.date >= date_trunc('month', CURRENT_DATE)
		GROUP BY l.id, l.full_name, l.grade, l.stream
		HAVING COUNT(*) FILTER (WHERE a.status = 'absent')::numeric / NULLIF(COUNT(*), 0) > $2
		ORDER BY attendance_rate DESC
	`, tenantID, threshold)
	if err != nil {
		return nil, fmt.Errorf("query chronic absenteeism: %w", err)
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var row map[string]interface{}
		var learnerID uuid.UUID
		var learnerName, grade, stream string
		var totalDays, absentDays int
		var attendanceRate float64
		if err := rows.Scan(
			&learnerID, &learnerName, &grade, &stream,
			&totalDays, &absentDays, &attendanceRate,
		); err != nil {
			return nil, fmt.Errorf("scan chronic absenteeism: %w", err)
		}
		row = map[string]interface{}{
			"learner_id":     learnerID,
			"learner_name":   learnerName,
			"grade":          grade,
			"stream":         stream,
			"total_days":     totalDays,
			"absent_days":    absentDays,
			"attendance_rate": attendanceRate,
		}
		results = append(results, row)
	}
	return results, rows.Err()
}
