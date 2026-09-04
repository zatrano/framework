package console

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zatrano/framework/kernel"
)

func TestConsumerModuleFallsBack(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	if got := consumerModule(app); got != "your/module" {
		t.Fatalf("got %q", got)
	}
}

func TestMakeFactoryUsesConsumerModule(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/shop\n\ngo 1.25\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	app := kernel.NewApplication(dir)
	cmd := &MakeFactoryCommand{app: app}
	if err := cmd.Handle([]string{"Product"}); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(dir, "app", "database", "factories", "product_factory.go"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	if strings.Contains(text, "github.com/zatrano/framework/app") {
		t.Fatalf("factory still imports framework app:\n%s", text)
	}
	if !strings.Contains(text, `"example.com/shop/app/models"`) {
		t.Fatalf("expected consumer module import:\n%s", text)
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
	if addonImportPath("ai") != "github.com/zatrano/framework/packages/ai" {
		t.Fatalf("ai import path: %s", addonImportPath("ai"))
	}
	if addonImportPath("oauth") != "github.com/zatrano/packages/oauth" {
		t.Fatalf("oauth import path: %s", addonImportPath("oauth"))
	}
}
