package learner

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Learner represents a learner record.
type Learner struct {
	ID          uuid.UUID   `json:"id"`
	TenantID    uuid.UUID   `json:"tenant_id"`
	UPI         string      `json:"upi"`
	FullName    string      `json:"full_name"`
	DateOfBirth *time.Time  `json:"date_of_birth,omitempty"`
	Grade       string      `json:"grade"`
	Stream      string      `json:"stream"`
	PhotoURL    *string     `json:"photo_url,omitempty"`
	GuardianIDs []uuid.UUID `json:"guardian_ids"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

// Service handles learner-related operations (EPIC 3 stub).
type Service struct {
	pool *pgxpool.Pool
}

// NewService creates a new learner service.
func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

// ListByGrade returns learners filtered by grade and stream.
func (s *Service) ListByGrade(ctx context.Context, tenantID uuid.UUID, grade, stream string) ([]Learner, error) {
	query := `
		SELECT id, tenant_id, upi, full_name, date_of_birth, grade, stream,
		       photo_url, guardian_ids, created_at, updated_at
		FROM learners
		WHERE tenant_id = $1
	`
	args := []any{tenantID}

	if grade != "" {
		args = append(args, grade)
		query += fmt.Sprintf(" AND grade = $%d", len(args))
	}
	if stream != "" {
		args = append(args, stream)
		query += fmt.Sprintf(" AND stream = $%d", len(args))
	}

	query += " ORDER BY full_name"

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query learners: %w", err)
	}
	defer rows.Close()

	var learners []Learner
	for rows.Next() {
		var l Learner
		if err := rows.Scan(
			&l.ID, &l.TenantID, &l.UPI, &l.FullName, &l.DateOfBirth, &l.Grade,
			&l.Stream, &l.PhotoURL, &l.GuardianIDs, &l.CreatedAt, &l.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan learner: %w", err)
		}
		learners = append(learners, l)
	}
	return learners, rows.Err()
}

// GetByID returns a single learner.
func (s *Service) GetByID(ctx context.Context, tenantID, learnerID uuid.UUID) (*Learner, error) {
	var l Learner
	err := s.pool.QueryRow(ctx, `
		SELECT id, tenant_id, upi, full_name, date_of_birth, grade, stream,
		       photo_url, guardian_ids, created_at, updated_at
		FROM learners
		WHERE tenant_id = $1 AND id = $2
	`, tenantID, learnerID).Scan(
		&l.ID, &l.TenantID, &l.UPI, &l.FullName, &l.DateOfBirth, &l.Grade,
		&l.Stream, &l.PhotoURL, &l.GuardianIDs, &l.CreatedAt, &l.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("learner not found")
		}
		return nil, fmt.Errorf("query learner: %w", err)
	}
	return &l, nil
}

// Create inserts a new learner.
func (s *Service) Create(ctx context.Context, tenantID uuid.UUID, l *Learner) (*Learner, error) {
	err := s.pool.QueryRow(ctx, `
		INSERT INTO learners (tenant_id, upi, full_name, date_of_birth, grade, stream, photo_url, guardian_ids)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, tenant_id, upi, full_name, date_of_birth, grade, stream,
		          photo_url, guardian_ids, created_at, updated_at
	`, tenantID, l.UPI, l.FullName, l.DateOfBirth, l.Grade, l.Stream, l.PhotoURL, l.GuardianIDs).Scan(
		&l.ID, &l.TenantID, &l.UPI, &l.FullName, &l.DateOfBirth, &l.Grade,
		&l.Stream, &l.PhotoURL, &l.GuardianIDs, &l.CreatedAt, &l.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert learner: %w", err)
	}
	return l, nil
}
