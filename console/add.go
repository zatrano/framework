package console

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/zatrano/framework/v2/console/generator"
	"github.com/zatrano/framework/v2/kernel"
)

func registerAddCommands(console *Application, app *kernel.Application) {
	console.Register(
		&AddWebCommand{app: app},
		&AddAPICommand{app: app},
	)
}

var presentationOverlaySkip = map[string]bool{
	"bootstrap/scaffold.go": true,
	"bootstrap/enabled.go":  true,
	"bootstrap/addons.go":   true,
	"go.mod":                true,
	"go.sum":                true,
	".env":                  true,
	"AGENTS.md":             true,
	"README.md":             true,
	"tests/feature_test.go": true,
}

func presentationPackages(kind string) []string {
	switch kind {
	case generator.ScaffoldWeb:
		return []string{"assets", "health", "localization", "view"}
	case generator.ScaffoldAPI:
		return []string{"health", "validation"}
	default:
		return nil
	}
}

const addWebHelp = `Add web presentation to this application

Usage:
  zatrano add:web

Adds HTML/view routes, views, assets, localization, and enables the web
default packages when missing. Existing application files are never
overwritten unless they still match an exact empty generator stub.
The existing root handler is preserved (API-first JSON / stays JSON).
Idempotent: already-present web presentation is a successful no-op.
`

const addAPIHelp = `Add API presentation to this application

Usage:
  zatrano add:api

Adds JSON/API routes and enables health + validation when missing. Existing
application files are never overwritten unless they still match an exact
empty generator stub. The existing root handler is preserved (Web HTML /
stays HTML). Idempotent: already-present API presentation is a successful no-op.
`

type AddWebCommand struct{ app *kernel.Application }

func (c *AddWebCommand) Name() string        { return "add:web" }
func (c *AddWebCommand) Description() string { return "Add web presentation to this application" }
func (c *AddWebCommand) Handle(args []string) error {
	if hasHelpFlag(args) {
		fmt.Print(addWebHelp)
		return nil
	}
	return addPresentation(c.app, generator.ScaffoldWeb)
}

type AddAPICommand struct{ app *kernel.Application }

func (c *AddAPICommand) Name() string        { return "add:api" }
func (c *AddAPICommand) Description() string { return "Add API presentation to this application" }
func (c *AddAPICommand) Handle(args []string) error {
	if hasHelpFlag(args) {
		fmt.Print(addAPIHelp)
		return nil
	}
	return addPresentation(c.app, generator.ScaffoldAPI)
}

func addPresentation(app *kernel.Application, kind string) error {
	if app == nil {
		return fmt.Errorf("application unavailable")
	}
	root := app.BasePath()
	if err := requireApplicationRoot(root); err != nil {
		return err
	}
	subs, err := presentationSubs(root)
	if err != nil {
		return err
	}
	res, added, err := overlayPresentation(root, kind, subs, true)
	if err != nil {
		return err
	}
	_ = applyPackageEnvList(app, presentationPackages(kind))
	if !res.Changed() && len(added) == 0 {
		fmt.Printf("%s presentation already present\n", kind)
		return nil
	}
	if len(res.Written) > 0 {
		fmt.Printf("Wrote %d file(s)\n", len(res.Written))
	}
	if len(res.Replaced) > 0 {
		fmt.Printf("Updated %d empty-stub file(s)\n", len(res.Replaced))
	}
	if len(res.Skipped) > 0 {
		fmt.Println("Skipped (already exists, not overwritten):")
		for _, p := range res.Skipped {
			fmt.Printf("  %s\n", p)
		}
	}
	if len(added) > 0 {
		fmt.Printf("Enabled packages: %s\n", strings.Join(added, ", "))
	}
	fmt.Println("Restart the app (or rebuild) to load providers.")
	return nil
}

func requireApplicationRoot(root string) error {
	for _, rel := range []string{
		filepath.Join("bootstrap", "enabled.go"),
		filepath.Join("cmd", "app"),
		"go.mod",
	} {
		if _, err := os.Stat(filepath.Join(root, rel)); err != nil {
			return fmt.Errorf("add:* must be run from a ZATRANO application directory (missing %s)", filepath.ToSlash(rel))
		}
	}
	return nil
}

