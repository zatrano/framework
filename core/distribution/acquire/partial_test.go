package acquire

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writePartialApp(t *testing.T) (app string, args []string) {
	t.Helper()
	root := t.TempDir()
	app = filepath.Join(root, "app")
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
	return app, []string{
		"example.com/a@v0.0.0",
		"example.com/b@v0.0.0",
		"github.com/zatrano/packages@v1.4.0",
		"example.com/d@v0.0.0",
	}
}

func TestExecuteTargetsSeparatesReportFromMutationState(t *testing.T) {
	requireGo(t)
	offlineGoEnv(t)
	app, args := writePartialApp(t)
	got, err := ExecuteTargets(context.Background(), app, args)
	if err == nil {
		t.Fatal("C must fail; must not report all targets acquired")
	}
	if strings.Join(got.Successful(), ",") != args[0]+","+args[1] {
		t.Fatalf("successful=%q", got.Successful())
	}
	if strings.Join(got.Failed(), ",") != args[2] {
		t.Fatalf("failed=%q", got.Failed())
	}
	if strings.Join(got.Unattempted(), ",") != args[3] {
		t.Fatalf("unattempted=%q", got.Unattempted())
	}
	if len(got.Failed()) != 1 || got.Reports[2].Status != StatusFailed {
		t.Fatalf("C %#v", got.Reports)
	}
	if strings.TrimSpace(got.Reports[2].Result.Stderr) == "" && got.Reports[2].Err == nil {
		t.Fatal("failed target must keep the process error")
	}

	in, err := Inspect(app)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := in.Requirement("example.com/a"); !ok {
		t.Fatal("report success A must match go.mod: a missing")
	}
	if _, ok := in.Requirement("example.com/b"); !ok {
		t.Fatal("report success B must match go.mod: b missing")
	}
	if _, ok := in.Requirement("example.com/d"); ok {
		t.Fatal("unattempted D must not appear in go.mod — report and mutation state diverged")
	}
	if _, ok := in.Requirement("github.com/zatrano/packages"); ok {
		t.Fatal("failed C must not be reported as acquired in go.mod")
	}
}

func TestExecuteTargetsDoesNotRollBackMutatedModules(t *testing.T) {
	requireGo(t)
	offlineGoEnv(t)
	app, args := writePartialApp(t)
	got, err := ExecuteTargets(context.Background(), app, args)
	if err == nil {
		t.Fatal("expected C failure")
	}
	if len(got.Successful()) != 2 {
		t.Fatalf("successful=%q", got.Successful())
	}
	in, err := Inspect(app)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := in.Requirement("example.com/a"); !ok {
		t.Fatal("partial apply rolled back A")
	}
	if _, ok := in.Requirement("example.com/b"); !ok {
		t.Fatal("partial apply rolled back B")
	}
}
