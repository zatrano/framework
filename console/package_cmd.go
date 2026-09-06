package console

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/zatrano/framework/v2/bootstrap"
	"github.com/zatrano/framework/v2/bootstrap/addons"
	"github.com/zatrano/framework/v2/bootstrap/stubs"
	"github.com/zatrano/framework/v2/kernel"
)

func registerPackageCommands(console *Application, app *kernel.Application) {
	console.Register(
		&PackageListCommand{app: app},
		&PackageEnableCommand{app: app},
		&PackageDisableCommand{app: app},
		&PackagePublishCommand{app: app},
		&PackageInstallCommand{app: app},
		&PackageStatusCommand{app: app},
		&PackagePresetCommand{app: app},
	)
	registerPackageHealthCommands(console, app)
}

type PackageListCommand struct{ app *kernel.Application }

func (c *PackageListCommand) Name() string { return "package:list" }
func (c *PackageListCommand) Description() string {
	return "List first-party packages (services by default; --all includes libraries)"
}
func (c *PackageListCommand) Handle(args []string) error {
	showAll := hasFlag(args, "--all", "-a")
	showLibs := hasFlag(args, "--libraries", "--libs")
	enabled := enabledSet(c.app)
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "PACKAGE\tKIND\tSTATUS\tHEAVY\tSTUBS\tDESCRIPTION")

	if showLibs && !showAll {
		for _, p := range catalogLibraries() {
			heavy := ""
			if p.Heavy {
				heavy = "yes"
			}
			desc := p.Description
			if desc == "" {
				desc = "import-only helper"
			}
			fmt.Fprintf(w, "%s\tlibrary\t-\t%s\t\t%s\n", p.Name, heavy, desc)
		}
		return w.Flush()
	}

	for _, m := range addons.Available() {
		status := "disabled"
		if enabled[m.Name] {
			status = "enabled"
		}
		heavy := ""
		if m.Heavy {
			heavy = "yes"
		}
		stub := ""
		if len(stubs.ForPackage(m.Name)) > 0 {
			stub = "yes"
		}
		desc := m.Description
		if info, ok := catalogLookup(m.Name); ok && info.Description != "" {
			desc = info.Description
		}
		fmt.Fprintf(w, "%s\tservice\t%s\t%s\t%s\t%s\n", m.Name, status, heavy, stub, desc)
	}
	if showAll {
		for _, p := range catalogLibraries() {
			heavy := ""
			if p.Heavy {
				heavy = "yes"
			}
			desc := p.Description
			if desc == "" {
				desc = "import-only helper"
			}
			fmt.Fprintf(w, "%s\tlibrary\t-\t%s\t\t%s\n", p.Name, heavy, desc)
		}
	}
	return w.Flush()
}

type PackageEnableCommand struct{ app *kernel.Application }

func (c *PackageEnableCommand) Name() string { return "package:enable" }
func (c *PackageEnableCommand) Description() string {
	return "Enable an addon package in bootstrap/enabled.go"
}
func (c *PackageEnableCommand) Handle(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: package:enable <name>")
	}
	name := strings.ToLower(strings.TrimSpace(args[0]))
	added, err := enablePackage(c.app, name)
	if err != nil {
		return err
	}
	if !added {
		fmt.Printf("Package %s is already enabled.\n", name)
	} else {
		fmt.Printf("Enabled package %s in bootstrap/enabled.go\n", name)
		if err := wireEnabledAddon(c.app, name); err != nil {
			fmt.Printf("Note: %v\n", err)
		} else {
			fmt.Println("Wrote blank-import in bootstrap/addons.go (and go get github.com/zatrano/packages when needed).")
		}
	}
	_ = applyPackageEnvList(c.app, []string{name})
	fmt.Println("Restart the app (or rebuild) to load the provider.")
	return nil
}

type PackageDisableCommand struct{ app *kernel.Application }

