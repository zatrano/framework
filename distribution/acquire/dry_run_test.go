package acquire

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/zatrano/framework/v2/distribution/registry"
)

func TestDryRunConsumesPlanGoGetArg(t *testing.T) {
	plan, err := FromResult(registry.Result{
		Package: registry.Package{Name: "session", Module: "github.com/zatrano/packages"},
		Release: registry.Release{Channel: registry.ChannelMain},
	})
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	rep, err := DryRun(Request{Root: root, GoGetArg: plan.GoGetArg()})
	if err != nil {
		t.Fatal(err)
	}
	if rep.GoGetArg != plan.GoGetArg() {
		t.Fatalf("GoGetArg=%q want %q", rep.GoGetArg, plan.GoGetArg())
	}
	if rep.Module != "github.com/zatrano/packages" || rep.Selected != "main" {
		t.Fatalf("module=%q selected=%q", rep.Module, rep.Selected)
	}
	if rep.Root != root {
		t.Fatalf("root=%q", rep.Root)
	}
	if len(rep.Command) != 3 || rep.Command[0] != "go" || rep.Command[1] != "get" || rep.Command[2] != plan.GoGetArg() {
		t.Fatalf("command=%q", rep.Command)
	}
}

func TestDryRunTargetsUsesTargetsOutput(t *testing.T) {
	session, err := FromResult(registry.Result{
		Package: registry.Package{Name: "session", Module: "github.com/zatrano/packages"},
		Release: registry.Release{Channel: registry.ChannelMain},
	})
	if err != nil {
		t.Fatal(err)
	}
	auth, err := FromResult(registry.Result{
		Package: registry.Package{Name: "auth", Module: "github.com/zatrano/packages"},
		Release: registry.Release{Channel: registry.ChannelMain},
	})
	if err != nil {
		t.Fatal(err)
	}
	args, err := Targets([]Plan{session, auth})
	if err != nil {
		t.Fatal(err)
	}
	if len(args) != 1 {
		t.Fatalf("targets=%d — shared module must collapse", len(args))
	}
	reps, err := DryRunTargets(t.TempDir(), args)
	if err != nil {
		t.Fatal(err)
	}
	if len(reps) != 1 || reps[0].GoGetArg != args[0] {
		t.Fatalf("reports=%v", reps)
	}
}

func TestDryRunRejectsLatest(t *testing.T) {
	root := t.TempDir()
	if _, err := DryRun(Request{Root: root, GoGetArg: "latest"}); err == nil {
		t.Fatal("latest must fail")
	}
	if _, err := DryRun(Request{Root: root, GoGetArg: "github.com/zatrano/packages@latest"}); err == nil {
		t.Fatal("@latest must fail")
	}
}

func TestDryRunRequiresRootAndArg(t *testing.T) {
	if _, err := DryRun(Request{GoGetArg: "github.com/zatrano/packages@main"}); err == nil {
		t.Fatal("empty root must fail")
	}
	if _, err := DryRun(Request{Root: t.TempDir()}); err == nil {
		t.Fatal("empty arg must fail")
	}
}

func TestDryRunDoesNotMutateModuleFiles(t *testing.T) {
	root := t.TempDir()
	modPath := filepath.Join(root, "go.mod")
	sumPath := filepath.Join(root, "go.sum")
	mod := []byte("module example.com/app\n\ngo 1.25\n")
	sum := []byte("example.com/dep v1.0.0 h1:abc\n")
	if err := os.WriteFile(modPath, mod, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sumPath, sum, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := DryRun(Request{Root: root, GoGetArg: "github.com/zatrano/packages@main"}); err != nil {
		t.Fatal(err)
	}
	gotMod, err := os.ReadFile(modPath)
	if err != nil {
		t.Fatal(err)
	}
	gotSum, err := os.ReadFile(sumPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(gotMod) != string(mod) || string(gotSum) != string(sum) {
		t.Fatal("dry-run must not write go.mod / go.sum")
	}
}

func TestDryRunReportHasNoMutationStatus(t *testing.T) {
	rt := reflect.TypeOf(DryRunReport{})
	for i := 0; i < rt.NumField(); i++ {
		name := strings.ToLower(rt.Field(i).Name)
		for _, ban := range []string{"installed", "acquired", "applied", "status", "success", "failed"} {
			if name == ban || strings.Contains(name, ban) {
				t.Errorf("DryRunReport field %s implies mutation", rt.Field(i).Name)
			}
		}
	}
	rep, err := DryRun(Request{Root: t.TempDir(), GoGetArg: "github.com/zatrano/packages@main"})
	if err != nil {
		t.Fatal(err)
	}
	dump := strings.ToLower(rep.Root + " " + rep.Module + " " + rep.Selected + " " + strings.Join(rep.Command, " "))
	for _, ban := range []string{"installed", "acquired", "applied"} {
		if strings.Contains(dump, ban) {
			t.Errorf("dry-run output contains %q", ban)
		}
	}
}

func TestDryRunUsesExplicitGoBinaryInCommandForm(t *testing.T) {
	rep, err := DryRun(Request{
		Root:     t.TempDir(),
		GoGetArg: "github.com/zatrano/packages@v1.4.0",
		Go:       "/opt/go/bin/go",
	})
	if err != nil {
		t.Fatal(err)
	}
	if rep.Selected != "v1.4.0" {
		t.Fatalf("selected=%q", rep.Selected)
	}
	if len(rep.Command) != 3 || rep.Command[0] != "/opt/go/bin/go" || rep.Command[1] != "get" {
		t.Fatalf("command=%q", rep.Command)
	}
}
