package doctor

import (
	"go/ast"
	"go/token"
	"path/filepath"
	"strings"
)

func checkControllers(root string) ([]Finding, error) {
	var out []Finding
	err := walkConsumerGo(root, func(rel, abs string, fset *token.FileSet, file *ast.File) {
		imports := importPathByName(file)
		types := map[string]token.Pos{}
		for _, decl := range file.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok || gd.Tok != token.TYPE {
				continue
			}
			for _, spec := range gd.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok || !strings.HasSuffix(ts.Name.Name, "Controller") {
					continue
				}
				if _, ok := ts.Type.(*ast.StructType); !ok {
					continue
				}
				types[ts.Name.Name] = ts.Pos()
				if !controllerPathAllowed(rel) {
					out = append(out, Finding{
						Rule:     "APP-CTL-001",
						Check:    "controllers",
						Severity: "error",
						File:     rel,
						Line:     fset.Position(ts.Pos()).Line,
						Found:    "type " + ts.Name.Name + " outside app/http/controllers/{web,api,admin}",
						Why:      "Controllers belong in the canonical HTTP controller packages.",
						How:      "Move this type into app/http/controllers/web, api, or admin.",
						See:      archSee + " §G",
					})
				}
			}
		}
		authFile := isAuthControllerFile(rel, types)
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv == nil || fn.Name == nil {
				continue
			}
			recv := recvTypeName(fn.Recv)
			if fn.Name.IsExported() && !strings.HasSuffix(recv, "Controller") && isHTTPControllerMethod(fn, imports) {
				out = append(out, Finding{
					Rule:     "APP-CTL-001",
					Check:    "controllers",
					Severity: "error",
					File:     rel,
					Line:     fset.Position(fn.Pos()).Line,
					Found:    recv + "." + fn.Name.Name + " is an HTTP entry but " + recv + " is not a *Controller",
					Why:      "HTTP methods belong on *Controller types in app/http/controllers/{web,api,admin}. Handler/Action types are not a second HTTP layer.",
					How:      "Rename the type to {Resource}Controller and keep it under the canonical controller package.",
					See:      archSee + " §G · ADR-0001",
				})
				continue
			}
			if !strings.HasSuffix(recv, "Controller") {
				continue
			}
			if fn.Name.IsExported() {
				if viol, line := controllerSignatureViolation(fn, imports, fset); viol != "" {
					out = append(out, Finding{
						Rule:     "APP-CTL-002",
						Check:    "controllers",
						Severity: "error",
						File:     rel,
						Line:     line,
						Found:    recv + "." + fn.Name.Name + " " + viol,
						Why:      "HTTP controller methods are (req *http.Request) *http.Response.",
						How:      "Change the signature to match generated controllers. Unexported helpers may use other signatures.",
						See:      archSee + " §G",
					})
				}
			}
			if controllerDirKind(rel) == "api" && funcCallsHTTP(fn, imports, "View") {
				out = append(out, Finding{
					Rule:     "APP-CTL-004",
					Check:    "controllers",
					Severity: "error",
					File:     rel,
					Line:     fset.Position(fn.Pos()).Line,
					Found:    recv + "." + fn.Name.Name + " calls http.View in an API controller",
					Why:      "API controllers return JSON, not views (ADR-0009).",
					How:      "Move HTML to app/http/controllers/web or return http.JSON.",
					See:      archSee + " §G · ADR-0009",
				})
			}
			if !authFile && funcCallsHTTP(fn, imports, "View") && funcCallsHTTP(fn, imports, "JSON") {
				out = append(out, Finding{
					Rule:     "APP-CTL-003",
					Check:    "controllers",
					Severity: "error",
					File:     rel,
					Line:     fset.Position(fn.Pos()).Line,
					Found:    recv + "." + fn.Name.Name + " mixes http.View and http.JSON",
					Why:      "Do not mix View and JSON in one resource method. make:auth AuthController is the documented exception.",
					How:      "Split into web and api controllers, or keep a single presentation.",
					See:      archSee + " §G · ADR-0009",
				})
			}
			if validationEnabled(root) && (fn.Name.Name == "Store" || fn.Name.Name == "Update") &&
				controllerDirKind(rel) != "" &&
				funcCallsORM(fn, imports, "Create", "Update", "InsertMany", "Upsert", "Delete") &&
				!funcCallsValidation(fn, imports, "ValidateForm") {
				out = append(out, Finding{
					Rule:     "APP-REQ-003",
					Check:    "requests",
					Severity: "error",
					File:     rel,
					Line:     fset.Position(fn.Pos()).Line,
					Found:    recv + "." + fn.Name.Name + " persists without validation.ValidateForm",
					Why:      "Mutating HTTP uses FormRequest when validation is enabled.",
					How:      "Call validation.ValidateForm(req, YourStoreRequest{}) before ORM writes.",
					See:      archSee + " §E · ADR-0002",
				})
			}
			if controllerDirKind(rel) != "" && funcCallsValidation(fn, imports, "Make") {
				out = append(out, Finding{
					Rule:     "APP-REQ-002",
					Check:    "requests",
					Severity: "error",
					File:     rel,
					Line:     fset.Position(fn.Pos()).Line,
					Found:    recv + "." + fn.Name.Name + " calls validation.Make",
					Why:      "Inline validation.Make in controllers is not the application path; use FormRequest + ValidateForm.",
					How:      "Replace Make with a FormRequest in app/http/requests and ValidateForm.",
					See:      archSee + " §E · ADR-0002",
				})
			}
		}
		if controllerDirKind(rel) != "" {
			out = append(out, controllerFileTransactions(rel, fset, file)...)
		}
	})
	return out, err
}

