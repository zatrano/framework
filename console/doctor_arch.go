package console

import (
	"go/ast"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const archSee = "docs/architecture/STANDARD.md"

var forbiddenLayerDirs = []string{
	"domain",
	"app/domain",
	"dtos",
	"dto",
	"app/dtos",
	"app/dto",
	"usecases",
	"usecase",
	"app/usecases",
	"app/usecase",
	"interactors",
	"interactor",
	"app/interactors",
	"app/interactor",
	"app/application",
	"handlers",
	"app/handlers",
	"app/http/handlers",
	"actions",
	"app/actions",
	"app/http/actions",
	"entities",
	"app/entities",
	"internal/usecase",
	"internal/usecases",
	"internal/interactor",
	"internal/interactors",
	"internal/entity",
	"internal/entities",
	"internal/dto",
	"internal/dtos",
}

var forbiddenPackageNames = map[string]bool{
	"usecase": true, "usecases": true,
	"interactor": true, "interactors": true,
	"dto": true, "dtos": true,
	"handlers": true, "actions": true, "entities": true,
}

func checkForbiddenLayers(root string) ([]Finding, error) {
	var out []Finding
	for _, dir := range forbiddenLayerDirs {
		if st, err := os.Stat(filepath.Join(root, filepath.FromSlash(dir))); err != nil || !st.IsDir() {
			continue
		}
		out = append(out, Finding{
			Rule:     "APP-LAY-001",
			Check:    "layers",
			Severity: "error",
			File:     dir,
			Found:    "forbidden architectural directory " + dir,
			Why:      "STANDARD rejects domain/usecase/interactor/dto/handler/action/entity layers as application architecture.",
			How:      "Delete this directory and keep Controller → FormRequest → optional Service → ORM / From(app).",
			See:      archSee + " §B · ADR-0001",
		})
	}
	err := walkConsumerGo(root, func(rel, abs string, fset *token.FileSet, file *ast.File) {
		pkg := file.Name.Name
		if forbiddenPackageNames[pkg] || (pkg == "domain" && consumerAppPackage(rel)) {
			out = append(out, Finding{
				Rule:     "APP-LAY-002",
				Check:    "layers",
				Severity: "error",
				File:     rel,
				Line:     fset.Position(file.Name.Pos()).Line,
				Found:    "package " + pkg,
				Why:      "Package names usecase/interactor/dto/handlers/actions/entities/domain are a second architecture, not a ZATRANO application package.",
				How:      "Move types into app/services, app/models, or app/http/requests with canonical names.",
				See:      archSee + " §B · ADR-0001",
			})
		}
		ast.Inspect(file, func(n ast.Node) bool {
			ts, ok := n.(*ast.TypeSpec)
			if !ok || ts.Name == nil || !ts.Name.IsExported() {
				return true
			}
			name := ts.Name.Name
			if _, ok := ts.Type.(*ast.InterfaceType); ok && strings.HasSuffix(name, "Repository") {
				out = append(out, Finding{
					Rule:     "APP-REP-001",
					Check:    "layers",
					Severity: "error",
					File:     rel,
					Line:     fset.Position(ts.Pos()).Line,
					Found:    "type " + name + " interface",
					Why:      "Repository interfaces are not required and must not become a second architecture (ADR-0003).",
					How:      "Delete the interface. Use orm.Query[T]() or an optional concrete struct wrapper.",
					See:      archSee + " §J · ADR-0003",
				})
				return true
			}
			if _, isStruct := ts.Type.(*ast.StructType); !isStruct {
				return true
			}
			rule, why := forbiddenLayerType(name)
			if rule == "" {
				return true
			}
			how := "Use a controller, FormRequest, or named application service verb (Place, Publish). Do not add UseCase/DTO/WebService types."
			see := archSee + " §B · ADR-0001 · ADR-0009"
			if rule == "APP-REP-001" {
				how = "Delete the generic repository abstraction. Optional concrete *Repository structs wrapping ORM are allowed."
				see = archSee + " §J · ADR-0003"
			}
			out = append(out, Finding{
				Rule:     rule,
				Check:    "layers",
				Severity: "error",
				File:     rel,
				Line:     fset.Position(ts.Pos()).Line,
				Found:    "type " + name,
				Why:      why,
				How:      how,
				See:      see,
			})
			return true
		})
	})
	return out, err
}

func consumerAppPackage(rel string) bool {
	rel = filepath.ToSlash(rel)
	return strings.HasPrefix(rel, "app/") || strings.HasPrefix(rel, "domain/")
}

func forbiddenLayerType(name string) (rule, why string) {
	switch {
	case strings.HasSuffix(name, "UseCase"), strings.HasSuffix(name, "Usecase"):
		return "APP-LAY-003", "UseCase types are not a ZATRANO application layer."
	case strings.HasSuffix(name, "Interactor"):
		return "APP-LAY-003", "Interactor types are a UseCase synonym, not a ZATRANO application layer."
	case strings.HasSuffix(name, "DTO"), strings.HasSuffix(name, "Dto"):
		return "APP-LAY-003", "DTO types are not a ZATRANO application layer; use FormRequest."
	case name == "WebService" || name == "ApiService" || strings.HasSuffix(name, "WebService") || strings.HasSuffix(name, "ApiService"):
		return "APP-LAY-003", "Transport-specific services are forbidden (ADR-0009). Share one optional service, two controllers."
	case strings.HasSuffix(name, "WebUseCase") || strings.HasSuffix(name, "ApiUseCase"):
		return "APP-LAY-003", "WebUseCase/ApiUseCase are forbidden (ADR-0009)."
	case name == "BaseRepository" || name == "GenericRepository" || strings.HasSuffix(name, "RepositoryFactory"):
		return "APP-REP-001", "Generic repository abstractions are not ZATRANO architecture (ADR-0003). Optional concrete *Repository structs wrapping ORM are allowed."
	default:
		return "", ""
	}
}

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
			See:      "docs/architecture/decisions/0010-unique-exists-fail-open.md",
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
				See:      "docs/architecture/decisions/0010-unique-exists-fail-open.md",
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
