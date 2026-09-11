package console

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/zatrano/framework/v2/console/generator"
	"github.com/zatrano/framework/v2/kernel"
)

const newHelp = `Create a new ZATRANO application

Usage:
  zatrano new <name> [--module path] [--replace /path/to/framework]

One application: HTML at / and JSON at /api. Controllers live in
app/http/controllers/web and app/http/controllers/api. Choose View or JSON
per handler; both are present.

Enabled presentation packages: assets, health, localization, view, validation.
Database, auth, queue, and other capabilities stay opt-in (package:enable).
`

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
	if err := applyStarter(dest, module, replace, generator.ScaffoldApp, ver); err != nil {
		return err
	}
	if _, err := WriteAgentsMarkdown(dest); err != nil {
		return err
	}
	if err := seedEnvFromExample(dest); err != nil {
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

func applyStarter(dest, module, replace, scaffold, ver string) error {
	fwVer := frameworkGoModVersion(ver)
	scaffoldVer := strings.TrimPrefix(fwVer, "v")
	replaceLine := newReplaceLine(replace, scaffold)
	subs := map[string]string{
		"__MODULE__":            module,
		"__APP_NAME__":          filepath.Base(dest),
		"__FRAMEWORK_VERSION__": fwVer,
		"__REPLACE_LINE__":      replaceLine,
		"__SCAFFOLD_NAME__":     scaffold,
		"__SCAFFOLD_VERSION__":  scaffoldVer,
	}
	switch scaffold {
	case generator.ScaffoldApp, generator.ScaffoldFull:
		if err := generator.Apply(generator.Request{
			FS:               starterTemplates,
			Root:             "templates/web",
			Dest:             dest,
			ScaffoldName:     scaffold,
			ScaffoldVersion:  scaffoldVer,
			SkipScaffoldMeta: true,
			Substitutions:    subs,
		}); err != nil {
			return err
		}
		if _, _, err := overlayPresentation(dest, generator.ScaffoldAPI, subs, false); err != nil {
			return err
		}
		webDig, err := generator.Digest(starterTemplates, "templates/web")
		if err != nil {
			return err
		}
		apiDig, err := generator.Digest(starterTemplates, "templates/api")
		if err != nil {
			return err
		}
		return generator.WriteScaffoldMeta(dest, scaffold, scaffoldVer, generator.CombinedDigest(webDig, apiDig))
	default:
		return generator.Apply(generator.Request{
			FS:              starterTemplates,
			Root:            "templates/" + scaffold,
			Dest:            dest,
			ScaffoldName:    scaffold,
			ScaffoldVersion: scaffoldVer,
			Substitutions:   subs,
		})
	}
}

func newReplaceLine(replace, scaffold string) string {
	if replace == "" {
		return ""
	}
	line := "\nreplace github.com/zatrano/framework/v2 => " + replace + "\n"
	if scaffold != generator.ScaffoldEmpty {
		line += packagesReplaceLines(replace)
	}
	return line
}

func parseNewArgs(args []string) (dir, module, replace string, err error) {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return "", "", "", fmt.Errorf("%s", strings.TrimSpace(newHelp))
	}
	dir = strings.TrimSpace(args[0])
	if dir == "" || strings.Contains(dir, "..") {
		return "", "", "", fmt.Errorf("invalid project name")
	}
	module = sanitizeModule(dir)
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
		return "v" + currentRelease
	}
	if strings.HasPrefix(v, "2.") || v == "2" {
		return "v" + v
	}
	return "v" + v
}
