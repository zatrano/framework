package report

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	zhttp "github.com/zatrano/framework/core/http"
)

// Event is a captured exception report.
type Event struct {
	ID        int64     `json:"id"`
	Message   string    `json:"message"`
	Level     string    `json:"level"`
	Path      string    `json:"path,omitempty"`
	Method    string    `json:"method,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// Manager stores recent exception reports and optionally POSTs them to a webhook.
type Manager struct {
	mu         sync.Mutex
	nextID     int64
	events     []Event
	limit      int
	webhookURL string
	client     *http.Client
}

// New creates a report manager.
func New(limit ...int) *Manager {
	n := 100
	if len(limit) > 0 && limit[0] > 0 {
		n = limit[0]
	}
	return &Manager{
		limit:  n,
		events: make([]Event, 0),
		nextID: 1,
		client: &http.Client{Timeout: 5 * time.Second},
	}
}

// SetWebhook configures an asynchronous JSON POST sink for captured events.
// An empty URL disables remote delivery. Events are always kept in the Recent buffer.
func (m *Manager) SetWebhook(url string) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.webhookURL = url
}

// Webhook returns the configured webhook URL (empty when unset).
func (m *Manager) Webhook() string {
	if m == nil {
		return ""
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.webhookURL
}

// Capture records an error in memory and, when a webhook is set, POSTs it asynchronously.
func (m *Manager) Capture(err error, req *zhttp.Request, level ...string) Event {
	if err == nil {
		return Event{}
	}
	lvl := "error"
	if len(level) > 0 && level[0] != "" {
		lvl = level[0]
	}
	ev := Event{
		Message:   err.Error(),
		Level:     lvl,
		CreatedAt: time.Now().UTC(),
	}
	if req != nil {
		ev.Path = req.Path()
		ev.Method = req.Method()
	}
	m.mu.Lock()
	ev.ID = m.nextID
	m.nextID++
	m.events = append([]Event{ev}, m.events...)
	if len(m.events) > m.limit {
		m.events = m.events[:m.limit]
	}
	webhook := m.webhookURL
	client := m.client
	m.mu.Unlock()

	if webhook != "" {
		go postWebhook(client, webhook, ev)
	}
	return ev
}

func postWebhook(client *http.Client, webhook string, ev Event) {
	body, err := json.Marshal(ev)
	if err != nil {
		return
	}
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhook, bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	_ = resp.Body.Close()
}

// Recent returns the latest events.
func (m *Manager) Recent(limit int) []Event {
	m.mu.Lock()
	defer m.mu.Unlock()
	if limit <= 0 || limit > len(m.events) {
		limit = len(m.events)
	}
	out := make([]Event, limit)
	copy(out, m.events[:limit])
	return out
}

// Count returns stored event count.
func (m *Manager) Count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.events)
}

// Clear removes all events.
func (m *Manager) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = m.events[:0]
}

// Reporter returns an exceptions.Reporter-compatible callback.
func (m *Manager) Reporter() func(err error, req *zhttp.Request) {
	return func(err error, req *zhttp.Request) {
		_ = m.Capture(err, req)
	}
}

// HTTPReporter POSTs JSON events to a remote URL (5s timeout, async via Report).
type HTTPReporter struct {
	URL    string
	Client *http.Client
}

// Report sends the event asynchronously. Prefer Manager.SetWebhook for in-memory + remote.
func (r *HTTPReporter) Report(ev Event) {
	if r == nil || r.URL == "" {
		return
	}
	client := r.Client
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	go postWebhook(client, r.URL, ev)
}
