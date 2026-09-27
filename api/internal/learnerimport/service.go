// Package learnerimport stages a learner roster imported from a CSV.
//
// A school's spreadsheet is rarely complete, so nothing here rejects a row for
// being unfinished. Rows land in a staging table with every field nullable, the
// school edits them, and a row becomes a real learner only once it holds a
// learner name, a learner number and a grade -- the fields `learners` itself
// declares NOT NULL.
//
// The sufficiency rule is owned by the database (the is_sufficient and missing
// generated columns, see migration 038). This package reads those columns
// rather than recomputing them, so the rule cannot drift between the API, the
// UI and a future report. MissingFields is the one place this package needs the
// rule in Go, and a test pins it to the SQL.
package learnerimport

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// maxRows caps a single upload. A school roster is a few hundred lines; a
// runaway file is a paste accident, and one giant transaction is worse than a
// second upload.
const maxRows = 5000

// Status values, mirroring the CHECK constraint on learner_import_rows.status.
const (
	StatusDraft    = "draft"
	StatusImported = "imported"
	StatusRejected = "rejected"
)

// ErrNotFound is returned when a batch or row does not exist for this tenant.
// It is a sentinel so the handler can map it to 404 without inspecting text.
var ErrNotFound = errors.New("not found")

// ErrNotSufficient is returned when a row is promoted before it holds the
// fields a learner requires.
var ErrNotSufficient = errors.New("row is not complete enough to import")

// ErrDuplicateNumber is returned when a staged row's learner number already
// belongs to a real learner. learners has UNIQUE (tenant_id, upi), so without
// this check the failure would surface as an opaque database error.
var ErrDuplicateNumber = errors.New("that learner number is already in use")

// ErrAlreadyImported is returned when a row that has become a real learner is
// edited or deleted as though it were still a draft.
var ErrAlreadyImported = errors.New("row has already been imported")

// ErrBatchHasLearners is returned when a batch that has already produced
// learners is deleted wholesale, which would erase the record of what was
// imported while those learners still exist.
var ErrBatchHasLearners = errors.New("batch has already produced learners")

// Batch is one uploaded file.
type Batch struct {
	ID       uuid.UUID `json:"id"`
	TenantID uuid.UUID `json:"tenant_id"`
	// Filename is nullable: a school can paste rows without choosing a file, and
	// the column stores that honestly rather than inventing an empty label.
	Filename *string `json:"filename"`
	// UploadedBy is null when the session had no staff record.
	UploadedBy   *uuid.UUID `json:"uploaded_by"`
	TotalRows    int        `json:"total_rows"`
	ReadyRows    int        `json:"ready_rows"`
	ImportedRows int        `json:"imported_rows"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// Row is one staged line of the file.
//
// ParentName, ParentPhone, StudentName, StudentNumber, Grade and Stream are
// all nullable pointers because "not filled in yet" is a real and common state
// here; a plain string would make an empty CSV cell indistinguishable from a
// field the user deliberately cleared.
type Row struct {
	ID       uuid.UUID `json:"id"`
	TenantID uuid.UUID `json:"tenant_id"`
	BatchID  uuid.UUID `json:"batch_id"`

	// RowNumber is the line in the source file, so an error can point the user
	// at the right row in Excel.
	RowNumber int `json:"row_number"`

	ParentName    *string    `json:"parent_name"`
	ParentPhone   *string    `json:"parent_phone"`
	StudentName   *string    `json:"student_name"`
	StudentNumber *string    `json:"student_number"`
	Grade         *string    `json:"grade"`
	Stream        *string    `json:"stream"`
	Tags          []string   `json:"tags"`
	Notes         *string    `json:"notes"`
	Status        string     `json:"status"`
	Problem       *string    `json:"problem,omitempty"`
	LearnerID     *uuid.UUID `json:"learner_id,omitempty"`
	GuardianID    *uuid.UUID `json:"guardian_id,omitempty"`

	// IsSufficient and Missing come from the database's generated columns.
	IsSufficient bool     `json:"is_sufficient"`
	Missing      []string `json:"missing"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CSVRow is one parsed line before it reaches the database.
type CSVRow struct {
	RowNumber     int
	ParentName    string
	ParentPhone   string
	StudentName   string
	StudentNumber string
	Grade         string
	Stream        string
	Tags          []string
}

// Service owns the staging tables.
type Service struct {
	pool *pgxpool.Pool
}

// NewService builds a Service over a pool.
func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

const rowColumns = `
	id, tenant_id, batch_id, row_number,
	parent_name, parent_phone, student_name, student_number, grade, stream,
	tags, notes, status, problem, learner_id, guardian_id,
	is_sufficient, missing, created_at, updated_at`

// scanRow reads one row in the order rowColumns lists, and normalises the two
// array columns so a client never has to handle a nil slice.
func scanRow(row pgx.Row) (*Row, error) {
	var r Row
	var tags, missing []string
	if err := row.Scan(
		&r.ID, &r.TenantID, &r.BatchID, &r.RowNumber,
		&r.ParentName, &r.ParentPhone, &r.StudentName, &r.StudentNumber,
		&r.Grade, &r.Stream,
		&tags, &r.Notes, &r.Status, &r.Problem, &r.LearnerID, &r.GuardianID,
		&r.IsSufficient, &missing, &r.CreatedAt, &r.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("scan import row: %w", err)
	}
	if tags == nil {
		tags = []string{}
	}
	if missing == nil {
		missing = []string{}
	}
	r.Tags, r.Missing = tags, missing
	return &r, nil
}

// nullIfEmpty maps a blank CSV cell to SQL NULL, so "left empty" is stored as
// "not filled in" rather than as an empty string that looks like a value.
func nullIfEmpty(s string) *string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return &s
}

