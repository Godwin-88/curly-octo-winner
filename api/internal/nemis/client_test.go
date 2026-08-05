package nemis

import (
	"context"
	"errors"
	"testing"
)

func TestSandboxValidateUPI(t *testing.T) {
	client := NewSandboxNEMISClient()
	ctx := context.Background()

	t.Run("valid TEST UPI", func(t *testing.T) {
		learner, err := client.ValidateUPI(ctx, "TEST12345678")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if learner.UPI != "TEST12345678" {
			t.Errorf("UPI = %q, want TEST12345678", learner.UPI)
		}
		if learner.FullName == "" {
			t.Error("FullName should not be empty")
		}
		if learner.Grade == "" {
			t.Error("Grade should not be empty")
		}
		if learner.SchoolCode != "TEST-SCHOOL-001" {
			t.Errorf("SchoolCode = %q, want TEST-SCHOOL-001", learner.SchoolCode)
		}
	})

	t.Run("invalid UPI format", func(t *testing.T) {
		_, err := client.ValidateUPI(ctx, "INVALID")
		if !errors.Is(err, ErrInvalidUPI) {
			t.Errorf("expected ErrInvalidUPI, got %v", err)
		}
	})

	t.Run("real UPI not found in sandbox", func(t *testing.T) {
		_, err := client.ValidateUPI(ctx, "ABCDEF1234567890")
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})

	t.Run("empty UPI", func(t *testing.T) {
		_, err := client.ValidateUPI(ctx, "")
		if !errors.Is(err, ErrInvalidUPI) {
			t.Errorf("expected ErrInvalidUPI, got %v", err)
		}
	})
}

func TestSandboxSearchLearner(t *testing.T) {
	client := NewSandboxNEMISClient()
	ctx := context.Background()

	results, err := client.SearchLearner(ctx, NEMISSearchRequest{
		FullName: "Test Learner",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestSandboxDeterministicData(t *testing.T) {
	client := NewSandboxNEMISClient()
	ctx := context.Background()

	// Same UPI should produce same data
	l1, _ := client.ValidateUPI(ctx, "TEST00000001")
	l2, _ := client.ValidateUPI(ctx, "TEST00000001")

	if l1.FullName != l2.FullName {
		t.Errorf("FullName not deterministic: %q vs %q", l1.FullName, l2.FullName)
	}
	if l1.Grade != l2.Grade {
		t.Errorf("Grade not deterministic: %q vs %q", l1.Grade, l2.Grade)
	}
	if l1.Gender != l2.Gender {
		t.Errorf("Gender not deterministic: %q vs %q", l1.Gender, l2.Gender)
	}
}
