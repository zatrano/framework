package kernel

import (
	"context"
	"fmt"
	stdhttp "net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/zatrano/framework/contracts"
	"github.com/zatrano/framework/kernel/config"
	"github.com/zatrano/framework/kernel/container"
	appcontext "github.com/zatrano/framework/kernel/context"
	"github.com/zatrano/framework/kernel/encryption"
	"github.com/zatrano/framework/kernel/env"
	"github.com/zatrano/framework/kernel/exceptions"
	"github.com/zatrano/framework/kernel/http"
	"github.com/zatrano/framework/kernel/log"
	"github.com/zatrano/framework/kernel/middleware"
	"github.com/zatrano/framework/kernel/report"
	"github.com/zatrano/framework/kernel/routing"
	"github.com/zatrano/framework/kernel/safepath"
	"github.com/zatrano/framework/kernel/trustedproxy"
)

// Provider boots services into the application.
type Provider = contracts.Provider

// Application is the ZATRANO application kernel.
// Foundation and addon services live in the container (see accessors / package From helpers).
// Kernel fields below are set by BootKernelServices.
type Application struct {
	basePath    string
	container   *container.Container
	config      *config.Repository
	router      *routing.Router
	logger      *log.Logger
	ctx         *appcontext.Store
	encrypter   *encryption.Encrypter
	exceptions  *exceptions.Handler
	reports     *report.Manager
	httpBridge  contracts.HTTPBridge
	migrations  any
	seeders     any
	providers   []contracts.Provider
	booted      bool
	environment string
}

// NewApplication creates a new application instance.
func NewApplication(basePath string) *Application {
	if basePath == "" {
		basePath, _ = os.Getwd()
	}

	app := &Application{
		basePath:  basePath,
		container: container.New(),
		config:    config.New(),
		router:    routing.New(),
		providers: make([]contracts.Provider, 0),
	}

	app.container.Instance("app", app)
	app.container.Instance("config", app.config)
	app.container.Instance("router", app.router)

	return app
}

// BasePath returns a path relative to the application root.
func (app *Application) BasePath(parts ...string) string {
	return filepath.Join(append([]string{app.basePath}, parts...)...)
}

// Container returns the service container.
func (app *Application) Container() contracts.Container {
	if app == nil || app.container == nil {
		return nil
	}
	return app.container
}

// Make resolves a service from the container.
func (app *Application) Make(abstract string) (any, error) {
	if app == nil || app.container == nil {
		return nil, fmt.Errorf("container unavailable")
	}
	return app.container.Make(abstract)
}

// Bound reports whether a service is registered.
func (app *Application) Bound(abstract string) bool {
	if app == nil || app.container == nil {
		return false
	}
	return app.container.Bound(abstract)
}

// Config returns the config repository.
func (app *Application) Config() contracts.ConfigRepository {
	if app == nil || app.config == nil {
		return nil
	}
	return app.config
}

// Router returns the HTTP router.
func (app *Application) Router() contracts.Router {
	if app == nil || app.router == nil {
		return nil
	}
	return &routerFacade{inner: app.router}
}

// Logger returns the application logger.
func (app *Application) Logger() contracts.Logger {
	if app == nil || app.logger == nil {
		return nil
	}
	return app.logger
}

// SetMigrations registers application migrations (typically []migration.Migration).
func (app *Application) SetMigrations(items any) {
	app.migrations = items
}

// Migrations returns registered migrations (opaque; cast in database helpers).
func (app *Application) Migrations() any {
	return app.migrations
}

// SetSeeders registers application seeders (typically []seeder.Seeder).
func (app *Application) SetSeeders(items any) {
	app.seeders = items
}

// Seeders returns registered seeders (opaque; cast in database helpers).
func (app *Application) Seeders() any {
	return app.seeders
}

// Environment returns the current environment name.
func (app *Application) Environment() string {
	return app.environment
}

// IsProduction reports whether the app runs in production.
func (app *Application) IsProduction() bool {
	return app.environment == "production"
}

// IsDebug reports whether debug mode is enabled.
func (app *Application) IsDebug() bool {
	return app.config.GetBool("app.debug", true)
}

// RegisterProviders registers service providers.
func (app *Application) RegisterProviders(providers ...Provider) {
	app.providers = append(app.providers, providers...)
}

