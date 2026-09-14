package describe

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

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
