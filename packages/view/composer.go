package view

import (
	"path"
	"strings"
)

// Composer injects data into views before render.
type Composer func(name string, data map[string]any)

// Composer registers a composer for matching view names ("*" = all, "auth.*" = prefix).
func (e *Engine) Composer(pattern string, composer Composer) {
	if composer == nil {
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.composers == nil {
		e.composers = make([]composerEntry, 0, 4)
	}
	e.composers = append(e.composers, composerEntry{pattern: strings.TrimSpace(pattern), fn: composer})
}

// Composers registers one composer for multiple patterns.
func (e *Engine) Composers(patterns []string, composer Composer) {
	for _, pattern := range patterns {
		e.Composer(pattern, composer)
	}
}

type composerEntry struct {
	pattern string
	fn      Composer
}

func (e *Engine) applyComposers(name string, data map[string]any) {
	e.mu.RLock()
	entries := append([]composerEntry{}, e.composers...)
	e.mu.RUnlock()
	for _, entry := range entries {
		if matchView(entry.pattern, name) {
			entry.fn(name, data)
		}
	}
}

func matchView(pattern, name string) bool {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" || pattern == "*" {
		return true
	}
	name = strings.ReplaceAll(name, "\\", "/")
	pattern = strings.ReplaceAll(pattern, "\\", "/")
	if strings.HasSuffix(pattern, ".*") {
		prefix := strings.TrimSuffix(pattern, ".*")
		return name == prefix || strings.HasPrefix(name, prefix+".") || strings.HasPrefix(name, prefix+"/")
	}
	if ok, _ := path.Match(pattern, name); ok {
		return true
	}
	return pattern == name
}
