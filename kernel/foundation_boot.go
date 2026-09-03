package kernel

import (
	"time"

	appcontext "github.com/zatrano/framework/packages/context"
	"github.com/zatrano/framework/packages/encryption"
	"github.com/zatrano/framework/packages/env"
	"github.com/zatrano/framework/packages/exceptions"
	"github.com/zatrano/framework/packages/hashing"
	"github.com/zatrano/framework/packages/health"
	"github.com/zatrano/framework/packages/http"
	"github.com/zatrano/framework/packages/maintenance"
	"github.com/zatrano/framework/packages/observability"
	"github.com/zatrano/framework/packages/ratelimit"
	"github.com/zatrano/framework/packages/report"
	urlgen "github.com/zatrano/framework/packages/url"
	"github.com/zatrano/framework/packages/version"
)

func (app *Application) BootKernelServices() error {
	app.exceptions = exceptions.New(app.IsDebug() || app.config.GetBool("app.debug", true))
	app.reports = report.New(200)
	if webhook := env.Get("ERROR_WEBHOOK_URL", env.Get("REPORT_WEBHOOK_URL", "")); webhook != "" {
		app.reports.SetWebhook(webhook)
	}
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

	app.metrics = observability.New()
	app.container.Instance("metrics", app.metrics)

	app.health = health.New()
	app.health.Disk(app.BasePath("storage"))
	app.container.Instance("health", app.health)

	app.rateLimiter = ratelimit.New()
	app.rateLimiter.For("api", ratelimit.Limit{MaxAttempts: 60, Decay: time.Minute})
	app.rateLimiter.For("login", ratelimit.Limit{MaxAttempts: 5, Decay: time.Minute})
	app.container.Instance("rateLimiter", app.rateLimiter)

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

	app.hasher = hashing.New()
	app.container.Instance("hash", app.hasher)

	app.maintenance = maintenance.New(app.BasePath("storage", "framework"))
	app.container.Instance("maintenance", app.maintenance)

	_ = version.LoadFile(app.BasePath("VERSION"))
	return nil
}
