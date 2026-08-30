package version

import (
	"os"
	"strings"
	"sync"
)

var (
	mu      sync.RWMutex
	current = "1.5.6"
)

// Set overrides the runtime version (tests / build injection).
func Set(v string) {
	mu.Lock()
	defer mu.Unlock()
	current = strings.TrimSpace(v)
}

// Get returns the current framework/app version.
func Get() string {
	mu.RLock()
	defer mu.RUnlock()
	return current
}

// LoadFile reads a VERSION file if present.
func LoadFile(path string) string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Get()
	}
	v := strings.TrimSpace(string(raw))
	if v != "" {
		Set(v)
	}
	return Get()
}