func controllerFileTransactions(rel string, fset *token.FileSet, file *ast.File) []Finding {
	imports := importPathByName(file)
	var out []Finding
	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		name := callSelName(call.Fun)
		if name != "Transaction" && name != "QueryTx" {
			return true
		}
		if !ormCall(call.Fun, imports) {
			return true
		}
		out = append(out, Finding{
			Rule:     "APP-CTL-005",
			Check:    "controllers",
			Severity: "error",
			File:     rel,
			Line:     fset.Position(call.Pos()).Line,
			Found:    "orm." + name + " in a controller file",
			Why:      "Controllers must not own transactions. The application service starts orm.Transaction.",
			How:      "Move the transaction into app/services and call that service from the controller.",
			See:      archSee + " §H · §M · ADR-0004",
		})
		return true
	})
	return out
}

func controllerPathAllowed(rel string) bool {
	rel = filepath.ToSlash(rel)
	return strings.HasPrefix(rel, "app/http/controllers/web/") ||
		strings.HasPrefix(rel, "app/http/controllers/api/") ||
		strings.HasPrefix(rel, "app/http/controllers/admin/")
}

func controllerDirKind(rel string) string {
	rel = filepath.ToSlash(rel)
	switch {
	case strings.HasPrefix(rel, "app/http/controllers/web/"):
		return "web"
	case strings.HasPrefix(rel, "app/http/controllers/api/"):
		return "api"
	case strings.HasPrefix(rel, "app/http/controllers/admin/"):
		return "admin"
	default:
		return ""
	}
}

func isAuthControllerFile(rel string, types map[string]token.Pos) bool {
	base := strings.ToLower(filepath.Base(rel))
	if base == "auth_controller.go" || base == "social_auth_controller.go" {
		return true
	}
	for name := range types {
		if name == "AuthController" || name == "SocialAuthController" {
			return true
		}
	}
	return false
}

func isHTTPControllerMethod(fn *ast.FuncDecl, imports map[string]string) bool {
	if fn.Type.Params == nil || len(fn.Type.Params.List) == 0 {
		return false
	}
	if !isHTTPNamedType(fn.Type.Params.List[0].Type, imports, "Request") {
		return false
	}
	if fn.Type.Results == nil || len(fn.Type.Results.List) != 1 {
		return false
	}
	return isHTTPNamedType(fn.Type.Results.List[0].Type, imports, "Response")
}

