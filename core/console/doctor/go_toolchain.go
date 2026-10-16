package doctor

import (
	"errors"
	"os/exec"
	"strings"

	"github.com/zatrano/framework/v3/core/kernel/goversion"
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

func checkGoToolchain(root string) ([]Finding, error) {
	ver, err := goVersionLookup(root)
	if err != nil {
		if errors.Is(err, errGoMissing) {
			return nil, nil
		}
		return nil, err
	}
	if !goversion.OlderThan(ver, RecommendedGoMinimum) {
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