// MissingFields returns the required fields a staged row is still missing.
//
// This duplicates the generated `missing` column in migration 038 by
// necessity: the API needs the rule in Go to explain a rejection before it
// reaches the database, and to validate a row the caller is trying to promote.
// TestMissingFieldsAgreesWithTheDatabaseRule pins the two together.
func MissingFields(parentName, studentName, studentNumber, grade string) []string {
	var missing []string
	if strings.TrimSpace(studentName) == "" {
		missing = append(missing, "student_name")
	}
	if strings.TrimSpace(studentNumber) == "" {
		missing = append(missing, "student_number")
	}
	if strings.TrimSpace(grade) == "" {
		missing = append(missing, "grade")
	}
	return missing
}

// ParseCSV turns pasted or uploaded CSV text into rows.
//
// The header is matched loosely: a school's spreadsheet says "Parent Name",
// "parent_name" or "Parent", and insisting on one spelling would make the
// import fail on a cosmetic difference. Only the learner name, learner number
// and grade columns are matched by name; anything unrecognised is ignored
// rather than rejected, because an extra column ("House", "Remarks") is not a
// reason to lose the row.
//
// Nothing is validated here. A row with two empty cells parses fine and is
// stored as-is: judging sufficiency is the staging table's job.
func ParseCSV(text string) ([]CSVRow, error) {
	r := csv.NewReader(strings.NewReader(text))
	// Real files contain ragged rows and stray quotes; both should cost the
	// offending field, not the whole upload.
	r.FieldsPerRecord = -1
	r.TrimLeadingSpace = true
	r.LazyQuotes = true

	records, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("could not read the CSV: %w", err)
	}
	// Drop trailing blank lines, which spreadsheets add on save.
	for len(records) > 0 && isBlankRecord(records[len(records)-1]) {
		records = records[:len(records)-1]
	}
	if len(records) == 0 {
		return nil, errors.New("the file is empty")
	}
	if len(records) > maxRows+1 {
		return nil, fmt.Errorf("that is %d rows; the limit is %d per import", len(records)-1, maxRows)
	}

	header := records[0]
	col := map[string]int{}
	for i, h := range header {
		col[normaliseHeader(h)] = i
	}
	// A file with no recognisable learner-name column is almost certainly not a
	// roster, and guessing would create a batch of empty rows.
	if _, ok := col["studentname"]; !ok {
		return nil, errors.New("no 'student name' column found; expected headings like " +
			"Parent Name, Student Name, Student Number, Grade")
	}

	rows := make([]CSVRow, 0, len(records)-1)
	for i, rec := range records[1:] {
		if isBlankRecord(rec) {
			continue
		}
		rows = append(rows, CSVRow{
			// +2 so the number matches the line in Excel: one for the header,
			// one because humans count from one.
			RowNumber:     i + 2,
			ParentName:    cell(rec, col, "parentname", "parent"),
			ParentPhone:   cell(rec, col, "parentphone", "phone", "phonenumber", "contact"),
			StudentName:   cell(rec, col, "studentname", "learner", "learnername", "name"),
			StudentNumber: cell(rec, col, "studentnumber", "number", "admissionnumber", "upi", "regno"),
			Grade:         cell(rec, col, "grade", "class", "standard"),
			Stream:        cell(rec, col, "stream"),
			Tags:          splitTags(cell(rec, col, "tags", "tag", "labels")),
		})
	}
	if len(rows) == 0 {
		return nil, errors.New("the file has a header but no rows under it")
	}
	return rows, nil
}

