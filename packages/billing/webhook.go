package billing

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	zhttp "github.com/zatrano/framework/packages/http"
)

// HandleHTTP processes a framework HTTP webhook request.
func (m *Manager) HandleHTTP(req *zhttp.Request) error {
	if req == nil {
		return fmt.Errorf("billing: empty webhook request")
	}
	body, err := req.Body()
	if err != nil {
		return err
	}
	return m.ProcessWebhook(req.HeadersMap(), body)
}

// HandleWebhook verifies and processes a provider webhook HTTP request.
func (m *Manager) HandleWebhook(r *http.Request) error {
	if r == nil || r.Body == nil {
		return fmt.Errorf("billing: empty webhook request")
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 2<<20))
	if err != nil {
		return err
	}
	headers := map[string]string{}
	for k, vals := range r.Header {
		if len(vals) > 0 {
			headers[strings.ToLower(k)] = vals[0]
		}
	}
	return m.ProcessWebhook(headers, body)
}

// ProcessWebhook verifies signature (when configured) and applies the event.
func (m *Manager) ProcessWebhook(headers map[string]string, body []byte) error {
	m.mu.RLock()
	secret := m.webhookSecret
	gw := m.gateways[m.defaultGW]
	m.mu.RUnlock()

	if secret != "" {
		sig := headers["stripe-signature"]
		if sig == "" {
			sig = headers["Stripe-Signature"]
		}
		if err := verifyStripeSignature(secret, sig, body, time.Now()); err != nil {
			return err
		}
	}

	var evt WebhookEvent
	if parser, ok := gw.(WebhookParser); ok {
		parsed, err := parser.ParseWebhook(headers, body)
		if err != nil {
			return err
		}
		if parsed != nil {
			evt = *parsed
		}
	} else {
		var raw map[string]any
		if err := json.Unmarshal(body, &raw); err != nil {
			return fmt.Errorf("billing: webhook decode: %w", err)
		}
		evt.Type = stringField(raw, "type")
		if data, ok := raw["data"].(map[string]any); ok {
			if obj, ok := data["object"].(map[string]any); ok {
				evt.Data = obj
			} else {
				evt.Data = data
			}
		}
	}
	return m.applyWebhookEvent(evt)
}

func (m *Manager) applyWebhookEvent(evt WebhookEvent) error {
	switch evt.Type {
	case "checkout.session.completed":
		session := &CheckoutSession{
			ID:         stringField(evt.Data, "id"),
			CustomerID: stringField(evt.Data, "customer"),
			URL:        stringField(evt.Data, "url"),
			Status:     stringField(evt.Data, "status"),
		}
		if session.Status == "" {
			session.Status = "complete"
		}
		m.rememberCheckout(session)
		// Activate pending subscription tied to this session.
		pendingID := "sub_pending_" + session.ID
		m.mu.Lock()
		if sub, ok := m.subscriptions[pendingID]; ok {
			sub.Status = "active"
			if sub.ID == pendingID {
				// Keep pending id until Stripe sends subscription id; mark active.
			}
			m.mu.Unlock()
			m.dispatch(EventCheckoutCompleted, CheckoutEvent{Session: session, At: time.Now().UTC()})
			m.dispatch(EventSubscriptionStarted, SubscriptionEvent{Subscription: sub, At: time.Now().UTC()})
			email := m.customerEmail(session.CustomerID)
			m.notify(email, SubscriptionStartedNotification{Subscription: sub})
			return nil
		}
		m.mu.Unlock()
		m.dispatch(EventCheckoutCompleted, CheckoutEvent{Session: session, At: time.Now().UTC()})
		return nil

	case "customer.subscription.updated", "customer.subscription.deleted", "customer.subscription.created":
		sub := stripeSubscriptionToLocal(evt.Data)
		if sub.ID == "" {
			return nil
		}
		m.rememberSubscription(sub)
		if sub.Status == "active" || sub.Status == "trialing" {
			m.dispatch(EventSubscriptionStarted, SubscriptionEvent{Subscription: sub, At: time.Now().UTC()})
			m.notify(m.customerEmail(sub.CustomerID), SubscriptionStartedNotification{Subscription: sub})
		}
		if sub.Status == "canceled" {
			m.dispatch(EventSubscriptionCanceled, SubscriptionEvent{Subscription: sub, At: time.Now().UTC()})
		}
		return nil

	case "invoice.paid":
		inv := &Invoice{
			ID:         stringField(evt.Data, "id"),
			CustomerID: stringField(evt.Data, "customer"),
			Status:     "paid",
			Currency:   stringField(evt.Data, "currency"),
			CreatedAt:  time.Now().UTC(),
		}
		if amt, ok := evt.Data["amount_paid"].(float64); ok {
			inv.Amount = int64(amt)
		}
		m.rememberInvoice(inv)
		m.dispatch(EventInvoicePaid, InvoiceEvent{Invoice: inv, At: time.Now().UTC()})
		m.notify(m.customerEmail(inv.CustomerID), InvoicePaidNotification{Invoice: inv})
		return nil
	default:
		return nil
	}
}

func (m *Manager) customerEmail(customerID string) string {
	if customerID == "" {
		return ""
	}
	if c, err := m.Customer(customerID); err == nil && c != nil {
		return c.Email
	}
	return ""
}

func verifyStripeSignature(secret, header string, payload []byte, now time.Time) error {
	if header == "" {
		return fmt.Errorf("billing: missing stripe-signature")
	}
	var timestamp string
	var signatures []string
	for _, part := range strings.Split(header, ",") {
		part = strings.TrimSpace(part)
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		switch kv[0] {
		case "t":
			timestamp = kv[1]
		case "v1":
			signatures = append(signatures, kv[1])
		}
	}
	if timestamp == "" || len(signatures) == 0 {
		return fmt.Errorf("billing: invalid stripe-signature header")
	}
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return fmt.Errorf("billing: invalid stripe signature timestamp")
	}
	if now.Unix()-ts > 300 || ts-now.Unix() > 300 {
		return fmt.Errorf("billing: stripe signature timestamp outside tolerance")
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(timestamp))
	_, _ = mac.Write([]byte("."))
	_, _ = mac.Write(payload)
	expect := hex.EncodeToString(mac.Sum(nil))
	for _, sig := range signatures {
		if hmac.Equal([]byte(expect), []byte(sig)) {
			return nil
		}
	}
	return fmt.Errorf("billing: stripe signature mismatch")
}
