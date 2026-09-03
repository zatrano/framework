package addons

import (
	"fmt"
	"strings"
	"time"

	appconfig "github.com/zatrano/framework/config"
	"github.com/zatrano/framework/core"
	"github.com/zatrano/framework/packages/ai"
	"github.com/zatrano/framework/packages/audit"
	"github.com/zatrano/framework/packages/backup"
	"github.com/zatrano/framework/packages/billing"
	"github.com/zatrano/framework/packages/bus"
	"github.com/zatrano/framework/packages/circuit"
	pkgconfig "github.com/zatrano/framework/packages/config"
	"github.com/zatrano/framework/packages/docs"
	"github.com/zatrano/framework/packages/enums"
	"github.com/zatrano/framework/packages/env"
	"github.com/zatrano/framework/packages/events"
	"github.com/zatrano/framework/packages/features"
	"github.com/zatrano/framework/packages/geo"
	"github.com/zatrano/framework/packages/graphql"
	"github.com/zatrano/framework/packages/hashid"
	"github.com/zatrano/framework/packages/http"
	"github.com/zatrano/framework/packages/inspector"
	"github.com/zatrano/framework/packages/lock"
	"github.com/zatrano/framework/packages/mongo"
	"github.com/zatrano/framework/packages/notification"
	"github.com/zatrano/framework/packages/oauth"
	"github.com/zatrano/framework/packages/octane"
	"github.com/zatrano/framework/packages/otp"
	"github.com/zatrano/framework/packages/pulse"
	"github.com/zatrano/framework/packages/search"
	"github.com/zatrano/framework/packages/shorturl"
	"github.com/zatrano/framework/packages/sitemap"
	"github.com/zatrano/framework/packages/social"
	"github.com/zatrano/framework/packages/tenancy"
	"github.com/zatrano/framework/packages/webauthn"
	"github.com/zatrano/framework/packages/webhooks"
	"github.com/zatrano/framework/packages/wellknown"
)

type FeaturesServiceProvider struct{}

func (p *FeaturesServiceProvider) Register(app *core.Application) error {
	mgr := features.New()
	mgr.Activate("welcome_banner")
	mgr.Deactivate("beta_editor")
	mgr.Rollout("new_dashboard", 25)
	app.Container().Instance("features", mgr)
	return nil
}

func (p *FeaturesServiceProvider) Boot(app *core.Application) error { return nil }

type TenancyServiceProvider struct{}

func (p *TenancyServiceProvider) Register(app *core.Application) error {
	mgr := tenancy.New()
	mgr.Register(tenancy.Tenant{ID: "acme", Name: "Acme Corp", Domain: "acme.localhost"})
	mgr.Register(tenancy.Tenant{ID: "globex", Name: "Globex", Domain: "globex.localhost"})
	mgr.SetResolver(mgr.HeaderOrHostResolver())
	mgr.Bootstrapping(func(t *tenancy.Tenant) error {
		if app.Context() != nil {
			app.Context().Put("tenant.id", t.ID)
			app.Context().Put("tenant.name", t.Name)
		}
		return nil
	})
	app.Container().Instance("tenancy", mgr)
	return nil
}

func (p *TenancyServiceProvider) Boot(app *core.Application) error { return nil }

type GraphQLServiceProvider struct{}

func (p *GraphQLServiceProvider) Register(app *core.Application) error {
	schema := graphql.NewSchema()
	schema.Query("health", func(args map[string]any) (any, error) {
		return "ok", nil
	})
	schema.Query("echo", func(args map[string]any) (any, error) {
		msg, _ := args["message"].(string)
		if msg == "" {
			msg = "hello"
		}
		return msg, nil
	})
	schema.Query("feature", func(args map[string]any) (any, error) {
		f := resolveFeatures(app)
		if f == nil {
			return false, nil
		}
		name, _ := args["name"].(string)
		return f.Active(name), nil
	})
	schema.Mutation("ping", func(args map[string]any) (any, error) {
		return map[string]any{"pong": true}, nil
	})
	app.Container().Instance("graphql", schema)
	return nil
}

