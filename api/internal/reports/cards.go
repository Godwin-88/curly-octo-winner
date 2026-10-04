package reports

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/shule360/api/pkg/apperr"
)

// A report card is a draft until it is published.
//
// A draft is built from the term's observations and can be rebuilt, edited or
// thrown away. Publishing makes it final: it is what the parent is given, so
// it is no longer edited, rebuilt or deleted. A principal who finds a mistake
// reopens it, which makes it a draft again.
const (
	StatusDraft = "draft"
	StatusFinal = "final"
)

// ReportCard is a CBC report card for a learner for one term.
type ReportCard struct {
	ID                    uuid.UUID         `json:"id"`
	TenantID              uuid.UUID         `json:"tenant_id"`
	LearnerID             uuid.UUID         `json:"learner_id"`
	LearnerName           string            `json:"learner_name,omitempty"`
	Grade                 string            `json:"grade,omitempty"`
	Stream                string            `json:"stream,omitempty"`
	UPI                   string            `json:"upi,omitempty"`
	Term                  int               `json:"term"`
	Year                  int               `json:"year"`
	Status                string            `json:"status"`
	OverallRating         *int              `json:"overall_rating,omitempty"`
	OverallLabel          string            `json:"overall_label,omitempty"`
	CoreCompetencyRemarks map[string]string `json:"core_competency_remarks"`
	TeacherComments       map[string]string `json:"teacher_comments"`
	AttendanceSummary     map[string]any    `json:"attendance_summary,omitempty"`
	GeneratedBy           *uuid.UUID        `json:"generated_by,omitempty"`
	GeneratedAt           time.Time         `json:"generated_at"`
	PublishedBy           *uuid.UUID        `json:"published_by,omitempty"`
	PublishedAt           *time.Time        `json:"published_at,omitempty"`
	CreatedAt             time.Time         `json:"created_at"`
	UpdatedAt             time.Time         `json:"updated_at"`
	// Items are filled in when one card is read, not in a list.
	Items []ReportCardItem `json:"items,omitempty"`
}

// ReportCardItem is one sub-strand on a report card: the level the learner
// reached at their most recent observation in the term.
type ReportCardItem struct {
	ID             uuid.UUID  `json:"id"`
	ReportCardID   uuid.UUID  `json:"report_card_id"`
	LearningAreaID *uuid.UUID `json:"learning_area_id,omitempty"`
	StrandID       *uuid.UUID `json:"strand_id,omitempty"`
	SubStrandID    *uuid.UUID `json:"sub_strand_id,omitempty"`
	LearningArea   string     `json:"learning_area,omitempty"`
	StrandName     string     `json:"strand_name,omitempty"`
	SubStrandName  string     `json:"sub_strand_name,omitempty"`
	RubricLevel    *int       `json:"rubric_level,omitempty"`
	RubricLabel    string     `json:"rubric_label,omitempty"`
	Comment        string     `json:"comment,omitempty"`
	SortOrder      int        `json:"sort_order"`
}

// CardInput is what a teacher writes on a card. Who generated or published it
// and whether it is final are never taken from a request.
type CardInput struct {
	OverallRating         *int              `json:"overall_rating,omitempty"`
	CoreCompetencyRemarks map[string]string `json:"core_competency_remarks,omitempty"`
	TeacherComments       map[string]string `json:"teacher_comments,omitempty"`
}

// CardFilter narrows a list of report cards.
type CardFilter struct {
	LearnerID string
	Grade     string
	Stream    string
	Status    string
	Term      int
	Year      int
}

// ClassResult says what generating a class's report cards did.
type ClassResult struct {
	Generated int `json:"generated"`
	// Published cards are left alone.
	SkippedPublished int `json:"skipped_published"`
	// Learners nobody has recorded an observation for this term.
	NoObservations []string `json:"no_observations"`
}

const maxCommentLength = 1000

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
		return ""
	}
}

