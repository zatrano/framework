package pkgmanager

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zatrano/framework/v3/core/distribution/acquire"
	"github.com/zatrano/framework/v3/core/distribution/manifest"
	"github.com/zatrano/framework/v3/core/distribution/registry"
)

func TestPackageAcquireDryRunUsesPlanGoGetArg(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/app\n\ngo 1.25\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	executed := false
	var buf bytes.Buffer
	cmd := &PackageAcquireCommand{
		out:  &buf,
		root: root,
		executeTargets: func(context.Context, string, []string) (acquire.ApplyResult, error) {
			executed = true
			return acquire.ApplyResult{}, errors.New("execute must not run on dry-run")
		},
	}
	if err := cmd.Handle([]string{"session", "--dry-run", "--format=json"}); err != nil {
		t.Fatal(err)
	}
	if executed {
		t.Fatal("dry-run must not call ExecuteTargets")
	}
	var report acquireCLIReport
	if err := json.Unmarshal(buf.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if report.Mode != "dry-run" || report.Enabled {
		t.Fatalf("%#v", report)
	}
	want := manifest.DefaultModule + "@main"
	if len(report.GoGetArgs) != 1 || report.GoGetArgs[0] != want {
		t.Fatalf("go_get_args=%v want %s", report.GoGetArgs, want)
	}
	if len(report.DryRun) != 1 || report.DryRun[0].GoGetArg != want {
		t.Fatalf("dry_run=%#v", report.DryRun)
	}
	rawKeys := jsonTopKeys(t, buf.Bytes())
	for _, ban := range []string{"installed", "acquired", "applied"} {
		if rawKeys[ban] {
			t.Fatalf("dry-run JSON must not include %q", ban)
		}
	}
	gotMod, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(gotMod), "module example.com/app") {
		t.Fatal("dry-run must not rewrite go.mod")
	}
}

