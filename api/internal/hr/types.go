package hr

import (
	"time"

	"github.com/google/uuid"
)

// StaffProfile is the HR profile for a staff member.
type StaffProfile struct {
	ID                uuid.UUID      `json:"id"`
	TenantID          uuid.UUID      `json:"tenant_id"`
	FullName          string         `json:"full_name"`
	Email             string         `json:"email"`
	Phone             *string        `json:"phone,omitempty"`
	Role              string         `json:"role"`
	IsActive          bool           `json:"is_active"`
	TSCNumber         *string        `json:"tsc_number,omitempty"`
	NationalID        *string        `json:"national_id,omitempty"`
	KRAPin            *string        `json:"kra_pin,omitempty"`
	DateOfBirth       *string        `json:"date_of_birth,omitempty"`
	Gender            *string        `json:"gender,omitempty"`
	Department        *string        `json:"department,omitempty"`
	JobTitle          *string        `json:"job_title,omitempty"`
	EmploymentType    string         `json:"employment_type"`
	HireDate          *string        `json:"hire_date,omitempty"`
	Qualifications    []any          `json:"qualifications,omitempty"`
	Subjects          []any          `json:"subjects,omitempty"`
	EmploymentHistory []any          `json:"employment_history,omitempty"`
	EmergencyContact  map[string]any `json:"emergency_contact,omitempty"`
	BankDetails       map[string]any `json:"bank_details,omitempty"`
	PhotoURL          *string        `json:"photo_url,omitempty"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
}

// StaffDocument is a document attached to a staff profile.
type StaffDocument struct {
	ID         uuid.UUID  `json:"id"`
	TenantID   uuid.UUID  `json:"tenant_id"`
	StaffID    uuid.UUID  `json:"staff_id"`
	DocType    string     `json:"doc_type"`
	FileName   string     `json:"file_name"`
	FileURL    string     `json:"file_url"`
	MimeType   *string    `json:"mime_type,omitempty"`
	FileSize   *int64     `json:"file_size,omitempty"`
	UploadedBy *uuid.UUID `json:"uploaded_by,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

// PayrollRun is a monthly payroll record for a staff member.
type PayrollRun struct {
	ID                   uuid.UUID  `json:"id"`
	TenantID             uuid.UUID  `json:"tenant_id"`
	StaffID              uuid.UUID  `json:"staff_id"`
	StaffName            string     `json:"staff_name,omitempty"`
	Month                int        `json:"month"`
	Year                 int        `json:"year"`
	BasicSalaryCents     int64      `json:"basic_salary_cents"`
	AllowancesCents      int64      `json:"allowances_cents"`
	GrossCents           int64      `json:"gross_cents"`
	PayeCents            int64      `json:"paye_cents"`
	NHIFCents            int64      `json:"nhif_cents"`
	NSSFCents            int64      `json:"nssf_cents"`
	OtherDeductionsCents int64      `json:"other_deductions_cents"`
	NetCents             int64      `json:"net_cents"`
	Status               string     `json:"status"`
	PaidAt               *time.Time `json:"paid_at,omitempty"`
	CreatedBy            *uuid.UUID `json:"created_by,omitempty"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
	// Items populated on detail responses
	Items []PayrollItem `json:"items,omitempty"`
}

// PayrollItem is a line item (earning or deduction) on a payroll run.
type PayrollItem struct {
	ID           uuid.UUID `json:"id"`
	TenantID     uuid.UUID `json:"tenant_id"`
	PayrollRunID uuid.UUID `json:"payroll_run_id"`
	ItemType     string    `json:"item_type"`
	Name         string    `json:"name"`
	AmountCents  int64     `json:"amount_cents"`
	SortOrder    int       `json:"sort_order"`
	CreatedAt    time.Time `json:"created_at"`
}

// LeaveRequest is a staff leave application.
type LeaveRequest struct {
	ID           uuid.UUID  `json:"id"`
	TenantID     uuid.UUID  `json:"tenant_id"`
	StaffID      uuid.UUID  `json:"staff_id"`
	StaffName    string     `json:"staff_name,omitempty"`
	LeaveType    string     `json:"leave_type"`
	StartDate    string     `json:"start_date"`
	EndDate      string     `json:"end_date"`
	Days         int        `json:"days"`
	Reason       *string    `json:"reason,omitempty"`
	Status       string     `json:"status"`
	ApprovedBy   *uuid.UUID `json:"approved_by,omitempty"`
	ApprovedAt   *time.Time `json:"approved_at,omitempty"`
	DenialReason *string    `json:"denial_reason,omitempty"`
	SubstituteID *uuid.UUID `json:"substitute_id,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// StaffAttendance is a daily attendance record for a staff member.
type StaffAttendance struct {
	ID        uuid.UUID  `json:"id"`
	TenantID  uuid.UUID  `json:"tenant_id"`
	StaffID   uuid.UUID  `json:"staff_id"`
	StaffName string     `json:"staff_name,omitempty"`
	Date      string     `json:"date"`
	ClockIn   *time.Time `json:"clock_in,omitempty"`
	ClockOut  *time.Time `json:"clock_out,omitempty"`
	Status    string     `json:"status"`
	Notes     *string    `json:"notes,omitempty"`
	MarkedBy  *uuid.UUID `json:"marked_by,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// StaffAppraisal is a TSC-aligned performance appraisal.
type StaffAppraisal struct {
	ID           uuid.UUID      `json:"id"`
	TenantID     uuid.UUID      `json:"tenant_id"`
	StaffID      uuid.UUID      `json:"staff_id"`
	StaffName    string         `json:"staff_name,omitempty"`
	Year         int            `json:"year"`
	Term         *int           `json:"term,omitempty"`
	AppraiserID  *uuid.UUID     `json:"appraiser_id,omitempty"`
	Scores       map[string]any `json:"scores,omitempty"`
	OverallScore *float64       `json:"overall_score,omitempty"`
	Rating       *string        `json:"rating,omitempty"`
	Comments     *string        `json:"comments,omitempty"`
	Status       string         `json:"status"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

// --- Request types ---

type CreateStaffRequest struct {
	FullName          string         `json:"full_name"`
	Email             string         `json:"email"`
	Phone             *string        `json:"phone,omitempty"`
	Role              string         `json:"role"`
	IsActive          *bool          `json:"is_active,omitempty"`
	TSCNumber         *string        `json:"tsc_number,omitempty"`
	NationalID        *string        `json:"national_id,omitempty"`
	KRAPin            *string        `json:"kra_pin,omitempty"`
	DateOfBirth       *string        `json:"date_of_birth,omitempty"`
	Gender            *string        `json:"gender,omitempty"`
	Department        *string        `json:"department,omitempty"`
	JobTitle          *string        `json:"job_title,omitempty"`
	EmploymentType    *string        `json:"employment_type,omitempty"`
	HireDate          *string        `json:"hire_date,omitempty"`
	Qualifications    []any          `json:"qualifications,omitempty"`
	Subjects          []any          `json:"subjects,omitempty"`
	EmploymentHistory []any          `json:"employment_history,omitempty"`
	EmergencyContact  map[string]any `json:"emergency_contact,omitempty"`
	BankDetails       map[string]any `json:"bank_details,omitempty"`
	PhotoURL          *string        `json:"photo_url,omitempty"`
}

type UpdateStaffRequest struct {
	FullName          *string        `json:"full_name,omitempty"`
	Email             *string        `json:"email,omitempty"`
	Phone             *string        `json:"phone,omitempty"`
	Role              *string        `json:"role,omitempty"`
	IsActive          *bool          `json:"is_active,omitempty"`
	TSCNumber         *string        `json:"tsc_number,omitempty"`
	NationalID        *string        `json:"national_id,omitempty"`
	KRAPin            *string        `json:"kra_pin,omitempty"`
	DateOfBirth       *string        `json:"date_of_birth,omitempty"`
	Gender            *string        `json:"gender,omitempty"`
	Department        *string        `json:"department,omitempty"`
	JobTitle          *string        `json:"job_title,omitempty"`
	EmploymentType    *string        `json:"employment_type,omitempty"`
	HireDate          *string        `json:"hire_date,omitempty"`
	Qualifications    []any          `json:"qualifications,omitempty"`
	Subjects          []any          `json:"subjects,omitempty"`
	EmploymentHistory []any          `json:"employment_history,omitempty"`
	EmergencyContact  map[string]any `json:"emergency_contact,omitempty"`
	BankDetails       map[string]any `json:"bank_details,omitempty"`
	PhotoURL          *string        `json:"photo_url,omitempty"`
}

type CreateStaffDocumentRequest struct {
	DocType    string     `json:"doc_type"`
	FileName   string     `json:"file_name"`
	FileURL    string     `json:"file_url"`
	MimeType   *string    `json:"mime_type,omitempty"`
	FileSize   *int64     `json:"file_size,omitempty"`
	UploadedBy *uuid.UUID `json:"uploaded_by,omitempty"`
}

type CreatePayrollRunRequest struct {
	StaffID              uuid.UUID          `json:"staff_id"`
	Month                int                `json:"month"`
	Year                 int                `json:"year"`
	BasicSalaryCents     int64              `json:"basic_salary_cents"`
	AllowancesCents      int64              `json:"allowances_cents"`
	PayeCents            int64              `json:"paye_cents"`
	NHIFCents            int64              `json:"nhif_cents"`
	NSSFCents            int64              `json:"nssf_cents"`
	OtherDeductionsCents int64              `json:"other_deductions_cents"`
	CreatedBy            *uuid.UUID         `json:"created_by,omitempty"`
	Items                []PayrollItemInput `json:"items,omitempty"`
}

type PayrollItemInput struct {
	ItemType    string `json:"item_type"`
	Name        string `json:"name"`
	AmountCents int64  `json:"amount_cents"`
	SortOrder   *int   `json:"sort_order,omitempty"`
}

type UpdatePayrollRunRequest struct {
	BasicSalaryCents     *int64  `json:"basic_salary_cents,omitempty"`
	AllowancesCents      *int64  `json:"allowances_cents,omitempty"`
	PayeCents            *int64  `json:"paye_cents,omitempty"`
	NHIFCents            *int64  `json:"nhif_cents,omitempty"`
	NSSFCents            *int64  `json:"nssf_cents,omitempty"`
	OtherDeductionsCents *int64  `json:"other_deductions_cents,omitempty"`
	Status               *string `json:"status,omitempty"`
}

type CreateLeaveRequest struct {
	StaffID      uuid.UUID  `json:"staff_id"`
	LeaveType    string     `json:"leave_type"`
	StartDate    string     `json:"start_date"`
	EndDate      string     `json:"end_date"`
	Reason       *string    `json:"reason,omitempty"`
	SubstituteID *uuid.UUID `json:"substitute_id,omitempty"`
}

type ApproveLeaveRequest struct {
	ApprovedBy   *uuid.UUID `json:"approved_by,omitempty"`
	SubstituteID *uuid.UUID `json:"substitute_id,omitempty"`
}

type DenyLeaveRequest struct {
	ApprovedBy   *uuid.UUID `json:"approved_by,omitempty"`
	DenialReason *string    `json:"denial_reason,omitempty"`
}

type CreateStaffAttendanceRequest struct {
	StaffID  uuid.UUID  `json:"staff_id"`
	Date     string     `json:"date"`
	ClockIn  *time.Time `json:"clock_in,omitempty"`
	ClockOut *time.Time `json:"clock_out,omitempty"`
	Status   *string    `json:"status,omitempty"`
	Notes    *string    `json:"notes,omitempty"`
	MarkedBy *uuid.UUID `json:"marked_by,omitempty"`
}

type UpdateStaffAttendanceRequest struct {
	ClockIn  *time.Time `json:"clock_in,omitempty"`
	ClockOut *time.Time `json:"clock_out,omitempty"`
	Status   *string    `json:"status,omitempty"`
	Notes    *string    `json:"notes,omitempty"`
}

type CreateAppraisalRequest struct {
	StaffID     uuid.UUID      `json:"staff_id"`
	Year        int            `json:"year"`
	Term        *int           `json:"term,omitempty"`
	AppraiserID *uuid.UUID     `json:"appraiser_id,omitempty"`
	Scores      map[string]any `json:"scores,omitempty"`
	Comments    *string        `json:"comments,omitempty"`
}

type UpdateAppraisalRequest struct {
	Scores       map[string]any `json:"scores,omitempty"`
	OverallScore *float64       `json:"overall_score,omitempty"`
	Rating       *string        `json:"rating,omitempty"`
	Comments     *string        `json:"comments,omitempty"`
	Status       *string        `json:"status,omitempty"`
}
