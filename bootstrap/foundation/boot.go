package foundation

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	appconfig "github.com/zatrano/framework/config"
	"github.com/zatrano/framework/core"
	"github.com/zatrano/framework/packages/apitoken"
	"github.com/zatrano/framework/packages/assets"
	"github.com/zatrano/framework/packages/auth"
	"github.com/zatrano/framework/packages/authorization"
	"github.com/zatrano/framework/packages/broadcasting"
	"github.com/zatrano/framework/packages/cache"
	pkgconfig "github.com/zatrano/framework/packages/config"
	"github.com/zatrano/framework/packages/database"
	"github.com/zatrano/framework/packages/database/query"
	"github.com/zatrano/framework/packages/env"
	"github.com/zatrano/framework/packages/events"
	"github.com/zatrano/framework/packages/filesystem"
	"github.com/zatrano/framework/packages/http"
	"github.com/zatrano/framework/packages/httpclient"
	"github.com/zatrano/framework/packages/localization"
	mongopkg "github.com/zatrano/framework/packages/mongo"
	"github.com/zatrano/framework/packages/notification"
	"github.com/zatrano/framework/packages/orm"
	"github.com/zatrano/framework/packages/queue"
	"github.com/zatrano/framework/packages/redisx"
	"github.com/zatrano/framework/packages/schedule"
	"github.com/zatrano/framework/packages/session"
	"github.com/zatrano/framework/packages/validation"
	"github.com/zatrano/framework/packages/view"
)

func BootDatabase(app *core.Application) error {
	pkgconfig.LoadIfAbsent(app.Config(), "database", appconfig.Database())
	defaultConn := app.Config().GetString("database.default", "sqlite")
	connections := map[string]database.ConnectionConfig{}

	rawConnections, ok := app.Config().Get("database.connections").(map[string]any)
	if !ok {
		rawConnections = map[string]any{}
	}

	for name, raw := range rawConnections {
		cfgMap, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		cfg := database.ConnectionConfig{
			Driver:   asString(cfgMap["driver"]),
			Host:     asString(cfgMap["host"]),
			Port:     asString(cfgMap["port"]),
			Database: asString(cfgMap["database"]),
			Username: asString(cfgMap["username"]),
			Password: asString(cfgMap["password"]),
			Charset:  asString(cfgMap["charset"]),
			SSLMode:  asString(cfgMap["sslmode"]),
			Service:  asString(cfgMap["service"]),
			URI:      asString(cfgMap["uri"]),
		}
		if database.IsDocumentStore(cfg.Driver) || database.IsDocumentStore(name) {
			if err := bootMongoConnection(app, name, cfg); err != nil {
				return err
			}
			continue
		}
		connections[name] = cfg
	}

	if len(connections) == 0 {
		if database.IsDocumentStore(defaultConn) {
			return nil
		}
		return fmt.Errorf("no SQL database connections configured")
	}

	sqlDefault := defaultConn
	if _, ok := connections[sqlDefault]; !ok {
		for name := range connections {
			sqlDefault = name
			break
		}
	}

	mgr := database.NewManager(database.Config{
		Default:     sqlDefault,
		Connections: connections,
	}, app.BasePath())
	app.Container().Instance("db", mgr)

	db, err := mgr.DB()
	if err != nil {
		return err
	}
	driver, err := mgr.DriverName()
	if err != nil {
		return err
	}
	orm.Configure(db, driver)
	orm.SetConnectionResolver(func(name string) (*sql.DB, string, error) {
		conn, err := mgr.Connection(name)
		if err != nil {
			return nil, "", err
		}
		d, err := mgr.DriverName(name)
		if err != nil {
			return nil, "", err
		}
		return conn, d, nil
	})
	if enc := app.Encrypter(); enc != nil {
		orm.SetCastEncrypter(enc)
	}
	if ev := events.From(app); ev != nil {
		orm.SetDispatcher(ev)
	}
	validation.SetDefaultPresenceChecker(func(table, column, value string) (bool, error) {
		table = strings.TrimSpace(table)
		column = strings.TrimSpace(column)
		if table == "" || column == "" {
			return false, nil
		}
		row, err := query.New(db, driver, table).Where(column, value).First()
		if err != nil {
			if err == sql.ErrNoRows {
				return false, nil
			}
			return false, err
		}
		return row != nil, nil
	})
	return nil
}

