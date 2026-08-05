package learner

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/shule360/api/internal/nemis"
)

// Learner represents a learner record.
type Learner struct {
	ID            uuid.UUID   `json:"id"`
	TenantID      uuid.UUID   `json:"tenant_id"`
	UPI           string      `json:"upi"`
	FullName      string      `json:"full_name"`
	DateOfBirth   *time.Time  `json:"date_of_birth,omitempty"`
	Grade         string      `json:"grade"`
	Stream        string      `json:"stream"`
	PhotoURL      *string     `json:"photo_url,omitempty"`
	GuardianIDs   []uuid.UUID `json:"guardian_ids"`
	BirthCertNo   *string     `json:"birth_cert_no,omitempty"`
	EntryLevel    *string     `json:"entry_level,omitempty"`
	SpecialNeeds  bool        `json:"special_needs"`
	IsActive      bool        `json:"is_active"`
	AdmissionDate *time.Time  `json:"admission_date,omitempty"`
	CreatedAt     time.Time   `json:"created_at"`
	UpdatedAt     time.Time   `json:"updated_at"`
}

// GuardianBrief is a lightweight guardian reference for learner detail responses.
type GuardianBrief struct {
	ID       uuid.UUID `json:"id"`
	FullName string    `json:"full_name"`
	Phone    string    `json:"phone"`
	Relation string    `json:"relation"`
}

// CreateLearnerRequest is the request payload for creating a learner.
type CreateLearnerRequest struct {
	UPI           string      `json:"upi"`
	FullName      string      `json:"full_name"`
	DateOfBirth   *time.Time  `json:"date_of_birth,omitempty"`
	Grade         string      `json:"grade"`
	Stream        string      `json:"stream"`
	PhotoURL      *string     `json:"photo_url,omitempty"`
	GuardianIDs   []uuid.UUID `json:"guardian_ids"`
	BirthCertNo   *string     `json:"birth_cert_no,omitempty"`
	EntryLevel    *string     `json:"entry_level,omitempty"`
	SpecialNeeds  bool        `json:"special_needs"`
	AdmissionDate *time.Time  `json:"admission_date,omitempty"`
}

// UpdateLearnerRequest is the request payload for updating a learner.
type UpdateLearnerRequest struct {
	FullName      *string     `json:"full_name,omitempty"`
	DateOfBirth   *time.Time  `json:"date_of_birth,omitempty"`
	Grade         *string     `json:"grade,omitempty"`
	Stream        *string     `json:"stream,omitempty"`
	PhotoURL      *string     `json:"photo_url,omitempty"`
	GuardianIDs   []uuid.UUID `json:"guardian_ids,omitempty"`
	BirthCertNo   *string     `json:"birth_cert_no,omitempty"`
	EntryLevel    *string     `json:"entry_level,omitempty"`
	SpecialNeeds  *bool       `json:"special_needs,omitempty"`
	AdmissionDate *time.Time  `json:"admission_date,omitempty"`
}

// Service handles learner-related operations (EPIC 3).
type Service struct {
	pool  *pgxpool.Pool
	nemis nemis.NEMISClient
}

// NewService creates a new learner service.
func NewService(pool *pgxpool.Pool, nemisClient nemis.NEMISClient) *Service {
	if nemisClient == nil {
		nemisClient = &nemis.SandboxNEMISClient{}
	}
	return &Service{pool: pool, nemis: nemisClient}
}

const learnerColumns = `
	id, tenant_id, upi, full_name, date_of_birth, grade, stream, photo_url,
	guardian_ids, birth_cert_no, entry_level, special_needs, is_active, admission_date,
	created_at, updated_at`

