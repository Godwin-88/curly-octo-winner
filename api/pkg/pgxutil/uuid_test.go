package pgxutil

import (
	"reflect"
	"testing"

	"github.com/google/uuid"
)

func TestUUIDArray(t *testing.T) {
	a := uuid.MustParse("c0000000-0000-0000-0000-000000000001")
	b := uuid.MustParse("c0000000-0000-0000-0000-000000000002")

	tests := []struct {
		name string
		in   []uuid.UUID
		want []string
	}{
		{"nil", nil, []string{}},
		{"empty", []uuid.UUID{}, []string{}},
		{"single", []uuid.UUID{a}, []string{a.String()}},
		{"many", []uuid.UUID{a, b}, []string{a.String(), b.String()}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := UUIDArray(tc.in)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("UUIDArray() = %#v, want %#v", got, tc.want)
			}
		})
	}
}

// A nil input must still produce a non-nil slice: pgx encodes nil []string as
// SQL NULL, which the NOT NULL guardian_ids column rejects.
func TestUUIDArrayNeverNil(t *testing.T) {
	if got := UUIDArray(nil); got == nil {
		t.Fatal("UUIDArray(nil) returned nil, want an empty slice")
	}
}