func bootMongoConnection(app *core.Application, name string, cfg database.ConnectionConfig) error {
	uri := strings.TrimSpace(cfg.URI)
	if uri == "" {
		uri = strings.TrimSpace(cfg.Database)
	}
	if uri == "" {
		uri = env.Get("MONGO_URI", "memory")
	}
	client := mongopkg.Connect(uri)
	if err := client.Ping(); err != nil {
		return fmt.Errorf("mongo connection [%s]: %w", name, err)
	}
	app.Container().Instance("mongo", client)
	if name != "" && name != "mongo" {
		app.Container().Instance("mongo."+name, client)
	}
	return nil
}

func BootCacheServices(app *core.Application) error {
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
		app.Container().Instance("redis", redisClient)
	} else if app.Logger() != nil {
		app.Logger().Debugf("redis unavailable, skipping redis cache/queue: %v", redisErr)
	}
	mgr := cache.NewManager(env.Get("CACHE_STORE", "file"), stores)
	app.Container().Instance("cache", mgr)
	if app.Health() != nil {
		app.Health().Custom("cache", func(ctx context.Context) error {
			if cache.From(app) == nil {
				return fmt.Errorf("cache unavailable")
			}
			return cache.From(app).Put("health:ping", "ok", 0)
		})
	}
	return nil
}

func BootEventsServices(app *core.Application) error {
	dispatcher := events.New()
	app.Container().Instance("events", dispatcher)
	orm.SetDispatcher(dispatcher)
	return nil
}

func BootQueueServices(app *core.Application) error {
	queues := map[string]queue.Queue{"sync": queue.NewSyncQueue()}
	if dbMgr := database.From(app); dbMgr != nil {
		if db, err := dbMgr.DB(); err == nil {
			driver, _ := dbMgr.DriverName()
			dbQueue := queue.NewDatabaseQueue(db, "jobs", driver)
			_ = dbQueue.EnsureTable()
			queues["database"] = dbQueue
		}
	}
	if raw, err := app.Make("redis"); err == nil {
		if client := redisx.ClientFrom(raw); client != nil {
			queues["redis"] = queue.NewRedisQueue(client, "zatrano:queues:default")
		}
	}
	mgr := queue.NewManager(env.Get("QUEUE_CONNECTION", "sync"), queues)
	app.Container().Instance("queue", mgr)
	return nil
}

