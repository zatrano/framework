package notification

import (
	"fmt"
	"strings"
	"sync"
)

// SmsManager resolves SMS drivers (Twilio, HTTP, memory, …) like billing gateways.
type SmsManager struct {
	mu            sync.RWMutex
	from          string
	defaultDriver string
	senders       map[string]SmsSender
}

// NewSmsManager creates an SMS driver manager.
func NewSmsManager(from ...string) *SmsManager {
	m := &SmsManager{senders: make(map[string]SmsSender)}
	if len(from) > 0 {
		m.from = from[0]
	}
	return m
}

// Extend registers an SMS driver.
func (m *SmsManager) Extend(name string, sender SmsSender) {
	if m == nil || name == "" || sender == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.senders[strings.ToLower(strings.TrimSpace(name))] = sender
}

// Use selects the default SMS driver.
func (m *SmsManager) Use(name string) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.defaultDriver = strings.ToLower(strings.TrimSpace(name))
}

// Sender returns the default or named SMS driver.
func (m *SmsManager) Sender(name ...string) SmsSender {
	if m == nil {
		return nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	key := m.defaultDriver
	if len(name) > 0 && strings.TrimSpace(name[0]) != "" {
		key = strings.ToLower(strings.TrimSpace(name[0]))
	}
	return m.senders[key]
}

// Drivers returns registered driver names.
func (m *SmsManager) Drivers() []string {
	if m == nil {
		return nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]string, 0, len(m.senders))
	for name := range m.senders {
		out = append(out, name)
	}
	return out
}

// From returns the default SMS from identity.
func (m *SmsManager) From() string {
	if m == nil {
		return ""
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.from
}

// SetFrom sets the default SMS from identity.
func (m *SmsManager) SetFrom(from string) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.from = from
}

// Send delivers a message with the default driver (or Meta["driver"]).
func (m *SmsManager) Send(message *SmsMessage) error {
	if message == nil {
		return nil
	}
	driver := ""
	if message.Meta != nil {
		if d, ok := message.Meta["driver"].(string); ok {
			driver = d
		}
	}
	sender := m.Sender(driver)
	if sender == nil {
		if driver == "" {
			return fmt.Errorf("notification: no default SMS driver configured")
		}
		return fmt.Errorf("notification: SMS driver [%s] is not defined", driver)
	}
	if message.From == "" {
		message.From = m.From()
	}
	return sender.Send(message)
}
