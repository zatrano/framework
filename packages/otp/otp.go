package otp

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"
)

// Store persists OTP codes.
type Store interface {
	Put(key, code string, ttl time.Duration) error
	Get(key string) (string, bool)
	Forget(key string) error
}

type memoryItem struct {
	code      string
	expiresAt time.Time
}

// MemoryStore is an in-memory OTP store.
type MemoryStore struct {
	mu    sync.RWMutex
	items map[string]memoryItem
}

// NewMemoryStore creates an empty memory store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{items: make(map[string]memoryItem)}
}

func (s *MemoryStore) Put(key, code string, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	s.items[key] = memoryItem{code: code, expiresAt: time.Now().Add(ttl)}
	return nil
}

func (s *MemoryStore) Get(key string) (string, bool) {
	s.mu.RLock()
	item, ok := s.items[key]
	s.mu.RUnlock()
	if !ok {
		return "", false
	}
	if time.Now().After(item.expiresAt) {
		_ = s.Forget(key)
		return "", false
	}
	return item.code, true
}

func (s *MemoryStore) Forget(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.items, key)
	return nil
}

// Manager issues and verifies one-time passwords.
type Manager struct {
	store  Store
	length int
	ttl    time.Duration
}

// New creates an OTP manager.
func New(store Store) *Manager {
	if store == nil {
		store = NewMemoryStore()
	}
	return &Manager{store: store, length: 6, ttl: 5 * time.Minute}
}

// WithLength sets code digit length (4–10).
func (m *Manager) WithLength(n int) *Manager {
	if n >= 4 && n <= 10 {
		m.length = n
	}
	return m
}

// WithTTL sets default expiry.
func (m *Manager) WithTTL(ttl time.Duration) *Manager {
	if ttl > 0 {
		m.ttl = ttl
	}
	return m
}

// Generate creates and stores a numeric OTP for key (e.g. phone or email).
func (m *Manager) Generate(key string, ttl ...time.Duration) (string, error) {
	key = normalizeKey(key)
	if key == "" {
		return "", fmt.Errorf("otp: key required")
	}
	code, err := randomDigits(m.length)
	if err != nil {
		return "", err
	}
	expiry := m.ttl
	if len(ttl) > 0 && ttl[0] > 0 {
		expiry = ttl[0]
	}
	if err := m.store.Put(key, code, expiry); err != nil {
		return "", err
	}
	return code, nil
}

// Verify checks a code and consumes it on success.
func (m *Manager) Verify(key, code string) bool {
	key = normalizeKey(key)
	code = strings.TrimSpace(code)
	stored, ok := m.store.Get(key)
	if !ok {
		return false
	}
	if stored != code {
		return false
	}
	_ = m.store.Forget(key)
	return true
}

// Peek returns whether a code is currently stored (does not consume).
func (m *Manager) Peek(key string) bool {
	_, ok := m.store.Get(normalizeKey(key))
	return ok
}

func normalizeKey(key string) string {
	return strings.ToLower(strings.TrimSpace(key))
}

func randomDigits(n int) (string, error) {
	var b strings.Builder
	b.Grow(n)
	for i := 0; i < n; i++ {
		v, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		b.WriteByte(byte('0' + v.Int64()))
	}
	return b.String(), nil
}
