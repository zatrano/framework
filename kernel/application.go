package kernel

import (
	"context"
	"errors"
	"fmt"
	stdlog "log"
	stdhttp "net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/zatrano/framework/v2/contracts"
	"github.com/zatrano/framework/v2/kernel/config"
	"github.com/zatrano/framework/v2/kernel/container"
	appcontext "github.com/zatrano/framework/v2/kernel/context"
	"github.com/zatrano/framework/v2/kernel/cookie"
	"github.com/zatrano/framework/v2/kernel/encryption"
	"github.com/zatrano/framework/v2/kernel/env"
	"github.com/zatrano/framework/v2/kernel/exceptions"
	"github.com/zatrano/framework/v2/kernel/http"
	"github.com/zatrano/framework/v2/kernel/log"
	"github.com/zatrano/framework/v2/kernel/middleware"
	"github.com/zatrano/framework/v2/kernel/report"
	"github.com/zatrano/framework/v2/kernel/routing"
	"github.com/zatrano/framework/v2/kernel/safepath"
	"github.com/zatrano/framework/v2/kernel/trustedproxy"
)

// Provider boots services into the application.
type Provider = contracts.Provider

type lifeState int

const (
	lifeCreated lifeState = iota
	lifeBootstrapping
	lifeBooted
	lifeStarting
	lifeRunning
	lifeStopping
	lifeStopped
	lifeBootFailed
)

