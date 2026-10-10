package doctor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseGoRelease(t *testing.T) {
	cases := []struct {
		in        string
		want      goRelease
		ok        bool
		olderThan string
		older     bool
	}{
		{in: "go1.26.8", want: goRelease{major: 1, minor: 26, patch: 8}, ok: true, olderThan: RecommendedGoMinimum, older: true},
		{in: "go1.26.9", want: goRelease{major: 1, minor: 26, patch: 9}, ok: true, olderThan: RecommendedGoMinimum, older: false},
		{in: "go1.26.10", want: goRelease{major: 1, minor: 26, patch: 10}, ok: true, olderThan: RecommendedGoMinimum, older: false},
		{in: "go1.27.2", want: goRelease{major: 1, minor: 27, patch: 2}, ok: true, olderThan: RecommendedGoMinimum, older: false},
		{in: "go1.25.14", want: goRelease{major: 1, minor: 25, patch: 14}, ok: true, olderThan: RecommendedGoMinimum, older: true},
		{in: "go1.26", want: goRelease{major: 1, minor: 26}, ok: true, olderThan: RecommendedGoMinimum, older: true},
		{in: "go1.26.9rc1", want: goRelease{major: 1, minor: 26, patch: 9, pre: true}, ok: true, olderThan: RecommendedGoMinimum, older: true},
		{in: "go1.27rc1", want: goRelease{major: 1, minor: 27, pre: true}, ok: true, olderThan: RecommendedGoMinimum, older: false},
		{in: "devel go1.27-abcdef", ok: false, olderThan: RecommendedGoMinimum, older: true},
	}
	for _, tc := range cases {
		got, ok := parseGoRelease(tc.in)
		if ok != tc.ok || got != tc.want {
			t.Errorf("parse %q = %+v %v, want %+v %v", tc.in, got, ok, tc.want, tc.ok)
		}
		if goOlderThan(tc.in, tc.olderThan) != tc.older {
			t.Errorf("goOlderThan(%q, %q) = %v, want %v", tc.in, tc.olderThan, !tc.older, tc.older)
		}
	}
	if goOlderThan("go1.26.8", "not-a-version") {
		t.Fatal("an unparsable minimum must not warn")
	}
}

func TestAPPGO001SkippedWhenGoMissing(t *testing.T) {
	root := doctorAppDir(t)
	prev := goVersionLookup
	goVersionLookup = func(string) (string, error) { return "", errGoMissing }
	t.Cleanup(func() { goVersionLookup = prev })
	if hasDoctorRule(mustDoctor(t, root), "APP-GO-001") {
		t.Fatal("missing go must skip APP-GO-001")
	}
}

func TestAPPGO001WarnsWhenToolchainIsOlder(t *testing.T) {
	root := doctorAppDir(t)
	wantDir, err := filepath.Abs(root)
	if err != nil {
		t.Fatal(err)
	}
	prev := goVersionLookup
	goVersionLookup = func(dir string) (string, error) {
		if dir != wantDir {
			t.Errorf("lookup dir = %q, want %q", dir, wantDir)
		}
		return "go1.25.14", nil
	}
	t.Cleanup(func() { goVersionLookup = prev })
	findings := mustDoctor(t, root)
	var got *Finding
	for i := range findings {
		if findings[i].Rule == "APP-GO-001" {
			got = &findings[i]
			break
		}
	}
	if got == nil {
		t.Fatalf("expected APP-GO-001:\n%s", FormatDoctorText(root, findings))
	}
	if got.Severity != "warning" {
		t.Fatalf("severity %q", got.Severity)
	}
	if !strings.Contains(got.Why, "stdlib security fixes require Go >= "+RecommendedGoMinimum) {
		t.Fatalf("why %q", got.Why)
	}
}

func TestAPPGO001SilentOnRecommendedToolchain(t *testing.T) {
	root := doctorAppDir(t)
	prev := goVersionLookup
	goVersionLookup = func(string) (string, error) { return RecommendedGoMinimum, nil }
	t.Cleanup(func() { goVersionLookup = prev })
	if hasDoctorRule(mustDoctor(t, root), "APP-GO-001") {
		t.Fatal("recommended toolchain must not warn")
	}
}

func doctorAppDir(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "app"), 0o755); err != nil {
		t.Fatal(err)
	}
	return root
}
