package attendance

import (
	"encoding/json"
	"github.com/google/uuid"
	"testing"
	"time"
)

func TestDateAcceptsBothWireFormats(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want time.Time
	}{
		{"date only", `"2026-09-26"`, time.Date(2026, 9, 26, 0, 0, 0, 0, time.UTC)},
		{"rfc3339", `"2026-09-26T00:00:00Z"`, time.Date(2026, 9, 26, 0, 0, 0, 0, time.UTC)},
		{"rfc3339 with offset", `"2026-09-26T00:00:00+03:00"`, time.Date(2026, 9, 26, 0, 0, 0, 0, time.FixedZone("", 3*3600))},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var d Date
			if err := json.Unmarshal([]byte(tc.in), &d); err != nil {
				t.Fatalf("Unmarshal(%s): %v", tc.in, err)
			}
			if !d.Time.Equal(tc.want) {
				t.Errorf("got %s, want %s", d.Time, tc.want)
			}
		})
	}
}

func TestDateRejectsRubbish(t *testing.T) {
	for _, in := range []string{`"26-09-2026"`, `"not a date"`, `"2026-13-45"`, `""`} {
		var d Date
		if err := json.Unmarshal([]byte(in), &d); err == nil {
			t.Errorf("Unmarshal(%s) should have failed, got %s", in, d.Time)
		}
	}
}

func TestDateMarshalsDateOnly(t *testing.T) {
	d := Date{Time: time.Date(2026, 9, 26, 14, 30, 0, 0, time.UTC)}
	b, err := json.Marshal(d)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if string(b) != `"2026-09-26"` {
		t.Errorf("got %s, want \"2026-09-26\"", b)
	}
}

// The whole reason Date exists: a register posted with a plain calendar date
// must decode, and the learner id must survive the round trip.
func TestCreateAttendanceRequestDecodesSnakeCase(t *testing.T) {
	body := `{
		"learner_id": "11111111-1111-1111-1111-111111111111",
		"date": "2026-09-26",
		"status": "absent",
		"reason": "Sick",
		"marked_by": "22222222-2222-2222-2222-222222222222"
	}`

	var req CreateAttendanceRequest
	if err := json.Unmarshal([]byte(body), &req); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if req.LearnerID.String() != "11111111-1111-1111-1111-111111111111" {
		t.Errorf("learner_id did not bind: %v", req.LearnerID)
	}
	if req.Status != AttendanceAbsent {
		t.Errorf("status = %q, want absent", req.Status)
	}
	if req.Reason != "Sick" {
		t.Errorf("reason = %q", req.Reason)
	}
	// Who marked the register is the session's, never the request's.
	if req.MarkedBy != uuid.Nil {
		t.Errorf("marked_by was read from the request: %v", req.MarkedBy)
	}
	if req.Date.IsZero() {
		t.Error("date did not bind")
	}
}

func TestTermDateRange(t *testing.T) {
	tests := []struct {
		term      int
		wantStart string
		wantEnd   string
	}{
		{1, "2026-01-01", "2026-04-30"},
		{2, "2026-05-01", "2026-08-31"},
		{3, "2026-09-01", "2026-12-31"},
	}
	for _, tc := range tests {
		start, end, err := TermDateRange(tc.term, 2026)
		if err != nil {
			t.Fatalf("TermDateRange(%d): %v", tc.term, err)
		}
		if got := start.Format(dateOnlyLayout); got != tc.wantStart {
			t.Errorf("term %d start = %s, want %s", tc.term, got, tc.wantStart)
		}
		if got := end.Format(dateOnlyLayout); got != tc.wantEnd {
			t.Errorf("term %d end = %s, want %s", tc.term, got, tc.wantEnd)
		}
		if !start.Before(end) {
			t.Errorf("term %d: start %s is not before end %s", tc.term, start, end)
		}
	}
}

func TestTermDateRangeRejectsBadTerm(t *testing.T) {
	for _, term := range []int{0, 4, -1} {
		if _, _, err := TermDateRange(term, 2026); err == nil {
			t.Errorf("TermDateRange(%d) should fail", term)
		}
	}
}

func TestValidStatuses(t *testing.T) {
	for _, s := range []AttendanceStatus{AttendancePresent, AttendanceAbsent, AttendanceLate, AttendanceExcused} {
		if !validStatuses[s] {
			t.Errorf("%q should be valid", s)
		}
	}
	for _, s := range []AttendanceStatus{"", "maybe", "PRESENT", "absent "} {
		if validStatuses[s] {
			t.Errorf("%q should be rejected", s)
		}
	}
}
