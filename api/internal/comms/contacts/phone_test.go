package contacts

import (
	"strings"
	"testing"
)

func TestNormalizePhone(t *testing.T) {
	valid := []struct {
		in   string
		want string
	}{
		{"0712345678", "+254712345678"},
		{"+254712345678", "+254712345678"},
		{"254712345678", "+254712345678"},
		{"712345678", "+254712345678"},
		{"+254 712 345 678", "+254712345678"},
		{"0712-345-678", "+254712345678"},
		{"  0712345678  ", "+254712345678"},
		{"+254 (712) 345/678", "+254712345678"},
	}
	for _, tc := range valid {
		got, err := NormalizePhone(tc.in)
		if err != nil {
			t.Errorf("NormalizePhone(%q) returned error: %v", tc.in, err)
			continue
		}
		if got != tc.want {
			t.Errorf("NormalizePhone(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}

	invalid := []string{
		"",
		"   ",
		"12345",            // too short
		"071234567890123",  // too long
		"+25471234567",     // 254 + 8 digits
		"+1 202 555 0143",  // not Kenyan
		"+254 20 123 4567", // valid Kenyan landline, but not a mobile
		"012345678",        // landline range
		"abcdefghij",
		"07 1234 567a",
	}
	for _, in := range invalid {
		if got, err := NormalizePhone(in); err == nil {
			t.Errorf("NormalizePhone(%q) = %q, want an error", in, got)
		}
	}
}

// Two spellings of the same person must collapse to one stored value,
// otherwise bulk import creates duplicate rows for a single human.
func TestNormalizePhoneCollapsesFormats(t *testing.T) {
	formats := []string{"0712345678", "+254712345678", "254712345678", "0712 345 678"}
	var first string
	for _, f := range formats {
		got, err := NormalizePhone(f)
		if err != nil {
			t.Fatalf("NormalizePhone(%q): %v", f, err)
		}
		if first == "" {
			first = got
			continue
		}
		if got != first {
			t.Fatalf("NormalizePhone(%q) = %q, want %q (all formats must collapse)", f, got, first)
		}
	}
}

func TestValidateEmail(t *testing.T) {
	for _, ok := range []string{"", "  ", "parent@example.com", "a.b+tag@sub.domain.co.ke"} {
		if err := ValidateEmail(ok); err != nil {
			t.Errorf("ValidateEmail(%q) = %v, want nil", ok, err)
		}
	}
	for _, bad := range []string{"parent", "parent@", "@example.com", "a@b@c.com", "parent@example", "parent@.com"} {
		if err := ValidateEmail(bad); err == nil {
			t.Errorf("ValidateEmail(%q) = nil, want an error", bad)
		}
	}
}

func TestNormalizeTags(t *testing.T) {
	got, err := NormalizeTags([]string{" Grade 4 ", "Grade 4", "PARENTS", "boarding, day"})
	if err != nil {
		t.Fatalf("NormalizeTags: %v", err)
	}
	want := []string{"grade 4", "parents", "boarding", "day"}
	if len(got) != len(want) {
		t.Fatalf("NormalizeTags = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("NormalizeTags = %v, want %v", got, want)
		}
	}

	if _, err := NormalizeTags([]string{strings.Repeat("x", 41)}); err == nil {
		t.Error("NormalizeTags accepted a 41 character tag, want an error")
	}
	if _, err := NormalizeTags([]string{"has,comma"}); err != nil {
		// Commas are a separator, not an error.
		t.Errorf("NormalizeTags(%q) = %v, want the comma to split", "has,comma", err)
	}
	if _, err := NormalizeTags([]string{`has"quote`}); err == nil {
		t.Error("NormalizeTags accepted a quote character, want an error")
	}

	many := make([]string, 21)
	for i := range many {
		many[i] = "tag" + string(rune('a'+i))
	}
	if _, err := NormalizeTags(many); err == nil {
		t.Error("NormalizeTags accepted 21 tags, want an error")
	}
}
