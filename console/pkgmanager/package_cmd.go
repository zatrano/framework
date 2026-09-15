package pkgmanager

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/zatrano/framework/v2/bootstrap"
	"github.com/zatrano/framework/v2/bootstrap/addons"
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
	registerPackageRegistryCommands(console, app)
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
		if len(m.ConfigFiles) > 0 {
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
	return "Enable an imported addon and publish its config stubs (not a module download)"
}
func (c *PackageInstallCommand) Handle(args []string) error {
	if len(args) < 1 {
		return cliErr(ExitUsage, fmt.Errorf("usage: package:install <name> [--force]"))
	}
	name := strings.ToLower(strings.TrimSpace(args[0]))
	force := hasFlag(args[1:], "--force", "-f")
	added, err := enablePackage(c.app, name)
	if err != nil {
		return cliFailed(ExitEnablement, "package:install", name, err, "import missing Requires (package:list / package:doctor), then retry; package:install is enablement, not module download")
	}
	if added {
		fmt.Printf("Enabled package %s\n", name)
	} else {
		fmt.Printf("Package %s already enabled\n", name)
	}
	if err := wireEnablement(c.app, name); err != nil {
		fmt.Printf("Note: %v\n", err)
	}
	if err := publishPackage(c.app, name, force); err != nil {
		return cliFailed(ExitEnablement, "package:install", name, err, "enablement may already be written; fix stub publish (package:publish) or unknown package name")
	}
	names, _ := enableRequiresNames(name)
	if len(names) == 0 {
		names = []string{name}
	}
	_ = applyPackageEnvList(c.app, names)
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
func publishPackage(app *kernel.Application, name string, force bool) error {
	meta, ok := addons.Lookup(name)
	if !ok {
		return fmt.Errorf("unknown package %q (see package:list)", name)
	}
	if len(meta.ConfigFiles) == 0 {
		fmt.Printf("Package %s has no config stubs to publish.\n", name)
		return nil
	}
	_, _, err := publishConfigFiles(app, meta.ConfigFiles, force, true)
	return err
}

// publishPackagesQuiet publishes stubs for packages that have them (no "no stubs" noise).
func publishPackagesQuiet(app *kernel.Application, names []string, force bool) (published, skipped int, err error) {
	for _, name := range names {
		meta, ok := addons.Lookup(name)
		if !ok || len(meta.ConfigFiles) == 0 {
			continue
		}
		p, s, err := publishConfigFiles(app, meta.ConfigFiles, force, true)
		if err != nil {
			return published, skipped, err
		}
		published += p
		skipped += s
	}
	return published, skipped, nil
}

func publishConfigFiles(app *kernel.Application, files map[string]string, force, verbose bool) (published, skipped int, err error) {
	dir := app.BasePath("config")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return 0, 0, err
	}
	names := addons.ConfigFileNames(addons.Meta{ConfigFiles: files})
	for _, file := range names {
		body, ok := files[file]
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
	b.WriteString("//\tzatrano package:enable social\n")
	b.WriteString("//\tzatrano package:disable mongo\n")
	b.WriteString("//\n")
	b.WriteString("// Keep this list explicit for production: only enable what the project needs.\n")
	b.WriteString("// Alternatives: App(), App(WithAddons(...)). WithAddons overrides this manifest.\n")
	return b.String()
}
