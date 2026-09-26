package contacts

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"strings"

	"github.com/google/uuid"
)

// BulkRow is one line of an import, parsed from pasted CSV or from JSON rows.
type BulkRow struct {
	// Line is 1-based and matches what the user sees in their file, so an
	// error can be pointed at the exact row that failed.
	Line         int
	FullName     string
	Phone        string
	Email        string
	Relationship string
	GradeStream  string
	Tags         []string
	Notes        string
	IsOptedOut   bool
}

// RowError explains why one row was rejected. A single bad line must never
// fail an import of five hundred good ones.
type RowError struct {
	Line    int    `json:"line"`
	Phone   string `json:"phone,omitempty"`
	Name    string `json:"name,omitempty"`
	Message string `json:"message"`
}

// BulkResult is the report the UI renders after an import.
type BulkResult struct {
	Created  int        `json:"created"`
	Updated  int        `json:"updated"`
	Skipped  int        `json:"skipped"`
	Total    int        `json:"total"`
	Errors   []RowError `json:"errors"`
	Contacts []Contact  `json:"contacts"`
}

const maxBulkRows = 1000

// columnAliases maps the header names a Kenyan school's spreadsheet is likely
// to use onto our fields.
var columnAliases = map[string]string{
	"name": "full_name", "full name": "full_name", "fullname": "full_name",
	"contact": "full_name", "contact name": "full_name",
	"parent": "full_name", "parent name": "full_name",
	"guardian": "full_name", "guardian name": "full_name",
	"student": "full_name", "student name": "full_name",
	"learner": "full_name", "pupil": "full_name",

	"phone": "phone", "phone number": "phone", "phone no": "phone",
	"phone #": "phone", "mobile": "phone", "mobile number": "phone",
	"msisdn": "phone", "cell": "phone", "cellphone": "phone", "cell phone": "phone",
	"contact number": "phone", "telephone": "phone", "tel": "phone",
	"number": "phone", "sms number": "phone",

	"email": "email", "e-mail": "email", "email address": "email",

	"relationship": "relationship", "relation": "relationship", "role": "relationship",
	"type": "relationship", "category": "relationship",

	"class": "grade_stream", "class name": "grade_stream", "grade": "grade_stream",
	"grade/stream": "grade_stream", "grade stream": "grade_stream",
	"grade/class": "grade_stream", "stream": "grade_stream", "group": "grade_stream",

	"tags": "tags", "tag": "tags", "labels": "tags", "label": "tags", "segment": "tags",

	"notes": "notes", "note": "notes", "comment": "notes", "comments": "notes",

	"opted out": "is_opted_out", "opt out": "is_opted_out", "opt-out": "is_opted_out",
	"opted_out": "is_opted_out", "unsubscribed": "is_opted_out",
}

func (r *BulkRow) setField(column, value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	switch column {
	case "full_name":
		r.FullName = value
	case "phone":
		r.Phone = value
	case "email":
		r.Email = value
	case "relationship":
		r.Relationship = value
	case "grade_stream":
		r.GradeStream = value
	case "tags":
		r.Tags = strings.Split(value, ",")
	case "notes":
		r.Notes = value
	case "is_opted_out":
		r.IsOptedOut = parseBool(value)
	}
}

// detectHeader returns mapped column names when rec looks like a header row,
// or nil when the first row is data.
func detectHeader(rec []string) []string {
	mapped := make([]string, len(rec))
	hits := 0
	for i, cell := range rec {
		key := strings.ToLower(strings.TrimSpace(strings.Trim(cell, `"'`)))
		if name, ok := columnAliases[key]; ok {
			mapped[i] = name
			hits++
		}
	}
	// A header must name a phone column, otherwise we would swallow the first
	// real row of a headerless paste.
	if hits == 0 {
		return nil
	}
	for _, name := range mapped {
		if name == "phone" {
			return mapped
		}
	}
	return nil
}

