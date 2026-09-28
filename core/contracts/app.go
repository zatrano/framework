package contracts

import (
	"context"
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
	Environment() string
	IsProduction() bool
	IsDebug() bool
	RegisterProviders(providers ...Provider)
	Bootstrap() error
	BootstrapContext(ctx context.Context) error
	Start() error
	StartContext(ctx context.Context) error
	Stop(ctx context.Context) error
	// Handle serves one HTTP request. Concrete type is *rawhttp.Ctx
	// (contracts stay dependency-neutral).
	Handle(ctx any)
	Run(addr string) error
	SetHTTPBridge(bridge HTTPBridge)
	HTTPBridge() HTTPBridge
}

// HTTPBridge is installed by template (Canvas render) and session at boot.
// Middleware/request/response are untyped so this package does not import
// core/kernel/http. Kernel asserts concrete types at the HTTP boundary.
type HTTPBridge interface {
	Middleware() []any
	// Finalize runs after routing. No net/http ResponseWriter (V3 / rawhttp).
	Finalize(req any, resp any) any
}

// Provider boots services into the application.
type Provider interface {
	Register(app App) error
	Boot(app App) error
}

// LifecycleProvider owns long-running work (queue workers, schedulers).
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
