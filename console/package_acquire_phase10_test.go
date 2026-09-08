package console

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/zatrano/framework/v2/distribution/acquire"
	"github.com/zatrano/framework/v2/distribution/manifest"
	"github.com/zatrano/framework/v2/distribution/registry"
)

func fakeAcquireCLI(root string, execFn func(context.Context, string, []string) (acquire.ApplyResult, error)) *PackageAcquireCommand {
	return &PackageAcquireCommand{
		out:  ioDiscard(),
		root: root,
		snapshotFiles: func(string) (acquire.FileSnapshot, error) {
			return acquire.FileSnapshot{Root: root}, nil
		},
		inspect: func(string) (acquire.Inspection, error) {
			return acquire.Inspection{Root: root, Module: "example.com/app"}, nil
		},
		executeTargets: execFn,
	}
}

func TestPackageAcquireUnknownIsResolutionFailure(t *testing.T) {
	err := (&PackageAcquireCommand{out: ioDiscard(), root: t.TempDir()}).Handle([]string{"does-not-exist"})
	if err == nil {
		t.Fatal("expected resolution failure")
	}
	if CodeFromError(err) != ExitResolution {
		t.Fatalf("exit=%d want %d (%v)", CodeFromError(err), ExitResolution, err)
	}
}

func TestPackageAcquireIncompatibleIsResolutionFailure(t *testing.T) {
	idx := registry.Index{
		Schema: registry.SchemaV1,
		Packages: []registry.Package{{
			Name: "session", Module: manifest.DefaultModule, Kind: manifest.KindService,
			Releases: []registry.Release{{Channel: registry.ChannelMain, FrameworkMin: "9.9.9"}},
		}},
	}
	err := (&PackageAcquireCommand{out: ioDiscard(), root: t.TempDir(), index: &idx}).Handle([]string{"session", "--framework=2.0.0"})
	if err == nil {
		t.Fatal("expected incompatible resolution failure")
	}
	if CodeFromError(err) != ExitResolution {
		t.Fatalf("exit=%d want %d (%v)", CodeFromError(err), ExitResolution, err)
	}
}

func TestPackageAcquireMissingModuleIsPlanningFailure(t *testing.T) {
	idx := registry.Index{
		Schema: registry.SchemaV1,
		Packages: []registry.Package{{
			Name: "broken", Kind: manifest.KindService,
			Releases: []registry.Release{{Channel: registry.ChannelMain}},
		}},
	}
	err := (&PackageAcquireCommand{out: ioDiscard(), root: t.TempDir(), index: &idx}).Handle([]string{"broken"})
	if err == nil {
		t.Fatal("expected planning failure")
	}
	if CodeFromError(err) != ExitPlanning {
		t.Fatalf("exit=%d want %d (%v)", CodeFromError(err), ExitPlanning, err)
	}
}

func TestPackageAcquireGoGetFailureIsAcquisitionFailure(t *testing.T) {
	root := t.TempDir()
	var buf bytes.Buffer
	cmd := fakeAcquireCLI(root, func(_ context.Context, gotRoot string, args []string) (acquire.ApplyResult, error) {
		return acquire.ApplyResult{
			Root:     gotRoot,
			Reports:  []acquire.TargetReport{{GoGetArg: args[0], Status: acquire.StatusFailed, Err: errors.New("proxy denied")}},
			Recovery: acquire.Recovery{Kind: acquire.RecoveryUnavailable},
		}, errors.New("proxy denied")
	})
	cmd.out = &buf
	err := cmd.Handle([]string{"session", "--format=json"})
	if err == nil || CodeFromError(err) != ExitAcquisition {
		t.Fatalf("exit=%d err=%v", CodeFromError(err), err)
	}
	var view acquireCLIView
	if err := json.Unmarshal(buf.Bytes(), &view); err != nil {
		t.Fatal(err)
	}
	if view.Acquisition != acquireStatusFailed || view.Enabled {
		t.Fatalf("%#v", view)
	}
	if view.Inspection == nil {
		t.Fatal("Inspect must not be discarded")
	}
}

