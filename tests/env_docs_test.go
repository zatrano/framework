package tests

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// envName is an HTTP_*, MAX_*, CORS_*, or TRUSTED_PROXIES variable the kernel reads.
var envName = regexp.MustCompile(`^(?:HTTP_[A-Z0-9_]+|MAX_[A-Z0-9_]+|CORS_[A-Z0-9_]+|TRUSTED_PROXIES)$`)

// envNameInDoc finds those names inside the Environment section. ^ and $ would
// only match a section that is exactly one name.
var envNameInDoc = regexp.MustCompile(`(?:HTTP_[A-Z0-9_]+|MAX_[A-Z0-9_]+|CORS_[A-Z0-9_]+|TRUSTED_PROXIES)`)

func TestEnvironmentVariablesAreDocumented(t *testing.T) {
	root := moduleRoot(t)
	doc, err := os.ReadFile(filepath.Join(root, "UPGRADING.md"))
	if err != nil {
		t.Fatal(err)
	}
	section := environmentSection(string(doc))
	if section == "" {
		t.Fatal("UPGRADING.md is missing the ## Environment section")
	}
	documented := map[string]bool{}
	for _, name := range envNameInDoc.FindAllString(section, -1) {
		documented[name] = true
	}
	if len(documented) == 0 {
		t.Fatal("Environment section lists no variables")
	}

	found := map[string]string{}
	fset := token.NewFileSet()
	err = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			base := d.Name()
			if base == ".git" || base == "vendor" || (base != "." && strings.HasPrefix(base, ".")) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			rel = path
		}
		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			for _, arg := range call.Args {
				lit, ok := arg.(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					continue
				}
				name, err := strconv.Unquote(lit.Value)
				if err != nil || !envName.MatchString(name) {
					continue
				}
				if _, seen := found[name]; !seen {
					found[name] = rel + ":" + strconv.Itoa(fset.Position(lit.Pos()).Line)
				}
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	var missing []string
	for name, at := range found {
		if !documented[name] {
			missing = append(missing, name+" ("+at+")")
		}
	}
	if len(missing) > 0 {
		t.Fatalf("environment variables read by code and missing from UPGRADING.md ## Environment:\n%s", strings.Join(missing, "\n"))
	}
}

func environmentSection(doc string) string {
	const heading = "## Environment"
	i := strings.Index(doc, heading)
	if i < 0 {
		return ""
	}
	rest := doc[i+len(heading):]
	if j := strings.Index(rest, "\n## "); j >= 0 {
		return rest[:j]
	}
	return rest
}
