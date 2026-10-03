package sms

import (
	"strings"
	"testing"
)

func TestCountSegments(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		units    int
		encoding string
	}{
		{"empty", "", 0, EncodingGSM7},
		{"short", "Hello", 1, EncodingGSM7},
		{"exactly 160 GSM", strings.Repeat("a", 160), 1, EncodingGSM7},
		{"161 GSM splits at 153", strings.Repeat("a", 161), 2, EncodingGSM7},
		{"306 GSM is two units", strings.Repeat("a", 306), 2, EncodingGSM7},
		{"307 GSM is three units", strings.Repeat("a", 307), 3, EncodingGSM7},
		{"460 GSM is four units", strings.Repeat("a", 460), 4, EncodingGSM7},
		{"Swahili text", "Habari za asubuhi, mwanafunzi wako amefika shuleni salama", 1, EncodingGSM7},
		// An extension character costs two septets: 159 + 2 = 161.
		{"euro sign counts twice", strings.Repeat("a", 159) + "€", 2, EncodingGSM7},
		// A curly apostrophe (as pasted from Word) is not in GSM-7.
		{"smart quote forces Unicode", "Ng’ombe", 1, EncodingUCS2},
		{"70 Unicode fits one unit", strings.Repeat("’", 70), 1, EncodingUCS2},
		{"71 Unicode splits at 67", strings.Repeat("’", 71), 2, EncodingUCS2},
		{"135 Unicode is three units", strings.Repeat("’", 135), 3, EncodingUCS2},
		// 100 GSM characters would be one unit; one emoji makes it two.
		{"one emoji in a long message", strings.Repeat("a", 100) + "🎉", 2, EncodingUCS2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CountSegments(tt.message)
			if got.Units != tt.units || got.Encoding != tt.encoding {
				t.Errorf("CountSegments() = %d units %s, want %d units %s", got.Units, got.Encoding, tt.units, tt.encoding)
			}
			if CalculateSMSUnits(tt.message) != tt.units {
				t.Errorf("CalculateSMSUnits() = %d, want %d", CalculateSMSUnits(tt.message), tt.units)
			}
		})
	}
}

// An emoji is two UTF-16 code units, and that is what is billed.
func TestCountSegmentsCountsSurrogatePairs(t *testing.T) {
	if got := CountSegments(strings.Repeat("🎉", 35)); got.Units != 1 || got.Length != 70 {
		t.Errorf("35 emoji = %d units, length %d; want 1 unit, length 70", got.Units, got.Length)
	}
	if got := CountSegments(strings.Repeat("🎉", 36)); got.Units != 2 {
		t.Errorf("36 emoji = %d units, want 2", got.Units)
	}
}

func TestVariablesAndInject(t *testing.T) {
	template := "Dear {{parent_name}}, {{ Learner_Name }} is in {{class}}. {{parent_name}}!"
	got := Variables(template)
	want := []string{"parent_name", "learner_name", "class"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("Variables() = %v, want %v", got, want)
	}

	out := InjectVariables(template, map[string]string{"parent_name": "Mary", "learner_name": "Amina"})
	// A name with no value stays in place so the caller can refuse the send.
	if out != "Dear Mary, Amina is in {{class}}. Mary!" {
		t.Errorf("InjectVariables() = %q", out)
	}
	if len(Variables("No variables here, even with { braces }")) != 0 {
		t.Error("Variables() found a variable in plain text")
	}
}

func TestDLRStatus(t *testing.T) {
	cases := map[string]string{
		"Success":   "delivered",
		"success":   "delivered",
		"Failed":    "failed",
		"Rejected":  "failed",
		"Sent":      "",
		"Submitted": "",
		"Buffered":  "",
		"":          "",
		"Nonsense":  "",
	}
	for reported, want := range cases {
		if got := DLRStatus(reported); got != want {
			t.Errorf("DLRStatus(%q) = %q, want %q", reported, got, want)
		}
	}
}

func TestSMSResultCostAndAccountLevel(t *testing.T) {
	if got := (SMSResult{Cost: "KES 0.8000"}).CostCents(); got != 80 {
		t.Errorf("CostCents(KES 0.8000) = %d, want 80", got)
	}
	if got := (SMSResult{Cost: "KES 1.6000"}).CostCents(); got != 160 {
		t.Errorf("CostCents(KES 1.6000) = %d, want 160", got)
	}
	if got := (SMSResult{Cost: "0"}).CostCents(); got != 0 {
		t.Errorf("CostCents(0) = %d, want 0", got)
	}
	if !(SMSResult{StatusCode: 405}).AccountLevel() || !(SMSResult{StatusCode: 402}).AccountLevel() {
		t.Error("insufficient balance and invalid sender id are account-level refusals")
	}
	if (SMSResult{StatusCode: 403}).AccountLevel() {
		t.Error("an invalid phone number is not an account-level refusal")
	}
}
