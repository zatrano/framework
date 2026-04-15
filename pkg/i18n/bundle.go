package i18n

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Bundle holds flattened message maps per locale tag (lowercase BCP47 base, e.g. en, tr).
type Bundle struct {
	defaultTag string
	supported  []string
	byLocale   map[string]map[string]string
}

// LoadDir reads one JSON file per locale: {dir}/{tag}.json (e.g. locales/en.json).
// Nested objects are flattened to dot keys ("a.b": "text"). Missing files yield an empty map for that locale.
func LoadDir(dir string, defaultLocale string, supported []string) (*Bundle, error) {
	dir = filepath.Clean(dir)
	st, err := os.Stat(dir)
	if err != nil {
		return nil, err
	}
	if !st.IsDir() {
		return nil, fmt.Errorf("i18n locales path is not a directory: %s", dir)
	}

	def := strings.ToLower(strings.TrimSpace(defaultLocale))
	if def == "" {
		return nil, fmt.Errorf("i18n: default locale is empty")
	}

	b := &Bundle{
		defaultTag: def,
		supported:  canonicalTags(supported),
		byLocale:   make(map[string]map[string]string),
	}
	for _, tag := range b.supported {
		path := filepath.Join(dir, tag+".json")
		raw, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				b.byLocale[tag] = map[string]string{}
				continue
			}
			return nil, fmt.Errorf("read %s: %w", path, err)
		}
		var root map[string]any
		if err := json.Unmarshal(raw, &root); err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
		flat := make(map[string]string)
		flatten("", root, flat)
		b.byLocale[tag] = flat
	}
	return b, nil
}

func canonicalTags(in []string) []string {
	out := make([]string, 0, len(in))
	seen := make(map[string]bool)
	for _, s := range in {
		t := strings.ToLower(strings.TrimSpace(s))
		if t == "" || seen[t] {
			continue
		}
		seen[t] = true
		out = append(out, t)
	}
	return out
}

func flatten(prefix string, v any, out map[string]string) {
	switch t := v.(type) {
	case map[string]any:
		for k, child := range t {
			next := k
			if prefix != "" {
				next = prefix + "." + k
			}
			flatten(next, child, out)
		}
	case string:
		if prefix != "" {
			out[prefix] = t
		}
	default:
		// numbers/bools ignored — use string leaves for copy
	}
}

// DefaultLocale returns the configured fallback tag.
func (b *Bundle) DefaultLocale() string { return b.defaultTag }

// Supported lists resolved locale tags.
func (b *Bundle) Supported() []string {
	cp := make([]string, len(b.supported))
	copy(cp, b.supported)
	return cp
}

// T returns the translation for key in locale, falling back to default locale, then key.
func (b *Bundle) T(locale, key string) string {
	locale = strings.ToLower(strings.TrimSpace(locale))
	key = strings.TrimSpace(key)
	if key == "" {
		return ""
	}
	if m := b.byLocale[locale]; m != nil {
		if v, ok := m[key]; ok && v != "" {
			return v
		}
	}
	if locale != b.defaultTag {
		if m := b.byLocale[b.defaultTag]; m != nil {
			if v, ok := m[key]; ok && v != "" {
				return v
			}
		}
	}
	return key
}

// MatchSupported returns the canonical supported tag for an arbitrary language tag, or "".
func (b *Bundle) MatchSupported(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	// strip quality part
	if i := strings.IndexByte(s, ';'); i >= 0 {
		s = strings.TrimSpace(s[:i])
	}
	for _, t := range b.supported {
		if strings.EqualFold(s, t) {
			return t
		}
	}
	if i := strings.IndexByte(s, '-'); i > 0 {
		base := s[:i]
		for _, t := range b.supported {
			if strings.EqualFold(base, t) {
				return t
			}
		}
	}
	return ""
}

// PickLocale resolves locale from query (?lang=), cookie, Accept-Language, then default.
func (b *Bundle) PickLocale(acceptLanguage, queryLang, cookieLang string) string {
	if q := strings.TrimSpace(queryLang); q != "" {
		if tag := b.MatchSupported(q); tag != "" {
			return tag
		}
	}
	if c := strings.TrimSpace(cookieLang); c != "" {
		if tag := b.MatchSupported(c); tag != "" {
			return tag
		}
	}
	for _, part := range strings.Split(acceptLanguage, ",") {
		lang := strings.TrimSpace(strings.Split(part, ";")[0])
		if tag := b.MatchSupported(lang); tag != "" {
			return tag
		}
	}
	return b.defaultTag
}

