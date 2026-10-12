package contracts

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestContractsAppMethodFreeze(t *testing.T) {
	allow := map[string]bool{
		"BasePath": true, "Container": true, "Make": true, "Bound": true,
		"Config": true, "Router": true, "Logger": true, "Context": true,
		"Encrypter": true, "Exceptions": true, "Reports": true,
		"Environment": true, "IsProduction": true, "IsDebug": true,
		"RegisterProviders": true, "Bootstrap": true, "BootstrapContext": true,
		"Start": true, "StartContext": true, "Stop": true,
		"Handle": true, "Run": true,
		"SetHTTPBridge": true, "HTTPBridge": true,
	}
	got := interfaceMethods(t, filepath.Join(moduleRoot(t), "core", "contracts", "app.go"), "App")
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

func TestContractsSurfacesStayFrozen(t *testing.T) {
	root := filepath.Join(moduleRoot(t), "core", "contracts")
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
