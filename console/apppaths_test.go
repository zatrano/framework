package console

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zatrano/framework/v2/kernel"
)

func TestConsumerModuleFallsBack(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	if got := consumerModule(app); got != "your/module" {
		t.Fatalf("got %q", got)
	}
}

func TestUpsertAddonBlankImport(t *testing.T) {
	dir := t.TempDir()
	if err := upsertAddonBlankImports(dir, []string{"github.com/zatrano/packages/oauth"}); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(dir, "bootstrap", "addons.go"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	if !strings.Contains(text, `_ "github.com/zatrano/packages/oauth"`) {
		t.Fatalf("missing blank import:\n%s", text)
	}
	if addonImportPath("ai") != "github.com/zatrano/packages/ai" {
		t.Fatalf("ai import path: %s", addonImportPath("ai"))
	}
	if addonImportPath("oauth") != "github.com/zatrano/packages/oauth" {
		t.Fatalf("oauth import path: %s", addonImportPath("oauth"))
	}
}
