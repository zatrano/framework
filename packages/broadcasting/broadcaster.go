package broadcasting

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Broadcaster publishes events to named channels.
type Broadcaster interface {
	Broadcast(channel string, event string, payload map[string]any) error
}

// Manager resolves broadcasters.
type Manager struct {
	defaultDriver string
	drivers       map[string]Broadcaster
	channels      *ChannelRegistry
}

// NewManager creates a broadcasting manager.
func NewManager(defaultDriver string, drivers map[string]Broadcaster) *Manager {
	return &Manager{defaultDriver: defaultDriver, drivers: drivers}
}

// Driver returns a named broadcaster.
func (m *Manager) Driver(name ...string) Broadcaster {
	driver := m.defaultDriver
	if len(name) > 0 && name[0] != "" {
		driver = name[0]
	}
	return m.drivers[driver]
}

// Broadcast publishes using the default driver.
func (m *Manager) Broadcast(channel, event string, payload map[string]any) error {
	return m.Driver().Broadcast(channel, event, payload)
}

type infoLogger interface {
	Infof(format string, args ...any)
}

// LogBroadcaster writes broadcast payloads to the logger.
type LogBroadcaster struct {
	logger infoLogger
}

// NewLogBroadcaster creates a log broadcaster.
func NewLogBroadcaster(logger infoLogger) *LogBroadcaster {
	return &LogBroadcaster{logger: logger}
}

// Broadcast logs the event.
func (b *LogBroadcaster) Broadcast(channel, event string, payload map[string]any) error {
	raw, _ := json.Marshal(payload)
	b.logger.Infof("broadcast channel=%s event=%s payload=%s", channel, event, string(raw))
	return nil
}

// NullBroadcaster discards broadcasts.
type NullBroadcaster struct{}

// Broadcast is a no-op.
func (NullBroadcaster) Broadcast(channel, event string, payload map[string]any) error {
	return nil
}

// FileBroadcaster appends broadcasts to a JSONL file for local debugging.
type FileBroadcaster struct {
	mu   sync.Mutex
	path string
}

// NewFileBroadcaster creates a file broadcaster.
func NewFileBroadcaster(path string) (*FileBroadcaster, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	return &FileBroadcaster{path: path}, nil
}

// Broadcast appends a JSON line.
func (b *FileBroadcaster) Broadcast(channel, event string, payload map[string]any) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	entry := map[string]any{
		"time":    time.Now().Format(time.RFC3339),
		"channel": channel,
		"event":   event,
		"payload": payload,
	}
	raw, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(b.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = fmt.Fprintln(f, string(raw))
	return err
}
