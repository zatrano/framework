package console

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zatrano/framework/v3/core/kernel"
)

func TestMakeControllerWebUsesTemplateWhenTemplateEnabled(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "bootstrap", "enabled.go"), "package bootstrap\nvar EnabledAddons = []string{\n\t\"template\",\n}\n")
	app := kernel.NewApplication(dir)
	cmd := &MakeHandlerCommand{app: app}
	if err := cmd.Handle([]string{"Post"}); err != nil {
		t.Fatal(err)
	}
	body := readFile(t, filepath.Join(dir, "app", "http", "handlers", "web", "post_handler.go"))
	if !strings.Contains(body, "Template(") || strings.Contains(body, "JSON(") {
		t.Fatalf("web handler with template enabled must use Template:\n%s", body)
	}
	if !strings.Contains(body, "post.index") {
		t.Fatalf("expected template name post.index:\n%s", body)
	}
}

func TestMakeControllerWebUsesTemplateWhenLegacyViewEnabled(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "bootstrap", "enabled.go"), "package bootstrap\nvar EnabledAddons = []string{\n\t\"view\",\n}\n")
	app := kernel.NewApplication(dir)
	cmd := &MakeHandlerCommand{app: app}
	if err := cmd.Handle([]string{"Post"}); err != nil {
		t.Fatal(err)
	}
	body := readFile(t, filepath.Join(dir, "app", "http", "handlers", "web", "post_handler.go"))
	if !strings.Contains(body, "Template(") {
		t.Fatalf("legacy view enablement must still scaffold Template:\n%s", body)
	}
}

func TestMakeControllerAPIUsesJSON(t *testing.T) {
	dir := t.TempDir()
	app := kernel.NewApplication(dir)
	cmd := &MakeHandlerCommand{app: app}
	if err := cmd.Handle([]string{"Post", "--api"}); err != nil {
		t.Fatal(err)
	}
	body := readFile(t, filepath.Join(dir, "app", "http", "handlers", "api", "post_handler.go"))
	if !strings.Contains(body, "JSON(") || strings.Contains(body, "Template(") || strings.Contains(body, "View(") {
		t.Fatalf("api controller must use JSON:\n%s", body)
	}
}

func TestMakeControllerEmptyUsesHTML(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "bootstrap", "enabled.go"), "package bootstrap\nvar EnabledAddons = []string{\n}\n")
	writeFile(t, filepath.Join(dir, "bootstrap", "scaffold.go"), "package bootstrap\nconst (\n\tScaffoldName      = \"empty\"\n)\n")
	app := kernel.NewApplication(dir)
	cmd := &MakeHandlerCommand{app: app}
	if err := cmd.Handle([]string{"Post"}); err != nil {
		t.Fatal(err)
	}
	body := readFile(t, filepath.Join(dir, "app", "http", "handlers", "web", "post_handler.go"))
	if !strings.Contains(body, "HTML(") || strings.Contains(body, "JSON(") || strings.Contains(body, "Template(") || strings.Contains(body, "View(") {
		t.Fatalf("empty web controller must use HTML:\n%s", body)
	}
}

func TestMakeControllerAPIScaffoldWebUsesJSON(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "bootstrap", "enabled.go"), "package bootstrap\nvar EnabledAddons = []string{\n\t\"health\",\n\t\"validation\",\n}\n")
	writeFile(t, filepath.Join(dir, "bootstrap", "scaffold.go"), "package bootstrap\nconst (\n\tScaffoldName      = \"api\"\n)\n")
	app := kernel.NewApplication(dir)
	cmd := &MakeHandlerCommand{app: app}
	if err := cmd.Handle([]string{"Post"}); err != nil {
		t.Fatal(err)
	}
	body := readFile(t, filepath.Join(dir, "app", "http", "handlers", "web", "post_handler.go"))
	if !strings.Contains(body, "JSON(") {
		t.Fatalf("api-profile web controller must use JSON:\n%s", body)
	}
}

func TestMakeServiceHasNoHandleMethod(t *testing.T) {
	dir := t.TempDir()
	app := kernel.NewApplication(dir)
	cmd := &MakeServiceCommand{app: app}
	if err := cmd.Handle([]string{"OrderPlacement"}); err != nil {
		t.Fatal(err)
	}
	body := readFile(t, filepath.Join(dir, "app", "services", "order_placement_service.go"))
	if strings.Contains(body, "Handle()") {
		t.Fatalf("service stub must not define Handle():\n%s", body)
	}
	if !strings.Contains(body, "func NewOrderPlacementService()") {
		t.Fatalf("expected constructor:\n%s", body)
	}
}

func TestMakeTestIsKernelOnly(t *testing.T) {
	dir := t.TempDir()
	app := kernel.NewApplication(dir)
	cmd := &MakeTestCommand{app: app}
	if err := cmd.Handle([]string{"Health"}); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(dir, "tests", "health_test.go"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	if strings.Contains(text, "github.com/zatrano/packages") {
		t.Fatalf("make:test must not import packages:\n%s", text)
	}
	if !strings.Contains(text, "net/http/httptest") || !strings.Contains(text, "bootstrap.App()") {
		t.Fatalf("make:test must use kernel HTTP test helpers:\n%s", text)
	}
}

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
