package pkgmanager

import (
	"fmt"
	"sort"
	"strings"

	"github.com/zatrano/framework/v2/bootstrap/addons"
	"github.com/zatrano/framework/v2/kernel"
)

type PackageEnableCommand struct{ app *kernel.Application }

func (c *PackageEnableCommand) Name() string { return "package:enable" }
func (c *PackageEnableCommand) Description() string {
	return "Enable an addon package in bootstrap/enabled.go"
}
func (c *PackageEnableCommand) Handle(args []string) error {
	if len(args) < 1 {
		return cliErr(ExitUsage, fmt.Errorf("usage: package:enable <name>"))
	}
	name := strings.ToLower(strings.TrimSpace(args[0]))
	added, err := enablePackage(c.app, name)
	if err != nil {
		return cliFailed(ExitEnablement, "package:enable", name, err, "import missing Requires (package:list / package:doctor), then retry; Optional dependencies are not auto-enabled")
	}
	names, _ := enableRequiresNames(name)
	if !added {
		fmt.Printf("Package %s is already enabled.\n", name)
	} else {
		extras := make([]string, 0)
		for _, n := range names {
			if n != "" && n != name {
				extras = append(extras, n)
			}
		}
		if len(extras) > 0 {
			fmt.Printf("Enabled package %s in bootstrap/enabled.go (also enabled Requires: %s)\n", name, strings.Join(extras, ", "))
		} else {
			fmt.Printf("Enabled package %s in bootstrap/enabled.go\n", name)
		}
		if _, imported := addons.Lookup(name); !imported {
			fmt.Println("Note: this process has not imported the package yet; Requires facts may be incomplete until bootstrap/addons.go is compiled. Re-run package:enable after rebuild if package:doctor reports enabled.requires.")
		}
		if err := wireEnablement(c.app, name); err != nil {
			fmt.Printf("Note: %v\n", err)
		} else {
			fmt.Println("Wrote blank-import in bootstrap/addons.go (and go get github.com/zatrano/packages when needed).")
		}
	}
	if len(names) == 0 {
		names = []string{name}
	}
	_ = applyPackageEnvList(c.app, names)
	if err := scaffoldPackageDirs(c.app, names); err != nil {
		return err
	}
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
		return cliErr(ExitUsage, fmt.Errorf("usage: package:disable <name>"))
	}
	name := strings.ToLower(strings.TrimSpace(args[0]))
	removed, err := disablePackage(c.app, name)
	if err != nil {
		return cliFailed(ExitEnablement, "package:disable", name, err, "disable the requiring addons first (package:status / package:doctor), then retry; modules are not removed")
	}
	if !removed {
		fmt.Printf("Package %s is already disabled.\n", name)
		return nil
	}
	fmt.Printf("Disabled package %s in bootstrap/enabled.go\n", name)
	if err := removeAddonBlankImport(c.app.BasePath(), addonImportPath(name)); err != nil {
		fmt.Printf("Note: %v\n", err)
	}
	return nil
}

func rejectEnableTarget(name string) error {
	info, inCatalog := catalogLookup(name)
	if inCatalog && info.EffectiveKind() == kernel.KindLibrary {
		return fmt.Errorf("%q is a library package (import-only); no package:enable needed — see package:list --libraries", name)
	}
	if inCatalog && info.Layer == kernel.LayerPrimitive {
		return fmt.Errorf("%q is a kernel primitive, not an addon", name)
	}
	if _, imported := addons.Lookup(name); !imported && (!inCatalog || info.EffectiveKind() != kernel.KindService) {
		return fmt.Errorf("unknown package %q (see package:list)", name)
	}
	return nil
}

