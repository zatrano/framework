package acquire

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Requirement is one go.mod require line as Go tooling left it.
// Inspection reports it; it does not pick or rewrite the version.
type Requirement struct {
	Path     string
	Version  string
	Indirect bool
}

// Checksum is one go.sum line (module, version token, hash).
type Checksum struct {
	Module  string
	Version string
	Hash    string
}

// Inspection is the module graph after Go tooling ran — not the process
// result, not a plan, and not ApplyResult.
type Inspection struct {
	Root         string
	Module       string
	Go           string
	Requirements []Requirement
	Checksums    []Checksum
	GoModMissing bool
	GoSumMissing bool
}

// Inspect reads go.mod and go.sum in root. It does not run go get, tidy,
// Resolve, or write those files. GoGetArg is not reinterpreted here.
func Inspect(root string) (Inspection, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return Inspection{}, fmt.Errorf("acquire: module root required")
	}
	in := Inspection{Root: root}
	modRaw, err := os.ReadFile(filepath.Join(root, "go.mod"))
	switch {
	case err == nil:
		parseGoMod(&in, string(modRaw))
	case os.IsNotExist(err):
		in.GoModMissing = true
	default:
		return Inspection{}, err
	}
	sumRaw, err := os.ReadFile(filepath.Join(root, "go.sum"))
	switch {
	case err == nil:
		in.Checksums = parseGoSum(string(sumRaw))
	case os.IsNotExist(err):
		in.GoSumMissing = true
	default:
		return Inspection{}, err
	}
	return in, nil
}

// Requirement returns the observed require for path, if any.
func (in Inspection) Requirement(path string) (Requirement, bool) {
	path = strings.TrimSpace(path)
	for _, r := range in.Requirements {
		if r.Path == path {
			return r, true
		}
	}
	return Requirement{}, false
}

// Sums returns observed go.sum lines whose module path matches.
func (in Inspection) Sums(module string) []Checksum {
	module = strings.TrimSpace(module)
	var out []Checksum
	for _, c := range in.Checksums {
		if c.Module == module {
			out = append(out, c)
		}
	}
	return out
}

func parseGoMod(in *Inspection, src string) {
	block := ""
	for _, raw := range strings.Split(src, "\n") {
		line := strings.TrimSpace(strings.TrimSuffix(raw, "\r"))
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}
		if block != "" {
			if line == ")" {
				block = ""
				continue
			}
			if block == "require" {
				if r, ok := parseRequireLine(line); ok {
					in.Requirements = append(in.Requirements, r)
				}
			}
			continue
		}
		switch {
		case strings.HasPrefix(line, "module "):
			in.Module = unquoteModToken(stripModComment(strings.TrimSpace(strings.TrimPrefix(line, "module "))))
		case strings.HasPrefix(line, "go "):
			fields := strings.Fields(stripModComment(strings.TrimPrefix(line, "go ")))
			if len(fields) > 0 {
				in.Go = fields[0]
			}
		case strings.HasPrefix(line, "require"):
			rest := strings.TrimSpace(strings.TrimPrefix(line, "require"))
			if rest == "(" || strings.HasPrefix(rest, "(") {
				block = "require"
				continue
			}
			if r, ok := parseRequireLine(rest); ok {
				in.Requirements = append(in.Requirements, r)
			}
		case isSkippedModDirective(line):
			if strings.HasSuffix(line, "(") {
				block = "skip"
			}
		}
	}
}

func isSkippedModDirective(line string) bool {
	for _, p := range []string{"replace", "exclude", "retract", "tool", "ignore"} {
		if line == p+" (" || strings.HasPrefix(line, p+" ") || strings.HasPrefix(line, p+"\t") {
			return true
		}
	}
	return false
}

func parseRequireLine(line string) (Requirement, bool) {
	indirect := false
	if i := strings.Index(line, "//"); i >= 0 {
		if strings.Contains(line[i:], "indirect") {
			indirect = true
		}
		line = strings.TrimSpace(line[:i])
	}
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return Requirement{}, false
	}
	return Requirement{
		Path:     unquoteModToken(fields[0]),
		Version:  fields[1],
		Indirect: indirect,
	}, true
}

func parseGoSum(src string) []Checksum {
	var out []Checksum
	for _, raw := range strings.Split(src, "\n") {
		line := strings.TrimSpace(strings.TrimSuffix(raw, "\r"))
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		out = append(out, Checksum{Module: fields[0], Version: fields[1], Hash: fields[2]})
	}
	return out
}

func stripModComment(s string) string {
	if i := strings.Index(s, "//"); i >= 0 {
		return strings.TrimSpace(s[:i])
	}
	return strings.TrimSpace(s)
}

func unquoteModToken(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}
