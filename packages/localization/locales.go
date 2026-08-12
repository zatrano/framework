package localization

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/zatrano/framework/packages/localization/defaults"
)

var localeLabels = map[string]string{
	"en": "English",
	"tr": "Türkçe",
}

// Published reports whether app lang/ overrides have been published.
func Published(path string) bool {
	if path == "" {
		return false
	}
	entries, err := os.ReadDir(path)
	if err != nil || len(entries) == 0 {
		return false
	}
	for _, entry := range entries {
		if entry.IsDir() || strings.HasSuffix(entry.Name(), ".json") {
			return true
		}
	}
	return false
}

// HasLocale reports whether a locale exists in app lang/ or built-in defaults.
func HasLocale(path, locale string) bool {
	locale = strings.TrimSpace(strings.ToLower(locale))
	if locale == "" {
		return false
	}
	for _, code := range Available(path) {
		if code == locale {
			return true
		}
	}
	return false
}

// Available lists locale codes from published lang/ when present, otherwise built-ins.
func Available(path string) []string {
	if Published(path) {
		if codes := scanDirLocales(path); len(codes) > 0 {
			return codes
		}
	}
	return scanFSLocales(defaults.FS)
}

// Options builds select options for the current locale.
func Options(path, current string) []map[string]any {
	current = strings.TrimSpace(strings.ToLower(current))
	out := make([]map[string]any, 0)
	for _, code := range Available(path) {
		label := localeLabels[code]
		if label == "" {
			label = strings.ToUpper(code)
		}
		out = append(out, map[string]any{
			"code":     code,
			"label":    label,
			"selected": code == current,
		})
	}
	return out
}

func scanDirLocales(root string) []string {
	seen := map[string]struct{}{}
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() {
			seen[strings.ToLower(name)] = struct{}{}
			continue
		}
		if strings.HasSuffix(name, ".json") {
			seen[strings.ToLower(strings.TrimSuffix(name, ".json"))] = struct{}{}
		}
	}
	return sortedKeys(seen)
}

func scanFSLocales(fsys fs.FS) []string {
	seen := map[string]struct{}{}
	_ = fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || path == "." {
			return err
		}
		parts := strings.Split(filepath.ToSlash(path), "/")
		if len(parts) == 0 {
			return nil
		}
		if d.IsDir() && len(parts) == 1 {
			seen[strings.ToLower(parts[0])] = struct{}{}
			return nil
		}
		if !d.IsDir() && strings.HasSuffix(parts[len(parts)-1], ".json") && len(parts) >= 2 {
			seen[strings.ToLower(parts[0])] = struct{}{}
		}
		if !d.IsDir() && len(parts) == 1 && strings.HasSuffix(parts[0], ".json") {
			seen[strings.ToLower(strings.TrimSuffix(parts[0], ".json"))] = struct{}{}
		}
		return nil
	})
	return sortedKeys(seen)
}

func sortedKeys(seen map[string]struct{}) []string {
	out := make([]string, 0, len(seen))
	for code := range seen {
		out = append(out, code)
	}
	sort.Strings(out)
	return out
}
