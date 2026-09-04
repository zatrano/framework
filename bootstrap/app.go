package bootstrap

import (
	"fmt"

	"github.com/zatrano/framework/bootstrap/addons"
	"github.com/zatrano/framework/contracts"
	"github.com/zatrano/framework/kernel"
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

// WithAddons selects a subset of blank-imported packages (does not merge EnabledAddons).
func WithAddons(names ...string) Option {
	return func(o *appOptions) {
		o.addonsSet = true
		o.addons = append([]string(nil), names...)
	}
}

// App creates the application.
// With no options it boots the kernel plus every package the process blank-imported
// (self-registered). It loads no session/database/view/auth unless those packages
// are imported.
func App(opts ...Option) *kernel.Application {
	cfg := appOptions{}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	var extra []contracts.Provider
	var err error
	if cfg.addonsSet {
		extra, err = addons.Select(cfg.addons...)
	} else {
		extra = addons.DefaultPackageProviders()
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
	if cfg.addonsSet {
		app.SetEnabledAddons(cfg.addons)
	} else {
		app.SetEnabledAddons(addons.Names())
	}
	return app
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
