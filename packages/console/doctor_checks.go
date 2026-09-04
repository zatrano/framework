package console

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/zatrano/framework/kernel"
)

var doctorRouteCalls = map[string]bool{
	"Get": true, "Post": true, "Put": true, "Patch": true, "Delete": true,
	"Head": true, "Options": true, "Any": true, "Match": true,
	"RegisterWeb": true, "RegisterAPI": true, "Controller": true,
	"ApplyWeb": true, "ApplyAPI": true,
}

func checkRouteLocation(root string) ([]Finding, error) {
	var out []Finding
	err := walkConsumerGo(root, func(rel, abs string, fset *token.FileSet, file *ast.File) {
		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			name := callSelName(call.Fun)
			if !isHTTPRouteCall(name, call) {
				return true
			}
			if routeCallAllowed(rel, name) {
				return true
			}
			path := firstStringArg(call)
			found := name + "()"
			if path != "" {
				found = fmt.Sprintf("%s(%q)", name, path)
			}
			how := "Move this call into app/routes/web or app/routes/api and register it with RegisterWeb/RegisterAPI."
			if name == "ApplyWeb" || name == "ApplyAPI" {
				how = "Keep ApplyWeb/ApplyAPI in an app/providers RouteServiceProvider Boot method."
			}
			out = append(out, Finding{
				Check:    "routes",
				Severity: "warning",
				File:     rel,
				Line:     fset.Position(call.Pos()).Line,
				Found:    found + " outside app/routes/{web,api}",
				Why:      "HTTP routes belong in self-registered web/api groups, not scattered through the app.",
				How:      how,
			})
			return true
		})
	})
	return out, err
}

func isHTTPRouteCall(name string, call *ast.CallExpr) bool {
	if !doctorRouteCalls[name] {
		return false
	}
	switch name {
	case "RegisterWeb", "RegisterAPI", "ApplyWeb", "ApplyAPI", "Controller":
		return true
	}
	if path := firstStringArg(call); strings.HasPrefix(path, "/") {
		return true
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	id, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}
	switch strings.ToLower(id.Name) {
	case "r", "router", "route", "rt", "mux":
		return true
	default:
		return false
	}
}

func routeCallAllowed(rel, call string) bool {
	rel = filepath.ToSlash(rel)
	switch call {
	case "ApplyWeb", "ApplyAPI":
		return strings.HasPrefix(rel, "app/providers/")
	default:
		return strings.HasPrefix(rel, "app/routes/web/") || strings.HasPrefix(rel, "app/routes/api/")
	}
}

func checkConcreteLeak(root string) ([]Finding, error) {
	fw, err := frameworkModuleRoot()
	if err != nil {
		return nil, err
	}
	concretes, err := collectContractConcretes(fw)
	if err != nil {
		return nil, err
	}
	var out []Finding
	err = walkConsumerGo(root, func(rel, abs string, fset *token.FileSet, file *ast.File) {
		for _, spec := range file.Imports {
			path, err := strconv.Unquote(spec.Path.Value)
			if err != nil {
				continue
			}
			iface, ok := concretes[path]
			if !ok {
				continue
			}
			if concreteImportAllowed(rel, path) {
				continue
			}
			line := fset.Position(spec.Pos()).Line
			out = append(out, Finding{
				Check:    "concrete",
				Severity: "warning",
				File:     rel,
				Line:     line,
				Found:    fmt.Sprintf("import %s (contracts.%s concrete)", path, iface),
				Why:      "Depending on framework concrete types couples the app to implementation details and weakens contract stability.",
				How:      fmt.Sprintf("Use contracts.%s via kernel.Application accessors instead of importing %s.", iface, path),
			})
		}
	})
	return out, err
}