func scanLearner(row pgx.Row) (*Learner, error) {
	var l Learner
	err := row.Scan(
		&l.ID, &l.TenantID, &l.UPI, &l.FullName, &l.DateOfBirth, &l.Grade,
		&l.Stream, &l.PhotoURL, &l.GuardianIDs, &l.BirthCertNo, &l.EntryLevel,
		&l.SpecialNeeds, &l.IsActive, &l.AdmissionDate, &l.CreatedAt, &l.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &l, nil
}

// List returns learners with optional grade/stream/search filters.
func (s *Service) List(ctx context.Context, tenantID uuid.UUID, grade, stream, search string, includeInactive bool) ([]Learner, error) {
	query := `
		SELECT ` + learnerColumns + `
		FROM learners
		WHERE tenant_id = $1
	`
	args := []any{tenantID}
	argN := 1

	if grade != "" {
		argN++
		args = append(args, grade)
		query += fmt.Sprintf(" AND grade = $%d", argN)
	}
	if stream != "" {
		argN++
		args = append(args, stream)
		query += fmt.Sprintf(" AND stream = $%d", argN)
	}
	if !includeInactive {
		query += " AND is_active = true"
	}
	if search != "" {
		argN++
		args = append(args, "%"+strings.ToLower(search)+"%")
		query += fmt.Sprintf(" AND (LOWER(full_name) LIKE $%d OR LOWER(upi) LIKE $%d)", argN, argN)
	}

	query += " ORDER BY full_name"

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query learners: %w", err)
	}
	defer rows.Close()

	var learners []Learner
	for rows.Next() {
		l, err := scanLearner(rows)
		if err != nil {
			return nil, fmt.Errorf("scan learner: %w", err)
		}
		learners = append(learners, *l)
	}
	return learners, rows.Err()
}

// ListByGrade returns learners filtered by grade and stream.
func (s *Service) ListByGrade(ctx context.Context, tenantID uuid.UUID, grade, stream string) ([]Learner, error) {
	return s.List(ctx, tenantID, grade, stream, "", false)
}

// GetByID returns a single learner.
func (s *Service) GetByID(ctx context.Context, tenantID, learnerID uuid.UUID) (*Learner, error) {
	l, err := scanLearner(s.pool.QueryRow(ctx, `
		SELECT `+learnerColumns+`
		FROM learners
		WHERE tenant_id = $1 AND id = $2
	`, tenantID, learnerID))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("learner not found")
		}
		return nil, fmt.Errorf("query learner: %w", err)
	}
	return l, nil
}

// GetByUPI returns a single learner by UPI.
func (s *Service) GetByUPI(ctx context.Context, tenantID uuid.UUID, upi string) (*Learner, error) {
	l, err := scanLearner(s.pool.QueryRow(ctx, `
		SELECT `+learnerColumns+`
		FROM learners
		WHERE tenant_id = $1 AND upi = $2
	`, tenantID, upi))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("learner not found")
		}
		return nil, fmt.Errorf("query learner by upi: %w", err)
	}
	return l, nil
}

// Create validates the UPI against NEMIS then inserts a new learner.
func (s *Service) Create(ctx context.Context, tenantID uuid.UUID, req CreateLearnerRequest) (*Learner, error) {
	if strings.TrimSpace(req.UPI) == "" {
		return nil, fmt.Errorf("upi is required")
	}
	if strings.TrimSpace(req.FullName) == "" {
		return nil, fmt.Errorf("full_name is required")
	}
	if strings.TrimSpace(req.Grade) == "" {
		return nil, fmt.Errorf("grade is required")
	}

	// Validate UPI against NEMIS (sandbox in dev, live in prod)
	if _, err := s.nemis.ValidateUPI(ctx, req.UPI); err != nil {
		return nil, fmt.Errorf("nemis validation failed: %w", err)
	}

	// Check for duplicate UPI within tenant
	existing, err := s.GetByUPI(ctx, tenantID, req.UPI)
	if err == nil && existing != nil {
		return nil, fmt.Errorf("learner with UPI %s already exists", req.UPI)
	}

	var l Learner
	err = s.pool.QueryRow(ctx, `
		INSERT INTO learners (tenant_id, upi, full_name, date_of_birth, grade, stream, photo_url, guardian_ids, birth_cert_no, entry_level, special_needs, admission_date)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING `+learnerColumns+`
	`, tenantID, req.UPI, req.FullName, req.DateOfBirth, req.Grade, req.Stream,
		req.PhotoURL, req.GuardianIDs, req.BirthCertNo, req.EntryLevel,
		req.SpecialNeeds, req.AdmissionDate).Scan(
		&l.ID, &l.TenantID, &l.UPI, &l.FullName, &l.DateOfBirth, &l.Grade,
		&l.Stream, &l.PhotoURL, &l.GuardianIDs, &l.BirthCertNo, &l.EntryLevel,
		&l.SpecialNeeds, &l.IsActive, &l.AdmissionDate, &l.CreatedAt, &l.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert learner: %w", err)
	}
	return &l, nil
}

