package core

import (
	"context"
	"fmt"
	stdhttp "net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	appconfig "github.com/zatrano/framework/config"
	"github.com/zatrano/framework/core/ai"
	"github.com/zatrano/framework/core/apitoken"
	"github.com/zatrano/framework/core/assets"
	"github.com/zatrano/framework/core/audit"
	"github.com/zatrano/framework/core/auth"
	"github.com/zatrano/framework/core/authorization"
	"github.com/zatrano/framework/core/backup"
	"github.com/zatrano/framework/core/billing"
	"github.com/zatrano/framework/core/broadcasting"
	"github.com/zatrano/framework/core/bus"
	"github.com/zatrano/framework/core/cache"
	"github.com/zatrano/framework/core/circuit"
	"github.com/zatrano/framework/core/config"
	"github.com/zatrano/framework/core/container"
	appcontext "github.com/zatrano/framework/core/context"
	"github.com/zatrano/framework/core/database"
	"github.com/zatrano/framework/core/database/migration"
	"github.com/zatrano/framework/core/database/query"
	"github.com/zatrano/framework/core/database/seeder"
	"github.com/zatrano/framework/core/docs"
	"github.com/zatrano/framework/core/encryption"
	"github.com/zatrano/framework/core/enums"
	"github.com/zatrano/framework/core/env"
	"github.com/zatrano/framework/core/events"
	"github.com/zatrano/framework/core/exceptions"
	"github.com/zatrano/framework/core/features"
	"github.com/zatrano/framework/core/filesystem"
	"github.com/zatrano/framework/core/flash"
	"github.com/zatrano/framework/core/geo"
	"github.com/zatrano/framework/core/graphql"
	"github.com/zatrano/framework/core/hashid"
	"github.com/zatrano/framework/core/hashing"
	"github.com/zatrano/framework/core/health"
	"github.com/zatrano/framework/core/http"
	"github.com/zatrano/framework/core/httpclient"
	"github.com/zatrano/framework/core/inspector"
	"github.com/zatrano/framework/core/localization"
	"github.com/zatrano/framework/core/lock"
	"github.com/zatrano/framework/core/log"
	"github.com/zatrano/framework/core/mail"
	"github.com/zatrano/framework/core/maintenance"
	"github.com/zatrano/framework/core/middleware"
	"github.com/zatrano/framework/core/mongo"
	"github.com/zatrano/framework/core/notification"
	"github.com/zatrano/framework/core/oauth"
	"github.com/zatrano/framework/core/observability"
	"github.com/zatrano/framework/core/octane"
	"github.com/zatrano/framework/core/orm"
	"github.com/zatrano/framework/core/otp"
	"github.com/zatrano/framework/core/pulse"
	"github.com/zatrano/framework/core/queue"
	"github.com/zatrano/framework/core/ratelimit"
	"github.com/zatrano/framework/core/report"
	"github.com/zatrano/framework/core/routing"
	"github.com/zatrano/framework/core/schedule"
	"github.com/zatrano/framework/core/search"
	"github.com/zatrano/framework/core/session"
	"github.com/zatrano/framework/core/shorturl"
	"github.com/zatrano/framework/core/sitemap"
	"github.com/zatrano/framework/core/social"
	"github.com/zatrano/framework/core/tenancy"
	"github.com/zatrano/framework/core/trustedproxy"
	urlgen "github.com/zatrano/framework/core/url"
	"github.com/zatrano/framework/core/validation"
	"github.com/zatrano/framework/core/view"
	"github.com/zatrano/framework/core/webauthn"
	"github.com/zatrano/framework/core/webhooks"
	"github.com/zatrano/framework/core/wellknown"
)

// Provider boots services into the application.
type Provider interface {
	Register(app *Application)
	Boot(app *Application)
}

