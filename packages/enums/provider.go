package enums

import (
	"github.com/zatrano/framework/bootstrap/addons"
	"github.com/zatrano/framework/kernel"
)

func init() {
	addons.Register(addons.Meta{
		Name:        "enums",
		Key:         "enums",
		Description: "String enum registry",
		Factory:     func() kernel.Provider { return &ServiceProvider{} },
	})
}

// ServiceProvider boots the enums addon.
type ServiceProvider struct{}

func (p *ServiceProvider) Register(app *kernel.Application) error {
	reg := NewRegistry()
	reg.Register(NewString("post_status", "draft:Draft", "published:Published", "archived:Archived"))
	reg.Register(NewString("user_role", "admin", "editor", "viewer"))
	app.Container().Instance("enums", reg)
	return nil
}

func (p *ServiceProvider) Boot(app *kernel.Application) error { return nil }
