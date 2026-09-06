package bootstrap

import (
	"fmt"
	"strings"

	"github.com/zatrano/framework/v2/bootstrap/addons"
	"github.com/zatrano/framework/v2/contracts"
	"github.com/zatrano/framework/v2/kernel"
)

type appOptions struct {
	addons       []string
	addonsSet    bool
	appProviders []kernel.Provider
}

// Option configures App().
type Option func(*appOptions)

// WithProviders appends application-layer providers (routes, migrations, …).
func WithProviders(providers ...kernel.Provider) Option {
	return func(o *appOptions) {
		o.appProviders = append(o.appProviders, providers...)
	}
}

// WithAddons selects an explicit subset of blank-imported packages.
// Names are intersected with the process registry (unknown names are skipped).
// This overrides a registered consumer manifest; it does not merge with it.
func WithAddons(names ...string) Option {
	return func(o *appOptions) {
		o.addonsSet = true
		o.addons = append([]string(nil), names...)
	}
}

// App creates the application.
//
// Enablement:
//   - WithAddons(names) → names ∩ Imported (explicit override)
//   - consumer RegisterEnablement → Enabled ∩ Imported
//   - no manifest → DefaultMetas() (all imported; legacy / G-001)
//
// init() only fills the addon registry and may register a manifest. It never
// decides enablement by itself. App() does not read the framework EnabledAddons
// variable.
func App(opts ...Option) *kernel.Application {
	cfg := appOptions{}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	var extra []contracts.Provider
	var enabled []string
	var err error
	switch {
	case cfg.addonsSet:
		extra, enabled, err = providersForNames(cfg.addons)
	case enablementRegistered():
		extra, enabled, err = providersForNames(enablementNames())
	default:
		extra, enabled = providersFromMetas(addons.DefaultMetas())
	}
	if err != nil {
		panic(err)
	}
	providers := []kernel.Provider{&KernelServiceProvider{}}
	providers = append(providers, extra...)
	providers = append(providers, cfg.appProviders...)
	app, err := Boot(providers)
	if err != nil {
		panic(err)
	}
	app.SetEnabledAddons(enabled)
	return app
}

func providersForNames(names []string) ([]contracts.Provider, []string, error) {
	metas, err := addons.Resolve(intersectImported(names)...)
	if err != nil {
		return nil, nil, err
	}
	extra, enabled := providersFromMetas(metas)
	return extra, enabled, nil
}

func intersectImported(names []string) []string {
	imported := map[string]bool{}
	for _, n := range addons.Names() {
		imported[n] = true
	}
	out := make([]string, 0, len(names))
	seen := map[string]bool{}
	for _, name := range names {
		name = strings.ToLower(strings.TrimSpace(name))
		if name == "" || seen[name] || !imported[name] {
			continue
		}
		seen[name] = true
		out = append(out, name)
	}
	return out
}

func providersFromMetas(metas []addons.Meta) ([]contracts.Provider, []string) {
	extra := make([]contracts.Provider, 0, len(metas))
	enabled := make([]string, 0, len(metas))
	for _, m := range metas {
		enabled = append(enabled, m.Name)
		if m.Factory == nil {
			continue
		}
		extra = append(extra, m.Factory())
	}
	return extra, enabled
}

// Boot assembles an application from providers.
func Boot(providers []kernel.Provider, _ ...any) (*kernel.Application, error) {
	basePath, _ := findBasePath()
	application := kernel.NewApplication(basePath)
	application.RegisterProviders(providers...)
	return application, nil
}

// ApplicationProviders is empty in the framework repo; consumer apps pass WithProviders.
func ApplicationProviders() []kernel.Provider {
	return nil
}

func findBasePath() (string, error) {
	wd, err := lookWorkingDirectory()
	if err != nil {
		return ".", err
	}
	return wd, nil
}

// KernelServiceProvider boots primitive kernel services only.
type KernelServiceProvider struct{}

func (p *KernelServiceProvider) Register(app contracts.App) error {
	k, ok := app.(*kernel.Application)
	if !ok {
		return fmt.Errorf("kernel boot requires *kernel.Application")
	}
	return k.BootKernelServices()
}

func (p *KernelServiceProvider) Boot(app contracts.App) error { return nil }
