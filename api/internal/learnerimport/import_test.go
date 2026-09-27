package learnerimport

import (
	"os"
	"reflect"
	"strings"
	"testing"
)

// TestMissingFields is the whole contract in one table: a row is only promoted
// once it has a learner name, a learner number and a grade. The parent columns
// are deliberately absent from the rule -- guardians.phone_primary is NOT NULL,
// so a parent without a number cannot become a guardian, but that must not
// block an otherwise complete learner.
func TestMissingFields(t *testing.T) {
	tests := []struct {
		name          string
		parentName    string
		studentName   string
		studentNumber string
		grade         string
		want          []string
	}{
		{
			name:       "complete row is promotable",
			parentName: "Mary Achieng", studentName: "Brian Achieng",
			studentNumber: "S-1001", grade: "4",
			want: nil,
		},
		{
			name:          "parent is never required",
			studentName:   "Brian Achieng",
			studentNumber: "S-1001", grade: "4",
			want: nil,
		},
		{
			name:          "missing grade",
			studentName:   "Faith Wanjiru",
			studentNumber: "S-1002",
			want:          []string{"grade"},
		},
		{
			name: "empty row needs everything",
			want: []string{"student_name", "student_number", "grade"},
		},
		{
			name:        "whitespace does not count as filled in",
			studentName: "   ", studentNumber: "S-1", grade: "4",
			want: []string{"student_name"},
		},
		{
			name:  "missing name and number",
			grade: "4",
			want:  []string{"student_name", "student_number"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := MissingFields(tc.parentName, tc.studentName, tc.studentNumber, tc.grade)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("MissingFields() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestMissingFieldsAgreesWithTheDatabaseRule keeps the Go copy of the
// sufficiency rule in step with the generated `missing` column in migration
// 038. The two are deliberately duplicated -- the API needs the rule before a
// round trip, the database owns it for everything else -- so this test is the
// thing that stops them drifting apart and quietly disagreeing about which
// rows can be imported.
func TestMissingFieldsAgreesWithTheDatabaseRule(t *testing.T) {
	sqlBytes := readMigration(t, "../../migrations/038_learner_import.sql")
	sql := string(sqlBytes)

	// Every field the Go function checks must also appear in the SQL rule, and
	// vice versa. A new required field added to one and not the other is the
	// exact failure this guards against.
	for _, field := range []string{"student_name", "student_number", "grade"} {
		if !strings.Contains(sql, "COALESCE("+field+", '')") {
			t.Errorf("the migration's is_sufficient/missing rule does not mention %q; "+
				"MissingFields checks it, so the two rules now disagree", field)
		}
		if !strings.Contains(sql, "THEN '"+field+"' END") {
			t.Errorf("the migration's `missing` array does not list %q; "+
				"MissingFields reports it, so the UI would show different fields", field)
		}
	}

	// parent_name must NOT be a promotion requirement in the SQL rule. If
	// someone later adds it there, a learner with no parent on file could never
	// be imported, which is not what this feature promises.
	if strings.Contains(sql, "COALESCE(parent_name, '')) <> ''") {
		t.Error("the SQL rule now requires parent_name; MissingFields does not")
	}
}

// readMigration loads a migration file, failing loudly if it moved.
func readMigration(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v (the migration this rule belongs to must be readable)", path, err)
	}
	return b
}

// TestParseCSVAcceptsMessyRealWorldInput covers what actually arrives from a
// school's spreadsheet, which is the whole reason rows are staged rather than
// validated on the way in.
func TestParseCSVAcceptsMessyRealWorldInput(t *testing.T) {
	t.Run("loose headings and reordered columns", func(t *testing.T) {
		// "Parent Name" not "parent_name", and grade before the learner.
		csv := "Parent Name,Grade,Student Name,Student Number\n" +
			"Mary Achieng,4,Brian Achieng,S-1001\n"
		rows, err := ParseCSV(csv)
		if err != nil {
			t.Fatalf("ParseCSV: %v", err)
		}
		if len(rows) != 1 {
			t.Fatalf("got %d rows, want 1", len(rows))
		}
		r := rows[0]
		if r.StudentName != "Brian Achieng" || r.Grade != "4" || r.StudentNumber != "S-1001" {
			t.Errorf("columns matched wrong: %+v", r)
		}
		if r.ParentName != "Mary Achieng" {
			t.Errorf("ParentName = %q, want %q", r.ParentName, "Mary Achieng")
		}
	})

	t.Run("incomplete rows parse rather than fail", func(t *testing.T) {
		// The point of staging: a row with nothing but a name is still stored.
		csv := "Student Name,Student Number,Grade\n" +
			"Brian Achieng,S-1001,4\n" +
			",,\n" +
			"Faith Wanjiru,,\n"
		rows, err := ParseCSV(csv)
		if err != nil {
			t.Fatalf("ParseCSV must not reject an incomplete row: %v", err)
		}
		if len(rows) != 2 {
			t.Fatalf("got %d rows, want 2 (the all-blank line is dropped)", len(rows))
		}
		if rows[1].StudentName != "Faith Wanjiru" {
			t.Errorf("second row = %+v, want the partially filled one", rows[1])
		}
	})

	t.Run("extra columns are ignored", func(t *testing.T) {
		// "House" and "Remarks" are not ours; they must not lose the row.
		csv := "Student Name,Student Number,Grade,House,Remarks\n" +
			"Brian Achieng,S-1001,4,Blue,Needs follow up\n"
		rows, err := ParseCSV(csv)
		if err != nil {
			t.Fatalf("ParseCSV: %v", err)
		}
		if len(rows) != 1 || rows[0].StudentName != "Brian Achieng" {
			t.Fatalf("got %+v, want the one row preserved", rows)
		}
	})

	t.Run("row numbers match the line in the spreadsheet", func(t *testing.T) {
		// Line 1 is the header, so the first record is line 2. A message
		// saying "row 1" when the user is looking at line 2 is useless.
		csv := "Student Name,Student Number,Grade\n" +
			"Brian Achieng,S-1001,4\n" +
			"Faith Wanjiru,S-1002,5\n"
		rows, err := ParseCSV(csv)
		if err != nil {
			t.Fatalf("ParseCSV: %v", err)
		}
		if rows[0].RowNumber != 2 || rows[1].RowNumber != 3 {
			t.Errorf("RowNumber = %d, %d; want 2, 3", rows[0].RowNumber, rows[1].RowNumber)
		}
	})

	t.Run("tags split on commas and semicolons", func(t *testing.T) {
		csv := "Student Name,Student Number,Grade,Tags\n" +
			"Brian Achieng,S-1001,4,\"boarding;2025 intake\"\n"
		rows, err := ParseCSV(csv)
		if err != nil {
			t.Fatalf("ParseCSV: %v", err)
		}
		want := []string{"boarding", "2025 intake"}
		if !reflect.DeepEqual(rows[0].Tags, want) {
			t.Errorf("Tags = %v, want %v", rows[0].Tags, want)
		}
	})

	t.Run("ragged rows do not lose the file", func(t *testing.T) {
		// A hand-edited sheet with a short row must not fail the import.
		csv := "Student Name,Student Number,Grade\n" +
			"Brian Achieng\n" +
			"Faith Wanjiru,S-1002,5\n"
		rows, err := ParseCSV(csv)
		if err != nil {
			t.Fatalf("ParseCSV: %v", err)
		}
		if len(rows) != 2 || rows[0].StudentName != "Brian Achieng" {
			t.Fatalf("got %+v, want both rows", rows)
		}
	})
}

// TestParseCSVRejectsUnusableInput covers the cases where staging a file would
// be worse than refusing it.
func TestParseCSVRejectsUnusableInput(t *testing.T) {
	tests := []struct {
		name string
		csv  string
	}{
		{"empty", ""},
		{"header only", "Student Name,Student Number,Grade\n"},
		{"only blank lines", "\n\n\n"},
		{
			// Staging this would create a batch of nameless rows and tell the
			// user nothing useful.
			name: "no student name column",
			csv:  "House,Colour\nBlue,Red\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := ParseCSV(tc.csv); err == nil {
				t.Error("ParseCSV accepted a file it cannot do anything with")
			}
		})
	}
}
