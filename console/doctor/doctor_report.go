package doctor

import (
	"go/ast"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func checkORMArchitecture(root string) ([]Finding, error) {
	var out []Finding
	err := walkConsumerGo(root, func(rel, abs string, fset *token.FileSet, file *ast.File) {
		imports := importPathByName(file)
		if !importsORM(imports) {
			return
		}
		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			if callSelName(call.Fun) != "With" || len(call.Args) == 0 {
				return true
			}
			if !ormCall(call.Fun, imports) {
				return true
			}
			lit, ok := call.Args[0].(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return true
			}
			out = append(out, Finding{
				Rule:     "APP-ORM-001",
				Check:    "orm",
				Severity: "error",
				File:     rel,
				Line:     fset.Position(call.Pos()).Line,
				Found:    "With(" + lit.Value + ")",
				Why:      "String eager loads are not an ORM API. Use With(orm.EagerHasMany[...]) with an explicit FK.",
				How:      "Replace With(\"name\") with a typed loader function.",
				See:      archSee + " §L",
			})
			return true
		})
	})
	return out, err
}

func checkValidationArchitecture(root string) ([]Finding, error) {
	if !hasUniqueOrExistsRules(root) {
		return nil, nil
	}
	if addonEnabled(root, "database") {
		return nil, nil
	}
	var out []Finding
	dir := filepath.Join(root, "app", "http", "requests")
	if st, err := os.Stat(dir); err != nil || !st.IsDir() {
		return []Finding{{
			Rule:     "APP-VAL-001",
			Check:    "validation",
			Severity: "error",
			File:     "bootstrap/enabled.go",
			Found:    "unique/exists rules without database enabled",
			Why:      "unique/exists cannot establish a database fact without the database package (ADR-0010).",
			How:      "Enable the database package before relying on unique/exists, or remove those rules.",
			See:      "https://zatrano.com/docs/application-engineering/adr-0010-unique-exists-fail-open",
		}}, nil
	}
	err := walkDirGo(dir, root, func(rel, abs string, fset *token.FileSet, file *ast.File) {
		ast.Inspect(file, func(n ast.Node) bool {
			lit, ok := n.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return true
			}
			s, err := strconv.Unquote(lit.Value)
			if err != nil {
				return true
			}
			if !ruleHasPresence(s) {
				return true
			}
			out = append(out, Finding{
				Rule:     "APP-VAL-001",
				Check:    "validation",
				Severity: "error",
				File:     rel,
				Line:     fset.Position(lit.Pos()).Line,
				Found:    s,
				Why:      "unique/exists fail closed when the PresenceChecker is unbound; enable database so lookups can run (ADR-0010).",
				How:      "Enable database in bootstrap/enabled.go, or drop unique/exists until the checker is bound.",
				See:      "https://zatrano.com/docs/application-engineering/adr-0010-unique-exists-fail-open",
			})
			return true
		})
	})
	return out, err
}

func hasUniqueOrExistsRules(root string) bool {
	found := false
	dir := filepath.Join(root, "app", "http", "requests")
	_ = walkDirGo(dir, root, func(rel, abs string, fset *token.FileSet, file *ast.File) {
		ast.Inspect(file, func(n ast.Node) bool {
			lit, ok := n.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return true
			}
			s, err := strconv.Unquote(lit.Value)
			if err != nil {
				return true
			}
			if ruleHasPresence(s) {
				found = true
			}
			return true
		})
	})
	return found
}

func ruleHasPresence(s string) bool {
	for _, part := range strings.Split(s, "|") {
		p := strings.TrimSpace(part)
		if strings.HasPrefix(p, "unique:") || strings.HasPrefix(p, "exists:") || p == "unique" || p == "exists" {
			return true
		}
	}
	return false
}

