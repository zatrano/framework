package tests

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUnsafeIsOnlyTheStaticRouteKey(t *testing.T) {
	root := moduleRoot(t)
	var hits []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "vendor", "testdata":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return err
		}
		ast.Inspect(file, func(n ast.Node) bool {
			sel, ok := n.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			id, ok := sel.X.(*ast.Ident)
			if !ok || id.Name != "unsafe" {
				return true
			}
			rel, _ := filepath.Rel(root, path)
			hits = append(hits, filepath.ToSlash(rel)+": unsafe."+sel.Sel.Name)
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 || hits[0] != "core/kernel/routing/router.go: unsafe.String" {
		t.Fatalf("unsafe calls:\n%s", strings.Join(hits, "\n"))
	}
	body, err := os.ReadFile(filepath.Join(root, "core", "kernel", "routing", "router.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "is not retained") {
		t.Fatal("the static lookup alias must stay documented as not retained")
	}
}
