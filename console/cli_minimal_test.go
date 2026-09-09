package console

import (
	"strings"
	"testing"

	"github.com/zatrano/framework/v2/kernel"
)

func TestDatabaseCommandsAbsentWithoutImport(t *testing.T) {
	cli := New(kernel.NewApplication(t.TempDir()))
	for _, name := range []string{
		"db:setup", "migrate", "make:migration", "db:seed",
		"cache:clear", "queue:work", "make:job",
		"make:view", "down", "up",
		"make:auth", "make:dashboard", "make:policy",
		"schedule:run", "make:notification", "lang:publish",
		"make:factory", "octane:start", "openapi:generate",
	} {
		if _, ok := cli.Commands()[name]; ok {
			t.Fatalf("command %q must not appear unless its package is imported", name)
		}
	}
	for _, name := range []string{"config:cache", "route:list", "storage:link", "make:test"} {
		if _, ok := cli.Commands()[name]; !ok {
			t.Fatalf("core command %q must remain on the framework CLI", name)
		}
	}
	for _, name := range []string{"make:request", "make:rule"} {
		if _, ok := cli.Commands()[name]; ok {
			t.Fatalf("command %q belongs to the validation package, not the kernel CLI", name)
		}
	}
}

func TestMakeRequestHintDoesNotImportPackages(t *testing.T) {
	cli := New(kernel.NewApplication(t.TempDir()))
	err := cli.Run([]string{"make:request", "Foo"})
	if err == nil {
		t.Fatal("expected make:request to fail on kernel CLI")
	}
	msg := err.Error()
	if !strings.Contains(msg, "validation") || !strings.Contains(msg, "go run ./cmd/app make:request") {
		t.Fatalf("expected actionable validation hint, got %q", msg)
	}
	if strings.Contains(msg, "github.com/zatrano/packages") && strings.Contains(msg, "\nimport") {
		t.Fatalf("hint must not import packages: %q", msg)
	}
}