func concreteImportAllowed(rel, importPath string) bool {
	rel = filepath.ToSlash(rel)
	if strings.HasSuffix(importPath, "/kernel") && strings.HasPrefix(rel, "app/console/") {
		return true
	}
	if strings.HasSuffix(importPath, "/packages/database/migration") && strings.HasPrefix(rel, "app/database/") {
		return true
	}
	if !strings.HasSuffix(importPath, "/packages/routing") {
		return false
	}
	if strings.HasPrefix(rel, "app/routes/web/") || strings.HasPrefix(rel, "app/routes/api/") {
		return true
	}
	if strings.HasPrefix(rel, "app/providers/") {
		return true
	}
	return false
}

func collectContractConcretes(fw string) (map[string]string, error) {
	files := []string{
		filepath.Join("kernel", "services.go"),
		filepath.Join("packages", "config", "assert.go"),
		filepath.Join("packages", "container", "assert.go"),
		filepath.Join("packages", "context", "assert.go"),
		filepath.Join("packages", "log", "assert.go"),
		filepath.Join("packages", "url", "assert.go"),
		filepath.Join("packages", "encryption", "assert.go"),
		filepath.Join("packages", "hashing", "assert.go"),
		filepath.Join("packages", "observability", "assert.go"),
		filepath.Join("packages", "database", "migration", "assert.go"),
	}
	out := map[string]string{}
	for _, rel := range files {
		part, err := parseContractConcretes(fw, filepath.Join(fw, rel))
		if err != nil {
			return nil, err
		}
		for ip, iface := range part {
			out[ip] = iface
		}
	}
	if out["github.com/zatrano/framework/packages/routing"] == "" {
		out["github.com/zatrano/framework/packages/routing"] = "Router"
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("console: no contract concrete bindings")
	}
	return out, nil
}

func parseContractConcretes(fw, path string) (map[string]string, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return nil, err
	}
	pkgPath := map[string]string{}
	for _, spec := range file.Imports {
		ip, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			continue
		}
		name := filepath.Base(ip)
		if spec.Name != nil {
			name = spec.Name.Name
		}
		pkgPath[name] = ip
	}
	out := map[string]string{}
	for _, decl := range file.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.VAR {
			continue
		}
		for _, spec := range gd.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok || len(vs.Names) == 0 || len(vs.Values) == 0 {
				continue
			}
			ifaceName := ""
			switch t := vs.Type.(type) {
			case *ast.Ident:
				ifaceName = t.Name
			case *ast.SelectorExpr:
				ifaceName = t.Sel.Name
			}
			star, ok := vs.Values[0].(*ast.CallExpr)
			if !ok {
				continue
			}
			paren, ok := star.Fun.(*ast.ParenExpr)
			if !ok {
				continue
			}
			starExpr, ok := paren.X.(*ast.StarExpr)
			if !ok {
				continue
			}
			ip := ""
			switch typed := starExpr.X.(type) {
			case *ast.SelectorExpr:
				pkgIdent, ok := typed.X.(*ast.Ident)
				if !ok {
					continue
				}
				ip = pkgPath[pkgIdent.Name]
			case *ast.Ident:
				rel, err := filepath.Rel(fw, filepath.Dir(path))
				if err != nil {
					continue
				}
				ip = "github.com/zatrano/framework/" + filepath.ToSlash(rel)
			}
			if ip == "" || ifaceName == "" {
				continue
			}
			out[ip] = ifaceName
		}
	}
	return out, nil
}

