package httputil

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

type row struct {
	ID string `json:"id"`
}

// A list endpoint that has nothing to return must answer [], never null: the
// web client calls rows.map() straight on the response.
func TestRespondOKWritesEmptyArrayForNilSlice(t *testing.T) {
	var rows []row

	rec := httptest.NewRecorder()
	RespondOK(rec, rows)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	// json.Encoder appends a newline, so compare the decoded value.
	var got []row
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("body is not a JSON array (%s): %v", rec.Body.String(), err)
	}
	if got == nil {
		t.Fatalf("body = %s, want []", rec.Body.String())
	}
	if len(got) != 0 {
		t.Errorf("got %d rows, want 0", len(got))
	}
}

func TestRespondOKPreservesPopulatedSlices(t *testing.T) {
	rows := []row{{ID: "a"}, {ID: "b"}}

	rec := httptest.NewRecorder()
	RespondOK(rec, rows)

	var got []row
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("got %d rows, want 2", len(got))
	}
}

// Objects, maps, nil and scalars must pass through untouched.
func TestRespondOKPassesOtherPayloadsThrough(t *testing.T) {
	tests := []struct {
		name    string
		payload any
		want    string
	}{
		{"nil", nil, "null"},
		{"struct", row{ID: "x"}, `{"id":"x"}`},
		{"map", map[string]int{"n": 1}, `{"n":1}`},
		{"empty non-nil slice", []row{}, "[]"},
		{"empty string", "", `""`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			RespondOK(rec, tc.payload)

			var got any
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatalf("decode %s: %v", rec.Body.String(), err)
			}
			wantJSON, _ := json.Marshal(tc.payload)
			var want any
			_ = json.Unmarshal(wantJSON, &want)

			if !reflect.DeepEqual(got, want) {
				t.Errorf("body = %s, want %s", rec.Body.String(), wantJSON)
			}
		})
	}
}