func (p *GraphQLServiceProvider) Boot(app *core.Application) error { return nil }

type AuditServiceProvider struct{}

func (p *AuditServiceProvider) Register(app *core.Application) error {
	memoryAudit := audit.NewMemoryStore(500)
	fileAudit, err := audit.NewFileStore(app.BasePath("storage", "logs", "audit.jsonl"))
	var mgr *audit.Manager
	if err != nil {
		if app.Logger() != nil {
			app.Logger().Errorf("audit store: %v", err)
		}
		mgr = audit.New(memoryAudit)
	} else {
		mgr = audit.New(&teeAuditStore{primary: memoryAudit, secondary: fileAudit})
	}
	app.Container().Instance("audit", mgr)
	return nil
}

func (p *AuditServiceProvider) Boot(app *core.Application) error { return nil }

type WebhooksServiceProvider struct{}

func (p *WebhooksServiceProvider) Register(app *core.Application) error {
	mgr := webhooks.New()
	mgr.Register(webhooks.Endpoint{
		URL:    env.Get("WEBHOOK_URL", "https://httpbin.org/post"),
		Secret: env.Get("WEBHOOK_SECRET", "zatrano-webhook-secret"),
		Events: []string{"user.created", "demo.ping", "*"},
	})
	app.Container().Instance("webhooks", mgr)
	return nil
}

func (p *WebhooksServiceProvider) Boot(app *core.Application) error { return nil }

type InspectorServiceProvider struct{}

func (p *InspectorServiceProvider) Register(app *core.Application) error {
	app.Container().Instance("inspector", inspector.New(200))
	return nil
}

func (p *InspectorServiceProvider) Boot(app *core.Application) error { return nil }

type SearchServiceProvider struct{}

func (p *SearchServiceProvider) Register(app *core.Application) error {
	app.Container().Instance("search", search.New(search.NewMemoryEngine()))
	return nil
}

func (p *SearchServiceProvider) Boot(app *core.Application) error { return nil }

type SocialServiceProvider struct{}

func (p *SocialServiceProvider) Register(app *core.Application) error {
	pkgconfig.LoadIfAbsent(app.Config(), "social", appconfig.Social())
	social.SetAllowStubProviders(!app.IsProduction())
	mgr := social.New()
	cfg := app.Config()
	redirectBase := strings.TrimRight(cfg.GetString("app.url", "http://localhost:8080"), "/")

	githubID := firstNonEmpty(cfg.GetString("social.github_client_id"), env.Get("GITHUB_CLIENT_ID", "github-client-id"))
	githubSecret := firstNonEmpty(cfg.GetString("social.github_client_secret"), env.Get("GITHUB_CLIENT_SECRET", "github-client-secret"))
	googleID := firstNonEmpty(cfg.GetString("social.google_client_id"), env.Get("GOOGLE_CLIENT_ID", "google-client-id"))
	googleSecret := firstNonEmpty(cfg.GetString("social.google_client_secret"), env.Get("GOOGLE_CLIENT_SECRET", "google-client-secret"))

	// Production must not fall back to StubProvider. At least one real provider is required.
	if app.IsProduction() && social.IsPlaceholder(googleID, googleSecret) && social.IsPlaceholder(githubID, githubSecret) {
		return fmt.Errorf("social: OAuth credentials are required in production (set GOOGLE_CLIENT_ID/SECRET and/or GITHUB_CLIENT_ID/SECRET)")
	}

	mgr.Extend("github", social.GitHub(social.Config{
		ClientID:     githubID,
		ClientSecret: githubSecret,
		RedirectURL:  firstNonEmpty(cfg.GetString("social.github_redirect_uri"), env.Get("GITHUB_REDIRECT_URI", redirectBase+"/auth/github/callback")),
	}))
	mgr.Extend("google", social.Google(social.Config{
		ClientID:     googleID,
		ClientSecret: googleSecret,
		RedirectURL:  firstNonEmpty(cfg.GetString("social.google_redirect_uri"), env.Get("GOOGLE_REDIRECT_URI", redirectBase+"/auth/google/callback")),
	}))
	app.Container().Instance("social", mgr)
	return nil
}