// Bootstrap loads environment, config, and core services.
func (app *Application) Bootstrap() error {
	_ = env.Load(app.BasePath(".env"))

	app.environment = env.Get("APP_ENV", "local")

	configCache := app.BasePath("storage", "framework", "cache", "config.json")
	if env.GetBool("APP_CONFIG_CACHE", true) && config.CacheExists(configCache) {
		if cached, err := config.LoadCache(configCache); err == nil {
			app.config.MergeCached(cached)
		}
	} else {
		app.config.Load("app", map[string]any{
			"name":     env.Get("APP_NAME", "ZATRANO"),
			"env":      app.environment,
			"debug":    env.GetBool("APP_DEBUG", true),
			"url":      env.Get("APP_URL", "http://localhost:8080"),
			"key":      env.Get("APP_KEY"),
			"locale":   env.Get("APP_LOCALE", "en"),
			"fallback": env.Get("APP_FALLBACK_LOCALE", "en"),
		})
	}
	if app.environment == "" {
		app.environment = app.config.GetString("app.env", "local")
	}

	if err := ensureProductionSecrets(app); err != nil {
		return err
	}

	logger, err := log.New(env.Get("LOG_LEVEL", "debug"), app.BasePath("storage", "logs", "zatrano.log"))
	if err != nil {
		return err
	}
	app.logger = logger
	app.container.Instance("log", logger)

	// Kernel only above. Foundation + packages + app providers Register next.
	for _, provider := range app.providers {
		if err := provider.Register(app); err != nil {
			return err
		}
	}

	app.router.Use(
		trustedproxy.FromEnv(),
		app.exceptionMiddleware(),
		middleware.RequestID,
		middleware.SecurityHeaders,
	)
	applyHTTPBridgeMiddleware(app)
	app.router.Use(
		middleware.TrimStrings(),
		middleware.ConvertEmptyStringsToNull("password", "password_confirmation", "current_password"),
	)
	if env.GetBool("CORS_ENABLED", true) {
		app.router.Use(middleware.CORSFromEnv())
	}
	if o := middlewareFrom(app, "octane"); o != nil {
		app.router.Use(o)
	}
	if o := middlewareFrom(app, "maintenance"); o != nil {
		app.router.Use(o)
	}
	if o := middlewareFrom(app, "metrics-timing"); o != nil {
		app.router.Use(o)
	}
	if o := middlewareFrom(app, "inspector"); o != nil {
		app.router.Use(o)
	}
	if o := middlewareFrom(app, "audit"); o != nil {
		app.router.Use(o)
	}

	for _, provider := range app.providers {
		if err := provider.Boot(app); err != nil {
			return err
		}
	}

	// Register named routes after boot.
	for _, route := range app.router.Routes() {
		app.router.RegisterName(route)
	}

	app.booted = true
	app.logger.Infof("%s application bootstrapped (%s)", app.config.GetString("app.name"), app.environment)
	return nil
}

// ServeHTTP implements net/stdhttp.Handler.
func (app *Application) ServeHTTP(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	req := http.NewRequest(r)
	middleware.ApplyMethodOverride(req)

	// Static files from public/ (never escape the public directory)
	if r.URL.Path != "/" {
		publicRoot := app.BasePath("public")
		publicPath, err := safepath.Resolve(publicRoot, r.URL.Path)
		if err == nil {
			if info, err := os.Stat(publicPath); err == nil && !info.IsDir() {
				stdhttp.ServeFile(w, r, publicPath)
				return
			}
		}
	}

	resp := app.router.Dispatch(req)
	if resp == nil {
		resp = http.Abort(204)
	}
	resp = finalizeHTTPBridge(app, w, req, resp)

	for _, c := range req.Cookies().Apply() {
		resp.WithCookie(c)
	}
	req.Cookies().Clear()

	_ = resp.WriteTo(w)
}

// Run starts the HTTP server with graceful shutdown on SIGINT/SIGTERM.
func (app *Application) Run(addr string) error {
	if !app.booted {
		if err := app.Bootstrap(); err != nil {
			return err
		}
	}
	if addr == "" {
		port := strings.TrimSpace(env.Get("APP_PORT", "8080"))
		if port == "" {
			port = "8080"
		}
		addr = ":" + port
	}

	server := &stdhttp.Server{
		Addr:              addr,
		Handler:           app,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		app.logger.Infof("ZATRANO server listening on %s", addr)
		if err := server.ListenAndServe(); err != nil && err != stdhttp.ErrServerClosed {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-errCh:
		return err
	case sig := <-sigCh:
		app.logger.Infof("shutting down gracefully (%v)...", sig)
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			return err
		}
		app.logger.Infof("server stopped")
		return nil
	}
}

func (app *Application) exceptionMiddleware() routing.MiddlewareFunc {
	if app.exceptions != nil {
		return app.exceptions.Middleware()
	}
	return middleware.Recover
}
