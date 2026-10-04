package curriculum

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// HasChildrenError reports that a curriculum item cannot be deleted because
// other rows still depend on it.
//
// This guard is deliberate: every child foreign key in 009_curriculum.sql and
// 010_assessments.sql is ON DELETE CASCADE, so an unguarded DELETE on a
// learning area would silently take its strands, their sub-strands, the
// learning outcomes under those, and every assessment recorded against them.
// Losing a term of assessment records to a mis-click is not recoverable, so the
// API refuses and explains what to remove first.
type HasChildrenError struct {
	Parent string // human name of the item targeted, e.g. "Mathematics"
	Child  string // what blocks it, e.g. "strands"
	Count  int    // how many of them exist
}

func (e *HasChildrenError) Error() string {
	noun := e.Child
	if e.Count == 1 {
		noun = singularise(e.Child)
	}
	return fmt.Sprintf("%q still has %d %s — remove them first", e.Parent, e.Count, noun)
}

// singularise handles the two plural forms this file produces. A tiny switch
// beats pulling in a dependency for the sake of one word in an error message.
func singularise(plural string) string {
	switch {
	case strings.HasSuffix(plural, "ies"):
		return strings.TrimSuffix(plural, "ies") + "y"
	case strings.HasSuffix(plural, "s"):
		return strings.TrimSuffix(plural, "s")
	default:
		return plural
	}
}

// ErrNotFound means "no such row for this tenant". Callers wrap it with
// context ("learning area: %w") and handlers translate it into a 404 with
// errors.Is, so no code has to string-match an error message.
var ErrNotFound = errors.New("not found")

// IsNotFound reports whether err means "no such row for this tenant".
func IsNotFound(err error) bool { return errors.Is(err, ErrNotFound) }

// childCount returns how many rows in childTable point at parentID, scoped to
// the tenant so another tenant's rows can never influence a decision.
func (s *Service) childCount(ctx context.Context, tenantID uuid.UUID, childTable, parentColumn string, parentID uuid.UUID) (int, error) {
	var n int
	// childTable and parentColumn are only ever the literals passed by the
	// callers in this file, never request input, so building the query with fmt
	// cannot be an injection vector.
	q := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE tenant_id = $1 AND %s = $2", childTable, parentColumn)
	if err := s.pool.QueryRow(ctx, q, tenantID, parentID).Scan(&n); err != nil {
		return 0, fmt.Errorf("count %s: %w", childTable, err)
	}
	return n, nil
}

