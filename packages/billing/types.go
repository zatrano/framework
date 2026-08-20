package billing

import "time"

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

// PaymentLineItem is an ad-hoc checkout line (amount in the smallest currency unit).
type PaymentLineItem struct {
	Name     string
	Amount   int64
	Quantity int64
}

// CheckoutInput configures hosted checkout (payment or subscription).
type CheckoutInput struct {
	CustomerID string
	PriceID    string
	Mode       string // payment | subscription
	TrialDays  int
	SuccessURL string
	CancelURL  string
	Currency   string
	LineItems  []PaymentLineItem
	PlanName   string // local subscription name when Mode=subscription
}
