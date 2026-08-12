package assets_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zatrano/framework/packages/assets"
)

func TestManifestURL(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.json")
	content := `{"css/app.css":{"file":"assets/app.css","src":"css/app.css","isEntry":true}}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	m := assets.New("http://localhost/build")
	if err := m.LoadFile(path); err != nil {
		t.Fatal(err)
	}
	got := m.URL("css/app.css")
	if got != "http://localhost/build/assets/app.css" {
		t.Fatalf("got %q", got)
	}
}
