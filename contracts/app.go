package contracts

import "net/http"

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
	ServeHTTP(w http.ResponseWriter, r *http.Request)
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
