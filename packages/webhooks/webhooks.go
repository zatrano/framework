package webhooks

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// Event is an outbound webhook payload envelope.
type Event struct {
	ID        string         `json:"id"`
	Type      string         `json:"type"`
	CreatedAt time.Time      `json:"created_at"`
	Data      map[string]any `json:"data"`
}

// Endpoint is a webhook destination.
type Endpoint struct {
	URL    string
	Secret string
	Events []string // empty = all
}

// Delivery records an attempt.
type Delivery struct {
	Endpoint string    `json:"endpoint"`
	EventID  string    `json:"event_id"`
	Status   int       `json:"status"`
	Success  bool      `json:"success"`
	Error    string    `json:"error,omitempty"`
	At       time.Time `json:"at"`
}

// Manager dispatches signed outbound webhooks.
type Manager struct {
	mu        sync.RWMutex
	endpoints []Endpoint
	client    *http.Client
	recent    []Delivery
	limit     int
}

// New creates a webhook manager.
func New() *Manager {
	return &Manager{
		endpoints: make([]Endpoint, 0),
		client:    &http.Client{Timeout: 10 * time.Second},
		recent:    make([]Delivery, 0),
		limit:     100,
	}
}

// Register adds an endpoint.
func (m *Manager) Register(endpoint Endpoint) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.endpoints = append(m.endpoints, endpoint)
}

// Dispatch sends an event to matching endpoints.
func (m *Manager) Dispatch(eventType string, data map[string]any) ([]Delivery, error) {
	event := Event{
		ID:        fmt.Sprintf("evt_%d", time.Now().UnixNano()),
		Type:      eventType,
		CreatedAt: time.Now().UTC(),
		Data:      data,
	}
	raw, err := json.Marshal(event)
	if err != nil {
		return nil, err
	}

	m.mu.RLock()
	endpoints := append([]Endpoint{}, m.endpoints...)
	m.mu.RUnlock()

	deliveries := make([]Delivery, 0)
	for _, endpoint := range endpoints {
		if !matches(endpoint.Events, eventType) {
			continue
		}
		delivery := m.send(endpoint, event.ID, raw)
		deliveries = append(deliveries, delivery)
		m.remember(delivery)
	}
	return deliveries, nil
}

// Recent returns recent deliveries.
func (m *Manager) Recent(limit int) []Delivery {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if limit <= 0 || limit > len(m.recent) {
		limit = len(m.recent)
	}
	out := make([]Delivery, 0, limit)
	for i := len(m.recent) - 1; i >= 0 && len(out) < limit; i-- {
		out = append(out, m.recent[i])
	}
	return out
}

func (m *Manager) send(endpoint Endpoint, eventID string, body []byte) Delivery {
	delivery := Delivery{
		Endpoint: endpoint.URL,
		EventID:  eventID,
		At:       time.Now().UTC(),
	}
	req, err := http.NewRequest(http.MethodPost, endpoint.URL, bytes.NewReader(body))
	if err != nil {
		delivery.Error = err.Error()
		return delivery
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "ZATRANO-Webhooks/1.0")
	req.Header.Set("X-Zatrano-Event", eventID)
	if endpoint.Secret != "" {
		req.Header.Set("X-Zatrano-Signature", Sign(endpoint.Secret, body))
	}

	resp, err := m.client.Do(req)
	if err != nil {
		delivery.Error = err.Error()
		return delivery
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	delivery.Status = resp.StatusCode
	delivery.Success = resp.StatusCode >= 200 && resp.StatusCode < 300
	if !delivery.Success {
		delivery.Error = fmt.Sprintf("unexpected status %d", resp.StatusCode)
	}
	return delivery
}

func (m *Manager) remember(delivery Delivery) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.recent = append(m.recent, delivery)
	if len(m.recent) > m.limit {
		m.recent = m.recent[len(m.recent)-m.limit:]
	}
}

// Sign creates an HMAC SHA-256 signature hex digest.
func Sign(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

// Verify checks an HMAC signature.
func Verify(secret, signature string, body []byte) bool {
	expected := Sign(secret, body)
	return hmac.Equal([]byte(expected), []byte(signature))
}

func matches(allowed []string, eventType string) bool {
	if len(allowed) == 0 {
		return true
	}
	for _, item := range allowed {
		if item == "*" || item == eventType {
			return true
		}
	}
	return false
}
