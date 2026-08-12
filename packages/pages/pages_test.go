package pages_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zatrano/framework/packages/pages"
	"github.com/zatrano/framework/packages/routing"
	"github.com/zatrano/framework/packages/view"
)

func TestPagesRegister(t *testing.T) {
	root := t.TempDir()
	pagesDir := filepath.Join(root, "pages")
	_ = os.MkdirAll(filepath.Join(pagesDir, "users"), 0o755)
	_ = os.WriteFile(filepath.Join(pagesDir, "about.html"), []byte("<h1>About</h1>"), 0o644)
	_ = os.WriteFile(filepath.Join(pagesDir, "users", "[id].html"), []byte("<p>id</p>"), 0o644)

	engine := view.New(root)
	router := routing.New()
	if err := pages.New(pagesDir, engine).Prefix("/pages").Register(router); err != nil {
		t.Fatal(err)
	}
	snap := router.Snapshot()
	foundAbout, foundUser := false, false
	for _, r := range snap {
		if r.Path == "/pages/about" {
			foundAbout = true
		}
		if r.Path == "/pages/users/{id}" {
			foundUser = true
		}
	}
	if !foundAbout || !foundUser {
		t.Fatalf("routes=%#v", snap)
	}
}
