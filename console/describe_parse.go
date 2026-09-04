package console

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
)

func frameworkModuleRoot() (string, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("console: cannot resolve caller path")
	}
	dir := filepath.Dir(file)
	for i := 0; i < 12; i++ {
		b, err := os.ReadFile(filepath.Join(dir, "go.mod"))
		if err == nil && bytes.Contains(b, []byte("module github.com/zatrano/framework")) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("console: framework module root not found")
}

func modulePath(root string) (string, error) {
	b, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module ")), nil
		}
	}
	return "", fmt.Errorf("console: module path not found in go.mod")
}

func parseContractInterfaces(dir string) (map[string]ContractType, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	out := map[string]ContractType{}
	fset := token.NewFileSet()
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return nil, err
		}
		rel := filepath.ToSlash(filepath.Join("contracts", e.Name()))
		for _, decl := range file.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok || gd.Tok != token.TYPE {
				continue
			}
			for _, spec := range gd.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				iface, ok := ts.Type.(*ast.InterfaceType)
				if !ok {
					continue
				}
				ct := ContractType{
					Name:    ts.Name.Name,
					File:    rel,
					Methods: interfaceMethods(fset, iface),
				}
				out[ct.Name] = ct
			}
		}
	}
	return out, nil
}

func interfaceMethods(fset *token.FileSet, iface *ast.InterfaceType) []ContractMethod {
	if iface == nil || iface.Methods == nil {
		return nil
	}
	out := make([]ContractMethod, 0, len(iface.Methods.List))
	for _, field := range iface.Methods.List {
		if len(field.Names) == 0 {
			emb := compactExpr(fset, field.Type)
			out = append(out, ContractMethod{Name: emb, Signature: emb})
			continue
		}
		ft, ok := field.Type.(*ast.FuncType)
		for _, name := range field.Names {
			sig := name.Name
			if ok {
				sig = name.Name + strings.TrimPrefix(compactExpr(fset, ft), "func")
			}
			out = append(out, ContractMethod{Name: name.Name, Signature: sig})
		}
	}
	return out
}

func parseCatalogLayers(path string) ([]CatalogLayerReport, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	var layers []CatalogLayerReport
	for _, decl := range file.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.CONST {
			continue
		}
		var lastType ast.Expr
		for _, spec := range gd.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok || len(vs.Names) == 0 || len(vs.Values) == 0 {
				continue
			}
			if vs.Type != nil {
				lastType = vs.Type
			}
			ident, ok := lastType.(*ast.Ident)
			if !ok || ident.Name != "Layer" {
				continue
			}
			lit, ok := vs.Values[0].(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				continue
			}
			val, err := strconv.Unquote(lit.Value)
			if err != nil {
				return nil, err
			}
			role := strings.TrimSpace(vs.Doc.Text())
			if role == "" && gd.Doc != nil && len(gd.Specs) == 1 {
				role = strings.TrimSpace(gd.Doc.Text())
			}
			layers = append(layers, CatalogLayerReport{
				Constant: vs.Names[0].Name,
				Name:     val,
				Role:     strings.TrimSuffix(role, "\n"),
			})
		}
	}
	if len(layers) == 0 {
		return nil, fmt.Errorf("console: no Layer constants in %s", path)
	}
	return layers, nil
}

func parseNamedFuncs(path string, names []string) ([]ContractMethod, error) {
	want := map[string]bool{}
	for _, n := range names {
		want[n] = true
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return nil, err
	}
	found := map[string]ContractMethod{}
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv != nil || fn.Name == nil || !want[fn.Name.Name] {
			continue
		}
		sig := fn.Name.Name + strings.TrimPrefix(compactExpr(fset, fn.Type), "func")
		found[fn.Name.Name] = ContractMethod{Name: fn.Name.Name, Signature: sig}
	}
	out := make([]ContractMethod, 0, len(names))
	for _, n := range names {
		m, ok := found[n]
		if !ok {
			return nil, fmt.Errorf("console: function %s not found in %s", n, path)
		}
		out = append(out, m)
	}
	return out, nil
}

func parseNamedInterface(dir, name string) (ContractType, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ContractType{}, err
	}
	fset := token.NewFileSet()
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return ContractType{}, err
		}
		for _, decl := range file.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok || gd.Tok != token.TYPE {
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
				rel := filepath.ToSlash(filepath.Join(filepath.Base(dir), e.Name()))
				return ContractType{
					Name:    name,
					File:    rel,
					Methods: interfaceMethods(fset, iface),
				}, nil
			}
		}
	}
	return ContractType{}, fmt.Errorf("console: interface %s not found in %s", name, dir)
}

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

func compactExpr(fset *token.FileSet, e ast.Expr) string {
	if e == nil {
		return ""
	}
	var buf bytes.Buffer
	cfg := printer.Config{Mode: printer.UseSpaces, Tabwidth: 4}
	if err := cfg.Fprint(&buf, fset, e); err != nil {
		return ""
	}
	return strings.Join(strings.Fields(buf.String()), " ")
}
