package auth

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/zatrano/framework/packages/http"
)

// ErrLockout is returned when login attempts are temporarily throttled.
var ErrLockout = errors.New("auth.lockout")

// AttemptCounter is optional shared storage for lockouts (typically a cache store).
type AttemptCounter interface {
	Get(key string) (any, bool)
	Put(key string, value any, ttl time.Duration) error
	Forget(key string) error
}

type lockoutRecord struct {
	attempts  int
	expiresAt time.Time
}

type lockoutStore struct {
	mu     sync.Mutex
	items  map[string]lockoutRecord
	cache  AttemptCounter
	prefix string
	max    int
	decay  time.Duration
}

func newLockoutStore(max int, decay time.Duration) *lockoutStore {
	if max <= 0 {
		max = 5
	}
	if decay <= 0 {
		decay = time.Minute
	}
	return &lockoutStore{
		items:  make(map[string]lockoutRecord),
		prefix: "auth:lockout:",
		max:    max,
		decay:  decay,
	}
}

func (s *lockoutStore) setCache(c AttemptCounter) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cache = c
}

func (s *lockoutStore) locked(key string) bool {
	if s.cache != nil {
		raw, ok := s.cache.Get(s.prefix + key)
		if !ok {
			return false
		}
		return toInt(raw) >= s.max
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.items[key]
	if !ok || time.Now().After(item.expiresAt) {
		delete(s.items, key)
		return false
	}
	return item.attempts >= s.max
}

func (s *lockoutStore) hit(key string) bool {
	if s.cache != nil {
		full := s.prefix + key
		attempts := 0
		if raw, ok := s.cache.Get(full); ok {
			attempts = toInt(raw)
		}
		attempts++
		_ = s.cache.Put(full, attempts, s.decay)
		return attempts >= s.max
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	item := s.items[key]
	if time.Now().After(item.expiresAt) {
		item = lockoutRecord{}
	}
	item.attempts++
	item.expiresAt = time.Now().Add(s.decay)
	s.items[key] = item
	return item.attempts >= s.max
}

func (s *lockoutStore) clear(key string) {
	if s.cache != nil {
		_ = s.cache.Forget(s.prefix + key)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.items, key)
}

func toInt(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	case string:
		var out int
		_, _ = fmt.Sscanf(n, "%d", &out)
		return out
	default:
		var out int
		_, _ = fmt.Sscanf(fmt.Sprint(v), "%d", &out)
		return out
	}
}

func lockoutKey(req *http.Request, credentials map[string]string) string {
	ip := ""
	if req != nil {
		ip = req.IP()
	}
	return strings.ToLower(strings.TrimSpace(credentials["email"])) + "|" + ip
}

func twoFactorLockoutKey(req *http.Request, userID any) string {
	ip := ""
	if req != nil {
		ip = req.IP()
	}
	return "2fa|" + fmt.Sprint(userID) + "|" + ip
}
