package notification

import (
	"strings"

	"github.com/zatrano/framework/packages/mail"
)

// Message is a central, channel-agnostic notification for web/API sends.
type Message struct {
	Base
	Channels []string
	Subject  string
	Body     string
	Data     map[string]any
}

// Via returns the selected channels (default: database).
func (m Message) Via() []string {
	if len(m.Channels) == 0 {
		return []string{"database"}
	}
	out := make([]string, 0, len(m.Channels))
	for _, ch := range m.Channels {
		ch = strings.TrimSpace(strings.ToLower(ch))
		if ch != "" {
			out = append(out, ch)
		}
	}
	if len(out) == 0 {
		return []string{"database"}
	}
	return out
}

// ToMail builds an email message.
func (m Message) ToMail(n Notifiable) *mail.Message {
	to := n.RouteNotificationFor("mail")
	msg := &mail.Message{
		Subject: m.Subject,
		Text:    m.Body,
		HTML:    m.Body,
	}
	if to != "" {
		msg.To = []string{to}
	}
	return msg
}

// ToDatabase stores the in-app payload.
func (m Message) ToDatabase(Notifiable) map[string]any {
	data := map[string]any{
		"subject": m.Subject,
		"body":    m.Body,
	}
	for k, v := range m.Data {
		data[k] = v
	}
	return data
}

// ToBroadcast publishes a compact payload.
func (m Message) ToBroadcast(Notifiable) map[string]any {
	return m.ToDatabase(nil)
}

// ToSms builds an SMS payload.
func (m Message) ToSms(n Notifiable) *SmsMessage {
	body := m.Body
	if body == "" {
		body = m.Subject
	}
	return &SmsMessage{
		To:   n.RouteNotificationFor("sms"),
		Body: body,
	}
}

// ToPush builds a push payload.
func (m Message) ToPush(Notifiable) map[string]any {
	return map[string]any{
		"title": m.Subject,
		"body":  m.Body,
		"data":  m.Data,
	}
}
