package console

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/zatrano/framework/v2/bootstrap/addons"
	"github.com/zatrano/framework/v2/kernel"
)

type packageEnvLookup struct {
	roots   []string
	appRoot string
}

func newPackageEnvLookup(app *kernel.Application) *packageEnvLookup {
	l := &packageEnvLookup{}
	seen := map[string]bool{}
	add := func(p string) {
		p = strings.TrimSpace(p)
		if p == "" {
			return
		}
		abs, err := filepath.Abs(p)
		if err != nil {
			return
		}
		st, err := os.Stat(abs)
		if err != nil || !st.IsDir() {
			return
		}
		if seen[abs] {
			return
		}
		seen[abs] = true
		l.roots = append(l.roots, abs)
	}
	add(os.Getenv("PACKAGES_DIR"))
	if app != nil {
		l.appRoot = app.BasePath()
		add(goListModuleDir(l.appRoot, "github.com/zatrano/packages"))
		add(filepath.Join(l.appRoot, "packages"))
		add(filepath.Join(filepath.Dir(l.appRoot), "packages"))
	}
	return l
}

func (l *packageEnvLookup) snippet(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		return ""
	}
	for _, root := range l.roots {
		body, err := os.ReadFile(filepath.Join(root, name, ".env.example"))
		if err == nil && len(strings.TrimSpace(string(body))) > 0 {
			return string(body)
		}
	}
	if l.appRoot != "" {
		dir := goListModuleDir(l.appRoot, "github.com/zatrano/packages/"+name)
		if dir != "" {
			body, err := os.ReadFile(filepath.Join(dir, ".env.example"))
			if err == nil && len(strings.TrimSpace(string(body))) > 0 {
				return string(body)
			}
		}
	}
	if m, ok := addons.Lookup(name); ok {
		return m.EnvExample
	}
	return ""
}

func goListModuleDir(dir, module string) string {
	if dir == "" || module == "" {
		return ""
	}
	if _, err := os.Stat(filepath.Join(dir, "go.mod")); err != nil {
		return ""
	}
	cmd := exec.Command("go", "list", "-m", "-f", "{{.Dir}}", module)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func applyPackageEnv(app *kernel.Application, name string) (merged bool, err error) {
	return applyPackageEnvWith(app, newPackageEnvLookup(app), name)
}

func applyPackageEnvList(app *kernel.Application, names []string) error {
	lookup := newPackageEnvLookup(app)
	var first error
	for _, name := range names {
		merged, err := applyPackageEnvWith(app, lookup, name)
		if err != nil {
			fmt.Printf("Note: %s env example: %v\n", name, err)
			if first == nil {
				first = err
			}
			continue
		}
		if merged {
			fmt.Printf("Merged %s environment keys into .env.example\n", name)
		}
	}
	return first
}

func applyPackageEnvWith(app *kernel.Application, lookup *packageEnvLookup, name string) (bool, error) {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		return false, nil
	}
	if lookup == nil {
		lookup = newPackageEnvLookup(app)
	}
	snippet := lookup.snippet(name)
	if strings.TrimSpace(snippet) == "" {
		return false, nil
	}
	root := "."
	if app != nil {
		root = app.BasePath()
	}
	examplePath := filepath.Join(root, ".env.example")
	n, err := mergePackageEnvFile(examplePath, name, snippet)
	if err != nil {
		return false, err
	}
	envPath := filepath.Join(root, ".env")
	if _, statErr := os.Stat(envPath); statErr == nil {
		if _, err := mergePackageEnvFile(envPath, name, snippet); err != nil {
			return n > 0, err
		}
	}
	return n > 0, nil
}
