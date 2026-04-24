// Package sentrylog — Sentry error tracking entegrasyonu.
// Production'da fark edilemeyen panikleri, 5xx hatalarını ve
// kritik servislerdeki exception'ları Sentry'ye iletir.
// SENTRY_DSN env değişkeni boşsa devre dışı kalır (dev ortamı için).
package sentrylog

import (
	"context"
	"strconv"
	"time"

	"github.com/zatrano/framework/configs/envconfig"
	"github.com/zatrano/framework/configs/logconfig"
	"github.com/zatrano/framework/packages/requestid"

	"github.com/getsentry/sentry-go"
	"github.com/gofiber/fiber/v3"
	"go.uber.org/zap"
)

var enabled bool

// Init — Sentry'yi başlatır. SENTRY_DSN boşsa sessizce atlar.
func Init(release string) {
	dsn := envconfig.String("SENTRY_DSN", "")
	if dsn == "" {
		logconfig.SLog.Info("SENTRY_DSN tanımlı değil, Sentry devre dışı")
		return
	}

	env := envconfig.String("APP_ENV", "development")
	if err := sentry.Init(sentry.ClientOptions{
		Dsn:              dsn,
		Environment:      env,
		Release:          release,
		TracesSampleRate: envconfig.Float("SENTRY_TRACES_SAMPLE_RATE", 0.1),
		BeforeSend: func(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
			// Development ortamında gönderme
			if env == "development" {
				return nil
			}
			return event
		},
		AttachStacktrace: true,
	}); err != nil {
		logconfig.Log.Error("Sentry başlatılamadı", zap.Error(err))
		return
	}

	enabled = true
	logconfig.SLog.Infow("Sentry başlatıldı",
		"environment", env,
		"release", release)
}

// Flush — uygulama kapanmadan önce bekleyen event'leri gönderir.
// main() defer'ında çağrılmalıdır.
func Flush() {
	if !enabled {
		return
	}
	sentry.Flush(2 * time.Second)
}

// CaptureError — bir hatayı Sentry'ye iletir.
func CaptureError(err error, extras ...map[string]interface{}) {
	if !enabled || err == nil {
		return
	}
	sentry.WithScope(func(scope *sentry.Scope) {
		for _, extra := range extras {
			for k, v := range extra {
				scope.SetExtra(k, v)
			}
		}
		sentry.CaptureException(err)
	})
}

// CaptureErrorCtx — context bilgisiyle (request_id, user) hata iletir.
func CaptureErrorCtx(ctx context.Context, err error) {
	if !enabled || err == nil {
		return
	}
	sentry.WithScope(func(scope *sentry.Scope) {
		if rid := requestid.FromContext(ctx); rid != "" {
			scope.SetTag("request_id", rid)
		}
		sentry.CaptureException(err)
	})
}

// Middleware — Fiber v3 için Sentry middleware'i.
// Panik ve 5xx hataları Sentry'ye iletilir; request_id tag olarak eklenir.
func Middleware() fiber.Handler {
	return func(c fiber.Ctx) error {
		if !enabled {
			return c.Next()
		}

		hub := sentry.CurrentHub().Clone()
		hub.Scope().SetTag("request_id", requestid.FromFiber(c))
		hub.Scope().SetTag("method", c.Method())
		hub.Scope().SetTag("path", c.Path())
		hub.Scope().SetTag("ip", c.IP())

		// Panik yakalama
		defer func() {
			if r := recover(); r != nil {
				hub.Recover(r)
				hub.Flush(2 * time.Second)
				panic(r) // fiber recover middleware'e ilet
			}
		}()

		err := c.Next()

		// 5xx hataları Sentry'ye ilet
		if c.Response().StatusCode() >= 500 {
			if err != nil {
				hub.CaptureException(err)
			} else {
				hub.CaptureMessage("5xx response without error object")
			}
			hub.Flush(1 * time.Second)
		}

		return err
	}
}

// SetUser — Sentry scope'una kullanıcı bilgisi ekler.
func SetUser(c fiber.Ctx, id uint, email string) {
	if !enabled {
		return
	}
	sentry.ConfigureScope(func(scope *sentry.Scope) {
		scope.SetUser(sentry.User{
			ID:    strconv.FormatUint(uint64(id), 10),
			Email: email,
		})
	})
}

// IsEnabled — Sentry'nin aktif olup olmadığını döner.
func IsEnabled() bool { return enabled }
