package console

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/zatrano/framework/v2/kernel"
)

//go:embed all:templates
var starterTemplates embed.FS

func registerNewCommand(console *Application, app *kernel.Application) {
	console.Register(&NewCommand{app: app})
}

// NewCommand scaffolds a consumer application (zatrano new).
type NewCommand struct {
	app *kernel.Application
}

func (c *NewCommand) Name() string        { return "new" }
func (c *NewCommand) Description() string { return "Create a new ZATRANO application" }
func (c *NewCommand) Handle(args []string) error {
	name, module, replace, minimal, err := parseNewArgs(args)
	if err != nil {
		return err
	}
	dest, err := filepath.Abs(name)
	if err != nil {
		return err
	}
	if _, err := os.Stat(dest); err == nil {
		return fmt.Errorf("directory already exists: %s", dest)
	}
	ver := "2.0.0"
	if c.app != nil {
		if v := c.app.Version(); v != "" {
			ver = v
		}
	}
	fwVer := frameworkGoModVersion(ver)
	replaceLine := ""
	if replace != "" {
		replaceLine = "\nreplace github.com/zatrano/framework/v2 => " + replace + "\n"
		if !minimal {
			replaceLine += packagesReplaceLines(replace)
		}
	}
	subs := map[string]string{
		"__MODULE__":            module,
		"__APP_NAME__":          filepath.Base(name),
		"__FRAMEWORK_VERSION__": fwVer,
		"__REPLACE_LINE__":      replaceLine,
	}
	err = fs.WalkDir(starterTemplates, "templates", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel := strings.TrimPrefix(filepath.ToSlash(path), "templates/")
		if rel == "" || rel == "templates" || path == "templates" {
			return nil
		}
		rel = renameTemplatePath(rel)
		out := filepath.Join(dest, filepath.FromSlash(rel))
		if d.IsDir() {
			return os.MkdirAll(out, 0o755)
		}
		raw, err := starterTemplates.ReadFile(path)
		if err != nil {
			return err
		}
		body := string(raw)
		for old, neu := range subs {
			body = strings.ReplaceAll(body, old, neu)
		}
		if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
			return err
		}
		mode := os.FileMode(0o644)
		if strings.HasSuffix(rel, ".sh") {
			mode = 0o755
		}
		return os.WriteFile(out, []byte(body), mode)
	})
	if err != nil {
		return err
	}
	if _, err := WriteAgentsMarkdown(dest); err != nil {
		return err
	}
	if minimal {
		if err := applyMinimalScaffold(dest, module, filepath.Base(name)); err != nil {
			return err
		}
	}
	if replace != "" {
		tidy := exec.Command("go", "mod", "tidy")
		tidy.Dir = dest
		tidy.Stdout = os.Stdout
		tidy.Stderr = os.Stderr
		if err := tidy.Run(); err != nil {
			return fmt.Errorf("go mod tidy: %w", err)
		}
	}
	fmt.Printf("Created %s (module %s)\n", dest, module)
	fmt.Println("Next:")
	fmt.Printf("  cd %s\n", filepath.Base(dest))
	if replace == "" {
		fmt.Println("  go mod tidy")
	}
	fmt.Println("  go run ./cmd/app key:generate")
	fmt.Println("  go run ./cmd/app serve")
	return nil
}

func parseNewArgs(args []string) (dir, module, replace string, minimal bool, err error) {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return "", "", "", false, fmt.Errorf("usage: zatrano new <name> [--module path] [--replace /path/to/framework] [--minimal]")
	}
	dir = strings.TrimSpace(args[0])
	if dir == "" || strings.Contains(dir, "..") {
		return "", "", "", false, fmt.Errorf("invalid project name")
	}
	module = sanitizeModule(dir)
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--module":
			if i+1 >= len(args) {
				return "", "", "", false, fmt.Errorf("--module requires a path")
			}
			i++
			module = strings.TrimSpace(args[i])
		case "--replace":
			if i+1 >= len(args) {
				return "", "", "", false, fmt.Errorf("--replace requires a path")
			}
			i++
			abs, aerr := filepath.Abs(args[i])
			if aerr != nil {
				return "", "", "", false, aerr
			}
			replace = filepath.ToSlash(abs)
		case "--minimal":
			minimal = true
		default:
			return "", "", "", false, fmt.Errorf("unknown flag %s", args[i])
		}
	}
	if module == "" {
		return "", "", "", false, fmt.Errorf("empty module path")
	}
	return dir, module, replace, minimal, nil
}

// nestedPackagesModules are separate Go modules under the packages checkout.
// Consumer replace directives must list them; a parent-module replace does not cover them.
var nestedPackagesModules = []string{
	"database/driver/sqlite",
	"database/driver/mysql",
	"database/driver/pgsql",
	"database/driver/mssql",
	"database/driver/oracle",
	"database/driver/mongo",
	"mongo",
	"webauthn",
	"qr",
}

func siblingPackagesDir(frameworkReplace string) string {
	candidate := filepath.Join(filepath.Dir(frameworkReplace), "packages")
	st, err := os.Stat(candidate)
	if err != nil || !st.IsDir() {
		return ""
	}
	return filepath.ToSlash(candidate)
}

