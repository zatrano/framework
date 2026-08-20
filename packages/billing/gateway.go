package billing

// Gateway is a pluggable payment provider (memory, stripe, …).
type Gateway interface {
	Name() string
	CreateCustomer(email, name string) (*Customer, error)
	CreateCheckout(input CheckoutInput) (*CheckoutSession, error)
	StartSubscription(customerID, name, priceID string, trialDays int, successURL, cancelURL string) (*Subscription, *CheckoutSession, error)
	CancelSubscription(id string, immediately bool) (*Subscription, error)
	CreateInvoice(customerID string, amount int64, currency string) (*Invoice, error)
}

// WebhookParser optionally verifies and parses provider webhooks.
type WebhookParser interface {
	ParseWebhook(headers map[string]string, body []byte) (*WebhookEvent, error)
}
