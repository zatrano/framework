package audit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/zatrano/framework/packages/http"
	"github.com/zatrano/framework/packages/routing"
)

// Event is a single audit log entry.
type Event struct {
	Time      time.Time      `json:"time"`
	Action    string         `json:"action"`
	Actor     string         `json:"actor,omitempty"`
	Subject   string         `json:"subject,omitempty"`
	IP        string         `json:"ip,omitempty"`
	Method    string         `json:"method,omitempty"`
	Path      string         `json:"path,omitempty"`
	Status    int            `json:"status,omitempty"`
	RequestID string         `json:"request_id,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

// Store persists audit events.
type Store interface {
	Write(event Event) error
	Recent(limit int) ([]Event, error)
}

// MemoryStore keeps events in memory.
type MemoryStore struct {
	mu     sync.Mutex
	events []Event
	limit  int
}

// NewMemoryStore creates an in-memory store.
func NewMemoryStore(limit int) *MemoryStore {
	if limit <= 0 {
		limit = 500
	}
	return &MemoryStore{events: make([]Event, 0, limit), limit: limit}
}

// Write appends an event.
func (s *MemoryStore) Write(event Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, event)
	if len(s.events) > s.limit {
		s.events = s.events[len(s.events)-s.limit:]
	}
	return nil
}

// Recent returns the newest events first.
func (s *MemoryStore) Recent(limit int) ([]Event, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if limit <= 0 || limit > len(s.events) {
		limit = len(s.events)
	}
	out := make([]Event, 0, limit)
	for i := len(s.events) - 1; i >= 0 && len(out) < limit; i-- {
		out = append(out, s.events[i])
	}
	return out, nil
}

// FileStore appends JSONL audit events to disk.
type FileStore struct {
	mu   sync.Mutex
	path string
}

// NewFileStore creates a JSONL file store.
func NewFileStore(path string) (*FileStore, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	_ = os.Chmod(filepath.Dir(path), 0o700)
	return &FileStore{path: path}, nil
}

// Write appends one JSON line.
func (s *FileStore) Write(event Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	raw, err := json.Marshal(event)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(s.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(append(raw, '\n'))
	return err
}

// Recent reads the last N events from the file.
func (s *FileStore) Recent(limit int) ([]Event, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	raw, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return []Event{}, nil
		}
		return nil, err
	}
	lines := splitLines(string(raw))
	if limit <= 0 || limit > len(lines) {
		limit = len(lines)
	}
	out := make([]Event, 0, limit)
	for i := len(lines) - 1; i >= 0 && len(out) < limit; i-- {
		if lines[i] == "" {
			continue
		}
		var event Event
		if err := json.Unmarshal([]byte(lines[i]), &event); err != nil {
			continue
		}
		out = append(out, event)
	}
	return out, nil
}

// Manager records audit events.
type Manager struct {
	store Store
}

// New creates an audit manager.
func New(store Store) *Manager {
	return &Manager{store: store}
}

// Record writes an audit event.
func (m *Manager) Record(action string, opts ...func(*Event)) error {
	event := Event{
		Time:   time.Now().UTC(),
		Action: action,
	}
	for _, opt := range opts {
		opt(&event)
	}
	return m.store.Write(event)
}

// Recent returns recent events.
func (m *Manager) Recent(limit int) ([]Event, error) {
	return m.store.Recent(limit)
}

// WithActor sets the actor.
func WithActor(actor string) func(*Event) {
	return func(e *Event) { e.Actor = actor }
}

// WithSubject sets the subject.
func WithSubject(subject string) func(*Event) {
	return func(e *Event) { e.Subject = subject }
}

// WithMetadata sets metadata.
func WithMetadata(meta map[string]any) func(*Event) {
	return func(e *Event) { e.Metadata = meta }
}

// Middleware records HTTP requests as audit events.
func (m *Manager) Middleware() routing.MiddlewareFunc {
	return func(next routing.HandlerFunc) routing.HandlerFunc {
		return func(req *http.Request) *http.Response {
			resp := next(req)
			status := 200
			if resp != nil {
				status = resp.StatusCode()
			}
			requestID, _ := req.Get("request_id").(string)
			_ = m.Record("http.request",
				WithActor(req.IP()),
				func(e *Event) {
					e.IP = req.IP()
					e.Method = req.Method()
					e.Path = req.Path()
					e.Status = status
					e.RequestID = requestID
				},
			)
			return resp
		}
	}
}

func splitLines(input string) []string {
	out := make([]string, 0)
	start := 0
	for i := 0; i < len(input); i++ {
		if input[i] == '\n' {
			out = append(out, input[start:i])
			start = i + 1
		}
	}
	if start < len(input) {
		out = append(out, input[start:])
	}
	return out
}
