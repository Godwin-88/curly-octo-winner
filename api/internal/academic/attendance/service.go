package attendance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/shule360/api/pkg/apperr"
)

// AttendanceStatus represents the possible attendance states.
type AttendanceStatus string

const (
	AttendancePresent AttendanceStatus = "present"
	AttendanceAbsent  AttendanceStatus = "absent"
	AttendanceLate    AttendanceStatus = "late"
	AttendanceExcused AttendanceStatus = "excused"
)

// Attendance represents a single attendance record.
type Attendance struct {
	ID          uuid.UUID        `json:"id"`
	TenantID    uuid.UUID        `json:"tenant_id"`
	LearnerID   uuid.UUID        `json:"learner_id"`
	Date        time.Time        `json:"date"`
	Status      AttendanceStatus `json:"status"`
	MarkedBy    *uuid.UUID       `json:"marked_by,omitempty"`
	Reason      string           `json:"reason,omitempty"`
	SMSNotified bool             `json:"sms_notified"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
}

// Date is a calendar date that accepts either YYYY-MM-DD or a full RFC3339
// timestamp on the wire.
//
// An attendance register is inherently a calendar date, and Go's time.Time JSON
// decoding accepts *only* RFC3339 — a body of {"date":"2026-09-26"} was
// rejected outright with "parsing time ... cannot parse \"\" as \"T\"", which
// means the single-learner mark endpoint could never be called with a plain
// date either. The same API already takes YYYY-MM-DD for /attendance/date, so
// accepting both here keeps the wire format consistent instead of pushing the
// problem onto every caller.
type Date struct {
	time.Time
}

const (
	dateOnlyLayout = "2006-01-02"
)

// UnmarshalJSON parses a date-only or RFC3339 string.
func (d *Date) UnmarshalJSON(b []byte) error {
	var raw string
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if t, err := time.Parse(dateOnlyLayout, raw); err == nil {
		d.Time = t
		return nil
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return fmt.Errorf("date must be YYYY-MM-DD or RFC3339, got %q", raw)
	}
	d.Time = t
	return nil
}

// MarshalJSON always writes the date-only form, so an API response never grows
// a surprising "T00:00:00Z" suffix.
func (d Date) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.Time.Format(dateOnlyLayout))
}

// CreateAttendanceRequest is the request payload for marking attendance.
//
// The JSON tags matter: without them Go matches incoming keys against the Go
// field names, and that match ignores underscores but not their presence — a
// JSON body of {"learner_id": ...} would silently leave LearnerID as the zero
// UUID and record attendance against nobody.
type CreateAttendanceRequest struct {
	LearnerID uuid.UUID        `json:"learner_id"`
	Date      Date             `json:"date"`
	Status    AttendanceStatus `json:"status"`
	Reason    string           `json:"reason"`
	// MarkedBy is whoever is signed in. Whether a parent was texted is not in
	// the request either: it is recorded when an alert is actually queued.
	MarkedBy uuid.UUID `json:"-"`
}

// eat is the school day's time zone. The API server runs in UTC, where "today"
// is still yesterday until 3 a.m. in Nairobi.
var eat = time.FixedZone("EAT", 3*60*60)

// Today is the current date in Kenya, as a date with no time of day.
func Today() time.Time {
	now := time.Now().In(eat)
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
}

// checkDate refuses a register for a day that has not happened.
func checkDate(d Date) error {
	if d.IsZero() {
		return apperr.Invalid("Choose the date of the register.")
	}
	day := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC)
	if day.After(Today()) {
		return apperr.Invalid("Attendance cannot be marked for a day that has not happened yet.")
	}
	return nil
}

// validStatuses is the closed set the attendance CHECK constraint enforces.
// Validate before writing so the user gets "that is not a status" instead of a
// database error.
var validStatuses = map[AttendanceStatus]bool{
	AttendancePresent: true,
	AttendanceAbsent:  true,
	AttendanceLate:    true,
	AttendanceExcused: true,
}

// ErrInvalidStatus is returned for a status outside the allowed set.
var ErrInvalidStatus = errors.New("status must be one of present, absent, late, excused")

// AttendanceSummary is a joined view with learner info.
type AttendanceSummary struct {
	ID          uuid.UUID        `json:"id"`
	LearnerID   uuid.UUID        `json:"learner_id"`
	LearnerName string           `json:"learner_name"`
	Grade       string           `json:"grade"`
	Stream      string           `json:"stream"`
	Date        time.Time        `json:"date"`
	Status      AttendanceStatus `json:"status"`
	Reason      string           `json:"reason,omitempty"`
	SMSNotified bool             `json:"sms_notified"`
	CreatedAt   time.Time        `json:"created_at"`
}

// Service handles attendance-related operations.
type Service struct {
	pool *pgxpool.Pool
}

// NewService creates a new attendance service.
func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

// markedByArg turns a zero-UUID marker into SQL NULL.
//
// marked_by is nullable and references staff(id), so passing the zero UUID does
// not mean "nobody" to Postgres — it means "this staff member does not exist",
// and the insert fails a foreign key check. Callers with no authenticated staff
// id leave it empty and get a NULL, which is what they mean.
func markedByArg(id uuid.UUID) any {
	if id == uuid.Nil {
		return nil
	}
	return id
}

// MarkAttendance records or updates an attendance mark for a learner on a date.
func (s *Service) MarkAttendance(ctx context.Context, tenantID uuid.UUID, req CreateAttendanceRequest) (*Attendance, error) {
	if !validStatuses[req.Status] {
		return nil, ErrInvalidStatus
	}
	if err := checkDate(req.Date); err != nil {
		return nil, err
	}
	var inSchool bool
	if err := s.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM learners WHERE tenant_id = $1 AND id = $2)`,
		tenantID, req.LearnerID).Scan(&inSchool); err != nil {
		return nil, fmt.Errorf("check learner: %w", err)
	}
	if !inSchool {
		return nil, apperr.Invalid("That learner is not in your school.")
	}
	// A changed mark keeps what is already known about the alert: the parent
	// was either texted or not, whatever the mark says now.
	row := s.pool.QueryRow(ctx, `
		INSERT INTO attendance (tenant_id, learner_id, date, status, marked_by, reason)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (tenant_id, learner_id, date)
		DO UPDATE SET status = EXCLUDED.status, marked_by = EXCLUDED.marked_by, reason = EXCLUDED.reason, updated_at = now()
		RETURNING `+attendanceColumns,
		tenantID, req.LearnerID, req.Date.Time, string(req.Status), markedByArg(req.MarkedBy), strings.TrimSpace(req.Reason))
	a, err := scanAttendance(row)
	if err != nil {
		return nil, fmt.Errorf("mark attendance: %w", err)
	}
	return a, nil
}

