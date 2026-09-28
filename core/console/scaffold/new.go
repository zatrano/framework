package scaffold

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"

	"github.com/zatrano/framework/v3/core/console/consolecore"
	"github.com/zatrano/framework/v3/core/console/describe"
	"github.com/zatrano/framework/v3/core/console/generator"
	"github.com/zatrano/framework/v3/core/kernel"
)

const newHelp = `Create a new ZATRANO application

Usage:
  zatrano new <name> [--module path] [--replace /path/to/framework]

Layout:
  core/       framework runtime (synced from this checkout when --replace is set)
  app/        application code (handlers, routes, providers)
  templates/  Canvas SSR templates
  database/   migrations, queries, sqlc, seeders

Handlers live in app/http/handlers/{web,api}. Default enabled package: health.
template, assets, localization, validation stay opt-in (package:enable).
`

// NewCommand scaffolds a consumer application (zatrano new).
type NewCommand struct {
	app         *kernel.Application
	writeAgents func(string) (string, error)
	seedEnv     func(string) error
}

func (c *NewCommand) Name() string        { return "new" }
func (c *NewCommand) Description() string { return "Create a new ZATRANO application" }
func (c *NewCommand) Handle(args []string) error {
	if hasHelpFlag(args) {
		fmt.Print(newHelp)
		return nil
	}
	name, module, replace, err := parseNewArgs(args)
	if err != nil {
		return err
	}
	dest, err := filepath.Abs(name)
	if err != nil {
		return err
	}
	ver := productVersion()
	if c.app != nil {
		if v := c.app.Version(); v != "" {
			ver = v
		}
	}
	if err := applyStarter(dest, module, replace, ver); err != nil {
		return err
	}
	if err := ensureAppLayout(dest, replace); err != nil {
		return err
	}
	if c.writeAgents == nil {
		c.writeAgents = describe.WriteAgentsMarkdown
	}
	if c.seedEnv == nil {
		c.seedEnv = consolecore.SeedEnvFromExample
	}
	if _, err := c.writeAgents(dest); err != nil {
		return err
	}
	if err := c.seedEnv(dest); err != nil {
		return err
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

func applyStarter(dest, module, replace, ver string) error {
	fwVer := frameworkGoModVersion(ver)
	scaffoldVer := strings.TrimPrefix(fwVer, "v")
	replaceLine := newReplaceLine(replace)
	subs := map[string]string{
		"__MODULE__":            module,
		"__APP_NAME__":          filepath.Base(dest),
		"__FRAMEWORK_VERSION__": fwVer,
		"__REPLACE_LINE__":      replaceLine,
		"__SCAFFOLD_NAME__":     generator.ScaffoldApp,
		"__SCAFFOLD_VERSION__":  scaffoldVer,
	}
	return generator.Apply(generator.Request{
		FS:              starterTemplates,
		Root:            "templates/web",
		Dest:            dest,
		ScaffoldName:    generator.ScaffoldApp,
		ScaffoldVersion: scaffoldVer,
		Substitutions:   subs,
	})
}

// ensureAppLayout creates V3 roots (core/, templates/, database/) and optionally
// syncs framework core sources into core/ when --replace points at a checkout.
func ensureAppLayout(dest, frameworkReplace string) error {
	for _, dir := range []string{
		"core",
		"templates",
		"templates/web",
		"templates/layouts",
		"templates/components",
		"database",
		"database/migrations",
		"database/queries",
		"database/sqlc",
		"database/seeders",
	} {
		if err := os.MkdirAll(filepath.Join(dest, dir), 0o755); err != nil {
			return err
		}
	}
	readme := filepath.Join(dest, "core", "README.md")
	if _, err := os.Stat(readme); err != nil {
		body := "# core/\n\nFramework runtime lives here.\n\n" +
			"With `zatrano new --replace`, sources are synced from the framework checkout.\n" +
			"Application code belongs in `app/`, not here.\n"
		if err := os.WriteFile(readme, []byte(body), 0o644); err != nil {
			return err
		}
	}
	if frameworkReplace == "" {
		return nil
	}
	srcCore := filepath.Join(filepath.FromSlash(frameworkReplace), "core")
	st, err := os.Stat(srcCore)
	if err != nil || !st.IsDir() {
		return nil
	}
	return syncDir(srcCore, filepath.Join(dest, "core"))
}

func syncDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		// Skip bulky / generated trees
		slash := filepath.ToSlash(rel)
		if strings.Contains(slash, "/testdata/") || strings.HasSuffix(slash, "_test.go") {
			return nil
		}
		out := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(out, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
			return err
		}
		return os.WriteFile(out, data, 0o644)
	})
}

