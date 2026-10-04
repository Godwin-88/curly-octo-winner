package reports

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Service handles report generation and analytics operations.
type Service struct {
	pool *pgxpool.Pool
}

// NewService creates a reports service.
func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

// --- Analytics ---

func (s *Service) SchoolOverview(ctx context.Context, tenantID uuid.UUID) (SchoolOverview, error) {
	var ov SchoolOverview
	err := s.pool.QueryRow(ctx, `SELECT tenant_id, learner_count FROM school_overview WHERE tenant_id = $1`, tenantID).
		Scan(&ov.TenantID, &ov.LearnerCount)
	if err != nil {
		return ov, err
	}
	return ov, nil
}

func (s *Service) StrandCoverage(ctx context.Context, tenantID uuid.UUID, grade, stream string, term, year int) ([]StrandCoverage, error) {
	query := `SELECT tenant_id, grade, stream, learning_area_id, learning_area, strand_id, strand_name,
		term, year, sub_strands_assessed, learners_assessed
		FROM strand_coverage WHERE tenant_id = $1`
	args := []any{tenantID}
	argIdx := 2
	if grade != "" {
		query += fmt.Sprintf(` AND grade = $%d`, argIdx)
		args = append(args, grade)
		argIdx++
	}
	if stream != "" {
		query += fmt.Sprintf(` AND stream = $%d`, argIdx)
		args = append(args, stream)
		argIdx++
	}
	if term > 0 {
		query += fmt.Sprintf(` AND term = $%d`, argIdx)
		args = append(args, term)
		argIdx++
	}
	if year > 0 {
		query += fmt.Sprintf(` AND year = $%d`, argIdx)
		args = append(args, year)
		argIdx++
	}
	query += ` ORDER BY grade, stream, learning_area, strand_name`

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []StrandCoverage
	for rows.Next() {
		var sc StrandCoverage
		if err := rows.Scan(&sc.TenantID, &sc.Grade, &sc.Stream, &sc.LearningAreaID, &sc.LearningArea,
			&sc.StrandID, &sc.StrandName, &sc.Term, &sc.Year, &sc.SubStrandsAssessed, &sc.LearnersAssessed); err != nil {
			return nil, err
		}
		out = append(out, sc)
	}
	return out, rows.Err()
}

func (s *Service) CompetencyDistribution(ctx context.Context, tenantID uuid.UUID, strandID, grade, stream string, term, year int) ([]CompetencyDistribution, error) {
	query := `SELECT tenant_id, grade, stream, strand_id, strand_name, term, year, rubric_level, learner_count
		FROM competency_distribution WHERE tenant_id = $1`
	args := []any{tenantID}
	argIdx := 2
	if strandID != "" {
		query += fmt.Sprintf(` AND strand_id = $%d`, argIdx)
		args = append(args, strandID)
		argIdx++
	}
	if grade != "" {
		query += fmt.Sprintf(` AND grade = $%d`, argIdx)
		args = append(args, grade)
		argIdx++
	}
	if stream != "" {
		query += fmt.Sprintf(` AND stream = $%d`, argIdx)
		args = append(args, stream)
		argIdx++
	}
	if term > 0 {
		query += fmt.Sprintf(` AND term = $%d`, argIdx)
		args = append(args, term)
		argIdx++
	}
	if year > 0 {
		query += fmt.Sprintf(` AND year = $%d`, argIdx)
		args = append(args, year)
		argIdx++
	}
	query += ` ORDER BY strand_name, rubric_level`

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []CompetencyDistribution
	for rows.Next() {
		var cd CompetencyDistribution
		if err := rows.Scan(&cd.TenantID, &cd.Grade, &cd.Stream, &cd.StrandID, &cd.StrandName,
			&cd.Term, &cd.Year, &cd.RubricLevel, &cd.LearnerCount); err != nil {
			return nil, err
		}
		out = append(out, cd)
	}
	return out, rows.Err()
}

