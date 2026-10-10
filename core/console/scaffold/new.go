package scaffold

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/zatrano/framework/v3/core/console/consolecore"
	"github.com/zatrano/framework/v3/core/console/describe"
	"github.com/zatrano/framework/v3/core/console/generator"
	"github.com/zatrano/framework/v3/core/kernel"
	"github.com/zatrano/framework/v3/core/ssr"
)

const newHelp = `Create a new ZATRANO application

Usage:
  zatrano new <name> [--module path] [--replace /path/to/framework] [--framework-version vX.Y.Z] [--no-tidy]

Layout:
  core/       framework runtime (synced from this checkout when --replace is set)
  app/        application code (handlers, routes, providers)
  templates/  Canvas SSR templates
  database/   migrations, queries, sqlc, seeders

Handlers live in app/http/handlers/{web,api}. Default enabled: health + template (Canvas).
assets, localization, validation stay opt-in (package:enable).

After the files are written, go mod tidy runs so the project builds.
That step is a convenience. --no-tidy skips it. --replace tries tidy the same way.
When tidy succeeds, the next steps are cd, key:generate, and serve.
If tidy is skipped, fails, or times out, the command still exits 0, the next
steps keep go mod tidy, and one line names the directory to tidy.

--framework-version sets the framework module version in the generated go.mod.
The default is this CLI's own version. A value without a v prefix, or a value
that is not a version, is an error and the project is not written.
package:enable still pins github.com/zatrano/packages@v1.15.0.
`

// defaultTidyTimeout bounds the automatic go mod tidy after scaffolding.
const defaultTidyTimeout = 120 * time.Second

// tidyRunner runs go mod tidy in a project directory.
// GOFLAGS, GOPROXY, and GOWORK are left unchanged.
type tidyRunner interface {
	Tidy(ctx context.Context, dir string) error
}

// commandTidyRunner invokes the user's go binary. It does not set
// GOFLAGS, GOPROXY, or GOWORK.
type commandTidyRunner struct{}