// enablementLookup reuses addons.Lookup for Requires facts. Optional is
// stripped so addons.Expand does not auto-enable Optional dependencies.
// Catalog services that are not yet imported are identity-only (no invented Requires).
func enablementLookup(name string) (addons.Meta, bool) {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		return addons.Meta{}, false
	}
	if m, ok := addons.Lookup(name); ok {
		m.Optional = nil
		return m, true
	}
	info, inCatalog := catalogLookup(name)
	if inCatalog && info.EffectiveKind() == kernel.KindService && info.Layer != kernel.LayerPrimitive {
		return addons.Meta{Name: name}, true
	}
	return addons.Meta{}, false
}

// enableRequiresNames is the Requires closure of name (Optional excluded).
// It is planning only: it does not write files.
func enableRequiresNames(name string) ([]string, error) {
	name = strings.ToLower(strings.TrimSpace(name))
	if err := rejectEnableTarget(name); err != nil {
		return nil, err
	}
	metas, err := addons.Expand([]string{name}, enablementLookup)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(metas))
	for _, m := range metas {
		if m.Name == "" {
			continue
		}
		out = append(out, m.Name)
	}
	sort.Strings(out)
	return out, nil
}

func enablePackage(app *kernel.Application, name string) (bool, error) {
	name = strings.ToLower(strings.TrimSpace(name))
	closure, err := enableRequiresNames(name)
	if err != nil {
		return false, err
	}
	list, _ := consumerManifest(app)
	seen := map[string]bool{}
	for _, n := range list {
		n = strings.ToLower(strings.TrimSpace(n))
		if n == "" {
			continue
		}
		seen[n] = true
	}
	added := false
	for _, n := range closure {
		if seen[n] {
			continue
		}
		list = append(list, n)
		seen[n] = true
		added = true
	}
	if !added {
		return false, nil
	}
	sort.Strings(list)
	if err := writeEnabledAddons(consumerEnabledPath(app), list); err != nil {
		return false, err
	}
	return true, nil
}

func disableRequiresBlockers(target string, remaining []string) []string {
	target = strings.ToLower(strings.TrimSpace(target))
	var blockers []string
	seen := map[string]bool{}
	for _, name := range remaining {
		name = strings.ToLower(strings.TrimSpace(name))
		if name == "" || name == target || seen[name] {
			continue
		}
		metas, err := addons.Expand([]string{name}, enablementLookup)
		if err != nil {
			if m, ok := addons.Lookup(name); ok {
				for _, req := range m.Requires {
					if req == target && !seen[name] {
						seen[name] = true
						blockers = append(blockers, name)
						break
					}
				}
			}
			continue
		}
		for _, m := range metas {
			if m.Name == target {
				seen[name] = true
				blockers = append(blockers, name)
				break
			}
		}
	}
	sort.Strings(blockers)
	return blockers
}

func disableBlockedError(target string, blockers []string) error {
	if len(blockers) == 1 {
		return fmt.Errorf("cannot disable %q:\nenabled addon %q requires it", target, blockers[0])
	}
	quoted := make([]string, 0, len(blockers))
	for _, b := range blockers {
		quoted = append(quoted, `"`+b+`"`)
	}
	return fmt.Errorf("cannot disable %q:\nenabled addons %s require it", target, strings.Join(quoted, ", "))
}

func disablePackage(app *kernel.Application, name string) (bool, error) {
	name = strings.ToLower(strings.TrimSpace(name))
	list, ok := consumerManifest(app)
	if !ok {
		return false, nil
	}
	remaining := make([]string, 0, len(list))
	found := false
	for _, n := range list {
		n = strings.ToLower(strings.TrimSpace(n))
		if n == "" {
			continue
		}
		if n == name {
			found = true
			continue
		}
		remaining = append(remaining, n)
	}
	if !found {
		return false, nil
	}
	if blockers := disableRequiresBlockers(name, remaining); len(blockers) > 0 {
		return false, disableBlockedError(name, blockers)
	}
	sort.Strings(remaining)
	if err := writeEnabledAddons(consumerEnabledPath(app), remaining); err != nil {
		return false, err
	}
	return true, nil
}
