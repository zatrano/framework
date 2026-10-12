package acquire

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
)

func TestAcquirePlanStructFreeze(t *testing.T) {
	allow := map[string]bool{
		"Schema": true, "Name": true, "Import": true, "Module": true,
		"Query": true, "Selected": true, "Kind": true, "Heavy": true,
	}
	got := structFields(t, filepath.Join(moduleRoot(t), "core", "distribution", "acquire", "plan.go"), "Plan")
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
	raw, err := os.ReadFile(filepath.Join(root, "core", "distribution", "acquire", "APPLY.md"))
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
	err = filepath.WalkDir(filepath.Join(root, "core", "distribution", "acquire"), func(path string, d fs.DirEntry, err error) error {
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
	raw, err := os.ReadFile(filepath.Join(root, "core", "distribution", "acquire", "ORCHESTRATION.md"))
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
	dir := filepath.Join(root, "core", "distribution", "acquire")
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
	dir := filepath.Join(root, "core", "distribution", "acquire")
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
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
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
	dir := filepath.Join(moduleRoot(t), "core", "distribution", "acquire")
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
	file, err := parser.ParseFile(fset, filepath.Join(moduleRoot(t), "core", "distribution", "acquire", "plan.go"), nil, 0)
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
	path := filepath.Join(moduleRoot(t), "core", "distribution", "acquire", "plan.go")
	allowImport := map[string]bool{
		"fmt": true, "strings": true, "sort": true,
		"github.com/zatrano/framework/v3/core/distribution/registry": true,
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

func TestAcquisitionHardeningDoesNotAddEngineFiles(t *testing.T) {
	root := moduleRoot(t)
	raw, err := os.ReadFile(filepath.Join(root, "core", "distribution", "acquire", "HARDENING.md"))
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
		filepath.Join(root, "core", "distribution", "acquire"),
		filepath.Join(root, "core", "console"),
	}
	for _, dir := range dirs {
		for _, name := range bannedProd {
			if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
				t.Errorf("%s/%s — Acquisition hardening must not add a new acquisition engine", filepath.Base(dir), name)
			}
		}
	}
}

func moduleRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	dir := filepath.Dir(file)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("module root")
		}
		dir = parent
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
