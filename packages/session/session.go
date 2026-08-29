package session

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/zatrano/framework/packages/safepath"
)

// Manager handles file-based sessions.
type Manager struct {
	mu       sync.Mutex
	path     string
	lifetime time.Duration
	cookie   string
}

// Bag is an in-memory session for a single request lifecycle.
type Bag struct {
	id       string
	values   map[string]any
	flash    map[string]any
	oldFlash map[string]any
	manager  *Manager
	changed  bool
}

// NewManager creates a session manager.
func NewManager(path string, lifetimeMinutes int) *Manager {
	_ = os.MkdirAll(path, 0o700)
	_ = os.Chmod(path, 0o700) // harden pre-existing dirs created with looser modes
	return &Manager{
		path:     path,
		lifetime: time.Duration(lifetimeMinutes) * time.Minute,
		cookie:   "zatrano_session",
	}
}

// CookieName returns the session cookie name.
func (m *Manager) CookieName() string {
	return m.cookie
}

// Destroy removes a persisted session by ID.
func (m *Manager) Destroy(id string) error {
	if id == "" || !validSessionID(id) {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	path := filepath.Join(m.path, id)
	if !underDir(m.path, path) {
		return nil
	}
	err := os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// DestroyOthersForUser removes all persisted sessions for userID except one.
func (m *Manager) DestroyOthersForUser(userID any, exceptSessionID string) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	entries, err := os.ReadDir(m.path)
	if err != nil {
		return 0, err
	}
	target := fmt.Sprint(userID)
	deleted := 0
	for _, entry := range entries {
		if entry.IsDir() || entry.Name() == exceptSessionID {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(m.path, entry.Name()))
		if err != nil {
			return deleted, err
		}
		var payload struct {
			Values map[string]any `json:"values"`
		}
		if json.Unmarshal(raw, &payload) != nil || payload.Values == nil {
			continue
		}
		if sessionBelongsToUser(payload.Values, target) {
			if err := os.Remove(filepath.Join(m.path, entry.Name())); err != nil && !os.IsNotExist(err) {
				return deleted, err
			}
			deleted++
		}
	}
	return deleted, nil
}

func sessionBelongsToUser(values map[string]any, target string) bool {
	if fmt.Sprint(values["auth_user_id"]) == target {
		return true
	}
	for k, v := range values {
		if strings.HasPrefix(k, "login_") && strings.HasSuffix(k, "_id") && fmt.Sprint(v) == target {
			return true
		}
	}
	return false
}

// Start loads or creates a session bag.
func (m *Manager) Start(id string) (*Bag, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if id == "" || !validSessionID(id) {
		return m.newBag()
	}

	path := filepath.Join(m.path, id)
	// Defense in depth: resolved path must stay under the session directory.
	if !underDir(m.path, path) {
		return m.newBag()
	}

	var (
		raw  []byte
		info os.FileInfo
	)
	err := withFlock(path+".lock", func() error {
		var readErr error
		raw, readErr = os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		info, readErr = os.Stat(path)
		return readErr
	})
	if err != nil {
		return m.newBag()
	}

	if info != nil && m.lifetime > 0 && time.Since(info.ModTime()) > m.lifetime {
		_ = withFlock(path+".lock", func() error {
			return os.Remove(path)
		})
		return m.newBag()
	}

	var payload struct {
		Values map[string]any `json:"values"`
		Flash  map[string]any `json:"flash"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return m.newBag()
	}

	values := payload.Values
	if values == nil {
		values = make(map[string]any)
	}
	oldFlash := payload.Flash
	if oldFlash == nil {
		oldFlash = make(map[string]any)
	}
	// Persist once after flash is consumed so the next request does not re-read it.
	changed := len(oldFlash) > 0
	return &Bag{
		id:       id,
		values:   values,
		flash:    make(map[string]any),
		oldFlash: oldFlash,
		manager:  m,
		changed:  changed,
	}, nil
}

func (m *Manager) newBag() (*Bag, error) {
	id, err := generateID()
	if err != nil {
		return nil, err
	}
	return &Bag{
		id:       id,
		values:   make(map[string]any),
		flash:    make(map[string]any),
		oldFlash: make(map[string]any),
		manager:  m,
		changed:  false,
	}, nil
}

// Save persists the session bag.
func (m *Manager) Save(bag *Bag) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if bag == nil || !validSessionID(bag.id) {
		return fmt.Errorf("session: invalid session id")
	}
	path := filepath.Join(m.path, bag.id)
	if !underDir(m.path, path) {
		return fmt.Errorf("session: path escape blocked")
	}

	payload := map[string]any{
		"values": bag.values,
		"flash":  bag.flash,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return withFlock(path+".lock", func() error {
		tmp := path + ".tmp"
		if err := os.WriteFile(tmp, raw, 0o600); err != nil {
			return err
		}
		if err := os.Rename(tmp, path); err != nil {
			_ = os.Remove(path)
			if err2 := os.Rename(tmp, path); err2 != nil {
				_ = os.Remove(tmp)
				return err2
			}
		}
		return nil
	})
}

// Get returns a session value.
func (b *Bag) Get(key string, fallback ...any) any {
	if value, ok := b.oldFlash[key]; ok {
		return value
	}
	if value, ok := b.values[key]; ok {
		return value
	}
	if len(fallback) > 0 {
		return fallback[0]
	}
	return nil
}

// Put stores a session value.
func (b *Bag) Put(key string, value any) {
	b.values[key] = value
	b.changed = true
}

// Flash stores a value for the next request only.
func (b *Bag) Flash(key string, value any) {
	b.flash[key] = value
	b.changed = true
}

// Pull gets and forgets a value.
func (b *Bag) Pull(key string, fallback ...any) any {
	value := b.Get(key, fallback...)
	b.Forget(key)
	return value
}

// Forget removes a value.
func (b *Bag) Forget(key string) {
	delete(b.values, key)
	delete(b.flash, key)
	delete(b.oldFlash, key)
	b.changed = true
}

// Regenerate creates a new session ID.
func (b *Bag) Regenerate() error {
	old := b.id
	id, err := generateID()
	if err != nil {
		return err
	}
	b.id = id
	b.changed = true
	if old != "" {
		_ = os.Remove(filepath.Join(b.manager.path, old))
	}
	return nil
}

// ID returns the session ID.
func (b *Bag) ID() string {
	return b.id
}

// Changed reports whether the bag needs persistence (writes or pending flash drain).
func (b *Bag) Changed() bool {
	return b != nil && b.changed
}

// Has reports whether the key exists in values or flashed data.
func (b *Bag) Has(key string) bool {
	if b == nil {
		return false
	}
	if _, ok := b.oldFlash[key]; ok {
		return true
	}
	_, ok := b.values[key]
	return ok
}

// Exists is an alias of Has.
func (b *Bag) Exists(key string) bool {
	return b.Has(key)
}

// All returns a shallow copy of session values (excludes flash).
func (b *Bag) All() map[string]any {
	out := make(map[string]any, len(b.values))
	for k, v := range b.values {
		out[k] = v
	}
	return out
}

// Flush removes all session values and flash data.
func (b *Bag) Flush() {
	b.values = make(map[string]any)
	b.flash = make(map[string]any)
	b.oldFlash = make(map[string]any)
	b.changed = true
}

// Invalidate flushes the session and regenerates the ID.
func (b *Bag) Invalidate() error {
	b.Flush()
	return b.Regenerate()
}

// Keep re-flashes selected old flash keys for another request.
func (b *Bag) Keep(keys ...string) {
	if b.flash == nil {
		b.flash = make(map[string]any)
	}
	for _, key := range keys {
		if value, ok := b.oldFlash[key]; ok {
			b.flash[key] = value
			b.changed = true
		}
	}
}

// ReFlash keeps all old flash data for another request.
func (b *Bag) ReFlash() {
	if b.flash == nil {
		b.flash = make(map[string]any)
	}
	for key, value := range b.oldFlash {
		b.flash[key] = value
	}
	b.changed = true
}

// Increment increases a numeric session value.
func (b *Bag) Increment(key string, by ...int) int {
	step := 1
	if len(by) > 0 {
		step = by[0]
	}
	current := toInt(b.Get(key), 0)
	next := current + step
	b.Put(key, next)
	return next
}

// Decrement decreases a numeric session value.
func (b *Bag) Decrement(key string, by ...int) int {
	step := 1
	if len(by) > 0 {
		step = by[0]
	}
	return b.Increment(key, -step)
}

func toInt(v any, fallback int) int {
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	case string:
		var parsed int
		if _, err := fmt.Sscanf(n, "%d", &parsed); err == nil {
			return parsed
		}
	}
	return fallback
}

func generateID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// validSessionID accepts only hex IDs produced by generateID (32 lowercase hex chars).
func validSessionID(id string) bool {
	if len(id) != 32 {
		return false
	}
	for _, r := range id {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return false
		}
	}
	return true
}

func underDir(root, candidate string) bool {
	return safepath.Under(root, candidate)
}