const reportCardColumns = `rc.id, rc.tenant_id, rc.learner_id, l.full_name, l.grade, COALESCE(l.stream, ''), COALESCE(l.upi, ''),
	rc.term, rc.year, rc.status, rc.overall_rating, rc.core_competency_remarks,
	rc.teacher_comments, rc.attendance_summary, rc.generated_by, rc.generated_at,
	rc.published_by, rc.published_at, rc.created_at, rc.updated_at`

const reportCardFrom = ` FROM report_cards rc JOIN learners l ON l.id = rc.learner_id AND l.tenant_id = rc.tenant_id `

func scanReportCard(row pgx.Row) (*ReportCard, error) {
	var rc ReportCard
	err := row.Scan(
		&rc.ID, &rc.TenantID, &rc.LearnerID, &rc.LearnerName, &rc.Grade, &rc.Stream, &rc.UPI,
		&rc.Term, &rc.Year, &rc.Status, &rc.OverallRating, &rc.CoreCompetencyRemarks,
		&rc.TeacherComments, &rc.AttendanceSummary, &rc.GeneratedBy, &rc.GeneratedAt,
		&rc.PublishedBy, &rc.PublishedAt, &rc.CreatedAt, &rc.UpdatedAt,
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
	if rc.OverallRating != nil {
		rc.OverallLabel = rubricLabel(*rc.OverallRating)
	}
	return &rc, nil
}

func (s *Service) listReportCardItems(ctx context.Context, tenantID, reportCardID uuid.UUID) ([]ReportCardItem, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT rci.id, rci.report_card_id, rci.learning_area_id, rci.strand_id, rci.sub_strand_id,
			COALESCE(la.name, ''), COALESCE(str.name, ''), COALESCE(s.name, ''),
			rci.rubric_level, COALESCE(rci.comment, ''), rci.sort_order
		FROM report_card_items rci
		LEFT JOIN sub_strands s ON s.id = rci.sub_strand_id AND s.tenant_id = rci.tenant_id
		LEFT JOIN strands str ON str.id = rci.strand_id AND str.tenant_id = rci.tenant_id
		LEFT JOIN learning_areas la ON la.id = rci.learning_area_id AND la.tenant_id = rci.tenant_id
		WHERE rci.tenant_id = $1 AND rci.report_card_id = $2
		ORDER BY rci.sort_order`, tenantID, reportCardID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []ReportCardItem{}
	for rows.Next() {
		var it ReportCardItem
		if err := rows.Scan(
			&it.ID, &it.ReportCardID, &it.LearningAreaID, &it.StrandID, &it.SubStrandID,
			&it.LearningArea, &it.StrandName, &it.SubStrandName,
			&it.RubricLevel, &it.Comment, &it.SortOrder,
		); err != nil {
			return nil, err
		}
		if it.RubricLevel != nil {
			it.RubricLabel = rubricLabel(*it.RubricLevel)
		}
		items = append(items, it)
	}
	return items, rows.Err()
}

// ListReportCards lists a school's report cards, newest term first.
func (s *Service) ListReportCards(ctx context.Context, tenantID uuid.UUID, f CardFilter) ([]ReportCard, error) {
	query := `SELECT ` + reportCardColumns + reportCardFrom + ` WHERE rc.tenant_id = $1`
	args := []any{tenantID}
	add := func(clause string, value any) {
		args = append(args, value)
		query += fmt.Sprintf(clause, len(args))
	}
	if f.LearnerID != "" {
		id, err := uuid.Parse(f.LearnerID)
		if err != nil {
			return nil, apperr.Invalid("learner_id is not a valid id.")
		}
		add(` AND rc.learner_id = $%d`, id)
	}
	if f.Grade != "" {
		add(` AND l.grade = $%d`, f.Grade)
	}
	if f.Stream != "" {
		add(` AND l.stream = $%d`, f.Stream)
	}
	if f.Status != "" {
		add(` AND rc.status = $%d`, f.Status)
	}
	if f.Term > 0 {
		add(` AND rc.term = $%d`, f.Term)
	}
	if f.Year > 0 {
		add(` AND rc.year = $%d`, f.Year)
	}
	query += ` ORDER BY rc.year DESC, rc.term DESC, l.grade, l.full_name`

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cards := []ReportCard{}
	for rows.Next() {
		rc, err := scanReportCard(rows)
		if err != nil {
			return nil, err
		}
		cards = append(cards, *rc)
	}
	return cards, rows.Err()
}

// GetReportCard returns one card with its items.
func (s *Service) GetReportCard(ctx context.Context, tenantID, id uuid.UUID) (*ReportCard, error) {
	rc, err := scanReportCard(s.pool.QueryRow(ctx,
		`SELECT `+reportCardColumns+reportCardFrom+` WHERE rc.tenant_id = $1 AND rc.id = $2`, tenantID, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperr.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if rc.Items, err = s.listReportCardItems(ctx, tenantID, rc.ID); err != nil {
		return nil, err
	}
	return rc, nil
}

func checkInput(in CardInput) error {
	if in.OverallRating != nil && (*in.OverallRating < 1 || *in.OverallRating > 4) {
		return apperr.Invalid("The overall rating is a level from 1 (Below Expectation) to 4 (Exceeding Expectation).")
	}
	for _, remarks := range []map[string]string{in.CoreCompetencyRemarks, in.TeacherComments} {
		for key, text := range remarks {
			if strings.TrimSpace(key) == "" {
				return apperr.Invalid("A comment needs a heading.")
			}
			if len([]rune(text)) > maxCommentLength {
				return apperr.Invalid("Keep each comment under %d characters.", maxCommentLength)
			}
		}
	}
	return nil
}

func checkTerm(term, year int) error {
	if term < 1 || term > 3 {
		return apperr.Invalid("Term must be 1, 2 or 3.")
	}
	if year < 2000 || year > time.Now().Year()+1 {
		return apperr.Invalid("That year is not one a report card can be made for.")
	}
	return nil
}

// termRange is the first and last day of a term (January-April, May-August,
// September-December).
func termRange(term, year int) (string, string) {
	switch term {
	case 1:
		return fmt.Sprintf("%d-01-01", year), fmt.Sprintf("%d-04-30", year)
	case 2:
		return fmt.Sprintf("%d-05-01", year), fmt.Sprintf("%d-08-31", year)
	default:
		return fmt.Sprintf("%d-09-01", year), fmt.Sprintf("%d-12-31", year)
	}
}

// GenerateReportCard builds (or rebuilds) a learner's draft card for a term
// from the observations recorded in it: for each sub-strand, the level at the
// most recent observation.
//
// A published card is not rebuilt. What a teacher has already written on a
// draft is kept unless the request replaces it.
func (s *Service) GenerateReportCard(ctx context.Context, tenantID, learnerID uuid.UUID, term, year int, actor uuid.UUID, in CardInput) (*ReportCard, error) {
	if err := checkTerm(term, year); err != nil {
		return nil, err
	}
	if err := checkInput(in); err != nil {
		return nil, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	// Lock the learner so two people generating the same card take turns.
	var learnerName string
	err = tx.QueryRow(ctx, `SELECT full_name FROM learners WHERE tenant_id = $1 AND id = $2 FOR UPDATE`,
		tenantID, learnerID).Scan(&learnerName)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperr.Invalid("That learner is not in your school.")
	}
	if err != nil {
		return nil, err
	}

	var existing string
	err = tx.QueryRow(ctx, `SELECT status FROM report_cards WHERE tenant_id = $1 AND learner_id = $2 AND term = $3 AND year = $4`,
		tenantID, learnerID, term, year).Scan(&existing)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	if existing == StatusFinal {
		return nil, apperr.Conflict("%s's report card for Term %d %d is published. A principal can reopen it if it needs to change.", learnerName, term, year)
	}

	rows, err := tx.Query(ctx, `
		SELECT sub_strand_id, strand_id, learning_area_id, rubric_level, note FROM (
			SELECT DISTINCT ON (a.sub_strand_id)
				a.sub_strand_id, str.id AS strand_id, la.id AS learning_area_id, a.rubric_level,
				COALESCE(a.note, '') AS note, la.name AS area, str.name AS strand, s.name AS sub_strand
			FROM assessments a
			JOIN sub_strands s ON s.id = a.sub_strand_id AND s.tenant_id = a.tenant_id
			JOIN strands str ON str.id = s.strand_id AND str.tenant_id = a.tenant_id
			JOIN learning_areas la ON la.id = str.learning_area_id AND la.tenant_id = a.tenant_id
			WHERE a.tenant_id = $1 AND a.learner_id = $2 AND a.term = $3 AND a.year = $4
			ORDER BY a.sub_strand_id, a.created_at DESC
		) latest
		ORDER BY area, strand, sub_strand
	`, tenantID, learnerID, term, year)
	if err != nil {
		return nil, err
	}
	type line struct {
		subStrand, strand, area uuid.UUID
		level                   int
		note                    string
	}
	var lines []line
	for rows.Next() {
		var l line
		if err := rows.Scan(&l.subStrand, &l.strand, &l.area, &l.level, &l.note); err != nil {
			rows.Close()
			return nil, err
		}
		lines = append(lines, l)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(lines) == 0 {
		return nil, apperr.Invalid("No observations are recorded for %s in Term %d %d, so there is nothing to put on a report card.", learnerName, term, year)
	}

	// The overall rating is the average level, to the nearest whole level,
	// unless the teacher sets it.
	overall := in.OverallRating
	if overall == nil {
		sum := 0
		for _, l := range lines {
			sum += l.level
		}
		rounded := int(float64(sum)/float64(len(lines)) + 0.5)
		overall = &rounded
	}

	// A learner who came late was at school: late counts as attended.
	start, end := termRange(term, year)
	var total, present, absent, late, excused int64
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*),
			COUNT(*) FILTER (WHERE status = 'present'),
			COUNT(*) FILTER (WHERE status = 'absent'),
			COUNT(*) FILTER (WHERE status = 'late'),
			COUNT(*) FILTER (WHERE status = 'excused')
		FROM attendance
		WHERE tenant_id = $1 AND learner_id = $2 AND date BETWEEN $3 AND $4
	`, tenantID, learnerID, start, end).Scan(&total, &present, &absent, &late, &excused); err != nil {
		return nil, err
	}
	rate := 0.0
	if total > 0 {
		rate = round2(float64(present+late) / float64(total) * 100)
	}
	attendance := map[string]any{
		"total_days": total, "present_days": present, "absent_days": absent,
		"late_days": late, "excused_days": excused, "attendance_rate": rate,
	}

	var id uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO report_cards (tenant_id, learner_id, term, year, status, overall_rating,
			core_competency_remarks, teacher_comments, attendance_summary, generated_by, generated_at)
		VALUES ($1, $2, $3, $4, 'draft', $5, COALESCE($6, '{}'::jsonb), COALESCE($7, '{}'::jsonb), $8, $9, now())
		ON CONFLICT (tenant_id, learner_id, term, year)
		DO UPDATE SET
			overall_rating = EXCLUDED.overall_rating,
			core_competency_remarks = COALESCE($6, report_cards.core_competency_remarks),
			teacher_comments = COALESCE($7, report_cards.teacher_comments),
			attendance_summary = EXCLUDED.attendance_summary,
			generated_by = EXCLUDED.generated_by,
			generated_at = EXCLUDED.generated_at
		RETURNING id`,
		tenantID, learnerID, term, year, overall,
		nilIfEmpty(in.CoreCompetencyRemarks), nilIfEmpty(in.TeacherComments), attendance, nullableID(actor)).Scan(&id)
	if err != nil {
		return nil, err
	}

	if _, err := tx.Exec(ctx, `DELETE FROM report_card_items WHERE tenant_id = $1 AND report_card_id = $2`, tenantID, id); err != nil {
		return nil, err
	}
	for i, l := range lines {
		if _, err := tx.Exec(ctx, `
			INSERT INTO report_card_items (tenant_id, report_card_id, learning_area_id, strand_id, sub_strand_id, rubric_level, comment, sort_order)
			VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7, ''), $8)`,
			tenantID, id, l.area, l.strand, l.subStrand, l.level, l.note, i+1); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.GetReportCard(ctx, tenantID, id)
}

// nilIfEmpty makes "not given" a NULL, so what is already on a draft is kept.
func nilIfEmpty(m map[string]string) any {
	if m == nil {
		return nil
	}
	return m
}

func nullableID(id uuid.UUID) any {
	if id == uuid.Nil {
		return nil
	}
	return id
}

// GenerateForClass builds the draft cards of every learner in a grade (and
// stream, if given) who has an observation this term. One learner's card
// failing does not stop the rest; published cards are left as they are.
func (s *Service) GenerateForClass(ctx context.Context, tenantID uuid.UUID, grade, stream string, term, year int, actor uuid.UUID) (*ClassResult, error) {
	grade, stream = strings.TrimSpace(grade), strings.TrimSpace(stream)
	if grade == "" {
		return nil, apperr.Invalid("Choose the grade to make report cards for.")
	}
	if err := checkTerm(term, year); err != nil {
		return nil, err
	}

	rows, err := s.pool.Query(ctx, `
		SELECT l.id, l.full_name,
			EXISTS (SELECT 1 FROM assessments a WHERE a.tenant_id = l.tenant_id AND a.learner_id = l.id AND a.term = $4 AND a.year = $5)
		FROM learners l
		WHERE l.tenant_id = $1 AND l.grade = $2 AND ($3 = '' OR l.stream = $3)
		ORDER BY l.full_name`, tenantID, grade, stream, term, year)
	if err != nil {
		return nil, err
	}
	type learner struct {
		id       uuid.UUID
		name     string
		observed bool
	}
	var learners []learner
	for rows.Next() {
		var l learner
		if err := rows.Scan(&l.id, &l.name, &l.observed); err != nil {
			rows.Close()
			return nil, err
		}
		learners = append(learners, l)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(learners) == 0 {
		return nil, apperr.Invalid("There are no learners in that class.")
	}

	result := &ClassResult{NoObservations: []string{}}
	for _, l := range learners {
		if !l.observed {
			result.NoObservations = append(result.NoObservations, l.name)
			continue
		}
		_, err := s.GenerateReportCard(ctx, tenantID, l.id, term, year, actor, CardInput{})
		var conflict *apperr.ConflictError
		var refusal *apperr.ValidationError
		switch {
		case err == nil:
			result.Generated++
		case errors.As(err, &conflict):
			result.SkippedPublished++
		case errors.As(err, &refusal):
			result.NoObservations = append(result.NoObservations, l.name)
		default:
			return nil, fmt.Errorf("report card for learner %s: %w", l.id, err)
		}
	}
	return result, nil
}

// lockCard reads a card's status inside a transaction, holding the row.
func lockCard(ctx context.Context, tx pgx.Tx, tenantID, id uuid.UUID) (string, error) {
	var status string
	err := tx.QueryRow(ctx, `SELECT status FROM report_cards WHERE tenant_id = $1 AND id = $2 FOR UPDATE`, tenantID, id).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", apperr.ErrNotFound
	}
	return status, err
}

const publishedRefusal = "This report card is published. A principal can reopen it if it needs to change."

// UpdateReportCard changes what a teacher wrote on a draft.
func (s *Service) UpdateReportCard(ctx context.Context, tenantID, id uuid.UUID, in CardInput) (*ReportCard, error) {
	if err := checkInput(in); err != nil {
		return nil, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	status, err := lockCard(ctx, tx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if status == StatusFinal {
		return nil, apperr.Conflict(publishedRefusal)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE report_cards SET
			overall_rating = COALESCE($3, overall_rating),
			core_competency_remarks = COALESCE($4, core_competency_remarks),
			teacher_comments = COALESCE($5, teacher_comments)
		WHERE tenant_id = $1 AND id = $2`,
		tenantID, id, in.OverallRating, nilIfEmpty(in.CoreCompetencyRemarks), nilIfEmpty(in.TeacherComments)); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.GetReportCard(ctx, tenantID, id)
}

