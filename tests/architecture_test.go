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
	if version != "2.0.12" {
		t.Fatalf("VERSION=%q want 2.0.12", version)
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

// Phase 1 freeze: kernel must not import github.com/zatrano/packages.
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

func TestRegistryDoesNotImportConsole(t *testing.T) {
	dir := filepath.Join(moduleRoot(t), "registry")
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
		`"github.com/zatrano/framework/v2/registry"`,
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

func TestPhase6SearchHitHasNoSelectionFields(t *testing.T) {
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

func TestPhase7PlanStructFreeze(t *testing.T) {
	allow := map[string]bool{
		"Schema": true, "Name": true, "Import": true, "Module": true,
		"Query": true, "Selected": true, "Kind": true, "Heavy": true,
	}
	got := structFields(t, filepath.Join(moduleRoot(t), "acquire", "plan.go"), "Plan")
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

func TestPhase8ApplySpecExistsWithoutImplementation(t *testing.T) {
	root := moduleRoot(t)
	raw, err := os.ReadFile(filepath.Join(root, "acquire", "APPLY.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	for _, want := range []string{
		"Draft — specification only",
		"not authorized",
		"No automatic tidy",
		"package:install",
		"zatrano.lock",
		"exec.Command",
		"fail-fast",
		"rollback guaranteed",
		"Enabled ∩ Imported",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("APPLY.md missing %q — Phase 8 is SPEC only", want)
		}
	}
	err = filepath.WalkDir(filepath.Join(root, "acquire"), func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		base := strings.ToLower(filepath.Base(path))
		if base == "apply.go" || (strings.HasPrefix(base, "apply_") && strings.HasSuffix(base, ".go")) {
			t.Errorf("%s — Apply implementation is not authorized", filepath.Base(path))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestPhase7AcquireExportsStayPlanOnly(t *testing.T) {
	allowFn := map[string]bool{"FromResult": true, "Targets": true}
	allowMethod := map[string]bool{"GoGetArg": true}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filepath.Join(moduleRoot(t), "acquire", "plan.go"), nil, 0)
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
				t.Errorf("Plan grew method %s — Apply stays out of Phase 7", fn.Name.Name)
			}
			continue
		}
		if !allowFn[fn.Name.Name] {
			t.Errorf("acquire grew %s — Phase 7 exports are FromResult and Targets", fn.Name.Name)
		}
	}
}

func TestAcquirePlanLayerDoesNotResolveOrApply(t *testing.T) {
	dir := filepath.Join(moduleRoot(t), "acquire")
	allowImport := map[string]bool{
		"fmt": true, "strings": true, "sort": true,
		"github.com/zatrano/framework/v2/registry": true,
	}
	bannedFn := map[string]bool{
		"Apply": true, "Install": true, "Tidy": true, "Download": true,
		"compareSemver": true, "latestCompatible": true, "MeetsFrameworkMin": true,
	}
	fset := token.NewFileSet()
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return err
		}
		base := filepath.Base(path)
		for _, spec := range file.Imports {
			imp := strings.Trim(spec.Path.Value, `"`)
			if !allowImport[imp] {
				t.Errorf("%s imports %s — Plan layer stays a pure translation", base, imp)
			}
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Name == nil {
				continue
			}
			if bannedFn[fn.Name.Name] {
				t.Errorf("%s defines %s — Apply/resolution stay out of acquire", base, fn.Name.Name)
			}
		}
		ast.Inspect(file, func(n ast.Node) bool {
			sel, ok := n.(*ast.SelectorExpr)
			if !ok || sel.Sel == nil {
				return true
			}
			if sel.Sel.Name == "Resolve" || sel.Sel.Name == "Search" || sel.Sel.Name == "Lookup" {
				t.Errorf("%s calls Index.%s — FromResult must not re-resolve", base, sel.Sel.Name)
			}
			return true
		})
		src, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		text := string(src)
		for _, ban := range []string{"os.WriteFile", "os.Create", "os.Mkdir", "exec.Command", "go mod tidy", "go mod edit"} {
			if strings.Contains(text, ban) {
				t.Errorf("%s contains %q — Apply is deferred", base, ban)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestConsoleDoesNotImportAcquireYet(t *testing.T) {
	dir := filepath.Join(moduleRoot(t), "console")
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
			if strings.HasSuffix(imp, "/acquire") {
				t.Errorf("%s imports acquire — Plan layer is frozen; Apply/CLI add is later (%s)", filepath.Base(path), imp)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
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

// Phase 1 freeze: contracts.App stays kernel-complete; no package capability methods.
func TestContractsAppMethodFreeze(t *testing.T) {
	allow := map[string]bool{
		"BasePath": true, "Container": true, "Make": true, "Bound": true,
		"Config": true, "Router": true, "Logger": true, "Context": true,
		"Encrypter": true, "Exceptions": true, "Reports": true,
		"Environment": true, "IsProduction": true, "IsDebug": true,
		"RegisterProviders": true, "Bootstrap": true, "Start": true, "Stop": true,
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
