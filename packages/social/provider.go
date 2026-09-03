package social

import (
	"fmt"
	"strings"

	"github.com/zatrano/framework/bootstrap/addons"
	appconfig "github.com/zatrano/framework/config"
	"github.com/zatrano/framework/kernel"
	pkgconfig "github.com/zatrano/framework/packages/config"
	"github.com/zatrano/framework/packages/env"
)

func init() {
	addons.Register(addons.Meta{
		Name:        "social",
		Key:         "social",
		Description: "Social OAuth login (GitHub/Google)",
		Factory:     func() kernel.Provider { return &ServiceProvider{} },
	})
}

// ServiceProvider boots the social addon.
type ServiceProvider struct{}

func (p *ServiceProvider) Register(app *kernel.Application) error {
	pkgconfig.LoadIfAbsent(app.Config(), "social", appconfig.Social())
	SetAllowStubProviders(!app.IsProduction())
	mgr := New()
	cfg := app.Config()
	redirectBase := strings.TrimRight(cfg.GetString("app.url", "http://localhost:8080"), "/")

	githubID := firstNonEmpty(cfg.GetString("social.github_client_id"), env.Get("GITHUB_CLIENT_ID", "github-client-id"))
	githubSecret := firstNonEmpty(cfg.GetString("social.github_client_secret"), env.Get("GITHUB_CLIENT_SECRET", "github-client-secret"))
	googleID := firstNonEmpty(cfg.GetString("social.google_client_id"), env.Get("GOOGLE_CLIENT_ID", "google-client-id"))
	googleSecret := firstNonEmpty(cfg.GetString("social.google_client_secret"), env.Get("GOOGLE_CLIENT_SECRET", "google-client-secret"))

	if app.IsProduction() && IsPlaceholder(googleID, googleSecret) && IsPlaceholder(githubID, githubSecret) {
		return fmt.Errorf("social: OAuth credentials are required in production (set GOOGLE_CLIENT_ID/SECRET and/or GITHUB_CLIENT_ID/SECRET)")
	}

	mgr.Extend("github", GitHub(Config{
		ClientID:     githubID,
		ClientSecret: githubSecret,
		RedirectURL:  firstNonEmpty(cfg.GetString("social.github_redirect_uri"), env.Get("GITHUB_REDIRECT_URI", redirectBase+"/auth/github/callback")),
	}))
	mgr.Extend("google", Google(Config{
		ClientID:     googleID,
		ClientSecret: googleSecret,
		RedirectURL:  firstNonEmpty(cfg.GetString("social.google_redirect_uri"), env.Get("GOOGLE_REDIRECT_URI", redirectBase+"/auth/google/callback")),
	}))
	app.Container().Instance("social", mgr)
	return nil
}

func (p *ServiceProvider) Boot(app *kernel.Application) error {
	if app == nil || app.Router() == nil {
		return nil
	}
	mgr := From(app)
	if mgr == nil {
		return nil
	}
	for _, name := range []string{"google", "github"} {
		prov, err := mgr.Driver(name)
		if err != nil {
			continue
		}
		stub, ok := prov.(*StubProvider)
		if !ok {
			continue
		}
		app.Router().Get("/oauth/"+name+"/authorize", stub.AuthorizeHandler()).As("social." + name + ".stub")
	}
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
