package apitoken

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/zatrano/framework/packages/auth"
	"github.com/zatrano/framework/packages/database/query"
	"github.com/zatrano/framework/packages/http"
	"github.com/zatrano/framework/packages/routing"
)

// Token is a personal access token record.
type Token struct {
	ID         int64      `json:"id"`
	Name       string     `json:"name"`
	TokenHash  string     `json:"-"`
	Abilities  []string   `json:"abilities"`
	UserID     any        `json:"user_id"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	PlainText  string     `json:"token,omitempty"` // only set on create
}

// Can reports whether the token has an ability (* = all).
func (t *Token) Can(ability string) bool {
	if t == nil {
		return false
	}
	for _, a := range t.Abilities {
		if a == "*" || a == ability {
			return true
		}
	}
	return false
}

// Expired reports whether the token is past expiry.
func (t *Token) Expired() bool {
	return t != nil && t.ExpiresAt != nil && time.Now().After(*t.ExpiresAt)
}

// Store persists personal access tokens.
type Store interface {
	Create(token *Token) error
	FindByHash(hash string) (*Token, error)
	Delete(id int64) error
	DeleteForUser(userID any) error
	ListForUser(userID any) ([]Token, error)
	Touch(id int64) error
}

// MemoryStore is an in-memory token store.
type MemoryStore struct {
	mu     sync.Mutex
	nextID int64
	items  map[string]*Token // hash -> token
}

// NewMemoryStore creates a memory store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{items: make(map[string]*Token), nextID: 1}
}

func (s *MemoryStore) Create(token *Token) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	token.ID = s.nextID
	s.nextID++
	token.CreatedAt = time.Now().UTC()
	cp := *token
	s.items[token.TokenHash] = &cp
	return nil
}

func (s *MemoryStore) FindByHash(hash string) (*Token, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.items[hash]
	if !ok {
		return nil, nil
	}
	cp := *t
	return &cp, nil
}

func (s *MemoryStore) Delete(id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for hash, t := range s.items {
		if t.ID == id {
			delete(s.items, hash)
			return nil
		}
	}
	return nil
}

func (s *MemoryStore) DeleteForUser(userID any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for hash, t := range s.items {
		if fmt.Sprint(t.UserID) == fmt.Sprint(userID) {
			delete(s.items, hash)
		}
	}
	return nil
}

func (s *MemoryStore) ListForUser(userID any) ([]Token, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Token, 0)
	for _, t := range s.items {
		if fmt.Sprint(t.UserID) == fmt.Sprint(userID) {
			out = append(out, *t)
		}
	}
	return out, nil
}

func (s *MemoryStore) Touch(id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	for _, t := range s.items {
		if t.ID == id {
			t.LastUsedAt = &now
			return nil
		}
	}
	return nil
}

// DatabaseStore stores tokens in personal_access_tokens.
type DatabaseStore struct {
	db     *sql.DB
	driver string
	table  string
}

// NewDatabaseStore creates a database-backed store.
func NewDatabaseStore(db *sql.DB, driver string) *DatabaseStore {
	return &DatabaseStore{db: db, driver: driver, table: "personal_access_tokens"}
}

func (s *DatabaseStore) Create(token *Token) error {
	abilities := strings.Join(token.Abilities, ",")
	id, err := query.New(s.db, s.driver, s.table).InsertGetID(map[string]any{
		"tokenable_id": token.UserID,
		"name":         token.Name,
		"token":        token.TokenHash,
		"abilities":    abilities,
		"expires_at":   token.ExpiresAt,
		"created_at":   time.Now().UTC(),
		"last_used_at": nil,
	})
	if err != nil {
		return err
	}
	token.ID = id
	token.CreatedAt = time.Now().UTC()
	return nil
}

func (s *DatabaseStore) FindByHash(hash string) (*Token, error) {
	row, err := query.New(s.db, s.driver, s.table).Where("token", hash).First()
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return rowToToken(row), nil
}

func (s *DatabaseStore) Delete(id int64) error {
	_, err := query.New(s.db, s.driver, s.table).Where("id", id).Delete()
	return err
}

func (s *DatabaseStore) DeleteForUser(userID any) error {
	_, err := query.New(s.db, s.driver, s.table).Where("tokenable_id", userID).Delete()
	return err
}

func (s *DatabaseStore) ListForUser(userID any) ([]Token, error) {
	rows, err := query.New(s.db, s.driver, s.table).Where("tokenable_id", userID).Get()
	if err != nil {
		return nil, err
	}
	out := make([]Token, 0, len(rows))
	for _, row := range rows {
		out = append(out, *rowToToken(row))
	}
	return out, nil
}

func (s *DatabaseStore) Touch(id int64) error {
	_, err := query.New(s.db, s.driver, s.table).Where("id", id).Update(map[string]any{
		"last_used_at": time.Now().UTC(),
	})
	return err
}

func rowToToken(row map[string]any) *Token {
	abilities := strings.Split(fmt.Sprint(row["abilities"]), ",")
	if len(abilities) == 1 && abilities[0] == "" {
		abilities = []string{"*"}
	}
	t := &Token{
		Name:      fmt.Sprint(row["name"]),
		TokenHash: fmt.Sprint(row["token"]),
		Abilities: abilities,
		UserID:    row["tokenable_id"],
	}
	switch id := row["id"].(type) {
	case int64:
		t.ID = id
	case int:
		t.ID = int64(id)
	default:
		fmt.Sscan(fmt.Sprint(row["id"]), &t.ID)
	}
	if v, ok := row["expires_at"].(time.Time); ok {
		t.ExpiresAt = &v
	}
	if v, ok := row["last_used_at"].(time.Time); ok {
		t.LastUsedAt = &v
	}
	if v, ok := row["created_at"].(time.Time); ok {
		t.CreatedAt = v
	}
	return t
}

// Manager issues and authenticates personal access tokens.
type Manager struct {
	store    Store
	provider auth.UserProvider
}

// New creates a token manager.
func New(store Store, provider auth.UserProvider) *Manager {
	return &Manager{store: store, provider: provider}
}

// Create issues a new token for a user. Returns the plain-text token once.
func (m *Manager) Create(userID any, name string, abilities []string, expiresIn ...time.Duration) (*Token, error) {
	if len(abilities) == 0 {
		abilities = []string{"*"}
	}
	plain, err := randomToken(40)
	if err != nil {
		return nil, err
	}
	token := &Token{
		Name:      name,
		TokenHash: Hash(plain),
		Abilities: abilities,
		UserID:    userID,
		PlainText: plain,
	}
	if len(expiresIn) > 0 && expiresIn[0] > 0 {
		exp := time.Now().UTC().Add(expiresIn[0])
		token.ExpiresAt = &exp
	}
	if err := m.store.Create(token); err != nil {
		return nil, err
	}
	return token, nil
}

// Find authenticates a plain-text bearer token.
func (m *Manager) Find(plain string) (*Token, auth.Authenticatable, error) {
	plain = strings.TrimSpace(plain)
	if plain == "" {
		return nil, nil, nil
	}
	token, err := m.store.FindByHash(Hash(plain))
	if err != nil || token == nil {
		return nil, nil, err
	}
	if token.Expired() {
		return nil, nil, nil
	}
	_ = m.store.Touch(token.ID)
	user, err := m.provider.RetrieveByID(token.UserID)
	if err != nil {
		return token, nil, err
	}
	return token, user, nil
}

// Revoke deletes a token by id.
func (m *Manager) Revoke(id int64) error {
	return m.store.Delete(id)
}

// RevokeAll deletes all tokens for a user.
func (m *Manager) RevokeAll(userID any) error {
	return m.store.DeleteForUser(userID)
}

// List returns tokens for a user.
func (m *Manager) List(userID any) ([]Token, error) {
	return m.store.ListForUser(userID)
}

// Middleware authenticates via Bearer personal access tokens.
func (m *Manager) Middleware(abilities ...string) routing.MiddlewareFunc {
	return func(next routing.HandlerFunc) routing.HandlerFunc {
		return func(req *http.Request) *http.Response {
			plain := req.BearerToken()
			if plain == "" {
				return http.JSON(map[string]any{"message": "Unauthenticated."}).Status(401)
			}
			token, user, err := m.Find(plain)
			if err != nil {
				return http.JSON(map[string]any{"message": err.Error()}).Status(500)
			}
			if token == nil || user == nil {
				return http.JSON(map[string]any{"message": "Unauthenticated."}).Status(401)
			}
			for _, ability := range abilities {
				if !token.Can(ability) {
					return http.JSON(map[string]any{"message": "Forbidden.", "ability": ability}).Status(403)
				}
			}
			req.Set("api_token", plain)
			req.Set("access_token", token)
			req.Set("user", user)
			req.Set(authRequestUserKey, user)
			return next(req)
		}
	}
}

const authRequestUserKey = "auth.user"

// Hash returns the SHA-256 hex digest of a plain token.
func Hash(plain string) string {
	sum := sha256.Sum256([]byte(plain))
	return hex.EncodeToString(sum[:])
}

func randomToken(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
