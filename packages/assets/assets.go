package assets

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Manifest maps logical entry points to built asset paths (Vite/Mix style).
type Manifest struct {
	mu    sync.RWMutex
	root  string
	items map[string]Entry
}

// Entry describes a built asset.
type Entry struct {
	File    string   `json:"file"`
	Src     string   `json:"src,omitempty"`
	CSS     []string `json:"css,omitempty"`
	IsEntry bool     `json:"isEntry,omitempty"`
}

// New creates an empty manifest helper rooted at public URL prefix.
func New(publicRoot string) *Manifest {
	return &Manifest{
		root:  strings.TrimRight(publicRoot, "/"),
		items: make(map[string]Entry),
	}
}

// LoadFile loads a Vite-style manifest JSON file.
func (m *Manifest) LoadFile(path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var data map[string]Entry
	if err := json.Unmarshal(raw, &data); err != nil {
		// Mix-style: { "/js/app.js": "/js/app.abc123.js" }
		var mix map[string]string
		if err2 := json.Unmarshal(raw, &mix); err2 != nil {
			return err
		}
		data = make(map[string]Entry, len(mix))
		for k, v := range mix {
			data[k] = Entry{File: strings.TrimPrefix(v, "/"), Src: k, IsEntry: true}
		}
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items = data
	return nil
}

// Set registers a manifest entry manually.
func (m *Manifest) Set(key string, entry Entry) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items[key] = entry
}

// URL resolves a logical asset path to a public URL.
func (m *Manifest) URL(path string) string {
	path = strings.TrimPrefix(path, "/")
	m.mu.RLock()
	defer m.mu.RUnlock()

	if entry, ok := m.items[path]; ok && entry.File != "" {
		return m.root + "/" + strings.TrimPrefix(entry.File, "/")
	}
	// Vite keys often include a source path prefix
	for key, entry := range m.items {
		if strings.HasSuffix(key, path) || key == "/"+path {
			if entry.File != "" {
				return m.root + "/" + strings.TrimPrefix(entry.File, "/")
			}
		}
	}
	return m.root + "/" + path
}

// CSS returns associated CSS URLs for an entry.
func (m *Manifest) CSS(path string) []string {
	path = strings.TrimPrefix(path, "/")
	m.mu.RLock()
	defer m.mu.RUnlock()
	entry, ok := m.items[path]
	if !ok {
		for key, e := range m.items {
			if strings.HasSuffix(key, path) {
				entry = e
				ok = true
				break
			}
		}
	}
	if !ok {
		return nil
	}
	out := make([]string, 0, len(entry.CSS))
	for _, css := range entry.CSS {
		out = append(out, m.root+"/"+strings.TrimPrefix(css, "/"))
	}
	return out
}

// All returns a copy of manifest entries.
func (m *Manifest) All() map[string]Entry {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make(map[string]Entry, len(m.items))
	for k, v := range m.items {
		out[k] = v
	}
	return out
}

// LoadDefault tries common manifest locations under basePath.
func LoadDefault(basePath, publicURL string) *Manifest {
	publicURL = strings.TrimRight(publicURL, "/")
	candidates := []string{
		filepath.Join(basePath, "public", "build", "manifest.json"),
		filepath.Join(basePath, "public", "mix-manifest.json"),
		filepath.Join(basePath, "public", "assets", "manifest.json"),
	}
	for _, path := range candidates {
		var m *Manifest
		if strings.Contains(path, "mix-manifest") {
			m = New(publicURL)
		} else {
			m = New(publicURL + "/build")
		}
		if err := m.LoadFile(path); err == nil {
			return m
		}
	}
	return New(publicURL + "/build")
}
