package providers

import (
	"github.com/zatrano/framework/core"
	"github.com/zatrano/framework/packages/assets"
	"github.com/zatrano/framework/packages/middleware/csrf"
	"github.com/zatrano/framework/packages/view"
)

// AppServiceProvider registers application-level services.
type AppServiceProvider struct{}

// Register registers bindings into the container.
func (p *AppServiceProvider) Register(app *core.Application) error {
	return nil
}

// Boot boots application services.
func (p *AppServiceProvider) Boot(app *core.Application) error {
	// Optional CDN-friendly public GETs (no XSRF seed without a session cookie):
	// csrf.SkipAnonymousSeed(func(req *http.Request) bool {
	// 	return req.Path() == "/" || strings.HasPrefix(req.Path(), "/docs")
	// })
	app.Router().Use(csrf.Except("/api"))

	if v := view.From(app); v != nil {
		v.Share("appUrl", app.Config().GetString("app.url"))
		if m := assets.From(app); m != nil && len(m.All()) > 0 {
			v.Share("assetCss", m.URL("css/app.css"))
			v.Share("assetJs", m.URL("js/app.js"))
		}
	}
	return nil
}