// reason is nullable; reading it into a string without COALESCE made every
// list of a learner's attendance fail on the first mark without one.
const attendanceColumns = `id, tenant_id, learner_id, date, status, marked_by, COALESCE(reason, ''), sms_notified, created_at, updated_at`

func scanAttendance(row pgx.Row) (*Attendance, error) {
	var a Attendance
	if err := row.Scan(
		&a.ID, &a.TenantID, &a.LearnerID, &a.Date, &a.Status,
		&a.MarkedBy, &a.Reason, &a.SMSNotified, &a.CreatedAt, &a.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return &a, nil
}

// ListByDate returns attendance records for a specific date.
func (s *Service) ListByDate(ctx context.Context, tenantID uuid.UUID, date time.Time) ([]Attendance, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, tenant_id, learner_id, date, status, marked_by, COALESCE(reason, ''), sms_notified, created_at, updated_at
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
		SELECT id, tenant_id, learner_id, date, status, marked_by, COALESCE(reason, ''), sms_notified, created_at, updated_at
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
			a.id, a.learner_id, l.full_name AS learner_name, l.grade, COALESCE(l.stream, ''),
			a.date, a.status, COALESCE(a.reason, ''), a.sms_notified, a.created_at
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
		SELECT id, tenant_id, learner_id, date, status, marked_by, COALESCE(reason, ''), sms_notified, created_at, updated_at
		FROM attendance
		WHERE tenant_id = $1 AND id = $2
	`, tenantID, id).Scan(
		&a.ID, &a.TenantID, &a.LearnerID, &a.Date, &a.Status,
		&a.MarkedBy, &a.Reason, &a.SMSNotified, &a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperr.ErrNotFound
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
// TermDateRange returns the first and last day of a school term.
//
// Kenyan school terms do not align with calendar quarters — Term 1 runs January
// to April, Term 2 May to August, Term 3 September to December — and a term's
// attendance is meaningless without that window. The end is inclusive, so it is
// the last day of the final month.
func TermDateRange(term, year int) (time.Time, time.Time, error) {
	switch term {
	case 1:
		return time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC),
			time.Date(year, time.April, 30, 0, 0, 0, 0, time.UTC), nil
	case 2:
		return time.Date(year, time.May, 1, 0, 0, 0, 0, time.UTC),
			time.Date(year, time.August, 31, 0, 0, 0, 0, time.UTC), nil
	case 3:
		return time.Date(year, time.September, 1, 0, 0, 0, 0, time.UTC),
			time.Date(year, time.December, 31, 0, 0, 0, 0, time.UTC), nil
	default:
		return time.Time{}, time.Time{}, fmt.Errorf("term must be 1, 2 or 3, got %d", term)
	}
}

// ChronicAbsenteeism lists learners whose attendance rate for the term falls
// below threshold.
func (s *Service) ChronicAbsenteeism(ctx context.Context, tenantID uuid.UUID, threshold float64, term int, year int) ([]map[string]interface{}, error) {
	start, end, err := TermDateRange(term, year)
	if err != nil {
		return nil, err
	}
	if threshold <= 0 || threshold > 100 {
		threshold = 75
	}

	rows, err := s.pool.Query(ctx, `
		SELECT
			l.id AS learner_id,
			l.full_name AS learner_name,
			l.grade,
			COALESCE(l.stream, ''),
			COUNT(*) AS total_days,
			COUNT(*) FILTER (WHERE a.status = 'absent') AS absent_days,
			ROUND(
				COUNT(*) FILTER (WHERE a.status IN ('present', 'late'))::numeric
					/ NULLIF(COUNT(*), 0) * 100, 1
			) AS attendance_rate
		FROM attendance a
		JOIN learners l ON l.id = a.learner_id AND l.tenant_id = a.tenant_id
		WHERE a.tenant_id = $1
		  AND a.date BETWEEN $2 AND $3
		GROUP BY l.id, l.full_name, l.grade, l.stream
		HAVING COUNT(*) FILTER (WHERE a.status IN ('present', 'late'))::numeric
			   / NULLIF(COUNT(*), 0) * 100 < $4
		ORDER BY attendance_rate ASC
	`, tenantID, start, end, threshold)
	if err != nil {
		return nil, fmt.Errorf("query chronic absenteeism: %w", err)
	}
	defer rows.Close()

	results := []map[string]interface{}{}
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
			"learner_id":      learnerID,
			"learner_name":    learnerName,
			"grade":           grade,
			"stream":          stream,
			"total_days":      totalDays,
			"absent_days":     absentDays,
			"attendance_rate": attendanceRate,
		}
		results = append(results, row)
	}
	return results, rows.Err()
}
