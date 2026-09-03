package billing

import (
	"strings"

	"github.com/zatrano/framework/bootstrap/addons"
	appconfig "github.com/zatrano/framework/config"
	"github.com/zatrano/framework/core"
	pkgconfig "github.com/zatrano/framework/packages/config"
	"github.com/zatrano/framework/packages/env"
	"github.com/zatrano/framework/packages/events"
	"github.com/zatrano/framework/packages/http"
	"github.com/zatrano/framework/packages/notification"
)

func init() {
	addons.Register(addons.Meta{
		Name:        "billing",
		Key:         "billing",
		Description: "Central billing (memory/stripe gateways, webhooks)",
		Factory:     func() core.Provider { return &ServiceProvider{} },
	})
}

// ServiceProvider boots the billing addon.
type ServiceProvider struct{}

func (p *ServiceProvider) Register(app *core.Application) error {
	pkgconfig.LoadIfAbsent(app.Config(), "billing", appconfig.Billing())
	baseURL := app.Config().GetString("app.url", env.Get("APP_URL", "http://localhost:8080"))
	mgr := NewManager(baseURL)

	successURL := app.Config().GetString("billing.success_url", "")
	cancelURL := app.Config().GetString("billing.cancel_url", "")
	if successURL != "" || cancelURL != "" {
		mgr.SetCheckoutURLs(successURL, cancelURL)
	}

	mgr.Extend("memory", NewMemoryGateway(baseURL))
	stripeKey := app.Config().GetString("billing.stripe_secret", env.Get("STRIPE_SECRET_KEY", ""))
	if stripeKey != "" {
		mgr.Extend("stripe", NewStripeGateway(stripeKey))
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
			case InvoicePaidNotification:
				notif = t
			case *InvoicePaidNotification:
				notif = *t
			case SubscriptionStartedNotification:
				notif = t
			case *SubscriptionStartedNotification:
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

func (p *ServiceProvider) Boot(app *core.Application) error {
	if app == nil || app.Router() == nil {
		return nil
	}
	mgr := From(app)
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
