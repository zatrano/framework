package core

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"time"

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
	appcontext "github.com/zatrano/framework/core/context"
	"github.com/zatrano/framework/core/docs"
	"github.com/zatrano/framework/core/encryption"
	"github.com/zatrano/framework/core/enums"
	"github.com/zatrano/framework/core/env"
	"github.com/zatrano/framework/core/events"
	"github.com/zatrano/framework/core/exceptions"
	"github.com/zatrano/framework/core/features"
	"github.com/zatrano/framework/core/filesystem"
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
	"github.com/zatrano/framework/core/mail"
	"github.com/zatrano/framework/core/maintenance"
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
	"github.com/zatrano/framework/core/redisx"
	"github.com/zatrano/framework/core/report"
	"github.com/zatrano/framework/core/schedule"
	"github.com/zatrano/framework/core/search"
	"github.com/zatrano/framework/core/shorturl"
	"github.com/zatrano/framework/core/sitemap"
	"github.com/zatrano/framework/core/social"
	"github.com/zatrano/framework/core/tenancy"
	urlgen "github.com/zatrano/framework/core/url"
	"github.com/zatrano/framework/core/version"
	"github.com/zatrano/framework/core/webauthn"
	"github.com/zatrano/framework/core/webhooks"
	"github.com/zatrano/framework/core/wellknown"
)

// Cache returns the cache manager.
func (app *Application) Cache() *cache.Manager {
	return app.cache
}

// Events returns the event dispatcher.
func (app *Application) Events() *events.Dispatcher {
	return app.events
}

// Mail returns the mail manager.
func (app *Application) Mail() *mail.Manager {
	return app.mail
}

// Queue returns the queue manager.
func (app *Application) Queue() *queue.Manager {
	return app.queue
}

// Auth returns the auth manager.
func (app *Application) Auth() *auth.Manager {
	return app.auth
}

// Translator returns the localization translator.
func (app *Application) Translator() *localization.Translator {
	return app.translator
}

// Scheduler returns the task scheduler.
func (app *Application) Scheduler() *schedule.Scheduler {
	return app.scheduler
}

// HTTP returns the HTTP client.
func (app *Application) HTTP() *httpclient.Client {
	return app.httpClient
}

// Notifications returns the notification manager.
func (app *Application) Notifications() *notification.Manager {
	return app.notifications
}

// PushSender returns the in-memory push notification sender stub.
func (app *Application) PushSender() *notification.MemoryPushSender {
	return app.pushSender
}

// SmsSender returns the in-memory SMS sender stub.
func (app *Application) SmsSender() *notification.MemorySmsSender {
	return app.smsSender
}

// Broadcaster returns the broadcasting manager.
func (app *Application) Broadcaster() *broadcasting.Manager {
	return app.broadcast
}

// Storage returns the filesystem manager.
func (app *Application) Storage() *filesystem.Manager {
	return app.files
}

// RateLimiter returns the rate limiter.
func (app *Application) RateLimiter() *ratelimit.Limiter {
	return app.rateLimiter
}

// Gate returns the authorization gate.
func (app *Application) Gate() *authorization.Gate {
	return app.gate
}

// Context returns the application context store.
func (app *Application) Context() *appcontext.Store {
	return app.ctx
}

// URL returns the URL generator.
func (app *Application) URL() *urlgen.Generator {
	return app.urls
}

