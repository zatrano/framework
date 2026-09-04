package kernel

import (
	appcontext "github.com/zatrano/framework/context"
	"github.com/zatrano/framework/encryption"
	"github.com/zatrano/framework/env"
	"github.com/zatrano/framework/exceptions"
	"github.com/zatrano/framework/http"
	"github.com/zatrano/framework/report"
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

	app.ctx = appcontext.New()
	app.container.Instance("context", app.ctx)

	encrypter, err := encryption.New(app.config.GetString("app.key", env.Get("APP_KEY", "zatrano-dev-key")))
	if err != nil {
		return err
	}
	app.encrypter = encrypter
	app.container.Instance("encrypter", app.encrypter)
	return nil
}
