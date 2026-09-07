package acquire

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/zatrano/framework/v2/distribution/registry"
)

var errMutation = errors.New("mutation failed")

type fakeRunner struct {
	calls    []Invocation
	exitCode int
	stdout   string
	stderr   string
	err      error
	started  chan struct{}
	wait     <-chan struct{}
}

func (f *fakeRunner) Run(ctx context.Context, inv Invocation) (InvocationResult, error) {
	f.calls = append(f.calls, inv)
	if f.started != nil {
		close(f.started)
	}
	if f.wait != nil {
		select {
		case <-ctx.Done():
			return InvocationResult{Invocation: inv}, ctx.Err()
		case <-f.wait:
		}
	}
	if ctx.Err() != nil {
		return InvocationResult{Invocation: inv}, ctx.Err()
	}
	return InvocationResult{
		Invocation: inv,
		ExitCode:   f.exitCode,
		Stdout:     f.stdout,
		Stderr:     f.stderr,
	}, f.err
}

func TestInvokePassesGoGetArgUnchanged(t *testing.T) {
	arg := "github.com/zatrano/packages@main"
	fake := &fakeRunner{}
	res, err := Invoke(context.Background(), fake, Request{
		Root:     t.TempDir(),
		GoGetArg: arg,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(fake.calls) != 1 {
		t.Fatalf("calls=%d", len(fake.calls))
	}
	got := fake.calls[0]
	if got.Path != "go" {
		t.Fatalf("path=%q", got.Path)
	}
	if len(got.Args) != 2 || got.Args[0] != "get" || got.Args[1] != arg {
		t.Fatalf("args=%q — GoGetArg must be one argv token", got.Args)
	}
	if strings.Contains(strings.Join(got.Args, " "), "latest") {
		t.Fatal("latest leaked into argv")
	}
	if res.Invocation.Args[1] != arg {
		t.Fatalf("result arg=%q", res.Invocation.Args[1])
	}
}

func TestInvokeConsumesPlanGoGetArgUnchanged(t *testing.T) {
	plan, err := FromResult(registry.Result{
		Package: registry.Package{Name: "session", Module: "github.com/zatrano/packages"},
		Release: registry.Release{Channel: registry.ChannelMain},
	})
	if err != nil {
		t.Fatal(err)
	}
	arg := plan.GoGetArg()
	if arg == "" || strings.Contains(strings.ToLower(arg), "latest") {
		t.Fatalf("plan GoGetArg=%q", arg)
	}
	fake := &fakeRunner{}
	if _, err := Invoke(context.Background(), fake, Request{Root: t.TempDir(), GoGetArg: arg}); err != nil {
		t.Fatal(err)
	}
	if fake.calls[0].Args[1] != arg {
		t.Fatalf("rewrote GoGetArg %q → %q", arg, fake.calls[0].Args[1])
	}
}

func TestInvokeRejectsLatest(t *testing.T) {
	fake := &fakeRunner{}
	root := t.TempDir()
	for _, arg := range []string{"latest", "github.com/zatrano/packages@latest"} {
		if _, err := Invoke(context.Background(), fake, Request{Root: root, GoGetArg: arg}); err == nil {
			t.Fatalf("expected error for %q", arg)
		}
	}
	if len(fake.calls) != 0 {
		t.Fatalf("runner must not run on latest: %+v", fake.calls)
	}
}

func TestInvokeRequiresRootAndArg(t *testing.T) {
	fake := &fakeRunner{}
	if _, err := Invoke(context.Background(), fake, Request{GoGetArg: "github.com/zatrano/packages@main"}); err == nil {
		t.Fatal("empty root")
	}
	if _, err := Invoke(context.Background(), fake, Request{Root: t.TempDir()}); err == nil {
		t.Fatal("empty arg")
	}
	var nilCtx context.Context
	if _, err := Invoke(nilCtx, fake, Request{Root: t.TempDir(), GoGetArg: "m@main"}); err == nil {
		t.Fatal("nil context")
	}
	if _, err := Invoke(context.Background(), nil, Request{Root: t.TempDir(), GoGetArg: "m@main"}); err == nil {
		t.Fatal("nil runner")
	}
	if len(fake.calls) != 0 {
		t.Fatal("invalid requests must not invoke the runner")
	}
}

func TestInvokeUsesExplicitGoBinaryAndDir(t *testing.T) {
	fake := &fakeRunner{}
	root := filepath.Join(t.TempDir(), "app")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := Invoke(context.Background(), fake, Request{
		Root:     root,
		GoGetArg: "github.com/zatrano/framework/v2@main",
		Go:       "/usr/local/go/bin/go",
	})
	if err != nil {
		t.Fatal(err)
	}
	got := fake.calls[0]
	if got.Path != "/usr/local/go/bin/go" {
		t.Fatalf("path=%q", got.Path)
	}
	if got.Dir != root {
		t.Fatalf("dir=%q want %q", got.Dir, root)
	}
}

func TestInvokeObservesExitAndStreams(t *testing.T) {
	fake := &fakeRunner{exitCode: 1, stdout: "out", stderr: "err-bytes"}
	res, err := Invoke(context.Background(), fake, Request{
		Root:     t.TempDir(),
		GoGetArg: "github.com/zatrano/packages@v1.4.0",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.ExitCode != 1 || res.Stdout != "out" || res.Stderr != "err-bytes" {
		t.Fatalf("%#v", res)
	}
}

func TestInvokeCancellationReachesRunner(t *testing.T) {
	started := make(chan struct{})
	wait := make(chan struct{})
	fake := &fakeRunner{started: started, wait: wait}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := Invoke(ctx, fake, Request{Root: t.TempDir(), GoGetArg: "github.com/zatrano/packages@main"})
		done <- err
	}()
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("runner was not invoked")
	}
	cancel()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected context error")
		}
	case <-time.After(2 * time.Second):
		close(wait)
		t.Fatal("Invoke did not observe cancellation")
	}
}

