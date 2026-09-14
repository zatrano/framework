package pkgmanager

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/zatrano/framework/v2/bootstrap"
	"github.com/zatrano/framework/v2/bootstrap/addons"
	"github.com/zatrano/framework/v2/kernel"
	"github.com/zatrano/framework/v2/kernel/encryption"
	"github.com/zatrano/framework/v2/kernel/env"
)

func runPackageDoctor(app *kernel.Application) []doctorFinding {
	var out []doctorFinding

	appVer := ""
	if app != nil {
		appVer = strings.TrimSpace(app.Version())
	}
	modBody := ""
	if app != nil {
		if raw, err := os.ReadFile(filepath.Join(app.BasePath(), "go.mod")); err == nil {
			modBody = string(raw)
		}
	}
	frameworkPin := goModRequireVersion(modBody, "github.com/zatrano/framework/v2")
	packagesPin := goModRequireVersion(modBody, "github.com/zatrano/packages")
	frameworkID := strings.TrimPrefix(strings.TrimSpace(frameworkPin), "v")
	frameworkSource := "go.mod require"
	if frameworkID == "" && appVer != "" {
		frameworkID = appVer
		frameworkSource = "VERSION"
	}
	running := frameworkID
	if appVer == "" {
		out = append(out, doctorFinding{
			Level:   "OK",
			Code:    "app.version",
			Message: "no VERSION file (optional application product identity; generated apps omit it)",
		})
	} else {
		out = append(out, doctorFinding{
			Level:   "OK",
			Code:    "app.version",
			Message: fmt.Sprintf("application VERSION %s", appVer),
		})
	}
	switch {
	case frameworkID == "":
		out = append(out, doctorFinding{
			Level:   "WARN",
			Code:    "framework.version",
			Message: "cannot determine framework version (no go.mod require for github.com/zatrano/framework/v2 and no VERSION file)",
		})
	default:
		msg := fmt.Sprintf("framework %s (%s)", frameworkID, frameworkSource)
		if appVer != "" && frameworkPin != "" && appVer != frameworkID && "v"+appVer != frameworkPin {
			msg += fmt.Sprintf("; application VERSION is %s", appVer)
		}
		out = append(out, doctorFinding{
			Level:   "OK",
			Code:    "framework.version",
			Message: msg,
		})
	}

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
	enabledSet := map[string]bool{}
	for _, name := range enabled {
		name = strings.ToLower(strings.TrimSpace(name))
		if name != "" {
			enabledSet[name] = true
		}
	}
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
		closure, cerr := addons.Expand([]string{name}, enablementLookup)
		if cerr != nil {
			out = append(out, doctorFinding{
				Level:   "ERROR",
				Code:    "enabled.requires",
				Message: fmt.Sprintf("%q Requires closure failed: %v — import the missing dependency then package:enable it", name, cerr),
			})
		} else {
			direct := map[string]bool{}
			for _, req := range meta.Requires {
				req = strings.ToLower(strings.TrimSpace(req))
				if req != "" {
					direct[req] = true
				}
			}
			missing := make([]string, 0)
			for _, dep := range closure {
				depName := strings.ToLower(strings.TrimSpace(dep.Name))
				if depName == "" || depName == name || enabledSet[depName] {
					continue
				}
				missing = append(missing, depName)
			}
			sort.Strings(missing)
			for _, depName := range missing {
				rel := "requires"
				if !direct[depName] {
					rel = "transitively requires"
				}
				out = append(out, doctorFinding{
					Level:   "ERROR",
					Code:    "enabled.requires",
					Message: fmt.Sprintf("%q %s %q which is not enabled (enable the Requires closure before boot)", name, rel, depName),
				})
			}
		}
		files := addons.ConfigFileNames(meta)
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

	imported := addons.Available()
	for _, m := range imported {
		modPath := "github.com/zatrano/packages/" + m.Name
		ver := packagesPin
		if ver == "" {
			ver = "-"
		}
		on := enabledSet[m.Name]
		min := strings.TrimSpace(m.FrameworkMin)
		if min == "" {
			min = "-"
		}
		compat := addons.MeetsFrameworkMin(running, m.FrameworkMin)
		bootable := on && compat
		level := "OK"
		if !compat {
			level = "WARN"
		}
		out = append(out, doctorFinding{
			Level: level,
			Code:  "package.imported",
			Message: fmt.Sprintf("package=%s module=%s version=%s enabled=%v framework_min=%s compatible=%v bootable=%v",
				m.Name, modPath, ver, on, min, compat, bootable),
		})
		if hasManifest && !on {
			out = append(out, doctorFinding{
				Level:   "WARN",
				Code:    "imported.disabled",
				Message: fmt.Sprintf("%q is imported but not enabled (will not boot via Enabled ∩ Imported; not an \"installed\" collapse)", m.Name),
			})
		}
	}
	if hasManifest {
		enabledNames := make([]string, 0, len(enabledSet))
		for name := range enabledSet {
			enabledNames = append(enabledNames, name)
		}
		sort.Strings(enabledNames)
		for _, name := range enabledNames {
			blockers := disableRequiresBlockers(name, enabledNames)
			if len(blockers) == 0 {
				continue
			}
			out = append(out, doctorFinding{
				Level:   "OK",
				Code:    "enabled.required_by",
				Message: fmt.Sprintf("%q is required by %s (package:disable %q would be refused)", name, strings.Join(blockers, ", "), name),
			})
		}
	}

	// Registered addons must appear in the CLI catalog. KindLibrary is
	// allowed: those packages register for CLI / init-only providers and
	// package:enable still rejects them. Unknown names are errors.
	badRegistry := 0
	for _, m := range imported {
		if !addons.MeetsFrameworkMin(running, m.FrameworkMin) {
			badRegistry++
			out = append(out, doctorFinding{
				Level:   "ERROR",
				Code:    "compatibility.framework",
				Message: fmt.Sprintf("registry package %q needs framework >= %s (running %s)", m.Name, m.FrameworkMin, running),
			})
			continue
		}
		info, ok := catalogLookup(m.Name)
		if !ok {
			badRegistry++
			out = append(out, doctorFinding{
				Level:   "ERROR",
				Code:    "catalog.unknown",
				Message: fmt.Sprintf("registry package %q is not in the CLI catalog", m.Name),
			})
			continue
		}
		kind := info.EffectiveKind()
		if kind != kernel.KindService && kind != kernel.KindLibrary {
			badRegistry++
			out = append(out, doctorFinding{
				Level:   "ERROR",
				Code:    "catalog.provider",
				Message: fmt.Sprintf("registry package %q has catalog kind %q", m.Name, kind),
			})
		}
	}
	if badRegistry == 0 {
		out = append(out, doctorFinding{
			Level:   "OK",
			Code:    "catalog.providers",
			Message: "every registered addon is in the CLI catalog",
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
		if out[i].Code != out[j].Code {
			return out[i].Code < out[j].Code
		}
		return out[i].Message < out[j].Message
	})
	return out
}
