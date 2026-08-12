package inspector

import (
	"sync"
	"time"

	"github.com/zatrano/framework/packages/http"
	"github.com/zatrano/framework/packages/routing"
)

// Entry is a recorded HTTP request.
type Entry struct {
	ID         string            `json:"id"`
	Time       time.Time         `json:"time"`
	Method     string            `json:"method"`
	Path       string            `json:"path"`
	Status     int               `json:"status"`
	Duration   string            `json:"duration"`
	DurationMs float64           `json:"duration_ms"`
	IP         string            `json:"ip,omitempty"`
	RequestID  string            `json:"request_id,omitempty"`
	UserAgent  string            `json:"user_agent,omitempty"`
	Query      map[string]string `json:"query,omitempty"`
}

// Manager records recent HTTP requests for debugging.
type Manager struct {
	mu      sync.Mutex
	entries []Entry
	limit   int
	enabled bool
}

// New creates an inspector with a ring buffer.
func New(limit int) *Manager {
	if limit <= 0 {
		limit = 200
	}
	return &Manager{entries: make([]Entry, 0, limit), limit: limit, enabled: true}
}

// Enable toggles recording.
func (m *Manager) Enable(v bool) { m.enabled = v }

// Enabled reports whether recording is on.
func (m *Manager) Enabled() bool { return m.enabled }

// Record appends an entry.
func (m *Manager) Record(entry Entry) {
	if m == nil || !m.enabled {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entries = append(m.entries, entry)
	if len(m.entries) > m.limit {
		m.entries = m.entries[len(m.entries)-m.limit:]
	}
}

// Recent returns newest entries first.
func (m *Manager) Recent(limit int) []Entry {
	m.mu.Lock()
	defer m.mu.Unlock()
	if limit <= 0 || limit > len(m.entries) {
		limit = len(m.entries)
	}
	out := make([]Entry, 0, limit)
	for i := len(m.entries) - 1; i >= 0 && len(out) < limit; i-- {
		out = append(out, m.entries[i])
	}
	return out
}

// Clear empties the buffer.
func (m *Manager) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entries = m.entries[:0]
}

// Count returns recorded entry count.
func (m *Manager) Count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.entries)
}

// Middleware records each request.
func (m *Manager) Middleware() routing.MiddlewareFunc {
	return func(next routing.HandlerFunc) routing.HandlerFunc {
		return func(req *http.Request) *http.Response {
			start := time.Now()
			resp := next(req)
			status := 200
			if resp != nil {
				status = resp.StatusCode()
			}
			dur := time.Since(start)
			requestID, _ := req.Get("request_id").(string)
			entry := Entry{
				ID:         requestID,
				Time:       start.UTC(),
				Method:     req.Method(),
				Path:       req.Path(),
				Status:     status,
				Duration:   dur.String(),
				DurationMs: float64(dur.Microseconds()) / 1000.0,
				RequestID:  requestID,
				UserAgent:  req.Header("User-Agent"),
			}
			if raw := req.Raw(); raw != nil {
				entry.IP = raw.RemoteAddr
				if raw.URL != nil {
					q := map[string]string{}
					for k, vals := range raw.URL.Query() {
						if len(vals) > 0 {
							q[k] = vals[0]
						}
					}
					if len(q) > 0 {
						entry.Query = q
					}
				}
			}
			m.Record(entry)
			return resp
		}
	}
}
