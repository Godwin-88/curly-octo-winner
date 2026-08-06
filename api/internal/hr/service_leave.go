package hr

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// --- Leave operations ---

const leaveColumns = `lr.id, lr.tenant_id, lr.staff_id, st.full_name, lr.leave_type,
	lr.start_date::text, lr.end_date::text, lr.days, lr.reason, lr.status,
	lr.approved_by, lr.approved_at, lr.denial_reason, lr.substitute_id, lr.created_at, lr.updated_at`

func scanLeave(row pgx.Row) (*LeaveRequest, error) {
	var lr LeaveRequest
	err := row.Scan(
		&lr.ID, &lr.TenantID, &lr.StaffID, &lr.StaffName, &lr.LeaveType,
		&lr.StartDate, &lr.EndDate, &lr.Days, &lr.Reason, &lr.Status,
		&lr.ApprovedBy, &lr.ApprovedAt, &lr.DenialReason, &lr.SubstituteID, &lr.CreatedAt, &lr.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &lr, nil
}

// ListLeaveRequests returns leave requests optionally filtered by status/staff/type.
func (s *Service) ListLeaveRequests(ctx context.Context, tenantID uuid.UUID, status, staffID, leaveType string) ([]LeaveRequest, error) {
	query := fmt.Sprintf(`SELECT %s FROM leave_requests lr
		JOIN staff st ON st.id = lr.staff_id
		WHERE lr.tenant_id = $1`, leaveColumns)
	args := []any{tenantID}
	argIdx := 2

	if status != "" {
		query += fmt.Sprintf(` AND lr.status = $%d`, argIdx)
		args = append(args, status)
		argIdx++
	}
	if staffID != "" {
		query += fmt.Sprintf(` AND lr.staff_id = $%d`, argIdx)
		args = append(args, staffID)
		argIdx++
	}
	if leaveType != "" {
		query += fmt.Sprintf(` AND lr.leave_type = $%d`, argIdx)
		args = append(args, leaveType)
		argIdx++
	}
	query += ` ORDER BY lr.created_at DESC`

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var leaves []LeaveRequest
	for rows.Next() {
		lr, err := scanLeave(rows)
		if err != nil {
			return nil, err
		}
		leaves = append(leaves, *lr)
	}
	return leaves, rows.Err()
}

// GetLeaveRequest returns a single leave request.
func (s *Service) GetLeaveRequest(ctx context.Context, tenantID, id uuid.UUID) (*LeaveRequest, error) {
	query := fmt.Sprintf(`SELECT %s FROM leave_requests lr
		JOIN staff st ON st.id = lr.staff_id
		WHERE lr.tenant_id = $1 AND lr.id = $2`, leaveColumns)
	return scanLeave(s.pool.QueryRow(ctx, query, tenantID, id))
}

// CreateLeaveRequest creates a leave request and computes days.
func (s *Service) CreateLeaveRequest(ctx context.Context, tenantID uuid.UUID, req CreateLeaveRequest) (*LeaveRequest, error) {
	if req.StaffID == uuid.Nil || req.LeaveType == "" || req.StartDate == "" || req.EndDate == "" {
		return nil, errors.New("staff_id, leave_type, start_date, and end_date are required")
	}
	start, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		return nil, errors.New("invalid start_date (expected YYYY-MM-DD)")
	}
	end, err := time.Parse("2006-01-02", req.EndDate)
	if err != nil {
		return nil, errors.New("invalid end_date (expected YYYY-MM-DD)")
	}
	if end.Before(start) {
		return nil, errors.New("end_date must be on or after start_date")
	}
	days := int(end.Sub(start).Hours()/24) + 1

	query := fmt.Sprintf(`INSERT INTO leave_requests (tenant_id, staff_id, leave_type, start_date, end_date, days, reason, substitute_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING %s`, leaveColumns+`
		FROM staff st WHERE st.id = leave_requests.staff_id`)
	return scanLeave(s.pool.QueryRow(ctx, query,
		tenantID, req.StaffID, req.LeaveType, req.StartDate, req.EndDate, days, req.Reason, req.SubstituteID,
	))
}

// ApproveLeaveRequest approves a pending leave request.
func (s *Service) ApproveLeaveRequest(ctx context.Context, tenantID, id uuid.UUID, req ApproveLeaveRequest) (*LeaveRequest, error) {
	query := fmt.Sprintf(`UPDATE leave_requests lr SET
		status = 'approved',
		approved_by = COALESCE($3, lr.approved_by),
		approved_at = now(),
		substitute_id = COALESCE($4, lr.substitute_id)
		WHERE lr.tenant_id = $1 AND lr.id = $2 AND lr.status = 'pending'
		RETURNING %s`, leaveColumns+`
		FROM staff st WHERE st.id = lr.staff_id`)
	return scanLeave(s.pool.QueryRow(ctx, query, tenantID, id, req.ApprovedBy, req.SubstituteID))
}

