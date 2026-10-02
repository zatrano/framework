package acquire

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSnapshotFilesAndRecoverFilesRoundTrip(t *testing.T) {
	dir := t.TempDir()
	mod := "module example.com/app\n\ngo 1.25.0\n"
	sum := "keep\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(mod), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "go.sum"), []byte(sum), 0o644); err != nil {
		t.Fatal(err)
	}
	snap, err := SnapshotFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	if snap.GoModMissing || snap.GoSumMissing {
		t.Fatalf("%#v", snap)
	}
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module mutated\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "go.sum"), []byte("other\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	rec, err := RecoverFiles(context.Background(), snap)
	if err != nil {
		t.Fatal(err)
	}
	if rec.Kind != RecoveryFiles || rec.Err != nil || !rec.RestoredGoMod || !rec.RestoredGoSum {
		t.Fatalf("%#v", rec)
	}
	gotMod, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	if string(gotMod) != mod {
		t.Fatalf("go.mod=%s", gotMod)
	}
	gotSum, err := os.ReadFile(filepath.Join(dir, "go.sum"))
	if err != nil {
		t.Fatal(err)
	}
	if string(gotSum) != sum {
		t.Fatalf("go.sum=%s", gotSum)
	}
}

func TestRecoverFilesRemovesFilesThatWereMissing(t *testing.T) {
	dir := t.TempDir()
	snap, err := SnapshotFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !snap.GoModMissing || !snap.GoSumMissing {
		t.Fatalf("%#v", snap)
	}
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module created\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "go.sum"), []byte("sum\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	rec, err := RecoverFiles(context.Background(), snap)
	if err != nil {
		t.Fatal(err)
	}
	if rec.Kind != RecoveryFiles {
		t.Fatalf("%#v", rec)
	}
	if _, err := os.Stat(filepath.Join(dir, "go.mod")); !os.IsNotExist(err) {
		t.Fatal("missing snapshot must delete go.mod")
	}
	if _, err := os.Stat(filepath.Join(dir, "go.sum")); !os.IsNotExist(err) {
		t.Fatal("missing snapshot must delete go.sum")
	}
}

func TestRecoverFilesDoesNotTouchOtherRootFilesOrModuleCache(t *testing.T) {
	dir := t.TempDir()
	cache := t.TempDir()
	t.Setenv("GOMODCACHE", cache)
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/app\n\ngo 1.25.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "keep.txt"), []byte("app-sidecar\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	sentinel := filepath.Join(cache, "not-undone")
	if err := os.WriteFile(sentinel, []byte("cache\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	snap, err := SnapshotFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module mutated\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	rec, err := RecoverFiles(context.Background(), snap)
	if err != nil || rec.Kind != RecoveryFiles {
		t.Fatalf("rec=%#v err=%v", rec, err)
	}
	keep, err := os.ReadFile(filepath.Join(dir, "keep.txt"))
	if err != nil || string(keep) != "app-sidecar\n" {
		t.Fatal("recovery must not rewrite unrelated files in the root")
	}
	if _, err := os.Stat(sentinel); err != nil {
		t.Fatal("restoring go.mod / go.sum must not undo the module cache")
	}
}

func TestRecoverFilesFailureIsDistinctFromAcquisitionError(t *testing.T) {
	dir := t.TempDir()
	mod := "module example.com/app\n\ngo 1.25.0\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(mod), 0o644); err != nil {
		t.Fatal(err)
	}
	snap, err := SnapshotFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	fake := &scriptedRunner{failArg: "example.com/c@v1.0.0", err: errMutation}
	_, acqErr := executeTargets(context.Background(), fake, Request{Root: dir}, []string{
		"example.com/c@v1.0.0",
	})
	if acqErr == nil {
		t.Fatal("expected acquisition failure")
	}
	if err := os.Remove(filepath.Join(dir, "go.mod")); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "go.mod"), 0o755); err != nil {
		t.Fatal(err)
	}
	rec, recErr := RecoverFiles(context.Background(), snap)
	if recErr == nil || rec.Kind != RecoveryFailed {
		t.Fatalf("expected recovery failure %#v %v", rec, recErr)
	}
	if recErr == acqErr || rec.Err == acqErr {
		t.Fatal("recovery failure must not reuse the acquisition error")
	}
	if rec.RestoredGoMod {
		t.Fatal("failed go.mod restore must not report RestoredGoMod")
	}
}

