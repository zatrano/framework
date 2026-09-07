package acquire

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/zatrano/framework/v2/distribution/registry"
)

func requireGo(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go executable not on PATH")
	}
}

func offlineGoEnv(t *testing.T) {
	t.Helper()
	t.Setenv("GOPROXY", "off")
	t.Setenv("GOSUMDB", "off")
	t.Setenv("GOTOOLCHAIN", "local")
}

func writeLocalApp(t *testing.T) (app, arg string) {
	t.Helper()
	root := t.TempDir()
	app = filepath.Join(root, "app")
	dep := filepath.Join(root, "dep")
	if err := os.Mkdir(app, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(dep, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dep, "go.mod"), []byte("module example.com/dep\n\ngo 1.25.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dep, "dep.go"), []byte("package dep\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	mod := "module example.com/app\n\ngo 1.25.0\n\nreplace example.com/dep => ../dep\n"
	if err := os.WriteFile(filepath.Join(app, "go.mod"), []byte(mod), 0o644); err != nil {
		t.Fatal(err)
	}
	return app, "example.com/dep@v0.0.0"
}

func assertGetArgv(t *testing.T, res InvocationResult, root, arg string) {
	t.Helper()
	if res.Invocation.Path != "go" {
		t.Fatalf("path=%q", res.Invocation.Path)
	}
	if res.Invocation.Dir != root {
		t.Fatalf("dir=%q want %q", res.Invocation.Dir, root)
	}
	got := res.Invocation.Args
	if len(got) != 2 || got[0] != "get" || got[1] != arg {
		t.Fatalf("args=%q — want {get, %s}", got, arg)
	}
	joined := strings.Join(got, " ")
	if strings.Contains(joined, "tidy") || strings.Contains(joined, "mod edit") {
		t.Fatalf("extra go command in argv: %q", got)
	}
	if strings.Contains(strings.ToLower(joined), "latest") {
		t.Fatal("latest leaked into argv")
	}
}

func TestExecuteRunsGoGetAndTransfersResult(t *testing.T) {
	requireGo(t)
	offlineGoEnv(t)
	app, arg := writeLocalApp(t)
	res, err := Execute(context.Background(), Request{Root: app, GoGetArg: arg})
	if err != nil {
		t.Fatalf("Execute: %v\nstdout=%s\nstderr=%s", err, res.Stdout, res.Stderr)
	}
	assertGetArgv(t, res, app, arg)
	if res.ExitCode != 0 {
		t.Fatalf("exit=%d stderr=%s", res.ExitCode, res.Stderr)
	}
}

func TestExecuteTransfersGoGetFailure(t *testing.T) {
	requireGo(t)
	offlineGoEnv(t)
	app, _ := writeLocalApp(t)
	arg := "github.com/zatrano/packages@v1.4.0"
	res, err := Execute(context.Background(), Request{Root: app, GoGetArg: arg})
	if err == nil {
		t.Fatal("expected go get failure with GOPROXY=off")
	}
	assertGetArgv(t, res, app, arg)
	if res.ExitCode == 0 {
		t.Fatal("failed go get must not report exit 0")
	}
	if strings.TrimSpace(res.Stderr) == "" {
		t.Fatal("stderr must reach InvocationResult")
	}
}

func TestExecuteConsumesFrozenGoGetArg(t *testing.T) {
	requireGo(t)
	offlineGoEnv(t)
	plan, err := FromResult(registry.Result{
		Package: registry.Package{Name: "session", Module: "github.com/zatrano/packages"},
		Release: registry.Release{Channel: registry.ChannelMain},
	})
	if err != nil {
		t.Fatal(err)
	}
	arg := plan.GoGetArg()
	app, _ := writeLocalApp(t)
	res, _ := Execute(context.Background(), Request{Root: app, GoGetArg: arg})
	assertGetArgv(t, res, app, arg)
}

func TestExecuteDoesNotStartOnLatest(t *testing.T) {
	requireGo(t)
	app, _ := writeLocalApp(t)
	res, err := Execute(context.Background(), Request{Root: app, GoGetArg: "github.com/zatrano/packages@latest"})
	if err == nil {
		t.Fatal("latest must not reach go get")
	}
	if res.Invocation.Path != "" || len(res.Invocation.Args) != 0 {
		t.Fatalf("process must not start: %#v", res.Invocation)
	}
}

func TestExecuteCancellationReachesGoProcess(t *testing.T) {
	requireGo(t)
	offlineGoEnv(t)
	app, arg := writeLocalApp(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := Execute(ctx, Request{Root: app, GoGetArg: arg})
	if err == nil {
		t.Fatal("expected context error")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("want context.Canceled, got %v", err)
	}
}

func TestExecuteTimeoutReachesGoProcess(t *testing.T) {
	requireGo(t)
	offlineGoEnv(t)
	app, arg := writeLocalApp(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
	defer cancel()
	time.Sleep(2 * time.Millisecond)
	_, err := Execute(ctx, Request{Root: app, GoGetArg: arg})
	if err == nil {
		t.Fatal("expected deadline error")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("want context.DeadlineExceeded, got %v", err)
	}
}
