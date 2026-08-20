package billing

import (
	"fmt"
	"strings"
)

// Billable is implemented by users/accounts that can be charged.
type Billable interface {
	BillingEmail() string
	BillingName() string
	GatewayCustomerID() string
	SetGatewayCustomerID(id string) error
}

// CustomerFor resolves or creates a gateway customer for a billable entity.
func (m *Manager) CustomerFor(b Billable) (*Customer, error) {
	if b == nil {
		return nil, fmt.Errorf("billing: billable is required")
	}
	if id := strings.TrimSpace(b.GatewayCustomerID()); id != "" {
		if c, err := m.Customer(id); err == nil {
			return c, nil
		}
	}
	email := strings.TrimSpace(b.BillingEmail())
	if email == "" {
		return nil, fmt.Errorf("billing: billable email is required")
	}
	c, err := m.CreateCustomer(email, b.BillingName())
	if err != nil {
		return nil, err
	}
	if err := b.SetGatewayCustomerID(c.ID); err != nil {
		return c, err
	}
	return c, nil
}

// SubscribeBillable starts a subscription for a billable entity.
func (m *Manager) SubscribeBillable(b Billable, name, priceID string, trialDays ...int) (*Subscription, error) {
	c, err := m.CustomerFor(b)
	if err != nil {
		return nil, err
	}
	return m.Subscribe(c.ID, name, priceID, trialDays...)
}

// CheckoutBillable creates a payment checkout for a billable entity.
func (m *Manager) CheckoutBillable(b Billable, priceID string) (*CheckoutSession, error) {
	c, err := m.CustomerFor(b)
	if err != nil {
		return nil, err
	}
	return m.Checkout(c.ID, priceID)
}

// SimpleBillable is an in-memory Billable for tests and demos.
type SimpleBillable struct {
	Email      string
	Name       string
	CustomerID string
}

// BillingEmail implements Billable.
func (b *SimpleBillable) BillingEmail() string { return b.Email }

// BillingName implements Billable.
func (b *SimpleBillable) BillingName() string { return b.Name }

// GatewayCustomerID implements Billable.
func (b *SimpleBillable) GatewayCustomerID() string { return b.CustomerID }

// SetGatewayCustomerID implements Billable.
func (b *SimpleBillable) SetGatewayCustomerID(id string) error {
	b.CustomerID = id
	return nil
}
