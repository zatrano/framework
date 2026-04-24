package main

import (
	"context"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	apiroutes "github.com/zatrano/framework/api/v1/routes"
	appcontainer "github.com/zatrano/framework/app"
	"github.com/zatrano/framework/configs/csrfconfig"
	"github.com/zatrano/framework/configs/databaseconfig"
	"github.com/zatrano/framework/configs/envconfig"
	"github.com/zatrano/framework/configs/fileconfig"
	"github.com/zatrano/framework/configs/logconfig"
	"github.com/zatrano/framework/configs/redisconfig"
	"github.com/zatrano/framework/configs/sessionconfig"
	"github.com/zatrano/framework/configs/validateconfig"
	"github.com/zatrano/framework/middlewares"
	"github.com/zatrano/framework/observability"
	"github.com/zatrano/framework/packages/flashmessages"
	"github.com/zatrano/framework/packages/i18n"
	"github.com/zatrano/framework/packages/mailqueue"
	"github.com/zatrano/framework/packages/renderer"
	"github.com/zatrano/framework/packages/requestid"
	"github.com/zatrano/framework/packages/sentrylog"
	"github.com/zatrano/framework/packages/templatehelpers"
	"github.com/zatrano/framework/routes"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/static"
	"github.com/gofiber/template/html/v2"
	"go.uber.org/zap"
)

// version — CI/CD tarafından build sırasında -ldflags ile enjekte edilir.
// Örnek: go build -ldflags="-X main.version=1.2.3" ./...
var version = "dev"