// DenyLeaveRequest denies a pending leave request.
func (s *Service) DenyLeaveRequest(ctx context.Context, tenantID, id uuid.UUID, req DenyLeaveRequest) (*LeaveRequest, error) {
	query := fmt.Sprintf(`UPDATE leave_requests lr SET
		status = 'denied',
		approved_by = COALESCE($3, lr.approved_by),
		approved_at = now(),
		denial_reason = COALESCE($4, lr.denial_reason)
		WHERE lr.tenant_id = $1 AND lr.id = $2 AND lr.status = 'pending'
		RETURNING %s`, leaveColumns+`
		FROM staff st WHERE st.id = lr.staff_id`)
	return scanLeave(s.pool.QueryRow(ctx, query, tenantID, id, req.ApprovedBy, req.DenialReason))
}

// CancelLeaveRequest cancels a leave request.
func (s *Service) CancelLeaveRequest(ctx context.Context, tenantID, id uuid.UUID) (*LeaveRequest, error) {
	query := fmt.Sprintf(`UPDATE leave_requests lr SET status = 'cancelled'
		WHERE lr.tenant_id = $1 AND lr.id = $2 AND lr.status IN ('pending', 'approved')
		RETURNING %s`, leaveColumns+`
		FROM staff st WHERE st.id = lr.staff_id`)
	return scanLeave(s.pool.QueryRow(ctx, query, tenantID, id))
}

// --- Staff attendance operations ---

const attendanceColumns = `sa.id, sa.tenant_id, sa.staff_id, st.full_name, sa.date::text,
	sa.clock_in, sa.clock_out, sa.status, sa.notes, sa.marked_by, sa.created_at, sa.updated_at`

