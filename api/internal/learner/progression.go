package learner

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// LearnerProgression represents a promotion/retention/transfer record.
type LearnerProgression struct {
	ID         uuid.UUID  `json:"id"`
	TenantID   uuid.UUID  `json:"tenant_id"`
	LearnerID  uuid.UUID  `json:"learner_id"`
	FromGrade  string     `json:"from_grade"`
	ToGrade    string     `json:"to_grade"`
	Action     string     `json:"action"`
	Term       *int       `json:"term,omitempty"`
	Year       int        `json:"year"`
	ApprovedBy *uuid.UUID `json:"approved_by,omitempty"`
	Notes      string     `json:"notes,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

// PromoteRequest is the request payload for promoting a learner to the next grade.
type PromoteRequest struct {
	LearnerID  uuid.UUID  `json:"learner_id"`
	ToGrade    string     `json:"to_grade"`
	Term       *int       `json:"term,omitempty"`
	Year       int        `json:"year"`
	ApprovedBy *uuid.UUID `json:"approved_by,omitempty"`
	Notes      string     `json:"notes,omitempty"`
}

// RetainRequest is the request payload for retaining a learner in the same grade.
type RetainRequest struct {
	LearnerID  uuid.UUID  `json:"learner_id"`
	Term       *int       `json:"term,omitempty"`
	Year       int        `json:"year"`
	ApprovedBy *uuid.UUID `json:"approved_by,omitempty"`
	Notes      string     `json:"notes,omitempty"`
}

// Promote moves a learner to a new grade and records the progression.
func (s *Service) Promote(ctx context.Context, tenantID uuid.UUID, req PromoteRequest) (*LearnerProgression, error) {
	if req.Year == 0 {
		req.Year = time.Now().Year()
	}
	if req.ToGrade == "" {
		return nil, fmt.Errorf("to_grade is required")
	}

	return s.applyProgression(ctx, tenantID, req.LearnerID, req.ToGrade, req.Term, req.Year, req.ApprovedBy, req.Notes, "promote")
}

// Retain keeps a learner in the same grade and records the retention.
func (s *Service) Retain(ctx context.Context, tenantID uuid.UUID, req RetainRequest) (*LearnerProgression, error) {
	if req.Year == 0 {
		req.Year = time.Now().Year()
	}

	learner, err := s.GetByID(ctx, tenantID, req.LearnerID)
	if err != nil {
		return nil, err
	}

	return s.applyProgression(ctx, tenantID, req.LearnerID, learner.Grade, req.Term, req.Year, req.ApprovedBy, req.Notes, "retain")
}

// TransferOut records a transfer out and deactivates the learner.
func (s *Service) TransferOut(ctx context.Context, tenantID, learnerID uuid.UUID, term *int, year int, approvedBy *uuid.UUID, notes string) (*LearnerProgression, error) {
	if year == 0 {
		year = time.Now().Year()
	}

	learner, err := s.GetByID(ctx, tenantID, learnerID)
	if err != nil {
		return nil, err
	}

	p, err := s.applyProgression(ctx, tenantID, learnerID, learner.Grade, term, year, approvedBy, notes, "transfer_out")
	if err != nil {
		return nil, err
	}

	if err := s.Deactivate(ctx, tenantID, learnerID); err != nil {
		return nil, err
	}
	return p, nil
}

// TransferIn records a transfer into another grade and reactivates the learner.
func (s *Service) TransferIn(ctx context.Context, tenantID, learnerID uuid.UUID, toGrade string, term *int, year int, approvedBy *uuid.UUID, notes string) (*LearnerProgression, error) {
	if year == 0 {
		year = time.Now().Year()
	}
	if toGrade == "" {
		return nil, fmt.Errorf("to_grade is required")
	}

	p, err := s.applyProgression(ctx, tenantID, learnerID, toGrade, term, year, approvedBy, notes, "transfer_in")
	if err != nil {
		return nil, err
	}

	if err := s.Reactivate(ctx, tenantID, learnerID); err != nil {
		return nil, err
	}
	return p, nil
}

// applyProgression inserts a progression record and updates the learner's grade.
func (s *Service) applyProgression(ctx context.Context, tenantID, learnerID uuid.UUID, toGrade string, term *int, year int, approvedBy *uuid.UUID, notes, action string) (*LearnerProgression, error) {
	learner, err := s.GetByID(ctx, tenantID, learnerID)
	if err != nil {
		return nil, err
	}

	var p LearnerProgression
	err = s.pool.QueryRow(ctx, `
		INSERT INTO learner_progressions (tenant_id, learner_id, from_grade, to_grade, action, term, year, approved_by, notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, tenant_id, learner_id, from_grade, to_grade, action, term, year, approved_by, notes, created_at, updated_at
	`, tenantID, learnerID, learner.Grade, toGrade, action, term, year, approvedBy, notes).Scan(
		&p.ID, &p.TenantID, &p.LearnerID, &p.FromGrade, &p.ToGrade, &p.Action,
		&p.Term, &p.Year, &p.ApprovedBy, &p.Notes, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert learner progression: %w", err)
	}

	// Update learner's grade (unless transfer_out keeps the current grade)
	if action != "transfer_out" {
		if _, err := s.pool.Exec(ctx, `
			UPDATE learners SET grade = $3, updated_at = now()
			WHERE tenant_id = $1 AND id = $2
		`, tenantID, learnerID, toGrade); err != nil {
			return nil, fmt.Errorf("update learner grade: %w", err)
		}
	}

	return &p, nil
}

// ListProgressions returns progression history for a learner.
func (s *Service) ListProgressions(ctx context.Context, tenantID, learnerID uuid.UUID) ([]LearnerProgression, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, tenant_id, learner_id, from_grade, to_grade, action, term, year, approved_by, notes, created_at, updated_at
		FROM learner_progressions
		WHERE tenant_id = $1 AND learner_id = $2
		ORDER BY created_at DESC
	`, tenantID, learnerID)
	if err != nil {
		return nil, fmt.Errorf("query learner progressions: %w", err)
	}
	defer rows.Close()

	var progressions []LearnerProgression
	for rows.Next() {
		var p LearnerProgression
		if err := rows.Scan(
			&p.ID, &p.TenantID, &p.LearnerID, &p.FromGrade, &p.ToGrade, &p.Action,
			&p.Term, &p.Year, &p.ApprovedBy, &p.Notes, &p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan learner progression: %w", err)
		}
		progressions = append(progressions, p)
	}
	return progressions, rows.Err()
}

// GetProgression returns a single progression record.
func (s *Service) GetProgression(ctx context.Context, tenantID, id uuid.UUID) (*LearnerProgression, error) {
	var p LearnerProgression
	err := s.pool.QueryRow(ctx, `
		SELECT id, tenant_id, learner_id, from_grade, to_grade, action, term, year, approved_by, notes, created_at, updated_at
		FROM learner_progressions
		WHERE tenant_id = $1 AND id = $2
	`, tenantID, id).Scan(
		&p.ID, &p.TenantID, &p.LearnerID, &p.FromGrade, &p.ToGrade, &p.Action,
		&p.Term, &p.Year, &p.ApprovedBy, &p.Notes, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("progression record not found")
		}
		return nil, fmt.Errorf("query learner progression: %w", err)
	}
	return &p, nil
}
