package kernel

import (
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// unsafeAllowlist lists the directories (relative to the repository root, slash
// separated) whose Go files may import unsafe or use //go:linkname. The
// framework needs neither; adding an entry here is a reviewed decision that
// also needs a line in docs/SECURITY-GUARANTEES.md.
var unsafeAllowlist = map[string]bool{}

// TestUnsafeOnlyInAllowlistedPackages enforces invariant I5 over every Go file
// in the repository, tests and nested modules included.
func TestUnsafeOnlyInAllowlistedPackages(t *testing.T) {
	root := repoRoot(t)
	var hits []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "vendor", "node_modules":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		rel = filepath.ToSlash(rel)
		if unsafeAllowlist[filepath.ToSlash(filepath.Dir(rel))] {
			return nil
		}
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			return err
		}
		for _, spec := range file.Imports {
			if p, _ := strconv.Unquote(spec.Path.Value); p == "unsafe" {
				hits = append(hits, rel+": import \"unsafe\"")
			}
		}
		for _, group := range file.Comments {
			for _, c := range group.List {
				if strings.HasPrefix(c.Text, "//go:linkname") {
					hits = append(hits, rel+": "+c.Text)
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) > 0 {
		sort.Strings(hits)
		t.Fatalf("unsafe outside the allowlist:\n%s", strings.Join(hits, "\n"))
	}
}

// TestUnsafeAllowlistScannerSeesImports guards the scanner itself: a file that
// imports unsafe must be reported.
func TestUnsafeAllowlistScannerSeesImports(t *testing.T) {
	src := "package p\n\nimport \"unsafe\"\n\n//go:linkname x runtime.x\nvar _ = unsafe.Sizeof(0)\n"
	file, err := parser.ParseFile(token.NewFileSet(), "p.go", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, spec := range file.Imports {
		if p, _ := strconv.Unquote(spec.Path.Value); p == "unsafe" {
			found = true
		}
	}
	linkname := false
	for _, group := range file.Comments {
		for _, c := range group.List {
			if strings.HasPrefix(c.Text, "//go:linkname") {
				linkname = true
			}
		}
	}
	if !found || !linkname {
		t.Fatalf("scanner missed unsafe=%v linkname=%v", found, linkname)
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	dir := filepath.Dir(file)
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("repository root not found")
		}
		dir = parent
	}
}
