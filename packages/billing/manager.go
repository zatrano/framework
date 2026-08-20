package billing

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// Dispatcher receives optional billing lifecycle events.
type Dispatcher interface {
	Dispatch(name string, event any) error
}

// Notifier delivers billing notifications asynchronously (prefer notification.Send).
type Notifier func(email string, n any) error

// Manager is the central billing entry point. App code uses From(app) only.
type Manager struct {
	mu            sync.RWMutex
	baseURL       string
	successURL    string
	cancelURL     string
	defaultGW     string
	gateways      map[string]Gateway
	webhookSecret string
	dispatcher    Dispatcher
	notifier      Notifier

	// Local mirror used for Subscribed/OnTrial and webhook sync (all gateways).
	customers     map[string]*Customer
	subscriptions map[string]*Subscription
	invoices      map[string]*Invoice
	checkouts     map[string]*CheckoutSession
}

// New creates a billing manager with a memory gateway as default.
func New(baseURL string) *Manager {
	return NewManager(baseURL)
}

// NewManager creates a billing manager and registers the memory gateway.
func NewManager(baseURL string) *Manager {
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	base := strings.TrimRight(baseURL, "/")
	m := &Manager{
		baseURL:       base,
		successURL:    base + "/billing/success",
		cancelURL:     base + "/billing/cancel",
		gateways:      make(map[string]Gateway),
		customers:     make(map[string]*Customer),
		subscriptions: make(map[string]*Subscription),
		invoices:      make(map[string]*Invoice),
		checkouts:     make(map[string]*CheckoutSession),
	}
	mem := NewMemoryGateway(base)
	m.Extend("memory", mem)
	m.Use("memory")
	return m
}

// Extend registers a gateway driver.
func (m *Manager) Extend(name string, gw Gateway) {
	if m == nil || name == "" || gw == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.gateways[name] = gw
}

// Use selects the default gateway.
func (m *Manager) Use(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.defaultGW = name
}

// Gateway returns the default gateway (or a named one).
func (m *Manager) Gateway(name ...string) Gateway {
	m.mu.RLock()
	defer m.mu.RUnlock()
	key := m.defaultGW
	if len(name) > 0 && name[0] != "" {
		key = name[0]
	}
	return m.gateways[key]
}

// SetCheckoutURLs overrides success/cancel redirect URLs.
func (m *Manager) SetCheckoutURLs(successURL, cancelURL string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if successURL != "" {
		m.successURL = successURL
	}
	if cancelURL != "" {
		m.cancelURL = cancelURL
	}
}

// SetWebhookSecret configures Stripe webhook signature verification.
func (m *Manager) SetWebhookSecret(secret string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.webhookSecret = strings.TrimSpace(secret)
}

// SetDispatcher configures lifecycle event dispatching.
func (m *Manager) SetDispatcher(d Dispatcher) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.dispatcher = d
}

// SetNotifier configures async notification delivery for receipts.
func (m *Manager) SetNotifier(fn Notifier) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.notifier = fn
}

// SetStripeKey registers/enables the Stripe gateway and makes it default when key is non-empty.
// Kept for older call sites; prefer Extend("stripe", ...) + Use("stripe").
func (m *Manager) SetStripeKey(key string) {
	key = strings.TrimSpace(key)
	if key == "" {
		return
	}
	m.Extend("stripe", NewStripeGateway(key))
	m.Use("stripe")
}

// StripeEnabled reports whether the default gateway is Stripe.
func (m *Manager) StripeEnabled() bool {
	gw := m.Gateway()
	return gw != nil && gw.Name() == "stripe"
}

func (m *Manager) urls() (success, cancel string) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.successURL, m.cancelURL
}

func (m *Manager) rememberCustomer(c *Customer) {
	if c == nil {
		return
	}
	m.mu.Lock()
	m.customers[c.ID] = c
	m.mu.Unlock()
	if mem, ok := m.Gateway().(*MemoryGateway); ok {
		mem.storeCustomer(c)
	}
}

func (m *Manager) rememberSubscription(sub *Subscription) {
	if sub == nil {
		return
	}
	m.mu.Lock()
	m.subscriptions[sub.ID] = sub
	m.mu.Unlock()
	if mem, ok := m.Gateway().(*MemoryGateway); ok {
		mem.storeSubscription(sub)
	}
}

