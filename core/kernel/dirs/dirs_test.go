package dirs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTemplatesDir(t *testing.T) {
	dir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(dir, "templates"), 0o755)
	// TemplatesDir requires contracts.App; exercise Dir via TemplatesDirForCreate with nil
	if TemplatesDir(nil) != "templates" {
		t.Fatal(TemplatesDir(nil))
	}
	if DatabaseDir(nil) != "database" {
		t.Fatal(DatabaseDir(nil))
	}
}

func TestCanonicalConsumerDirs(t *testing.T) {
	dirs := CanonicalConsumerDirs()
	need := map[string]bool{"core": false, "app/http/handlers/web": false, "templates": false, "database": false}
	for _, d := range dirs {
		if _, ok := need[d]; ok {
			need[d] = true
		}
	}
	for k, ok := range need {
		if !ok {
			t.Fatalf("missing %s", k)
		}
	}
}

func TestHandlerRelAllowed(t *testing.T) {
	if !HandlerRelAllowed("app/http/handlers/web/home_handler.go") {
		t.Fatal("web")
	}
	if !HandlerRelAllowed("app/http/handlers/api/home_handler.go") {
		t.Fatal("api")
	}
	if HandlerRelAllowed("app/http/handlers/home.go") {
		t.Fatal("surface required")
	}
	if HandlerKind("app/http/handlers/api/x.go") != SurfaceAPI {
		t.Fatal("api kind")
	}
}