// Update updates editable fields of a learner.
func (s *Service) Update(ctx context.Context, tenantID, learnerID uuid.UUID, req UpdateLearnerRequest) (*Learner, error) {
	// Ensure learner exists and belongs to tenant
	if _, err := s.GetByID(ctx, tenantID, learnerID); err != nil {
		return nil, err
	}

	l, err := scanLearner(s.pool.QueryRow(ctx, `
		UPDATE learners SET
			full_name = COALESCE($3, full_name),
			date_of_birth = COALESCE($4, date_of_birth),
			grade = COALESCE($5, grade),
			stream = COALESCE($6, stream),
			photo_url = COALESCE($7, photo_url),
			guardian_ids = COALESCE($8, guardian_ids),
			birth_cert_no = COALESCE($9, birth_cert_no),
			entry_level = COALESCE($10, entry_level),
			special_needs = COALESCE($11, special_needs),
			admission_date = COALESCE($12, admission_date),
			updated_at = now()
		WHERE tenant_id = $1 AND id = $2
		RETURNING `+learnerColumns+`
	`, tenantID, learnerID, req.FullName, req.DateOfBirth, req.Grade, req.Stream,
		req.PhotoURL, req.GuardianIDs, req.BirthCertNo, req.EntryLevel,
		req.SpecialNeeds, req.AdmissionDate))
	if err != nil {
		return nil, fmt.Errorf("update learner: %w", err)
	}
	return l, nil
}

// Deactivate soft-deletes a learner (sets is_active = false).
func (s *Service) Deactivate(ctx context.Context, tenantID, learnerID uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE learners SET is_active = false, updated_at = now()
		WHERE tenant_id = $1 AND id = $2
	`, tenantID, learnerID)
	if err != nil {
		return fmt.Errorf("deactivate learner: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("learner not found")
	}
	return nil
}

// Reactivate sets a learner back to active.
func (s *Service) Reactivate(ctx context.Context, tenantID, learnerID uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE learners SET is_active = true, updated_at = now()
		WHERE tenant_id = $1 AND id = $2
	`, tenantID, learnerID)
	if err != nil {
		return fmt.Errorf("reactivate learner: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("learner not found")
	}
	return nil
}

// ListGuardians returns the guardian records referenced by a learner.
func (s *Service) ListGuardians(ctx context.Context, tenantID, learnerID uuid.UUID) ([]GuardianBrief, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT g.id, g.full_name, g.phone, g.relation
		FROM guardians g
		JOIN learners l ON l.tenant_id = g.tenant_id
		WHERE l.tenant_id = $1 AND l.id = $2 AND g.id = ANY(l.guardian_ids)
		ORDER BY g.full_name
	`, tenantID, learnerID)
	if err != nil {
		return nil, fmt.Errorf("query guardians: %w", err)
	}
	defer rows.Close()

	var guardians []GuardianBrief
	for rows.Next() {
		var g GuardianBrief
		if err := rows.Scan(&g.ID, &g.FullName, &g.Phone, &g.Relation); err != nil {
			return nil, fmt.Errorf("scan guardian: %w", err)
		}
		guardians = append(guardians, g)
	}
	return guardians, rows.Err()
}
