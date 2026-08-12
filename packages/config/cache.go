package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// CachePath is the default relative path for cached config.
const CacheFileName = "config.json"

// SaveCache writes the repository snapshot to path.
func SaveCache(path string, repo *Repository) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(repo.All(), "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o644)
}

// LoadCache reads a cached config map from path.
func LoadCache(path string) (map[string]any, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var data map[string]any
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, err
	}
	return data, nil
}

// ClearCache removes the config cache file.
func ClearCache(path string) error {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// CacheExists reports whether a cache file is present.
func CacheExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// MergeCached loads cached config into the repository (top-level keys).
func (r *Repository) MergeCached(data map[string]any) {
	for key, value := range data {
		if nested, ok := value.(map[string]any); ok {
			r.Load(key, nested)
			continue
		}
		r.Set(key, value)
	}
}

// MustSaveCache saves or panics (CLI helpers).
func MustSaveCache(path string, repo *Repository) {
	if err := SaveCache(path, repo); err != nil {
		panic(fmt.Errorf("config cache: %w", err))
	}
}