func main() {
	// ─── 1. Ortam ayarlarını yükle ─────────────────────────────────────────────
	envconfig.Load()
	envconfig.LoadIfDev()

	// ─── 2. Günlükleyici ───────────────────────────────────────────────────────
	logconfig.InitLogger()
	defer logconfig.SyncLogger()

	appEnv := envconfig.String("APP_ENV", "development")
	logconfig.SLog.Infow("Runtime",
		"version", version,
		"env", appEnv,
		"num_cpu", runtime.NumCPU(),
		"gomaxprocs", runtime.GOMAXPROCS(0),
		"go_version", runtime.Version(),
	)

	// ─── 3. Config doğrulama — FAIL FAST ─────────────────────────────────────
	if err := validateconfig.ValidateAll(); err != nil {
		logconfig.Log.Fatal("Ortam değişkeni doğrulama başarısız", zap.Error(err))
		os.Exit(1)
	}
	if envconfig.IsProd() {
		if err := validateconfig.ValidateProduction(); err != nil {
			logconfig.Log.Fatal("Production config doğrulama başarısız", zap.Error(err))
			os.Exit(1)
		}
	}
	logconfig.SLog.Info("Ortam değişkenleri doğrulandı")

	// ─── 4. Sentry — hata takibi ──────────────────────────────────────────────
	sentrylog.Init(version)
	defer sentrylog.Flush()

	// ─── 5. Prometheus metrikleri ───────────────────────────────────────────────
	observability.Register()

	// ─── 6. i18n — çoklu dil ─────────────────────────────────────────────────
	i18n.Init()

	// ─── 7. Veritabanı + Redis + Oturum ────────────────────────────────────────
	databaseconfig.InitDB()
	defer databaseconfig.CloseDB()

	redisconfig.InitRedis()
	defer redisconfig.Close()

	sessionconfig.InitSession()

	// ─── 8. Dosya yükleme ─────────────────────────────────────────────────────
	fileconfig.InitFileConfig()
	fileconfig.Config.SetAllowedExtensions("uploads", []string{"jpg", "jpeg", "png", "webp", "pdf"})

	// ─── 9. Template engine ───────────────────────────────────────────────────
	engine := html.New("./views", ".html")
	engine.AddFunc("getFlashMessages", flashmessages.GetFlashMessages)
	// i18n template helper ekle
	engine.AddFunc("t", func(key string, args ...interface{}) string {
		return i18n.Translate(envconfig.String("DEFAULT_LANG", i18n.DefaultLang), key, args...)
	})
	engine.AddFuncMap(templatehelpers.TemplateHelpers())
	if !envconfig.IsProd() {
		engine.Reload(true)
	}

	// ─── 10. DI Container ────────────────────────────────────────────────────
	container := appcontainer.Build()

	// ─── 11. Asenkron Mail Kuyruğu ───────────────────────────────────────────
	mailqueue.Init(container.Mail, 3, 200)

	// ─── 12. Fiber app ────────────────────────────────────────────────────────
	app := fiber.New(fiber.Config{
		Views:       engine,
		IdleTimeout: 60 * time.Second,
		ReadTimeout: 30 * time.Second,
		BodyLimit:   10 * 1024 * 1024,

		TrustProxy: true,
		TrustProxyConfig: fiber.TrustProxyConfig{
			Proxies: []string{"127.0.0.1", "::1"},
		},
		ProxyHeader: "CF-Connecting-IP",

		ErrorHandler: func(c fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			message := i18n.T(c, "error.internal")
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}

			// 5xx → Sentry'ye bildir
			if code >= 500 {
				sentrylog.CaptureErrorCtx(c.Context(), err)
				logconfig.Log.Error("Fiber request error",
					zap.Error(err),
					zap.Int("status_code", code),
					zap.String("method", c.Method()),
					zap.String("path", c.Path()),
					zap.String("ip", c.IP()),
					zap.String("request_id", requestid.FromFiber(c)),
				)
			}

			// API istekleri JSON hata dönsün
			if strings.HasPrefix(c.Path(), "/api/") {
				return c.Status(code).JSON(fiber.Map{
					"error": fiber.Map{
						"status":  code,
						"message": message,
					},
				})
			}

			// HTML istekleri hata sayfası dönsün
			if strings.Contains(c.Get("Accept"), "text/html") && code >= 400 {
				title := i18n.T(c, "error.internal")
				msg := message
				switch code {
				case 404:
					title = i18n.T(c, "error.not_found")
					msg = title
				case 403:
					title = i18n.T(c, "error.forbidden")
					msg = title
				}
				return renderer.Render(c, "website/error", "layouts/website", fiber.Map{
					"Code": code, "Title": title, "Message": msg,
				}, code)
			}
			return c.Status(code).SendString(message)
		},
	})

	// ─── Lifecycle Hooks ──────────────────────────────────────────────────────
	app.Hooks().OnListen(func(data fiber.ListenData) error {
		logconfig.SLog.Infow("Uygulama dinleniyor",
			"host", data.Host, "port", data.Port,
			"env", appEnv, "version", version)
		return nil
	})
	app.Hooks().OnShutdown(func() error {
		logconfig.Log.Info("HTTP sunucusu kapatıldı")
		return nil
	})

	// ─── Sistem uç noktaları ─────────────────────────────────────────────────
	app.Get("/health", func(c fiber.Ctx) error {
		db, _ := databaseconfig.GetDB().DB()
		dbOk := db.Ping() == nil
		pingCtx, cancel := context.WithTimeout(c.Context(), 2*time.Second)
		defer cancel()
		_, redisErr := redisconfig.GetClient().Ping(pingCtx).Result()
		redisOk := redisErr == nil
		allOk := dbOk && redisOk
		status := 200
		if !allOk {
			status = 503
		}
		return c.Status(status).JSON(fiber.Map{
			"ok": allOk, "database": dbOk, "redis": redisOk,
			"version": version, "timestamp": time.Now().Unix(),
		})
	})

	app.Get("/readyz", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{"ok": true, "version": version})
	})

	app.Get("/metrics", observability.MetricsHandler())

	// ─── .well-known isteği ────────────────────────────────────────────────────
	app.Use(func(c fiber.Ctx) error {
		if strings.HasPrefix(c.Path(), "/.well-known") {
			return c.SendStatus(fiber.StatusNoContent)
		}
		return c.Next()
	})

	// ─── Genel middleware zinciri ───────────────────────────────────────────
	// Sıra: recover → requestid → sentry → metrics → log → i18n → security headers → input sanitizer → compress
	app.Use(recover.New())
	app.Use(requestid.Middleware())        // 1. İstek kimliği — log korelasyonu
	app.Use(sentrylog.Middleware())        // 2. Sentry — panik ve 5xx yakalama
	app.Use(observability.Middleware())    // 3. Prometheus metrikleri
	app.Use(middlewares.ZapLogger())       // 4. Yapılandırılmış loglama (request_id içerir)
	app.Use(i18n.Middleware())             // 5. Dil tespiti
	app.Use(middlewares.SecurityHeaders()) // 6. Güvenlik başlıkları
	app.Use(middlewares.InputSanitizer())  // 7. Form gövdelerinde şüpheli desen işaretleme (web formları)
	app.Use(middlewares.Compress())        // 8. gzip/brotli

	// ─── Statik dosyalar ──────────────────────────────────────────────────────
	app.Get("/*", static.New("./public", static.Config{
		Browse:    false,
		ByteRange: true,
		Next: func(c fiber.Ctx) bool {
			return !strings.ContainsRune(c.Path(), '.')
		},
	}))
	app.Get("/uploads/*", static.New(fileconfig.Config.BasePath, static.Config{
		Browse:    false,
		ByteRange: true,
	}))

	// ─── Method Override ──────────────────────────────────────────────────────
	app.Use(func(c fiber.Ctx) error {
		if c.Method() == fiber.MethodPost {
			if m := c.FormValue("_method"); m == "PUT" || m == "DELETE" || m == "PATCH" {
				c.Method(m)
			}
		}
		return c.Next()
	})

	// ─── CSRF (sadece web form rota'ları için) ───────────────────────────────
	// Not: /api/* uç noktaları JWT kullanır, CSRF muaf
	app.Use(csrfconfig.SetupCSRF())

	// ─── Rate limiting ────────────────────────────────────────────────────────
	app.Use(middlewares.GlobalRateLimit())
	app.Use(middlewares.FormPostRateLimit())

	// ─── Rota grupları ───────────────────────────────────────────────────────
	routes.SetupRoutes(app, container)       // Web (HTML/form, session auth)
	apiroutes.SetupAPIRoutes(app, container) // REST API v1 (JWT auth)

	// ─── Server başlat ────────────────────────────────────────────────────────
	startServer(app)
}

