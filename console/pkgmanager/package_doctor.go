package pkgmanager

import (
	"fmt"
	"sort"
	"strings"

	"github.com/zatrano/framework/v2/bootstrap"
	"github.com/zatrano/framework/v2/kernel"
)

type doctorFinding struct {
	Level   string // OK | WARN | ERROR
	Code    string
	Message string
}

func registerPackageHealthCommands(console *Application, app *kernel.Application) {
	console.Register(
		&PackageDoctorCommand{app: app},
		&PackageInitCommand{app: app},
	)
}

type PackageDoctorCommand struct{ app *kernel.Application }

func (c *PackageDoctorCommand) Name() string { return "package:doctor" }
func (c *PackageDoctorCommand) Description() string {
	return "Validate EnabledAddons, registry, stubs, and lean production readiness"
}
func (c *PackageDoctorCommand) Handle(args []string) error {
	findings := runPackageDoctor(c.app)
	errors := 0
	warns := 0
	for _, f := range findings {
		fmt.Printf("%-5s  %-18s  %s\n", f.Level, f.Code, f.Message)
		switch f.Level {
		case "ERROR":
			errors++
		case "WARN":
			warns++
		}
	}
	fmt.Printf("\nSummary: %d error(s), %d warning(s), %d check(s)\n", errors, warns, len(findings))
	if errors > 0 {
		return fmt.Errorf("package:doctor found %d error(s)\nNext: fix ERROR findings above (enabled.requires, enabled.library, compatibility.framework, catalog.unknown), then rerun package:doctor", errors)
	}
	return nil
}

type PackageInitCommand struct{ app *kernel.Application }

func (c *PackageInitCommand) Name() string { return "package:init" }
func (c *PackageInitCommand) Description() string {
	return "Onboard EnabledAddons from a preset (api|web), publish stubs, then run doctor"
}
func (c *PackageInitCommand) Handle(args []string) error {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" || strings.HasPrefix(args[0], "-") {
		return fmt.Errorf("usage: package:init <api|web> [--force] [--merge]")
	}
	preset := strings.ToLower(strings.TrimSpace(args[0]))
	if preset != "api" && preset != "web" {
		return fmt.Errorf("usage: package:init <api|web> [--force] [--merge]")
	}
	current, hasManifest := consumerManifest(c.app)
	if hasManifest && len(current) > 0 && !hasFlag(args, "--force", "-f", "--merge", "-m") {
		return fmt.Errorf("enablement manifest is not empty (%d packages); pass --force to replace or --merge to union", len(current))
	}

	list, ok := bootstrap.Preset(preset)
	if !ok {
		return fmt.Errorf("unknown preset %q", preset)
	}
	merge := hasFlag(args, "--merge", "-m")
	force := hasFlag(args, "--force", "-f")
	final := list
	if merge {
		seen := map[string]bool{}
		final = nil
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
	fmt.Printf("Initialized EnabledAddons with preset %q (%d packages)\n", preset, len(final))

	published, skipped, err := publishPackagesQuiet(c.app, final, force)
	if err != nil {
		return err
	}
	if published > 0 || skipped > 0 {
		fmt.Printf("Config stubs: published=%d skipped=%d\n", published, skipped)
	}

	fmt.Println()
	fmt.Println("Running package:doctor …")
	findings := runPackageDoctor(c.app)
	errors := 0
	for _, f := range findings {
		fmt.Printf("%-5s  %-18s  %s\n", f.Level, f.Code, f.Message)
		if f.Level == "ERROR" {
			errors++
		}
	}
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  1. Use bootstrap.App() (or App(WithPresetAPI())/App(WithPresetWeb())) in your entrypoint")
	fmt.Println("  2. Rebuild/restart the process")
	fmt.Println("  3. zatrano package:status")
	if errors > 0 {
		return fmt.Errorf("package:init completed with %d doctor error(s)", errors)
	}
	return nil
}
