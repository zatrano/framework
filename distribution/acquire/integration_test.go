package acquire

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zatrano/framework/v2/distribution/registry"
)

func mustTaggedPlan(t *testing.T, name, module, version string) Plan {
	t.Helper()
	p, err := FromResult(registry.Result{
		Package: registry.Package{Name: name, Module: module},
		Release: registry.Release{Version: version},
	})
	if err != nil {
		t.Fatal(err)
	}
	arg := p.GoGetArg()
	if arg == "" || strings.Contains(strings.ToLower(arg), "latest") {
		t.Fatalf("GoGetArg=%q", arg)
	}
	return p
}

func writeIntegrationApp(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	app := filepath.Join(root, "app")
	if err := os.Mkdir(app, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"a", "b", "d"} {
		dir := filepath.Join(root, name)
		if err := os.Mkdir(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/"+name+"\n\ngo 1.25.0\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name+".go"), []byte("package "+name+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	mod := "module example.com/app\n\ngo 1.25.0\n\nreplace example.com/a => ../a\nreplace example.com/b => ../b\nreplace example.com/d => ../d\n"
	if err := os.WriteFile(filepath.Join(app, "go.mod"), []byte(mod), 0o644); err != nil {
		t.Fatal(err)
	}
	return app
}

func TestAcquireIntegrationPlanExecuteInspect(t *testing.T) {
	requireGo(t)
	offlineGoEnv(t)
	app, arg := writeLocalApp(t)
	plan := mustTaggedPlan(t, "dep", "example.com/dep", "0.0.0")
	if plan.GoGetArg() != arg {
		t.Fatalf("GoGetArg=%q want %q", plan.GoGetArg(), arg)
	}
	args, err := Targets([]Plan{plan})
	if err != nil {
		t.Fatal(err)
	}
	if len(args) != 1 || args[0] != arg {
		t.Fatalf("Targets=%q", args)
	}
	res, err := Execute(context.Background(), Request{Root: app, GoGetArg: plan.GoGetArg()})
	if err != nil {
		t.Fatalf("Execute: %v\n%s", err, res.Stderr)
	}
	assertGetArgv(t, res, app, arg)
	in, err := Inspect(app)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := in.Requirement("example.com/dep"); !ok {
		t.Fatal("Inspect must observe the require Go wrote")
	}
	if _, err := os.Stat(filepath.Join(app, "zatrano.lock")); !os.IsNotExist(err) {
		t.Fatal("acquisition must not create zatrano.lock")
	}
}

func TestAcquireIntegrationTargetsCollapseSharedModule(t *testing.T) {
	requireGo(t)
	offlineGoEnv(t)
	app, arg := writeLocalApp(t)
	session := mustTaggedPlan(t, "session", "example.com/dep", "0.0.0")
	auth := mustTaggedPlan(t, "auth", "example.com/dep", "0.0.0")
	args, err := Targets([]Plan{session, auth})
	if err != nil {
		t.Fatal(err)
	}
	if len(args) != 1 || args[0] != arg {
		t.Fatalf("shared module must collapse to one GoGetArg, got %q", args)
	}
	got, err := ExecuteTargets(context.Background(), app, args)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Successful()) != 1 || got.Successful()[0] != arg {
		t.Fatalf("successful=%q", got.Successful())
	}
	if got.Recovery.Kind != RecoveryUnavailable {
		t.Fatalf("ExecuteTargets must not restore files: %#v", got.Recovery)
	}
	in, err := Inspect(app)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := in.Requirement("example.com/dep"); !ok {
		t.Fatal("one require after collapsed targets")
	}
}

func TestAcquireIntegrationFailFastPartialThenRecover(t *testing.T) {
	requireGo(t)
	offlineGoEnv(t)
	app := writeIntegrationApp(t)
	plans := []Plan{
		mustTaggedPlan(t, "alpha", "example.com/a", "0.0.0"),
		mustTaggedPlan(t, "beta", "example.com/b", "0.0.0"),
		mustTaggedPlan(t, "gamma", "example.com/c", "0.0.0"),
		mustTaggedPlan(t, "delta", "example.com/d", "0.0.0"),
	}
	args, err := Targets(plans)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		plans[0].GoGetArg(),
		plans[1].GoGetArg(),
		plans[2].GoGetArg(),
		plans[3].GoGetArg(),
	}
	if strings.Join(args, ",") != strings.Join(want, ",") {
		t.Fatalf("Targets order=%q want %q", args, want)
	}
	before, err := SnapshotFiles(app)
	if err != nil {
		t.Fatal(err)
	}
	cache := t.TempDir()
	t.Setenv("GOMODCACHE", cache)
	sentinel := filepath.Join(cache, "not-undone")
	if err := os.WriteFile(sentinel, []byte("cache\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, acqErr := ExecuteTargets(context.Background(), app, args)
	if acqErr == nil {
		t.Fatal("C must fail")
	}
	if strings.Join(got.Successful(), ",") != want[0]+","+want[1] {
		t.Fatalf("successful=%q", got.Successful())
	}
	if strings.Join(got.Failed(), ",") != want[2] {
		t.Fatalf("failed=%q", got.Failed())
	}
	if strings.Join(got.Unattempted(), ",") != want[3] {
		t.Fatalf("unattempted=%q", got.Unattempted())
	}
	if got.Recovery.Kind != RecoveryUnavailable {
		t.Fatalf("ExecuteTargets restored files: %#v", got.Recovery)
	}
	for _, rep := range got.Reports {
		joined := strings.Join(rep.Result.Invocation.Args, " ")
		if strings.Contains(joined, "tidy") {
			t.Fatalf("tidy leaked into argv: %q", rep.Result.Invocation.Args)
		}
	}
	in, err := Inspect(app)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := in.Requirement("example.com/a"); !ok {
		t.Fatal("A missing after partial apply")
	}
	if _, ok := in.Requirement("example.com/d"); ok {
		t.Fatal("D must remain unattempted")
	}
	rec, recErr := RecoverFiles(context.Background(), before)
	if recErr != nil || rec.Kind != RecoveryFiles {
		t.Fatalf("rec=%#v err=%v", rec, recErr)
	}
	got = got.WithRecovery(rec)
	if strings.Join(got.Successful(), ",") != want[0]+","+want[1] || strings.Join(got.Failed(), ",") != want[2] || strings.Join(got.Unattempted(), ",") != want[3] {
		t.Fatalf("recovery rewrote reports: success=%q failed=%q unattempted=%q", got.Successful(), got.Failed(), got.Unattempted())
	}
	in, err = Inspect(app)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := in.Requirement("example.com/a"); ok {
		t.Fatal("RecoverFiles must restore go.mod so A is gone")
	}
	if _, err := os.Stat(sentinel); err != nil {
		t.Fatal("file recovery must not undo the module cache")
	}
	if _, err := os.Stat(filepath.Join(app, "zatrano.lock")); !os.IsNotExist(err) {
		t.Fatal("recovery must not create zatrano.lock")
	}
}

func TestAcquireIntegrationConflictingPinsDoNotMutate(t *testing.T) {
	app := t.TempDir()
	mod := "module example.com/app\n\ngo 1.25.0\n"
	if err := os.WriteFile(filepath.Join(app, "go.mod"), []byte(mod), 0o644); err != nil {
		t.Fatal(err)
	}
	session := mustTaggedPlan(t, "session", "example.com/shared", "0.1.0")
	auth := mustTaggedPlan(t, "auth", "example.com/shared", "1.0.0")
	_, err := Targets([]Plan{session, auth})
	if err == nil {
		t.Fatal("conflicting pins must fail before Apply")
	}
	got, err := os.ReadFile(filepath.Join(app, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != mod {
		t.Fatalf("conflict mutated go.mod:\n%s", got)
	}
}
