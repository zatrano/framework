// Package goversion compares Go release names such as "go1.26.9".
// The doctor check and the production boot warning share it.
package goversion

import (
	"strconv"
	"strings"
)

// RecommendedMinimum is the oldest Go release that contains the current
// stdlib security fixes. Review it on every framework release. The release
// workflow compares it with the newest stable patch on the same minor from
// https://go.dev/dl/?mode=json and warns when this constant is more than
// two patches behind.
const RecommendedMinimum = "go1.26.9"

// Release is a parsed Go release name.
type Release struct {
	Major int
	Minor int
	Patch int
	Pre   bool
}

// Parse reads "go1.26", "go1.26.9", or "go1.27rc1". Text after the first
// space is ignored, so runtime.Version() of a release build parses.
// A devel build does not parse.
func Parse(s string) (Release, bool) {
	s = strings.TrimSpace(s)
	if i := strings.IndexAny(s, " \t"); i >= 0 {
		s = s[:i]
	}
	if !strings.HasPrefix(s, "go") {
		return Release{}, false
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
		return Release{}, false
	}
	major, err1 := strconv.Atoi(parts[0])
	minor, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil || major < 0 || minor < 0 {
		return Release{}, false
	}
	patch := 0
	if len(parts) == 3 {
		p, err := strconv.Atoi(parts[2])
		if err != nil || p < 0 {
			return Release{}, false
		}
		patch = p
	}
	return Release{Major: major, Minor: minor, Patch: patch, Pre: pre}, true
}

// OlderThan reports whether have is an older release than min.
// An unparsable have is treated as older. An unparsable min is not.
func OlderThan(have, min string) bool {
	want, ok := Parse(min)
	if !ok {
		return false
	}
	got, ok := Parse(have)
	if !ok {
		return true
	}
	if got.Major != want.Major {
		return got.Major < want.Major
	}
	if got.Minor != want.Minor {
		return got.Minor < want.Minor
	}
	if got.Patch != want.Patch {
		return got.Patch < want.Patch
	}
	return got.Pre && !want.Pre
}
