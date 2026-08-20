package billing

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/zatrano/framework/packages/support/uuid"
)

// MemoryGateway is an in-process billing driver for demos and tests.
type MemoryGateway struct {
	mu            sync.RWMutex
	baseURL       string
	customers     map[string]*Customer
	subscriptions map[string]*Subscription
	invoices      map[string]*Invoice
	checkouts     map[string]*CheckoutSession
}

// NewMemoryGateway creates a memory gateway.
func NewMemoryGateway(baseURL string) *MemoryGateway {
	base := strings.TrimRight(baseURL, "/")
	if base == "" {
		base = "http://localhost:8080"
	}
	return &MemoryGateway{
		baseURL:       base,
		customers:     make(map[string]*Customer),
		subscriptions: make(map[string]*Subscription),
		invoices:      make(map[string]*Invoice),
		checkouts:     make(map[string]*CheckoutSession),
	}
}

// Name returns the driver name.
func (g *MemoryGateway) Name() string { return "memory" }

// CreateCustomer registers or returns an existing customer by email.
func (g *MemoryGateway) CreateCustomer(email, name string) (*Customer, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" {
		return nil, fmt.Errorf("billing: email is required")
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	for _, c := range g.customers {
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
	g.customers[c.ID] = c
	return c, nil
}

// CreateCheckout creates a fake hosted checkout URL.
func (g *MemoryGateway) CreateCheckout(input CheckoutInput) (*CheckoutSession, error) {
	if input.CustomerID == "" {
		return nil, fmt.Errorf("billing: customer_id is required")
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if _, ok := g.customers[input.CustomerID]; !ok {
		return nil, fmt.Errorf("billing: customer [%s] not found", input.CustomerID)
	}
	session := &CheckoutSession{
		ID:         "cs_" + uuid.New()[:8],
		CustomerID: input.CustomerID,
		PriceID:    input.PriceID,
		Status:     "open",
	}
	session.URL = fmt.Sprintf("%s/billing/checkout/%s", g.baseURL, session.ID)
	g.checkouts[session.ID] = session
	return session, nil
}

// StartSubscription activates a local subscription immediately.
func (g *MemoryGateway) StartSubscription(customerID, name, priceID string, trialDays int, _, _ string) (*Subscription, *CheckoutSession, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if _, ok := g.customers[customerID]; !ok {
		return nil, nil, fmt.Errorf("billing: customer [%s] not found", customerID)
	}
	if name == "" {
		name = "default"
	}
	if priceID == "" {
		return nil, nil, fmt.Errorf("billing: price_id is required")
	}
	sub := &Subscription{
		ID:         "sub_" + uuid.New()[:8],
		CustomerID: customerID,
		Name:       name,
		PriceID:    priceID,
		Status:     "active",
		CreatedAt:  time.Now().UTC(),
	}
	if trialDays > 0 {
		t := time.Now().UTC().Add(time.Duration(trialDays) * 24 * time.Hour)
		sub.TrialEnds = &t
		sub.Status = "trialing"
	}
	g.subscriptions[sub.ID] = sub
	return sub, nil, nil
}

// CancelSubscription ends a local subscription.
func (g *MemoryGateway) CancelSubscription(id string, immediately bool) (*Subscription, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	sub, ok := g.subscriptions[id]
	if !ok {
		return nil, fmt.Errorf("billing: subscription [%s] not found", id)
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

// CreateInvoice records a paid local invoice.
func (g *MemoryGateway) CreateInvoice(customerID string, amount int64, currency string) (*Invoice, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if _, ok := g.customers[customerID]; !ok {
		return nil, fmt.Errorf("billing: customer [%s] not found", customerID)
	}
	if amount <= 0 {
		return nil, fmt.Errorf("billing: amount must be positive")
	}
	if currency == "" {
		currency = "usd"
	}
	inv := &Invoice{
		ID:         "in_" + uuid.New()[:8],
		CustomerID: customerID,
		Amount:     amount,
		Currency:   strings.ToLower(currency),
		Status:     "paid",
		CreatedAt:  time.Now().UTC(),
	}
	g.invoices[inv.ID] = inv
	return inv, nil
}

func (g *MemoryGateway) storeCustomer(c *Customer) {
	if c == nil {
		return
	}
	g.mu.Lock()
	g.customers[c.ID] = c
	g.mu.Unlock()
}

func (g *MemoryGateway) storeSubscription(sub *Subscription) {
	if sub == nil {
		return
	}
	g.mu.Lock()
	g.subscriptions[sub.ID] = sub
	g.mu.Unlock()
}

func (g *MemoryGateway) storeInvoice(inv *Invoice) {
	if inv == nil {
		return
	}
	g.mu.Lock()
	g.invoices[inv.ID] = inv
	g.mu.Unlock()
}

func (g *MemoryGateway) getCustomer(id string) (*Customer, bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	c, ok := g.customers[id]
	return c, ok
}

func (g *MemoryGateway) listSubscriptions(customerID string) []*Subscription {
	g.mu.RLock()
	defer g.mu.RUnlock()
	out := make([]*Subscription, 0)
	for _, sub := range g.subscriptions {
		if sub.CustomerID == customerID {
			cp := *sub
			out = append(out, &cp)
		}
	}
	return out
}

func (g *MemoryGateway) allSubscriptions() []*Subscription {
	g.mu.RLock()
	defer g.mu.RUnlock()
	out := make([]*Subscription, 0, len(g.subscriptions))
	for _, sub := range g.subscriptions {
		cp := *sub
		out = append(out, &cp)
	}
	return out
}

func (g *MemoryGateway) getCheckout(id string) (*CheckoutSession, bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	s, ok := g.checkouts[id]
	return s, ok
}

func (g *MemoryGateway) putCheckout(s *CheckoutSession) {
	if s == nil {
		return
	}
	g.mu.Lock()
	g.checkouts[s.ID] = s
	g.mu.Unlock()
}