func (p *SocialServiceProvider) Boot(app *core.Application) error {
	if app == nil || app.Router() == nil {
		return nil
	}
	mgr := social.From(app)
	if mgr == nil {
		return nil
	}
	for _, name := range []string{"google", "github"} {
		prov, err := mgr.Driver(name)
		if err != nil {
			continue
		}
		stub, ok := prov.(*social.StubProvider)
		if !ok {
			continue
		}
		app.Router().Get("/oauth/"+name+"/authorize", stub.AuthorizeHandler()).As("social." + name + ".stub")
	}
	return nil
}

type EnumsServiceProvider struct{}

func (p *EnumsServiceProvider) Register(app *core.Application) error {
	reg := enums.NewRegistry()
	reg.Register(enums.NewString("post_status", "draft:Draft", "published:Published", "archived:Archived"))
	reg.Register(enums.NewString("user_role", "admin", "editor", "viewer"))
	app.Container().Instance("enums", reg)
	return nil
}

func (p *EnumsServiceProvider) Boot(app *core.Application) error { return nil }

type BusServiceProvider struct{}

func (p *BusServiceProvider) Register(app *core.Application) error {
	app.Container().Instance("bus", bus.New())
	return nil
}

func (p *BusServiceProvider) Boot(app *core.Application) error { return nil }

type PulseServiceProvider struct{}

func (p *PulseServiceProvider) Register(app *core.Application) error {
	if app.Metrics() == nil {
		return nil
	}
	dash := pulse.New(app.Metrics()).WithExtra(func() map[string]any {
		extra := map[string]any{}
		if insp := resolveInspector(app); insp != nil {
			extra["inspector_entries"] = insp.Count()
		}
		if s := resolveSearch(app); s != nil {
			extra["search_docs"] = s.Count()
		}
		return extra
	})
	app.Container().Instance("pulse", dash)
	return nil
}

func (p *PulseServiceProvider) Boot(app *core.Application) error { return nil }

type BackupServiceProvider struct{}

func (p *BackupServiceProvider) Register(app *core.Application) error {
	mgr, err := backup.ManagerFromApp(app, "")
	if err != nil {
		return err
	}
	app.Container().Instance("backup", mgr)
	return nil
}

func (p *BackupServiceProvider) Boot(app *core.Application) error { return nil }

type DocsServiceProvider struct{}

func (p *DocsServiceProvider) Register(app *core.Application) error {
	app.Container().Instance("docs", docs.New(app.BasePath("docs")))
	return nil
}

func (p *DocsServiceProvider) Boot(app *core.Application) error { return nil }

type BillingServiceProvider struct{}

func (p *BillingServiceProvider) Register(app *core.Application) error {
	pkgconfig.LoadIfAbsent(app.Config(), "billing", appconfig.Billing())
	baseURL := app.Config().GetString("app.url", env.Get("APP_URL", "http://localhost:8080"))
	mgr := billing.NewManager(baseURL)

	successURL := app.Config().GetString("billing.success_url", "")
	cancelURL := app.Config().GetString("billing.cancel_url", "")
	if successURL != "" || cancelURL != "" {
		mgr.SetCheckoutURLs(successURL, cancelURL)
	}

	mgr.Extend("memory", billing.NewMemoryGateway(baseURL))
	stripeKey := app.Config().GetString("billing.stripe_secret", env.Get("STRIPE_SECRET_KEY", ""))
	if stripeKey != "" {
		mgr.Extend("stripe", billing.NewStripeGateway(stripeKey))
	}

	defaultGW := strings.ToLower(strings.TrimSpace(app.Config().GetString("billing.default", env.Get("BILLING_GATEWAY", ""))))
	if defaultGW == "" {
		if stripeKey != "" {
			defaultGW = "stripe"
		} else {
			defaultGW = "memory"
		}
	}
	mgr.Use(defaultGW)

	mgr.SetWebhookSecret(app.Config().GetString("billing.stripe_webhook_secret", env.Get("STRIPE_WEBHOOK_SECRET", "")))
	if d := events.From(app); d != nil {
		mgr.SetDispatcher(d)
	}
	if n := notification.From(app); n != nil {
		mgr.SetNotifier(func(email string, msg any) error {
			if email == "" || msg == nil {
				return nil
			}
			var notif notification.Notification
			switch t := msg.(type) {
			case billing.InvoicePaidNotification:
				notif = t
			case *billing.InvoicePaidNotification:
				notif = *t
			case billing.SubscriptionStartedNotification:
				notif = t
			case *billing.SubscriptionStartedNotification:
				notif = *t
			default:
				return nil
			}
			return n.Send(notification.Recipient{Email: email, ID: email}, notif)
		})
	}

	app.Container().Instance("billing", mgr)
	return nil
}

