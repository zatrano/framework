package reference

import (
	"context"
	"errors"
	"fmt"

	"github.com/zatrano/framework/v2/bootstrap"
	"github.com/zatrano/framework/v2/contracts"
	"github.com/zatrano/framework/v2/examples/reference/internal/domain"
	"github.com/zatrano/framework/v2/examples/reference/internal/repository"
	"github.com/zatrano/framework/v2/examples/reference/internal/service"
	"github.com/zatrano/framework/v2/examples/reference/internal/transport"
	"github.com/zatrano/framework/v2/examples/reference/internal/worker"
	"github.com/zatrano/framework/v2/kernel"
	"github.com/zatrano/framework/v2/kernel/routing"
)

const (
	keySettings = "reference.settings"
	keyItems    = "reference.items"
	keyService  = "reference.service"
	keyWorker   = "reference.worker"
)

var (
	_ contracts.Provider          = (*ConfigProvider)(nil)
	_ contracts.Provider          = (*CoreProvider)(nil)
	_ contracts.Provider          = (*HTTPProvider)(nil)
	_ contracts.LifecycleProvider = (*WorkerProvider)(nil)
)

// Providers is the application provider list. bootstrap.App prepends
// KernelServiceProvider.
func Providers() []kernel.Provider {
	return []kernel.Provider{
		&ConfigProvider{},
		&CoreProvider{},
		&HTTPProvider{},
		&WorkerProvider{},
	}
}

// Assemble builds an application the same way a test (or a consumer that
// cannot use bootstrap.App) would: KernelServiceProvider plus application
// providers. It does not call Bootstrap.
func Assemble(base string, extra ...kernel.Provider) *kernel.Application {
	app := kernel.NewApplication(base)
	all := []kernel.Provider{&bootstrap.KernelServiceProvider{}}
	all = append(all, Providers()...)
	all = append(all, extra...)
	app.RegisterProviders(all...)
	return app
}

// ConfigProvider loads environment settings during Register.
type ConfigProvider struct{}

func (p *ConfigProvider) Name() string { return "reference.ConfigProvider" }

func (p *ConfigProvider) Register(app contracts.App) error {
	settings, err := LoadSettings()
	if err != nil {
		return err
	}
	app.Container().Instance(keySettings, settings)
	return nil
}

func (p *ConfigProvider) Boot(contracts.App) error { return nil }

// CoreProvider binds the domain service and in-memory repository.
type CoreProvider struct{}

func (p *CoreProvider) Name() string { return "reference.CoreProvider" }

func (p *CoreProvider) Register(app contracts.App) error {
	repo := repository.NewMemory()
	repo.Seed(domain.Item{ID: "1", Name: "alpha"})
	app.Container().Instance(keyItems, repo)
	app.Container().Instance(keyService, service.New(repo))
	return nil
}

func (p *CoreProvider) Boot(contracts.App) error { return nil }

// HTTPProvider registers routes during Boot (before the kernel freezes routing).
type HTTPProvider struct{}

func (p *HTTPProvider) Name() string { return "reference.HTTPProvider" }

func (p *HTTPProvider) Register(contracts.App) error { return nil }

func (p *HTTPProvider) Boot(app contracts.App) error {
	r := routing.From(app)
	if r == nil {
		return errors.New("router unavailable")
	}
	settings, err := settingsFrom(app)
	if err != nil {
		return err
	}
	items, err := itemsFrom(app)
	if err != nil {
		return err
	}
	w, err := workerFrom(app)
	if err != nil {
		return err
	}
	r.Get("/up", transport.Up).As("up")
	r.Get("/api/v1/status", transport.Status(settings.Name, w)).As("status")
	r.Get("/api/v1/items/{id}", transport.ShowItem(items)).As("items.show")
	return nil
}

// WorkerProvider starts the ticker in Start and stops it in Stop.
// Stop does not receive App; the provider keeps the ticker pointer from Register.
type WorkerProvider struct {
	ticker *worker.Ticker
}

func (p *WorkerProvider) Name() string { return "reference.WorkerProvider" }

func (p *WorkerProvider) Register(app contracts.App) error {
	w := &worker.Ticker{}
	p.ticker = w
	app.Container().Instance(keyWorker, w)
	return nil
}

func (p *WorkerProvider) Boot(contracts.App) error { return nil }

func (p *WorkerProvider) Start(app contracts.App) error {
	w, err := workerFrom(app)
	if err != nil {
		return err
	}
	p.ticker = w
	settings, err := settingsFrom(app)
	if err != nil {
		return err
	}
	return w.Start(settings.PollInterval)
}

func (p *WorkerProvider) Stop(ctx context.Context) error {
	if p.ticker == nil {
		return nil
	}
	return p.ticker.Stop(ctx)
}

func settingsFrom(app contracts.App) (Settings, error) {
	raw, err := app.Make(keySettings)
	if err != nil {
		return Settings{}, err
	}
	s, ok := raw.(Settings)
	if !ok {
		return Settings{}, fmt.Errorf("reference.settings has unexpected type %T", raw)
	}
	return s, nil
}

func itemsFrom(app contracts.App) (*service.Items, error) {
	raw, err := app.Make(keyService)
	if err != nil {
		return nil, err
	}
	s, ok := raw.(*service.Items)
	if !ok {
		return nil, fmt.Errorf("reference.service has unexpected type %T", raw)
	}
	return s, nil
}

func workerFrom(app contracts.App) (*worker.Ticker, error) {
	raw, err := app.Make(keyWorker)
	if err != nil {
		return nil, err
	}
	w, ok := raw.(*worker.Ticker)
	if !ok {
		return nil, fmt.Errorf("reference.worker has unexpected type %T", raw)
	}
	return w, nil
}
