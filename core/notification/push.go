package notification

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// PushNotification optionally supplies a push payload.
type PushNotification interface {
	ToPush(notifiable Notifiable) map[string]any
}

// PushSender delivers push payloads to a device token.
type PushSender interface {
	Send(deviceToken string, payload map[string]any) error
}

// MemoryPushSender records push deliveries for tests/demos.
type MemoryPushSender struct {
	mu      sync.Mutex
	Entries []PushEntry
}

// PushEntry is a recorded push delivery.
type PushEntry struct {
	Token   string
	Payload map[string]any
}

// Send records the push.
func (s *MemoryPushSender) Send(deviceToken string, payload map[string]any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Entries = append(s.Entries, PushEntry{Token: deviceToken, Payload: payload})
	return nil
}

// Last returns the most recent entry.
func (s *MemoryPushSender) Last() (PushEntry, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.Entries) == 0 {
		return PushEntry{}, false
	}
	return s.Entries[len(s.Entries)-1], true
}

// HTTPPushSender POSTs JSON to a webhook endpoint.
// Body: {"token":"...","payload":{...}} with Authorization: Bearer {Token}.
type HTTPPushSender struct {
	Endpoint string
	Token    string
	Client   *http.Client
}

// Send delivers the push via HTTP.
func (s *HTTPPushSender) Send(deviceToken string, payload map[string]any) error {
	if strings.TrimSpace(s.Endpoint) == "" {
		return fmt.Errorf("notification: PUSH_URL / HTTPPushSender.Endpoint is required")
	}
	body, err := json.Marshal(map[string]any{
		"token":   deviceToken,
		"payload": payload,
	})
	if err != nil {
		return err
	}
	client := s.Client
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	req, err := http.NewRequest(http.MethodPost, s.Endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if s.Token != "" {
		req.Header.Set("Authorization", "Bearer "+s.Token)
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := strings.TrimSpace(string(raw))
		if msg == "" {
			msg = resp.Status
		}
		return fmt.Errorf("notification: push http %d: %s", resp.StatusCode, msg)
	}
	return nil
}

// PushChannel sends notifications via a push sender.
type PushChannel struct {
	sender PushSender
}

// NewPushChannel creates a push notification channel.
func NewPushChannel(sender PushSender) *PushChannel {
	if sender == nil {
		sender = &MemoryPushSender{}
	}
	return &PushChannel{sender: sender}
}

// Send delivers the push representation.
func (c *PushChannel) Send(notifiable Notifiable, notification Notification) error {
	var payload map[string]any
	if p, ok := notification.(PushNotification); ok {
		payload = p.ToPush(notifiable)
	}
	if payload == nil {
		payload = notification.ToBroadcast(notifiable)
	}
	if payload == nil {
		payload = map[string]any{"type": fmt.Sprintf("%T", notification)}
	}
	token := notifiable.RouteNotificationFor("push")
	if token == "" {
		token = fmt.Sprintf("user:%v", notifiable.NotificationID())
	}
	return c.sender.Send(token, payload)
}