func (c *PackageDisableCommand) Name() string { return "package:disable" }
func (c *PackageDisableCommand) Description() string {
	return "Disable an addon package in bootstrap/enabled.go"
}
func (c *PackageDisableCommand) Handle(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: package:disable <name>")
	}
	name := strings.ToLower(strings.TrimSpace(args[0]))
	removed, err := disablePackage(c.app, name)
	if err != nil {
		return err
	}
	if !removed {
		return fmt.Errorf("package %q is not enabled", name)
	}
	fmt.Printf("Disabled package %s in bootstrap/enabled.go\n", name)
	if err := removeAddonBlankImport(c.app.BasePath(), addonImportPath(name)); err != nil {
		fmt.Printf("Note: %v\n", err)
	}
	return nil
}

type PackagePublishCommand struct{ app *kernel.Application }

func (c *PackagePublishCommand) Name() string { return "package:publish" }
func (c *PackagePublishCommand) Description() string {
	return "Publish config stubs for an addon into config/"
}
func (c *PackagePublishCommand) Handle(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: package:publish <name> [--force]")
	}
	name := strings.ToLower(strings.TrimSpace(args[0]))
	force := hasFlag(args[1:], "--force", "-f")
	return publishPackage(c.app, name, force)
}

type PackageInstallCommand struct{ app *kernel.Application }

func (c *PackageInstallCommand) Name() string { return "package:install" }
func (c *PackageInstallCommand) Description() string {
	return "Enable an addon and publish its config stubs"
}
func (c *PackageInstallCommand) Handle(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: package:install <name> [--force]")
	}
	name := strings.ToLower(strings.TrimSpace(args[0]))
	force := hasFlag(args[1:], "--force", "-f")
	added, err := enablePackage(c.app, name)
	if err != nil {
		return err
	}
	if added {
		fmt.Printf("Enabled package %s\n", name)
	} else {
		fmt.Printf("Package %s already enabled\n", name)
	}
	if err := wireEnabledAddon(c.app, name); err != nil {
		fmt.Printf("Note: %v\n", err)
	}
	if err := publishPackage(c.app, name, force); err != nil {
		return err
	}
	_ = applyPackageEnvList(c.app, []string{name})
	fmt.Println("Restart the app (or rebuild) to load the provider.")
	return nil
}

type PackagePresetCommand struct{ app *kernel.Application }

func (c *PackagePresetCommand) Name() string { return "package:preset" }
func (c *PackagePresetCommand) Description() string {
	return "Apply a lean addon preset to bootstrap/enabled.go (api|web)"
}
func (c *PackagePresetCommand) Handle(args []string) error {
	if len(args) < 1 || args[0] == "list" || args[0] == "--list" {
		fmt.Println("PRESET\tPACKAGES")
		for _, name := range bootstrap.PresetNames() {
			list, _ := bootstrap.Preset(name)
			fmt.Printf("%s\t%s\n", name, strings.Join(list, ", "))
		}
		fmt.Println("Usage: package:preset <api|web> [--merge] [--force] [--no-publish]")
		return nil
	}
	name := strings.ToLower(strings.TrimSpace(args[0]))
	list, ok := bootstrap.Preset(name)
	if !ok {
		return fmt.Errorf("unknown preset %q (api|web)", name)
	}
	merge := hasFlag(args[1:], "--merge", "-m")
	force := hasFlag(args[1:], "--force", "-f")
	noPublish := hasFlag(args[1:], "--no-publish")
	final := list
	if merge {
		seen := map[string]bool{}
		final = nil
		current, _ := consumerManifest(c.app)
		for _, n := range append(append([]string{}, current...), list...) {
			n = strings.ToLower(strings.TrimSpace(n))
			if n == "" || seen[n] {
				continue
			}
			seen[n] = true
			final = append(final, n)
		}
		sort.Strings(final)
	}
	path := c.app.BasePath("bootstrap", "enabled.go")
	if err := writeEnabledAddons(path, final); err != nil {
		return err
	}
	if err := wireEnabledAddons(c.app, final); err != nil {
		fmt.Printf("Note: %v\n", err)
	}
	if merge {
		fmt.Printf("Merged preset %q into %s (%d packages)\n", name, path, len(final))
	} else {
		fmt.Printf("Applied preset %q to %s (%d packages)\n", name, path, len(final))
	}
	if !noPublish {
		published, skipped, err := publishPackagesQuiet(c.app, final, force)
		if err != nil {
			return err
		}
		if published > 0 || skipped > 0 {
			fmt.Printf("Config stubs: published=%d skipped=%d\n", published, skipped)
		}
	}
	_ = applyPackageEnvList(c.app, final)
	fmt.Println("Next: rebuild/restart, then `zatrano package:status`.")
	fmt.Println("Tip: production entrypoint → bootstrap.App() (consumer manifest, or DefaultMetas if none).")
	return nil
}