func TestPackageAcquireSnapshotFailureIsDistinctFromRecovery(t *testing.T) {
	root := t.TempDir()
	var buf bytes.Buffer
	recovered := false
	cmd := fakeAcquireCLI(root, successfulExecute())
	cmd.out = &buf
	cmd.snapshotFiles = func(string) (acquire.FileSnapshot, error) {
		return acquire.FileSnapshot{}, errors.New("go.mod unreadable")
	}
	cmd.recoverFiles = func(context.Context, acquire.FileSnapshot) (acquire.Recovery, error) {
		recovered = true
		return acquire.Recovery{Kind: acquire.RecoveryFiles}, nil
	}
	if err := cmd.Handle([]string{"session", "--format=json"}); err != nil {
		t.Fatal(err)
	}
	if recovered {
		t.Fatal("snapshot failure must not attempt recovery")
	}
	var view acquireCLIView
	if err := json.Unmarshal(buf.Bytes(), &view); err != nil {
		t.Fatal(err)
	}
	if view.SnapshotError == "" || view.RecoveryError != "" {
		t.Fatalf("snapshot vs recovery: %#v", view)
	}
	if view.Recovery != string(acquire.RecoveryUnavailable) {
		t.Fatalf("recovery=%q", view.Recovery)
	}
}

func TestPackageAcquireRecoveryFailureIsReported(t *testing.T) {
	root := t.TempDir()
	var buf bytes.Buffer
	cmd := fakeAcquireCLI(root, func(_ context.Context, gotRoot string, args []string) (acquire.ApplyResult, error) {
		return acquire.ApplyResult{
			Root:     gotRoot,
			Reports:  []acquire.TargetReport{{GoGetArg: args[0], Status: acquire.StatusFailed, Err: errors.New("proxy denied")}},
			Recovery: acquire.Recovery{Kind: acquire.RecoveryUnavailable},
		}, errors.New("proxy denied")
	})
	cmd.out = &buf
	cmd.recoverFiles = func(context.Context, acquire.FileSnapshot) (acquire.Recovery, error) {
		return acquire.Recovery{Kind: acquire.RecoveryFailed}, errors.New("restore denied")
	}
	err := cmd.Handle([]string{"session", "--format=json"})
	if err == nil || CodeFromError(err) != ExitAcquisition {
		t.Fatalf("acquisition remains the primary failure: %v", err)
	}
	var view acquireCLIView
	if err := json.Unmarshal(buf.Bytes(), &view); err != nil {
		t.Fatal(err)
	}
	if view.Recovery != string(acquire.RecoveryFailed) || view.RecoveryError == "" {
		t.Fatalf("recovery failure must be observable: %#v", view)
	}
	if view.Acquisition != acquireStatusFailed {
		t.Fatalf("%#v", view)
	}
	found := false
	for _, msg := range view.Errors {
		if strings.Contains(msg, "restore denied") {
			found = true
		}
	}
	if !found {
		t.Fatalf("errors=%v", view.Errors)
	}
}

func TestPackageAcquireJSONPreservesTargetStatusesAndInspection(t *testing.T) {
	root := t.TempDir()
	var buf bytes.Buffer
	cmd := fakeAcquireCLI(root, func(_ context.Context, gotRoot string, args []string) (acquire.ApplyResult, error) {
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
	})
	cmd.out = &buf
	cmd.recoverFiles = func(context.Context, acquire.FileSnapshot) (acquire.Recovery, error) {
		return acquire.Recovery{Kind: acquire.RecoveryFiles}, nil
	}
	if err := cmd.Handle([]string{"session", "--format=json"}); err == nil {
		t.Fatal("expected acquisition error")
	}
	var view acquireCLIView
	if err := json.Unmarshal(buf.Bytes(), &view); err != nil {
		t.Fatal(err)
	}
	if len(view.Targets) != 3 {
		t.Fatalf("targets=%#v", view.Targets)
	}
	if view.Targets[0].Status != string(acquire.StatusSuccess) || view.Targets[1].Status != string(acquire.StatusFailed) || view.Targets[2].Status != string(acquire.StatusUnattempted) {
		t.Fatalf("target statuses=%#v", view.Targets)
	}
	if view.Inspection == nil || view.Inspection.Module != "example.com/app" {
		t.Fatalf("inspection discarded: %#v", view.Inspection)
	}
	if view.Acquisition == acquireStatusSuccess {
		t.Fatal("partial acquisition must not be complete success")
	}
}

