package doctor

import (
	"errors"
	"os/exec"
	"strconv"
	"strings"
)

// errGoMissing means the project directory has no go binary to ask.
// APP-GO-001 is skipped in that case.
var errGoMissing = errors.New("go toolchain not found")

// goVersionLookup reports `go env GOVERSION` for dir.
// Tests replace it. Production runs the go on PATH with dir as the working directory.
var goVersionLookup = lookupGoVersion

func lookupGoVersion(dir string) (string, error) {
	path, err := exec.LookPath("go")
	if err != nil {
		return "", errGoMissing
	}
	cmd := exec.Command(path, "env", "GOVERSION")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		var execErr *exec.Error
		if errors.As(err, &execErr) || errors.Is(err, exec.ErrNotFound) {
			return "", errGoMissing
		}
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

type goRelease struct {
	major int
	minor int
	patch int
	pre   bool
}

func parseGoRelease(s string) (goRelease, bool) {
	s = strings.TrimSpace(s)
	if i := strings.IndexAny(s, " \t"); i >= 0 {
		s = s[:i]
	}
	if !strings.HasPrefix(s, "go") {
		return goRelease{}, false
	}
	rest := strings.TrimPrefix(s, "go")
	pre := false
	for _, suf := range []string{"rc", "beta", "alpha"} {
		if i := strings.Index(rest, suf); i >= 0 {
			rest = strings.TrimSuffix(rest[:i], ".")
			pre = true
			break
		}
	}
	parts := strings.Split(rest, ".")
	if len(parts) < 2 || len(parts) > 3 {
		return goRelease{}, false
	}
	major, err1 := strconv.Atoi(parts[0])
	minor, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil || major < 0 || minor < 0 {
		return goRelease{}, false
	}
	patch := 0
	if len(parts) == 3 {
		p, err := strconv.Atoi(parts[2])
		if err != nil || p < 0 {
			return goRelease{}, false
		}
		patch = p
	}
	return goRelease{major: major, minor: minor, patch: patch, pre: pre}, true
}

// goOlderThan reports whether have is an older release than min.
// An unparsable have is treated as older. An unparsable min is not.
func goOlderThan(have, min string) bool {
	want, ok := parseGoRelease(min)
	if !ok {
		return false
	}
	got, ok := parseGoRelease(have)
	if !ok {
		return true
	}
	if got.major != want.major {
		return got.major < want.major
	}
	if got.minor != want.minor {
		return got.minor < want.minor
	}
	if got.patch != want.patch {
		return got.patch < want.patch
	}
	return got.pre && !want.pre
}

func checkGoToolchain(root string) ([]Finding, error) {
	ver, err := goVersionLookup(root)
	if err != nil {
		if errors.Is(err, errGoMissing) {
			return nil, nil
		}
		return nil, err
	}
	if !goOlderThan(ver, RecommendedGoMinimum) {
		return nil, nil
	}
	return []Finding{{
		Rule:     "APP-GO-001",
		Check:    "go-toolchain",
		Severity: "warning",
		File:     ".",
		Found:    ver,
		Why:      "stdlib security fixes require Go >= " + RecommendedGoMinimum,
		How:      "Build this project with Go " + RecommendedGoMinimum + " or newer.",
		See:      "SECURITY.md",
	}}, nil
}
