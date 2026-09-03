package oauth

import (
	"strings"

	"github.com/zatrano/framework/bootstrap/addons"
	appconfig "github.com/zatrano/framework/config"
	"github.com/zatrano/framework/kernel"
	pkgconfig "github.com/zatrano/framework/packages/config"
	"github.com/zatrano/framework/packages/env"
)

func init() {
	addons.Register(addons.Meta{
		Name:        "oauth",
		Key:         "oauth",
		Description: "OAuth2 authorization server",
		Factory:     func() kernel.Provider { return &ServiceProvider{} },
	})
}

// ServiceProvider boots the OAuth addon.
type ServiceProvider struct{}

func (p *ServiceProvider) Register(app *kernel.Application) error {
	pkgconfig.LoadIfAbsent(app.Config(), "oauth", appconfig.OAuth())
	var server *Server
	storePath := strings.TrimSpace(app.Config().GetString("oauth.store_path", env.Get("OAUTH_STORE_PATH", "")))
	if storePath != "" {
		oa, err := NewWithStore(storePath)
		if err != nil {
			if app.Logger() != nil {
				app.Logger().Errorf("oauth store: %v", err)
			}
			server = New()
		} else {
			server = oa
		}
	} else {
		server = New()
	}
	app.Container().Instance("oauth", server)
	return nil
}

func (p *ServiceProvider) Boot(app *kernel.Application) error { return nil }
