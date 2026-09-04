package contracts

import stdhttp "net/http"

// App is the stable application surface for providers, addons, and CLI helpers.
type App interface {
	BasePath(parts ...string) string
	Container() Container
	Make(abstract string) (any, error)
	Bound(abstract string) bool
	Config() ConfigRepository
	Router() Router
	Logger() Logger
	RateLimiter() RateLimiter
	Context() ContextStore
	URL() URLGenerator
	Encrypter() Encrypter
	Hash() Hasher
	Metrics() Metrics
	Health() Health
	Maintenance() Maintenance
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

// HTTPBridge is installed by session (and similar) at boot.
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

// Migrator runs outstanding migrations and rollbacks.
type Migrator interface {
	Migrate() error
	Rollback() error
	Status() error
	Fresh() error
}
