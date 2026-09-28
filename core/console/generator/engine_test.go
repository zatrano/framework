package generator

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

func TestStripTmplSuffix(t *testing.T) {
	if got := StripTmplSuffix("go.mod.tmpl"); got != "go.mod" {
		t.Fatalf("got %q", got)
	}
	if got := StripTmplSuffix("cmd/app/main.go.tmpl"); got != "cmd/app/main.go" {
		t.Fatalf("got %q", got)
	}
}

func TestApplySubstitutesAndWritesScaffoldMeta(t *testing.T) {
	fsys := fstest.MapFS{
		"templates/empty/cmd/app/main.go.tmpl": {Data: []byte("module=__MODULE__\n")},
		"templates/empty/README.md":            {Data: []byte("app=__APP_NAME__\n")},
	}
	dest := filepath.Join(t.TempDir(), "app")
	err := Apply(Request{
		FS:              fsys,
		Root:            "templates/empty",
		Dest:            dest,
		ScaffoldName:    ScaffoldEmpty,
		ScaffoldVersion: "2.0.28",
		Substitutions: map[string]string{
			"__MODULE__":   "example.com/demo",
			"__APP_NAME__": "demo",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	main, err := os.ReadFile(filepath.Join(dest, "cmd", "app", "main.go"))
	if err != nil {
		t.Fatal(err)
	}
	if string(main) != "module=example.com/demo\n" {
		t.Fatalf("main.go: %q", main)
	}
	meta, err := os.ReadFile(filepath.Join(dest, "bootstrap", "scaffold.go"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(meta)
	if !strings.Contains(text, `ScaffoldName      = "empty"`) {
		t.Fatalf("scaffold.go:\n%s", text)
	}
	if !strings.Contains(text, "sha256:") {
		t.Fatalf("missing digest:\n%s", text)
	}
}

func TestApplyRejectsExistingDestAndPathEscape(t *testing.T) {
	dir := t.TempDir()
	if err := Apply(Request{FS: fstest.MapFS{}, Root: "templates/web", Dest: dir}); err == nil {
		t.Fatal("expected existing dest error")
	}
	fsys := fstest.MapFS{"templates/web/ok.go": {Data: []byte("x")}}
	if err := Apply(Request{FS: fsys, Root: "templates/../secret", Dest: filepath.Join(t.TempDir(), "n")}); err == nil {
		t.Fatal("expected path escape error")
	}
}

func TestDigestStable(t *testing.T) {
	fsys := fstest.MapFS{
		"templates/web/a.go": {Data: []byte("one")},
		"templates/web/b.go": {Data: []byte("two")},
	}
	d1, err := Digest(fsys, "templates/web")
	if err != nil {
		t.Fatal(err)
	}
	d2, err := Digest(fs.FS(fsys), "templates/web")
	if err != nil {
		t.Fatal(err)
	}
	if d1 != d2 || !strings.HasPrefix(d1, "sha256:") {
		t.Fatalf("d1=%s d2=%s", d1, d2)
	}
}

func TestDigestUsesSortedSlashPaths(t *testing.T) {
	fsys := fstest.MapFS{
		"templates/web/z.go": {Data: []byte("z")},
		"templates/web/a.go": {Data: []byte("a")},
	}
	d1, err := Digest(fsys, "templates/web")
	if err != nil {
		t.Fatal(err)
	}
	fsys2 := fstest.MapFS{
		"templates/web/a.go": {Data: []byte("a")},
		"templates/web/z.go": {Data: []byte("z")},
	}
	d2, err := Digest(fsys2, "templates/web")
	if err != nil {
		t.Fatal(err)
	}
	if d1 != d2 {
		t.Fatalf("map declaration order must not affect digest: %s vs %s", d1, d2)
	}
}

func TestWriteExclusive(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app", "x.go")
	if err := WriteExclusive(path, "package app\n"); err != nil {
		t.Fatal(err)
	}
	if err := WriteExclusive(path, "nope"); err == nil {
		t.Fatal("expected exists error")
	}
}

func TestOverlayWritesSkipsAndReplacesBaseStub(t *testing.T) {
	TestOverlayReplacesExactKnownStub(t)
}

func TestOverlayReplacesExactKnownStub(t *testing.T) {
	fsys := fstest.MapFS{
		"templates/web/new.go":    {Data: []byte("new")},
		"templates/web/keep.go":   {Data: []byte("web-keep")},
		"templates/web/home.go":   {Data: []byte("view")},
		"templates/empty/keep.go": {Data: []byte("empty-keep")},
		"templates/empty/home.go": {Data: []byte("html")},
	}
	dest := t.TempDir()
	if err := os.WriteFile(filepath.Join(dest, "keep.go"), []byte("user"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dest, "home.go"), []byte("html"), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := Overlay(OverlayRequest{
		FS: fsys, Root: "templates/web", Dest: dest,
		BaseFS: fsys, BaseRoot: "templates/empty",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Written) != 1 || res.Written[0] != "new.go" {
		t.Fatalf("written=%v", res.Written)
	}
	if len(res.Replaced) != 1 || res.Replaced[0] != "home.go" {
		t.Fatalf("replaced=%v", res.Replaced)
	}
	if len(res.Skipped) != 1 || res.Skipped[0] != "keep.go" {
		t.Fatalf("skipped=%v", res.Skipped)
	}
	home, _ := os.ReadFile(filepath.Join(dest, "home.go"))
	if string(home) != "view" {
		t.Fatalf("home=%q", home)
	}
	keep, _ := os.ReadFile(filepath.Join(dest, "keep.go"))
	if string(keep) != "user" {
		t.Fatalf("keep overwritten: %q", keep)
	}
}

func TestOverlayDoesNotOverwriteUserCode(t *testing.T) {
	fsys := fstest.MapFS{
		"templates/web/home.go":   {Data: []byte("view")},
		"templates/empty/home.go": {Data: []byte("html")},
	}
	dest := t.TempDir()
	user := []byte("package web\n// customized\n")
	if err := os.WriteFile(filepath.Join(dest, "home.go"), user, 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := Overlay(OverlayRequest{
		FS: fsys, Root: "templates/web", Dest: dest,
		BaseFS: fsys, BaseRoot: "templates/empty",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Replaced) != 0 || len(res.Skipped) != 1 {
		t.Fatalf("replaced=%v skipped=%v", res.Replaced, res.Skipped)
	}
	got, _ := os.ReadFile(filepath.Join(dest, "home.go"))
	if string(got) != string(user) {
		t.Fatalf("user source changed: %q", got)
	}
}

func TestOverlayRejectsModifiedStub(t *testing.T) {
	fsys := fstest.MapFS{
		"templates/web/home.go":   {Data: []byte("view")},
		"templates/empty/home.go": {Data: []byte("html")},
	}
	dest := t.TempDir()
	// Exact stub plus one trailing space is not a known stub.
	if err := os.WriteFile(filepath.Join(dest, "home.go"), []byte("html "), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := Overlay(OverlayRequest{
		FS: fsys, Root: "templates/web", Dest: dest,
		BaseFS: fsys, BaseRoot: "templates/empty",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Replaced) != 0 || len(res.Skipped) != 1 {
		t.Fatalf("modified stub must not be replaced: replaced=%v skipped=%v", res.Replaced, res.Skipped)
	}
	got, _ := os.ReadFile(filepath.Join(dest, "home.go"))
	if string(got) != "html " {
		t.Fatalf("got %q", got)
	}
}