func checkAppLayout(root string) ([]Finding, error) {
	required, err := requiredStarterAppDirs()
	if err != nil {
		return nil, err
	}
	var out []Finding
	for _, dir := range required {
		if st, err := os.Stat(filepath.Join(root, filepath.FromSlash(dir))); err != nil || !st.IsDir() {
			out = append(out, Finding{
				Check:    "layout",
				Severity: "warning",
				File:     dir,
				Found:    "missing directory " + dir,
				Why:      "zatrano new places application code in this tree; agents and doctor assume it.",
				How:      "Create " + dir + " (or regenerate the app with zatrano new) and keep types in the starter locations.",
			})
		}
	}
	unexpected := []struct {
		path string
		why  string
		how  string
	}{
		{"application", "Legacy application/ trees are not the V2 consumer layout.", "Move code into app/ (http/controllers, services, providers) and delete application/."},
		{"routes", "Top-level routes/ is the old skeleton; V2 routes live under app/routes/{web,api}.", "Move route files into app/routes/web and app/routes/api with RegisterWeb/RegisterAPI."},
		{"app/controllers", "Controllers belong under app/http/controllers, not app/controllers.", "Move files into app/http/controllers/{web,api}."},
		{"app/config", "Application config is not an app/config tree; framework config lives in config/ and addon providers load their own maps.", "Keep settings in .env / published config stubs; do not add app/config."},
	}
	for _, u := range unexpected {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(u.path))); err != nil {
			continue
		}
		out = append(out, Finding{
			Check:    "layout",
			Severity: "warning",
			File:     u.path,
			Found:    "unexpected path " + u.path,
			Why:      u.why,
			How:      u.how,
		})
	}
	return out, nil
}

func requiredStarterAppDirs() ([]string, error) {
	seen := map[string]bool{}
	var dirs []string
	err := fs.WalkDir(starterTemplates, "templates/app", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		name := d.Name()
		if name == ".gitkeep" {
			return nil
		}
		rel := strings.TrimPrefix(filepath.ToSlash(path), "templates/")
		dir := filepath.ToSlash(filepath.Dir(rel))
		if dir == "." || seen[dir] {
			return nil
		}
		seen[dir] = true
		dirs = append(dirs, dir)
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(dirs) == 0 {
		return nil, fmt.Errorf("console: no starter app directories in templates")
	}
	sort.Strings(dirs)
	return dirs, nil
}

func checkProviders(root string) ([]Finding, error) {
	dir := filepath.Join(root, "app", "providers")
	st, err := os.Stat(dir)
	if err != nil || !st.IsDir() {
		return nil, nil
	}
	types := map[string]*providerShape{}
	err = walkDirGo(dir, root, func(rel, abs string, fset *token.FileSet, file *ast.File) {
		collectProviderShapes(types, rel, fset, file)
	})
	if err != nil {
		return nil, err
	}
	var out []Finding
	for name, p := range types {
		if !p.Register && !p.Boot && !strings.HasSuffix(name, "Provider") {
			continue
		}
		if p.Register && p.Boot {
			continue
		}
		file := p.File
		line := p.Line
		missing := "Boot"
		if !p.Register {
			missing = "Register"
		}
		if !p.Register && !p.Boot {
			missing = "Register and Boot"
		}
		out = append(out, Finding{
			Check:    "providers",
			Severity: "warning",
			File:     file,
			Line:     line,
			Found:    fmt.Sprintf("type %s missing %s", name, missing),
			Why:      "kernel.Provider requires both Register and Boot so bootstrap.WithProviders can load the type.",
			How:      fmt.Sprintf("Add func (p *%s) %s(app contracts.App) error on this type.", name, missing),
		})
	}
	addonNames := map[string]bool{}
	for _, p := range kernel.PackagesByLayer(kernel.LayerAddon) {
		addonNames[p.Name] = true
	}
	err = walkConsumerGo(root, func(rel, abs string, fset *token.FileSet, file *ast.File) {
		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			name := callSelName(call.Fun)
			if name != "Load" && name != "LoadIfAbsent" {
				return true
			}
			cfgName := firstStringArg(call)
			if !addonNames[cfgName] {
				return true
			}
			out = append(out, Finding{
				Check:    "providers",
				Severity: "warning",
				File:     rel,
				Line:     fset.Position(call.Pos()).Line,
				Found:    fmt.Sprintf("%s(%q) loads addon config from application code", name, cfgName),
				Why:      "Addon configuration is owned by the addon's own Provider, not kernel or the consumer app.",
				How:      "Remove this Load/LoadIfAbsent; blank-import the addon and let its Provider register config.",
			})
			return true
		})
	})
	return out, err
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
	roots := []string{"app", "cmd", "bootstrap", "routes", "application"}
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