func (m *Manager) rememberInvoice(inv *Invoice) {
	if inv == nil {
		return
	}
	m.mu.Lock()
	m.invoices[inv.ID] = inv
	m.mu.Unlock()
	if mem, ok := m.Gateway().(*MemoryGateway); ok {
		mem.storeInvoice(inv)
	}
}

func (m *Manager) rememberCheckout(s *CheckoutSession) {
	if s == nil {
		return
	}
	m.mu.Lock()
	m.checkouts[s.ID] = s
	m.mu.Unlock()
	if mem, ok := m.Gateway().(*MemoryGateway); ok {
		mem.putCheckout(s)
	}
}

func (m *Manager) dispatch(name string, event any) {
	m.mu.RLock()
	d := m.dispatcher
	m.mu.RUnlock()
	if d != nil {
		_ = d.Dispatch(name, event)
	}
}

func (m *Manager) notify(email string, n any) {
	m.mu.RLock()
	fn := m.notifier
	m.mu.RUnlock()
	if fn != nil && email != "" {
		_ = fn(email, n)
	}
}

// CreateCustomer registers a customer via the default gateway.
func (m *Manager) CreateCustomer(email, name string) (*Customer, error) {
	gw := m.Gateway()
	if gw == nil {
		return nil, fmt.Errorf("billing: no gateway configured")
	}
	c, err := gw.CreateCustomer(email, name)
	if err != nil {
		return nil, err
	}
	m.rememberCustomer(c)
	m.dispatch(EventCustomerCreated, CustomerEvent{Customer: c, At: time.Now().UTC()})
	return c, nil
}

// Customer returns a customer by ID from the local mirror.
func (m *Manager) Customer(id string) (*Customer, error) {
	m.mu.RLock()
	c, ok := m.customers[id]
	m.mu.RUnlock()
	if ok {
		return c, nil
	}
	if mem, ok := m.Gateway().(*MemoryGateway); ok {
		if c, ok := mem.getCustomer(id); ok {
			m.rememberCustomer(c)
			return c, nil
		}
	}
	return nil, fmt.Errorf("billing: customer [%s] not found", id)
}

// Subscribe starts a subscription via the default gateway.
func (m *Manager) Subscribe(customerID, name, priceID string, trialDays ...int) (*Subscription, error) {
	if _, err := m.Customer(customerID); err != nil {
		return nil, err
	}
	gw := m.Gateway()
	if gw == nil {
		return nil, fmt.Errorf("billing: no gateway configured")
	}
	trial := 0
	if len(trialDays) > 0 && trialDays[0] > 0 {
		trial = trialDays[0]
	}
	success, cancel := m.urls()
	sub, session, err := gw.StartSubscription(customerID, name, priceID, trial, success, cancel)
	if err != nil {
		return nil, err
	}
	m.rememberSubscription(sub)
	if session != nil {
		m.rememberCheckout(session)
	}
	return sub, nil
}

// Cancel ends a subscription immediately or at period end.
func (m *Manager) Cancel(subscriptionID string, immediately bool) (*Subscription, error) {
	gw := m.Gateway()
	if gw == nil {
		return nil, fmt.Errorf("billing: no gateway configured")
	}

	// Pending Stripe checkouts are local-only.
	if strings.HasPrefix(subscriptionID, "sub_pending_") {
		m.mu.Lock()
		sub, ok := m.subscriptions[subscriptionID]
		if !ok {
			m.mu.Unlock()
			return nil, fmt.Errorf("billing: subscription [%s] not found", subscriptionID)
		}
		now := time.Now().UTC()
		if immediately {
			sub.Status = "canceled"
			sub.EndsAt = &now
		} else {
			sub.Status = "canceling"
			end := now.Add(30 * 24 * time.Hour)
			sub.EndsAt = &end
		}
		m.mu.Unlock()
		return sub, nil
	}

	sub, err := gw.CancelSubscription(subscriptionID, immediately)
	if err != nil {
		return nil, err
	}
	m.rememberSubscription(sub)
	m.dispatch(EventSubscriptionCanceled, SubscriptionEvent{Subscription: sub, At: time.Now().UTC()})
	return sub, nil
}