// PublishReportCard makes a draft final and records who did it.
func (s *Service) PublishReportCard(ctx context.Context, tenantID, id, actor uuid.UUID) (*ReportCard, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	status, err := lockCard(ctx, tx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if status == StatusFinal {
		return nil, apperr.Conflict("This report card is already published.")
	}
	var items int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM report_card_items WHERE tenant_id = $1 AND report_card_id = $2`, tenantID, id).Scan(&items); err != nil {
		return nil, err
	}
	if items == 0 {
		return nil, apperr.Invalid("This report card has nothing on it. Generate it again after recording observations.")
	}
	if _, err := tx.Exec(ctx, `
		UPDATE report_cards SET status = 'final', published_at = now(), published_by = $3
		WHERE tenant_id = $1 AND id = $2`, tenantID, id, nullableID(actor)); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.GetReportCard(ctx, tenantID, id)
}

// ReopenReportCard makes a published card a draft again.
func (s *Service) ReopenReportCard(ctx context.Context, tenantID, id uuid.UUID) (*ReportCard, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	status, err := lockCard(ctx, tx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if status != StatusFinal {
		return nil, apperr.Conflict("This report card is not published, so there is nothing to reopen.")
	}
	if _, err := tx.Exec(ctx, `
		UPDATE report_cards SET status = 'draft', published_at = NULL, published_by = NULL
		WHERE tenant_id = $1 AND id = $2`, tenantID, id); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.GetReportCard(ctx, tenantID, id)
}

// DeleteReportCard throws a draft away. A published card is never deleted.
func (s *Service) DeleteReportCard(ctx context.Context, tenantID, id uuid.UUID) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	status, err := lockCard(ctx, tx, tenantID, id)
	if err != nil {
		return err
	}
	if status == StatusFinal {
		return apperr.Conflict("A published report card cannot be deleted. A principal can reopen it first.")
	}
	if _, err := tx.Exec(ctx, `DELETE FROM report_cards WHERE tenant_id = $1 AND id = $2`, tenantID, id); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// School is what a report card prints about the school.
type School struct {
	Name, Phone, Email, Address, County, Footer string
}

// ReportCardPDF produces the card as a PDF. Nothing is stored: the document is
// made from the card each time it is asked for, so it cannot go stale.
func (s *Service) ReportCardPDF(ctx context.Context, tenantID, id uuid.UUID) ([]byte, string, error) {
	card, err := s.GetReportCard(ctx, tenantID, id)
	if err != nil {
		return nil, "", err
	}
	var school School
	if err := s.pool.QueryRow(ctx, `
		SELECT t.name, COALESCE(t.phone, ''), COALESCE(t.email, ''), COALESCE(t.address, ''), COALESCE(t.county, ''),
			COALESCE(ts.report_card_footer, '')
		FROM tenants t LEFT JOIN tenant_settings ts ON ts.tenant_id = t.id
		WHERE t.id = $1`, tenantID).Scan(&school.Name, &school.Phone, &school.Email, &school.Address, &school.County, &school.Footer); err != nil {
		return nil, "", fmt.Errorf("load school: %w", err)
	}
	pdf, err := RenderReportCardPDF(school, card)
	if err != nil {
		return nil, "", err
	}
	return pdf, ReportCardFileName(card), nil
}
