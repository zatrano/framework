package console

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zatrano/framework/kernel"
)

func TestWriteDatabaseDriversFileEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bootstrap", "database_drivers.go")
	if err := writeDatabaseDriversFile(path, nil); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	if strings.Contains(text, "import (") {
		t.Fatalf("empty drivers should not import: %s", text)
	}
	if !strings.Contains(text, "No database drivers linked") {
		t.Fatalf("missing none comment: %s", text)
	}
}

func TestWriteDatabaseDriversFileSQLite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bootstrap", "database_drivers.go")
	if err := writeDatabaseDriversFile(path, []string{"sqlite"}); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	if !strings.Contains(text, `github.com/zatrano/framework/packages/database/driver/sqlite`) {
		t.Fatalf("sqlite import missing: %s", text)
	}
}

func TestDBSetupYesLinksNoDatabase(t *testing.T) {
	dir := t.TempDir()
	app := kernel.NewApplication(dir)
	cmd := &DBSetupCommand{app: app}
	if err := cmd.Handle([]string{"--yes"}); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(dir, "bootstrap", "database_drivers.go"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), "driver/sqlite") {
		t.Fatalf("sqlite should not be linked by default:\n%s", body)
	}
	envBody, err := os.ReadFile(filepath.Join(dir, ".env"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(envBody), "DB_CONNECTION=sqlite") {
		t.Fatalf(".env still defaults sqlite:\n%s", envBody)
	}
}