func presentationSubs(root string) (map[string]string, error) {
	mod, err := modulePath(root)
	if err != nil {
		return nil, err
	}
	name := filepath.Base(root)
	return map[string]string{
		"__MODULE__":            mod,
		"__APP_NAME__":          name,
		"__FRAMEWORK_VERSION__": "v" + currentRelease,
		"__REPLACE_LINE__":      "",
		"__SCAFFOLD_NAME__":     "",
		"__SCAFFOLD_VERSION__":  currentRelease,
	}, nil
}

func overlayPresentation(dest, kind string, subs map[string]string, fetchModule bool) (generator.OverlayResult, []string, error) {
	var combined generator.OverlayResult
	res, err := generator.Overlay(generator.OverlayRequest{
		FS:            starterTemplates,
		Root:          "templates/" + kind,
		Dest:          dest,
		Substitutions: subs,
		Skip:          presentationOverlaySkip,
		BaseFS:        starterTemplates,
		BaseRoot:      "templates/empty",
	})
	if err != nil {
		return combined, nil, err
	}
	combined = mergeOverlayResults(combined, res)
	overlayRoot := "templates/overlays/" + kind
	if _, err := fs.Stat(starterTemplates, overlayRoot); err == nil {
		extra, err := generator.Overlay(generator.OverlayRequest{
			FS:            starterTemplates,
			Root:          overlayRoot,
			Dest:          dest,
			Substitutions: subs,
			Skip:          presentationOverlaySkip,
		})
		if err != nil {
			return combined, nil, err
		}
		combined = mergeOverlayResults(combined, extra)
	}
	pkgs := presentationPackages(kind)
	added, err := mergeEnabledPackageNames(dest, pkgs)
	if err != nil {
		return combined, nil, err
	}
	imports := make([]string, 0, len(pkgs))
	for _, name := range pkgs {
		imports = append(imports, addonImportPath(name))
	}
	if err := upsertAddonBlankImports(dest, imports); err != nil {
		return combined, nil, err
	}
	ensureSiblingPackagesReplace(dest)
	if fetchModule {
		if err := ensurePackagesModule(dest); err != nil {
			return combined, added, err
		}
	}
	return combined, added, nil
}

func mergeOverlayResults(a, b generator.OverlayResult) generator.OverlayResult {
	a.Written = append(a.Written, b.Written...)
	a.Identical = append(a.Identical, b.Identical...)
	a.Replaced = append(a.Replaced, b.Replaced...)
	a.Skipped = append(a.Skipped, b.Skipped...)
	sort.Strings(a.Written)
	sort.Strings(a.Identical)
	sort.Strings(a.Replaced)
	sort.Strings(a.Skipped)
	return a
}

func mergeEnabledPackageNames(root string, names []string) ([]string, error) {
	path := filepath.Join(root, "bootstrap", "enabled.go")
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	current := parseEnabledAddons(string(body))
	seen := map[string]bool{}
	final := make([]string, 0, len(current)+len(names))
	for _, n := range current {
		n = strings.ToLower(strings.TrimSpace(n))
		if n == "" || seen[n] {
			continue
		}
		seen[n] = true
		final = append(final, n)
	}
	var added []string
	for _, n := range names {
		n = strings.ToLower(strings.TrimSpace(n))
		if n == "" || seen[n] {
			continue
		}
		seen[n] = true
		final = append(final, n)
		added = append(added, n)
	}
	if len(added) == 0 {
		return nil, nil
	}
	sort.Strings(final)
	if err := writeEnabledAddons(path, final); err != nil {
		return nil, err
	}
	return added, nil
}

func ensureSiblingPackagesReplace(root string) {
	path := filepath.Join(root, "go.mod")
	body, err := os.ReadFile(path)
	if err != nil {
		return
	}
	text := string(body)
	if strings.Contains(text, "replace github.com/zatrano/packages =>") {
		return
	}
	fw := frameworkReplacePath(text)
	if fw == "" {
		return
	}
	extra := packagesReplaceLines(fw)
	if extra == "" {
		return
	}
	if !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	_ = os.WriteFile(path, []byte(text+"\n"+extra), 0o644)
}

func frameworkReplacePath(modText string) string {
	const prefix = "replace github.com/zatrano/framework/v2 =>"
	for _, raw := range strings.Split(modText, "\n") {
		line := strings.TrimSpace(raw)
		if strings.HasPrefix(line, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(line, prefix))
		}
	}
	return ""
}
