package bootstrap

import (
	"fmt"

	"github.com/zatrano/framework/app/providers"
	"github.com/zatrano/framework/bootstrap/addons"
	"github.com/zatrano/framework/bootstrap/foundation"
	"github.com/zatrano/framework/core"
)

// App creates the configured application (foundation + EnabledAddons + app providers).
// Production baseline: keep EnabledAddons lean (or empty) and enable packages via CLI.
func App() *core.Application {
	app, err := Boot(foundation.Providers(), EnabledAddons, ApplicationProviders()...)
	if err != nil {
		panic(err)
	}
	return app
}

// APIApp boots foundation + PresetAPI addons + app providers.
func APIApp() *core.Application {
	app, err := Boot(foundation.Providers(), PresetAPI, ApplicationProviders()...)
	if err != nil {
		panic(err)
	}
	return app
}

// WebApp boots foundation + PresetWeb addons + app providers.
func WebApp() *core.Application {
	app, err := Boot(foundation.Providers(), PresetWeb, ApplicationProviders()...)
	if err != nil {
		panic(err)
	}
	return app
}

// DemoApp boots foundation + every DemoAddons package + app providers (full exploration stack).
func DemoApp() *core.Application {
	app, err := Boot(foundation.Providers(), DemoAddons, ApplicationProviders()...)
	if err != nil {
		panic(err)
	}
	return app
}

// MinimalApp boots foundation + app providers with no addons.
// This is the smallest *useful web/API app* (DB, auth, session, view, …) — not the kernel.
func MinimalApp() *core.Application {
	app, err := Boot(foundation.Providers(), nil, ApplicationProviders()...)
	if err != nil {
		panic(err)
	}
	return app
}

// CoreApp boots only the secure kernel (env/config/log/HTTP stack; no DB/session/auth/addons/app routes).
func CoreApp() *core.Application {
	app, err := Boot(foundation.KernelProviders(), nil)
	if err != nil {
		panic(err)
	}
	return app
}

// KernelApp is an alias of CoreApp (kept for older call sites).
func KernelApp() *core.Application { return CoreApp() }

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

// ApplicationProviders returns app-layer service providers (routes, auth wiring, etc.).
func ApplicationProviders() []core.Provider {
	return []core.Provider{
		&providers.AppServiceProvider{},
		&providers.DatabaseServiceProvider{},
		&providers.AuthServiceProvider{},
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
