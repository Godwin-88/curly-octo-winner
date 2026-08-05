package curriculum

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// LearningArea represents a KICD learning area (e.g. Mathematics, English).
type LearningArea struct {
	ID          uuid.UUID `json:"id"`
	TenantID    uuid.UUID `json:"tenant_id"`
	Name        string    `json:"name"`
	KICDCode    string    `json:"kicd_code"`
	GradeLevel  string    `json:"grade_level"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Strand represents a KICD strand within a learning area.
type Strand struct {
	ID              uuid.UUID `json:"id"`
	TenantID        uuid.UUID `json:"tenant_id"`
	LearningAreaID  uuid.UUID `json:"learning_area_id"`
	Name            string    `json:"name"`
	KICDCode        string    `json:"kicd_code"`
	Description     string    `json:"description,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// SubStrand represents a KICD sub-strand within a strand.
type SubStrand struct {
	ID          uuid.UUID `json:"id"`
	TenantID    uuid.UUID `json:"tenant_id"`
	StrandID    uuid.UUID `json:"strand_id"`
	Name        string    `json:"name"`
	KICDCode    string    `json:"kicd_code"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// LearningOutcome represents a specific learning outcome (SLO) within a sub-strand.
type LearningOutcome struct {
	ID          uuid.UUID `json:"id"`
	TenantID    uuid.UUID `json:"tenant_id"`
	SubStrandID uuid.UUID `json:"sub_strand_id"`
	Description string    `json:"description"`
	SortOrder   int       `json:"sort_order"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CoreCompetency represents a KICD core competency.
type CoreCompetency struct {
	ID          uuid.UUID `json:"id"`
	TenantID    uuid.UUID `json:"tenant_id"`
	Name        string    `json:"name"`
	KICDCode    string    `json:"kicd_code"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Value represents a KICD value strand.
type Value struct {
	ID          uuid.UUID `json:"id"`
	TenantID    uuid.UUID `json:"tenant_id"`
	Name        string    `json:"name"`
	KICDCode    string    `json:"kicd_code"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Service handles curriculum-related operations.
type Service struct {
	pool *pgxpool.Pool
}

// NewService creates a new curriculum service.
func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

// ListLearningAreas returns all learning areas for a tenant.
func (s *Service) ListLearningAreas(ctx context.Context, tenantID uuid.UUID) ([]LearningArea, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, tenant_id, name, kicd_code, grade_level, description, created_at, updated_at
		FROM learning_areas
		WHERE tenant_id = $1
		ORDER BY grade_level, name
	`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("query learning areas: %w", err)
	}
	defer rows.Close()

	var areas []LearningArea
	for rows.Next() {
		var la LearningArea
		if err := rows.Scan(
			&la.ID, &la.TenantID, &la.Name, &la.KICDCode,
			&la.GradeLevel, &la.Description, &la.CreatedAt, &la.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan learning area: %w", err)
		}
		areas = append(areas, la)
	}
	return areas, rows.Err()
}

// GetLearningArea returns a single learning area.
func (s *Service) GetLearningArea(ctx context.Context, tenantID, id uuid.UUID) (*LearningArea, error) {
	var la LearningArea
	err := s.pool.QueryRow(ctx, `
		SELECT id, tenant_id, name, kicd_code, grade_level, description, created_at, updated_at
		FROM learning_areas
		WHERE tenant_id = $1 AND id = $2
	`, tenantID, id).Scan(
		&la.ID, &la.TenantID, &la.Name, &la.KICDCode,
		&la.GradeLevel, &la.Description, &la.CreatedAt, &la.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("learning area not found")
		}
		return nil, fmt.Errorf("query learning area: %w", err)
	}
	return &la, nil
}

// CreateLearningArea inserts a new learning area.
func (s *Service) CreateLearningArea(ctx context.Context, tenantID uuid.UUID, la *LearningArea) (*LearningArea, error) {
	err := s.pool.QueryRow(ctx, `
		INSERT INTO learning_areas (tenant_id, name, kicd_code, grade_level, description)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, tenant_id, name, kicd_code, grade_level, description, created_at, updated_at
	`, tenantID, la.Name, la.KICDCode, la.GradeLevel, la.Description).Scan(
		&la.ID, &la.TenantID, &la.Name, &la.KICDCode,
		&la.GradeLevel, &la.Description, &la.CreatedAt, &la.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert learning area: %w", err)
	}
	return la, nil
}

// ListStrands returns all strands for a learning area.
func (s *Service) ListStrands(ctx context.Context, tenantID, learningAreaID uuid.UUID) ([]Strand, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, tenant_id, learning_area_id, name, kicd_code, description, created_at, updated_at
		FROM strands
		WHERE tenant_id = $1 AND learning_area_id = $2
		ORDER BY name
	`, tenantID, learningAreaID)
	if err != nil {
		return nil, fmt.Errorf("query strands: %w", err)
	}
	defer rows.Close()

	var strands []Strand
	for rows.Next() {
		var st Strand
		if err := rows.Scan(
			&st.ID, &st.TenantID, &st.LearningAreaID, &st.Name, &st.KICDCode,
			&st.Description, &st.CreatedAt, &st.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan strand: %w", err)
		}
		strands = append(strands, st)
	}
	return strands, rows.Err()
}

// CreateStrand inserts a new strand.
func (s *Service) CreateStrand(ctx context.Context, tenantID uuid.UUID, st *Strand) (*Strand, error) {
	err := s.pool.QueryRow(ctx, `
		INSERT INTO strands (tenant_id, learning_area_id, name, kicd_code, description)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, tenant_id, learning_area_id, name, kicd_code, description, created_at, updated_at
	`, tenantID, st.LearningAreaID, st.Name, st.KICDCode, st.Description).Scan(
		&st.ID, &st.TenantID, &st.LearningAreaID, &st.Name, &st.KICDCode,
		&st.Description, &st.CreatedAt, &st.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert strand: %w", err)
	}
	return st, nil
}

// ListSubStrands returns all sub-strands for a strand.
func (s *Service) ListSubStrands(ctx context.Context, tenantID, strandID uuid.UUID) ([]SubStrand, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, tenant_id, strand_id, name, kicd_code, description, created_at, updated_at
		FROM sub_strands
		WHERE tenant_id = $1 AND strand_id = $2
		ORDER BY name
	`, tenantID, strandID)
	if err != nil {
		return nil, fmt.Errorf("query sub-strands: %w", err)
	}
	defer rows.Close()

	var subStrands []SubStrand
	for rows.Next() {
		var ss SubStrand
		if err := rows.Scan(
			&ss.ID, &ss.TenantID, &ss.StrandID, &ss.Name, &ss.KICDCode,
			&ss.Description, &ss.CreatedAt, &ss.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan sub-strand: %w", err)
		}
		subStrands = append(subStrands, ss)
	}
	return subStrands, rows.Err()
}

// CreateSubStrand inserts a new sub-strand.
func (s *Service) CreateSubStrand(ctx context.Context, tenantID uuid.UUID, ss *SubStrand) (*SubStrand, error) {
	err := s.pool.QueryRow(ctx, `
		INSERT INTO sub_strands (tenant_id, strand_id, name, kicd_code, description)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, tenant_id, strand_id, name, kicd_code, description, created_at, updated_at
	`, tenantID, ss.StrandID, ss.Name, ss.KICDCode, ss.Description).Scan(
		&ss.ID, &ss.TenantID, &ss.StrandID, &ss.Name, &ss.KICDCode,
		&ss.Description, &ss.CreatedAt, &ss.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert sub-strand: %w", err)
	}
	return ss, nil
}

// ListLearningOutcomes returns all learning outcomes for a sub-strand.
func (s *Service) ListLearningOutcomes(ctx context.Context, tenantID, subStrandID uuid.UUID) ([]LearningOutcome, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, tenant_id, sub_strand_id, description, sort_order, created_at, updated_at
		FROM learning_outcomes
		WHERE tenant_id = $1 AND sub_strand_id = $2
		ORDER BY sort_order
	`, tenantID, subStrandID)
	if err != nil {
		return nil, fmt.Errorf("query learning outcomes: %w", err)
	}
	defer rows.Close()

	var outcomes []LearningOutcome
	for rows.Next() {
		var lo LearningOutcome
		if err := rows.Scan(
			&lo.ID, &lo.TenantID, &lo.SubStrandID, &lo.Description,
			&lo.SortOrder, &lo.CreatedAt, &lo.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan learning outcome: %w", err)
		}
		outcomes = append(outcomes, lo)
	}
	return outcomes, rows.Err()
}

// CreateLearningOutcome inserts a new learning outcome.
func (s *Service) CreateLearningOutcome(ctx context.Context, tenantID uuid.UUID, lo *LearningOutcome) (*LearningOutcome, error) {
	err := s.pool.QueryRow(ctx, `
		INSERT INTO learning_outcomes (tenant_id, sub_strand_id, description, sort_order)
		VALUES ($1, $2, $3, $4)
		RETURNING id, tenant_id, sub_strand_id, description, sort_order, created_at, updated_at
	`, tenantID, lo.SubStrandID, lo.Description, lo.SortOrder).Scan(
		&lo.ID, &lo.TenantID, &lo.SubStrandID, &lo.Description,
		&lo.SortOrder, &lo.CreatedAt, &lo.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert learning outcome: %w", err)
	}
	return lo, nil
}

// ListCoreCompetencies returns all core competencies for a tenant.
func (s *Service) ListCoreCompetencies(ctx context.Context, tenantID uuid.UUID) ([]CoreCompetency, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, tenant_id, name, kicd_code, description, created_at, updated_at
		FROM core_competencies
		WHERE tenant_id = $1
		ORDER BY name
	`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("query core competencies: %w", err)
	}
	defer rows.Close()

	var competencies []CoreCompetency
	for rows.Next() {
		var cc CoreCompetency
		if err := rows.Scan(
			&cc.ID, &cc.TenantID, &cc.Name, &cc.KICDCode,
			&cc.Description, &cc.CreatedAt, &cc.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan core competency: %w", err)
		}
		competencies = append(competencies, cc)
	}
	return competencies, rows.Err()
}

// CreateCoreCompetency inserts a new core competency.
func (s *Service) CreateCoreCompetency(ctx context.Context, tenantID uuid.UUID, cc *CoreCompetency) (*CoreCompetency, error) {
	err := s.pool.QueryRow(ctx, `
		INSERT INTO core_competencies (tenant_id, name, kicd_code, description)
		VALUES ($1, $2, $3, $4)
		RETURNING id, tenant_id, name, kicd_code, description, created_at, updated_at
	`, tenantID, cc.Name, cc.KICDCode, cc.Description).Scan(
		&cc.ID, &cc.TenantID, &cc.Name, &cc.KICDCode,
		&cc.Description, &cc.CreatedAt, &cc.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert core competency: %w", err)
	}
	return cc, nil
}

// ListValues returns all values for a tenant.
func (s *Service) ListValues(ctx context.Context, tenantID uuid.UUID) ([]Value, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, tenant_id, name, kicd_code, description, created_at, updated_at
		FROM values
		WHERE tenant_id = $1
		ORDER BY name
	`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("query values: %w", err)
	}
	defer rows.Close()

	var vals []Value
	for rows.Next() {
		var v Value
		if err := rows.Scan(
			&v.ID, &v.TenantID, &v.Name, &v.KICDCode,
			&v.Description, &v.CreatedAt, &v.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan value: %w", err)
		}
		vals = append(vals, v)
	}
	return vals, rows.Err()
}

// CreateValue inserts a new value.
func (s *Service) CreateValue(ctx context.Context, tenantID uuid.UUID, v *Value) (*Value, error) {
	err := s.pool.QueryRow(ctx, `
		INSERT INTO values (tenant_id, name, kicd_code, description)
		VALUES ($1, $2, $3, $4)
		RETURNING id, tenant_id, name, kicd_code, description, created_at, updated_at
	`, tenantID, v.Name, v.KICDCode, v.Description).Scan(
		&v.ID, &v.TenantID, &v.Name, &v.KICDCode,
		&v.Description, &v.CreatedAt, &v.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert value: %w", err)
	}
	return v, nil
}