// normaliseHeader reduces a heading to letters and digits so "Student Name",
// "student_name" and "STUDENT  NAME" all collapse to "studentname".
func normaliseHeader(h string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(h)) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// cell returns the first matching column's value, or "" if the column is
// absent or the row is short. A short row is normal in a hand-edited sheet.
func cell(rec []string, col map[string]int, names ...string) string {
	for _, n := range names {
		if i, ok := col[n]; ok && i < len(rec) {
			return strings.TrimSpace(rec[i])
		}
	}
	return ""
}

// splitTags accepts the two ways a spreadsheet carries a tag list.
func splitTags(v string) []string {
	if v == "" {
		return []string{}
	}
	parts := strings.FieldsFunc(v, func(r rune) bool { return r == ';' || r == ',' || r == '|' })
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func isBlankRecord(rec []string) bool {
	for _, f := range rec {
		if strings.TrimSpace(f) != "" {
			return false
		}
	}
	return true
}

// CreateBatch stores a whole upload as one batch and its rows.
//
// Every row is inserted regardless of how empty it is. That is the contract:
// the school gets its 298 good rows now and finishes the rest in the editor,
// rather than being told to go fix the spreadsheet first.
//
// The batch and its rows go in one transaction so a half-written upload never
// appears in the staging list.
func (s *Service) CreateBatch(ctx context.Context, tenantID, actorID uuid.UUID, filename string, rows []CSVRow) (*Batch, error) {
	if len(rows) == 0 {
		return nil, errors.New("there are no rows to import")
	}
	if len(rows) > maxRows {
		return nil, fmt.Errorf("that is %d rows; the limit is %d per import", len(rows), maxRows)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin import: %w", err)
	}
	defer tx.Rollback(ctx)

	var actor *uuid.UUID
	if actorID != uuid.Nil {
		actor = &actorID
	}

	var b Batch
	err = tx.QueryRow(ctx, `
		INSERT INTO learner_import_batches (tenant_id, filename, uploaded_by)
		VALUES ($1, $2, $3)
		RETURNING id, tenant_id, filename, uploaded_by,
		          total_rows, ready_rows, imported_rows, created_at, updated_at
	`, tenantID, nullIfEmpty(filename), actor).Scan(
		&b.ID, &b.TenantID, &b.Filename, &b.UploadedBy,
		&b.TotalRows, &b.ReadyRows, &b.ImportedRows, &b.CreatedAt, &b.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create import batch: %w", err)
	}

	// One statement for the whole file rather than one round trip per row: a
	// 500-line roster is a single unnest instead of 500 inserts.
	//
	// Tags travel as one chr(31)-joined string per row rather than a text[][]:
	// unnest flattens a multidimensional array to its elements, so a nested
	// array silently arrives as text and the insert fails on the column type.
	// chr(31) (the ASCII unit separator) cannot appear in a tag a person would
	// type, so joining on it is unambiguous.
	_, err = tx.Exec(ctx, `
		INSERT INTO learner_import_rows
			(tenant_id, batch_id, row_number, parent_name, parent_phone,
			 student_name, student_number, grade, stream, tags)
		SELECT $1, $2, t.row_number, t.parent_name, t.parent_phone,
		       t.student_name, t.student_number, t.grade, t.stream,
		       COALESCE(NULLIF(string_to_array(t.tags_joined, chr(31)), ARRAY['']), '{}')
		FROM unnest($3::int[], $4::text[], $5::text[], $6::text[],
		            $7::text[], $8::text[], $9::text[], $10::text[])
			AS t(row_number, parent_name, parent_phone, student_name,
			     student_number, grade, stream, tags_joined)
	`, tenantID, b.ID,
		ints(rows, func(r CSVRow) int { return r.RowNumber }),
		texts(rows, func(r CSVRow) string { return r.ParentName }),
		texts(rows, func(r CSVRow) string { return r.ParentPhone }),
		texts(rows, func(r CSVRow) string { return r.StudentName }),
		texts(rows, func(r CSVRow) string { return r.StudentNumber }),
		texts(rows, func(r CSVRow) string { return r.Grade }),
		texts(rows, func(r CSVRow) string { return r.Stream }),
		joinedTags(rows),
	)
	if err != nil {
		return nil, fmt.Errorf("insert import rows: %w", err)
	}

	if err := refreshBatchCounts(ctx, tx, b.ID); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit import: %w", err)
	}
	return s.GetBatch(ctx, tenantID, b.ID)
}