func isBlankRow(rec []string) bool {
	for _, cell := range rec {
		if strings.TrimSpace(cell) != "" {
			return false
		}
	}
	return true
}

func parseBool(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "y", "opt-out", "opted out", "unsubscribed":
		return true
	}
	return false
}

// ParseCSV turns pasted spreadsheet text into rows.
//
// Deliberately forgiving, because the input is a copy-paste from Excel: a
// header row is detected and used for column mapping, comma/tab/semicolon
// delimiters are sniffed, ragged rows are tolerated, and blank lines are
// skipped while every row keeps its ORIGINAL 1-based line number — a school
// fixing a rejected row needs it to point at the right line of their file.
//
// Without a header, the first two columns are read as name then phone: the
// order a school types them in.
func ParseCSV(text string) ([]BulkRow, error) {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	if strings.TrimSpace(text) == "" {
		return nil, fmt.Errorf("nothing to import")
	}

	delimiter := ','
	switch {
	case strings.Contains(text, "\t"):
		delimiter = '\t'
	case !strings.Contains(text, ",") && strings.Contains(text, ";"):
		delimiter = ';'
	}

	// Split on physical lines ourselves instead of letting encoding/csv drop
	// blank lines, which would renumber every row after the first gap.
	lines := strings.Split(text, "\n")
	rows := make([]BulkRow, 0, len(lines))
	var header []string

	for i, line := range lines {
		lineNo := i + 1
		if strings.TrimSpace(line) == "" {
			continue
		}

		rec, err := parseLine(line, delimiter)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNo, err)
		}
		if isBlankRow(rec) {
			continue
		}

		if header == nil {
			if h := detectHeader(rec); h != nil {
				header = h
				continue
			}
		}

		row := BulkRow{Line: lineNo}
		if header != nil {
			for col, name := range header {
				if col < len(rec) {
					row.setField(name, rec[col])
				}
			}
		} else {
			if len(rec) > 0 {
				row.FullName = strings.TrimSpace(rec[0])
			}
			if len(rec) > 1 {
				row.Phone = strings.TrimSpace(rec[1])
			}
			// A single-column paste of bare numbers is a phone list.
			if row.FullName == "" && row.Phone == "" && len(rec) > 0 {
				row.Phone = strings.TrimSpace(rec[0])
			}
		}
		rows = append(rows, row)
	}

	if len(rows) == 0 {
		return nil, fmt.Errorf("no rows found — each line needs a name and a phone number")
	}
	if len(rows) > maxBulkRows {
		return nil, fmt.Errorf("that is %d rows; the limit is %d per import", len(rows), maxBulkRows)
	}
	return rows, nil
}

// parseLine reads a single physical line as a CSV record.
func parseLine(line string, delimiter rune) ([]string, error) {
	reader := csv.NewReader(strings.NewReader(line))
	reader.Comma = delimiter
	reader.FieldsPerRecord = -1
	reader.TrimLeadingSpace = true
	reader.LazyQuotes = true

	rec, err := reader.Read()
	if err != nil {
		if err == io.EOF {
			return nil, nil
		}
		return nil, err
	}
	return rec, nil
}

