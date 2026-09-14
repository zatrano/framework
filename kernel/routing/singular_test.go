package routing

import "testing"

func TestSingular(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"", ""},
		{"cat", "cat"},
		{"cats", "cat"},
		{"notes", "note"},
		{"children", "child"},
		{"CHILDREN", "CHILD"},
		{"Children", "Child"},
		{"cities", "city"},
		{"leaves", "leaf"},
		{"boxes", "box"},
		{"watches", "watch"},
		{"dishes", "dish"},
		{"classes", "class"},
		{"buzzes", "buzz"},
		{"buses", "bus"},
		{"bus", "bu"},
		{"ss", "ss"},
	}
	for _, tc := range cases {
		if got := singular(tc.in); got != tc.want {
			t.Errorf("singular(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
}

func TestMatchCaseEmptyValue(t *testing.T) {
	if got := matchCase("X", ""); got != "" {
		t.Fatalf("matchCase empty value: %q", got)
	}
	if got := matchCase("", "child"); got != "child" {
		t.Fatalf("matchCase empty sample: %q", got)
	}
}
