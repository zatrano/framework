package tests

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/zatrano/framework/v2/kernel"
)

func moduleRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), ".."))
}

func TestProductAndModuleIdentity(t *testing.T) {
	root := moduleRoot(t)
	raw, err := os.ReadFile(filepath.Join(root, "VERSION"))
	if err != nil {
		t.Fatal(err)
	}
	version := strings.TrimSpace(string(raw))
	if version != "2.0.28" {
		t.Fatalf("VERSION=%q want 2.0.28", version)
	}

	mod, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	path := ""
	for _, line := range strings.Split(string(mod), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			path = strings.TrimSpace(strings.TrimPrefix(line, "module "))
			break
		}
	}
	if path != "github.com/zatrano/framework/v2" {
		t.Fatalf("module path=%q", path)
	}

	app := kernel.NewApplication(root)
	if got := app.Version(); got != version {
		t.Fatalf("Version()=%q want %q", got, version)
	}
}

// Architecture: kernel must not import github.com/zatrano/packages.
func TestFrameworkDoesNotImportPackagesModule(t *testing.T) {
	root := moduleRoot(t)
	fset := token.NewFileSet()
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := filepath.Base(path)
			if name == "vendor" || name == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		file, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		for _, spec := range file.Imports {
			imp := strings.Trim(spec.Path.Value, `"`)
			if imp == "github.com/zatrano/packages" || strings.HasPrefix(imp, "github.com/zatrano/packages/") {
				t.Errorf("%s imports %s", rel, imp)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestDistributionProtocolLivesUnderDistribution(t *testing.T) {
	root := moduleRoot(t)
	for _, name := range []string{"acquire", "manifest", "registry"} {
		if st, err := os.Stat(filepath.Join(root, name)); err == nil && st.IsDir() {
			t.Errorf("%s/ must not sit at module root — use distribution/%s", name, name)
		}
		dir := filepath.Join(root, "distribution", name)
		if st, err := os.Stat(dir); err != nil || !st.IsDir() {
			t.Errorf("missing distribution/%s", name)
		}
	}
}

func TestRegistryDoesNotImportConsole(t *testing.T) {
	dir := filepath.Join(moduleRoot(t), "distribution", "registry")
	fset := token.NewFileSet()
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		file, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, spec := range file.Imports {
			imp := strings.Trim(spec.Path.Value, `"`)
			if strings.Contains(imp, "/console") {
				t.Errorf("%s imports %s — CLI consumes registry, not the reverse", filepath.Base(path), imp)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestCLIDoesNotReimplementRegistryResolution(t *testing.T) {
	path := filepath.Join(moduleRoot(t), "console", "package_registry.go")
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(src)
	for _, want := range []string{
		`"github.com/zatrano/framework/v2/distribution/registry"`,
		"idx.Search(",
		"idx.Lookup(",
		"idx.Resolve(",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("package_registry.go must call registry API %s", want)
		}
	}

	banned := map[string]bool{
		"compareSemver": true, "latestCompatible": true, "MeetsFrameworkMin": true,
		"releaseOK": true, "normalizeVersion": true, "semverParts": true,
	}
	fset := token.NewFileSet()
	err = filepath.WalkDir(filepath.Join(moduleRoot(t), "console"), func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return err
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Name == nil {
				continue
			}
			if banned[fn.Name.Name] {
				t.Errorf("%s defines %s — resolution stays in package registry", filepath.Base(path), fn.Name.Name)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestEnablementCommandsDoNotImportRegistry(t *testing.T) {
	root := filepath.Join(moduleRoot(t), "console")
	files := []string{"package_cmd.go", "package_doctor.go", "package_wire.go", "package_env.go"}
	fset := token.NewFileSet()
	for _, name := range files {
		path := filepath.Join(root, name)
		file, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		for _, spec := range file.Imports {
			imp := strings.Trim(spec.Path.Value, `"`)
			if strings.HasSuffix(imp, "/registry") || strings.HasSuffix(imp, "/acquire") {
				t.Errorf("%s imports %s — enablement stays off registry/acquire", name, imp)
			}
		}
	}
}

func TestSearchHitHasNoSelectionFields(t *testing.T) {
	allow := map[string]bool{
		"Name": true, "Import": true, "Module": true, "Kind": true,
		"Layer": true, "Heavy": true, "Description": true,
	}
	got := structFields(t, filepath.Join(moduleRoot(t), "console", "package_registry.go"), "searchHit")
	for name := range got {
		if !allow[name] {
			t.Errorf("searchHit grew %s — search must not carry selection/release fields", name)
		}
	}
	for _, ban := range []string{"Selected", "Releases", "Release", "Version"} {
		if got[ban] {
			t.Errorf("searchHit must not have %s", ban)
		}
	}
}

func TestAcquirePlanStructFreeze(t *testing.T) {
	allow := map[string]bool{
		"Schema": true, "Name": true, "Import": true, "Module": true,
		"Query": true, "Selected": true, "Kind": true, "Heavy": true,
	}
	got := structFields(t, filepath.Join(moduleRoot(t), "distribution", "acquire", "plan.go"), "Plan")
	for name := range got {
		if !allow[name] {
			t.Errorf("Plan grew %s — Plan is not enablement, lock, or Apply state", name)
		}
	}
	for name := range allow {
		if !got[name] {
			t.Errorf("Plan lost %s", name)
		}
	}
	for _, ban := range []string{"Enabled", "Imported", "Stubs", "Enablement", "Lock", "Checksum", "Apply"} {
		if got[ban] {
			t.Errorf("Plan must not have %s", ban)
		}
	}
}

func TestApplySurfaceStaysFrozen(t *testing.T) {
	root := moduleRoot(t)
	raw, err := os.ReadFile(filepath.Join(root, "distribution", "acquire", "APPLY.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	for _, want := range []string{
		"**Status:** Frozen",
		"not authorized",
		"No automatic tidy",
		"package:install",
		"zatrano.lock",
		"exec.Command",
		"fail-fast",
		"rollback guaranteed",
		"Enabled ∩ Imported",
		"complete and frozen",
		"later specification",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("APPLY.md missing %q — Apply contract stays frozen", want)
		}
	}
	if strings.Contains(text, "Current gate") {
		t.Error("APPLY.md still names a current gate — Apply contract has no remaining implementation step")
	}
	allowed := map[string]bool{
		"apply.go": true, "process.go": true, "exec_runner.go": true, "inspect.go": true,
		"lock.go": true, "recovery.go": true, "plan.go": true, "doc.go": true, "dry_run.go": true,
	}
	err = filepath.WalkDir(filepath.Join(root, "distribution", "acquire"), func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		base := strings.ToLower(filepath.Base(path))
		if !strings.HasSuffix(base, ".go") || strings.HasSuffix(base, "_test.go") {
			return nil
		}
		if !allowed[base] {
			t.Errorf("%s — Apply production surface is frozen", filepath.Base(path))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestDryRunContractGate(t *testing.T) {
	root := moduleRoot(t)
	raw, err := os.ReadFile(filepath.Join(root, "distribution", "acquire", "ORCHESTRATION.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	for _, want := range []string{
		"# Acquisition / Enablement Contracts",
		"ACCEPTED",
		"Contract A — Dry-run",
		"Contract A complete",
		"Contract B complete",
		"Contract C OPEN",
		"COMPLETE / FROZEN",
		"IMPLEMENTED",
		"MUST NOT enable automatically",
		"explicitly opened",
		"Acquisition ≠ Enablement",
		"func Apply",
		"package:install",
		"implicit transaction",
		"renamed equivalent",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("ORCHESTRATION.md missing %q — Contract A stays bounded", want)
		}
	}
	if strings.Contains(text, "NOT ACCEPTED") {
		t.Error("ORCHESTRATION.md still says NOT ACCEPTED — SPEC was accepted")
	}
	dir := filepath.Join(root, "distribution", "acquire")
	if _, err := os.Stat(filepath.Join(dir, "dry_run.go")); err != nil {
		t.Fatal("missing dry_run.go — Contract A DryRun")
	}
	if _, err := os.Stat(filepath.Join(dir, "dry_run_test.go")); err != nil {
		t.Fatal("missing dry_run_test.go — Contract A tests")
	}
	src, err := os.ReadFile(filepath.Join(dir, "dry_run.go"))
	if err != nil {
		t.Fatal(err)
	}
	body := string(src)
	for _, want := range []string{"func DryRun(", "func DryRunTargets("} {
		if !strings.Contains(body, want) {
			t.Errorf("dry_run.go missing %s", want)
		}
	}
	for _, ban := range []string{"Invoke(", "Execute(", "ExecuteTargets(", "lockMutation(", "RecoverFiles(", "SnapshotFiles(", "os/exec"} {
		if strings.Contains(body, ban) {
			t.Errorf("dry_run.go contains %s — DryRun must not execute or recover", ban)
		}
	}
	for _, name := range []string{"acquire_cmd.go"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			t.Errorf("%s — unauthorized acquire production file", name)
		}
	}
}

func TestApplyProcessInvocationBoundary(t *testing.T) {
	root := moduleRoot(t)
	dir := filepath.Join(root, "distribution", "acquire")
	for _, name := range []string{"apply.go", "process.go", "exec_runner.go"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Fatalf("missing invocation file %s", name)
		}
	}
	allowApplyImport := map[string]bool{"context": true, "fmt": true, "strings": true}
	allowProcessImport := map[string]bool{"context": true}
	allowRecoveryImport := map[string]bool{"context": true, "fmt": true, "os": true, "path/filepath": true, "strings": true}
	allowDryRunImport := map[string]bool{"fmt": true, "strings": true}
	fset := token.NewFileSet()
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		base := filepath.Base(path)
		src, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		text := string(src)
		if strings.Contains(text, "type ApplyResult ") || strings.Contains(text, "type ApplyResult\t") {
			if base != "apply.go" {
				t.Errorf("%s invents ApplyResult — partial-apply report lives in apply.go", base)
			}
		}
		if strings.Contains(text, "RecoveryGuaranteed") || strings.Contains(text, "RecoveryTransactional") {
			t.Errorf("%s claims guaranteed/transactional rollback", base)
		}
		file, parseErr := parser.ParseFile(fset, path, nil, 0)
		if parseErr != nil {
			return parseErr
		}
		if strings.HasSuffix(base, "_test.go") {
			if base == "apply_test.go" && (strings.Contains(text, "ExecRunner") || strings.Contains(text, "os/exec")) {
				t.Errorf("%s must use a fake Runner — Execute owns real go get", base)
			}
			if base == "lock_test.go" && (strings.Contains(text, "ExecRunner") || strings.Contains(text, "os/exec")) {
				t.Errorf("%s must prove serialization with a fake Runner — not real go get", base)
			}
			if base == "partial_test.go" {
				if !strings.Contains(text, "ExecuteTargets(") {
					t.Error("partial_test.go must exercise ExecuteTargets")
				}
				if !strings.Contains(text, "Inspect(") {
					t.Error("partial_test.go must compare ApplyResult with Inspect mutation state")
				}
				if strings.Contains(text, "func Rollback") || strings.Contains(text, "go mod tidy") {
					t.Error("partial apply must not implement rollback or tidy")
				}
			}
			if base == "recovery_test.go" {
				if !strings.Contains(text, "RecoverFiles(") || !strings.Contains(text, "SnapshotFiles(") {
					t.Error("recovery_test.go must exercise SnapshotFiles and RecoverFiles")
				}
				if !strings.Contains(text, "GOMODCACHE") {
					t.Error("recovery_test.go must prove file restore does not undo the module cache")
				}
				if strings.Contains(text, "func Rollback") {
					t.Error("do not add transactional Rollback")
				}
			}
			if base == "integration_test.go" {
				for _, want := range []string{
					"FromResult(", "Targets(", "GoGetArg", "Execute(", "ExecuteTargets(", "Inspect(", "SnapshotFiles(", "RecoverFiles(",
				} {
					if !strings.Contains(text, want) {
						t.Errorf("integration_test.go must wire %s on a real module root", want)
					}
				}
				if strings.Contains(text, "ExecRunner") || strings.Contains(text, "os/exec") {
					t.Error("integration must use Execute / ExecuteTargets, not a second process abstraction")
				}
				if strings.Contains(text, "func Apply") || strings.Contains(text, "func Rollback") {
					t.Error("integration must not add Apply or Rollback")
				}
				if strings.Contains(text, ".Resolve(") {
					t.Error("integration must not re-resolve; start from FromResult")
				}
			}
			if base == "execute_test.go" {
				if !strings.Contains(text, "Execute(") {
					t.Error("execute_test.go must exercise Execute")
				}
				if strings.Contains(text, "os.ReadFile") || strings.Contains(text, "os.ReadDir") {
					t.Error("execute_test.go must not inspect go.mod / go.sum")
				}
				if strings.Contains(text, "golang.org/x/mod") || strings.Contains(text, "modfile") {
					t.Error("execute_test.go must not parse module files")
				}
			}
			if base == "dry_run_test.go" {
				if !strings.Contains(text, "DryRun(") {
					t.Error("dry_run_test.go must exercise DryRun")
				}
				if strings.Contains(text, "Execute(") || strings.Contains(text, "Invoke(") || strings.Contains(text, "ExecRunner") || strings.Contains(text, "os/exec") {
					t.Error("dry-run tests must not call the execution boundary")
				}
			}
			return nil
		}
		if base == "apply.go" {
			for _, spec := range file.Imports {
				imp := strings.Trim(spec.Path.Value, `"`)
				if !allowApplyImport[imp] {
					t.Errorf("apply.go imports %s — Invoke must not reach exec or the registry", imp)
				}
			}
		}
		if base == "process.go" {
			for _, spec := range file.Imports {
				imp := strings.Trim(spec.Path.Value, `"`)
				if !allowProcessImport[imp] {
					t.Errorf("process.go imports %s — boundary types stay process-free of exec", imp)
				}
			}
		}
		if base != "exec_runner.go" {
			for _, spec := range file.Imports {
				if strings.Trim(spec.Path.Value, `"`) == "os/exec" {
					t.Errorf("%s imports os/exec — only exec_runner.go may invoke a process", base)
				}
			}
			if strings.Contains(text, "exec.Command") {
				t.Errorf("%s contains exec.Command — do not scatter process starts", base)
			}
		}
		if base == "inspect.go" {
			if strings.Contains(text, "os.WriteFile") || strings.Contains(text, "os.Create") || strings.Contains(text, "os.OpenFile") {
				t.Error("inspect.go must not mutate go.mod / go.sum")
			}
			if strings.Contains(text, "golang.org/x/mod") {
				t.Error("inspect.go must not import x/mod — observation only")
			}
			if strings.Contains(text, "lockMutation") {
				t.Error("Inspect must not take the mutation lock")
			}
		}
		if base == "recovery.go" {
			for _, spec := range file.Imports {
				imp := strings.Trim(spec.Path.Value, `"`)
				if !allowRecoveryImport[imp] {
					t.Errorf("recovery.go imports %s — file restore only", imp)
				}
			}
			if !strings.Contains(text, "lockMutation(") {
				t.Error("RecoverFiles must serialize through lockMutation")
			}
			if strings.Contains(text, "os.WriteFile") && (!strings.Contains(text, "go.mod") || !strings.Contains(text, "go.sum")) {
				t.Error("recovery must write only go.mod / go.sum")
			}
			if strings.Contains(text, "go mod tidy") || strings.Contains(text, "zatrano.lock") {
				t.Errorf("recovery.go must not tidy or write a lockfile")
			}
			if strings.Contains(text, "Invoke(") || strings.Contains(text, "Execute(") {
				t.Error("RecoverFiles must not run go get")
			}
		}
		if base == "dry_run.go" {
			for _, spec := range file.Imports {
				imp := strings.Trim(spec.Path.Value, `"`)
				if !allowDryRunImport[imp] {
					t.Errorf("dry_run.go imports %s — DryRun is report-only", imp)
				}
			}
			if strings.Contains(text, "lockMutation(") || strings.Contains(text, "Invoke(") || strings.Contains(text, "Execute(") {
				t.Error("dry_run.go must not call the execution boundary")
			}
		}
		if base == "lock.go" {
			if strings.Contains(text, "zatrano.lock") || strings.Contains(text, "os.WriteFile") || strings.Contains(text, "os.Create") {
				t.Error("mutation lock must not write a lockfile")
			}
		}
		if base == "exec_runner.go" {
			if !strings.Contains(text, "exec.CommandContext") {
				t.Error("exec_runner.go must pass context via CommandContext")
			}
			if strings.Contains(text, "exec.Command(") {
				t.Error("exec_runner.go must not use exec.Command without context")
			}
			if strings.Contains(text, "sh -c") || strings.Contains(text, "/bin/sh") {
				t.Error("exec_runner.go must not interpolate a shell")
			}
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Name == nil {
				continue
			}
			switch fn.Name.Name {
			case "Apply", "Install", "Tidy", "Download", "Rollback", "Revert", "Preview", "Acquire", "compareSemver", "latestCompatible", "MeetsFrameworkMin":
				t.Errorf("%s defines %s — Apply production surface is frozen", base, fn.Name.Name)
			case "DryRun", "DryRunTargets":
				if base != "dry_run.go" {
					t.Errorf("%s defines %s — DryRun lives in dry_run.go", base, fn.Name.Name)
				}
			case "AllAcquired":
				t.Errorf("%s defines AllAcquired — partial apply must not claim all targets acquired", base)
			}
		}
		ast.Inspect(file, func(n ast.Node) bool {
			sel, ok := n.(*ast.SelectorExpr)
			if !ok || sel.Sel == nil {
				return true
			}
			if sel.Sel.Name == "Resolve" || sel.Sel.Name == "Search" || sel.Sel.Name == "Lookup" {
				t.Errorf("%s calls Index.%s — Invoke must not re-resolve", base, sel.Sel.Name)
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	src, err := os.ReadFile(filepath.Join(dir, "apply.go"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(src)
	if !strings.Contains(text, "func Execute(") {
		t.Error("apply.go must export Execute as the acquisition go get entry")
	}
	if !strings.Contains(text, "func ExecuteTargets(") {
		t.Error("apply.go must export ExecuteTargets as the partial-apply entry")
	}
	if !strings.Contains(text, "type ApplyResult ") {
		t.Error("apply.go must report partial apply as ApplyResult")
	}
	if !strings.Contains(text, "StatusUnattempted") || !strings.Contains(text, "StatusFailed") || !strings.Contains(text, "StatusSuccess") {
		t.Error("ApplyResult must distinguish success, failed, and unattempted")
	}
	if strings.Contains(text, "func Apply(") {
		t.Error("do not add a general Apply API; ExecuteTargets is the partial-apply gate")
	}
	if !strings.Contains(text, "RecoveryUnavailable") {
		t.Error("ExecuteTargets must leave recovery unavailable")
	}
	if !strings.Contains(text, "func (r ApplyResult) WithRecovery(") && !strings.Contains(text, "func (r ApplyResult) WithRecovery (") {
		if !strings.Contains(text, "WithRecovery(") {
			t.Error("ApplyResult must attach recovery without rewriting target reports")
		}
	}
	if !strings.Contains(text, "ExecRunner{}") {
		t.Error("Execute must bind ExecRunner; do not scatter exec.Command")
	}
	if !strings.Contains(text, "lockMutation(") {
		t.Error("Execute must serialize mutation per module root")
	}
	if _, err := os.Stat(filepath.Join(dir, "lock.go")); err != nil {
		t.Fatal("missing lock.go")
	}
	if _, err := os.Stat(filepath.Join(dir, "lock_test.go")); err != nil {
		t.Fatal("missing lock_test.go")
	}
	insp, err := os.ReadFile(filepath.Join(dir, "inspect.go"))
	if err != nil {
		t.Fatal("missing inspect.go")
	}
	if !strings.Contains(string(insp), "func Inspect(") {
		t.Error("inspect.go must export Inspect")
	}
	if _, err := os.Stat(filepath.Join(dir, "inspect_test.go")); err != nil {
		t.Fatal("missing inspect_test.go")
	}
	if _, err := os.Stat(filepath.Join(dir, "execute_test.go")); err != nil {
		t.Fatal("missing execute_test.go — go get execution tests belong there, not in apply_test.go")
	}
	if _, err := os.Stat(filepath.Join(dir, "partial_test.go")); err != nil {
		t.Fatal("missing partial_test.go — partial-apply mutation vs report tests belong there")
	}
	if _, err := os.Stat(filepath.Join(dir, "recovery.go")); err != nil {
		t.Fatal("missing recovery.go")
	}
	if _, err := os.Stat(filepath.Join(dir, "recovery_test.go")); err != nil {
		t.Fatal("missing recovery_test.go — guaranteed vs file recovery tests belong there")
	}
	if _, err := os.Stat(filepath.Join(dir, "integration_test.go")); err != nil {
		t.Fatal("missing integration_test.go — Plan through recovery on a real module root")
	}
}

func TestAcquireExportsStayFrozen(t *testing.T) {
	allowFn := map[string]bool{
		"FromResult": true, "Targets": true, "Invoke": true, "Execute": true, "ExecuteTargets": true,
		"Inspect": true, "SnapshotFiles": true, "RecoverFiles": true, "DryRun": true, "DryRunTargets": true,
	}
	allowMethod := map[string]bool{
		"GoGetArg": true, "Run": true, "WithRecovery": true, "Successful": true, "Failed": true,
		"Unattempted": true, "Requirement": true, "Sums": true,
	}
	fset := token.NewFileSet()
	dir := filepath.Join(moduleRoot(t), "distribution", "acquire")
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		file, parseErr := parser.ParseFile(fset, path, nil, 0)
		if parseErr != nil {
			return parseErr
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Name == nil || !fn.Name.IsExported() {
				continue
			}
			if fn.Recv != nil {
				if !allowMethod[fn.Name.Name] {
					t.Errorf("%s grew method %s — Apply contract exports are frozen", filepath.Base(path), fn.Name.Name)
				}
				continue
			}
			if !allowFn[fn.Name.Name] {
				t.Errorf("%s grew %s — Apply contract exports are frozen", filepath.Base(path), fn.Name.Name)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestAcquireExportsStayPlanOnly(t *testing.T) {
	allowFn := map[string]bool{"FromResult": true, "Targets": true}
	allowMethod := map[string]bool{"GoGetArg": true}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filepath.Join(moduleRoot(t), "distribution", "acquire", "plan.go"), nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name == nil || !fn.Name.IsExported() {
			continue
		}
		if fn.Recv != nil {
			if !allowMethod[fn.Name.Name] {
				t.Errorf("Plan grew method %s — Apply stays out of the acquisition plan", fn.Name.Name)
			}
			continue
		}
		if !allowFn[fn.Name.Name] {
			t.Errorf("acquire grew %s — Acquisition plan exports are FromResult and Targets", fn.Name.Name)
		}
	}
}

func TestAcquirePlanLayerDoesNotResolveOrApply(t *testing.T) {
	path := filepath.Join(moduleRoot(t), "distribution", "acquire", "plan.go")
	allowImport := map[string]bool{
		"fmt": true, "strings": true, "sort": true,
		"github.com/zatrano/framework/v2/distribution/registry": true,
	}
	bannedFn := map[string]bool{
		"Apply": true, "Install": true, "Tidy": true, "Download": true,
		"compareSemver": true, "latestCompatible": true, "MeetsFrameworkMin": true,
		"Invoke": true,
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, spec := range file.Imports {
		imp := strings.Trim(spec.Path.Value, `"`)
		if !allowImport[imp] {
			t.Errorf("plan.go imports %s — Plan layer stays a pure translation", imp)
		}
	}
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name == nil {
			continue
		}
		if bannedFn[fn.Name.Name] {
			t.Errorf("plan.go defines %s — Apply/resolution stay out of the acquisition plan", fn.Name.Name)
		}
	}
	ast.Inspect(file, func(n ast.Node) bool {
		sel, ok := n.(*ast.SelectorExpr)
		if !ok || sel.Sel == nil {
			return true
		}
		if sel.Sel.Name == "Resolve" || sel.Sel.Name == "Search" || sel.Sel.Name == "Lookup" {
			t.Errorf("plan.go calls Index.%s — FromResult must not re-resolve", sel.Sel.Name)
		}
		return true
	})
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(src)
	for _, ban := range []string{"os.WriteFile", "os.Create", "os.Mkdir", "exec.Command", "go mod tidy", "go mod edit"} {
		if strings.Contains(text, ban) {
			t.Errorf("plan.go contains %q — Acquisition plan stays filesystem-free", ban)
		}
	}
}

func TestConsoleAcquireCLIMayImportAcquire(t *testing.T) {
	dir := filepath.Join(moduleRoot(t), "console")
	fset := token.NewFileSet()
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		base := filepath.Base(path)
		file, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, spec := range file.Imports {
			imp := strings.Trim(spec.Path.Value, `"`)
			if !strings.HasSuffix(imp, "/acquire") {
				continue
			}
			if base != "package_acquire.go" && !strings.HasPrefix(base, "package_acquire") {
				t.Errorf("%s imports acquire — only package:acquire may consume acquire (%s)", base, imp)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestCLIAcquireGate(t *testing.T) {
	root := moduleRoot(t)
	path := filepath.Join(root, "console", "package_acquire.go")
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatal("missing console/package_acquire.go — Contract B CLI acquisition")
	}
	text := string(src)
	for _, want := range []string{
		`"github.com/zatrano/framework/v2/distribution/acquire"`,
		`"github.com/zatrano/framework/v2/distribution/registry"`,
		"idx.Resolve(",
		"acquire.FromResult(",
		"acquire.Targets(",
		"acquire.DryRunTargets(",
		"acquire.ExecuteTargets(",
		"acquire.Inspect(",
		"acquire.SnapshotFiles(",
		"acquire.RecoverFiles(",
		`return "package:acquire"`,
	} {
		if !strings.Contains(text, want) {
			t.Errorf("package_acquire.go must orchestrate %s", want)
		}
	}
	for _, ban := range []string{
		"compareSemver", "latestCompatible", "MeetsFrameworkMin",
		"exec.Command", "os/exec", "go mod tidy", "zatrano.lock",
		"func Apply", "func Rollback",
	} {
		if strings.Contains(text, ban) {
			t.Errorf("package_acquire.go contains %s — CLI must not own acquisition mechanics", ban)
		}
	}
	if !strings.Contains(text, `hasFlag(args, "--enable")`) {
		t.Error("package_acquire.go must gate enablement on explicit --enable")
	}
	if !strings.Contains(text, "enablePackage(") {
		t.Error("package_acquire.go --enable must reuse enablePackage")
	}
	if !strings.Contains(text, `"enabled": false`) && !strings.Contains(text, "Enabled: false") && !strings.Contains(text, "enabled: false") && !strings.Contains(text, `json:"enabled"`) {
		t.Error("package_acquire.go must still report enabled as a distinct field")
	}
	cmdSrc, err := os.ReadFile(filepath.Join(root, "console", "package_cmd.go"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(cmdSrc), "/acquire") {
		t.Error("package_cmd.go must not import acquire — package:install stays enablement")
	}
}

func TestExplicitEnablementAfterAcquire(t *testing.T) {
	root := moduleRoot(t)
	raw, err := os.ReadFile(filepath.Join(root, "distribution", "acquire", "ORCHESTRATION.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	for _, want := range []string{
		"Contract C OPEN",
		"Acquisition ≠ Enablement",
		"MUST NOT enable automatically",
		"--enable",
		"not_requested",
		"Apply contract — v2.0.22",
		"package:install",
		"implicit transaction",
		"enablePackage",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("ORCHESTRATION.md missing %q — Contract C explicit workflow", want)
		}
	}
	for _, name := range []string{
		"acquire_enable.go",
		"enablement.go",
		"package_acquire_enable.go",
	} {
		if _, err := os.Stat(filepath.Join(root, "distribution", "acquire", name)); err == nil {
			t.Errorf("%s — Contract C must not add an acquire-side enablement engine", name)
		}
		if _, err := os.Stat(filepath.Join(root, "console", name)); err == nil {
			t.Errorf("console/%s — Contract C stays in package_acquire.go", name)
		}
	}
	cmdSrc, err := os.ReadFile(filepath.Join(root, "console", "package_cmd.go"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(cmdSrc), "/acquire") {
		t.Error("package_cmd.go must not import acquire — package:install stays enablement")
	}
	if !strings.Contains(string(cmdSrc), `return "package:install"`) || !strings.Contains(string(cmdSrc), "enablePackage(") {
		t.Error("package:install must remain enablement")
	}
	dry, err := os.ReadFile(filepath.Join(root, "distribution", "acquire", "dry_run.go"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(dry), "enablePackage") {
		t.Error("dry_run.go must not enable — Contract A stays frozen")
	}
}

func TestAcquisitionHardeningSpecIsAccepted(t *testing.T) {
	root := moduleRoot(t)
	raw, err := os.ReadFile(filepath.Join(root, "distribution", "acquire", "HARDENING.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	for _, want := range []string{
		"# Acquisition Production Hardening",
		"**Status:** Accepted",
		"**Acceptance:** ACCEPTED",
		"Implementation: COMPLETE",
		"func Apply",
		"package:install",
		"smallest possible change",
		"Exit-code interpretation belongs to the CLI boundary",
		"No implicit acquisition → enablement",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("HARDENING.md missing %q — Acquisition hardening SPEC must stay accepted", want)
		}
	}
	for _, ban := range []string{
		"**Status:** Draft",
		"NOT ACCEPTED",
		"Implementation: LOCKED",
	} {
		if strings.Contains(text, ban) {
			t.Errorf("HARDENING.md still contains %q", ban)
		}
	}
	bannedProd := []string{
		"lifecycle.go",
		"e2e.go",
		"harden.go",
		"timeout.go",
		"cancel.go",
		"exit_code.go",
		"acquire_lifecycle.go",
		"package_lifecycle.go",
		"package_acquire_e2e.go",
	}
	dirs := []string{
		filepath.Join(root, "distribution", "acquire"),
		filepath.Join(root, "console"),
	}
	for _, dir := range dirs {
		for _, name := range bannedProd {
			if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
				t.Errorf("%s/%s — Acquisition hardening must not add a new acquisition engine", filepath.Base(dir), name)
			}
		}
	}
}

func TestAcquisitionExitCodesStayAtCLIBoundary(t *testing.T) {
	root := moduleRoot(t)
	err := filepath.WalkDir(filepath.Join(root, "distribution", "acquire"), func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		src, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		text := string(src)
		for _, ban := range []string{"ExitUsage", "ExitResolution", "ExitPlanning", "ExitAcquisition", "ExitEnablement", "ExitCanceled", "ExitRuntimeBoot", "ExitRuntimeShutdown", "ExitRuntimeCanceled", "ExitRuntimeTimeout", "CodeFromError", "type CLIError"} {
			if strings.Contains(text, ban) {
				t.Errorf("%s contains %s — exit-code logic stays at the CLI boundary", filepath.Base(path), ban)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	cli, err := os.ReadFile(filepath.Join(root, "console", "cli_exit.go"))
	if err != nil {
		t.Fatal("missing console/cli_exit.go — classified exit codes belong to the CLI")
	}
	text := string(cli)
	for _, want := range []string{"ExitUsage", "ExitResolution", "ExitPlanning", "ExitAcquisition", "ExitEnablement", "ExitCanceled", "ExitRuntimeBoot", "ExitRuntimeShutdown", "ExitRuntimeCanceled", "ExitRuntimeTimeout", "func CodeFromError"} {
		if !strings.Contains(text, want) {
			t.Errorf("cli_exit.go missing %s", want)
		}
	}
	mainSrc, err := os.ReadFile(filepath.Join(root, "cmd", "zatrano", "main.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(mainSrc), "console.CodeFromError(") {
		t.Error("cmd/zatrano must map CLI errors to exit codes")
	}
}

func TestAcquisitionJSONPresentsExistingState(t *testing.T) {
	src, err := os.ReadFile(filepath.Join(moduleRoot(t), "console", "package_acquire.go"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(src)
	for _, want := range []string{
		`json:"acquisition"`,
		`json:"enablement"`,
		`json:"inspection,omitempty"`,
		`json:"recovery,omitempty"`,
		`json:"targets,omitempty"`,
		`json:"errors,omitempty"`,
		`json:"status"`,
	} {
		if !strings.Contains(text, want) {
			t.Errorf("package_acquire.go JSON missing %s", want)
		}
	}
	for _, ban := range []string{"AcquireStateMachine", "type LifecycleService", "func Apply("} {
		if strings.Contains(text, ban) {
			t.Errorf("package_acquire.go contains %s — JSON must not invent a second state model", ban)
		}
	}
}

func TestKernelHasZeroThirdPartyDependencies(t *testing.T) {
	root := moduleRoot(t)
	mod, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(string(mod), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "require ") && line != "require (" {
			t.Errorf("go.mod has a third-party require: %s", line)
		}
	}

	fset := token.NewFileSet()
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := filepath.Base(path)
			if name == "vendor" || name == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		file, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		for _, spec := range file.Imports {
			imp := strings.Trim(spec.Path.Value, `"`)
			if !strings.Contains(imp, ".") {
				continue
			}
			if imp == "github.com/zatrano/framework/v2" || strings.HasPrefix(imp, "github.com/zatrano/framework/v2/") {
				continue
			}
			t.Errorf("%s imports third-party %s", rel, imp)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestKernelCatalogIsPrimitiveOnly(t *testing.T) {
	for _, p := range kernel.Catalog {
		if p.Layer != kernel.LayerPrimitive {
			t.Errorf("kernel.Catalog %q layer=%s want primitive", p.Name, p.Layer)
		}
	}
	for _, name := range []string{"auth", "database", "ai", "agent", "billing"} {
		if _, ok := kernel.LookupPackage(name); ok {
			t.Errorf("kernel catalog must not contain %s", name)
		}
	}
}

func TestKernelConfigHasNoPackageSchemas(t *testing.T) {
	dir := filepath.Join(moduleRoot(t), "kernel", "config")
	for _, name := range []string{"auth.go", "database.go", "session.go", "notifications.go"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			t.Errorf("package-specific config must not live in kernel: %s", name)
		}
	}
	banned := []string{"func Auth(", "func Database(", "func Session(", "func Notifications("}
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		body, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		src := string(body)
		for _, fn := range banned {
			if strings.Contains(src, fn) {
				t.Errorf("%s still defines package schema %s", filepath.Base(path), fn)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// Architecture: contracts.App stays kernel-complete; no package capability methods.
func TestContractsAppMethodFreeze(t *testing.T) {
	allow := map[string]bool{
		"BasePath": true, "Container": true, "Make": true, "Bound": true,
		"Config": true, "Router": true, "Logger": true, "Context": true,
		"Encrypter": true, "Exceptions": true, "Reports": true,
		"Environment": true, "IsProduction": true, "IsDebug": true,
		"RegisterProviders": true, "Bootstrap": true, "BootstrapContext": true,
		"Start": true, "StartContext": true, "Stop": true,
		"ServeHTTP": true, "Run": true,
		"SetHTTPBridge": true, "HTTPBridge": true,
	}
	got := interfaceMethods(t, filepath.Join(moduleRoot(t), "contracts", "app.go"), "App")
	for name := range got {
		if !allow[name] {
			t.Errorf("contracts.App grew %s — add a container From(app) helper instead", name)
		}
	}
	for name := range allow {
		if !got[name] {
			t.Errorf("contracts.App lost %s", name)
		}
	}
}

func TestRequestCoreFileStaysPrimitive(t *testing.T) {
	banned := []string{
		"Input", "All", "Only", "OnlyFilled", "Except", "ExceptFilled", "ExceptEmpty",
		"Boolean", "BooleanOK", "Integer", "IntegerOK", "Float", "FloatOK",
		"Enum", "EnumOr", "Date", "DateOr", "Filled", "Has", "Missing",
		"TransformInputs", "Merge", "Replace", "Forget", "Pull",
	}
	got := receiverMethods(t, filepath.Join(moduleRoot(t), "kernel", "http", "request.go"), "Request")
	for _, name := range banned {
		if got[name] {
			t.Errorf("kernel/http/request.go must not grow input helpers; %s belongs in input.go", name)
		}
	}
}

func interfaceMethods(t *testing.T, path, name string) map[string]bool {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]bool{}
	for _, decl := range file.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range gd.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok || ts.Name.Name != name {
				continue
			}
			iface, ok := ts.Type.(*ast.InterfaceType)
			if !ok {
				continue
			}
			for _, m := range iface.Methods.List {
				if len(m.Names) > 0 {
					out[m.Names[0].Name] = true
				}
			}
		}
	}
	if len(out) == 0 {
		t.Fatalf("no methods on %s in %s", name, path)
	}
	return out
}

func receiverMethods(t *testing.T, path, recv string) map[string]bool {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]bool{}
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv == nil || fn.Name == nil || !fn.Name.IsExported() {
			continue
		}
		if len(fn.Recv.List) == 0 {
			continue
		}
		expr := fn.Recv.List[0].Type
		if star, ok := expr.(*ast.StarExpr); ok {
			expr = star.X
		}
		id, ok := expr.(*ast.Ident)
		if !ok || id.Name != recv {
			continue
		}
		out[fn.Name.Name] = true
	}
	return out
}

func TestContractsSurfacesStayFrozen(t *testing.T) {
	root := filepath.Join(moduleRoot(t), "contracts")
	want := map[string]map[string]bool{
		"Router": {
			"Get": true, "Post": true, "Use": true, "Group": true, "Name": true,
			"Snapshot": true, "SaveCache": true,
		},
		"Route": {"As": true},
		"Container": {
			"Instance": true, "Make": true, "Bound": true,
		},
		"ConfigRepository": {
			"Get": true, "GetString": true, "GetInt": true, "GetBool": true,
			"All": true, "Load": true,
		},
		"HTTPBridge":        {"Middleware": true, "Finalize": true},
		"Provider":          {"Register": true, "Boot": true},
		"LifecycleProvider": {"Start": true, "Stop": true},
	}
	for name, allow := range want {
		file := "app.go"
		switch name {
		case "Router", "Route":
			file = "router.go"
		case "Container":
			file = "container.go"
		case "ConfigRepository":
			file = "config.go"
		}
		got := interfaceMethods(t, filepath.Join(root, file), name)
		for method := range got {
			if !allow[method] {
				t.Errorf("contracts.%s grew %s — keep the ABI minimal", name, method)
			}
		}
		for method := range allow {
			if !got[method] {
				t.Errorf("contracts.%s lost %s", name, method)
			}
		}
	}
}

func TestRequestStructHasNoSessionField(t *testing.T) {
	fields := structFields(t, filepath.Join(moduleRoot(t), "kernel", "http", "request.go"), "Request")
	if fields["session"] {
		t.Fatal("session must be a request attribute, not a Request field")
	}
}

func TestKernelHTTPDoesNotImportSessionPackage(t *testing.T) {
	dir := filepath.Join(moduleRoot(t), "kernel", "http")
	fset := token.NewFileSet()
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		file, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, spec := range file.Imports {
			imp := strings.Trim(spec.Path.Value, `"`)
			if strings.Contains(imp, "session") && strings.Contains(imp, "zatrano") {
				t.Errorf("%s imports %s", filepath.Base(path), imp)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func structFields(t *testing.T, path, name string) map[string]bool {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]bool{}
	for _, decl := range file.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range gd.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok || ts.Name.Name != name {
				continue
			}
			st, ok := ts.Type.(*ast.StructType)
			if !ok {
				continue
			}
			for _, field := range st.Fields.List {
				for _, ident := range field.Names {
					out[ident.Name] = true
				}
			}
		}
	}
	if len(out) == 0 {
		t.Fatalf("no fields on %s in %s", name, path)
	}
	return out
}
