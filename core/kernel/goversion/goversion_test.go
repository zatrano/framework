package goversion

import "testing"

func TestParseAndOlderThan(t *testing.T) {
	cases := []struct {
		in    string
		want  Release
		ok    bool
		older bool
	}{
		{in: "go1.26.8", want: Release{Major: 1, Minor: 26, Patch: 8}, ok: true, older: true},
		{in: "go1.26.9", want: Release{Major: 1, Minor: 26, Patch: 9}, ok: true, older: false},
		{in: "go1.26.10", want: Release{Major: 1, Minor: 26, Patch: 10}, ok: true, older: false},
		{in: "go1.27.2", want: Release{Major: 1, Minor: 27, Patch: 2}, ok: true, older: false},
		{in: "go1.25.14", want: Release{Major: 1, Minor: 25, Patch: 14}, ok: true, older: true},
		{in: "go1.26", want: Release{Major: 1, Minor: 26}, ok: true, older: true},
		{in: "go1.26.9rc1", want: Release{Major: 1, Minor: 26, Patch: 9, Pre: true}, ok: true, older: true},
		{in: "go1.27rc1", want: Release{Major: 1, Minor: 27, Pre: true}, ok: true, older: false},
		{in: "go1.26.9 X:boringcrypto", want: Release{Major: 1, Minor: 26, Patch: 9}, ok: true, older: false},
		{in: "devel go1.27-abcdef", ok: false, older: true},
	}
	for _, tc := range cases {
		got, ok := Parse(tc.in)
		if ok != tc.ok || got != tc.want {
			t.Errorf("Parse(%q) = %+v %v, want %+v %v", tc.in, got, ok, tc.want, tc.ok)
		}
		if OlderThan(tc.in, RecommendedMinimum) != tc.older {
			t.Errorf("OlderThan(%q, %q) = %v, want %v", tc.in, RecommendedMinimum, !tc.older, tc.older)
		}
	}
	if OlderThan("go1.26.8", "not-a-version") {
		t.Fatal("an unparsable minimum must not warn")
	}
}