func (s *Service) TeacherVelocity(ctx context.Context, tenantID uuid.UUID, term, year int) ([]TeacherVelocity, error) {
	query := `SELECT tenant_id, teacher_id, teacher_name, term, year, week_start, assessment_count
		FROM teacher_assessment_velocity WHERE tenant_id = $1`
	args := []any{tenantID}
	argIdx := 2
	if term > 0 {
		query += fmt.Sprintf(` AND term = $%d`, argIdx)
		args = append(args, term)
		argIdx++
	}
	if year > 0 {
		query += fmt.Sprintf(` AND year = $%d`, argIdx)
		args = append(args, year)
		argIdx++
	}
	query += ` ORDER BY week_start DESC, teacher_name`

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []TeacherVelocity
	for rows.Next() {
		var tv TeacherVelocity
		if err := rows.Scan(&tv.TenantID, &tv.TeacherID, &tv.TeacherName, &tv.Term, &tv.Year,
			&tv.WeekStart, &tv.AssessmentCount); err != nil {
			return nil, err
		}
		out = append(out, tv)
	}
	return out, rows.Err()
}

func (s *Service) LearnerPortfolio(ctx context.Context, tenantID uuid.UUID, grade, stream string, term, year int) ([]LearnerPortfolio, error) {
	query := `SELECT tenant_id, learner_id, learner_name, grade, stream, term, year,
		learning_areas_assessed, overall_avg_rubric, attendance_rate
		FROM learner_portfolio WHERE tenant_id = $1`
	args := []any{tenantID}
	argIdx := 2
	if grade != "" {
		query += fmt.Sprintf(` AND grade = $%d`, argIdx)
		args = append(args, grade)
		argIdx++
	}
	if stream != "" {
		query += fmt.Sprintf(` AND stream = $%d`, argIdx)
		args = append(args, stream)
		argIdx++
	}
	if term > 0 {
		query += fmt.Sprintf(` AND term = $%d`, argIdx)
		args = append(args, term)
		argIdx++
	}
	if year > 0 {
		query += fmt.Sprintf(` AND year = $%d`, argIdx)
		args = append(args, year)
		argIdx++
	}
	query += ` ORDER BY overall_avg_rubric ASC, learner_name`

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []LearnerPortfolio
	for rows.Next() {
		var lp LearnerPortfolio
		if err := rows.Scan(&lp.TenantID, &lp.LearnerID, &lp.LearnerName, &lp.Grade, &lp.Stream,
			&lp.Term, &lp.Year, &lp.LearningAreasAssessed, &lp.OverallAvgRubric, &lp.AttendanceRate); err != nil {
			return nil, err
		}
		out = append(out, lp)
	}
	return out, rows.Err()
}

func (s *Service) AtRiskLearners(ctx context.Context, tenantID uuid.UUID, term, year int) ([]AlertLearner, error) {
	portfolios, err := s.LearnerPortfolio(ctx, tenantID, "", "", term, year)
	if err != nil {
		return nil, err
	}
	var out []AlertLearner
	for _, lp := range portfolios {
		if lp.OverallAvgRubric < 2.5 || lp.AttendanceRate < 75 {
			out = append(out, AlertLearner{
				LearnerID:        lp.LearnerID,
				LearnerName:      lp.LearnerName,
				Grade:            lp.Grade,
				Stream:           lp.Stream,
				Term:             lp.Term,
				Year:             lp.Year,
				OverallAvgRubric: lp.OverallAvgRubric,
				AttendanceRate:   lp.AttendanceRate,
				AssessedAreas:    lp.LearningAreasAssessed,
			})
		}
	}
	return out, nil
}

func (s *Service) LearningAreaPerformance(ctx context.Context, tenantID, learnerID uuid.UUID, term, year int) ([]LearningAreaPerformance, error) {
	query := `SELECT tenant_id, learner_id, term, year, learning_area_id, learning_area, assessment_count, avg_rubric_level
		FROM learning_area_performance WHERE tenant_id = $1 AND learner_id = $2`
	args := []any{tenantID, learnerID}
	argIdx := 3
	if term > 0 {
		query += fmt.Sprintf(` AND term = $%d`, argIdx)
		args = append(args, term)
		argIdx++
	}
	if year > 0 {
		query += fmt.Sprintf(` AND year = $%d`, argIdx)
		args = append(args, year)
		argIdx++
	}
	query += ` ORDER BY learning_area`

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []LearningAreaPerformance
	for rows.Next() {
		var lap LearningAreaPerformance
		if err := rows.Scan(&lap.TenantID, &lap.LearnerID, &lap.Term, &lap.Year,
			&lap.LearningAreaID, &lap.LearningArea, &lap.AssessmentCount, &lap.AvgRubricLevel); err != nil {
			return nil, err
		}
		out = append(out, lap)
	}
	return out, rows.Err()
}

func round2(v float64) float64 {
	return float64(int(v*100+0.5)) / 100
}
