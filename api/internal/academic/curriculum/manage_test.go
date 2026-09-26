package curriculum

import "testing"

func TestHasChildrenErrorPluralises(t *testing.T) {
	tests := []struct {
		parent string
		child  string
		count  int
		want   string
	}{
		{"Mathematics", "strands", 1, `"Mathematics" still has 1 strand — remove them first`},
		{"Mathematics", "strands", 3, `"Mathematics" still has 3 strands — remove them first`},
		{"Numbers", "sub-strands", 1, `"Numbers" still has 1 sub-strand — remove them first`},
		{"Counting", "learning outcomes", 2, `"Counting" still has 2 learning outcomes — remove them first`},
		{"Counting", "recorded assessments", 1, `"Counting" still has 1 recorded assessment — remove them first`},
		{"Counting", "recorded assessments", 4, `"Counting" still has 4 recorded assessments — remove them first`},
	}

	for _, tc := range tests {
		err := &HasChildrenError{Parent: tc.parent, Child: tc.child, Count: tc.count}
		if got := err.Error(); got != tc.want {
			t.Errorf("got %q, want %q", got, tc.want)
		}
	}
}
