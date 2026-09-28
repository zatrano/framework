package doctor

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

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

type providerShape struct {
	File     string
	Line     int
	Register bool
	Boot     bool
}

func collectProviderShapes(types map[string]*providerShape, rel string, fset *token.FileSet, file *ast.File) {
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.GenDecl:
			if d.Tok != token.TYPE {
				continue
			}
			for _, spec := range d.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				if _, ok := ts.Type.(*ast.StructType); !ok {
					continue
				}
				name := ts.Name.Name
				if types[name] == nil {
					types[name] = &providerShape{File: rel, Line: fset.Position(ts.Pos()).Line}
				}
			}
		case *ast.FuncDecl:
			if d.Recv == nil || d.Name == nil || (d.Name.Name != "Register" && d.Name.Name != "Boot") {
				continue
			}
			recv := recvTypeName(d.Recv)
			if recv == "" {
				continue
			}
			if types[recv] == nil {
				types[recv] = &providerShape{File: rel, Line: fset.Position(d.Pos()).Line}
			}
			if d.Name.Name == "Register" {
				types[recv].Register = true
			} else {
				types[recv].Boot = true
			}
		}
	}
}

func recvTypeName(recv *ast.FieldList) string {
	if recv == nil || len(recv.List) == 0 {
		return ""
	}
	expr := recv.List[0].Type
	if star, ok := expr.(*ast.StarExpr); ok {
		expr = star.X
	}
	id, ok := expr.(*ast.Ident)
	if !ok {
		return ""
	}
	return id.Name
}

func walkConsumerGo(root string, fn func(rel, abs string, fset *token.FileSet, file *ast.File)) error {
	roots := []string{
		"app", "cmd", "bootstrap", "routes", "application", "internal",
		"domain", "handlers", "dtos", "dto", "usecases", "usecase", "entities", "actions",
		"interactors", "interactor",
	}
	for _, name := range roots {
		dir := filepath.Join(root, name)
		if st, err := os.Stat(dir); err != nil || !st.IsDir() {
			continue
		}
		if err := walkDirGo(dir, root, fn); err != nil {
			return err
		}
	}
	return nil
}

func walkDirGo(dir, root string, fn func(rel, abs string, fset *token.FileSet, file *ast.File)) error {
	return filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case "testdata", "vendor", "node_modules", ".git":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			rel = path
		}
		fn(filepath.ToSlash(rel), path, fset, file)
		return nil
	})
}
