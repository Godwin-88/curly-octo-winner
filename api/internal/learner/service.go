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
	"github.com/shule360/api/pkg/pgxutil"
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
// limit/offset enable server-side pagination (0/0 = unpaginated, used by
// rosters and exports). The returned total counts rows matching the filters,
// ignoring limit/offset, so clients can render page counts.
func (s *Service) List(ctx context.Context, tenantID uuid.UUID, grade, stream, search string, includeInactive bool, limit, offset int) ([]Learner, int64, error) {
	where := `
		WHERE tenant_id = $1
	`
	args := []any{tenantID}
	argN := 1

	if grade != "" {
		argN++
		args = append(args, grade)
		where += fmt.Sprintf(" AND grade = $%d", argN)
	}
	if stream != "" {
		argN++
		args = append(args, stream)
		where += fmt.Sprintf(" AND stream = $%d", argN)
	}
	if !includeInactive {
		where += " AND is_active = true"
	}
	if search != "" {
		argN++
		args = append(args, "%"+strings.ToLower(search)+"%")
		where += fmt.Sprintf(" AND (LOWER(full_name) LIKE $%d OR LOWER(upi) LIKE $%d)", argN, argN)
	}

	// Total matching rows (ignoring pagination) for page counts.
	var total int64
	if err := s.pool.QueryRow(ctx, "SELECT COUNT(*) FROM learners "+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count learners: %w", err)
	}

	query := "SELECT " + learnerColumns + " FROM learners " + where + " ORDER BY full_name"
	if limit > 0 {
		argN++
		args = append(args, limit)
		query += fmt.Sprintf(" LIMIT $%d", argN)
		if offset > 0 {
			argN++
			args = append(args, offset)
			query += fmt.Sprintf(" OFFSET $%d", argN)
		}
	}

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query learners: %w", err)
	}
	defer rows.Close()

	var learners []Learner
	for rows.Next() {
		l, err := scanLearner(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan learner: %w", err)
		}
		learners = append(learners, *l)
	}
	return learners, total, rows.Err()
}

// ListByGrade returns learners filtered by grade and stream (unpaginated).
func (s *Service) ListByGrade(ctx context.Context, tenantID uuid.UUID, grade, stream string) ([]Learner, error) {
	learners, _, err := s.List(ctx, tenantID, grade, stream, "", false, 0, 0)
	return learners, err
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
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8::uuid[], $9, $10, $11, $12)
		RETURNING `+learnerColumns+`
	`, tenantID, req.UPI, req.FullName, req.DateOfBirth, req.Grade, req.Stream,
		req.PhotoURL, pgxutil.UUIDArray(req.GuardianIDs), req.BirthCertNo, req.EntryLevel,
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

	// guardian_ids is only overwritten when the caller actually sent the field.
	// A nil slice is passed as SQL NULL so COALESCE keeps the existing value;
	// an explicit empty array clears the guardians.
	var guardianArg *[]string
	if req.GuardianIDs != nil {
		ids := pgxutil.UUIDArray(req.GuardianIDs)
		guardianArg = &ids
	}

	l, err := scanLearner(s.pool.QueryRow(ctx, `
		UPDATE learners SET
			full_name = COALESCE($3, full_name),
			date_of_birth = COALESCE($4, date_of_birth),
			grade = COALESCE($5, grade),
			stream = COALESCE($6, stream),
			photo_url = COALESCE($7, photo_url),
			guardian_ids = COALESCE($8::uuid[], guardian_ids),
			birth_cert_no = COALESCE($9, birth_cert_no),
			entry_level = COALESCE($10, entry_level),
			special_needs = COALESCE($11, special_needs),
			admission_date = COALESCE($12, admission_date),
			updated_at = now()
		WHERE tenant_id = $1 AND id = $2
		RETURNING `+learnerColumns+`
	`, tenantID, learnerID, req.FullName, req.DateOfBirth, req.Grade, req.Stream,
		req.PhotoURL, guardianArg, req.BirthCertNo, req.EntryLevel,
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
// GuardianDirectoryEntry is a tenant-wide guardian summary used by audience
// pickers (Communications → SMS/WhatsApp) so staff select guardians by name
// instead of pasting raw UUIDs.
type GuardianDirectoryEntry struct {
	ID            uuid.UUID `json:"id"`
	FullName      string    `json:"full_name"`
	Phone         string    `json:"phone"`
	LearnerCount  int       `json:"learner_count"`
	IsSMSOptedOut bool      `json:"is_sms_opted_out"`
}

// ListTenantGuardians returns every guardian in the tenant with their linked
// learner count. search matches name or phone (case-insensitive prefix/substring).
// Results are capped to keep the picker responsive on large schools.
func (s *Service) ListTenantGuardians(ctx context.Context, tenantID uuid.UUID, search string) ([]GuardianDirectoryEntry, error) {
	search = strings.TrimSpace(search)
	pattern := "%" + strings.ToLower(search) + "%"

	rows, err := s.pool.Query(ctx, `
		SELECT g.id,
		       g.full_name,
		       COALESCE(g.phone_wa, g.phone_primary) AS phone,
		       COALESCE((
		           SELECT COUNT(DISTINCT l.id)
		           FROM learners l
		           WHERE l.tenant_id = g.tenant_id AND g.id = ANY(l.guardian_ids)
		       ), 0),
		       g.is_sms_opted_out
		FROM guardians g
		WHERE g.tenant_id = $1
		  AND ($2 = '%%' OR LOWER(g.full_name) LIKE $2
		       OR LOWER(COALESCE(g.phone_wa, g.phone_primary)) LIKE $2)
		ORDER BY g.full_name
		LIMIT 500
	`, tenantID, pattern)
	if err != nil {
		return nil, fmt.Errorf("query guardian directory: %w", err)
	}
	defer rows.Close()

	guardians := make([]GuardianDirectoryEntry, 0, 32)
	for rows.Next() {
		var g GuardianDirectoryEntry
		if err := rows.Scan(&g.ID, &g.FullName, &g.Phone, &g.LearnerCount, &g.IsSMSOptedOut); err != nil {
			return nil, fmt.Errorf("scan guardian: %w", err)
		}
		guardians = append(guardians, g)
	}
	return guardians, rows.Err()
}

func (s *Service) ListGuardians(ctx context.Context, tenantID, learnerID uuid.UUID) ([]GuardianBrief, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT g.id, g.full_name, COALESCE(g.phone_wa, g.phone_primary)
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
		if err := rows.Scan(&g.ID, &g.FullName, &g.Phone); err != nil {
			return nil, fmt.Errorf("scan guardian: %w", err)
		}
		guardians = append(guardians, g)
	}
	return guardians, rows.Err()
}
