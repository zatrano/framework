package pkgmanager

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zatrano/framework/v2/kernel"
)

func TestUpsertAddonBlankImport(t *testing.T) {
	dir := t.TempDir()
	if err := upsertAddonBlankImports(dir, []string{"github.com/zatrano/packages/auth/oauth"}); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(dir, "bootstrap", "addons.go"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	if !strings.Contains(text, `_ "github.com/zatrano/packages/auth/oauth"`) {
		t.Fatalf("missing blank import:\n%s", text)
	}
	if addonImportPath("ai") != "github.com/zatrano/packages/ai" {
		t.Fatalf("ai import path: %s", addonImportPath("ai"))
	}
	if addonImportPath("oauth") != "github.com/zatrano/packages/auth/oauth" {
		t.Fatalf("oauth import path: %s", addonImportPath("oauth"))
	}
	if addonImportPath("authorization") != "github.com/zatrano/packages/auth/authorization" {
		t.Fatalf("authorization import path: %s", addonImportPath("authorization"))
	}
	if addonImportPath("apitoken") != "github.com/zatrano/packages/auth/token" {
		t.Fatalf("apitoken import path: %s", addonImportPath("apitoken"))
	}
	if addonImportPath("social") != "github.com/zatrano/packages/auth/social" {
		t.Fatalf("social import path: %s", addonImportPath("social"))
	}
	if addonImportPath("webauthn") != "github.com/zatrano/packages/webauthn" {
		t.Fatalf("webauthn import path: %s", addonImportPath("webauthn"))
	}
}

func TestWireEnablementRewritesLegacyAuthImports(t *testing.T) {
	dir := t.TempDir()
	if err := upsertAddonBlankImports(dir, []string{"github.com/zatrano/packages/oauth"}); err != nil {
		t.Fatal(err)
	}
	app := kernel.NewApplication(dir)
	if err := wireEnabledAddons(app, []string{"oauth"}); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(dir, "bootstrap", "addons.go"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	if !strings.Contains(text, `_ "github.com/zatrano/packages/auth/oauth"`) {
		t.Fatalf("canonical import missing:\n%s", text)
	}
	if strings.Contains(text, `_ "github.com/zatrano/packages/oauth"`) {
		t.Fatalf("legacy import must be removed:\n%s", text)
	}
}
