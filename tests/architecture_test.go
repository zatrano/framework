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

	"github.com/zatrano/framework/kernel"
)

func moduleRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), ".."))
}

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
}

func TestContractsAppMethodFreeze(t *testing.T) {
	allow := map[string]bool{
		"BasePath": true, "Container": true, "Make": true, "Bound": true,
		"Config": true, "Router": true, "Logger": true, "Context": true,
		"Encrypter": true, "Exceptions": true, "Reports": true,
		"SetMigrations": true, "Migrations": true, "SetSeeders": true, "Seeders": true,
		"Environment": true, "IsProduction": true, "IsDebug": true,
		"RegisterProviders": true, "Bootstrap": true, "ServeHTTP": true, "Run": true,
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
