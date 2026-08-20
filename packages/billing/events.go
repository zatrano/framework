package billing

import "time"

const (
	EventCustomerCreated      = "billing.customer_created"
	EventSubscriptionStarted  = "billing.subscription_started"
	EventSubscriptionCanceled = "billing.subscription_canceled"
	EventInvoicePaid          = "billing.invoice_paid"
	EventCheckoutCompleted    = "billing.checkout_completed"
)

// CustomerEvent is dispatched when a customer is created.
type CustomerEvent struct {
	Customer *Customer
	At       time.Time
}

// SubscriptionEvent is dispatched for subscription lifecycle changes.
type SubscriptionEvent struct {
	Subscription *Subscription
	At           time.Time
}

// InvoiceEvent is dispatched when an invoice is paid.
type InvoiceEvent struct {
	Invoice *Invoice
	At      time.Time
}

// CheckoutEvent is dispatched when checkout completes.
type CheckoutEvent struct {
	Session *CheckoutSession
	At      time.Time
}

// WebhookEvent is a normalized provider webhook payload.
type WebhookEvent struct {
	Type string
	Data map[string]any
}
