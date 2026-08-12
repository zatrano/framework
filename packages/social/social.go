package social

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"
)

// User is a normalized social identity.
type User struct {
	ID       string         `json:"id"`
	Nickname string         `json:"nickname,omitempty"`
	Name     string         `json:"name,omitempty"`
	Email    string         `json:"email,omitempty"`
	Avatar   string         `json:"avatar,omitempty"`
	Provider string         `json:"provider"`
	Token    string         `json:"token,omitempty"`
	Raw      map[string]any `json:"raw,omitempty"`
}

// Provider drives a social OAuth flow.
type Provider interface {
	Name() string
	RedirectURL(state string) string
	UserFromCode(code string) (*User, error)
}

// Config holds OAuth client settings.
type Config struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       []string
}

// Manager resolves named social providers.
type Manager struct {
	mu        sync.RWMutex
	providers map[string]Provider
	states    map[string]time.Time
	ttl       time.Duration
}

// New creates a social auth manager.
func New() *Manager {
	return &Manager{
		providers: make(map[string]Provider),
		states:    make(map[string]time.Time),
		ttl:       10 * time.Minute,
	}
}

// Extend registers a provider.
func (m *Manager) Extend(name string, provider Provider) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.providers[strings.ToLower(name)] = provider
}

// Driver returns a named provider.
func (m *Manager) Driver(name string) (Provider, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.providers[strings.ToLower(name)]
	if !ok {
		return nil, fmt.Errorf("social provider [%s] is not configured", name)
	}
	return p, nil
}

// Redirect builds an authorize URL and stores CSRF state.
func (m *Manager) Redirect(name string) (string, string, error) {
	provider, err := m.Driver(name)
	if err != nil {
		return "", "", err
	}
	state, err := randomState()
	if err != nil {
		return "", "", err
	}
	m.mu.Lock()
	m.states[state] = time.Now().Add(m.ttl)
	m.mu.Unlock()
	return provider.RedirectURL(state), state, nil
}

// ValidateState checks and consumes an OAuth state token.
func (m *Manager) ValidateState(state string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	exp, ok := m.states[state]
	if !ok {
		return false
	}
	delete(m.states, state)
	return time.Now().Before(exp)
}

// User resolves the social user from a callback code.
func (m *Manager) User(name, code string) (*User, error) {
	provider, err := m.Driver(name)
	if err != nil {
		return nil, err
	}
	return provider.UserFromCode(code)
}

// StubProvider is a local/dev OAuth stub (no real network).
type StubProvider struct {
	name  string
	cfg   Config
	base  string
	users map[string]*User
}

// NewStubProvider creates a stub OAuth provider.
func NewStubProvider(name string, cfg Config) *StubProvider {
	base := "https://oauth.zatrano.test/" + name
	return &StubProvider{
		name: name,
		cfg:  cfg,
		base: base,
		users: map[string]*User{
			"demo": {
				ID:       name + "-demo",
				Nickname: "demo",
				Name:     "Demo " + name,
				Email:    "demo@" + name + ".test",
				Avatar:   "https://www.gravatar.com/avatar/?d=mp",
				Provider: name,
			},
		},
	}
}

func (p *StubProvider) Name() string { return p.name }

func (p *StubProvider) RedirectURL(state string) string {
	values := url.Values{}
	values.Set("client_id", p.cfg.ClientID)
	values.Set("redirect_uri", p.cfg.RedirectURL)
	values.Set("response_type", "code")
	values.Set("state", state)
	if len(p.cfg.Scopes) > 0 {
		values.Set("scope", strings.Join(p.cfg.Scopes, " "))
	}
	return p.base + "/authorize?" + values.Encode()
}

func (p *StubProvider) UserFromCode(code string) (*User, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return nil, fmt.Errorf("missing authorization code")
	}
	if user, ok := p.users[code]; ok {
		cp := *user
		cp.Token = "stub-token-" + code
		cp.Raw = map[string]any{"code": code}
		return &cp, nil
	}
	// Any unknown code still yields a deterministic stub user in local mode.
	return &User{
		ID:       p.name + "-" + code,
		Nickname: code,
		Name:     "OAuth User",
		Email:    code + "@" + p.name + ".test",
		Avatar:   "https://www.gravatar.com/avatar/?d=identicon",
		Provider: p.name,
		Token:    "stub-token-" + code,
		Raw:      map[string]any{"code": code},
	}, nil
}

func randomState() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