func (app *Application) bootSupportServices() error {
	app.exceptions = exceptions.New(app.IsDebug() || app.config.GetBool("app.debug", true))
	app.reports = report.New(200)
	app.exceptions.ReportUsing(app.reports.Reporter())
	app.exceptions.ReportUsing(func(err error, req *http.Request) {
		path := ""
		if req != nil {
			path = req.Method() + " " + req.Path()
		}
		if app.logger != nil {
			app.logger.Errorf("exception on %s: %v", path, err)
		}
	})
	app.container.Instance("exceptions", app.exceptions)
	app.container.Instance("report", app.reports)

	fileStore, err := cache.NewFileStore(app.BasePath("storage", "framework", "cache"))
	if err != nil {
		return err
	}
	stores := map[string]cache.Store{
		"file":   fileStore,
		"memory": cache.NewMemoryStore(),
	}

	redisClient, redisErr := redisx.Connect(redisx.Config{
		Host:     env.Get("REDIS_HOST", "127.0.0.1"),
		Port:     env.Get("REDIS_PORT", "6379"),
		Password: env.Get("REDIS_PASSWORD"),
		DB:       redisx.ParseDB(env.Get("REDIS_DB", "0")),
	})
	if redisErr == nil {
		stores["redis"] = cache.NewRedisStore(redisClient, "zatrano_cache:")
	} else if app.logger != nil {
		app.logger.Debugf("redis unavailable, skipping redis cache/queue: %v", redisErr)
	}

	app.cache = cache.NewManager(env.Get("CACHE_STORE", "file"), stores)
	app.container.Instance("cache", app.cache)

	app.events = events.New()
	app.container.Instance("events", app.events)
	orm.SetDispatcher(app.events)

	logMailer := mail.NewLogMailer(app.logger)
	smtpMailer := mail.NewSMTPMailer(mail.SMTPConfig{
		Host:       env.Get("MAIL_HOST", "127.0.0.1"),
		Port:       env.Get("MAIL_PORT", "2525"),
		Username:   env.Get("MAIL_USERNAME"),
		Password:   env.Get("MAIL_PASSWORD"),
		Encryption: env.Get("MAIL_ENCRYPTION"),
	})
	app.mail = mail.NewManager(
		env.Get("MAIL_MAILER", "log"),
		env.Get("MAIL_FROM_ADDRESS", "hello@example.com"),
		env.Get("MAIL_FROM_NAME", app.config.GetString("app.name", "ZATRANO")),
		map[string]mail.Mailer{
			"log":  logMailer,
			"smtp": smtpMailer,
		},
	)
	app.container.Instance("mail", app.mail)

	syncQueue := queue.NewSyncQueue()
	queues := map[string]queue.Queue{
		"sync": syncQueue,
	}

	connection := env.Get("QUEUE_CONNECTION", "sync")
	if app.db != nil {
		db, err := app.db.DB()
		if err == nil {
			driver, _ := app.db.DriverName()
			dbQueue := queue.NewDatabaseQueue(db, "jobs", driver)
			_ = dbQueue.EnsureTable()
			queues["database"] = dbQueue
		}
	}
	if redisClient != nil {
		queues["redis"] = queue.NewRedisQueue(redisClient, "zatrano:queues:default")
	}
	app.queue = queue.NewManager(connection, queues)
	app.container.Instance("queue", app.queue)

	authManager := auth.NewManager(app.config.GetString("auth.defaults.guard", "web"))
	authManager.SetSessionManager(app.session)
	authManager.SetDispatcher(app.Events())
	if app.db != nil {
		db, err := app.db.DB()
		if err == nil {
			driver, _ := app.db.DriverName()
			providers := map[string]auth.UserProvider{}
			if rawProviders, ok := app.config.Get("auth.providers").(map[string]any); ok {
				for name, raw := range rawProviders {
					pcfg, _ := raw.(map[string]any)
					table := "users"
					if pcfg != nil {
						if t := strings.TrimSpace(fmt.Sprint(pcfg["table"])); t != "" && t != "<nil>" {
							table = t
						}
					}
					providers[name] = auth.NewDatabaseUserProvider(db, driver, table)
				}
			}
			if len(providers) == 0 {
				providers["users"] = auth.NewDatabaseUserProvider(db, driver, "users")
			}
			defaultProvider := app.config.GetString("auth.defaults.provider", "users")
			if rawGuards, ok := app.config.Get("auth.guards").(map[string]any); ok {
				for name, raw := range rawGuards {
					gcfg, _ := raw.(map[string]any)
					providerName := defaultProvider
					if gcfg != nil {
						if p := strings.TrimSpace(fmt.Sprint(gcfg["provider"])); p != "" && p != "<nil>" {
							providerName = p
						}
					}
					provider := providers[providerName]
					if provider == nil {
						provider = providers["users"]
					}
					if provider == nil {
						continue
					}
					authManager.Extend(name, auth.NewGuard(name, provider))
				}
			}
			if authManager.Guard() == nil {
				provider := providers[defaultProvider]
				if provider == nil {
					provider = providers["users"]
				}
				if provider != nil {
					authManager.Extend(authManager.GetDefaultDriver(), auth.NewGuard(authManager.GetDefaultDriver(), provider))
				}
			}
		}
	}
	app.auth = authManager
	app.container.Instance("auth", app.auth)

	app.maintenance = maintenance.New(app.BasePath("storage", "framework"))
	app.container.Instance("maintenance", app.maintenance)

	if app.db != nil {
		db, err := app.db.DB()
		if err == nil {
			driver, _ := app.db.DriverName()
			provider := auth.NewDatabaseUserProvider(db, driver, "users")
			tokens := auth.NewDatabaseTokenRepository(db, driver, time.Hour)
			app.passwords = auth.NewPasswordBroker(tokens, provider, time.Hour)
			app.passwords.SetDispatcher(app.Events())
			app.passwords.SetMailer(func(email, token, resetURL string) error {
				body := fmt.Sprintf("Reset your password using token %s\n\n%s?email=%s&token=%s", token, resetURL, email, token)
				return app.mail.To(email, "Reset Password", body)
			})
			app.container.Instance("passwords", app.passwords)

			app.tokens = apitoken.New(apitoken.NewDatabaseStore(db, driver), provider)
			app.container.Instance("tokens", app.tokens)
		}
	}
	if app.tokens == nil {
		app.tokens = apitoken.New(apitoken.NewMemoryStore(), nil)
		app.container.Instance("tokens", app.tokens)
	}

	locale := app.config.GetString("app.locale", env.Get("APP_LOCALE", "en"))
	fallback := app.config.GetString("app.fallback", env.Get("APP_FALLBACK_LOCALE", "en"))
	app.translator = localization.New(app.BasePath("lang"), locale, fallback)
	_ = app.translator.Load(locale)
	if fallback != locale {
		_ = app.translator.Load(fallback)
	}
	app.container.Instance("translator", app.translator)

	app.scheduler = schedule.New()
	app.container.Instance("scheduler", app.scheduler)

	app.httpClient = httpclient.New()
	app.container.Instance("http", app.httpClient)

	fileBroadcast, err := broadcasting.NewFileBroadcaster(app.BasePath("storage", "logs", "broadcast.jsonl"))
	if err != nil {
		return err
	}
	app.broadcast = broadcasting.NewManager(env.Get("BROADCAST_CONNECTION", "log"), map[string]broadcasting.Broadcaster{
		"log":  broadcasting.NewLogBroadcaster(app.logger),
		"file": fileBroadcast,
		"null": broadcasting.NullBroadcaster{},
	})
	app.broadcast.Channel("public", func(req *http.Request, channel string) bool {
		return true
	})
	app.broadcast.Channel("private.*", func(req *http.Request, channel string) bool {
		return app.auth != nil && app.auth.Check(req)
	})
	app.container.Instance("broadcast", app.broadcast)

	app.notifications = notification.NewManager()
	app.notifications.Extend("mail", notification.NewMailChannel(app.mail))
	app.notifications.Extend("broadcast", notification.NewBroadcastChannel(app.broadcast))
	app.pushSender = &notification.MemoryPushSender{}
	app.notifications.Extend("push", notification.NewPushChannel(app.pushSender))
	app.smsSender = &notification.MemorySmsSender{}
	smsFrom := env.Get("SMS_FROM", env.Get("APP_NAME", "ZATRANO"))
	app.notifications.Extend("sms", notification.NewSmsChannel(app.smsSender, smsFrom))
	if app.db != nil {
		if db, err := app.db.DB(); err == nil {
			driver, _ := app.db.DriverName()
			app.notifications.Extend("database", notification.NewDatabaseChannel(db, "notifications", driver))
			store := notification.NewStore(db, "notifications", driver)
			app.notifications.SetStore(store)
		}
	}
	app.container.Instance("notifications", app.notifications)

	localDisk, err := filesystem.NewLocalDisk(app.BasePath("storage", "app"))
	if err != nil {
		return err
	}
	publicDisk, err := filesystem.NewLocalDisk(app.BasePath("storage", "app", "public"))
	if err != nil {
		return err
	}
	appURL := app.config.GetString("app.url", env.Get("APP_URL", "http://localhost:8080"))
	publicDisk.SetBaseURL(strings.TrimRight(appURL, "/") + "/storage")
	signingKey := app.config.GetString("app.key", env.Get("APP_KEY", "zatrano-dev-key"))
	localDisk.SetSigningKey(signingKey)
	localDisk.SetServePath("/storage/temporary")
	localDisk.SetBaseURL(strings.TrimRight(appURL, "/"))
	publicDisk.SetSigningKey(signingKey)
	publicDisk.SetServePath("/storage/temporary")

	s3Disk := filesystem.NewCloudDisk(
		env.Get("AWS_BUCKET", "zatrano"),
		env.Get("AWS_URL", "https://s3.example.com"),
	)

	app.files = filesystem.NewManager(env.Get("FILESYSTEM_DISK", "local"), map[string]filesystem.Disk{
		"local":  localDisk,
		"public": publicDisk,
		"s3":     s3Disk,
	})
	app.container.Instance("files", app.files)

	app.metrics = observability.New()
	app.container.Instance("metrics", app.metrics)

	app.health = health.New()
	if app.db != nil {
		if db, err := app.db.DB(); err == nil {
			app.health.Database(db)
		}
	}
	app.health.Disk(app.BasePath("storage"))
	app.health.Custom("cache", func(ctx context.Context) error {
		if app.cache == nil {
			return fmt.Errorf("cache unavailable")
		}
		return app.cache.Put("health:ping", "ok", 0)
	})
	app.container.Instance("health", app.health)

	app.rateLimiter = ratelimit.New()
	app.rateLimiter.For("api", ratelimit.Limit{MaxAttempts: 60, Decay: time.Minute})
	app.rateLimiter.For("login", ratelimit.Limit{MaxAttempts: 5, Decay: time.Minute})
	app.container.Instance("rateLimiter", app.rateLimiter)

	app.gate = authorization.New()
	app.container.Instance("gate", app.gate)

	app.ctx = appcontext.New()
	app.container.Instance("context", app.ctx)

	app.urls = urlgen.New(app.router, app.config.GetString("app.url", env.Get("APP_URL", "http://localhost:8080")))
	app.urls.SetSigningKey(app.config.GetString("app.key", env.Get("APP_KEY")))
	app.container.Instance("url", app.urls)

	encrypter, err := encryption.New(app.config.GetString("app.key", env.Get("APP_KEY", "zatrano-dev-key")))
	if err != nil {
		return err
	}
	app.encrypter = encrypter
	app.container.Instance("encrypter", app.encrypter)
	orm.SetCastEncrypter(encrypter)
	if app.auth != nil {
		app.auth.SetEncrypter(app.encrypter)
	}

	app.hasher = hashing.New()
	app.container.Instance("hash", app.hasher)

	app.features = features.New()
	app.features.Activate("welcome_banner")
	app.features.Deactivate("beta_editor")
	app.features.Rollout("new_dashboard", 25)
	app.container.Instance("features", app.features)

	app.tenancy = tenancy.New()
	app.tenancy.Register(tenancy.Tenant{ID: "acme", Name: "Acme Corp", Domain: "acme.localhost"})
	app.tenancy.Register(tenancy.Tenant{ID: "globex", Name: "Globex", Domain: "globex.localhost"})
	app.tenancy.SetResolver(app.tenancy.HeaderOrHostResolver())
	app.tenancy.Bootstrapping(func(t *tenancy.Tenant) error {
		if app.ctx != nil {
			app.ctx.Put("tenant.id", t.ID)
			app.ctx.Put("tenant.name", t.Name)
		}
		return nil
	})
	app.container.Instance("tenancy", app.tenancy)

	app.graphql = graphql.NewSchema()
	app.graphql.Query("health", func(args map[string]any) (any, error) {
		return "ok", nil
	})
	app.graphql.Query("echo", func(args map[string]any) (any, error) {
		msg, _ := args["message"].(string)
		if msg == "" {
			msg = "hello"
		}
		return msg, nil
	})
	app.graphql.Query("feature", func(args map[string]any) (any, error) {
		name, _ := args["name"].(string)
		return app.features.Active(name), nil
	})
	app.graphql.Mutation("ping", func(args map[string]any) (any, error) {
		return map[string]any{"pong": true}, nil
	})
	app.container.Instance("graphql", app.graphql)

	memoryAudit := audit.NewMemoryStore(500)
	fileAudit, err := audit.NewFileStore(app.BasePath("storage", "logs", "audit.jsonl"))
	if err != nil {
		return err
	}
	app.audit = audit.New(&teeAuditStore{primary: memoryAudit, secondary: fileAudit})
	app.container.Instance("audit", app.audit)

	app.webhooks = webhooks.New()
	app.webhooks.Register(webhooks.Endpoint{
		URL:    env.Get("WEBHOOK_URL", "https://httpbin.org/post"),
		Secret: env.Get("WEBHOOK_SECRET", "zatrano-webhook-secret"),
		Events: []string{"user.created", "demo.ping", "*"},
	})
	app.container.Instance("webhooks", app.webhooks)

	_ = version.LoadFile(app.BasePath("VERSION"))

	app.inspector = inspector.New(200)
	app.container.Instance("inspector", app.inspector)

	app.search = search.New(search.NewMemoryEngine())
	app.container.Instance("search", app.search)

	publicURL := strings.TrimRight(app.config.GetString("app.url", "http://localhost:8080"), "/")
	app.assets = assets.LoadDefault(app.BasePath(), publicURL)
	app.container.Instance("assets", app.assets)

	app.social = social.New()
	redirectBase := strings.TrimRight(app.config.GetString("app.url", "http://localhost:8080"), "/")
	app.social.Extend("github", social.GitHub(social.Config{
		ClientID:     env.Get("GITHUB_CLIENT_ID", "github-client-id"),
		ClientSecret: env.Get("GITHUB_CLIENT_SECRET", "github-client-secret"),
		RedirectURL:  redirectBase + "/auth/github/callback",
	}))
	app.social.Extend("google", social.Google(social.Config{
		ClientID:     env.Get("GOOGLE_CLIENT_ID", "google-client-id"),
		ClientSecret: env.Get("GOOGLE_CLIENT_SECRET", "google-client-secret"),
		RedirectURL:  env.Get("GOOGLE_REDIRECT_URI", redirectBase+"/auth/google/callback"),
	}))
	app.container.Instance("social", app.social)

	app.enums = enums.NewRegistry()
	app.enums.Register(enums.NewString("post_status", "draft:Draft", "published:Published", "archived:Archived"))
	app.enums.Register(enums.NewString("user_role", "admin", "editor", "viewer"))
	app.container.Instance("enums", app.enums)

	app.bus = bus.New()
	app.container.Instance("bus", app.bus)

	app.pulse = pulse.New(app.metrics).WithExtra(func() map[string]any {
		extra := map[string]any{}
		if app.inspector != nil {
			extra["inspector_entries"] = app.inspector.Count()
		}
		if app.search != nil {
			extra["search_docs"] = app.search.Count()
		}
		return extra
	})
	app.container.Instance("pulse", app.pulse)

	dbPath := app.config.GetString("database.connections.sqlite.database", "database/database.sqlite")
	if !filepath.IsAbs(dbPath) {
		dbPath = app.BasePath(dbPath)
	}
	app.backups = backup.New(dbPath, app.BasePath("storage", "backups"))
	app.container.Instance("backup", app.backups)

	app.docs = docs.New(app.BasePath("docs"))
	app.container.Instance("docs", app.docs)

	baseURL := app.config.GetString("app.url", env.Get("APP_URL", "http://localhost:8080"))
	app.billing = billing.New(baseURL)
	app.container.Instance("billing", app.billing)

	app.mongo = mongo.Connect(env.Get("MONGO_URI", "memory"))
	app.container.Instance("mongo", app.mongo)

	app.oauth = oauth.New()
	app.container.Instance("oauth", app.oauth)

	workers := env.GetInt("OCTANE_WORKERS", 0)
	app.octane = octane.New(workers)
	app.container.Instance("octane", app.octane)

	app.ai = ai.New()
	if driver := env.Get("AI_DRIVER", "fake"); driver != "" {
		app.ai.Use(driver)
	}
	app.container.Instance("ai", app.ai)

	base := strings.TrimRight(app.config.GetString("app.url", env.Get("APP_URL", "http://localhost:8080")), "/")
	app.sitemap = sitemap.New(base)
	app.sitemap.Add("/", sitemap.URL{Priority: 1.0, ChangeFreq: "daily"})
	app.sitemap.Add("/up", sitemap.URL{Priority: 0.1, ChangeFreq: "monthly"})
	app.container.Instance("sitemap", app.sitemap)

	app.locks = lock.New()
	app.container.Instance("lock", app.locks)

	app.circuits = circuit.New(circuit.Settings{
		FailureThreshold: env.GetInt("CIRCUIT_FAILURE_THRESHOLD", 5),
		SuccessThreshold: env.GetInt("CIRCUIT_SUCCESS_THRESHOLD", 2),
		Timeout:          time.Duration(env.GetInt("CIRCUIT_TIMEOUT_SECONDS", 30)) * time.Second,
	})
	app.container.Instance("circuit", app.circuits)

	app.hashids = hashid.New(env.Get("HASHID_SALT", app.config.GetString("app.key", "zatrano")), env.GetInt("HASHID_MIN_LENGTH", 8))
	app.container.Instance("hashid", app.hashids)

	app.shorturls = shorturl.New(base, env.Get("SHORTURL_PREFIX", "/s"))
	app.container.Instance("shorturl", app.shorturls)

	app.wellknown = wellknown.New(wellknown.Config{
		ContactEmail:  env.Get("SECURITY_CONTACT_EMAIL", "security@zatrano.test"),
		ContactURL:    env.Get("SECURITY_CONTACT_URL", base+"/contact"),
		Canonical:     base + "/.well-known/security.txt",
		PolicyURL:     env.Get("SECURITY_POLICY_URL", base+"/documentation"),
		PreferredLang: env.Get("APP_LOCALE", "en"),
	})
	app.container.Instance("wellknown", app.wellknown)

	app.geo = geo.New()
	app.container.Instance("geo", app.geo)

	rpID := env.Get("WEBAUTHN_RP_ID", "localhost")
	rpName := env.Get("WEBAUTHN_RP_NAME", env.Get("APP_NAME", "ZATRANO"))
	app.webauthn = webauthn.New(rpID, rpName)
	app.container.Instance("webauthn", app.webauthn)

	app.otp = otp.New(otp.NewMemoryStore()).WithTTL(5 * time.Minute)
	app.container.Instance("otp", app.otp)

	return nil
}