func (commandTidyRunner) Tidy(ctx context.Context, dir string) error {
	cmd := exec.CommandContext(ctx, "go", "mod", "tidy")
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// NewCommand scaffolds a consumer application (zatrano new).
type NewCommand struct {
	app         *kernel.Application
	writeAgents func(string) (string, error)
	seedEnv     func(string) error
	// tidy overrides the go mod tidy process. Nil uses the user's go.
	tidy tidyRunner
	// tidyLimit overrides the 120s tidy budget. Zero uses defaultTidyTimeout.
	tidyLimit time.Duration
}

func (c *NewCommand) Name() string        { return "new" }
func (c *NewCommand) Description() string { return "Create a new ZATRANO application" }
func (c *NewCommand) Handle(args []string) error {
	if hasHelpFlag(args) {
		fmt.Print(newHelp)
		return nil
	}
	name, module, replace, frameworkVersion, noTidy, err := parseNewArgs(args)
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
	if frameworkVersion != "" {
		ver = strings.TrimPrefix(frameworkVersion, "v")
	}
	if err := applyStarter(dest, module, replace, ver); err != nil {
		return err
	}
	if err := ensureAppLayout(dest, replace); err != nil {
		return err
	}
	if err := writeWebCanvasStarter(dest); err != nil {
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
	tidyOK := false
	if !noTidy {
		tidyOK = c.runTidy(dest) == nil
	}
	fmt.Printf("Created %s (module %s)\n", dest, module)
	fmt.Println("Next:")
	fmt.Printf("  cd %s\n", filepath.Base(dest))
	if !tidyOK {
		fmt.Println("  go mod tidy")
	}
	fmt.Println("  go run ./cmd/app key:generate")
	fmt.Println("  go run ./cmd/app serve")
	if !tidyOK {
		fmt.Printf("Run `go mod tidy` in %s before building\n", dest)
	}
	return nil
}

func (c *NewCommand) tidyBudget() time.Duration {
	if c != nil && c.tidyLimit > 0 {
		return c.tidyLimit
	}
	return defaultTidyTimeout
}

func (c *NewCommand) runTidy(dir string) error {
	runner := tidyRunner(commandTidyRunner{})
	if c != nil && c.tidy != nil {
		runner = c.tidy
	}
	ctx, cancel := context.WithTimeout(context.Background(), c.tidyBudget())
	defer cancel()
	return runner.Tidy(ctx, dir)
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

// writeWebCanvasStarter writes Canvas starter templates (layouts + welcome).
func writeWebCanvasStarter(dest string) error {
	return ssr.WriteStarterTemplates(kernel.NewApplication(dest))
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
	var b strings.Builder
	b.WriteString("\nreplace github.com/zatrano/framework/v3 => " + goModPath(replace) + "\n")
	if rh := siblingRawHTTPDir(replace); rh != "" {
		b.WriteString("replace github.com/zatrano/rawhttp => " + goModPath(rh) + "\n")
	}
	if cv := siblingCanvasDir(replace); cv != "" {
		b.WriteString("replace github.com/zatrano/canvas => " + goModPath(cv) + "\n")
	}
	b.WriteString(packagesReplaceLines(replace))
	return b.String()
}

func siblingRawHTTPDir(frameworkReplace string) string {
	return siblingModuleDir(frameworkReplace, "rawhttp")
}

func siblingCanvasDir(frameworkReplace string) string {
	return siblingModuleDir(frameworkReplace, "canvas")
}

func siblingModuleDir(frameworkReplace, name string) string {
	candidate := filepath.Join(filepath.Dir(frameworkReplace), name)
	st, err := os.Stat(candidate)
	if err != nil || !st.IsDir() {
		return ""
	}
	if _, err := os.Stat(filepath.Join(candidate, "go.mod")); err != nil {
		return ""
	}
	return filepath.ToSlash(candidate)
}

// goModPath quotes filesystem paths that contain spaces (required by go.mod).
func goModPath(p string) string {
	p = filepath.ToSlash(p)
	if strings.ContainsAny(p, " \t") {
		return strconv.Quote(p)
	}
	return p
}

func parseNewArgs(args []string) (dir, module, replace, frameworkVersion string, noTidy bool, err error) {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return "", "", "", "", false, fmt.Errorf("%s", strings.TrimSpace(newHelp))
	}
	dir = strings.TrimSpace(args[0])
	if dir == "" || strings.Contains(dir, "..") {
		return "", "", "", "", false, fmt.Errorf("invalid project name")
	}
	module = sanitizeModule(filepath.Base(filepath.Clean(dir)))
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--module":
			if i+1 >= len(args) {
				return "", "", "", "", false, fmt.Errorf("--module requires a path")
			}
			i++
			module = strings.TrimSpace(args[i])
		case "--replace":
			if i+1 >= len(args) {
				return "", "", "", "", false, fmt.Errorf("--replace requires a path")
			}
			i++
			abs, aerr := filepath.Abs(args[i])
			if aerr != nil {
				return "", "", "", "", false, aerr
			}
			replace = filepath.ToSlash(abs)
		case "--no-tidy":
			noTidy = true
		case "--framework-version":
			if i+1 >= len(args) {
				return "", "", "", "", false, fmt.Errorf("--framework-version requires a version")
			}
			i++
			frameworkVersion, err = parseFrameworkVersion(args[i])
			if err != nil {
				return "", "", "", "", false, err
			}
		default:
			return "", "", "", "", false, fmt.Errorf("unknown flag %s", args[i])
		}
	}
	if module == "" {
		return "", "", "", "", false, fmt.Errorf("empty module path")
	}
	return dir, module, replace, frameworkVersion, noTidy, nil
}

// parseFrameworkVersion accepts a Go module version (vMAJOR.MINOR.PATCH,
// optional prerelease). A missing v prefix and any other shape are errors.
func parseFrameworkVersion(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("--framework-version requires a version")
	}
	if !strings.HasPrefix(raw, "v") {
		return "", fmt.Errorf("invalid framework version %q: must start with v", raw)
	}
	if !validGoModuleVersion(raw) {
		return "", fmt.Errorf("invalid framework version %q: not a version", raw)
	}
	return raw, nil
}

func validGoModuleVersion(v string) bool {
	if len(v) < 6 || v[0] != 'v' {
		return false
	}
	rest := v[1:]
	num, pre, hasPre := strings.Cut(rest, "-")
	if strings.Contains(num, "+") {
		return false
	}
	parts := strings.Split(num, ".")
	if len(parts) != 3 {
		return false
	}
	for _, p := range parts {
		if !semverNumeric(p) {
			return false
		}
	}
	if !hasPre {
		return true
	}
	if pre == "" || strings.Contains(pre, "+") {
		return false
	}
	for _, id := range strings.Split(pre, ".") {
		if !semverPrerelease(id) {
			return false
		}
	}
	return true
}

func semverNumeric(p string) bool {
	if p == "" || (len(p) > 1 && p[0] == '0') {
		return false
	}
	for i := 0; i < len(p); i++ {
		if p[i] < '0' || p[i] > '9' {
			return false
		}
	}
	return true
}

func semverPrerelease(id string) bool {
	if id == "" {
		return false
	}
	numeric := true
	for i := 0; i < len(id); i++ {
		c := id[i]
		switch {
		case c >= '0' && c <= '9':
		case (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || c == '-':
			numeric = false
		default:
			return false
		}
	}
	if numeric && len(id) > 1 && id[0] == '0' {
		return false
	}
	return true
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
