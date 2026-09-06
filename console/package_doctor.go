package console

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/zatrano/framework/v2/bootstrap"
	"github.com/zatrano/framework/v2/bootstrap/addons"
	"github.com/zatrano/framework/v2/bootstrap/stubs"
	"github.com/zatrano/framework/v2/kernel"
	"github.com/zatrano/framework/v2/kernel/encryption"
	"github.com/zatrano/framework/v2/kernel/env"
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
		return fmt.Errorf("package:doctor found %d error(s)", errors)
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

func runPackageDoctor(app *kernel.Application) []doctorFinding {
	var out []doctorFinding

	enabled, hasManifest := consumerManifest(app)
	if !hasManifest {
		out = append(out, doctorFinding{
			Level:   "WARN",
			Code:    "enabled.nomanifest",
			Message: "no enablement manifest; App() uses DefaultMetas (all imported addons)",
		})
	} else if len(enabled) == 0 {
		out = append(out, doctorFinding{
			Level:   "WARN",
			Code:    "enabled.empty",
			Message: "enablement manifest is empty — kernel-only unless App(WithAddons(...))",
		})
	} else {
		out = append(out, doctorFinding{
			Level:   "OK",
			Code:    "enabled.count",
			Message: fmt.Sprintf("%d package(s) in enablement manifest", len(enabled)),
		})
	}

	unknown := 0
	notImported := 0
	heavy := 0
	libs := 0
	missingStub := 0
	for _, name := range enabled {
		name = strings.ToLower(strings.TrimSpace(name))
		if name == "" {
			continue
		}
		if info, ok := catalogLookup(name); ok && info.EffectiveKind() == kernel.KindLibrary {
			libs++
			out = append(out, doctorFinding{
				Level:   "ERROR",
				Code:    "enabled.library",
				Message: fmt.Sprintf("%q is a library package — remove from EnabledAddons (import it instead)", name),
			})
			continue
		}
		meta, ok := addons.Lookup(name)
		if !ok {
			if info, catalogOK := catalogLookup(name); catalogOK && info.EffectiveKind() == kernel.KindService {
				notImported++
				out = append(out, doctorFinding{
					Level:   "WARN",
					Code:    "enabled.not_imported",
					Message: fmt.Sprintf("%q is enabled but not imported (will not boot)", name),
				})
				continue
			}
			unknown++
			out = append(out, doctorFinding{
				Level:   "ERROR",
				Code:    "enabled.unknown",
				Message: fmt.Sprintf("%q is not in the addon registry (package:list)", name),
			})
			continue
		}
		if meta.Heavy {
			heavy++
			out = append(out, doctorFinding{
				Level:   "WARN",
				Code:    "enabled.heavy",
				Message: fmt.Sprintf("%q is heavy (separate module / large deps)", name),
			})
		}
		files := stubs.ForPackage(name)
		if len(files) == 0 {
			continue
		}
		for _, file := range files {
			target := filepath.Join(app.BasePath("config"), file)
			if _, err := os.Stat(target); err != nil {
				missingStub++
				out = append(out, doctorFinding{
					Level:   "WARN",
					Code:    "stub.missing",
					Message: fmt.Sprintf("%s missing for %q — run package:publish %s", file, name, name),
				})
			}
		}
	}
	if unknown == 0 && libs == 0 && notImported == 0 && len(enabled) > 0 {
		out = append(out, doctorFinding{
			Level:   "OK",
			Code:    "enabled.registry",
			Message: "all EnabledAddons resolve in the service registry",
		})
	}
	if heavy == 0 && len(enabled) > 0 {
		out = append(out, doctorFinding{
			Level:   "OK",
			Code:    "enabled.lean",
			Message: "no heavy addons enabled",
		})
	}
	if missingStub == 0 {
		out = append(out, doctorFinding{
			Level:   "OK",
			Code:    "stub.present",
			Message: "required config stubs present (or none needed)",
		})
	}

	// Registered providers must appear in the catalog as KindService.
	// LayerAddon services now live in github.com/zatrano/packages and are
	// registered only when the application blank-imports them.
	badRegistry := 0
	for _, m := range addons.Available() {
		info, ok := catalogLookup(m.Name)
		if !ok || info.EffectiveKind() != kernel.KindService {
			badRegistry++
			out = append(out, doctorFinding{
				Level:   "ERROR",
				Code:    "catalog.provider",
				Message: fmt.Sprintf("registry package %q is not a catalog KindService", m.Name),
			})
		}
	}
	if badRegistry == 0 {
		out = append(out, doctorFinding{
			Level:   "OK",
			Code:    "catalog.providers",
			Message: "every registered addon is a catalog KindService",
		})
	}

	envName := env.NormalizeAppEnv(env.Get("APP_ENV", "local"))
	if envName == "" {
		envName = "local"
	}
	key := strings.TrimSpace(env.Get("APP_KEY", ""))
	if envName == "production" {
		if key == "" || key == "zatrano-dev-key" || key == encryption.LocalDevKey || key == "password" {
			out = append(out, doctorFinding{
				Level:   "ERROR",
				Code:    "secrets.app_key",
				Message: "production APP_KEY is missing or too weak",
			})
		} else if _, err := encryption.New(key); err != nil {
			out = append(out, doctorFinding{
				Level:   "ERROR",
				Code:    "secrets.app_key",
				Message: "production APP_KEY is missing or too weak",
			})
		} else {
			out = append(out, doctorFinding{
				Level:   "OK",
				Code:    "secrets.app_key",
				Message: "production APP_KEY looks set",
			})
		}
	} else {
		out = append(out, doctorFinding{
			Level:   "OK",
			Code:    "env.nonprod",
			Message: fmt.Sprintf("APP_ENV=%s (production key checks skipped)", envName),
		})
	}

	boot := bootstrap.CurrentBootProfile("app")
	out = append(out, doctorFinding{
		Level:   "OK",
		Code:    "boot.profile",
		Message: fmt.Sprintf("APP_BOOT resolves to %q", boot),
	})
	sort.SliceStable(out, func(i, j int) bool {
		rank := map[string]int{"ERROR": 0, "WARN": 1, "OK": 2}
		if rank[out[i].Level] != rank[out[j].Level] {
			return rank[out[i].Level] < rank[out[j].Level]
		}
		return out[i].Code < out[j].Code
	})
	return out
}
