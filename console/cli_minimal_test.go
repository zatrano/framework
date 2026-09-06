package console

import (
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
	for _, name := range []string{"config:cache", "route:list", "make:request", "make:rule", "storage:link", "make:test"} {
		if _, ok := cli.Commands()[name]; !ok {
			t.Fatalf("core command %q must remain on the framework CLI", name)
		}
	}
}
