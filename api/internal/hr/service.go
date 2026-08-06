package hr

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Service handles HR domain operations.
type Service struct {
	pool *pgxpool.Pool
}

// NewService creates an HR service.
func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

// --- Staff profile operations ---

const staffColumns = `s.id, s.tenant_id, s.full_name, s.email, s.phone, s.role::text, s.is_active,
	s.tsc_number, s.national_id, s.kra_pin, s.date_of_birth::text, s.gender, s.department,
	s.job_title, s.employment_type, s.hire_date::text, s.qualifications, s.subjects,
	s.employment_history, s.emergency_contact, s.bank_details, s.photo_url, s.created_at, s.updated_at`

func scanStaff(row pgx.Row) (*StaffProfile, error) {
	var st StaffProfile
	err := row.Scan(
		&st.ID, &st.TenantID, &st.FullName, &st.Email, &st.Phone, &st.Role, &st.IsActive,
		&st.TSCNumber, &st.NationalID, &st.KRAPin, &st.DateOfBirth, &st.Gender, &st.Department,
		&st.JobTitle, &st.EmploymentType, &st.HireDate, &st.Qualifications, &st.Subjects,
		&st.EmploymentHistory, &st.EmergencyContact, &st.BankDetails, &st.PhotoURL, &st.CreatedAt, &st.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if st.Qualifications == nil {
		st.Qualifications = []any{}
	}
	if st.Subjects == nil {
		st.Subjects = []any{}
	}
	if st.EmploymentHistory == nil {
		st.EmploymentHistory = []any{}
	}
	if st.EmergencyContact == nil {
		st.EmergencyContact = map[string]any{}
	}
	if st.BankDetails == nil {
		st.BankDetails = map[string]any{}
	}
	return &st, nil
}

// ListStaff returns staff profiles optionally filtered by role/department/employment type.
func (s *Service) ListStaff(ctx context.Context, tenantID uuid.UUID, role, department, employmentType string, includeInactive bool) ([]StaffProfile, error) {
	query := fmt.Sprintf(`SELECT %s FROM staff s WHERE s.tenant_id = $1`, staffColumns)
	args := []any{tenantID}
	argIdx := 2

	if role != "" {
		query += fmt.Sprintf(` AND s.role::text = $%d`, argIdx)
		args = append(args, role)
		argIdx++
	}
	if department != "" {
		query += fmt.Sprintf(` AND s.department = $%d`, argIdx)
		args = append(args, department)
		argIdx++
	}
	if employmentType != "" {
		query += fmt.Sprintf(` AND s.employment_type = $%d`, argIdx)
		args = append(args, employmentType)
		argIdx++
	}
	if !includeInactive {
		query += ` AND s.is_active = true`
	}
	query += ` ORDER BY s.full_name`

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var staff []StaffProfile
	for rows.Next() {
		st, err := scanStaff(rows)
		if err != nil {
			return nil, err
		}
		staff = append(staff, *st)
	}
	return staff, rows.Err()
}

// GetStaff returns a single staff profile.
func (s *Service) GetStaff(ctx context.Context, tenantID, id uuid.UUID) (*StaffProfile, error) {
	query := fmt.Sprintf(`SELECT %s FROM staff s WHERE s.tenant_id = $1 AND s.id = $2`, staffColumns)
	return scanStaff(s.pool.QueryRow(ctx, query, tenantID, id))
}

// CreateStaff inserts a new staff profile.
func (s *Service) CreateStaff(ctx context.Context, tenantID uuid.UUID, req CreateStaffRequest) (*StaffProfile, error) {
	if req.FullName == "" || req.Email == "" {
		return nil, errors.New("full_name and email are required")
	}
	role := req.Role
	if role == "" {
		role = "teacher"
	}
	employmentType := "permanent"
	if req.EmploymentType != nil {
		employmentType = *req.EmploymentType
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	query := fmt.Sprintf(`INSERT INTO staff (tenant_id, full_name, email, phone, role, is_active,
		tsc_number, national_id, kra_pin, date_of_birth, gender, department, job_title,
		employment_type, hire_date, qualifications, subjects, employment_history,
		emergency_contact, bank_details, photo_url)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21)
		RETURNING %s`, staffColumns)
	return scanStaff(s.pool.QueryRow(ctx, query,
		tenantID, req.FullName, req.Email, req.Phone, role, isActive,
		req.TSCNumber, req.NationalID, req.KRAPin, req.DateOfBirth, req.Gender, req.Department, req.JobTitle,
		employmentType, req.HireDate, req.Qualifications, req.Subjects, req.EmploymentHistory,
		req.EmergencyContact, req.BankDetails, req.PhotoURL,
	))
}

// UpdateStaff partially updates a staff profile.
func (s *Service) UpdateStaff(ctx context.Context, tenantID, id uuid.UUID, req UpdateStaffRequest) (*StaffProfile, error) {
	if _, err := s.GetStaff(ctx, tenantID, id); err != nil {
		return nil, err
	}
	query := fmt.Sprintf(`UPDATE staff s SET
		full_name = COALESCE($3, s.full_name),
		email = COALESCE($4, s.email),
		phone = COALESCE($5, s.phone),
		role = COALESCE($6, s.role),
		is_active = COALESCE($7, s.is_active),
		tsc_number = COALESCE($8, s.tsc_number),
		national_id = COALESCE($9, s.national_id),
		kra_pin = COALESCE($10, s.kra_pin),
		date_of_birth = COALESCE($11, s.date_of_birth),
		gender = COALESCE($12, s.gender),
		department = COALESCE($13, s.department),
		job_title = COALESCE($14, s.job_title),
		employment_type = COALESCE($15, s.employment_type),
		hire_date = COALESCE($16, s.hire_date),
		qualifications = COALESCE($17, s.qualifications),
		subjects = COALESCE($18, s.subjects),
		employment_history = COALESCE($19, s.employment_history),
		emergency_contact = COALESCE($20, s.emergency_contact),
		bank_details = COALESCE($21, s.bank_details),
		photo_url = COALESCE($22, s.photo_url)
		WHERE s.tenant_id = $1 AND s.id = $2
		RETURNING %s`, staffColumns)
	return scanStaff(s.pool.QueryRow(ctx, query,
		tenantID, id, req.FullName, req.Email, req.Phone, req.Role, req.IsActive,
		req.TSCNumber, req.NationalID, req.KRAPin, req.DateOfBirth, req.Gender, req.Department, req.JobTitle,
		req.EmploymentType, req.HireDate, req.Qualifications, req.Subjects, req.EmploymentHistory,
		req.EmergencyContact, req.BankDetails, req.PhotoURL,
	))
}

// DeleteStaff deactivates a staff member (soft delete via is_active=false).
func (s *Service) DeleteStaff(ctx context.Context, tenantID, id uuid.UUID) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE staff SET is_active = false WHERE tenant_id = $1 AND id = $2`, tenantID, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// --- Staff documents ---

const staffDocColumns = `id, tenant_id, staff_id, doc_type, file_name, file_url, mime_type, file_size, uploaded_by, created_at`

func scanStaffDoc(row pgx.Row) (*StaffDocument, error) {
	var d StaffDocument
	err := row.Scan(&d.ID, &d.TenantID, &d.StaffID, &d.DocType, &d.FileName, &d.FileURL,
		&d.MimeType, &d.FileSize, &d.UploadedBy, &d.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

// ListStaffDocuments returns documents for a staff member.
func (s *Service) ListStaffDocuments(ctx context.Context, tenantID, staffID uuid.UUID) ([]StaffDocument, error) {
	query := fmt.Sprintf(`SELECT %s FROM staff_documents WHERE tenant_id = $1 AND staff_id = $2 ORDER BY created_at DESC`, staffDocColumns)
	rows, err := s.pool.Query(ctx, query, tenantID, staffID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var docs []StaffDocument
	for rows.Next() {
		d, err := scanStaffDoc(rows)
		if err != nil {
			return nil, err
		}
		docs = append(docs, *d)
	}
	return docs, rows.Err()
}

// CreateStaffDocument adds a document to a staff profile.
func (s *Service) CreateStaffDocument(ctx context.Context, tenantID, staffID uuid.UUID, req CreateStaffDocumentRequest) (*StaffDocument, error) {
	if req.DocType == "" || req.FileName == "" || req.FileURL == "" {
		return nil, errors.New("doc_type, file_name, and file_url are required")
	}
	query := fmt.Sprintf(`INSERT INTO staff_documents (tenant_id, staff_id, doc_type, file_name, file_url, mime_type, file_size, uploaded_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING %s`, staffDocColumns)
	return scanStaffDoc(s.pool.QueryRow(ctx, query,
		tenantID, staffID, req.DocType, req.FileName, req.FileURL, req.MimeType, req.FileSize, req.UploadedBy,
	))
}

// DeleteStaffDocument removes a staff document.
func (s *Service) DeleteStaffDocument(ctx context.Context, tenantID, docID uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM staff_documents WHERE tenant_id = $1 AND id = $2`, tenantID, docID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// --- Payroll operations ---

const payrollColumns = `pr.id, pr.tenant_id, pr.staff_id, st.full_name, pr.month, pr.year,
	pr.basic_salary_cents, pr.allowances_cents, pr.gross_cents, pr.paye_cents, pr.nhif_cents,
	pr.nssf_cents, pr.other_deductions_cents, pr.net_cents, pr.status, pr.paid_at,
	pr.created_by, pr.created_at, pr.updated_at`

func scanPayroll(row pgx.Row) (*PayrollRun, error) {
	var pr PayrollRun
	err := row.Scan(
		&pr.ID, &pr.TenantID, &pr.StaffID, &pr.StaffName, &pr.Month, &pr.Year,
		&pr.BasicSalaryCents, &pr.AllowancesCents, &pr.GrossCents, &pr.PayeCents, &pr.NHIFCents,
		&pr.NSSFCents, &pr.OtherDeductionsCents, &pr.NetCents, &pr.Status, &pr.PaidAt,
		&pr.CreatedBy, &pr.CreatedAt, &pr.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &pr, nil
}

const payrollItemColumns = `id, tenant_id, payroll_run_id, item_type, name, amount_cents, sort_order, created_at`

func scanPayrollItem(row pgx.Row) (*PayrollItem, error) {
	var it PayrollItem
	err := row.Scan(&it.ID, &it.TenantID, &it.PayrollRunID, &it.ItemType, &it.Name,
		&it.AmountCents, &it.SortOrder, &it.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &it, nil
}

func (s *Service) listPayrollItems(ctx context.Context, tenantID, runID uuid.UUID) ([]PayrollItem, error) {
	query := fmt.Sprintf(`SELECT %s FROM payroll_items WHERE tenant_id = $1 AND payroll_run_id = $2 ORDER BY sort_order, created_at`, payrollItemColumns)
	rows, err := s.pool.Query(ctx, query, tenantID, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []PayrollItem
	for rows.Next() {
		it, err := scanPayrollItem(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *it)
	}
	return items, rows.Err()
}

// ListPayrollRuns returns payroll runs optionally filtered by month/year/status.
func (s *Service) ListPayrollRuns(ctx context.Context, tenantID uuid.UUID, month, year int, status string) ([]PayrollRun, error) {
	query := fmt.Sprintf(`SELECT %s FROM payroll_runs pr
		JOIN staff st ON st.id = pr.staff_id
		WHERE pr.tenant_id = $1`, payrollColumns)
	args := []any{tenantID}
	argIdx := 2

	if month > 0 {
		query += fmt.Sprintf(` AND pr.month = $%d`, argIdx)
		args = append(args, month)
		argIdx++
	}
	if year > 0 {
		query += fmt.Sprintf(` AND pr.year = $%d`, argIdx)
		args = append(args, year)
		argIdx++
	}
	if status != "" {
		query += fmt.Sprintf(` AND pr.status = $%d`, argIdx)
		args = append(args, status)
		argIdx++
	}
	query += ` ORDER BY pr.year DESC, pr.month DESC, st.full_name`

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var runs []PayrollRun
	for rows.Next() {
		pr, err := scanPayroll(rows)
		if err != nil {
			return nil, err
		}
		runs = append(runs, *pr)
	}
	return runs, rows.Err()
}

// GetPayrollRun returns a payroll run with items.
func (s *Service) GetPayrollRun(ctx context.Context, tenantID, id uuid.UUID) (*PayrollRun, error) {
	query := fmt.Sprintf(`SELECT %s FROM payroll_runs pr
		JOIN staff st ON st.id = pr.staff_id
		WHERE pr.tenant_id = $1 AND pr.id = $2`, payrollColumns)
	pr, err := scanPayroll(s.pool.QueryRow(ctx, query, tenantID, id))
	if err != nil {
		return nil, err
	}
	items, err := s.listPayrollItems(ctx, tenantID, pr.ID)
	if err != nil {
		return nil, err
	}
	pr.Items = items
	return pr, nil
}

// CreatePayrollRun creates a payroll run and computes gross/net.
func (s *Service) CreatePayrollRun(ctx context.Context, tenantID uuid.UUID, req CreatePayrollRunRequest) (*PayrollRun, error) {
	if req.StaffID == uuid.Nil || req.Month < 1 || req.Month > 12 || req.Year <= 0 {
		return nil, errors.New("staff_id, month (1-12), and year are required")
	}
	if req.BasicSalaryCents < 0 || req.AllowancesCents < 0 || req.PayeCents < 0 ||
		req.NHIFCents < 0 || req.NSSFCents < 0 || req.OtherDeductionsCents < 0 {
		return nil, errors.New("payroll amounts cannot be negative")
	}

	gross := req.BasicSalaryCents + req.AllowancesCents
	net := gross - req.PayeCents - req.NHIFCents - req.NSSFCents - req.OtherDeductionsCents
	if net < 0 {
		net = 0
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	query := fmt.Sprintf(`INSERT INTO payroll_runs (tenant_id, staff_id, month, year,
		basic_salary_cents, allowances_cents, gross_cents, paye_cents, nhif_cents, nssf_cents,
		other_deductions_cents, net_cents, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING %s`, payrollColumns+`
		FROM staff st WHERE st.id = payroll_runs.staff_id`)
	pr, err := scanPayroll(tx.QueryRow(ctx, query,
		tenantID, req.StaffID, req.Month, req.Year,
		req.BasicSalaryCents, req.AllowancesCents, gross, req.PayeCents, req.NHIFCents, req.NSSFCents,
		req.OtherDeductionsCents, net, req.CreatedBy,
	))
	if err != nil {
		return nil, err
	}

	for idx, item := range req.Items {
		itemType := item.ItemType
		if itemType != "earning" && itemType != "deduction" {
			itemType = "earning"
		}
		sortOrder := idx
		if item.SortOrder != nil {
			sortOrder = *item.SortOrder
		}
		itemQuery := fmt.Sprintf(`INSERT INTO payroll_items (tenant_id, payroll_run_id, item_type, name, amount_cents, sort_order)
			VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING %s`, payrollItemColumns)
		it, err := scanPayrollItem(tx.QueryRow(ctx, itemQuery,
			tenantID, pr.ID, itemType, item.Name, item.AmountCents, sortOrder,
		))
		if err != nil {
			return nil, err
		}
		pr.Items = append(pr.Items, *it)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return pr, nil
}

// UpdatePayrollRun partially updates a payroll run and recomputes gross/net.
func (s *Service) UpdatePayrollRun(ctx context.Context, tenantID, id uuid.UUID, req UpdatePayrollRunRequest) (*PayrollRun, error) {
	if _, err := s.GetPayrollRun(ctx, tenantID, id); err != nil {
		return nil, err
	}
	query := fmt.Sprintf(`UPDATE payroll_runs pr SET
		basic_salary_cents = COALESCE($3, pr.basic_salary_cents),
		allowances_cents = COALESCE($4, pr.allowances_cents),
		paye_cents = COALESCE($5, pr.paye_cents),
		nhif_cents = COALESCE($6, pr.nhif_cents),
		nssf_cents = COALESCE($7, pr.nssf_cents),
		other_deductions_cents = COALESCE($8, pr.other_deductions_cents),
		status = COALESCE($9, pr.status),
		gross_cents = COALESCE($3, pr.basic_salary_cents) + COALESCE($4, pr.allowances_cents),
		net_cents = GREATEST(COALESCE($3, pr.basic_salary_cents) + COALESCE($4, pr.allowances_cents)
			- COALESCE($5, pr.paye_cents) - COALESCE($6, pr.nhif_cents)
			- COALESCE($7, pr.nssf_cents) - COALESCE($8, pr.other_deductions_cents), 0),
		paid_at = CASE WHEN $9 = 'paid' THEN COALESCE(pr.paid_at, now()) ELSE pr.paid_at END
		WHERE pr.tenant_id = $1 AND pr.id = $2
		RETURNING %s`, payrollColumns+`
		FROM staff st WHERE st.id = pr.staff_id`)
	pr, err := scanPayroll(s.pool.QueryRow(ctx, query,
		tenantID, id, req.BasicSalaryCents, req.AllowancesCents, req.PayeCents, req.NHIFCents,
		req.NSSFCents, req.OtherDeductionsCents, req.Status,
	))
	if err != nil {
		return nil, err
	}
	items, err := s.listPayrollItems(ctx, tenantID, pr.ID)
	if err != nil {
		return nil, err
	}
	pr.Items = items
	return pr, nil
}

// DeletePayrollRun removes a payroll run.
func (s *Service) DeletePayrollRun(ctx context.Context, tenantID, id uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM payroll_runs WHERE tenant_id = $1 AND id = $2`, tenantID, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