func TestPackageAcquireTimeoutIsCanceled(t *testing.T) {
	root := t.TempDir()
	var buf bytes.Buffer
	var sawTimeout bool
	cmd := fakeAcquireCLI(root, func(ctx context.Context, gotRoot string, args []string) (acquire.ApplyResult, error) {
		select {
		case <-ctx.Done():
			sawTimeout = true
			return acquire.ApplyResult{
				Root:     gotRoot,
				Reports:  []acquire.TargetReport{{GoGetArg: args[0], Status: acquire.StatusFailed, Err: ctx.Err()}},
				Recovery: acquire.Recovery{Kind: acquire.RecoveryUnavailable},
			}, ctx.Err()
		case <-time.After(2 * time.Second):
			t.Fatal("timeout context was not cancelled")
		}
		return acquire.ApplyResult{}, errors.New("unreachable")
	})
	cmd.out = &buf
	err := cmd.Handle([]string{"session", "--timeout=20ms", "--format=json"})
	if !sawTimeout {
		t.Fatal("ExecuteTargets must receive the CLI timeout context")
	}
	if err == nil || CodeFromError(err) != ExitCanceled {
		t.Fatalf("exit=%d err=%v", CodeFromError(err), err)
	}
	var view acquireCLIView
	if err := json.Unmarshal(buf.Bytes(), &view); err != nil {
		t.Fatal(err)
	}
	if view.Acquisition == acquireStatusSuccess {
		t.Fatal("timeout must not be reported as success")
	}
}

func TestPackageAcquireParentCancelIsCanceled(t *testing.T) {
	root := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	cmd := fakeAcquireCLI(root, func(ctx context.Context, gotRoot string, args []string) (acquire.ApplyResult, error) {
		return acquire.ApplyResult{
			Root:     gotRoot,
			Reports:  []acquire.TargetReport{{GoGetArg: args[0], Status: acquire.StatusFailed, Err: ctx.Err()}},
			Recovery: acquire.Recovery{Kind: acquire.RecoveryUnavailable},
		}, ctx.Err()
	})
	cmd.ctx = ctx
	err := cmd.Handle([]string{"session", "--format=json"})
	if CodeFromError(err) != ExitCanceled {
		t.Fatalf("exit=%d err=%v", CodeFromError(err), err)
	}
}

func TestPackageAcquireUsageIsUsageExit(t *testing.T) {
	err := (&PackageAcquireCommand{out: ioDiscard(), root: t.TempDir()}).Handle(nil)
	if CodeFromError(err) != ExitUsage {
		t.Fatalf("exit=%d err=%v", CodeFromError(err), err)
	}
}

func TestPackageAcquireEnablementFailureKeepsAcquisitionSuccess(t *testing.T) {
	root := t.TempDir()
	var buf bytes.Buffer
	cmd := fakeAcquireCLI(root, successfulExecute())
	cmd.out = &buf
	cmd.enableFn = func(string) (bool, error) { return false, errors.New("bootstrap write failed") }
	err := cmd.Handle([]string{"session", "--enable", "--format=json"})
	if CodeFromError(err) != ExitEnablement {
		t.Fatalf("exit=%d err=%v", CodeFromError(err), err)
	}
	var view acquireCLIView
	if err := json.Unmarshal(buf.Bytes(), &view); err != nil {
		t.Fatal(err)
	}
	if view.Acquisition != acquireStatusSuccess || view.Enablement != enablementFailed || view.Enabled {
		t.Fatalf("%#v", view)
	}
}

func TestCodeFromErrorNilIsSuccess(t *testing.T) {
	if CodeFromError(nil) != ExitSuccess {
		t.Fatal("nil error must be success")
	}
	if CodeFromError(errors.New("plain")) != ExitGeneral {
		t.Fatal("unclassified errors stay general")
	}
}

func TestPackageAcquireAlreadyPresentIsDeterministic(t *testing.T) {
	requireGoTool(t)
	pkgDir := requirePackagesCheckout(t)
	root := writeIsolatedConsumer(t, pkgDir, nil)
	cmd := &PackageAcquireCommand{out: ioDiscard(), root: root}
	if err := cmd.Handle([]string{"session", "--format=json"}); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	cmd.out = &buf
	if err := cmd.Handle([]string{"session", "--format=json"}); err != nil {
		t.Fatalf("second acquire of present module: %v\n%s", err, buf.String())
	}
	var view acquireCLIView
	if err := json.Unmarshal(buf.Bytes(), &view); err != nil {
		t.Fatal(err)
	}
	if view.Acquisition != acquireStatusSuccess {
		t.Fatalf("already-present must stay success: %#v", view)
	}
}

func TestPackageAcquireAlreadyEnabledIsDeterministic(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "bootstrap"), 0o755); err != nil {
		t.Fatal(err)
	}
	calls := 0
	cmd := fakeAcquireCLI(root, successfulExecute())
	cmd.app = nil
	cmd.enableFn = func(name string) (bool, error) {
		calls++
		if calls == 1 {
			return true, nil
		}
		return false, nil
	}
	if err := cmd.Handle([]string{"session", "--enable"}); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Handle([]string{"session", "--enable"}); err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("calls=%d", calls)
	}
}
