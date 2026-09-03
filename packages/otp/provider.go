package otp

import (
	"time"

	"github.com/zatrano/framework/bootstrap/addons"
	"github.com/zatrano/framework/core"
)

func init() {
	addons.Register(addons.Meta{
		Name:        "otp",
		Key:         "otp",
		Description: "One-time passwords",
		Factory:     func() core.Provider { return &ServiceProvider{} },
	})
}

// ServiceProvider boots the OTP addon.
type ServiceProvider struct{}

func (p *ServiceProvider) Register(app *core.Application) error {
	app.Container().Instance("otp", New(NewMemoryStore()).WithTTL(5*time.Minute))
	return nil
}

func (p *ServiceProvider) Boot(app *core.Application) error { return nil }