func startServer(app *fiber.App) {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	port := envconfig.Int("APP_PORT", 3000)
	host := envconfig.String("APP_HOST", "127.0.0.1")
	address := host + ":" + strconv.Itoa(port)

	baseURL := envconfig.String("APP_BASE_URL", "")
	if baseURL == "" {
		if envconfig.IsProd() {
			logconfig.Log.Fatal("APP_BASE_URL production ortamında boş olamaz")
		} else {
			baseURL = "http://localhost:" + strconv.Itoa(port)
		}
	}
	if envconfig.IsProd() && !strings.HasPrefix(baseURL, "https://") {
		logconfig.Log.Warn("APP_BASE_URL HTTPS değil, production için önerilmez",
			zap.String("base_url", baseURL))
	}

	go func() {
		if err := app.Listen(address); err != nil {
			logconfig.Log.Fatal("Sunucu dinlenemedi",
				zap.String("address", address), zap.Error(err))
		}
	}()

	<-ctx.Done()
	logconfig.Log.Info("Kapatma sinyali alındı...")

	// 1. Mail kuyruğunu bitir
	mqCtx, mqCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer mqCancel()
	mailqueue.Shutdown(mqCtx)

	// 2. HTTP sunucusunu kapat
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := app.ShutdownWithContext(shutdownCtx); err != nil {
		logconfig.Log.Error("Sunucu kapatılırken hata", zap.Error(err))
	} else {
		logconfig.Log.Info("Sunucu başarıyla kapatıldı")
	}

	// 3. Sentry bekleyen event'leri gönder
	sentrylog.Flush()

	logconfig.Log.Info("Uygulama sonlandırıldı.")
}
