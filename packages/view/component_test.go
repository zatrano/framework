package view_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zatrano/framework/packages/view"
)

func TestComponentRender(t *testing.T) {
	dir := t.TempDir()
	compDir := filepath.Join(dir, "components")
	if err := os.MkdirAll(compDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(compDir, "alert.html"), []byte(`ALERT:{{ index . "message" }}`), 0o644); err != nil {
		t.Fatal(err)
	}
	engine := view.New(dir)
	out, err := engine.Component("alert", map[string]any{"message": "hi"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "ALERT:hi") {
		t.Fatalf("got %q", out)
	}
}