func (p *BillingServiceProvider) Boot(app *core.Application) error {
	if app == nil || app.Router() == nil {
		return nil
	}
	mgr := billing.From(app)
	if mgr == nil {
		return nil
	}
	app.Router().Post("/billing/webhook", func(req *http.Request) *http.Response {
		if err := mgr.HandleHTTP(req); err != nil {
			return http.JSON(map[string]any{"message": err.Error()}).Status(400)
		}
		return http.JSON(map[string]any{"received": true})
	}).As("billing.webhook")
	return nil
}

type MongoServiceProvider struct{}

func (p *MongoServiceProvider) Register(app *core.Application) error {
	pkgconfig.LoadIfAbsent(app.Config(), "mongo", appconfig.Mongo())
	if app.Bound("mongo") {
		return nil // already wired via db:setup / BootDatabase
	}
	uri := app.Config().GetString("mongo.uri", env.Get("MONGO_URI", "memory"))
	client := mongo.Connect(uri)
	if err := client.Ping(); err != nil {
		return fmt.Errorf("mongo: %w", err)
	}
	app.Container().Instance("mongo", client)
	return nil
}

func (p *MongoServiceProvider) Boot(app *core.Application) error { return nil }

type OAuthServiceProvider struct{}

func (p *OAuthServiceProvider) Register(app *core.Application) error {
	pkgconfig.LoadIfAbsent(app.Config(), "oauth", appconfig.OAuth())
	var server *oauth.Server
	storePath := strings.TrimSpace(app.Config().GetString("oauth.store_path", env.Get("OAUTH_STORE_PATH", "")))
	if storePath != "" {
		oa, err := oauth.NewWithStore(storePath)
		if err != nil {
			if app.Logger() != nil {
				app.Logger().Errorf("oauth store: %v", err)
			}
			server = oauth.New()
		} else {
			server = oa
		}
	} else {
		server = oauth.New()
	}
	app.Container().Instance("oauth", server)
	return nil
}

func (p *OAuthServiceProvider) Boot(app *core.Application) error { return nil }

type OctaneServiceProvider struct{}

func (p *OctaneServiceProvider) Register(app *core.Application) error {
	app.Container().Instance("octane", octane.New(env.GetInt("OCTANE_WORKERS", 0)))
	return nil
}

func (p *OctaneServiceProvider) Boot(app *core.Application) error { return nil }

type AIServiceProvider struct{}