// UpdateLearningArea edits a learning area in place.
func (s *Service) UpdateLearningArea(ctx context.Context, tenantID, id uuid.UUID, la *LearningArea) (*LearningArea, error) {
	err := s.pool.QueryRow(ctx, `
		UPDATE learning_areas
		SET name = $3, kicd_code = $4, grade_level = $5, description = $6, updated_at = now()
		WHERE tenant_id = $1 AND id = $2
		RETURNING id, tenant_id, name, COALESCE(kicd_code, ''), grade_level, COALESCE(description, ''), created_at, updated_at
	`, tenantID, id, la.Name, nullIfEmpty(la.KICDCode), la.GradeLevel, la.Description).Scan(
		&la.ID, &la.TenantID, &la.Name, &la.KICDCode,
		&la.GradeLevel, &la.Description, &la.CreatedAt, &la.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("learning area: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("update learning area: %w", err)
	}
	return la, nil
}

// DeleteLearningArea removes a learning area that has no strands.
func (s *Service) DeleteLearningArea(ctx context.Context, tenantID, id uuid.UUID) error {
	var name string
	err := s.pool.QueryRow(ctx,
		`SELECT name FROM learning_areas WHERE tenant_id = $1 AND id = $2`, tenantID, id).Scan(&name)
	if err != nil {
		if err == pgx.ErrNoRows {
			return fmt.Errorf("learning area: %w", ErrNotFound)
		}
		return fmt.Errorf("load learning area: %w", err)
	}

	n, err := s.childCount(ctx, tenantID, "strands", "learning_area_id", id)
	if err != nil {
		return err
	}
	if n > 0 {
		return &HasChildrenError{Parent: name, Child: "strands", Count: n}
	}

	if _, err := s.pool.Exec(ctx,
		`DELETE FROM learning_areas WHERE tenant_id = $1 AND id = $2`, tenantID, id); err != nil {
		return fmt.Errorf("delete learning area: %w", err)
	}
	return nil
}

// UpdateStrand edits a strand in place.
func (s *Service) UpdateStrand(ctx context.Context, tenantID, id uuid.UUID, st *Strand) (*Strand, error) {
	err := s.pool.QueryRow(ctx, `
		UPDATE strands
		SET name = $3, kicd_code = $4, description = $5, updated_at = now()
		WHERE tenant_id = $1 AND id = $2
		RETURNING id, tenant_id, learning_area_id, name, COALESCE(kicd_code, ''), COALESCE(description, ''), created_at, updated_at
	`, tenantID, id, st.Name, nullIfEmpty(st.KICDCode), st.Description).Scan(
		&st.ID, &st.TenantID, &st.LearningAreaID, &st.Name, &st.KICDCode,
		&st.Description, &st.CreatedAt, &st.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("strand: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("update strand: %w", err)
	}
	return st, nil
}

// DeleteStrand removes a strand that has no sub-strands.
func (s *Service) DeleteStrand(ctx context.Context, tenantID, id uuid.UUID) error {
	var name string
	err := s.pool.QueryRow(ctx,
		`SELECT name FROM strands WHERE tenant_id = $1 AND id = $2`, tenantID, id).Scan(&name)
	if err != nil {
		if err == pgx.ErrNoRows {
			return fmt.Errorf("strand: %w", ErrNotFound)
		}
		return fmt.Errorf("load strand: %w", err)
	}

	n, err := s.childCount(ctx, tenantID, "sub_strands", "strand_id", id)
	if err != nil {
		return err
	}
	if n > 0 {
		return &HasChildrenError{Parent: name, Child: "sub-strands", Count: n}
	}

	if _, err := s.pool.Exec(ctx,
		`DELETE FROM strands WHERE tenant_id = $1 AND id = $2`, tenantID, id); err != nil {
		return fmt.Errorf("delete strand: %w", err)
	}
	return nil
}

// UpdateSubStrand edits a sub-strand in place.
func (s *Service) UpdateSubStrand(ctx context.Context, tenantID, id uuid.UUID, ss *SubStrand) (*SubStrand, error) {
	err := s.pool.QueryRow(ctx, `
		UPDATE sub_strands
		SET name = $3, kicd_code = $4, description = $5, updated_at = now()
		WHERE tenant_id = $1 AND id = $2
		RETURNING id, tenant_id, strand_id, name, COALESCE(kicd_code, ''), COALESCE(description, ''), created_at, updated_at
	`, tenantID, id, ss.Name, nullIfEmpty(ss.KICDCode), ss.Description).Scan(
		&ss.ID, &ss.TenantID, &ss.StrandID, &ss.Name, &ss.KICDCode,
		&ss.Description, &ss.CreatedAt, &ss.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("sub-strand: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("update sub-strand: %w", err)
	}
	return ss, nil
}

// DeleteSubStrand removes a sub-strand, but only while nothing is recorded
// against it: learning outcomes are curriculum content, assessments are a
// teacher's term of work.
func (s *Service) DeleteSubStrand(ctx context.Context, tenantID, id uuid.UUID) error {
	var name string
	err := s.pool.QueryRow(ctx,
		`SELECT name FROM sub_strands WHERE tenant_id = $1 AND id = $2`, tenantID, id).Scan(&name)
	if err != nil {
		if err == pgx.ErrNoRows {
			return fmt.Errorf("sub-strand: %w", ErrNotFound)
		}
		return fmt.Errorf("load sub-strand: %w", err)
	}

	outcomes, err := s.childCount(ctx, tenantID, "learning_outcomes", "sub_strand_id", id)
	if err != nil {
		return err
	}
	if outcomes > 0 {
		return &HasChildrenError{Parent: name, Child: "learning outcomes", Count: outcomes}
	}

	assessments, err := s.childCount(ctx, tenantID, "assessments", "sub_strand_id", id)
	if err != nil {
		return err
	}
	if assessments > 0 {
		return &HasChildrenError{Parent: name, Child: "recorded assessments", Count: assessments}
	}

	if _, err := s.pool.Exec(ctx,
		`DELETE FROM sub_strands WHERE tenant_id = $1 AND id = $2`, tenantID, id); err != nil {
		return fmt.Errorf("delete sub-strand: %w", err)
	}
	return nil
}

// UpdateCoreCompetency edits a core competency in place.
func (s *Service) UpdateCoreCompetency(ctx context.Context, tenantID, id uuid.UUID, cc *CoreCompetency) (*CoreCompetency, error) {
	err := s.pool.QueryRow(ctx, `
		UPDATE core_competencies
		SET name = $3, kicd_code = $4, description = $5, updated_at = now()
		WHERE tenant_id = $1 AND id = $2
		RETURNING id, tenant_id, name, COALESCE(kicd_code, ''), COALESCE(description, ''), created_at, updated_at
	`, tenantID, id, cc.Name, nullIfEmpty(cc.KICDCode), cc.Description).Scan(
		&cc.ID, &cc.TenantID, &cc.Name, &cc.KICDCode, &cc.Description,
		&cc.CreatedAt, &cc.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("core competency: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("update core competency: %w", err)
	}
	return cc, nil
}

// DeleteCoreCompetency removes a core competency.
func (s *Service) DeleteCoreCompetency(ctx context.Context, tenantID, id uuid.UUID) error {
	tag, err := s.pool.Exec(ctx,
		`DELETE FROM core_competencies WHERE tenant_id = $1 AND id = $2`, tenantID, id)
	if err != nil {
		return fmt.Errorf("delete core competency: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("core competency: %w", ErrNotFound)
	}
	return nil
}

// UpdateValue edits a KICD value in place.
func (s *Service) UpdateValue(ctx context.Context, tenantID, id uuid.UUID, v *Value) (*Value, error) {
	err := s.pool.QueryRow(ctx, `
		UPDATE values
		SET name = $3, kicd_code = $4, description = $5, updated_at = now()
		WHERE tenant_id = $1 AND id = $2
		RETURNING id, tenant_id, name, COALESCE(kicd_code, ''), COALESCE(description, ''), created_at, updated_at
	`, tenantID, id, v.Name, nullIfEmpty(v.KICDCode), v.Description).Scan(
		&v.ID, &v.TenantID, &v.Name, &v.KICDCode, &v.Description,
		&v.CreatedAt, &v.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("value: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("update value: %w", err)
	}
	return v, nil
}

// DeleteValue removes a KICD value.
func (s *Service) DeleteValue(ctx context.Context, tenantID, id uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM values WHERE tenant_id = $1 AND id = $2`, tenantID, id)
	if err != nil {
		return fmt.Errorf("delete value: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("value: %w", ErrNotFound)
	}
	return nil
}
