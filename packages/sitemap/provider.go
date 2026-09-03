package sitemap

import (
	"strings"

	"github.com/zatrano/framework/bootstrap/addons"
	"github.com/zatrano/framework/core"
	"github.com/zatrano/framework/packages/env"
)

func init() {
	addons.Register(addons.Meta{
		Name:        "sitemap",
		Key:         "sitemap",
		Description: "XML sitemap builder",
		Factory:     func() core.Provider { return &ServiceProvider{} },
	})
}

// ServiceProvider boots the sitemap addon.
type ServiceProvider struct{}

func (p *ServiceProvider) Register(app *core.Application) error {
	base := strings.TrimRight(app.Config().GetString("app.url", env.Get("APP_URL", "http://localhost:8080")), "/")
	builder := New(base)
	builder.Add("/", URL{Priority: 1.0, ChangeFreq: "daily"})
	builder.Add("/up", URL{Priority: 0.1, ChangeFreq: "monthly"})
	app.Container().Instance("sitemap", builder)
	return nil
}

func (p *ServiceProvider) Boot(app *core.Application) error { return nil }