func BootAuthServices(app *core.Application) error {
	pkgconfig.LoadIfAbsent(app.Config(), "auth", appconfig.Auth())
	app.Container().Instance("gate", authorization.New())
	authManager := auth.NewManager(app.Config().GetString("auth.defaults.guard", "web"))
	authManager.SetSessionManager(session.From(app))
	authManager.SetDispatcher(events.From(app))
	if max := app.Config().GetInt("auth.lockout.max_attempts", 5); max > 0 {
		decayMin := app.Config().GetInt("auth.lockout.decay_minutes", 1)
		if decayMin <= 0 {
			decayMin = 1
		}
		authManager.SetLockout(max, time.Duration(decayMin)*time.Minute)
	}
	if issuer := strings.TrimSpace(app.Config().GetString("auth.two_factor.issuer", "")); issuer != "" {
		authManager.SetTwoFactorIssuer(issuer)
	}
	authManager.SetRememberDeviceDays(app.Config().GetInt("auth.two_factor.remember_device_days", 30))
	if c := cache.From(app); c != nil {
		authManager.SetLockoutCache(c)
	}
	if dbMgr := database.From(app); dbMgr != nil {
		db, err := dbMgr.DB()
		if err == nil {
			driver, _ := dbMgr.DriverName()
			providers := map[string]auth.UserProvider{}
			if rawProviders, ok := app.Config().Get("auth.providers").(map[string]any); ok {
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
			defaultProvider := app.Config().GetString("auth.defaults.provider", "users")
			if rawGuards, ok := app.Config().Get("auth.guards").(map[string]any); ok {
				for name, raw := range rawGuards {
					gcfg, _ := raw.(map[string]any)
					guardDriver := "session"
					providerName := defaultProvider
					if gcfg != nil {
						if d := strings.TrimSpace(fmt.Sprint(gcfg["driver"])); d != "" && d != "<nil>" {
							guardDriver = strings.ToLower(d)
						}
						if p := strings.TrimSpace(fmt.Sprint(gcfg["provider"])); p != "" && p != "<nil>" {
							providerName = p
						}
					}
					// Session guards only; token/PAT auth uses packages/apitoken middleware.
					if guardDriver != "session" {
						continue
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
	app.Container().Instance("auth", authManager)
	if enc := app.Encrypter(); enc != nil {
		authManager.SetEncrypter(enc)
	}

	authManager.SetVerificationURLGenerator(func(user auth.Authenticatable) (string, error) {
		if user == nil || app.URL() == nil {
			return "", fmt.Errorf("verification url unavailable")
		}
		email := auth.EmailForVerification(user)
		return app.URL().Signed("/auth/email/verify/"+fmt.Sprint(user.AuthID()), 60*time.Minute, map[string]string{
			"hash": auth.EmailHash(email),
		})
	})
	authManager.SetEmailVerificationSender(func(user auth.Authenticatable, verifyURL string) error {
		n := notification.From(app)
		if n == nil {
			return fmt.Errorf("notifications not configured")
		}
		return n.Send(notification.Recipient{
			ID:    fmt.Sprint(user.AuthID()),
			Email: auth.EmailForVerification(user),
		}, notification.VerifyEmailNotification{VerifyURL: verifyURL})
	})
	authManager.SetPasswordChangedSender(func(user auth.Authenticatable) error {
		n := notification.From(app)
		if n == nil {
			return nil
		}
		return n.Send(notification.Recipient{
			ID:    fmt.Sprint(user.AuthID()),
			Email: auth.EmailForVerification(user),
		}, notification.PasswordChangedNotification{})
	})

	if dbMgr := database.From(app); dbMgr != nil {
		if db, err := dbMgr.DB(); err == nil {
			driver, _ := dbMgr.DriverName()
			brokerName := app.Config().GetString("auth.defaults.passwords", "users")
			passCfg, _ := app.Config().Get("auth.passwords." + brokerName).(map[string]any)
			table := "password_reset_tokens"
			expireMin := 60
			throttleSec := 60
			providerName := app.Config().GetString("auth.defaults.provider", "users")
			if passCfg != nil {
				if t := strings.TrimSpace(fmt.Sprint(passCfg["table"])); t != "" && t != "<nil>" {
					table = t
				}
				if p := strings.TrimSpace(fmt.Sprint(passCfg["provider"])); p != "" && p != "<nil>" {
					providerName = p
				}
				if v, ok := asInt(passCfg["expire"]); ok && v > 0 {
					expireMin = v
				}
				if v, ok := asInt(passCfg["throttle"]); ok && v >= 0 {
					throttleSec = v
				}
			}
			provTable := "users"
			if rawProviders, ok := app.Config().Get("auth.providers").(map[string]any); ok {
				if raw, ok := rawProviders[providerName].(map[string]any); ok {
					if t := strings.TrimSpace(fmt.Sprint(raw["table"])); t != "" && t != "<nil>" {
						provTable = t
					}
				}
			}
			provider := auth.NewDatabaseUserProvider(db, driver, provTable)
			tokens := auth.NewDatabaseTokenRepositoryTable(db, driver, table, time.Duration(expireMin)*time.Minute)
			passwords := auth.NewPasswordBroker(tokens, provider, time.Duration(expireMin)*time.Minute)
			passwords.SetThrottle(time.Duration(throttleSec) * time.Second)
			passwords.SetDispatcher(events.From(app))
			passwords.SetSessionManager(session.From(app))
			passwords.SetNotifier(func(email, token, resetURL string) error {
				n := notification.From(app)
				if n == nil {
					return fmt.Errorf("notifications not configured")
				}
				return n.Send(notification.Recipient{ID: email, Email: email}, notification.PasswordResetNotification{
					Token:         token,
					ResetURL:      resetURL,
					ExpireMinutes: expireMin,
				})
			})
			app.Container().Instance("passwords", passwords)
			app.Container().Instance("tokens", apitoken.New(apitoken.NewDatabaseStore(db, driver), provider))
		}
	}
	if apitoken.From(app) == nil {
		app.Container().Instance("tokens", apitoken.New(apitoken.NewMemoryStore(), nil))
	}
	if app.Health() != nil && database.From(app) != nil {
		if db, err := database.From(app).DB(); err == nil {
			app.Health().Database(db)
		}
	}
	return nil
}

func BootLocalizationServices(app *core.Application) error {
	locale := app.Config().GetString("app.locale", env.Get("APP_LOCALE", "en"))
	fallback := app.Config().GetString("app.fallback", env.Get("APP_FALLBACK_LOCALE", "en"))
	translator := localization.New(app.BasePath("lang"), locale, fallback)
	_ = translator.Load(locale)
	if fallback != locale {
		_ = translator.Load(fallback)
	}
	app.Container().Instance("translator", translator)
	return nil
}

func BootScheduleServices(app *core.Application) error {
	app.Container().Instance("scheduler", schedule.New())
	return nil
}

func BootHTTPClientServices(app *core.Application) error {
	app.Container().Instance("http", httpclient.New())
	return nil
}

func BootBroadcastingServices(app *core.Application) error {
	fileBroadcast, err := broadcasting.NewFileBroadcaster(app.BasePath("storage", "logs", "broadcast.jsonl"))
	if err != nil {
		return err
	}
	mgr := broadcasting.NewManager(env.Get("BROADCAST_CONNECTION", "log"), map[string]broadcasting.Broadcaster{
		"log":  broadcasting.NewLogBroadcaster(app.Logger()),
		"file": fileBroadcast,
		"null": broadcasting.NullBroadcaster{},
	})
	mgr.Channel("public", func(req *http.Request, channel string) bool {
		return true
	})
	mgr.Channel("private.*", func(req *http.Request, channel string) bool {
		return auth.From(app) != nil && auth.From(app).Check(req)
	})
	app.Container().Instance("broadcast", mgr)
	return nil
}

func BootNotificationServices(app *core.Application) error {
	pkgconfig.LoadIfAbsent(app.Config(), "notifications", appconfig.Notifications())
	mgr := notification.NewManager()
	if app.Logger() != nil {
		mgr.SetErrorHandler(func(err error) {
			app.Logger().Errorf("notification: %v", err)
		})
	}

	mgr.SetMail(notification.NewMailManager(
		env.Get("MAIL_MAILER", "log"),
		env.Get("MAIL_FROM_ADDRESS", "hello@example.com"),
		env.Get("MAIL_FROM_NAME", app.Config().GetString("app.name", "ZATRANO")),
		map[string]notification.Mailer{
			"log": notification.NewLogMailer(app.Logger()),
			"smtp": notification.NewSMTPMailer(notification.SMTPConfig{
				Host:       env.Get("MAIL_HOST", "127.0.0.1"),
				Port:       env.Get("MAIL_PORT", "2525"),
				Username:   env.Get("MAIL_USERNAME"),
				Password:   env.Get("MAIL_PASSWORD"),
				Encryption: env.Get("MAIL_ENCRYPTION"),
			}),
		},
	))

	if broadcasting.From(app) != nil {
		mgr.Extend("broadcast", notification.NewBroadcastChannel(broadcasting.From(app)))
	}
	var pushSender notification.PushSender
	switch strings.ToLower(strings.TrimSpace(env.Get("PUSH_DRIVER", "memory"))) {
	case "http":
		pushSender = &notification.HTTPPushSender{
			Endpoint: env.Get("PUSH_URL", ""),
			Token:    env.Get("PUSH_TOKEN", ""),
		}
	default:
		pushSender = &notification.MemoryPushSender{}
	}
	mgr.Extend("push", notification.NewPushChannel(pushSender))

	smsFrom := env.Get("SMS_FROM", env.Get("APP_NAME", "ZATRANO"))
	smsMgr := notification.NewSmsManager(smsFrom)
	smsMgr.Extend("memory", &notification.MemorySmsSender{})
	smsMgr.Extend("log", &notification.LogSmsSender{
		Log: func(format string, args ...any) {
			if app.Logger() != nil {
				app.Logger().Infof(format, args...)
			}
		},
	})
	smsMgr.Extend("http", &notification.HTTPSmsSender{
		Endpoint: env.Get("SMS_URL", ""),
		Token:    env.Get("SMS_TOKEN", ""),
		Method:   env.Get("SMS_METHOD", "POST"),
	})
	smsMgr.Extend("twilio", &notification.TwilioSmsSender{
		AccountSID: env.Get("TWILIO_ACCOUNT_SID", ""),
		AuthToken:  env.Get("TWILIO_AUTH_TOKEN", ""),
		From:       env.Get("TWILIO_FROM", smsFrom),
	})
	defaultSMS := strings.ToLower(strings.TrimSpace(env.Get("SMS_DRIVER", "memory")))
	if smsMgr.Sender(defaultSMS) == nil {
		defaultSMS = "memory"
	}
	smsMgr.Use(defaultSMS)
	mgr.SetSms(smsMgr)
	if tr := localization.From(app); tr != nil {
		mgr.SetTranslator(tr)
	}
	mgr.SetMailDefaults(
		app.Config().GetString("app.locale", env.Get("APP_LOCALE", "en")),
		app.Config().GetString("app.name", env.Get("APP_NAME", "ZATRANO")),
	)
	if dbMgr := database.From(app); dbMgr != nil {
		if db, err := dbMgr.DB(); err == nil {
			driver, _ := dbMgr.DriverName()
			mgr.Extend("database", notification.NewDatabaseChannel(db, "notifications", driver))
			mgr.SetStore(notification.NewStore(db, "notifications", driver))
		}
	}
	app.Container().Instance("notifications", mgr)
	return nil
}

func BootFilesystemServices(app *core.Application) error {
	localDisk, err := filesystem.NewLocalDisk(app.BasePath("storage", "app"))
	if err != nil {
		return err
	}
	publicDisk, err := filesystem.NewLocalDisk(app.BasePath("storage", "app", "public"))
	if err != nil {
		return err
	}
	appURL := app.Config().GetString("app.url", env.Get("APP_URL", "http://localhost:8080"))
	publicDisk.SetBaseURL(strings.TrimRight(appURL, "/") + "/storage")
	signingKey := app.Config().GetString("app.key", env.Get("APP_KEY", "zatrano-dev-key"))
	localDisk.SetSigningKey(signingKey)
	localDisk.SetServePath("/storage/temporary")
	localDisk.SetBaseURL(strings.TrimRight(appURL, "/"))
	publicDisk.SetSigningKey(signingKey)
	publicDisk.SetServePath("/storage/temporary")
	mgr := filesystem.NewManager(env.Get("FILESYSTEM_DISK", "local"), map[string]filesystem.Disk{
		"local":  localDisk,
		"public": publicDisk,
		"s3": filesystem.NewCloudDisk(
			env.Get("AWS_BUCKET", "zatrano"),
			env.Get("AWS_URL", "https://s3.example.com"),
		),
	})
	app.Container().Instance("files", mgr)
	return nil
}

func BootAssetsServices(app *core.Application) error {
	publicURL := strings.TrimRight(app.Config().GetString("app.url", "http://localhost:8080"), "/")
	mgr := assets.LoadDefault(app.BasePath(), publicURL)
	app.Container().Instance("assets", mgr)
	return nil
}

func BootViewSession(app *core.Application) error {
	engine := view.New(app.BasePath("views"))
	engine.EnableCache(!app.IsDebug())
	engine.Share("appName", app.Config().GetString("app.name", "ZATRANO"))
	if tr := localization.From(app); tr != nil {
		engine.Share("locale", tr.GetLocale())
		engine.AddFunc("trans", func(localeOrKey string, args ...any) string {
			locale := ""
			key := localeOrKey
			var replace map[string]string
			if len(args) == 0 {
				return tr.Get(key)
			}
			// trans .locale "key" [replace]
			if s, ok := args[0].(string); ok {
				locale = localeOrKey
				key = s
				if len(args) > 1 {
					replace = coerceStringMap(args[1])
				}
				return tr.GetFor(locale, key, replace)
			}
			replace = coerceStringMap(args[0])
			return tr.Get(key, replace)
		})
		engine.AddFunc("dict", func(pairs ...any) map[string]any {
			out := map[string]any{}
			for i := 0; i+1 < len(pairs); i += 2 {
				out[fmt.Sprint(pairs[i])] = pairs[i+1]
			}
			return out
		})
		engine.AddFunc("choice", func(key string, number any) string {
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
			return tr.Choice(key, n)
		})
	}
	if a := assets.From(app); a != nil {
		engine.AddFunc("vite", func(path string) string {
			return a.URL(path)
		})
		engine.AddFunc("mix", func(path string) string {
			return a.URL(path)
		})
	}
	engine.SetEnvironment(app.Environment())
	app.Container().Instance("view", engine)
	if n := notification.From(app); n != nil {
		n.SetMailView(engine)
	}
	if s := schedule.From(app); s != nil {
		s.SetMutexPath(app.BasePath("storage", "framework", "schedule"))
	}

	sess := session.NewManager(
		app.BasePath("storage", "framework", "sessions"),
		env.GetInt("SESSION_LIFETIME", 120),
	)
	app.Container().Instance("session", sess)
	if a := auth.From(app); a != nil {
		a.SetSessionManager(sess)
	}
	if p := auth.Passwords(app); p != nil {
		p.SetSessionManager(sess)
	}
	installHTTPBridge(app)
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

func coerceStringMap(v any) map[string]string {
	if v == nil {
		return nil
	}
	if m, ok := v.(map[string]string); ok {
		return m
	}
	if m, ok := v.(map[string]any); ok {
		out := make(map[string]string, len(m))
		for k, val := range m {
			out[k] = asString(val)
		}
		return out
	}
	return nil
}

func asInt(value any) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case float64:
		return int(v), true
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(v))
		return n, err == nil
	default:
		n, err := strconv.Atoi(strings.TrimSpace(fmt.Sprint(value)))
		return n, err == nil
	}
}
