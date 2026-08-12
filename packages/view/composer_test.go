package view_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zatrano/framework/packages/view"
)

func TestComposer(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "welcome.html")
	if err := os.WriteFile(path, []byte(`Hello {{ index . "title" }} / {{ index . "from_composer" }}`), 0o644); err != nil {
		t.Fatal(err)
	}
	engine := view.New(dir)
	engine.Composer("welcome", func(name string, data map[string]any) {
		data["from_composer"] = "yes"
	})
	out, err := engine.Render("welcome", map[string]any{"title": "ZATRANO"})
	if err != nil {
		t.Fatal(err)
	}
	if out != "Hello ZATRANO / yes" {
		t.Fatalf("got %q", out)
	}
}