func TestExecuteTargetsLeavesRecoveryUnavailable(t *testing.T) {
	fake := &scriptedRunner{failArg: "example.com/c@v1.0.0", err: errMutation}
	got, err := executeTargets(context.Background(), fake, Request{Root: t.TempDir()}, []string{
		"example.com/a@v1.0.0",
		"example.com/c@v1.0.0",
		"example.com/d@v1.0.0",
	})
	if err == nil {
		t.Fatal("expected acquisition failure")
	}
	if got.Recovery.Kind != RecoveryUnavailable {
		t.Fatalf("ExecuteTargets must not restore files: %#v", got.Recovery)
	}
}

func TestWithRecoveryPreservesPartialApplyReports(t *testing.T) {
	a, b, c, d := "example.com/a@v1.0.0", "example.com/b@v1.0.0", "example.com/c@v1.0.0", "example.com/d@v1.0.0"
	fake := &scriptedRunner{failArg: c, err: errMutation}
	got, err := executeTargets(context.Background(), fake, Request{Root: t.TempDir()}, []string{a, b, c, d})
	if err == nil {
		t.Fatal("expected acquisition failure")
	}
	attached := got.WithRecovery(Recovery{Kind: RecoveryFiles, RestoredGoMod: true, RestoredGoSum: true})
	if strings.Join(attached.Successful(), ",") != a+","+b {
		t.Fatalf("successful=%q", attached.Successful())
	}
	if strings.Join(attached.Failed(), ",") != c {
		t.Fatalf("failed=%q", attached.Failed())
	}
	if strings.Join(attached.Unattempted(), ",") != d {
		t.Fatalf("unattempted=%q", attached.Unattempted())
	}
	if attached.Recovery.Kind != RecoveryFiles {
		t.Fatalf("%#v", attached.Recovery)
	}
	if got.Recovery.Kind != RecoveryUnavailable {
		t.Fatal("WithRecovery must not mutate the original result")
	}
}

func TestRecoverFilesDoesNotClaimTransactionalRollback(t *testing.T) {
	if RecoveryFiles == "transactional" || RecoveryFiles == "guaranteed" {
		t.Fatal("RecoveryFiles must not be a transactional kind")
	}
	if RecoveryUnavailable != "unavailable" || RecoveryFailed != "failed" {
		t.Fatalf("kinds=%q %q", RecoveryUnavailable, RecoveryFailed)
	}
}

func TestExecuteTargetsThenRecoverFilesSeparatesMutationFromReports(t *testing.T) {
	requireGo(t)
	offlineGoEnv(t)
	app, args := writePartialApp(t)
	snap, err := SnapshotFiles(app)
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
	if got.Recovery.Kind != RecoveryUnavailable {
		t.Fatalf("auto-recovery %#v", got.Recovery)
	}
	in, err := Inspect(app)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := in.Requirement("example.com/a"); !ok {
		t.Fatal("A must remain until RecoverFiles")
	}
	rec, recErr := RecoverFiles(context.Background(), snap)
	if recErr != nil || rec.Kind != RecoveryFiles {
		t.Fatalf("rec=%#v err=%v", rec, recErr)
	}
	got = got.WithRecovery(rec)
	if strings.Join(got.Successful(), ",") != args[0]+","+args[1] {
		t.Fatalf("reports rewritten: successful=%q", got.Successful())
	}
	if strings.Join(got.Failed(), ",") != args[2] || strings.Join(got.Unattempted(), ",") != args[3] {
		t.Fatalf("failed=%q unattempted=%q", got.Failed(), got.Unattempted())
	}
	in, err = Inspect(app)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := in.Requirement("example.com/a"); ok {
		t.Fatal("file recovery must restore go.mod so A is gone")
	}
	if _, ok := in.Requirement("example.com/b"); ok {
		t.Fatal("file recovery must restore go.mod so B is gone")
	}
	if _, err := os.Stat(sentinel); err != nil {
		t.Fatal("file recovery must not undo the module cache")
	}
}