type PackageStatusCommand struct{ app *kernel.Application }

func (c *PackageStatusCommand) Name() string { return "package:status" }
func (c *PackageStatusCommand) Description() string {
	return "Show which enabled addons are bound in the container after boot"
}
func (c *PackageStatusCommand) Handle(args []string) error {
	if err := c.app.Bootstrap(); err != nil {
		return err
	}
	_, hasManifest := consumerManifest(c.app)
	enabled := enabledSet(c.app)
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "PACKAGE\tENABLED\tBOUND\tKEY")
	boundExtra := false
	for _, m := range addons.Available() {
		on := enabled[m.Name]
		if len(args) > 0 {
			want := strings.ToLower(args[0])
			if m.Name != want {
				continue
			}
		}
		bound := c.app.Bound(m.Key)
		if bound && !on {
			boundExtra = true
		}
		fmt.Fprintf(w, "%s\t%v\t%v\t%s\n", m.Name, on, bound, m.Key)
	}
	if err := w.Flush(); err != nil {
		return err
	}
	if !hasManifest {
		fmt.Println("Note: no consumer enablement manifest; App() uses DefaultMetas (all imported addons).")
	} else if boundExtra {
		fmt.Println("Note: BOUND without ENABLED means App(WithAddons(...)) override or a custom Boot.")
	}
	return nil
}

func consumerEnabledPath(app *kernel.Application) string {
	if app == nil {
		return filepath.Join("bootstrap", "enabled.go")
	}
	return app.BasePath("bootstrap", "enabled.go")
}

func consumerManifest(app *kernel.Application) (names []string, ok bool) {
	body, err := os.ReadFile(consumerEnabledPath(app))
	if err != nil {
		return nil, false
	}
	return parseEnabledAddons(string(body)), true
}

func enabledSet(app *kernel.Application) map[string]bool {
	out := map[string]bool{}
	if names, ok := consumerManifest(app); ok {
		for _, name := range names {
			out[strings.ToLower(name)] = true
		}
		return out
	}
	if app != nil {
		for _, name := range app.EnabledAddons() {
			out[strings.ToLower(name)] = true
		}
	}
	return out
}

func hasFlag(args []string, flags ...string) bool {
	set := map[string]bool{}
	for _, f := range flags {
		set[f] = true
	}
	for _, a := range args {
		if set[a] {
			return true
		}
	}
	return false
}

func enablePackage(app *kernel.Application, name string) (bool, error) {
	info, inCatalog := catalogLookup(name)
	if inCatalog && info.EffectiveKind() == kernel.KindLibrary {
		return false, fmt.Errorf("%q is a library package (import-only); no package:enable needed — see package:list --libraries", name)
	}
	if inCatalog && info.Layer == kernel.LayerPrimitive {
		return false, fmt.Errorf("%q is a kernel primitive, not an addon", name)
	}
	if _, imported := addons.Lookup(name); !imported && (!inCatalog || info.EffectiveKind() != kernel.KindService) {
		return false, fmt.Errorf("unknown package %q (see package:list)", name)
	}
	list, _ := consumerManifest(app)
	for _, n := range list {
		if strings.EqualFold(n, name) {
			return false, nil
		}
	}
	list = append(list, name)
	sort.Strings(list)
	if err := writeEnabledAddons(consumerEnabledPath(app), list); err != nil {
		return false, err
	}
	return true, nil
}

func disablePackage(app *kernel.Application, name string) (bool, error) {
	list, ok := consumerManifest(app)
	if !ok {
		return false, nil
	}
	next := make([]string, 0, len(list))
	found := false
	for _, n := range list {
		if strings.EqualFold(n, name) {
			found = true
			continue
		}
		next = append(next, n)
	}
	if !found {
		return false, nil
	}
	sort.Strings(next)
	if err := writeEnabledAddons(consumerEnabledPath(app), next); err != nil {
		return false, err
	}
	return true, nil
}

