package reports

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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

// --- Report cards ---

const reportCardColumns = `rc.id, rc.tenant_id, rc.learner_id, l.full_name, l.grade, l.stream, l.upi,
	rc.term, rc.year, rc.status, rc.overall_rating, rc.core_competency_remarks,
	rc.teacher_comments, rc.attendance_summary, rc.generated_by, rc.generated_at,
	rc.created_at, rc.updated_at`

func scanReportCard(row pgx.Row) (*ReportCard, error) {
	var rc ReportCard
	err := row.Scan(
		&rc.ID, &rc.TenantID, &rc.LearnerID, &rc.LearnerName, &rc.Grade, &rc.Stream, &rc.UPI,
		&rc.Term, &rc.Year, &rc.Status, &rc.OverallRating, &rc.CoreCompetencyRemarks,
		&rc.TeacherComments, &rc.AttendanceSummary, &rc.GeneratedBy, &rc.GeneratedAt,
		&rc.CreatedAt, &rc.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if rc.CoreCompetencyRemarks == nil {
		rc.CoreCompetencyRemarks = map[string]string{}
	}
	if rc.TeacherComments == nil {
		rc.TeacherComments = map[string]string{}
	}
	return &rc, nil
}

const reportCardItemColumns = `rci.id, rci.tenant_id, rci.report_card_id, rci.learning_area_id, rci.strand_id, rci.sub_strand_id,
	COALESCE(la.name, ''), COALESCE(str.name, ''), COALESCE(s.name, ''),
	rci.rubric_level, rci.comment, rci.sort_order, rci.created_at`

func (s *Service) listReportCardItems(ctx context.Context, tenantID, reportCardID uuid.UUID) ([]ReportCardItem, error) {
	query := fmt.Sprintf(`SELECT %s FROM report_card_items rci
		LEFT JOIN sub_strands s ON s.id = rci.sub_strand_id AND s.tenant_id = rci.tenant_id
		LEFT JOIN strands str ON str.id = rci.strand_id AND str.tenant_id = rci.tenant_id
		LEFT JOIN learning_areas la ON la.id = rci.learning_area_id AND la.tenant_id = rci.tenant_id
		WHERE rci.tenant_id = $1 AND rci.report_card_id = $2
		ORDER BY rci.sort_order, rci.created_at`, reportCardItemColumns)
	rows, err := s.pool.Query(ctx, query, tenantID, reportCardID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []ReportCardItem
	for rows.Next() {
		var it ReportCardItem
		var rubricLevel *int
		if err := rows.Scan(
			&it.ID, &it.TenantID, &it.ReportCardID, &it.LearningAreaID, &it.StrandID, &it.SubStrandID,
			&it.LearningArea, &it.StrandName, &it.SubStrandName,
			&rubricLevel, &it.Comment, &it.SortOrder, &it.CreatedAt,
		); err != nil {
			return nil, err
		}
		if rubricLevel != nil {
			it.RubricLevel = rubricLevel
			it.RubricLabel = rubricLabel(*rubricLevel)
		}
		items = append(items, it)
	}
	return items, rows.Err()
}

func rubricLabel(level int) string {
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

func (s *Service) ListReportCards(ctx context.Context, tenantID uuid.UUID, learnerID string, term, year int) ([]ReportCard, error) {
	query := fmt.Sprintf(`SELECT %s FROM report_cards rc
		JOIN learners l ON l.id = rc.learner_id
		WHERE rc.tenant_id = $1`, reportCardColumns)
	args := []any{tenantID}
	argIdx := 2

	if learnerID != "" {
		query += fmt.Sprintf(` AND rc.learner_id = $%d`, argIdx)
		args = append(args, learnerID)
		argIdx++
	}
	if term > 0 {
		query += fmt.Sprintf(` AND rc.term = $%d`, argIdx)
		args = append(args, term)
		argIdx++
	}
	if year > 0 {
		query += fmt.Sprintf(` AND rc.year = $%d`, argIdx)
		args = append(args, year)
		argIdx++
	}
	query += ` ORDER BY rc.year DESC, rc.term DESC, l.full_name`

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cards []ReportCard
	for rows.Next() {
		rc, err := scanReportCard(rows)
		if err != nil {
			return nil, err
		}
		cards = append(cards, *rc)
	}
	return cards, rows.Err()
}

func (s *Service) GetReportCard(ctx context.Context, tenantID, id uuid.UUID) (*ReportCard, error) {
	query := fmt.Sprintf(`SELECT %s FROM report_cards rc
		JOIN learners l ON l.id = rc.learner_id
		WHERE rc.tenant_id = $1 AND rc.id = $2`, reportCardColumns)
	rc, err := scanReportCard(s.pool.QueryRow(ctx, query, tenantID, id))
	if err != nil {
		return nil, err
	}
	items, err := s.listReportCardItems(ctx, tenantID, rc.ID)
	if err != nil {
		return nil, err
	}
	rc.Items = items
	return rc, nil
}

func (s *Service) GenerateReportCard(ctx context.Context, tenantID, learnerID uuid.UUID, term, year int, req GenerateReportCardRequest) (*ReportCard, error) {
	err := s.pool.QueryRow(ctx,
		`SELECT full_name FROM learners WHERE tenant_id = $1 AND id = $2`,
		tenantID, learnerID).Scan(new(string))
	if err != nil {
		return nil, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	status := "final"
	if req.Status != nil {
		status = *req.Status
	}
	overallRating := req.OverallRating
	generatedBy := req.GeneratedBy
	generatedAt := time.Now()

	rows, err := tx.Query(ctx, `
		SELECT DISTINCT ON (a.sub_strand_id)
			a.sub_strand_id, s.name, str.id, str.name, la.id, la.name,
			a.rubric_level, a.note
		FROM assessments a
		JOIN sub_strands s ON s.id = a.sub_strand_id AND s.tenant_id = a.tenant_id
		JOIN strands str ON str.id = s.strand_id AND str.tenant_id = a.tenant_id
		JOIN learning_areas la ON la.id = str.learning_area_id AND la.tenant_id = a.tenant_id
		WHERE a.tenant_id = $1 AND a.learner_id = $2 AND a.term = $3 AND a.year = $4
		ORDER BY a.sub_strand_id, a.created_at DESC
	`, tenantID, learnerID, term, year)
	if err != nil {
		return nil, err
	}

	type aggItem struct {
		subStrandID    uuid.UUID
		subStrandName  string
		strandID       uuid.UUID
		strandName     string
		learningAreaID uuid.UUID
		learningArea   string
		rubricLevel    int
		note           *string
	}
	var items []aggItem
	for rows.Next() {
		var it aggItem
		if err := rows.Scan(
			&it.subStrandID, &it.subStrandName, &it.strandID, &it.strandName,
			&it.learningAreaID, &it.learningArea, &it.rubricLevel, &it.note,
		); err != nil {
			rows.Close()
			return nil, err
		}
		items = append(items, it)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(items) == 0 && status != "draft" {
		return nil, errors.New("no assessments found for this learner/term; cannot finalize report card")
	}

	if overallRating == nil && len(items) > 0 {
		sum := 0
		for _, it := range items {
			sum += it.rubricLevel
		}
		avg := float64(sum) / float64(len(items))
		rounded := int(avg + 0.5)
		if rounded < 1 {
			rounded = 1
		}
		if rounded > 4 {
			rounded = 4
		}
		overallRating = &rounded
	}

	attendanceSummary := map[string]any{}
	var termStart, termEnd time.Time
	switch term {
	case 1:
		termStart = time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
		termEnd = time.Date(year, 4, 30, 23, 59, 59, 0, time.UTC)
	case 2:
		termStart = time.Date(year, 5, 1, 0, 0, 0, 0, time.UTC)
		termEnd = time.Date(year, 8, 31, 23, 59, 59, 0, time.UTC)
	default:
		termStart = time.Date(year, 9, 1, 0, 0, 0, 0, time.UTC)
		termEnd = time.Date(year, 12, 31, 23, 59, 59, 0, time.UTC)
	}

	var totalDays, presentDays, absentDays, lateDays, excusedDays int64
	var attendanceRate float64
	err = tx.QueryRow(ctx, `
		SELECT
			COUNT(*),
			COUNT(*) FILTER (WHERE status = 'present'),
			COUNT(*) FILTER (WHERE status = 'absent'),
			COUNT(*) FILTER (WHERE status = 'late'),
			COUNT(*) FILTER (WHERE status = 'excused')
		FROM attendance
		WHERE tenant_id = $1 AND learner_id = $2 AND date >= $3 AND date <= $4
	`, tenantID, learnerID, termStart.Format("2006-01-02"), termEnd.Format("2006-01-02")).
		Scan(&totalDays, &presentDays, &absentDays, &lateDays, &excusedDays)
	if err != nil {
		return nil, err
	}
	if totalDays > 0 {
		attendanceRate = float64(presentDays) / float64(totalDays) * 100
	}
	attendanceSummary["total_days"] = totalDays
	attendanceSummary["present_days"] = presentDays
	attendanceSummary["absent_days"] = absentDays
	attendanceSummary["late_days"] = lateDays
	attendanceSummary["excused_days"] = excusedDays
	attendanceSummary["attendance_rate"] = round2(attendanceRate)

	coreRemarks := req.CoreCompetencyRemarks
	if coreRemarks == nil {
		coreRemarks = map[string]string{}
	}
	teacherComments := req.TeacherComments
	if teacherComments == nil {
		teacherComments = map[string]string{}
	}

	rcID := uuid.UUID{}
	err = tx.QueryRow(ctx, `
		INSERT INTO report_cards (tenant_id, learner_id, term, year, status, overall_rating,
			core_competency_remarks, teacher_comments, attendance_summary, generated_by, generated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (tenant_id, learner_id, term, year)
		DO UPDATE SET
			status = EXCLUDED.status,
			overall_rating = EXCLUDED.overall_rating,
			core_competency_remarks = EXCLUDED.core_competency_remarks,
			teacher_comments = EXCLUDED.teacher_comments,
			attendance_summary = EXCLUDED.attendance_summary,
			generated_by = EXCLUDED.generated_by,
			generated_at = EXCLUDED.generated_at
		RETURNING id`, tenantID, learnerID, term, year, status, overallRating,
		coreRemarks, teacherComments, attendanceSummary, generatedBy, generatedAt).Scan(&rcID)
	if err != nil {
		return nil, err
	}

	if _, err := tx.Exec(ctx, `DELETE FROM report_card_items WHERE report_card_id = $1`, rcID); err != nil {
		return nil, err
	}
	for idx, it := range items {
		comment := it.note
		rubric := it.rubricLevel
		if _, err := tx.Exec(ctx, `
			INSERT INTO report_card_items (tenant_id, report_card_id, learning_area_id, strand_id,
				sub_strand_id, rubric_level, comment, sort_order)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
			tenantID, rcID, it.learningAreaID, it.strandID, it.subStrandID,
			rubric, comment, idx+1,
		); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.GetReportCard(ctx, tenantID, rcID)
}

func (s *Service) UpdateReportCard(ctx context.Context, tenantID, id uuid.UUID, req UpdateReportCardRequest) (*ReportCard, error) {
	if _, err := s.GetReportCard(ctx, tenantID, id); err != nil {
		return nil, err
	}
	query := fmt.Sprintf(`UPDATE report_cards rc SET
		status = COALESCE($3, rc.status),
		overall_rating = COALESCE($4, rc.overall_rating),
		core_competency_remarks = COALESCE($5, rc.core_competency_remarks),
		teacher_comments = COALESCE($6, rc.teacher_comments)
		WHERE rc.tenant_id = $1 AND rc.id = $2
		RETURNING %s`, reportCardColumns+` FROM learners l WHERE l.id = rc.learner_id`)
	rc, err := scanReportCard(s.pool.QueryRow(ctx, query,
		tenantID, id, req.Status, req.OverallRating, req.CoreCompetencyRemarks, req.TeacherComments,
	))
	if err != nil {
		return nil, err
	}
	items, err := s.listReportCardItems(ctx, tenantID, rc.ID)
	if err != nil {
		return nil, err
	}
	rc.Items = items
	return rc, nil
}

func (s *Service) DeleteReportCard(ctx context.Context, tenantID, id uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM report_cards WHERE tenant_id = $1 AND id = $2`, tenantID, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
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
