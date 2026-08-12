package shorturl

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/zatrano/framework/packages/http"
	"github.com/zatrano/framework/packages/routing"
	"github.com/zatrano/framework/packages/support/uuid"
)

// Link is a shortened URL record.
type Link struct {
	Code      string     `json:"code"`
	URL       string     `json:"url"`
	Hits      int64      `json:"hits"`
	CreatedAt time.Time  `json:"created_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// Manager stores and resolves short links.
type Manager struct {
	mu      sync.Mutex
	baseURL string
	prefix  string
	links   map[string]*Link
}

// New creates a short URL manager.
func New(baseURL string, prefix ...string) *Manager {
	p := "/s"
	if len(prefix) > 0 && prefix[0] != "" {
		p = strings.TrimRight(prefix[0], "/")
		if !strings.HasPrefix(p, "/") {
			p = "/" + p
		}
	}
	return &Manager{
		baseURL: strings.TrimRight(baseURL, "/"),
		prefix:  p,
		links:   make(map[string]*Link),
	}
}

// Create shortens a destination URL.
func (m *Manager) Create(destination string, ttl ...time.Duration) (*Link, error) {
	destination = strings.TrimSpace(destination)
	if destination == "" {
		return nil, fmt.Errorf("shorturl: url is required")
	}
	if !strings.HasPrefix(destination, "http://") && !strings.HasPrefix(destination, "https://") {
		if strings.HasPrefix(destination, "/") {
			destination = m.baseURL + destination
		} else {
			return nil, fmt.Errorf("shorturl: url must be absolute or start with /")
		}
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	code := strings.ReplaceAll(uuid.New(), "-", "")[:8]
	link := &Link{
		Code:      code,
		URL:       destination,
		CreatedAt: time.Now().UTC(),
	}
	if len(ttl) > 0 && ttl[0] > 0 {
		exp := time.Now().UTC().Add(ttl[0])
		link.ExpiresAt = &exp
	}
	m.links[code] = link
	return link, nil
}

// Resolve returns a link by code.
func (m *Manager) Resolve(code string) (*Link, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	link, ok := m.links[code]
	if !ok {
		return nil, fmt.Errorf("shorturl: not found")
	}
	if link.ExpiresAt != nil && time.Now().After(*link.ExpiresAt) {
		delete(m.links, code)
		return nil, fmt.Errorf("shorturl: expired")
	}
	link.Hits++
	cp := *link
	return &cp, nil
}

// ShortURL builds the public short URL.
func (m *Manager) ShortURL(code string) string {
	return m.baseURL + m.prefix + "/" + code
}

// Prefix returns the path prefix.
func (m *Manager) Prefix() string { return m.prefix }

// RedirectHandler handles GET /s/{code}.
func (m *Manager) RedirectHandler() routing.HandlerFunc {
	return func(req *http.Request) *http.Response {
		code := req.Route("code")
		link, err := m.Resolve(code)
		if err != nil {
			return http.Abort(404, "Short URL not found")
		}
		return http.Redirect(link.URL, 302)
	}
}
