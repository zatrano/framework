package billing

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/zatrano/framework/packages/support/uuid"
)

// StripeGateway talks to the Stripe REST API.
type StripeGateway struct {
	secretKey  string
	httpClient *http.Client
}

// NewStripeGateway creates a Stripe gateway.
func NewStripeGateway(secretKey string) *StripeGateway {
	return &StripeGateway{
		secretKey:  strings.TrimSpace(secretKey),
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// Name returns the driver name.
func (g *StripeGateway) Name() string { return "stripe" }

// CreateCustomer creates a Stripe customer.
func (g *StripeGateway) CreateCustomer(email, name string) (*Customer, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" {
		return nil, fmt.Errorf("billing: email is required")
	}
	form := url.Values{}
	form.Set("email", email)
	if name != "" {
		form.Set("name", name)
	}
	raw, err := g.form(http.MethodPost, "https://api.stripe.com/v1/customers", form)
	if err != nil {
		return nil, err
	}
	id := stringField(raw, "id")
	if id == "" {
		return nil, fmt.Errorf("billing: stripe customer id missing")
	}
	return &Customer{
		ID:        id,
		Email:     email,
		Name:      name,
		CreatedAt: time.Now().UTC(),
	}, nil
}

// CreateCheckout creates a Stripe Checkout Session.
func (g *StripeGateway) CreateCheckout(input CheckoutInput) (*CheckoutSession, error) {
	if input.CustomerID == "" {
		return nil, fmt.Errorf("billing: customer_id is required")
	}
	mode := input.Mode
	if mode == "" {
		mode = "payment"
	}
	form := url.Values{}
	form.Set("mode", mode)
	form.Set("customer", input.CustomerID)
	form.Set("success_url", input.SuccessURL+"?session_id={CHECKOUT_SESSION_ID}")
	form.Set("cancel_url", input.CancelURL)

	if len(input.LineItems) > 0 {
		currency := strings.ToLower(input.Currency)
		if currency == "" {
			currency = "usd"
		}
		for i, item := range input.LineItems {
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
	} else {
		if input.PriceID == "" {
			return nil, fmt.Errorf("billing: price_id is required")
		}
		form.Set("line_items[0][price]", input.PriceID)
		form.Set("line_items[0][quantity]", "1")
		if mode == "subscription" && input.TrialDays > 0 {
			form.Set("subscription_data[trial_period_days]", fmt.Sprintf("%d", input.TrialDays))
		}
	}

	raw, err := g.form(http.MethodPost, "https://api.stripe.com/v1/checkout/sessions", form)
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
		CustomerID: input.CustomerID,
		PriceID:    input.PriceID,
		URL:        checkoutURL,
		Status:     status,
	}, nil
}

// StartSubscription creates a subscription-mode Checkout Session and a pending local sub.
func (g *StripeGateway) StartSubscription(customerID, name, priceID string, trialDays int, successURL, cancelURL string) (*Subscription, *CheckoutSession, error) {
	if name == "" {
		name = "default"
	}
	if priceID == "" {
		return nil, nil, fmt.Errorf("billing: price_id is required")
	}
	session, err := g.CreateCheckout(CheckoutInput{
		CustomerID: customerID,
		PriceID:    priceID,
		Mode:       "subscription",
		TrialDays:  trialDays,
		SuccessURL: successURL,
		CancelURL:  cancelURL,
		PlanName:   name,
	})
	if err != nil {
		return nil, nil, err
	}
	sub := &Subscription{
		ID:         "sub_pending_" + session.ID,
		CustomerID: customerID,
		Name:       name,
		PriceID:    priceID,
		Status:     "incomplete",
		CreatedAt:  time.Now().UTC(),
	}
	if trialDays > 0 {
		t := time.Now().UTC().Add(time.Duration(trialDays) * 24 * time.Hour)
		sub.TrialEnds = &t
	}
	return sub, session, nil
}

// CancelSubscription cancels a Stripe subscription.
func (g *StripeGateway) CancelSubscription(id string, immediately bool) (*Subscription, error) {
	if strings.HasPrefix(id, "sub_pending_") {
		return nil, fmt.Errorf("billing: pending subscription [%s] cannot be canceled via Stripe", id)
	}
	path := fmt.Sprintf("https://api.stripe.com/v1/subscriptions/%s", url.PathEscape(id))
	if !immediately {
		form := url.Values{}
		form.Set("cancel_at_period_end", "true")
		raw, err := g.form(http.MethodPost, path, form)
		if err != nil {
			return nil, err
		}
		return stripeSubscriptionToLocal(raw), nil
	}
	raw, err := g.form(http.MethodDelete, path, nil)
	if err != nil {
		return nil, err
	}
	return stripeSubscriptionToLocal(raw), nil
}

// CreateInvoice creates and pays a Stripe invoice.
func (g *StripeGateway) CreateInvoice(customerID string, amount int64, currency string) (*Invoice, error) {
	if amount <= 0 {
		return nil, fmt.Errorf("billing: amount must be positive")
	}
	if currency == "" {
		currency = "usd"
	}
	currency = strings.ToLower(currency)

	item := url.Values{}
	item.Set("customer", customerID)
	item.Set("amount", fmt.Sprintf("%d", amount))
	item.Set("currency", currency)
	item.Set("description", "ZATRANO charge")
	if _, err := g.form(http.MethodPost, "https://api.stripe.com/v1/invoiceitems", item); err != nil {
		return nil, err
	}

	form := url.Values{}
	form.Set("customer", customerID)
	form.Set("auto_advance", "true")
	form.Set("currency", currency)
	form.Set("pending_invoice_items_behavior", "include")
	raw, err := g.form(http.MethodPost, "https://api.stripe.com/v1/invoices", form)
	if err != nil {
		return nil, err
	}
	invID := stringField(raw, "id")
	if invID != "" {
		_, _ = g.form(http.MethodPost, "https://api.stripe.com/v1/invoices/"+url.PathEscape(invID)+"/finalize", nil)
		payRaw, payErr := g.form(http.MethodPost, "https://api.stripe.com/v1/invoices/"+url.PathEscape(invID)+"/pay", nil)
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
	return inv, nil
}

func (g *StripeGateway) form(method, endpoint string, form url.Values) (map[string]any, error) {
	if g.secretKey == "" {
		return nil, fmt.Errorf("billing: stripe key missing")
	}
	client := g.httpClient
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
	req.SetBasicAuth(g.secretKey, "")
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