// Application is the ZATRANO application kernel.
// Foundation and addon services live in the container (see accessors / package From helpers).
// Kernel fields below are set by BootKernelServices.
type Application struct {
	basePath           string
	container          *container.Container
	config             *config.Repository
	router             *routing.Router
	logger             *log.Logger
	ctx                *appcontext.Store
	encrypter          *encryption.Encrypter
	exceptions         *exceptions.Handler
	reports            *report.Manager
	httpBridge         contracts.HTTPBridge
	httpBridgeCaptured contracts.HTTPBridge
	httpBridgeFrozen   bool
	providers          []contracts.Provider
	life               lifeState
	lifeMu             sync.Mutex
	transitionMu       sync.Mutex
	environment        string
	enabledAddons      []string
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

// SetEnabledAddons records which imported addons this application will boot.
// Call before Bootstrap. Names are informational (imported ≠ enabled ≠ booted).
func (app *Application) SetEnabledAddons(names []string) {
	app.lifeMu.Lock()
	defer app.lifeMu.Unlock()
	if app.life != lifeCreated {
		panic("application: cannot change enabled addons after bootstrap")
	}
	app.enabledAddons = append([]string(nil), names...)
}

// EnabledAddons returns the enable-set recorded at construction (may be empty).
func (app *Application) EnabledAddons() []string {
	app.lifeMu.Lock()
	defer app.lifeMu.Unlock()
	return append([]string(nil), app.enabledAddons...)
}

// RegisterProviders registers service providers. Illegal after bootstrap has started.
func (app *Application) RegisterProviders(providers ...Provider) {
	app.lifeMu.Lock()
	defer app.lifeMu.Unlock()
	if app.life != lifeCreated {
		panic("application: cannot register providers after bootstrap")
	}
	app.providers = append(app.providers, providers...)
}

// Bootstrapped reports whether Bootstrap has completed.
func (app *Application) Bootstrapped() bool {
	app.lifeMu.Lock()
	defer app.lifeMu.Unlock()
	return app.life == lifeBooted || app.life == lifeStarting || app.life == lifeRunning || app.life == lifeStopping || app.life == lifeStopped
}

// httpReady reports whether the HTTP pipeline is usable. This is a Booted
// capability, not a Running one: Start launches process lifecycle providers.
func (app *Application) httpReady() bool {
	if app == nil {
		return false
	}
	app.lifeMu.Lock()
	defer app.lifeMu.Unlock()
	switch app.life {
	case lifeBooted, lifeStarting, lifeRunning, lifeStopping, lifeStopped:
		return true
	default:
		return false
	}
}

// BootstrapFailed reports whether Bootstrap returned an error. Failed
// applications cannot be retried; construct a new Application.
func (app *Application) BootstrapFailed() bool {
	app.lifeMu.Lock()
	defer app.lifeMu.Unlock()
	return app.life == lifeBootFailed
}

// Bootstrap loads environment, config, and core services.
// A second call is a no-op so providers, routes, and middleware are not registered twice.
// Bootstrap, Start, and Stop share one lock so concurrent Boot/Start cannot observe a
// half-open boot (C-005): the later call waits, then no-ops or continues from Booted.
func (app *Application) Bootstrap() error {
	app.transitionMu.Lock()
	defer app.transitionMu.Unlock()
	return app.bootstrapSerial()
}

func (app *Application) bootstrapSerial() error {
	app.lifeMu.Lock()
	switch app.life {
	case lifeBooted, lifeStarting, lifeRunning, lifeStopping, lifeStopped:
		app.lifeMu.Unlock()
		return nil
	case lifeBootstrapping:
		app.lifeMu.Unlock()
		return errors.New("application: bootstrap already in progress")
	case lifeBootFailed:
		app.lifeMu.Unlock()
		return errors.New("application: bootstrap failed; create a new Application")
	}
	app.life = lifeBootstrapping
	app.lifeMu.Unlock()

	if err := app.bootstrapLocked(); err != nil {
		app.lifeMu.Lock()
		app.life = lifeBootFailed
		app.lifeMu.Unlock()
		return err
	}
	app.lifeMu.Lock()
	app.life = lifeBooted
	app.lifeMu.Unlock()
	return nil
}

func (app *Application) loadEnvAppConfig() {
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

func (app *Application) bootstrapLocked() error {
	_ = env.Load(app.BasePath(".env"))

	app.environment = env.NormalizeAppEnv(env.Get("APP_ENV", "local"))

	configCache := app.BasePath("storage", "framework", "cache", "config.json")
	loadedCache := false
	if env.GetBool("APP_CONFIG_CACHE", true) && config.CacheExists(configCache) {
		if cached, err := config.LoadCache(configCache); err == nil {
			app.config.MergeCached(cached)
			loadedCache = true
		}
	}
	if !loadedCache {
		app.loadEnvAppConfig()
	}
	if app.environment == "" {
		app.environment = env.NormalizeAppEnv(app.config.GetString("app.env", "local"))
	}
	if app.environment == "" {
		app.environment = "local"
	}
	cookie.SetProductionPolicy(app.IsProduction())

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

	proxyMW, err := trustedproxy.FromEnv(app.IsProduction())
	if err != nil {
		return err
	}
	app.router.Use(
		proxyMW,
		app.exceptionMiddleware(),
		middleware.RequestID,
		middleware.SecurityHeadersWith(middleware.SecurityHeaderConfig{
			EnableHSTSOnHTTPS: app.IsProduction(),
		}),
	)
	if err := applyHTTPBridgeMiddleware(app); err != nil {
		return err
	}
	app.router.Use(
		middleware.TrimStrings(),
		middleware.ConvertEmptyStringsToNull("password", "password_confirmation", "current_password"),
	)
	if env.GetBool("CORS_ENABLED", true) {
		app.router.Use(middleware.CORSFromEnv(app.Environment()))
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

	// Register named routes after boot, then freeze the routing table.
	for _, route := range app.router.Routes() {
		app.router.RegisterName(route)
	}
	if err := app.router.Freeze(); err != nil {
		return err
	}
	app.config.Freeze()
	app.container.Freeze()

	app.logger.Infof("%s application bootstrapped (%s)", app.config.GetString("app.name"), app.environment)
	return nil
}

// Start launches optional LifecycleProvider.Start hooks. Bootstrap runs first.
// A second call while running is a no-op. Stopped applications cannot restart.
// Concurrent Bootstrap/Start/Stop calls are serialized. If a provider's Start
// returns an error, that provider must clean up any work it already launched;
// the kernel only stops providers that returned nil from Start.
func (app *Application) Start() error {
	app.transitionMu.Lock()
	defer app.transitionMu.Unlock()

	if err := app.bootstrapSerial(); err != nil {
		return err
	}

	app.lifeMu.Lock()
	switch app.life {
	case lifeRunning:
		app.lifeMu.Unlock()
		return nil
	case lifeStopped, lifeStopping:
		app.lifeMu.Unlock()
		return errors.New("application: cannot start a stopped application")
	case lifeBooted:
		app.life = lifeStarting
		app.lifeMu.Unlock()
	default:
		state := app.life
		app.lifeMu.Unlock()
		return fmt.Errorf("application: cannot start from state %d", state)
	}

	var started []contracts.LifecycleProvider
	for _, p := range app.providers {
		lp, ok := p.(contracts.LifecycleProvider)
		if !ok {
			continue
		}
		if err := lp.Start(app); err != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			_ = app.stopLifecycle(ctx, started)
			cancel()
			app.lifeMu.Lock()
			app.life = lifeBooted
			app.lifeMu.Unlock()
			return err
		}
		started = append(started, lp)
	}
	app.lifeMu.Lock()
	if app.life == lifeStarting {
		app.life = lifeRunning
	}
	app.lifeMu.Unlock()
	return nil
}

// Stop runs LifecycleProvider.Stop in reverse start order. No-op unless Running.
func (app *Application) Stop(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	app.transitionMu.Lock()
	defer app.transitionMu.Unlock()

	app.lifeMu.Lock()
	if app.life != lifeRunning {
		app.lifeMu.Unlock()
		return nil
	}
	app.life = lifeStopping
	app.lifeMu.Unlock()

	var started []contracts.LifecycleProvider
	for _, p := range app.providers {
		if lp, ok := p.(contracts.LifecycleProvider); ok {
			started = append(started, lp)
		}
	}
	err := app.stopLifecycle(ctx, started)
	app.lifeMu.Lock()
	app.life = lifeStopped
	app.lifeMu.Unlock()
	return err
}

func (app *Application) stopLifecycle(ctx context.Context, started []contracts.LifecycleProvider) error {
	var first error
	for i := len(started) - 1; i >= 0; i-- {
		if err := started[i].Stop(ctx); err != nil && first == nil {
			first = err
		}
	}
	return first
}

// ServeHTTP implements net/stdhttp.Handler.
func (app *Application) ServeHTTP(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	cw := http.TrackCommit(w)
	defer app.recoverServeHTTP(cw)

	if !app.httpReady() {
		_ = http.Abort(stdhttp.StatusServiceUnavailable).WriteTo(cw)
		return
	}

	if r != nil && r.Body != nil {
		r.Body = stdhttp.MaxBytesReader(cw, r.Body, http.MaxRequestBytes())
	}
	req := http.NewRequest(r)
	middleware.ApplyMethodOverride(req)

	var resp *http.Response
	if file := app.publicFile(req); file != nil {
		resp = app.router.Through(req, func(*http.Request) *http.Response { return file })
	} else {
		resp = app.router.Dispatch(req)
	}
	if resp == nil {
		resp = http.Abort(204)
	}
	resp = finalizeHTTPBridge(app, cw, req, resp)
	if cw.Committed() {
		return
	}
	if resp == nil {
		resp = http.Abort(204)
	}

	for _, c := range req.Cookies().Apply() {
		resp.WithCookie(c)
	}
	req.Cookies().Clear()

	_ = resp.WriteTo(cw)
}

func (app *Application) recoverServeHTTP(w *http.CommitWriter) {
	recovered := recover()
	if recovered == nil {
		return
	}
	if app != nil && app.logger != nil {
		app.logger.Errorf("http: panic recovered: %v", recovered)
	} else {
		stdlog.Printf("http: panic recovered: %v", recovered)
	}
	if w == nil || w.Committed() {
		return
	}
	defer func() { _ = recover() }()
	_ = http.Abort(stdhttp.StatusInternalServerError, "Internal Server Error").WriteTo(w)
}

func (app *Application) publicFile(req *http.Request) *http.Response {
	if req == nil || req.Raw() == nil {
		return nil
	}
	switch req.Method() {
	case stdhttp.MethodGet, stdhttp.MethodHead:
	default:
		return nil
	}
	path := req.Path()
	if path == "/" {
		return nil
	}
	publicPath, err := safepath.Resolve(app.BasePath("public"), path)
	if err != nil {
		return nil
	}
	publicPath, err = safepath.EvalUnder(app.BasePath("public"), publicPath)
	if err != nil {
		return nil
	}
	info, err := os.Stat(publicPath)
	if err != nil || info.IsDir() {
		return nil
	}
	return http.PublicFile(publicPath, req.Raw())
}

// Run starts the HTTP server with graceful shutdown on SIGINT/SIGTERM.
func (app *Application) Run(addr string) error {
	if err := app.Start(); err != nil {
		return err
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
		MaxHeaderBytes:    1 << 20,
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
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		stopErr := app.Stop(ctx)
		if err != nil {
			return err
		}
		return stopErr
	case sig := <-sigCh:
		app.logger.Infof("shutting down gracefully (%v)...", sig)
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			_ = app.Stop(ctx)
			return err
		}
		if err := app.Stop(ctx); err != nil {
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