type teeAuditStore struct {
	primary   audit.Store
	secondary audit.Store
}

func (s *teeAuditStore) Write(event audit.Event) error {
	if err := s.primary.Write(event); err != nil {
		return err
	}
	return s.secondary.Write(event)
}

func (s *teeAuditStore) Recent(limit int) ([]audit.Event, error) {
	return s.primary.Recent(limit)
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

// Features returns the feature flag manager.
func (app *Application) Features() *features.Manager {
	return app.features
}

// Tenancy returns the tenancy manager.
func (app *Application) Tenancy() *tenancy.Manager {
	return app.tenancy
}

// GraphQL returns the GraphQL schema.
func (app *Application) GraphQL() *graphql.Schema {
	return app.graphql
}

// Audit returns the audit log manager.
func (app *Application) Audit() *audit.Manager {
	return app.audit
}

// Webhooks returns the outbound webhook manager.
func (app *Application) Webhooks() *webhooks.Manager {
	return app.webhooks
}

// Version returns the application/framework version.
func (app *Application) Version() string {
	return version.Get()
}

// Passwords returns the password reset broker.
func (app *Application) Passwords() *auth.PasswordBroker {
	return app.passwords
}

// Maintenance returns the maintenance mode manager.
func (app *Application) Maintenance() *maintenance.Manager {
	return app.maintenance
}

// Tokens returns the personal access token manager.
func (app *Application) Tokens() *apitoken.Manager {
	return app.tokens
}

// Search returns the search manager.
func (app *Application) Search() *search.Manager {
	return app.search
}

// Inspector returns the request inspector.
func (app *Application) Inspector() *inspector.Manager {
	return app.inspector
}

// Assets returns the asset manifest helper.
func (app *Application) Assets() *assets.Manifest {
	return app.assets
}

// Exceptions returns the exception handler.
func (app *Application) Exceptions() *exceptions.Handler {
	return app.exceptions
}

// Social returns the social OAuth manager.
func (app *Application) Social() *social.Manager {
	return app.social
}

// Enums returns the enum registry.
func (app *Application) Enums() *enums.Registry {
	return app.enums
}

// Bus returns the command bus.
func (app *Application) Bus() *bus.Bus {
	return app.bus
}

// Pulse returns the metrics pulse dashboard.
func (app *Application) Pulse() *pulse.Dashboard {
	return app.pulse
}

// Backup returns the database backup manager.
func (app *Application) Backup() *backup.Manager {
	return app.backups
}

// Docs returns the documentation repository.
func (app *Application) Docs() *docs.Repository {
	return app.docs
}

// Billing returns the billing manager.
func (app *Application) Billing() *billing.Manager {
	return app.billing
}

// Mongo returns the MongoDB stub client.
func (app *Application) Mongo() *mongo.Client {
	return app.mongo
}

// OAuth returns the OAuth2 authorization server.
func (app *Application) OAuth() *oauth.Server {
	return app.oauth
}

// Octane returns the concurrent runtime tracker.
func (app *Application) Octane() *octane.Runtime {
	return app.octane
}

// AI returns the AI manager.
func (app *Application) AI() *ai.Manager {
	return app.ai
}

// Sitemap returns the sitemap builder.
func (app *Application) Sitemap() *sitemap.Builder {
	return app.sitemap
}

// Lock returns the lock manager.
func (app *Application) Lock() *lock.Manager {
	return app.locks
}

// Circuit returns the circuit breaker manager.
func (app *Application) Circuit() *circuit.Manager {
	return app.circuits
}

// HashID returns the hashid hasher.
func (app *Application) HashID() *hashid.Hasher {
	return app.hashids
}

// ShortURL returns the short URL manager.
func (app *Application) ShortURL() *shorturl.Manager {
	return app.shorturls
}

// WellKnown returns the well-known / security.txt repository.
func (app *Application) WellKnown() *wellknown.Repository {
	return app.wellknown
}

// Geo returns the geolocation resolver.
func (app *Application) Geo() *geo.Resolver {
	return app.geo
}

// Reports returns the exception report manager.
func (app *Application) Reports() *report.Manager {
	return app.reports
}

// WebAuthn returns the WebAuthn stub manager.
func (app *Application) WebAuthn() *webauthn.Manager {
	return app.webauthn
}

// OTP returns the one-time password manager.
func (app *Application) OTP() *otp.Manager {
	return app.otp
}

// Transaction runs a database transaction on the default connection.
func (app *Application) Transaction(fn func(tx *sql.Tx) error) error {
	if app.db == nil {
		return fmt.Errorf("database not configured")
	}
	return app.db.Transaction(fn)
}

// DispatchEvent fires an application event.
func (app *Application) DispatchEvent(name string, event any) error {
	if app.events == nil {
		return fmt.Errorf("events not configured")
	}
	return app.events.Dispatch(name, event)
}

// Trans is a helper for translations.
func (app *Application) Trans(key string, replace ...map[string]string) string {
	if app.translator == nil {
		return key
	}
	return app.translator.Get(key, replace...)
}
