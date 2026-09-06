package bootstrap

import (
	"strings"
	"sync"
)

// enablement is the process-level consumer manifest. Application bootstrap
// packages call RegisterEnablement from init(); addon init() must not.
//
// App() never reads the framework EnabledAddons variable. That slice is a
// leftover compile artifact for old generated assignments.
var (
	enablementMu      sync.RWMutex
	enablementPresent bool
	enablement        []string
)

// RegisterEnablement records the consumer enablement manifest for this process.
// An empty list is still a registered manifest (kernel-only unless WithAddons).
// If this is never called, App() falls back to DefaultMetas() (all imported).
func RegisterEnablement(names []string) {
	copied := make([]string, 0, len(names))
	seen := map[string]bool{}
	for _, name := range names {
		name = strings.ToLower(strings.TrimSpace(name))
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		copied = append(copied, name)
	}
	enablementMu.Lock()
	defer enablementMu.Unlock()
	enablementPresent = true
	enablement = copied
}

func clearEnablement() {
	enablementMu.Lock()
	defer enablementMu.Unlock()
	enablementPresent = false
	enablement = nil
}

func enablementRegistered() bool {
	enablementMu.RLock()
	defer enablementMu.RUnlock()
	return enablementPresent
}

func enablementNames() []string {
	enablementMu.RLock()
	defer enablementMu.RUnlock()
	return append([]string(nil), enablement...)
}
