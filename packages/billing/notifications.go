package billing

import (
	"fmt"

	"github.com/zatrano/framework/packages/notification"
)

// InvoicePaidNotification is sent when an invoice is paid.
type InvoicePaidNotification struct {
	notification.Base
	Invoice *Invoice
}

// Via uses the mail channel.
func (InvoicePaidNotification) Via() []string { return []string{"mail"} }

// ToMail builds the receipt email.
func (n InvoicePaidNotification) ToMail(notifiable notification.Notifiable) *notification.MailMessage {
	email := notifiable.RouteNotificationFor("mail")
	amount := int64(0)
	currency := "usd"
	id := ""
	if n.Invoice != nil {
		amount = n.Invoice.Amount
		currency = n.Invoice.Currency
		id = n.Invoice.ID
	}
	body := fmt.Sprintf("Payment received for invoice %s: %d %s", id, amount, currency)
	msg := &notification.MailMessage{
		Subject: "Payment received",
		Text:    body,
		HTML:    "<p>" + body + "</p>",
	}
	if email != "" {
		msg.To = []string{email}
	}
	return msg
}

// SubscriptionStartedNotification is sent when a subscription becomes active.
type SubscriptionStartedNotification struct {
	notification.Base
	Subscription *Subscription
}

// Via uses the mail channel.
func (SubscriptionStartedNotification) Via() []string { return []string{"mail"} }

// ToMail builds the welcome subscription email.
func (n SubscriptionStartedNotification) ToMail(notifiable notification.Notifiable) *notification.MailMessage {
	email := notifiable.RouteNotificationFor("mail")
	name := "default"
	id := ""
	if n.Subscription != nil {
		name = n.Subscription.Name
		id = n.Subscription.ID
	}
	body := fmt.Sprintf("Your subscription %s (%s) is active.", name, id)
	msg := &notification.MailMessage{
		Subject: "Subscription started",
		Text:    body,
		HTML:    "<p>" + body + "</p>",
	}
	if email != "" {
		msg.To = []string{email}
	}
	return msg
}
