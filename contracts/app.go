package contracts

import (
	"context"
	stdhttp "net/http"
)

// App is the stable kernel surface for providers, addons, and CLI helpers.
// Package services (auth, database, queue, AI, …) resolve via their own From(app)
// helpers — they are not methods on this interface.
type App interface {
	BasePath(parts ...string) string
	Container() Container
	Make(abstract string) (any, error)
	Bound(abstract string) bool
	Config() ConfigRepository
	Router() Router
	Logger() Logger
	Context() ContextStore
	Encrypter() Encrypter
	Exceptions() Exceptions
	Reports() Reports
	SetMigrations(items any)
	Migrations() any
	SetSeeders(items any)
	Seeders() any
	Environment() string
	IsProduction() bool
	IsDebug() bool
	RegisterProviders(providers ...Provider)
	Bootstrap() error
	ServeHTTP(w stdhttp.ResponseWriter, r *stdhttp.Request)
	Run(addr string) error
	SetHTTPBridge(bridge HTTPBridge)
	HTTPBridge() HTTPBridge
}

// HTTPBridge is installed by view (render) and session (cookies, flash) at boot.
// Middleware entries are routing.MiddlewareFunc values.
// Finalize req is *framework/http.Request and resp is *framework/http.Response.
type HTTPBridge interface {
	Middleware() []any
	Finalize(w stdhttp.ResponseWriter, req any, resp any) any
}

// Provider boots services into the application.
type Provider interface {
	Register(app App) error
	Boot(app App) error
}

// LifecycleProvider is an optional Provider that owns long-running work
// (queue workers, schedulers, consumers). Boot initializes; Start launches;
// Stop shuts down with the process.
type LifecycleProvider interface {
	Provider
	Start(app App) error
	Stop(ctx context.Context) error
}

// Migrator runs outstanding migrations and rollbacks.
type Migrator interface {
	Migrate() error
	Rollback() error
	Status() error
	Fresh() error
}
