package doctor

import (
	"fmt"
	"go/ast"
	"go/token"
	"strconv"
	"strings"
)

// checkHTTPLifecycle flags request-scoped rawhttp / net/http server leaks in app code.
// Kernel Body() returns an owned copy; Ctx() and lasting *Request pointers remain unsafe.
func checkHTTPLifecycle(root string) ([]Finding, error) {
	var out []Finding
	err := walkConsumerGo(root, func(rel, abs string, fset *token.FileSet, file *ast.File) {
		pkgAlias := map[string]string{}
		hasKernelHTTP := false
		for _, spec := range file.Imports {
			path, err := strconv.Unquote(spec.Path.Value)
			if err != nil {
				continue
			}
			name := importLocalName(spec, path)
			pkgAlias[name] = path
			line := fset.Position(spec.Pos()).Line

			if path == "github.com/zatrano/rawhttp" {
				out = append(out, Finding{
					Rule:     "APP-HTTP-001",
					Check:    "http-lifecycle",
					Severity: "error",
					File:     rel,
					Line:     line,
					Found:    "import github.com/zatrano/rawhttp",
					Why:      "App code must not use *rawhttp.Ctx (API model A). Request-scoped Ctx is invalid after Handle→Commit.",
					How:      "Use kernel *http.Request / *http.Response. For async work pass owned Method/Path/Body copies or a DTO — never *Ctx.",
				})
			}
			if path == "github.com/zatrano/packages/database" || strings.HasPrefix(path, "github.com/zatrano/packages/database/") {
				out = append(out, Finding{
					Rule:     "APP-DB-001",
					Check:    "http-lifecycle",
					Severity: "error",
					File:     rel,
					Line:     line,
					Found:    "import " + path,
					Why:      "packages/database was removed from V3 (v2 query builder). Use packages/db/* adapters.",
					How:      "Import github.com/zatrano/packages/db/postgres (or mysql/mariadb/sqlite/sqlserver/oracle). See packages/db/README.md.",
				})
			}
			if path == "github.com/zatrano/packages/factory" || strings.HasPrefix(path, "github.com/zatrano/packages/factory/") {
				out = append(out, Finding{
					Rule:     "APP-FAC-001",
					Check:    "http-lifecycle",
					Severity: "error",
					File:     rel,
					Line:     line,
					Found:    "import " + path,
					Why:      "packages/factory was removed from V3 (v2 orm model factories). V3 apps use database/seeders + sqlc.",
					How:      "Remove the factory import; seed via app SQL/sqlc repositories.",
				})
			}
			if path == "github.com/zatrano/framework/v3/core/kernel/http" || strings.HasSuffix(path, "/core/kernel/http") {
				hasKernelHTTP = true
			}
		}

		inHandlers := strings.Contains(rel, "/http/handlers/") || strings.HasPrefix(rel, "app/http/handlers/")

		ast.Inspect(file, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.SelectorExpr:
				if x.Sel == nil {
					return true
				}
				pkg, ok := x.X.(*ast.Ident)
				if !ok {
					return true
				}
				if pkgAlias[pkg.Name] != "net/http" {
					return true
				}
				switch x.Sel.Name {
				case "ResponseWriter", "Handler", "HandlerFunc", "ServeMux":
					out = append(out, Finding{
						Rule:     "APP-HTTP-002",
						Check:    "http-lifecycle",
						Severity: "error",
						File:     rel,
						Line:     fset.Position(x.Pos()).Line,
						Found:    fmt.Sprintf("net/http.%s", x.Sel.Name),
						Why:      "V3 server path is rawhttp; net/http ResponseWriter/ServeHTTP must not appear in app handlers.",
						How:      "Return *http.Response from kernel handlers; use Application.Handle / Commit.",
					})
				}
			case *ast.CallExpr:
				sel, ok := x.Fun.(*ast.SelectorExpr)
				if !ok || sel.Sel == nil {
					return true
				}
				line := fset.Position(x.Pos()).Line
				if sel.Sel.Name == "ServeHTTP" {
					if pkg, ok := sel.X.(*ast.Ident); ok && pkgAlias[pkg.Name] == "net/http" {
						out = append(out, Finding{
							Rule:     "APP-HTTP-002",
							Check:    "http-lifecycle",
							Severity: "error",
							File:     rel,
							Line:     line,
							Found:    "net/http.ServeHTTP",
							Why:      "ServeHTTP is not part of the V3/rawhttp server path.",
							How:      "Use Application.Handle at the kernel boundary only.",
						})
					}
				}
				if sel.Sel.Name == "Ctx" && len(x.Args) == 0 && hasKernelHTTP && inHandlers {
					out = append(out, Finding{
						Rule:     "APP-HTTP-003",
						Check:    "http-lifecycle",
						Severity: "error",
						File:     rel,
						Line:     line,
						Found:    ".Ctx()",
						Why:      "*rawhttp.Ctx is request-scoped (zero-copy / pool). Storing or using it after Commit is undefined.",
						How:      "Use Method(), Path(), Header(), Body() (owned copy). For background jobs copy into a DTO; do not keep *http.Request or Ctx().",
					})
				}
			case *ast.ValueSpec:
				if !isPackageLevel(file, n) {
					return true
				}
				for _, name := range x.Names {
					if name == nil || name.Name == "_" {
						continue
					}
					if x.Type == nil || !isKernelHTTPRequestPtr(x.Type, pkgAlias) {
						continue
					}
					out = append(out, Finding{
						Rule:     "APP-HTTP-004",
						Check:    "http-lifecycle",
						Severity: "error",
						File:     rel,
						Line:     fset.Position(name.Pos()).Line,
						Found:    fmt.Sprintf("package-level %s *http.Request", name.Name),
						Why:      "*http.Request holds a request-scoped rawhttp.Ctx; package-level storage outlives Handle→Commit.",
						How:      "Do not store *http.Request globally. Copy Method/Path/Body into owned values for async work.",
					})
				}
			}
			return true
		})
	})
	return out, err
}

func importLocalName(spec *ast.ImportSpec, path string) string {
	if spec.Name != nil {
		return spec.Name.Name
	}
	if i := strings.LastIndex(path, "/"); i >= 0 {
		return path[i+1:]
	}
	return path
}

func isPackageLevel(file *ast.File, n ast.Node) bool {
	for _, decl := range file.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.VAR {
			continue
		}
		for _, spec := range gd.Specs {
			if spec == n {
				return true
			}
		}
	}
	return false
}

func isKernelHTTPRequestPtr(typ ast.Expr, pkgAlias map[string]string) bool {
	star, ok := typ.(*ast.StarExpr)
	if !ok {
		return false
	}
	sel, ok := star.X.(*ast.SelectorExpr)
	if !ok || sel.Sel == nil || sel.Sel.Name != "Request" {
		return false
	}
	pkg, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}
	path := pkgAlias[pkg.Name]
	return path == "github.com/zatrano/framework/v3/core/kernel/http" || strings.HasSuffix(path, "/core/kernel/http")
}