// Subscribed reports whether the customer has an active/trialing subscription.
func (m *Manager) Subscribed(customerID, name string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if name == "" {
		name = "default"
	}
	for _, sub := range m.subscriptions {
		if sub.CustomerID == customerID && sub.Name == name {
			if sub.Status == "active" || sub.Status == "trialing" || sub.Status == "canceling" {
				if sub.EndsAt != nil && time.Now().After(*sub.EndsAt) {
					continue
				}
				return true
			}
		}
	}
	return false
}

// OnTrial reports whether the customer is currently on trial.
func (m *Manager) OnTrial(customerID, name string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if name == "" {
		name = "default"
	}
	now := time.Now().UTC()
	for _, sub := range m.subscriptions {
		if sub.CustomerID == customerID && sub.Name == name && sub.TrialEnds != nil {
			if now.Before(*sub.TrialEnds) && (sub.Status == "trialing" || sub.Status == "active") {
				return true
			}
		}
	}
	return false
}

// Checkout creates a hosted checkout session URL.
func (m *Manager) Checkout(customerID, priceID string) (*CheckoutSession, error) {
	if _, err := m.Customer(customerID); err != nil {
		return nil, err
	}
	if priceID == "" {
		return nil, fmt.Errorf("billing: price_id is required")
	}
	gw := m.Gateway()
	if gw == nil {
		return nil, fmt.Errorf("billing: no gateway configured")
	}
	success, cancel := m.urls()
	session, err := gw.CreateCheckout(CheckoutInput{
		CustomerID: customerID,
		PriceID:    priceID,
		Mode:       "payment",
		SuccessURL: success,
		CancelURL:  cancel,
	})
	if err != nil {
		return nil, err
	}
	m.rememberCheckout(session)
	return session, nil
}

// CheckoutPayment creates a checkout session with inline price_data line items.
func (m *Manager) CheckoutPayment(customerID, currency string, lineItems []PaymentLineItem) (*CheckoutSession, error) {
	if _, err := m.Customer(customerID); err != nil {
		return nil, err
	}
	if len(lineItems) == 0 {
		return nil, fmt.Errorf("billing: line_items are required")
	}
	gw := m.Gateway()
	if gw == nil {
		return nil, fmt.Errorf("billing: no gateway configured")
	}
	success, cancel := m.urls()
	session, err := gw.CreateCheckout(CheckoutInput{
		CustomerID: customerID,
		Mode:       "payment",
		Currency:   currency,
		LineItems:  lineItems,
		SuccessURL: success,
		CancelURL:  cancel,
	})
	if err != nil {
		return nil, err
	}
	m.rememberCheckout(session)
	return session, nil
}

// Invoice charges a customer once.
func (m *Manager) Invoice(customerID string, amount int64, currency string) (*Invoice, error) {
	if _, err := m.Customer(customerID); err != nil {
		return nil, err
	}
	gw := m.Gateway()
	if gw == nil {
		return nil, fmt.Errorf("billing: no gateway configured")
	}
	inv, err := gw.CreateInvoice(customerID, amount, currency)
	if err != nil {
		return nil, err
	}
	m.rememberInvoice(inv)
	if inv.Status == "paid" {
		email := ""
		if c, err := m.Customer(customerID); err == nil {
			email = c.Email
		}
		m.notify(email, InvoicePaidNotification{Invoice: inv})
		m.dispatch(EventInvoicePaid, InvoiceEvent{Invoice: inv, At: time.Now().UTC()})
	}
	return inv, nil
}

// SubscriptionsFor returns subscriptions for a customer.
func (m *Manager) SubscriptionsFor(customerID string) []*Subscription {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*Subscription, 0)
	for _, sub := range m.subscriptions {
		if sub.CustomerID == customerID {
			cp := *sub
			out = append(out, &cp)
		}
	}
	return out
}

// CheckoutSession returns a stored checkout session by ID.
func (m *Manager) CheckoutSession(id string) (*CheckoutSession, bool) {
	m.mu.RLock()
	s, ok := m.checkouts[id]
	m.mu.RUnlock()
	if ok {
		return s, true
	}
	if mem, ok := m.Gateway().(*MemoryGateway); ok {
		return mem.getCheckout(id)
	}
	return nil, false
}
