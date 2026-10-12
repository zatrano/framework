package console

import (
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestConsoleAcquireCLIMayImportAcquire(t *testing.T) {
	dir := filepath.Join(moduleRoot(t), "core", "console")
	fset := token.NewFileSet()
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		base := filepath.Base(path)
		file, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, spec := range file.Imports {
			imp := strings.Trim(spec.Path.Value, `"`)
			if !strings.HasSuffix(imp, "/acquire") {
				continue
			}
			if base != "package_acquire.go" && !strings.HasPrefix(base, "package_acquire") {
				t.Errorf("%s imports acquire — only package:acquire may consume acquire (%s)", base, imp)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestAcquisitionExitCodesStayAtCLIBoundary(t *testing.T) {
	root := moduleRoot(t)
	err := filepath.WalkDir(filepath.Join(root, "core", "distribution", "acquire"), func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		src, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		text := string(src)
		for _, ban := range []string{"ExitUsage", "ExitResolution", "ExitPlanning", "ExitAcquisition", "ExitEnablement", "ExitCanceled", "ExitRuntimeBoot", "ExitRuntimeShutdown", "ExitRuntimeCanceled", "ExitRuntimeTimeout", "CodeFromError", "type CLIError"} {
			if strings.Contains(text, ban) {
				t.Errorf("%s contains %s — exit-code logic stays at the CLI boundary", filepath.Base(path), ban)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	cli, err := os.ReadFile(filepath.Join(root, "core", "console", "cli_exit.go"))
	if err != nil {
		t.Fatal("missing core/console/cli_exit.go — classified exit codes belong to the CLI")
	}
	text := string(cli)
	for _, want := range []string{"ExitUsage", "ExitResolution", "ExitPlanning", "ExitAcquisition", "ExitEnablement", "ExitCanceled", "ExitRuntimeBoot", "ExitRuntimeShutdown", "ExitRuntimeCanceled", "ExitRuntimeTimeout", "func CodeFromError"} {
		if !strings.Contains(text, want) {
			t.Errorf("cli_exit.go missing %s", want)
		}
	}
	mainSrc, err := os.ReadFile(filepath.Join(root, "cmd", "zatrano", "main.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(mainSrc), "console.CodeFromError(") {
		t.Error("cmd/zatrano must map CLI errors to exit codes")
	}
}

func moduleRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	dir := filepath.Dir(file)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("module root")
		}
		dir = parent
	}
}
