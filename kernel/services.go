package kernel

import (
	"github.com/zatrano/framework/contracts"
	"github.com/zatrano/framework/packages/version"
)

// RateLimiter returns the rate limiter.
func (app *Application) RateLimiter() contracts.RateLimiter {
	if app == nil || app.rateLimiter == nil {
		return nil
	}
	return app.rateLimiter
}

// Context returns the application context store.
func (app *Application) Context() contracts.ContextStore {
	if app == nil || app.ctx == nil {
		return nil
	}
	return app.ctx
}

// URL returns the URL generator.
func (app *Application) URL() contracts.URLGenerator {
	if app == nil || app.urls == nil {
		return nil
	}
	return app.urls
}

// Encrypter returns the encryption service.
func (app *Application) Encrypter() contracts.Encrypter {
	if app == nil || app.encrypter == nil {
		return nil
	}
	return app.encrypter
}

// Hash returns the hashing manager.
func (app *Application) Hash() contracts.Hasher {
	if app == nil || app.hasher == nil {
		return nil
	}
	return app.hasher
}

// Metrics returns the observability metrics collector.
func (app *Application) Metrics() contracts.Metrics {
	if app == nil || app.metrics == nil {
		return nil
	}
	return app.metrics
}

// Health returns the health check manager.
func (app *Application) Health() contracts.Health {
	if app == nil || app.health == nil {
		return nil
	}
	return app.health
}

// Version returns the application/framework version.
func (app *Application) Version() string {
	return version.Get()
}

// Maintenance returns the maintenance mode manager.
func (app *Application) Maintenance() contracts.Maintenance {
	if app == nil || app.maintenance == nil {
		return nil
	}
	return app.maintenance
}

// Exceptions returns the exception handler.
func (app *Application) Exceptions() contracts.Exceptions {
	if app == nil || app.exceptions == nil {
		return nil
	}
	return app.exceptions
}

// Reports returns the exception report manager.
func (app *Application) Reports() contracts.Reports {
	if app == nil || app.reports == nil {
		return nil
	}
	return app.reports
}