func newReplaceLine(replace string) string {
	if replace == "" {
		return ""
	}
	return "\nreplace github.com/zatrano/framework/v3 => " + goModPath(replace) + "\n" + packagesReplaceLines(replace)
}

// goModPath quotes filesystem paths that contain spaces (required by go.mod).
func goModPath(p string) string {
	p = filepath.ToSlash(p)
	if strings.ContainsAny(p, " \t") {
		return strconv.Quote(p)
	}
	return p
}

func parseNewArgs(args []string) (dir, module, replace string, err error) {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return "", "", "", fmt.Errorf("%s", strings.TrimSpace(newHelp))
	}
	dir = strings.TrimSpace(args[0])
	if dir == "" || strings.Contains(dir, "..") {
		return "", "", "", fmt.Errorf("invalid project name")
	}
	module = sanitizeModule(filepath.Base(filepath.Clean(dir)))
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--module":
			if i+1 >= len(args) {
				return "", "", "", fmt.Errorf("--module requires a path")
			}
			i++
			module = strings.TrimSpace(args[i])
		case "--replace":
			if i+1 >= len(args) {
				return "", "", "", fmt.Errorf("--replace requires a path")
			}
			i++
			abs, aerr := filepath.Abs(args[i])
			if aerr != nil {
				return "", "", "", aerr
			}
			replace = filepath.ToSlash(abs)
		default:
			return "", "", "", fmt.Errorf("unknown flag %s", args[i])
		}
	}
	if module == "" {
		return "", "", "", fmt.Errorf("empty module path")
	}
	return dir, module, replace, nil
}

func hasHelpFlag(args []string) bool {
	for _, a := range args {
		if a == "--help" || a == "-h" {
			return true
		}
	}
	return false
}

// nestedPackagesModules are separate Go modules under the packages checkout.
// Consumer replace directives must list them; a parent-module replace does not cover them.
// V3 SQL adapters live under db/*; legacy database/driver/* is not listed.
var nestedPackagesModules = []string{
	"db/postgres",
	"db/mysql",
	"db/mariadb",
	"db/sqlite",
	"db/sqlserver",
	"db/oracle",
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
	b.WriteString("replace github.com/zatrano/packages => " + goModPath(pkg) + "\n")
	for _, rel := range nestedPackagesModules {
		p := filepath.Join(filepath.FromSlash(pkg), filepath.FromSlash(rel))
		st, err := os.Stat(p)
		if err != nil || !st.IsDir() {
			continue
		}
		// Nested modules need their own go.mod; skip plain packages dirs.
		if _, err := os.Stat(filepath.Join(p, "go.mod")); err != nil {
			continue
		}
		mod := nestedModulePath(p, rel)
		b.WriteString("replace " + mod + " => " + goModPath(filepath.ToSlash(p)) + "\n")
	}
	return b.String()
}

// nestedModulePath reads module from go.mod when present; otherwise packages/<rel>.
func nestedModulePath(dir, rel string) string {
	data, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "module ") {
				if m := strings.TrimSpace(strings.TrimPrefix(line, "module ")); m != "" {
					return m
				}
			}
		}
	}
	return "github.com/zatrano/packages/" + rel
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
// module path github.com/zatrano/framework/v3 (must be a v3.* semver).
func frameworkGoModVersion(product string) string {
	v := strings.TrimSpace(product)
	v = strings.TrimPrefix(v, "v")
	if v == "" {
		return "v" + consolecore.CurrentRelease
	}
	// Strip prerelease suffix for go.mod (v3.0.0-dev → v3.0.0).
	if i := strings.IndexByte(v, '-'); i >= 0 {
		v = v[:i]
	}
	if v == "" {
		return "v" + consolecore.CurrentRelease
	}
	// Module path ends in /v3 — refuse v1/v2 tags.
	if strings.HasPrefix(v, "1.") || v == "1" || strings.HasPrefix(v, "2.") || v == "2" {
		return "v" + consolecore.CurrentRelease
	}
	return "v" + v
}
