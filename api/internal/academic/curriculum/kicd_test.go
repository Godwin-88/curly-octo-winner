package curriculum

import "testing"

func TestNullIfEmpty(t *testing.T) {
	if got := nullIfEmpty(""); got != nil {
		t.Errorf(`nullIfEmpty("") = %v, want nil (a blank code must be stored as NULL so it does not collide with other code-less rows)`, *got)
	}
	got := nullIfEmpty("MTH")
	if got == nil {
		t.Fatal(`nullIfEmpty("MTH") = nil, want a pointer to "MTH"`)
	}
	if *got != "MTH" {
		t.Errorf("nullIfEmpty(\"MTH\") = %q, want \"MTH\"", *got)
	}
}
