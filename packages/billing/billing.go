package billing

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/zatrano/framework/packages/support/uuid"
)

// Customer is a billable account.
type Customer struct {
	ID                   string    `json:"id"`
	Email                string    `json:"email"`
	Name                 string    `json:"name,omitempty"`
	DefaultPaymentMethod string    `json:"default_payment_method,omitempty"`
	CreatedAt            time.Time `json:"created_at"`
}

// Subscription is a recurring plan attachment.
type Subscription struct {
	ID         string     `json:"id"`
	CustomerID string     `json:"customer_id"`
	Name       string     `json:"name"`
	PriceID    string     `json:"price_id"`
	Status     string     `json:"status"`
	TrialEnds  *time.Time `json:"trial_ends_at,omitempty"`
	EndsAt     *time.Time `json:"ends_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

// Invoice is a one-off charge record.
type Invoice struct {
	ID         string    `json:"id"`
	CustomerID string    `json:"customer_id"`
	Amount     int64     `json:"amount"`
	Currency   string    `json:"currency"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
}

// CheckoutSession is a hosted checkout session.
type CheckoutSession struct {
	ID         string `json:"id"`
	CustomerID string `json:"customer_id"`
	PriceID    string `json:"price_id"`
	URL        string `json:"url"`
	Status     string `json:"status"`
}

// Manager provides billing APIs. Without STRIPE_SECRET_KEY it stays in-memory
// for demos/tests; with a key it calls the Stripe REST API for customers,
// Checkout Sessions, and subscription cancel.
type Manager struct {
	mu            sync.RWMutex
	baseURL       string
	stripeKey     string
	successURL    string
	cancelURL     string
	httpClient    *http.Client
	customers     map[string]*Customer
	subscriptions map[string]*Subscription
	invoices      map[string]*Invoice
	checkouts     map[string]*CheckoutSession
}

// New creates a billing manager (in-memory until SetStripeKey is called).
func New(baseURL string) *Manager {
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	base := strings.TrimRight(baseURL, "/")
	return &Manager{
		baseURL:       base,
		successURL:    base + "/billing/success",
		cancelURL:     base + "/billing/cancel",
		httpClient:    &http.Client{Timeout: 30 * time.Second},
		customers:     make(map[string]*Customer),
		subscriptions: make(map[string]*Subscription),
		invoices:      make(map[string]*Invoice),
		checkouts:     make(map[string]*CheckoutSession),
	}
}

// SetStripeKey enables Stripe REST mode when key is non-empty.
func (m *Manager) SetStripeKey(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stripeKey = strings.TrimSpace(key)
}

// StripeEnabled reports whether a Stripe secret key is configured.
func (m *Manager) StripeEnabled() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.stripeKey != ""
}

// SetCheckoutURLs overrides success/cancel redirect URLs used for Stripe Checkout.
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

// CreateCustomer registers a customer (Stripe Customers API when configured).
func (m *Manager) CreateCustomer(email, name string) (*Customer, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" {
		return nil, fmt.Errorf("billing: email is required")
	}
	if m.StripeEnabled() {
		form := url.Values{}
		form.Set("email", email)
		if name != "" {
			form.Set("name", name)
		}
		raw, err := m.stripeForm(http.MethodPost, "https://api.stripe.com/v1/customers", form)
		if err != nil {
			return nil, err
		}
		id := stringField(raw, "id")
		if id == "" {
			return nil, fmt.Errorf("billing: stripe customer id missing")
		}
		c := &Customer{
			ID:        id,
			Email:     email,
			Name:      name,
			CreatedAt: time.Now().UTC(),
		}
		m.mu.Lock()
		m.customers[c.ID] = c
		m.mu.Unlock()
		return c, nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	for _, c := range m.customers {
		if c.Email == email {
			return c, nil
		}
	}
	c := &Customer{
		ID:        "cus_" + uuid.New()[:8],
		Email:     email,
		Name:      name,
		CreatedAt: time.Now().UTC(),
	}
	m.customers[c.ID] = c
	return c, nil
}

// Customer returns a customer by ID.
func (m *Manager) Customer(id string) (*Customer, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	c, ok := m.customers[id]
	if !ok {
		return nil, fmt.Errorf("billing: customer [%s] not found", id)
	}
	return c, nil
}

