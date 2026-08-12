package maintenance

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/zatrano/framework/packages/http"
	"github.com/zatrano/framework/packages/routing"
)

// Payload describes an active maintenance window.
type Payload struct {
	Message    string   `json:"message"`
	RetryAfter int      `json:"retry_after"`
	AllowedIPs []string `json:"allowed_ips"`
	Secret     string   `json:"secret"`
	Time       string   `json:"time"`
}

// Manager toggles maintenance mode via a down file.
type Manager struct {
	path string
}

// New creates a maintenance manager rooted at framework storage.
func New(frameworkPath string) *Manager {
	return &Manager{path: filepath.Join(frameworkPath, "down")}
}

// Path returns the down-file path.
func (m *Manager) Path() string { return m.path }

// Active reports whether maintenance mode is enabled.
func (m *Manager) Active() bool {
	_, err := os.Stat(m.path)
	return err == nil
}

// Enable writes the down file.
func (m *Manager) Enable(payload Payload) error {
	if payload.Message == "" {
		payload.Message = "Application is in maintenance mode."
	}
	if payload.RetryAfter <= 0 {
		payload.RetryAfter = 60
	}
	if payload.Time == "" {
		payload.Time = time.Now().UTC().Format(time.RFC3339)
	}
	if err := os.MkdirAll(filepath.Dir(m.path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.path, raw, 0o644)
}

// Disable removes the down file.
func (m *Manager) Disable() error {
	if err := os.Remove(m.path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// Payload loads the current maintenance payload.
func (m *Manager) Payload() (Payload, error) {
	raw, err := os.ReadFile(m.path)
	if err != nil {
		return Payload{}, err
	}
	var payload Payload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return Payload{Message: string(raw), RetryAfter: 60}, nil
	}
	return payload, nil
}

// Middleware returns 503 while maintenance mode is active.
func (m *Manager) Middleware(except ...string) routing.MiddlewareFunc {
	except = append(except, "/up", "/health")
	return func(next routing.HandlerFunc) routing.HandlerFunc {
		return func(req *http.Request) *http.Response {
			if !m.Active() {
				return next(req)
			}
			path := req.Path()
			for _, prefix := range except {
				if prefix != "" && (path == prefix || strings.HasPrefix(path, prefix+"/")) {
					return next(req)
				}
			}

			payload, _ := m.Payload()
			if payload.Secret != "" {
				if req.Query("secret") == payload.Secret || req.Header("X-Maintenance-Secret") == payload.Secret {
					return next(req)
				}
			}
			if ipAllowed(clientIP(req), payload.AllowedIPs) {
				return next(req)
			}

			body := map[string]any{
				"message":     payload.Message,
				"retry_after": payload.RetryAfter,
			}
			resp := http.JSON(body).Status(503)
			if !req.WantsJSON() {
				resp = http.HTML(fmt.Sprintf("<h1>Service Unavailable</h1><p>%s</p>", payload.Message)).Status(503)
			}
			resp.Header("Retry-After", fmt.Sprintf("%d", payload.RetryAfter))
			return resp
		}
	}
}

func clientIP(req *http.Request) string {
	raw := req.Raw()
	if raw == nil {
		return ""
	}
	host, _, err := net.SplitHostPort(raw.RemoteAddr)
	if err != nil {
		return raw.RemoteAddr
	}
	return host
}

func ipAllowed(ip string, allowed []string) bool {
	if ip == "" || len(allowed) == 0 {
		return false
	}
	for _, entry := range allowed {
		if entry == "*" || entry == ip {
			return true
		}
	}
	return false
}
