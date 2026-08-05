package assessment

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Assessment represents a formative assessment observation.
type Assessment struct {
	ID           uuid.UUID   `json:"id"`
	TenantID     uuid.UUID   `json:"tenant_id"`
	LearnerID    uuid.UUID   `json:"learner_id"`
	SubStrandID  uuid.UUID   `json:"sub_strand_id"`
	RubricLevel  int         `json:"rubric_level"`
	Note         string      `json:"note"`
	EvidenceURLs []string    `json:"evidence_urls"`
	TeacherID    uuid.UUID   `json:"teacher_id"`
	Term         int         `json:"term"`
	Year         int         `json:"year"`
	CreatedAt    time.Time   `json:"created_at"`
	UpdatedAt    time.Time   `json:"updated_at"`
}

// CreateAssessmentRequest is the request payload for creating an assessment.
type CreateAssessmentRequest struct {
	LearnerID   uuid.UUID
	SubStrandID uuid.UUID
	RubricLevel int
	Note        string
	EvidenceURLs []string
	TeacherID   uuid.UUID
	Term        int
	Year        int
}

// AssessmentSummary is a joined view of assessment with learner and strand info.
type AssessmentSummary struct {
	ID            uuid.UUID `json:"id"`
	LearnerID     uuid.UUID `json:"learner_id"`
	LearnerName   string    `json:"learner_name"`
	Grade         string    `json:"grade"`
	Stream        string    `json:"stream"`
	SubStrandID   uuid.UUID `json:"sub_strand_id"`
	SubStrandName string    `json:"sub_strand_name"`
	SubStrandCode string    `json:"sub_strand_code"`
	StrandName    string    `json:"strand_name"`
	LearningArea  string    `json:"learning_area"`
	RubricLevel   int       `json:"rubric_level"`
	RubricLabel   string    `json:"rubric_label"`
	Note          string    `json:"note"`
	Term          int       `json:"term"`
	Year          int       `json:"year"`
	TeacherID     uuid.UUID `json:"teacher_id"`
	CreatedAt     time.Time `json:"created_at"`
}

// RubricLevelLabel returns the CBC rubric label for a numeric level.
func RubricLevelLabel(level int) string {
	switch level {
	case 1:
		return "Below Expectation"
	case 2:
		return "Approaching Expectation"
	case 3:
		return "Meeting Expectation"
	case 4:
		return "Exceeding Expectation"
	default:
		return "Unknown"
	}
}

// Service handles assessment-related operations.
type Service struct {
	pool *pgxpool.Pool
}

// NewService creates a new assessment service.
func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

// Create inserts a new assessment observation.
func (s *Service) Create(ctx context.Context, tenantID uuid.UUID, req CreateAssessmentRequest) (*Assessment, error) {
	if req.RubricLevel < 1 || req.RubricLevel > 4 {
		return nil, fmt.Errorf("rubric_level must be between 1 and 4")
	}

	var a Assessment
	err := s.pool.QueryRow(ctx, `
		INSERT INTO assessments (tenant_id, learner_id, sub_strand_id, rubric_level, note, evidence_urls, teacher_id, term, year)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, tenant_id, learner_id, sub_strand_id, rubric_level, note, evidence_urls, teacher_id, term, year, created_at, updated_at
	`, tenantID, req.LearnerID, req.SubStrandID, req.RubricLevel, req.Note, req.EvidenceURLs, req.TeacherID, req.Term, req.Year).Scan(
		&a.ID, &a.TenantID, &a.LearnerID, &a.SubStrandID, &a.RubricLevel,
		&a.Note, &a.EvidenceURLs, &a.TeacherID, &a.Term, &a.Year,
		&a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert assessment: %w", err)
	}
	return &a, nil
}

// ListByLearner returns all assessments for a specific learner in a term/year.
func (s *Service) ListByLearner(ctx context.Context, tenantID, learnerID uuid.UUID, term, year int) ([]Assessment, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, tenant_id, learner_id, sub_strand_id, rubric_level, note, evidence_urls, teacher_id, term, year, created_at, updated_at
		FROM assessments
		WHERE tenant_id = $1 AND learner_id = $2 AND term = $3 AND year = $4
		ORDER BY created_at DESC
	`, tenantID, learnerID, term, year)
	if err != nil {
		return nil, fmt.Errorf("query assessments: %w", err)
	}
	defer rows.Close()

	var assessments []Assessment
	for rows.Next() {
		var a Assessment
		if err := rows.Scan(
			&a.ID, &a.TenantID, &a.LearnerID, &a.SubStrandID, &a.RubricLevel,
			&a.Note, &a.EvidenceURLs, &a.TeacherID, &a.Term, &a.Year,
			&a.CreatedAt, &a.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan assessment: %w", err)
		}
		assessments = append(assessments, a)
	}
	return assessments, rows.Err()
}

// ListByTermYear returns all assessments for a tenant in a term/year.
func (s *Service) ListByTermYear(ctx context.Context, tenantID uuid.UUID, term, year int) ([]Assessment, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, tenant_id, learner_id, sub_strand_id, rubric_level, note, evidence_urls, teacher_id, term, year, created_at, updated_at
		FROM assessments
		WHERE tenant_id = $1 AND term = $2 AND year = $3
		ORDER BY created_at DESC
	`, tenantID, term, year)
	if err != nil {
		return nil, fmt.Errorf("query assessments: %w", err)
	}
	defer rows.Close()

	var assessments []Assessment
	for rows.Next() {
		var a Assessment
		if err := rows.Scan(
			&a.ID, &a.TenantID, &a.LearnerID, &a.SubStrandID, &a.RubricLevel,
			&a.Note, &a.EvidenceURLs, &a.TeacherID, &a.Term, &a.Year,
			&a.CreatedAt, &a.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan assessment: %w", err)
		}
		assessments = append(assessments, a)
	}
	return assessments, rows.Err()
}

