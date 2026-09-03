package bootstrap

import (
	"fmt"
	"strings"

	"github.com/zatrano/framework/app/providers"
	"github.com/zatrano/framework/bootstrap/addons"
	"github.com/zatrano/framework/bootstrap/foundation"
	"github.com/zatrano/framework/core"
)

type appOptions struct {
	kernelOnly bool
	addons     []string
	addonsSet  bool
}

// Option configures App().
type Option func(*appOptions)

// WithAddons sets the addon list (does not merge EnabledAddons).
func WithAddons(names ...string) Option {
	return func(o *appOptions) {
		o.kernelOnly = false
		o.addonsSet = true
		o.addons = append([]string(nil), names...)
	}
}

// Minimal boots foundation + app providers with no addons.
func Minimal() Option {
	return WithAddons()
}

// Kernel boots only the secure kernel (no DB/session/auth/addons/app routes).
func Kernel() Option {
	return func(o *appOptions) {
		o.kernelOnly = true
		o.addonsSet = true
		o.addons = nil
	}
}

// WithDemo boots foundation + every DemoAddons package + app providers.
func WithDemo() Option {
	return func(o *appOptions) {
		o.kernelOnly = false
		o.addonsSet = true
		o.addons = append([]string(nil), DemoAddons...)
	}
}

// WithPresetAPI boots foundation + PresetAPI ∪ EnabledAddons + app providers.
func WithPresetAPI() Option {
	return func(o *appOptions) {
		o.kernelOnly = false
		o.addonsSet = true
		o.addons = mergeAddonNames(PresetAPI, EnabledAddons)
	}
}

// WithPresetWeb boots foundation + PresetWeb ∪ EnabledAddons + app providers.
func WithPresetWeb() Option {
	return func(o *appOptions) {
		o.kernelOnly = false
		o.addonsSet = true
		o.addons = mergeAddonNames(PresetWeb, EnabledAddons)
	}
}

// App creates the configured application.
// With no options: foundation + EnabledAddons + app providers.
func App(opts ...Option) *core.Application {
	cfg := appOptions{}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	if cfg.kernelOnly {
		app, err := Boot(foundation.KernelProviders(), nil)
		if err != nil {
			panic(err)
		}
		return app
	}
	names := EnabledAddons
	if cfg.addonsSet {
		names = cfg.addons
	}
	app, err := Boot(foundation.Providers(), names, ApplicationProviders()...)
	if err != nil {
		panic(err)
	}
	return app
}

// Boot assembles an application from foundation providers, addon names, and app providers.
func Boot(foundationProviders []core.Provider, addonNames []string, appProviders ...core.Provider) (*core.Application, error) {
	basePath, _ := findBasePath()
	application := core.NewApplication(basePath)
	application.RegisterProviders(foundationProviders...)
	selected, err := addons.Select(addonNames...)
	if err != nil {
		return nil, fmt.Errorf("bootstrap addons: %w", err)
	}
	application.RegisterProviders(selected...)
	application.RegisterProviders(appProviders...)
	return application, nil
}

// mergeAddonNames concatenates preset + EnabledAddons with first-seen order, skipping blanks/dupes.
func mergeAddonNames(lists ...[]string) []string {
	seen := make(map[string]bool)
	out := make([]string, 0)
	for _, list := range lists {
		for _, name := range list {
			name = strings.TrimSpace(name)
			if name == "" || seen[name] {
				continue
			}
			seen[name] = true
			out = append(out, name)
		}
	}
	return out
}

// ApplicationProviders returns app-layer service providers (routes, etc.).
// Auth/dashboard providers are added by make:auth / make:dashboard.
func ApplicationProviders() []core.Provider {
	return []core.Provider{
		&providers.AppServiceProvider{},
		&providers.DatabaseServiceProvider{},
		&providers.EventServiceProvider{},
		&providers.ScheduleServiceProvider{},
		&providers.RouteServiceProvider{},
	}
}

// MinimalProviders is foundation + app wiring without addons.
func MinimalProviders() []core.Provider {
	out := make([]core.Provider, 0, len(foundation.Providers())+len(ApplicationProviders()))
	out = append(out, foundation.Providers()...)
	out = append(out, ApplicationProviders()...)
	return out
}

// CoreProviders boots secure kernel only.
func CoreProviders() []core.Provider {
	return foundation.KernelProviders()
}

// KernelProviders is an alias of CoreProviders.
func KernelProviders() []core.Provider { return CoreProviders() }

func findBasePath() (string, error) {
	wd, err := lookWorkingDirectory()
	if err != nil {
		return ".", err
	}
	return wd, nil
}
