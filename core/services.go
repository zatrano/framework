package core

import (
	appcontext "github.com/zatrano/framework/packages/context"
	"github.com/zatrano/framework/packages/encryption"
	"github.com/zatrano/framework/packages/exceptions"
	"github.com/zatrano/framework/packages/hashing"
	"github.com/zatrano/framework/packages/health"
	"github.com/zatrano/framework/packages/maintenance"
	"github.com/zatrano/framework/packages/observability"
	"github.com/zatrano/framework/packages/ratelimit"
	"github.com/zatrano/framework/packages/report"
	urlgen "github.com/zatrano/framework/packages/url"
	"github.com/zatrano/framework/packages/version"
)

// RateLimiter returns the rate limiter.
func (app *Application) RateLimiter() *ratelimit.Limiter {
	return app.rateLimiter
}

// Context returns the application context store.
func (app *Application) Context() *appcontext.Store {
	return app.ctx
}

// URL returns the URL generator.
func (app *Application) URL() *urlgen.Generator {
	return app.urls
}

// Encrypter returns the encryption service.
func (app *Application) Encrypter() *encryption.Encrypter {
	return app.encrypter
}

// Hash returns the hashing manager.
func (app *Application) Hash() *hashing.Manager {
	return app.hasher
}

// Metrics returns the observability metrics collector.
func (app *Application) Metrics() *observability.Metrics {
	return app.metrics
}

// Health returns the health check manager.
func (app *Application) Health() *health.Manager {
	return app.health
}

// Version returns the application/framework version.
func (app *Application) Version() string {
	return version.Get()
}

// Maintenance returns the maintenance mode manager.
func (app *Application) Maintenance() *maintenance.Manager {
	return app.maintenance
}

// Exceptions returns the exception handler.
func (app *Application) Exceptions() *exceptions.Handler {
	return app.exceptions
}

// Reports returns the exception report manager.
func (app *Application) Reports() *report.Manager {
	return app.reports
}
