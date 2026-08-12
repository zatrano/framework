package notification

import (
	"fmt"
	"sync"
)

// SmsMessage is an SMS payload.
type SmsMessage struct {
	To   string
	Body string
	From string
	Meta map[string]any
}

// SmsNotification optionally supplies an SMS representation.
type SmsNotification interface {
	ToSms(notifiable Notifiable) *SmsMessage
}

// SmsSender delivers SMS messages.
type SmsSender interface {
	Send(message *SmsMessage) error
}

// MemorySmsSender records SMS deliveries for tests and demos.
type MemorySmsSender struct {
	mu      sync.Mutex
	Entries []*SmsMessage
}

// Send records the SMS.
func (s *MemorySmsSender) Send(message *SmsMessage) error {
	if message == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *message
	s.Entries = append(s.Entries, &cp)
	return nil
}

// Last returns the most recent SMS.
func (s *MemorySmsSender) Last() (*SmsMessage, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.Entries) == 0 {
		return nil, false
	}
	return s.Entries[len(s.Entries)-1], true
}

// LogSmsSender writes SMS payloads to a logger-style sink.
type LogSmsSender struct {
	Log func(format string, args ...any)
}

// Send logs the SMS.
func (s *LogSmsSender) Send(message *SmsMessage) error {
	if message == nil {
		return nil
	}
	log := s.Log
	if log == nil {
		log = func(format string, args ...any) { fmt.Printf(format+"\n", args...) }
	}
	log("sms to=%s from=%s body=%s", message.To, message.From, message.Body)
	return nil
}

// SmsChannel sends notifications via SMS.
type SmsChannel struct {
	sender SmsSender
	from   string
}

// NewSmsChannel creates an SMS notification channel.
func NewSmsChannel(sender SmsSender, from ...string) *SmsChannel {
	if sender == nil {
		sender = &MemorySmsSender{}
	}
	ch := &SmsChannel{sender: sender}
	if len(from) > 0 {
		ch.from = from[0]
	}
	return ch
}

// Send delivers the SMS representation.
func (c *SmsChannel) Send(notifiable Notifiable, notification Notification) error {
	var message *SmsMessage
	if n, ok := notification.(SmsNotification); ok {
		message = n.ToSms(notifiable)
	}
	if message == nil {
		return nil
	}
	if message.To == "" {
		message.To = notifiable.RouteNotificationFor("sms")
	}
	if message.To == "" {
		return fmt.Errorf("notification: sms recipient is empty")
	}
	if message.From == "" {
		message.From = c.from
	}
	return c.sender.Send(message)
}
