package console

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zatrano/framework/v2/kernel"
)

func TestMakeCommandGeneratesContractsApp(t *testing.T) {
	dir := t.TempDir()
	app := kernel.NewApplication(dir)
	cmd := &MakeCommandCommand{app: app}
	if err := cmd.Handle([]string{"ReportDigest"}); err != nil {
		t.Fatal(err)
	}

	commandPath := filepath.Join(dir, "app", "console", "commands", "report_digest_command.go")
	body, err := os.ReadFile(commandPath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	if !strings.Contains(text, "App contracts.App") {
		t.Fatalf("make:command must store contracts.App:\n%s", text)
	}
	if strings.Contains(text, "*kernel.Application") || strings.Contains(text, `"github.com/zatrano/framework/v2/kernel"`) {
		t.Fatalf("make:command must not leak *kernel.Application:\n%s", text)
	}

	kernelPath := filepath.Join(dir, "app", "console", "kernel.go")
	kbody, err := os.ReadFile(kernelPath)
	if err != nil {
		t.Fatal(err)
	}
	ktext := string(kbody)
	if !strings.Contains(ktext, "func Register(cli *coreconsole.Application, app contracts.App)") {
		t.Fatalf("make:command kernel.go must take contracts.App:\n%s", ktext)
	}
	if strings.Contains(ktext, "*kernel.Application") {
		t.Fatalf("make:command kernel.go must not leak *kernel.Application:\n%s", ktext)
	}
}

func TestMakeCommandDoesNotRewriteExistingKernel(t *testing.T) {
	dir := t.TempDir()
	kernelPath := filepath.Join(dir, "app", "console", "kernel.go")
	if err := os.MkdirAll(filepath.Dir(kernelPath), 0o755); err != nil {
		t.Fatal(err)
	}
	existing := "package console\n\nfunc Register() {}\n"
	if err := os.WriteFile(kernelPath, []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}
	app := kernel.NewApplication(dir)
	cmd := &MakeCommandCommand{app: app}
	if err := cmd.Handle([]string{"Ping"}); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(kernelPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != existing {
		t.Fatalf("existing kernel.go rewritten:\n%s", got)
	}
}