// ListSummariesByLearner returns assessment summaries joined with learner and strand info.
func (s *Service) ListSummariesByLearner(ctx context.Context, tenantID, learnerID uuid.UUID, term, year int) ([]AssessmentSummary, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT
			a.id, a.learner_id, l.full_name AS learner_name, l.grade, l.stream,
			a.sub_strand_id, s.name AS sub_strand_name, s.kicd_code AS sub_strand_code,
			str.name AS strand_name, la.name AS learning_area,
			a.rubric_level,
			CASE a.rubric_level
				WHEN 1 THEN 'Below Expectation'
				WHEN 2 THEN 'Approaching Expectation'
				WHEN 3 THEN 'Meeting Expectation'
				WHEN 4 THEN 'Exceeding Expectation'
			END AS rubric_label,
			a.note, a.term, a.year, a.teacher_id, a.created_at
		FROM assessments a
		JOIN learners l ON l.id = a.learner_id AND l.tenant_id = a.tenant_id
		JOIN sub_strands s ON s.id = a.sub_strand_id AND s.tenant_id = a.tenant_id
		JOIN strands str ON str.id = s.strand_id AND str.tenant_id = a.tenant_id
		JOIN learning_areas la ON la.id = str.learning_area_id AND la.tenant_id = a.tenant_id
		WHERE a.tenant_id = $1 AND a.learner_id = $2 AND a.term = $3 AND a.year = $4
		ORDER BY a.created_at DESC
	`, tenantID, learnerID, term, year)
	if err != nil {
		return nil, fmt.Errorf("query assessment summaries: %w", err)
	}
	defer rows.Close()

	var summaries []AssessmentSummary
	for rows.Next() {
		var as AssessmentSummary
		if err := rows.Scan(
			&as.ID, &as.LearnerID, &as.LearnerName, &as.Grade, &as.Stream,
			&as.SubStrandID, &as.SubStrandName, &as.SubStrandCode,
			&as.StrandName, &as.LearningArea, &as.RubricLevel, &as.RubricLabel,
			&as.Note, &as.Term, &as.Year, &as.TeacherID, &as.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan assessment summary: %w", err)
		}
		summaries = append(summaries, as)
	}
	return summaries, rows.Err()
}

// GetByID returns a single assessment.
func (s *Service) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*Assessment, error) {
	var a Assessment
	err := s.pool.QueryRow(ctx, `
		SELECT id, tenant_id, learner_id, sub_strand_id, rubric_level, note, evidence_urls, teacher_id, term, year, created_at, updated_at
		FROM assessments
		WHERE tenant_id = $1 AND id = $2
	`, tenantID, id).Scan(
		&a.ID, &a.TenantID, &a.LearnerID, &a.SubStrandID, &a.RubricLevel,
		&a.Note, &a.EvidenceURLs, &a.TeacherID, &a.Term, &a.Year,
		&a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("assessment not found")
		}
		return nil, fmt.Errorf("query assessment: %w", err)
	}
	return &a, nil
}

// Delete removes an assessment.
func (s *Service) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `
		DELETE FROM assessments
		WHERE tenant_id = $1 AND id = $2
	`, tenantID, id)
	if err != nil {
		return fmt.Errorf("delete assessment: %w", err)
	}
	return nil
}

// CompetencyDistribution returns the count of assessments per rubric level for a sub-strand.
func (s *Service) CompetencyDistribution(ctx context.Context, tenantID, subStrandID uuid.UUID, term, year int) (map[int]int, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT rubric_level, COUNT(*)
		FROM assessments
		WHERE tenant_id = $1 AND sub_strand_id = $2 AND term = $3 AND year = $4
		GROUP BY rubric_level
	`, tenantID, subStrandID, term, year)
	if err != nil {
		return nil, fmt.Errorf("query competency distribution: %w", err)
	}
	defer rows.Close()

	dist := map[int]int{1: 0, 2: 0, 3: 0, 4: 0}
	for rows.Next() {
		var level, count int
		if err := rows.Scan(&level, &count); err != nil {
			return nil, fmt.Errorf("scan distribution: %w", err)
		}
		dist[level] = count
	}
	return dist, rows.Err()
}

// StrandsCoverage returns which strands have assessments for a learner.
func (s *Service) StrandsCoverage(ctx context.Context, tenantID, learnerID uuid.UUID, term, year int) (map[uuid.UUID]bool, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT DISTINCT s.strand_id
		FROM assessments a
		JOIN sub_strands s ON s.id = a.sub_strand_id AND s.tenant_id = a.tenant_id
		WHERE a.tenant_id = $1 AND a.learner_id = $2 AND a.term = $3 AND a.year = $4
	`, tenantID, learnerID, term, year)
	if err != nil {
		return nil, fmt.Errorf("query strands coverage: %w", err)
	}
	defer rows.Close()

	covered := make(map[uuid.UUID]bool)
	for rows.Next() {
		var strandID uuid.UUID
		if err := rows.Scan(&strandID); err != nil {
			return nil, fmt.Errorf("scan coverage: %w", err)
		}
		covered[strandID] = true
	}
	return covered, rows.Err()
}