func (p *AIServiceProvider) Register(app *core.Application) error {
	pkgconfig.LoadIfAbsent(app.Config(), "ai", appconfig.AI())
	mgr := ai.New()
	var logFn ai.LogFn
	if lg := app.Logger(); lg != nil {
		logFn = lg.Infof
	}
	cfg := map[string]any{}
	if raw := app.Config().Get("ai"); raw != nil {
		if m, ok := raw.(map[string]any); ok {
			cfg = m
		}
	}
	// Merge top-level GetString fallbacks for flat keys when nested map missing fields.
	if cfg["api_key"] == nil || fmt.Sprint(cfg["api_key"]) == "" {
		cfg["api_key"] = app.Config().GetString("ai.api_key", env.Get("AI_API_KEY", env.Get("OPENAI_API_KEY", "")))
	}
	if cfg["driver"] == nil || fmt.Sprint(cfg["driver"]) == "" {
		cfg["driver"] = app.Config().GetString("ai.driver", env.Get("AI_DRIVER", ""))
	}
	if cfg["default"] == nil || fmt.Sprint(cfg["default"]) == "" {
		cfg["default"] = app.Config().GetString("ai.default", env.Get("AI_DEFAULT", env.Get("AI_DRIVER", "")))
	}
	if cfg["base_url"] == nil || fmt.Sprint(cfg["base_url"]) == "" {
		cfg["base_url"] = app.Config().GetString("ai.base_url", env.Get("AI_BASE_URL", ""))
	}
	if cfg["model"] == nil || fmt.Sprint(cfg["model"]) == "" {
		cfg["model"] = app.Config().GetString("ai.model", env.Get("AI_MODEL", ""))
	}
	if cfg["timeout"] == nil {
		cfg["timeout"] = app.Config().GetInt("ai.timeout", env.GetInt("AI_TIMEOUT", 30))
	}
	if cfg["temperature"] == nil || fmt.Sprint(cfg["temperature"]) == "" {
		cfg["temperature"] = app.Config().GetString("ai.temperature", env.Get("AI_TEMPERATURE", ""))
	}
	if cfg["max_tokens"] == nil {
		cfg["max_tokens"] = app.Config().GetInt("ai.max_tokens", env.GetInt("AI_MAX_TOKENS", 0))
	}
	if cfg["providers"] == nil {
		if raw := app.Config().Get("ai.providers"); raw != nil {
			cfg["providers"] = raw
		}
	}
	if cfg["profiles"] == nil {
		if raw := app.Config().Get("ai.profiles"); raw != nil {
			cfg["profiles"] = raw
		}
	}
	if err := mgr.BootConfig(cfg, logFn); err != nil {
		return err
	}
	app.Container().Instance("ai", mgr)
	return nil
}

func (p *AIServiceProvider) Boot(app *core.Application) error {
	if app == nil || app.Router() == nil {
		return nil
	}
	mgr := ai.From(app)
	if mgr == nil {
		return nil
	}
	app.Router().Post("/demo/ai/chat", ai.DemoChatHandler(mgr)).As("demo.ai.chat")
	return nil
}

type SitemapServiceProvider struct{}

func (p *SitemapServiceProvider) Register(app *core.Application) error {
	base := strings.TrimRight(app.Config().GetString("app.url", env.Get("APP_URL", "http://localhost:8080")), "/")
	builder := sitemap.New(base)
	builder.Add("/", sitemap.URL{Priority: 1.0, ChangeFreq: "daily"})
	builder.Add("/up", sitemap.URL{Priority: 0.1, ChangeFreq: "monthly"})
	app.Container().Instance("sitemap", builder)
	return nil
}

func (p *SitemapServiceProvider) Boot(app *core.Application) error { return nil }

type LockServiceProvider struct{}

func (p *LockServiceProvider) Register(app *core.Application) error {
	app.Container().Instance("lock", lock.New())
	return nil
}

func (p *LockServiceProvider) Boot(app *core.Application) error { return nil }

type CircuitServiceProvider struct{}

func (p *CircuitServiceProvider) Register(app *core.Application) error {
	app.Container().Instance("circuit", circuit.New(circuit.Settings{
		FailureThreshold: env.GetInt("CIRCUIT_FAILURE_THRESHOLD", 5),
		SuccessThreshold: env.GetInt("CIRCUIT_SUCCESS_THRESHOLD", 2),
		Timeout:          time.Duration(env.GetInt("CIRCUIT_TIMEOUT_SECONDS", 30)) * time.Second,
	}))
	return nil
}

func (p *CircuitServiceProvider) Boot(app *core.Application) error { return nil }

type HashIDServiceProvider struct{}

func (p *HashIDServiceProvider) Register(app *core.Application) error {
	app.Container().Instance("hashid", hashid.New(
		env.Get("HASHID_SALT", app.Config().GetString("app.key", "zatrano")),
		env.GetInt("HASHID_MIN_LENGTH", 8),
	))
	return nil
}

