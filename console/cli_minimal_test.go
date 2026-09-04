package console

import (
	"testing"

	"github.com/zatrano/framework/kernel"
)

func TestDatabaseCommandsAbsentWithoutImport(t *testing.T) {
	cli := New(kernel.NewApplication(t.TempDir()))
	for _, name := range []string{"db:setup", "migrate", "make:migration", "db:seed"} {
		if _, ok := cli.Commands()[name]; ok {
			t.Fatalf("command %q must not appear unless the database package is imported", name)
		}
	}
}
