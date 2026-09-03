package webhooks

import (
	"github.com/zatrano/framework/bootstrap/addons"
	"github.com/zatrano/framework/core"
	"github.com/zatrano/framework/packages/env"
)

func init() {
	addons.Register(addons.Meta{
		Name:        "webhooks",
		Key:         "webhooks",
		Description: "Outbound signed webhooks",
		Factory:     func() core.Provider { return &ServiceProvider{} },
	})
}

// ServiceProvider boots the webhooks addon.
type ServiceProvider struct{}

func (p *ServiceProvider) Register(app *core.Application) error {
	mgr := New()
	mgr.Register(Endpoint{
		URL:    env.Get("WEBHOOK_URL", "https://httpbin.org/post"),
		Secret: env.Get("WEBHOOK_SECRET", "zatrano-webhook-secret"),
		Events: []string{"user.created", "demo.ping", "*"},
	})
	app.Container().Instance("webhooks", mgr)
	return nil
}

func (p *ServiceProvider) Boot(app *core.Application) error { return nil }
