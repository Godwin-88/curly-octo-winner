package sms

import (
	"testing"
)

func TestNormalizeKenyanPhone(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "Safaricom 07 format",
			input: "0712345678",
			want:  "+254712345678",
		},
		{
			name:  "Airtel 07 format with spaces",
			input: "07 1234 5678",
			want:  "+254712345678",
		},
		{
			name:  "New Safaricom 01 format",
			input: "0112345678",
			want:  "+254112345678",
		},
		{
			name:  "Full +254 format",
			input: "+254712345678",
			want:  "+254712345678",
		},
		{
			name:  "254 without plus",
			input: "254712345678",
			want:  "+254712345678",
		},
		{
			name:  "7-digit local format (missing leading 0)",
			input: "712345678",
			want:  "+254712345678",
		},
		{
			name:  "With dashes",
			input: "0712-345-678",
			want:  "+254712345678",
		},
		{
			name:  "With parentheses",
			input: "(0)712 345 678",
			want:  "+254712345678",
		},
		{
			name:  "With dots",
			input: "0712.345.678",
			want:  "+254712345678",
		},
		{
			name:    "Empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "Too short",
			input:   "071234",
			wantErr: true,
		},
		{
			name:    "Invalid format",
			input:   "12345",
			wantErr: true,
		},
		{
			name:    "Non-Kenyan number",
			input:   "+255712345678",
			wantErr: true,
		},
		{
			name:    "Invalid +254 length",
			input:   "+25471234",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeKenyanPhone(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("NormalizeKenyanPhone(%q) expected error, got %q", tt.input, got)
				}
				return
			}
			if err != nil {
				t.Errorf("NormalizeKenyanPhone(%q) unexpected error: %v", tt.input, err)
				return
			}
			if got != tt.want {
				t.Errorf("NormalizeKenyanPhone(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestCalculateSMSUnits(t *testing.T) {
	tests := []struct {
		name    string
		message string
		want    int
	}{
		{"Empty message", "", 0},
		{"Short message", "Hello", 1},
		{"Exactly 160 chars", string(make([]rune, 160)), 1},
		{"161 chars", string(make([]rune, 161)), 2},
		{"320 chars", string(make([]rune, 320)), 2},
		{"321 chars", string(make([]rune, 321)), 3},
		{"Over 480 chars (capped at 3)", string(make([]rune, 1000)), 3},
		{"Swahili text", "Habari za asubuhi, mwanafunzi wako amefika shuleni salama", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateSMSUnits(tt.message)
			if got != tt.want {
				t.Errorf("CalculateSMSUnits() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestEstimateCost(t *testing.T) {
	client := NewATClient("test-key", "sandbox", "SHULE360", false)
	estimate := client.EstimateCost(100, 1)

	if estimate.RecipientCount != 100 {
		t.Errorf("RecipientCount = %d, want 100", estimate.RecipientCount)
	}
	if estimate.SMSUnits != 1 {
		t.Errorf("SMSUnits = %d, want 1", estimate.SMSUnits)
	}
	// 100 recipients * 1 unit * 0.80 KES = 80.00
	if estimate.EstimatedKES != 80.0 {
		t.Errorf("EstimatedKES = %.2f, want 80.00", estimate.EstimatedKES)
	}
}

func TestNewATClientSandboxVsProduction(t *testing.T) {
	sandbox := NewATClient("key", "sandbox", "SHULE360", false)
	if sandbox.baseURL != "https://api.sandbox.africastalking.com" {
		t.Errorf("sandbox baseURL = %q, want sandbox URL", sandbox.baseURL)
	}

	prod := NewATClient("key", "school", "SHULE360", true)
	if prod.baseURL != "https://api.africastalking.com" {
		t.Errorf("production baseURL = %q, want production URL", prod.baseURL)
	}
}
