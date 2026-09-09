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
  zatrano new <name> [--module path] [--replace /path/to/framework] [--web|--api|--full]

Canonical profiles:
  empty    zatrano new myapp          opinion-free application foundation
  web      zatrano new myapp --web    full-capacity HTML presentation defaults
  api      zatrano new myapp --api    full-capacity JSON/API presentation defaults
  full     zatrano new myapp --full   web + API presentation composition

empty is not a reduced platform. full is not every package.
--web, --api and --full are mutually exclusive. --minimal is not a scaffold.

add:web / add:api compose presentation onto an existing app and preserve the
existing root handler. --full starts with HTML / plus JSON /api.
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
	name, module, replace, scaffold, err := parseNewArgs(args)
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
	fwVer := frameworkGoModVersion(ver)
	scaffoldVer := strings.TrimPrefix(fwVer, "v")
	replaceLine := newReplaceLine(replace, scaffold)
	subs := map[string]string{
		"__MODULE__":            module,
		"__APP_NAME__":          filepath.Base(name),
		"__FRAMEWORK_VERSION__": fwVer,
		"__REPLACE_LINE__":      replaceLine,
		"__SCAFFOLD_NAME__":     scaffold,
		"__SCAFFOLD_VERSION__":  scaffoldVer,
	}
	switch scaffold {
	case generator.ScaffoldFull:
		if err := generator.Apply(generator.Request{
			FS:               starterTemplates,
			Root:             "templates/web",
			Dest:             dest,
			ScaffoldName:     generator.ScaffoldFull,
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
		if err := generator.WriteScaffoldMeta(dest, generator.ScaffoldFull, scaffoldVer, generator.CombinedDigest(webDig, apiDig)); err != nil {
			return err
		}
	default:
		err = generator.Apply(generator.Request{
			FS:              starterTemplates,
			Root:            "templates/" + scaffold,
			Dest:            dest,
			ScaffoldName:    scaffold,
			ScaffoldVersion: scaffoldVer,
			Substitutions:   subs,
		})
		if err != nil {
			return err
		}
	}
	if _, err := WriteAgentsMarkdown(dest); err != nil {
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
	fmt.Printf("Created %s (module %s, profile %s)\n", dest, module, scaffold)
	fmt.Println("Next:")
	fmt.Printf("  cd %s\n", filepath.Base(dest))
	if replace == "" {
		fmt.Println("  go mod tidy")
	}
	fmt.Println("  go run ./cmd/app key:generate")
	fmt.Println("  go run ./cmd/app serve")
	return nil
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

func parseNewArgs(args []string) (dir, module, replace, scaffold string, err error) {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return "", "", "", "", fmt.Errorf("%s", strings.TrimSpace(newHelp))
	}
	dir = strings.TrimSpace(args[0])
	if dir == "" || strings.Contains(dir, "..") {
		return "", "", "", "", fmt.Errorf("invalid project name")
	}
	module = sanitizeModule(dir)
	scaffold = generator.ScaffoldEmpty
	var profile string
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--module":
			if i+1 >= len(args) {
				return "", "", "", "", fmt.Errorf("--module requires a path")
			}
			i++
			module = strings.TrimSpace(args[i])
		case "--replace":
			if i+1 >= len(args) {
				return "", "", "", "", fmt.Errorf("--replace requires a path")
			}
			i++
			abs, aerr := filepath.Abs(args[i])
			if aerr != nil {
				return "", "", "", "", aerr
			}
			replace = filepath.ToSlash(abs)
		case "--web":
			if err := setNewProfile(&profile, generator.ScaffoldWeb); err != nil {
				return "", "", "", "", err
			}
			scaffold = generator.ScaffoldWeb
		case "--api":
			if err := setNewProfile(&profile, generator.ScaffoldAPI); err != nil {
				return "", "", "", "", err
			}
			scaffold = generator.ScaffoldAPI
		case "--full":
			if err := setNewProfile(&profile, generator.ScaffoldFull); err != nil {
				return "", "", "", "", err
			}
			scaffold = generator.ScaffoldFull
		case "--minimal":
			return "", "", "", "", fmt.Errorf("--minimal is no longer a supported scaffold profile; use --api, --web, --full, or no profile")
		default:
			return "", "", "", "", fmt.Errorf("unknown flag %s", args[i])
		}
	}
	if module == "" {
		return "", "", "", "", fmt.Errorf("empty module path")
	}
	return dir, module, replace, scaffold, nil
}

func setNewProfile(current *string, next string) error {
	if *current != "" && *current != next {
		return fmt.Errorf("--web, --api and --full are mutually exclusive")
	}
	*current = next
	return nil
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
