package console

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/zatrano/framework/bootstrap"
	"github.com/zatrano/framework/bootstrap/addons"
	"github.com/zatrano/framework/bootstrap/stubs"
	"github.com/zatrano/framework/kernel"
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
	enabled := enabledSet()
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "PACKAGE\tKIND\tSTATUS\tHEAVY\tSTUBS\tDESCRIPTION")

	if showLibs && !showAll {
		for _, p := range kernel.LibraryCatalog() {
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
		if info, ok := kernel.LookupPackage(m.Name); ok && info.Description != "" {
			desc = info.Description
		}
		fmt.Fprintf(w, "%s\tservice\t%s\t%s\t%s\t%s\n", m.Name, status, heavy, stub, desc)
	}
	if showAll {
		for _, p := range kernel.LibraryCatalog() {
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
		return nil
	}
	fmt.Printf("Enabled package %s in bootstrap/enabled.go\n", name)
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
	if err := publishPackage(c.app, name, force); err != nil {
		return err
	}
	fmt.Println("Restart the app (or rebuild) to load the provider.")
	return nil
}

type PackagePresetCommand struct{ app *kernel.Application }

func (c *PackagePresetCommand) Name() string { return "package:preset" }
func (c *PackagePresetCommand) Description() string {
	return "Apply a lean addon preset to bootstrap/enabled.go (api|web|demo)"
}
func (c *PackagePresetCommand) Handle(args []string) error {
	if len(args) < 1 || args[0] == "list" || args[0] == "--list" {
		fmt.Println("PRESET\tPACKAGES")
		for _, name := range bootstrap.PresetNames() {
			list, _ := bootstrap.Preset(name)
			fmt.Printf("%s\t%s\n", name, strings.Join(list, ", "))
		}
		fmt.Println("Usage: package:preset <api|web|demo> [--merge] [--force] [--no-publish]")
		return nil
	}
	name := strings.ToLower(strings.TrimSpace(args[0]))
	list, ok := bootstrap.Preset(name)
	if !ok {
		return fmt.Errorf("unknown preset %q (api|web|demo)", name)
	}
	merge := hasFlag(args[1:], "--merge", "-m")
	force := hasFlag(args[1:], "--force", "-f")
	noPublish := hasFlag(args[1:], "--no-publish")
	final := list
	if merge {
		seen := map[string]bool{}
		final = nil
		for _, n := range append(append([]string{}, bootstrap.EnabledAddons...), list...) {
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
	fmt.Println("Next: rebuild/restart, then `zatrano package:status`.")
	fmt.Println("Tip: production entrypoint → bootstrap.App() (reads EnabledAddons).")
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
	enabled := enabledSet()
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
	if boundExtra {
		fmt.Println("Note: BOUND without ENABLED means this process used App(WithDemo()) (or custom Boot), not App()+EnabledAddons.")
	}
	return nil
}

func enabledSet() map[string]bool {
	out := map[string]bool{}
	for _, name := range bootstrap.EnabledAddons {
		out[strings.ToLower(name)] = true
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
	if info, ok := kernel.LookupPackage(name); ok && info.EffectiveKind() == kernel.KindLibrary {
		return false, fmt.Errorf("%q is a library package (import-only); no package:enable needed — see package:list --libraries", name)
	}
	if _, ok := addons.Lookup(name); !ok {
		return false, fmt.Errorf("unknown package %q (see package:list)", name)
	}
	list := append([]string{}, bootstrap.EnabledAddons...)
	for _, n := range list {
		if strings.EqualFold(n, name) {
			return false, nil
		}
	}
	path := app.BasePath("bootstrap", "enabled.go")
	if inserted, err := insertEnabledAddonName(path, name); err != nil {
		return false, err
	} else if inserted {
		return true, nil
	}
	list = append(list, name)
	sort.Strings(list)
	if err := writeEnabledAddons(path, list); err != nil {
		return false, err
	}
	return true, nil
}

// insertEnabledAddonName surgically appends a package name into an existing EnabledAddons slice.
func insertEnabledAddonName(path, name string) (bool, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	src := string(body)
	idx := strings.Index(src, "var EnabledAddons")
	if idx < 0 {
		return false, nil
	}
	closeIdx := findEnabledAddonsClose(src, idx)
	if closeIdx <= idx {
		return false, nil
	}
	block := src[idx : closeIdx+1]
	if strings.Contains(block, `"`+name+`"`) {
		return false, nil
	}
	out := src[:closeIdx] + "\t\"" + name + "\",\n" + src[closeIdx:]
	return true, os.WriteFile(path, []byte(out), 0o644)
}

func disablePackage(app *kernel.Application, name string) (bool, error) {
	list := make([]string, 0, len(bootstrap.EnabledAddons))
	found := false
	for _, n := range bootstrap.EnabledAddons {
		if strings.EqualFold(n, name) {
			found = true
			continue
		}
		list = append(list, n)
	}
	if !found {
		return false, nil
	}
	sort.Strings(list)
	if err := writeEnabledAddons(app.BasePath("bootstrap", "enabled.go"), list); err != nil {
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
	preamble := defaultEnabledAddonsPreamble()
	if existing, err := os.ReadFile(path); err == nil {
		src := string(existing)
		if idx := strings.Index(src, "var EnabledAddons"); idx >= 0 {
			preamble = src[:idx]
			if preamble != "" && !strings.HasSuffix(preamble, "\n") {
				preamble += "\n"
			}
			// Prefer surgical rewrite of the slice only when a clear closing brace exists.
			if closeIdx := findEnabledAddonsClose(src, idx); closeIdx > idx {
				var b strings.Builder
				b.WriteString(preamble)
				b.WriteString("var EnabledAddons = []string{\n")
				for _, name := range names {
					b.WriteString("\t\"" + name + "\",\n")
				}
				b.WriteString("}")
				tail := src[closeIdx+1:]
				if !strings.HasPrefix(tail, "\n") && tail != "" {
					b.WriteString("\n")
				}
				b.WriteString(tail)
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					return err
				}
				return os.WriteFile(path, []byte(b.String()), 0o644)
			}
		}
	}
	var b strings.Builder
	b.WriteString(preamble)
	b.WriteString("var EnabledAddons = []string{\n")
	for _, name := range names {
		b.WriteString("\t\"" + name + "\",\n")
	}
	b.WriteString("}\n")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func defaultEnabledAddonsPreamble() string {
	var b strings.Builder
	b.WriteString("package bootstrap\n\n")
	b.WriteString("// EnabledAddons lists first-party addon packages registered by App().\n")
	b.WriteString("//\n")
	b.WriteString("// Quick start:\n")
	b.WriteString("//\tzatrano package:preset api          // lean API set\n")
	b.WriteString("//\tzatrano package:preset web          // lean web set\n")
	b.WriteString("//\tzatrano package:preset web --merge  // union with current\n")
	b.WriteString("//\tzatrano package:list\n")
	b.WriteString("//\tzatrano package:enable mongo\n")
	b.WriteString("//\tzatrano package:install billing\n")
	b.WriteString("//\n")
	b.WriteString("// Entrypoint: bootstrap.App() reads this list.\n")
	b.WriteString("// Alternatives: App(Minimal()), App(WithPresetAPI()), App(WithPresetWeb()), App(WithDemo()), App(Kernel()).\n")
	b.WriteString("// Keep this list explicit for production: only enable what the project needs.\n")
	return b.String()
}

func findEnabledAddonsClose(src string, varIdx int) int {
	rest := src[varIdx:]
	open := strings.Index(rest, "{")
	if open < 0 {
		return -1
	}
	depth := 0
	for i := open; i < len(rest); i++ {
		switch rest[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return varIdx + i
			}
		}
	}
	return -1
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
