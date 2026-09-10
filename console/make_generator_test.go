package console

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zatrano/framework/v2/kernel"
)

func TestMakeControllerWebUsesViewWhenViewEnabled(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "bootstrap", "enabled.go"), "package bootstrap\nvar EnabledAddons = []string{\n\t\"view\",\n}\n")
	app := kernel.NewApplication(dir)
	cmd := &MakeControllerCommand{app: app}
	if err := cmd.Handle([]string{"Post"}); err != nil {
		t.Fatal(err)
	}
	body := readFile(t, filepath.Join(dir, "app", "http", "controllers", "web", "post_controller.go"))
	if !strings.Contains(body, "View(") || strings.Contains(body, "JSON(") {
		t.Fatalf("web controller with view enabled must use View:\n%s", body)
	}
	if !strings.Contains(body, "post.index") {
		t.Fatalf("expected view name post.index:\n%s", body)
	}
}

func TestMakeControllerAPIUsesJSON(t *testing.T) {
	dir := t.TempDir()
	app := kernel.NewApplication(dir)
	cmd := &MakeControllerCommand{app: app}
	if err := cmd.Handle([]string{"Post", "--api"}); err != nil {
		t.Fatal(err)
	}
	body := readFile(t, filepath.Join(dir, "app", "http", "controllers", "api", "post_controller.go"))
	if !strings.Contains(body, "JSON(") || strings.Contains(body, "View(") {
		t.Fatalf("api controller must use JSON:\n%s", body)
	}
}

func TestMakeControllerEmptyUsesHTML(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "bootstrap", "enabled.go"), "package bootstrap\nvar EnabledAddons = []string{\n}\n")
	writeFile(t, filepath.Join(dir, "bootstrap", "scaffold.go"), "package bootstrap\nconst (\n\tScaffoldName      = \"empty\"\n)\n")
	app := kernel.NewApplication(dir)
	cmd := &MakeControllerCommand{app: app}
	if err := cmd.Handle([]string{"Post"}); err != nil {
		t.Fatal(err)
	}
	body := readFile(t, filepath.Join(dir, "app", "http", "controllers", "web", "post_controller.go"))
	if !strings.Contains(body, "HTML(") || strings.Contains(body, "JSON(") || strings.Contains(body, "View(") {
		t.Fatalf("empty web controller must use HTML:\n%s", body)
	}
}

func TestMakeControllerAPIScaffoldWebUsesJSON(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "bootstrap", "enabled.go"), "package bootstrap\nvar EnabledAddons = []string{\n\t\"health\",\n\t\"validation\",\n}\n")
	writeFile(t, filepath.Join(dir, "bootstrap", "scaffold.go"), "package bootstrap\nconst (\n\tScaffoldName      = \"api\"\n)\n")
	app := kernel.NewApplication(dir)
	cmd := &MakeControllerCommand{app: app}
	if err := cmd.Handle([]string{"Post"}); err != nil {
		t.Fatal(err)
	}
	body := readFile(t, filepath.Join(dir, "app", "http", "controllers", "web", "post_controller.go"))
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