func routerVerbOnAppRouter(rel string, fset *token.FileSet, file *ast.File, imports map[string]string) []Finding {
	var out []Finding
	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		switch sel.Sel.Name {
		case "Put", "Patch", "Delete", "Resource":
		default:
			return true
		}
		inner, ok := sel.X.(*ast.CallExpr)
		if !ok || callSelName(inner.Fun) != "Router" {
			return true
		}
		out = append(out, Finding{
			Rule:     "APP-ROUTE-002",
			Check:    "routes",
			Severity: "error",
			File:     rel,
			Line:     fset.Position(call.Pos()).Line,
			Found:    "app.Router()." + sel.Sel.Name,
			Why:      "contracts.Router has no Put/Patch/Delete/Resource. Application routes use routing.From(app).",
			How:      "Call routing.From(app).Put/Patch/Delete/Resource from app/routes/{web,api}.",
			See:      archSee + " §N",
		})
		return true
	})
	_ = imports
	return out
}

func importPathByName(file *ast.File) map[string]string {
	out := map[string]string{}
	for _, spec := range file.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			continue
		}
		name := ""
		if spec.Name != nil {
			name = spec.Name.Name
		} else {
			name = filepath.Base(path)
		}
		out[name] = path
	}
	return out
}

func hasDotImport(imports map[string]string, suffix string) bool {
	path, ok := imports["."]
	return ok && strings.Contains(path, suffix)
}

func importsORM(imports map[string]string) bool {
	for _, p := range imports {
		if ormImportPath(p) {
			return true
		}
	}
	return false
}

func ormImportPath(p string) bool {
	return p == "github.com/zatrano/packages/orm" || (strings.HasSuffix(p, "/orm") && strings.Contains(p, "zatrano/packages"))
}

func funcCallsHTTP(fn *ast.FuncDecl, imports map[string]string, name string) bool {
	found := false
	ast.Inspect(fn, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		if !callNamed(call.Fun, name) {
			return true
		}
		if httpCall(call.Fun, imports) {
			found = true
		}
		return true
	})
	return found
}

func funcCallsORM(fn *ast.FuncDecl, imports map[string]string, names ...string) bool {
	want := map[string]bool{}
	for _, n := range names {
		want[n] = true
	}
	found := false
	ast.Inspect(fn, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel := callSelName(call.Fun)
		if !want[sel] {
			return true
		}
		if ormCall(call.Fun, imports) {
			found = true
		}
		return true
	})
	return found
}

func funcCallsValidation(fn *ast.FuncDecl, imports map[string]string, name string) bool {
	found := false
	ast.Inspect(fn, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		if callSelName(call.Fun) != name {
			return true
		}
		if validationCall(call.Fun, imports) {
			found = true
		}
		return true
	})
	return found
}

func callNamed(fun ast.Expr, name string) bool {
	return callSelName(fun) == name
}

func httpCall(fun ast.Expr, imports map[string]string) bool {
	return callFromPackage(fun, imports, "/kernel/http")
}

func ormCall(fun ast.Expr, imports map[string]string) bool {
	return callFromPackage(fun, imports, "/orm")
}

func validationCall(fun ast.Expr, imports map[string]string) bool {
	return callFromPackage(fun, imports, "/validation")
}

func callFromPackage(fun ast.Expr, imports map[string]string, suffix string) bool {
	switch x := fun.(type) {
	case *ast.Ident:
		return hasDotImport(imports, suffix)
	case *ast.SelectorExpr:
		id, ok := x.X.(*ast.Ident)
		if !ok {
			return callFromPackage(x.X, imports, suffix)
		}
		path := imports[id.Name]
		return strings.Contains(path, suffix)
	case *ast.IndexExpr:
		return callFromPackage(x.X, imports, suffix)
	case *ast.IndexListExpr:
		return callFromPackage(x.X, imports, suffix)
	case *ast.CallExpr:
		return callFromPackage(x.Fun, imports, suffix)
	default:
		return false
	}
}

func validationEnabled(root string) bool {
	return addonEnabled(root, "validation")
}

func addonEnabled(root, name string) bool {
	body, err := os.ReadFile(filepath.Join(root, "bootstrap", "enabled.go"))
	if err != nil {
		return false
	}
	for _, n := range parseEnabledAddons(string(body)) {
		if n == name {
			return true
		}
	}
	return false
}
