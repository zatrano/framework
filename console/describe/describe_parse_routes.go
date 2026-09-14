package describe

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func parseSelfRegistration(root, modPath string) (SelfRegistrationInfo, error) {
	path := filepath.Join(root, "bootstrap", "addons", "registry.go")
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return SelfRegistrationInfo{}, err
	}
	info := SelfRegistrationInfo{
		Package:             modPath + "/bootstrap/addons",
		MetaType:            "Meta",
		FactoryField:        "Factory",
		ConsumerBlankImport: true,
	}
	for _, imp := range file.Imports {
		p, _ := strconv.Unquote(imp.Path.Value)
		if strings.Contains(p, "/packages/") || strings.Contains(p, "github.com/zatrano/packages") {
			info.RegistryImportsAddons = true
		}
	}
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.GenDecl:
			if d.Tok != token.TYPE {
				continue
			}
			for _, spec := range d.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok || ts.Name.Name != "Meta" {
					continue
				}
				st, ok := ts.Type.(*ast.StructType)
				if !ok || st.Fields == nil {
					continue
				}
				for _, f := range st.Fields.List {
					for _, n := range f.Names {
						info.MetaFields = append(info.MetaFields, n.Name)
						if n.Name == "Factory" {
							info.FactoryReturns = compactExpr(fset, f.Type)
						}
					}
				}
			}
		case *ast.FuncDecl:
			if d.Recv != nil || d.Name == nil {
				continue
			}
			sig := d.Name.Name + strings.TrimPrefix(compactExpr(fset, d.Type), "func")
			switch d.Name.Name {
			case "Register":
				info.Register = sig
				if d.Doc != nil && strings.Contains(d.Doc.Text(), "init()") {
					info.RegisterCalledFrom = "init"
				}
			case "Select":
				info.Select = sig
			case "Lookup":
				info.Lookup = sig
			case "Available":
				info.Available = sig
			}
		}
	}
	if info.Register == "" || len(info.MetaFields) == 0 {
		return SelfRegistrationInfo{}, fmt.Errorf("console: incomplete addon registry parse in %s", path)
	}
	if info.RegisterCalledFrom == "" {
		info.RegisterCalledFrom = "init"
	}
	return info, nil
}

var sampleRouteCalls = map[string]bool{
	"Get": true, "Post": true, "Put": true, "Patch": true, "Delete": true,
	"Head": true, "Options": true, "Any": true, "Match": true,
	"RegisterWeb": true, "RegisterAPI": true, "Controller": true,
}

func parseSampleRoutes(scanRoot string) ([]SampleRoute, error) {
	empty := []SampleRoute{}
	if scanRoot == "" {
		return empty, nil
	}
	routesDir := filepath.Join(scanRoot, "app", "routes")
	st, err := os.Stat(routesDir)
	if err != nil || !st.IsDir() {
		return empty, nil
	}
	var out []SampleRoute
	err = filepath.WalkDir(routesDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			base := d.Name()
			if base == "testdata" || base == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		routes, err := sampleRoutesInFile(scanRoot, path)
		if err != nil {
			return err
		}
		out = append(out, routes...)
		return nil
	})
	if err != nil {
		return nil, err
	}
	if out == nil {
		out = []SampleRoute{}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].File != out[j].File {
			return out[i].File < out[j].File
		}
		return out[i].Line < out[j].Line
	})
	return out, nil
}

func sampleRoutesInFile(scanRoot, path string) ([]SampleRoute, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return nil, err
	}
	rel, err := filepath.Rel(scanRoot, path)
	if err != nil {
		rel = path
	}
	rel = filepath.ToSlash(rel)
	group := ""
	switch {
	case strings.Contains(rel, "/routes/web/"):
		group = "web"
	case strings.Contains(rel, "/routes/api/"):
		group = "api"
	}
	var out []SampleRoute
	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		name := callSelName(call.Fun)
		if !sampleRouteCalls[name] {
			return true
		}
		sr := SampleRoute{
			File:  rel,
			Line:  fset.Position(call.Pos()).Line,
			Group: group,
			Call:  name,
			Path:  firstStringArg(call),
		}
		out = append(out, sr)
		return true
	})
	return out, nil
}

func callSelName(fun ast.Expr) string {
	switch x := fun.(type) {
	case *ast.Ident:
		return x.Name
	case *ast.SelectorExpr:
		return x.Sel.Name
	case *ast.IndexExpr:
		return callSelName(x.X)
	case *ast.IndexListExpr:
		return callSelName(x.X)
	default:
		return ""
	}
}

func firstStringArg(call *ast.CallExpr) string {
	for _, arg := range call.Args {
		if lit, ok := arg.(*ast.BasicLit); ok && lit.Kind == token.STRING {
			s, err := strconv.Unquote(lit.Value)
			if err == nil {
				return s
			}
		}
	}
	return ""
}
