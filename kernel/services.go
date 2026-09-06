package kernel

import (
	"os"
	"strings"

	"github.com/zatrano/framework/v2/contracts"
)

var _ contracts.App = (*Application)(nil)

func lookup[T any](app *Application, key string) T {
	var zero T
	if app == nil {
		return zero
	}
	raw, err := app.Make(key)
	if err != nil || raw == nil {
		return zero
	}
	v, _ := raw.(T)
	return v
}

// RateLimiter returns the rate limiter when a package has registered one.
// Prefer ratelimit.From(app) from application and addon code.
func (app *Application) RateLimiter() contracts.RateLimiter {
	return lookup[contracts.RateLimiter](app, "rateLimiter")
}

// Context returns the application context store.
func (app *Application) Context() contracts.ContextStore {
	if app == nil || app.ctx == nil {
		return nil
	}
	return app.ctx
}

// URL returns the URL generator when a package has registered one.
func (app *Application) URL() contracts.URLGenerator {
	return lookup[contracts.URLGenerator](app, "url")
}

// Encrypter returns the encryption service.
func (app *Application) Encrypter() contracts.Encrypter {
	if app == nil || app.encrypter == nil {
		return nil
	}
	return app.encrypter
}

// Hash returns the hashing manager when a package has registered one.
func (app *Application) Hash() contracts.Hasher {
	return lookup[contracts.Hasher](app, "hash")
}

// Metrics returns the observability metrics collector when registered.
func (app *Application) Metrics() contracts.Metrics {
	return lookup[contracts.Metrics](app, "metrics")
}

// Health returns the health check manager when registered.
func (app *Application) Health() contracts.Health {
	return lookup[contracts.Health](app, "health")
}

// Version returns the application/framework version from VERSION.
func (app *Application) Version() string {
	if app == nil {
		return ""
	}
	raw, err := os.ReadFile(app.BasePath("VERSION"))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(raw))
}

// Maintenance returns the maintenance mode manager when registered.
func (app *Application) Maintenance() contracts.Maintenance {
	return lookup[contracts.Maintenance](app, "maintenance")
}

// Exceptions returns the exception handler.
func (app *Application) Exceptions() contracts.Exceptions {
	if app == nil || app.exceptions == nil {
		return nil
	}
	return &exceptionsFacade{inner: app.exceptions}
}

// Reports returns the exception report manager.
func (app *Application) Reports() contracts.Reports {
	if app == nil || app.reports == nil {
		return nil
	}
	return &reportsFacade{inner: app.reports}
}
