package kernel_test

import (
	"testing"

	"github.com/zatrano/framework/contracts"
	"github.com/zatrano/framework/kernel"
	"github.com/zatrano/framework/kernel/config"
	"github.com/zatrano/framework/kernel/http"
	"github.com/zatrano/framework/kernel/routing"
)

type countingProvider struct {
	registers int
	boots     int
}

func (p *countingProvider) Register(app contracts.App) error {
	p.registers++
	app.Router().Get("/once", func(req *http.Request) *http.Response {
		return http.Text("ok")
	})
	return nil
}

func (p *countingProvider) Boot(app contracts.App) error {
	p.boots++
	return nil
}

func closeAppLog(t *testing.T, app *kernel.Application) {
	t.Helper()
	if c, ok := app.Logger().(interface{ Close() error }); ok && c != nil {
		_ = c.Close()
	}
}

func TestBootstrapIsIdempotent(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	p := &countingProvider{}
	app.RegisterProviders(p)
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	if p.registers != 1 || p.boots != 1 {
		t.Fatalf("register=%d boot=%d", p.registers, p.boots)
	}
	if !app.Bootstrapped() {
		t.Fatal("expected bootstrapped")
	}
	r := routing.From(app)
	if r == nil || !r.Frozen() {
		t.Fatal("expected frozen router")
	}
	defer func() {
		if recover() == nil {
			t.Fatal("expected freeze panic")
		}
	}()
	r.Get("/late", func(req *http.Request) *http.Response {
		return http.Text("no")
	})
}

func TestRegisterProvidersAfterBootstrapPanics(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	app.RegisterProviders(&countingProvider{})
}

func TestConfigFrozenAfterBootstrap(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	repo, ok := app.Config().(*config.Repository)
	if !ok || repo == nil || !repo.Frozen() {
		t.Fatal("expected frozen config repository")
	}
}
