package pkgmanager

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

func TestCLIDoesNotReimplementRegistryResolution(t *testing.T) {
	path := filepath.Join(moduleRoot(t), "core", "console", "pkgmanager", "package_registry.go")
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(src)
	for _, want := range []string{
		`"github.com/zatrano/framework/v3/core/distribution/registry"`,
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
	err = filepath.WalkDir(filepath.Join(moduleRoot(t), "core", "console"), func(path string, d fs.DirEntry, err error) error {
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
	root := filepath.Join(moduleRoot(t), "core", "console", "pkgmanager")
	files := []string{"package_cmd.go", "package_doctor.go", "package_wire.go", "package_env.go", "package_enable.go"}
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
	got := structFields(t, filepath.Join(moduleRoot(t), "core", "console", "pkgmanager", "package_registry.go"), "searchHit")
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

func TestCLIAcquireGate(t *testing.T) {
	root := moduleRoot(t)
	path := filepath.Join(root, "core", "console", "pkgmanager", "package_acquire.go")
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatal("missing core/console/pkgmanager/package_acquire.go — Contract B CLI acquisition")
	}
	reportSrc, err := os.ReadFile(filepath.Join(root, "core", "console", "pkgmanager", "package_acquire_report.go"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(src) + "\n" + string(reportSrc)
	for _, want := range []string{
		`"github.com/zatrano/framework/v3/core/distribution/acquire"`,
		`"github.com/zatrano/framework/v3/core/distribution/registry"`,
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
	cmdSrc, err := os.ReadFile(filepath.Join(root, "core", "console", "pkgmanager", "package_cmd.go"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(cmdSrc), "/acquire") {
		t.Error("package_cmd.go must not import acquire — package:install stays enablement")
	}
}

func TestExplicitEnablementAfterAcquire(t *testing.T) {
	root := moduleRoot(t)
	raw, err := os.ReadFile(filepath.Join(root, "core", "distribution", "acquire", "ORCHESTRATION.md"))
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
		if _, err := os.Stat(filepath.Join(root, "core", "distribution", "acquire", name)); err == nil {
			t.Errorf("%s — Contract C must not add an acquire-side enablement engine", name)
		}
		if _, err := os.Stat(filepath.Join(root, "core", "console", name)); err == nil {
			t.Errorf("core/console/%s — Contract C stays in package_acquire.go", name)
		}
	}
	cmdSrc, err := os.ReadFile(filepath.Join(root, "core", "console", "pkgmanager", "package_cmd.go"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(cmdSrc), "/acquire") {
		t.Error("package_cmd.go must not import acquire — package:install stays enablement")
	}
	enableSrc, err := os.ReadFile(filepath.Join(root, "core", "console", "pkgmanager", "package_enable.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(cmdSrc), `return "package:install"`) || !strings.Contains(string(enableSrc), "func enablePackage(") {
		t.Error("package:install must remain enablement")
	}
	dry, err := os.ReadFile(filepath.Join(root, "core", "distribution", "acquire", "dry_run.go"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(dry), "enablePackage") {
		t.Error("dry_run.go must not enable — Contract A stays frozen")
	}
}

func TestAcquisitionJSONPresentsExistingState(t *testing.T) {
	src, err := os.ReadFile(filepath.Join(moduleRoot(t), "core", "console", "pkgmanager", "package_acquire_report.go"))
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
			t.Errorf("package_acquire_report.go JSON missing %s", want)
		}
	}
	for _, ban := range []string{"AcquireStateMachine", "type LifecycleService", "func Apply("} {
		if strings.Contains(text, ban) {
			t.Errorf("package_acquire_report.go contains %s — JSON must not invent a second state model", ban)
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