// Application is the ZATRANO application kernel.
type Application struct {
	basePath      string
	container     *container.Container
	config        *config.Repository
	router        *routing.Router
	view          *view.Engine
	logger        *log.Logger
	session       *session.Manager
	db            *database.Manager
	cache         *cache.Manager
	events        *events.Dispatcher
	mail          *mail.Manager
	queue         *queue.Manager
	auth          *auth.Manager
	translator    *localization.Translator
	scheduler     *schedule.Scheduler
	httpClient    *httpclient.Client
	notifications *notification.Manager
	pushSender    notification.PushSender
	smsSender     notification.SmsSender
	broadcast     *broadcasting.Manager
	files         *filesystem.Manager
	rateLimiter   *ratelimit.Limiter
	gate          *authorization.Gate
	ctx           *appcontext.Store
	urls          *urlgen.Generator
	encrypter     *encryption.Encrypter
	hasher        *hashing.Manager
	metrics       *observability.Metrics
	health        *health.Manager
	features      *features.Manager
	tenancy       *tenancy.Manager
	graphql       *graphql.Schema
	audit         *audit.Manager
	webhooks      *webhooks.Manager
	passwords     *auth.PasswordBroker
	maintenance   *maintenance.Manager
	tokens        *apitoken.Manager
	search        *search.Manager
	inspector     *inspector.Manager
	assets        *assets.Manifest
	exceptions    *exceptions.Handler
	social        *social.Manager
	enums         *enums.Registry
	bus           *bus.Bus
	pulse         *pulse.Dashboard
	backups       *backup.Manager
	docs          *docs.Repository
	billing       *billing.Manager
	mongo         *mongo.Client
	oauth         *oauth.Server
	octane        *octane.Runtime
	ai            *ai.Manager
	sitemap       *sitemap.Builder
	locks         *lock.Manager
	circuits      *circuit.Manager
	hashids       *hashid.Hasher
	shorturls     *shorturl.Manager
	wellknown     *wellknown.Repository
	geo           *geo.Resolver
	reports       *report.Manager
	webauthn      *webauthn.Manager
	otp           *otp.Manager
	migrations    []migration.Migration
	seeders       []seeder.Seeder
	providers     []Provider
	booted        bool
	environment   string
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
		providers: make([]Provider, 0),
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
func (app *Application) Container() *container.Container {
	return app.container
}

// Config returns the config repository.
func (app *Application) Config() *config.Repository {
	return app.config
}

// Router returns the HTTP router.
func (app *Application) Router() *routing.Router {
	return app.router
}

// View returns the view engine.
func (app *Application) View() *view.Engine {
	return app.view
}

// Logger returns the application logger.
func (app *Application) Logger() *log.Logger {
	return app.logger
}

// Session returns the session manager.
func (app *Application) Session() *session.Manager {
	return app.session
}

// DB returns the database manager.
func (app *Application) DB() *database.Manager {
	return app.db
}

// Table starts a query builder on a table.
func (app *Application) Table(table string) (*query.Builder, error) {
	if app.db == nil {
		return nil, fmt.Errorf("database not configured")
	}
	return app.db.Table(table)
}

// SetMigrations registers application migrations.
func (app *Application) SetMigrations(items []migration.Migration) {
	app.migrations = items
}

// Migrations returns registered migrations.
func (app *Application) Migrations() []migration.Migration {
	return app.migrations
}

// SetSeeders registers application seeders.
func (app *Application) SetSeeders(items ...seeder.Seeder) {
	app.seeders = items
}

// Seeders returns registered seeders.
func (app *Application) Seeders() []seeder.Seeder {
	return app.seeders
}

// Migrator creates a migrator for registered migrations.
func (app *Application) Migrator() (*migration.Migrator, error) {
	if err := app.ensureDatabase(); err != nil {
		return nil, err
	}
	db, err := app.db.DB()
	if err != nil {
		return nil, err
	}
	driver, err := app.db.DriverName()
	if err != nil {
		return nil, err
	}
	return migration.NewMigrator(db, driver, app.migrations), nil
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

		app.config.Load("database", map[string]any{
			"default": env.Get("DB_CONNECTION", "sqlite"),
			"connections": map[string]any{
				"sqlite": map[string]any{
					"driver":   "sqlite",
					"database": env.Get("DB_DATABASE", "database/database.sqlite"),
				},
				"mysql": map[string]any{
					"driver":   "mysql",
					"host":     env.Get("DB_HOST", "127.0.0.1"),
					"port":     env.Get("DB_PORT", "3306"),
					"database": env.Get("DB_DATABASE", "zatrano"),
					"username": env.Get("DB_USERNAME", "root"),
					"password": env.Get("DB_PASSWORD", ""),
				},
				"pgsql": map[string]any{
					"driver":   "pgsql",
					"host":     env.Get("DB_HOST", "127.0.0.1"),
					"port":     env.Get("DB_PORT", "5432"),
					"database": env.Get("DB_DATABASE", "zatrano"),
					"username": env.Get("DB_USERNAME", "root"),
					"password": env.Get("DB_PASSWORD", ""),
				},
			},
		})
		app.config.Load("auth", appconfig.Auth())
		app.config.Load("notifications", appconfig.Notifications())
	}
	if app.environment == "" {
		app.environment = app.config.GetString("app.env", "local")
	}

	logger, err := log.New(env.Get("LOG_LEVEL", "debug"), app.BasePath("storage", "logs", "zatrano.log"))
	if err != nil {
		return err
	}
	app.logger = logger
	app.container.Instance("log", logger)

	if err := app.bootDatabase(); err != nil {
		return err
	}

	if err := app.bootSupportServices(); err != nil {
		return err
	}

	app.view = view.New(app.BasePath("views"))
	app.view.EnableCache(!app.IsDebug())
	app.view.Share("appName", app.config.GetString("app.name", "ZATRANO"))
	if app.translator != nil {
		app.view.Share("locale", app.translator.GetLocale())
		app.view.AddFunc("trans", func(key string) string {
			return app.translator.Get(key)
		})
		app.view.AddFunc("choice", func(key string, number any) string {
			n := 0
			switch v := number.(type) {
			case int:
				n = v
			case int64:
				n = int(v)
			case float64:
				n = int(v)
			case string:
				if parsed, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
					n = parsed
				}
			default:
				n, _ = strconv.Atoi(fmt.Sprint(number))
			}
			return app.translator.Choice(key, n)
		})
	}
	if app.assets != nil {
		app.view.AddFunc("vite", func(path string) string {
			return app.assets.URL(path)
		})
		app.view.AddFunc("mix", func(path string) string {
			return app.assets.URL(path)
		})
	}
	app.view.SetEnvironment(app.environment)
	app.container.Instance("view", app.view)
	if app.mail != nil {
		app.mail.SetView(app.view)
	}
	if app.scheduler != nil {
		app.scheduler.SetMutexPath(app.BasePath("storage", "framework", "schedule"))
	}

	app.session = session.NewManager(
		app.BasePath("storage", "framework", "sessions"),
		env.GetInt("SESSION_LIFETIME", 120),
	)
	app.container.Instance("session", app.session)
	if app.auth != nil {
		app.auth.SetSessionManager(app.session)
	}
	if app.passwords != nil {
		app.passwords.SetSessionManager(app.session)
	}

	app.router.Use(
		trustedproxy.FromEnv(),
		app.exceptionMiddleware(),
		middleware.RequestID,
		middleware.SecurityHeaders,
		observability.Timing(app.metrics, func(format string, args ...any) {
			if app.logger != nil {
				app.logger.Infof(format, args...)
			}
		}),
		app.sessionMiddleware(),
		app.localeMiddleware(),
		middleware.TrimStrings(),
		middleware.ConvertEmptyStringsToNull("password", "password_confirmation", "current_password"),
		app.viewMiddleware(),
	)
	if env.GetBool("CORS_ENABLED", true) {
		app.router.Use(middleware.CORSFromEnv())
	}
	if app.octane != nil {
		app.router.Use(app.octane.Middleware())
	}
	if app.maintenance != nil {
		app.router.Use(app.maintenance.Middleware())
	}
	if app.inspector != nil {
		app.router.Use(app.inspector.Middleware())
	}
	if app.audit != nil {
		app.router.Use(app.audit.Middleware())
	}

	for _, provider := range app.providers {
		provider.Register(app)
	}
	for _, provider := range app.providers {
		provider.Boot(app)
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

	// Static files from public/
	publicPath := app.BasePath("public", filepath.Clean(r.URL.Path))
	if r.URL.Path != "/" {
		if info, err := os.Stat(publicPath); err == nil && !info.IsDir() {
			stdhttp.ServeFile(w, r, publicPath)
			return
		}
	}

	resp := app.router.Dispatch(req)
	if resp == nil {
		resp = http.Abort(204)
	}

	if resp.ViewName() != "" && app.view != nil {
		data := resp.ViewData()
		if data == nil {
			data = map[string]any{}
		}
		if sess := req.Session(); sess != nil {
			if token, ok := sess.Get("_csrf_token").(string); ok {
				data["_token"] = token
			}
			data["flash"] = flash.All(req)
			data["old"] = flash.OldInput(req)
			data["errors"] = validation.ErrorsFromSession(req)
			data["errorBags"] = validation.ErrorBagsFromSession(req)
		} else {
			data["old"] = map[string]string{}
			data["errors"] = validation.NewMessageBag(nil)
			data["errorBags"] = map[string]any{}
		}
		authenticated := app.auth != nil && app.auth.Check(req)
		data["auth"] = authenticated
		data["guest"] = !authenticated
		if app.translator != nil {
			data["locale"] = app.translator.GetLocale()
			langPath := app.BasePath("lang")
			data["langPublished"] = localization.Published(langPath)
			data["locales"] = localization.Options(langPath, app.translator.GetLocale())
		}
		var user auth.Authenticatable
		if authenticated {
			user = app.auth.User(req)
			data["user"] = user
		}
		if app.gate != nil {
			data["__can"] = func(ability string, args ...any) bool {
				return app.gate.Allows(user, ability, args...)
			}
		}
		html, err := app.view.Render(resp.ViewName(), data)
		if err != nil {
			if app.IsDebug() {
				resp = http.HTML(fmt.Sprintf("<h1>View Error</h1><pre>%v</pre>", err)).Status(500)
			} else {
				resp = http.Abort(500, "View rendering failed")
			}
		} else {
			resp.SetContent([]byte(html), "text/html; charset=utf-8")
		}
	}

	if bag, ok := req.Session().(*session.Bag); ok && bag != nil {
		_ = app.session.Save(bag)
		stdhttp.SetCookie(w, &stdhttp.Cookie{
			Name:     app.session.CookieName(),
			Value:    bag.ID(),
			Path:     "/",
			HttpOnly: true,
			SameSite: stdhttp.SameSiteLaxMode,
			MaxAge:   int(time.Hour.Seconds() * 2),
		})
	}

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

func (app *Application) sessionMiddleware() routing.MiddlewareFunc {
	return func(next routing.HandlerFunc) routing.HandlerFunc {
		return func(req *http.Request) *http.Response {
			id := req.Cookie(app.session.CookieName())
			bag, err := app.session.Start(id)
			if err == nil {
				req.SetSession(bag)
			}
			return next(req)
		}
	}
}

func (app *Application) localeMiddleware() routing.MiddlewareFunc {
	return func(next routing.HandlerFunc) routing.HandlerFunc {
		return func(req *http.Request) *http.Response {
			if app.translator != nil {
				langPath := app.BasePath("lang")
				locale := ""
				if sess := req.Session(); sess != nil {
					if raw, ok := sess.Get("locale").(string); ok {
						locale = strings.TrimSpace(strings.ToLower(raw))
					}
				}
				if locale == "" {
					locale = negotiateLocale(req.Header("Accept-Language"), localization.Available(langPath))
				}
				if locale != "" && localization.HasLocale(langPath, locale) {
					app.translator.SetLocale(locale)
					_ = app.translator.Load(locale)
				}
				if app.view != nil {
					app.view.Share("locale", app.translator.GetLocale())
				}
			}
			return next(req)
		}
	}
}

func negotiateLocale(header string, available []string) string {
	header = strings.TrimSpace(header)
	if header == "" || len(available) == 0 {
		return ""
	}
	allowed := map[string]string{}
	for _, code := range available {
		code = strings.ToLower(strings.TrimSpace(code))
		allowed[code] = code
		if i := strings.IndexByte(code, '-'); i > 0 {
			allowed[code[:i]] = code
		}
	}
	for _, part := range strings.Split(header, ",") {
		tag := strings.TrimSpace(strings.Split(part, ";")[0])
		tag = strings.ToLower(tag)
		if tag == "" || tag == "*" {
			continue
		}
		if code, ok := allowed[tag]; ok {
			return code
		}
		if i := strings.IndexByte(tag, '-'); i > 0 {
			if code, ok := allowed[tag[:i]]; ok {
				return code
			}
		}
	}
	return ""
}

func (app *Application) exceptionMiddleware() routing.MiddlewareFunc {
	if app.exceptions != nil {
		return app.exceptions.Middleware()
	}
	return middleware.Recover
}

func (app *Application) viewMiddleware() routing.MiddlewareFunc {
	return func(next routing.HandlerFunc) routing.HandlerFunc {
		return func(req *http.Request) *http.Response {
			resp := next(req)
			return resp
		}
	}
}

func (app *Application) bootDatabase() error {
	defaultConn := app.config.GetString("database.default", "sqlite")
	connections := map[string]database.ConnectionConfig{}

	rawConnections, ok := app.config.Get("database.connections").(map[string]any)
	if !ok {
		rawConnections = map[string]any{}
	}

	for name, raw := range rawConnections {
		cfgMap, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		connections[name] = database.ConnectionConfig{
			Driver:   asString(cfgMap["driver"]),
			Host:     asString(cfgMap["host"]),
			Port:     asString(cfgMap["port"]),
			Database: asString(cfgMap["database"]),
			Username: asString(cfgMap["username"]),
			Password: asString(cfgMap["password"]),
			Charset:  asString(cfgMap["charset"]),
		}
	}

	app.db = database.NewManager(database.Config{
		Default:     defaultConn,
		Connections: connections,
	}, app.basePath)
	app.container.Instance("db", app.db)

	db, err := app.db.DB()
	if err != nil {
		return err
	}
	driver, err := app.db.DriverName()
	if err != nil {
		return err
	}
	orm.Configure(db, driver)
	if app.events != nil {
		orm.SetDispatcher(app.events)
	}
	return nil
}

func (app *Application) ensureDatabase() error {
	if !app.booted {
		if err := app.Bootstrap(); err != nil {
			return err
		}
	}
	if app.db == nil {
		return fmt.Errorf("database not configured")
	}
	return nil
}

func asString(value any) string {
	if value == nil {
		return ""
	}
	if s, ok := value.(string); ok {
		return s
	}
	return fmt.Sprint(value)
}
