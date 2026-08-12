package localization

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/zatrano/framework/packages/localization/defaults"
)

// Translator resolves translation strings.
type Translator struct {
	mu       sync.RWMutex
	path     string
	locale   string
	fallback string
	lines    map[string]map[string]string // locale -> key -> value
}

// New creates a translator rooted at a lang directory.
func New(path, locale, fallback string) *Translator {
	if locale == "" {
		locale = "en"
	}
	if fallback == "" {
		fallback = locale
	}
	return &Translator{
		path:     path,
		locale:   locale,
		fallback: fallback,
		lines:    make(map[string]map[string]string),
	}
}

// SetLocale sets the active locale.
func (t *Translator) SetLocale(locale string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.locale = locale
}

// GetLocale returns the active locale.
func (t *Translator) GetLocale() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.locale
}

// Load loads JSON translation files for a locale.
// Built-in defaults ship under core/localization/defaults; optional overrides live in lang/.
// Expected app files: lang/{locale}.json or lang/{locale}/*.json
func (t *Translator) Load(locale string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	lines := map[string]string{}
	if err := mergeLocaleFS(defaults.FS, locale, lines); err != nil {
		return err
	}
	if t.path != "" {
		if err := mergeLocaleDir(t.path, locale, lines); err != nil {
			return err
		}
	}
	t.lines[locale] = lines
	return nil
}

func mergeLocaleFS(fsys fs.FS, locale string, lines map[string]string) error {
	single := locale + ".json"
	if raw, err := fs.ReadFile(fsys, single); err == nil {
		var data map[string]string
		if err := json.Unmarshal(raw, &data); err != nil {
			return err
		}
		for key, value := range data {
			lines[key] = value
		}
	}

	entries, err := fs.ReadDir(fsys, locale)
	if err != nil {
		return nil
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		raw, err := fs.ReadFile(fsys, locale+"/"+entry.Name())
		if err != nil {
			continue
		}
		group := strings.TrimSuffix(entry.Name(), ".json")
		var data map[string]string
		if err := json.Unmarshal(raw, &data); err != nil {
			return err
		}
		for key, value := range data {
			lines[group+"."+key] = value
		}
	}
	return nil
}

func mergeLocaleDir(root, locale string, lines map[string]string) error {
	single := filepath.Join(root, locale+".json")
	if raw, err := os.ReadFile(single); err == nil {
		var data map[string]string
		if err := json.Unmarshal(raw, &data); err != nil {
			return err
		}
		for key, value := range data {
			lines[key] = value
		}
	}

	dir := filepath.Join(root, locale)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}
		group := strings.TrimSuffix(entry.Name(), ".json")
		var data map[string]string
		if err := json.Unmarshal(raw, &data); err != nil {
			return err
		}
		for key, value := range data {
			lines[group+"."+key] = value
		}
	}
	return nil
}

// Get returns a translation string with replacements.
// Replacements use :name placeholders.
func (t *Translator) Get(key string, replace ...map[string]string) string {
	t.mu.RLock()
	locale := t.locale
	fallback := t.fallback
	t.mu.RUnlock()

	value, ok := t.lookup(locale, key)
	if !ok {
		value, ok = t.lookup(fallback, key)
	}
	if !ok {
		value = key
	}

	if len(replace) > 0 {
		for name, replacement := range replace[0] {
			value = strings.ReplaceAll(value, ":"+name, replacement)
		}
	}
	return value
}

// Choice chooses a pluralization line.
// Lines are separated by | and may use {0}, {1}, {2..} style markers.
func (t *Translator) Choice(key string, number int, replace ...map[string]string) string {
	line := t.Get(key)
	parts := strings.Split(line, "|")
	selected := parts[0]
	if number == 0 && len(parts) > 0 {
		selected = parts[0]
	}
	if number == 1 && len(parts) > 1 {
		selected = parts[1]
	}
	if number > 1 && len(parts) > 2 {
		selected = parts[2]
	} else if number > 1 && len(parts) > 1 {
		selected = parts[len(parts)-1]
	}

	replacements := map[string]string{"count": fmt.Sprint(number)}
	if len(replace) > 0 {
		for key, value := range replace[0] {
			replacements[key] = value
		}
	}
	for name, value := range replacements {
		selected = strings.ReplaceAll(selected, ":"+name, value)
	}
	return strings.TrimSpace(selected)
}

// Has reports whether a key exists for the current locale or fallback.
func (t *Translator) Has(key string) bool {
	t.mu.RLock()
	locale := t.locale
	fallback := t.fallback
	t.mu.RUnlock()
	if _, ok := t.lookup(locale, key); ok {
		return true
	}
	_, ok := t.lookup(fallback, key)
	return ok
}

func (t *Translator) lookup(locale, key string) (string, bool) {
	t.mu.RLock()
	lines, ok := t.lines[locale]
	t.mu.RUnlock()
	if !ok {
		_ = t.Load(locale)
		t.mu.RLock()
		lines, ok = t.lines[locale]
		t.mu.RUnlock()
		if !ok {
			return "", false
		}
	}
	value, found := lines[key]
	return value, found
}