func TestInvokeDoesNotMutateGoMod(t *testing.T) {
	dir := t.TempDir()
	modPath := filepath.Join(dir, "go.mod")
	want := "module example.com/app\n\ngo 1.25.0\n"
	if err := os.WriteFile(modPath, []byte(want), 0o644); err != nil {
		t.Fatal(err)
	}
	sumPath := filepath.Join(dir, "go.sum")
	if err := os.WriteFile(sumPath, []byte("keep\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	fake := &fakeRunner{stdout: "ok"}
	if _, err := Invoke(context.Background(), fake, Request{
		Root:     dir,
		GoGetArg: "github.com/zatrano/packages@main",
	}); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(modPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Fatalf("go.mod mutated:\n%s", got)
	}
	sum, err := os.ReadFile(sumPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(sum) != "keep\n" {
		t.Fatalf("go.sum mutated:\n%s", sum)
	}
}

func TestInvokeDoesNotBuildAShellString(t *testing.T) {
	fake := &fakeRunner{}
	arg := "github.com/zatrano/packages@main"
	if _, err := Invoke(context.Background(), fake, Request{Root: t.TempDir(), GoGetArg: arg}); err != nil {
		t.Fatal(err)
	}
	got := fake.calls[0]
	joined := strings.Join(append([]string{got.Path}, got.Args...), " ")
	if strings.Contains(joined, "sh -c") || strings.Contains(got.Path, "sh") {
		t.Fatalf("shell interpolation: %#v", got)
	}
	if len(got.Args) != 2 {
		t.Fatalf("want argv {get, arg}, got %q", got.Args)
	}
}

type scriptedRunner struct {
	calls   []Invocation
	failArg string
	stderr  string
	err     error
}

func (s *scriptedRunner) Run(_ context.Context, inv Invocation) (InvocationResult, error) {
	s.calls = append(s.calls, inv)
	res := InvocationResult{Invocation: inv}
	if len(inv.Args) > 1 && inv.Args[1] == s.failArg {
		res.ExitCode = 1
		res.Stderr = s.stderr
		return res, s.err
	}
	return res, nil
}

func TestExecuteTargetsReportsPartialFailFast(t *testing.T) {
	a, b, c, d := "example.com/a@v1.0.0", "example.com/b@v1.0.0", "example.com/c@v1.0.0", "example.com/d@v1.0.0"
	fake := &scriptedRunner{failArg: c, stderr: "go get: module not found", err: errMutation}
	got, err := executeTargets(context.Background(), fake, Request{Root: t.TempDir()}, []string{a, b, c, d})
	if err == nil {
		t.Fatal("first failure must not be reported as success")
	}
	if err != errMutation {
		t.Fatalf("must preserve the mutation error, got %v", err)
	}
	if strings.Join(got.Successful(), ",") != a+","+b {
		t.Fatalf("successful=%q", got.Successful())
	}
	if strings.Join(got.Failed(), ",") != c {
		t.Fatalf("failed=%q", got.Failed())
	}
	if strings.Join(got.Unattempted(), ",") != d {
		t.Fatalf("unattempted=%q", got.Unattempted())
	}
	if len(got.Reports) != 4 {
		t.Fatalf("reports=%d — success/failed/unattempted must not collapse", len(got.Reports))
	}
	if got.Reports[0].Status != StatusSuccess || got.Reports[1].Status != StatusSuccess {
		t.Fatalf("A/B %#v", got.Reports[:2])
	}
	if got.Reports[2].Status != StatusFailed || got.Reports[2].Err == nil {
		t.Fatalf("C %#v", got.Reports[2])
	}
	if got.Reports[3].Status != StatusUnattempted || got.Reports[3].Err != nil {
		t.Fatalf("D %#v", got.Reports[3])
	}
	if len(fake.calls) != 3 {
		t.Fatalf("fail-fast must not start D: calls=%d", len(fake.calls))
	}
	if fake.calls[2].Args[1] != c {
		t.Fatalf("last attempted=%q", fake.calls[2].Args)
	}
}

func TestExecuteTargetsDoesNotStartAfterFailure(t *testing.T) {
	fake := &scriptedRunner{failArg: "example.com/c@v1.0.0", err: errMutation}
	_, err := executeTargets(context.Background(), fake, Request{Root: t.TempDir()}, []string{
		"example.com/a@v1.0.0",
		"example.com/c@v1.0.0",
		"example.com/d@v1.0.0",
	})
	if err == nil {
		t.Fatal("expected failure")
	}
	for _, call := range fake.calls {
		if len(call.Args) > 1 && call.Args[1] == "example.com/d@v1.0.0" {
			t.Fatal("new acquisition started after the first failure")
		}
	}
}

func TestExecuteTargetsDoesNotRollBackSuccesses(t *testing.T) {
	a, c := "example.com/a@v1.0.0", "example.com/c@v1.0.0"
	fake := &scriptedRunner{failArg: c, err: errMutation}
	got, err := executeTargets(context.Background(), fake, Request{Root: t.TempDir()}, []string{a, c})
	if err == nil {
		t.Fatal("expected failure")
	}
	if len(got.Successful()) != 1 || got.Successful()[0] != a {
		t.Fatalf("successful=%q", got.Successful())
	}
	for _, call := range fake.calls {
		joined := strings.Join(call.Args, " ")
		if strings.Contains(joined, "tidy") || strings.Contains(call.Path, "rollback") {
			t.Fatalf("rollback/tidy invocation: %#v", call)
		}
	}
	if len(fake.calls) != 2 {
		t.Fatalf("must not issue extra undo gets: calls=%d", len(fake.calls))
	}
}

func TestExecuteTargetsRejectsLatestWithoutStartingLaterTargets(t *testing.T) {
	fake := &scriptedRunner{}
	got, err := executeTargets(context.Background(), fake, Request{Root: t.TempDir()}, []string{
		"example.com/a@v1.0.0",
		"example.com/b@latest",
		"example.com/d@v1.0.0",
	})
	if err == nil {
		t.Fatal("latest must fail")
	}
	if strings.Join(got.Successful(), ",") != "example.com/a@v1.0.0" {
		t.Fatalf("successful=%q", got.Successful())
	}
	if strings.Join(got.Failed(), ",") != "example.com/b@latest" {
		t.Fatalf("failed=%q", got.Failed())
	}
	if strings.Join(got.Unattempted(), ",") != "example.com/d@v1.0.0" {
		t.Fatalf("unattempted=%q", got.Unattempted())
	}
	if len(fake.calls) != 1 {
		t.Fatalf("later targets must not run: %+v", fake.calls)
	}
}

func TestExecuteTargetsEmptyIsNotAllAcquired(t *testing.T) {
	fake := &scriptedRunner{}
	got, err := executeTargets(context.Background(), fake, Request{Root: t.TempDir()}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Successful()) != 0 || len(got.Failed()) != 0 || len(got.Unattempted()) != 0 {
		t.Fatalf("empty sequence must not claim targets: %#v", got)
	}
	if len(fake.calls) != 0 {
		t.Fatal("empty sequence must not invoke go get")
	}
}
