package contacts

import (
	"strings"
	"testing"
)

func TestParseCSVWithHeader(t *testing.T) {
	text := "Name,Phone,Email,Relationship,Tags\n" +
		"Jane Doe,0712345678,jane@example.com,parent,grade 4\n" +
		"John Roe,+254722334455,,sponsor,alumni"

	rows, err := ParseCSV(text)
	if err != nil {
		t.Fatalf("ParseCSV: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2", len(rows))
	}

	first := rows[0]
	if first.FullName != "Jane Doe" || first.Phone != "0712345678" {
		t.Errorf("row 1 = %+v, want Jane Doe / 0712345678", first)
	}
	if first.Email != "jane@example.com" || first.Relationship != "parent" {
		t.Errorf("row 1 email/relationship = %q/%q", first.Email, first.Relationship)
	}
	if len(first.Tags) != 1 || first.Tags[0] != "grade 4" {
		t.Errorf("row 1 tags = %v, want [grade 4]", first.Tags)
	}
	// Line numbers must point at the user's file, header included.
	if first.Line != 2 || rows[1].Line != 3 {
		t.Errorf("line numbers = %d, %d; want 2, 3", first.Line, rows[1].Line)
	}
	if rows[1].Phone != "+254722334455" {
		t.Errorf("row 2 phone = %q", rows[1].Phone)
	}
}

func TestParseCSVWithoutHeader(t *testing.T) {
	rows, err := ParseCSV("Jane Doe,0712345678\nJohn Roe,0722334455")
	if err != nil {
		t.Fatalf("ParseCSV: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2", len(rows))
	}
	if rows[0].FullName != "Jane Doe" || rows[0].Phone != "0712345678" {
		t.Errorf("row 1 = %+v", rows[0])
	}
	if rows[0].Line != 1 {
		t.Errorf("line = %d, want 1", rows[0].Line)
	}
}

func TestParseCSVDelimiters(t *testing.T) {
	// Tab separated, as pasted straight out of Excel.
	tabbed := "Jane\t0712345678\nJohn\t0722334455"
	rows, err := ParseCSV(tabbed)
	if err != nil {
		t.Fatalf("tabbed: %v", err)
	}
	if len(rows) != 2 || rows[1].Phone != "0722334455" {
		t.Errorf("tabbed rows = %+v", rows)
	}

	// Semicolon separated (European Excel).
	semi, err := ParseCSV("Jane;0712345678")
	if err != nil {
		t.Fatalf("semicolon: %v", err)
	}
	if len(semi) != 1 || semi[0].Phone != "0712345678" {
		t.Errorf("semicolon rows = %+v", semi)
	}
}