// validateRow applies the same rules as the single-contact endpoint so both
// paths behave identically.
func validateRow(row BulkRow, defaultRelationship string) (ContactInput, error) {
	name := strings.TrimSpace(row.FullName)
	if name == "" {
		return ContactInput{}, fmt.Errorf("name is missing")
	}
	if len(name) > 255 {
		return ContactInput{}, fmt.Errorf("name is longer than 255 characters")
	}

	phone, err := NormalizePhone(row.Phone)
	if err != nil {
		return ContactInput{}, err
	}
	if err := ValidateEmail(row.Email); err != nil {
		return ContactInput{}, err
	}

	tags, err := NormalizeTags(row.Tags)
	if err != nil {
		return ContactInput{}, err
	}

	rel := strings.TrimSpace(row.Relationship)
	if rel == "" {
		rel = strings.TrimSpace(defaultRelationship)
	}
	if len(rel) > 50 {
		return ContactInput{}, fmt.Errorf("relationship is longer than 50 characters")
	}

	grade := strings.TrimSpace(row.GradeStream)
	if len(grade) > 100 {
		return ContactInput{}, fmt.Errorf("class/grade is longer than 100 characters")
	}

	notes := strings.TrimSpace(row.Notes)
	if len(notes) > 2000 {
		return ContactInput{}, fmt.Errorf("notes are longer than 2000 characters")
	}

	return ContactInput{
		FullName:     name,
		Phone:        phone,
		Email:        strings.TrimSpace(row.Email),
		Relationship: rel,
		GradeStream:  grade,
		Tags:         tags,
		Notes:        notes,
		IsOptedOut:   row.IsOptedOut,
	}, nil
}

// Import inserts or merges a batch of rows.
//
// updateExisting decides what happens to a phone number already in the book:
// merge the row into it, or skip the row and say why. Each row is its own
// operation, so a failure late in the file never discards the good rows above
// it — the whole point of a bulk importer.
func (s *Service) Import(ctx context.Context, tenantID, actorID uuid.UUID, rows []BulkRow, updateExisting bool, defaultRelationship string) (*BulkResult, error) {
	result := &BulkResult{Total: len(rows), Errors: []RowError{}, Contacts: []Contact{}}

	// Phones seen earlier in this same file: a spreadsheet often lists the
	// same parent twice, and keeping both would be a lie.
	seen := make(map[string]int, len(rows))

	for _, row := range rows {
		input, err := validateRow(row, defaultRelationship)
		if err != nil {
			result.Skipped++
			result.Errors = append(result.Errors, RowError{Line: row.Line, Phone: row.Phone, Name: row.FullName, Message: err.Error()})
			continue
		}

		if firstLine, dup := seen[input.Phone]; dup {
			result.Skipped++
			result.Errors = append(result.Errors, RowError{
				Line: row.Line, Phone: row.Phone, Name: row.FullName,
				Message: fmt.Sprintf("duplicate of row %d in this file", firstLine),
			})
			continue
		}
		seen[input.Phone] = row.Line

		existing, err := s.FindByPhone(ctx, tenantID, input.Phone)
		switch {
		case err == nil:
			if !updateExisting {
				result.Skipped++
				result.Errors = append(result.Errors, RowError{
					Line: row.Line, Phone: row.Phone, Name: row.FullName,
					Message: fmt.Sprintf("already saved as %q", existing.FullName),
				})
				continue
			}
			merged, err := s.Merge(ctx, tenantID, existing.ID, input)
			if err != nil {
				result.Skipped++
				result.Errors = append(result.Errors, RowError{Line: row.Line, Phone: row.Phone, Name: row.FullName, Message: err.Error()})
				continue
			}
			if !merged.IsActive {
				// A re-import should not silently ignore a contact the school
				// archived on purpose: bring it back and say so via Updated.
				if err := s.Restore(ctx, tenantID, merged.ID); err != nil {
					result.Skipped++
					result.Errors = append(result.Errors, RowError{Line: row.Line, Phone: row.Phone, Name: row.FullName, Message: err.Error()})
					continue
				}
				merged.IsActive = true
			}
			result.Updated++
			result.Contacts = append(result.Contacts, *merged)

		case err == ErrNotFound:
			created, err := s.CreateWithSource(ctx, tenantID, actorID, input, "import")
			if err != nil {
				result.Skipped++
				result.Errors = append(result.Errors, RowError{Line: row.Line, Phone: row.Phone, Name: row.FullName, Message: err.Error()})
				continue
			}
			result.Created++
			result.Contacts = append(result.Contacts, *created)

		default:
			return nil, err
		}
	}

	return result, nil
}