func publishPackage(app *kernel.Application, name string, force bool) error {
	if _, ok := addons.Lookup(name); !ok {
		return fmt.Errorf("unknown package %q (see package:list)", name)
	}
	files := stubs.ForPackage(name)
	if len(files) == 0 {
		fmt.Printf("Package %s has no config stubs to publish.\n", name)
		return nil
	}
	_, _, err := publishStubFiles(app, files, force, true)
	return err
}

// publishPackagesQuiet publishes stubs for packages that have them (no "no stubs" noise).
func publishPackagesQuiet(app *kernel.Application, names []string, force bool) (published, skipped int, err error) {
	for _, name := range names {
		files := stubs.ForPackage(name)
		if len(files) == 0 {
			continue
		}
		p, s, err := publishStubFiles(app, files, force, true)
		if err != nil {
			return published, skipped, err
		}
		published += p
		skipped += s
	}
	return published, skipped, nil
}

func publishStubFiles(app *kernel.Application, files []string, force, verbose bool) (published, skipped int, err error) {
	dir := app.BasePath("config")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return 0, 0, err
	}
	for _, file := range files {
		body, ok := stubs.Files[file]
		if !ok {
			continue
		}
		target := filepath.Join(dir, file)
		if !force {
			if _, err := os.Stat(target); err == nil {
				skipped++
				if verbose {
					fmt.Printf("Skipped %s (exists; use --force)\n", target)
				}
				continue
			}
		}
		if err := os.WriteFile(target, []byte(body), 0o644); err != nil {
			return published, skipped, err
		}
		published++
		if verbose {
			fmt.Printf("Published %s\n", target)
		}
	}
	return published, skipped, nil
}

func writeEnabledAddons(path string, names []string) error {
	var b strings.Builder
	b.WriteString(defaultEnabledAddonsPreamble())
	b.WriteString("var EnabledAddons = []string{\n")
	for _, name := range names {
		b.WriteString("\t\"" + name + "\",\n")
	}
	b.WriteString("}\n\n")
	b.WriteString("func init() {\n")
	b.WriteString("\tfwbootstrap.RegisterEnablement(EnabledAddons)\n")
	b.WriteString("}\n")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func defaultEnabledAddonsPreamble() string {
	var b strings.Builder
	b.WriteString("package bootstrap\n\n")
	b.WriteString("import (\n")
	b.WriteString("\tfwbootstrap \"github.com/zatrano/framework/v2/bootstrap\"\n")
	b.WriteString(")\n\n")
	b.WriteString("// EnabledAddons is this application's enablement manifest.\n")
	b.WriteString("//\n")
	b.WriteString("// init() registers the list; App() boots Enabled ∩ Imported.\n")
	b.WriteString("// WithAddons is an explicit override. Missing this file (legacy apps)\n")
	b.WriteString("// falls back to all imported addons.\n")
	b.WriteString("//\n")
	b.WriteString("// Quick start:\n")
	b.WriteString("//\tzatrano package:list\n")
	b.WriteString("//\tzatrano package:enable mongo\n")
	b.WriteString("//\tzatrano package:install billing\n")
	b.WriteString("//\tzatrano package:disable mongo\n")
	b.WriteString("//\n")
	b.WriteString("// Keep this list explicit for production: only enable what the project needs.\n")
	b.WriteString("// Alternatives: App(), App(WithAddons(...)). WithAddons overrides this manifest.\n")
	return b.String()
}

func parseEnabledAddons(src string) []string {
	re := regexp.MustCompile(`(?s)var EnabledAddons = \[\]string\{(.*?)\}`)
	m := re.FindStringSubmatch(src)
	if len(m) < 2 {
		return nil
	}
	out := []string{}
	for _, line := range strings.Split(m[1], "\n") {
		line = strings.TrimSpace(line)
		line = strings.TrimSuffix(line, ",")
		line = strings.Trim(line, `"`)
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}