func packagesReplaceLines(frameworkReplace string) string {
	pkg := siblingPackagesDir(frameworkReplace)
	if pkg == "" {
		return ""
	}
	var b strings.Builder
	b.WriteString("replace github.com/zatrano/packages => " + pkg + "\n")
	for _, rel := range nestedPackagesModules {
		p := filepath.ToSlash(filepath.Join(filepath.FromSlash(pkg), filepath.FromSlash(rel)))
		st, err := os.Stat(filepath.FromSlash(p))
		if err != nil || !st.IsDir() {
			continue
		}
		mod := "github.com/zatrano/packages/" + rel
		b.WriteString("replace " + mod + " => " + p + "\n")
	}
	return b.String()
}

func applyMinimalScaffold(dest, module, appName string) error {
	files := map[string]string{
		filepath.Join("app", "providers", "app_service_provider.go"): `package providers

import (
	"github.com/zatrano/framework/v2/contracts"
	"github.com/zatrano/framework/v2/kernel/middleware/csrf"
	"github.com/zatrano/framework/v2/kernel/routing"
)

type AppServiceProvider struct{}

func (p *AppServiceProvider) Register(app contracts.App) error { return nil }

func (p *AppServiceProvider) Boot(app contracts.App) error {
	if r := routing.From(app); r != nil {
		r.Use(csrf.Except("/api"))
	}
	return nil
}
`,
		filepath.Join("app", "providers", "providers.go"): `package providers

import "github.com/zatrano/framework/v2/contracts"

func All() []contracts.Provider {
	return []contracts.Provider{
		&AppServiceProvider{},
		&RouteServiceProvider{},
	}
}
`,
		filepath.Join("app", "http", "controllers", "web", "home_controller.go"): `package web

import "github.com/zatrano/framework/v2/kernel/http"

type HomeController struct{}

func (c *HomeController) Index(req *http.Request) *http.Response {
	return http.HTML("<h1>` + appName + `</h1>")
}
`,
		filepath.Join("app", "http", "controllers", "api", "home_controller.go"): `package api

import "github.com/zatrano/framework/v2/kernel/http"

type HomeController struct{}

func (c *HomeController) Index(req *http.Request) *http.Response {
	return http.JSON(map[string]any{"name": "` + appName + `"})
}
`,
		filepath.Join("app", "routes", "web", "web.go"): `package web

import (
	webctrl "` + module + `/app/http/controllers/web"

	"github.com/zatrano/framework/v2/kernel/routing"
)

func init() {
	routing.RegisterWeb(registerWeb)
}

func registerWeb(router *routing.Router) {
	if router == nil {
		return
	}
	routing.Controller(router, &webctrl.HomeController{}, func(r routing.RouteRegistrar, c *webctrl.HomeController) {
		r.Get("/", c.Index).As("home")
	})
}
`,
		filepath.Join("app", "routes", "web", "health.go"): `package web

import (
	"github.com/zatrano/framework/v2/kernel/http"
	"github.com/zatrano/framework/v2/kernel/routing"
)

func init() {
	routing.RegisterWeb(registerHealth)
}

func registerHealth(router *routing.Router) {
	if router == nil {
		return
	}
	router.Get("/health", func(req *http.Request) *http.Response {
		return http.JSON(map[string]any{"status": "ok"})
	}).As("health")
}
`,
		filepath.Join("app", "database", "migrations", "migrations.go"): `package migrations

func All() any { return nil }
`,
		filepath.Join("app", "database", "seeders", "database_seeder.go"): `package seeders

func All() any { return nil }
`,
	}
	for rel, body := range files {
		path := filepath.Join(dest, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			return err
		}
	}
	migDir := filepath.Join(dest, "app", "database", "migrations")
	entries, err := os.ReadDir(migDir)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	for _, e := range entries {
		name := e.Name()
		if name == "migrations.go" || e.IsDir() {
			continue
		}
		if err := os.Remove(filepath.Join(migDir, name)); err != nil {
			return err
		}
	}
	if err := writeEnabledAddons(filepath.Join(dest, "bootstrap", "enabled.go"), nil); err != nil {
		return err
	}
	return nil
}

func sanitizeModule(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ReplaceAll(name, "\\", "/")
	name = strings.Trim(name, "/")
	if strings.Contains(name, "/") {
		return name
	}
	var b strings.Builder
	for _, r := range strings.ToLower(name) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' {
			b.WriteRune(r)
		}
	}
	out := b.String()
	if out == "" {
		return "myapp"
	}
	return out
}

// frameworkGoModVersion maps the product VERSION onto a go.mod require for
// module path github.com/zatrano/framework/v2.
func frameworkGoModVersion(product string) string {
	v := strings.TrimSpace(product)
	v = strings.TrimPrefix(v, "v")
	if v == "" || strings.Contains(v, "-") {
		return "v2.0.0"
	}
	if strings.HasPrefix(v, "2.") || v == "2" {
		return "v" + v
	}
	return "v" + v
}

func renameTemplatePath(rel string) string {
	return strings.TrimSuffix(rel, ".tmpl")
}