func TestParseCSVRaggedAndBlankRows(t *testing.T) {
	text := "Name,Phone,Email\n" +
		"Jane,0712345678,jane@example.com\n" +
		"\n" +
		"John,0722334455\n" + // missing the optional email column
		"   ,  ,\n" + // whitespace-only row must be dropped

		"Roe,0733445566"

	rows, err := ParseCSV(text)
	if err != nil {
		t.Fatalf("ParseCSV: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("got %d rows, want 3: %+v", len(rows), rows)
	}
	if rows[1].Phone != "0722334455" || rows[1].Email != "" {
		t.Errorf("row 2 = %+v", rows[1])
	}
	// Line numbers stay aligned with the source even when rows are skipped.
	if rows[2].Line != 6 {
		t.Errorf("row 3 line = %d, want 6", rows[2].Line)
	}
}

func TestParseCSVHeaderAliases(t *testing.T) {
	// A Kenyan school's sheet: "Parent Name" / "Phone Number" / "Class".
	rows, err := ParseCSV("Parent Name,Phone Number,Class\nJane,0712345678,Grade 4")
	if err != nil {
		t.Fatalf("ParseCSV: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("got %d rows, want 1", len(rows))
	}
	if rows[0].FullName != "Jane" || rows[0].Phone != "0712345678" || rows[0].GradeStream != "Grade 4" {
		t.Errorf("row = %+v", rows[0])
	}
}

// A first row of data that merely *contains* the word "name" must not be
// mistaken for a header and silently dropped.
func TestParseCSVDataRowNotMistakenForHeader(t *testing.T) {
	rows, err := ParseCSV("Name Of The Shop,0712345678")
	if err != nil {
		t.Fatalf("ParseCSV: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("got %d rows, want 1", len(rows))
	}
	if rows[0].Phone != "0712345678" {
		t.Errorf("row = %+v", rows[0])
	}
}

func TestParseCSVOptedOut(t *testing.T) {
	rows, err := ParseCSV("Name,Phone,Opted Out\nJane,0712345678,yes\nJohn,0722334455,no")
	if err != nil {
		t.Fatalf("ParseCSV: %v", err)
	}
	if !rows[0].IsOptedOut {
		t.Error("row 1 should be opted out")
	}
	if rows[1].IsOptedOut {
		t.Error("row 2 should not be opted out")
	}
}

func TestParseCSVErrors(t *testing.T) {
	for _, in := range []string{"", "   ", "\n\n\n"} {
		if _, err := ParseCSV(in); err == nil {
			t.Errorf("ParseCSV(%q) = nil error, want an error", in)
		}
	}

	// Over the row limit.
	var b strings.Builder
	for i := 0; i < maxBulkRows+5; i++ {
		b.WriteString("Jane Doe,0712345678\n")
	}
	if _, err := ParseCSV(b.String()); err == nil {
		t.Error("ParseCSV accepted more than the row limit, want an error")
	}
}

func TestParseCSVQuotedNames(t *testing.T) {
	rows, err := ParseCSV(`"Doe, Jane",0712345678`)
	if err != nil {
		t.Fatalf("ParseCSV: %v", err)
	}
	if rows[0].FullName != "Doe, Jane" {
		t.Errorf("full name = %q, want %q", rows[0].FullName, "Doe, Jane")
	}
}

func TestPreviewRowsCountersPartition(t *testing.T) {
	// Three rows: one new, one already in the book, one broken. The three
	// counters must sum to the row count — an import dialog that adds up to
	// more rows than were pasted is just wrong.
	rows := []BulkRow{
		{Line: 2, FullName: "Jane", Phone: "0712345678"},
		{Line: 3, FullName: "Peter", Phone: "0712345678"}, // in-file duplicate
		{Line: 4, FullName: "Broken", Phone: "12345"},
	}

	got := previewRows(rows, "parent")
	if got.Total != 3 {
		t.Fatalf("Total = %d, want 3", got.Total)
	}
	if got.Valid != 1 {
		t.Errorf("Valid (new) = %d, want 1", got.Valid)
	}
	if got.Duplicates != 1 {
		t.Errorf("Duplicates = %d, want 1", got.Duplicates)
	}
	if got.Invalid != 1 {
		t.Errorf("Invalid = %d, want 1", got.Invalid)
	}
	if sum := got.Valid + got.Duplicates + got.Invalid; sum != got.Total {
		t.Errorf("counters sum to %d, want %d (they must partition the rows)", sum, got.Total)
	}
}

// validateRow is the gate every imported row passes, so its messages are what
// the school reads when something is wrong.
func TestValidateRow(t *testing.T) {
	ok, err := validateRow(BulkRow{
		Line: 2, FullName: "Jane Doe", Phone: "0712345678", Email: "jane@example.com",
		Relationship: "parent", GradeStream: "Grade 4", Tags: []string{" Grade 4 ", "parents"},
	}, "")
	if err != nil {
		t.Fatalf("validateRow: %v", err)
	}
	if ok.Phone != "+254712345678" {
		t.Errorf("phone = %q, want normalized", ok.Phone)
	}
	if len(ok.Tags) != 2 {
		t.Errorf("tags = %v, want 2", ok.Tags)
	}

	// A blank relationship falls back to the import default.
	withDefault, err := validateRow(BulkRow{Line: 3, FullName: "Jane", Phone: "0712345678"}, "guardian")
	if err != nil {
		t.Fatalf("validateRow with default: %v", err)
	}
	if withDefault.Relationship != "guardian" {
		t.Errorf("relationship = %q, want guardian", withDefault.Relationship)
	}

	cases := map[string]BulkRow{
		"missing name":   {Line: 2, Phone: "0712345678"},
		"bad phone":      {Line: 2, FullName: "Jane", Phone: "12345"},
		"bad email":      {Line: 2, FullName: "Jane", Phone: "0712345678", Email: "nope"},
		"overlong name":  {Line: 2, FullName: strings.Repeat("x", 256), Phone: "0712345678"},
		"overlong rel":   {Line: 2, FullName: "Jane", Phone: "0712345678", Relationship: strings.Repeat("r", 51)},
		"bad tag":        {Line: 2, FullName: "Jane", Phone: "0712345678", Tags: []string{`has"quote`}},
		"overlong notes": {Line: 2, FullName: "Jane", Phone: "0712345678", Notes: strings.Repeat("n", 2001)},
		"overlong class": {Line: 2, FullName: "Jane", Phone: "0712345678", GradeStream: strings.Repeat("g", 101)},
	}
	for name, row := range cases {
		if _, err := validateRow(row, ""); err == nil {
			t.Errorf("%s: validateRow accepted it, want an error", name)
		}
	}
}
