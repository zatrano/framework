package bootstrap

import (
	"fmt"

	"github.com/zatrano/framework/bootstrap/addons"
	appconfig "github.com/zatrano/framework/config"
	"github.com/zatrano/framework/kernel"
	pkgconfig "github.com/zatrano/framework/packages/config"
	"github.com/zatrano/framework/packages/env"
	"github.com/zatrano/framework/packages/mongo"
	"github.com/zatrano/framework/packages/webauthn"
)

func init() {
	// Nested modules cannot import the root module, so they self-register here.
	addons.Register(addons.Meta{
		Name:        "mongo",
		Key:         "mongo",
		Description: "MongoDB client (separate module)",
		Heavy:       true,
		Factory:     func() kernel.Provider { return &mongoServiceProvider{} },
	})
	addons.Register(addons.Meta{
		Name:        "webauthn",
		Key:         "webauthn",
		Description: "WebAuthn/passkeys (separate module)",
		Heavy:       true,
		Factory:     func() kernel.Provider { return &webauthnServiceProvider{} },
	})
}

type mongoServiceProvider struct{}

func (p *mongoServiceProvider) Register(app *kernel.Application) error {
	pkgconfig.LoadIfAbsent(app.Config(), "mongo", appconfig.Mongo())
	if app.Bound("mongo") {
		return nil
	}
	uri := app.Config().GetString("mongo.uri", env.Get("MONGO_URI", "memory"))
	client := mongo.Connect(uri)
	if err := client.Ping(); err != nil {
		return fmt.Errorf("mongo: %w", err)
	}
	app.Container().Instance("mongo", client)
	return nil
}

func (p *mongoServiceProvider) Boot(app *kernel.Application) error { return nil }

type webauthnServiceProvider struct{}

func (p *webauthnServiceProvider) Register(app *kernel.Application) error {
	pkgconfig.LoadIfAbsent(app.Config(), "webauthn", appconfig.WebAuthn())
	cfg := app.Config()
	rpID := cfg.GetString("webauthn.rp_id", env.Get("WEBAUTHN_RP_ID", ""))
	rpOrigin := cfg.GetString("webauthn.rp_origin", env.Get("WEBAUTHN_RP_ORIGIN", ""))
	rpName := cfg.GetString("webauthn.rp_display_name", env.Get("WEBAUTHN_RP_DISPLAY_NAME", env.Get("WEBAUTHN_RP_NAME", env.Get("APP_NAME", "ZATRANO"))))
	app.Container().Instance("webauthn", webauthn.New(rpID, rpOrigin, rpName))
	return nil
}

func (p *webauthnServiceProvider) Boot(app *kernel.Application) error { return nil }
