package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// StoredRun is a durable snapshot of a finished (or failed) agent run.
type StoredRun struct {
	ID        string    `json:"id"`
	Agent     string    `json:"agent"`
	Message   string    `json:"message"`
	Content   string    `json:"content,omitempty"` // final assistant text
	Steps     int       `json:"steps,omitempty"`
	Error     string    `json:"error,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// ResultStore persists queued (or manual) run outcomes by ID.
type ResultStore interface {
	Put(ctx context.Context, run StoredRun) error
	Get(ctx context.Context, id string) (StoredRun, bool, error)
	Delete(ctx context.Context, id string) error
}

// MemoryResultStore is an in-process ResultStore.
type MemoryResultStore struct {
	mu   sync.RWMutex
	data map[string]StoredRun
}

// NewMemoryResultStore creates an empty memory result store.
func NewMemoryResultStore() *MemoryResultStore {
	return &MemoryResultStore{data: make(map[string]StoredRun)}
}

// Put implements ResultStore.
func (s *MemoryResultStore) Put(ctx context.Context, run StoredRun) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil {
		return fmt.Errorf("agent: result store is nil")
	}
	id := strings.TrimSpace(run.ID)
	if id == "" {
		return fmt.Errorf("agent: stored run id is required")
	}
	run.ID = id
	if run.CreatedAt.IsZero() {
		run.CreatedAt = time.Now().UTC()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.data == nil {
		s.data = make(map[string]StoredRun)
	}
	s.data[id] = run
	return nil
}

// Get implements ResultStore.
func (s *MemoryResultStore) Get(ctx context.Context, id string) (StoredRun, bool, error) {
	if err := ctx.Err(); err != nil {
		return StoredRun{}, false, err
	}
	if s == nil {
		return StoredRun{}, false, fmt.Errorf("agent: result store is nil")
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	run, ok := s.data[strings.TrimSpace(id)]
	return run, ok, nil
}

// Delete implements ResultStore.
func (s *MemoryResultStore) Delete(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil {
		return fmt.Errorf("agent: result store is nil")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, strings.TrimSpace(id))
	return nil
}

// JSONFileResultStore persists runs as a JSON object map on disk.
type JSONFileResultStore struct {
	Path string

	mu     sync.RWMutex
	data   map[string]StoredRun
	loaded bool
}

// NewJSONFileResultStore returns a file-backed result store.
func NewJSONFileResultStore(path string) *JSONFileResultStore {
	return &JSONFileResultStore{Path: path, data: make(map[string]StoredRun)}
}

func (s *JSONFileResultStore) ensureLoadedLocked() error {
	if s.loaded {
		return nil
	}
	if s.data == nil {
		s.data = make(map[string]StoredRun)
	}
	raw, err := os.ReadFile(s.Path)
	if err != nil {
		if os.IsNotExist(err) {
			s.loaded = true
			return nil
		}
		return err
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		s.loaded = true
		return nil
	}
	if err := json.Unmarshal(raw, &s.data); err != nil {
		return fmt.Errorf("agent: load results %s: %w", s.Path, err)
	}
	s.loaded = true
	return nil
}

func (s *JSONFileResultStore) persistLocked() error {
	if strings.TrimSpace(s.Path) == "" {
		return fmt.Errorf("agent: JSONFileResultStore path is required")
	}
	dir := filepath.Dir(s.Path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	raw, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.Path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.Path)
}

// Put implements ResultStore.
func (s *JSONFileResultStore) Put(ctx context.Context, run StoredRun) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil {
		return fmt.Errorf("agent: result store is nil")
	}
	id := strings.TrimSpace(run.ID)
	if id == "" {
		return fmt.Errorf("agent: stored run id is required")
	}
	run.ID = id
	if run.CreatedAt.IsZero() {
		run.CreatedAt = time.Now().UTC()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureLoadedLocked(); err != nil {
		return err
	}
	s.data[id] = run
	return s.persistLocked()
}

// Get implements ResultStore.
func (s *JSONFileResultStore) Get(ctx context.Context, id string) (StoredRun, bool, error) {
	if err := ctx.Err(); err != nil {
		return StoredRun{}, false, err
	}
	if s == nil {
		return StoredRun{}, false, fmt.Errorf("agent: result store is nil")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureLoadedLocked(); err != nil {
		return StoredRun{}, false, err
	}
	run, ok := s.data[strings.TrimSpace(id)]
	return run, ok, nil
}

// Delete implements ResultStore.
func (s *JSONFileResultStore) Delete(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil {
		return fmt.Errorf("agent: result store is nil")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureLoadedLocked(); err != nil {
		return err
	}
	delete(s.data, strings.TrimSpace(id))
	return s.persistLocked()
}

// OutcomeToStored converts a RunOutcome into a StoredRun (ID required).
func OutcomeToStored(o RunOutcome) StoredRun {
	run := StoredRun{
		ID:        o.ID,
		Agent:     o.Agent,
		Message:   o.Message,
		CreatedAt: time.Now().UTC(),
	}
	if o.Result != nil {
		run.Steps = o.Result.Steps
		if o.Result.Response != nil {
			run.Content = o.Result.Response.Message.Content
		}
	}
	if o.Err != nil {
		run.Error = o.Err.Error()
	}
	return run
}