// refreshBatchCounts recomputes a batch's counters from its rows. They are
// cached on the batch for a cheap list view, and every write path must refresh
// them or the "12 ready" figure silently drifts.
func refreshBatchCounts(ctx context.Context, tx pgx.Tx, batchID uuid.UUID) error {
	_, err := tx.Exec(ctx, `
		UPDATE learner_import_batches b
		SET total_rows = s.total,
		    ready_rows = s.ready,
		    imported_rows = s.imported,
		    updated_at = now()
		FROM (
			SELECT
				count(*) AS total,
				count(*) FILTER (WHERE is_sufficient AND status = 'draft') AS ready,
				count(*) FILTER (WHERE status = 'imported') AS imported
			FROM learner_import_rows WHERE batch_id = $1
		) s
		WHERE b.id = $1
	`, batchID)
	if err != nil {
		return fmt.Errorf("refresh import counts: %w", err)
	}
	return nil
}

// GetBatch reads one batch, scoped to the tenant.
func (s *Service) GetBatch(ctx context.Context, tenantID, id uuid.UUID) (*Batch, error) {
	var b Batch
	err := s.pool.QueryRow(ctx, `
		SELECT id, tenant_id, filename, uploaded_by,
		       total_rows, ready_rows, imported_rows, created_at, updated_at
		FROM learner_import_batches
		WHERE tenant_id = $1 AND id = $2
	`, tenantID, id).Scan(
		&b.ID, &b.TenantID, &b.Filename, &b.UploadedBy,
		&b.TotalRows, &b.ReadyRows, &b.ImportedRows, &b.CreatedAt, &b.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get import batch: %w", err)
	}
	return &b, nil
}