func controllerSignatureViolation(fn *ast.FuncDecl, imports map[string]string, fset *token.FileSet) (string, int) {
	if fn.Type.Params == nil || len(fn.Type.Params.List) == 0 {
		if fn.Name.Name == "Handle" {
			return "must be (req *http.Request) *http.Response, not Handle()", fset.Position(fn.Pos()).Line
		}
		return "", 0
	}
	first := fn.Type.Params.List[0].Type
	if !isHTTPNamedType(first, imports, "Request") {
		return "", 0
	}
	if fn.Type.Results == nil || len(fn.Type.Results.List) != 1 || !isHTTPNamedType(fn.Type.Results.List[0].Type, imports, "Response") {
		return "HTTP methods must return *http.Response", fset.Position(fn.Pos()).Line
	}
	return "", 0
}

func isHTTPNamedType(expr ast.Expr, imports map[string]string, name string) bool {
	if star, ok := expr.(*ast.StarExpr); ok {
		expr = star.X
	}
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name == name && hasDotImport(imports, "/kernel/http")
	case *ast.SelectorExpr:
		id, ok := t.X.(*ast.Ident)
		if !ok || t.Sel.Name != name {
			return false
		}
		return strings.HasSuffix(imports[id.Name], "/kernel/http")
	default:
		return false
	}
}

func checkRequests(root string) ([]Finding, error) {
	var out []Finding
	err := walkConsumerGo(root, func(rel, abs string, fset *token.FileSet, file *ast.File) {
		imports := importPathByName(file)
		if misplacedValidationMakePath(rel) {
			ast.Inspect(file, func(n ast.Node) bool {
				fn, ok := n.(*ast.FuncDecl)
				if !ok {
					return true
				}
				if !funcCallsValidation(fn, imports, "Make") {
					return true
				}
				out = append(out, Finding{
					Rule:     "APP-REQ-002",
					Check:    "requests",
					Severity: "error",
					File:     rel,
					Line:     fset.Position(fn.Pos()).Line,
					Found:    "validation.Make in " + rel,
					Why:      "Inline validation.Make is not the application path; keep rules on FormRequest and call ValidateForm from HTTP.",
					How:      "Move rules to app/http/requests and call validation.ValidateForm from the controller.",
					See:      archSee + " §E · ADR-0002",
				})
				return false
			})
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || !isFormRequestRules(fn) {
				continue
			}
			recv := recvTypeName(fn.Recv)
			if recv == "" || !badFormRequestName(recv) {
				continue
			}
			out = append(out, Finding{
				Rule:     "APP-REQ-001",
				Check:    "requests",
				Severity: "error",
				File:     rel,
				Line:     fset.Position(fn.Pos()).Line,
				Found:    "type " + recv,
				Why:      "FormRequest types are named {Resource}StoreRequest, UpdateRequest, or IndexRequest — not a bare {Resource}Request, Form, DTO, or Input.",
				How:      "Rename to PostStoreRequest / PostUpdateRequest / PostIndexRequest (or an allowed {Action}Request).",
				See:      archSee + " §E",
			})
		}
	})
	return out, err
}

func misplacedValidationMakePath(rel string) bool {
	rel = filepath.ToSlash(rel)
	return strings.HasPrefix(rel, "app/services/") ||
		strings.HasPrefix(rel, "app/models/") ||
		strings.HasPrefix(rel, "app/repositories/")
}

func isFormRequestRules(fn *ast.FuncDecl) bool {
	if fn == nil || fn.Name == nil || fn.Name.Name != "Rules" || fn.Recv == nil {
		return false
	}
	if fn.Type.Results == nil || len(fn.Type.Results.List) != 1 {
		return false
	}
	return isMapStringString(fn.Type.Results.List[0].Type)
}

func isMapStringString(expr ast.Expr) bool {
	m, ok := expr.(*ast.MapType)
	if !ok {
		return false
	}
	k, ok1 := m.Key.(*ast.Ident)
	v, ok2 := m.Value.(*ast.Ident)
	return ok1 && ok2 && k.Name == "string" && v.Name == "string"
}

func badFormRequestName(name string) bool {
	if !strings.HasSuffix(name, "Request") {
		return true
	}
	stem := strings.TrimSuffix(name, "Request")
	allow := []string{
		"Store", "Update", "Index", "Upload", "Bulk",
		"Login", "Register", "Forgot", "Reset", "Change", "Confirm",
		"Profile", "Password", "Completion", "File", "TwoFactor", "Logout",
	}
	for _, a := range allow {
		if strings.Contains(stem, a) {
			return false
		}
	}
	return stem != ""
}
