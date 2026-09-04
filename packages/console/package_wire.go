package console

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/zatrano/framework/kernel"
)

func addonImportPath(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	if info, ok := kernel.LookupPackage(name); ok && info.Layer == kernel.LayerIntelligence {
		return "github.com/zatrano/framework/packages/" + name
	}
	return "github.com/zatrano/packages/" + name
}

func wireEnabledAddon(app *kernel.Application, name string) error {
	return wireEnabledAddons(app, []string{name})
}

func wireEnabledAddons(app *kernel.Application, names []string) error {
	if app == nil {
		return fmt.Errorf("application unavailable")
	}
	root := app.BasePath()
	imports := make([]string, 0, len(names))
	needPackages := false
	for _, name := range names {
		imports = append(imports, addonImportPath(name))
		info, ok := kernel.LookupPackage(name)
		if !ok || info.Layer == kernel.LayerAddon {
			needPackages = true
		}
	}
	if err := upsertAddonBlankImports(root, imports); err != nil {
		return err
	}
	if !needPackages {
		return nil
	}
	return ensurePackagesModule(root)
}

func upsertAddonBlankImports(root string, extra []string) error {
	path := filepath.Join(root, "bootstrap", "addons.go")
	existing, err := readBlankImports(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	set := map[string]bool{}
	for _, e := range existing {
		set[e] = true
	}
	for _, e := range extra {
		e = strings.TrimSpace(e)
		if e != "" {
			set[e] = true
		}
	}
	list := make([]string, 0, len(set))
	for e := range set {
		list = append(list, e)
	}
	sort.Strings(list)
	return writeAddonBlankImports(path, list)
}

func removeAddonBlankImport(root, importPath string) error {
	path := filepath.Join(root, "bootstrap", "addons.go")
	existing, err := readBlankImports(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	out := make([]string, 0, len(existing))
	for _, e := range existing {
		if e != importPath {
			out = append(out, e)
		}
	}
	return writeAddonBlankImports(path, out)
}

func readBlankImports(path string) ([]string, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, body, parser.ImportsOnly)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, spec := range f.Imports {
		if spec.Name == nil || spec.Name.Name != "_" {
			continue
		}
		p, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			continue
		}
		out = append(out, p)
	}
	return out, nil
}

func writeAddonBlankImports(path string, imports []string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	var b strings.Builder
	b.WriteString("package bootstrap\n\n")
	b.WriteString("// Addon blank-imports register providers from github.com/zatrano/packages\n")
	b.WriteString("// (and in-tree intelligence packages). Maintained by package:enable.\n")
	if len(imports) == 0 {
		return os.WriteFile(path, []byte(b.String()), 0o644)
	}
	b.WriteString("\nimport (\n")
	for _, imp := range imports {
		b.WriteString("\t_ \"" + imp + "\"\n")
	}
	b.WriteString(")\n")
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func ensurePackagesModule(root string) error {
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		return nil
	}
	mod, err := modulePath(root)
	if err != nil || mod == "github.com/zatrano/framework" {
		return nil
	}
	cmd := exec.Command("go", "get", "github.com/zatrano/packages@dev")
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("go get github.com/zatrano/packages@dev failed: %w\n%s\nNext: go get github.com/zatrano/packages@dev", err, strings.TrimSpace(string(out)))
	}
	return nil
}