// ListBatches returns the school's uploads, newest first.
func (s *Service) ListBatches(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]Batch, int, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	var total int
	if err := s.pool.QueryRow(ctx,
		`SELECT count(*) FROM learner_import_batches WHERE tenant_id = $1`,
		tenantID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count import batches: %w", err)
	}

	rows, err := s.pool.Query(ctx, `
		SELECT id, tenant_id, filename, uploaded_by,
		       total_rows, ready_rows, imported_rows, created_at, updated_at
		FROM learner_import_batches
		WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, tenantID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list import batches: %w", err)
	}
	defer rows.Close()

	out := []Batch{}
	for rows.Next() {
		var b Batch
		if err := rows.Scan(
			&b.ID, &b.TenantID, &b.Filename, &b.UploadedBy,
			&b.TotalRows, &b.ReadyRows, &b.ImportedRows, &b.CreatedAt, &b.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan import batch: %w", err)
		}
		out = append(out, b)
	}
	return out, total, rows.Err()
}

// ListRows returns a batch's rows, optionally only the ones matching a status
// or tag, in file order so the list reads like the spreadsheet it came from.
func (s *Service) ListRows(ctx context.Context, tenantID, batchID uuid.UUID, status, tag string) ([]Row, int, error) {
	var total int
	if err := s.pool.QueryRow(ctx, `
		SELECT count(*) FROM learner_import_rows
		WHERE tenant_id = $1 AND batch_id = $2
		  AND ($3 = '' OR status = $3)
		  AND ($4 = '' OR tags @> ARRAY[$4]::text[])
	`, tenantID, batchID, status, tag).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count import rows: %w", err)
	}

	rows, err := s.pool.Query(ctx, `
		SELECT `+rowColumns+`
		FROM learner_import_rows
		WHERE tenant_id = $1 AND batch_id = $2
		  AND ($3 = '' OR status = $3)
		  AND ($4 = '' OR tags @> ARRAY[$4]::text[])
		ORDER BY row_number
	`, tenantID, batchID, status, tag)
	if err != nil {
		return nil, 0, fmt.Errorf("list import rows: %w", err)
	}
	defer rows.Close()

	out := []Row{}
	for rows.Next() {
		r, err := scanRow(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *r)
	}
	return out, total, rows.Err()
}

// GetRow reads one staged row, scoped to the tenant.
func (s *Service) GetRow(ctx context.Context, tenantID, id uuid.UUID) (*Row, error) {
	return scanRow(s.pool.QueryRow(ctx,
		`SELECT `+rowColumns+` FROM learner_import_rows WHERE tenant_id = $1 AND id = $2`,
		tenantID, id))
}

// RowPatch is a partial edit of a staged row. Every field is a pointer so that
// "leave this alone" and "clear this field" stay distinguishable -- the usual
// reason an in-place editor silently refuses to let someone empty a column.
type RowPatch struct {
	ParentName    *string   `json:"parent_name"`
	ParentPhone   *string   `json:"parent_phone"`
	StudentName   *string   `json:"student_name"`
	StudentNumber *string   `json:"student_number"`
	Grade         *string   `json:"grade"`
	Stream        *string   `json:"stream"`
	Tags          *[]string `json:"tags"`
	Notes         *string   `json:"notes"`
	Status        *string   `json:"status"`
}

// UpdateRow applies a partial edit and returns the saved row.
//
// is_sufficient and missing are recomputed by the database on write, so
// filling in the missing grade is all it takes for a row to become promotable:
// the caller never tells the server "this one is ready now".
func (s *Service) UpdateRow(ctx context.Context, tenantID, id uuid.UUID, p RowPatch) (*Row, error) {
	if p.Status != nil {
		switch *p.Status {
		case StatusDraft, StatusImported, StatusRejected:
		default:
			return nil, fmt.Errorf("status must be one of %s, %s or %s",
				StatusDraft, StatusImported, StatusRejected)
		}
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin row update: %w", err)
	}
	defer tx.Rollback(ctx)

	var batchID uuid.UUID
	if err := tx.QueryRow(ctx,
		`SELECT batch_id FROM learner_import_rows WHERE tenant_id = $1 AND id = $2`,
		tenantID, id).Scan(&batchID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("load row: %w", err)
	}

	// COALESCE keeps an omitted field as it was. A field sent as "" is stored
	// as an empty string, which the generated `missing` column still treats as
	// absent, so clearing a cell and leaving it alone both behave correctly.
	_, err = tx.Exec(ctx, `
		UPDATE learner_import_rows SET
			parent_name    = COALESCE($3, parent_name),
			parent_phone   = COALESCE($4, parent_phone),
			student_name   = COALESCE($5, student_name),
			student_number = COALESCE($6, student_number),
			grade          = COALESCE($7, grade),
			stream         = COALESCE($8, stream),
			tags           = COALESCE($9, tags),
			notes          = COALESCE($10, notes),
			status         = COALESCE($11, status),
			-- A hand-edited row has cleared whatever was blocking it.
			problem        = NULL,
			updated_at     = now()
		WHERE tenant_id = $1 AND id = $2
	`, tenantID, id, p.ParentName, p.ParentPhone, p.StudentName, p.StudentNumber,
		p.Grade, p.Stream, p.Tags, p.Notes, p.Status)
	if err != nil {
		return nil, fmt.Errorf("update import row: %w", err)
	}

	if err := refreshBatchCounts(ctx, tx, batchID); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit row update: %w", err)
	}
	return s.GetRow(ctx, tenantID, id)
}

// DeleteRow removes one staged row. It is refused once the row has been
// promoted, because by then that row is the record of a real learner.
func (s *Service) DeleteRow(ctx context.Context, tenantID, id uuid.UUID) error {
	var batchID uuid.UUID
	var status string
	err := s.pool.QueryRow(ctx,
		`SELECT batch_id, status FROM learner_import_rows WHERE tenant_id = $1 AND id = $2`,
		tenantID, id).Scan(&batchID, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("load row: %w", err)
	}
	if status == StatusImported {
		return fmt.Errorf("%w; delete the learner instead", ErrAlreadyImported)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin row delete: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx,
		`DELETE FROM learner_import_rows WHERE tenant_id = $1 AND id = $2`, tenantID, id); err != nil {
		return fmt.Errorf("delete import row: %w", err)
	}
	if err := refreshBatchCounts(ctx, tx, batchID); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit row delete: %w", err)
	}
	return nil
}

// PromoteResult reports what one promotion did.
type PromoteResult struct {
	Row       Row       `json:"row"`
	LearnerID uuid.UUID `json:"learner_id"`
	// GuardianID is set only when the row carried both a parent name and a
	// parent phone. guardians.phone_primary is NOT NULL, so a parent with no
	// number cannot become a guardian record -- the learner is still imported
	// and the omission is reported rather than silently dropping the parent.
	GuardianID *uuid.UUID `json:"guardian_id,omitempty"`
	Note       string     `json:"note,omitempty"`
}

// Promote turns a sufficient staged row into a real learner (and, when it has
// enough for one, a guardian), then marks the row imported.
//
// It refuses a row that is not sufficient, listing what is missing, so the
// caller gets a useful answer instead of a generic failure. A learner number
// that already exists in this school is reported rather than creating a second
// learner with the same number -- the UNIQUE constraint on (tenant_id, upi)
// would otherwise surface as an opaque database error.
func (s *Service) Promote(ctx context.Context, tenantID, rowID uuid.UUID) (*PromoteResult, error) {
	row, err := s.GetRow(ctx, tenantID, rowID)
	if err != nil {
		return nil, err
	}
	if row.Status == StatusImported {
		return nil, fmt.Errorf("%w; it cannot be promoted again", ErrAlreadyImported)
	}
	if !row.IsSufficient {
		return nil, fmt.Errorf("%w: still needs %s",
			ErrNotSufficient, strings.Join(row.Missing, ", "))
	}
	studentName := strings.TrimSpace(*row.StudentName)
	studentNumber := strings.TrimSpace(*row.StudentNumber)
	grade := strings.TrimSpace(*row.Grade)

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin promote: %w", err)
	}
	defer tx.Rollback(ctx)

	// Guard against importing the same number twice from two different rows.
	var existing string
	err = tx.QueryRow(ctx,
		`SELECT full_name FROM learners WHERE tenant_id = $1 AND upi = $2`,
		tenantID, studentNumber).Scan(&existing)
	if err == nil {
		return nil, fmt.Errorf("%w: number %s already belongs to %s",
			ErrDuplicateNumber, studentNumber, existing)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("check learner number: %w", err)
	}

	var learnerID uuid.UUID
	if err := tx.QueryRow(ctx, `
		INSERT INTO learners (tenant_id, upi, full_name, grade, stream)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`, tenantID, studentNumber, studentName, grade, row.Stream).Scan(&learnerID); err != nil {
		return nil, fmt.Errorf("create learner from import: %w", err)
	}

	result := &PromoteResult{LearnerID: learnerID}

	// A guardian needs a name AND a phone: phone_primary is NOT NULL. A parent
	// with no number is left out of the guardian table and the omission is
	// surfaced, rather than blocking an otherwise complete learner.
	parentName := trimPtr(row.ParentName)
	parentPhone := trimPtr(row.ParentPhone)
	if parentName != "" && parentPhone != "" {
		var guardianID uuid.UUID
		if err := tx.QueryRow(ctx, `
			INSERT INTO guardians (tenant_id, full_name, phone_primary)
			VALUES ($1, $2, $3)
			RETURNING id
		`, tenantID, parentName, parentPhone).Scan(&guardianID); err != nil {
			return nil, fmt.Errorf("create guardian from import: %w", err)
		}
		// Link the guardian to the learner it belongs to.
		if _, err := tx.Exec(ctx,
			`UPDATE learners SET guardian_ids = array_append(guardian_ids, $1) WHERE id = $2`,
			guardianID, learnerID); err != nil {
			return nil, fmt.Errorf("link guardian: %w", err)
		}
		result.GuardianID = &guardianID
	} else if parentName != "" {
		result.Note = "the parent was skipped because no phone number was given"
	}

	if _, err := tx.Exec(ctx, `
		UPDATE learner_import_rows
		SET status = 'imported', learner_id = $2, guardian_id = $3,
		    problem = NULL, updated_at = now()
		WHERE tenant_id = $1 AND id = $4
	`, tenantID, learnerID, result.GuardianID, rowID); err != nil {
		return nil, fmt.Errorf("mark row imported: %w", err)
	}

	if err := refreshBatchCounts(ctx, tx, row.BatchID); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit promote: %w", err)
	}

	saved, err := s.GetRow(ctx, tenantID, rowID)
	if err != nil {
		return nil, err
	}
	result.Row = *saved
	return result, nil
}

// trimPtr returns the trimmed value of a nullable field, or "" when unset.
func trimPtr(s *string) string {
	if s == nil {
		return ""
	}
	return strings.TrimSpace(*s)
}

// ints projects one field out of a row slice for a bulk unnest.
func ints(rows []CSVRow, f func(CSVRow) int) []int {
	out := make([]int, len(rows))
	for i, r := range rows {
		out[i] = f(r)
	}
	return out
}

// texts projects one field out of a row slice for a bulk unnest. Empty stays
// empty here; the column accepts it, and the generated `missing` column treats
// a blank exactly like NULL when deciding sufficiency.
func texts(rows []CSVRow, f func(CSVRow) string) []string {
	out := make([]string, len(rows))
	for i, r := range rows {
		out[i] = f(r)
	}
	return out
}

// tagSeparator joins a row's tags into one string inside the bulk INSERT.
//
// chr(31) is the ASCII "unit separator": it is not something a person types
// into a spreadsheet cell, so unlike a comma or semicolon it cannot appear
// inside a tag and split it in two.
const tagSeparator = "\x1f"

// joinedTags projects the tag lists as one joined string per row. A row with no
// tags becomes "", which the SQL turns back into an empty array rather than an
// array holding one empty string.
func joinedTags(rows []CSVRow) []string {
	out := make([]string, len(rows))
	for i, r := range rows {
		out[i] = strings.Join(r.Tags, tagSeparator)
	}
	return out
}

// DeleteBatch discards an upload and every row in it.
//
// Promoted rows are protected: a batch that has already produced real learners
// cannot be deleted wholesale, because that would erase the record of what was
// imported while those learners still exist. The user is told to delete the
// learners first, or to delete the remaining rows individually.
func (s *Service) DeleteBatch(ctx context.Context, tenantID, batchID uuid.UUID) error {
	var imported int
	err := s.pool.QueryRow(ctx, `
		SELECT count(*) FROM learner_import_rows
		WHERE tenant_id = $1 AND batch_id = $2 AND status = 'imported'
	`, tenantID, batchID).Scan(&imported)
	if err != nil {
		return fmt.Errorf("count imported rows: %w", err)
	}
	if imported > 0 {
		return fmt.Errorf("%w: this file produced %d learner(s); "+
			"delete the remaining rows instead", ErrBatchHasLearners, imported)
	}

	// Rows cascade from the batch, so one delete is enough.
	tag, err := s.pool.Exec(ctx,
		`DELETE FROM learner_import_batches WHERE tenant_id = $1 AND id = $2`,
		tenantID, batchID)
	if err != nil {
		return fmt.Errorf("delete import batch: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