// Subscribe starts a subscription. With Stripe, this creates a Checkout Session
// (mode=subscription) and a local tracking record with status "incomplete".
func (m *Manager) Subscribe(customerID, name, priceID string, trialDays ...int) (*Subscription, error) {
	if _, err := m.Customer(customerID); err != nil {
		return nil, err
	}
	if name == "" {
		name = "default"
	}
	if priceID == "" {
		return nil, fmt.Errorf("billing: price_id is required")
	}

	trial := 0
	if len(trialDays) > 0 && trialDays[0] > 0 {
		trial = trialDays[0]
	}

	if m.StripeEnabled() {
		session, err := m.createStripeCheckout(customerID, priceID, trial, "subscription")
		if err != nil {
			return nil, err
		}
		sub := &Subscription{
			ID:         "sub_pending_" + session.ID,
			CustomerID: customerID,
			Name:       name,
			PriceID:    priceID,
			Status:     "incomplete",
			CreatedAt:  time.Now().UTC(),
		}
		if trial > 0 {
			t := time.Now().UTC().Add(time.Duration(trial) * 24 * time.Hour)
			sub.TrialEnds = &t
		}
		m.mu.Lock()
		m.subscriptions[sub.ID] = sub
		m.checkouts[session.ID] = session
		m.mu.Unlock()
		return sub, nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	sub := &Subscription{
		ID:         "sub_" + uuid.New()[:8],
		CustomerID: customerID,
		Name:       name,
		PriceID:    priceID,
		Status:     "active",
		CreatedAt:  time.Now().UTC(),
	}
	if trial > 0 {
		t := time.Now().UTC().Add(time.Duration(trial) * 24 * time.Hour)
		sub.TrialEnds = &t
		sub.Status = "trialing"
	}
	m.subscriptions[sub.ID] = sub
	return sub, nil
}

// Cancel ends a subscription immediately or at period end.
// With Stripe and a real subscription id (sub_…), calls the Stripe cancel API.
func (m *Manager) Cancel(subscriptionID string, immediately bool) (*Subscription, error) {
	if m.StripeEnabled() && strings.HasPrefix(subscriptionID, "sub_") && !strings.HasPrefix(subscriptionID, "sub_pending_") {
		path := fmt.Sprintf("https://api.stripe.com/v1/subscriptions/%s", url.PathEscape(subscriptionID))
		form := url.Values{}
		if !immediately {
			form.Set("cancel_at_period_end", "true")
			raw, err := m.stripeForm(http.MethodPost, path, form)
			if err != nil {
				return nil, err
			}
			sub := stripeSubscriptionToLocal(raw)
			m.mu.Lock()
			m.subscriptions[sub.ID] = sub
			m.mu.Unlock()
			return sub, nil
		}
		raw, err := m.stripeForm(http.MethodDelete, path, nil)
		if err != nil {
			return nil, err
		}
		sub := stripeSubscriptionToLocal(raw)
		m.mu.Lock()
		m.subscriptions[sub.ID] = sub
		m.mu.Unlock()
		return sub, nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	sub, ok := m.subscriptions[subscriptionID]
	if !ok {
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

// Checkout creates a hosted checkout session URL (Stripe Checkout when configured).
// With Stripe, uses mode=payment for one-time price checkout.
func (m *Manager) Checkout(customerID, priceID string) (*CheckoutSession, error) {
	if _, err := m.Customer(customerID); err != nil {
		return nil, err
	}
	if priceID == "" {
		return nil, fmt.Errorf("billing: price_id is required")
	}
	if m.StripeEnabled() {
		session, err := m.createStripeCheckout(customerID, priceID, 0, "payment")
		if err != nil {
			return nil, err
		}
		m.mu.Lock()
		m.checkouts[session.ID] = session
		m.mu.Unlock()
		return session, nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	session := &CheckoutSession{
		ID:         "cs_" + uuid.New()[:8],
		CustomerID: customerID,
		PriceID:    priceID,
		Status:     "open",
	}
	session.URL = fmt.Sprintf("%s/billing/checkout/%s", m.baseURL, session.ID)
	m.checkouts[session.ID] = session
	return session, nil
}

// PaymentLineItem is an ad-hoc checkout line (amount in the smallest currency unit).
type PaymentLineItem struct {
	Name     string
	Amount   int64
	Quantity int64
}

// CheckoutPayment creates a Stripe Checkout Session with mode=payment and inline price_data.
func (m *Manager) CheckoutPayment(customerID, currency string, lineItems []PaymentLineItem) (*CheckoutSession, error) {
	if _, err := m.Customer(customerID); err != nil {
		return nil, err
	}
	if len(lineItems) == 0 {
		return nil, fmt.Errorf("billing: line_items are required")
	}
	if currency == "" {
		currency = "usd"
	}
	currency = strings.ToLower(currency)

	if m.StripeEnabled() {
		m.mu.RLock()
		successURL := m.successURL
		cancelURL := m.cancelURL
		m.mu.RUnlock()

		form := url.Values{}
		form.Set("mode", "payment")
		form.Set("customer", customerID)
		form.Set("success_url", successURL+"?session_id={CHECKOUT_SESSION_ID}")
		form.Set("cancel_url", cancelURL)
		for i, item := range lineItems {
			qty := item.Quantity
			if qty <= 0 {
				qty = 1
			}
			if item.Name == "" {
				return nil, fmt.Errorf("billing: line_items[%d].name is required", i)
			}
			if item.Amount <= 0 {
				return nil, fmt.Errorf("billing: line_items[%d].amount must be positive", i)
			}
			prefix := fmt.Sprintf("line_items[%d]", i)
			form.Set(prefix+"[price_data][currency]", currency)
			form.Set(prefix+"[price_data][unit_amount]", fmt.Sprintf("%d", item.Amount))
			form.Set(prefix+"[price_data][product_data][name]", item.Name)
			form.Set(prefix+"[quantity]", fmt.Sprintf("%d", qty))
		}
		raw, err := m.stripeForm(http.MethodPost, "https://api.stripe.com/v1/checkout/sessions", form)
		if err != nil {
			return nil, err
		}
		id := stringField(raw, "id")
		checkoutURL := stringField(raw, "url")
		status := stringField(raw, "status")
		if status == "" {
			status = "open"
		}
		if id == "" || checkoutURL == "" {
			return nil, fmt.Errorf("billing: stripe checkout session incomplete")
		}
		session := &CheckoutSession{
			ID:         id,
			CustomerID: customerID,
			URL:        checkoutURL,
			Status:     status,
		}
		m.mu.Lock()
		m.checkouts[session.ID] = session
		m.mu.Unlock()
		return session, nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	session := &CheckoutSession{
		ID:         "cs_" + uuid.New()[:8],
		CustomerID: customerID,
		Status:     "open",
	}
	session.URL = fmt.Sprintf("%s/billing/checkout/%s", m.baseURL, session.ID)
	m.checkouts[session.ID] = session
	return session, nil
}

func (m *Manager) createStripeCheckout(customerID, priceID string, trialDays int, mode string) (*CheckoutSession, error) {
	if mode == "" {
		mode = "subscription"
	}
	m.mu.RLock()
	successURL := m.successURL
	cancelURL := m.cancelURL
	m.mu.RUnlock()

	form := url.Values{}
	form.Set("mode", mode)
	form.Set("customer", customerID)
	form.Set("success_url", successURL+"?session_id={CHECKOUT_SESSION_ID}")
	form.Set("cancel_url", cancelURL)
	form.Set("line_items[0][price]", priceID)
	form.Set("line_items[0][quantity]", "1")
	if mode == "subscription" && trialDays > 0 {
		form.Set("subscription_data[trial_period_days]", fmt.Sprintf("%d", trialDays))
	}
	raw, err := m.stripeForm(http.MethodPost, "https://api.stripe.com/v1/checkout/sessions", form)
	if err != nil {
		return nil, err
	}
	id := stringField(raw, "id")
	checkoutURL := stringField(raw, "url")
	status := stringField(raw, "status")
	if status == "" {
		status = "open"
	}
	if id == "" || checkoutURL == "" {
		return nil, fmt.Errorf("billing: stripe checkout session incomplete")
	}
	return &CheckoutSession{
		ID:         id,
		CustomerID: customerID,
		PriceID:    priceID,
		URL:        checkoutURL,
		Status:     status,
	}, nil
}

// Invoice charges a customer once (in-memory record; Stripe invoice create when configured).
func (m *Manager) Invoice(customerID string, amount int64, currency string) (*Invoice, error) {
	if _, err := m.Customer(customerID); err != nil {
		return nil, err
	}
	if amount <= 0 {
		return nil, fmt.Errorf("billing: amount must be positive")
	}
	if currency == "" {
		currency = "usd"
	}
	currency = strings.ToLower(currency)

	if m.StripeEnabled() {
		form := url.Values{}
		form.Set("customer", customerID)
		form.Set("auto_advance", "true")
		form.Set("currency", currency)
		form.Set("pending_invoice_items_behavior", "include")
		item := url.Values{}
		item.Set("customer", customerID)
		item.Set("amount", fmt.Sprintf("%d", amount))
		item.Set("currency", currency)
		item.Set("description", "ZATRANO charge")
		if _, err := m.stripeForm(http.MethodPost, "https://api.stripe.com/v1/invoiceitems", item); err != nil {
			return nil, err
		}
		raw, err := m.stripeForm(http.MethodPost, "https://api.stripe.com/v1/invoices", form)
		if err != nil {
			return nil, err
		}
		invID := stringField(raw, "id")
		if invID != "" {
			_, _ = m.stripeForm(http.MethodPost, "https://api.stripe.com/v1/invoices/"+url.PathEscape(invID)+"/finalize", nil)
			payRaw, payErr := m.stripeForm(http.MethodPost, "https://api.stripe.com/v1/invoices/"+url.PathEscape(invID)+"/pay", nil)
			if payErr == nil {
				raw = payRaw
			}
		}
		inv := &Invoice{
			ID:         stringField(raw, "id"),
			CustomerID: customerID,
			Amount:     amount,
			Currency:   currency,
			Status:     stringField(raw, "status"),
			CreatedAt:  time.Now().UTC(),
		}
		if inv.ID == "" {
			inv.ID = "in_" + uuid.New()[:8]
		}
		if inv.Status == "" {
			inv.Status = "open"
		}
		m.mu.Lock()
		m.invoices[inv.ID] = inv
		m.mu.Unlock()
		return inv, nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	inv := &Invoice{
		ID:         "in_" + uuid.New()[:8],
		CustomerID: customerID,
		Amount:     amount,
		Currency:   currency,
		Status:     "paid",
		CreatedAt:  time.Now().UTC(),
	}
	m.invoices[inv.ID] = inv
	return inv, nil
}

// SubscriptionsFor returns subscriptions for a customer.
func (m *Manager) SubscriptionsFor(customerID string) []*Subscription {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*Subscription, 0)
	for _, sub := range m.subscriptions {
		if sub.CustomerID == customerID {
			out = append(out, sub)
		}
	}
	return out
}

// CheckoutSession returns a stored checkout session by ID.
func (m *Manager) CheckoutSession(id string) (*CheckoutSession, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.checkouts[id]
	return s, ok
}

func (m *Manager) stripeForm(method, endpoint string, form url.Values) (map[string]any, error) {
	m.mu.RLock()
	key := m.stripeKey
	client := m.httpClient
	m.mu.RUnlock()
	if key == "" {
		return nil, fmt.Errorf("billing: stripe key missing")
	}
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	var body io.Reader
	if form != nil {
		body = strings.NewReader(form.Encode())
	}
	req, err := http.NewRequest(method, endpoint, body)
	if err != nil {
		return nil, err
	}
	if form != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	req.SetBasicAuth(key, "")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil, err
	}
	var payload map[string]any
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &payload); err != nil {
			return nil, fmt.Errorf("billing: stripe decode: %w", err)
		}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := resp.Status
		if errObj, ok := payload["error"].(map[string]any); ok {
			if m, ok := errObj["message"].(string); ok && m != "" {
				msg = m
			}
		}
		return nil, fmt.Errorf("billing: stripe %d: %s", resp.StatusCode, msg)
	}
	if payload == nil {
		payload = map[string]any{}
	}
	return payload, nil
}

func stringField(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	default:
		s := strings.TrimSpace(fmt.Sprint(t))
		if s == "<nil>" {
			return ""
		}
		return s
	}
}

func stripeSubscriptionToLocal(raw map[string]any) *Subscription {
	sub := &Subscription{
		ID:         stringField(raw, "id"),
		CustomerID: stringField(raw, "customer"),
		Status:     stringField(raw, "status"),
		CreatedAt:  time.Now().UTC(),
		Name:       "default",
	}
	if sub.Status == "canceled" || sub.Status == "cancelled" {
		sub.Status = "canceled"
		now := time.Now().UTC()
		sub.EndsAt = &now
	} else if raw["cancel_at_period_end"] == true {
		sub.Status = "canceling"
	}
	return sub
}
