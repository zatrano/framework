package consolecore

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zatrano/framework/v2/kernel"
)

type stubCmd struct {
	name string
	err  error
	ran  *bool
}

func (s stubCmd) Name() string        { return s.name }
func (s stubCmd) Description() string { return s.name + " desc" }
func (s stubCmd) Handle(args []string) error {
	if s.ran != nil {
		*s.ran = true
	}
	return s.err
}

func TestApplicationRunAndList(t *testing.T) {
	app := New(kernel.NewApplication(t.TempDir()))
	ran := false
	app.Register(stubCmd{name: "list", ran: &ran}, stubCmd{name: "version"}, stubCmd{name: "ping", ran: &ran})
	if err := app.Run(nil); err != nil {
		t.Fatal(err)
	}
	if !ran {
		t.Fatal("empty args must run list")
	}
	ran = false
	if err := app.Run([]string{"ping"}); err != nil {
		t.Fatal(err)
	}
	if !ran {
		t.Fatal("named command")
	}
	if err := app.Run([]string{"missing"}); err == nil || !strings.Contains(err.Error(), "missing") {
		t.Fatalf("unknown: %v", err)
	}
	called := false
	app.Unknown = func(name string) error {
		called = true
		return errors.New("hint:" + name)
	}
	if err := app.Run([]string{"nope"}); err == nil || !called {
		t.Fatalf("Unknown callback: %v", err)
	}
	if cmds := app.Commands(); cmds["ping"] == nil {
		t.Fatal("Commands")
	}
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	err = app.WriteCommandList()
	_ = w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	if err != nil || !strings.Contains(buf.String(), "ping") {
		t.Fatalf("list=%q err=%v", buf.String(), err)
	}
}

func TestCLIErrorAndClassify(t *testing.T) {
	if CodeFromError(nil) != ExitSuccess {
		t.Fatal("nil")
	}
	if CodeFromError(errors.New("x")) != ExitGeneral {
		t.Fatal("plain")
	}
	ce := &CLIError{Code: ExitUsage, Err: errors.New("bad")}
	if ce.Error() != "bad" || ce.Unwrap().Error() != "bad" || ce.ExitCode() != ExitUsage {
		t.Fatal(ce)
	}
	if CodeFromError(ce) != ExitUsage {
		t.Fatal("as")
	}
	if CliErr(ExitUsage, nil) != nil {
		t.Fatal("nil cli")
	}
	if CliFailed(ExitUsage, "act", "tgt", nil, "") != nil {
		t.Fatal("nil failed")
	}
	err := CliFailed(ExitUsage, "act", "tgt", errors.New("e"), "retry")
	if !strings.Contains(err.Error(), "act") || !strings.Contains(err.Error(), "retry") {
		t.Fatal(err)
	}
	if ClassifyContextError(nil) != nil {
		t.Fatal("nil ctx")
	}
	if CodeFromError(ClassifyContextError(context.Canceled)) != ExitCanceled {
		t.Fatal("cancel")
	}
	if ClassifyRuntimeError(nil) != nil {
		t.Fatal("nil runtime")
	}
	if CodeFromError(ClassifyRuntimeError(context.Canceled)) != ExitRuntimeCanceled {
		t.Fatal("runtime cancel")
	}
	if CodeFromError(ClassifyRuntimeError(context.DeadlineExceeded)) != ExitRuntimeTimeout {
		t.Fatal("timeout")
	}
	if CodeFromError(ClassifyRuntimeError(kernel.ErrRuntimeBoot)) != ExitRuntimeBoot {
		t.Fatal("boot")
	}
	if CodeFromError(ClassifyRuntimeError(kernel.ErrRuntimeShutdown)) != ExitRuntimeShutdown {
		t.Fatal("shutdown")
	}
}

func TestModulePathSeedEnvAndVersion(t *testing.T) {
	dir := t.TempDir()
	if _, err := ModulePath(dir); err == nil {
		t.Fatal("missing go.mod")
	}
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/app\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	mod, err := ModulePath(dir)
	if err != nil || mod != "example.com/app" {
		t.Fatalf("mod=%q %v", mod, err)
	}
	if ConsumerModule(nil) != "your/module" {
		t.Fatal("nil app")
	}
	k := kernel.NewApplication(dir)
	if ConsumerModule(k) != "example.com/app" {
		t.Fatal(ConsumerModule(k))
	}
	if err := SeedEnvFromExample(dir); err == nil {
		t.Fatal("missing example")
	}
	if err := os.WriteFile(filepath.Join(dir, ".env.example"), []byte("A=1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := SeedEnvFromExample(dir); err != nil {
		t.Fatal(err)
	}
	if err := SeedEnvFromExample(dir); err != nil {
		t.Fatal(err)
	}
	root, err := FrameworkModuleRoot()
	if err != nil {
		t.Fatal(err)
	}
	if ProductVersionAt(root) == "" {
		t.Fatal("version")
	}
	if ProductVersionAt(t.TempDir()) != CurrentRelease {
		t.Fatal("fallback")
	}
	if ProductVersion() == "" {
		t.Fatal("ProductVersion")
	}
	got, err := FormatFromArgs([]string{"--json"})
	if err != nil || got != "json" {
		t.Fatal(got, err)
	}
	if _, err := FormatFromArgs([]string{"--format=xml"}); err == nil {
		t.Fatal("bad format")
	}
	if ToExported("") != "" {
		t.Fatal("empty export")
	}
}