func TestPackageAcquireExecuteDelegatesConcreteGoGetArg(t *testing.T) {
	root := t.TempDir()
	var gotArgs []string
	var buf bytes.Buffer
	cmd := &PackageAcquireCommand{
		out:  &buf,
		root: root,
		snapshotFiles: func(string) (acquire.FileSnapshot, error) {
			return acquire.FileSnapshot{Root: root}, nil
		},
		inspect: func(string) (acquire.Inspection, error) {
			return acquire.Inspection{Root: root}, nil
		},
		executeTargets: func(_ context.Context, gotRoot string, args []string) (acquire.ApplyResult, error) {
			if gotRoot != root {
				t.Fatalf("root=%q", gotRoot)
			}
			gotArgs = append([]string(nil), args...)
			reps := make([]acquire.TargetReport, len(args))
			for i, arg := range args {
				reps[i] = acquire.TargetReport{GoGetArg: arg, Status: acquire.StatusSuccess}
			}
			return acquire.ApplyResult{Root: gotRoot, Reports: reps, Recovery: acquire.Recovery{Kind: acquire.RecoveryUnavailable}}, nil
		},
	}
	if err := cmd.Handle([]string{"session", "--format=json"}); err != nil {
		t.Fatal(err)
	}
	want := manifest.DefaultModule + "@main"
	if len(gotArgs) != 1 || gotArgs[0] != want {
		t.Fatalf("delegated args=%v want %s", gotArgs, want)
	}
	var report acquireCLIReport
	if err := json.Unmarshal(buf.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if report.Enabled || report.Mode != "execute" {
		t.Fatalf("%#v", report)
	}
	if len(report.Successful) != 1 || report.Successful[0] != want {
		t.Fatalf("successful=%v", report.Successful)
	}
	if report.Recovery != string(acquire.RecoveryUnavailable) {
		t.Fatalf("recovery=%q", report.Recovery)
	}
}

func TestPackageAcquirePresentsPartialResultWithoutInventingSuccess(t *testing.T) {
	root := t.TempDir()
	var buf bytes.Buffer
	cmd := &PackageAcquireCommand{
		out:  &buf,
		root: root,
		snapshotFiles: func(string) (acquire.FileSnapshot, error) {
			return acquire.FileSnapshot{Root: root}, nil
		},
		recoverFiles: func(context.Context, acquire.FileSnapshot) (acquire.Recovery, error) {
			return acquire.Recovery{Kind: acquire.RecoveryFiles}, nil
		},
		inspect: func(string) (acquire.Inspection, error) {
			return acquire.Inspection{Root: root}, nil
		},
		executeTargets: func(_ context.Context, gotRoot string, args []string) (acquire.ApplyResult, error) {
			arg := args[0]
			return acquire.ApplyResult{
				Root: gotRoot,
				Reports: []acquire.TargetReport{
					{GoGetArg: arg, Status: acquire.StatusSuccess},
					{GoGetArg: "example.com/c@main", Status: acquire.StatusFailed, Err: errors.New("proxy denied")},
					{GoGetArg: "example.com/d@main", Status: acquire.StatusUnattempted},
				},
				Recovery: acquire.Recovery{Kind: acquire.RecoveryUnavailable},
			}, errors.New("proxy denied")
		},
	}
	err := cmd.Handle([]string{"session", "--format=json"})
	if err == nil {
		t.Fatal("expected execute error")
	}
	var report acquireCLIReport
	if err := json.Unmarshal(buf.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if report.Enabled {
		t.Fatal("acquisition must not enable")
	}
	if len(report.Successful) != 1 || len(report.Failed) != 1 || len(report.Unattempted) != 1 {
		t.Fatalf("partial %#v", report)
	}
	if report.Unattempted[0] != "example.com/d@main" {
		t.Fatalf("unattempted=%v", report.Unattempted)
	}
	if report.Recovery != string(acquire.RecoveryFiles) {
		t.Fatalf("recovery must stay distinct from success: %q", report.Recovery)
	}
	raw := buf.String()
	if strings.Contains(strings.ToLower(raw), "all targets acquired") {
		t.Fatal("must not invent all-acquired")
	}
}

func TestPackageAcquireDoesNotWriteEnablement(t *testing.T) {
	root := t.TempDir()
	cmd := &PackageAcquireCommand{
		out:  ioDiscard(),
		root: root,
		snapshotFiles: func(string) (acquire.FileSnapshot, error) {
			return acquire.FileSnapshot{Root: root}, nil
		},
		inspect: func(string) (acquire.Inspection, error) {
			return acquire.Inspection{Root: root}, nil
		},
		executeTargets: func(_ context.Context, gotRoot string, args []string) (acquire.ApplyResult, error) {
			return acquire.ApplyResult{
				Root:     gotRoot,
				Reports:  []acquire.TargetReport{{GoGetArg: args[0], Status: acquire.StatusSuccess}},
				Recovery: acquire.Recovery{Kind: acquire.RecoveryUnavailable},
			}, nil
		},
	}
	if err := cmd.Handle([]string{"session"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "bootstrap", "enabled.go")); err == nil {
		t.Fatal("package:acquire must not write bootstrap/enabled.go")
	}
}

func TestPackageInstallRemainsEnablement(t *testing.T) {
	src, err := os.ReadFile("package_cmd.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(src)
	if !strings.Contains(text, `return "package:install"`) {
		t.Fatal("package:install missing")
	}
	if strings.Contains(text, "distribution/acquire") {
		t.Fatal("package:install must not import acquire")
	}
	if !strings.Contains(text, "enablePackage(") {
		t.Fatal("package:install stays enablement")
	}
}

func TestPackageAcquireUnknownPackage(t *testing.T) {
	cmd := &PackageAcquireCommand{out: ioDiscard(), root: t.TempDir()}
	if err := cmd.Handle([]string{"does-not-exist"}); err == nil {
		t.Fatal("expected resolve error")
	}
}

func TestPackageAcquireUsesInjectedIndexResolve(t *testing.T) {
	idx := registry.Index{
		Schema: registry.SchemaV1,
		Packages: []registry.Package{{
			Name: "session", Import: "github.com/zatrano/packages/session",
			Module: manifest.DefaultModule, Kind: manifest.KindService,
			Releases: []registry.Release{{Channel: registry.ChannelMain}},
		}},
	}
	root := t.TempDir()
	var buf bytes.Buffer
	cmd := &PackageAcquireCommand{out: &buf, root: root, index: &idx}
	if err := cmd.Handle([]string{"session", "--dry-run", "--format=json"}); err != nil {
		t.Fatal(err)
	}
	var report acquireCLIReport
	if err := json.Unmarshal(buf.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if report.GoGetArgs[0] != manifest.DefaultModule+"@main" {
		t.Fatalf("%v", report.GoGetArgs)
	}
}

func successfulExecute() func(context.Context, string, []string) (acquire.ApplyResult, error) {
	return func(_ context.Context, gotRoot string, args []string) (acquire.ApplyResult, error) {
		return acquire.ApplyResult{
			Root:     gotRoot,
			Reports:  []acquire.TargetReport{{GoGetArg: args[0], Status: acquire.StatusSuccess}},
			Recovery: acquire.Recovery{Kind: acquire.RecoveryUnavailable},
		}, nil
	}
}

func TestPackageAcquireEnablementNotRequestedByDefault(t *testing.T) {
	root := t.TempDir()
	called := false
	var buf bytes.Buffer
	cmd := &PackageAcquireCommand{
		out:  &buf,
		root: root,
		snapshotFiles: func(string) (acquire.FileSnapshot, error) {
			return acquire.FileSnapshot{Root: root}, nil
		},
		inspect:        func(string) (acquire.Inspection, error) { return acquire.Inspection{Root: root}, nil },
		executeTargets: successfulExecute(),
		enableFn: func(string) (bool, error) {
			called = true
			return true, nil
		},
	}
	if err := cmd.Handle([]string{"session", "--format=json"}); err != nil {
		t.Fatal(err)
	}
	if called {
		t.Fatal("default acquire must not enable")
	}
	var report acquireCLIReport
	if err := json.Unmarshal(buf.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if report.Acquisition != "success" || report.Enablement != "not_requested" || report.Enabled {
		t.Fatalf("%#v", report)
	}
}

func TestPackageAcquireExplicitEnableAfterSuccess(t *testing.T) {
	root := t.TempDir()
	var gotName string
	recovered := false
	var buf bytes.Buffer
	cmd := &PackageAcquireCommand{
		out:  &buf,
		root: root,
		snapshotFiles: func(string) (acquire.FileSnapshot, error) {
			return acquire.FileSnapshot{Root: root}, nil
		},
		recoverFiles: func(context.Context, acquire.FileSnapshot) (acquire.Recovery, error) {
			recovered = true
			return acquire.Recovery{Kind: acquire.RecoveryFiles}, nil
		},
		inspect:        func(string) (acquire.Inspection, error) { return acquire.Inspection{Root: root}, nil },
		executeTargets: successfulExecute(),
		enableFn: func(name string) (bool, error) {
			gotName = name
			return true, nil
		},
	}
	if err := cmd.Handle([]string{"session", "--enable", "--format=json"}); err != nil {
		t.Fatal(err)
	}
	if recovered {
		t.Fatal("successful acquire+enable must not recover")
	}
	if gotName != "session" {
		t.Fatalf("enable name=%q", gotName)
	}
	var report acquireCLIReport
	if err := json.Unmarshal(buf.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if report.Acquisition != "success" || report.Enablement != "success" || !report.Enabled {
		t.Fatalf("%#v", report)
	}
}

func TestPackageAcquireDoesNotEnableWhenAcquisitionFails(t *testing.T) {
	root := t.TempDir()
	called := false
	var buf bytes.Buffer
	cmd := &PackageAcquireCommand{
		out:  &buf,
		root: root,
		snapshotFiles: func(string) (acquire.FileSnapshot, error) {
			return acquire.FileSnapshot{Root: root}, nil
		},
		recoverFiles: func(context.Context, acquire.FileSnapshot) (acquire.Recovery, error) {
			return acquire.Recovery{Kind: acquire.RecoveryFiles}, nil
		},
		inspect: func(string) (acquire.Inspection, error) { return acquire.Inspection{Root: root}, nil },
		executeTargets: func(_ context.Context, gotRoot string, args []string) (acquire.ApplyResult, error) {
			return acquire.ApplyResult{
				Root:     gotRoot,
				Reports:  []acquire.TargetReport{{GoGetArg: args[0], Status: acquire.StatusFailed, Err: errors.New("proxy denied")}},
				Recovery: acquire.Recovery{Kind: acquire.RecoveryUnavailable},
			}, errors.New("proxy denied")
		},
		enableFn: func(string) (bool, error) {
			called = true
			return true, nil
		},
	}
	if err := cmd.Handle([]string{"session", "--enable", "--format=json"}); err == nil {
		t.Fatal("expected acquisition error")
	}
	if called {
		t.Fatal("acquisition failure must not start enablement")
	}
	var report acquireCLIReport
	if err := json.Unmarshal(buf.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if report.Acquisition != "failed" || report.Enablement != "not_requested" || report.Enabled {
		t.Fatalf("%#v", report)
	}
}

func TestPackageAcquireEnablementFailureDoesNotRollbackAcquisition(t *testing.T) {
	root := t.TempDir()
	recovered := false
	var buf bytes.Buffer
	cmd := &PackageAcquireCommand{
		out:  &buf,
		root: root,
		snapshotFiles: func(string) (acquire.FileSnapshot, error) {
			return acquire.FileSnapshot{Root: root}, nil
		},
		recoverFiles: func(context.Context, acquire.FileSnapshot) (acquire.Recovery, error) {
			recovered = true
			return acquire.Recovery{Kind: acquire.RecoveryFiles}, nil
		},
		inspect:        func(string) (acquire.Inspection, error) { return acquire.Inspection{Root: root}, nil },
		executeTargets: successfulExecute(),
		enableFn: func(string) (bool, error) {
			return false, errors.New("bootstrap write failed")
		},
	}
	err := cmd.Handle([]string{"session", "--enable", "--format=json"})
	if err == nil || CodeFromError(err) != ExitEnablement || !strings.Contains(err.Error(), "bootstrap write failed") {
		t.Fatalf("enablement error: %v", err)
	}
	if recovered {
		t.Fatal("enablement failure must not recover/rollback acquisition")
	}
	var report acquireCLIReport
	if err := json.Unmarshal(buf.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if report.Acquisition != "success" || report.Enablement != "failed" || report.Enabled {
		t.Fatalf("must not collapse into acquisition failure: %#v", report)
	}
	if len(report.Successful) != 1 {
		t.Fatalf("acquisition success must remain visible: %#v", report)
	}
}

func TestPackageAcquireDryRunDoesNotEnableEvenWithFlag(t *testing.T) {
	root := t.TempDir()
	called := false
	executed := false
	var buf bytes.Buffer
	cmd := &PackageAcquireCommand{
		out:  &buf,
		root: root,
		executeTargets: func(context.Context, string, []string) (acquire.ApplyResult, error) {
			executed = true
			return acquire.ApplyResult{}, errors.New("execute must not run")
		},
		enableFn: func(string) (bool, error) {
			called = true
			return true, nil
		},
	}
	if err := cmd.Handle([]string{"session", "--dry-run", "--enable", "--format=json"}); err != nil {
		t.Fatal(err)
	}
	if executed || called {
		t.Fatal("dry-run must not execute or enable")
	}
	var report acquireCLIReport
	if err := json.Unmarshal(buf.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if report.Mode != "dry-run" || report.Acquisition != "not_executed" || report.Enablement != "not_requested" || report.Enabled {
		t.Fatalf("%#v", report)
	}
}