func scanAttendance(row pgx.Row) (*StaffAttendance, error) {
	var a StaffAttendance
	err := row.Scan(
		&a.ID, &a.TenantID, &a.StaffID, &a.StaffName, &a.Date,
		&a.ClockIn, &a.ClockOut, &a.Status, &a.Notes, &a.MarkedBy, &a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// ListStaffAttendance returns attendance records optionally filtered by date/staff/status.
func (s *Service) ListStaffAttendance(ctx context.Context, tenantID uuid.UUID, date, staffID, status string) ([]StaffAttendance, error) {
	query := fmt.Sprintf(`SELECT %s FROM staff_attendance sa
		JOIN staff st ON st.id = sa.staff_id
		WHERE sa.tenant_id = $1`, attendanceColumns)
	args := []any{tenantID}
	argIdx := 2

	if date != "" {
		query += fmt.Sprintf(` AND sa.date = $%d`, argIdx)
		args = append(args, date)
		argIdx++
	}
	if staffID != "" {
		query += fmt.Sprintf(` AND sa.staff_id = $%d`, argIdx)
		args = append(args, staffID)
		argIdx++
	}
	if status != "" {
		query += fmt.Sprintf(` AND sa.status = $%d`, argIdx)
		args = append(args, status)
		argIdx++
	}
	query += ` ORDER BY sa.date DESC, st.full_name`

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []StaffAttendance
	for rows.Next() {
		a, err := scanAttendance(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, *a)
	}
	return records, rows.Err()
}

// CreateStaffAttendance upserts a daily attendance record.
func (s *Service) CreateStaffAttendance(ctx context.Context, tenantID uuid.UUID, req CreateStaffAttendanceRequest) (*StaffAttendance, error) {
	if req.StaffID == uuid.Nil || req.Date == "" {
		return nil, errors.New("staff_id and date are required")
	}
	status := "present"
	if req.Status != nil {
		status = *req.Status
	}
	query := fmt.Sprintf(`INSERT INTO staff_attendance (tenant_id, staff_id, date, clock_in, clock_out, status, notes, marked_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (tenant_id, staff_id, date) DO UPDATE SET
			clock_in = COALESCE(EXCLUDED.clock_in, staff_attendance.clock_in),
			clock_out = COALESCE(EXCLUDED.clock_out, staff_attendance.clock_out),
			status = EXCLUDED.status,
			notes = COALESCE(EXCLUDED.notes, staff_attendance.notes),
			marked_by = COALESCE(EXCLUDED.marked_by, staff_attendance.marked_by)
		RETURNING %s`, attendanceColumns+`
		FROM staff st WHERE st.id = staff_attendance.staff_id`)
	return scanAttendance(s.pool.QueryRow(ctx, query,
		tenantID, req.StaffID, req.Date, req.ClockIn, req.ClockOut, status, req.Notes, req.MarkedBy,
	))
}

// UpdateStaffAttendance partially updates an attendance record.
func (s *Service) UpdateStaffAttendance(ctx context.Context, tenantID, id uuid.UUID, req UpdateStaffAttendanceRequest) (*StaffAttendance, error) {
	query := fmt.Sprintf(`UPDATE staff_attendance sa SET
		clock_in = COALESCE($3, sa.clock_in),
		clock_out = COALESCE($4, sa.clock_out),
		status = COALESCE($5, sa.status),
		notes = COALESCE($6, sa.notes)
		WHERE sa.tenant_id = $1 AND sa.id = $2
		RETURNING %s`, attendanceColumns+`
		FROM staff st WHERE st.id = sa.staff_id`)
	return scanAttendance(s.pool.QueryRow(ctx, query, tenantID, id, req.ClockIn, req.ClockOut, req.Status, req.Notes))
}

// DeleteStaffAttendance removes an attendance record.
func (s *Service) DeleteStaffAttendance(ctx context.Context, tenantID, id uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM staff_attendance WHERE tenant_id = $1 AND id = $2`, tenantID, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// --- Appraisal operations ---

const appraisalColumns = `sa.id, sa.tenant_id, sa.staff_id, st.full_name, sa.year, sa.term,
	sa.appraiser_id, sa.scores, sa.overall_score, sa.rating, sa.comments, sa.status,
	sa.created_at, sa.updated_at`

func scanAppraisal(row pgx.Row) (*StaffAppraisal, error) {
	var a StaffAppraisal
	err := row.Scan(
		&a.ID, &a.TenantID, &a.StaffID, &a.StaffName, &a.Year, &a.Term,
		&a.AppraiserID, &a.Scores, &a.OverallScore, &a.Rating, &a.Comments, &a.Status,
		&a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if a.Scores == nil {
		a.Scores = map[string]any{}
	}
	return &a, nil
}

// ListAppraisals returns appraisals optionally filtered by staff/year/term.
func (s *Service) ListAppraisals(ctx context.Context, tenantID uuid.UUID, staffID string, year, term int) ([]StaffAppraisal, error) {
	query := fmt.Sprintf(`SELECT %s FROM staff_appraisals sa
		JOIN staff st ON st.id = sa.staff_id
		WHERE sa.tenant_id = $1`, appraisalColumns)
	args := []any{tenantID}
	argIdx := 2

	if staffID != "" {
		query += fmt.Sprintf(` AND sa.staff_id = $%d`, argIdx)
		args = append(args, staffID)
		argIdx++
	}
	if year > 0 {
		query += fmt.Sprintf(` AND sa.year = $%d`, argIdx)
		args = append(args, year)
		argIdx++
	}
	if term > 0 {
		query += fmt.Sprintf(` AND sa.term = $%d`, argIdx)
		args = append(args, term)
		argIdx++
	}
	query += ` ORDER BY sa.year DESC, sa.term DESC, st.full_name`

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var appraisals []StaffAppraisal
	for rows.Next() {
		a, err := scanAppraisal(rows)
		if err != nil {
			return nil, err
		}
		appraisals = append(appraisals, *a)
	}
	return appraisals, rows.Err()
}

// GetAppraisal returns a single appraisal.
func (s *Service) GetAppraisal(ctx context.Context, tenantID, id uuid.UUID) (*StaffAppraisal, error) {
	query := fmt.Sprintf(`SELECT %s FROM staff_appraisals sa
		JOIN staff st ON st.id = sa.staff_id
		WHERE sa.tenant_id = $1 AND sa.id = $2`, appraisalColumns)
	return scanAppraisal(s.pool.QueryRow(ctx, query, tenantID, id))
}

// CreateAppraisal creates a TSC-aligned appraisal.
func (s *Service) CreateAppraisal(ctx context.Context, tenantID uuid.UUID, req CreateAppraisalRequest) (*StaffAppraisal, error) {
	if req.StaffID == uuid.Nil || req.Year <= 0 {
		return nil, errors.New("staff_id and year are required")
	}
	query := fmt.Sprintf(`INSERT INTO staff_appraisals (tenant_id, staff_id, year, term, appraiser_id, scores, comments)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING %s`, appraisalColumns+`
		FROM staff st WHERE st.id = staff_appraisals.staff_id`)
	return scanAppraisal(s.pool.QueryRow(ctx, query,
		tenantID, req.StaffID, req.Year, req.Term, req.AppraiserID, req.Scores, req.Comments,
	))
}

// UpdateAppraisal partially updates an appraisal.
func (s *Service) UpdateAppraisal(ctx context.Context, tenantID, id uuid.UUID, req UpdateAppraisalRequest) (*StaffAppraisal, error) {
	query := fmt.Sprintf(`UPDATE staff_appraisals sa SET
		scores = COALESCE($3, sa.scores),
		overall_score = COALESCE($4, sa.overall_score),
		rating = COALESCE($5, sa.rating),
		comments = COALESCE($6, sa.comments),
		status = COALESCE($7, sa.status)
		WHERE sa.tenant_id = $1 AND sa.id = $2
		RETURNING %s`, appraisalColumns+`
		FROM staff st WHERE st.id = sa.staff_id`)
	return scanAppraisal(s.pool.QueryRow(ctx, query,
		tenantID, id, req.Scores, req.OverallScore, req.Rating, req.Comments, req.Status,
	))
}

// DeleteAppraisal removes an appraisal.
func (s *Service) DeleteAppraisal(ctx context.Context, tenantID, id uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM staff_appraisals WHERE tenant_id = $1 AND id = $2`, tenantID, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