func (p *HashIDServiceProvider) Boot(app *core.Application) error { return nil }

type ShortURLServiceProvider struct{}

func (p *ShortURLServiceProvider) Register(app *core.Application) error {
	base := strings.TrimRight(app.Config().GetString("app.url", env.Get("APP_URL", "http://localhost:8080")), "/")
	app.Container().Instance("shorturl", shorturl.New(base, env.Get("SHORTURL_PREFIX", "/s")))
	return nil
}

func (p *ShortURLServiceProvider) Boot(app *core.Application) error { return nil }

type WellKnownServiceProvider struct{}

func (p *WellKnownServiceProvider) Register(app *core.Application) error {
	base := strings.TrimRight(app.Config().GetString("app.url", env.Get("APP_URL", "http://localhost:8080")), "/")
	app.Container().Instance("wellknown", wellknown.New(wellknown.Config{
		ContactEmail:  env.Get("SECURITY_CONTACT_EMAIL", "security@zatrano.test"),
		ContactURL:    env.Get("SECURITY_CONTACT_URL", base+"/contact"),
		Canonical:     base + "/.well-known/security.txt",
		PolicyURL:     env.Get("SECURITY_POLICY_URL", base+"/documentation"),
		PreferredLang: env.Get("APP_LOCALE", "en"),
	}))
	return nil
}

func (p *WellKnownServiceProvider) Boot(app *core.Application) error { return nil }

type GeoServiceProvider struct{}

func (p *GeoServiceProvider) Register(app *core.Application) error {
	app.Container().Instance("geo", geo.New())
	return nil
}

func (p *GeoServiceProvider) Boot(app *core.Application) error { return nil }

type WebAuthnServiceProvider struct{}

func (p *WebAuthnServiceProvider) Register(app *core.Application) error {
	pkgconfig.LoadIfAbsent(app.Config(), "webauthn", appconfig.WebAuthn())
	cfg := app.Config()
	rpID := cfg.GetString("webauthn.rp_id", env.Get("WEBAUTHN_RP_ID", ""))
	rpOrigin := cfg.GetString("webauthn.rp_origin", env.Get("WEBAUTHN_RP_ORIGIN", ""))
	rpName := cfg.GetString("webauthn.rp_display_name", env.Get("WEBAUTHN_RP_DISPLAY_NAME", env.Get("WEBAUTHN_RP_NAME", env.Get("APP_NAME", "ZATRANO"))))
	app.Container().Instance("webauthn", webauthn.New(rpID, rpOrigin, rpName))
	return nil
}

func (p *WebAuthnServiceProvider) Boot(app *core.Application) error { return nil }

type OTPServiceProvider struct{}

func (p *OTPServiceProvider) Register(app *core.Application) error {
	app.Container().Instance("otp", otp.New(otp.NewMemoryStore()).WithTTL(5*time.Minute))
	return nil
}

func (p *OTPServiceProvider) Boot(app *core.Application) error { return nil }

func resolveFeatures(app *core.Application) *features.Manager {
	raw, err := app.Make("features")
	if err != nil {
		return nil
	}
	v, _ := raw.(*features.Manager)
	return v
}

func resolveInspector(app *core.Application) *inspector.Manager {
	raw, err := app.Make("inspector")
	if err != nil {
		return nil
	}
	v, _ := raw.(*inspector.Manager)
	return v
}

func resolveSearch(app *core.Application) *search.Manager {
	raw, err := app.Make("search")
	if err != nil {
		return nil
	}
	v, _ := raw.(*search.Manager)
	return v
}

type teeAuditStore struct {
	primary   audit.Store
	secondary audit.Store
}

func (s *teeAuditStore) Write(event audit.Event) error {
	if err := s.primary.Write(event); err != nil {
		return err
	}
	return s.secondary.Write(event)
}

func (s *teeAuditStore) Recent(limit int) ([]audit.Event, error) {
	return s.primary.Recent(limit)
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
