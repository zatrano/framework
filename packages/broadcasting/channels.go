package broadcasting

import (
	"fmt"
	"strings"
	"sync"

	"github.com/zatrano/framework/packages/http"
	"github.com/zatrano/framework/packages/routing"
)

// ChannelAuth decides whether a request may subscribe to a channel.
type ChannelAuth func(req *http.Request, channel string) bool

// ChannelRegistry stores channel authorization callbacks.
type ChannelRegistry struct {
	mu       sync.RWMutex
	channels map[string]ChannelAuth
}

// NewChannelRegistry creates an empty channel registry.
func NewChannelRegistry() *ChannelRegistry {
	return &ChannelRegistry{channels: make(map[string]ChannelAuth)}
}

// Channel registers an authorizer for an exact or pattern channel name.
// Patterns may end with ".*" (prefix match).
func (r *ChannelRegistry) Channel(name string, auth ChannelAuth) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.channels[name] = auth
}

// Authorize checks whether the request may join the channel.
func (r *ChannelRegistry) Authorize(req *http.Request, channel string) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if auth, ok := r.channels[channel]; ok {
		return auth(req, channel), nil
	}
	for pattern, auth := range r.channels {
		if strings.HasSuffix(pattern, ".*") {
			prefix := strings.TrimSuffix(pattern, ".*")
			if strings.HasPrefix(channel, prefix) {
				return auth(req, channel), nil
			}
		}
	}
	return false, fmt.Errorf("channel [%s] is not defined", channel)
}

// Channels returns the channel registry on the manager (lazy).
func (m *Manager) Channels() *ChannelRegistry {
	if m.channels == nil {
		m.channels = NewChannelRegistry()
	}
	return m.channels
}

// Channel registers authorization on the manager.
func (m *Manager) Channel(name string, auth ChannelAuth) {
	m.Channels().Channel(name, auth)
}

// AuthHandler handles private channel authorization requests.
func (m *Manager) AuthHandler() routing.HandlerFunc {
	return func(req *http.Request) *http.Response {
		channel := req.Input("channel_name", req.Query("channel_name"))
		if channel == "" {
			return http.JSON(map[string]any{"message": "channel_name required"}).Status(422)
		}
		ok, err := m.Channels().Authorize(req, channel)
		if err != nil {
			return http.JSON(map[string]any{"message": err.Error()}).Status(404)
		}
		if !ok {
			return http.JSON(map[string]any{"message": "Forbidden"}).Status(403)
		}
		return http.JSON(map[string]any{
			"auth":    true,
			"channel": channel,
		})
	}
}
