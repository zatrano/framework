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

	"github.com/zatrano/framework/kernel"
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
	name, module, replace, err := parseNewArgs(args)
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
	ver := "2.0.0-dev"
	if c.app != nil {
		if v := c.app.Version(); v != "" {
			ver = v
		}
	}
	fwVer := frameworkGoModVersion(ver)
	replaceLine := ""
	if replace != "" {
		replaceLine = "\nreplace github.com/zatrano/framework => " + replace + "\n"
		if pkg := siblingPackagesDir(replace); pkg != "" {
			replaceLine += "replace github.com/zatrano/packages => " + pkg + "\n"
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

func parseNewArgs(args []string) (dir, module, replace string, err error) {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return "", "", "", fmt.Errorf("usage: zatrano new <name> [--module path] [--replace /path/to/framework]")
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

func siblingPackagesDir(frameworkReplace string) string {
	candidate := filepath.Join(filepath.Dir(frameworkReplace), "packages")
	st, err := os.Stat(candidate)
	if err != nil || !st.IsDir() {
		return ""
	}
	return filepath.ToSlash(candidate)
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

// frameworkGoModVersion maps the product VERSION onto a go.mod require that is
// valid for module path github.com/zatrano/framework (major v0/v1 until /v2).
func frameworkGoModVersion(product string) string {
	v := strings.TrimSpace(product)
	v = strings.TrimPrefix(v, "v")
	if v == "" {
		return "v0.0.0"
	}
	if strings.HasPrefix(v, "2.") || strings.HasPrefix(v, "2-") || v == "2" {
		return "v0.0.0"
	}
	return "v" + v
}

func renameTemplatePath(rel string) string {
	return strings.TrimSuffix(rel, ".tmpl")
}
