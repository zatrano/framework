package http

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

func TestRequestCoreFileStaysPrimitive(t *testing.T) {
	banned := []string{
		"Input", "All", "Only", "OnlyFilled", "Except", "ExceptFilled", "ExceptEmpty",
		"Boolean", "BooleanOK", "Integer", "IntegerOK", "Float", "FloatOK",
		"Enum", "EnumOr", "Date", "DateOr", "Filled", "Has", "Missing",
		"TransformInputs", "Merge", "Replace", "Forget", "Pull",
	}
	got := receiverMethods(t, filepath.Join(moduleRoot(t), "core", "kernel", "http", "request.go"), "Request")
	for _, name := range banned {
		if got[name] {
			t.Errorf("core/kernel/http/request.go must not grow input helpers; %s belongs in input.go", name)
		}
	}
}

func TestKernelHTTPDoesNotImportRouting(t *testing.T) {
	root := filepath.Join(moduleRoot(t), "core", "kernel", "http")
	fset := token.NewFileSet()
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		file, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, spec := range file.Imports {
			imp := strings.Trim(spec.Path.Value, `"`)
			if strings.Contains(imp, "/kernel/routing") {
				t.Errorf("%s imports routing (cycle)", filepath.Base(path))
			}
			if strings.HasPrefix(imp, "github.com/zatrano/packages/") {
				t.Errorf("%s imports %s", filepath.Base(path), imp)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestRequestStructHasNoSessionField(t *testing.T) {
	fields := structFields(t, filepath.Join(moduleRoot(t), "core", "kernel", "http", "request.go"), "Request")
	if fields["session"] {
		t.Fatal("session must be a request attribute, not a Request field")
	}
}

func TestKernelHTTPDoesNotImportSessionPackage(t *testing.T) {
	dir := filepath.Join(moduleRoot(t), "core", "kernel", "http")
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

func TestKernelHTTPProductionImportsStayStdlibAndKernel(t *testing.T) {
	dir := filepath.Join(moduleRoot(t), "core", "kernel", "http")
	fset := token.NewFileSet()
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		file, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, spec := range file.Imports {
			imp := strings.Trim(spec.Path.Value, `"`)
			if strings.HasPrefix(imp, "github.com/zatrano/packages") {
				t.Errorf("%s imports %s", filepath.Base(path), imp)
			}
			if strings.HasPrefix(imp, "github.com/") && !strings.HasPrefix(imp, "github.com/zatrano/framework/") && !strings.HasPrefix(imp, "github.com/zatrano/rawhttp") && !strings.HasPrefix(imp, "github.com/zatrano/canvas") {
				t.Errorf("%s third-party import %s", filepath.Base(path), imp)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
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
