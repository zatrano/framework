package cache

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Store is the cache contract.
type Store interface {
	Get(key string) (any, bool)
	Put(key string, value any, ttl time.Duration) error
	Forever(key string, value any) error
	Forget(key string) error
	Flush() error
	Has(key string) bool
	Pull(key string) (any, bool)
	Remember(key string, ttl time.Duration, callback func() (any, error)) (any, error)
}

type item struct {
	Value     any       `json:"value"`
	ExpiresAt time.Time `json:"expires_at"`
	Forever   bool      `json:"forever"`
}

// FileStore is a file-based cache store.
type FileStore struct {
	mu   sync.Mutex
	path string
}

// NewFileStore creates a file cache store.
func NewFileStore(path string) (*FileStore, error) {
	if err := os.MkdirAll(path, 0o755); err != nil {
		return nil, err
	}
	return &FileStore{path: path}, nil
}

// Get returns a cached value.
func (s *FileStore) Get(key string) (any, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	payload, err := s.read(key)
	if err != nil {
		return nil, false
	}
	if !payload.Forever && !payload.ExpiresAt.IsZero() && time.Now().After(payload.ExpiresAt) {
		_ = os.Remove(s.filename(key))
		return nil, false
	}
	return payload.Value, true
}

// Put stores a value with TTL.
func (s *FileStore) Put(key string, value any, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	payload := item{Value: value}
	if ttl > 0 {
		payload.ExpiresAt = time.Now().Add(ttl)
	} else {
		payload.Forever = true
	}
	return s.write(key, payload)
}

// Forever stores a value indefinitely.
func (s *FileStore) Forever(key string, value any) error {
	return s.Put(key, value, 0)
}

// Forget removes a key.
func (s *FileStore) Forget(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	err := os.Remove(s.filename(key))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// Flush clears the cache.
func (s *FileStore) Flush() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	entries, err := os.ReadDir(s.path)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		_ = os.Remove(filepath.Join(s.path, entry.Name()))
	}
	return nil
}

// Has reports whether a key exists.
func (s *FileStore) Has(key string) bool {
	_, ok := s.Get(key)
	return ok
}

// Pull gets and deletes a key.
func (s *FileStore) Pull(key string) (any, bool) {
	value, ok := s.Get(key)
	if ok {
		_ = s.Forget(key)
	}
	return value, ok
}

// Remember returns cached value or stores callback result.
func (s *FileStore) Remember(key string, ttl time.Duration, callback func() (any, error)) (any, error) {
	if value, ok := s.Get(key); ok {
		return value, nil
	}
	value, err := callback()
	if err != nil {
		return nil, err
	}
	if err := s.Put(key, value, ttl); err != nil {
		return nil, err
	}
	return value, nil
}

func (s *FileStore) filename(key string) string {
	sum := sha1.Sum([]byte(key))
	return filepath.Join(s.path, hex.EncodeToString(sum[:])+".cache")
}

func (s *FileStore) read(key string) (item, error) {
	raw, err := os.ReadFile(s.filename(key))
	if err != nil {
		return item{}, err
	}
	var payload item
	if err := json.Unmarshal(raw, &payload); err != nil {
		return item{}, err
	}
	return payload, nil
}

func (s *FileStore) write(key string, payload item) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return os.WriteFile(s.filename(key), raw, 0o644)
}

// Manager resolves cache stores.
type Manager struct {
	defaultStore string
	stores       map[string]Store
}

// NewManager creates a cache manager.
func NewManager(defaultStore string, stores map[string]Store) *Manager {
	return &Manager{defaultStore: defaultStore, stores: stores}
}

// Store returns a named store or the default.
func (m *Manager) Store(name ...string) Store {
	storeName := m.defaultStore
	if len(name) > 0 && name[0] != "" {
		storeName = name[0]
	}
	return m.stores[storeName]
}

// Get proxies to the default store.
func (m *Manager) Get(key string) (any, bool) { return m.Store().Get(key) }

// Put proxies to the default store.
func (m *Manager) Put(key string, value any, ttl time.Duration) error {
	return m.Store().Put(key, value, ttl)
}

// Forever proxies to the default store.
func (m *Manager) Forever(key string, value any) error { return m.Store().Forever(key, value) }

// Forget proxies to the default store.
func (m *Manager) Forget(key string) error { return m.Store().Forget(key) }

// Flush proxies to the default store.
func (m *Manager) Flush() error { return m.Store().Flush() }

// Remember proxies to the default store.
func (m *Manager) Remember(key string, ttl time.Duration, callback func() (any, error)) (any, error) {
	return m.Store().Remember(key, ttl, callback)
}
