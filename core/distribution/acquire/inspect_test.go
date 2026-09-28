package acquire

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInspectParsesRequireAndChecksums(t *testing.T) {
	dir := t.TempDir()
	mod := "module example.com/app\n\ngo 1.25.0\n\nrequire (\n\texample.com/a v1.2.3\n\texample.com/b v0.1.0 // indirect\n)\n\nreplace example.com/dep => ../dep\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(mod), 0o644); err != nil {
		t.Fatal(err)
	}
	sum := "example.com/a v1.2.3 h1:abc\nexample.com/a v1.2.3/go.mod h1:def\nexample.com/b v0.1.0 h1:ghi\n"
	if err := os.WriteFile(filepath.Join(dir, "go.sum"), []byte(sum), 0o644); err != nil {
		t.Fatal(err)
	}
	in, err := Inspect(dir)
	if err != nil {
		t.Fatal(err)
	}
	if in.Root != dir || in.Module != "example.com/app" || in.Go != "1.25.0" {
		t.Fatalf("%#v", in)
	}
	a, ok := in.Requirement("example.com/a")
	if !ok || a.Version != "v1.2.3" || a.Indirect {
		t.Fatalf("a=%#v ok=%v", a, ok)
	}
	b, ok := in.Requirement("example.com/b")
	if !ok || b.Version != "v0.1.0" || !b.Indirect {
		t.Fatalf("b=%#v ok=%v", b, ok)
	}
	if _, ok := in.Requirement("example.com/dep"); ok {
		t.Fatal("replace must not be reported as a require")
	}
	sums := in.Sums("example.com/a")
	if len(sums) != 2 || sums[0].Hash != "h1:abc" || sums[1].Version != "v1.2.3/go.mod" {
		t.Fatalf("sums=%#v", sums)
	}
}

func TestInspectDoesNotMutateModuleFiles(t *testing.T) {
	dir := t.TempDir()
	mod := "module example.com/app\n\ngo 1.25.0\n"
	sum := "keep\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(mod), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "go.sum"), []byte(sum), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Inspect(dir); err != nil {
		t.Fatal(err)
	}
	gotMod, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	if string(gotMod) != mod {
		t.Fatalf("go.mod mutated:\n%s", gotMod)
	}
	gotSum, err := os.ReadFile(filepath.Join(dir, "go.sum"))
	if err != nil {
		t.Fatal(err)
	}
	if string(gotSum) != sum {
		t.Fatalf("go.sum mutated:\n%s", gotSum)
	}
}

func TestInspectReportsMissingFiles(t *testing.T) {
	dir := t.TempDir()
	in, err := Inspect(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !in.GoModMissing || !in.GoSumMissing {
		t.Fatalf("%#v", in)
	}
	if len(in.Requirements) != 0 || len(in.Checksums) != 0 {
		t.Fatalf("missing files must not invent state: %#v", in)
	}
}

func TestInspectRequiresRoot(t *testing.T) {
	if _, err := Inspect("  "); err == nil {
		t.Fatal("empty root")
	}
}

func TestInspectSeparatedFromInvocation(t *testing.T) {
	requireGo(t)
	offlineGoEnv(t)
	app, arg := writeLocalApp(t)
	res, err := Execute(context.Background(), Request{Root: app, GoGetArg: arg})
	if err != nil {
		t.Fatalf("Execute: %v\n%s", err, res.Stderr)
	}
	in, err := Inspect(app)
	if err != nil {
		t.Fatal(err)
	}
	if res.ExitCode != 0 {
		t.Fatalf("process exit=%d", res.ExitCode)
	}
	req, ok := in.Requirement("example.com/dep")
	if !ok {
		t.Fatal("inspection must observe the require Go wrote")
	}
	if req.Version == "" || strings.Contains(strings.ToLower(req.Version), "latest") {
		t.Fatalf("inspection selected or missed version: %#v", req)
	}
	if res.Invocation.Args[1] != arg {
		t.Fatalf("GoGetArg rewritten at invoke: %q", res.Invocation.Args)
	}
}

func TestInspectDoesNotTreatFailedGetAsRequire(t *testing.T) {
	requireGo(t)
	offlineGoEnv(t)
	app, _ := writeLocalApp(t)
	res, err := Execute(context.Background(), Request{
		Root:     app,
		GoGetArg: "github.com/zatrano/packages@v1.4.0",
	})
	if err == nil || res.ExitCode == 0 {
		t.Fatal("expected failed go get")
	}
	in, err := Inspect(app)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := in.Requirement("github.com/zatrano/packages"); ok {
		t.Fatal("failed go get must not be reported as an acquired require")
	}
}
